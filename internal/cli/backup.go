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

// Backup écrit une copie cohérente de la base, à chaud, dans dest (VACUUM INTO).
func Backup(ctx context.Context, store *database.Store, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("le fichier %s existe déjà", dest)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := store.WriteDB.ExecContext(ctx, "VACUUM INTO ?", dest); err != nil {
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
