package api_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/balsakup/frigg/internal/api"
)

func TestOpenAndDuplicateSeasons(t *testing.T) {
	env, admin := seasonEnv(t)
	first := createSeason(t, admin, "2026-09-01", "2027-08-31")

	opened := expectSeason(t, admin.Do("POST", seasonPath(first.ID, "/open"), nil), http.StatusOK)
	if opened.Status != "open" || opened.OpenedAt == nil || opened.Actions.Delete || opened.Actions.Open {
		t.Fatalf("ouverture : %+v", opened)
	}
	if l := listSeasons(t, admin); l.Setup != "ready" {
		t.Fatalf("setup après ouverture : %s", l.Setup)
	}
	expectSeason(t, admin.Do("POST", seasonPath(first.ID, "/open"), nil), http.StatusOK)
	if n := seasonLogs(t, env, "season.open"); n != 1 {
		t.Fatalf("%d ligne(s) season.open, attendu 1 (réouverture sans effet)", n)
	}

	second := expectSeason(t, admin.Do("POST", seasonPath(first.ID, "/duplicate"), nil), http.StatusCreated)
	if second.Label != "2027-2028" || second.StartsOn != "2027-09-01" || second.EndsOn != "2028-08-31" || second.Status != "draft" {
		t.Fatalf("duplication : %+v", second)
	}
	expectProblem(t, admin.Do("POST", seasonPath(first.ID, "/duplicate"), nil), http.StatusConflict, api.MsgSeasonsChanged)

	third := expectSeason(t, admin.Do("POST", seasonPath(second.ID, "/duplicate"), nil), http.StatusCreated)
	if third.Actions.Open || third.Actions.OpenBlockedReason == nil || *third.Actions.OpenBlockedReason != "Ouvrez d'abord la saison 2027-2028." {
		t.Fatalf("ouverture bloquée : %+v", third.Actions)
	}
	expectProblem(t, admin.Do("POST", seasonPath(third.ID, "/open"), nil), http.StatusConflict, fmt.Sprintf(api.MsgPreviousDraft, "2027-2028"))

	expectSeason(t, admin.Do("POST", seasonPath(second.ID, "/open"), nil), http.StatusOK)
	l := listSeasons(t, admin)
	open := 0
	for _, it := range l.Items {
		if it.Status == "open" {
			open++
		}
	}
	if open != 2 || l.CurrentID == nil || *l.CurrentID != first.ID {
		t.Fatalf("pré-inscriptions : %d ouvertes, courante %v", open, l.CurrentID)
	}
	if n := seasonLogs(t, env, "season.duplicate"); n != 2 {
		t.Fatalf("%d ligne(s) season.duplicate", n)
	}

	// La troisième, restée en brouillon, est close une fois sa fin passée.
	env.SetToday("2029-09-01")
	expectProblem(t, admin.Do("POST", seasonPath(third.ID, "/open"), nil), http.StatusConflict, api.MsgSeasonClosedOpen)
}

func TestConcurrentDuplicate(t *testing.T) {
	_, admin := seasonEnv(t)
	first := createSeason(t, admin, "2026-09-01", "2027-08-31")

	var wg sync.WaitGroup
	statuses := make([]int, 2)
	for i := range statuses {
		wg.Go(func() {
			statuses[i] = admin.Do("POST", seasonPath(first.ID, "/duplicate"), nil).Status
		})
	}
	wg.Wait()
	created, conflicts := 0, 0
	for _, s := range statuses {
		switch s {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicts++
		}
	}
	if created != 1 || conflicts != 1 {
		t.Fatalf("statuts %v, attendu une création et un conflit", statuses)
	}
	if l := listSeasons(t, admin); len(l.Items) != 2 {
		t.Fatalf("%d saisons, attendu 2", len(l.Items))
	}
}
