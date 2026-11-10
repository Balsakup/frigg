package memberimport

import (
	"slices"
	"strconv"
	"testing"

	"github.com/balsakup/frigg/internal/csvfile"
	"github.com/balsakup/frigg/internal/customfield"
)

func TestTemplateHeaders(t *testing.T) {
	defs := defsFixture()
	defs[0].Label = "Licence"
	many := customfield.Definition{ID: 20, Label: "Groupe", Type: customfield.TypeChoice}
	for i := range 12 {
		many.Choices = append(many.Choices, customfield.Choice{ID: int64(100 + i), Label: "G" + strconv.Itoa(i+1)})
	}
	defs = append(defs, many)
	got := TemplateHeaders(defs)
	want := []string{
		"Prénom *", "Nom *", "Date de naissance * (JJ/MM/AAAA)", "Adresse électronique", "Téléphone",
		"Adresse", "Complément d'adresse", "Code postal", "Commune",
		"Licence", "Catégorie (Poussin, Benjamin, Minime)", "Bénévole (oui/non)", "Arrivée (JJ/MM/AAAA)",
		"Taille (cm) (nombre)", "Groupe (G1, G2, G3, G4, G5, G6, G7, G8, G9, G10, …)",
	}
	if !slices.Equal(got, want) {
		t.Errorf("TemplateHeaders =\n%q\nattendu\n%q", got, want)
	}
	targets := Suggest(got, defs)
	for i, tg := range targets {
		if tg == None {
			t.Errorf("colonne %q non reconnue", got[i])
		}
	}
	if targets[9] != FieldTarget(3) || targets[14] != FieldTarget(20) {
		t.Errorf("cibles %q", targets)
	}
}

func TestRejectedRecord(t *testing.T) {
	tab := csvfile.Table{Headers: []string{"Nom", "Prénom", "Naissance", "Tél."}}
	labels := ColumnLabels(tab.Headers)
	if got := RejectedHeaders(tab); !slices.Equal(got, []string{"Nom", "Prénom", "Naissance", "Tél.", "Raison du rejet"}) {
		t.Errorf("RejectedHeaders = %q", got)
	}
	rowData := csvfile.Row{Line: 4, Cells: []string{"Durand", "Hugo", "3/4/12", "+32 470"}}
	errRow := Result{Status: StatusError, Parsed: Parsed{Errors: []CellError{
		{Column: 2, Err: ErrTwoDigitYear},
		{Column: -1, Err: ErrExtraColumns},
	}}}
	got := RejectedRecord(errRow, rowData, labels, plainMessage)
	want := []string{"Durand", "Hugo", "3/4/12", "'+32 470", "Naissance : " + ErrTwoDigitYear.Error() + " ; " + ErrExtraColumns.Error()}
	if !slices.Equal(got, want) {
		t.Errorf("RejectedRecord = %q", got)
	}
	dupMember := Result{Status: StatusDuplicate, DuplicateOfMember: &Identity{ID: 1, FirstName: "Marie", LastName: "Dupont", BirthDate: "2012-03-12"}}
	if r := Reason(dupMember, labels, plainMessage); r != "Doublon de la fiche Dupont Marie (12/03/2012)" {
		t.Errorf("raison %q", r)
	}
	if r := Reason(Result{Status: StatusDuplicate, DuplicateOfLine: 5}, labels, plainMessage); r != "Doublon de la ligne 5" {
		t.Errorf("raison %q", r)
	}
}
