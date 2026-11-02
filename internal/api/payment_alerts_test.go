package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/auth"
)

// paidByCard paie l'adhésion de la fixture par carte (faux HelloAsso).
func (f cardFixture) paidByCard(t *testing.T, amount int64) api.StartCardPayment {
	t.Helper()
	st := f.start(t)
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, amount, 0)
	if _, out := f.confirm(t, f.member, st.OnlinePaymentID, "return"); out.Status != "succeeded" {
		t.Fatalf("paiement par carte : %+v", out)
	}
	return st
}

func TestAdminCardPaymentDetail(t *testing.T) {
	f := newCardFixture(t)
	f.paidByCard(t, 3000)
	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), http.StatusOK)
	if len(got.OnlinePayments) != 1 {
		t.Fatalf("tentatives : %+v", got.OnlinePayments)
	}
	op := got.OnlinePayments[0]
	if op.Status != "succeeded" || op.OrderID == nil || *op.ReceivedAmountCents != 3000 || op.PaidAt == nil ||
		op.HelloAssoURL == nil || !strings.Contains(*op.HelloAssoURL, "helloasso-sandbox.com") {
		t.Fatalf("tentative : %+v", op)
	}
	// Vérification proposée : un remboursement a pu être fait dans HelloAsso.
	if got.Actions.CancelPayment || !got.Actions.VerifyCardPayment || got.CardPaymentPending {
		t.Fatalf("actions : %+v, en cours %v", got.Actions, got.CardPaymentPending)
	}
}

func TestAdminVerifyCardPayment(t *testing.T) {
	f := newCardFixture(t)
	f.start(t)
	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), http.StatusOK)
	if !got.CardPaymentPending || !got.Actions.VerifyCardPayment {
		t.Fatalf("tentative en cours : %v %+v", got.CardPaymentPending, got.Actions)
	}
	f.env.HelloAsso.Down(true)
	if res := f.admin.Do("POST", adminMembershipPath(f.ms.ID, "/card-payment/verify"), nil); res.Status != http.StatusServiceUnavailable {
		t.Fatalf("vérification HelloAsso injoignable : %d", res.Status)
	}
	f.env.HelloAsso.Down(false)
	f.env.Advance(10 * time.Second)
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3000, 0)
	got = expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(f.ms.ID, "/card-payment/verify"), nil), http.StatusOK)
	if got.Payment.Status != "paid" || *got.Payment.PaidMethod != "card" {
		t.Fatalf("vérification : %+v", got.Payment)
	}
}

func TestCancelPaidByCardAndResolve(t *testing.T) {
	f := newCardFixture(t)
	f.paidByCard(t, 3000)
	got := expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(f.ms.ID, "/cancel"), map[string]any{"reason": "Départ"}), http.StatusOK)
	if len(got.Alerts) != 1 || got.Alerts[0].Kind != "refund_due" || got.Alerts[0].ResolvedAt != nil {
		t.Fatalf("signalement : %+v", got.Alerts)
	}
	dash := f.admin.Do("GET", "/api/admin/memberships/dashboard", nil)
	if !strings.Contains(dash.Body, `"paymentAlerts":1`) {
		t.Fatalf("tableau de bord : %s", dash.Body)
	}
	page := adminMemberships(t, f.admin, "?alerts=open")
	if page.Total != 1 || page.Items[0].OpenAlerts != 1 {
		t.Fatalf("liste filtrée : %+v", page)
	}

	path := fmt.Sprintf("/api/admin/payment-alerts/%d/resolve", got.Alerts[0].ID)
	member := f.env.LoggedIn(f.env.Seed(auth.RoleMember, true))
	if res := member.Do("POST", path, nil); res.Status != http.StatusForbidden {
		t.Fatalf("traitement par un membre : %d", res.Status)
	}
	got = expectAdminMembership(t, f.admin.Do("POST", path, nil), http.StatusOK)
	if got.Alerts[0].ResolvedAt == nil || got.Alerts[0].ResolvedBy == nil {
		t.Fatalf("signalement traité : %+v", got.Alerts[0])
	}
	if res := f.admin.Do("POST", path, nil); res.Status != http.StatusConflict {
		t.Fatalf("déjà traité : %d", res.Status)
	}
	if res := f.admin.Do("POST", "/api/admin/payment-alerts/99999/resolve", nil); res.Status != http.StatusNotFound {
		t.Fatalf("inexistant : %d", res.Status)
	}
	if adminMemberships(t, f.admin, "?alerts=open").Total != 0 {
		t.Fatal("signalement traité encore listé")
	}
}

func TestCardListAndStats(t *testing.T) {
	f := newCardFixture(t)
	f.paidByCard(t, 3000)
	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), http.StatusOK)
	order := *got.OnlinePayments[0].OrderID

	// Une seconde adhésion payée par chèque.
	c, _, m := f.msFixture.member(t, "Zoé")
	ms := request(t, c, m.ID, f.season.ID, f.adult.ID, "first")
	validate(t, f.admin, ms.ID, f.adult.ID, "first")
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(ms.ID, "/payment"), map[string]any{
		"method": "cheque", "receivedOn": msToday,
	}), http.StatusOK)

	page := adminMemberships(t, f.admin, "?paidMethod=card")
	if page.Total != 1 || page.Items[0].ID != f.ms.ID || page.Items[0].Payment.Online == nil || !page.Items[0].Payment.Online.Test {
		t.Fatalf("filtre carte : %+v", page)
	}
	if page := adminMemberships(t, f.admin, fmt.Sprintf("?q=%d", order)); page.Total != 1 || page.Items[0].ID != f.ms.ID {
		t.Fatalf("recherche par commande : %+v", page)
	}

	res := f.admin.Do("GET", fmt.Sprintf("/api/admin/memberships/stats?season=%d", f.season.ID), nil)
	var st api.MembershipStats
	res.JSON(t, &st)
	// Le paiement par carte est en mode test : exclu de l'encaissé réel.
	if st.PaidCents != 6000 || st.CollectedByMethod.Cheque != 6000 || st.CollectedByMethod.Card != 0 || st.CollectedTest != 3000 {
		t.Fatalf("chiffres : %+v", st)
	}
}
