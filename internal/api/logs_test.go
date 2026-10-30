package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/mail/mailtest"
)

// Les journaux ne doivent contenir ni mot de passe, ni jeton, ni cookie de
// session, ni contenu d'e-mail (principe VII).
func TestLogsContainNoSecrets(t *testing.T) {
	env := apitest.New(t)
	email := env.Seed(auth.RoleMember, true)
	c := env.Client()
	res := c.Do("POST", "/api/auth/login", map[string]string{"email": email, "password": apitest.Password})
	cookie := res.Header.Get("Set-Cookie")
	sessionToken := strings.TrimPrefix(strings.SplitN(cookie, ";", 2)[0], "frigg_session=")

	env.Client().Do("POST", "/api/auth/password-reset", map[string]string{"email": email})
	env.FlushMail()
	sent, _ := env.Mail.Last(email)
	token := mailtest.Token(sent.Msg)
	if res := env.Client().Do("POST", "/api/auth/password-reset/confirm", map[string]string{"token": token, "password": "Secret2026"}); res.Status != http.StatusNoContent {
		t.Fatalf("réinitialisation : %d", res.Status)
	}

	logs := env.Logs.String()
	if logs == "" {
		t.Fatal("aucun journal capturé")
	}
	for name, secret := range map[string]string{
		"mot de passe":         apitest.Password,
		"nouveau mot de passe": "Secret2026",
		"jeton":                token,
		"cookie de session":    sessionToken,
		"objet de l'e-mail":    sent.Msg.Subject,
	} {
		if secret != "" && strings.Contains(logs, secret) {
			t.Errorf("%s présent dans les journaux", name)
		}
	}
}
