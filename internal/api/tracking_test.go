package api_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/public"
)

// Espace de suivi par lien personnel (016, US2) : lecture, annulation,
// annonce d'un moyen de paiement, demande de lien et limites.

// newTrackFixture : jeu de données des adhésions derrière un proxy de
// confiance, moyens de paiement manuels activés.
func newTrackFixture(t *testing.T) *pubFixture {
	t.Helper()
	f := newPubFixture(t, false)
	setPayment(t, f.admin, nil)
	return f
}

// trackedMember crée par l'administration une fiche sans compte d'adresse email.
func trackedMember(t *testing.T, f msFixture, first, email string) int64 {
	t.Helper()
	return expectMember(t, f.admin.Do("POST", "/api/admin/members", memberBody(first, "Martin", "1990-05-01",
		"email", email, "accountId", nil, "confirmDuplicate", true)), http.StatusCreated).ID
}

// pendingFor insère une demande d'adhésion en attente (tarif « Adulte »).
func pendingFor(t *testing.T, f msFixture, memberID int64) int64 {
	t.Helper()
	id, err := f.env.Store.Write.CreateMembership(context.Background(), sqlc.CreateMembershipParams{
		MemberID: memberID, SeasonID: f.season.ID, FeeID: f.adult.ID, Status: "pending",
		DeclaredKind: sql.NullString{String: "first", Valid: true}, Now: "2026-11-01T10:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// validatedFor enregistre par l'administration une adhésion validée.
func validatedFor(t *testing.T, f msFixture, memberID, feeID int64) api.AdminMembership {
	t.Helper()
	return expectAdminMembership(t, f.admin.Do("POST", "/api/admin/memberships", map[string]any{
		"memberId": memberID, "seasonId": f.season.ID, "feeId": feeID, "kind": "first",
	}), http.StatusCreated)
}

func trackingPath(id int64, suffix string) string {
	return fmt.Sprintf("/api/tracking/memberships/%d%s", id, suffix)
}

func expectTracked(t *testing.T, res apitest.Response, status int) api.TrackedMembership {
	t.Helper()
	return expectJSON[api.TrackedMembership](t, res, status)
}

// issueLink crée un lien personnel valide pour email et renvoie son jeton.
func issueLink(t *testing.T, env *apitest.Env, email string) string {
	t.Helper()
	raw, hash := auth.NewToken()
	now := time.Now().UTC()
	if _, err := env.Store.Write.CreateTrackingLink(context.Background(), sqlc.CreateTrackingLinkParams{
		Email: email, TokenHash: hash,
		ExpiresAt: now.Add(public.LinkTTL).Format("2006-01-02T15:04:05Z"),
		CreatedAt: now.Format("2006-01-02T15:04:05Z"),
	}); err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestGetTracking(t *testing.T) {
	f := newTrackFixture(t)
	ev := evFixture{msFixture: f.msFixture}.publishedEvent(t, eventBody("Stage", paidEvent("35")))
	claire := trackedMember(t, f.msFixture, "Claire", "famille@exemple.fr")
	leo := trackedMember(t, f.msFixture, "Léo", "Famille@Exemple.fr")
	zoe := trackedMember(t, f.msFixture, "Zoé", "autre@exemple.fr")
	pending := pendingFor(t, f.msFixture, claire)
	validated := validatedFor(t, f.msFixture, leo, f.young.ID)
	pendingFor(t, f.msFixture, zoe)
	reg := expectJSON[api.AdminRegistration](t, f.admin.Do("POST", eventPath(ev.ID, "/registrations"),
		map[string]any{"memberId": leo}), http.StatusCreated)

	c := f.env.TrackingClient("famille@exemple.fr")
	got := expectJSON[api.Tracking](t, c.Do("GET", "/api/tracking", nil), http.StatusOK)
	if got.Email != "famille@exemple.fr" || len(got.Members) != 2 || !got.CanCreateAccount {
		t.Fatalf("suivi : %+v", got)
	}
	byKey := map[string]api.TrackedItem{}
	for _, it := range got.Items {
		byKey[fmt.Sprintf("%s-%d", it.Kind, it.ID)] = it
		if it.Member.ID == zoe {
			t.Fatalf("fiche d'une autre adresse : %+v", it)
		}
	}
	if len(got.Items) != 3 {
		t.Fatalf("éléments : %+v", got.Items)
	}
	p := byKey[fmt.Sprintf("membership-%d", pending)]
	if p.Status != "pending" || !p.Actions.Cancel || p.Actions.Pay || p.Member.FirstName != "Claire" ||
		!strings.HasPrefix(p.Label, "Adhésion 2026-2027 — Adulte") {
		t.Fatalf("demande en attente : %+v", p)
	}
	v := byKey[fmt.Sprintf("membership-%d", validated.ID)]
	if v.Status != "validated" || v.Actions.Cancel || !v.Actions.Pay || v.AmountCents == nil || *v.AmountCents != 3000 ||
		v.PaymentStatus == nil || *v.PaymentStatus != "due" || v.Reference == nil {
		t.Fatalf("adhésion validée : %+v", v)
	}
	r := byKey[fmt.Sprintf("registration-%d", reg.ID)]
	if r.Status != "confirmed" || r.Label != "Stage" || !r.Actions.Pay || r.AmountCents == nil || *r.AmountCents != 3500 {
		t.Fatalf("inscription : %+v", r)
	}

	// Une adresse qui a un compte ne peut pas en créer un autre.
	account := f.env.Seed(auth.RoleMember, true)
	trackedMember(t, f.msFixture, "Paul", account)
	if got := expectJSON[api.Tracking](t, f.env.TrackingClient(account).Do("GET", "/api/tracking", nil), http.StatusOK); got.CanCreateAccount || len(got.Members) != 1 {
		t.Fatalf("adresse d'un compte : %+v", got)
	}
}

func TestCancelTrackingMembership(t *testing.T) {
	f := newTrackFixture(t)
	claire := trackedMember(t, f.msFixture, "Claire", "claire@exemple.fr")
	leo := trackedMember(t, f.msFixture, "Léo", "claire@exemple.fr")
	pending := pendingFor(t, f.msFixture, claire)
	validated := validatedFor(t, f.msFixture, leo, f.young.ID)
	c := f.env.TrackingClient("claire@exemple.fr")

	got := expectTracked(t, c.Do("POST", trackingPath(pending, "/cancel"), nil), http.StatusOK)
	if got.Status != "cancelled" || got.CancelledAt == nil {
		t.Fatalf("annulation : %+v", got)
	}
	if n := count(t, f.env, `SELECT count(*) FROM memberships WHERE id = ? AND cancelled_by IS NULL AND status = 'cancelled'`, pending); n != 1 {
		t.Fatal("auteur de l'annulation renseigné")
	}
	expectProblem(t, c.Do("POST", trackingPath(pending, "/cancel"), nil), http.StatusConflict, api.MsgMembershipChanged)
	expectProblem(t, c.Do("POST", trackingPath(validated.ID, "/cancel"), nil), http.StatusConflict, api.MsgMembershipChanged)
	n := 0
	for _, line := range strings.Split(f.env.Logs.String(), "\n") {
		if strings.Contains(line, `"action":"membership.cancel"`) && strings.Contains(line, `"space":"tracking"`) {
			n++
			if strings.Contains(line, "@") || strings.Contains(line, "Claire") {
				t.Errorf("donnée personnelle journalisée : %s", line)
			}
		}
	}
	if n != 1 {
		t.Fatalf("journal membership.cancel : %d", n)
	}
}

func TestAnnounceTrackingMembershipPayment(t *testing.T) {
	f := newTrackFixture(t)
	leo := trackedMember(t, f.msFixture, "Léo", "leo@exemple.fr")
	ms := validatedFor(t, f.msFixture, leo, f.young.ID)
	c := f.env.TrackingClient("leo@exemple.fr")

	got := expectTracked(t, c.Do("GET", trackingPath(ms.ID, ""), nil), http.StatusOK)
	if !got.Actions.Pay || len(got.PaymentMethods) != 3 || got.Payment == nil || got.Payment.Status != "due" {
		t.Fatalf("page de paiement : %+v", got)
	}
	got = expectTracked(t, c.Do("PUT", trackingPath(ms.ID, "/payment"), map[string]any{"method": "cheque"}), http.StatusOK)
	if got.Payment.Status != "announced" || *got.Payment.AnnouncedMethod != "cheque" {
		t.Fatalf("chèque : %+v", got.Payment)
	}
	got = expectTracked(t, c.Do("PUT", trackingPath(ms.ID, "/payment"), map[string]any{"method": "transfer"}), http.StatusOK)
	if got.Payment.Status != "announced" || *got.Payment.AnnouncedMethod != "transfer" {
		t.Fatalf("virement : %+v", got.Payment)
	}
	res := c.Do("PUT", trackingPath(ms.ID, "/payment"), map[string]any{"method": "card"})
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("carte annoncée : %d %s", res.Status, res.Body)
	}
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(ms.ID, "/payment"), map[string]any{
		"method": "transfer", "receivedOn": msToday,
	}), http.StatusOK)
	expectProblem(t, c.Do("PUT", trackingPath(ms.ID, "/payment"), map[string]any{"method": "cash"}), http.StatusConflict, api.MsgMembershipChanged)
}

func TestRequestTrackingLink(t *testing.T) {
	f := newTrackFixture(t)
	account := f.env.Seed(auth.RoleMember, true)
	trackedMember(t, f.msFixture, "Léo", "leo@exemple.fr")
	anonymous := f.env.Client()
	ask := func(ip, email string) apitest.Response {
		return anonymous.Do("POST", "/api/public/tracking/link-request", map[string]string{"email": email}, "X-Forwarded-For", ip)
	}
	for _, email := range []string{"inconnu@exemple.fr", account, "LEO@exemple.fr"} {
		expectAccepted(t, ask(f.nextIP(), email))
	}
	if n := len(f.env.Mails(account)) + len(f.env.Mails("inconnu@exemple.fr")); n != 0 {
		t.Fatalf("e-mails vers une adresse non suivie : %d", n)
	}
	mails := f.env.Mails("leo@exemple.fr")
	if len(mails) != 1 || mails[0].Subject != "Votre lien de suivi" || !strings.Contains(mails[0].Link, "http://frigg.test/suivi#jeton=") {
		t.Fatalf("e-mail de lien : %+v", mails)
	}
	res := ask(f.nextIP(), "pas-une-adresse")
	if res.Status != http.StatusUnprocessableEntity || !strings.Contains(res.Body, "body.email") {
		t.Fatalf("adresse mal formée : %d %s", res.Status, res.Body)
	}

	// Limite par IP : 10 demandes par heure.
	ip := f.nextIP()
	for i := range 10 {
		if res := ask(ip, fmt.Sprintf("x%d@exemple.fr", i)); res.Status != http.StatusAccepted {
			t.Fatalf("demande %d : %d", i, res.Status)
		}
	}
	if res := ask(ip, "y@exemple.fr"); res.Status != http.StatusTooManyRequests {
		t.Fatalf("11e demande : %d", res.Status)
	}
	if publicLogs(t, f.env, "tracking_link_request") != 13 {
		t.Fatalf("journal tracking_link_request : %d", publicLogs(t, f.env, "tracking_link_request"))
	}
}

func TestCloseTracking(t *testing.T) {
	f := newTrackFixture(t)
	email := f.env.Seed(auth.RoleMember, true)
	c := f.env.LoggedIn(email)
	trackedMember(t, f.msFixture, "Léo", "leo@exemple.fr")
	if res := c.Do("POST", "/api/public/tracking/open", map[string]string{"token": issueLink(t, f.env, "leo@exemple.fr")}); res.Status != http.StatusNoContent {
		t.Fatalf("ouverture : %d", res.Status)
	}
	expectJSON[api.Tracking](t, c.Do("GET", "/api/tracking", nil), http.StatusOK)
	if res := c.Do("POST", "/api/tracking/close", nil); res.Status != http.StatusNoContent {
		t.Fatalf("fermeture : %d", res.Status)
	}
	expectProblem(t, c.Do("GET", "/api/tracking", nil), http.StatusUnauthorized, api.MsgLinkInvalid)
	if res := c.Do("GET", "/api/session", nil); res.Status != http.StatusOK {
		t.Fatalf("session de compte fermée : %d", res.Status)
	}
}

// Limites par adresse (R9) : un lien qui a fui reste borné, même réparti sur
// plusieurs IP.
func TestTrackingEmailLimits(t *testing.T) {
	f := newTrackFixture(t)
	trackedMember(t, f.msFixture, "Léo", "leo@exemple.fr")
	c := f.env.TrackingClient("leo@exemple.fr")
	ips := []string{f.nextIP(), f.nextIP()}
	for i := range 120 {
		if res := c.Do("GET", "/api/tracking", nil, "X-Forwarded-For", ips[i%2]); res.Status != http.StatusOK {
			t.Fatalf("appel %d : %d", i, res.Status)
		}
	}
	if res := c.Do("GET", "/api/tracking", nil, "X-Forwarded-For", f.nextIP()); res.Status != http.StatusTooManyRequests {
		t.Fatalf("121e appel : %d", res.Status)
	}

	token := issueLink(t, f.env, "zoe@exemple.fr")
	anonymous := f.env.Client()
	for i := range 30 {
		if res := anonymous.Do("POST", "/api/public/tracking/open", map[string]string{"token": token}, "X-Forwarded-For", f.nextIP()); res.Status != http.StatusNoContent {
			t.Fatalf("ouverture %d : %d", i, res.Status)
		}
	}
	if res := anonymous.Do("POST", "/api/public/tracking/open", map[string]string{"token": token}, "X-Forwarded-For", f.nextIP()); res.Status != http.StatusTooManyRequests {
		t.Fatalf("31e ouverture : %d", res.Status)
	}
}

// Aucune opération par lien ne modifie l'identité, les coordonnées ni les
// champs personnalisés (FR-016).
func TestTrackingScope(t *testing.T) {
	f := newTrackFixture(t)
	leo := trackedMember(t, f.msFixture, "Léo", "leo@exemple.fr")
	ms := validatedFor(t, f.msFixture, leo, f.young.ID)
	for path, item := range f.env.API.OpenAPI().Paths {
		for method, op := range map[string]*hOp{"POST": wrap(item.Post), "PUT": wrap(item.Put), "PATCH": wrap(item.Patch), "DELETE": wrap(item.Delete)} {
			if op == nil {
				continue
			}
			if access, _ := api.AccessOf(op.Operation); access == auth.Link &&
				(strings.Contains(path, "/members/") || strings.HasSuffix(path, "/members") ||
					strings.Contains(path, "/fields") || method == "PATCH") {
				t.Errorf("%s %s : opération par lien sur une fiche ou des champs", method, path)
			}
		}
	}
	before, err := f.env.Store.Read.GetMember(context.Background(), leo)
	if err != nil {
		t.Fatal(err)
	}
	c := f.env.TrackingClient("leo@exemple.fr")
	// Une propriété inattendue est refusée ; l'annonce seule est acceptée.
	if res := c.Do("PUT", trackingPath(ms.ID, "/payment"), map[string]any{"method": "cheque", "firstName": "Pirate"}); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("propriété inattendue : %d %s", res.Status, res.Body)
	}
	expectTracked(t, c.Do("PUT", trackingPath(ms.ID, "/payment"), map[string]any{"method": "cheque"}), http.StatusOK)
	expectTracked(t, c.Do("GET", trackingPath(ms.ID, ""), nil), http.StatusOK)
	after, err := f.env.Store.Read.GetMember(context.Background(), leo)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("fiche modifiée : %+v → %+v", before, after)
	}
}

func trackingRegistrationPath(id int64, suffix string) string {
	return fmt.Sprintf("/api/tracking/registrations/%d%s", id, suffix)
}

func expectTrackedReg(t *testing.T, res apitest.Response, status int) api.TrackedRegistration {
	t.Helper()
	return expectJSON[api.TrackedRegistration](t, res, status)
}

// adminRegisters inscrit une fiche par l'administration.
func adminRegisters(t *testing.T, f msFixture, eventID, memberID int64) int64 {
	t.Helper()
	return expectJSON[api.AdminRegistration](t, f.admin.Do("POST", eventPath(eventID, "/registrations"),
		map[string]any{"memberId": memberID}), http.StatusCreated).ID
}

// Inscriptions suivies par lien (016 US3, scénarios 9 et 10) : lecture,
// annulation selon 012 FR-013, annonce d'un moyen, e-mails de 012 FR-030 à
// l'adresse de la fiche avec un lien personnel.
func TestTrackingRegistration(t *testing.T) {
	f := newTrackFixture(t)
	ev := evFixture{msFixture: f.msFixture}
	paid := ev.publishedEvent(t, eventBody("Stage", paidEvent("35")))
	free := ev.publishedEvent(t, eventBody("Portes ouvertes", paidEvent("0")))
	closing := ev.publishedEvent(t, eventBody("Tournoi", paidEvent("0"), func(b map[string]any) { b["registrationClosesAt"] = "2026-11-10T10:00" }))
	email := "famille@exemple.fr"
	leo := trackedMember(t, f.msFixture, "Léo", email)
	zoe := trackedMember(t, f.msFixture, "Zoé", "autre@exemple.fr")

	// Confirmation d'une inscription d'une fiche sans compte : lien personnel
	// vers l'inscription (016 FR-035, T037).
	paidReg := adminRegisters(t, f.msFixture, paid.ID, leo)
	mails := f.env.Mails(email)
	last := mails[len(mails)-1]
	if last.Subject != "Inscription confirmée : Stage" ||
		!strings.HasPrefix(last.Link, fmt.Sprintf("http://frigg.test/suivi/inscriptions/%d#jeton=", paidReg)) {
		t.Fatalf("confirmation : %+v", last)
	}
	freeReg := adminRegisters(t, f.msFixture, free.ID, leo)
	if last := f.env.Mails(email); !strings.HasPrefix(last[len(last)-1].Link, "http://frigg.test/suivi#jeton=") {
		t.Fatalf("confirmation gratuite : %+v", last[len(last)-1])
	}
	other := adminRegisters(t, f.msFixture, paid.ID, zoe)
	closingReg := adminRegisters(t, f.msFixture, closing.ID, leo)

	c := f.env.TrackingClient(email)
	got := expectTrackedReg(t, c.Do("GET", trackingRegistrationPath(paidReg, ""), nil), http.StatusOK)
	if got.State != "confirmed" || got.AmountCents != 3500 || got.Payment == nil || got.Payment.Status != "due" ||
		!got.Actions.Cancel || !got.Actions.Pay || len(got.PaymentMethods) != 3 || got.Member.FirstName != "Léo" {
		t.Fatalf("inscription suivie : %+v", got)
	}
	// Hors périmètre : 404, comme une inscription inexistante.
	for _, path := range []string{"", "/cancel", "/payment"} {
		method := map[string]string{"": "GET", "/cancel": "POST", "/payment": "PUT"}[path]
		if res := c.Do(method, trackingRegistrationPath(other, path), map[string]any{"method": "cheque"}); res.Status != http.StatusNotFound {
			t.Errorf("%s %s d'une autre adresse : %d", method, path, res.Status)
		}
	}

	// Annonce d'un moyen (012 FR-022) ; la carte n'est pas un moyen manuel.
	got = expectTrackedReg(t, c.Do("PUT", trackingRegistrationPath(paidReg, "/payment"), map[string]any{"method": "cheque"}), http.StatusOK)
	if got.Payment.Status != "announced" || *got.Payment.AnnouncedMethod != "cheque" {
		t.Fatalf("annonce : %+v", got.Payment)
	}
	if res := c.Do("PUT", trackingRegistrationPath(paidReg, "/payment"), map[string]any{"method": "card"}); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("carte annoncée : %d", res.Status)
	}

	// Annulation (012 FR-013) : gratuite, puis payante non payée.
	got = expectTrackedReg(t, c.Do("POST", trackingRegistrationPath(freeReg, "/cancel"), nil), http.StatusOK)
	if got.State != "cancelled" || got.Actions.Cancel {
		t.Fatalf("annulation gratuite : %+v", got)
	}
	expectProblem(t, c.Do("POST", trackingRegistrationPath(freeReg, "/cancel"), nil), http.StatusConflict, api.MsgRegistrationChanged)
	// Payée : refusée.
	if _, err := f.env.Store.WriteDB.Exec(`UPDATE event_registrations SET payment_status = 'paid', paid_method = 'cheque',
		paid_on = '2026-11-01' WHERE id = ?`, paidReg); err != nil {
		t.Fatal(err)
	}
	expectProblem(t, c.Do("POST", trackingRegistrationPath(paidReg, "/cancel"), nil), http.StatusConflict, api.MsgRegistrationPaid)
	if _, err := f.env.Store.WriteDB.Exec(`UPDATE event_registrations SET payment_status = 'announced', paid_method = NULL,
		paid_on = NULL WHERE id = ?`, paidReg); err != nil {
		t.Fatal(err)
	}
	// Hors période : refusée.
	f.env.SetToday("2026-11-15")
	expectProblem(t, c.Do("POST", trackingRegistrationPath(closingReg, "/cancel"), nil), http.StatusConflict, api.MsgRegistrationsClosed)
	f.env.SetToday(msToday)
	got = expectTrackedReg(t, c.Do("POST", trackingRegistrationPath(paidReg, "/cancel"), nil), http.StatusOK)
	if got.State != "cancelled" {
		t.Fatalf("annulation payante non payée : %+v", got)
	}
	var cancelledBy sql.NullInt64
	if err := f.env.Store.ReadDB.QueryRow(`SELECT cancelled_by FROM event_registrations WHERE id = ?`, paidReg).Scan(&cancelledBy); err != nil || cancelledBy.Valid {
		t.Fatalf("auteur de l'annulation : %v %v", cancelledBy, err)
	}
	if n := strings.Count(f.env.Logs.String(), `"action":"registration.cancel","event_id"`); n != 2 {
		t.Errorf("journaux d'annulation : %d", n)
	}
	noLeak(t, f.env, email, "Léo")
}

// Retrait par un administrateur d'une inscription d'une fiche sans compte :
// e-mail de retrait à l'adresse de la fiche, avec un lien personnel
// (016 US3 scénario 10, FR-027).
func TestTrackedRegistrationRemovedByAdmin(t *testing.T) {
	f := newPubFixture(t, false)
	ev := f.publicEvent(t, "Stage", withCapacity(1))
	slug := slugOf(t, ev)
	email := "claire@exemple.fr"
	expectRegOutcome(t, f.confirm(f.env.Client(), f.depositRegistration(t, slug, regBody(email))), "tracking")
	var regID int64
	if err := f.env.Store.ReadDB.QueryRow(`SELECT id FROM event_registrations`).Scan(&regID); err != nil {
		t.Fatal(err)
	}
	if got := getPublicEvent(t, f.env.Client(), slug); got.ClosedReason == nil || *got.ClosedReason != "full" {
		t.Fatalf("complet : %+v", got)
	}
	expectJSON[api.AdminRegistration](t, f.admin.Do("POST", adminRegistrationPath(regID, "/cancel"), nil), http.StatusOK)
	mails := f.env.Mails(email)
	last := mails[len(mails)-1]
	if last.Subject != "Inscription annulée : Stage" || !strings.HasPrefix(last.Link, "http://frigg.test/suivi#jeton=") {
		t.Fatalf("e-mail de retrait : %+v", last)
	}
	// La place est libérée.
	if got := getPublicEvent(t, f.env.Client(), slug); !got.RegistrationOpen || *got.PlacesLeft != 1 {
		t.Fatalf("place libérée : %+v", got)
	}
	// Le lien de l'e-mail ouvre l'inscription annulée.
	c := f.env.Client()
	if res := c.Do("POST", "/api/public/tracking/open", map[string]string{"token": last.Token}); res.Status != http.StatusNoContent {
		t.Fatalf("ouverture du lien : %d", res.Status)
	}
	if got := expectTrackedReg(t, c.Do("GET", trackingRegistrationPath(regID, ""), nil), http.StatusOK); got.State != "cancelled" {
		t.Fatalf("inscription : %+v", got)
	}
}
