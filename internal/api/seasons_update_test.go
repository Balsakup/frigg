package api_test

import (
	"net/http"
	"testing"

	"github.com/balsakup/frigg/internal/api"
)

func TestUpdateSeason(t *testing.T) {
	env, admin := seasonEnv(t)
	first := createSeason(t, admin, "2026-09-01", "2027-08-31")

	s := expectSeason(t, admin.Do("PATCH", seasonPath(first.ID, ""), map[string]string{"startsOn": "2026-08-15", "endsOn": "2027-07-31"}), http.StatusOK)
	if s.StartsOn != "2026-08-15" || s.EndsOn != "2027-07-31" || s.Label != "2026-2027" {
		t.Fatalf("brouillon modifié : %+v", s)
	}
	expectSeason(t, admin.Do("PATCH", seasonPath(first.ID, ""), map[string]string{"startsOn": "2026-09-01", "endsOn": "2027-08-31"}), http.StatusOK)
	expectSeason(t, admin.Do("POST", seasonPath(first.ID, "/open"), nil), http.StatusOK)

	expectProblem(t, admin.Do("PATCH", seasonPath(first.ID, ""), map[string]string{"startsOn": "2026-08-15", "endsOn": "2027-08-31"}),
		http.StatusConflict, api.MsgSeasonStartLocked)
	expectFieldError(t, admin.Do("PATCH", seasonPath(first.ID, ""), map[string]string{"endsOn": "2026-10-30"}), "body.endsOn", api.MsgEndInPast)

	next := expectSeason(t, admin.Do("POST", seasonPath(first.ID, "/duplicate"), nil), http.StatusCreated)
	got := expectSeason(t, admin.Do("GET", seasonPath(first.ID, ""), nil), http.StatusOK)
	if got.ShiftsSeasonLabel == nil || *got.ShiftsSeasonLabel != "2027-2028" {
		t.Fatalf("shiftsSeasonLabel : %v", got.ShiftsSeasonLabel)
	}
	expectSeason(t, admin.Do("PATCH", seasonPath(first.ID, ""), map[string]string{"endsOn": "2027-08-15"}), http.StatusOK)
	shifted := expectSeason(t, admin.Do("GET", seasonPath(next.ID, ""), nil), http.StatusOK)
	if shifted.StartsOn != "2027-08-16" || shifted.EndsOn != "2028-08-31" {
		t.Fatalf("saison suivante non décalée : %+v", shifted)
	}
	expectFieldError(t, admin.Do("PATCH", seasonPath(first.ID, ""), map[string]string{"endsOn": "2028-08-30"}),
		"body.endsOn", "Cette date rendrait la saison 2027-2028 invalide.")
	if n := seasonLogs(t, env, "season.update"); n != 3 {
		t.Fatalf("%d ligne(s) season.update", n)
	}

	expectSeason(t, admin.Do("POST", seasonPath(next.ID, "/open"), nil), http.StatusOK)
	got = expectSeason(t, admin.Do("GET", seasonPath(first.ID, ""), nil), http.StatusOK)
	if got.ShiftsSeasonLabel != nil || got.Actions.EditEndsOn {
		t.Fatalf("suivante ouverte : %+v", got)
	}
	expectProblem(t, admin.Do("PATCH", seasonPath(first.ID, ""), map[string]string{"endsOn": "2027-08-20"}),
		http.StatusConflict, api.MsgNextSeasonOpen)

	env.SetToday("2028-09-01")
	expectProblem(t, admin.Do("PATCH", seasonPath(first.ID, ""), map[string]string{"endsOn": "2028-12-31"}),
		http.StatusConflict, api.MsgSeasonClosed)
}

func TestDeleteSeason(t *testing.T) {
	env, admin := seasonEnv(t)
	first := createSeason(t, admin, "2026-09-01", "2027-08-31")
	second := createSeason(t, admin, "2027-09-01", "2028-08-31")
	third := createSeason(t, admin, "2028-09-01", "2029-08-31")

	expectProblem(t, admin.Do("DELETE", seasonPath(second.ID, ""), nil), http.StatusConflict, api.MsgDeleteLastOnly)
	if res := admin.Do("DELETE", seasonPath(third.ID, ""), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", res.Status, res.Body)
	}
	expectProblem(t, admin.Do("GET", seasonPath(third.ID, ""), nil), http.StatusNotFound, api.MsgSeasonGone)
	expectProblem(t, admin.Do("DELETE", seasonPath(third.ID, ""), nil), http.StatusNotFound, api.MsgSeasonGone)

	expectSeason(t, admin.Do("POST", seasonPath(first.ID, "/open"), nil), http.StatusOK)
	expectSeason(t, admin.Do("POST", seasonPath(second.ID, "/open"), nil), http.StatusOK)
	expectProblem(t, admin.Do("DELETE", seasonPath(second.ID, ""), nil), http.StatusConflict, api.MsgDeleteOpened)
	if n := seasonLogs(t, env, "season.delete"); n != 1 {
		t.Fatalf("%d ligne(s) season.delete", n)
	}
}
