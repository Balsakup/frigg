package event

import (
	"fmt"

	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
)

// PaymentSnapshot présente une inscription au domaine de paiement des
// adhésions (research.md 012 › R3) : une inscription confirmée d'un événement
// non annulé se comporte comme une adhésion validée, sinon comme une adhésion
// annulée. CanPay, CanPayByCard, CanCancelPayment, SettleOnline et
// RefundCancels s'appliquent alors tels quels.
func PaymentSnapshot(r RegSnapshot, eventCancelled bool) membership.Snapshot {
	status := membership.Cancelled
	if r.Confirmed && !eventCancelled {
		status = membership.Validated
	}
	return membership.Snapshot{
		Status:        status,
		PaymentStatus: r.PaymentStatus,
		AmountCents:   r.AmountCents,
		PaidMethod:    r.PaidMethod,
	}
}

// ReferencePrefix renvoie le préfixe des références de paiement des
// inscriptions d'une saison : « EVT- », années de début et de fin sur deux
// chiffres, « - » (R5).
func ReferencePrefix(starts, ends season.Date) string {
	return fmt.Sprintf("EVT-%02d%02d-", starts.Year()%100, ends.Year()%100)
}
