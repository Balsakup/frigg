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
	"github.com/balsakup/frigg/internal/helloasso/hatest"
)

// cardFixture : jeu de données des adhésions relié à un faux HelloAsso, carte
// activée et testée en sandbox, une adhésion « Jeune » (30 €) validée à payer.
type cardFixture struct {
	msFixture
	member *apitest.Client
	email  string
	ms     api.MyMembership
}

func newCardFixture(t *testing.T) cardFixture {
	t.Helper()
	env := apitest.NewWithHelloAsso(t)
	env.SetToday(msToday)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	expectSeason(t, admin.Do("POST", seasonPath(s.ID, "/open"), nil), http.StatusOK)
	f := msFixture{env: env, admin: admin, season: s}
	f.adult = expectFee(t, admin.Do("POST", feesPath(s.ID), map[string]any{
		"label": "Adulte", "amount": "60", "hasRenewalAmount": true, "renewalAmount": "40",
	}), http.StatusCreated)
	f.young = createFee(t, admin, s.ID, map[string]string{"label": "Jeune", "amount": "30"})
	f.free = createFee(t, admin, s.ID, map[string]string{"label": "Bienfaiteur", "amount": "0"})
	setPayment(t, admin, nil)
	env.EnableCard(helloasso.Sandbox)

	c, email, m := f.member(t, "Léo")
	ms := request(t, c, m.ID, s.ID, f.young.ID, "first")
	validate(t, admin, ms.ID, f.young.ID, "first")
	return cardFixture{msFixture: f, member: c, email: email, ms: ms}
}

func cardPath(id int64, suffix string) string {
	return fmt.Sprintf("/api/me/memberships/%d/card-payment%s", id, suffix)
}

func (f cardFixture) start(t *testing.T) api.StartCardPayment {
	t.Helper()
	res := f.member.Do("POST", cardPath(f.ms.ID, ""), nil)
	if res.Status != http.StatusOK {
		t.Fatalf("lancement : %d %s", res.Status, res.Body)
	}
	var out api.StartCardPayment
	res.JSON(t, &out)
	return out
}

func (f cardFixture) confirm(t *testing.T, c *apitest.Client, attempt int64, outcome string) (int, api.ConfirmCardPayment) {
	t.Helper()
	res := c.Do("POST", cardPath(f.ms.ID, "/confirm"), map[string]any{"onlinePaymentId": attempt, "outcome": outcome})
	var out api.ConfirmCardPayment
	if res.Status == http.StatusOK {
		res.JSON(t, &out)
	}
	return res.Status, out
}

func (f cardFixture) mine(t *testing.T) api.MyMembership {
	t.Helper()
	return expectMine(t, f.member.Do("GET", myMembershipPath(f.ms.ID), nil), http.StatusOK)
}

func (f cardFixture) paidMails() int {
	n := 0
	for _, m := range f.env.Mail.Messages() {
		if m.To == f.email && strings.HasPrefix(m.Msg.Subject, "Paiement de l'adhésion") {
			n++
		}
	}
	return n
}

func TestCardOffered(t *testing.T) {
	f := newCardFixture(t)
	got := f.mine(t)
	if !got.Actions.PayByCard || len(got.PaymentMethods) != 4 || got.PaymentMethods[0].Method != "card" {
		t.Fatalf("moyens : %+v, actions %+v", got.PaymentMethods, got.Actions)
	}
	card := got.PaymentMethods[0]
	if card.AmountCents == nil || *card.AmountCents != 3000 || !card.Test {
		t.Fatalf("carte : %+v", card)
	}
}

func TestStartCardPayment(t *testing.T) {
	f := newCardFixture(t)
	first := f.start(t)
	if first.Resumed || !strings.HasPrefix(first.RedirectURL, f.env.HelloAsso.URL) {
		t.Fatalf("lancement : %+v", first)
	}
	in := f.env.HelloAsso.LastIntent()
	body := in.Body
	if body["totalAmount"].(float64) != 3000 {
		t.Fatalf("montant envoyé : %v", body["totalAmount"])
	}
	item := body["itemName"].(string)
	if !strings.Contains(item, "2026-2027") || !strings.Contains(item, "Jeune") || !strings.Contains(item, "ADH-") ||
		strings.Contains(item, "Léo") || strings.Contains(item, "Martin") {
		t.Fatalf("intitulé : %q", item)
	}
	payer := body["payer"].(map[string]any)
	if payer["email"] != f.email || payer["firstName"] != nil {
		t.Fatalf("payeur : %v", payer)
	}
	meta := body["metadata"].(map[string]any)["frigg"].(map[string]any)
	if int64(meta["payment"].(float64)) != first.OnlinePaymentID || int64(meta["membership"].(float64)) != f.ms.ID {
		t.Fatalf("métadonnées : %v", meta)
	}
	if ret := body["returnUrl"].(string); !strings.Contains(ret, fmt.Sprintf("/adhesions/%d?", f.ms.ID)) || !strings.Contains(ret, "carte=retour") {
		t.Fatalf("returnUrl : %q", ret)
	}

	again := f.start(t)
	if !again.Resumed || again.OnlinePaymentID != first.OnlinePaymentID || f.env.HelloAsso.Calls("POST checkout") != 1 {
		t.Fatalf("reprise : %+v, %d intentions", again, f.env.HelloAsso.Calls("POST checkout"))
	}
	f.env.Advance(15 * time.Minute)
	if later := f.start(t); later.Resumed || later.OnlinePaymentID == first.OnlinePaymentID {
		t.Fatalf("page expirée reprise : %+v", later)
	}
}

func TestStartCardPaymentRefusals(t *testing.T) {
	f := newCardFixture(t)
	f.env.HelloAsso.Down(true)
	if res := f.member.Do("POST", cardPath(f.ms.ID, ""), nil); res.Status != http.StatusServiceUnavailable {
		t.Fatalf("HelloAsso injoignable : %d %s", res.Status, res.Body)
	}
	f.env.HelloAsso.Down(false)
	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), http.StatusOK)
	if len(got.OnlinePayments) != 1 || got.OnlinePayments[0].Status != "failed" || *got.OnlinePayments[0].ErrorCode != "unavailable" {
		t.Fatalf("tentative échouée : %+v", got.OnlinePayments)
	}

	// Carte désactivée → 422 ; adhésion payée → 409.
	setPayment(t, f.admin, map[string]any{
		"card": map[string]any{"enabled": false}, "cash": map[string]any{"enabled": true, "instructions": "Au local"},
		"cheque": map[string]any{"enabled": false}, "transfer": map[string]any{"enabled": false},
	})
	if res := f.member.Do("POST", cardPath(f.ms.ID, ""), nil); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("carte désactivée : %d", res.Status)
	}
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(f.ms.ID, "/payment"), map[string]any{
		"method": "cash", "receivedOn": msToday,
	}), http.StatusOK)
	f.env.EnableCard(helloasso.Sandbox)
	if res := f.member.Do("POST", cardPath(f.ms.ID, ""), nil); res.Status != http.StatusConflict {
		t.Fatalf("adhésion payée : %d", res.Status)
	}
}

func TestConfirmCardPayment(t *testing.T) {
	f := newCardFixture(t)
	st := f.start(t)
	in := f.env.HelloAsso.LastIntent()

	// Retour « réussi » sans paiement chez HelloAsso : rien n'est constaté.
	if code, out := f.confirm(t, f.member, st.OnlinePaymentID, "return"); code != http.StatusOK || out.Status != "pending" ||
		out.Membership.Payment.Status != "due" {
		t.Fatalf("retour falsifié : %d %+v", code, out)
	}

	f.env.HelloAsso.Authorize(in.ID, 3300, 300)
	f.env.Advance(10 * time.Second)
	code, out := f.confirm(t, f.member, st.OnlinePaymentID, "return")
	p := out.Membership.Payment
	if code != http.StatusOK || out.Status != "succeeded" || p.Status != "paid" || *p.PaidMethod != "card" ||
		*p.PaidAmountCents != 3000 || p.Online == nil || !p.Online.Test || p.Online.OrderID == 0 {
		t.Fatalf("paiement constaté : %d %+v %+v", code, out, p)
	}
	f.env.FlushMail()
	if f.paidMails() != 1 {
		t.Fatalf("e-mails de constatation : %d", f.paidMails())
	}
	// Retour rejoué : aucun nouvel effet.
	f.env.Advance(10 * time.Second)
	f.confirm(t, f.member, st.OnlinePaymentID, "return")
	f.env.FlushMail()
	if f.paidMails() != 1 {
		t.Fatalf("e-mails après rejeu : %d", f.paidMails())
	}
	if !strings.Contains(f.env.Logs.String(), `"msg":"paiement en ligne vérifié"`) {
		t.Fatal("vérification non journalisée")
	}
	for _, forbidden := range []string{hatest.ClientSecret, "tok-", f.email, "Léo", "/pay/"} {
		if strings.Contains(f.env.Logs.String(), forbidden) {
			t.Fatalf("%q journalisé", forbidden)
		}
	}
}

func TestConfirmErrorAndBack(t *testing.T) {
	f := newCardFixture(t)
	st := f.start(t)
	if _, out := f.confirm(t, f.member, st.OnlinePaymentID, "error"); out.Status != "failed" {
		t.Fatalf("erreur : %+v", out)
	}
	// Un paiement autorisé découvert ensuite l'emporte.
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3000, 0)
	f.env.Advance(10 * time.Second)
	if _, out := f.confirm(t, f.member, st.OnlinePaymentID, "return"); out.Status != "succeeded" || out.Membership.Payment.Status != "paid" {
		t.Fatalf("échec puis paiement : %+v", out)
	}

	g := newCardFixture(t)
	st2 := g.start(t)
	if _, out := g.confirm(t, g.member, st2.OnlinePaymentID, "back"); out.Status != "abandoned" || out.Membership.Payment.Status != "due" {
		t.Fatalf("retour arrière : %+v", out)
	}
}

func TestConfirmThrottled(t *testing.T) {
	f := newCardFixture(t)
	st := f.start(t)
	f.confirm(t, f.member, st.OnlinePaymentID, "return")
	f.confirm(t, f.member, st.OnlinePaymentID, "return")
	if n := f.env.HelloAsso.Calls("GET checkout"); n != 1 {
		t.Fatalf("relectures HelloAsso : %d, attendu 1", n)
	}
}

func TestCardPaymentAlerts(t *testing.T) {
	// Adhésion déjà payée par chèque → doublon.
	f := newCardFixture(t)
	st := f.start(t)
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(f.ms.ID, "/payment"), map[string]any{
		"method": "cheque", "receivedOn": msToday,
	}), http.StatusOK)
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3000, 0)
	f.confirm(t, f.member, st.OnlinePaymentID, "return")
	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), http.StatusOK)
	if *got.Payment.PaidMethod != "cheque" || len(got.Alerts) != 1 || got.Alerts[0].Kind != "duplicate" {
		t.Fatalf("doublon : %+v %+v", got.Payment, got.Alerts)
	}

	// Montant différent.
	g := newCardFixture(t)
	st = g.start(t)
	g.env.HelloAsso.Authorize(g.env.HelloAsso.LastIntent().ID, 2000, 0)
	g.confirm(t, g.member, st.OnlinePaymentID, "return")
	got = expectAdminMembership(t, g.admin.Do("GET", adminMembershipPath(g.ms.ID, ""), nil), http.StatusOK)
	if got.Payment.Status != "paid" || len(got.Alerts) != 1 || got.Alerts[0].Kind != "amount_mismatch" {
		t.Fatalf("montant différent : %+v %+v", got.Payment, got.Alerts)
	}

	// Adhésion annulée pendant le paiement.
	h := newCardFixture(t)
	st = h.start(t)
	expectAdminMembership(t, h.admin.Do("POST", adminMembershipPath(h.ms.ID, "/cancel"), map[string]any{"reason": "Départ"}), http.StatusOK)
	h.env.HelloAsso.Authorize(h.env.HelloAsso.LastIntent().ID, 3000, 0)
	h.confirm(t, h.member, st.OnlinePaymentID, "return")
	got = expectAdminMembership(t, h.admin.Do("GET", adminMembershipPath(h.ms.ID, ""), nil), http.StatusOK)
	if got.Status != "cancelled" || len(got.Alerts) != 1 || got.Alerts[0].Kind != "refund_due" {
		t.Fatalf("adhésion annulée : %s %+v", got.Status, got.Alerts)
	}
}

func TestCardMethodRefusedElsewhere(t *testing.T) {
	f := newCardFixture(t)
	res := f.member.Do("PUT", myMembershipPath(f.ms.ID)+"/payment", map[string]any{"method": "card"})
	if res.Status != http.StatusUnprocessableEntity || !strings.Contains(string(res.Body), "en ligne") {
		t.Fatalf("annonce de la carte : %d %s", res.Status, res.Body)
	}
	if res := f.admin.Do("POST", adminMembershipPath(f.ms.ID, "/payment"), map[string]any{"method": "card", "receivedOn": msToday}); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("constatation manuelle de la carte : %d", res.Status)
	}
	st := f.start(t)
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3000, 0)
	f.confirm(t, f.member, st.OnlinePaymentID, "return")
	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), http.StatusOK)
	if got.Actions.CancelPayment {
		t.Fatal("annulation de la constatation proposée pour une carte")
	}
	if res := f.admin.Do("DELETE", adminMembershipPath(f.ms.ID, "/payment"), nil); res.Status != http.StatusConflict {
		t.Fatalf("annulation de la constatation d'une carte : %d", res.Status)
	}
}

func TestConcurrentNotificationAndReturn(t *testing.T) {
	f := newCardFixture(t)
	st := f.start(t)
	in := f.env.HelloAsso.LastIntent()
	f.env.HelloAsso.Authorize(in.ID, 3000, 0)
	notif := notificationBody(st.OnlinePaymentID, f.ms.ID)
	var wg sync.WaitGroup
	for i := range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				f.env.Client().Do("POST", notificationPath(), rawJSON(notif))
			} else {
				f.confirm(t, f.member, st.OnlinePaymentID, "return")
			}
		}()
	}
	wg.Wait()
	f.env.FlushMail()
	if n := f.paidMails(); n != 1 {
		t.Fatalf("e-mails de constatation : %d, attendu 1", n)
	}
	if got := f.mine(t); got.Payment.Status != "paid" {
		t.Fatalf("adhésion : %+v", got.Payment)
	}
}
