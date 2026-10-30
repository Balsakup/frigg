package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate applique les migrations embarquées. Un échec doit empêcher le
// démarrage (principe V).
func Migrate(ctx context.Context, db *sql.DB) error {
	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, fsys)
	if err != nil {
		return fmt.Errorf("préparation des migrations : %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("application des migrations : %w", err)
	}
	return nil
}
