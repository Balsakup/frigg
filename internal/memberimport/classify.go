package memberimport

import "github.com/balsakup/frigg/internal/member"

// Status est l'état d'une ligne à l'aperçu.
type Status string

const (
	StatusReady     Status = "ready"
	StatusError     Status = "error"
	StatusDuplicate Status = "duplicate"
)

// Identity est une fiche existante, réduite à ce qui sert aux doublons.
type Identity struct {
	ID                  int64
	FirstName, LastName string
	BirthDate           string // AAAA-MM-JJ
}

// Result est une ligne classée.
type Result struct {
	Parsed
	Status            Status
	DuplicateOfMember *Identity
	DuplicateOfLine   int
}

func identityKey(first, last, birth string) string {
	return member.IdentityKey(first, last) + "\x00" + birth
}

// Classify classe les lignes avec la règle de doublon de la saisie manuelle
// (005, checkDuplicate) : prénom et nom repliés et date de naissance. Une
// ligne en erreur n'occupe pas de clé : la ligne identique suivante, si elle
// est valide, est importée (research.md › R6).
func Classify(rows []Parsed, existing []Identity) []Result {
	known := make(map[string]*Identity, len(existing))
	for i := range existing {
		e := &existing[i]
		k := identityKey(e.FirstName, e.LastName, e.BirthDate)
		if _, ok := known[k]; !ok {
			known[k] = e
		}
	}
	seen := map[string]int{}
	out := make([]Result, len(rows))
	for i, p := range rows {
		r := Result{Parsed: p, Status: StatusReady}
		switch {
		case len(p.Errors) > 0:
			r.Status = StatusError
		default:
			k := identityKey(p.Input.FirstName, p.Input.LastName, p.Input.BirthDate.String())
			if e, ok := known[k]; ok {
				r.Status, r.DuplicateOfMember = StatusDuplicate, e
			} else if line, ok := seen[k]; ok {
				r.Status, r.DuplicateOfLine = StatusDuplicate, line
			} else {
				seen[k] = p.Line
			}
		}
		out[i] = r
	}
	return out
}

// Counts compte les lignes par état.
func Counts(results []Result) (ready, errs, duplicates int) {
	for _, r := range results {
		switch r.Status {
		case StatusReady:
			ready++
		case StatusError:
			errs++
		case StatusDuplicate:
			duplicates++
		}
	}
	return ready, errs, duplicates
}
