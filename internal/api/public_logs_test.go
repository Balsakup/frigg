package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
)

// Journaux des parcours publics (016, FR-042, research.md › R16) : aucune
// ligne ne porte d'adresse électronique, de nom, de jeton ni de slug, et le
// journal des requêtes ne voit jamais le fragment #jeton= (point de
// vigilance 1 du plan).

// bareTokenRe repère une chaîne base64url isolée de 43 caractères : la forme
// d'un jeton brut (auth.NewToken).
var bareTokenRe = regexp.MustCompile(`(^|[^A-Za-z0-9_-])[A-Za-z0-9_-]{43}([^A-Za-z0-9_-]|$)`)

// visitLink simule l'ouverture d'un lien reçu par e-mail : le navigateur
// n'envoie jamais le fragment au serveur.
func visitLink(t *testing.T, c *apitest.Client, m apitest.Mail) {
	t.Helper()
	path, ok := strings.CutPrefix(m.Link, "http://frigg.test")
	if !ok {
		t.Fatalf("lien inattendu : %q", m.Link)
	}
	if res := c.Do("GET", path, nil); res.Status != http.StatusOK {
		t.Fatalf("page %s : %d", path, res.Status)
	}
}

// latestMail renvoie le dernier e-mail mis en file pour email.
func latestMail(t *testing.T, env *apitest.Env, email string) apitest.Mail {
	t.Helper()
	mails := env.Mails(email)
	if len(mails) == 0 {
		t.Fatalf("aucun e-mail pour %s", email)
	}
	return mails[len(mails)-1]
}

// checkPublicLogs vérifie l'ensemble des journaux : ni secret littéral, ni
// adresse, ni jeton de 43 caractères ; chemins des requêtes sans « jeton ».
// Elle renvoie les lignes décodées par action.
func checkPublicLogs(t *testing.T, env *apitest.Env, secrets ...string) map[string][]map[string]any {
	t.Helper()
	byAction := map[string][]map[string]any{}
	for line := range strings.SplitSeq(strings.TrimSpace(env.Logs.String()), "\n") {
		for _, s := range secrets {
			if s != "" && strings.Contains(line, s) {
				t.Errorf("%q journalisé : %s", s, line)
			}
		}
		if strings.Contains(line, "@") {
			t.Errorf("adresse électronique journalisée : %s", line)
		}
		if bareTokenRe.MatchString(line) {
			t.Errorf("jeton journalisé : %s", line)
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("ligne illisible %q : %v", line, err)
		}
		if path, ok := m["path"].(string); ok && strings.Contains(path, "jeton") {
			t.Errorf("jeton dans le journal des requêtes : %s", line)
		}
		if action, ok := m["action"].(string); ok {
			byAction[action] = append(byAction[action], m)
		}
	}
	return byAction
}

// expectAttrs vérifie qu'une action est journalisée et que ses lignes
// portent exactement les attributs propres attendus.
func expectAttrs(t *testing.T, byAction map[string][]map[string]any, action string, attrs ...string) {
	t.Helper()
	lines := byAction[action]
	if len(lines) == 0 {
		t.Errorf("action %s non journalisée", action)
		return
	}
	if len(attrs) == 0 {
		return
	}
	common := []string{"time", "level", "msg", "action", "request_id"}
	for _, line := range lines {
		for key := range line {
			if !slices.Contains(common, key) && !slices.Contains(attrs, key) {
				t.Errorf("%s : attribut inattendu %q (%v)", action, key, line)
			}
		}
		for _, key := range attrs {
			if _, ok := line[key]; !ok {
				t.Errorf("%s : attribut %q absent (%v)", action, key, line)
			}
		}
	}
}

func TestPublicJourneysLogs(t *testing.T) {
	f := newPubFixture(t, true)
	setPayment(t, f.admin, nil)
	const email = "ondine.vasseur@exemple.fr"
	identity := func(extra ...any) []any {
		return append([]any{"firstName", "Ondine", "lastName", "Vasseur"}, extra...)
	}
	// Fiche existante sans adresse : doublon probable de la demande publique.
	existing := expectMember(t, f.admin.Do("POST", "/api/admin/members",
		memberBody("Ondine", "Vasseur", "1985-03-02", "accountId", nil)), http.StatusCreated)

	// US1 : dépôt ignoré (piège à robots), dépôt, confirmation, suivi.
	expectAccepted(t, submit(f.env.Client(), f.nextIP(), f.publicBody(email, identity("website", "http://robot.exemple")...)))
	token := f.deposit(t, f.publicBody(email, identity()...))
	c := f.env.Client()
	visitLink(t, c, latestMail(t, f.env, email))
	expectOutcome(t, f.confirm(c, token), "tracking")
	if res := c.Do("GET", "/api/tracking", nil); res.Status != http.StatusOK {
		t.Fatalf("suivi : %d %s", res.Status, res.Body)
	}
	pending := adminMemberships(t, f.admin, "?origin=public&status=pending")
	if len(pending.Items) != 1 {
		t.Fatalf("demande publique : %+v", pending)
	}
	// Rattachement du doublon.
	expectAdminMembership(t, f.admin.Do("POST", attachPath(pending.Items[0].ID), map[string]any{"memberId": existing.ID}), http.StatusOK)

	// US2 : demande de lien, ouverture, envoi par l'administration.
	expectAccepted(t, f.env.Client().Do("POST", "/api/public/tracking/link-request", map[string]string{"email": email}, "X-Forwarded-For", f.nextIP()))
	link := f.env.Client()
	visitLink(t, link, latestMail(t, f.env, email))
	if res := link.Do("POST", "/api/public/tracking/open", map[string]string{"token": latestMail(t, f.env, email).Token}); res.Status != http.StatusNoContent {
		t.Fatalf("ouverture : %d %s", res.Status, res.Body)
	}
	if res := f.admin.Do("POST", fmt.Sprintf("/api/admin/members/%d/tracking-link", existing.ID), nil); res.Status != http.StatusNoContent {
		t.Fatalf("envoi du lien : %d %s", res.Status, res.Body)
	}
	visitLink(t, f.env.Client(), latestMail(t, f.env, email))

	// US3 : événement public, inscription, régénération du lien.
	ev := f.publicEvent(t, "Stage découverte", withPrice("35"))
	slug := slugOf(t, ev)
	visitor := f.env.Client()
	if res := visitor.Do("GET", "/e/"+slug, nil); res.Status != http.StatusOK {
		t.Fatalf("page publique : %d", res.Status)
	}
	getPublicEvent(t, visitor, slug)
	regToken := f.depositRegistration(t, slug, regBody(email, identity()...))
	expectRegOutcome(t, f.confirm(visitor, regToken), "tracking")
	regenerated := expectJSON[api.AdminEvent](t, f.admin.Do("POST", eventPath(ev.ID, "/public-link"), nil), http.StatusOK)
	newSlug := slugOf(t, regenerated)
	if newSlug == slug {
		t.Fatal("lien public non régénéré")
	}
	if res := visitor.Do("GET", "/e/"+newSlug, nil); res.Status != http.StatusOK {
		t.Fatalf("nouvelle page publique : %d", res.Status)
	}
	getPublicEvent(t, visitor, newSlug)

	// US4 : création du compte depuis le suivi.
	if res := createTrackingAccount(link, existing.ID); res.Status != http.StatusCreated {
		t.Fatalf("création du compte : %d %s", res.Status, res.Body)
	}

	// US5 : suppression en lot (fiche inconnue : ignorée).
	if res := f.admin.Do("POST", "/api/admin/members/inactive/delete", map[string]any{"ids": []int64{999999}}); res.Status != http.StatusOK {
		t.Fatalf("suppression des fiches inactives : %d %s", res.Status, res.Body)
	}

	secrets := []string{email, "Ondine", "Vasseur", slug, newSlug}
	for _, m := range f.env.Mails(email) {
		secrets = append(secrets, m.Token)
	}
	byAction := checkPublicLogs(t, f.env, secrets...)
	expectAttrs(t, byAction, "public_submit")
	expectAttrs(t, byAction, "public_submit_rejected", "kind", "reason")
	expectAttrs(t, byAction, "public_confirm")
	expectAttrs(t, byAction, "tracking_link_request", "outcome")
	expectAttrs(t, byAction, "tracking_link_issued", "tracking_link_id")
	expectAttrs(t, byAction, "tracking_link_opened", "tracking_link_id")
	expectAttrs(t, byAction, "tracking_link_sent", "actor_id", "member_id")
	expectAttrs(t, byAction, "tracking_account_created", "account_id")
	expectAttrs(t, byAction, "membership_attached")
	expectAttrs(t, byAction, "event_public_link_regenerated", "actor_id", "event_id")
	expectAttrs(t, byAction, "inactive_members_deleted", "actor_id", "deleted", "skipped")

	// Journal des requêtes : les deux slugs sont masqués.
	masked := 0
	for _, line := range logLines(t, f.env) {
		if path, _ := line["path"].(string); path == "/e/…" || strings.HasPrefix(path, api.PublicEventPathPrefix+"…") {
			masked++
		}
	}
	if masked < 5 {
		t.Errorf("requêtes au slug masqué : %d", masked)
	}
}

// Paiement par carte depuis un lien personnel : ni adresse, ni nom, ni jeton
// dans les journaux.
func TestTrackingCardLogs(t *testing.T) {
	f := newTrackCardFixture(t)
	st := f.start(t)
	f.env.HelloAsso.Authorize(f.env.HelloAsso.LastIntent().ID, 3000, 0)
	f.env.Advance(10 * time.Second)
	if out := f.confirm(t, st.OnlinePaymentID, "return"); out.Status != "succeeded" {
		t.Fatalf("constatation : %+v", out)
	}
	secrets := []string{trackedEmailAddr, "Léo"}
	for _, m := range f.env.Mails(trackedEmailAddr) {
		secrets = append(secrets, m.Token)
	}
	checkPublicLogs(t, f.env, secrets...)
}
