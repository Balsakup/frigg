package api_test

import (
	"net/http"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

// TestMembersIsolation vérifie le cloisonnement des fiches par compte (US2,
// FR-008, SC-002) : une fiche d'un autre compte, sans compte ou inexistante
// donne toujours la même réponse et reste intacte.
func TestMembersIsolation(t *testing.T) {
	env := apitest.New(t)
	env.SetToday(membersToday)
	a := env.LoggedIn(env.Seed(auth.RoleMember, true))
	b := env.LoggedIn(env.Seed(auth.RoleMember, true))
	c := env.LoggedIn(env.Seed(auth.RoleAdmin, true))

	fa := createMine(t, a, memberBody("Léo", "Martin", "2015-04-12", "phone", "0612345678"))
	fn := insertOrphan(t, env, "Jeanne", "Petit", "2010-01-01")
	before := expectMember(t, a.Do("GET", myMemberPath(fa.ID), nil), http.StatusOK)

	var reference string
	for _, id := range []int64{fa.ID, fn, 99999} {
		for _, op := range []struct {
			method string
			body   any
		}{
			{"GET", nil},
			{"PUT", memberBody("Pirate", "Pirate", "2000-01-01")},
			{"DELETE", nil},
		} {
			res := b.Do(op.method, myMemberPath(id), op.body)
			expectProblem(t, res, http.StatusNotFound, api.MsgMemberGone)
			if op.method == "GET" {
				if reference == "" {
					reference = res.Body
				} else if res.Body != reference {
					t.Errorf("réponse différente pour %d : %s ≠ %s", id, res.Body, reference)
				}
			}
		}
	}

	after := expectMember(t, a.Do("GET", myMemberPath(fa.ID), nil), http.StatusOK)
	if after.FirstName != before.FirstName || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("fiche de A modifiée : %+v", after)
	}
	for _, client := range []*apitest.Client{b, c} {
		for _, m := range myMembers(t, client).Items {
			if m.ID == fa.ID || m.ID == fn {
				t.Fatalf("fiche %d visible d'un autre compte", m.ID)
			}
		}
	}

	// Propriété inconnue refusée : impossible de changer de rattachement.
	res := b.Do("PUT", myMemberPath(fa.ID), memberBody("Léo", "Martin", "2015-04-12", "accountId", 1))
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("accountId dans le corps membre : %d, attendu 422", res.Status)
	}
	if res := a.Do("PUT", myMemberPath(fa.ID), memberBody("Léo", "Martin", "2015-04-12", "accountId", 1)); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("accountId dans le corps membre (propriétaire) : %d, attendu 422", res.Status)
	}

	for _, path := range []string{"/api/admin/members", adminMemberPath(fa.ID)} {
		if res := b.Do("GET", path, nil); res.Status != http.StatusForbidden {
			t.Errorf("%s pour un membre : %d, attendu 403", path, res.Status)
		}
	}
}
