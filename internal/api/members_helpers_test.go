package api_test

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Date de référence des tests de la fonctionnalité 005.
const membersToday = "2026-11-01"

// memberBody renvoie un corps de fiche minimal valide, complété par extra.
func memberBody(first, last, birth string, extra ...any) map[string]any {
	b := map[string]any{"firstName": first, "lastName": last, "birthDate": birth}
	for i := 0; i+1 < len(extra); i += 2 {
		b[extra[i].(string)] = extra[i+1]
	}
	return b
}

func myMemberPath(id int64) string { return "/api/me/members/" + strconv.FormatInt(id, 10) }

func adminMemberPath(id int64) string { return "/api/admin/members/" + strconv.FormatInt(id, 10) }

// expectMember vérifie le statut et décode la fiche.
func expectMember(t *testing.T, res apitest.Response, status int) api.AdminMember {
	t.Helper()
	if res.Status != status {
		t.Fatalf("statut %d, attendu %d : %s", res.Status, status, res.Body)
	}
	var m api.AdminMember
	res.JSON(t, &m)
	return m
}

// createMine crée une fiche dans l'espace membre du client c.
func createMine(t *testing.T, c *apitest.Client, body map[string]any) api.Member {
	t.Helper()
	return expectMember(t, c.Do("POST", "/api/me/members", body), http.StatusCreated).Member
}

// myMembers renvoie la liste « Mes membres » du client c.
func myMembers(t *testing.T, c *apitest.Client) api.MyMemberList {
	t.Helper()
	res := c.Do("GET", "/api/me/members", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("liste : %d %s", res.Status, res.Body)
	}
	var l api.MyMemberList
	res.JSON(t, &l)
	return l
}

// insertOrphan insère directement une fiche sans compte.
func insertOrphan(t *testing.T, env *apitest.Env, first, last, birth string) int64 {
	t.Helper()
	id, err := env.Store.Write.CreateMember(context.Background(), sqlc.CreateMemberParams{
		FirstName: first, LastName: last, BirthDate: birth,
		Now: "2026-11-01T10:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// insertForAccount insère directement une fiche rattachée à un compte.
func insertForAccount(t *testing.T, env *apitest.Env, accountID int64, first, last, birth string) int64 {
	t.Helper()
	m, err := env.Store.Write.CreateMemberForAccount(context.Background(), sqlc.CreateMemberForAccountParams{
		AccountID: sql.NullInt64{Int64: accountID, Valid: true},
		FirstName: first, LastName: last, BirthDate: birth,
		Now: "2026-11-01T10:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	return m.ID
}

// memberLogs compte les lignes de journal d'une action sur les fiches et
// vérifie qu'aucune ne contient les données personnelles forbidden.
func memberLogs(t *testing.T, env *apitest.Env, action, space string, forbidden ...string) int {
	t.Helper()
	n := 0
	for _, line := range strings.Split(env.Logs.String(), "\n") {
		if !strings.Contains(line, `"action":"`+action+`"`) || !strings.Contains(line, `"space":"`+space+`"`) {
			continue
		}
		n++
		for _, f := range forbidden {
			if strings.Contains(line, f) {
				t.Errorf("donnée personnelle %q dans le journal : %s", f, line)
			}
		}
		if !strings.Contains(line, `"actor_id"`) || !strings.Contains(line, `"member_id"`) {
			t.Errorf("journal incomplet : %s", line)
		}
	}
	return n
}
