package api_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

// Événement « Ouvert au public » (016, US3) : page publique, dépôt et
// confirmation d'une inscription.

// publicEvent crée et publie un événement ouvert au public.
func (f *pubFixture) publicEvent(t *testing.T, title string, edits ...func(map[string]any)) api.AdminEvent {
	t.Helper()
	edits = append([]func(map[string]any){func(b map[string]any) { b["openness"] = "public" }}, edits...)
	return evFixture{msFixture: f.msFixture}.publishedEvent(t, eventBody(title, edits...))
}

func withCapacity(n int) func(map[string]any) {
	return func(b map[string]any) { b["capacity"] = n }
}

func withPrice(price string) func(map[string]any) {
	return func(b map[string]any) { b["price"] = price }
}

// slugOf renvoie le slug du lien public d'un événement.
func slugOf(t *testing.T, ev api.AdminEvent) string {
	t.Helper()
	if ev.PublicURL == nil {
		t.Fatalf("événement %d sans lien public", ev.ID)
	}
	_, slug, ok := strings.Cut(*ev.PublicURL, "/e/")
	if !ok || len(slug) != 22 {
		t.Fatalf("lien public %q", *ev.PublicURL)
	}
	return slug
}

func publicEventPath(slug string) string { return "/api/public/events/" + slug }

func getPublicEvent(t *testing.T, c *apitest.Client, slug string) api.PublicEvent {
	t.Helper()
	return expectJSON[api.PublicEvent](t, c.Do("GET", publicEventPath(slug), nil), http.StatusOK)
}

// regBody renvoie un dépôt d'inscription valide pour l'adresse email.
func regBody(email string, extra ...any) map[string]any {
	b := map[string]any{
		"firstName": "Claire", "lastName": "Martin", "birthDate": "1985-03-02", "email": email, "consent": true,
	}
	for i := 0; i+1 < len(extra); i += 2 {
		b[extra[i].(string)] = extra[i+1]
	}
	return b
}

func submitRegistration(c *apitest.Client, ip, slug string, body map[string]any) apitest.Response {
	return c.Do("POST", publicEventPath(slug)+"/registrations", body, "X-Forwarded-For", ip)
}

// depositRegistration dépose une inscription et renvoie le jeton de
// confirmation reçu par e-mail.
func (f *pubFixture) depositRegistration(t *testing.T, slug string, body map[string]any) string {
	t.Helper()
	email := auth.NormalizeEmail(body["email"].(string))
	before := len(f.env.Mails(email))
	expectAccepted(t, submitRegistration(f.env.Client(), f.nextIP(), slug, body))
	mails := f.env.Mails(email)
	if len(mails) != before+1 || mails[len(mails)-1].Token == "" {
		t.Fatalf("e-mail de confirmation absent : %+v", mails)
	}
	return mails[len(mails)-1].Token
}

func expectRegOutcome(t *testing.T, res apitest.Response, outcome string) api.ConfirmOutcome {
	t.Helper()
	if res.Status != http.StatusOK {
		t.Fatalf("confirmation : %d %s", res.Status, res.Body)
	}
	var o api.ConfirmOutcome
	res.JSON(t, &o)
	if o.Outcome != outcome || o.Kind != "registration" {
		t.Fatalf("issue %+v, attendu %s", o, outcome)
	}
	return o
}

// publicEventKeys est la liste fermée des champs de PublicEvent
// (contracts/api.md, principe II).
var publicEventKeys = []string{
	"closedReason", "description", "endsAt", "location", "placesLeft", "priceCents",
	"registrationClosesAt", "registrationOpen", "registrationOpensAt", "startsAt", "title",
}

func TestGetPublicEvent(t *testing.T) {
	f := newPubFixture(t, false)
	anonymous := f.env.Client()
	createField(t, f.admin, fieldBody("member", "Club précédent", "short_text", "visibility", "editable"))
	ev := f.publicEvent(t, "Stage découverte", withCapacity(2), withPrice("35"),
		func(b map[string]any) { b["description"] = "Venez essayer." })
	slug := slugOf(t, ev)
	leo := trackedMember(t, f.msFixture, "Léo", "leo@exemple.fr")
	expectJSON[api.AdminRegistration](t, f.admin.Do("POST", eventPath(ev.ID, "/registrations"), map[string]any{"memberId": leo}), http.StatusCreated)

	// Liste fermée des champs, comparée sur le JSON brut.
	res := anonymous.Do("GET", publicEventPath(slug), nil)
	if res.Status != http.StatusOK {
		t.Fatalf("page publique : %d %s", res.Status, res.Body)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(res.Body), &raw); err != nil {
		t.Fatal(err)
	}
	if keys := slices.Sorted(maps.Keys(raw)); !slices.Equal(keys, publicEventKeys) {
		t.Fatalf("champs %v, attendu %v", keys, publicEventKeys)
	}
	for _, leak := range []string{"Léo", "leo@exemple.fr", "Club précédent", fmt.Sprintf(`"id":%d`, ev.ID)} {
		if strings.Contains(res.Body, leak) {
			t.Errorf("%q transmis par la page publique", leak)
		}
	}
	got := getPublicEvent(t, anonymous, slug)
	if got.Title != "Stage découverte" || got.Location != "Salle polyvalente" || got.Description == nil ||
		*got.Description != "Venez essayer." || got.PriceCents != 3500 || got.PlacesLeft == nil || *got.PlacesLeft != 1 ||
		!got.RegistrationOpen || got.ClosedReason != nil || !got.RegistrationClosesAt.Equal(got.StartsAt) {
		t.Fatalf("page publique : %+v", got)
	}

	// 404 identique : slug inconnu, brouillon, événement non public, slug
	// régénéré (FR-025).
	notFound := anonymous.Do("GET", publicEventPath("AAAAAAAAAAAAAAAAAAAAAA"), nil)
	if notFound.Status != http.StatusNotFound || !strings.Contains(notFound.Body, "Événement introuvable.") {
		t.Fatalf("slug inconnu : %d %s", notFound.Status, notFound.Body)
	}
	draft := evFixture{msFixture: f.msFixture}.createEvent(t, eventBody("Brouillon", func(b map[string]any) { b["openness"] = "public" }))
	hidden := f.publicEvent(t, "Devenu interne")
	hiddenSlug := slugOf(t, hidden)
	expectAdminEvent(t, f.admin.Do("PUT", eventPath(hidden.ID, ""), eventBody("Devenu interne", paidEvent("0"))), http.StatusOK)
	regenerated := f.publicEvent(t, "Régénéré")
	oldSlug := slugOf(t, regenerated)
	expectAdminEvent(t, f.admin.Do("POST", eventPath(regenerated.ID, "/public-link"), nil), http.StatusOK)
	for name, s := range map[string]string{"brouillon": slugOf(t, draft), "non public": hiddenSlug, "régénéré": oldSlug, "trop long": strings.Repeat("A", 300)} {
		res := anonymous.Do("GET", publicEventPath(s), nil)
		if res.Status != notFound.Status || res.Body != notFound.Body {
			t.Errorf("%s : %d %s, attendu la réponse d'un lien inconnu", name, res.Status, res.Body)
		}
		if res := submitRegistration(anonymous, f.nextIP(), s, regBody("x@exemple.fr")); res.Status != http.StatusNotFound || res.Body != notFound.Body {
			t.Errorf("dépôt %s : %d %s", name, res.Status, res.Body)
		}
	}

	// Raisons de fermeture.
	full := f.publicEvent(t, "Complet", withCapacity(1))
	expectJSON[api.AdminRegistration](t, f.admin.Do("POST", eventPath(full.ID, "/registrations"), map[string]any{"memberId": leo}), http.StatusCreated)
	notYet := f.publicEvent(t, "Bientôt", func(b map[string]any) { b["registrationOpensAt"] = "2026-11-20T10:00" })
	closing := f.publicEvent(t, "Clos bientôt", func(b map[string]any) { b["registrationClosesAt"] = "2026-11-10T10:00" })
	cancelled := f.publicEvent(t, "Annulé")
	expectAdminEvent(t, f.admin.Do("POST", eventPath(cancelled.ID, "/cancel"), map[string]any{"message": "Gymnase fermé pour travaux"}), http.StatusOK)
	reason := func(ev api.AdminEvent) string {
		t.Helper()
		got := getPublicEvent(t, anonymous, slugOf(t, ev))
		if got.ClosedReason == nil {
			if !got.RegistrationOpen {
				t.Fatalf("%s : fermé sans raison", ev.Title)
			}
			return ""
		}
		if got.RegistrationOpen {
			t.Fatalf("%s : ouvert avec la raison %s", ev.Title, *got.ClosedReason)
		}
		return *got.ClosedReason
	}
	for ev, want := range map[*api.AdminEvent]string{&full: "full", &notYet: "not_yet_open", &closing: "", &cancelled: "cancelled"} {
		if got := reason(*ev); got != want {
			t.Errorf("%s : raison %q, attendu %q", ev.Title, got, want)
		}
	}
	if res := anonymous.Do("GET", publicEventPath(slugOf(t, cancelled)), nil); strings.Contains(res.Body, "Gymnase fermé") {
		t.Error("message d'annulation transmis par la page publique")
	}
	f.env.SetToday("2026-11-15")
	if got := reason(closing); got != "closed" {
		t.Errorf("clos : %q", got)
	}
	f.env.SetToday("2026-12-13")
	if got := reason(ev); got != "finished" {
		t.Errorf("terminé : %q", got)
	}
}

func TestSubmitPublicRegistration(t *testing.T) {
	f := newPubFixture(t, false)
	anonymous := f.env.Client()
	ev := f.publicEvent(t, "Stage découverte", withCapacity(1))
	slug := slugOf(t, ev)

	// Dépôt valide : réponse neutre, un dépôt sans champ personnalisé, un
	// e-mail, aucune place réservée.
	email := "claire@exemple.fr"
	expectAccepted(t, submitRegistration(anonymous, f.nextIP(), slug, regBody(" Claire@Exemple.fr ", "createAccount", true)))
	if n := count(t, f.env, `SELECT count(*) FROM public_submissions WHERE kind = 'registration' AND event_id = ? AND email = ?
		AND custom_values = '[]' AND create_account = 1 AND season_id IS NULL`, ev.ID, email); n != 1 {
		t.Fatalf("dépôts : %d", n)
	}
	mails := f.env.Mails(email)
	if len(mails) != 1 || mails[0].Subject != "Confirmez votre demande" ||
		!strings.Contains(mails[0].Text, "inscription de Claire Martin à « Stage découverte »") {
		t.Fatalf("e-mail : %+v", mails)
	}
	if n := count(t, f.env, `SELECT count(*) FROM event_registrations`) + count(t, f.env, `SELECT count(*) FROM members`); n != 0 {
		t.Fatalf("inscription ou fiche créée avant confirmation : %d", n)
	}
	if got := getPublicEvent(t, anonymous, slug); *got.PlacesLeft != 1 {
		t.Fatalf("place réservée avant confirmation : %+v", got)
	}

	// Aucun champ personnalisé accepté (FR-026), saisie invalide : 422.
	if res := submitRegistration(anonymous, f.nextIP(), slug, regBody("x@exemple.fr", "customFields", []any{})); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("champs personnalisés : %d %s", res.Status, res.Body)
	}
	expectFieldError(t, submitRegistration(anonymous, f.nextIP(), slug, regBody("")), "body.email", "L'adresse électronique est obligatoire.")
	expectFieldError(t, submitRegistration(anonymous, f.nextIP(), slug, regBody("x@exemple.fr", "consent", false)), "body.consent", "Vous devez accepter le traitement de vos données.")
	if n := len(f.env.Mails("x@exemple.fr")); n != 0 {
		t.Fatalf("%d e-mails après une saisie invalide", n)
	}

	// Piège : même réponse, rien d'enregistré.
	expectAccepted(t, submitRegistration(anonymous, f.nextIP(), slug, regBody("robot@exemple.fr", "website", "https://spam.example")))
	if n := count(t, f.env, `SELECT count(*) FROM public_submissions WHERE email = 'robot@exemple.fr'`) + len(f.env.Mails("robot@exemple.fr")); n != 0 {
		t.Fatalf("piège : %d", n)
	}
	if !strings.Contains(f.env.Logs.String(), `"action":"public_submit_rejected","kind":"registration","reason":"honeypot"`) {
		t.Error("journal du piège absent")
	}

	// Complet au moment du dépôt : 409 avec la raison.
	leo := trackedMember(t, f.msFixture, "Léo", "leo@exemple.fr")
	expectJSON[api.AdminRegistration](t, f.admin.Do("POST", eventPath(ev.ID, "/registrations"), map[string]any{"memberId": leo}), http.StatusCreated)
	if res := submitRegistration(anonymous, f.nextIP(), slug, regBody("y@exemple.fr")); res.Status != http.StatusConflict ||
		!strings.Contains(res.Body, "Cet événement est complet.") {
		t.Fatalf("complet : %d %s", res.Status, res.Body)
	}

	// Limites (R9) : 10 dépôts par IP et par heure.
	other := f.publicEvent(t, "Portes ouvertes")
	ip := f.nextIP()
	for i := range 10 {
		expectAccepted(t, submitRegistration(anonymous, ip, slugOf(t, other), regBody(fmt.Sprintf("ip%d@exemple.fr", i))))
	}
	if res := submitRegistration(anonymous, ip, slugOf(t, other), regBody("ip10@exemple.fr")); res.Status != http.StatusTooManyRequests {
		t.Fatalf("11e dépôt : %d", res.Status)
	}
	noLeak(t, f.env, email, "Claire", "Martin", slug, slugOf(t, other))
}

func TestConfirmPublicRegistration(t *testing.T) {
	f := newPubFixture(t, false)
	setPayment(t, f.admin, nil)
	ev := f.publicEvent(t, "Stage découverte", withCapacity(2), withPrice("35"))
	slug := slugOf(t, ev)
	email := "claire@exemple.fr"
	token := f.depositRegistration(t, slug, regBody(email, "createAccount", true))

	c := f.env.Client()
	o := expectRegOutcome(t, f.confirm(c, token), "tracking")
	if !o.CreateAccount || o.Reason != nil {
		t.Fatalf("issue : %+v", o)
	}

	// Fiche publique sans compte et inscription confirmée, sans auteur, prix
	// figé, référence EVT-, à payer.
	var (
		regID, memberID int64
		viaPublic       int
		registeredBy    sql.NullInt64
		amount          int64
		reference       string
		status, payment string
		origin          string
		account         sql.NullInt64
	)
	if err := f.env.Store.ReadDB.QueryRow(`SELECT r.id, r.member_id, r.via_public, r.registered_by, r.amount_cents,
		r.payment_reference, r.status, r.payment_status, m.origin, m.account_id
		FROM event_registrations r JOIN members m ON m.id = r.member_id`).
		Scan(&regID, &memberID, &viaPublic, &registeredBy, &amount, &reference, &status, &payment, &origin, &account); err != nil {
		t.Fatal(err)
	}
	if viaPublic != 1 || registeredBy.Valid || amount != 3500 || reference != "EVT-2627-0001" || status != "confirmed" ||
		payment != "due" || origin != "public" || account.Valid {
		t.Fatalf("inscription : %d %v %d %s %s %s %s %v", viaPublic, registeredBy, amount, reference, status, payment, origin, account)
	}
	if n := count(t, f.env, `SELECT count(*) FROM public_submissions`); n != 0 {
		t.Fatalf("dépôt conservé : %d", n)
	}
	if got := getPublicEvent(t, f.env.Client(), slug); *got.PlacesLeft != 1 {
		t.Fatalf("place non décomptée : %+v", got)
	}

	// E-mail de confirmation avec lien personnel vers l'inscription.
	mails := f.env.Mails(email)
	last := mails[len(mails)-1]
	if last.Subject != "Inscription confirmée : Stage découverte" ||
		!strings.Contains(last.Link, fmt.Sprintf("http://frigg.test/suivi/inscriptions/%d#jeton=", regID)) ||
		!strings.Contains(last.Text, "35,00") || !strings.Contains(last.Text, "EVT-2627-0001") {
		t.Fatalf("e-mail : %+v", last)
	}
	// L'accès par lien est ouvert dans la session qui a confirmé.
	got := expectJSON[api.TrackedRegistration](t, c.Do("GET", fmt.Sprintf("/api/tracking/registrations/%d", regID), nil), http.StatusOK)
	if got.AmountCents != 3500 || got.Payment == nil || got.Payment.Status != "due" || !got.Actions.Pay || !got.Actions.Cancel {
		t.Fatalf("inscription suivie : %+v", got)
	}

	// Même identité : aucune seconde inscription, renvoi au suivi.
	again := f.depositRegistration(t, slug, regBody(email))
	expectRegOutcome(t, f.confirm(f.env.Client(), again), "duplicate")
	if n := count(t, f.env, `SELECT count(*) FROM event_registrations WHERE member_id = ?`, memberID); n != 1 {
		t.Fatalf("inscriptions de la fiche : %d", n)
	}

	// Administration : origine publique, filtre.
	adm := expectJSON[api.AdminRegistration](t, f.admin.Do("GET", adminRegistrationPath(regID, ""), nil), http.StatusOK)
	if !adm.ViaPublic || adm.RegisteredBy != nil {
		t.Fatalf("inscription vue par l'administration : %+v", adm)
	}
	leo := trackedMember(t, f.msFixture, "Léo", "leo@exemple.fr")
	expectJSON[api.AdminRegistration](t, f.admin.Do("POST", eventPath(ev.ID, "/registrations"), map[string]any{"memberId": leo}), http.StatusCreated)
	page := expectJSON[api.AdminRegistrationPage](t, f.admin.Do("GET", "/api/admin/registrations?origin=public", nil), http.StatusOK)
	if page.Total != 1 || page.Items[0].ID != regID || !page.Items[0].ViaPublic {
		t.Fatalf("filtre origin=public : %+v", page)
	}
	if all := expectJSON[api.AdminRegistrationPage](t, f.admin.Do("GET", "/api/admin/registrations", nil), http.StatusOK); all.Total != 2 {
		t.Fatalf("sans filtre : %d", all.Total)
	}

	// Événement gratuit : ni référence ni paiement ; la fiche est réutilisée.
	free := f.publicEvent(t, "Portes ouvertes")
	expectRegOutcome(t, f.confirm(f.env.Client(), f.depositRegistration(t, slugOf(t, free), regBody(email))), "tracking")
	var ref, pay sql.NullString
	var member int64
	if err := f.env.Store.ReadDB.QueryRow(`SELECT payment_reference, payment_status, member_id FROM event_registrations WHERE event_id = ?`, free.ID).
		Scan(&ref, &pay, &member); err != nil {
		t.Fatal(err)
	}
	if ref.Valid || pay.Valid || member != memberID {
		t.Fatalf("inscription gratuite : %v %v %d", ref, pay, member)
	}
	noLeak(t, f.env, email, "Claire", slug)
}

func TestConfirmPublicRegistrationClosed(t *testing.T) {
	f := newPubFixture(t, false)
	ev := f.publicEvent(t, "Stage", withCapacity(1))
	slug := slugOf(t, ev)
	first := f.depositRegistration(t, slug, regBody("a@exemple.fr", "firstName", "Anne"))
	second := f.depositRegistration(t, slug, regBody("b@exemple.fr", "firstName", "Bruno"))
	expectRegOutcome(t, f.confirm(f.env.Client(), first), "tracking")
	o := expectRegOutcome(t, f.confirm(f.env.Client(), second), "closed")
	if o.Reason == nil || *o.Reason != "Cet événement est complet." {
		t.Fatalf("raison : %+v", o)
	}
	if n := count(t, f.env, `SELECT count(*) FROM members WHERE first_name = 'Bruno'`); n != 0 {
		t.Fatal("fiche créée pour une inscription impossible")
	}

	cancelled := f.publicEvent(t, "Annulé")
	token := f.depositRegistration(t, slugOf(t, cancelled), regBody("c@exemple.fr"))
	expectAdminEvent(t, f.admin.Do("POST", eventPath(cancelled.ID, "/cancel"), map[string]any{}), http.StatusOK)
	if o := expectRegOutcome(t, f.confirm(f.env.Client(), token), "closed"); *o.Reason != "Cet événement est annulé." {
		t.Fatalf("annulé : %+v", o)
	}

	// L'événement a quitté « Ouvert au public » entre le dépôt et la
	// confirmation.
	internal := f.publicEvent(t, "Interne")
	token = f.depositRegistration(t, slugOf(t, internal), regBody("d@exemple.fr"))
	expectAdminEvent(t, f.admin.Do("PUT", eventPath(internal.ID, ""), eventBody("Interne", paidEvent("0"))), http.StatusOK)
	expectRegOutcome(t, f.confirm(f.env.Client(), token), "closed")
}

// 50 confirmations simultanées pour 3 places : 3 inscriptions (SC-006).
func TestConfirmPublicRegistrationConcurrent(t *testing.T) {
	f := newPubFixture(t, false)
	ev := f.publicEvent(t, "Tournoi", withCapacity(3))
	slug := slugOf(t, ev)
	const n = 50
	tokens := make([]string, n)
	clients := make([]*apitest.Client, n)
	ips := make([]string, n)
	for i := range n {
		tokens[i] = f.depositRegistration(t, slug, regBody(fmt.Sprintf("c%d@exemple.fr", i), "firstName", fmt.Sprintf("Joueur%c", 'A'+i)))
		clients[i], ips[i] = f.env.Client(), f.nextIP()
	}
	outcomes := make([]string, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			res := clients[i].Do("POST", "/api/public/submissions/confirm", map[string]string{"token": tokens[i]}, "X-Forwarded-For", ips[i])
			var o api.ConfirmOutcome
			if res.Status == http.StatusOK {
				_ = json.Unmarshal([]byte(res.Body), &o)
			}
			outcomes[i] = fmt.Sprintf("%d %s", res.Status, o.Outcome)
		})
	}
	wg.Wait()
	tally := map[string]int{}
	for _, o := range outcomes {
		tally[o]++
	}
	if tally["200 tracking"] != 3 || tally["200 closed"] != n-3 {
		t.Fatalf("issues : %v", tally)
	}
	if got := count(t, f.env, `SELECT count(*) FROM event_registrations WHERE event_id = ? AND status = 'confirmed'`, ev.ID); got != 3 {
		t.Fatalf("%d inscriptions, attendu 3", got)
	}
	if got := getPublicEvent(t, f.env.Client(), slug); got.ClosedReason == nil || *got.ClosedReason != "full" {
		t.Fatalf("page publique : %+v", got)
	}
}

// Adresse d'un compte actif : l'inscription va à une fiche du compte et la
// confirmation de 012 part à l'adresse du compte (FR-033).
func TestConfirmPublicRegistrationAccount(t *testing.T) {
	f := newPubFixture(t, false)
	setPayment(t, f.admin, nil)
	email := f.env.Seed(auth.RoleMember, true)
	account := f.env.LoggedIn(email)
	ev := f.publicEvent(t, "Stage", withPrice("10"))
	expectRegOutcome(t, f.confirm(f.env.Client(), f.depositRegistration(t, slugOf(t, ev), regBody(email))), "account")
	got := expectJSON[struct {
		Items []api.MyRegistration `json:"items"`
	}](t, account.Do("GET", "/api/me/registrations", nil), http.StatusOK)
	if len(got.Items) != 1 || got.Items[0].Event.ID != ev.ID || got.Items[0].Payment == nil || got.Items[0].Payment.Status != "due" {
		t.Fatalf("mes inscriptions : %+v", got.Items)
	}
	mails := f.env.Mails(email)
	last := mails[len(mails)-1]
	if last.Subject != "Inscription confirmée : Stage" || strings.Contains(last.Text, "#jeton=") ||
		!strings.Contains(last.Text, fmt.Sprintf("/mes-inscriptions/%d", got.Items[0].ID)) {
		t.Fatalf("e-mail : %+v", last)
	}
}

// Pour un compte connecté, « Ouvert au public » se comporte comme « Toutes
// les fiches » (FR-023).
func TestMemberRegistersToPublicEvent(t *testing.T) {
	f := newEvFixture(t)
	ev := f.publishedEvent(t, eventBody("Portes ouvertes", func(b map[string]any) { b["openness"] = "public" }))
	got := expectJSON[api.MyEvent](t, f.claire.Do("GET", myEventPath(ev.ID, ""), nil), http.StatusOK)
	if got.Openness != "public" {
		t.Fatalf("ouverture : %q", got.Openness)
	}
	for _, m := range got.Members {
		if !m.Eligible {
			t.Fatalf("fiche inéligible : %+v", m)
		}
	}
	out := register(t, f.claire, ev.ID, http.StatusCreated, f.lucas.ID)
	if len(out.CreatedRegistrationIDs) != 1 {
		t.Fatalf("inscription : %+v", out)
	}
	if n := count(t, f.env, `SELECT count(*) FROM event_registrations WHERE via_public = 1`); n != 0 {
		t.Fatal("inscription depuis l'espace membre marquée publique")
	}
}

// Les pages publiques ne sont pas indexées par les moteurs de recherche
// (FR-045).
func TestPublicPagesNoIndex(t *testing.T) {
	f := newPubFixture(t, true)
	slug := slugOf(t, f.publicEvent(t, "Portes ouvertes"))
	for _, path := range []string{"/adherer", "/e/" + slug, "/api/public/membership", publicEventPath(slug)} {
		res := f.env.Client().Do("GET", path, nil)
		if res.Status != http.StatusOK || !strings.Contains(res.Header.Get("X-Robots-Tag"), "noindex") {
			t.Errorf("%s : %d, X-Robots-Tag %q", path, res.Status, res.Header.Get("X-Robots-Tag"))
		}
	}
}
