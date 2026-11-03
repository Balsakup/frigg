package communication

import (
	"slices"
	"testing"
)

func TestBuildAudience(t *testing.T) {
	cands := []Candidate{
		{MemberID: 1, FirstName: "Léo", LastName: "Martin", Situation: Due, AccountEmail: "claire@exemple.fr", MemberEmail: "leo@exemple.fr"},
		{MemberID: 2, FirstName: "Inès", LastName: "Martin", Situation: Due, AccountEmail: "Claire@Exemple.fr"},
		{MemberID: 3, FirstName: "Paul", LastName: "Durand", Situation: Announced, AccountEmail: "paul@exemple.fr"},
		{MemberID: 4, FirstName: "Anne", LastName: "Petit", Situation: UpToDate, MemberEmail: " anne@exemple.fr "},
		{MemberID: 5, FirstName: "Hugo", LastName: "Roux", Situation: Due},
		{MemberID: 6, FirstName: "Éric", LastName: "petit", Situation: Due, MemberEmail: "ANNE@exemple.fr"},
	}
	a := BuildAudience(cands)
	if a.MembersCount() != 6 || len(a.Members) != 5 || len(a.Unreachable) != 1 || a.Unreachable[0].MemberID != 5 {
		t.Fatalf("compteurs : %d joignables, %d non joignables", len(a.Members), len(a.Unreachable))
	}
	var order []int64
	for _, m := range a.Members {
		order = append(order, m.MemberID)
	}
	// Durand, Martin Inès, Martin Léo, Petit Anne, petit Éric (sans casse ni accents).
	if !slices.Equal(order, []int64{3, 2, 1, 4, 6}) {
		t.Errorf("ordre : %v", order)
	}
	if a.Members[2].ContactEmail != "claire@exemple.fr" {
		t.Errorf("le compte prime sur la fiche : %s", a.Members[2].ContactEmail)
	}
	if len(a.Recipients) != 3 {
		t.Fatalf("destinataires : %+v", a.Recipients)
	}
	want := []Recipient{
		{Email: "paul@exemple.fr", MemberIDs: []int64{3}},
		{Email: "Claire@Exemple.fr", MemberIDs: []int64{2, 1}},
		{Email: "anne@exemple.fr", MemberIDs: []int64{4, 6}},
	}
	for i, r := range a.Recipients {
		if r.Email != want[i].Email || !slices.Equal(r.MemberIDs, want[i].MemberIDs) {
			t.Errorf("destinataire %d : %+v, attendu %+v", i, r, want[i])
		}
	}
}

func TestBuildAudienceEmpty(t *testing.T) {
	a := BuildAudience(nil)
	if a.MembersCount() != 0 || a.Members == nil || a.Unreachable == nil || a.Recipients == nil {
		t.Fatalf("audience vide : %+v", a)
	}
}

func TestFilter(t *testing.T) {
	cands := []Candidate{
		{MemberID: 1, FeeID: 10, Situation: Due},
		{MemberID: 2, FeeID: 20, Situation: Due},
		{MemberID: 3, FeeID: 10, Situation: UpToDate},
		{MemberID: 4, FeeID: 10, Situation: Announced},
	}
	ids := func(cs []Candidate) []int64 {
		var out []int64
		for _, c := range cs {
			out = append(out, c.MemberID)
		}
		return out
	}
	if got := ids(Filter(cands, SituationsOf(PresetUnpaid), nil)); !slices.Equal(got, []int64{1, 2, 4}) {
		t.Errorf("en impayé : %v", got)
	}
	if got := ids(Filter(cands, []Situation{Due}, []int64{20})); !slices.Equal(got, []int64{2}) {
		t.Errorf("à payer, tarif 20 : %v", got)
	}
	if got := Filter(cands, []Situation{UpToDate}, []int64{20}); len(got) != 0 {
		t.Errorf("cible vide : %v", got)
	}
}
