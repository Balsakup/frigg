// Commande e2eseed : prépare une base jetable pour les tests de bout en bout.
// Elle ne fait pas partie du binaire frigg.
//
// Utilisation : go run ./internal/tools/e2eseed <chemin de la base>
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Password est le mot de passe de tous les comptes de test.
const Password = "Motdepasse1"

// Un compte par test : les tests tournent en parallèle et la limite de
// connexions par adresse (5 par quart d'heure) s'applique aussi à eux.
var accounts = []struct {
	email    string
	role     string
	verified bool
}{
	{"admin@exemple.fr", auth.RoleAdmin, true},
	{"membre@exemple.fr", auth.RoleMember, true},
	{"membre-connexion-1@exemple.fr", auth.RoleMember, true},
	{"membre-connexion-2@exemple.fr", auth.RoleMember, true},
	{"membre-connexion-3@exemple.fr", auth.RoleMember, true},
	{"membre-navigation-1@exemple.fr", auth.RoleMember, true},
	{"membre-navigation-2@exemple.fr", auth.RoleMember, true},
	{"membre-navigation-3@exemple.fr", auth.RoleMember, true},
	{"admin-navigation@exemple.fr", auth.RoleAdmin, true},
	{"admin-invitation@exemple.fr", auth.RoleAdmin, true},
	{"membre-reinitialisation@exemple.fr", auth.RoleMember, true},
	{"membre-compte@exemple.fr", auth.RoleMember, true},
	{"membre-non-confirme@exemple.fr", auth.RoleMember, false},
	{"membre-clavier@exemple.fr", auth.RoleMember, true},
	{"admin-clavier@exemple.fr", auth.RoleAdmin, true},
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "utilisation : e2eseed <chemin de la base>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string) error {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := os.Remove(path + suffix); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	ctx := context.Background()
	store, err := database.Open(path)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := database.Migrate(ctx, store.WriteDB); err != nil {
		return err
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	hash, err := auth.HashPassword(Password)
	if err != nil {
		return err
	}
	return store.WriteTx(ctx, func(q *sqlc.Queries) error {
		if err := q.SetAssociationName(ctx, sqlc.SetAssociationNameParams{
			Name:      sql.NullString{String: "Les Sentiers du Val", Valid: true},
			UpdatedAt: now,
		}); err != nil {
			return err
		}
		for _, a := range accounts {
			p := sqlc.CreateAccountParams{Email: a.email, PasswordHash: hash, Role: a.role, CreatedAt: now}
			if a.verified {
				p.EmailVerifiedAt = sql.NullString{String: now, Valid: true}
			}
			if _, err := q.CreateAccount(ctx, p); err != nil {
				return fmt.Errorf("compte %s : %w", a.email, err)
			}
		}
		return nil
	})
}
