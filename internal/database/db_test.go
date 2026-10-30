package database

import (
	"context"
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
