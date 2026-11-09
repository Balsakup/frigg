package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
)

// evFixture : jeu de données des adhésions (saison 2026-2027 au 1er novembre
// 2026) et une famille : Claire gère sa fiche (adhérente validée) et celle de
// son fils Lucas (non adhérent).
type evFixture struct {
	msFixture
	claire      *apitest.Client
	claireEmail string
	self, lucas api.Member
}

func newEvFixture(t *testing.T) evFixture {
	t.Helper()
	return withFamily(t, newMsFixture(t))
}

func withFamily(t *testing.T, ms msFixture) evFixture {
	t.Helper()
	c, email, self := ms.member(t, "Claire")
	lucas := createMine(t, c, memberBody("Lucas", "Martin", "2015-04-12", "confirmDuplicate", false))
	m := request(t, c, self.ID, ms.season.ID, ms.free.ID, "first")
	validate(t, ms.admin, m.ID, ms.free.ID, "first")
	return evFixture{msFixture: ms, claire: c, claireEmail: email, self: self, lucas: lucas}
}

func eventBody(title string, edits ...func(map[string]any)) map[string]any {
	b := map[string]any{
		"title": title, "location": "Salle polyvalente",
		"startsAt": "2026-12-12T14:00", "endsAt": "2026-12-12T17:00",
		"openness": "season_members",
	}
	for _, e := range edits {
		e(b)
	}
	return b
}

func eventPath(id int64, suffix string) string {
	return fmt.Sprintf("/api/admin/events/%d%s", id, suffix)
}

func myEventPath(id int64, suffix string) string {
	return fmt.Sprintf("/api/me/events/%d%s", id, suffix)
}

func myRegistrationPath(id int64, suffix string) string {
	return fmt.Sprintf("/api/me/registrations/%d%s", id, suffix)
}

func adminRegistrationPath(id int64, suffix string) string {
	return fmt.Sprintf("/api/admin/registrations/%d%s", id, suffix)
}

func expectJSON[T any](t *testing.T, res apitest.Response, status int) T {
	t.Helper()
	if res.Status != status {
		t.Fatalf("statut %d, attendu %d : %s", res.Status, status, res.Body)
	}
	var out T
	res.JSON(t, &out)
	return out
}

func expectAdminEvent(t *testing.T, res apitest.Response, status int) api.AdminEvent {
	t.Helper()
	return expectJSON[api.AdminEvent](t, res, status)
}

// createEvent crée un brouillon ; publish le publie.
func (f evFixture) createEvent(t *testing.T, body map[string]any) api.AdminEvent {
	t.Helper()
	return expectAdminEvent(t, f.admin.Do("POST", "/api/admin/events", body), http.StatusCreated)
}

func (f evFixture) publish(t *testing.T, id int64) api.AdminEvent {
	t.Helper()
	return expectAdminEvent(t, f.admin.Do("POST", eventPath(id, "/publish"), nil), http.StatusOK)
}

func (f evFixture) publishedEvent(t *testing.T, body map[string]any) api.AdminEvent {
	t.Helper()
	return f.publish(t, f.createEvent(t, body).ID)
}

// register inscrit des fiches depuis le compte c et attend le statut donné.
func register(t *testing.T, c *apitest.Client, eventID int64, status int, memberIDs ...int64) api.RegisterResult {
	t.Helper()
	res := c.Do("POST", myEventPath(eventID, "/registrations"), map[string]any{"memberIds": memberIDs})
	if res.Status != status {
		t.Fatalf("inscription : statut %d, attendu %d : %s", res.Status, status, res.Body)
	}
	var out api.RegisterResult
	if status == http.StatusCreated {
		res.JSON(t, &out)
	}
	return out
}

func (f evFixture) myRegistration(t *testing.T, id int64) api.MyRegistration {
	t.Helper()
	return expectJSON[api.MyRegistration](t, f.claire.Do("GET", myRegistrationPath(id, ""), nil), http.StatusOK)
}

// mails renvoie les e-mails envoyés à to dont l'objet commence par prefix.
func mails(env *apitest.Env, to, prefix string) int {
	env.FlushMail()
	n := 0
	for _, m := range env.Mail.Messages() {
		if m.To == to && strings.HasPrefix(m.Msg.Subject, prefix) {
			n++
		}
	}
	return n
}

// lastMail renvoie le texte du dernier e-mail envoyé à to.
func lastMail(t *testing.T, env *apitest.Env, to string) string {
	t.Helper()
	env.FlushMail()
	m, ok := env.Mail.Last(to)
	if !ok {
		t.Fatalf("aucun e-mail pour %s", to)
	}
	return m.Msg.Subject + "\n" + m.Msg.Text
}

// noLeak vérifie qu'aucune ligne de journal ne contient les textes interdits.
func noLeak(t *testing.T, env *apitest.Env, forbidden ...string) {
	t.Helper()
	logs := env.Logs.String()
	for _, f := range forbidden {
		if strings.Contains(logs, f) {
			t.Errorf("%q journalisé", f)
		}
	}
}
