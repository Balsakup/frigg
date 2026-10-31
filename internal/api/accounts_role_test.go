package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

func TestSetAccountRole(t *testing.T) {
	env := apitest.New(t)
	adminEmail := env.Seed(auth.RoleAdmin, true)
	admin := env.LoggedIn(adminEmail)
	member := env.Seed(auth.RoleMember, true)
	memberSession := env.LoggedIn(member)
	other := env.Seed(auth.RoleAdmin, true)
	otherSession := env.LoggedIn(other)

	// Promotion : effective à la requête suivante, sans déconnexion (FR-009).
	if res := memberSession.Do("GET", "/api/admin/accounts", nil); res.Status != http.StatusForbidden {
		t.Fatalf("membre avant promotion : %d", res.Status)
	}
	a := expectAccount(t, admin.Do("PUT", accountPath(accountID(t, env, member), "/role"), map[string]string{"role": auth.RoleAdmin}))
	if a.Role != auth.RoleAdmin {
		t.Fatalf("rôle %q après promotion", a.Role)
	}
	if res := memberSession.Do("GET", "/api/admin/accounts", nil); res.Status != http.StatusOK {
		t.Fatalf("après promotion : %d, attendu 200", res.Status)
	}

	// Rétrogradation : droits retirés dès la requête suivante.
	expectAccount(t, admin.Do("PUT", accountPath(accountID(t, env, other), "/role"), map[string]string{"role": auth.RoleMember}))
	if res := otherSession.Do("GET", "/api/admin/accounts", nil); res.Status != http.StatusForbidden {
		t.Fatalf("après rétrogradation : %d, attendu 403", res.Status)
	}
	var s api.Session
	res := otherSession.Do("GET", "/api/session", nil)
	res.JSON(t, &s)
	if res.Status != http.StatusOK || s.Account.Role != auth.RoleMember {
		t.Fatalf("session après rétrogradation : %d %+v", res.Status, s.Account)
	}

	// Même rôle : rien ne change, rien n'est journalisé.
	expectAccount(t, admin.Do("PUT", accountPath(accountID(t, env, other), "/role"), map[string]string{"role": auth.RoleMember}))
	if n := actionLogs(t, env, "account.role_change"); n != 2 {
		t.Fatalf("%d lignes de journal account.role_change, attendu 2", n)
	}

	if res := admin.Do("PUT", accountPath(accountID(t, env, other), "/role"), map[string]string{"role": "owner"}); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("rôle inconnu : %d, attendu 422", res.Status)
	}
	expectProblem(t, admin.Do("PUT", accountPath(accountID(t, env, adminEmail), "/role"), map[string]string{"role": auth.RoleMember}),
		http.StatusConflict, api.MsgSelfAction)
	expectProblem(t, admin.Do("PUT", accountPath(999, "/role"), map[string]string{"role": auth.RoleMember}), http.StatusNotFound, api.MsgAccountGone)
}

// TestSetAccountRoleLastAdmin vérifie le garde-fou au niveau de la requête :
// inatteignable par l'API, l'acteur étant lui-même administrateur actif.
func TestSetAccountRoleLastAdmin(t *testing.T) {
	env := apitest.New(t)
	ctx := context.Background()
	only := accountID(t, env, env.Seed(auth.RoleAdmin, true))
	disabled := accountID(t, env, env.Seed(auth.RoleAdmin, true))
	if _, err := env.Store.WriteDB.Exec("UPDATE accounts SET disabled_at = '2026-10-31T10:00:00Z' WHERE id = ?", disabled); err != nil {
		t.Fatal(err)
	}

	if n, err := env.Store.Write.SetAccountRole(ctx, sqlc.SetAccountRoleParams{ID: only, Role: auth.RoleMember}); err != nil || n != 0 {
		t.Fatalf("rétrogradation du dernier administrateur actif : %d ligne(s), %v", n, err)
	}
	if n, err := env.Store.Write.SetAccountRole(ctx, sqlc.SetAccountRoleParams{ID: disabled, Role: auth.RoleMember}); err != nil || n != 1 {
		t.Fatalf("rétrogradation d'un administrateur désactivé : %d ligne(s), %v", n, err)
	}
}
