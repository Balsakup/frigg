package api_test

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
)

// accountID renvoie l'identifiant du compte email.
func accountID(t *testing.T, env *apitest.Env, email string) int64 {
	t.Helper()
	a, err := env.Store.Read.GetAccountByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("compte %s : %v", email, err)
	}
	return a.ID
}

// accountPath renvoie le chemin d'API du compte, suivi de suffix.
func accountPath(id int64, suffix string) string {
	return "/api/admin/accounts/" + strconv.FormatInt(id, 10) + suffix
}

// expectAccount vérifie le statut 200 et décode le compte.
func expectAccount(t *testing.T, res apitest.Response) api.AdminAccount {
	t.Helper()
	if res.Status != http.StatusOK {
		t.Fatalf("statut %d, attendu 200 : %s", res.Status, res.Body)
	}
	var a api.AdminAccount
	res.JSON(t, &a)
	return a
}

// expectProblem vérifie le statut et le détail d'une erreur.
func expectProblem(t *testing.T, res apitest.Response, status int, detail string) {
	t.Helper()
	var p apitest.Problem
	res.JSON(t, &p)
	if res.Status != status || p.Detail != detail {
		t.Fatalf("réponse %d %q, attendu %d %q", res.Status, p.Detail, status, detail)
	}
}

// actionLogs compte les lignes de journal d'une action d'administration et
// vérifie qu'aucune ne contient d'adresse électronique.
func actionLogs(t *testing.T, env *apitest.Env, action string) int {
	t.Helper()
	n := 0
	for _, line := range strings.Split(env.Logs.String(), "\n") {
		if !strings.Contains(line, `"action":"`+action+`"`) {
			continue
		}
		n++
		if strings.Contains(line, "@") {
			t.Errorf("adresse électronique dans le journal : %s", line)
		}
		if !strings.Contains(line, `"actor_id"`) || !strings.Contains(line, `"target_id"`) {
			t.Errorf("journal incomplet : %s", line)
		}
	}
	return n
}
