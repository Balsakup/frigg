package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
)

func TestCancelRestoreEvent(t *testing.T) {
	f := newEvFixture(t)
	setPayment(t, f.admin, nil)
	ev := f.publishedEvent(t, eventBody("Tournoi", paidEvent("10")))
	out := register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID, f.lucas.ID)
	paid := out.CreatedRegistrationIDs[0]
	expectJSON[api.AdminRegistration](t, f.admin.Do("POST", adminRegistrationPath(paid, "/payment"), map[string]any{
		"method": "cash", "receivedOn": "2026-11-01",
	}), http.StatusOK)

	long := strings.Repeat("a", 1001)
	if res := f.admin.Do("POST", eventPath(ev.ID, "/cancel"), map[string]any{"message": long}); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("message trop long : %d", res.Status)
	}
	before := mails(f.env, f.claireEmail, "Événement annulé")
	got := expectAdminEvent(t, f.admin.Do("POST", eventPath(ev.ID, "/cancel"), map[string]any{"message": "Salle indisponible."}), http.StatusOK)
	if got.Status != "cancelled" || got.CancelMessage == nil || got.ConfirmedCount != 2 || !got.Actions.Restore || got.Actions.Cancel {
		t.Fatalf("annulé : %+v", got)
	}
	if n := mails(f.env, f.claireEmail, "Événement annulé"); n != before+1 {
		t.Fatalf("e-mails d'annulation : %d", n-before)
	}
	if text := lastMail(t, f.env, f.claireEmail); !strings.Contains(text, "Salle indisponible.") || !strings.Contains(text, "remboursement") {
		t.Fatalf("e-mail : %s", text)
	}

	// Côté membre : visible et annulé, plus rien à régler ni à payer.
	mine := expectJSON[api.MyEvent](t, f.claire.Do("GET", myEventPath(ev.ID, ""), nil), http.StatusOK)
	if mine.Phase != "cancelled" || mine.RegistrationOpen {
		t.Fatalf("membre : %+v", mine.EventSummary)
	}
	dues := expectJSON[struct {
		Items []api.Due `json:"items"`
	}](t, f.claire.Do("GET", "/api/me/dues", nil), http.StatusOK)
	if len(dues.Items) != 0 {
		t.Fatalf("à régler : %+v", dues.Items)
	}
	expectProblem(t, f.claire.Do("PUT", myRegistrationPath(out.CreatedRegistrationIDs[1], "/payment"), map[string]any{"method": "cash"}),
		http.StatusConflict, api.MsgNotPayableRegistration)
	expectProblem(t, f.admin.Do("POST", eventPath(ev.ID, "/cancel"), map[string]any{}), http.StatusConflict, api.MsgEventChanged)

	// Rétablissement : paiements intacts, e-mail.
	got = expectAdminEvent(t, f.admin.Do("POST", eventPath(ev.ID, "/restore"), nil), http.StatusOK)
	if got.Status != "published" || got.CancelMessage != nil || got.Totals.PaidCount != 1 {
		t.Fatalf("rétabli : %+v", got)
	}
	if n := mails(f.env, f.claireEmail, "Événement rétabli"); n != 1 {
		t.Fatalf("e-mails de rétablissement : %d", n)
	}

	// Suppression refusée avec des inscriptions.
	expectProblem(t, f.admin.Do("DELETE", eventPath(ev.ID, ""), nil), http.StatusConflict, api.MsgEventHasRegistrations)
	// Terminé : plus d'annulation ni de rétablissement.
	f.env.SetToday("2026-12-13")
	expectProblem(t, f.admin.Do("POST", eventPath(ev.ID, "/cancel"), map[string]any{}), http.StatusConflict, api.MsgEventChanged)

	logs := f.env.Logs.String()
	if !strings.Contains(logs, `"action":"event.cancel"`) || !strings.Contains(logs, `"notified":1`) {
		t.Fatal("journal d'annulation")
	}
	noLeak(t, f.env, "Salle indisponible.", "Tournoi")
}

func TestDeleteEvent(t *testing.T) {
	f := newEvFixture(t)
	draft := f.createEvent(t, eventBody("Erreur"))
	if res := f.admin.Do("DELETE", eventPath(draft.ID, ""), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", res.Status, res.Body)
	}
	expectProblem(t, f.admin.Do("GET", eventPath(draft.ID, ""), nil), http.StatusNotFound, api.MsgEventGone)

	// Une inscription annulée suffit à empêcher la suppression.
	ev := f.publishedEvent(t, eventBody("AG"))
	out := register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID)
	expectJSON[api.MyRegistration](t, f.claire.Do("POST", myRegistrationPath(out.CreatedRegistrationIDs[0], "/cancel"), nil), http.StatusOK)
	expectProblem(t, f.admin.Do("DELETE", eventPath(ev.ID, ""), nil), http.StatusConflict, api.MsgEventHasRegistrations)
}
