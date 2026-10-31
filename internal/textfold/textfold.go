// Package textfold replie le texte saisi pour les comparaisons sans casse ni
// accents (constitution, principe VI).
package textfold

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Fold renvoie la forme repliée de s : espaces de début et de fin retirés,
// espaces intérieurs réduits à un seul (insécables compris, puisque
// strings.Fields suit la propriété Unicode White_Space), diacritiques
// supprimés et minuscules. Deux textes de même forme repliée sont
// considérés comme identiques.
func Fold(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range norm.NFD.String(s) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
