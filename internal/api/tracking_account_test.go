package api_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Création d'un compte depuis le suivi par lien (016 US4, research.md › R12).

func createTrackingAccount(c *apitest.Client, self any) apitest.Response {
	return c.Do("POST", "/api/tracking/account", map[string]any{"password": apitest.Password, "selfMemberId": self})
}

// selfOf renvoie, par fiche, son compte et son marquage « Vous ».
func selfOf(t *testing.T, env *apitest.Env, id int64) (sql.NullInt64, int64) {
	t.Helper()
	m, err := env.Store.Read.GetMember(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return m.AccountID, m.IsSelf
}

func TestCreateTrackingAccount(t *testing.T) {
	f := newTrackFixture(t)
	email := "claire@exemple.fr"
	claire := trackedMember(t, f.msFixture, "Claire", email)
	pending := pendingFor(t, f.msFixture, claire)
	oldToken := issueLink(t, f.env, email)
	c := f.env.TrackingClient(email)

	res := createTrackingAccount(c, nil)
	if res.Status != http.StatusCreated {
		t.Fatalf("création : %d %s", res.Status, res.Body)
	}
	var session api.Session
	res.JSON(t, &session)
	if session.Account.Email != email || session.Account.Role != auth.RoleMember || !session.Account.EmailVerified {
		t.Fatalf("session : %+v", session)
	}
	if res := c.Do("GET", "/api/session", nil); res.Status != http.StatusOK {
		t.Fatalf("session de compte : %d", res.Status)
	}
	// L'accès par lien est retiré de la session.
	expectProblem(t, c.Do("GET", "/api/tracking", nil), http.StatusUnauthorized, api.MsgLinkInvalid)
	// Fiche unique : rattachée et marquée « Vous » d'office.
	if account, self := selfOf(t, f.env, claire); account.Int64 != session.Account.ID || self != 1 {
		t.Fatalf("fiche : compte %v, vous %d", account, self)
	}
	if res := c.Do("GET", myMembershipPath(pending), nil); res.Status != http.StatusOK {
		t.Fatalf("demande dans l'espace membre : %d", res.Status)
	}
	// Tous les liens de l'adresse sont révoqués.
	if n := count(t, f.env, `SELECT count(*) FROM tracking_links WHERE email = ? AND revoked_at IS NULL`, email); n != 0 {
		t.Fatalf("liens actifs : %d", n)
	}
	expectProblem(t, f.env.Client().Do("POST", "/api/public/tracking/open", map[string]string{"token": oldToken}), http.StatusGone, api.MsgLinkInvalid)

	n := 0
	for _, line := range strings.Split(f.env.Logs.String(), "\n") {
		if strings.Contains(line, `"action":"tracking_account_created"`) {
			n++
			if !strings.Contains(line, fmt.Sprintf(`"account_id":%d`, session.Account.ID)) || strings.Contains(line, "@") || strings.Contains(line, "Claire") {
				t.Errorf("journal : %s", line)
			}
		}
	}
	if n != 1 {
		t.Fatalf("journal tracking_account_created : %d", n)
	}
}

func TestCreateTrackingAccountSelf(t *testing.T) {
	f := newTrackFixture(t)
	email := "famille@exemple.fr"
	claire := trackedMember(t, f.msFixture, "Claire", email)
	leo := trackedMember(t, f.msFixture, "Léo", "Famille@Exemple.fr")
	zoe := trackedMember(t, f.msFixture, "Zoé", "autre@exemple.fr")
	c := f.env.TrackingClient(email)

	// Fiche hors périmètre : 422, rien n'est créé.
	res := createTrackingAccount(c, zoe)
	if res.Status != http.StatusUnprocessableEntity || !strings.Contains(res.Body, api.MsgSelfMemberOutOfScope) {
		t.Fatalf("fiche hors périmètre : %d %s", res.Status, res.Body)
	}
	if n := count(t, f.env, `SELECT count(*) FROM accounts WHERE email = ?`, email); n != 0 {
		t.Fatal("compte créé malgré le refus")
	}

	if res := createTrackingAccount(c, leo); res.Status != http.StatusCreated {
		t.Fatalf("création : %d %s", res.Status, res.Body)
	}
	if _, self := selfOf(t, f.env, leo); self != 1 {
		t.Fatal("fiche choisie non marquée « Vous »")
	}
	if account, self := selfOf(t, f.env, claire); !account.Valid || self != 0 {
		t.Fatalf("autre fiche : compte %v, vous %d", account, self)
	}
	if account, _ := selfOf(t, f.env, zoe); account.Valid {
		t.Fatal("fiche d'une autre adresse rattachée")
	}

	// Plusieurs fiches, aucune désignée.
	g := newTrackFixture(t)
	a := trackedMember(t, g.msFixture, "Claire", email)
	b := trackedMember(t, g.msFixture, "Léo", email)
	if res := createTrackingAccount(g.env.TrackingClient(email), nil); res.Status != http.StatusCreated {
		t.Fatalf("création sans « Vous » : %d %s", res.Status, res.Body)
	}
	for _, id := range []int64{a, b} {
		if account, self := selfOf(t, g.env, id); !account.Valid || self != 0 {
			t.Fatalf("fiche %d : compte %v, vous %d", id, account, self)
		}
	}
}

func TestCreateTrackingAccountRefused(t *testing.T) {
	f := newTrackFixture(t)
	email := "leo@exemple.fr"
	trackedMember(t, f.msFixture, "Léo", email)
	c := f.env.TrackingClient(email)

	// Mot de passe non conforme (001 FR-008).
	res := c.Do("POST", "/api/tracking/account", map[string]any{"password": "court"})
	if res.Status != http.StatusUnprocessableEntity || !strings.Contains(res.Body, "body.password") {
		t.Fatalf("mot de passe : %d %s", res.Status, res.Body)
	}

	// Un compte existe déjà pour l'adresse.
	account := f.env.Seed(auth.RoleMember, true)
	trackedMember(t, f.msFixture, "Paul", account)
	expectProblem(t, createTrackingAccount(f.env.TrackingClient(account), nil), http.StatusConflict, api.MsgTrackingAccountExists)

	// 21 fiches suivies.
	many := "nombreux@exemple.fr"
	for i := range 21 {
		if _, err := f.env.Store.Write.CreateMember(context.Background(), sqlc.CreateMemberParams{
			FirstName: fmt.Sprintf("Enfant%d", i), LastName: "Nombreux", BirthDate: "2010-01-01",
			Email: sql.NullString{String: many, Valid: true}, Now: "2026-11-01T10:00:00Z",
		}); err != nil {
			t.Fatal(err)
		}
	}
	mc := f.env.TrackingClient(many)
	if got := expectJSON[api.Tracking](t, mc.Do("GET", "/api/tracking", nil), http.StatusOK); got.CanCreateAccount {
		t.Fatal("création proposée au-delà de 20 fiches")
	}
	expectProblem(t, createTrackingAccount(mc, nil), http.StatusConflict, api.MsgTrackingTooManyMembers)
	if n := count(t, f.env, `SELECT count(*) FROM accounts WHERE email IN (?, ?)`, email, many); n != 0 {
		t.Fatalf("comptes créés : %d", n)
	}
}
