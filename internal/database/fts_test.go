package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/database/sqlc"
)

func TestFTSTrigramQuery(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", "", false},
		{"  ", "", false},
		{"du", "", false},
		{"a b", "", false},
		{" pont ", `"pont"`, true},
		{"exemple  dup", `"exemple" "dup"`, true},
		{"de pont", `"pont"`, true},
		{"hél", `"hél"`, true},
		{`a"bc`, `"a""bc"`, true},
		{"OR NEAR dup*", `"NEAR" "dup*"`, true},
	}
	for _, c := range cases {
		got, ok := FTSTrigramQuery(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("FTSTrigramQuery(%q) = (%q, %v), attendu (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestAccountsSearch(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "frigg.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := Migrate(ctx, store.WriteDB); err != nil {
		t.Fatal(err)
	}
	ids := map[string]int64{}
	for _, email := range []string{"jean.dupont@exemple.fr", "hélène.lefèvre@asso.fr", "marc.dupré@exemple.fr"} {
		a, err := store.Write.CreateAccount(ctx, sqlc.CreateAccountParams{Email: email, PasswordHash: "x", Role: "member", CreatedAt: "2026-10-31T00:00:00Z"})
		if err != nil {
			t.Fatal(err)
		}
		ids[email] = a.ID
	}

	search := func(input string) []string {
		t.Helper()
		q, ok := FTSTrigramQuery(input)
		if !ok {
			t.Fatalf("saisie %q inutilisable", input)
		}
		rows, err := store.Read.ListAccounts(ctx, sqlc.ListAccountsParams{Query: sql.NullString{String: q, Valid: true}, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, r := range rows {
			got = append(got, r.Email)
		}
		return got
	}

	for input, want := range map[string]string{
		"pont":        "jean.dupont@exemple.fr",
		"n.dup":       "jean.dupont@exemple.fr",
		"LEFÈ":        "hélène.lefèvre@asso.fr",
		"dupre":       "marc.dupré@exemple.fr",
		"exemple dup": "jean.dupont@exemple.fr,marc.dupré@exemple.fr",
	} {
		if got := strings.Join(search(input), ","); got != want {
			t.Errorf("recherche %q : %q, attendu %q", input, got, want)
		}
	}

	// Les triggers suivent les modifications et suppressions.
	if _, err := store.WriteDB.Exec("UPDATE accounts SET email = 'jean.durand@exemple.fr' WHERE id = ?", ids["jean.dupont@exemple.fr"]); err != nil {
		t.Fatal(err)
	}
	if _, err := store.WriteDB.Exec("DELETE FROM accounts WHERE id = ?", ids["marc.dupré@exemple.fr"]); err != nil {
		t.Fatal(err)
	}
	if got := search("dup"); len(got) != 0 {
		t.Errorf("recherche dup après modification : %v, attendu aucun résultat", got)
	}
	if got := strings.Join(search("durand"), ","); got != "jean.durand@exemple.fr" {
		t.Errorf("recherche durand : %q", got)
	}
	if _, err := store.WriteDB.Exec("INSERT INTO accounts_fts (accounts_fts) VALUES ('integrity-check')"); err != nil {
		t.Errorf("index incohérent : %v", err)
	}
}
