package membership

import (
	"testing"
	"time"
)

func TestSettleOnline(t *testing.T) {
	due := Snapshot{Status: Validated, PaymentStatus: Due, AmountCents: 4000}
	announced := Snapshot{Status: Validated, PaymentStatus: Announced, AmountCents: 4000}
	paid := Snapshot{Status: Validated, PaymentStatus: Paid, AmountCents: 4000, PaidMethod: Cheque}
	cancelled := Snapshot{Status: Cancelled, PaymentStatus: Due, AmountCents: 4000}
	ok := Verified{Authorized: true, ReceivedCents: 4000, PaidAt: time.Now()}
	cases := []struct {
		name    string
		attempt OnlineStatus
		m       Snapshot
		v       Verified
		want    Settlement
	}{
		{"non autorisé, sans indice", OnlinePending, due, Verified{}, Settlement{NewStatus: OnlinePending}},
		{"non autorisé, retour erreur", OnlinePending, due, Verified{Hint: HintError}, Settlement{NewStatus: OnlineFailed}},
		{"non autorisé, retour annulé", OnlinePending, due, Verified{Hint: HintBack}, Settlement{NewStatus: OnlineAbandoned}},
		{"non autorisé, retour succès falsifié", OnlinePending, due, Verified{Hint: HintReturn}, Settlement{NewStatus: OnlinePending}},
		{"non autorisé, déjà échouée", OnlineFailed, due, Verified{Hint: HintBack}, Settlement{NewStatus: OnlineFailed}},
		{"autorisé, à payer", OnlinePending, due, ok, Settlement{NewStatus: OnlineSucceeded, RecordPayment: true}},
		{"autorisé, annoncée", OnlinePending, announced, ok, Settlement{NewStatus: OnlineSucceeded, RecordPayment: true}},
		{"autorisé, montant différent", OnlinePending, due, Verified{Authorized: true, ReceivedCents: 6000},
			Settlement{NewStatus: OnlineSucceeded, RecordPayment: true, Alert: AlertAmountMismatch}},
		{"autorisé, déjà payée", OnlinePending, paid, ok, Settlement{NewStatus: OnlineSucceeded, Alert: AlertDuplicate}},
		{"autorisé, annulée", OnlinePending, cancelled, ok, Settlement{NewStatus: OnlineSucceeded, Alert: AlertRefundDue}},
		{"échouée puis autorisé", OnlineFailed, due, ok, Settlement{NewStatus: OnlineSucceeded, RecordPayment: true}},
		{"expirée puis autorisé", OnlineExpired, due, ok, Settlement{NewStatus: OnlineSucceeded, RecordPayment: true}},
		{"réussie rejouée", OnlineSucceeded, paid, ok, Settlement{NewStatus: OnlineSucceeded}},
	}
	for _, c := range cases {
		if got := SettleOnline(c.attempt, c.m, c.v); got != c.want {
			t.Errorf("%s : %+v, attendu %+v", c.name, got, c.want)
		}
	}
}

func TestCardAvailability(t *testing.T) {
	cases := []struct {
		name                string
		configured, enabled bool
		testedEnv           string
		testOK              bool
		want                bool
	}{
		{"disponible", true, true, "sandbox", true, true},
		{"non configuré", false, true, "sandbox", true, false},
		{"désactivé", true, false, "sandbox", true, false},
		{"jamais testé", true, true, "", false, false},
		{"test en échec", true, true, "sandbox", false, false},
		{"autre environnement testé", true, true, "production", true, false},
	}
	for _, c := range cases {
		if got := CardAvailability(c.configured, c.enabled, c.testedEnv, c.testOK, "sandbox"); got != c.want {
			t.Errorf("%s : %v", c.name, got)
		}
	}
}

func TestCardActions(t *testing.T) {
	due := Snapshot{Status: Validated, PaymentStatus: Due, AmountCents: 4000}
	if !MemberActionsFor(due, true).PayByCard {
		t.Error("carte non proposée sur une adhésion à payer")
	}
	if MemberActionsFor(due, false).PayByCard {
		t.Error("carte proposée alors qu'elle est indisponible")
	}
	if CanPayByCard(Snapshot{Status: Validated, PaymentStatus: Due, AmountCents: MinCardAmount - 1}, true) {
		t.Error("carte proposée sous le montant minimal")
	}
	if CanPayByCard(Snapshot{Status: Validated, PaymentStatus: Paid, AmountCents: 4000}, true) {
		t.Error("carte proposée sur une adhésion payée")
	}
	card := Snapshot{Status: Validated, PaymentStatus: Paid, AmountCents: 4000, PaidMethod: Card}
	if CanCancelPayment(card) || AdminActionsFor(card).CancelPayment {
		t.Error("constatation d'un paiement par carte annulable")
	}
	if !CanCancelPayment(Snapshot{Status: Validated, PaymentStatus: Paid, PaidMethod: Cheque}) {
		t.Error("constatation d'un chèque non annulable")
	}
	for m, want := range map[Method]bool{Cash: true, Cheque: true, Transfer: true, Card: false, "x": false} {
		if ManualMethod(m) != want {
			t.Errorf("ManualMethod(%s) = %v", m, !want)
		}
	}
}

func TestRefundCancels(t *testing.T) {
	paid := Snapshot{Status: Validated, PaymentStatus: Paid, PaidMethod: Card, AmountCents: 3000}
	cases := []struct {
		name    string
		attempt OnlineStatus
		m       Snapshot
		byIt    bool
		want    bool
	}{
		{"adhésion payée par cette tentative", OnlineSucceeded, paid, true, true},
		{"paiement en double remboursé", OnlineSucceeded, paid, false, false},
		{"adhésion déjà annulée", OnlineSucceeded, Snapshot{Status: Cancelled, PaymentStatus: Paid, PaidMethod: Card}, true, false},
		{"tentative déjà remboursée", OnlineRefunded, paid, true, false},
		{"payée par chèque", OnlineSucceeded, Snapshot{Status: Validated, PaymentStatus: Paid, PaidMethod: Cheque}, true, false},
	}
	for _, c := range cases {
		if got := RefundCancels(c.attempt, c.m, c.byIt); got != c.want {
			t.Errorf("%s : %v", c.name, got)
		}
	}
}
