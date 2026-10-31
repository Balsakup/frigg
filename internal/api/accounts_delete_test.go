package api_test

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

func TestDeleteAccount(t *testing.T) {
	env := apitest.New(t)
	ctx := context.Background()
	adminEmail := env.Seed(auth.RoleAdmin, true)
	admin := env.LoggedIn(adminEmail)

	// Compte à supprimer : un administrateur qui a invité quelqu'un, qui a lui-même
	// été invité, dont un lien de réinitialisation est en file d'attente.
	target := env.Seed(auth.RoleAdmin, true)
	id := accountID(t, env, target)
	targetSession := env.LoggedIn(target)
	invite(t, env, targetSession, "invite-par-supprime@exemple.fr", "")
	if _, err := env.Store.WriteDB.Exec(`INSERT INTO invitations (email, role, token_hash, invited_by, status, expires_at, created_at, updated_at)
		VALUES (?, 'admin', x'00', NULL, 'accepted', '2026-11-07T00:00:00Z', '2026-10-31T00:00:00Z', '2026-10-31T00:00:00Z')`, target); err != nil {
		t.Fatal(err)
	}
	env.Client().Do("POST", "/api/auth/password-reset", map[string]string{"email": target})

	if res := admin.Do("DELETE", accountPath(id, ""), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", res.Status, res.Body)
	}
	expectProblem(t, admin.Do("GET", accountPath(id, ""), nil), http.StatusNotFound, api.MsgAccountGone)
	if res := targetSession.Do("GET", "/api/session", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("session du compte supprimé : %d, attendu 401", res.Status)
	}

	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := env.Store.ReadDB.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := count("SELECT count(*) FROM one_time_tokens WHERE account_id = ?", id); n != 0 {
		t.Errorf("%d jeton(s) restant(s)", n)
	}
	if n := count("SELECT count(*) FROM invitations WHERE email = ?", target); n != 0 {
		t.Errorf("%d invitation(s) adressée(s) au compte supprimé", n)
	}
	if n := count("SELECT count(*) FROM outbound_emails WHERE recipient = ?", target); n != 0 {
		t.Errorf("%d e-mail(s) en file pour le compte supprimé", n)
	}

	var list struct{ Items []api.Invitation }
	admin.Do("GET", "/api/admin/invitations", nil).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].Email != "invite-par-supprime@exemple.fr" || list.Items[0].InvitedBy != nil {
		t.Fatalf("invitations après suppression de l'auteur : %+v", list.Items)
	}

	expectProblem(t, env.Client().Do("POST", "/api/auth/login", map[string]string{"email": target, "password": apitest.Password}),
		http.StatusUnauthorized, api.MsgBadCredentials)
	if res := admin.Do("POST", "/api/admin/invitations", map[string]string{"email": target}); res.Status != http.StatusCreated {
		t.Fatalf("nouvelle invitation de l'adresse supprimée : %d %s", res.Status, res.Body)
	}
	expectProblem(t, admin.Do("DELETE", accountPath(accountID(t, env, adminEmail), ""), nil), http.StatusConflict, api.MsgSelfAction)
	expectProblem(t, admin.Do("DELETE", accountPath(id, ""), nil), http.StatusNotFound, api.MsgAccountGone)
	if n := actionLogs(t, env, "account.delete"); n != 1 {
		t.Fatalf("%d lignes de journal account.delete, attendu 1", n)
	}

	// Garde-fou au niveau de la requête : le dernier administrateur actif reste.
	if n, err := env.Store.Write.DeleteAccount(ctx, accountID(t, env, adminEmail)); err != nil || n != 0 {
		t.Fatalf("suppression du dernier administrateur actif : %d ligne(s), %v", n, err)
	}
	if _, err := env.Store.Read.GetAccountByEmail(ctx, adminEmail); err == sql.ErrNoRows {
		t.Fatal("dernier administrateur supprimé")
	}
}
