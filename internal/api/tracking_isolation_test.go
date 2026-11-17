package api_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

// Isolation d'un lien personnel (016 FR-014, SC-007) : avec le lien de A,
// toute donnée hors du périmètre reçoit exactement la réponse d'une donnée
// inexistante, et le périmètre est réévalué à chaque requête.
func TestTrackingIsolation(t *testing.T) {
	f := newTrackFixture(t)
	const a, b = "a@exemple.fr", "b@exemple.fr"
	kept := trackedMember(t, f.msFixture, "Alice", a)
	cased := trackedMember(t, f.msFixture, "Arthur", "A@Exemple.FR")
	attached := trackedMember(t, f.msFixture, "Anna", a)
	moved := trackedMember(t, f.msFixture, "Adam", a)
	deleted := trackedMember(t, f.msFixture, "Agathe", a)
	other := trackedMember(t, f.msFixture, "Bruno", b)
	mKept := pendingFor(t, f.msFixture, kept)
	mCased := validatedFor(t, f.msFixture, cased, f.young.ID).ID
	mAttached := pendingFor(t, f.msFixture, attached)
	mMoved := pendingFor(t, f.msFixture, moved)
	mDeleted := pendingFor(t, f.msFixture, deleted)
	mOther := validatedFor(t, f.msFixture, other, f.young.ID).ID

	link := f.env.TrackingClient(a)
	for _, id := range []int64{mKept, mCased, mAttached, mMoved, mDeleted} {
		expectTracked(t, link.Do("GET", trackingPath(id, ""), nil), http.StatusOK)
	}

	// Rattachement à un compte (005 FR-012), changement d'adresse et
	// suppression par un administrateur.
	owner := f.env.Seed(auth.RoleMember, true)
	expectMember(t, f.admin.Do("PUT", adminMemberPath(attached), memberBody("Anna", "Martin", "1990-05-01",
		"email", a, "accountId", accountID(t, f.env, owner), "expectedAccountId", nil, "confirmDuplicate", true)), http.StatusOK)
	expectMember(t, f.admin.Do("PUT", adminMemberPath(moved), memberBody("Adam", "Martin", "1990-05-01",
		"email", "ailleurs@exemple.fr", "accountId", nil, "expectedAccountId", nil, "confirmDuplicate", true)), http.StatusOK)
	if res := f.admin.Do("DELETE", adminMemberPath(deleted), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", res.Status, res.Body)
	}

	ops := []struct {
		method, suffix string
		body           any
	}{
		{"GET", "", nil},
		{"POST", "/cancel", nil},
		{"PUT", "/payment", map[string]any{"method": "cheque"}},
		{"POST", "/card-payment", nil},
		{"POST", "/card-payment/confirm", map[string]any{"onlinePaymentId": 1, "outcome": "return"}},
	}
	for _, op := range ops {
		ref := link.Do(op.method, trackingPath(99999, op.suffix), op.body)
		if ref.Status != http.StatusNotFound {
			t.Fatalf("%s %s inexistante : %d %s", op.method, op.suffix, ref.Status, ref.Body)
		}
		for name, id := range map[string]int64{"de B": mOther, "rattachée": mAttached, "déplacée": mMoved, "supprimée": mDeleted} {
			res := link.Do(op.method, trackingPath(id, op.suffix), op.body)
			if res.Status != ref.Status || res.Body != ref.Body {
				t.Errorf("%s %s adhésion %s : %d %s, attendu %s", op.method, op.suffix, name, res.Status, res.Body, ref.Body)
			}
		}
	}
	if n := count(t, f.env, `SELECT count(*) FROM memberships WHERE id IN (?, ?) AND status = 'pending'`, mAttached, mMoved); n != 2 {
		t.Fatal("adhésion hors périmètre modifiée")
	}

	got := expectJSON[api.Tracking](t, link.Do("GET", "/api/tracking", nil), http.StatusOK)
	ids := []int64{}
	for _, it := range got.Items {
		ids = append(ids, it.ID)
	}
	slices.Sort(ids)
	want := []int64{mKept, mCased}
	slices.Sort(want)
	if !slices.Equal(ids, want) || len(got.Members) != 2 {
		t.Fatalf("périmètre : %v, attendu %v (%+v)", ids, want, got.Members)
	}

	// Une session de compte sans lien n'a pas accès au suivi.
	member, _, mine := f.member(t, "Marc")
	mMine := request(t, member, mine.ID, f.season.ID, f.adult.ID, "first")
	for _, client := range []*apitest.Client{member, f.admin} {
		expectProblem(t, client.Do("GET", "/api/tracking", nil), http.StatusUnauthorized, api.MsgLinkInvalid)
		expectProblem(t, client.Do("GET", trackingPath(mKept, ""), nil), http.StatusUnauthorized, api.MsgLinkInvalid)
	}

	// Une session de compte munie d'un lien ne voit que le périmètre du lien.
	if res := member.Do("POST", "/api/public/tracking/open", map[string]string{"token": issueLink(t, f.env, a)}); res.Status != http.StatusNoContent {
		t.Fatalf("ouverture : %d", res.Status)
	}
	expectTracked(t, member.Do("GET", trackingPath(mKept, ""), nil), http.StatusOK)
	ref := member.Do("GET", trackingPath(99999, ""), nil)
	if res := member.Do("GET", trackingPath(mMine.ID, ""), nil); res.Status != http.StatusNotFound || res.Body != ref.Body {
		t.Fatalf("adhésion du compte par le lien : %d %s", res.Status, res.Body)
	}
}
