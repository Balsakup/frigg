package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureWritableDirCreates(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data", "files")
	if err := ensureWritableDir(dir); err != nil {
		t.Fatalf("répertoire absent : %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("répertoire non créé : %v", err)
	}
	if perm := info.Mode().Perm(); perm&^0o750 != 0 {
		t.Errorf("droits %o, attendu au plus 0750", perm)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("fichier témoin non supprimé : %v", entries)
	}
}

func TestEnsureWritableDirReadOnly(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root peut écrire dans un répertoire en lecture seule")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	err := ensureWritableDir(dir)
	if err == nil {
		t.Fatal("erreur attendue pour un répertoire en lecture seule")
	}
	for _, want := range []string{"n'est pas accessible en écriture par l'utilisateur", "UID 65532"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message %q ne contient pas %q", err, want)
		}
	}
}
