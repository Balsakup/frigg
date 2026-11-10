package api

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
)

// Remises et dons (specs/014-payment-adjustments) : analyse des montants
// saisis en euros et traduction des erreurs du domaine.

const (
	MsgAdjustmentReasonRequired = "Indiquez le motif de l'ajustement."
	MsgAdjustmentReasonTooLong  = "Le motif ne doit pas dépasser 200 caractères."
	MsgReceivedBelowDue         = "Le montant reçu ne peut pas être inférieur au montant à payer. Ajustez d'abord le montant à payer."
)

// parseOptionalAmount analyse un montant saisi en euros (research.md 014 ›
// R4) ; vide : nil.
func parseOptionalAmount(text, location string) (*int64, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	cents, err := season.ParseAmount(text)
	if err != nil {
		msg := MsgAmountFormat
		if errors.Is(err, season.ErrAmountTooHigh) {
			msg = MsgAmountTooHigh
		}
		return nil, FieldErrors(location, msg)
	}
	return &cents, nil
}

// adjust calcule le montant à payer à partir du montant théorique et des
// champs saisis amount et adjustmentReason sous prefix (« body »…).
func adjust(base int64, amount, reason, prefix string) (membership.Adjustment, error) {
	requested, err := parseOptionalAmount(amount, prefix+".amount")
	if err != nil {
		return membership.Adjustment{}, err
	}
	a, err := membership.Adjust(base, requested, reason)
	switch {
	case errors.Is(err, membership.ErrAdjustmentReasonRequired):
		return a, FieldErrors(prefix+".adjustmentReason", MsgAdjustmentReasonRequired)
	case errors.Is(err, membership.ErrAdjustmentReasonTooLong):
		return a, FieldErrors(prefix+".adjustmentReason", MsgAdjustmentReasonTooLong)
	}
	return a, err
}

// receivedAmount renvoie le montant reçu d'un paiement saisi sous location :
// le montant à payer si le champ est vide, refus s'il lui est inférieur.
func receivedAmount(body RecordPaymentBody, location string, due int64) (int64, error) {
	got, err := parseOptionalAmount(body.ReceivedAmount, location+".receivedAmount")
	if err != nil {
		return 0, err
	}
	if got == nil {
		return due, nil
	}
	if membership.CheckReceived(due, *got) != nil {
		return 0, FieldErrors(location+".receivedAmount", MsgReceivedBelowDue)
	}
	return *got, nil
}

// adjustmentColumns convertit un ajustement en paramètres de requête.
func adjustmentColumns(a membership.Adjustment) (base sql.NullInt64, reason sql.NullString) {
	if a.BaseCents != nil {
		base = nid(*a.BaseCents)
		reason = ns(a.Reason)
	}
	return base, reason
}

// basePtr lit une colonne base_amount_cents.
func basePtr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	b := v.Int64
	return &b
}
