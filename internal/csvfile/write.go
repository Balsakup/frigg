// Package csvfile lit et écrit les fichiers CSV échangés avec les tableurs,
// sans base ni HTTP (research.md 013 › R2, R3).
package csvfile

import (
	"encoding/csv"
	"io"
	"strings"
)

// bom est la marque d'ordre des octets UTF-8 : accents corrects à
// l'ouverture dans Excel.
const bom = "\xef\xbb\xbf"

// formulaStarts sont les caractères qui font interpréter une cellule comme
// une formule par un tableur.
const formulaStarts = "=+-@\t\r"

// SafeCell neutralise une cellule interprétable comme une formule en la
// préfixant d'une apostrophe. Decode retire exactement ce préfixe.
func SafeCell(v string) string {
	if v != "" && strings.ContainsRune(formulaStarts, rune(v[0])) {
		return "'" + v
	}
	return v
}

// NewWriter écrit la marque d'ordre des octets puis renvoie un writer CSV
// séparé par des points-virgules (export par défaut des tableurs en
// français).
func NewWriter(w io.Writer) *csv.Writer {
	_, _ = io.WriteString(w, bom)
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	return cw
}
