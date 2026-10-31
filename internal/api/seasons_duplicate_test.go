package api_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
)

func TestDuplicateSeasonCopiesFees(t *testing.T) {
	env, admin := seasonEnv(t)
	src := createSeason(t, admin, "2026-09-01", "2027-08-31")
	createFee(t, admin, src.ID, map[string]string{"label": "Plein tarif", "amount": "30"})
	createFee(t, admin, src.ID, map[string]string{"label": "Tarif réduit", "amount": "15,50", "description": "Étudiants"})
	expectFee(t, admin.Do("POST", feesPath(src.ID), map[string]any{"label": "Adulte compétition", "amount": "85", "hasRenewalAmount": true, "renewalAmount": "55"}), http.StatusCreated)

	dup := expectSeason(t, admin.Do("POST", seasonPath(src.ID, "/duplicate"), nil), http.StatusCreated)
	before := listFees(t, admin, src.ID)
	copied := listFees(t, admin, dup.ID)
	if len(copied.Items) != 3 {
		t.Fatalf("%d tarifs recopiés, attendu 3", len(copied.Items))
	}
	for i, f := range copied.Items {
		o := before.Items[i]
		if f.ID == o.ID || f.Label != o.Label || f.AmountCents != o.AmountCents || !equalPtr(f.Description, o.Description) || !equalPtr(f.RenewalAmountCents, o.RenewalAmountCents) {
			t.Errorf("copie %d : %+v, source %+v", i, f, o)
		}
	}
	if !strings.Contains(env.Logs.String(), `"fees_copied":3`) {
		t.Error("fees_copied absent du journal de duplication")
	}

	// Copies indépendantes.
	createFee(t, admin, dup.ID, map[string]string{"label": "Famille", "amount": "50"})
	if got := feeLabels(listFees(t, admin, src.ID)); !slices.Equal(got, feeLabels(before)) {
		t.Fatalf("source modifiée : %v", got)
	}

	// Saison sans tarif : copie sans tarif ; création manuelle : aucun tarif.
	if res := admin.Do("DELETE", seasonPath(dup.ID, ""), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression de la copie : %d %s", res.Status, res.Body)
	}
	bare := createSeason(t, admin, "2027-09-01", "2028-08-31")
	if n := len(listFees(t, admin, bare.ID).Items); n != 0 {
		t.Fatalf("création manuelle : %d tarifs", n)
	}
	dup2 := expectSeason(t, admin.Do("POST", seasonPath(bare.ID, "/duplicate"), nil), http.StatusCreated)
	if n := len(listFees(t, admin, dup2.ID).Items); n != 0 {
		t.Fatalf("copie d'une saison sans tarif : %d tarifs", n)
	}
	if !strings.Contains(env.Logs.String(), `"fees_copied":0`) {
		t.Error("fees_copied à 0 absent du journal")
	}
}

func TestDuplicateClosedSeasonCopiesFeesReadOnly(t *testing.T) {
	env, admin := seasonEnv(t)
	env.SetToday("2023-10-01")
	src := createSeason(t, admin, "2023-09-01", "2024-08-31")
	createFee(t, admin, src.ID, map[string]string{"label": "Plein tarif", "amount": "25"})

	env.SetToday("2026-10-31")
	dup := expectSeason(t, admin.Do("POST", seasonPath(src.ID, "/duplicate"), nil), http.StatusCreated)
	if dup.Status != "closed" {
		t.Fatalf("copie passée : état %s", dup.Status)
	}
	l := listFees(t, admin, dup.ID)
	if len(l.Items) != 1 || l.Actions != (api.MembershipFeeActions{}) {
		t.Fatalf("copie close : %+v", l)
	}
}

func equalPtr[T comparable](a, b *T) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}
