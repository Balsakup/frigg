package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/balsakup/frigg/internal/config"
	"github.com/balsakup/frigg/internal/database"
)

// Backup écrit une copie cohérente de la base, à chaud, dans dest (VACUUM INTO),
// lisible par son seul propriétaire.
func Backup(ctx context.Context, store *database.Store, dest string) error {
	// Fichier créé vide en 0600 avant VACUUM INTO (qui accepte un fichier vide) :
	// la sauvegarde contient les empreintes de mots de passe et les données
	// personnelles, elle n'est jamais lisible par d'autres utilisateurs.
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("le fichier %s existe déjà", dest)
	}
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if _, err := store.WriteDB.ExecContext(ctx, "VACUUM INTO ?", dest); err != nil {
		_ = os.Remove(dest)
		return fmt.Errorf("sauvegarde impossible : %w", err)
	}
	return nil
}

// BackupCommand est la commande `frigg backup <fichier>`.
func BackupCommand(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return errors.New("utilisation : frigg backup <fichier>")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	store, err := database.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := Backup(ctx, store, args[0]); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Base sauvegardée dans %s.\nPensez à sauvegarder aussi le répertoire des fichiers déposés : %s\n", args[0], cfg.DataDir)
	return nil
}
