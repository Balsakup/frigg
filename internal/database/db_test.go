package database

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"strings"
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
	row, err := store.Read.GetAssociation(ctx)
	if err != nil {
		t.Fatalf("ligne association absente : %v", err)
	}
	name := row.Name
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

// Paramètres généraux (010) : contraintes, e-mails existants conservés et
// retour arrière de la migration 00010.
func TestAssociationSettingsMigration(t *testing.T) {
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
	if _, err := provider.UpTo(ctx, 9); err != nil {
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
	exec(`UPDATE association SET name = 'Les Sentiers du Val' WHERE id = 1`)
	exec(`INSERT INTO seasons (id, starts_on, ends_on, created_at) VALUES (1, '2026-09-01', '2027-08-31', ?)`, ts)
	exec(`INSERT INTO communications (id, draft_key, season_id, situations, target_label, subject, body,
		members_count, unreachable_count, recipients_count, created_at)
		VALUES (1, '00000001-0000-4000-8000-000000000000', 1, 'due', 'Cible', 'Objet', 'Texte', 1, 0, 1, ?)`, ts)
	exec(`INSERT INTO outbound_emails (id, recipient, subject, body_text, body_html, status, next_attempt_at, created_at, communication_id)
		VALUES (1, 'b@exemple.fr', 'Objet', 't', 'h', 'pending', ?, ?, 1)`, ts, ts)
	exec(`INSERT INTO communication_recipients (communication_id, email, members_count, outbound_email_id, status)
		VALUES (1, 'b@exemple.fr', 1, 1, 'pending')`)

	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	var name string
	var wrap int
	var footer sql.NullString
	if err := db.QueryRow(`SELECT name FROM association WHERE id = 1`).Scan(&name); err != nil || name != "Les Sentiers du Val" {
		t.Fatalf("nom conservé : %q (%v)", name, err)
	}
	if err := db.QueryRow(`SELECT wrap, footer FROM outbound_emails WHERE id = 1`).Scan(&wrap, &footer); err != nil || wrap != 0 || footer.Valid {
		t.Fatalf("e-mail existant : wrap=%d footer=%v (%v)", wrap, footer, err)
	}
	exec(`UPDATE outbound_emails SET status = 'sent', sent_at = ? WHERE id = 1`, ts)
	var status string
	if err := db.QueryRow(`SELECT status FROM communication_recipients WHERE outbound_email_id = 1`).Scan(&status); err != nil || status != "sent" {
		t.Fatalf("déclencheur de 009 après 00010 : %q (%v)", status, err)
	}

	for _, bad := range []string{
		`UPDATE association SET rna = 'W12'`,
		`UPDATE association SET siret = '1234'`,
		`UPDATE association SET primary_color = '#FFF'`,
		`UPDATE association SET primary_color = '#1F5278'`,
		`UPDATE association SET website = 'ftp://x'`,
		`UPDATE association SET phone = '+33123456789'`,
		`UPDATE association SET founded_on = '2026-13-01'`,
	} {
		if _, err := db.ExecContext(ctx, bad); err == nil {
			t.Errorf("accepté : %s", bad)
		}
	}
	exec(`UPDATE association SET rna = 'W751234567', siret = '73282932000074', primary_color = '#1f5278',
		website = 'https://exemple.fr', phone = '0123456789', founded_on = '1987-05-12' WHERE id = 1`)
	insertLogo := func(id int, web, email string) error {
		_, err := db.ExecContext(ctx, `INSERT INTO association_logo (id, original_name, source_type, source_size_bytes,
			web_name, web_type, web_size_bytes, web_width, web_height, web_sha256,
			email_name, email_type, email_size_bytes, email_width, email_height, email_sha256, uploaded_at)
			VALUES (?, 'logo.png', 'image/png', 1000, ?, 'image/png', 100, 512, 256, printf('%064d', 0),
			?, 'image/png', 100, 192, 96, printf('%064d', 0), ?)`, id, web, email, ts)
		return err
	}
	if err := insertLogo(1, strings.Repeat("a", 32), strings.Repeat("b", 32)); err != nil {
		t.Fatal(err)
	}
	if insertLogo(2, strings.Repeat("c", 32), strings.Repeat("d", 32)) == nil {
		t.Error("deuxième logo accepté")
	}

	if _, err := provider.DownTo(ctx, 9); err != nil {
		t.Fatalf("retour arrière : %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM outbound_emails`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("e-mails après retour arrière : %d (%v)", n, err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("réapplication : %v", err)
	}
}

// Champs personnalisés (011) : contraintes des valeurs, contrainte différée
// sur les choix, cascades et cohérence de l'index FTS.
func TestCustomFieldsMigration(t *testing.T) {
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
	if _, err := provider.UpTo(ctx, 10); err != nil {
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
	exec(`INSERT INTO seasons (id, starts_on, ends_on, created_at) VALUES (1, '2026-09-01', '2027-08-31', ?)`, ts)
	exec(`INSERT INTO membership_fees (id, season_id, label, amount_cents, created_at, updated_at) VALUES (1, 1, 'Adulte', 3000, ?, ?)`, ts, ts)
	exec(`INSERT INTO members (id, first_name, last_name, birth_date, created_at, updated_at) VALUES (1, 'Lucas', 'Martin', '2012-04-02', ?, ?)`, ts, ts)
	exec(`INSERT INTO members (id, first_name, last_name, birth_date, created_at, updated_at) VALUES (2, 'Claire', 'Martin', '1980-01-01', ?, ?)`, ts, ts)
	exec(`INSERT INTO memberships (id, member_id, season_id, fee_id, status, requested_at, updated_at) VALUES (1, 1, 1, 1, 'pending', ?, ?)`, ts, ts)

	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO custom_fields (id, scope, label, type, position, created_at, updated_at) VALUES (1, 'member', 'Licence', 'short_text', 1, ?, ?)`, ts, ts)
	exec(`INSERT INTO custom_fields (id, scope, label, type, position, created_at, updated_at) VALUES (2, 'membership', 'Taille', 'choice', 1, ?, ?)`, ts, ts)
	exec(`INSERT INTO custom_field_choices (id, field_id, label, position) VALUES (10, 2, 'S', 1), (11, 2, 'M', 2)`)

	for _, bad := range []string{
		`INSERT INTO custom_fields (scope, label, type, required, position, created_at, updated_at) VALUES ('member', 'X', 'bool', 1, 2, 'a', 'a')`,
		`INSERT INTO member_field_values (member_id, field_id, value_text, value_number, updated_at) VALUES (2, 1, 'a', 1, 'a')`,
		`INSERT INTO member_field_values (member_id, field_id, updated_at) VALUES (2, 1, 'a')`,
		`INSERT INTO member_field_values (member_id, field_id, value_date, updated_at) VALUES (2, 1, '2026-02-30', 'a')`,
		`INSERT INTO member_field_values (member_id, field_id, value_text, updated_at) VALUES (2, 1, '', 'a')`,
	} {
		if _, err := db.ExecContext(ctx, bad); err == nil {
			t.Errorf("accepté : %s", bad)
		}
	}

	exec(`INSERT INTO member_field_values (member_id, field_id, value_text, updated_at) VALUES (1, 1, '1234567', ?)`, ts)
	exec(`INSERT INTO member_field_values (member_id, field_id, value_text, updated_at) VALUES (2, 1, '7654321', ?)`, ts)
	exec(`INSERT INTO membership_field_values (membership_id, field_id, value_choice_id, updated_at) VALUES (1, 2, 10, ?)`, ts)

	var found int64
	if err := db.QueryRow(`SELECT rowid FROM member_field_values_fts WHERE member_field_values_fts MATCH '"3456"'`).Scan(&found); err != nil {
		t.Fatalf("recherche trigram : %v", err)
	}

	// Choix utilisé : refusé à la validation de la transaction.
	if _, err := db.ExecContext(ctx, `DELETE FROM custom_field_choices WHERE id = 10`); err == nil {
		t.Error("suppression d'un choix utilisé acceptée")
	}
	exec(`DELETE FROM custom_field_choices WHERE id = 11`)

	integrity := func(when string) {
		t.Helper()
		if _, err := db.ExecContext(ctx, `INSERT INTO member_field_values_fts (member_field_values_fts) VALUES ('integrity-check')`); err != nil {
			t.Fatalf("index FTS incohérent après %s : %v", when, err)
		}
	}
	exec(`DELETE FROM members WHERE id = 1`)
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM member_field_values WHERE member_id = 1`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("valeurs de la fiche supprimée : %d (%v)", n, err)
	}
	integrity("la suppression d'une fiche")

	// Champ portant une valeur de choix : la cascade réussit quel que soit l'ordre.
	exec(`INSERT INTO members (id, first_name, last_name, birth_date, created_at, updated_at) VALUES (3, 'Léa', 'Roux', '2010-01-01', ?, ?)`, ts, ts)
	exec(`INSERT INTO memberships (id, member_id, season_id, fee_id, status, requested_at, updated_at) VALUES (2, 3, 1, 1, 'pending', ?, ?)`, ts, ts)
	exec(`INSERT INTO membership_field_values (membership_id, field_id, value_choice_id, updated_at) VALUES (2, 2, 10, ?)`, ts)
	exec(`DELETE FROM custom_fields WHERE id = 2`)
	if err := db.QueryRow(`SELECT (SELECT count(*) FROM membership_field_values) + (SELECT count(*) FROM custom_field_choices)`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("valeurs et choix du champ supprimé : %d (%v)", n, err)
	}
	exec(`DELETE FROM custom_fields WHERE id = 1`)
	integrity("la suppression d'un champ")

	if _, err := provider.Down(ctx); err != nil {
		t.Fatalf("Down de 00011 : %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("Up après Down : %v", err)
	}
}

// Événements (012) : reconstruction de online_payments et payment_alerts sur
// une base peuplée, contraintes des inscriptions et retour arrière de la
// migration 00012 (research.md 012 › R2).
func TestEventsMigration(t *testing.T) {
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
	if _, err := provider.UpTo(ctx, 11); err != nil {
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
	refused := func(what, q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err == nil {
			t.Errorf("%s accepté", what)
		}
	}
	foreignKeysOn := func() {
		t.Helper()
		var on int
		if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&on); err != nil || on != 1 {
			t.Fatalf("foreign_keys = %d (%v), attendu 1", on, err)
		}
		rows, err := db.Query(`PRAGMA foreign_key_check`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		if rows.Next() {
			t.Fatal("violation de clé étrangère après migration")
		}
	}

	// Base 00011 : adhésion payée par carte, avec un signalement ouvert.
	exec(`INSERT INTO accounts (id, email, password_hash, role, created_at) VALUES (1, 'a@exemple.fr', 'h', 'admin', ?)`, ts)
	exec(`INSERT INTO seasons (id, starts_on, ends_on, created_at) VALUES (1, '2026-09-01', '2027-08-31', ?)`, ts)
	exec(`INSERT INTO membership_fees (id, season_id, label, amount_cents, created_at, updated_at) VALUES (1, 1, 'Adulte', 4000, ?, ?)`, ts, ts)
	exec(`INSERT INTO members (id, first_name, last_name, birth_date, created_at, updated_at) VALUES (1, 'Lucas', 'Martin', '2015-01-01', ?, ?), (2, 'Claire', 'Martin', '1985-01-01', ?, ?)`, ts, ts, ts, ts)
	exec(`INSERT INTO memberships (id, member_id, season_id, fee_id, status, kind, amount_cents, requested_at, updated_at, payment_reference, payment_status)
		VALUES (1, 1, 1, 1, 'validated', 'first', 4000, ?, ?, 'ADH-2627-0001', 'due')`, ts, ts)
	exec(`INSERT INTO online_payments (id, membership_id, environment, amount_cents, status, checkout_intent_id, order_id, helloasso_payment_id,
		received_amount_cents, paid_at, created_at, finished_at, updated_at)
		VALUES (7, 1, 'sandbox', 4000, 'succeeded', 70, 71, 72, 4000, ?, ?, ?, ?)`, ts, ts, ts, ts)
	exec(`UPDATE memberships SET payment_status = 'paid', paid_method = 'card', paid_on = '2026-09-01', paid_online_payment_id = 7, paid_amount_cents = 4000 WHERE id = 1`)
	exec(`INSERT INTO payment_alerts (id, membership_id, online_payment_id, kind, created_at) VALUES (3, 1, 7, 'duplicate', ?)`, ts)

	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("migration 00012 : %v", err)
	}
	foreignKeysOn()

	var membershipID, paidBy sql.NullInt64
	var status string
	if err := db.QueryRow(`SELECT membership_id, status FROM online_payments WHERE id = 7`).Scan(&membershipID, &status); err != nil ||
		membershipID.Int64 != 1 || status != "succeeded" {
		t.Fatalf("tentative conservée : %v %s (%v)", membershipID, status, err)
	}
	if err := db.QueryRow(`SELECT paid_online_payment_id FROM memberships WHERE id = 1`).Scan(&paidBy); err != nil || paidBy.Int64 != 7 {
		t.Fatalf("paid_online_payment_id = %v (%v)", paidBy, err)
	}
	if err := db.QueryRow(`SELECT membership_id FROM payment_alerts WHERE id = 3`).Scan(&membershipID); err != nil || membershipID.Int64 != 1 {
		t.Fatalf("signalement conservé : %v (%v)", membershipID, err)
	}

	exec(`INSERT INTO events (id, season_id, title, location, starts_at, ends_at, price_cents, capacity, status, published_at, created_at, updated_at)
		VALUES (1, 1, 'Stage', 'Gymnase', '2026-12-12T13:00:00Z', '2026-12-12T16:00:00Z', 3500, 20, 'published', ?, ?, ?)`, ts, ts, ts)
	refused("capacité 0", `INSERT INTO events (season_id, title, location, starts_at, ends_at, capacity, created_at, updated_at)
		VALUES (1, 'X', 'Y', '2026-12-12T13:00:00Z', '2026-12-12T16:00:00Z', 0, ?, ?)`, ts, ts)
	refused("fin avant début", `INSERT INTO events (season_id, title, location, starts_at, ends_at, created_at, updated_at)
		VALUES (1, 'X', 'Y', '2026-12-12T13:00:00Z', '2026-12-12T12:00:00Z', ?, ?)`, ts, ts)
	refused("inscription payante sans référence", `INSERT INTO event_registrations (event_id, member_id, status, registered_at, amount_cents, payment_status, updated_at)
		VALUES (1, 1, 'confirmed', ?, 3500, 'due', ?)`, ts, ts)
	exec(`INSERT INTO event_registrations (id, event_id, member_id, status, registered_at, cancelled_at, amount_cents, payment_reference, payment_status, updated_at)
		VALUES (1, 1, 1, 'cancelled', ?, ?, 3500, 'EVT-2627-0001', 'due', ?)`, ts, ts, ts)
	exec(`INSERT INTO event_registrations (id, event_id, member_id, status, registered_at, amount_cents, payment_reference, payment_status, updated_at)
		VALUES (2, 1, 1, 'confirmed', ?, 3500, 'EVT-2627-0002', 'due', ?)`, ts, ts)
	refused("deuxième inscription confirmée", `INSERT INTO event_registrations (event_id, member_id, status, registered_at, amount_cents, payment_reference, payment_status, updated_at)
		VALUES (1, 1, 'confirmed', ?, 3500, 'EVT-2627-0003', 'due', ?)`, ts, ts)
	refused("tentative à deux cibles", `INSERT INTO online_payments (membership_id, registration_id, environment, amount_cents, status, created_at, updated_at)
		VALUES (1, 2, 'sandbox', 3500, 'pending', ?, ?)`, ts, ts)
	refused("tentative sans cible", `INSERT INTO online_payments (environment, amount_cents, status, created_at, updated_at)
		VALUES ('sandbox', 3500, 'pending', ?, ?)`, ts, ts)
	exec(`INSERT INTO online_payments (id, registration_id, environment, amount_cents, status, created_at, updated_at)
		VALUES (8, 2, 'sandbox', 3500, 'pending', ?, ?)`, ts, ts)
	exec(`INSERT INTO payment_alerts (registration_id, online_payment_id, kind, created_at) VALUES (2, 8, 'unverified', ?)`, ts)
	refused("signalement à deux cibles", `INSERT INTO payment_alerts (membership_id, registration_id, online_payment_id, kind, created_at) VALUES (1, 2, 8, 'duplicate', ?)`, ts)

	exec(`INSERT INTO event_registrations_fts (event_registrations_fts) VALUES ('integrity-check')`)
	exec(`DELETE FROM members WHERE id = 1`)
	var n int
	for _, q := range []string{
		`SELECT count(*) FROM event_registrations`,
		`SELECT count(*) FROM online_payments`,
		`SELECT count(*) FROM payment_alerts`,
	} {
		if err := db.QueryRow(q).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s = %d après suppression de la fiche (%v)", q, n, err)
		}
	}
	exec(`INSERT INTO event_registrations_fts (event_registrations_fts) VALUES ('integrity-check')`)

	if _, err := provider.DownTo(ctx, 11); err != nil {
		t.Fatalf("retour arrière : %v", err)
	}
	foreignKeysOn()
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("réapplication : %v", err)
	}
	foreignKeysOn()
}

// Remises et dons (014) : reprise du montant reçu des paiements manuels,
// contraintes de l'ajustement et retour arrière de 00013 (research.md 014 › R1, R2).
func TestPaymentAdjustmentsMigration(t *testing.T) {
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
	if _, err := provider.UpTo(ctx, 12); err != nil {
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
	refused := func(what, q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err == nil {
			t.Errorf("%s accepté", what)
		}
	}
	paid := func(table string, id int64) sql.NullInt64 {
		t.Helper()
		var v sql.NullInt64
		if err := db.QueryRow(`SELECT paid_amount_cents FROM `+table+` WHERE id = ?`, id).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}

	// Base 00012 : adhésions payée en espèces, payée par carte, due ;
	// inscriptions payée par virement et due.
	exec(`INSERT INTO seasons (id, starts_on, ends_on, created_at) VALUES (1, '2026-09-01', '2027-08-31', ?)`, ts)
	exec(`INSERT INTO membership_fees (id, season_id, label, amount_cents, created_at, updated_at) VALUES (1, 1, 'Adulte', 4500, ?, ?)`, ts, ts)
	exec(`INSERT INTO members (id, first_name, last_name, birth_date, created_at, updated_at) VALUES
		(1, 'A', 'A', '1980-01-01', ?, ?), (2, 'B', 'B', '1980-01-01', ?, ?), (3, 'C', 'C', '1980-01-01', ?, ?)`, ts, ts, ts, ts, ts, ts)
	exec(`INSERT INTO memberships (id, member_id, season_id, fee_id, status, kind, amount_cents, requested_at, updated_at, payment_reference,
		payment_status, paid_method, paid_on) VALUES (1, 1, 1, 1, 'validated', 'first', 4500, ?, ?, 'ADH-2627-0001', 'paid', 'cash', '2026-09-02')`, ts, ts)
	exec(`INSERT INTO memberships (id, member_id, season_id, fee_id, status, kind, amount_cents, requested_at, updated_at, payment_reference, payment_status)
		VALUES (2, 2, 1, 1, 'validated', 'first', 4500, ?, ?, 'ADH-2627-0002', 'due')`, ts, ts)
	exec(`INSERT INTO online_payments (id, membership_id, environment, amount_cents, status, checkout_intent_id, order_id, helloasso_payment_id,
		received_amount_cents, paid_at, created_at, finished_at, updated_at)
		VALUES (7, 2, 'sandbox', 4500, 'succeeded', 70, 71, 72, 4000, ?, ?, ?, ?)`, ts, ts, ts, ts)
	exec(`UPDATE memberships SET payment_status = 'paid', paid_method = 'card', paid_on = '2026-09-01', paid_online_payment_id = 7, paid_amount_cents = 4000 WHERE id = 2`)
	exec(`INSERT INTO memberships (id, member_id, season_id, fee_id, status, kind, amount_cents, requested_at, updated_at, payment_reference, payment_status)
		VALUES (3, 3, 1, 1, 'validated', 'first', 4500, ?, ?, 'ADH-2627-0003', 'due')`, ts, ts)
	exec(`INSERT INTO events (id, season_id, title, location, starts_at, ends_at, price_cents, status, published_at, created_at, updated_at)
		VALUES (1, 1, 'Stage', 'Gymnase', '2026-12-12T13:00:00Z', '2026-12-12T16:00:00Z', 3500, 'published', ?, ?, ?)`, ts, ts, ts)
	exec(`INSERT INTO event_registrations (id, event_id, member_id, status, registered_at, amount_cents, payment_reference, payment_status,
		paid_method, paid_on, updated_at) VALUES (1, 1, 1, 'confirmed', ?, 3500, 'EVT-2627-0001', 'paid', 'transfer', '2026-09-03', ?)`, ts, ts)
	exec(`INSERT INTO event_registrations (id, event_id, member_id, status, registered_at, amount_cents, payment_reference, payment_status, updated_at)
		VALUES (2, 1, 2, 'confirmed', ?, 3500, 'EVT-2627-0002', 'due', ?)`, ts, ts)

	if _, err := provider.UpTo(ctx, 13); err != nil {
		t.Fatalf("migration 00013 : %v", err)
	}
	if v := paid("memberships", 1); v.Int64 != 4500 {
		t.Errorf("espèces : paid_amount_cents = %v, attendu 4500", v)
	}
	if v := paid("memberships", 2); v.Int64 != 4000 {
		t.Errorf("carte : paid_amount_cents = %v, attendu 4000 inchangé", v)
	}
	if v := paid("memberships", 3); v.Valid {
		t.Errorf("due : paid_amount_cents = %v, attendu NULL", v)
	}
	if v := paid("event_registrations", 1); v.Int64 != 3500 {
		t.Errorf("virement : paid_amount_cents = %v, attendu 3500", v)
	}
	if v := paid("event_registrations", 2); v.Valid {
		t.Errorf("inscription due : paid_amount_cents = %v, attendu NULL", v)
	}

	long := strings.Repeat("x", 201)
	for _, table := range []string{"memberships", "event_registrations"} {
		refused(table+" : motif sans montant théorique", `UPDATE `+table+` SET adjustment_reason = 'remise' WHERE id = 2`)
		refused(table+" : montant théorique sans motif", `UPDATE `+table+` SET base_amount_cents = 5000 WHERE id = 2`)
		refused(table+" : montant théorique égal au montant", `UPDATE `+table+` SET base_amount_cents = amount_cents, adjustment_reason = 'x' WHERE id = 2`)
		refused(table+" : motif de 201 caractères", `UPDATE `+table+` SET base_amount_cents = 9000, adjustment_reason = ? WHERE id = 2`, long)
		exec(`UPDATE ` + table + ` SET base_amount_cents = 9000, adjustment_reason = 'remise' WHERE id = 2`)
	}

	if _, err := provider.DownTo(ctx, 12); err != nil {
		t.Fatalf("retour arrière : %v", err)
	}
	for _, table := range []string{"memberships", "event_registrations"} {
		if _, err := db.ExecContext(ctx, `SELECT base_amount_cents FROM `+table); err == nil {
			t.Errorf("%s : base_amount_cents présente après retour arrière", table)
		}
	}
	if v := paid("memberships", 1); v.Valid {
		t.Errorf("retour arrière espèces : %v, attendu NULL", v)
	}
	if v := paid("memberships", 2); v.Int64 != 4000 {
		t.Errorf("retour arrière carte : %v, attendu 4000", v)
	}
	if v := paid("event_registrations", 1); v.Valid {
		t.Errorf("retour arrière virement : %v, attendu NULL", v)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("réapplication : %v", err)
	}
}
