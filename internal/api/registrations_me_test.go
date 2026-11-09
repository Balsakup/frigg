package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

func TestRegisterAndCancel(t *testing.T) {
	f := newEvFixture(t)
	ev := f.publishedEvent(t, eventBody("AG", func(b map[string]any) { b["capacity"] = 5 }))

	// Fiche non adhérente : refus sur l'élément concerné.
	res := f.claire.Do("POST", myEventPath(ev.ID, "/registrations"), map[string]any{"memberIds": []int64{f.lucas.ID}})
	if res.Status != http.StatusUnprocessableEntity || !strings.Contains(string(res.Body), "body.memberIds[0]") ||
		!strings.Contains(string(res.Body), "n'est pas adhérent(e) de la saison 2026-2027") {
		t.Fatalf("non adhérent : %d %s", res.Status, res.Body)
	}

	out := register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID)
	if len(out.CreatedRegistrationIDs) != 1 || out.Event.ConfirmedCount != 1 || *out.Event.PlacesLeft != 4 {
		t.Fatalf("inscription : %+v", out)
	}
	if n := mails(f.env, f.claireEmail, "Inscription confirmée"); n != 1 {
		t.Fatalf("e-mails de confirmation : %d", n)
	}
	// Déjà inscrite.
	res = f.claire.Do("POST", myEventPath(ev.ID, "/registrations"), map[string]any{"memberIds": []int64{f.self.ID}})
	if res.Status != http.StatusUnprocessableEntity || !strings.Contains(string(res.Body), "déjà inscrit") {
		t.Fatalf("doublon : %d %s", res.Status, res.Body)
	}
	// Fiche d'un autre compte : 404 sans dire laquelle.
	_, _, stranger := f.member(t, "Paul")
	expectProblem(t, f.claire.Do("POST", myEventPath(ev.ID, "/registrations"), map[string]any{"memberIds": []int64{stranger.ID}}),
		http.StatusNotFound, api.MsgMemberGone)

	list := expectJSON[struct {
		Items []api.MyRegistration `json:"items"`
	}](t, f.claire.Do("GET", "/api/me/registrations", nil), http.StatusOK)
	if len(list.Items) != 1 || list.Items[0].State != "confirmed" || !list.Items[0].Actions.Cancel || list.Items[0].Payment != nil {
		t.Fatalf("mes inscriptions : %+v", list.Items)
	}
	id := out.CreatedRegistrationIDs[0]
	got := expectJSON[api.MyRegistration](t, f.claire.Do("POST", myRegistrationPath(id, "/cancel"), nil), http.StatusOK)
	if got.State != "cancelled" || got.CancelledAt == nil {
		t.Fatalf("annulation : %+v", got)
	}
	expectProblem(t, f.claire.Do("POST", myRegistrationPath(id, "/cancel"), nil), http.StatusConflict, api.MsgRegistrationChanged)
	// Réinscription : nouvelle ligne.
	again := register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID)
	if again.CreatedRegistrationIDs[0] == id || again.Event.ConfirmedCount != 1 {
		t.Fatalf("réinscription : %+v", again)
	}

	// Un autre compte ne voit pas l'inscription.
	paul := f.env.LoggedIn(f.env.Seed(auth.RoleMember, true))
	expectProblem(t, paul.Do("GET", myRegistrationPath(id, ""), nil), http.StatusNotFound, api.MsgRegistrationGone)
	expectProblem(t, paul.Do("POST", myRegistrationPath(again.CreatedRegistrationIDs[0], "/cancel"), nil), http.StatusNotFound, api.MsgRegistrationGone)

	// Période close.
	f.env.SetToday("2026-12-13")
	expectProblem(t, f.claire.Do("POST", myRegistrationPath(again.CreatedRegistrationIDs[0], "/cancel"), nil), http.StatusConflict, api.MsgRegistrationsClosed)
	if logs := f.env.Logs.String(); strings.Count(logs, `"action":"registration.create"`) != 2 || strings.Count(logs, `"action":"registration.cancel"`) != 1 {
		t.Fatal("journaux des inscriptions")
	}
	noLeak(t, f.env, "Claire", "Lucas", f.claireEmail)
}

func TestRegisterClosedAndFull(t *testing.T) {
	f := newEvFixture(t)
	open := func(b map[string]any) { b["openness"] = "all_members" }
	draft := f.createEvent(t, eventBody("Brouillon", open))
	expectProblem(t, f.claire.Do("POST", myEventPath(draft.ID, "/registrations"), map[string]any{"memberIds": []int64{f.self.ID}}),
		http.StatusNotFound, api.MsgEventGone)

	closed := f.publishedEvent(t, eventBody("Clos", open, func(b map[string]any) { b["registrationClosesAt"] = "2026-11-01T10:00" }))
	expectProblem(t, f.claire.Do("POST", myEventPath(closed.ID, "/registrations"), map[string]any{"memberIds": []int64{f.self.ID}}),
		http.StatusConflict, api.MsgRegistrationsClosed)

	small := f.publishedEvent(t, eventBody("Petit", open, func(b map[string]any) { b["capacity"] = 1 }))
	// 1 place pour 2 fiches : tout ou rien.
	expectProblem(t, f.claire.Do("POST", myEventPath(small.ID, "/registrations"), map[string]any{"memberIds": []int64{f.self.ID, f.lucas.ID}}),
		http.StatusConflict, "Complet : il reste 1 place(s).")
	register(t, f.claire, small.ID, http.StatusCreated, f.lucas.ID)
	expectProblem(t, f.claire.Do("POST", myEventPath(small.ID, "/registrations"), map[string]any{"memberIds": []int64{f.self.ID}}),
		http.StatusConflict, "Complet : il reste 0 place(s).")

	// Deux fiches d'un coup : un seul e-mail qui nomme les deux.
	both := f.publishedEvent(t, eventBody("Pique-nique", open))
	before := mails(f.env, f.claireEmail, "Inscription confirmée")
	register(t, f.claire, both.ID, http.StatusCreated, f.self.ID, f.lucas.ID)
	if mails(f.env, f.claireEmail, "Inscription confirmée") != before+1 {
		t.Fatal("un seul e-mail attendu pour deux fiches")
	}
	if text := lastMail(t, f.env, f.claireEmail); !strings.Contains(text, "Lucas Martin") || !strings.Contains(text, "Claire Martin") {
		t.Fatalf("e-mail : %s", text)
	}
}

// 50 inscriptions simultanées sur 3 places : exactement 3 réussissent (SC-004).
func TestRegisterConcurrency(t *testing.T) {
	f := newEvFixture(t)
	ev := f.publishedEvent(t, eventBody("Atelier", func(b map[string]any) { b["openness"], b["capacity"] = "all_members", 3 }))
	// Un compte gère au plus 20 fiches : trois comptes se partagent les 50.
	type target struct {
		c  *apitest.Client
		id int64
	}
	var targets []target
	for a := range 3 {
		c := f.claire
		if a > 0 {
			c, _, _ = f.member(t, "Parent")
		}
		for i := 0; i < 17 && len(targets) < 50; i++ {
			m := createMine(t, c, memberBody(fmt.Sprintf("Enfant%d%02d", a, i), "Martin", "2016-01-01", "confirmDuplicate", true))
			targets = append(targets, target{c, m.ID})
		}
	}
	var wg sync.WaitGroup
	statuses := make([]int, len(targets))
	for i, tg := range targets {
		wg.Go(func() {
			statuses[i] = tg.c.Do("POST", myEventPath(ev.ID, "/registrations"), map[string]any{"memberIds": []int64{tg.id}}).Status
		})
	}
	wg.Wait()
	created, full := 0, 0
	for _, s := range statuses {
		switch s {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			full++
		default:
			t.Errorf("statut inattendu %d", s)
		}
	}
	if created != 3 || full != 47 {
		t.Fatalf("créées %d, complets %d", created, full)
	}
	got := expectAdminEvent(t, f.admin.Do("GET", eventPath(ev.ID, ""), nil), http.StatusOK)
	if got.ConfirmedCount != 3 {
		t.Fatalf("inscrits : %d", got.ConfirmedCount)
	}
}
