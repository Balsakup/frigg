package cli

import (
	"fmt"
	"os"
)

// ensureWritableDir crée dir au besoin puis vérifie qu'on peut y écrire, pour
// qu'un volume mal attribué arrête le démarrage avec un message clair plutôt
// qu'une erreur SQLite ou un échec tardif au premier dépôt de fichier.
func ensureWritableDir(dir string) error {
	err := os.MkdirAll(dir, 0o750)
	if err == nil {
		var f *os.File
		if f, err = os.CreateTemp(dir, ".frigg-write-check-*"); err == nil {
			name := f.Name()
			err = f.Close()
			if rmErr := os.Remove(name); err == nil {
				err = rmErr
			}
		}
	}
	if err != nil {
		return fmt.Errorf("le répertoire %s n'est pas accessible en écriture par l'utilisateur %d : %w "+
			"(en conteneur, le volume doit appartenir à l'UID 65532)", dir, os.Getuid(), err)
	}
	return nil
}
