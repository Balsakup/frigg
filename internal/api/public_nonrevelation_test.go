package api_test

import (
	"net/http"
	"testing"

	"github.com/balsakup/frigg/internal/auth"
)

// Non-révélation (016, SC-005) : la réponse d'un dépôt est identique octet
// pour octet (statut, corps, en-têtes hors Date, cookie de session et
// identifiant de requête) pour
// une adresse inconnue, l'adresse d'un compte et l'adresse d'une fiche sans
// compte.
func TestSubmitPublicMembershipNonRevelation(t *testing.T) {
	f := newPubFixture(t, true)
	accountEmail := f.env.Seed(auth.RoleMember, true)
	expectMember(t, f.admin.Do("POST", "/api/admin/members", memberBody("Paul", "Durand", "1990-05-06",
		"email", "fiche@exemple.fr", "accountId", nil)), http.StatusCreated)

	addresses := map[string]string{
		"inconnue":          "inconnue@exemple.fr",
		"compte":            accountEmail,
		"fiche sans compte": "fiche@exemple.fr",
	}
	var (
		ref     string
		refBody string
		refHead http.Header
	)
	for name, email := range addresses {
		res := submit(f.env.Client(), f.nextIP(), f.publicBody(email))
		if res.Status != http.StatusAccepted {
			t.Fatalf("%s : %d %s", name, res.Status, res.Body)
		}
		head := res.Header.Clone()
		head.Del("Date")
		head.Del("Set-Cookie")
		// Identifiant aléatoire propre à chaque requête : il ne dépend pas de
		// l'adresse.
		head.Del("X-Request-Id")
		if ref == "" {
			ref, refBody, refHead = name, res.Body, head
			continue
		}
		if res.Body != refBody {
			t.Errorf("corps différents : %s %q, %s %q", ref, refBody, name, res.Body)
		}
		if len(head) != len(refHead) {
			t.Errorf("en-têtes différents : %s %v, %s %v", ref, refHead, name, head)
		}
		for k, v := range refHead {
			if got := head.Values(k); len(got) != len(v) || (len(v) > 0 && got[0] != v[0]) {
				t.Errorf("en-tête %s : %s %v, %s %v", k, ref, v, name, got)
			}
		}
		if len(f.env.Mails(email)) != 1 {
			t.Errorf("%s : e-mail de confirmation absent", name)
		}
	}
}
