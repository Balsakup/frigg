package api_test

import (
	"net/http"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/auth"
)

func TestCreateAndListSeasons(t *testing.T) {
	env, admin := seasonEnv(t)

	l := listSeasons(t, admin)
	if len(l.Items) != 0 || l.Today != "2026-10-31" || l.CurrentID != nil || l.NextStartsOn != nil || l.Setup != "missing" {
		t.Fatalf("liste vide : %+v", l)
	}

	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	if s.Label != "2026-2027" || s.Status != "draft" || !s.IsCurrent || !s.Actions.Open || !s.Actions.Delete || !s.Actions.EditStartsOn {
		t.Fatalf("première saison : %+v", s)
	}
	if s.EndsOnMin != "2026-10-31" || s.EndsOnMax != "2028-08-31" {
		t.Fatalf("bornes : %s %s", s.EndsOnMin, s.EndsOnMax)
	}
	if n := seasonLogs(t, env, "season.create"); n != 1 {
		t.Fatalf("%d ligne(s) de journal season.create", n)
	}

	l = listSeasons(t, admin)
	if l.NextStartsOn == nil || *l.NextStartsOn != "2027-09-01" || l.Setup != "current-draft" || l.CurrentID == nil || *l.CurrentID != s.ID {
		t.Fatalf("liste après création : %+v", l)
	}
	if *l.NextEndsOnMin != "2027-09-02" || *l.NextEndsOnMax != "2029-08-31" {
		t.Fatalf("bornes de la prochaine : %s %s", *l.NextEndsOnMin, *l.NextEndsOnMax)
	}

	next := createSeason(t, admin, "2027-09-01", "2028-08-31")
	l = listSeasons(t, admin)
	if len(l.Items) != 2 || l.Items[0].ID != next.ID || l.Items[1].ID != s.ID {
		t.Fatalf("ordre de la liste : %+v", l.Items)
	}

	expectProblem(t, admin.Do("POST", "/api/admin/seasons", map[string]string{"startsOn": "2028-08-01", "endsOn": "2029-08-31"}),
		http.StatusConflict, api.MsgSeasonsChanged)

	got := expectSeason(t, admin.Do("GET", seasonPath(s.ID, ""), nil), http.StatusOK)
	if got.ID != s.ID || got.Label != "2026-2027" {
		t.Fatalf("fiche : %+v", got)
	}
	expectProblem(t, admin.Do("GET", seasonPath(999, ""), nil), http.StatusNotFound, api.MsgSeasonGone)
}

func TestCreateSeasonValidation(t *testing.T) {
	_, admin := seasonEnv(t)
	post := func(start, end string) any {
		return map[string]string{"startsOn": start, "endsOn": end}
	}
	expectFieldError(t, admin.Do("POST", "/api/admin/seasons", post("2026-09-01", "2026-10-30")), "body.endsOn", api.MsgEndInPast)
	expectFieldError(t, admin.Do("POST", "/api/admin/seasons", post("2026-09-01", "2026-09-01")), "body.endsOn", api.MsgEndBeforeStart)
	expectFieldError(t, admin.Do("POST", "/api/admin/seasons", post("2026-09-01", "2028-09-01")), "body.endsOn", api.MsgSeasonTooLong)
	expectFieldError(t, admin.Do("POST", "/api/admin/seasons", post("2026-02-30", "2027-08-31")), "body.startsOn", api.MsgInvalidDate)
	if l := listSeasons(t, admin); len(l.Items) != 0 {
		t.Fatalf("saison créée malgré les erreurs : %+v", l.Items)
	}
}

func TestSeasonHomonymLabels(t *testing.T) {
	_, admin := seasonEnv(t)
	first := createSeason(t, admin, "2027-01-01", "2027-06-30")
	if first.Label != "2027" {
		t.Fatalf("libellé seul : %q", first.Label)
	}
	createSeason(t, admin, "2027-07-01", "2027-12-31")
	l := listSeasons(t, admin)
	if l.Items[0].Label != "2027 (juil.–déc.)" || l.Items[1].Label != "2027 (janv.–juin)" {
		t.Fatalf("libellés : %q %q", l.Items[0].Label, l.Items[1].Label)
	}
}

func TestSeasonsAccess(t *testing.T) {
	env, _ := seasonEnv(t)
	member := env.LoggedIn(env.Seed(auth.RoleMember, true))
	anonymous := env.Client()
	for _, c := range []struct{ method, path string }{
		{"GET", "/api/admin/seasons"},
		{"GET", "/api/admin/seasons/1"},
		{"POST", "/api/admin/seasons"},
		{"PATCH", "/api/admin/seasons/1"},
		{"POST", "/api/admin/seasons/1/open"},
		{"POST", "/api/admin/seasons/1/duplicate"},
		{"DELETE", "/api/admin/seasons/1"},
	} {
		var body any
		if c.method == "POST" || c.method == "PATCH" {
			body = map[string]string{"startsOn": "2026-09-01", "endsOn": "2027-08-31"}
		}
		if res := member.Do(c.method, c.path, body); res.Status != http.StatusForbidden {
			t.Errorf("%s %s membre : %d", c.method, c.path, res.Status)
		}
		if res := anonymous.Do(c.method, c.path, body); res.Status != http.StatusUnauthorized {
			t.Errorf("%s %s anonyme : %d", c.method, c.path, res.Status)
		}
	}
}
