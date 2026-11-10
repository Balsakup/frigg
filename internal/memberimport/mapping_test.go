package memberimport

import (
	"slices"
	"strings"
	"testing"
)

func TestSuggestStandard(t *testing.T) {
	headers := []string{"Prénom", "NOM", "Date de naissance *", "Tél.", "E-mail", "CP", "Ville", "Complément d'adresse", "Remarques", "Raison du rejet", "Nom", "Courriel"}
	want := []Target{FirstName, LastName, BirthDate, Phone, Email, PostalCode, City, AddressLine2, None, None, None, None}
	if got := Suggest(headers, nil); !slices.Equal(got, want) {
		t.Errorf("Suggest = %q\nattendu  %q", got, want)
	}
	if got := Suggest([]string{"Date de naissance (JJ/MM/AAAA)", "Adresse"}, nil); !slices.Equal(got, []Target{BirthDate, AddressLine1}) {
		t.Errorf("parenthèse finale : %q", got)
	}
}

func TestSuggestCustomFields(t *testing.T) {
	defs := defsFixture()
	headers := []string{"Numéro de licence", "Taille (cm)", "Catégorie (Poussin, Benjamin)", "Nom"}
	want := []Target{FieldTarget(3), FieldTarget(9), FieldTarget(5), LastName}
	if got := Suggest(headers, defs); !slices.Equal(got, want) {
		t.Errorf("Suggest = %q, attendu %q", got, want)
	}
	// Un synonyme de la fiche l'emporte sur un champ personnalisé de même libellé.
	withNom := append(defs, defsFixture()[0])
	withNom[len(withNom)-1].ID, withNom[len(withNom)-1].Label = 11, "Ville"
	if got := Suggest([]string{"Ville"}, withNom); got[0] != City {
		t.Errorf("synonyme d'abord : %q", got)
	}
}

func TestColumnLabels(t *testing.T) {
	got := ColumnLabels([]string{"Nom", "", "nom", "Prénom"})
	want := []string{"Nom", "Colonne 2", "Colonne 3", "Prénom"}
	if !slices.Equal(got, want) {
		t.Errorf("ColumnLabels = %q", got)
	}
}

func TestValidateMapping(t *testing.T) {
	defs := defsFixture()
	labels := []string{"Prénom", "Nom", "Naissance", "Mail"}
	ok := []Target{FirstName, LastName, BirthDate, FieldTarget(3)}
	if errs := ValidateMapping(ok, labels, defs); errs != nil {
		t.Errorf("correspondance valide refusée : %+v", errs)
	}
	cases := []struct {
		name string
		m    []Target
		want []MappingError
	}{
		{"longueur", []Target{FirstName}, []MappingError{{-1, MsgMappingLength}}},
		{"cible inconnue", []Target{FirstName, LastName, BirthDate, "foo"}, []MappingError{{3, MsgMappingUnknown}}},
		{"champ archivé", []Target{FirstName, LastName, BirthDate, FieldTarget(99)}, []MappingError{{3, MsgMappingUnknown}}},
		{"doublon", []Target{FirstName, LastName, BirthDate, LastName}, []MappingError{{3, "Ce champ est déjà associé à la colonne « Nom »."}}},
		{"exigés absents", []Target{None, None, None, Email}, []MappingError{
			{-1, "Associez une colonne au champ « Prénom »."},
			{-1, "Associez une colonne au champ « Nom »."},
			{-1, "Associez une colonne au champ « Date de naissance »."},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ValidateMapping(c.m, labels, defs); !slices.Equal(got, c.want) {
				t.Errorf("ValidateMapping = %+v\nattendu %+v", got, c.want)
			}
		})
	}
}

func TestFieldTarget(t *testing.T) {
	if id, ok := FieldTarget(42).FieldID(); !ok || id != 42 {
		t.Errorf("FieldID = %d, %v", id, ok)
	}
	for _, bad := range []Target{"field:", "field:x", "field:-1", FirstName} {
		if _, ok := bad.FieldID(); ok {
			t.Errorf("%q accepté", bad)
		}
	}
	if !strings.HasPrefix(string(FieldTarget(1)), "field:") {
		t.Error("préfixe")
	}
}
