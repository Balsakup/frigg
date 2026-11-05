package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api/apitest"
)

// expectNoLeak vérifie qu'un corps de réponse ne mentionne pas le champ réservé.
func expectNoLeak(t *testing.T, what string, res apitest.Response, id int64, secrets ...string) {
	t.Helper()
	for _, s := range append(secrets, fmt.Sprintf(`"fieldId":%d,`, id), fmt.Sprintf(`"fieldId":%d}`, id), fmt.Sprintf(`"id":%d,"label"`, id)) {
		if strings.Contains(res.Body, s) {
			t.Errorf("%s : %q présent dans la réponse %s", what, s, res.Body)
		}
	}
}

// TestReservedFieldsNeverReachOwner : un champ réservé à l'administration
// (libellé, définition, valeur) n'est jamais transmis au compte qui gère la
// fiche, et une écriture sur ce champ reçoit la même réponse qu'un champ
// inexistant (FR-020, SC-003).
func TestReservedFieldsNeverReachOwner(t *testing.T) {
	f := newMsFixture(t)
	const label, secret = "Remarque-secrete-bureau", "valeur-secrete-xyz"
	reserved := createField(t, f.admin, fieldBody("member", label, "short_text"))
	msReserved := createField(t, f.admin, fieldBody("membership", label+"-adhesion", "short_text"))
	c, _, m := f.member(t, "Lucas")
	insertMemberValue(t, f.env, m.ID, reserved.ID, secret)

	expectNoLeak(t, "get-my-member", c.Do("GET", myMemberPath(m.ID), nil), reserved.ID, label, secret)
	expectNoLeak(t, "list-my-members", c.Do("GET", "/api/me/members", nil), reserved.ID, label, secret)
	expectNoLeak(t, "list-my-custom-fields", c.Do("GET", "/api/me/custom-fields", nil), reserved.ID, label, label+"-adhesion")
	res := c.Do("PUT", myMemberPath(m.ID), memberBody("Lucas", "Martin", "2010-04-12", "confirmDuplicate", true))
	if res.Status != http.StatusOK {
		t.Fatalf("update-my-member : %d %s", res.Status, res.Body)
	}
	expectNoLeak(t, "update-my-member", res, reserved.ID, label, secret)

	// Même erreur pour le champ réservé et pour un champ inexistant (corps
	// identiques une fois l'identifiant remplacé).
	write := func(id int64) string {
		r := c.Do("PUT", myMemberPath(m.ID), memberBody("Lucas", "Martin", "2010-04-12", "confirmDuplicate", true,
			"customFields", []map[string]any{cfText(id, "x")}))
		if r.Status != http.StatusUnprocessableEntity {
			t.Fatalf("écriture du champ %d : %d %s", id, r.Status, r.Body)
		}
		return strings.ReplaceAll(r.Body, fmt.Sprintf("f%d", id), "fID")
	}
	if a, b := write(reserved.ID), write(99999); a != b {
		t.Errorf("réponses différentes : %s ≠ %s", a, b)
	}

	// Adhésions (US3) : get-my-membership sans le champ réservé.
	ms := request(t, c, m.ID, f.season.ID, f.adult.ID, "first")
	expectNoLeak(t, "get-my-membership", c.Do("GET", myMembershipPath(ms.ID), nil), msReserved.ID, label+"-adhesion")
	expectNoLeak(t, "list-my-memberships", c.Do("GET", "/api/me/memberships", nil), msReserved.ID, label+"-adhesion")
}
