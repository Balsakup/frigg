package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/config"
)

var docsPaths = []string{"/api/openapi.json", "/api/schemas/Session.json", "/api/docs"}

func TestDocsDisabledByDefault(t *testing.T) {
	env := apitest.New(t)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	for _, p := range append(docsPaths, "/api/openapi", "/api/schemas") {
		if res := admin.Do("GET", p, nil); res.Status != http.StatusNotFound {
			t.Errorf("%s sans drapeau : %d, attendu 404", p, res.Status)
		}
	}
}

func TestDocsReservedToAdmins(t *testing.T) {
	env := apitest.New(t, func(c *config.Config) { c.APIDocs = true })
	member := env.LoggedIn(env.Seed(auth.RoleMember, true))
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	for _, p := range docsPaths {
		if res := env.Client().Do("GET", p, nil); res.Status != http.StatusUnauthorized {
			t.Errorf("%s anonyme : %d, attendu 401", p, res.Status)
		}
		if res := member.Do("GET", p, nil); res.Status != http.StatusForbidden {
			t.Errorf("%s membre : %d, attendu 403", p, res.Status)
		}
		if res := admin.Do("GET", p, nil); res.Status != http.StatusOK {
			t.Errorf("%s administrateur : %d, attendu 200", p, res.Status)
		}
	}
	res := admin.Do("GET", "/api/docs", nil)
	if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "https://unpkg.com") {
		t.Errorf("CSP de /api/docs : %q", csp)
	}
	res = admin.Do("GET", "/api/session", nil)
	if csp := res.Header.Get("Content-Security-Policy"); strings.Contains(csp, "unpkg") || !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("CSP hors /api/docs : %q", csp)
	}
	if !strings.Contains(env.Logs.String(), "documentation d'API activée") {
		t.Error("avertissement de démarrage absent")
	}
}
