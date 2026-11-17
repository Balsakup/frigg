package api_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

// Fiches publiques inactives (016 US5, FR-041, research.md › R14). Le jour est
// le 17 novembre 2026 : la coupure tombe le 17 novembre 2025.

const inactiveToday = "2026-11-17"

// inactiveFixture insère directement les données datées du passé.
type inactiveFixture struct {
	env   *apitest.Env
	admin *apitest.Client
	t     *testing.T
}

func (f inactiveFixture) exec(query string, args ...any) int64 {
	f.t.Helper()
	res, err := f.env.Store.WriteDB.Exec(query, args...)
	if err != nil {
		f.t.Fatalf("%s : %v", query, err)
	}
	id, _ := res.LastInsertId()
	return id
}

// returning exécute une requête RETURNING id.
func (f inactiveFixture) returning(query string, args ...any) int64 {
	f.t.Helper()
	var id int64
	if err := f.env.Store.WriteDB.QueryRow(query, args...).Scan(&id); err != nil {
		f.t.Fatalf("%s : %v", query, err)
	}
	return id
}

// member insère une fiche sans compte d'origine origin créée à createdAt.
func (f inactiveFixture) member(first, origin, createdAt string) int64 {
	f.t.Helper()
	return f.exec(`INSERT INTO members (first_name, last_name, birth_date, email, origin, created_at, updated_at)
		VALUES (?, 'Public', '1990-01-01', ?, ?, ?, ?)`, first, strings.ToLower(first)+"@exemple.fr", origin, createdAt, createdAt)
}

// validated insère une adhésion validée gratuite d'une saison du passé.
func (f inactiveFixture) validated(memberID int64, startsOn, endsOn string) {
	f.t.Helper()
	season := f.exec(`INSERT INTO seasons (starts_on, ends_on, opened_at, created_at) VALUES (?, ?, ?, ?)`,
		startsOn, endsOn, startsOn+"T00:00:00Z", startsOn+"T00:00:00Z")
	fee := f.exec(`INSERT INTO membership_fees (season_id, label, amount_cents, created_at, updated_at)
		VALUES (?, 'Gratuit', 0, ?, ?)`, season, startsOn+"T00:00:00Z", startsOn+"T00:00:00Z")
	f.exec(`INSERT INTO memberships (member_id, season_id, fee_id, status, kind, amount_cents, requested_at, decided_at, updated_at)
		VALUES (?, ?, ?, 'validated', 'first', 0, ?, ?, ?)`, memberID, season, fee, startsOn+"T10:00:00Z", startsOn+"T10:00:00Z", startsOn+"T10:00:00Z")
}

// pending insère une demande en attente dans une saison du passé.
func (f inactiveFixture) pending(memberID int64) {
	f.t.Helper()
	season := f.returning(`INSERT INTO seasons (starts_on, ends_on, opened_at, created_at)
		VALUES ('2023-09-01', '2024-08-31', '2023-09-01T00:00:00Z', '2023-09-01T00:00:00Z')
		ON CONFLICT (starts_on) DO UPDATE SET ends_on = excluded.ends_on RETURNING id`)
	var fee int64
	if err := f.env.Store.ReadDB.QueryRow(`SELECT id FROM membership_fees WHERE season_id = ?`, season).Scan(&fee); err != nil {
		fee = f.exec(`INSERT INTO membership_fees (season_id, label, amount_cents, created_at, updated_at)
			VALUES (?, 'Gratuit', 0, '2023-09-01T00:00:00Z', '2023-09-01T00:00:00Z')`, season)
	}
	f.exec(`INSERT INTO memberships (member_id, season_id, fee_id, status, declared_kind, requested_at, updated_at)
		VALUES (?, ?, ?, 'pending', 'first', '2023-09-02T10:00:00Z', '2023-09-02T10:00:00Z')`, memberID, season, fee)
}

// registration insère une inscription gratuite à un événement terminé à endsAt.
func (f inactiveFixture) registration(memberID int64, endsAt, status string) {
	f.t.Helper()
	season := f.returning(`INSERT INTO seasons (starts_on, ends_on, opened_at, created_at)
		VALUES ('2022-09-01', '2023-08-31', '2022-09-01T00:00:00Z', '2022-09-01T00:00:00Z')
		ON CONFLICT (starts_on) DO UPDATE SET ends_on = excluded.ends_on RETURNING id`)
	event := f.exec(`INSERT INTO events (season_id, title, location, starts_at, ends_at, status, published_at, created_at, updated_at)
		VALUES (?, 'Stage', 'Gymnase', ?, ?, 'published', '2022-09-01T00:00:00Z', '2022-09-01T00:00:00Z', '2022-09-01T00:00:00Z')`,
		season, "2022-09-02T08:00:00Z", endsAt)
	var cancelledAt any
	if status == "cancelled" {
		cancelledAt = "2022-09-03T10:00:00Z"
	}
	f.exec(`INSERT INTO event_registrations (event_id, member_id, status, registered_at, cancelled_at, amount_cents, updated_at)
		VALUES (?, ?, ?, '2022-09-02T10:00:00Z', ?, 0, '2022-09-02T10:00:00Z')`, event, memberID, status, cancelledAt)
}

func inactiveNames(p api.AdminMemberPage) []string {
	out := make([]string, len(p.Items))
	for i, m := range p.Items {
		out[i] = m.FirstName
	}
	slices.Sort(out)
	return out
}

func TestInactivePublicMembers(t *testing.T) {
	env := apitest.New(t)
	env.SetToday(inactiveToday)
	adminEmail := env.Seed(auth.RoleAdmin, true)
	f := inactiveFixture{env: env, admin: env.LoggedIn(adminEmail), t: t}
	old := "2025-10-01T10:00:00Z"

	f.member("Ancienne", "public", old)
	f.member("Recente", "public", "2025-12-20T10:00:00Z")
	// Créée le 17 novembre 2025 à 0 h 30 à Paris : pas avant la coupure.
	f.member("Limite", "public", "2025-11-16T23:30:00Z")
	f.pending(f.member("Attente", "public", old))
	f.validated(f.member("SaisonRecente", "public", old), "2025-06-01", "2026-05-31")
	f.validated(f.member("SaisonAncienne", "public", old), "2024-06-01", "2025-05-31")
	f.registration(f.member("EvenementAncien", "public", old), "2025-10-10T18:00:00Z", "confirmed")
	f.registration(f.member("EvenementRecent", "public", old), "2026-01-10T18:00:00Z", "confirmed")
	f.registration(f.member("Desinscrite", "public", old), "2026-01-10T18:00:00Z", "cancelled")
	f.member("Interne", "internal", old)
	account := env.Seed(auth.RoleMember, true)
	withAccount := f.member("AvecCompte", "public", old)
	f.exec(`UPDATE members SET account_id = (SELECT id FROM accounts WHERE email = ?) WHERE id = ?`, account, withAccount)

	want := []string{"Ancienne", "Desinscrite", "EvenementAncien", "SaisonAncienne"}
	p := adminPage(t, f.admin, "?inactive=true")
	if got := inactiveNames(p); fmt.Sprint(got) != fmt.Sprint(want) || p.Total != 4 {
		t.Fatalf("fiches inactives : %v (total %d)", got, p.Total)
	}
	ids := map[string]int64{}
	for _, m := range adminPage(t, f.admin, "?pageSize=100").Items {
		ids[m.FirstName] = m.ID
		if m.Inactive != slices.Contains(want, m.FirstName) {
			t.Errorf("%s : inactive = %v", m.FirstName, m.Inactive)
		}
	}
	if p := adminPage(t, f.admin, "?inactive=true&origin=public&account=none"); p.Total != 4 {
		t.Fatalf("filtres combinés : %v", inactiveNames(p))
	}
	got := expectJSON[api.AdminMember](t, f.admin.Do("GET", fmt.Sprintf("/api/admin/members/%d", ids["Ancienne"]), nil), http.StatusOK)
	if !got.Inactive {
		t.Fatal("fiche inactive non signalée")
	}

	dash := expectJSON[api.MembershipDashboard](t, f.admin.Do("GET", "/api/admin/memberships/dashboard", nil), http.StatusOK)
	if dash.InactivePublicCount != 4 {
		t.Fatalf("tableau de bord : %d", dash.InactivePublicCount)
	}

	// Membre : 403 ; liste vide ou trop longue : 422.
	member := env.LoggedIn(account)
	body := map[string]any{"ids": []int64{ids["Ancienne"]}}
	if res := member.Do("POST", "/api/admin/members/inactive/delete", body); res.Status != http.StatusForbidden {
		t.Fatalf("membre : %d", res.Status)
	}
	for _, list := range [][]int64{{}, make([]int64, 501)} {
		if res := f.admin.Do("POST", "/api/admin/members/inactive/delete", map[string]any{"ids": list}); res.Status != http.StatusUnprocessableEntity {
			t.Fatalf("%d identifiants : %d", len(list), res.Status)
		}
	}

	// « Ancienne » redevient active avant la suppression : elle est ignorée,
	// comme une fiche interne, une fiche active et une fiche inconnue.
	f.pending(ids["Ancienne"])
	res := f.admin.Do("POST", "/api/admin/members/inactive/delete", map[string]any{"ids": []int64{
		ids["Ancienne"], ids["SaisonAncienne"], ids["EvenementAncien"], ids["Desinscrite"], ids["Desinscrite"],
		ids["Interne"], ids["Recente"], 999999,
	}})
	out := expectJSON[api.DeleteInactiveMembersResult](t, res, http.StatusOK)
	if out.Deleted != 3 || out.Skipped != 4 {
		t.Fatalf("suppression : %+v", out)
	}
	for _, name := range []string{"SaisonAncienne", "EvenementAncien", "Desinscrite"} {
		if n := count(t, env, `SELECT count(*) FROM members WHERE id = ?`, ids[name]); n != 0 {
			t.Errorf("%s non supprimée", name)
		}
	}
	// Adhésions et inscriptions partent avec la fiche (006 FR-021, 012).
	if n := count(t, env, `SELECT count(*) FROM memberships WHERE member_id = ?`, ids["SaisonAncienne"]) +
		count(t, env, `SELECT count(*) FROM event_registrations WHERE member_id IN (?, ?)`, ids["EvenementAncien"], ids["Desinscrite"]); n != 0 {
		t.Fatalf("%d adhésions ou inscriptions orphelines", n)
	}
	if n := count(t, env, `SELECT count(*) FROM members WHERE id IN (?, ?, ?)`, ids["Ancienne"], ids["Interne"], ids["Recente"]); n != 3 {
		t.Fatal("fiche ignorée supprimée")
	}
	if p := adminPage(t, f.admin, "?inactive=true"); p.Total != 0 {
		t.Fatalf("fiches inactives restantes : %v", inactiveNames(p))
	}
	n := 0
	for _, line := range strings.Split(env.Logs.String(), "\n") {
		if strings.Contains(line, `"action":"inactive_members_deleted"`) {
			n++
			if !strings.Contains(line, `"deleted":3`) || !strings.Contains(line, `"skipped":4`) || strings.Contains(line, "@") {
				t.Errorf("journal : %s", line)
			}
		}
	}
	if n != 1 {
		t.Fatalf("journal inactive_members_deleted : %d", n)
	}
}
