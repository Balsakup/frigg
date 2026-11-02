package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
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
