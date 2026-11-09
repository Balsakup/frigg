package api_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
)

func TestCreateAndPublishEvent(t *testing.T) {
	f := newEvFixture(t)

	res := f.admin.Do("GET", "/api/admin/events", nil)
	if res.Status != http.StatusOK || !strings.Contains(string(res.Body), `"items":[]`) {
		t.Fatalf("liste vide : %d %s", res.Status, res.Body)
	}

	ev := f.createEvent(t, eventBody("Assemblée générale", func(b map[string]any) { b["price"] = "15,50" }))
	if ev.Status != "draft" || ev.Phase != "draft" || ev.Season.ID != f.season.ID || ev.PriceCents != 1550 {
		t.Fatalf("brouillon : %+v", ev)
	}
	if ev.StartsAt.UTC().Format(time.RFC3339) != "2026-12-12T13:00:00Z" || ev.Local.StartsAt != "2026-12-12T14:00" {
		t.Fatalf("dates : %v %+v", ev.StartsAt, ev.Local)
	}
	if !ev.Actions.Publish || ev.Actions.Cancel || !ev.Actions.Delete || ev.Window != nil {
		t.Fatalf("actions d'un brouillon : %+v", ev.Actions)
	}
	free := f.createEvent(t, eventBody("Gratuit", func(b map[string]any) { b["price"] = "" }))
	if free.PriceCents != 0 {
		t.Fatalf("prix vide : %d", free.PriceCents)
	}

	pub := f.publish(t, ev.ID)
	if pub.Status != "published" || pub.Phase != "upcoming" || pub.PublishedAt == nil || pub.Window == nil {
		t.Fatalf("publié : %+v", pub)
	}
	expectProblem(t, f.admin.Do("POST", eventPath(ev.ID, "/publish"), nil), http.StatusConflict, api.MsgEventChanged)
	expectProblem(t, f.admin.Do("GET", eventPath(999, ""), nil), http.StatusNotFound, api.MsgEventGone)

	if n := strings.Count(f.env.Logs.String(), `"action":"event.`); n != 3 {
		t.Errorf("journaux event.* : %d", n)
	}
	noLeak(t, f.env, "Assemblée générale", "Salle polyvalente")
}

func TestEventValidation(t *testing.T) {
	f := newEvFixture(t)
	cases := []struct {
		name     string
		edit     func(map[string]any)
		location string
	}{
		{"fin avant début", func(b map[string]any) { b["endsAt"] = "2026-12-12T13:00" }, "body.endsAt"},
		{"hors saison", func(b map[string]any) { b["startsAt"], b["endsAt"] = "2027-09-05T10:00", "2027-09-05T12:00" }, "body.startsAt"},
		{"heure inexistante", func(b map[string]any) { b["startsAt"], b["endsAt"] = "2027-03-28T02:30", "2027-03-28T05:00" }, "body.startsAt"},
		{"titre vide", func(b map[string]any) { b["title"] = "  " }, "body.title"},
		{"prix invalide", func(b map[string]any) { b["price"] = "abc" }, "body.price"},
		{"capacité", func(b map[string]any) { b["capacity"] = 0 }, "body.capacity"},
		{"clôture après la fin", func(b map[string]any) { b["registrationClosesAt"] = "2026-12-13T10:00" }, "body.registrationClosesAt"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := f.admin.Do("POST", "/api/admin/events", eventBody("X", c.edit))
			if res.Status != http.StatusUnprocessableEntity || !strings.Contains(string(res.Body), c.location) {
				t.Fatalf("%d %s", res.Status, res.Body)
			}
		})
	}
	res := f.admin.Do("POST", "/api/admin/events", eventBody(strings.Repeat("é", 121)))
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("titre 121 : %d", res.Status)
	}
}

func TestUpdateEventNotifiesOncePerAddress(t *testing.T) {
	f := newEvFixture(t)
	ev := f.publishedEvent(t, eventBody("Stage", func(b map[string]any) {
		b["openness"], b["capacity"] = "all_members", 10
	}))
	register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID, f.lucas.ID)
	f.env.FlushMail()
	before := mails(f.env, f.claireEmail, "Changement")

	// Titre et description : aucun e-mail.
	upd := expectAdminEvent(t, f.admin.Do("PUT", eventPath(ev.ID, ""), eventBody("Stage d'hiver", func(b map[string]any) {
		b["openness"], b["capacity"], b["description"] = "all_members", 10, "Programme"
	})), http.StatusOK)
	if upd.Title != "Stage d'hiver" || mails(f.env, f.claireEmail, "Changement") != before {
		t.Fatalf("modification sans e-mail : %+v", upd)
	}

	// Lieu modifié : un seul e-mail pour Claire, qui nomme ses deux fiches.
	expectAdminEvent(t, f.admin.Do("PUT", eventPath(ev.ID, ""), eventBody("Stage d'hiver", func(b map[string]any) {
		b["openness"], b["capacity"], b["location"] = "all_members", 10, "Gymnase"
	})), http.StatusOK)
	if got := mails(f.env, f.claireEmail, "Changement"); got != before+1 {
		t.Fatalf("e-mails de changement : %d", got-before)
	}
	text := lastMail(t, f.env, f.claireEmail)
	if !strings.Contains(text, "Lucas Martin") || !strings.Contains(text, "Claire Martin") || !strings.Contains(text, "auparavant : Salle polyvalente") {
		t.Fatalf("e-mail : %s", text)
	}

	// Capacité sous le nombre d'inscrits.
	res := f.admin.Do("PUT", eventPath(ev.ID, ""), eventBody("Stage d'hiver", func(b map[string]any) {
		b["openness"], b["capacity"] = "all_members", 1
	}))
	if res.Status != http.StatusUnprocessableEntity || !strings.Contains(string(res.Body), "body.capacity") {
		t.Fatalf("capacité : %d %s", res.Status, res.Body)
	}

	// Nouveau prix : les inscriptions existantes restent gratuites.
	expectAdminEvent(t, f.admin.Do("PUT", eventPath(ev.ID, ""), eventBody("Stage d'hiver", func(b map[string]any) {
		b["openness"], b["capacity"], b["price"] = "all_members", 10, "20"
	})), http.StatusOK)
	got := expectJSON[api.MyEvent](t, f.claire.Do("GET", myEventPath(ev.ID, ""), nil), http.StatusOK)
	for _, m := range got.Members {
		if m.Registration == nil || m.Registration.Payment != nil {
			t.Fatalf("inscription existante devenue payante : %+v", m)
		}
	}

	// Événement terminé : les dates ne changent plus, le titre si.
	f.env.SetToday("2026-12-13")
	expectProblem(t, f.admin.Do("PUT", eventPath(ev.ID, ""), eventBody("Stage d'hiver", func(b map[string]any) {
		b["openness"], b["capacity"], b["price"], b["startsAt"] = "all_members", 10, "20", "2026-12-12T13:00"
	})), http.StatusConflict, api.MsgEventFinished)
	expectAdminEvent(t, f.admin.Do("PUT", eventPath(ev.ID, ""), eventBody("Compte rendu", func(b map[string]any) {
		b["openness"], b["capacity"], b["price"], b["location"] = "all_members", 10, "20", "Gymnase"
	})), http.StatusOK)
	noLeak(t, f.env, "Gymnase", "Stage d'hiver", "Lucas")
}

func TestListEventsFilters(t *testing.T) {
	f := newEvFixture(t)
	f.env.SetToday("2026-10-01")
	past := f.publishedEvent(t, eventBody("Passé", func(b map[string]any) {
		b["startsAt"], b["endsAt"] = "2026-10-10T10:00", "2026-10-10T12:00"
	}))
	f.env.SetToday(msToday)
	draft := f.createEvent(t, eventBody("Brouillon", func(b map[string]any) { b["startsAt"] = "2026-12-01T10:00" }))
	up := f.publishedEvent(t, eventBody("À venir"))
	ids := func(q string) []int64 {
		out := expectJSON[struct {
			Items []api.EventSummary `json:"items"`
		}](t, f.admin.Do("GET", "/api/admin/events"+q, nil), http.StatusOK)
		var r []int64
		for _, e := range out.Items {
			r = append(r, e.ID)
		}
		return r
	}
	eq := func(got []int64, want ...int64) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%v, attendu %v", got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("%v, attendu %v", got, want)
			}
		}
	}
	eq(ids(""), past.ID, draft.ID, up.ID)
	eq(ids("?when=upcoming"), draft.ID, up.ID)
	eq(ids("?when=past"), past.ID)
	eq(ids("?status=finished"), past.ID)
	eq(ids("?status=draft"), draft.ID)
	eq(ids("?season=all&status=published"), past.ID, up.ID)
}
