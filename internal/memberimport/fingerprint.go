package memberimport

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"

	"github.com/balsakup/frigg/internal/customfield"
)

// Fingerprint résume les champs personnalisés importables : identifiant,
// type, libellé et choix. Elle change dès qu'un champ est créé, modifié ou
// archivé, ce qui invalide un aperçu (FR-017, research.md › R8).
func Fingerprint(defs []customfield.Definition) string {
	sorted := slices.Clone(defs)
	slices.SortFunc(sorted, func(a, b customfield.Definition) int { return cmp.Compare(a.ID, b.ID) })
	var b strings.Builder
	for _, d := range sorted {
		b.WriteString(strconv.FormatInt(d.ID, 10) + "\x1f" + string(d.Type) + "\x1f" + d.Label + "\x1f" + strconv.FormatBool(d.Required))
		choices := slices.Clone(d.Choices)
		slices.SortFunc(choices, func(a, b customfield.Choice) int { return cmp.Compare(a.ID, b.ID) })
		for _, c := range choices {
			b.WriteString("\x1f" + strconv.FormatInt(c.ID, 10) + "\x1e" + c.Label)
		}
		b.WriteString("\x1d")
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:16])
}
