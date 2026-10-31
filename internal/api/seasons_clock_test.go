package api_test

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
)

func TestSeasonClosureFollowsClock(t *testing.T) {
	env, admin := seasonEnv(t)
	env.SetToday("2026-08-01")
	first := createSeason(t, admin, "2025-09-01", "2026-08-31")
	expectSeason(t, admin.Do("POST", seasonPath(first.ID, "/open"), nil), http.StatusOK)
	second := expectSeason(t, admin.Do("POST", seasonPath(first.ID, "/duplicate"), nil), http.StatusCreated)

	env.SetToday("2026-08-31")
	s := expectSeason(t, admin.Do("GET", seasonPath(first.ID, ""), nil), http.StatusOK)
	if s.Status != "open" || !s.IsCurrent {
		t.Fatalf("jour de fin : %+v", s)
	}

	// Le brouillon dont la fin passe est clos, sans aucune écriture.
	env.SetToday("2026-09-01")
	byID := func(l api.SeasonList, id int64) api.Season {
		for _, it := range l.Items {
			if it.ID == id {
				return it
			}
		}
		t.Fatalf("saison %d absente", id)
		return api.Season{}
	}
	l := listSeasons(t, admin)
	closed := byID(l, first.ID)
	if closed.Status != "closed" || closed.IsCurrent || closed.Actions != (api.SeasonActions{}) {
		t.Fatalf("lendemain de la fin : %+v", closed)
	}
	if l.CurrentID == nil || *l.CurrentID != second.ID || l.Setup != "current-draft" {
		t.Fatalf("saison courante : %v %s", l.CurrentID, l.Setup)
	}

	expectSeason(t, admin.Do("POST", seasonPath(second.ID, "/open"), nil), http.StatusOK)
	if l := listSeasons(t, admin); l.Setup != "ready" {
		t.Fatalf("setup : %s", l.Setup)
	}

	env.SetToday("2027-09-01")
	l = listSeasons(t, admin)
	last := byID(l, second.ID)
	if l.CurrentID != nil || l.Setup != "no-current" || last.Status != "closed" || !last.Actions.Duplicate || last.Actions.Delete {
		t.Fatalf("après la dernière : %+v %+v", l, last)
	}

	// Un brouillon jamais ouvert est clos lui aussi.
	draft := expectSeason(t, admin.Do("POST", seasonPath(second.ID, "/duplicate"), nil), http.StatusCreated)
	env.SetToday("2028-09-01")
	d := expectSeason(t, admin.Do("GET", seasonPath(draft.ID, ""), nil), http.StatusOK)
	if d.Status != "closed" || !d.Actions.Delete {
		t.Fatalf("brouillon passé : %+v", d)
	}
}

// TestSeasonsUseInjectedClock vérifie que la date du jour des saisons vient
// toujours de Deps.Now (research.md 003 › R3).
func TestSeasonsUseInjectedClock(t *testing.T) {
	for _, path := range []string{"seasons.go", "../season/season.go", "../database/queries/seasons.sql"} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		src := string(b)
		for _, forbidden := range []string{"time.Now()", "date('now'", "'localtime'"} {
			if strings.Contains(src, forbidden) {
				t.Errorf("%s utilise %s : passer par l'horloge injectée", path, forbidden)
			}
		}
	}
}

// TestFillGapAfterLongInterruption : après plus de 24 mois sans saison, les
// saisons manquantes sont recréées une à une, entièrement passées, jusqu'à
// rejoindre la date du jour.
func TestFillGapAfterLongInterruption(t *testing.T) {
	env, admin := seasonEnv(t)
	env.SetToday("2022-10-01")
	old := createSeason(t, admin, "2022-09-01", "2023-08-31")
	expectSeason(t, admin.Do("POST", seasonPath(old.ID, "/open"), nil), http.StatusOK)

	env.SetToday("2026-10-31")
	l := listSeasons(t, admin)
	if l.Setup != "no-current" || *l.NextStartsOn != "2023-09-01" || *l.NextEndsOnMin != "2023-09-02" {
		t.Fatalf("liste après l'interruption : %+v", l)
	}

	// Création manuelle d'une saison passée : close dès sa création.
	gap := createSeason(t, admin, "2023-09-01", "2024-08-31")
	if gap.Status != "closed" || !gap.Actions.Delete || !gap.Actions.Duplicate {
		t.Fatalf("saison comblant le trou : %+v", gap)
	}
	// Comblement raté (mauvaise fin) : la saison close jamais ouverte se supprime.
	if res := admin.Do("DELETE", seasonPath(gap.ID, ""), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression d'une saison de comblement : %d %s", res.Status, res.Body)
	}
	gap = createSeason(t, admin, "2023-09-01", "2024-08-31")
	// Duplication d'une saison passée : copie passée elle aussi.
	dup := expectSeason(t, admin.Do("POST", seasonPath(gap.ID, "/duplicate"), nil), http.StatusCreated)
	if dup.StartsOn != "2024-09-01" || dup.EndsOn != "2025-08-31" || dup.Status != "closed" {
		t.Fatalf("duplication passée : %+v", dup)
	}
	dup = expectSeason(t, admin.Do("POST", seasonPath(dup.ID, "/duplicate"), nil), http.StatusCreated)
	current := expectSeason(t, admin.Do("POST", seasonPath(dup.ID, "/duplicate"), nil), http.StatusCreated)
	if current.Label != "2026-2027" || !current.IsCurrent || current.Status != "draft" {
		t.Fatalf("saison rejoignant aujourd'hui : %+v", current)
	}
	if l := listSeasons(t, admin); l.Setup != "current-draft" || len(l.Items) != 5 {
		t.Fatalf("liste comblée : setup %s, %d saisons", l.Setup, len(l.Items))
	}
}
