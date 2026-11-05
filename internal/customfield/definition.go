// Package customfield porte les règles des champs personnalisés des fiches
// membres et des adhésions, sans base ni HTTP (research.md 011 › R2) :
// validation des définitions et des valeurs, visibilité selon l'audience,
// champs obligatoires et filtres de liste.
package customfield

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/balsakup/frigg/internal/textfold"
)

// Scope est la portée d'un champ : les fiches membres ou les adhésions.
type Scope string

const (
	ScopeMember     Scope = "member"
	ScopeMembership Scope = "membership"
)

// Type est la forme de la valeur attendue.
type Type string

const (
	TypeShortText Type = "short_text"
	TypeLongText  Type = "long_text"
	TypeNumber    Type = "number"
	TypeDate      Type = "date"
	TypeBool      Type = "bool"
	TypeChoice    Type = "choice"
)

// Visibility règle ce que voit et modifie le compte qui gère la fiche.
type Visibility string

const (
	// VisibilityAdmin : réservé à l'administration.
	VisibilityAdmin Visibility = "admin"
	// VisibilityVisible : affiché en lecture seule au compte gestionnaire.
	VisibilityVisible Visibility = "visible"
	// VisibilityEditable : rempli par le compte gestionnaire.
	VisibilityEditable Visibility = "editable"
)

const (
	// MaxActive est le nombre maximal de champs actifs par portée (FR-010).
	MaxActive  = 30
	MaxLabel   = 80
	MaxHelp    = 200
	MinChoices = 2
	MaxChoices = 50
	MaxShort   = 200
	MaxLong    = 2000
)

// Messages d'erreur renvoyés à l'interface.
const (
	MsgLabelRequired  = "Indiquez le libellé du champ."
	MsgLabelTooLong   = "Le libellé ne doit pas dépasser 80 caractères."
	MsgLabelTaken     = "Un champ porte déjà ce libellé."
	MsgHelpTooLong    = "Le texte d'aide ne doit pas dépasser 200 caractères."
	MsgChoicesCount   = "Indiquez de 2 à 50 choix."
	MsgChoiceRequired = "Indiquez le libellé du choix."
	MsgChoiceTooLong  = "Le choix ne doit pas dépasser 80 caractères."
	MsgChoiceTaken    = "Ce choix figure déjà dans la liste."
	MsgChoicesUnused  = "Seule une liste de choix porte des choix."
	MsgBoolRequired   = "Un champ Oui/non ne peut pas être obligatoire."
	MsgUnknownField   = "Ce champ n'existe pas ou n'est pas modifiable."
	MsgRequired       = "Ce champ est obligatoire."
	MsgIncompatible   = "Valeur incompatible avec ce champ."
	MsgShortTooLong   = "200 caractères au plus."
	MsgShortOneLine   = "Saisissez le texte sur une seule ligne."
	MsgLongTooLong    = "2 000 caractères au plus."
	MsgNumber         = "Saisissez un nombre."
	MsgDate           = "Saisissez une date valide."
	MsgUnknownChoice  = "Choix inconnu."
	MsgUnknownFilter  = "Filtre inconnu ou périmé."
)

// FieldError est une erreur rattachée à un champ du corps (chemin JSON
// camelCase, par exemple « label », « choices[1] » ou « customFields.f7 »).
type FieldError struct {
	Field   string
	Message string
}

// FieldErrors regroupe toutes les erreurs d'une saisie.
type FieldErrors []FieldError

func (e FieldErrors) Error() string {
	msgs := make([]string, len(e))
	for i, fe := range e {
		msgs[i] = fe.Field + " : " + fe.Message
	}
	return strings.Join(msgs, " ; ")
}

// Choice est une valeur proposée par une liste de choix.
type Choice struct {
	ID    int64
	Label string
}

// Definition décrit un champ personnalisé enregistré.
type Definition struct {
	ID         int64
	Scope      Scope
	Label      string
	Type       Type
	Visibility Visibility
	Required   bool
	Help       *string
	Choices    []Choice
	Archived   bool
}

// ChoiceInput est un choix saisi ; ID vaut 0 pour un nouveau choix.
type ChoiceInput struct {
	ID    int64
	Label string
}

// DefinitionInput est une définition saisie, avant nettoyage.
type DefinitionInput struct {
	Scope      Scope
	Label      string
	Type       Type
	Visibility Visibility
	Required   bool
	Help       *string
	Choices    []ChoiceInput
}

// squash retire les espaces de bord et réduit les espaces intérieurs.
func squash(s string) string { return strings.Join(strings.Fields(s), " ") }

// ValidateDefinition nettoie une définition et vérifie ses règles (FR-002,
// FR-003). existing contient tous les champs de la portée, archivés compris,
// pour l'unicité du libellé ; excludeID écarte le champ modifié.
func ValidateDefinition(in DefinitionInput, existing []Definition, excludeID int64) (DefinitionInput, FieldErrors) {
	var errs FieldErrors
	out := in
	out.Label = squash(in.Label)
	switch n := utf8.RuneCountInString(out.Label); {
	case n == 0:
		errs = append(errs, FieldError{"label", MsgLabelRequired})
	case n > MaxLabel:
		errs = append(errs, FieldError{"label", MsgLabelTooLong})
	default:
		key := textfold.Fold(out.Label)
		for _, d := range existing {
			if d.ID != excludeID && d.Scope == in.Scope && textfold.Fold(d.Label) == key {
				errs = append(errs, FieldError{"label", MsgLabelTaken})
				break
			}
		}
	}

	out.Help = nil
	if in.Help != nil {
		if h := strings.TrimSpace(*in.Help); h != "" {
			if utf8.RuneCountInString(h) > MaxHelp {
				errs = append(errs, FieldError{"help", MsgHelpTooLong})
			}
			out.Help = &h
		}
	}

	if in.Type == TypeBool && in.Required {
		errs = append(errs, FieldError{"required", MsgBoolRequired})
	}

	out.Choices = nil
	if in.Type != TypeChoice {
		if len(in.Choices) > 0 {
			errs = append(errs, FieldError{"choices", MsgChoicesUnused})
		}
		return out, errs
	}
	if len(in.Choices) < MinChoices || len(in.Choices) > MaxChoices {
		errs = append(errs, FieldError{"choices", MsgChoicesCount})
	}
	seen := map[string]bool{}
	for i, c := range in.Choices {
		label := squash(c.Label)
		field := fmt.Sprintf("choices[%d]", i)
		switch n := utf8.RuneCountInString(label); {
		case n == 0:
			errs = append(errs, FieldError{field, MsgChoiceRequired})
		case n > MaxLabel:
			errs = append(errs, FieldError{field, MsgChoiceTooLong})
		default:
			key := textfold.Fold(label)
			if seen[key] {
				errs = append(errs, FieldError{field, MsgChoiceTaken})
			}
			seen[key] = true
		}
		out.Choices = append(out.Choices, ChoiceInput{ID: c.ID, Label: label})
	}
	return out, errs
}

// ValidScope, ValidType et ValidVisibility vérifient les énumérations.
func ValidScope(s Scope) bool { return s == ScopeMember || s == ScopeMembership }

func ValidType(t Type) bool {
	switch t {
	case TypeShortText, TypeLongText, TypeNumber, TypeDate, TypeBool, TypeChoice:
		return true
	}
	return false
}

func ValidVisibility(v Visibility) bool {
	return v == VisibilityAdmin || v == VisibilityVisible || v == VisibilityEditable
}
