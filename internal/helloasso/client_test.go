package helloasso_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/helloasso"
	"github.com/balsakup/frigg/internal/helloasso/hatest"
)

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

func newClient(f *hatest.Fake, c *clock) *helloasso.Client {
	return helloasso.New(helloasso.Config{
		ClientID: hatest.ClientID, ClientSecret: hatest.ClientSecret, Organization: hatest.Organization,
		Environment: helloasso.Sandbox, APIURL: f.URL,
	}, nil, c.now)
}

func checkout(id int64) helloasso.CheckoutRequest {
	return helloasso.CheckoutRequest{
		AmountCents: 4000, ItemName: "Adhésion 2026-2027 — Jeune — ADH-2627-0042",
		Payer:     &helloasso.Payer{Email: "claire@example.org"},
		ReturnURL: "https://frigg.example/adhesions/42?carte=retour&tentative=7",
		ErrorURL:  "https://frigg.example/adhesions/42?carte=erreur&tentative=7",
		BackURL:   "https://frigg.example/adhesions/42?carte=annule&tentative=7",
		Metadata:  helloasso.Metadata{Payment: id, Membership: 42, Reference: "ADH-2627-0042"},
	}
}

func TestTokenReusedAndRefreshed(t *testing.T) {
	f := hatest.New(t)
	c := &clock{t: time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)}
	cl := newClient(f, c)
	ctx := context.Background()
	for range 3 {
		if _, err := cl.Organization(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if f.Tokens() != 1 {
		t.Fatalf("jetons émis : %d, attendu 1", f.Tokens())
	}
	c.add(1799*time.Second - 30*time.Second) // dans la marge de 60 s
	if _, err := cl.Organization(ctx); err != nil {
		t.Fatal(err)
	}
	if f.Tokens() != 2 {
		t.Fatalf("jeton non renouvelé avant expiration : %d", f.Tokens())
	}
}

func TestUnauthorizedRetriedOnce(t *testing.T) {
	f := hatest.New(t)
	cl := newClient(f, &clock{t: time.Now()})
	f.UnauthorizedOnce()
	name, err := cl.Organization(context.Background())
	if err != nil || name != hatest.OrgName {
		t.Fatalf("Organization = %q, %v", name, err)
	}
	if f.Tokens() != 2 {
		t.Fatalf("jetons : %d, attendu 2 (renouvellement après 401)", f.Tokens())
	}
}

func TestTypedErrors(t *testing.T) {
	ctx := context.Background()
	f := hatest.New(t)
	f.RejectCredentials()
	_, err := newClient(f, &clock{t: time.Now()}).Organization(ctx)
	if !errors.Is(err, helloasso.ErrUnauthorized) || helloasso.ErrorCode(err) != "unauthorized" {
		t.Fatalf("identifiants refusés : %v", err)
	}

	f2 := hatest.New(t)
	other := helloasso.New(helloasso.Config{ClientID: hatest.ClientID, ClientSecret: hatest.ClientSecret,
		Organization: "inconnue", Environment: helloasso.Sandbox, APIURL: f2.URL}, nil, nil)
	if _, err := other.Organization(ctx); helloasso.ErrorCode(err) != "not_found" {
		t.Fatalf("association inconnue : %v", err)
	}

	f3 := hatest.New(t)
	f3.Down(true)
	if _, err := newClient(f3, &clock{t: time.Now()}).Organization(ctx); helloasso.ErrorCode(err) != "unavailable" {
		t.Fatalf("service injoignable : %v", err)
	}

	for _, e := range []error{err} {
		if e != nil && strings.Contains(e.Error(), hatest.ClientSecret) {
			t.Fatal("le secret apparaît dans une erreur")
		}
	}
}

func TestConnectionTestBoundedTo10s(t *testing.T) {
	if testing.Short() {
		t.Skip("test lent")
	}
	f := hatest.New(t)
	f.Delay = 6 * time.Second // jeton + appel = 12 s sans borne globale
	start := time.Now()
	_, err := newClient(f, &clock{t: time.Now()}).Organization(context.Background())
	if helloasso.ErrorCode(err) != "unavailable" {
		t.Fatalf("erreur %v, attendu unavailable", err)
	}
	if d := time.Since(start); d > 11*time.Second {
		t.Fatalf("test de connexion trop long : %s", d)
	}
}

func TestCheckoutLifecycle(t *testing.T) {
	ctx := context.Background()
	f := hatest.New(t)
	cl := newClient(f, &clock{t: time.Now()})
	in, err := cl.CreateCheckout(ctx, checkout(7))
	if err != nil {
		t.Fatal(err)
	}
	sent := f.Intent(in.ID).Body
	if sent["totalAmount"].(float64) != 4000 || sent["initialAmount"].(float64) != 4000 || sent["containsDonation"] != false {
		t.Fatalf("corps envoyé : %v", sent)
	}
	if !strings.HasPrefix(in.RedirectURL, f.URL) {
		t.Fatalf("redirectUrl = %q", in.RedirectURL)
	}

	ch, err := cl.GetCheckout(ctx, in.ID)
	if err != nil || ch.Order != nil || ch.Metadata == nil || ch.Metadata.Payment != 7 || ch.Metadata.Membership != 42 {
		t.Fatalf("intention non payée : %+v, %v", ch, err)
	}

	f.Authorize(in.ID, 4300, 300)
	ch, err = cl.GetCheckout(ctx, in.ID)
	if err != nil || ch.Order == nil {
		t.Fatalf("intention payée : %+v, %v", ch, err)
	}
	p, ok := ch.Order.AuthorizedPayment()
	if !ok || p.ReceivedCents() != 4000 || p.Date.IsZero() {
		t.Fatalf("paiement : %+v", p)
	}
}

func TestUnexpectedRedirectHostRejected(t *testing.T) {
	f := hatest.New(t)
	f.RedirectHost = "https://pirate.example"
	_, err := newClient(f, &clock{t: time.Now()}).CreateCheckout(context.Background(), checkout(1))
	if !errors.Is(err, helloasso.ErrRedirectHost) {
		t.Fatalf("erreur %v, attendu ErrRedirectHost", err)
	}
}

func TestItemNameTruncated(t *testing.T) {
	long := strings.Repeat("é", 300)
	if n := len([]rune(helloasso.TruncateItemName(long))); n != helloasso.MaxItemName {
		t.Fatalf("intitulé tronqué à %d caractères", n)
	}
}

func TestNotificationKey(t *testing.T) {
	k := helloasso.NotificationKey("s3cret")
	if len(k) != 32 || k != helloasso.NotificationKey("s3cret") || k == helloasso.NotificationKey("autre") {
		t.Fatalf("clé instable ou mal formée : %q", k)
	}
	if strings.Contains(k, "s3cret") || !helloasso.ValidNotificationKey("s3cret", k) ||
		helloasso.ValidNotificationKey("s3cret", "abc") || helloasso.ValidNotificationKey("", "") {
		t.Fatal("validation de clé incorrecte")
	}
}

func TestParseNotification(t *testing.T) {
	tests := []struct {
		body string
		want helloasso.Notification
		ok   bool
	}{
		{`{"eventType":"Order","data":{"id":11,"items":[]},"metadata":{"frigg":{"payment":7,"membership":42}}}`,
			helloasso.Notification{EventType: "Order", Payment: 7, OrderID: 11}, true},
		{`{"eventType":"Payment","data":{"id":555,"state":"Refunded","order":{"id":98765}}}`,
			helloasso.Notification{EventType: "Payment", HelloAssoPaymentID: 555, OrderID: 98765}, true},
		{`{"eventType":"Payment","data":{"metadata":{"frigg":{"payment":8}}}}`,
			helloasso.Notification{EventType: "Payment", Payment: 8}, true},
		{`{"eventType":"Form","data":{}}`, helloasso.Notification{EventType: "Form"}, false},
		{`pas du json`, helloasso.Notification{}, false},
		{``, helloasso.Notification{}, false},
	}
	for _, tt := range tests {
		got, ok := helloasso.ParseNotification([]byte(tt.body))
		if got != tt.want || ok != tt.ok {
			t.Errorf("%s → %+v %v", tt.body, got, ok)
		}
	}
}

// Une intention d'inscription porte Registration ; une intention antérieure
// à 012 n'a que Membership (012 › R4).
func TestCheckoutMetadataTargets(t *testing.T) {
	f := hatest.New(t)
	cl := newClient(f, &clock{t: time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)})
	ctx := context.Background()

	req := checkout(9)
	req.Metadata = helloasso.Metadata{Payment: 9, Registration: 5, Reference: "EVT-2627-0001"}
	in, err := cl.CreateCheckout(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	ch, err := cl.GetCheckout(ctx, in.ID)
	if err != nil || ch.Metadata == nil || ch.Metadata.Registration != 5 || ch.Metadata.Membership != 0 {
		t.Fatalf("intention d'inscription : %+v, %v", ch.Metadata, err)
	}

	old, err := cl.CreateCheckout(ctx, checkout(10))
	if err != nil {
		t.Fatal(err)
	}
	f.SetMetadata(old.ID, `{"frigg":{"payment":10,"membership":42}}`)
	ch, err = cl.GetCheckout(ctx, old.ID)
	if err != nil || ch.Metadata == nil || ch.Metadata.Membership != 42 || ch.Metadata.Registration != 0 {
		t.Fatalf("intention ancienne : %+v, %v", ch.Metadata, err)
	}
}
