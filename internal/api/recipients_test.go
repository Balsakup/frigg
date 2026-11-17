package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/auth"
)

// Destinataire unifié des e-mails d'adhésion et d'inscription (016 FR-035,
// FR-040, research.md › R8).

// lastMailTo renvoie le dernier e-mail mis en file pour to dont l'objet
// commence par prefix.
func lastMailTo(t *testing.T, f msFixture, to, prefix string) (string, string) {
	t.Helper()
	mails := f.env.Mails(to)
	for i := len(mails) - 1; i >= 0; i-- {
		if strings.HasPrefix(mails[i].Subject, prefix) {
			return mails[i].Text, mails[i].Link
		}
	}
	t.Fatalf("aucun e-mail « %s » pour %s", prefix, to)
	return "", ""
}

func TestRecipientTrackedMember(t *testing.T) {
	f := newTrackFixture(t)
	const email = "importee@exemple.fr"
	leo := trackedMember(t, f.msFixture, "Léo", email)
	ms := validatedFor(t, f.msFixture, leo, f.young.ID)
	_, link := lastMailTo(t, f.msFixture, email, "Adhésion de Léo Martin validée")
	if !strings.HasPrefix(link, fmt.Sprintf("http://frigg.test/suivi/adhesions/%d#jeton=", ms.ID)) {
		t.Fatalf("lien de paiement : %q", link)
	}
	if n := count(t, f.env, `SELECT count(*) FROM tracking_links WHERE email = ?`, email); n != 1 {
		t.Fatalf("liens personnels : %d", n)
	}
	// Le lien de l'e-mail ouvre la page de paiement.
	token := link[strings.Index(link, "#jeton=")+len("#jeton="):]
	c := f.env.Client()
	if res := c.Do("POST", "/api/public/tracking/open", map[string]string{"token": token}); res.Status != http.StatusNoContent {
		t.Fatalf("ouverture : %d", res.Status)
	}
	expectTracked(t, c.Do("GET", trackingPath(ms.ID, ""), nil), http.StatusOK)

	// Constatation : e-mail avec un lien personnel neuf vers le suivi.
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(ms.ID, "/payment"), map[string]any{
		"method": "cheque", "receivedOn": msToday,
	}), http.StatusOK)
	text, link := lastMailTo(t, f.msFixture, email, "Paiement de l'adhésion")
	if !strings.HasPrefix(link, "http://frigg.test/suivi#jeton=") || !strings.Contains(text, "lien personnel") {
		t.Fatalf("e-mail de constatation : %q\n%s", link, text)
	}

	// Refus d'une demande gratuite ou non : lien vers le suivi.
	zoe := trackedMember(t, f.msFixture, "Zoé", email)
	pending := pendingFor(t, f.msFixture, zoe)
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(pending, "/refuse"), map[string]any{"reason": "Dossier incomplet"}), http.StatusOK)
	if _, link := lastMailTo(t, f.msFixture, email, "Demande d'adhésion de Zoé Martin refusée"); !strings.HasPrefix(link, "http://frigg.test/suivi#jeton=") {
		t.Fatalf("e-mail de refus : %q", link)
	}
	if n := count(t, f.env, `SELECT count(*) FROM tracking_links WHERE email = ?`, email); n != 3 {
		t.Fatalf("liens personnels : %d", n)
	}
}

func TestRecipientAccountUnchanged(t *testing.T) {
	f := newTrackFixture(t)
	c, email, m := f.member(t, "Léo")
	ms := request(t, c, m.ID, f.season.ID, f.young.ID, "first")
	validate(t, f.admin, ms.ID, f.young.ID, "first")
	text, link := lastMailTo(t, f.msFixture, email, "Adhésion de Léo Martin validée")
	if link != "" || !strings.Contains(text, fmt.Sprintf("http://frigg.test/adhesions/%d", ms.ID)) {
		t.Fatalf("e-mail du compte : %q\n%s", link, text)
	}
	if n := count(t, f.env, `SELECT count(*) FROM tracking_links`); n != 0 {
		t.Fatalf("lien personnel émis pour un compte : %d", n)
	}
}

func TestRecipientNoAddress(t *testing.T) {
	f := newTrackFixture(t)
	orphan := insertOrphan(t, f.env, "Jeanne", "Petit", "2010-01-01")
	before := count(t, f.env, `SELECT count(*) FROM outbound_emails`)
	validatedFor(t, f.msFixture, orphan, f.young.ID)
	if after := count(t, f.env, `SELECT count(*) FROM outbound_emails`); after != before {
		t.Fatalf("e-mail pour une fiche sans adresse : %d → %d", before, after)
	}
}

func TestRecipientRegistrations(t *testing.T) {
	f := newTrackFixture(t)
	ev := evFixture{msFixture: f.msFixture}.publishedEvent(t, eventBody("Stage", paidEvent("35")))
	const email = "suivi@exemple.fr"
	leo := trackedMember(t, f.msFixture, "Léo", email)
	zoe := trackedMember(t, f.msFixture, "Zoé", email)
	regLeo := expectJSON[api.AdminRegistration](t, f.admin.Do("POST", eventPath(ev.ID, "/registrations"), map[string]any{"memberId": leo}), http.StatusCreated)
	regZoe := expectJSON[api.AdminRegistration](t, f.admin.Do("POST", eventPath(ev.ID, "/registrations"), map[string]any{"memberId": zoe}), http.StatusCreated)

	expectJSON[api.AdminRegistration](t, f.admin.Do("POST", adminRegistrationPath(regLeo.ID, "/payment"), map[string]any{
		"method": "cheque", "receivedOn": msToday,
	}), http.StatusOK)
	if _, link := lastMailTo(t, f.msFixture, email, "Paiement reçu : Stage"); !strings.HasPrefix(link, "http://frigg.test/suivi#jeton=") {
		t.Fatalf("e-mail de paiement : %q", link)
	}
	expectJSON[api.AdminRegistration](t, f.admin.Do("POST", adminRegistrationPath(regZoe.ID, "/cancel"), nil), http.StatusOK)
	if _, link := lastMailTo(t, f.msFixture, email, "Inscription annulée : Stage"); !strings.HasPrefix(link, "http://frigg.test/suivi#jeton=") {
		t.Fatalf("e-mail de retrait : %q", link)
	}

	// Un e-mail groupé par adresse porte un seul lien (012 FR-031).
	before := count(t, f.env, `SELECT count(*) FROM tracking_links WHERE email = ?`, email)
	expectAdminEvent(t, f.admin.Do("PUT", eventPath(ev.ID, ""), eventBody("Stage", paidEvent("35"), func(b map[string]any) {
		b["location"] = "Gymnase"
	})), http.StatusOK)
	if _, link := lastMailTo(t, f.msFixture, email, "Changement : Stage"); !strings.HasPrefix(link, "http://frigg.test/suivi#jeton=") {
		t.Fatalf("e-mail de changement : %q", link)
	}
	if after := count(t, f.env, `SELECT count(*) FROM tracking_links WHERE email = ?`, email); after != before+1 {
		t.Fatalf("liens émis pour un e-mail : %d", after-before)
	}
}

// Compte désactivé : aucun e-mail d'adhésion (006 FR-023), e-mail
// d'inscription à l'adresse de la fiche sans lien personnel (012).
func TestRecipientDisabledAccount(t *testing.T) {
	f := newTrackFixture(t)
	ev := evFixture{msFixture: f.msFixture}.publishedEvent(t, eventBody("Stage", paidEvent("35")))
	email := f.env.Seed(auth.RoleMember, true)
	c := f.env.LoggedIn(email)
	const fiche = "fiche@exemple.fr"
	m := createMine(t, c, memberBody("Léo", "Martin", "2010-04-12", "email", fiche, "confirmDuplicate", true))
	ms := request(t, c, m.ID, f.season.ID, f.young.ID, "first")
	reg := register(t, c, ev.ID, http.StatusCreated, m.ID).CreatedRegistrationIDs[0]
	expectAccount(t, f.admin.Do("POST", accountPath(accountID(t, f.env, email), "/disable"), nil))

	before := len(f.env.Mails(email)) + len(f.env.Mails(fiche))
	validate(t, f.admin, ms.ID, f.young.ID, "first")
	if after := len(f.env.Mails(email)) + len(f.env.Mails(fiche)); after != before {
		t.Fatalf("e-mail d'adhésion pour un compte désactivé : %d", after-before)
	}
	expectJSON[api.AdminRegistration](t, f.admin.Do("POST", adminRegistrationPath(reg, "/cancel"), nil), http.StatusOK)
	text, link := lastMailTo(t, f.msFixture, fiche, "Inscription annulée : Stage")
	if link != "" || strings.Contains(text, "jeton") {
		t.Fatalf("lien personnel pour une fiche d'un compte désactivé : %s", text)
	}
	if n := count(t, f.env, `SELECT count(*) FROM tracking_links`); n != 0 {
		t.Fatalf("liens personnels : %d", n)
	}
}
