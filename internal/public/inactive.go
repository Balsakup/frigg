package public

import (
	"time"

	"github.com/balsakup/frigg/internal/season"
)

// Fiches publiques inactives (FR-041, research.md › R14). Inactive est le
// miroir testable de la requête ListInactivePublicMembers : les mêmes bornes,
// sans base. Une date ou un instant égal à la coupure compte comme récent.

// InactiveCutoff renvoie le jour de coupure : aujourd'hui (heure de Paris)
// moins InactiveAfterMonths mois.
func InactiveCutoff(today season.Date) season.Date {
	return today.AddMonths(-InactiveAfterMonths)
}

// CutoffInstant renvoie le début du jour de coupure à Paris.
func CutoffInstant(cutoff season.Date) time.Time {
	t, _ := time.ParseInLocation("2006-01-02", cutoff.String(), season.Paris)
	return t
}

// Inactive dit si une fiche d'origine publique sans compte est inactive à la
// coupure :
//   - createdAt : création de la fiche ;
//   - pending : une demande d'adhésion est en attente ;
//   - lastValidatedSeasonEnd : fin de la dernière saison d'une adhésion
//     validée (valeur zéro : aucune) ;
//   - lastEventEnd : fin du dernier événement d'une inscription confirmée
//     (valeur zéro : aucune).
func Inactive(createdAt time.Time, pending bool, lastValidatedSeasonEnd season.Date, lastEventEnd time.Time, cutoff season.Date) bool {
	at := CutoffInstant(cutoff)
	return createdAt.Before(at) &&
		!pending &&
		lastValidatedSeasonEnd.Before(cutoff) &&
		lastEventEnd.Before(at)
}
