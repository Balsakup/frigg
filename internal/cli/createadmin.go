package cli

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/config"
	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// AdminInput est la saisie de create-admin.
type AdminInput struct {
	AssociationName string // ignoré si le nom est déjà défini
	Email           string
	Password        string
}

// CreateAdmin crée le premier administrateur (FR-016) et, si besoin, enregistre
// le nom de l'association, dans une seule transaction.
func CreateAdmin(ctx context.Context, store *database.Store, in AdminInput) error {
	email := auth.NormalizeEmail(in.Email)
	if !auth.ValidateEmail(email) {
		return fmt.Errorf("l'adresse électronique %q n'est pas valide", in.Email)
	}
	if msgs := auth.ValidatePassword(in.Password); len(msgs) > 0 {
		return errors.New(strings.Join(msgs, "\n"))
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	return store.WriteTx(ctx, func(q *sqlc.Queries) error {
		name, err := q.GetAssociation(ctx)
		if err != nil {
			return err
		}
		if !name.Valid {
			assoc := strings.TrimSpace(in.AssociationName)
			if assoc == "" || len([]rune(assoc)) > 120 {
				return errors.New("le nom de l'association doit contenir de 1 à 120 caractères")
			}
			if err := q.SetAssociationName(ctx, sqlc.SetAssociationNameParams{
				Name: sql.NullString{String: assoc, Valid: true}, UpdatedAt: now,
			}); err != nil {
				return err
			}
		}
		if _, err := q.GetAccountByEmail(ctx, email); err == nil {
			return fmt.Errorf("un compte existe déjà avec l'adresse %s", email)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		_, err = q.CreateAccount(ctx, sqlc.CreateAccountParams{
			Email:           email,
			PasswordHash:    hash,
			Role:            auth.RoleAdmin,
			EmailVerifiedAt: sql.NullString{String: now, Valid: true},
			CreatedAt:       now,
		})
		return err
	})
}

// CreateAdminCommand est la commande interactive `frigg create-admin`.
func CreateAdminCommand(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("create-admin", flag.ContinueOnError)
	fs.SetOutput(stderr)
	emailFlag := fs.String("email", "", "adresse électronique de l'administrateur")
	if err := fs.Parse(args); err != nil {
		return err
	}
	stdin := int(os.Stdin.Fd())
	if !term.IsTerminal(stdin) {
		return errors.New("create-admin doit être lancé dans un terminal")
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
	if err := database.Migrate(ctx, store.WriteDB); err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	ask := func(prompt string) (string, error) {
		fmt.Fprint(stdout, prompt)
		line, err := reader.ReadString('\n')
		return strings.TrimSpace(line), err
	}
	askSecret := func(prompt string) (string, error) {
		fmt.Fprint(stdout, prompt)
		b, err := term.ReadPassword(stdin)
		fmt.Fprintln(stdout)
		return string(b), err
	}

	var in AdminInput
	name, err := store.Read.GetAssociation(ctx)
	if err != nil {
		return err
	}
	if !name.Valid {
		if in.AssociationName, err = ask("Nom de l'association : "); err != nil {
			return err
		}
	}
	in.Email = *emailFlag
	if in.Email == "" {
		if in.Email, err = ask("Adresse électronique : "); err != nil {
			return err
		}
	}
	if in.Password, err = askSecret("Mot de passe : "); err != nil {
		return err
	}
	confirm, err := askSecret("Confirmez le mot de passe : ")
	if err != nil {
		return err
	}
	if confirm != in.Password {
		return errors.New("les deux mots de passe ne correspondent pas")
	}

	if err := CreateAdmin(ctx, store, in); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Administrateur %s créé.\n", auth.NormalizeEmail(in.Email))
	return nil
}
