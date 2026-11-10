package csvfile

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// Limits borne un fichier lu.
type Limits struct {
	MaxBytes   int64
	MaxRows    int
	MaxColumns int
}

// DefaultLimits sont les limites de l'importation des membres (FR-002).
var DefaultLimits = Limits{MaxBytes: 5 << 20, MaxRows: 2000, MaxColumns: 100}

// Encodages reconnus.
const (
	EncodingUTF8        = "utf-8"
	EncodingWindows1252 = "windows-1252"
)

// Table est un fichier lu : intitulés et lignes de données non vides.
type Table struct {
	Headers   []string
	Rows      []Row
	Encoding  string
	Separator rune
}

// Row est une ligne de données. Line est le numéro de ligne affiché par le
// tableur, la ligne d'intitulés valant 1. Cells a
// toujours autant de cellules que d'intitulés ; Extra signale des cellules
// non vides au-delà.
type Row struct {
	Line  int
	Cells []string
	Extra bool
}

var (
	ErrEmpty          = errors.New("fichier vide")
	ErrBinary         = errors.New("fichier binaire")
	ErrTooLarge       = errors.New("fichier trop volumineux")
	ErrTooManyRows    = errors.New("trop de lignes")
	ErrTooManyColumns = errors.New("trop de colonnes")
	ErrNoData         = errors.New("aucune ligne de données")
)

// MalformedError signale un enregistrement illisible (guillemet non refermé).
type MalformedError struct{ Line int }

func (e *MalformedError) Error() string { return fmt.Sprintf("CSV illisible à la ligne %d", e.Line) }

// Decode lit un fichier CSV : encodage (UTF-8 ou Windows-1252) et séparateur
// (« ; », « , » ou tabulation) détectés, limites vérifiées, cellules
// débarrassées des espaces de bord et de la neutralisation de SafeCell.
func Decode(data []byte, l Limits) (Table, error) {
	if int64(len(data)) > l.MaxBytes {
		return Table{}, ErrTooLarge
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return Table{}, ErrEmpty
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return Table{}, ErrBinary
	}
	t := Table{Encoding: EncodingUTF8}
	data = bytes.TrimPrefix(data, []byte(bom))
	text := string(data)
	if !utf8.Valid(data) {
		decoded, err := charmap.Windows1252.NewDecoder().Bytes(data)
		if err != nil {
			return Table{}, ErrBinary
		}
		text, t.Encoding = string(decoded), EncodingWindows1252
	}
	if strings.TrimSpace(text) == "" {
		return Table{}, ErrEmpty
	}
	t.Separator = detectSeparator(text)

	r := csv.NewReader(strings.NewReader(text))
	r.Comma = t.Separator
	r.FieldsPerRecord = -1
	r.LazyQuotes = false
	// Numéro de ligne du tableur : ligne physique de début de
	// l'enregistrement, moins les sauts de ligne contenus dans les cellules
	// des enregistrements précédents (encoding/csv saute les lignes vides).
	embedded := 0
	for first := true; ; first = false {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			line := 1
			var pe *csv.ParseError
			if errors.As(err, &pe) {
				line = pe.StartLine - embedded
			}
			return Table{}, &MalformedError{Line: line}
		}
		start, _ := r.FieldPos(0)
		record := start - embedded
		for _, v := range rec {
			embedded += strings.Count(v, "\n")
		}
		if first {
			if len(rec) > l.MaxColumns {
				return Table{}, ErrTooManyColumns
			}
			t.Headers = make([]string, len(rec))
			for i, h := range rec {
				t.Headers[i] = unescape(strings.TrimSpace(h))
			}
			continue
		}
		row, empty := makeRow(record, rec, len(t.Headers))
		if empty {
			continue
		}
		if len(t.Rows) == l.MaxRows {
			return Table{}, ErrTooManyRows
		}
		t.Rows = append(t.Rows, row)
	}
	if len(t.Rows) == 0 {
		return Table{}, ErrNoData
	}
	return t, nil
}

// makeRow ajuste un enregistrement au nombre d'intitulés ; empty signale une
// ligne dont toutes les cellules sont vides.
func makeRow(line int, rec []string, width int) (Row, bool) {
	row := Row{Line: line, Cells: make([]string, width)}
	empty := true
	for i, v := range rec {
		v = unescape(strings.TrimSpace(v))
		if v != "" {
			empty = false
		}
		if i < width {
			row.Cells[i] = v
		} else if v != "" {
			row.Extra = true
		}
	}
	return row, empty
}

// unescape retire l'apostrophe ajoutée par SafeCell, et rien d'autre.
func unescape(v string) string {
	if len(v) >= 2 && v[0] == '\'' && strings.ContainsRune(formulaStarts, rune(v[1])) {
		return v[1:]
	}
	return v
}

// detectSeparator compte « ; », « , » et tabulations hors guillemets sur la
// première ligne logique ; « ; » l'emporte en cas d'égalité.
func detectSeparator(text string) rune {
	counts := map[rune]int{}
	quoted := false
	for _, c := range text {
		if c == '"' {
			quoted = !quoted
			continue
		}
		if quoted {
			continue
		}
		if c == '\n' || c == '\r' {
			break
		}
		counts[c]++
	}
	best := ';'
	for _, c := range []rune{',', '\t'} {
		if counts[c] > counts[best] {
			best = c
		}
	}
	return best
}
