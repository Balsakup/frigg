package csvfile

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

func win1252(t *testing.T, s string) []byte {
	t.Helper()
	b, err := charmap.Windows1252.NewEncoder().Bytes([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func decode(t *testing.T, data []byte) Table {
	t.Helper()
	tab, err := Decode(data, DefaultLimits)
	if err != nil {
		t.Fatalf("Decode : %v", err)
	}
	return tab
}

func TestDecodeEncoding(t *testing.T) {
	const content = "Nom;Prénom\r\nLefèvre;Françoise œuvre\r\n"
	cases := []struct {
		name string
		data []byte
		enc  string
	}{
		{"utf-8", []byte(content), EncodingUTF8},
		{"utf-8 avec BOM", []byte(bom + content), EncodingUTF8},
		{"windows-1252", win1252(t, content), EncodingWindows1252},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tab := decode(t, c.data)
			if tab.Encoding != c.enc {
				t.Errorf("encodage %q, attendu %q", tab.Encoding, c.enc)
			}
			if !slices.Equal(tab.Headers, []string{"Nom", "Prénom"}) {
				t.Errorf("intitulés %q", tab.Headers)
			}
			if got := tab.Rows[0].Cells; !slices.Equal(got, []string{"Lefèvre", "Françoise œuvre"}) {
				t.Errorf("cellules %q", got)
			}
		})
	}
}

func TestDecodeSeparator(t *testing.T) {
	cases := []struct {
		name string
		data string
		want rune
	}{
		{"point-virgule", "a;b;c\n1;2;3\n", ';'},
		{"virgule", "a,b,c\n1,2,3\n", ','},
		{"tabulation", "a\tb\tc\n1\t2\t3\n", '\t'},
		{"égalité", "a;b,c\n1;2,3\n", ';'},
		{"aucun", "nom\nDupont\n", ';'},
		{"virgule entre guillemets ignorée", "\"a,b,c\";d\n1;2\n", ';'},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := decode(t, []byte(c.data)).Separator; got != c.want {
				t.Errorf("séparateur %q, attendu %q", got, c.want)
			}
		})
	}
}

func TestDecodeQuotes(t *testing.T) {
	tab := decode(t, []byte("Nom;Adresse\r\nDupont;\"1 rue; bât. A\r\n2e étage\"\r\nMartin;Lyon\r\n"))
	if got := tab.Rows[0].Cells[1]; got != "1 rue; bât. A\r\n2e étage" && got != "1 rue; bât. A\n2e étage" {
		t.Errorf("valeur entre guillemets %q", got)
	}
	if tab.Rows[1].Line != 3 {
		t.Errorf("ligne %d, attendu 3 (numéro d'enregistrement)", tab.Rows[1].Line)
	}

	_, err := Decode([]byte("Nom;Prénom\nDupont;Marie\nMartin;\"Paul\nDurand;Hugo\n"), DefaultLimits)
	var me *MalformedError
	if !errors.As(err, &me) || me.Line != 3 {
		t.Errorf("guillemet non refermé : %v, attendu MalformedError ligne 3", err)
	}
}

func TestDecodeRejected(t *testing.T) {
	many := func(n int) []byte {
		var b strings.Builder
		b.WriteString("Nom\n")
		for range n {
			b.WriteString("x\n")
		}
		return []byte(b.String())
	}
	cases := []struct {
		name string
		data []byte
		want error
	}{
		{"vide", nil, ErrEmpty},
		{"espaces", []byte(" \r\n "), ErrEmpty},
		{"classeur zip", []byte("PK\x03\x04\x14\x00\x00"), ErrBinary},
		{"en-tête seul", []byte("Nom;Prénom\r\n"), ErrNoData},
		{"en-tête et lignes vides", []byte("Nom;Prénom\r\n;\r\n ; \r\n"), ErrNoData},
		{"2 001 lignes", many(2001), ErrTooManyRows},
		{"101 colonnes", []byte(strings.Repeat("a;", 100) + "a\n" + "x\n"), ErrTooManyColumns},
		{"5 Mio + 1 octet", bytes.Repeat([]byte("a"), 5<<20+1), ErrTooLarge},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Decode(c.data, DefaultLimits); !errors.Is(err, c.want) {
				t.Errorf("erreur %v, attendu %v", err, c.want)
			}
		})
	}
	if tab := decode(t, many(2000)); len(tab.Rows) != 2000 {
		t.Errorf("2 000 lignes : %d lues", len(tab.Rows))
	}
	if tab := decode(t, []byte(strings.Repeat("a;", 99)+"a\nx\n")); len(tab.Headers) != 100 {
		t.Errorf("100 colonnes : %d lues", len(tab.Headers))
	}
}

func TestDecodeRows(t *testing.T) {
	tab := decode(t, []byte("A;B;C\n1;2;3\n\n;;\n4\n5;6;7;;\n8;9;10;11\n"))
	want := []Row{
		{Line: 2, Cells: []string{"1", "2", "3"}},
		{Line: 5, Cells: []string{"4", "", ""}},
		{Line: 6, Cells: []string{"5", "6", "7"}},
		{Line: 7, Cells: []string{"8", "9", "10"}, Extra: true},
	}
	if len(tab.Rows) != len(want) {
		t.Fatalf("%d lignes, attendu %d : %+v", len(tab.Rows), len(want), tab.Rows)
	}
	for i, w := range want {
		g := tab.Rows[i]
		if g.Line != w.Line || g.Extra != w.Extra || !slices.Equal(g.Cells, w.Cells) {
			t.Errorf("ligne %d : %+v, attendu %+v", i, g, w)
		}
	}
}

func TestSafeCellRoundTrip(t *testing.T) {
	values := []string{"+32470123456", "=SOMME(A1)", "-5", "@cmd", "'Ohana", "Dupont"}
	var buf bytes.Buffer
	w := NewWriter(&buf)
	_ = w.Write([]string{"Valeur"})
	for _, v := range values {
		_ = w.Write([]string{SafeCell(v)})
	}
	w.Flush()
	if !bytes.HasPrefix(buf.Bytes(), []byte(bom)) {
		t.Fatal("BOM absente")
	}
	if !strings.Contains(buf.String(), "'+32470123456") {
		t.Errorf("téléphone non neutralisé : %q", buf.String())
	}
	tab := decode(t, buf.Bytes())
	for i, v := range values {
		if got := tab.Rows[i].Cells[0]; got != v {
			t.Errorf("aller-retour %q → %q", v, got)
		}
	}
	if got := decode(t, []byte("Nom\n  Dupont \t\n")).Rows[0].Cells[0]; got != "Dupont" {
		t.Errorf("espaces de bord : %q", got)
	}
}
