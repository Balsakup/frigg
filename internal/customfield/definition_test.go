package customfield

import (
	"strings"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func hasField(errs FieldErrors, field, msg string) bool {
	for _, e := range errs {
		if e.Field == field && (msg == "" || e.Message == msg) {
			return true
		}
	}
	return false
}

func TestValidateDefinition(t *testing.T) {
	existing := []Definition{
		{ID: 1, Scope: ScopeMember, Label: "Numéro de licence"},
		{ID: 2, Scope: ScopeMember, Label: "Ancien champ", Archived: true},
		{ID: 3, Scope: ScopeMembership, Label: "Taille"},
	}
	base := func(label string) DefinitionInput {
		return DefinitionInput{Scope: ScopeMember, Label: label, Type: TypeShortText, Visibility: VisibilityAdmin}
	}
	cases := []struct {
		name      string
		in        DefinitionInput
		excludeID int64
		field     string
		msg       string
	}{
		{"libellé vide", base("   "), 0, "label", MsgLabelRequired},
		{"libellé de 81 runes", base(strings.Repeat("é", 81)), 0, "label", MsgLabelTooLong},
		{"doublon sans casse ni accents", base("numero de LICENCE"), 0, "label", MsgLabelTaken},
		{"doublon d'un champ archivé", base("ancien  champ"), 0, "label", MsgLabelTaken},
		{"aide trop longue", DefinitionInput{Scope: ScopeMember, Label: "X", Type: TypeDate, Help: new(strings.Repeat("a", 201))}, 0, "help", MsgHelpTooLong},
		{"oui/non obligatoire", DefinitionInput{Scope: ScopeMember, Label: "X", Type: TypeBool, Required: true}, 0, "required", MsgBoolRequired},
		{"un seul choix", DefinitionInput{Scope: ScopeMember, Label: "X", Type: TypeChoice, Choices: []ChoiceInput{{Label: "A"}}}, 0, "choices", MsgChoicesCount},
		{"choix en double", DefinitionInput{Scope: ScopeMember, Label: "X", Type: TypeChoice, Choices: []ChoiceInput{{Label: "Élevé"}, {Label: "eleve"}}}, 0, "choices[1]", MsgChoiceTaken},
		{"choix vide", DefinitionInput{Scope: ScopeMember, Label: "X", Type: TypeChoice, Choices: []ChoiceInput{{Label: "A"}, {Label: " "}}}, 0, "choices[1]", MsgChoiceRequired},
		{"choix sur un texte", DefinitionInput{Scope: ScopeMember, Label: "X", Type: TypeShortText, Choices: []ChoiceInput{{Label: "A"}, {Label: "B"}}}, 0, "choices", MsgChoicesUnused},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, errs := ValidateDefinition(c.in, existing, c.excludeID)
			if !hasField(errs, c.field, c.msg) {
				t.Fatalf("erreur %s « %s » attendue, obtenu %v", c.field, c.msg, errs)
			}
		})
	}

	t.Run("valides", func(t *testing.T) {
		for _, in := range []DefinitionInput{
			base(strings.Repeat("é", 80)),
			base("Taille"), // autre portée
			{Scope: ScopeMember, Label: "Choix", Type: TypeChoice, Choices: make([]ChoiceInput, 0)},
		} {
			if in.Type == TypeChoice {
				for i := range 50 {
					in.Choices = append(in.Choices, ChoiceInput{Label: "C" + strings.Repeat("x", i)})
				}
			}
			if out, errs := ValidateDefinition(in, existing, 0); errs != nil {
				t.Errorf("%q refusé : %v", out.Label, errs)
			}
		}
	})
	t.Run("modification du champ lui-même", func(t *testing.T) {
		if _, errs := ValidateDefinition(base("Numéro de licence"), existing, 1); errs != nil {
			t.Fatal(errs)
		}
	})
	t.Run("51 choix", func(t *testing.T) {
		in := DefinitionInput{Scope: ScopeMember, Label: "X", Type: TypeChoice}
		for i := range 51 {
			in.Choices = append(in.Choices, ChoiceInput{Label: "C" + strings.Repeat("x", i)})
		}
		if _, errs := ValidateDefinition(in, existing, 0); !hasField(errs, "choices", MsgChoicesCount) {
			t.Fatal(errs)
		}
	})
	t.Run("nettoyage", func(t *testing.T) {
		out, errs := ValidateDefinition(DefinitionInput{Scope: ScopeMember, Label: "  Taille   de t-shirt ", Type: TypeChoice,
			Help: new("   "), Choices: []ChoiceInput{{ID: 4, Label: " S "}, {Label: "M"}}}, existing, 0)
		if errs != nil {
			t.Fatal(errs)
		}
		if out.Label != "Taille de t-shirt" || out.Help != nil || out.Choices[0] != (ChoiceInput{ID: 4, Label: "S"}) {
			t.Fatalf("%+v", out)
		}
	})
}
