package membership

// Hint est l'aide « Adhérent la saison précédente dans Frigg » (research.md
// 006 › R6). Elle ne décide jamais du type d'adhésion.
type Hint string

const (
	HintYes     Hint = "yes"
	HintNo      Hint = "no"
	HintUnknown Hint = "unknown"
)

// PreviousSeasonHint : « oui » si le membre a une adhésion validée dans la
// saison précédente ; « non » si cette saison existe et compte au moins une
// adhésion validée ; « inconnu » sinon (pas de saison précédente, ou saison
// antérieure à l'usage de Frigg pour les adhésions).
func PreviousSeasonHint(hasPrevious bool, previousValidated int64, memberValidated bool) Hint {
	switch {
	case !hasPrevious || previousValidated == 0:
		return HintUnknown
	case memberValidated:
		return HintYes
	default:
		return HintNo
	}
}
