package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
)

func TestAdminRegistrations(t *testing.T) {
	f := newEvFixture(t)
	setPayment(t, f.admin, nil)
	ev := f.publishedEvent(t, eventBody("Stage", paidEvent("35"), func(b map[string]any) { b["capacity"] = 3 }))
	out := register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID)
	claireReg := out.CreatedRegistrationIDs[0]

	// Fiche sans compte, inscrite par l'administration hors période et déjà payée.
	if res := f.admin.Do("POST", "/api/admin/members", memberBody("Paul", "Durand", "2000-02-02", "accountId", nil)); res.Status != http.StatusCreated {
		t.Fatalf("fiche : %d %s", res.Status, res.Body)
	}
	members := expectJSON[struct {
		Items []api.AdminMember `json:"items"`
	}](t, f.admin.Do("GET", "/api/admin/members?q=Durand", nil), http.StatusOK)
	if len(members.Items) != 1 {
		t.Fatalf("fiche : %+v", members.Items)
	}
	paulID := members.Items[0].ID
	f.env.SetToday("2026-12-12") // après la clôture (début à 14 h, midi ici)
	reg := expectJSON[api.AdminRegistration](t, f.admin.Do("POST", eventPath(ev.ID, "/registrations"), map[string]any{
		"memberId": paulID, "payment": map[string]any{"method": "cash", "receivedOn": "2026-12-10"},
	}), http.StatusCreated)
	if reg.Payment == nil || reg.Payment.Status != "paid" || *reg.Payment.PaidMethod != "cash" || reg.ContactEmail != nil || reg.IsSeasonMember {
		t.Fatalf("inscription admin : %+v", reg)
	}

	// Lucas (non adhérent) est éligible sur un événement ouvert à tous.
	reg2 := expectJSON[api.AdminRegistration](t, f.admin.Do("POST", eventPath(ev.ID, "/registrations"), map[string]any{"memberId": f.lucas.ID}), http.StatusCreated)
	res := f.admin.Do("POST", eventPath(ev.ID, "/registrations"), map[string]any{"memberId": f.lucas.ID})
	if res.Status != http.StatusUnprocessableEntity || !strings.Contains(string(res.Body), "body.memberId") {
		t.Fatalf("déjà inscrit : %d %s", res.Status, res.Body)
	}
	if res := f.admin.Do("POST", "/api/admin/members", memberBody("Zoé", "Durand", "2001-03-03", "accountId", nil)); res.Status != http.StatusCreated {
		t.Fatalf("fiche : %d %s", res.Status, res.Body)
	}
	zoe := expectJSON[struct {
		Items []api.AdminMember `json:"items"`
	}](t, f.admin.Do("GET", "/api/admin/members?q=Zo%C3%A9", nil), http.StatusOK).Items[0]
	expectProblem(t, f.admin.Do("POST", eventPath(ev.ID, "/registrations"), map[string]any{"memberId": zoe.ID}),
		http.StatusConflict, "Complet : il reste 0 place(s).")

	// Passage en « adhérents de la saison » : Lucas signalé non éligible, pas retiré.
	expectAdminEvent(t, f.admin.Do("PUT", eventPath(ev.ID, ""), eventBody("Stage", func(b map[string]any) {
		b["price"], b["capacity"] = "35", 3
	})), http.StatusOK)
	list := expectJSON[struct {
		Items []api.AdminRegistration `json:"items"`
	}](t, f.admin.Do("GET", eventPath(ev.ID, "/registrations"), nil), http.StatusOK)
	if len(list.Items) != 3 {
		t.Fatalf("inscrits : %d", len(list.Items))
	}
	for _, r := range list.Items {
		if r.ID == reg2.ID && r.Eligible {
			t.Fatal("Lucas devrait être signalé non éligible")
		}
		if r.ID == claireReg && (!r.Eligible || r.ContactEmail == nil || *r.ContactEmail != f.claireEmail) {
			t.Fatalf("Claire : %+v", r)
		}
	}

	// Totaux.
	got := expectAdminEvent(t, f.admin.Do("GET", eventPath(ev.ID, ""), nil), http.StatusOK)
	if got.Totals.DueCents != 10500 || got.Totals.CollectedCents != 3500 || got.Totals.PaidCount != 1 {
		t.Fatalf("totaux : %+v", got.Totals)
	}

	// Constatation d'un paiement, puis annulation de la constatation.
	paid := expectJSON[api.AdminRegistration](t, f.admin.Do("POST", adminRegistrationPath(claireReg, "/payment"), map[string]any{
		"method": "cheque", "receivedOn": "2026-12-11",
	}), http.StatusOK)
	if paid.Payment.Status != "paid" || !paid.Actions.CancelPayment {
		t.Fatalf("constatée : %+v", paid)
	}
	if n := mails(f.env, f.claireEmail, "Paiement reçu"); n != 1 {
		t.Fatalf("e-mails de constatation : %d", n)
	}
	back := expectJSON[api.AdminRegistration](t, f.admin.Do("DELETE", adminRegistrationPath(claireReg, "/payment"), nil), http.StatusOK)
	if back.Payment.Status != "due" {
		t.Fatalf("annulation de constatation : %+v", back.Payment)
	}

	// Retrait : e-mail, place libérée.
	removed := expectJSON[api.AdminRegistration](t, f.admin.Do("POST", adminRegistrationPath(claireReg, "/cancel"), nil), http.StatusOK)
	if removed.State != "cancelled" || removed.CancelledBy == nil {
		t.Fatalf("retrait : %+v", removed)
	}
	if n := mails(f.env, f.claireEmail, "Inscription annulée"); n != 1 {
		t.Fatalf("e-mails de retrait : %d", n)
	}
	all := expectJSON[struct {
		Items []api.AdminRegistration `json:"items"`
	}](t, f.admin.Do("GET", eventPath(ev.ID, "/registrations?state=all"), nil), http.StatusOK)
	if len(all.Items) != 3 {
		t.Fatalf("avec les annulées : %d", len(all.Items))
	}

	// Rubrique « Inscriptions » : filtres et recherche.
	page := func(q string) api.AdminRegistrationPage {
		return expectJSON[api.AdminRegistrationPage](t, f.admin.Do("GET", "/api/admin/registrations"+q, nil), http.StatusOK)
	}
	if p := page("?q=EVT-26"); p.Total != 3 {
		t.Fatalf("recherche par référence : %d", p.Total)
	}
	if p := page("?q=durand"); p.Total != 1 || p.Items[0].Member.LastName != "Durand" {
		t.Fatalf("recherche par nom : %+v", p)
	}
	if p := page("?state=cancelled"); p.Total != 1 {
		t.Fatalf("filtre état : %d", p.Total)
	}
	if p := page("?payment=paid"); p.Total != 1 {
		t.Fatalf("filtre paiement : %d", p.Total)
	}
	if res := f.admin.Do("GET", "/api/admin/registrations?q=du", nil); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("recherche trop courte : %d", res.Status)
	}

	// Après la fin, plus de retrait.
	f.env.SetToday("2026-12-13")
	expectProblem(t, f.admin.Do("POST", adminRegistrationPath(reg2.ID, "/cancel"), nil), http.StatusConflict, api.MsgRegistrationChanged)

	// Fiche : nombre d'inscriptions payées à venir.
	f.env.SetToday("2026-12-12")
	res = f.admin.Do("GET", adminMemberPath(paulID), nil)
	if !strings.Contains(string(res.Body), `"paidUpcomingRegistrations":1`) {
		t.Fatalf("fiche : %s", res.Body)
	}
	noLeak(t, f.env, "Durand", "Claire", "Stage")
}

func TestRegistrationAlertsOnDashboard(t *testing.T) {
	f := newEvCardFixture(t)
	st := f.start(t)
	in := f.env.HelloAsso.LastIntent()
	if _, err := f.env.Store.WriteDB.Exec(`UPDATE event_registrations SET status = 'cancelled', cancelled_at = '2026-11-01T11:00:00Z' WHERE id = ?`, f.reg); err != nil {
		t.Fatal(err)
	}
	f.env.HelloAsso.Authorize(in.ID, 3500, 0)
	f.env.Client().Do("POST", notificationPath(), rawJSON(registrationNotification(st.OnlinePaymentID, f.reg)))

	dash := expectJSON[api.MembershipDashboard](t, f.admin.Do("GET", "/api/admin/memberships/dashboard", nil), http.StatusOK)
	if dash.PaymentAlerts != 0 || dash.RegistrationPaymentAlerts != 1 {
		t.Fatalf("tableau de bord : %+v", dash)
	}
	p := expectJSON[api.AdminRegistrationPage](t, f.admin.Do("GET", "/api/admin/registrations?alerts=open", nil), http.StatusOK)
	if p.Total != 1 || p.Items[0].OpenAlerts != 1 {
		t.Fatalf("filtre signalements : %+v", p)
	}
	detail := expectJSON[api.AdminRegistration](t, f.admin.Do("GET", adminRegistrationPath(f.reg, ""), nil), http.StatusOK)
	if len(detail.Alerts) != 1 || len(detail.OnlinePayments) != 1 || !detail.Actions.VerifyCardPayment {
		t.Fatalf("détail : %+v", detail)
	}
	// Le signalement d'inscription a son opération ; l'ancienne le refuse.
	expectProblem(t, f.admin.Do("POST", "/api/admin/payment-alerts/"+itoa(detail.Alerts[0].ID)+"/resolve", nil), http.StatusNotFound, api.MsgAlertGone)
	resolved := expectJSON[api.AdminRegistration](t, f.admin.Do("POST", "/api/admin/registration-alerts/"+itoa(detail.Alerts[0].ID)+"/resolve", nil), http.StatusOK)
	if resolved.OpenAlerts != 0 {
		t.Fatalf("traité : %+v", resolved)
	}
	if p := expectJSON[api.AdminRegistrationPage](t, f.admin.Do("GET", "/api/admin/registrations?alerts=open", nil), http.StatusOK); p.Total != 0 {
		t.Fatalf("après traitement : %d", p.Total)
	}
	expectJSON[api.AdminRegistration](t, f.admin.Do("POST", adminRegistrationPath(f.reg, "/card-payment/verify"), nil), http.StatusOK)
}
