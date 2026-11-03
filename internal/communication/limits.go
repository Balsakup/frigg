// Package communication porte les règles pures des communications (envoi
// d'e-mails groupés aux adhérents d'une saison, specs/009-bulk-email) : cible,
// audience, pièces jointes. Il n'accède pas à la base.
package communication

import "regexp"

// Limites d'une communication (FR-010, FR-017a, FR-017c). 1 Mo = 1 048 576 octets.
const (
	MaxFiles       = 5
	MaxFileBytes   = 5 << 20
	MaxTotalBytes  = 10 << 20
	SubjectMax     = 150
	BodyMax        = 10000
	TargetLabelMax = 300
	MaxFees        = 50
)

// DraftKeyPattern est le format de la clé de préparation : un UUID v4 en
// minuscules, généré par le navigateur.
const DraftKeyPattern = `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`

var draftKeyRe = regexp.MustCompile(DraftKeyPattern)

// ValidDraftKey indique si key est une clé de préparation bien formée.
func ValidDraftKey(key string) bool { return draftKeyRe.MatchString(key) }
