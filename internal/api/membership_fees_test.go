package api_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

func feesPath(seasonID int64) string { return seasonPath(seasonID, "/fees") }

func feePath(seasonID, feeID int64) string {
	return seasonPath(seasonID, "/fees/"+strconv.FormatInt(feeID, 10))
}

// expectFee vérifie le statut et décode le tarif.
func expectFee(t *testing.T, res apitest.Response, status int) api.MembershipFee {
	t.Helper()
	if res.Status != status {
		t.Fatalf("statut %d, attendu %d : %s", res.Status, status, res.Body)
	}
	var f api.MembershipFee
	res.JSON(t, &f)
	return f
}

// createFee ajoute un tarif et le renvoie.
func createFee(t *testing.T, c *apitest.Client, seasonID int64, body map[string]string) api.MembershipFee {
	t.Helper()
	return expectFee(t, c.Do("POST", feesPath(seasonID), body), http.StatusCreated)
}

func listFees(t *testing.T, c *apitest.Client, seasonID int64) api.MembershipFeeList {
	t.Helper()
	res := c.Do("GET", feesPath(seasonID), nil)
	if res.Status != http.StatusOK {
		t.Fatalf("tarifs : %d %s", res.Status, res.Body)
	}
	var l api.MembershipFeeList
	res.JSON(t, &l)
	return l
}

func feeLabels(l api.MembershipFeeList) []string {
	out := make([]string, len(l.Items))
	for i, f := range l.Items {
		out[i] = f.Label
	}
	return out
}

// feeLogs compte les lignes de journal d'une action sur les tarifs, vérifie
// leurs attributs et l'absence des libellés donnés.
func feeLogs(t *testing.T, env *apitest.Env, action string, labels ...string) int {
	t.Helper()
	n := 0
	for _, line := range strings.Split(env.Logs.String(), "\n") {
		if !strings.Contains(line, `"action":"`+action+`"`) {
			continue
		}
		n++
		for _, attr := range []string{`"actor_id"`, `"season_id"`, `"fee_id"`} {
			if !strings.Contains(line, attr) {
				t.Errorf("journal sans %s : %s", attr, line)
			}
		}
		for _, l := range labels {
			if strings.Contains(line, l) {
				t.Errorf("libellé %q journalisé : %s", l, line)
			}
		}
	}
	return n
}

func TestCreateMembershipFees(t *testing.T) {
	env, admin := seasonEnv(t)
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")

	l := listFees(t, admin, s.ID)
	if len(l.Items) != 0 || l.Actions != (api.MembershipFeeActions{Add: true, Edit: true, Delete: true}) {
		t.Fatalf("liste vide : %+v", l)
	}

	full := createFee(t, admin, s.ID, map[string]string{"label": "Plein tarif", "amount": "30"})
	if full.AmountCents != 3000 || full.Description != nil {
		t.Fatalf("plein tarif : %+v", full)
	}
	reduced := createFee(t, admin, s.ID, map[string]string{"label": "  Tarif réduit ", "amount": "15,50", "description": "Étudiants"})
	if reduced.Label != "Tarif réduit" || reduced.AmountCents != 1550 || reduced.Description == nil || *reduced.Description != "Étudiants" {
		t.Fatalf("tarif réduit : %+v", reduced)
	}
	createFee(t, admin, s.ID, map[string]string{"label": "Membre d'honneur", "amount": "0"})
	if got, want := feeLabels(listFees(t, admin, s.ID)), []string{"Plein tarif", "Tarif réduit", "Membre d'honneur"}; !slices.Equal(got, want) {
		t.Fatalf("ordre : %v, attendu %v", got, want)
	}

	post := func(body map[string]string) apitest.Response { return admin.Do("POST", feesPath(s.ID), body) }
	expectFieldError(t, post(map[string]string{"label": "plein TARIF", "amount": "10"}), "body.label", api.MsgFeeLabelTaken)
	expectFieldError(t, post(map[string]string{"label": "   ", "amount": "10"}), "body.label", api.MsgFeeLabelEmpty)
	if res := post(map[string]string{"label": strings.Repeat("a", 81), "amount": "10"}); res.Status != http.StatusUnprocessableEntity || !strings.Contains(res.Body, "body.label") {
		t.Fatalf("libellé trop long : %d %s", res.Status, res.Body)
	}
	expectFieldError(t, post(map[string]string{"label": "Négatif", "amount": "-5"}), "body.amount", api.MsgAmountFormat)
	expectFieldError(t, post(map[string]string{"label": "Décimales", "amount": "12,345"}), "body.amount", api.MsgAmountFormat)
	expectFieldError(t, post(map[string]string{"label": "Cher", "amount": "20000"}), "body.amount", api.MsgAmountTooHigh)
	if res := post(map[string]string{"label": "Long", "amount": "1", "description": strings.Repeat("d", 501)}); res.Status != http.StatusUnprocessableEntity || !strings.Contains(res.Body, "body.description") {
		t.Fatalf("description trop longue : %d %s", res.Status, res.Body)
	}
	both := post(map[string]string{"label": "", "amount": "abc"})
	expectFieldError(t, both, "body.label", api.MsgFeeLabelEmpty)
	expectFieldError(t, both, "body.amount", api.MsgAmountFormat)

	expectProblem(t, admin.Do("GET", feesPath(9999), nil), http.StatusNotFound, api.MsgSeasonGone)
	expectProblem(t, admin.Do("POST", feesPath(9999), map[string]string{"label": "X", "amount": "1"}), http.StatusNotFound, api.MsgSeasonGone)

	if n := feeLogs(t, env, "fee.create", "Plein tarif", "Tarif réduit"); n != 3 {
		t.Fatalf("%d ligne(s) fee.create, attendu 3", n)
	}
	if !strings.Contains(env.Logs.String(), `"amount_cents":1550`) {
		t.Error("montant absent du journal")
	}
}

func TestMembershipFeesLimit(t *testing.T) {
	_, admin := seasonEnv(t)
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	for i := range 20 {
		createFee(t, admin, s.ID, map[string]string{"label": fmt.Sprintf("Tarif %d", i+1), "amount": "10"})
	}
	if l := listFees(t, admin, s.ID); l.Actions.Add || !l.Actions.Edit {
		t.Fatalf("actions à 20 tarifs : %+v", l.Actions)
	}
	expectProblem(t, admin.Do("POST", feesPath(s.ID), map[string]string{"label": "Tarif 21", "amount": "10"}), http.StatusConflict, api.MsgTooManyFees)
}

func TestMembershipFeesConcurrentCreate(t *testing.T) {
	_, admin := seasonEnv(t)
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	var wg sync.WaitGroup
	statuses := make([]int, 2)
	for i := range statuses {
		wg.Go(func() {
			statuses[i] = admin.Do("POST", feesPath(s.ID), map[string]string{"label": "Plein tarif", "amount": "30"}).Status
		})
	}
	wg.Wait()
	slices.Sort(statuses)
	if statuses[0] != http.StatusCreated || statuses[1] != http.StatusUnprocessableEntity {
		t.Fatalf("statuts %v, attendu une 201 et une 422", statuses)
	}
	if n := len(listFees(t, admin, s.ID).Items); n != 1 {
		t.Fatalf("%d tarifs, attendu 1", n)
	}
}

func TestUpdateAndDeleteMembershipFees(t *testing.T) {
	env, admin := seasonEnv(t)
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	full := createFee(t, admin, s.ID, map[string]string{"label": "Plein tarif", "amount": "30"})
	reduced := createFee(t, admin, s.ID, map[string]string{"label": "Tarif réduit", "amount": "15"})

	put := func(id int64, body map[string]string) apitest.Response { return admin.Do("PUT", feePath(s.ID, id), body) }
	got := expectFee(t, put(full.ID, map[string]string{"label": "Plein tarif", "amount": "32", "description": "Adultes"}), http.StatusOK)
	if got.AmountCents != 3200 || got.Description == nil || *got.Description != "Adultes" || got.UpdatedAt.Before(got.CreatedAt) {
		t.Fatalf("modification : %+v", got)
	}
	if got := expectFee(t, put(full.ID, map[string]string{"label": "Plein tarif", "amount": "32"}), http.StatusOK); got.Description != nil {
		t.Fatalf("description retirée : %+v", got)
	}
	expectFieldError(t, put(full.ID, map[string]string{"label": "tarif REDUIT", "amount": "32"}), "body.label", api.MsgFeeLabelTaken)
	expectFee(t, put(full.ID, map[string]string{"label": "PLEIN TARIF", "amount": "32"}), http.StatusOK)

	// Saison ouverte : tout reste modifiable (FR-008).
	expectSeason(t, admin.Do("POST", seasonPath(s.ID, "/open"), nil), http.StatusOK)
	expectFee(t, put(reduced.ID, map[string]string{"label": "Réduit", "amount": "12,50"}), http.StatusOK)

	other := createSeason(t, admin, "2027-09-01", "2028-08-31")
	expectProblem(t, admin.Do("PUT", feePath(other.ID, full.ID), map[string]string{"label": "X", "amount": "1"}), http.StatusNotFound, api.MsgFeeGone)
	expectProblem(t, admin.Do("DELETE", feePath(other.ID, full.ID), nil), http.StatusNotFound, api.MsgFeeGone)

	if res := admin.Do("DELETE", feePath(s.ID, full.ID), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", res.Status, res.Body)
	}
	expectProblem(t, put(full.ID, map[string]string{"label": "X", "amount": "1"}), http.StatusNotFound, api.MsgFeeGone)
	expectProblem(t, admin.Do("DELETE", feePath(s.ID, full.ID), nil), http.StatusNotFound, api.MsgFeeGone)
	// Dernier tarif d'une saison ouverte.
	if res := admin.Do("DELETE", feePath(s.ID, reduced.ID), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression du dernier tarif : %d %s", res.Status, res.Body)
	}

	if n := feeLogs(t, env, "fee.update", "Plein tarif", "Réduit"); n != 4 {
		t.Fatalf("%d ligne(s) fee.update, attendu 4", n)
	}
	if n := feeLogs(t, env, "fee.delete"); n != 2 {
		t.Fatalf("%d ligne(s) fee.delete, attendu 2", n)
	}
}

func TestMembershipFeesClosedSeason(t *testing.T) {
	env, admin := seasonEnv(t)
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	fee := createFee(t, admin, s.ID, map[string]string{"label": "Plein tarif", "amount": "30"})

	env.SetToday("2027-09-01")
	l := listFees(t, admin, s.ID)
	if len(l.Items) != 1 || l.Actions != (api.MembershipFeeActions{}) {
		t.Fatalf("saison close : %+v", l)
	}
	body := map[string]string{"label": "Autre", "amount": "1"}
	expectProblem(t, admin.Do("POST", feesPath(s.ID), body), http.StatusConflict, api.MsgSeasonClosed)
	expectProblem(t, admin.Do("PUT", feePath(s.ID, fee.ID), body), http.StatusConflict, api.MsgSeasonClosed)
	expectProblem(t, admin.Do("DELETE", feePath(s.ID, fee.ID), nil), http.StatusConflict, api.MsgSeasonClosed)
}

func TestDeleteSeasonCascadesFees(t *testing.T) {
	env, admin := seasonEnv(t)
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	createFee(t, admin, s.ID, map[string]string{"label": "Plein tarif", "amount": "30"})
	createFee(t, admin, s.ID, map[string]string{"label": "Réduit", "amount": "15"})
	if res := admin.Do("DELETE", seasonPath(s.ID, ""), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression de la saison : %d %s", res.Status, res.Body)
	}
	var n int
	if err := env.Store.ReadDB.QueryRowContext(context.Background(), "SELECT count(*) FROM membership_fees WHERE season_id = ?", s.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("%d tarif(s) restant(s) après suppression de la saison", n)
	}
}

func TestOpenSeasonWithoutFees(t *testing.T) {
	_, admin := seasonEnv(t)
	first := createSeason(t, admin, "2026-09-01", "2027-08-31")
	expectSeason(t, admin.Do("POST", seasonPath(first.ID, "/open"), nil), http.StatusOK)
	next := createSeason(t, admin, "2027-09-01", "2028-08-31")
	if !next.Actions.Open {
		t.Fatalf("saison sans tarif : ouverture non proposée %+v", next.Actions)
	}
	expectSeason(t, admin.Do("POST", seasonPath(next.ID, "/open"), nil), http.StatusOK)
}

func TestMembershipFeesAccess(t *testing.T) {
	env, admin := seasonEnv(t)
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	fee := createFee(t, admin, s.ID, map[string]string{"label": "Plein tarif", "amount": "30"})
	member := env.LoggedIn(env.Seed(auth.RoleMember, true))
	anonymous := env.Client()
	body := map[string]string{"label": "Autre", "amount": "1"}
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{"GET", feesPath(s.ID), nil},
		{"POST", feesPath(s.ID), body},
		{"PUT", feePath(s.ID, fee.ID), body},
		{"DELETE", feePath(s.ID, fee.ID), nil},
	} {
		if res := member.Do(c.method, c.path, c.body); res.Status != http.StatusForbidden {
			t.Errorf("%s %s membre : %d", c.method, c.path, res.Status)
		}
		if res := anonymous.Do(c.method, c.path, c.body); res.Status != http.StatusUnauthorized {
			t.Errorf("%s %s anonyme : %d", c.method, c.path, res.Status)
		}
	}
	if n := len(listFees(t, admin, s.ID).Items); n != 1 {
		t.Fatalf("%d tarifs après les refus, attendu 1", n)
	}
}

func TestMembershipFeeRenewalAmount(t *testing.T) {
	_, admin := seasonEnv(t)
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")

	res := admin.Do("POST", feesPath(s.ID), map[string]any{"label": "Adulte compétition", "amount": "85", "hasRenewalAmount": true, "renewalAmount": "55"})
	fee := expectFee(t, res, http.StatusCreated)
	if fee.AmountCents != 8500 || fee.RenewalAmountCents == nil || *fee.RenewalAmountCents != 5500 {
		t.Fatalf("renouvellement : %+v", fee)
	}
	// Montant ignoré tant que la case n'est pas cochée.
	plain := expectFee(t, admin.Do("POST", feesPath(s.ID), map[string]any{"label": "Adulte loisir", "amount": "75", "renewalAmount": "45"}), http.StatusCreated)
	if plain.RenewalAmountCents != nil {
		t.Fatalf("sans case cochée : %+v", plain)
	}
	expectFieldError(t, admin.Do("POST", feesPath(s.ID), map[string]any{"label": "Jeune", "amount": "50", "hasRenewalAmount": true}), "body.renewalAmount", api.MsgRenewalRequired)
	expectFieldError(t, admin.Do("POST", feesPath(s.ID), map[string]any{"label": "Jeune", "amount": "50", "hasRenewalAmount": true, "renewalAmount": "12,345"}), "body.renewalAmount", api.MsgAmountFormat)
	expectFieldError(t, admin.Do("POST", feesPath(s.ID), map[string]any{"label": "Jeune", "amount": "50", "hasRenewalAmount": true, "renewalAmount": "20000"}), "body.renewalAmount", api.MsgAmountTooHigh)

	if got := listFees(t, admin, s.ID).Items[0]; got.RenewalAmountCents == nil || *got.RenewalAmountCents != 5500 {
		t.Fatalf("liste : %+v", got)
	}
	cleared := expectFee(t, admin.Do("PUT", feePath(s.ID, fee.ID), map[string]any{"label": "Adulte compétition", "amount": "85", "hasRenewalAmount": false, "renewalAmount": "55"}), http.StatusOK)
	if cleared.RenewalAmountCents != nil {
		t.Fatalf("case décochée : %+v", cleared)
	}
}
