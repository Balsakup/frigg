package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
)

// Rattachement d'une demande publique à une fiche existante (016, FR-038,
// research.md › R13).

// publicRequest dépose et confirme une demande publique pour Claire Martin
// (1985-03-02) et renvoie l'adhésion vue par l'administration.
func (f *pubFixture) publicRequest(t *testing.T, email string, extra ...any) api.AdminMembership {
	t.Helper()
	expectOutcome(t, f.confirm(f.env.Client(), f.deposit(t, f.publicBody(email, extra...))), "tracking")
	p := adminMemberships(t, f.admin, "?origin=public&status=pending")
	for _, row := range p.Items {
		if !row.ViaPublic {
			t.Fatalf("filtre origine : %+v", row)
		}
	}
	if len(p.Items) == 0 {
		t.Fatal("demande publique absente de la liste filtrée")
	}
	return expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(p.Items[len(p.Items)-1].ID, ""), nil), http.StatusOK)
}

func attachPath(id int64) string { return adminMembershipPath(id, "/attach") }

func TestAttachMembershipToMember(t *testing.T) {
	f := newPubFixture(t, true)
	club := createField(t, f.admin, fieldBody("member", "Club précédent", "short_text", "visibility", "editable"))
	licence := createField(t, f.admin, fieldBody("member", "Licence", "short_text", "visibility", "editable"))
	// Fiche existante, sans adresse, avec une licence déjà renseignée.
	existing := expectMember(t, f.admin.Do("POST", "/api/admin/members", memberBody("Claire", "Martin", "1985-03-02",
		"accountId", nil, "customFields", []map[string]any{{"fieldId": licence.ID, "text": "L-1"}})), http.StatusCreated)
	// Demande d'une autre personne : elle n'apparaît pas dans le filtre d'origine.
	c, _, other := f.member(t, "Hugo")
	request(t, c, other.ID, f.season.ID, f.young.ID, "first")

	ms := f.publicRequest(t, "claire@exemple.fr", "customFields", []map[string]any{
		{"fieldId": club.ID, "text": "AS Lyon"}, {"fieldId": licence.ID, "text": "L-2"},
	})
	if !ms.ViaPublic || len(ms.DuplicateCandidates) != 1 || ms.DuplicateCandidates[0].ID != existing.ID {
		t.Fatalf("doublon probable : %+v", ms.DuplicateCandidates)
	}
	if p := adminMemberships(t, f.admin, "?origin=public"); p.Total != 1 {
		t.Fatalf("filtre origine : %d", p.Total)
	}
	source := ms.Member.ID

	got := expectAdminMembership(t, f.admin.Do("POST", attachPath(ms.ID), map[string]any{"memberId": existing.ID}), http.StatusOK)
	if got.Member.ID != existing.ID || got.Status != "pending" || len(got.DuplicateCandidates) != 0 {
		t.Fatalf("après rattachement : %+v", got)
	}
	m := expectMember(t, f.admin.Do("GET", adminMemberPath(existing.ID), nil), http.StatusOK)
	if m.Email == nil || *m.Email != "claire@exemple.fr" {
		t.Fatalf("adresse non reportée : %v", m.Email)
	}
	if v := findValue(t, m.CustomFields, club.ID); v.Text == nil || *v.Text != "AS Lyon" {
		t.Fatalf("champ manquant non copié : %+v", v)
	}
	if v := findValue(t, m.CustomFields, licence.ID); v.Text == nil || *v.Text != "L-1" {
		t.Fatalf("champ de la cible remplacé : %+v", v)
	}
	if res := f.admin.Do("GET", adminMemberPath(source), nil); res.Status != http.StatusNotFound {
		t.Fatalf("fiche publique conservée : %d", res.Status)
	}
	if n := membershipLogs(t, f.env, "membership_attached", "claire@exemple.fr", "Claire"); n != 1 {
		t.Errorf("journal membership_attached : %d", n)
	}

	// Demande déjà rattachée puis validée : plus rattachable.
	validate(t, f.admin, ms.ID, f.adult.ID, "first")
	res := f.admin.Do("POST", attachPath(ms.ID), map[string]any{"memberId": source})
	if res.Status != http.StatusConflict || !strings.Contains(res.Body, "Seule une demande publique en attente") {
		t.Fatalf("demande validée : %d %s", res.Status, res.Body)
	}
}

func TestAttachMembershipToMemberConflicts(t *testing.T) {
	f := newPubFixture(t, true)
	// Cible déjà adhérente de la saison (006 FR-003).
	target := expectMember(t, f.admin.Do("POST", "/api/admin/members", memberBody("Claire", "Martin", "1985-03-02", "accountId", nil)), http.StatusCreated)
	expectAdminMembership(t, f.admin.Do("POST", "/api/admin/memberships", map[string]any{
		"memberId": target.ID, "seasonId": f.season.ID, "feeId": f.adult.ID, "kind": "first",
	}), http.StatusCreated)
	ms := f.publicRequest(t, "claire@exemple.fr")
	res := f.admin.Do("POST", attachPath(ms.ID), map[string]any{"memberId": target.ID})
	if res.Status != http.StatusConflict || !strings.Contains(res.Body, "Ce membre a déjà une adhésion en cours pour cette saison.") {
		t.Fatalf("cible déjà adhérente : %d %s", res.Status, res.Body)
	}
	if got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(ms.ID, ""), nil), http.StatusOK); got.Member.ID != ms.Member.ID {
		t.Fatal("demande déplacée malgré le conflit")
	}

	// Demande qui n'est pas publique.
	c, _, mine := f.member(t, "Lucas")
	own := request(t, c, mine.ID, f.season.ID, f.adult.ID, "first")
	if res := f.admin.Do("POST", attachPath(own.ID), map[string]any{"memberId": target.ID}); res.Status != http.StatusConflict {
		t.Fatalf("demande non publique : %d %s", res.Status, res.Body)
	}
	// Fiche cible inconnue ou identique.
	expectFieldError(t, f.admin.Do("POST", attachPath(ms.ID), map[string]any{"memberId": 9999}), "body.memberId", "Cette fiche n'existe plus.")
	expectFieldError(t, f.admin.Do("POST", attachPath(ms.ID), map[string]any{"memberId": ms.Member.ID}), "body.memberId", "Choisissez une autre fiche que celle de la demande.")

	// Un membre n'y a pas accès.
	if res := c.Do("POST", attachPath(ms.ID), map[string]any{"memberId": target.ID}); res.Status != http.StatusForbidden {
		t.Fatalf("membre : %d", res.Status)
	}
}
