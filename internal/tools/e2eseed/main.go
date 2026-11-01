// Commande e2eseed : prépare une base jetable pour les tests de bout en bout.
// Elle ne fait pas partie du binaire frigg.
//
// Utilisation : go run ./internal/tools/e2eseed <chemin de la base> [adhesions]
//
// Le mode « adhesions » prépare la base du second serveur e2e : une saison
// ouverte contenant la date du jour, des tarifs, des fiches, des adhésions et
// les moyens de paiement (research.md 006 › R13). Les saisons étant un état
// global, le serveur principal reste sans saison (test des saisons, 003).
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
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
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
	// Gestion des comptes (002) : un administrateur par fichier e2e.
	{"admin-comptes@exemple.fr", auth.RoleAdmin, true},
	{"admin-desactivation@exemple.fr", auth.RoleAdmin, true},
	{"admin-role@exemple.fr", auth.RoleAdmin, true},
	{"admin-suppression@exemple.fr", auth.RoleAdmin, true},
	{"admin-lien@exemple.fr", auth.RoleAdmin, true},
	{"membre-a-desactiver@exemple.fr", auth.RoleMember, true},
	{"membre-desactive@exemple.fr", auth.RoleMember, true},
	{"membre-a-promouvoir@exemple.fr", auth.RoleMember, true},
	{"admin-a-retrograder@exemple.fr", auth.RoleAdmin, true},
	{"admin-a-supprimer@exemple.fr", auth.RoleAdmin, true},
	{"membre-lien@exemple.fr", auth.RoleMember, true},
	{"hélène.lefèvre@exemple.fr", auth.RoleMember, true},
	// Gestion des saisons (003) : aucune saison n'est créée, les tests partent
	// d'une base sans saison.
	{"admin-saisons@exemple.fr", auth.RoleAdmin, true},
	// Gestion des membres (005) : chaque test part de comptes sans fiche.
	{"membre-fiches@exemple.fr", auth.RoleMember, true},
	{"membre-fiches-voisin@exemple.fr", auth.RoleMember, true},
	{"admin-membres@exemple.fr", auth.RoleAdmin, true},
	{"membre-rattachement@exemple.fr", auth.RoleMember, true},
	{"admin-suppression-fiches@exemple.fr", auth.RoleAdmin, true},
	{"membre-fiches-conservees@exemple.fr", auth.RoleMember, true},
	{"membre-fiches-effacees@exemple.fr", auth.RoleMember, true},
}

// listAccounts est le nombre de comptes membre-liste-NN (pagination).
const listAccounts = 30

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 || (len(os.Args) == 3 && os.Args[2] != "adhesions") {
		fmt.Fprintln(os.Stderr, "utilisation : e2eseed <chemin de la base> [adhesions]")
		os.Exit(2)
	}
	if err := run(os.Args[1], len(os.Args) == 3); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string, memberships bool) error {
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
		for i := 1; i <= listAccounts; i++ {
			email := fmt.Sprintf("membre-liste-%02d@exemple.fr", i)
			if _, err := q.CreateAccount(ctx, sqlc.CreateAccountParams{Email: email, PasswordHash: hash, Role: auth.RoleMember, CreatedAt: now}); err != nil {
				return fmt.Errorf("compte %s : %w", email, err)
			}
		}

		disabled, err := q.GetAccountByEmail(ctx, "membre-desactive@exemple.fr")
		if err != nil {
			return err
		}
		if _, err := q.DisableAccount(ctx, sqlc.DisableAccountParams{ID: disabled.ID, Now: sql.NullString{String: now, Valid: true}}); err != nil {
			return err
		}

		// Invitation en attente envoyée par le compte que le test de suppression efface.
		author, err := q.GetAccountByEmail(ctx, "admin-a-supprimer@exemple.fr")
		if err != nil {
			return err
		}
		_, tokenHash := auth.NewToken()
		_, err = q.CreateInvitation(ctx, sqlc.CreateInvitationParams{
			Email:     "invite-par-supprime@exemple.fr",
			Role:      auth.RoleMember,
			TokenHash: tokenHash,
			InvitedBy: sql.NullInt64{Int64: author.ID, Valid: true},
			ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour).Format("2006-01-02T15:04:05Z"),
			Now:       now,
		})
		if err != nil || !memberships {
			return err
		}
		return seedMemberships(ctx, q, hash, now)
	})
}

// Comptes du serveur des adhésions : un compte par test (limite de connexions).
var membershipAccounts = []struct {
	email, role string
	members     [][2]string // prénom, nom
}{
	{"admin-adhesions@exemple.fr", auth.RoleAdmin, nil},
	{"admin-adhesions-creation@exemple.fr", auth.RoleAdmin, nil},
	{"admin-adhesions-correction@exemple.fr", auth.RoleAdmin, nil},
	{"admin-adhesions-chiffres@exemple.fr", auth.RoleAdmin, nil},
	{"admin-adhesions-parametres@exemple.fr", auth.RoleAdmin, nil},
	{"membre-adhesions-demande@exemple.fr", auth.RoleMember, [][2]string{{"Léo", "Martin"}, {"Inès", "Martin"}}},
	{"membre-adhesions-paiement@exemple.fr", auth.RoleMember, [][2]string{{"Hugo", "Bernard"}}},
	{"membre-adhesions-voisin@exemple.fr", auth.RoleMember, [][2]string{{"Zélie", "Roux"}}},
	{"membre-adhesions-decision@exemple.fr", auth.RoleMember, [][2]string{{"Théo", "Durand"}, {"Zoé", "Durand"}, {"Marc", "Durand"}}},
}

// seedMemberships prépare le jeu de données des adhésions. L'adhésion de Hugo
// Bernard (à payer) est créée en premier : son identifiant est 1.
func seedMemberships(ctx context.Context, q *sqlc.Queries, hash, now string) error {
	today := season.Today(time.Now())
	start := season.MustDate(fmt.Sprintf("%04d-%02d-01", today.Year(), int(today.Month())))
	end := start.AddMonths(12).AddDays(-1)
	s, err := q.CreateSeason(ctx, sqlc.CreateSeasonParams{StartsOn: start.String(), EndsOn: end.String(), CreatedAt: now})
	if err != nil {
		return err
	}
	if _, err := q.OpenSeason(ctx, sqlc.OpenSeasonParams{Now: sql.NullString{String: now, Valid: true}, ID: s.ID}); err != nil {
		return err
	}
	fee := func(label string, amount int64, renewal sql.NullInt64) (int64, error) {
		f, err := q.CreateMembershipFee(ctx, sqlc.CreateMembershipFeeParams{
			SeasonID: s.ID, Label: label, AmountCents: amount, RenewalAmountCents: renewal, Now: now,
		})
		return f.ID, err
	}
	adult, err := fee("Adulte", 6000, sql.NullInt64{Int64: 4000, Valid: true})
	if err != nil {
		return err
	}
	if _, err := fee("Jeune", 3000, sql.NullInt64{}); err != nil {
		return err
	}
	if _, err := fee("Bienfaiteur", 0, sql.NullInt64{}); err != nil {
		return err
	}
	if err := q.UpdatePaymentSettings(ctx, sqlc.UpdatePaymentSettingsParams{
		CashEnabled: 1, CashInstructions: sql.NullString{String: "Au local de l'association,\nle samedi de 10 h à 12 h.", Valid: true},
		ChequeEnabled: 1, ChequePayee: sql.NullString{String: "Les Sentiers du Val", Valid: true},
		ChequeAddress: sql.NullString{String: "12 rue des Lilas\n35000 Rennes", Valid: true},
		TransferEnabled: 1, TransferHolder: sql.NullString{String: "Les Sentiers du Val", Valid: true},
		TransferIban: sql.NullString{String: "FR7630006000011234567890189", Valid: true},
		TransferBic:  sql.NullString{String: "BNPAFRPP", Valid: true},
		Now:          now,
	}); err != nil {
		return err
	}

	members := map[string]int64{}
	accounts := map[string]int64{}
	for _, a := range membershipAccounts {
		acc, err := q.CreateAccount(ctx, sqlc.CreateAccountParams{
			Email: a.email, PasswordHash: hash, Role: a.role, CreatedAt: now,
			EmailVerifiedAt: sql.NullString{String: now, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("compte %s : %w", a.email, err)
		}
		accounts[a.email] = acc.ID
		for _, m := range a.members {
			r, err := q.CreateMemberForAccount(ctx, sqlc.CreateMemberForAccountParams{
				AccountID: sql.NullInt64{Int64: acc.ID, Valid: true}, FirstName: m[0], LastName: m[1],
				BirthDate: "2012-03-15", Now: now,
			})
			if err != nil {
				return err
			}
			members[m[0]] = r.ID
		}
	}
	if _, err := q.CreateMember(ctx, sqlc.CreateMemberParams{FirstName: "Jeanne", LastName: "Petit", BirthDate: "1975-06-01", Now: now}); err != nil {
		return err
	}

	admin := sql.NullInt64{Int64: accounts["admin-adhesions@exemple.fr"], Valid: true}
	str := func(v string) sql.NullString { return sql.NullString{String: v, Valid: true} }
	ref := membership.NextReference(membership.ReferencePrefix(start, end), nil)
	// 1. Hugo Bernard : validée, à payer (paiement en ligne de l'adhérent).
	paying := sql.NullInt64{Int64: accounts["membre-adhesions-paiement@exemple.fr"], Valid: true}
	if _, err := q.CreateMembership(ctx, sqlc.CreateMembershipParams{
		MemberID: members["Hugo"], SeasonID: s.ID, FeeID: adult, Status: "validated",
		DeclaredKind: str("first"), Kind: str("first"), AmountCents: sql.NullInt64{Int64: 6000, Valid: true},
		RequestedBy: paying, Now: now, DecidedBy: admin, DecidedAt: str(now),
		PaymentReference: str(ref), PaymentStatus: str("due"),
	}); err != nil {
		return err
	}
	// Théo et Zoé Durand : demandes en attente ; Marc Durand : validée, à payer.
	deciding := sql.NullInt64{Int64: accounts["membre-adhesions-decision@exemple.fr"], Valid: true}
	for _, first := range []string{"Théo", "Zoé"} {
		if _, err := q.CreateMembership(ctx, sqlc.CreateMembershipParams{
			MemberID: members[first], SeasonID: s.ID, FeeID: adult, Status: "pending",
			DeclaredKind: str("renewal"), RequestedBy: deciding, Now: now,
		}); err != nil {
			return err
		}
	}
	_, err = q.CreateMembership(ctx, sqlc.CreateMembershipParams{
		MemberID: members["Marc"], SeasonID: s.ID, FeeID: adult, Status: "validated",
		Kind: str("first"), AmountCents: sql.NullInt64{Int64: 6000, Valid: true},
		CreatedBy: admin, Now: now, DecidedBy: admin, DecidedAt: str(now),
		PaymentReference: str(membership.NextReference(membership.ReferencePrefix(start, end), []string{ref})),
		PaymentStatus:    str("due"),
	})
	return err
}
