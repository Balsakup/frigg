package database

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestOpenAndMigrate(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "frigg.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for range 2 {
		if err := Migrate(ctx, store.WriteDB); err != nil {
			t.Fatalf("migration : %v", err)
		}
	}

	var mode string
	if err := store.WriteDB.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("journal_mode = %q (%v), attendu wal", mode, err)
	}
	var fk int
	if err := store.ReadDB.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil || fk != 1 {
		t.Fatalf("foreign_keys = %d (%v), attendu 1", fk, err)
	}
	if _, err := store.ReadDB.Exec("UPDATE association SET name = 'x' WHERE id = 1"); err == nil {
		t.Fatal("écriture acceptée sur le pool de lecture")
	}
	name, err := store.Read.GetAssociation(ctx)
	if err != nil {
		t.Fatalf("ligne association absente : %v", err)
	}
	if name.Valid {
		t.Fatalf("nom attendu NULL, obtenu %q", name.String)
	}
}

// Les fichiers temporaires de SQLite restent en mémoire : l'instance doit
// fonctionner avec un système de fichiers racine en lecture seule.
func TestTempStoreInMemory(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "frigg.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for name, db := range map[string]interface {
		QueryRow(string, ...any) *sql.Row
	}{"d'écriture": store.WriteDB, "de lecture": store.ReadDB} {
		var mode int
		if err := db.QueryRow("PRAGMA temp_store").Scan(&mode); err != nil || mode != 2 {
			t.Errorf("pool %s : temp_store = %d (%v), attendu 2 (MEMORY)", name, mode, err)
		}
	}
}

// Paiement par carte (008) : seule une tentative HelloAsso peut constater un
// paiement par carte, et une tentative référence une adhésion existante.
func TestCardPaymentConstraints(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "frigg.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := Migrate(ctx, store.WriteDB); err != nil {
		t.Fatal(err)
	}
	const ts = "2026-09-01T10:00:00Z"
	for _, q := range []string{
		`INSERT INTO seasons (id, starts_on, ends_on, created_at) VALUES (1, '2026-09-01', '2027-08-31', '` + ts + `')`,
		`INSERT INTO membership_fees (id, season_id, label, amount_cents, created_at, updated_at) VALUES (1, 1, 'Adulte', 6000, '` + ts + `', '` + ts + `')`,
		`INSERT INTO members (id, first_name, last_name, birth_date, created_at, updated_at) VALUES (1, 'Léo', 'Martin', '2015-01-01', '` + ts + `', '` + ts + `')`,
		`INSERT INTO memberships (id, member_id, season_id, fee_id, status, kind, amount_cents, requested_at, payment_reference, payment_status, updated_at)
		 VALUES (1, 1, 1, 1, 'validated', 'first', 6000, '` + ts + `', 'ADH-2627-0001', 'due', '` + ts + `')`,
	} {
		if _, err := store.WriteDB.ExecContext(ctx, q); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	if _, err := store.WriteDB.ExecContext(ctx, `UPDATE memberships SET payment_status = 'paid', paid_method = 'card', paid_on = '2026-09-02' WHERE id = 1`); err == nil {
		t.Fatal("paiement par carte sans tentative accepté")
	}
	if _, err := store.WriteDB.ExecContext(ctx, `INSERT INTO online_payments (membership_id, environment, amount_cents, status, created_at, updated_at)
		VALUES (99, 'sandbox', 100, 'pending', ?, ?)`, ts, ts); err == nil {
		t.Fatal("tentative sur une adhésion inexistante acceptée")
	}
	var enabled int
	if err := store.ReadDB.QueryRow(`SELECT card_enabled FROM payment_settings WHERE id = 1`).Scan(&enabled); err != nil || enabled != 0 {
		t.Fatalf("card_enabled = %d (%v)", enabled, err)
	}
}

// Communications (009) : contraintes, déclencheurs d'état et retour arrière
// de la migration 00009 sur une base qui contient déjà des e-mails.
func TestCommunicationsMigration(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "frigg.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, store.WriteDB, fsys)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 8); err != nil {
		t.Fatal(err)
	}
	db := store.WriteDB
	const ts = "2026-09-01T10:00:00Z"
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	exec(`INSERT INTO outbound_emails (id, recipient, subject, body_text, body_html, status, next_attempt_at, created_at)
		VALUES (1, 'a@exemple.fr', 'Ancien', 't', 'h', 'sent', ?, ?)`, ts, ts)
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	var cid sql.NullInt64
	var from sql.NullString
	if err := db.QueryRow(`SELECT communication_id, from_name FROM outbound_emails WHERE id = 1`).Scan(&cid, &from); err != nil || cid.Valid || from.Valid {
		t.Fatalf("e-mail existant : communication_id=%v from_name=%v (%v)", cid, from, err)
	}

	exec(`INSERT INTO seasons (id, starts_on, ends_on, created_at) VALUES (1, '2026-09-01', '2027-08-31', ?)`, ts)
	insertCommunication := func(id int, situations string) error {
		_, err := db.ExecContext(ctx, `INSERT INTO communications (id, draft_key, season_id, situations, target_label, subject, body,
			members_count, unreachable_count, recipients_count, created_at)
			VALUES (?, printf('%08d-0000-4000-8000-000000000000', ?), 1, ?, 'Cible', 'Objet', 'Texte', 1, 0, 1, ?)`, id, id, situations, ts)
		return err
	}
	for _, bad := range []string{"due,announced", "", "paid"} {
		if insertCommunication(9, bad) == nil {
			t.Errorf("situations %q acceptées", bad)
		}
	}
	if err := insertCommunication(1, "announced,due"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO communication_attachments (draft_key, stored_name, original_name, content_type, size_bytes, sha256, created_at)
		VALUES ('00000001-0000-4000-8000-000000000000', '../../etc/passwd', 'a.pdf', 'application/pdf', 1, printf('%064d', 0), ?)`, ts); err == nil {
		t.Error("stored_name hors format accepté")
	}

	for i, status := range []string{"pending", "pending", "pending", "sent"} {
		id := i + 10
		exec(`INSERT INTO outbound_emails (id, recipient, subject, body_text, body_html, status, next_attempt_at, created_at, communication_id)
			VALUES (?, 'b@exemple.fr', 'Objet', 't', 'h', ?, ?, ?, 1)`, id, status, ts, ts)
		exec(`INSERT INTO communication_recipients (communication_id, email, members_count, outbound_email_id, status)
			VALUES (1, 'b@exemple.fr', 1, ?, ?)`, id, status)
	}
	recipient := func(emailID int) (string, sql.NullString) {
		t.Helper()
		var status string
		var sent sql.NullString
		if err := db.QueryRow(`SELECT status, sent_at FROM communication_recipients WHERE id = ?`, emailID-9).Scan(&status, &sent); err != nil {
			t.Fatal(err)
		}
		return status, sent
	}
	exec(`UPDATE outbound_emails SET status = 'sent', sent_at = ? WHERE id = 10`, ts)
	if s, at := recipient(10); s != "sent" || at.String != ts {
		t.Errorf("envoyé : %s %v", s, at)
	}
	exec(`UPDATE outbound_emails SET status = 'failed' WHERE id = 11`)
	if s, _ := recipient(11); s != "failed" {
		t.Errorf("échec : %s", s)
	}
	exec(`DELETE FROM outbound_emails WHERE id = 12`)
	if s, _ := recipient(12); s != "failed" {
		t.Errorf("e-mail en attente supprimé : %s", s)
	}
	exec(`DELETE FROM outbound_emails WHERE id = 13`)
	var linked sql.NullInt64
	if s, _ := recipient(13); s != "sent" {
		t.Errorf("e-mail envoyé purgé : %s", s)
	}
	if err := db.QueryRow(`SELECT outbound_email_id FROM communication_recipients WHERE id = 4`).Scan(&linked); err != nil || linked.Valid {
		t.Errorf("outbound_email_id = %v (%v), attendu NULL", linked, err)
	}

	if _, err := provider.DownTo(ctx, 8); err != nil {
		t.Fatalf("retour arrière : %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM outbound_emails`).Scan(&n); err != nil || n != 3 {
		t.Fatalf("e-mails après retour arrière : %d (%v)", n, err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("réapplication : %v", err)
	}
}
