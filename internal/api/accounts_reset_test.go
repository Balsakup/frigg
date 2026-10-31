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

func TestSendAccountPasswordReset(t *testing.T) {
	env := apitest.New(t)
	adminEmail := env.Seed(auth.RoleAdmin, true)
	admin := env.LoggedIn(adminEmail)
	member := env.Seed(auth.RoleMember, true)
	id := accountID(t, env, member)

	send := func() string {
		t.Helper()
		if res := admin.Do("POST", accountPath(id, "/password-reset"), nil); res.Status != http.StatusAccepted {
			t.Fatalf("envoi du lien : %d %s", res.Status, res.Body)
		}
		env.FlushMail()
		sent, ok := env.Mail.Last(member)
		if !ok || !strings.Contains(sent.Msg.Subject, "Réinitialisation") {
			t.Fatalf("e-mail de réinitialisation absent : %+v", sent)
		}
		return mailtest.Token(sent.Msg)
	}
	confirm := func(token string) int {
		return env.Client().Do("POST", "/api/auth/password-reset/confirm", map[string]string{"token": token, "password": "Nouveau2026"}).Status
	}

	first := send()
	second := send()
	if s := confirm(first); s != http.StatusGone {
		t.Fatalf("premier lien après un second envoi : %d, attendu 410", s)
	}
	if s := confirm(second); s != http.StatusNoContent {
		t.Fatalf("second lien : %d, attendu 204", s)
	}
	if s := confirm(second); s != http.StatusGone {
		t.Fatalf("lien réutilisé : %d, attendu 410", s)
	}
	if n := actionLogs(t, env, "account.password_reset_sent"); n != 2 {
		t.Fatalf("%d lignes de journal, attendu 2", n)
	}

	expectProblem(t, admin.Do("POST", accountPath(accountID(t, env, adminEmail), "/password-reset"), nil), http.StatusConflict, api.MsgSelfAction)
	expectProblem(t, admin.Do("POST", accountPath(999, "/password-reset"), nil), http.StatusNotFound, api.MsgAccountGone)
	expectAccount(t, admin.Do("POST", accountPath(id, "/disable"), nil))
	expectProblem(t, admin.Do("POST", accountPath(id, "/password-reset"), nil), http.StatusConflict, api.MsgAccountDisabled)
}

func TestSendAccountPasswordResetRateLimit(t *testing.T) {
	env := apitest.New(t)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	path := accountPath(accountID(t, env, env.Seed(auth.RoleMember, true)), "/password-reset")
	for i := range 30 {
		if res := admin.Do("POST", path, nil); res.Status != http.StatusAccepted {
			t.Fatalf("envoi %d : %d", i+1, res.Status)
		}
	}
	if res := admin.Do("POST", path, nil); res.Status != http.StatusTooManyRequests {
		t.Fatalf("31ᵉ envoi : %d, attendu 429", res.Status)
	}
}
