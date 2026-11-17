// Package public porte les règles des parcours publics, sans base ni HTTP
// (specs/016-public-journeys) : durées de vie, validation d'un dépôt et
// décision de confirmation.
package public

import (
	"time"

	"github.com/balsakup/frigg/internal/member"
)

const (
	// ConfirmTTL est la durée de validité du lien de confirmation d'un dépôt
	// (FR-009).
	ConfirmTTL = 48 * time.Hour
	// SubmissionRetention est la durée de conservation d'un dépôt non
	// confirmé, comptée depuis son envoi (FR-008).
	SubmissionRetention = 7 * 24 * time.Hour
	// LinkTTL est la durée de validité d'un lien personnel de suivi (FR-013).
	LinkTTL = 30 * 24 * time.Hour
	// LinkPurgeGrace est le délai après expiration au-delà duquel un lien
	// personnel est purgé (FR-013, research.md › R2).
	LinkPurgeGrace = 24 * time.Hour
	// InactiveAfterMonths est l'ancienneté, en mois sans activité, au-delà de
	// laquelle une fiche d'origine publique est inactive (FR-041).
	InactiveAfterMonths = 12
	// MaxManagedMembers est le nombre maximal de fiches gérées par un compte
	// (005 FR-004).
	MaxManagedMembers = member.MaxPerAccount
)
