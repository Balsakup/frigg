package communication

import (
	"cmp"
	"slices"
	"strings"

	"github.com/balsakup/frigg/internal/textfold"
)

// Candidate est une adhésion validée de la saison, lue en base.
type Candidate struct {
	MemberID     int64
	FirstName    string
	LastName     string
	FeeID        int64
	FeeLabel     string
	Situation    Situation
	AccountEmail string // compte qui gère la fiche, vide si aucun
	MemberEmail  string // adresse de la fiche, vide si aucune
}

// Target est un adhérent ciblé avec son adresse de contact (vide : non joignable).
type Target struct {
	Candidate
	ContactEmail string
}

// Recipient est une adresse distincte et les adhérents qu'elle représente.
type Recipient struct {
	Email     string
	MemberIDs []int64
}

// Audience est le résultat d'une cible.
type Audience struct {
	Members     []Target // joignables, par nom puis prénom
	Unreachable []Target // sans adresse de contact
	Recipients  []Recipient
}

// MembersCount compte les adhérents ciblés, non joignables compris.
func (a Audience) MembersCount() int { return len(a.Members) + len(a.Unreachable) }

// Filter retient les candidats dont la situation figure dans situations et,
// si feeIDs n'est pas vide, dont le tarif figure dans feeIDs.
func Filter(cands []Candidate, situations []Situation, feeIDs []int64) []Candidate {
	out := make([]Candidate, 0, len(cands))
	for _, c := range cands {
		if !slices.Contains(situations, c.Situation) {
			continue
		}
		if len(feeIDs) > 0 && !slices.Contains(feeIDs, c.FeeID) {
			continue
		}
		out = append(out, c)
	}
	return out
}

// BuildAudience trie les candidats par nom puis prénom (sans casse ni
// accents), retient l'adresse du compte s'il y en a un, sinon celle de la
// fiche, et regroupe les adresses identiques sans tenir compte de la casse
// (research.md › R2). La première graphie rencontrée est conservée.
func BuildAudience(cands []Candidate) Audience {
	sorted := slices.Clone(cands)
	slices.SortStableFunc(sorted, func(a, b Candidate) int {
		return cmp.Or(
			cmp.Compare(textfold.Fold(a.LastName), textfold.Fold(b.LastName)),
			cmp.Compare(textfold.Fold(a.FirstName), textfold.Fold(b.FirstName)),
			cmp.Compare(a.MemberID, b.MemberID),
		)
	})
	a := Audience{Members: []Target{}, Unreachable: []Target{}, Recipients: []Recipient{}}
	index := map[string]int{}
	for _, c := range sorted {
		email := strings.TrimSpace(c.AccountEmail)
		if email == "" {
			email = strings.TrimSpace(c.MemberEmail)
		}
		t := Target{Candidate: c, ContactEmail: email}
		if email == "" {
			a.Unreachable = append(a.Unreachable, t)
			continue
		}
		a.Members = append(a.Members, t)
		key := strings.ToLower(email)
		if i, ok := index[key]; ok {
			a.Recipients[i].MemberIDs = append(a.Recipients[i].MemberIDs, c.MemberID)
			continue
		}
		index[key] = len(a.Recipients)
		a.Recipients = append(a.Recipients, Recipient{Email: email, MemberIDs: []int64{c.MemberID}})
	}
	return a
}
