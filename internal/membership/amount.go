package membership

import (
	"errors"
	"unicode/utf8"
)

// Remises et dons (research.md 014 › R3) : montant à payer ajusté par un
// administrateur, montant reçu et chiffres. Règles partagées par les
// adhésions et les inscriptions.

// MaxAdjustmentReason est la longueur maximale du motif d'un ajustement.
const MaxAdjustmentReason = 200

var (
	ErrAdjustmentReasonRequired = errors.New("motif de l'ajustement manquant")
	ErrAdjustmentReasonTooLong  = errors.New("motif de l'ajustement trop long")
	// ErrReceivedBelowDue : pas de paiement partiel, une remise passe par un
	// ajustement motivé du montant à payer.
	ErrReceivedBelowDue = errors.New("montant reçu inférieur au montant à payer")
)

// Adjustment est le montant à payer retenu. BaseCents et Reason ne sont
// renseignés que s'il diffère du montant théorique.
type Adjustment struct {
	AmountCents int64
	BaseCents   *int64
	Reason      string
}

// Adjust calcule le montant à payer à partir du montant théorique base, du
// montant demandé (nil : le théorique) et du motif.
func Adjust(base int64, requested *int64, reason string) (Adjustment, error) {
	if requested == nil || *requested == base {
		return Adjustment{AmountCents: base}, nil
	}
	c := cleanText(reason)
	switch {
	case c == "":
		return Adjustment{}, ErrAdjustmentReasonRequired
	case utf8.RuneCountInString(c) > MaxAdjustmentReason:
		return Adjustment{}, ErrAdjustmentReasonTooLong
	}
	b := base
	return Adjustment{AmountCents: *requested, BaseCents: &b, Reason: c}, nil
}

// Theoretical renvoie le montant théorique : base s'il y a un ajustement,
// sinon le montant à payer.
func Theoretical(amount int64, base *int64) int64 {
	if base != nil {
		return *base
	}
	return amount
}

// AdjustmentOf renvoie l'ajustement signé : négatif pour une remise, positif
// pour une majoration, 0 sans ajustement.
func AdjustmentOf(amount int64, base *int64) int64 {
	if base == nil {
		return 0
	}
	return amount - *base
}

// CheckReceived refuse un montant reçu inférieur au montant à payer.
func CheckReceived(due, received int64) error {
	if received < due {
		return ErrReceivedBelowDue
	}
	return nil
}

// Donation renvoie le don d'un paiement : l'excédent du montant reçu sur le
// montant à payer pour un moyen manuel, 0 pour la carte (le montant reçu par
// carte est constaté par HelloAsso, hors contribution volontaire).
func Donation(m Method, due, received int64) int64 {
	if !ManualMethod(m) || received <= due {
		return 0
	}
	return received - due
}

// AmountRow est une adhésion validée ou une inscription confirmée, réduite à
// ses montants.
type AmountRow struct {
	AmountCents     int64
	BaseCents       *int64
	PaymentStatus   PaymentStatus
	AnnouncedMethod Method
	PaidMethod      Method
	// PaidAmountCents : montant reçu d'une ligne payée.
	PaidAmountCents int64
	// PaidTest : payée par carte en mode test.
	PaidTest bool
}

// AmountTotals sont les montants cumulés (data-model.md 014 › AmountTotals).
// Theoretical − Discounts + Surcharges = Due.
type AmountTotals struct {
	Theoretical, Discounts, Surcharges, Due int64
	// Collected : montants reçus hors mode test, dons compris.
	Collected         int64
	CollectedByMethod PaidByMethod
	CollectedTest     int64
	Donations         int64
	// Remaining : montants à payer non encore payés.
	Remaining         int64
	RemainingByMethod DueByMethod
}

// Add ajoute une ligne aux totaux.
func (t *AmountTotals) Add(r AmountRow) {
	t.Theoretical += Theoretical(r.AmountCents, r.BaseCents)
	switch adj := AdjustmentOf(r.AmountCents, r.BaseCents); {
	case adj < 0:
		t.Discounts -= adj
	case adj > 0:
		t.Surcharges += adj
	}
	t.Due += r.AmountCents
	switch r.PaymentStatus {
	case Paid:
		received := r.PaidAmountCents
		if r.PaidTest {
			t.CollectedTest += received
			return
		}
		t.Collected += received
		t.Donations += Donation(r.PaidMethod, r.AmountCents, received)
		switch r.PaidMethod {
		case Card:
			t.CollectedByMethod.Card += received
		case Cash:
			t.CollectedByMethod.Cash += received
		case Cheque:
			t.CollectedByMethod.Cheque += received
		case Transfer:
			t.CollectedByMethod.Transfer += received
		}
	case Due, Announced:
		t.Remaining += r.AmountCents
		switch r.AnnouncedMethod {
		case Cash:
			t.RemainingByMethod.Cash += r.AmountCents
		case Cheque:
			t.RemainingByMethod.Cheque += r.AmountCents
		case Transfer:
			t.RemainingByMethod.Transfer += r.AmountCents
		default:
			t.RemainingByMethod.None += r.AmountCents
		}
	}
}
