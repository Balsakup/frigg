package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

// msFixture est le jeu de données type des adhésions : saison 2026-2027
// ouverte, tarifs « Adulte » (60 €, renouvellement 40 €), « Jeune » (30 €) et
// « Bienfaiteur » (gratuit), au 1er novembre 2026.
type msFixture struct {
	env                *apitest.Env
	admin              *apitest.Client
	season             api.Season
	adult, young, free api.MembershipFee
}

const msToday = "2026-11-01"

func newMsFixture(t *testing.T) msFixture {
	t.Helper()
	env := apitest.New(t)
	env.SetToday(msToday)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	expectSeason(t, admin.Do("POST", seasonPath(s.ID, "/open"), nil), http.StatusOK)
	f := msFixture{env: env, admin: admin, season: s}
	f.adult = expectFee(t, admin.Do("POST", feesPath(s.ID), map[string]any{
		"label": "Adulte", "amount": "60", "hasRenewalAmount": true, "renewalAmount": "40",
	}), http.StatusCreated)
	f.young = createFee(t, admin, s.ID, map[string]string{"label": "Jeune", "amount": "30"})
	f.free = createFee(t, admin, s.ID, map[string]string{"label": "Bienfaiteur", "amount": "0"})
	return f
}

// member renvoie un client membre connecté et une de ses fiches.
func (f msFixture) member(t *testing.T, first string) (*apitest.Client, string, api.Member) {
	t.Helper()
	email := f.env.Seed(auth.RoleMember, true)
	c := f.env.LoggedIn(email)
	m := createMine(t, c, memberBody(first, "Martin", "2010-04-12", "confirmDuplicate", true))
	return c, email, m
}

func myMembershipPath(id int64) string { return fmt.Sprintf("/api/me/memberships/%d", id) }

func adminMembershipPath(id int64, suffix string) string {
	return fmt.Sprintf("/api/admin/memberships/%d%s", id, suffix)
}

func expectMine(t *testing.T, res apitest.Response, status int) api.MyMembership {
	t.Helper()
	if res.Status != status {
		t.Fatalf("statut %d, attendu %d : %s", res.Status, status, res.Body)
	}
	var m api.MyMembership
	res.JSON(t, &m)
	return m
}

func expectAdminMembership(t *testing.T, res apitest.Response, status int) api.AdminMembership {
	t.Helper()
	if res.Status != status {
		t.Fatalf("statut %d, attendu %d : %s", res.Status, status, res.Body)
	}
	var m api.AdminMembership
	res.JSON(t, &m)
	return m
}

// request dépose une demande d'adhésion.
func request(t *testing.T, c *apitest.Client, memberID, seasonID, feeID int64, kind string) api.MyMembership {
	t.Helper()
	return expectMine(t, c.Do("POST", "/api/me/memberships", map[string]any{
		"memberId": memberID, "seasonId": seasonID, "feeId": feeID, "declaredKind": kind,
	}), http.StatusCreated)
}

// validate valide une demande avec le tarif et le type donnés.
func validate(t *testing.T, admin *apitest.Client, id, feeID int64, kind string) api.AdminMembership {
	t.Helper()
	return expectAdminMembership(t, admin.Do("POST", adminMembershipPath(id, "/validate"), map[string]any{
		"feeId": feeID, "kind": kind,
	}), http.StatusOK)
}

func offer(t *testing.T, c *apitest.Client) api.MembershipOffer {
	t.Helper()
	res := c.Do("GET", "/api/me/memberships/offer", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("offre : %d %s", res.Status, res.Body)
	}
	var o api.MembershipOffer
	res.JSON(t, &o)
	return o
}

func adminMemberships(t *testing.T, c *apitest.Client, query string) api.AdminMembershipPage {
	t.Helper()
	res := c.Do("GET", "/api/admin/memberships"+query, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("liste : %d %s", res.Status, res.Body)
	}
	var p api.AdminMembershipPage
	res.JSON(t, &p)
	return p
}

// setPayment active les trois moyens de paiement.
func setPayment(t *testing.T, admin *apitest.Client, body map[string]any) {
	t.Helper()
	if body == nil {
		body = map[string]any{
			"cash":     map[string]any{"enabled": true, "instructions": "Au local, le samedi matin"},
			"cheque":   map[string]any{"enabled": true, "payee": "Association Frigg", "address": "1 rue du Port\n75001 Paris"},
			"transfer": map[string]any{"enabled": true, "holder": "Association Frigg", "iban": "FR76 3000 6000 0112 3456 7890 189", "bic": "BNPAFRPP"},
		}
	}
	res := admin.Do("PUT", "/api/admin/settings/payment", body)
	if res.Status != http.StatusOK {
		t.Fatalf("paramètres : %d %s", res.Status, res.Body)
	}
}

// membershipLogs compte les lignes de journal d'une action sur les adhésions
// et vérifie qu'aucune ne contient les textes interdits.
func membershipLogs(t *testing.T, env *apitest.Env, action string, forbidden ...string) int {
	t.Helper()
	n := 0
	for _, line := range strings.Split(env.Logs.String(), "\n") {
		if !strings.Contains(line, `"action":"`+action+`"`) {
			continue
		}
		n++
		for _, attr := range []string{`"actor_id"`, `"membership_id"`, `"member_id"`, `"season_id"`, `"space"`} {
			if !strings.Contains(line, attr) {
				t.Errorf("journal sans %s : %s", attr, line)
			}
		}
		for _, f := range forbidden {
			if strings.Contains(line, f) {
				t.Errorf("%q journalisé : %s", f, line)
			}
		}
	}
	return n
}

func itoa(v int64) string { return fmt.Sprint(v) }
