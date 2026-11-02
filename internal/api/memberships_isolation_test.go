package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// TestMembershipsIsolation vérifie le cloisonnement des adhésions par compte
// (FR-008, SC-004) : une adhésion d'une fiche d'un autre compte, d'une fiche
// sans compte ou inexistante donne toujours la même réponse et reste intacte.
func TestMembershipsIsolation(t *testing.T) {
	f := newMsFixture(t)
	setPayment(t, f.admin, nil)
	a, _, leo := f.member(t, "Léo")
	b, _, zoe := f.member(t, "Zoé")
	ma := request(t, a, leo.ID, f.season.ID, f.adult.ID, "first")
	validate(t, f.admin, ma.ID, f.adult.ID, "first")
	orphan := insertOrphan(t, f.env, "Jeanne", "Petit", "2010-01-01")
	mo := expectAdminMembership(t, f.admin.Do("POST", "/api/admin/memberships", map[string]any{
		"memberId": orphan, "seasonId": f.season.ID, "feeId": f.young.ID, "kind": "first",
	}), http.StatusCreated)

	read := func(id int64) sqlc.Membership {
		t.Helper()
		r, err := f.env.Store.Read.GetMembership(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		return r.Membership
	}
	beforeA, beforeO := read(ma.ID), read(mo.ID)

	for _, id := range []int64{ma.ID, mo.ID, 99999} {
		for _, op := range []struct {
			method, suffix string
			body           any
		}{
			{"GET", "", nil},
			{"POST", "/cancel", nil},
			{"PUT", "/payment", map[string]any{"method": "transfer"}},
		} {
			res := b.Do(op.method, myMembershipPath(id)+op.suffix, op.body)
			expectProblem(t, res, http.StatusNotFound, api.MsgMembershipGone)
		}
	}
	if read(ma.ID) != beforeA || read(mo.ID) != beforeO {
		t.Fatal("adhésion modifiée par un autre compte")
	}

	// Demande pour une fiche d'un autre compte.
	res := b.Do("POST", "/api/me/memberships", map[string]any{
		"memberId": leo.ID, "seasonId": f.season.ID, "feeId": f.adult.ID, "declaredKind": "first",
	})
	expectProblem(t, res, http.StatusNotFound, api.MsgMemberGone)

	// Listes : seulement ses propres fiches.
	request(t, b, zoe.ID, f.season.ID, f.young.ID, "first")
	var list api.MyMembershipList
	b.Do("GET", "/api/me/memberships", nil).JSON(t, &list)
	for _, s := range list.Seasons {
		for _, m := range s.Items {
			if m.Member.ID != zoe.ID {
				t.Errorf("adhésion d'une autre fiche listée : %+v", m)
			}
		}
	}

	// Un administrateur dans l'espace membre ne voit que ses fiches.
	f.admin.Do("GET", "/api/me/memberships", nil).JSON(t, &list)
	if len(list.Seasons) != 0 {
		t.Errorf("l'administrateur voit des adhésions d'autres comptes : %+v", list)
	}
	expectProblem(t, f.admin.Do("GET", myMembershipPath(ma.ID), nil), http.StatusNotFound, api.MsgMembershipGone)

	// Rattachement changé après validation : l'ancien compte perd l'accès,
	// le nouveau le gagne.
	res = f.admin.Do("GET", adminMemberPath(leo.ID), nil)
	before := expectMember(t, res, http.StatusOK)
	newOwner := f.env.Seed(auth.RoleMember, true)
	c := f.env.LoggedIn(newOwner)
	ownerID := accountID(t, f.env, newOwner)
	res = f.admin.Do("PUT", adminMemberPath(leo.ID), map[string]any{
		"firstName": before.FirstName, "lastName": before.LastName, "birthDate": before.BirthDate,
		"accountId": ownerID, "expectedAccountId": before.Account.ID, "confirmDuplicate": true,
	})
	if res.Status != http.StatusOK {
		t.Fatalf("changement de rattachement : %d %s", res.Status, res.Body)
	}
	expectProblem(t, a.Do("GET", myMembershipPath(ma.ID), nil), http.StatusNotFound, api.MsgMembershipGone)
	if got := expectMine(t, c.Do("GET", myMembershipPath(ma.ID), nil), http.StatusOK); !got.Actions.Pay {
		t.Errorf("nouveau compte : %+v", got)
	}
}

// TestCardPaymentIsolation : un compte ne lance ni ne confirme le paiement par
// carte d'une adhésion qu'il ne gère pas, et aucune tentative n'est créée.
func TestCardPaymentIsolation(t *testing.T) {
	f := newCardFixture(t)
	b, _, _ := f.msFixture.member(t, "Zoé")
	orphan := insertOrphan(t, f.env, "Jeanne", "Petit", "2010-01-01")
	mo := expectAdminMembership(t, f.admin.Do("POST", "/api/admin/memberships", map[string]any{
		"memberId": orphan, "seasonId": f.season.ID, "feeId": f.young.ID, "kind": "first",
	}), http.StatusCreated)
	st := f.start(t)

	count := func() int {
		var n int
		if err := f.env.Store.ReadDB.QueryRow(`SELECT count(*) FROM online_payments`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := count()
	for _, id := range []int64{f.ms.ID, mo.ID, 99999} {
		expectProblem(t, b.Do("POST", cardPath(id, ""), nil), http.StatusNotFound, api.MsgMembershipGone)
		res := b.Do("POST", cardPath(id, "/confirm"), map[string]any{"onlinePaymentId": st.OnlinePaymentID, "outcome": "return"})
		expectProblem(t, res, http.StatusNotFound, api.MsgOnlinePaymentGone)
	}
	if count() != before {
		t.Fatal("tentative créée par un autre compte")
	}
	// La tentative d'une adhésion ne se confirme pas sous une autre.
	res := f.member.Do("POST", cardPath(99999, "/confirm"), map[string]any{"onlinePaymentId": st.OnlinePaymentID, "outcome": "return"})
	expectProblem(t, res, http.StatusNotFound, api.MsgOnlinePaymentGone)
}
