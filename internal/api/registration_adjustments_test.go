package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
)

// Remises et dons sur les inscriptions (specs/014-payment-adjustments US4).

func expectAdminRegistration(t *testing.T, res apitest.Response, status int) api.AdminRegistration {
	t.Helper()
	return expectJSON[api.AdminRegistration](t, res, status)
}

func openToAll(b map[string]any) { b["openness"] = "all_members" }

func TestRegisterMemberWithAdjustment(t *testing.T) {
	f := newEvFixture(t)
	setPayment(t, f.admin, nil)
	ev := f.publishedEvent(t, eventBody("Stage", paidEvent("35")))
	path := eventPath(ev.ID, "/registrations")

	expectFieldError(t, f.admin.Do("POST", path, map[string]any{"memberId": f.self.ID, "amount": "20"}),
		"body.adjustmentReason", api.MsgAdjustmentReasonRequired)
	r := expectAdminRegistration(t, f.admin.Do("POST", path, map[string]any{
		"memberId": f.self.ID, "amount": "20", "adjustmentReason": "bénévole encadrant",
	}), http.StatusCreated)
	if r.AmountCents != 2000 || r.BaseAmountCents != 3500 || r.AdjustmentCents != -1500 ||
		r.AdjustmentReason == nil || r.Payment == nil || r.Payment.Status != "due" || !r.Actions.AdjustAmount {
		t.Fatalf("inscription ajustée : %+v", r)
	}

	// Événement payant ajusté à 0 € avec un paiement : gratuite, paiement ignoré.
	z := expectAdminRegistration(t, f.admin.Do("POST", path, map[string]any{
		"memberId": f.lucas.ID, "amount": "0", "adjustmentReason": "invité",
		"payment": map[string]any{"method": "cash", "receivedOn": msToday},
	}), http.StatusCreated)
	if z.AmountCents != 0 || z.Payment != nil {
		t.Fatalf("gratuite par ajustement : %+v", z)
	}

	// Événement gratuit ajusté à 10 € avec un paiement : référence et paiement.
	free := f.publishedEvent(t, eventBody("Repas", openToAll))
	p := expectAdminRegistration(t, f.admin.Do("POST", eventPath(free.ID, "/registrations"), map[string]any{
		"memberId": f.self.ID, "amount": "10", "adjustmentReason": "participation aux frais",
		"payment": map[string]any{"method": "cash", "receivedOn": msToday, "receivedAmount": "12"},
	}), http.StatusCreated)
	if p.Payment == nil || !strings.HasPrefix(p.Payment.Reference, "EVT-") || p.Payment.Status != "paid" ||
		*p.Payment.PaidAmountCents != 1200 || p.Payment.DonationCents != 200 {
		t.Fatalf("événement gratuit ajusté : %+v", p.Payment)
	}
}

func TestAdjustRegistrationAmount(t *testing.T) {
	f := newEvFixture(t)
	setPayment(t, f.admin, nil)
	ev := f.publishedEvent(t, eventBody("Stage", paidEvent("35")))
	out := register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID, f.lucas.ID)
	id := out.CreatedRegistrationIDs[0]
	adjustPath := adminRegistrationPath(id, "/amount")
	before := expectAdminRegistration(t, f.admin.Do("GET", adminRegistrationPath(id, ""), nil), http.StatusOK)
	ref := before.Payment.Reference

	expectFieldError(t, f.admin.Do("PUT", adjustPath, map[string]any{"amount": "20"}), "body.adjustmentReason", api.MsgAdjustmentReasonRequired)
	expectFieldError(t, f.admin.Do("PUT", adjustPath, map[string]any{"amount": "x", "adjustmentReason": "r"}), "body.amount", api.MsgAmountFormat)

	// 0 € : gratuite, référence conservée en base.
	got := expectAdminRegistration(t, f.admin.Do("PUT", adjustPath, map[string]any{"amount": "0", "adjustmentReason": "invité"}), http.StatusOK)
	if got.Payment != nil || got.AmountCents != 0 || got.BaseAmountCents != 3500 {
		t.Fatalf("à 0 € : %+v", got)
	}
	var stored string
	if err := f.env.Store.WriteDB.QueryRow(`SELECT payment_reference FROM event_registrations WHERE id = ?`, id).Scan(&stored); err != nil || stored != ref {
		t.Fatalf("référence conservée : %q (%v)", stored, err)
	}
	// Retour au prix : à payer, même référence.
	got = expectAdminRegistration(t, f.admin.Do("PUT", adjustPath, map[string]any{}), http.StatusOK)
	if got.Payment == nil || got.Payment.Reference != ref || got.Payment.Status != "due" || got.AdjustmentReason != nil {
		t.Fatalf("retour au prix : %+v", got)
	}
	if !strings.Contains(f.env.Logs.String(), `"action":"registration.adjust"`) {
		t.Fatal("ajustement non journalisé")
	}
	noLeak(t, f.env, "invité")

	// Le prix de l'événement change : montants existants inchangés.
	expectAdminRegistration(t, f.admin.Do("PUT", adjustPath, map[string]any{"amount": "30", "adjustmentReason": "fratrie"}), http.StatusOK)
	expectAdminEvent(t, f.admin.Do("PUT", eventPath(ev.ID, ""), eventBody("Stage", paidEvent("50"))), http.StatusOK)
	got = expectAdminRegistration(t, f.admin.Do("GET", adminRegistrationPath(id, ""), nil), http.StatusOK)
	if got.AmountCents != 3000 || got.BaseAmountCents != 3500 {
		t.Fatalf("après changement de prix : %+v", got)
	}

	// Constatation avec don, refus d'un montant inférieur, annulation.
	payPath := adminRegistrationPath(id, "/payment")
	expectFieldError(t, f.admin.Do("POST", payPath, map[string]any{"method": "cash", "receivedOn": msToday, "receivedAmount": "25"}),
		"body.receivedAmount", api.MsgReceivedBelowDue)
	got = expectAdminRegistration(t, f.admin.Do("POST", payPath, map[string]any{"method": "cash", "receivedOn": msToday, "receivedAmount": "35"}), http.StatusOK)
	if *got.Payment.PaidAmountCents != 3500 || got.Payment.DonationCents != 500 || got.Actions.AdjustAmount {
		t.Fatalf("paiement avec don : %+v %+v", got.Payment, got.Actions)
	}
	text := lastMail(t, f.env, f.claireEmail)
	if !strings.Contains(text, "Dont 5,00") || !strings.Contains(text, "de don") {
		t.Fatalf("e-mail de paiement : %s", text)
	}
	expectProblem(t, f.admin.Do("PUT", adjustPath, map[string]any{"amount": "20", "adjustmentReason": "x"}), http.StatusConflict, api.MsgRegistrationChanged)
	got = expectAdminRegistration(t, f.admin.Do("DELETE", payPath, nil), http.StatusOK)
	if got.Payment.PaidAmountCents != nil {
		t.Fatalf("après annulation : %+v", got.Payment)
	}

	// Espace membre : ni montant théorique, ni ajustement, ni motif.
	for _, p := range []string{myRegistrationPath(id, ""), "/api/me/registrations", "/api/me/dues"} {
		res := f.claire.Do("GET", p, nil)
		body := string(res.Body)
		if res.Status != http.StatusOK || strings.Contains(body, "baseAmountCents") || strings.Contains(body, "adjustmentCents") ||
			strings.Contains(body, "adjustmentReason") || strings.Contains(body, "fratrie") {
			t.Fatalf("%s expose l'ajustement : %d %s", p, res.Status, body)
		}
	}

	// Événement annulé : ajustement refusé.
	other := out.CreatedRegistrationIDs[1]
	if res := f.admin.Do("POST", eventPath(ev.ID, "/cancel"), map[string]any{"message": "Salle indisponible."}); res.Status != http.StatusOK {
		t.Fatalf("annulation de l'événement : %d %s", res.Status, res.Body)
	}
	expectProblem(t, f.admin.Do("PUT", adminRegistrationPath(other, "/amount"), map[string]any{"amount": "20", "adjustmentReason": "x"}),
		http.StatusConflict, api.MsgRegistrationChanged)
}

func TestRegistrationExportAmounts(t *testing.T) {
	f := newEvFixture(t)
	setPayment(t, f.admin, nil)
	ev := f.publishedEvent(t, eventBody("Stage", paidEvent("35")))
	out := register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID, f.lucas.ID)
	expectAdminRegistration(t, f.admin.Do("PUT", adminRegistrationPath(out.CreatedRegistrationIDs[0], "/amount"), map[string]any{
		"amount": "20", "adjustmentReason": "=HYPERLINK(\"x\")",
	}), http.StatusOK)
	expectAdminRegistration(t, f.admin.Do("POST", adminRegistrationPath(out.CreatedRegistrationIDs[1], "/payment"), map[string]any{
		"method": "cash", "receivedOn": msToday, "receivedAmount": "40",
	}), http.StatusOK)
	res := f.admin.Do("GET", eventPath(ev.ID, "/registrations.csv"), nil)
	if res.Status != http.StatusOK {
		t.Fatalf("export : %d %s", res.Status, res.Body)
	}
	csv := string(res.Body)
	for _, want := range []string{"Montant théorique", "Montant à payer", "Motif d'ajustement", "Montant reçu", "Don", "'=HYPERLINK", "40,00", "5,00"} {
		if !strings.Contains(csv, want) {
			t.Errorf("CSV sans %q :\n%s", want, csv)
		}
	}
	if strings.Contains(csv, "Montant dû") {
		t.Error("ancien en-tête « Montant dû »")
	}
}
