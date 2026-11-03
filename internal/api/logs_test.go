package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/helloasso"
	"github.com/balsakup/frigg/internal/helloasso/hatest"
	"github.com/balsakup/frigg/internal/mail/mailtest"
	"time"
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

// TestCardPaymentLogs : un parcours complet de paiement par carte ne journalise
// ni e-mail, ni nom, ni secret, ni jeton, ni page de paiement HelloAsso
// (research.md 008 › R14).
func TestCardPaymentLogs(t *testing.T) {
	f := newCardFixture(t)
	if res := f.admin.Do("POST", "/api/admin/settings/payment/helloasso-test", nil); res.Status != http.StatusOK {
		t.Fatalf("test de connexion : %d", res.Status)
	}
	st := f.start(t)
	f.confirm(t, f.member, st.OnlinePaymentID, "return")
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3000, 300)
	f.env.Advance(10 * time.Second)
	f.env.Client().Do("POST", notificationPath(), rawJSON(notificationBody(st.OnlinePaymentID, f.ms.ID)))
	f.env.FlushMail()

	logs := f.env.Logs.String()
	for _, want := range []string{"paiement en ligne démarré", "paiement en ligne vérifié", "test de connexion HelloAsso"} {
		if !strings.Contains(logs, want) {
			t.Errorf("journal attendu absent : %q", want)
		}
	}
	for _, forbidden := range []string{
		f.email, "Léo", "Martin", hatest.ClientSecret, "tok-", "ref-", st.RedirectURL, "/pay/",
		helloasso.NotificationKey(hatest.ClientSecret),
	} {
		if strings.Contains(logs, forbidden) {
			t.Errorf("%q journalisé", forbidden)
		}
	}
}
