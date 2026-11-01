package membership

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/balsakup/frigg/internal/season"
)

// ReferencePrefix renvoie le préfixe des références de paiement d'une saison :
// « ADH- », années de début et de fin sur deux chiffres, « - » (research.md
// 006 › R4).
func ReferencePrefix(starts, ends season.Date) string {
	return fmt.Sprintf("ADH-%02d%02d-", starts.Year()%100, ends.Year()%100)
}

// NextReference renvoie la référence suivante pour un préfixe, à partir des
// références existantes de ce préfixe : numéro le plus élevé plus un, sur
// quatre chiffres au moins.
func NextReference(prefix string, existing []string) string {
	last := 0
	for _, r := range existing {
		n, err := strconv.Atoi(strings.TrimPrefix(r, prefix))
		if err == nil && n > last {
			last = n
		}
	}
	return fmt.Sprintf("%s%04d", prefix, last+1)
}
