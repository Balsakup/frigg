package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
)

func TestMyEventsListing(t *testing.T) {
	f := newEvFixture(t)
	f.env.SetToday("2026-10-01")
	f.publishedEvent(t, eventBody("Passé", func(b map[string]any) {
		b["startsAt"], b["endsAt"] = "2026-10-10T10:00", "2026-10-10T12:00"
	}))
	f.env.SetToday(msToday)
	later := f.publishedEvent(t, eventBody("Plus tard", func(b map[string]any) {
		b["startsAt"], b["endsAt"], b["capacity"] = "2027-01-20T10:00", "2027-01-20T12:00", 12
	}))
	soon := f.publishedEvent(t, eventBody("Bientôt"))
	draft := f.createEvent(t, eventBody("Brouillon secret"))
	cancelled := f.publishedEvent(t, eventBody("Annulé", func(b map[string]any) { b["startsAt"] = "2026-12-01T10:00" }))
	expectAdminEvent(t, f.admin.Do("POST", eventPath(cancelled.ID, "/cancel"), map[string]any{}), http.StatusOK)

	res := f.claire.Do("GET", "/api/me/events", nil)
	out := expectJSON[struct {
		Items []api.EventSummary `json:"items"`
	}](t, res, http.StatusOK)
	var titles []string
	for _, e := range out.Items {
		titles = append(titles, e.Title)
	}
	if strings.Join(titles, ",") != "Annulé,Bientôt,Plus tard" {
		t.Fatalf("liste : %v", titles)
	}
	if strings.Contains(string(res.Body), "Brouillon secret") {
		t.Fatal("brouillon transmis à un membre")
	}
	if out.Items[2].PlacesLeft == nil || *out.Items[2].PlacesLeft != 12 || !out.Items[1].RegistrationOpen || out.Items[0].RegistrationOpen {
		t.Fatalf("places et période : %+v", out.Items)
	}

	expectProblem(t, f.claire.Do("GET", myEventPath(draft.ID, ""), nil), http.StatusNotFound, api.MsgEventGone)
	expectProblem(t, f.claire.Do("GET", myEventPath(9999, ""), nil), http.StatusNotFound, api.MsgEventGone)
	_ = later
	_ = soon
}

func TestMyEventEligibility(t *testing.T) {
	f := newEvFixture(t)
	other, _, stranger := f.member(t, "Paul")
	_ = other
	ev := f.publishedEvent(t, eventBody("AG"))
	got := expectJSON[api.MyEvent](t, f.claire.Do("GET", myEventPath(ev.ID, ""), nil), http.StatusOK)
	if len(got.Members) != 2 || got.Members[0].Member.FirstName != "Claire" || got.Members[1].Member.FirstName != "Lucas" {
		t.Fatalf("fiches : %+v", got.Members)
	}
	if !got.Members[0].Eligible || got.Members[1].Eligible || *got.Members[1].Reason != "not_season_member" {
		t.Fatalf("éligibilité : %+v", got.Members)
	}
	for _, m := range got.Members {
		if m.Member.ID == stranger.ID {
			t.Fatal("fiche d'un autre compte transmise")
		}
	}

	register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID)
	got = expectJSON[api.MyEvent](t, f.claire.Do("GET", myEventPath(ev.ID, ""), nil), http.StatusOK)
	if got.Members[0].Registration == nil || got.Members[0].Eligible || *got.Members[0].Reason != "already_registered" {
		t.Fatalf("inscrite : %+v", got.Members[0])
	}

	open := f.publishedEvent(t, eventBody("Portes ouvertes", func(b map[string]any) { b["openness"] = "all_members" }))
	got = expectJSON[api.MyEvent](t, f.claire.Do("GET", myEventPath(open.ID, ""), nil), http.StatusOK)
	if !got.Members[0].Eligible || !got.Members[1].Eligible {
		t.Fatalf("toutes les fiches : %+v", got.Members)
	}
}
