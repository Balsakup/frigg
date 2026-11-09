package event

import (
	"errors"
	"fmt"
	"time"

	"github.com/balsakup/frigg/internal/membership"
)

// Ineligibility est la raison pour laquelle une fiche ne peut pas être
// inscrite.
type Ineligibility string

const (
	Eligible          Ineligibility = ""
	NotSeasonMember   Ineligibility = "not_season_member"
	AlreadyRegistered Ineligibility = "already_registered"
)

// Eligibility dit si une fiche peut être inscrite (FR-011, R7).
func Eligibility(o Openness, isSeasonMember, alreadyRegistered bool) Ineligibility {
	switch {
	case alreadyRegistered:
		return AlreadyRegistered
	case o == SeasonMembers && !isSeasonMember:
		return NotSeasonMember
	default:
		return Eligible
	}
}

// Window renvoie la période d'inscription de l'espace membre : de l'ouverture
// (ou de la publication) à la clôture (ou au début de l'événement).
func Window(s State) (opens, closes time.Time) {
	switch {
	case s.RegistrationOpensAt != nil:
		opens = *s.RegistrationOpensAt
	case s.PublishedAt != nil:
		opens = *s.PublishedAt
	}
	closes = s.StartsAt
	if s.RegistrationClosesAt != nil {
		closes = *s.RegistrationClosesAt
	}
	return opens, closes
}

// WindowOpen : la période d'inscription est ouverte à l'instant now.
func WindowOpen(s State, now time.Time) bool {
	opens, closes := Window(s)
	return !now.Before(opens) && now.Before(closes)
}

// CanRegister : inscription depuis l'espace membre (FR-012).
func CanRegister(s State, now time.Time) bool {
	return s.Status == Published && !Finished(s, now) && WindowOpen(s, now)
}

// CanAdminRegister : inscription par un administrateur, hors période, avant
// la fin (FR-016).
func CanAdminRegister(s State, now time.Time) bool {
	return s.Status == Published && !Finished(s, now)
}

// CapacityLeft renvoie le nombre de places restantes ; nil : illimité.
func CapacityLeft(capacity *int, confirmed int) *int {
	if capacity == nil {
		return nil
	}
	left := max(*capacity-confirmed, 0)
	return &left
}

// ErrFull : places insuffisantes pour la demande (R6, tout ou rien).
type ErrFull struct{ Left int }

func (e *ErrFull) Error() string { return fmt.Sprintf("complet : %d place(s) restante(s)", e.Left) }

// CheckPlaces vérifie qu'il reste assez de places pour requested inscriptions.
func CheckPlaces(capacity *int, confirmed, requested int) error {
	left := CapacityLeft(capacity, confirmed)
	if left != nil && *left < requested {
		return &ErrFull{Left: *left}
	}
	return nil
}

// ErrCapacityBelow : capacité inférieure au nombre d'inscrits (FR-007).
var ErrCapacityBelow = errors.New("capacité inférieure au nombre d'inscrits")

// CheckCapacityChange refuse une capacité inférieure au nombre d'inscrits.
func CheckCapacityChange(capacity *int, confirmed int) error {
	if capacity != nil && *capacity < confirmed {
		return ErrCapacityBelow
	}
	return nil
}

// RegSnapshot est l'état d'une inscription utile aux décisions.
type RegSnapshot struct {
	Confirmed     bool
	AmountCents   int64
	PaymentStatus membership.PaymentStatus
	PaidMethod    membership.Method
	// CardPending : une tentative de paiement par carte est en cours.
	CardPending bool
}

// CanUnregister : annulation par le compte (FR-013) : confirmée, pendant la
// période, ni payée ni en cours de paiement par carte.
func CanUnregister(s State, r RegSnapshot, now time.Time) bool {
	return r.Confirmed && s.Status == Published && WindowOpen(s, now) &&
		r.PaymentStatus != membership.Paid && !r.CardPending
}

// CanRemove : retrait par un administrateur, avant la fin (FR-016).
func CanRemove(s State, r RegSnapshot, now time.Time) bool {
	return r.Confirmed && !Finished(s, now)
}

// Actions d'administration sur un événement.

func CanUpdate(s State) bool                     { return s.Status != Cancelled }
func CanUpdateDates(s State, now time.Time) bool { return s.Status != Cancelled && !Finished(s, now) }
func CanPublish(s State, now time.Time) bool     { return s.Status == Draft && !Finished(s, now) }
func CanCancel(s State, now time.Time) bool      { return s.Status == Published && !Finished(s, now) }
func CanRestore(s State, now time.Time) bool     { return s.Status == Cancelled && !Finished(s, now) }
func CanDelete(registrations int) bool           { return registrations == 0 }
