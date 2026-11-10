package memberimport

import (
	"errors"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/csvfile"
	"github.com/balsakup/frigg/internal/member"
)

func TestParseDate(t *testing.T) {
	for _, s := range []string{"3/4/2012", "03-04-2012", "03.04.2012", "2012-04-03", " 3/04/2012 "} {
		d, err := ParseDate(s)
		if err != nil || d.String() != "2012-04-03" {
			t.Errorf("ParseDate(%q) = %v, %v", s, d, err)
		}
	}
	cases := map[string]error{
		"3/4/12":     ErrTwoDigitYear,
		"31/02/2012": ErrDateInvalid,
		"2012/04/03": ErrDateFormat,
		"12/03":      ErrDateFormat,
		"3/4/201":    ErrDateFormat,
		"hier":       ErrDateFormat,
	}
	for s, want := range cases {
		if _, err := ParseDate(s); !errors.Is(err, want) {
			t.Errorf("ParseDate(%q) : %v, attendu %v", s, err, want)
		}
	}
}

var stdMapping = []Target{LastName, FirstName, BirthDate, Email, Phone, AddressLine1, PostalCode, City}

func row(cells ...string) csvfile.Row { return csvfile.Row{Line: 2, Cells: cells} }

func TestParseRowValid(t *testing.T) {
	p := ParseRow(row("Lefèvre", " Léa ", "03/04/2014", "Lea@Example.org", "06 12 34 56 78", "3 rue Neuve", "69001", "Lyon"), stdMapping, nil, today)
	if len(p.Errors) != 0 {
		t.Fatalf("erreurs : %+v", p.Errors)
	}
	c := p.Input
	if c.FirstName != "Léa" || c.BirthDate.String() != "2014-04-03" || *c.Email != "lea@example.org" || *c.Phone != "0612345678" || *c.City != "Lyon" {
		t.Errorf("fiche %+v", c)
	}
	if p.Raw.FirstName != " Léa " || p.Raw.BirthDate != "03/04/2014" {
		t.Errorf("valeurs brutes %+v", p.Raw)
	}
}

func TestParseRowErrors(t *testing.T) {
	p := ParseRow(row("", "Paul", "01/01/2015", "paul.example.org", "", "1 place", "1000", "Lyon"), stdMapping, nil, today)
	type want struct {
		col int
		err error
	}
	expected := []want{{0, member.ErrRequired}, {3, member.ErrEmailFormat}, {6, member.ErrPostalCode}}
	if len(p.Errors) != len(expected) {
		t.Fatalf("erreurs %+v", p.Errors)
	}
	for i, w := range expected {
		if p.Errors[i].Column != w.col || !errors.Is(p.Errors[i].Err, w.err) {
			t.Errorf("erreur %d : %+v, attendu %+v", i, p.Errors[i], w)
		}
	}

	extra := row("Dupont", "Marie", "12/03/2012", "", "", "", "", "")
	extra.Extra = true
	if p := ParseRow(extra, stdMapping, nil, today); len(p.Errors) != 1 || p.Errors[0].Column != -1 || !errors.Is(p.Errors[0].Err, ErrExtraColumns) {
		t.Errorf("ligne longue : %+v", p.Errors)
	}

	long := strings.Repeat("a", 81)
	p = ParseRow(row(long, "Marie", "12/03/2012", "", "", "", "", ""), stdMapping, nil, today)
	var le *LengthError
	if len(p.Errors) != 1 || !errors.As(p.Errors[0].Err, &le) || le.Max != 80 || p.Errors[0].Column != 0 {
		t.Errorf("nom trop long : %+v", p.Errors)
	}

	p = ParseRow(row("Martin", "Paul", "3/4/12", "", "", "", "", ""), stdMapping, nil, today)
	if len(p.Errors) != 1 || !errors.Is(p.Errors[0].Err, ErrTwoDigitYear) || p.Errors[0].Column != 2 {
		t.Errorf("année sur 2 chiffres : %+v", p.Errors)
	}
	p = ParseRow(row("Martin", "Paul", "", "", "", "", "", ""), stdMapping, nil, today)
	if len(p.Errors) != 1 || !errors.Is(p.Errors[0].Err, ErrBirthRequired) {
		t.Errorf("date absente : %+v", p.Errors)
	}
	p = ParseRow(row("Martin", "Paul", "01/01/2030", "", "", "", "", ""), stdMapping, nil, today)
	if len(p.Errors) != 1 || !errors.Is(p.Errors[0].Err, member.ErrBirthFuture) {
		t.Errorf("date future : %+v", p.Errors)
	}
}

func TestParseRowAddressColumn(t *testing.T) {
	// Code postal et commune non associés : l'erreur porte sur la colonne d'adresse.
	m := []Target{LastName, FirstName, BirthDate, AddressLine1}
	p := ParseRow(row("Durand", "Hugo", "30/06/2011", "1 place"), m, nil, today)
	if len(p.Errors) != 2 {
		t.Fatalf("erreurs %+v", p.Errors)
	}
	for _, e := range p.Errors {
		if e.Column != 3 || !errors.Is(e.Err, member.ErrAddressIncomplete) {
			t.Errorf("erreur %+v", e)
		}
	}
}

var fieldMapping = []Target{LastName, FirstName, BirthDate, FieldTarget(3), FieldTarget(5), FieldTarget(7), FieldTarget(8), FieldTarget(9)}

func TestParseRowCustomFieldsAccepted(t *testing.T) {
	defs := defsFixture()
	p := ParseRow(row("Lefèvre", "Léa", "03/04/2014", strings.Repeat("é", 200), "POUSSIN", "x", "03/04/2012", "12,5"), fieldMapping, defs, today)
	if len(p.Errors) != 0 {
		t.Fatalf("erreurs %+v", p.Errors)
	}
	if v := p.Fields[3]; v.Text == nil || len([]rune(*v.Text)) != 200 {
		t.Errorf("texte court %+v", v)
	}
	if v := p.Fields[5]; v.ChoiceID == nil || *v.ChoiceID != 51 {
		t.Errorf("choix %+v", v)
	}
	if v := p.Fields[7]; v.Bool == nil || !*v.Bool {
		t.Errorf("oui/non %+v", v)
	}
	if v := p.Fields[8]; v.Date == nil || *v.Date != "2012-04-03" {
		t.Errorf("date %+v", v)
	}
	if v := p.Fields[9]; v.Number == nil || *v.Number != 12.5 {
		t.Errorf("nombre %+v", v)
	}
	if len(p.MissingRequired) != 0 {
		t.Errorf("manquants %v", p.MissingRequired)
	}

	for _, yes := range []string{"Oui", "VRAI", "1", "x"} {
		if p := ParseRow(row("A", "B", "03/04/2014", "", "", yes, "", ""), fieldMapping, defs, today); p.Fields[7].Bool == nil {
			t.Errorf("%q non coché", yes)
		}
	}
	for _, no := range []string{"non", "Faux", "0"} {
		p := ParseRow(row("A", "B", "03/04/2014", "", "", no, "", ""), fieldMapping, defs, today)
		if len(p.Errors) != 0 || !p.Fields[7].IsEmpty() {
			t.Errorf("%q : %+v %+v", no, p.Errors, p.Fields[7])
		}
	}
}

func TestParseRowCustomFieldsRejected(t *testing.T) {
	defs := defsFixture()
	p := ParseRow(row("Lefèvre", "Léa", "03/04/2014", strings.Repeat("é", 201), "Cadet", "peut-être", "", "douze"), fieldMapping, defs, today)
	cols := map[int]string{}
	for _, e := range p.Errors {
		cols[e.Column] = e.Err.Error()
	}
	want := map[int]string{
		3: "200 caractères au plus.",
		4: "Choix non reconnu. Choix possibles : Poussin, Benjamin, Minime",
		5: ErrBoolValue.Error(),
		7: "Saisissez un nombre.",
	}
	for c, msg := range want {
		if cols[c] != msg {
			t.Errorf("colonne %d : %q, attendu %q", c, cols[c], msg)
		}
	}
	many := make([]string, 12)
	for i := range many {
		many[i] = string(rune('A' + i))
	}
	if got := (&ChoiceError{Choices: many}).Error(); !strings.HasSuffix(got, "I, J…") {
		t.Errorf("choix tronqués : %q", got)
	}
}

func TestParseRowMissingRequired(t *testing.T) {
	defs := defsFixture()
	p := ParseRow(row("Martin", "Paul", "03/04/2012"), []Target{LastName, FirstName, BirthDate}, defs, today)
	if len(p.Errors) != 0 || len(p.MissingRequired) != 1 || p.MissingRequired[0] != 3 {
		t.Errorf("non associé : %+v %v", p.Errors, p.MissingRequired)
	}
}
