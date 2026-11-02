// Commande e2eseed : prépare une base jetable pour les tests de bout en bout.
// Elle ne fait pas partie du binaire frigg.
//
// Utilisation : go run ./internal/tools/e2eseed <chemin de la base> [adhesions|carte]
//
// Le mode « adhesions » prépare la base du second serveur e2e : une saison
// ouverte contenant la date du jour, des tarifs, des fiches, des adhésions et
// les moyens de paiement (research.md 006 › R13). Les saisons étant un état
// global, le serveur principal reste sans saison (test des saisons, 003).
//
// Le mode « carte » ajoute au jeu des adhésions la carte bancaire activée et
// testée en sandbox, et des adhésions à payer par carte (research.md 008 › R15).
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
	mode := ""
	if len(os.Args) == 3 {
		mode = os.Args[2]
	}
	if len(os.Args) < 2 || len(os.Args) > 3 || (mode != "" && mode != "adhesions" && mode != "carte") {
		fmt.Fprintln(os.Stderr, "utilisation : e2eseed <chemin de la base> [adhesions|carte]")
		os.Exit(2)
	}
	if err := run(os.Args[1], mode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path, mode string) error {
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
		if err != nil || mode == "" {
			return err
		}
		if err := seedMemberships(ctx, q, hash, now); err != nil || mode != "carte" {
			return err
		}
		return seedCard(ctx, q, hash, now)
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

// Comptes du serveur du paiement par carte (008) : une adhésion « Jeune »
// (30 €) validée à payer par compte.
var cardAccounts = []struct{ email, first, last string }{
	{"membre-carte-paiement@exemple.fr", "Nina", "Leroy"},
	{"membre-carte-refus@exemple.fr", "Paul", "Morel"},
	{"membre-carte-notification@exemple.fr", "Lina", "Fabre"},
	{"membre-carte-clavier@exemple.fr", "Sacha", "Blanc"},
	// Déjà payée par carte (administration, US3).
	{"membre-carte-payee@exemple.fr", "Rémi", "Garnier"},
}

// seedCard active la carte (test de connexion réussi en sandbox) et crée les
// adhésions à payer par carte, ainsi qu'un administrateur.
func seedCard(ctx context.Context, q *sqlc.Queries, hash, now string) error {
	settings, err := q.GetPaymentSettings(ctx)
	if err != nil {
		return err
	}
	if err := q.UpdatePaymentSettings(ctx, sqlc.UpdatePaymentSettingsParams{
		CashEnabled: settings.CashEnabled, CashInstructions: settings.CashInstructions,
		ChequeEnabled: settings.ChequeEnabled, ChequePayee: settings.ChequePayee,
		ChequeAddress: settings.ChequeAddress, ChequeInstructions: settings.ChequeInstructions,
		TransferEnabled: settings.TransferEnabled, TransferHolder: settings.TransferHolder,
		TransferIban: settings.TransferIban, TransferBic: settings.TransferBic,
		TransferInstructions: settings.TransferInstructions, CardEnabled: 1, Now: now,
	}); err != nil {
		return err
	}
	str := func(v string) sql.NullString { return sql.NullString{String: v, Valid: true} }
	if err := q.RecordHelloAssoTest(ctx, sqlc.RecordHelloAssoTestParams{
		Now: str(now), Environment: str("sandbox"), Ok: sql.NullInt64{Int64: 1, Valid: true},
		OrganizationName: str("Les Sentiers du Val (test)"),
	}); err != nil {
		return err
	}
	for _, email := range []string{"admin-carte@exemple.fr", "admin-carte-parametres@exemple.fr"} {
		if _, err := q.CreateAccount(ctx, sqlc.CreateAccountParams{
			Email: email, PasswordHash: hash, Role: auth.RoleAdmin, CreatedAt: now, EmailVerifiedAt: str(now),
		}); err != nil {
			return err
		}
	}
	seasons, err := q.ListSeasons(ctx)
	if err != nil || len(seasons) == 0 {
		return fmt.Errorf("saison des adhésions absente : %w", err)
	}
	s := seasons[0]
	fees, err := q.ListMembershipFees(ctx, s.ID)
	if err != nil {
		return err
	}
	var young int64
	for _, f := range fees {
		if f.Label == "Jeune" {
			young = f.ID
		}
	}
	start, end := season.MustDate(s.StartsOn), season.MustDate(s.EndsOn)
	for _, a := range cardAccounts {
		acc, err := q.CreateAccount(ctx, sqlc.CreateAccountParams{
			Email: a.email, PasswordHash: hash, Role: auth.RoleMember, CreatedAt: now, EmailVerifiedAt: str(now),
		})
		if err != nil {
			return err
		}
		m, err := q.CreateMemberForAccount(ctx, sqlc.CreateMemberForAccountParams{
			AccountID: sql.NullInt64{Int64: acc.ID, Valid: true}, FirstName: a.first, LastName: a.last,
			BirthDate: "2012-03-15", Now: now,
		})
		if err != nil {
			return err
		}
		refs, err := q.ListPaymentReferences(ctx, membership.ReferencePrefix(start, end))
		if err != nil {
			return err
		}
		existing := make([]string, 0, len(refs))
		for _, r := range refs {
			existing = append(existing, r.String)
		}
		ms, err := q.CreateMembership(ctx, sqlc.CreateMembershipParams{
			MemberID: m.ID, SeasonID: s.ID, FeeID: young, Status: "validated",
			DeclaredKind: str("first"), Kind: str("first"), AmountCents: sql.NullInt64{Int64: 3000, Valid: true},
			RequestedBy: sql.NullInt64{Int64: acc.ID, Valid: true}, Now: now, DecidedAt: str(now),
			PaymentReference: str(membership.NextReference(membership.ReferencePrefix(start, end), existing)),
			PaymentStatus:    str("due"),
		})
		if err != nil {
			return err
		}
		if a.first != "Rémi" {
			continue
		}
		op, err := q.CreateOnlinePayment(ctx, sqlc.CreateOnlinePaymentParams{
			MembershipID: ms, AccountID: sql.NullInt64{Int64: acc.ID, Valid: true}, Environment: "sandbox",
			AmountCents: 3000, Now: now,
		})
		if err != nil {
			return err
		}
		if _, err := q.SucceedOnlinePayment(ctx, sqlc.SucceedOnlinePaymentParams{
			OrderID: sql.NullInt64{Int64: 77777, Valid: true}, HelloassoPaymentID: sql.NullInt64{Int64: 77778, Valid: true},
			ReceivedAmountCents: sql.NullInt64{Int64: 3000, Valid: true}, PaidAt: str(now), Now: str(now), ID: op.ID,
		}); err != nil {
			return err
		}
		if _, err := q.RecordCardPayment(ctx, sqlc.RecordCardPaymentParams{
			PaidOn: str(season.Today(time.Now()).String()), Now: str(now),
			PaidAmountCents: sql.NullInt64{Int64: 3000, Valid: true}, OnlinePaymentID: sql.NullInt64{Int64: op.ID, Valid: true}, ID: ms,
		}); err != nil {
			return err
		}
	}
	return nil
}
