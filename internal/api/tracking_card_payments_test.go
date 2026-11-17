package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/helloasso"
)

// Paiement par carte depuis un lien personnel (016 › R11, FR-019 à FR-021) :
// même circuit que l'espace membre, payeur et retour propres au lien.

// trackCardFixture : faux HelloAsso, carte activée en sandbox, une fiche sans
// compte (Léo, leo@exemple.fr) et son adhésion « Jeune » (30 €) à payer.
type trackCardFixture struct {
	msFixture
	link *apitest.Client
	ms   api.AdminMembership
}

const trackedEmailAddr = "leo@exemple.fr"

func newTrackCardFixture(t *testing.T) trackCardFixture {
	t.Helper()
	env := apitest.NewWithHelloAsso(t)
	env.SetToday(msToday)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	expectSeason(t, admin.Do("POST", seasonPath(s.ID, "/open"), nil), http.StatusOK)
	f := msFixture{env: env, admin: admin, season: s}
	f.adult = createFee(t, admin, s.ID, map[string]string{"label": "Adulte", "amount": "60"})
	f.young = createFee(t, admin, s.ID, map[string]string{"label": "Jeune", "amount": "30"})
	setPayment(t, admin, nil)
	env.EnableCard(helloasso.Sandbox)
	leo := trackedMember(t, f, "Léo", trackedEmailAddr)
	ms := validatedFor(t, f, leo, f.young.ID)
	return trackCardFixture{msFixture: f, link: env.TrackingClient(trackedEmailAddr), ms: ms}
}

func (f trackCardFixture) start(t *testing.T) api.StartCardPayment {
	t.Helper()
	return expectJSON[api.StartCardPayment](t, f.link.Do("POST", trackingPath(f.ms.ID, "/card-payment"), nil), http.StatusOK)
}

func (f trackCardFixture) confirm(t *testing.T, attempt int64, outcome string) api.ConfirmTrackedCardPayment {
	t.Helper()
	return expectJSON[api.ConfirmTrackedCardPayment](t, f.link.Do("POST", trackingPath(f.ms.ID, "/card-payment/confirm"),
		map[string]any{"onlinePaymentId": attempt, "outcome": outcome}), http.StatusOK)
}

func (f trackCardFixture) paidMails(t *testing.T) []apitest.Mail {
	t.Helper()
	var out []apitest.Mail
	for _, m := range f.env.Mails(trackedEmailAddr) {
		if strings.HasPrefix(m.Subject, "Paiement de l'adhésion") {
			out = append(out, m)
		}
	}
	return out
}

func TestTrackingCardStart(t *testing.T) {
	f := newTrackCardFixture(t)
	got := expectTracked(t, f.link.Do("GET", trackingPath(f.ms.ID, ""), nil), http.StatusOK)
	if !got.Actions.PayByCard || got.PaymentMethods[0].Method != "card" || !got.PaymentMethods[0].Test {
		t.Fatalf("carte proposée en mode test : %+v", got.PaymentMethods)
	}
	st := f.start(t)
	body := f.env.HelloAsso.LastIntent().Body
	payer := body["payer"].(map[string]any)
	if payer["email"] != trackedEmailAddr || payer["firstName"] != "Léo" || payer["lastName"] != "Martin" || len(payer) != 3 {
		t.Fatalf("payeur : %v", payer)
	}
	if ret := body["returnUrl"].(string); ret != fmt.Sprintf("http://frigg.test/suivi/adhesions/%d?tentative=%d&carte=retour", f.ms.ID, st.OnlinePaymentID) {
		t.Fatalf("returnUrl : %q", ret)
	}
	if item := body["itemName"].(string); strings.Contains(item, "Léo") || strings.Contains(item, "Martin") {
		t.Fatalf("intitulé : %q", item)
	}
	if n := count(t, f.env, `SELECT count(*) FROM online_payments WHERE id = ? AND account_id IS NULL AND via_link = 1`, st.OnlinePaymentID); n != 1 {
		t.Fatal("tentative sans origine « lien »")
	}
	if again := f.start(t); !again.Resumed || again.OnlinePaymentID != st.OnlinePaymentID {
		t.Fatalf("reprise : %+v", again)
	}
	for _, forbidden := range []string{trackedEmailAddr, "Léo"} {
		if strings.Contains(f.env.Logs.String(), forbidden) {
			t.Fatalf("%q journalisé", forbidden)
		}
	}
}

func TestTrackingCardConfirmByReturn(t *testing.T) {
	f := newTrackCardFixture(t)
	st := f.start(t)
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3000, 0)
	f.env.Advance(10 * time.Second)
	out := f.confirm(t, st.OnlinePaymentID, "return")
	p := out.Membership.Payment
	if out.Status != "succeeded" || p.Status != "paid" || *p.PaidMethod != "card" || p.Online == nil || !p.Online.Test {
		t.Fatalf("constatation : %+v %+v", out, p)
	}
	f.env.Advance(10 * time.Second)
	f.confirm(t, st.OnlinePaymentID, "return")
	mails := f.paidMails(t)
	if len(mails) != 1 || !strings.Contains(mails[0].Link, "http://frigg.test/suivi#jeton=") {
		t.Fatalf("e-mails de constatation : %+v", mails)
	}
	// Une tentative d'une autre adhésion est introuvable par ce chemin.
	res := f.link.Do("POST", trackingPath(99999, "/card-payment/confirm"), map[string]any{"onlinePaymentId": st.OnlinePaymentID, "outcome": "return"})
	if res.Status != http.StatusNotFound {
		t.Fatalf("tentative hors cible : %d", res.Status)
	}
}

func TestTrackingCardConfirmByNotification(t *testing.T) {
	f := newTrackCardFixture(t)
	st := f.start(t)
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3000, 0)
	for range 2 {
		if res := f.env.Client().Do("POST", notificationPath(), rawJSON(notificationBody(st.OnlinePaymentID, f.ms.ID))); res.Status != http.StatusOK {
			t.Fatalf("notification : %d", res.Status)
		}
		f.env.Advance(10 * time.Second)
	}
	if got := expectTracked(t, f.link.Do("GET", trackingPath(f.ms.ID, ""), nil), http.StatusOK); got.Payment.Status != "paid" {
		t.Fatalf("adhésion : %+v", got.Payment)
	}
	if n := len(f.paidMails(t)); n != 1 {
		t.Fatalf("e-mails de constatation : %d", n)
	}
}

// Paiement par carte d'une inscription depuis un lien personnel (016 › R11) :
// retour sur /suivi/inscriptions/{id}, annulation refusée pendant le
// paiement.
func TestTrackingRegistrationCard(t *testing.T) {
	f := newTrackCardFixture(t)
	ev := evFixture{msFixture: f.msFixture}.publishedEvent(t, eventBody("Stage", paidEvent("35")))
	leo := f.ms.Member.ID
	reg := adminRegisters(t, f.msFixture, ev.ID, leo)

	got := expectTrackedReg(t, f.link.Do("GET", trackingRegistrationPath(reg, ""), nil), http.StatusOK)
	if !got.Actions.PayByCard || got.PaymentMethods[0].Method != "card" {
		t.Fatalf("carte proposée : %+v", got.PaymentMethods)
	}
	st := expectJSON[api.StartCardPayment](t, f.link.Do("POST", trackingRegistrationPath(reg, "/card-payment"), nil), http.StatusOK)
	body := f.env.HelloAsso.LastIntent().Body
	payer := body["payer"].(map[string]any)
	if payer["email"] != trackedEmailAddr || payer["firstName"] != "Léo" || len(payer) != 3 {
		t.Fatalf("payeur : %v", payer)
	}
	if ret := body["returnUrl"].(string); ret != fmt.Sprintf("http://frigg.test/suivi/inscriptions/%d?tentative=%d&carte=retour", reg, st.OnlinePaymentID) {
		t.Fatalf("returnUrl : %q", ret)
	}
	if item := body["itemName"].(string); strings.Contains(item, "Léo") || !strings.Contains(item, "Stage") {
		t.Fatalf("intitulé : %q", item)
	}
	if n := count(t, f.env, `SELECT count(*) FROM online_payments WHERE id = ? AND registration_id = ? AND account_id IS NULL AND via_link = 1`,
		st.OnlinePaymentID, reg); n != 1 {
		t.Fatal("tentative sans origine « lien »")
	}
	// Paiement par carte en cours : annulation refusée (012 FR-013).
	expectProblem(t, f.link.Do("POST", trackingRegistrationPath(reg, "/cancel"), nil), http.StatusConflict, api.MsgRegistrationPaid)

	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3500, 0)
	f.env.Advance(10 * time.Second)
	out := expectJSON[api.ConfirmTrackedRegistrationCard](t, f.link.Do("POST", trackingRegistrationPath(reg, "/card-payment/confirm"),
		map[string]any{"onlinePaymentId": st.OnlinePaymentID, "outcome": "return"}), http.StatusOK)
	if out.Status != "succeeded" || out.Registration.Payment.Status != "paid" || *out.Registration.Payment.PaidMethod != "card" {
		t.Fatalf("constatation : %+v %+v", out, out.Registration.Payment)
	}
	// Une tentative présentée pour une autre inscription est introuvable.
	if res := f.link.Do("POST", trackingRegistrationPath(99999, "/card-payment/confirm"),
		map[string]any{"onlinePaymentId": st.OnlinePaymentID, "outcome": "return"}); res.Status != http.StatusNotFound {
		t.Fatalf("tentative hors cible : %d", res.Status)
	}
	// La tentative d'adhésion n'est pas acceptée par le chemin des inscriptions.
	if res := f.link.Do("POST", trackingPath(f.ms.ID, "/card-payment/confirm"),
		map[string]any{"onlinePaymentId": st.OnlinePaymentID, "outcome": "return"}); res.Status != http.StatusNotFound {
		t.Fatalf("tentative d'inscription par le chemin des adhésions : %d", res.Status)
	}
	noLeak(t, f.env, trackedEmailAddr, "Léo")
}
