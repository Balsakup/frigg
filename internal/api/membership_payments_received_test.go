package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
)

// Montant reçu et dons sur les adhésions (specs/014-payment-adjustments US2).
func TestRecordPaymentWithDonation(t *testing.T) {
	f := newMsFixture(t)
	setPayment(t, f.admin, nil)
	c, email, ms := pendingRequest(t, f)
	validate(t, f.admin, ms.ID, f.adult.ID, "renewal") // 40 €
	payPath := adminMembershipPath(ms.ID, "/payment")
	pay := func(received string) map[string]any {
		b := map[string]any{"method": "transfer", "receivedOn": msToday}
		if received != "" {
			b["receivedAmount"] = received
		}
		return b
	}

	expectFieldError(t, f.admin.Do("POST", payPath, pay("35")), "body.receivedAmount", api.MsgReceivedBelowDue)
	expectFieldError(t, f.admin.Do("POST", payPath, pay("trente")), "body.receivedAmount", api.MsgAmountFormat)

	got := expectAdminMembership(t, f.admin.Do("POST", payPath, pay("75")), http.StatusOK)
	p := got.Payment
	if p == nil || p.Status != "paid" || p.PaidAmountCents == nil || *p.PaidAmountCents != 7500 || p.DonationCents != 3500 {
		t.Fatalf("paiement avec don : %+v", p)
	}
	f.env.FlushMail()
	sent, ok := f.env.Mail.Last(email)
	if !ok || !strings.Contains(sent.Msg.Text, "75,00") || !strings.Contains(sent.Msg.Text, "Dont 35,00") || !strings.Contains(sent.Msg.Text, "de don : merci") {
		t.Fatalf("e-mail de paiement : %q", sent.Msg.Text)
	}
	mine := expectMine(t, c.Do("GET", myMembershipPath(ms.ID), nil), http.StatusOK)
	if mine.Payment == nil || mine.Payment.DonationCents != 3500 {
		t.Fatalf("don côté membre : %+v", mine.Payment)
	}
	if !strings.Contains(f.env.Logs.String(), `"received_cents":7500`) || !strings.Contains(f.env.Logs.String(), `"donation_cents":3500`) {
		t.Fatal("montant reçu non journalisé")
	}

	// Annulation de la constatation : montant reçu effacé.
	got = expectAdminMembership(t, f.admin.Do("DELETE", payPath, nil), http.StatusOK)
	if got.Payment.PaidAmountCents != nil || got.Payment.DonationCents != 0 {
		t.Fatalf("après annulation : %+v", got.Payment)
	}
	// Sans montant saisi : le montant à payer, sans don.
	got = expectAdminMembership(t, f.admin.Do("POST", payPath, pay("")), http.StatusOK)
	if *got.Payment.PaidAmountCents != 4000 || got.Payment.DonationCents != 0 {
		t.Fatalf("montant par défaut : %+v", got.Payment)
	}
}
