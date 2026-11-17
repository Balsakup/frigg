package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api/apitest"
)

// Balayeur des parcours publics (research.md 016 › R3) : dépôts de plus de
// 7 jours et liens expirés depuis plus de 24 heures.
func TestPublicSweeperPurges(t *testing.T) {
	ctx := context.Background()
	env := apitest.New(t)
	env.SetToday("2026-11-17")
	now := time.Date(2026, 11, 17, 11, 0, 0, 0, time.UTC) // midi à Paris
	at := func(d time.Duration) string { return now.Add(d).Format("2006-01-02T15:04:05Z") }
	db := env.Store.WriteDB
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	exec(`INSERT INTO seasons (id, starts_on, ends_on, created_at) VALUES (1, '2026-09-01', '2027-08-31', ?)`, at(0))
	exec(`INSERT INTO membership_fees (id, season_id, label, amount_cents, created_at, updated_at) VALUES (1, 1, 'Adulte', 4000, ?, ?)`, at(0), at(0))
	submission := func(id int, age time.Duration) {
		t.Helper()
		exec(`INSERT INTO public_submissions (id, kind, season_id, fee_id, declared_kind, first_name, last_name, birth_date, email,
			consented_at, token_hash, expires_at, created_at) VALUES (?, 'membership', 1, 1, 'first', 'Léa', 'Roux', '2010-01-01', 'l@exemple.fr', ?, ?, ?, ?)`,
			id, at(-age), []byte{byte(id)}, at(-age+48*time.Hour), at(-age))
	}
	link := func(id int, expiredSince time.Duration) {
		t.Helper()
		exec(`INSERT INTO tracking_links (id, email, token_hash, expires_at, created_at) VALUES (?, 'l@exemple.fr', ?, ?, ?)`,
			id, []byte{byte(id)}, at(-expiredSince), at(-expiredSince-30*24*time.Hour))
	}
	day := 24 * time.Hour
	submission(1, 8*day)
	submission(2, 6*day)
	link(1, 2*day)
	link(2, 12*time.Hour)
	link(3, -10*day) // encore valable

	env.PublicSweeper.Tick(ctx)

	ids := func(table string) []int {
		t.Helper()
		rows, err := db.Query(`SELECT id FROM ` + table + ` ORDER BY id`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []int
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			out = append(out, id)
		}
		return out
	}
	if got := ids("public_submissions"); len(got) != 1 || got[0] != 2 {
		t.Errorf("dépôts restants = %v, attendu [2]", got)
	}
	if got := ids("tracking_links"); len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Errorf("liens restants = %v, attendu [2 3]", got)
	}
	if n := publicLogs(t, env, "public_purge"); n != 1 {
		t.Errorf("journal public_purge : %d", n)
	}
}
