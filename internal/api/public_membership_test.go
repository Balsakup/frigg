package api_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/config"
)

// Parcours publics (016, US1) : page d'adhésion publique et dépôt.

// neutralBody est le corps exact de la réponse 202 d'un dépôt (contracts/api.md).
const neutralBody = `{"message":"Si l'adresse est valide, un e-mail vient de vous être envoyé."}`

// pubFixture reprend msFixture (saison 2026-2027 ouverte, tarifs « Adulte »,
// « Jeune », « Bienfaiteur ») derrière un proxy de confiance : chaque
// requête publique peut venir d'une IP distincte (X-Forwarded-For).
type pubFixture struct {
	msFixture
	ip atomic.Int64
}

func newPubFixture(t *testing.T, enabled bool) *pubFixture {
	t.Helper()
	env := apitest.New(t, func(c *config.Config) { c.TrustedProxies = []string{"127.0.0.1", "::1"} })
	env.SetToday(msToday)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	expectSeason(t, admin.Do("POST", seasonPath(s.ID, "/open"), nil), http.StatusOK)
	f := &pubFixture{msFixture: msFixture{env: env, admin: admin, season: s}}
	f.adult = expectFee(t, admin.Do("POST", feesPath(s.ID), map[string]any{
		"label": "Adulte", "amount": "60", "hasRenewalAmount": true, "renewalAmount": "40",
	}), http.StatusCreated)
	f.young = createFee(t, admin, s.ID, map[string]string{"label": "Jeune", "amount": "30"})
	f.free = createFee(t, admin, s.ID, map[string]string{"label": "Bienfaiteur", "amount": "0"})
	if enabled {
		f.setPublic(t, true)
	}
	return f
}

// setPublic active ou désactive l'accueil des demandes publiques.
func (f *pubFixture) setPublic(t *testing.T, on bool) {
	t.Helper()
	if res := putGeneral(f.admin, map[string]any{"name": "Club de Frigg", "publicMembershipEnabled": on}); res.Status != http.StatusOK {
		t.Fatalf("paramètres généraux : %d %s", res.Status, res.Body)
	}
}

// nextIP renvoie une IP de documentation inédite.
func (f *pubFixture) nextIP() string {
	n := f.ip.Add(1)
	return fmt.Sprintf("198.51.%d.%d", n/250, n%250+1)
}

// publicBody renvoie un dépôt valide pour l'adresse email, complété par extra.
func (f *pubFixture) publicBody(email string, extra ...any) map[string]any {
	b := map[string]any{
		"firstName": "Claire", "lastName": "Martin", "birthDate": "1985-03-02", "email": email,
		"consent": true, "seasonId": f.season.ID, "feeId": f.adult.ID, "declaredKind": "first",
	}
	for i := 0; i+1 < len(extra); i += 2 {
		b[extra[i].(string)] = extra[i+1]
	}
	return b
}

// submit dépose une demande d'adhésion publique depuis l'IP ip.
func submit(c *apitest.Client, ip string, body map[string]any) apitest.Response {
	return c.Do("POST", "/api/public/membership", body, "X-Forwarded-For", ip)
}

// expectAccepted vérifie la réponse 202 neutre.
func expectAccepted(t *testing.T, res apitest.Response) {
	t.Helper()
	if res.Status != http.StatusAccepted || strings.TrimSpace(res.Body) != neutralBody {
		t.Fatalf("dépôt : %d %s", res.Status, res.Body)
	}
}

// deposit dépose une demande et renvoie le jeton de confirmation reçu par
// e-mail.
func (f *pubFixture) deposit(t *testing.T, body map[string]any) string {
	t.Helper()
	email := auth.NormalizeEmail(body["email"].(string))
	before := len(f.env.Mails(email))
	expectAccepted(t, submit(f.env.Client(), f.nextIP(), body))
	mails := f.env.Mails(email)
	if len(mails) != before+1 || mails[len(mails)-1].Token == "" {
		t.Fatalf("e-mail de confirmation absent : %+v", mails)
	}
	return mails[len(mails)-1].Token
}

// confirm confirme un dépôt depuis le client c.
func (f *pubFixture) confirm(c *apitest.Client, token string) apitest.Response {
	return c.Do("POST", "/api/public/submissions/confirm", map[string]string{"token": token}, "X-Forwarded-For", f.nextIP())
}

func publicPage(t *testing.T, c *apitest.Client) (api.PublicMembershipPage, string) {
	t.Helper()
	res := c.Do("GET", "/api/public/membership", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("page publique : %d %s", res.Status, res.Body)
	}
	var p api.PublicMembershipPage
	res.JSON(t, &p)
	return p, res.Body
}

// count renvoie le résultat d'une requête de comptage.
func count(t *testing.T, env *apitest.Env, query string, args ...any) int {
	t.Helper()
	var n int
	if err := env.Store.ReadDB.QueryRowContext(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPublicMembershipPage(t *testing.T) {
	f := newPubFixture(t, false)
	anonymous := f.env.Client()

	// Accueil fermé par défaut (FR-004, FR-005).
	if p, _ := publicPage(t, anonymous); p.Open || len(p.Seasons) != 0 || len(p.MemberFields) != 0 {
		t.Fatalf("accueil fermé : %+v", p)
	}
	res := submit(anonymous, f.nextIP(), f.publicBody("claire@exemple.fr"))
	if res.Status != http.StatusConflict || !strings.Contains(res.Body, "Les demandes d'adhésion ne sont pas ouvertes.") {
		t.Fatalf("dépôt sur accueil fermé : %d %s", res.Status, res.Body)
	}
	if n := count(t, f.env, `SELECT count(*) FROM public_submissions`); n != 0 {
		t.Fatalf("%d dépôts enregistrés", n)
	}

	// Accueil ouvert : saison ouverte avec tarif seulement ; champs « Rempli
	// par le membre » actifs seulement (FR-002).
	f.setPublic(t, true)
	createSeason(t, f.admin, "2027-09-01", "2028-08-31") // brouillon
	editable := createField(t, f.admin, fieldBody("member", "Club précédent", "short_text", "visibility", "editable", "required", true))
	createField(t, f.admin, fieldBody("member", "Remarque du bureau", "short_text"))
	createField(t, f.admin, fieldBody("member", "Groupe affiché", "short_text", "visibility", "visible"))
	archived := createField(t, f.admin, fieldBody("member", "Ancien champ", "short_text", "visibility", "editable"))
	expectField(t, f.admin.Do("POST", customFieldPath(archived.ID, "/archive"), nil), http.StatusOK)
	size := createField(t, f.admin, fieldBody("membership", "Taille", "choice", "choices", []string{"S", "M"}, "visibility", "editable"))
	createField(t, f.admin, fieldBody("membership", "Licence payée", "bool"))

	p, raw := publicPage(t, anonymous)
	if !p.Open || len(p.Seasons) != 1 || p.Seasons[0].ID != f.season.ID || len(p.Seasons[0].Fees) != 3 {
		t.Fatalf("saisons : %+v", p.Seasons)
	}
	if fee := p.Seasons[0].Fees[0]; fee.Label != "Adulte" || fee.Amounts.First != 6000 || fee.Amounts.Renewal != 4000 {
		t.Fatalf("tarif : %+v", fee)
	}
	if len(p.MemberFields) != 1 || p.MemberFields[0].ID != editable.ID || !p.MemberFields[0].Required {
		t.Fatalf("champs de membre : %+v", p.MemberFields)
	}
	if len(p.MembershipFields) != 1 || p.MembershipFields[0].ID != size.ID {
		t.Fatalf("champs d'adhésion : %+v", p.MembershipFields)
	}
	for _, leak := range []string{"Remarque du bureau", "Groupe affiché", "Ancien champ", "Licence payée", "2027-2028"} {
		if strings.Contains(raw, leak) {
			t.Errorf("%q transmis par la page publique", leak)
		}
	}
}

func TestSubmitPublicMembership(t *testing.T) {
	f := newPubFixture(t, true)
	anonymous := f.env.Client()

	// Dépôt valide : réponse neutre, un dépôt, un e-mail, ni fiche ni adhésion.
	email := "claire@exemple.fr"
	expectAccepted(t, submit(anonymous, f.nextIP(), f.publicBody("  Claire@Exemple.FR ", "createAccount", true)))
	if n := count(t, f.env, `SELECT count(*) FROM public_submissions WHERE email = ? AND create_account = 1`, email); n != 1 {
		t.Fatalf("dépôts : %d", n)
	}
	mails := f.env.Mails(email)
	if len(mails) != 1 || mails[0].Subject != "Confirmez votre demande" ||
		!strings.Contains(mails[0].Link, "http://frigg.test/confirmer-demande#jeton=") ||
		!strings.Contains(mails[0].Text, "adhésion de Claire Martin pour la saison 2026-2027") {
		t.Fatalf("e-mail : %+v", mails)
	}
	if n := count(t, f.env, `SELECT count(*) FROM members`) + count(t, f.env, `SELECT count(*) FROM memberships`); n != 0 {
		t.Fatalf("fiche ou adhésion créée avant confirmation : %d", n)
	}
	if n := publicLogs(t, f.env, "public_submit"); n != 1 {
		t.Errorf("journal public_submit : %d", n)
	}

	// Saisie invalide : 422, sans dépôt ni e-mail.
	other := createSeason(t, f.admin, "2027-09-01", "2028-08-31")
	expectSeason(t, f.admin.Do("POST", seasonPath(other.ID, "/open"), nil), http.StatusOK)
	otherFee := createFee(t, f.admin, other.ID, map[string]string{"label": "Adulte", "amount": "70"})
	expectFieldError(t, submit(anonymous, f.nextIP(), f.publicBody("")), "body.email", "L'adresse électronique est obligatoire.")
	expectFieldError(t, submit(anonymous, f.nextIP(), f.publicBody("x@exemple.fr", "consent", false)), "body.consent", "Vous devez accepter le traitement de vos données.")
	expectFieldError(t, submit(anonymous, f.nextIP(), f.publicBody("x@exemple.fr", "feeId", otherFee.ID)), "body.feeId", "Choisissez un tarif de cette saison.")
	expectFieldError(t, submit(anonymous, f.nextIP(), f.publicBody("x@exemple.fr", "declaredKind", "")), "body.declaredKind", "Choisissez une première adhésion ou un renouvellement.")
	if n := len(f.env.Mails("x@exemple.fr")); n != 0 {
		t.Fatalf("%d e-mails après une saisie invalide", n)
	}

	// Piège rempli : même réponse, ni dépôt ni e-mail.
	expectAccepted(t, submit(anonymous, f.nextIP(), f.publicBody("robot@exemple.fr", "website", "https://spam.example")))
	if n := count(t, f.env, `SELECT count(*) FROM public_submissions WHERE email = 'robot@exemple.fr'`) + len(f.env.Mails("robot@exemple.fr")); n != 0 {
		t.Fatalf("piège : %d", n)
	}
	if !strings.Contains(f.env.Logs.String(), `"action":"public_submit_rejected","kind":"membership","reason":"honeypot"`) {
		t.Errorf("journal du piège absent :\n%s", f.env.Logs.String())
	}
	if n := publicLogs(t, f.env, "public_submit_rejected"); n != 5 {
		t.Errorf("journaux public_submit_rejected : %d, attendu 5", n)
	}
}

// Limites (research.md 016 › R9) : 5 e-mails par adresse et par heure, avec
// une réponse inchangée ; 10 dépôts par IP et par heure.
func TestSubmitPublicMembershipLimits(t *testing.T) {
	f := newPubFixture(t, true)
	anonymous := f.env.Client()
	for i := range 6 {
		expectAccepted(t, submit(anonymous, f.nextIP(), f.publicBody("limite@exemple.fr", "firstName", fmt.Sprintf("Claire%c", 'A'+i))))
	}
	if n := len(f.env.Mails("limite@exemple.fr")); n != 5 {
		t.Fatalf("%d e-mails, attendu 5", n)
	}
	if !strings.Contains(f.env.Logs.String(), `"reason":"rate_email"`) {
		t.Error("journal rate_email absent")
	}

	ip := f.nextIP()
	for i := range 10 {
		expectAccepted(t, submit(anonymous, ip, f.publicBody(fmt.Sprintf("ip%d@exemple.fr", i))))
	}
	res := submit(anonymous, ip, f.publicBody("ip10@exemple.fr"))
	if res.Status != http.StatusTooManyRequests || res.Header.Get("Retry-After") == "" {
		t.Fatalf("11e dépôt : %d %s", res.Status, res.Body)
	}
}

// Protection CSRF (FR-003) : un dépôt public venu d'une origine tierce est
// refusé avant tout traitement, sans dépôt ni e-mail.
func TestSubmitPublicMembershipCrossOrigin(t *testing.T) {
	f := newPubFixture(t, true)
	email := "claire@exemple.fr"
	for _, header := range [][]string{
		{"Origin", "https://autre.exemple"},
		{"Sec-Fetch-Site", "cross-site"},
	} {
		res := f.env.Client().Do("POST", "/api/public/membership", f.publicBody(email),
			"X-Forwarded-For", f.nextIP(), header[0], header[1])
		if res.Status != http.StatusForbidden {
			t.Errorf("%s %s : %d %s, attendu 403", header[0], header[1], res.Status, res.Body)
		}
	}
	if n := count(t, f.env, `SELECT count(*) FROM public_submissions`) + len(f.env.Mails(email)); n != 0 {
		t.Fatalf("dépôt d'origine croisée traité : %d", n)
	}
	expectAccepted(t, f.env.Client().Do("POST", "/api/public/membership", f.publicBody(email),
		"X-Forwarded-For", f.nextIP(), "Sec-Fetch-Site", "same-origin"))
}
