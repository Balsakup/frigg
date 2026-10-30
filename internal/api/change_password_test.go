package api_test

import (
	"net/http"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

func TestChangePassword(t *testing.T) {
	env := apitest.New(t)
	email := env.Seed(auth.RoleMember, true)
	c := env.LoggedIn(email)
	other := env.LoggedIn(email)

	res := c.Do("PUT", "/api/me/password", map[string]string{"currentPassword": "Faux2026", "newPassword": "Nouveau2026"})
	var p apitest.Problem
	res.JSON(t, &p)
	if res.Status != http.StatusUnprocessableEntity || len(p.Errors) != 1 || p.Errors[0].Location != "body.currentPassword" || p.Errors[0].Message != api.MsgWrongCurrentPassword {
		t.Fatalf("mot de passe actuel faux : %d %+v", res.Status, p)
	}
	res = c.Do("PUT", "/api/me/password", map[string]string{"currentPassword": apitest.Password, "newPassword": "nouveau"})
	res.JSON(t, &p)
	if res.Status != http.StatusUnprocessableEntity || p.Errors[0].Location != "body.newPassword" {
		t.Fatalf("règle non respectée : %d %+v", res.Status, p)
	}

	if res := c.Do("PUT", "/api/me/password", map[string]string{"currentPassword": apitest.Password, "newPassword": "Nouveau2026"}); res.Status != http.StatusNoContent {
		t.Fatalf("changement : %d %s", res.Status, res.Body)
	}
	if res := c.Do("GET", "/api/session", nil); res.Status != http.StatusOK {
		t.Fatalf("session courante : %d, attendu 200", res.Status)
	}
	if res := other.Do("GET", "/api/session", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("autre session : %d, attendu 401", res.Status)
	}
	if res := env.Client().Do("POST", "/api/auth/login", map[string]string{"email": email, "password": "Nouveau2026"}); res.Status != http.StatusOK {
		t.Fatalf("connexion avec le nouveau mot de passe : %d", res.Status)
	}
}
