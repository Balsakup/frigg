package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

func TestLogin(t *testing.T) {
	env := apitest.New(t)
	email := env.Seed(auth.RoleMember, true)
	c := env.Client()

	res := c.Do("POST", "/api/auth/login", map[string]string{"email": "  " + strings.ToUpper(email) + " ", "password": apitest.Password})
	if res.Status != http.StatusOK {
		t.Fatalf("connexion : %d %s", res.Status, res.Body)
	}
	if !strings.Contains(res.Header.Get("Set-Cookie"), "frigg_session=") {
		t.Fatal("cookie de session absent")
	}
	var s api.Session
	res.JSON(t, &s)
	if s.Account.Email != email || s.Account.Role != auth.RoleMember || !s.Account.EmailVerified || s.Association.Name != "Mon association" {
		t.Fatalf("session inattendue : %+v", s)
	}
	a, _ := env.Store.Read.GetAccountByEmail(t.Context(), email)
	if !a.LastLoginAt.Valid {
		t.Fatal("dernière connexion non enregistrée")
	}
	if res := c.Do("GET", "/api/session", nil); res.Status != http.StatusOK {
		t.Fatalf("session après connexion : %d", res.Status)
	}
}

func TestLoginFailuresAreIndistinguishable(t *testing.T) {
	env := apitest.New(t)
	email := env.Seed(auth.RoleMember, true)
	c := env.Client()

	wrong := c.Do("POST", "/api/auth/login", map[string]string{"email": email, "password": "Mauvais1"})
	unknown := c.Do("POST", "/api/auth/login", map[string]string{"email": "inconnu@exemple.fr", "password": "Mauvais1"})
	var p1, p2 apitest.Problem
	wrong.JSON(t, &p1)
	unknown.JSON(t, &p2)
	if wrong.Status != http.StatusUnauthorized || unknown.Status != http.StatusUnauthorized {
		t.Fatalf("statuts %d et %d, attendu 401", wrong.Status, unknown.Status)
	}
	if p1.Detail != api.MsgBadCredentials || p2.Detail != api.MsgBadCredentials {
		t.Fatalf("messages différents : %q / %q", p1.Detail, p2.Detail)
	}
}

func TestLoginRateLimit(t *testing.T) {
	env := apitest.New(t)
	email := env.Seed(auth.RoleMember, true)
	c := env.Client()
	for i := range 5 {
		if res := c.Do("POST", "/api/auth/login", map[string]string{"email": email, "password": "Mauvais1"}); res.Status != http.StatusUnauthorized {
			t.Fatalf("tentative %d : %d", i+1, res.Status)
		}
	}
	res := c.Do("POST", "/api/auth/login", map[string]string{"email": email, "password": apitest.Password})
	var p apitest.Problem
	res.JSON(t, &p)
	if res.Status != http.StatusTooManyRequests || p.Detail != api.MsgTooManyAttempts || res.Header.Get("Retry-After") == "" {
		t.Fatalf("6e tentative : %d %+v", res.Status, p)
	}
}

func TestLogout(t *testing.T) {
	env := apitest.New(t)
	c := env.LoggedIn(env.Seed(auth.RoleMember, true))
	if res := c.Do("POST", "/api/auth/logout", nil); res.Status != http.StatusNoContent {
		t.Fatalf("déconnexion : %d %s", res.Status, res.Body)
	}
	if res := c.Do("GET", "/api/session", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("session après déconnexion : %d", res.Status)
	}
}

func TestCrossOriginProtection(t *testing.T) {
	env := apitest.New(t)
	email := env.Seed(auth.RoleMember, true)
	body := map[string]string{"email": email, "password": apitest.Password}

	if res := env.Client().Do("POST", "/api/auth/login", body, "Sec-Fetch-Site", "cross-site"); res.Status != http.StatusForbidden {
		t.Errorf("Sec-Fetch-Site cross-site : %d, attendu 403", res.Status)
	}
	if res := env.Client().Do("POST", "/api/auth/login", body, "Origin", "https://autre.exemple"); res.Status != http.StatusForbidden {
		t.Errorf("Origin étrangère : %d, attendu 403", res.Status)
	}
	if res := env.Client().Do("POST", "/api/auth/login", body, "Sec-Fetch-Site", "same-origin"); res.Status != http.StatusOK {
		t.Errorf("Sec-Fetch-Site same-origin : %d, attendu 200", res.Status)
	}
}
