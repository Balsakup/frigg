package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// refundNotification est la notification de remboursement de HelloAsso : le
// paiement HelloAsso, sans les métadonnées de Frigg.
func refundNotification(paymentID, orderID int64) string {
	return fmt.Sprintf(`{"eventType":"Payment","data":{"id":%d,"state":"Refunded","order":{"id":%d}}}`, paymentID, orderID)
}

func (f cardFixture) cancelledMails() int {
	n := 0
	for _, m := range f.env.Mail.Messages() {
		if m.To == f.email && strings.Contains(m.Msg.Subject, "annulée") {
			n++
		}
	}
	return n
}

// TestRefundCancelsMembership : un remboursement vérifié chez HelloAsso annule
// l'adhésion réglée par ce paiement, une seule fois.
func TestRefundCancelsMembership(t *testing.T) {
	f := newCardFixture(t)
	f.paidByCard(t, 3000)
	in := f.env.HelloAsso.LastIntent()
	body := refundNotification(in.PaymentID, in.OrderID)

	// Notification sans remboursement réel chez HelloAsso : rien ne change.
	f.env.Advance(10 * time.Second)
	if res := f.env.Client().Do("POST", notificationPath(), rawJSON(body)); res.Status != http.StatusOK {
		t.Fatalf("notification : %d", res.Status)
	}
	if got := f.mine(t); got.Status != "validated" || got.Payment.Status != "paid" {
		t.Fatalf("adhésion annulée sans remboursement vérifié : %s %+v", got.Status, got.Payment)
	}

	f.env.HelloAsso.Refund(in.ID)
	f.env.Advance(10 * time.Second)
	if res := f.env.Client().Do("POST", notificationPath(), rawJSON(body)); res.Status != http.StatusOK {
		t.Fatalf("notification de remboursement : %d", res.Status)
	}
	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), http.StatusOK)
	if got.Status != "cancelled" || got.Reason == nil || *got.Reason != "Paiement remboursé dans HelloAsso." ||
		got.OnlinePayments[0].Status != "refunded" || got.CancelledBy != nil {
		t.Fatalf("adhésion après remboursement : %s %v %+v", got.Status, got.Reason, got.OnlinePayments)
	}
	f.env.FlushMail()
	if f.cancelledMails() != 1 {
		t.Fatalf("e-mails d'annulation : %d", f.cancelledMails())
	}
	// Rejouée : aucun nouvel effet.
	f.env.Advance(10 * time.Second)
	f.env.Client().Do("POST", notificationPath(), rawJSON(body))
	f.env.FlushMail()
	if f.cancelledMails() != 1 {
		t.Fatalf("e-mails après rejeu : %d", f.cancelledMails())
	}
	if !strings.Contains(f.env.Logs.String(), `"msg":"paiement en ligne remboursé"`) {
		t.Fatal("remboursement non journalisé")
	}
}

// TestRefundAfterCancellationResolvesAlert : adhésion déjà annulée, le
// remboursement clôt le signalement « remboursement à effectuer ».
func TestRefundAfterCancellationResolvesAlert(t *testing.T) {
	f := newCardFixture(t)
	f.paidByCard(t, 3000)
	got := expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(f.ms.ID, "/cancel"), map[string]any{"reason": "Départ"}), http.StatusOK)
	if len(got.Alerts) != 1 || got.Alerts[0].Kind != "refund_due" {
		t.Fatalf("signalement : %+v", got.Alerts)
	}
	in := f.env.HelloAsso.LastIntent()
	f.env.HelloAsso.Refund(in.ID)
	f.env.Advance(10 * time.Second)
	f.env.Client().Do("POST", notificationPath(), rawJSON(refundNotification(in.PaymentID, in.OrderID)))
	got = expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), http.StatusOK)
	if got.Alerts[0].ResolvedAt == nil || got.Reason == nil || *got.Reason != "Départ" || got.OnlinePayments[0].Status != "refunded" {
		t.Fatalf("après remboursement : %+v, motif %v", got.Alerts, got.Reason)
	}
	f.env.FlushMail()
	if f.cancelledMails() != 1 { // seulement l'annulation par l'administrateur
		t.Fatalf("e-mails d'annulation : %d", f.cancelledMails())
	}
}

// TestRefundOfDuplicateKeepsMembership : le remboursement d'un paiement en
// double ne touche pas l'adhésion réglée par un autre moyen.
func TestRefundOfDuplicateKeepsMembership(t *testing.T) {
	f := newCardFixture(t)
	st := f.start(t)
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(f.ms.ID, "/payment"), map[string]any{
		"method": "cheque", "receivedOn": msToday,
	}), http.StatusOK)
	in := f.env.HelloAsso.LastIntent()
	f.env.HelloAsso.Authorize(in.ID, 3000, 0)
	f.confirm(t, f.member, st.OnlinePaymentID, "return")
	f.env.HelloAsso.Refund(in.ID)
	f.env.Advance(10 * time.Second)
	f.env.Client().Do("POST", notificationPath(), rawJSON(refundNotification(in.PaymentID, in.OrderID)))
	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), http.StatusOK)
	if got.Status != "validated" || *got.Payment.PaidMethod != "cheque" || got.Alerts[0].Kind != "duplicate" || got.Alerts[0].ResolvedAt == nil {
		t.Fatalf("doublon remboursé : %s %+v %+v", got.Status, got.Payment, got.Alerts)
	}
}

// TestAdminVerifyDetectsRefund : un remboursement déjà fait (notification
// perdue) est rattrapé par « Vérifier auprès de HelloAsso ».
func TestAdminVerifyDetectsRefund(t *testing.T) {
	f := newCardFixture(t)
	f.paidByCard(t, 3000)
	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), http.StatusOK)
	if !got.Actions.VerifyCardPayment {
		t.Fatal("vérification non proposée sur une adhésion payée par carte")
	}
	f.env.HelloAsso.Refund(f.env.HelloAsso.LastIntent().ID)
	f.env.Advance(10 * time.Second)
	got = expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(f.ms.ID, "/card-payment/verify"), nil), http.StatusOK)
	if got.Status != "cancelled" || got.OnlinePayments[0].Status != "refunded" || got.Actions.VerifyCardPayment {
		t.Fatalf("après vérification : %s %+v %+v", got.Status, got.OnlinePayments, got.Actions)
	}
}
