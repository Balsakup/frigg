package api_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

func adminPage(t *testing.T, c *apitest.Client, query string) api.AdminMemberPage {
	t.Helper()
	res := c.Do("GET", "/api/admin/members"+query, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("liste %s : %d %s", query, res.Status, res.Body)
	}
	var p api.AdminMemberPage
	res.JSON(t, &p)
	return p
}

func names(p api.AdminMemberPage) []string {
	out := make([]string, len(p.Items))
	for i, m := range p.Items {
		out[i] = m.LastName + " " + m.FirstName
	}
	return out
}

func TestAdminMembersList(t *testing.T) {
	env := apitest.New(t)
	env.SetToday(membersToday)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	aEmail := env.Seed(auth.RoleMember, true)
	a := env.LoggedIn(aEmail)
	bEmail := env.Seed(auth.RoleMember, true)
	aID, bID := accountID(t, env, aEmail), accountID(t, env, bEmail)

	createMine(t, a, memberBody("Zoé", "Dupont", "2012-01-01"))
	createMine(t, a, memberBody("Élodie", "Dupont", "2014-01-01", "phone", "06 12 34 56 78"))
	insertForAccount(t, env, bID, "Claire", "Martin", "1980-01-01")
	orphan := insertOrphan(t, env, "Paul", "Émery", "1990-01-01")

	p := adminPage(t, admin, "")
	if fmt.Sprint(names(p)) != "[Dupont Élodie Dupont Zoé Émery Paul Martin Claire]" || p.Total != 4 || p.Page != 1 || p.PageSize != 25 {
		t.Fatalf("liste : %v %+v", names(p), p)
	}
	if p.Items[0].Account == nil || p.Items[0].Account.ID != aID || p.Items[0].Account.Email != aEmail || p.Items[2].Account != nil {
		t.Fatalf("rattachements : %+v %+v", p.Items[0].Account, p.Items[2].Account)
	}
	if p := adminPage(t, admin, "?pageSize=2&page=2"); fmt.Sprint(names(p)) != "[Émery Paul Martin Claire]" || p.Total != 4 {
		t.Fatalf("page 2 : %v", names(p))
	}
	if p := adminPage(t, admin, "?page=9"); len(p.Items) != 0 || p.Total != 4 {
		t.Fatalf("page hors limites : %+v", p)
	}
	if p := adminPage(t, admin, "?q=elod"); fmt.Sprint(names(p)) != "[Dupont Élodie]" {
		t.Fatalf("recherche sans accents : %v", names(p))
	}
	if p := adminPage(t, admin, "?q=dupo%20zoe"); fmt.Sprint(names(p)) != "[Dupont Zoé]" {
		t.Fatalf("deux fragments : %v", names(p))
	}
	if p := adminPage(t, admin, "?q=3456"); fmt.Sprint(names(p)) != "[Dupont Élodie]" {
		t.Fatalf("recherche par téléphone : %v", names(p))
	}
	expectFieldError(t, admin.Do("GET", "/api/admin/members?q=ex", nil), "query.q", api.MsgSearchTooShort)
	if p := adminPage(t, admin, "?account=none"); len(p.Items) != 1 || p.Items[0].ID != orphan {
		t.Fatalf("sans compte : %v", names(p))
	}
	if p := adminPage(t, admin, fmt.Sprintf("?account=%d", bID)); fmt.Sprint(names(p)) != "[Martin Claire]" {
		t.Fatalf("par compte : %v", names(p))
	}

	m := expectMember(t, admin.Do("GET", adminMemberPath(orphan), nil), http.StatusOK)
	if m.FirstName != "Paul" || m.Account != nil || m.Age != 36 {
		t.Fatalf("fiche : %+v", m)
	}
	expectProblem(t, admin.Do("GET", adminMemberPath(99999), nil), http.StatusNotFound, api.MsgMemberGone)

	if res := admin.Do("POST", accountPath(bID, "/disable"), nil); res.Status != http.StatusOK {
		t.Fatalf("désactivation : %d %s", res.Status, res.Body)
	}
	p = adminPage(t, admin, fmt.Sprintf("?account=%d", bID))
	if len(p.Items) != 1 || !p.Items[0].Account.Disabled {
		t.Fatalf("compte désactivé : %+v", p.Items)
	}
}

func adminBody(first, last, birth string, account *int64, extra ...any) map[string]any {
	b := memberBody(first, last, birth, extra...)
	b["accountId"] = account
	return b
}

func TestAdminMembersWrite(t *testing.T) {
	env := apitest.New(t)
	env.SetToday(membersToday)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	aEmail, bEmail := env.Seed(auth.RoleMember, true), env.Seed(auth.RoleMember, true)
	a, b := env.LoggedIn(aEmail), env.LoggedIn(bEmail)
	aID, bID := accountID(t, env, aEmail), accountID(t, env, bEmail)

	// Création rattachée, puis sans compte.
	m := expectMember(t, admin.Do("POST", "/api/admin/members", adminBody("Jeanne", "Petit", "2010-01-01", &aID)), http.StatusCreated)
	if m.Account == nil || m.Account.ID != aID || len(myMembers(t, a).Items) != 1 {
		t.Fatalf("création rattachée : %+v", m)
	}
	orphan := expectMember(t, admin.Do("POST", "/api/admin/members", adminBody("Paul", "Émery", "1990-01-01", nil)), http.StatusCreated)
	if orphan.Account != nil {
		t.Fatalf("création sans compte : %+v", orphan)
	}
	missing := int64(99999)
	expectFieldError(t, admin.Do("POST", "/api/admin/members", adminBody("X", "Y", "2000-01-01", &missing)), "body.accountId", api.MsgAccountGone)
	if res := admin.Do("POST", "/api/admin/members", adminBody("X", "Y", "2000-01-01", nil, "isSelf", true)); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("isSelf dans le corps admin : %d, attendu 422", res.Status)
	}
	if res := admin.Do("POST", "/api/admin/members", memberBody("X", "Y", "2000-01-01")); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("accountId absent : %d, attendu 422", res.Status)
	}

	// Fiche « Vous » de A changée de compte : signalement retiré.
	self := createMine(t, a, memberBody("Claire", "Martin", "1980-01-01", "isSelf", true))
	update := func(id int64, expected *int64, body map[string]any) apitest.Response {
		body["expectedAccountId"] = expected
		return admin.Do("PUT", adminMemberPath(id), body)
	}
	kept := expectMember(t, update(self.ID, &aID, adminBody("Claire", "Martin", "1980-01-02", &aID)), http.StatusOK)
	if !kept.IsSelf {
		t.Fatal("« Vous » perdu sans changement de compte")
	}
	moved := expectMember(t, update(self.ID, &aID, adminBody("Claire", "Martin", "1980-01-02", &bID)), http.StatusOK)
	if moved.IsSelf || moved.Account.ID != bID {
		t.Fatalf("changement de compte : %+v", moved)
	}
	if len(myMembers(t, a).Items) != 1 || len(myMembers(t, b).Items) != 1 {
		t.Fatal("la fiche n'a pas changé d'espace")
	}
	detached := expectMember(t, update(self.ID, &bID, adminBody("Claire", "Martin", "1980-01-02", nil)), http.StatusOK)
	if detached.Account != nil || detached.IsSelf {
		t.Fatalf("détachement : %+v", detached)
	}

	// Rattachement modifié entre-temps : refus, fiche inchangée.
	expectProblem(t, update(self.ID, &aID, adminBody("Claire", "Martin", "1980-01-02", &aID)), http.StatusConflict, api.MsgMemberAccountChanged)
	if m := expectMember(t, admin.Do("GET", adminMemberPath(self.ID), nil), http.StatusOK); m.Account != nil {
		t.Fatal("fiche modifiée malgré le conflit")
	}
	if res := admin.Do("PUT", adminMemberPath(self.ID), adminBody("Claire", "Martin", "1980-01-02", nil)); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("expectedAccountId absent : %d, attendu 422", res.Status)
	}

	// Doublon.
	expectFieldError(t, admin.Do("POST", "/api/admin/members", adminBody("jeanne", "PETIT", "2010-01-01", nil)), "body.confirmDuplicate", api.MsgDuplicate)
	expectMember(t, admin.Do("POST", "/api/admin/members", adminBody("jeanne", "PETIT", "2010-01-01", nil, "confirmDuplicate", true)), http.StatusCreated)

	// Plafond du compte cible ; compte désactivé accepté.
	for i := range 20 {
		insertForAccount(t, env, bID, fmt.Sprintf("Enfant%02d", i), "Durand", "2015-01-01")
	}
	expectProblem(t, update(orphan.ID, nil, adminBody("Paul", "Émery", "1990-01-01", &bID)), http.StatusConflict, api.MsgAccountMembersFull)
	if res := admin.Do("POST", accountPath(aID, "/disable"), nil); res.Status != http.StatusOK {
		t.Fatalf("désactivation : %d", res.Status)
	}
	expectMember(t, update(orphan.ID, nil, adminBody("Paul", "Émery", "1990-01-01", &aID)), http.StatusOK)

	// Suppression.
	if res := admin.Do("DELETE", adminMemberPath(m.ID), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", res.Status, res.Body)
	}
	expectProblem(t, admin.Do("GET", adminMemberPath(m.ID), nil), http.StatusNotFound, api.MsgMemberGone)
	expectProblem(t, admin.Do("DELETE", adminMemberPath(m.ID), nil), http.StatusNotFound, api.MsgMemberGone)
	expectProblem(t, update(m.ID, nil, adminBody("Jeanne", "Petit", "2010-01-01", nil)), http.StatusNotFound, api.MsgMemberGone)

	if n := memberLogs(t, env, "member.create", "admin", "Petit", "Jeanne"); n != 3 {
		t.Errorf("%d journaux member.create (admin), attendu 3", n)
	}
	if n := memberLogs(t, env, "member.attach", "admin", "Martin"); n != 4 {
		t.Errorf("%d journaux member.attach, attendu 4", n)
	}
	if n := memberLogs(t, env, "member.delete", "admin", "Petit"); n != 1 {
		t.Errorf("%d journaux member.delete (admin), attendu 1", n)
	}
}

// TestListMembersVolume vérifie SC-004 : 2 000 fiches, recherche et liste
// complète rapides (research.md 005 › R7).
func TestListMembersVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("test de volume")
	}
	env := apitest.New(t)
	env.SetToday(membersToday)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	seedVolume(t, env)

	for _, q := range []string{"?q=dupo&page=3", "?pageSize=100&page=20"} {
		start := time.Now()
		p := adminPage(t, admin, q)
		if d := time.Since(start); d > 500*time.Millisecond {
			t.Errorf("%s : %v, attendu moins de 500 ms", q, d)
		}
		if len(p.Items) == 0 {
			t.Fatalf("%s : page vide", q)
		}
		for i := 1; i < len(p.Items); i++ {
			if names(p)[i-1] > names(p)[i] && p.Items[i-1].LastName == p.Items[i].LastName {
				t.Errorf("%s : ordre %q puis %q", q, names(p)[i-1], names(p)[i])
			}
		}
	}
}

func BenchmarkListMembers(b *testing.B) {
	env := apitest.New(b)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	seedVolume(b, env)
	b.ResetTimer()
	for range b.N {
		admin.Do("GET", "/api/admin/members?page=40", nil)
	}
}

func seedVolume(t testing.TB, env *apitest.Env) {
	t.Helper()
	ctx := context.Background()
	lasts := []string{"Dupont", "Émery", "Martin", "Lefèvre", "Ørsted", "Durand", "Petit", "Zola"}
	firsts := []string{"Élodie", "Zoé", "Léo", "Paul", "Anaïs", "Jérôme", "Chloé", "Noé"}
	accounts := make([]int64, 200)
	for i := range accounts {
		a, err := env.Store.Write.CreateAccount(ctx, sqlc.CreateAccountParams{
			Email: fmt.Sprintf("volume%d@exemple.fr", i), PasswordHash: "x", Role: auth.RoleMember,
			CreatedAt: "2026-11-01T10:00:00Z",
		})
		if err != nil {
			t.Fatal(err)
		}
		accounts[i] = a.ID
	}
	err := env.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		for i := range 2000 {
			if _, err := q.CreateMember(ctx, sqlc.CreateMemberParams{
				AccountID: sql.NullInt64{Int64: accounts[i%200], Valid: true},
				FirstName: firsts[i%len(firsts)], LastName: fmt.Sprintf("%s%d", lasts[i%len(lasts)], i/100),
				BirthDate: "2000-01-01", Now: "2026-11-01T10:00:00Z",
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Origine, preuves et lien personnel d'une fiche sans compte (016 FR-040,
// research.md › R4, R9).
func TestAdminMemberTracking(t *testing.T) {
	f := newPubFixture(t, true)
	const email = "claire@exemple.fr"
	expectOutcome(t, f.confirm(f.env.Client(), f.deposit(t, f.publicBody(email))), "tracking")
	id := int64(count(t, f.env, `SELECT id FROM members WHERE origin = 'public'`))
	got := expectMember(t, f.admin.Do("GET", adminMemberPath(id), nil), http.StatusOK)
	if got.Origin != "public" || got.EmailConfirmedAt == nil || got.ConsentedAt == nil || !got.TrackedByLink {
		t.Fatalf("fiche publique : %+v", got)
	}

	// Plafond d'e-mails de l'adresse, dépôt compris : 5 par heure.
	for i := range 4 {
		if res := f.admin.Do("POST", adminMemberPath(id)+"/tracking-link", nil); res.Status != http.StatusNoContent {
			t.Fatalf("envoi %d : %d %s", i, res.Status, res.Body)
		}
	}
	expectProblem(t, f.admin.Do("POST", adminMemberPath(id)+"/tracking-link", nil), http.StatusTooManyRequests, api.MsgTooManyMails)
	links := 0
	for _, m := range f.env.Mails(email) {
		if m.Subject == "Votre lien de suivi" && strings.HasPrefix(m.Link, "http://frigg.test/suivi#jeton=") {
			links++
		}
	}
	// Un lien à la confirmation, puis quatre envois de l'administrateur.
	if links != 5 {
		t.Fatalf("e-mails de lien : %d", links)
	}
	if n := publicLogs(t, f.env, "tracking_link_sent"); n != 4 {
		t.Fatalf("journal tracking_link_sent : %d", n)
	}

	// Une adresse modifiée n'est plus confirmée.
	got = expectMember(t, f.admin.Do("PUT", adminMemberPath(id), memberBody("Claire", "Martin", "1985-03-02",
		"email", "autre@exemple.fr", "accountId", nil, "expectedAccountId", nil)), http.StatusOK)
	if got.EmailConfirmedAt != nil || got.ConsentedAt == nil {
		t.Fatalf("adresse modifiée : %+v", got)
	}

	// Fiche sans adresse ou rattachée à un compte : 409.
	orphan := insertOrphan(t, f.env, "Jeanne", "Petit", "2010-01-01")
	expectProblem(t, f.admin.Do("POST", adminMemberPath(orphan)+"/tracking-link", nil), http.StatusConflict, api.MsgNotTrackable)
	owner := f.env.Seed(auth.RoleMember, true)
	managed := insertForAccount(t, f.env, accountID(t, f.env, owner), "Paul", "Durand", "2000-02-02")
	expectProblem(t, f.admin.Do("POST", adminMemberPath(managed)+"/tracking-link", nil), http.StatusConflict, api.MsgNotTrackable)
	if got := expectMember(t, f.admin.Do("GET", adminMemberPath(managed), nil), http.StatusOK); got.TrackedByLink || got.Origin != "internal" {
		t.Fatalf("fiche d'un compte : %+v", got)
	}
	expectProblem(t, f.admin.Do("POST", adminMemberPath(99999)+"/tracking-link", nil), http.StatusNotFound, api.MsgMemberGone)
}

// Filtre d'origine (016 FR-037) : combiné à la recherche et aux fiches sans
// compte.
func TestAdminMembersOriginFilter(t *testing.T) {
	env := apitest.New(t)
	env.SetToday(membersToday)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	accountEmail := env.Seed(auth.RoleMember, true)
	insertPublic := func(first, last string) int64 {
		res, err := env.Store.WriteDB.Exec(`INSERT INTO members (first_name, last_name, birth_date, email, origin,
			email_confirmed_at, consented_at, created_at, updated_at)
			VALUES (?, ?, '1990-01-01', 'public@exemple.fr', 'public', ?, ?, ?, ?)`,
			first, last, "2026-10-01T10:00:00Z", "2026-10-01T10:00:00Z", "2026-10-01T10:00:00Z", "2026-10-01T10:00:00Z")
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		return id
	}
	insertPublic("Claire", "Martin")
	insertPublic("Paul", "Durand")
	attached := insertPublic("Léa", "Martin")
	if _, err := env.Store.WriteDB.Exec(`UPDATE members SET account_id = (SELECT id FROM accounts WHERE email = ?) WHERE id = ?`, accountEmail, attached); err != nil {
		t.Fatal(err)
	}
	insertOrphan(t, env, "Jean", "Martin", "1980-01-01")

	for query, want := range map[string]string{
		"?origin=public":                       "[Durand Paul Martin Claire Martin Léa]",
		"?origin=public&q=martin":              "[Martin Claire Martin Léa]",
		"?origin=public&account=none":          "[Durand Paul Martin Claire]",
		"?origin=public&q=martin&account=none": "[Martin Claire]",
		"?q=martin&account=none":               "[Martin Claire Martin Jean]",
	} {
		p := adminPage(t, admin, query)
		if fmt.Sprint(names(p)) != want {
			t.Errorf("%s : %v, attendu %s", query, names(p), want)
		}
		for _, m := range p.Items {
			if strings.Contains(query, "origin") && m.Origin != "public" {
				t.Errorf("%s : origine %s", query, m.Origin)
			}
		}
	}
	if res := admin.Do("GET", "/api/admin/members?origin=internal", nil); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("origine inconnue : %d", res.Status)
	}
}

// Accueil désactivé (016 US5, scénario 5) : un lien personnel existant
// permet encore le suivi et le paiement ; seul un nouveau dépôt est refusé.
func TestTrackingWhenPublicClosed(t *testing.T) {
	f := newPubFixture(t, true)
	setPayment(t, f.admin, nil)
	leo := trackedMember(t, f.msFixture, "Léo", "leo@exemple.fr")
	ms := validatedFor(t, f.msFixture, leo, f.young.ID)
	c := f.env.TrackingClient("leo@exemple.fr")
	f.setPublic(t, false)

	if got := expectJSON[api.Tracking](t, c.Do("GET", "/api/tracking", nil), http.StatusOK); len(got.Items) != 1 {
		t.Fatalf("suivi : %+v", got)
	}
	got := expectTracked(t, c.Do("PUT", trackingPath(ms.ID, "/payment"), map[string]any{"method": "cheque"}), http.StatusOK)
	if got.Payment == nil || got.Payment.Status != "announced" {
		t.Fatalf("annonce : %+v", got.Payment)
	}
	res := submit(f.env.Client(), f.nextIP(), f.publicBody("claire@exemple.fr"))
	if res.Status != http.StatusConflict || !strings.Contains(res.Body, "Les demandes d'adhésion ne sont pas ouvertes.") {
		t.Fatalf("nouveau dépôt : %d %s", res.Status, res.Body)
	}
}
