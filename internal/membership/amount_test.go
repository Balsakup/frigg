package membership

import (
	"errors"
	"strings"
	"testing"
)

func cents(v int64) *int64 { return &v }

func TestAdjust(t *testing.T) {
	tests := []struct {
		name      string
		requested *int64
		reason    string
		want      Adjustment
		err       error
	}{
		{name: "absent : théorique", want: Adjustment{AmountCents: 7500}},
		{name: "égal : motif ignoré", requested: cents(7500), reason: "remise", want: Adjustment{AmountCents: 7500}},
		{name: "remise", requested: cents(7000), reason: "  réduction   de 5 € ", want: Adjustment{AmountCents: 7000, BaseCents: cents(7500), Reason: "réduction de 5 €"}},
		{name: "majoration", requested: cents(9000), reason: "soutien", want: Adjustment{AmountCents: 9000, BaseCents: cents(7500), Reason: "soutien"}},
		{name: "gratuit", requested: cents(0), reason: "bénévole", want: Adjustment{AmountCents: 0, BaseCents: cents(7500), Reason: "bénévole"}},
		{name: "motif vide", requested: cents(7000), err: ErrAdjustmentReasonRequired},
		{name: "motif d'espaces", requested: cents(7000), reason: " \n ", err: ErrAdjustmentReasonRequired},
		{name: "200 caractères", requested: cents(7000), reason: strings.Repeat("é", 200), want: Adjustment{AmountCents: 7000, BaseCents: cents(7500), Reason: strings.Repeat("é", 200)}},
		{name: "201 caractères", requested: cents(7000), reason: strings.Repeat("é", 201), err: ErrAdjustmentReasonTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Adjust(7500, tt.requested, tt.reason)
			if !errors.Is(err, tt.err) {
				t.Fatalf("erreur %v, attendu %v", err, tt.err)
			}
			if err != nil {
				return
			}
			if got.AmountCents != tt.want.AmountCents || got.Reason != tt.want.Reason ||
				(got.BaseCents == nil) != (tt.want.BaseCents == nil) ||
				(got.BaseCents != nil && *got.BaseCents != *tt.want.BaseCents) {
				t.Errorf("%+v, attendu %+v", got, tt.want)
			}
		})
	}
}

func TestTheoreticalAndAdjustment(t *testing.T) {
	if Theoretical(7000, nil) != 7000 || Theoretical(7000, cents(7500)) != 7500 {
		t.Error("montant théorique")
	}
	if AdjustmentOf(7000, nil) != 0 || AdjustmentOf(7000, cents(7500)) != -500 || AdjustmentOf(9000, cents(7500)) != 1500 {
		t.Error("ajustement")
	}
}

func TestCheckReceivedAndDonation(t *testing.T) {
	if CheckReceived(4500, 4000) != ErrReceivedBelowDue || CheckReceived(4500, 4500) != nil || CheckReceived(4500, 8000) != nil {
		t.Error("CheckReceived")
	}
	tests := []struct {
		m             Method
		due, received int64
		want          int64
	}{
		{Cash, 4500, 5000, 500},
		{Transfer, 4500, 8000, 3500},
		{Cheque, 4500, 4500, 0},
		{Card, 4500, 5000, 0},
	}
	for _, tt := range tests {
		if got := Donation(tt.m, tt.due, tt.received); got != tt.want {
			t.Errorf("Donation(%s, %d, %d) = %d, attendu %d", tt.m, tt.due, tt.received, got, tt.want)
		}
	}
}

func TestAmountTotals(t *testing.T) {
	rows := []AmountRow{
		// Remise payée : 70 € au lieu de 75 €.
		{AmountCents: 7000, BaseCents: cents(7500), PaymentStatus: Paid, PaidMethod: Transfer, PaidAmountCents: 7000},
		// Majoration annoncée par chèque : 50 € au lieu de 45 €.
		{AmountCents: 5000, BaseCents: cents(4500), PaymentStatus: Announced, AnnouncedMethod: Cheque},
		// Payée avec un don de 35 €.
		{AmountCents: 4500, PaymentStatus: Paid, PaidMethod: Transfer, PaidAmountCents: 8000},
		// Gratuite par ajustement.
		{AmountCents: 0, BaseCents: cents(4500)},
		// Carte en mode test.
		{AmountCents: 4500, PaymentStatus: Paid, PaidMethod: Card, PaidAmountCents: 4500, PaidTest: true},
		// Carte réelle avec écart : pas de don.
		{AmountCents: 4500, PaymentStatus: Paid, PaidMethod: Card, PaidAmountCents: 5000},
		// Due sans moyen.
		{AmountCents: 3000, PaymentStatus: Due},
	}
	var got AmountTotals
	for _, r := range rows {
		got.Add(r)
	}
	want := AmountTotals{
		Theoretical: 7500 + 4500 + 4500 + 4500 + 4500 + 4500 + 3000,
		Discounts:   500 + 4500,
		Surcharges:  500,
		Due:         7000 + 5000 + 4500 + 0 + 4500 + 4500 + 3000,
		Collected:   7000 + 8000 + 5000,
		CollectedByMethod: PaidByMethod{
			Transfer: 15000, Card: 5000,
		},
		CollectedTest:     4500,
		Donations:         3500,
		Remaining:         8000,
		RemainingByMethod: DueByMethod{Cheque: 5000, None: 3000},
	}
	if got != want {
		t.Errorf("totaux\n  obtenu  %+v\n  attendu %+v", got, want)
	}
	if got.Theoretical-got.Discounts+got.Surcharges != got.Due {
		t.Errorf("théorique − remises + majorations ≠ à payer : %+v", got)
	}
}
