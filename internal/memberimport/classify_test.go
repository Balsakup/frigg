package memberimport

import (
	"testing"

	"github.com/balsakup/frigg/internal/csvfile"
)

func parsed(t *testing.T, line int, last, first, birth string) Parsed {
	t.Helper()
	return ParseRow(csvfile.Row{Line: line, Cells: []string{last, first, birth}}, []Target{LastName, FirstName, BirthDate}, nil, today)
}

func TestClassify(t *testing.T) {
	existing := []Identity{{ID: 12, FirstName: "Marie", LastName: "Dupont", BirthDate: "2012-03-12"}}
	rows := []Parsed{
		parsed(t, 2, "Lefèvre  ", " Léa", "03/04/2014"),
		parsed(t, 3, "DUPONT", "marie", "12/03/2012"),
		parsed(t, 4, "Martin", "Paul", "3/4/12"),
		parsed(t, 5, "Martin", "Paul", "15-09-2013"),
		parsed(t, 6, "martin", "PAUL", "15.09.2013"),
		parsed(t, 7, "lefevre", "lea", "03/04/2014"),
		parsed(t, 8, "Lefèvre", "Léa", "04/04/2014"),
		parsed(t, 9, "", "Bob", "01/01/2015"),
		parsed(t, 10, "", "Bob", "01/01/2015"),
	}
	res := Classify(rows, existing)
	want := []struct {
		status Status
		member int64
		line   int
	}{
		{StatusReady, 0, 0},
		{StatusDuplicate, 12, 0},
		{StatusError, 0, 0},
		{StatusReady, 0, 0},
		{StatusDuplicate, 0, 5},
		{StatusDuplicate, 0, 2},
		{StatusReady, 0, 0},
		{StatusError, 0, 0},
		{StatusError, 0, 0},
	}
	for i, w := range want {
		r := res[i]
		var id int64
		if r.DuplicateOfMember != nil {
			id = r.DuplicateOfMember.ID
		}
		if r.Status != w.status || id != w.member || r.DuplicateOfLine != w.line {
			t.Errorf("ligne %d : %s fiche %d ligne %d, attendu %+v", r.Line, r.Status, id, r.DuplicateOfLine, w)
		}
	}
	if ready, errs, dup := Counts(res); ready != 3 || errs != 3 || dup != 3 {
		t.Errorf("Counts = %d, %d, %d", ready, errs, dup)
	}
}

func TestClassifyErrorDoesNotTakeKey(t *testing.T) {
	bad := parsed(t, 2, "Martin", "Paul", "15/09/2013")
	bad.Errors = append(bad.Errors, CellError{Column: -1, Err: ErrExtraColumns})
	res := Classify([]Parsed{bad, parsed(t, 3, "Martin", "Paul", "15/09/2013")}, nil)
	if res[0].Status != StatusError || res[1].Status != StatusReady {
		t.Errorf("états %s, %s", res[0].Status, res[1].Status)
	}
}
