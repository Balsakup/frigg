package communication

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Situation est la situation de paiement d'un adhérent, déduite de l'état du
// paiement de son adhésion (research.md › R1).
type Situation string

const (
	UpToDate  Situation = "up_to_date" // payée ou gratuite
	Announced Situation = "announced"  // moyen choisi, réception non constatée
	Due       Situation = "due"        // aucun moyen choisi ou carte non aboutie
)

// Situations est l'ordre d'affichage des situations.
var Situations = []Situation{UpToDate, Announced, Due}

// Preset est une cible prédéfinie ; custom désigne toute autre combinaison.
type Preset string

const (
	PresetAll      Preset = "all"
	PresetUpToDate Preset = "up_to_date"
	PresetUnpaid   Preset = "unpaid"
	PresetCustom   Preset = "custom"
)

// ErrNoSituation et ErrUnknownSituation signalent une liste de situations
// invalide.
var (
	ErrNoSituation      = errors.New("aucune situation de paiement")
	ErrUnknownSituation = errors.New("situation de paiement inconnue")
)

// SituationsOf renvoie les situations d'une cible prédéfinie (nil pour custom).
func SituationsOf(p Preset) []Situation {
	switch p {
	case PresetAll:
		return []Situation{UpToDate, Announced, Due}
	case PresetUpToDate:
		return []Situation{UpToDate}
	case PresetUnpaid:
		return []Situation{Announced, Due}
	}
	return nil
}

// NormalizeSituations dédoublonne et trie les situations (ordre alphabétique,
// celui du stockage) ; elle refuse une liste vide ou une valeur inconnue.
func NormalizeSituations(in []Situation) ([]Situation, error) {
	if len(in) == 0 {
		return nil, ErrNoSituation
	}
	out := make([]Situation, 0, len(in))
	for _, s := range in {
		if !slices.Contains(Situations, s) {
			return nil, ErrUnknownSituation
		}
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	slices.Sort(out)
	return out, nil
}

// PresetOf renvoie la cible prédéfinie correspondant à des situations.
func PresetOf(situations []Situation) Preset {
	norm, err := NormalizeSituations(situations)
	if err != nil {
		return PresetCustom
	}
	for _, p := range []Preset{PresetAll, PresetUpToDate, PresetUnpaid} {
		want, _ := NormalizeSituations(SituationsOf(p))
		if slices.Equal(norm, want) {
			return p
		}
	}
	return PresetCustom
}

// EncodeSituations sérialise des situations normalisées pour le stockage.
func EncodeSituations(s []Situation) string {
	parts := make([]string, len(s))
	for i, v := range s {
		parts[i] = string(v)
	}
	return strings.Join(parts, ",")
}

// DecodeSituations relit des situations stockées.
func DecodeSituations(v string) []Situation {
	if v == "" {
		return []Situation{}
	}
	parts := strings.Split(v, ",")
	out := make([]Situation, len(parts))
	for i, p := range parts {
		out[i] = Situation(p)
	}
	return out
}

// NormalizeFeeIDs dédoublonne et trie des identifiants de tarifs.
func NormalizeFeeIDs(ids []int64) []int64 {
	out := slices.Clone(ids)
	slices.Sort(out)
	return slices.Compact(out)
}

// EncodeFeeIDs sérialise des identifiants de tarifs ; vide = tous.
func EncodeFeeIDs(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(parts, ",")
}

// DecodeFeeIDs relit des identifiants de tarifs stockés.
func DecodeFeeIDs(v string) []int64 {
	out := []int64{}
	if v == "" {
		return out
	}
	for _, p := range strings.Split(v, ",") {
		if id, err := strconv.ParseInt(p, 10, 64); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// SituationLabel est le libellé français d'une situation.
func SituationLabel(s Situation) string {
	switch s {
	case UpToDate:
		return "À jour"
	case Announced:
		return "Paiement annoncé"
	case Due:
		return "À payer"
	}
	return string(s)
}

// PresetLabel est le libellé français d'une cible.
func PresetLabel(p Preset) string {
	switch p {
	case PresetAll:
		return "Tous les adhérents"
	case PresetUpToDate:
		return "À jour"
	case PresetUnpaid:
		return "En impayé"
	}
	return "Personnalisée"
}

// TargetLabel décrit une cible, figée à l'envoi (FR-018) : « 2026-2027 · En
// impayé » ou « 2026-2027 · Personnalisée : À payer · Tarifs : Jeune, Adulte ».
func TargetLabel(seasonLabel string, situations []Situation, feeLabels []string) string {
	p := PresetOf(situations)
	label := seasonLabel + " · " + PresetLabel(p)
	if p == PresetCustom {
		names := make([]string, 0, len(situations))
		for _, s := range Situations {
			if slices.Contains(situations, s) {
				names = append(names, SituationLabel(s))
			}
		}
		label += " : " + strings.Join(names, ", ")
	}
	if len(feeLabels) > 0 {
		label += " · Tarifs : " + strings.Join(feeLabels, ", ")
	}
	return truncate(label, TargetLabelMax)
}

// truncate coupe s à max caractères, points de suspension compris.
func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max-1]) + "…"
}
