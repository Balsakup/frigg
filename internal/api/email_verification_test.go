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

func resendVerification(t *testing.T, env *apitest.Env, c *apitest.Client, email string) string {
	t.Helper()
	if res := c.Do("POST", "/api/me/email-verification", nil); res.Status != http.StatusAccepted {
		t.Fatalf("renvoi : %d %s", res.Status, res.Body)
	}
	env.FlushMail()
	sent, _ := env.Mail.Last(email)
	if !strings.Contains(sent.Msg.Text, "/confirmer-adresse#") || !strings.Contains(sent.Msg.Text, "48 heures") {
		t.Fatalf("e-mail inattendu : %s", sent.Msg.Text)
	}
	return mailtest.Token(sent.Msg)
}

func TestEmailVerification(t *testing.T) {
	env := apitest.New(t)
	email := env.Seed(auth.RoleMember, false)
	c := env.LoggedIn(email)

	first := resendVerification(t, env, c, email)
	token := resendVerification(t, env, c, email)
	if res := env.Client().Do("POST", "/api/auth/email-verification/confirm", map[string]string{"token": first}); res.Status != http.StatusGone {
		t.Fatalf("ancien lien : %d, attendu 410", res.Status)
	}
	// La confirmation fonctionne sans être connecté.
	if res := env.Client().Do("POST", "/api/auth/email-verification/confirm", map[string]string{"token": token}); res.Status != http.StatusNoContent {
		t.Fatalf("confirmation : %d %s", res.Status, res.Body)
	}
	var s api.Session
	c.Do("GET", "/api/session", nil).JSON(t, &s)
	if !s.Account.EmailVerified {
		t.Fatal("adresse non confirmée")
	}
	if res := env.Client().Do("POST", "/api/auth/email-verification/confirm", map[string]string{"token": token}); res.Status != http.StatusGone {
		t.Fatalf("lien réutilisé : %d, attendu 410", res.Status)
	}
	res := c.Do("POST", "/api/me/email-verification", nil)
	var p apitest.Problem
	res.JSON(t, &p)
	if res.Status != http.StatusConflict || p.Detail != api.MsgAlreadyVerified {
		t.Fatalf("adresse déjà confirmée : %d %q", res.Status, p.Detail)
	}
}

func TestEmailVerificationExpiredAndRateLimited(t *testing.T) {
	env := apitest.New(t)
	email := env.Seed(auth.RoleMember, false)
	c := env.LoggedIn(email)
	token := resendVerification(t, env, c, email)
	if _, err := env.Store.WriteDB.Exec("UPDATE one_time_tokens SET expires_at = '2000-01-01T00:00:00Z'"); err != nil {
		t.Fatal(err)
	}
	if res := env.Client().Do("POST", "/api/auth/email-verification/confirm", map[string]string{"token": token}); res.Status != http.StatusGone {
		t.Fatalf("lien expiré : %d, attendu 410", res.Status)
	}
	resendVerification(t, env, c, email)
	resendVerification(t, env, c, email)
	if res := c.Do("POST", "/api/me/email-verification", nil); res.Status != http.StatusTooManyRequests {
		t.Fatalf("4e renvoi en une heure : %d, attendu 429", res.Status)
	}
}
