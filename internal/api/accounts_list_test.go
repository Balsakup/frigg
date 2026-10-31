package api_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// insertAccount crée un compte sans mot de passe utilisable (liste et recherche).
func insertAccount(t testing.TB, q *sqlc.Queries, email, role string) int64 {
	t.Helper()
	a, err := q.CreateAccount(context.Background(), sqlc.CreateAccountParams{
		Email: email, PasswordHash: "x", Role: role, CreatedAt: "2026-10-31T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	return a.ID
}

func listAccounts(t *testing.T, c *apitest.Client, query string) api.AdminAccountPage {
	t.Helper()
	res := c.Do("GET", "/api/admin/accounts"+query, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("liste %q : %d %s", query, res.Status, res.Body)
	}
	var page api.AdminAccountPage
	res.JSON(t, &page)
	return page
}

func emails(page api.AdminAccountPage) string {
	var out []string
	for _, a := range page.Items {
		out = append(out, a.Email)
	}
	return strings.Join(out, ",")
}

func TestListAccounts(t *testing.T) {
	env := apitest.New(t)
	adminEmail := env.Seed(auth.RoleAdmin, true)
	admin := env.LoggedIn(adminEmail)
	insertAccount(t, env.Store.Write, "jean.dupont@exemple.fr", auth.RoleMember)
	insertAccount(t, env.Store.Write, "hélène.lefèvre@asso.fr", auth.RoleAdmin)
	disabled := insertAccount(t, env.Store.Write, "marc.dupré@exemple.fr", auth.RoleMember)
	if _, err := env.Store.WriteDB.Exec("UPDATE accounts SET disabled_at = '2026-10-31T10:00:00Z' WHERE id = ?", disabled); err != nil {
		t.Fatal(err)
	}

	page := listAccounts(t, admin, "")
	if page.Total != 4 || page.Page != 1 || page.PageSize != 25 {
		t.Fatalf("page inattendue : total %d, page %d, taille %d", page.Total, page.Page, page.PageSize)
	}
	if got := emails(page); got != adminEmail+",hélène.lefèvre@asso.fr,jean.dupont@exemple.fr,marc.dupré@exemple.fr" {
		t.Fatalf("tri inattendu : %s", got)
	}
	if !page.Items[0].IsSelf || page.Items[1].IsSelf {
		t.Fatalf("isSelf inattendu : %+v", page.Items[:2])
	}
	marc := page.Items[3]
	if marc.Status != "disabled" || marc.DisabledAt == nil || marc.LastLoginAt != nil || marc.EmailVerified {
		t.Fatalf("compte désactivé inattendu : %+v", marc)
	}

	for query, want := range map[string]string{
		"?q=pont":                          "jean.dupont@exemple.fr",
		"?q=LEF%C3%88":                     "hélène.lefèvre@asso.fr",
		"?q=exemple+dup":                   "jean.dupont@exemple.fr,marc.dupré@exemple.fr",
		"?q=de+pont":                       "jean.dupont@exemple.fr",
		"?role=admin":                      adminEmail + ",hélène.lefèvre@asso.fr",
		"?status=disabled":                 "marc.dupré@exemple.fr",
		"?status=active&q=exemple":         adminEmail + ",jean.dupont@exemple.fr",
		"?q=dup&status=active&role=member": "jean.dupont@exemple.fr",
		"?q=aucun-resultat":                "",
		"?pageSize=2&page=2":               "jean.dupont@exemple.fr,marc.dupré@exemple.fr",
	} {
		if got := emails(listAccounts(t, admin, query)); got != want {
			t.Errorf("%s : %q, attendu %q", query, got, want)
		}
	}
	if p := listAccounts(t, admin, "?q=exemple+dup"); p.Total != 2 {
		t.Errorf("total filtré %d, attendu 2", p.Total)
	}
	if p := listAccounts(t, admin, "?page=9"); len(p.Items) != 0 || p.Total != 4 {
		t.Errorf("page au-delà de la dernière : %d éléments, total %d", len(p.Items), p.Total)
	}

	res := admin.Do("GET", "/api/admin/accounts?q=du", nil)
	var p apitest.Problem
	res.JSON(t, &p)
	if res.Status != http.StatusUnprocessableEntity || len(p.Errors) != 1 || p.Errors[0].Location != "query.q" || p.Errors[0].Message != api.MsgSearchTooShort {
		t.Errorf("recherche trop courte : %d %s", res.Status, res.Body)
	}
	for _, query := range []string{"?pageSize=101", "?page=0", "?role=owner", "?status=x"} {
		if res := admin.Do("GET", "/api/admin/accounts"+query, nil); res.Status != http.StatusUnprocessableEntity {
			t.Errorf("%s : statut %d, attendu 422", query, res.Status)
		}
	}
}

func TestGetAccount(t *testing.T) {
	env := apitest.New(t)
	adminEmail := env.Seed(auth.RoleAdmin, true)
	admin := env.LoggedIn(adminEmail)
	member := env.Seed(auth.RoleMember, false)

	a := expectAccount(t, admin.Do("GET", accountPath(accountID(t, env, member), ""), nil))
	if a.Email != member || a.Role != auth.RoleMember || a.Status != "active" || a.EmailVerified || a.IsSelf {
		t.Fatalf("fiche inattendue : %+v", a)
	}
	if self := expectAccount(t, admin.Do("GET", accountPath(accountID(t, env, adminEmail), ""), nil)); !self.IsSelf || self.LastLoginAt == nil {
		t.Fatalf("propre fiche inattendue : %+v", self)
	}
	expectProblem(t, admin.Do("GET", accountPath(999, ""), nil), http.StatusNotFound, api.MsgAccountGone)
}

// TestListAccountsScale vérifie SC-001 : recherche et pagination restent
// immédiates avec 2 000 comptes.
func TestListAccountsScale(t *testing.T) {
	if testing.Short() {
		t.Skip("test de volume")
	}
	env := apitest.New(t)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	err := env.Store.WriteTx(context.Background(), func(q *sqlc.Queries) error {
		for i := range 2000 {
			insertAccount(t, q, fmt.Sprintf("prenom-%04d.nom-%04d@exemple.fr", i, i), auth.RoleMember)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for query, check := range map[string]func(api.AdminAccountPage) bool{
		"?q=nom-1234": func(p api.AdminAccountPage) bool { return p.Total == 1 && len(p.Items) == 1 },
		"?page=80":    func(p api.AdminAccountPage) bool { return p.Total == 2001 && len(p.Items) == 25 },
	} {
		start := time.Now()
		page := listAccounts(t, admin, query)
		if d := time.Since(start); d > 200*time.Millisecond {
			t.Errorf("%s : %v, attendu moins de 200 ms", query, d)
		}
		if !check(page) {
			t.Errorf("%s : résultat inattendu (total %d, %d éléments)", query, page.Total, len(page.Items))
		}
	}
}
