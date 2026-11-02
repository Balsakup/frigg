package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/balsakup/frigg/internal/database"
)

func TestBackup(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	if err := CreateAdmin(ctx, store, AdminInput{AssociationName: "Les Sentiers du Val", Email: "claire@exemple.fr", Password: "Motdepasse1"}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "sauvegarde.db")
	if err := Backup(ctx, store, dest); err != nil {
		t.Fatal(err)
	}

	restored, err := database.Open(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if _, err := restored.Read.GetAccountByEmail(ctx, "claire@exemple.fr"); err != nil {
		t.Fatalf("compte absent de la sauvegarde : %v", err)
	}

	if err := Backup(ctx, store, dest); err == nil || err.Error() != "le fichier "+dest+" existe déjà" {
		t.Fatalf("écrasement : %v", err)
	}
}

// La sauvegarde contient les empreintes de mots de passe et les données
// personnelles : elle n'est lisible que par son propriétaire.
func TestBackupIsPrivate(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	dest := filepath.Join(t.TempDir(), "sauvegarde.db")
	if err := Backup(ctx, store, dest); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if perm := st.Mode().Perm(); perm != 0o600 {
		t.Fatalf("permissions : %o, attendu 600", perm)
	}
}
