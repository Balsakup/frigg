package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database"
)

func newStore(t *testing.T) *database.Store {
	t.Helper()
	store, err := database.Open(filepath.Join(t.TempDir(), "frigg.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	if err := database.Migrate(context.Background(), store.WriteDB); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestCreateAdmin(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	err := CreateAdmin(ctx, store, AdminInput{AssociationName: " Les Sentiers du Val ", Email: " Claire@Exemple.fr", Password: "Motdepasse1"})
	if err != nil {
		t.Fatal(err)
	}
	a, err := store.Read.GetAccountByEmail(ctx, "claire@exemple.fr")
	if err != nil {
		t.Fatal(err)
	}
	if a.Role != auth.RoleAdmin || !a.EmailVerifiedAt.Valid || !auth.VerifyPassword("Motdepasse1", a.PasswordHash) {
		t.Fatalf("compte inattendu : %+v", a)
	}
	name, _ := store.Read.GetAssociation(ctx)
	if name.String != "Les Sentiers du Val" {
		t.Fatalf("nom de l'association : %q", name.String)
	}

	// Nom déjà défini : non redemandé, conservé.
	if err := CreateAdmin(ctx, store, AdminInput{Email: "thomas@exemple.fr", Password: "Motdepasse1"}); err != nil {
		t.Fatalf("second administrateur : %v", err)
	}
	if name, _ := store.Read.GetAssociation(ctx); name.String != "Les Sentiers du Val" {
		t.Fatalf("nom modifié : %q", name.String)
	}
}

func TestCreateAdminRejects(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	if err := CreateAdmin(ctx, store, AdminInput{AssociationName: "A", Email: "claire@exemple.fr", Password: "Motdepasse1"}); err != nil {
		t.Fatal(err)
	}

	err := CreateAdmin(ctx, store, AdminInput{Email: "CLAIRE@exemple.fr", Password: "Autremotdepasse2"})
	if err == nil || err.Error() != "un compte existe déjà avec l'adresse claire@exemple.fr" {
		t.Fatalf("adresse en double : %v", err)
	}

	err = CreateAdmin(ctx, store, AdminInput{Email: "thomas@exemple.fr", Password: "court"})
	if err == nil || !strings.Contains(err.Error(), auth.MsgPasswordTooShort) || !strings.Contains(err.Error(), auth.MsgPasswordNoUpper) {
		t.Fatalf("mot de passe non conforme : %v", err)
	}
	if _, err := store.Read.GetAccountByEmail(ctx, "thomas@exemple.fr"); err == nil {
		t.Fatal("compte créé malgré l'erreur")
	}
}

func TestCreateAdminRequiresAssociationName(t *testing.T) {
	err := CreateAdmin(context.Background(), newStore(t), AdminInput{Email: "claire@exemple.fr", Password: "Motdepasse1"})
	if err == nil {
		t.Fatal("nom d'association vide accepté")
	}
}
