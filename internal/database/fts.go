package database

import (
	"strings"
	"unicode/utf8"
)

// ftsMinTermLength est la longueur minimale d'un fragment pour le tokenizer
// trigram : en dessous, FTS5 ne trouve rien.
const ftsMinTermLength = 3

// FTSTrigramQuery transforme une saisie libre en expression MATCH pour une
// table FTS5 trigram (constitution 1.4.0, principe VI ; research.md › R7) :
// découpage aux espaces, fragments de moins de 3 caractères écartés, chaque
// fragment entre guillemets pour neutraliser la syntaxe FTS5, tous requis.
// ok est faux si aucun fragment n'est utilisable.
func FTSTrigramQuery(input string) (query string, ok bool) {
	var terms []string
	for _, w := range strings.Fields(input) {
		if utf8.RuneCountInString(w) < ftsMinTermLength {
			continue
		}
		terms = append(terms, `"`+strings.ReplaceAll(w, `"`, `""`)+`"`)
	}
	return strings.Join(terms, " "), len(terms) > 0
}
