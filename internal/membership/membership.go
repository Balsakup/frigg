// Package membership porte les règles des adhésions et de leur paiement
// (research.md 006 › R2) : montant appliqué, transitions d'état, actions
// autorisées, aide « saison précédente », références de paiement, paramètres
// des moyens de paiement et chiffres d'une saison. Il ne dépend pas de la base.
package membership

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/balsakup/frigg/internal/season"
)

// Status est l'état d'une adhésion.
type Status string

const (
	Pending   Status = "pending"
	Validated Status = "validated"
	Refused   Status = "refused"
	Cancelled Status = "cancelled"
)

// Kind est le type d'une adhésion.
type Kind string

const (
	First   Kind = "first"
	Renewal Kind = "renewal"
)

// PaymentStatus est l'état du paiement d'une adhésion validée ; vide : pas de
// paiement (adhésion non validée ou gratuite).
type PaymentStatus string

const (
	NoPayment PaymentStatus = ""
	Due       PaymentStatus = "due"
	Announced PaymentStatus = "announced"
	Paid      PaymentStatus = "paid"
)

// Method est un moyen de paiement.
type Method string

const (
	Cash     Method = "cash"
	Cheque   Method = "cheque"
	Transfer Method = "transfer"
	// Card : carte bancaire via HelloAsso (research.md 008). Ni annoncée ni
	// constatée à la main : seul HelloAsso constate un paiement par carte.
	Card Method = "card"
)

// Methods liste les moyens dans l'ordre d'affichage, la carte en premier.
var Methods = []Method{Card, Cash, Cheque, Transfer}

// MinCardAmount est le montant minimal payable par carte (centimes).
const MinCardAmount = 100

// ManualMethod indique un moyen que le compte annonce et qu'un administrateur
// constate lui-même (tous sauf la carte).
func ManualMethod(m Method) bool {
	switch m {
	case Cash, Cheque, Transfer:
		return true
	}
	return false
}

const (
	// MaxReason est la longueur maximale d'un motif de refus ou d'annulation.
	MaxReason = 500
	// MaxNote est la longueur maximale du commentaire d'un paiement.
	MaxNote = 200
)

var (
	ErrReasonRequired = errors.New("motif manquant")
	ErrReasonTooLong  = errors.New("motif trop long")
	ErrNoteTooLong    = errors.New("commentaire trop long")
	// ErrTransition : l'état actuel de l'adhésion ne permet pas l'action.
	ErrTransition = errors.New("transition impossible")
)

// AppliedAmount renvoie le montant d'un tarif pour un type : le montant de
// renouvellement si le tarif en a un et que le type est « renouvellement »,
// le montant principal sinon.
func AppliedAmount(fee season.Fee, k Kind) int64 {
	if k == Renewal && fee.RenewalAmountCents != nil {
		return *fee.RenewalAmountCents
	}
	return fee.AmountCents
}

// Snapshot est l'état d'une adhésion utile aux décisions.
type Snapshot struct {
	Status        Status
	PaymentStatus PaymentStatus
	// SeasonClosed : la date de fin de la saison est passée.
	SeasonClosed bool
	// AmountCents : montant appliqué (adhésion validée).
	AmountCents int64
	// PaidMethod : moyen constaté d'une adhésion payée.
	PaidMethod Method
}

// CanCancelRequest : le compte annule une demande en attente (FR-009).
func CanCancelRequest(s Snapshot) bool { return s.Status == Pending }

// CanDecide : validation ou refus d'une demande en attente, y compris sur une
// saison close (FR-017).
func CanDecide(s Snapshot) bool { return s.Status == Pending }

// CanUpdate : correction d'une adhésion validée non payée d'une saison non
// close (FR-016).
func CanUpdate(s Snapshot) bool {
	return s.Status == Validated && s.PaymentStatus != Paid && !s.SeasonClosed
}

// CanCancel : annulation par un administrateur d'une adhésion validée d'une
// saison non close (FR-015).
func CanCancel(s Snapshot) bool { return s.Status == Validated && !s.SeasonClosed }

// CanPay : choix ou changement du moyen de paiement par le compte (FR-022b).
func CanPay(s Snapshot) bool {
	return s.Status == Validated && (s.PaymentStatus == Due || s.PaymentStatus == Announced)
}

// CanRecordPayment : constatation du paiement, y compris sur une saison close
// (FR-017, FR-022d).
func CanRecordPayment(s Snapshot) bool { return CanPay(s) }

// CanCancelPayment : annulation de la constatation (US4 §10).
func CanCancelPayment(s Snapshot) bool {
	return s.Status == Validated && s.PaymentStatus == Paid && s.PaidMethod != Card
}

// CanPayByCard : paiement en ligne par le compte (research.md 008 › R7) :
// adhésion payable, montant suffisant et carte disponible.
func CanPayByCard(s Snapshot, cardAvailable bool) bool {
	return cardAvailable && CanPay(s) && s.AmountCents >= MinCardAmount
}

// MemberActions sont les actions proposées au compte qui gère la fiche.
type MemberActions struct {
	Cancel    bool
	Pay       bool
	PayByCard bool
}

// AdminActions sont les actions proposées à un administrateur.
type AdminActions struct {
	Validate, Refuse, Update, Cancel, RecordPayment, CancelPayment bool
}

// MemberActionsFor calcule les actions de l'espace membre.
func MemberActionsFor(s Snapshot, cardAvailable bool) MemberActions {
	return MemberActions{Cancel: CanCancelRequest(s), Pay: CanPay(s), PayByCard: CanPayByCard(s, cardAvailable)}
}

// AdminActionsFor calcule les actions de l'administration.
func AdminActionsFor(s Snapshot) AdminActions {
	return AdminActions{
		Validate:      CanDecide(s),
		Refuse:        CanDecide(s),
		Update:        CanUpdate(s),
		Cancel:        CanCancel(s),
		RecordPayment: CanRecordPayment(s),
		CancelPayment: CanCancelPayment(s),
	}
}

// PaymentAfter calcule l'état du paiement après une validation ou une
// correction donnant le montant amount : aucun paiement pour 0 €, sinon « à
// payer », ou l'état et le moyen annoncé actuels s'il y en avait un. Une
// référence est nécessaire dès que le paiement existe.
func PaymentAfter(amount int64, current PaymentStatus, announced Method) (PaymentStatus, Method) {
	switch {
	case amount == 0:
		return NoPayment, ""
	case current == NoPayment:
		return Due, ""
	default:
		return current, announced
	}
}

// CleanReason nettoie un motif : espaces de début et de fin retirés, espaces
// intérieurs réduits, retours à la ligne conservés.
func CleanReason(s string) (string, error) {
	c := cleanText(s)
	switch {
	case c == "":
		return "", ErrReasonRequired
	case utf8.RuneCountInString(c) > MaxReason:
		return "", ErrReasonTooLong
	}
	return c, nil
}

// CleanNote nettoie le commentaire facultatif d'un paiement ; vide : absent.
func CleanNote(s string) (string, error) {
	c := cleanText(s)
	if utf8.RuneCountInString(c) > MaxNote {
		return "", ErrNoteTooLong
	}
	return c, nil
}

// cleanText réduit les espaces de chaque ligne et retire les lignes vides de
// début et de fin.
func cleanText(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i, l := range lines {
		lines[i] = strings.Join(strings.Fields(l), " ")
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}
