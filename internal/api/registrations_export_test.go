package api_test

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
)

func TestExportEventRegistrations(t *testing.T) {
	f := newEvFixture(t)
	setPayment(t, f.admin, nil)
	ev := f.publishedEvent(t, eventBody("Stage", paidEvent("35")))
	register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID)
	// Un nom qui ressemble à une formule est neutralisé.
	res := f.admin.Do("POST", "/api/admin/members", memberBody("=1+1", "Martinez", "2015-04-12", "accountId", nil))
	if res.Status != http.StatusCreated {
		t.Fatalf("fiche : %d %s", res.Status, res.Body)
	}
	var created struct {
		ID int64 `json:"id"`
	}
	res.JSON(t, &created)
	expectJSON[api.AdminRegistration](t, f.admin.Do("POST", eventPath(ev.ID, "/registrations"), map[string]any{"memberId": created.ID}), http.StatusCreated)

	res = f.admin.Do("GET", eventPath(ev.ID, "/registrations.csv"), nil)
	if res.Status != http.StatusOK || res.Header.Get("Content-Type") != "text/csv; charset=utf-8" ||
		res.Header.Get("Content-Disposition") != fmt.Sprintf(`attachment; filename="inscrits-%d.csv"`, ev.ID) {
		t.Fatalf("export : %d %v", res.Status, res.Header)
	}
	if !strings.HasPrefix(res.Body, "\xef\xbb\xbf") {
		t.Fatal("BOM UTF-8 absent")
	}
	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(res.Body, "\xef\xbb\xbf")))
	r.Comma = ';'
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0][0] != "Nom" || rows[0][4] != "Adresse de contact" || rows[0][8] != "Montant à payer" || rows[0][14] != "Référence" {
		t.Fatalf("en-tête : %v", rows)
	}
	var lucas []string
	for _, row := range rows[1:] {
		if row[0] == "Martinez" {
			lucas = row
		}
	}
	if lucas == nil || lucas[1] != "'=1+1" || lucas[5] != "non" || lucas[10] != "À payer" || lucas[14] != "EVT-2627-0002" {
		t.Fatalf("ligne : %v", lucas)
	}
	if res := f.claire.Do("GET", eventPath(ev.ID, "/registrations.csv"), nil); res.Status != http.StatusForbidden {
		t.Fatalf("membre : %d", res.Status)
	}
}

// SC-007 : 500 inscrits listés et exportés en moins de 2 s chacun.
func TestEventRegistrationsVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("volume")
	}
	f := newEvFixture(t)
	ev := f.publishedEvent(t, eventBody("Fête", func(b map[string]any) { b["openness"] = "all_members" }))
	tx, err := f.env.Store.WriteDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	const ts = "2026-11-01T10:00:00Z"
	for i := range 500 {
		res, err := tx.Exec(`INSERT INTO members (first_name, last_name, birth_date, created_at, updated_at) VALUES (?, 'Volume', '2000-01-01', ?, ?)`,
			fmt.Sprintf("P%03d", i), ts, ts)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		if _, err := tx.Exec(`INSERT INTO event_registrations (event_id, member_id, status, registered_at, amount_cents, updated_at)
			VALUES (?, ?, 'confirmed', ?, 0, ?)`, ev.ID, id, ts, ts); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	timed := func(name, path string) string {
		t.Helper()
		start := time.Now()
		res := f.admin.Do("GET", path, nil)
		if d := time.Since(start); res.Status != http.StatusOK || d > 2*time.Second {
			t.Fatalf("%s : %d en %s", name, res.Status, d)
		}
		return res.Body
	}
	timed("liste", eventPath(ev.ID, "/registrations"))
	timed("événement", eventPath(ev.ID, ""))
	body := timed("export", eventPath(ev.ID, "/registrations.csv"))
	if n := strings.Count(body, "\n"); n != 501 {
		t.Fatalf("lignes exportées : %d", n)
	}
	var plan string
	rows, err := f.env.Store.ReadDB.Query(`EXPLAIN QUERY PLAN SELECT * FROM event_registrations r WHERE r.event_id = ? AND r.status = 'confirmed'`, ev.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatal(err)
		}
		plan += detail + "\n"
	}
	if !strings.Contains(plan, "event_registrations_event") && !strings.Contains(plan, "event_registrations_one_confirmed") {
		t.Fatalf("plan sans index : %s", plan)
	}
}
