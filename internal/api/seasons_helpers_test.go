package api_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

// seasonEnv crée un environnement au 31 octobre 2026 et un administrateur
// connecté.
func seasonEnv(t *testing.T) (*apitest.Env, *apitest.Client) {
	t.Helper()
	env := apitest.New(t)
	env.SetToday("2026-10-31")
	return env, env.LoggedIn(env.Seed(auth.RoleAdmin, true))
}

func seasonPath(id int64, suffix string) string {
	return "/api/admin/seasons/" + strconv.FormatInt(id, 10) + suffix
}

// expectSeason vérifie le statut et décode la saison.
func expectSeason(t *testing.T, res apitest.Response, status int) api.Season {
	t.Helper()
	if res.Status != status {
		t.Fatalf("statut %d, attendu %d : %s", res.Status, status, res.Body)
	}
	var s api.Season
	res.JSON(t, &s)
	return s
}

// createSeason crée une saison et la renvoie.
func createSeason(t *testing.T, c *apitest.Client, start, end string) api.Season {
	t.Helper()
	return expectSeason(t, c.Do("POST", "/api/admin/seasons", map[string]string{"startsOn": start, "endsOn": end}), http.StatusCreated)
}

func listSeasons(t *testing.T, c *apitest.Client) api.SeasonList {
	t.Helper()
	res := c.Do("GET", "/api/admin/seasons", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("liste : %d %s", res.Status, res.Body)
	}
	var l api.SeasonList
	res.JSON(t, &l)
	return l
}

// expectFieldError vérifie une erreur 422 sur un champ.
func expectFieldError(t *testing.T, res apitest.Response, location, message string) {
	t.Helper()
	var p apitest.Problem
	res.JSON(t, &p)
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("statut %d, attendu 422 : %s", res.Status, res.Body)
	}
	for _, e := range p.Errors {
		if e.Location == location && e.Message == message {
			return
		}
	}
	t.Fatalf("erreur %q sur %s absente : %s", message, location, res.Body)
}

// seasonLogs compte les lignes de journal d'une action sur les saisons et
// vérifie leurs attributs.
func seasonLogs(t *testing.T, env *apitest.Env, action string) int {
	t.Helper()
	n := 0
	for _, line := range strings.Split(env.Logs.String(), "\n") {
		if !strings.Contains(line, `"action":"`+action+`"`) {
			continue
		}
		n++
		if !strings.Contains(line, `"actor_id"`) || !strings.Contains(line, `"season_id"`) {
			t.Errorf("journal incomplet : %s", line)
		}
	}
	return n
}
