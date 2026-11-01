package membership

import (
	"errors"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/season"
)

func TestAppliedAmount(t *testing.T) {
	adult := season.Fee{AmountCents: 6000, RenewalAmountCents: new(int64(4000))}
	young := season.Fee{AmountCents: 3000}
	free := season.Fee{AmountCents: 0}
	cases := []struct {
		name string
		fee  season.Fee
		kind Kind
		want int64
	}{
		{"première adhésion", adult, First, 6000},
		{"renouvellement", adult, Renewal, 4000},
		{"renouvellement sans montant distinct", young, Renewal, 3000},
		{"gratuit", free, Renewal, 0},
	}
	for _, c := range cases {
		if got := AppliedAmount(c.fee, c.kind); got != c.want {
			t.Errorf("%s : %d, attendu %d", c.name, got, c.want)
		}
	}
}

func TestActions(t *testing.T) {
	cases := []struct {
		name   string
		s      Snapshot
		member MemberActions
		admin  AdminActions
	}{
		{"en attente", Snapshot{Status: Pending},
			MemberActions{Cancel: true}, AdminActions{Validate: true, Refuse: true}},
		{"en attente, saison close", Snapshot{Status: Pending, SeasonClosed: true},
			MemberActions{Cancel: true}, AdminActions{Validate: true, Refuse: true}},
		{"validée à payer", Snapshot{Status: Validated, PaymentStatus: Due},
			MemberActions{Pay: true}, AdminActions{Update: true, Cancel: true, RecordPayment: true}},
		{"validée annoncée", Snapshot{Status: Validated, PaymentStatus: Announced},
			MemberActions{Pay: true}, AdminActions{Update: true, Cancel: true, RecordPayment: true}},
		{"validée payée", Snapshot{Status: Validated, PaymentStatus: Paid},
			MemberActions{}, AdminActions{Cancel: true, CancelPayment: true}},
		{"validée gratuite", Snapshot{Status: Validated},
			MemberActions{}, AdminActions{Update: true, Cancel: true}},
		{"validée à payer, saison close", Snapshot{Status: Validated, PaymentStatus: Due, SeasonClosed: true},
			MemberActions{Pay: true}, AdminActions{RecordPayment: true}},
		{"validée payée, saison close", Snapshot{Status: Validated, PaymentStatus: Paid, SeasonClosed: true},
			MemberActions{}, AdminActions{CancelPayment: true}},
		{"refusée", Snapshot{Status: Refused}, MemberActions{}, AdminActions{}},
		{"annulée payée", Snapshot{Status: Cancelled, PaymentStatus: Paid}, MemberActions{}, AdminActions{}},
	}
	for _, c := range cases {
		if got := MemberActionsFor(c.s); got != c.member {
			t.Errorf("%s : actions membre %+v, attendu %+v", c.name, got, c.member)
		}
		if got := AdminActionsFor(c.s); got != c.admin {
			t.Errorf("%s : actions admin %+v, attendu %+v", c.name, got, c.admin)
		}
	}
}

func TestPaymentAfter(t *testing.T) {
	cases := []struct {
		name          string
		amount        int64
		current       PaymentStatus
		announced     Method
		wantStatus    PaymentStatus
		wantAnnounced Method
	}{
		{"gratuit", 0, NoPayment, "", NoPayment, ""},
		{"devenu gratuit", 0, Announced, Transfer, NoPayment, ""},
		{"nouveau paiement", 3000, NoPayment, "", Due, ""},
		{"moyen annoncé conservé", 3000, Announced, Cheque, Announced, Cheque},
	}
	for _, c := range cases {
		s, m := PaymentAfter(c.amount, c.current, c.announced)
		if s != c.wantStatus || m != c.wantAnnounced {
			t.Errorf("%s : %q %q, attendu %q %q", c.name, s, m, c.wantStatus, c.wantAnnounced)
		}
	}
}

func TestCleanReason(t *testing.T) {
	if got, err := CleanReason("  doublon   de fiche  "); err != nil || got != "doublon de fiche" {
		t.Errorf("nettoyage : %q, %v", got, err)
	}
	if got, _ := CleanReason("ligne 1\n  ligne   2 \n\n"); got != "ligne 1\nligne 2" {
		t.Errorf("multiligne : %q", got)
	}
	if _, err := CleanReason("   \n "); !errors.Is(err, ErrReasonRequired) {
		t.Errorf("vide : %v", err)
	}
	if _, err := CleanReason(strings.Repeat("é", 500)); err != nil {
		t.Errorf("500 caractères : %v", err)
	}
	if _, err := CleanReason(strings.Repeat("é", 501)); !errors.Is(err, ErrReasonTooLong) {
		t.Errorf("501 caractères : %v", err)
	}
	if got, err := CleanNote("  "); err != nil || got != "" {
		t.Errorf("commentaire vide : %q, %v", got, err)
	}
	if _, err := CleanNote(strings.Repeat("a", 201)); !errors.Is(err, ErrNoteTooLong) {
		t.Errorf("commentaire trop long : %v", err)
	}
}
