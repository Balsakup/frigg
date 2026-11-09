package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/helloasso"
)

// evCardFixture : événement payant (35 €) relié à un faux HelloAsso, carte
// activée en sandbox ; Claire a inscrit sa fiche.
type evCardFixture struct {
	evFixture
	event api.AdminEvent
	reg   int64
}

func newEvCardFixture(t *testing.T) evCardFixture {
	t.Helper()
	env := apitest.NewWithHelloAsso(t)
	env.SetToday(msToday)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	expectSeason(t, admin.Do("POST", seasonPath(s.ID, "/open"), nil), http.StatusOK)
	ms := msFixture{env: env, admin: admin, season: s}
	ms.young = createFee(t, admin, s.ID, map[string]string{"label": "Jeune", "amount": "30"})
	ms.free = createFee(t, admin, s.ID, map[string]string{"label": "Bienfaiteur", "amount": "0"})
	setPayment(t, admin, nil)
	env.EnableCard(helloasso.Sandbox)
	f := withFamily(t, ms)
	ev := f.publishedEvent(t, eventBody("Stage de printemps", paidEvent("35")))
	out := register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID)
	return evCardFixture{evFixture: f, event: ev, reg: out.CreatedRegistrationIDs[0]}
}

func (f evCardFixture) start(t *testing.T) api.StartCardPayment {
	t.Helper()
	return expectJSON[api.StartCardPayment](t, f.claire.Do("POST", myRegistrationPath(f.reg, "/card-payment"), nil), http.StatusOK)
}

func (f evCardFixture) confirm(t *testing.T, attempt int64, outcome string) api.ConfirmRegistrationCard {
	t.Helper()
	return expectJSON[api.ConfirmRegistrationCard](t, f.claire.Do("POST", myRegistrationPath(f.reg, "/card-payment/confirm"),
		map[string]any{"onlinePaymentId": attempt, "outcome": outcome}), http.StatusOK)
}

func registrationNotification(payment, registration int64) string {
	return fmt.Sprintf(`{"eventType":"Order","data":{"id":1,"items":[]},"metadata":{"frigg":{"payment":%d,"registration":%d}}}`, payment, registration)
}

func TestRegistrationCardPayment(t *testing.T) {
	f := newEvCardFixture(t)
	got := f.myRegistration(t, f.reg)
	if !got.Actions.PayByCard || got.PaymentMethods[0].Method != "card" || !got.PaymentMethods[0].Test {
		t.Fatalf("carte proposée : %+v %+v", got.Actions, got.PaymentMethods)
	}

	st := f.start(t)
	in := f.env.HelloAsso.LastIntent()
	body := fmt.Sprint(in.Body)
	if !strings.Contains(body, "Événement Stage de printemps — EVT-2627-0001") || strings.Contains(body, "Claire Martin — ") {
		t.Fatalf("intitulé : %s", body)
	}
	if !strings.Contains(body, fmt.Sprintf("/mes-inscriptions/%d?tentative=%d&carte=retour", f.reg, st.OnlinePaymentID)) {
		t.Fatalf("adresse de retour : %s", body)
	}
	if again := f.start(t); !again.Resumed || again.OnlinePaymentID != st.OnlinePaymentID {
		t.Fatalf("reprise : %+v", again)
	}

	// Retour sans paiement : rien n'est constaté.
	if out := f.confirm(t, st.OnlinePaymentID, "return"); out.Status != "pending" || out.Registration.Payment.Status != "due" {
		t.Fatalf("retour falsifié : %+v", out)
	}
	// Une tentative en cours bloque l'annulation par le compte.
	if got := f.myRegistration(t, f.reg); got.Actions.Cancel {
		t.Fatal("annulation proposée pendant un paiement par carte")
	}

	f.env.HelloAsso.Authorize(in.ID, 3500, 0)
	f.env.Advance(10 * time.Second)
	out := f.confirm(t, st.OnlinePaymentID, "return")
	p := out.Registration.Payment
	if out.Status != "succeeded" || p.Status != "paid" || *p.PaidMethod != "card" || p.Online == nil || !p.Online.Test {
		t.Fatalf("constaté : %+v %+v", out, p)
	}
	if n := mails(f.env, f.claireEmail, "Paiement reçu"); n != 1 {
		t.Fatalf("e-mails de constatation : %d", n)
	}
	noLeak(t, f.env, "Claire", "Stage de printemps", f.claireEmail)
}

func TestRegistrationCardByNotificationOnly(t *testing.T) {
	f := newEvCardFixture(t)
	st := f.start(t)
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3500, 0)
	// Notification et retour concurrents : une seule constatation.
	var wg sync.WaitGroup
	wg.Go(func() {
		f.env.Client().Do("POST", notificationPath(), rawJSON(registrationNotification(st.OnlinePaymentID, f.reg)))
	})
	wg.Go(func() {
		f.claire.Do("POST", myRegistrationPath(f.reg, "/card-payment/confirm"), map[string]any{"onlinePaymentId": st.OnlinePaymentID, "outcome": "return"})
	})
	wg.Wait()
	if got := f.myRegistration(t, f.reg); got.Payment.Status != "paid" {
		t.Fatalf("inscription : %+v", got.Payment)
	}
	if n := mails(f.env, f.claireEmail, "Paiement reçu"); n != 1 {
		t.Fatalf("e-mails de constatation : %d", n)
	}
}

func TestRegistrationCardAlertsAndRefund(t *testing.T) {
	f := newEvCardFixture(t)
	st := f.start(t)
	in := f.env.HelloAsso.LastIntent()
	// Inscription annulée avant le paiement : signalement « à rembourser ».
	expectJSON[api.MyRegistration](t, f.claire.Do("POST", myRegistrationPath(f.reg, "/cancel"), nil), http.StatusConflict)
	if _, err := f.env.Store.WriteDB.Exec(`UPDATE event_registrations SET status = 'cancelled', cancelled_at = '2026-11-01T11:00:00Z' WHERE id = ?`, f.reg); err != nil {
		t.Fatal(err)
	}
	f.env.HelloAsso.Authorize(in.ID, 3500, 0)
	f.env.Client().Do("POST", notificationPath(), rawJSON(registrationNotification(st.OnlinePaymentID, f.reg)))
	var kind string
	if err := f.env.Store.ReadDB.QueryRow(`SELECT kind FROM payment_alerts WHERE registration_id = ?`, f.reg).Scan(&kind); err != nil || kind != "refund_due" {
		t.Fatalf("signalement : %q (%v)", kind, err)
	}
}

func TestRegistrationRefundCancels(t *testing.T) {
	f := newEvCardFixture(t)
	st := f.start(t)
	in := f.env.HelloAsso.LastIntent()
	f.env.HelloAsso.Authorize(in.ID, 3500, 0)
	if out := f.confirm(t, st.OnlinePaymentID, "return"); out.Status != "succeeded" {
		t.Fatalf("paiement : %+v", out)
	}
	in = f.env.HelloAsso.LastIntent()
	f.env.HelloAsso.Refund(in.ID)
	f.env.Advance(10 * time.Second)
	if res := f.env.Client().Do("POST", notificationPath(), rawJSON(refundNotification(in.PaymentID, in.OrderID))); res.Status != http.StatusOK {
		t.Fatalf("notification : %d", res.Status)
	}
	got := f.myRegistration(t, f.reg)
	if got.State != "cancelled" {
		t.Fatalf("inscription après remboursement : %+v", got)
	}
	if text := lastMail(t, f.env, f.claireEmail); !strings.Contains(text, "Inscription annulée") || !strings.Contains(text, "remboursé") {
		t.Fatalf("e-mail : %s", text)
	}
}

func TestRegistrationCardSweep(t *testing.T) {
	f := newEvCardFixture(t)
	st := f.start(t)
	f.env.Advance(2 * time.Hour)
	f.env.Sweeper.Tick(t.Context())
	var status string
	if err := f.env.Store.ReadDB.QueryRow(`SELECT status FROM online_payments WHERE id = ?`, st.OnlinePaymentID).Scan(&status); err != nil || status != "expired" {
		t.Fatalf("tentative balayée : %q (%v)", status, err)
	}
}

func TestRegistrationCardRefusedUnderMinimum(t *testing.T) {
	f := newEvCardFixture(t)
	cheap := f.publishedEvent(t, eventBody("Goûter", paidEvent("0,50")))
	out := register(t, f.claire, cheap.ID, http.StatusCreated, f.lucas.ID)
	res := f.claire.Do("POST", myRegistrationPath(out.CreatedRegistrationIDs[0], "/card-payment"), nil)
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("montant < 1 € : %d %s", res.Status, res.Body)
	}
}
