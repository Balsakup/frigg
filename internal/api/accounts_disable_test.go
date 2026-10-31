package api_test

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/mail/mailtest"
)

func TestDisableAndEnableAccount(t *testing.T) {
	env := apitest.New(t)
	adminEmail := env.Seed(auth.RoleAdmin, true)
	admin := env.LoggedIn(adminEmail)
	member := env.Seed(auth.RoleMember, false)
	id := accountID(t, env, member)
	session := env.LoggedIn(member)

	// Lien de réinitialisation obtenu avant la désactivation.
	env.Client().Do("POST", "/api/auth/password-reset", map[string]string{"email": member})
	env.FlushMail()
	sent, _ := env.Mail.Last(member)
	earlyToken := mailtest.Token(sent.Msg)
	mailsBefore := len(env.Mail.Messages())

	a := expectAccount(t, admin.Do("POST", accountPath(id, "/disable"), nil))
	if a.Status != "disabled" || a.DisabledAt == nil {
		t.Fatalf("compte non désactivé : %+v", a)
	}
	expectProblem(t, session.Do("GET", "/api/session", nil), http.StatusUnauthorized, "Votre session a expiré.")
	expectProblem(t, env.Client().Do("POST", "/api/auth/login", map[string]string{"email": member, "password": apitest.Password}),
		http.StatusUnauthorized, api.MsgBadCredentials)
	if res := env.Client().Do("POST", "/api/auth/password-reset", map[string]string{"email": member}); res.Status != http.StatusAccepted {
		t.Fatalf("demande de réinitialisation : %d", res.Status)
	}
	env.FlushMail()
	if n := len(env.Mail.Messages()); n != mailsBefore {
		t.Fatalf("%d e-mail(s) envoyé(s) à un compte désactivé", n-mailsBefore)
	}
	if res := env.Client().Do("POST", "/api/auth/password-reset/confirm", map[string]string{"token": earlyToken, "password": "Nouveau2026"}); res.Status != http.StatusGone {
		t.Fatalf("lien antérieur à la désactivation : %d, attendu 410", res.Status)
	}
	expectProblem(t, admin.Do("POST", "/api/admin/invitations", map[string]string{"email": member}), http.StatusConflict, api.MsgAccountExists)

	expectAccount(t, admin.Do("POST", accountPath(id, "/disable"), nil))
	if n := actionLogs(t, env, "account.disable"); n != 1 {
		t.Fatalf("%d lignes de journal account.disable, attendu 1", n)
	}
	expectProblem(t, admin.Do("POST", accountPath(accountID(t, env, adminEmail), "/disable"), nil), http.StatusConflict, api.MsgSelfAction)

	a = expectAccount(t, admin.Do("POST", accountPath(id, "/enable"), nil))
	if a.Status != "active" || a.DisabledAt != nil || a.EmailVerified {
		t.Fatalf("compte réactivé inattendu : %+v", a)
	}
	env.LoggedIn(member)
	if res := session.Do("GET", "/api/session", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("ancienne session après réactivation : %d, attendu 401", res.Status)
	}
	expectAccount(t, admin.Do("POST", accountPath(id, "/enable"), nil))
	if n := actionLogs(t, env, "account.enable"); n != 1 {
		t.Fatalf("%d lignes de journal account.enable, attendu 1", n)
	}
	expectProblem(t, admin.Do("POST", accountPath(999, "/disable"), nil), http.StatusNotFound, api.MsgAccountGone)
}

func TestAdminGuards(t *testing.T) {
	env := apitest.New(t)
	emailA, emailB := env.Seed(auth.RoleAdmin, true), env.Seed(auth.RoleAdmin, true)
	idA, idB := accountID(t, env, emailA), accountID(t, env, emailB)
	a, b := env.LoggedIn(emailA), env.LoggedIn(emailB)

	// B, désactivé par A, ne peut plus agir.
	expectAccount(t, a.Do("POST", accountPath(idB, "/disable"), nil))
	if res := b.Do("POST", accountPath(idA, "/disable"), nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("action d'un administrateur désactivé : %d, attendu 401", res.Status)
	}

	// Au niveau des requêtes (inatteignable par l'API, l'acteur étant lui-même
	// administrateur actif) : A, dernier administrateur actif, ne peut pas être
	// désactivé ; B, administrateur désactivé, peut être réactivé.
	n, err := env.Store.Write.DisableAccount(context.Background(), sqlc.DisableAccountParams{ID: idA})
	if err != nil || n != 0 {
		t.Fatalf("désactivation du dernier administrateur actif : %d ligne(s), %v", n, err)
	}
	expectAccount(t, a.Do("POST", accountPath(idB, "/enable"), nil))
}

// TestConcurrentMutualDisable vérifie SC-004 : deux administrateurs qui se
// désactivent mutuellement en même temps laissent un administrateur actif.
func TestConcurrentMutualDisable(t *testing.T) {
	for range 10 {
		env := apitest.New(t)
		emailA, emailB := env.Seed(auth.RoleAdmin, true), env.Seed(auth.RoleAdmin, true)
		idA, idB := accountID(t, env, emailA), accountID(t, env, emailB)
		a, b := env.LoggedIn(emailA), env.LoggedIn(emailB)

		var wg sync.WaitGroup
		statuses := make([]int, 2)
		for i, call := range []func() apitest.Response{
			func() apitest.Response { return a.Do("POST", accountPath(idB, "/disable"), nil) },
			func() apitest.Response { return b.Do("POST", accountPath(idA, "/disable"), nil) },
		} {
			wg.Go(func() { statuses[i] = call().Status })
		}
		wg.Wait()

		ok := 0
		for _, s := range statuses {
			if s == http.StatusOK {
				ok++
			}
		}
		var active int
		if err := env.Store.ReadDB.QueryRow("SELECT count(*) FROM accounts WHERE role = 'admin' AND disabled_at IS NULL").Scan(&active); err != nil {
			t.Fatal(err)
		}
		if ok > 1 || active < 1 {
			t.Fatalf("statuts %v : %d succès, %d administrateur(s) actif(s)", statuses, ok, active)
		}
	}
}
