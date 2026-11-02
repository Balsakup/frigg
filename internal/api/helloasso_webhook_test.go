package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/helloasso"
	"github.com/balsakup/frigg/internal/helloasso/hatest"
)

func notificationPath() string {
	return "/api/webhooks/helloasso/" + helloasso.NotificationKey(hatest.ClientSecret)
}

func notificationBody(payment, membership int64) string {
	return fmt.Sprintf(`{"eventType":"Order","data":{"id":1,"items":[]},"metadata":{"frigg":{"payment":%d,"membership":%d}}}`, payment, membership)
}

func rawJSON(s string) apitest.Raw { return apitest.Raw(s) }

func TestNotificationRecordsPayment(t *testing.T) {
	f := newCardFixture(t)
	st := f.start(t)
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3000, 0)
	res := f.env.Client().Do("POST", notificationPath(), rawJSON(notificationBody(st.OnlinePaymentID, f.ms.ID)))
	if res.Status != http.StatusOK || res.Body != "" {
		t.Fatalf("notification : %d %q", res.Status, res.Body)
	}
	if got := f.mine(t); got.Payment.Status != "paid" || *got.Payment.PaidMethod != "card" {
		t.Fatalf("adhésion : %+v", got.Payment)
	}
	if res.Header.Get("Set-Cookie") != "" {
		t.Fatal("cookie posé par la notification")
	}
	// Rejouée : 200, sans constatation supplémentaire.
	f.env.Advance(10 * time.Second)
	if res := f.env.Client().Do("POST", notificationPath(), rawJSON(notificationBody(st.OnlinePaymentID, f.ms.ID))); res.Status != http.StatusOK {
		t.Fatalf("notification rejouée : %d", res.Status)
	}
	f.env.FlushMail()
	if f.paidMails() != 1 {
		t.Fatalf("e-mails : %d", f.paidMails())
	}
}

// Sans la clé valide, la réponse est identique quelle que soit l'entrée et
// rien n'est traité (constitution v1.7.0, principe II).
func TestNotificationWithoutValidKey(t *testing.T) {
	f := newCardFixture(t)
	st := f.start(t)
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3000, 0)
	calls := f.env.HelloAsso.TotalCalls()
	valid := notificationBody(st.OnlinePaymentID, f.ms.ID)
	huge := `{"pad":"` + strings.Repeat("x", 2<<20) + `"}`
	cases := []struct{ path, body string }{
		{"/api/webhooks/helloasso/" + helloasso.NotificationKey("autre-secret"), valid},
		{"/api/webhooks/helloasso/abc", valid},
		{"/api/webhooks/helloasso/" + strings.Repeat("z", 200), valid},
		{"/api/webhooks/helloasso/abc", "pas du json"},
		{"/api/webhooks/helloasso/abc", huge},
		{"/api/webhooks/helloasso/abc", ""},
	}
	var first *apitest.Response
	for _, c := range cases {
		res := f.env.Client().Do("POST", c.path, rawJSON(c.body))
		if res.Status != http.StatusOK || res.Body != "" || strings.Contains(res.Header.Get("Content-Type"), "problem") {
			t.Fatalf("%s : %d %q", c.path, res.Status, res.Body)
		}
		if first == nil {
			first = &res
		} else if res.Header.Get("Content-Type") != first.Header.Get("Content-Type") || res.Header.Get("Content-Length") != first.Header.Get("Content-Length") {
			t.Fatalf("réponses différentes : %v / %v", res.Header, first.Header)
		}
	}
	if f.env.HelloAsso.TotalCalls() != calls {
		t.Fatal("HelloAsso appelé sans clé valide")
	}
	if got := f.mine(t); got.Payment.Status != "due" {
		t.Fatalf("adhésion modifiée sans clé valide : %+v", got.Payment)
	}
}

func TestNotificationIgnoredWithValidKey(t *testing.T) {
	f := newCardFixture(t)
	f.start(t)
	for _, body := range []string{
		`{"eventType":"Form","data":{}}`,
		`{"eventType":"Order","data":{},"metadata":{"autre":1}}`,
		notificationBody(999999, f.ms.ID),
		`illisible`,
		`{"pad":"` + strings.Repeat("x", 2<<20) + `"}`,
	} {
		if res := f.env.Client().Do("POST", notificationPath(), rawJSON(body)); res.Status != http.StatusOK || res.Body != "" {
			t.Fatalf("notification ignorée : %d %q", res.Status, res.Body)
		}
	}
	if got := f.mine(t); got.Payment.Status != "due" {
		t.Fatalf("adhésion modifiée : %+v", got.Payment)
	}
}

// Avec la clé valide, une erreur passagère répond 503 pour que HelloAsso
// renvoie la notification.
func TestNotificationRetryWhenHelloAssoDown(t *testing.T) {
	f := newCardFixture(t)
	st := f.start(t)
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3000, 0)
	f.env.HelloAsso.Down(true)
	res := f.env.Client().Do("POST", notificationPath(), rawJSON(notificationBody(st.OnlinePaymentID, f.ms.ID)))
	if res.Status != http.StatusServiceUnavailable || res.Body != "" {
		t.Fatalf("HelloAsso injoignable : %d %q", res.Status, res.Body)
	}
	f.env.HelloAsso.Down(false)
	f.env.Advance(10 * time.Second)
	if res := f.env.Client().Do("POST", notificationPath(), rawJSON(notificationBody(st.OnlinePaymentID, f.ms.ID))); res.Status != http.StatusOK {
		t.Fatalf("renvoi : %d", res.Status)
	}
	if got := f.mine(t); got.Payment.Status != "paid" {
		t.Fatalf("adhésion après renvoi : %+v", got.Payment)
	}
	dash := f.admin.Do("GET", "/api/admin/memberships/dashboard", nil)
	if !strings.Contains(dash.Body, `"helloassoFailing":false`) {
		t.Fatalf("incident non effacé : %s", dash.Body)
	}
}

func TestNotificationRateLimitedAndCrossOrigin(t *testing.T) {
	f := newCardFixture(t)
	c := f.env.Client()
	if res := c.Do("POST", notificationPath(), rawJSON(`{}`), "Sec-Fetch-Site", "cross-site"); res.Status != http.StatusForbidden {
		t.Fatalf("requête d'origine croisée : %d", res.Status)
	}
	limited := false
	for range 70 {
		if res := c.Do("POST", "/api/webhooks/helloasso/abc", rawJSON(`{}`)); res.Status == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("débit non limité")
	}
}
