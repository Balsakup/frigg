package communication

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileStore(t *testing.T) {
	s, err := NewFileStore(filepath.Join(t.TempDir(), "communications"))
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("%PDF-1.7 contenu")
	stored, size, sum, err := s.Save(bytes.NewReader(data), MaxFileBytes)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(data)
	if !ValidStoredName(stored) || size != int64(len(data)) || sum != hex.EncodeToString(h[:]) {
		t.Fatalf("Save : %q %d %q", stored, size, sum)
	}
	entries, _ := os.ReadDir(s.Dir)
	if len(entries) != 1 || entries[0].Name() != stored {
		t.Fatalf("répertoire : %v (aucun fichier temporaire ne doit rester)", entries)
	}
	if info, _ := os.Stat(filepath.Join(s.Dir, stored)); info.Mode().Perm() != 0o640 {
		t.Errorf("permissions %v", info.Mode().Perm())
	}
	got, err := s.Read(stored, size, sum)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("Read : %q %v", got, err)
	}
	if err := s.Verify(stored, size, sum); err != nil {
		t.Fatal(err)
	}
	if err := s.Verify(stored, size, strings.Repeat("0", 64)); !errors.Is(err, ErrCorrupted) {
		t.Errorf("empreinte différente : %v", err)
	}
	if _, err := s.Open("../" + stored); !errors.Is(err, ErrInvalidName) {
		t.Errorf("nom invalide : %v", err)
	}
	if err := s.Remove(stored); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove(stored); err != nil {
		t.Errorf("suppression idempotente : %v", err)
	}
	if err := s.Remove("../../etc/passwd"); !errors.Is(err, ErrInvalidName) {
		t.Errorf("suppression d'un nom invalide : %v", err)
	}
}

func TestFileStoreLimits(t *testing.T) {
	s, _ := NewFileStore(t.TempDir())
	if _, _, _, err := s.Save(bytes.NewReader(make([]byte, 11)), 10); !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("trop gros : %v", err)
	}
	if _, _, _, err := s.Save(bytes.NewReader(nil), 10); !errors.Is(err, ErrEmptyFile) {
		t.Errorf("vide : %v", err)
	}
	if entries, _ := os.ReadDir(s.Dir); len(entries) != 0 {
		t.Errorf("fichiers restants : %v", entries)
	}
}

func TestPurgeOrphans(t *testing.T) {
	s, _ := NewFileStore(t.TempDir())
	known, _, _, _ := s.Save(strings.NewReader("a"), 10)
	old, _, _, _ := s.Save(strings.NewReader("b"), 10)
	recent, _, _, _ := s.Save(strings.NewReader("c"), 10)
	tmp := filepath.Join(s.Dir, tempPrefix+"x")
	other := filepath.Join(s.Dir, "autre.txt")
	for _, p := range []string{tmp, other} {
		_ = os.WriteFile(p, []byte("x"), 0o640)
	}
	past := time.Now().Add(-2 * time.Hour)
	for _, p := range []string{filepath.Join(s.Dir, known), filepath.Join(s.Dir, old), tmp, other} {
		_ = os.Chtimes(p, past, past)
	}
	n, err := s.PurgeOrphans(map[string]bool{known: true}, time.Now().Add(-time.Hour))
	if err != nil || n != 2 {
		t.Fatalf("PurgeOrphans : %d %v", n, err)
	}
	for name, want := range map[string]bool{known: true, old: false, recent: true, tempPrefix + "x": false, "autre.txt": true} {
		_, err := os.Stat(filepath.Join(s.Dir, name))
		if (err == nil) != want {
			t.Errorf("%s : présent=%v, attendu %v", name, err == nil, want)
		}
	}
}
