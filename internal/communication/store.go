package communication

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrInvalidName signale un nom de fichier qui n'est pas un nom généré.
var ErrInvalidName = errors.New("nom de fichier invalide")

// ErrCorrupted signale un fichier dont la taille ou l'empreinte a changé.
var ErrCorrupted = errors.New("fichier altéré")

// FileStore range les pièces jointes dans un répertoire, sous un nom généré
// (constitution, Fichiers ; research.md › R5). Aucun chemin n'est construit
// depuis une saisie.
type FileStore struct {
	Dir string
}

// NewFileStore crée le répertoire au besoin.
func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("répertoire des pièces jointes : %w", err)
	}
	return &FileStore{Dir: dir}, nil
}

const tempPrefix = ".upload-"

// Save écrit r (au plus max octets) dans un fichier temporaire puis le renomme
// sous un nom généré. Elle renvoie ce nom, la taille et l'empreinte SHA-256.
func (s *FileStore) Save(r io.Reader, max int64) (stored string, size int64, sum string, err error) {
	tmp, err := os.CreateTemp(s.Dir, tempPrefix+"*")
	if err != nil {
		return "", 0, "", err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	h := sha256.New()
	size, err = io.Copy(io.MultiWriter(tmp, h), io.LimitReader(r, max+1))
	if err != nil {
		return "", 0, "", err
	}
	if size > max {
		return "", 0, "", ErrFileTooLarge
	}
	if size == 0 {
		return "", 0, "", ErrEmptyFile
	}
	if err = tmp.Chmod(0o640); err != nil {
		return "", 0, "", err
	}
	if err = tmp.Sync(); err != nil {
		return "", 0, "", err
	}
	if err = tmp.Close(); err != nil {
		return "", 0, "", err
	}
	stored = NewStoredName()
	if err = os.Rename(tmp.Name(), filepath.Join(s.Dir, stored)); err != nil {
		return "", 0, "", err
	}
	return stored, size, hex.EncodeToString(h.Sum(nil)), nil
}

func (s *FileStore) path(stored string) (string, error) {
	if !ValidStoredName(stored) {
		return "", ErrInvalidName
	}
	return filepath.Join(s.Dir, stored), nil
}

// Open ouvre un fichier enregistré.
func (s *FileStore) Open(stored string) (*os.File, error) {
	p, err := s.path(stored)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

// Read lit un fichier enregistré et vérifie sa taille et son empreinte.
func (s *FileStore) Read(stored string, size int64, sum string) ([]byte, error) {
	f, err := s.Open(stored)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, size+1))
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(data)
	if int64(len(data)) != size || hex.EncodeToString(h[:]) != sum {
		return nil, ErrCorrupted
	}
	return data, nil
}

// Verify vérifie qu'un fichier enregistré est intact.
func (s *FileStore) Verify(stored string, size int64, sum string) error {
	f, err := s.Open(stored)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, size+1))
	if err != nil {
		return err
	}
	if n != size || hex.EncodeToString(h.Sum(nil)) != sum {
		return ErrCorrupted
	}
	return nil
}

// Remove supprime un fichier enregistré ; un fichier déjà absent n'est pas
// une erreur.
func (s *FileStore) Remove(stored string) error {
	p, err := s.path(stored)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// PurgeOrphans supprime les fichiers du répertoire inconnus de la base
// (known) et les fichiers temporaires, modifiés avant olderThan. Elle renvoie
// le nombre de fichiers supprimés.
func (s *FileStore) PurgeOrphans(known map[string]bool, olderThan time.Time) (int, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || known[name] || !(ValidStoredName(name) || strings.HasPrefix(name, tempPrefix)) {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.ModTime().Before(olderThan) {
			continue
		}
		if err := os.Remove(filepath.Join(s.Dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return n, err
		}
		n++
	}
	return n, nil
}
