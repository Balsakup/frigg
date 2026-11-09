package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
)

func paidEvent(price string) func(map[string]any) {
	return func(b map[string]any) { b["openness"], b["price"] = "all_members", price }
}

func TestPaidRegistration(t *testing.T) {
	f := newEvFixture(t)
	setPayment(t, f.admin, nil)
	ev := f.publishedEvent(t, eventBody("Stage", paidEvent("35")))

	out := register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID, f.lucas.ID)
	if len(out.CreatedRegistrationIDs) != 2 {
		t.Fatalf("créées : %+v", out)
	}
	first := f.myRegistration(t, out.CreatedRegistrationIDs[0])
	second := f.myRegistration(t, out.CreatedRegistrationIDs[1])
	if first.AmountCents != 3500 || first.Payment == nil || first.Payment.Reference != "EVT-2627-0001" ||
		first.Payment.Status != "due" || second.Payment.Reference != "EVT-2627-0002" {
		t.Fatalf("paiement : %+v / %+v", first.Payment, second.Payment)
	}
	if len(first.PaymentMethods) != 3 || !first.Actions.Pay || !first.Actions.Cancel {
		t.Fatalf("moyens : %+v %+v", first.PaymentMethods, first.Actions)
	}
	text := lastMail(t, f.env, f.claireEmail)
	if !strings.Contains(text, "70,00") || !strings.Contains(text, "EVT-2627-0002") || !strings.Contains(text, "/a-regler") {
		t.Fatalf("e-mail : %s", text)
	}

	// Le prix change : les montants figés restent.
	expectAdminEvent(t, f.admin.Do("PUT", eventPath(ev.ID, ""), eventBody("Stage", paidEvent("50"))), http.StatusOK)
	if got := f.myRegistration(t, first.ID); got.AmountCents != 3500 {
		t.Fatalf("montant figé : %d", got.AmountCents)
	}

	// Choix du chèque.
	got := expectJSON[api.MyRegistration](t, f.claire.Do("PUT", myRegistrationPath(first.ID, "/payment"), map[string]any{"method": "cheque"}), http.StatusOK)
	if got.Payment.Status != "announced" || *got.Payment.AnnouncedMethod != "cheque" {
		t.Fatalf("annonce : %+v", got.Payment)
	}
	res := f.claire.Do("PUT", myRegistrationPath(first.ID, "/payment"), map[string]any{"method": "card"})
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("carte annoncée : %d", res.Status)
	}

	// « À régler » : les deux inscriptions, sans fiche d'autrui.
	dues := expectJSON[struct {
		Items []api.Due `json:"items"`
	}](t, f.claire.Do("GET", "/api/me/dues", nil), http.StatusOK)
	if len(dues.Items) != 2 || dues.Items[0].Kind != "registration" || dues.Items[0].Label != "Stage" {
		t.Fatalf("à régler : %+v", dues.Items)
	}

	// Payée par chèque (constatation en base, l'opération admin vient en US5) :
	// l'annulation par le compte est refusée.
	if _, err := f.env.Store.WriteDB.Exec(`UPDATE event_registrations SET payment_status = 'paid', paid_method = 'cheque',
		paid_on = '2026-11-01' WHERE id = ?`, first.ID); err != nil {
		t.Fatal(err)
	}
	expectProblem(t, f.claire.Do("POST", myRegistrationPath(first.ID, "/cancel"), nil), http.StatusConflict, api.MsgRegistrationPaid)
	if got := f.myRegistration(t, first.ID); got.Actions.Cancel || got.Actions.Pay || len(got.PaymentMethods) != 0 {
		t.Fatalf("payée : %+v", got)
	}
	dues = expectJSON[struct {
		Items []api.Due `json:"items"`
	}](t, f.claire.Do("GET", "/api/me/dues", nil), http.StatusOK)
	if len(dues.Items) != 1 {
		t.Fatalf("à régler après paiement : %d", len(dues.Items))
	}

	// Inscription annulée : plus payable, hors « À régler ».
	expectJSON[api.MyRegistration](t, f.claire.Do("POST", myRegistrationPath(second.ID, "/cancel"), nil), http.StatusOK)
	expectProblem(t, f.claire.Do("PUT", myRegistrationPath(second.ID, "/payment"), map[string]any{"method": "cash"}),
		http.StatusConflict, api.MsgNotPayableRegistration)
	dues = expectJSON[struct {
		Items []api.Due `json:"items"`
	}](t, f.claire.Do("GET", "/api/me/dues", nil), http.StatusOK)
	if len(dues.Items) != 0 {
		t.Fatalf("à régler : %+v", dues.Items)
	}
	noLeak(t, f.env, "Claire", "Lucas", "Stage")
}

func TestDuesMixMembershipsAndRegistrations(t *testing.T) {
	f := newEvFixture(t)
	setPayment(t, f.admin, nil)
	ms := request(t, f.claire, f.lucas.ID, f.season.ID, f.young.ID, "first")
	validate(t, f.admin, ms.ID, f.young.ID, "first")
	ev := f.publishedEvent(t, eventBody("Stage", paidEvent("10")))
	register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID)
	free := f.publishedEvent(t, eventBody("AG"))
	register(t, f.claire, free.ID, http.StatusCreated, f.self.ID)

	dues := expectJSON[struct {
		Items []api.Due `json:"items"`
	}](t, f.claire.Do("GET", "/api/me/dues", nil), http.StatusOK)
	if len(dues.Items) != 2 || dues.Items[0].Kind != "membership" || dues.Items[0].Label != "Adhésion 2026-2027" ||
		dues.Items[1].Kind != "registration" || dues.Items[1].AmountCents != 1000 {
		t.Fatalf("à régler : %+v", dues.Items)
	}
}
