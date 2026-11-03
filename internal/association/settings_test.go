package association

import (
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/season"
)

func ptr(s string) *string { return &s }

func today(t *testing.T) season.Date {
	t.Helper()
	d, err := season.ParseDate("2026-11-03")
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func fields(errs []FieldError) map[string]string {
	m := map[string]string{}
	for _, e := range errs {
		m[e.Field] = e.Message
	}
	return m
}

func TestNormalizeName(t *testing.T) {
	for _, c := range []struct {
		name, msg string
	}{
		{"Les Sentiers du Val", ""},
		{"  Les Sentiers  ", ""},
		{"", MsgNameRequired},
		{"   ", MsgNameRequired},
		{strings.Repeat("é", 120), ""},
		{strings.Repeat("é", 121), MsgNameTooLong},
	} {
		out, errs := Normalize(Settings{Name: c.name}, today(t))
		if got := fields(errs)["name"]; got != c.msg {
			t.Errorf("nom %q : erreur %q, attendu %q", c.name, got, c.msg)
		}
		if c.msg == "" && out.Name != strings.TrimSpace(c.name) {
			t.Errorf("nom %q normalisé en %q", c.name, out.Name)
		}
	}
}

func TestNormalizeTextFields(t *testing.T) {
	out, errs := Normalize(Settings{
		Name:      "Club",
		Tagline:   ptr("  Randonnées depuis 1987  "),
		ShortName: ptr("   "),
		Purpose:   ptr(""),
	}, today(t))
	if len(errs) != 0 {
		t.Fatalf("erreurs inattendues : %v", errs)
	}
	if out.Tagline == nil || *out.Tagline != "Randonnées depuis 1987" {
		t.Errorf("tagline : %v", out.Tagline)
	}
	if out.ShortName != nil || out.Purpose != nil {
		t.Errorf("champs vides non nuls : %v %v", out.ShortName, out.Purpose)
	}

	_, errs = Normalize(Settings{Name: "Club", Tagline: ptr(strings.Repeat("a", 150))}, today(t))
	if len(errs) != 0 {
		t.Errorf("tagline de 150 caractères refusée : %v", errs)
	}
	_, errs = Normalize(Settings{
		Name:      "",
		Tagline:   ptr(strings.Repeat("a", 151)),
		ShortName: ptr(strings.Repeat("a", 21)),
		Purpose:   ptr(strings.Repeat("a", 2001)),
	}, today(t))
	f := fields(errs)
	for _, k := range []string{"name", "tagline", "shortName", "purpose"} {
		if f[k] == "" {
			t.Errorf("pas d'erreur sur %s : %v", k, errs)
		}
	}
}

func TestNormalizeLegalAndContact(t *testing.T) {
	out, errs := Normalize(Settings{
		Name:         "Club",
		RNA:          ptr("w751234567"),
		SIRET:        ptr("732 829 320 00074"),
		FoundedOn:    ptr("2026-11-03"),
		Address:      Address{Line1: ptr("1 rue du Moulin"), PostalCode: ptr("75001"), City: ptr("Paris")},
		ContactEmail: ptr(" bureau@sentiers.fr "),
		Phone:        ptr("+33 1 23 45 67 89"),
		Website:      ptr("https://sentiers.fr"),
	}, today(t))
	if len(errs) != 0 {
		t.Fatalf("erreurs inattendues : %v", errs)
	}
	if *out.RNA != "W751234567" || *out.SIRET != "73282932000074" || *out.Phone != "0123456789" || *out.ContactEmail != "bureau@sentiers.fr" {
		t.Errorf("normalisation : %+v", out)
	}

	_, errs = Normalize(Settings{
		Name:         "Club",
		SIRET:        ptr("73282932000075"),
		FoundedOn:    ptr("2026-11-04"),
		Website:      ptr("www.exemple.fr"),
		Address:      Address{PostalCode: ptr("7500")},
		ContactEmail: ptr("bureau@"),
		Phone:        ptr("123"),
		RNA:          ptr("W12"),
	}, today(t))
	f := fields(errs)
	want := map[string]string{
		"siret": MsgSIRETInvalid, "foundedOn": MsgFoundedInFuture, "website": MsgWebURLInvalid,
		"address.postalCode": MsgPostalCode, "contactEmail": MsgEmailInvalid, "phone": MsgPhoneInvalid, "rna": MsgRNAInvalid,
	}
	for k, v := range want {
		if f[k] != v {
			t.Errorf("%s : %q, attendu %q", k, f[k], v)
		}
	}
	if len(errs) != len(want) {
		t.Errorf("erreurs : %v", errs)
	}

	_, errs = Normalize(Settings{Name: "Club", FoundedOn: ptr("2026-02-30")}, today(t))
	if fields(errs)["foundedOn"] != MsgDateInvalid {
		t.Errorf("date invalide acceptée : %v", errs)
	}
}

func TestNormalizeSocial(t *testing.T) {
	out, errs := Normalize(Settings{Name: "Club", Social: Social{
		Instagram: ptr("https://instagram.com/club"),
		Facebook:  ptr(""),
		Bluesky:   ptr("instagram.com/x"),
	}}, today(t))
	if fields(errs)["social.bluesky"] != MsgWebURLInvalid || len(errs) != 1 {
		t.Errorf("erreurs : %v", errs)
	}
	if out.Social.Instagram == nil || out.Social.Facebook != nil {
		t.Errorf("liens : %+v", out.Social)
	}
}
