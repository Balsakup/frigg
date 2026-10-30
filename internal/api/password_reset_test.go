package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/mail/mailtest"
)

func requestReset(t *testing.T, env *apitest.Env, email string) string {
	t.Helper()
	if res := env.Client().Do("POST", "/api/auth/password-reset", map[string]string{"email": email}); res.Status != http.StatusAccepted {
		t.Fatalf("demande : %d %s", res.Status, res.Body)
	}
	env.FlushMail()
	sent, ok := env.Mail.Last(email)
	if !ok {
		return ""
	}
	return mailtest.Token(sent.Msg)
}

func TestPasswordReset(t *testing.T) {
	env := apitest.New(t)
	email := env.Seed(auth.RoleMember, true)
	other := env.LoggedIn(email)

	first := requestReset(t, env, email)
	token := requestReset(t, env, email)
	sent, _ := env.Mail.Last(email)
	if token == "" || !strings.Contains(sent.Msg.Text, "/reinitialiser-mot-de-passe#") || !strings.Contains(sent.Msg.Text, "1 heure") {
		t.Fatalf("e-mail inattendu : %s", sent.Msg.Text)
	}
	if res := env.Client().Do("POST", "/api/auth/password-reset/confirm", map[string]string{"token": first, "password": "Nouveau2026"}); res.Status != http.StatusGone {
		t.Fatalf("ancien jeton : %d, attendu 410", res.Status)
	}

	res := env.Client().Do("POST", "/api/auth/password-reset/confirm", map[string]string{"token": token, "password": "nouveau"})
	var p apitest.Problem
	res.JSON(t, &p)
	if res.Status != http.StatusUnprocessableEntity || len(p.Errors) == 0 || p.Errors[0].Location != "body.password" {
		t.Fatalf("règle de mot de passe : %d %+v", res.Status, p)
	}

	if res := env.Client().Do("POST", "/api/auth/password-reset/confirm", map[string]string{"token": token, "password": "Nouveau2026"}); res.Status != http.StatusNoContent {
		t.Fatalf("confirmation : %d %s", res.Status, res.Body)
	}
	if res := other.Do("GET", "/api/session", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("ancienne session : %d, attendu 401", res.Status)
	}
	if res := env.Client().Do("POST", "/api/auth/login", map[string]string{"email": email, "password": "Nouveau2026"}); res.Status != http.StatusOK {
		t.Fatalf("connexion avec le nouveau mot de passe : %d", res.Status)
	}
	res = env.Client().Do("POST", "/api/auth/password-reset/confirm", map[string]string{"token": token, "password": "Autre2026"})
	res.JSON(t, &p)
	if res.Status != http.StatusGone || p.Detail != api.MsgLinkGone {
		t.Fatalf("jeton réutilisé : %d %q", res.Status, p.Detail)
	}
}

func TestPasswordResetUnknownEmailIsIndistinguishable(t *testing.T) {
	env := apitest.New(t)
	if token := requestReset(t, env, "inconnu@exemple.fr"); token != "" {
		t.Fatal("e-mail envoyé pour une adresse inconnue")
	}
	if len(env.Mail.Messages()) != 0 {
		t.Fatal("e-mail envoyé pour une adresse inconnue")
	}
}

func TestPasswordResetExpiredAndRateLimited(t *testing.T) {
	env := apitest.New(t)
	email := env.Seed(auth.RoleMember, true)
	token := requestReset(t, env, email)
	if _, err := env.Store.WriteDB.Exec("UPDATE one_time_tokens SET expires_at = '2000-01-01T00:00:00Z'"); err != nil {
		t.Fatal(err)
	}
	if res := env.Client().Do("POST", "/api/auth/password-reset/confirm", map[string]string{"token": token, "password": "Nouveau2026"}); res.Status != http.StatusGone {
		t.Fatalf("jeton expiré : %d, attendu 410", res.Status)
	}
	requestReset(t, env, email)
	requestReset(t, env, email)
	if res := env.Client().Do("POST", "/api/auth/password-reset", map[string]string{"email": email}); res.Status != http.StatusTooManyRequests {
		t.Fatalf("4e demande en une heure : %d, attendu 429", res.Status)
	}
}
