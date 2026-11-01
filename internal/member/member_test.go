package member_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/balsakup/frigg/internal/member"
	"github.com/balsakup/frigg/internal/season"
)

var today = season.MustDate("2026-11-01")

func valid() member.Input {
	return member.Input{FirstName: "Léo", LastName: "Martin", BirthDate: season.MustDate("2015-04-12")}
}

// fieldErrs renvoie les couples champ/erreur d'une erreur de validation.
func fieldErrs(t *testing.T, err error) map[string]error {
	t.Helper()
	var fes member.FieldErrors
	if !errors.As(err, &fes) {
		t.Fatalf("FieldErrors attendu, obtenu %v", err)
	}
	out := map[string]error{}
	for _, fe := range fes {
		out[fe.Field] = fe.Err
	}
	return out
}

func TestValidateNames(t *testing.T) {
	in := valid()
	in.FirstName = "  Jean   Pierre "
	in.LastName = " Martin  Durand "
	c, err := member.Validate(in, today)
	if err != nil {
		t.Fatal(err)
	}
	if c.FirstName != "Jean Pierre" || c.LastName != "Martin Durand" {
		t.Fatalf("noms nettoyés : %q %q", c.FirstName, c.LastName)
	}

	in = valid()
	in.FirstName, in.LastName = "   ", ""
	got := fieldErrs(t, func() error { _, err := member.Validate(in, today); return err }())
	if got["firstName"] != member.ErrRequired || got["lastName"] != member.ErrRequired {
		t.Fatalf("erreurs : %v", got)
	}
}

func TestValidateBirthDate(t *testing.T) {
	cases := []struct {
		date string
		want error
	}{
		{"2026-11-01", nil},
		{"2026-11-02", member.ErrBirthFuture},
		{"1906-11-01", nil},
		{"1906-10-31", member.ErrBirthTooOld},
	}
	for _, tc := range cases {
		in := valid()
		in.BirthDate = season.MustDate(tc.date)
		_, err := member.Validate(in, today)
		if tc.want == nil {
			if err != nil {
				t.Errorf("%s : %v", tc.date, err)
			}
			continue
		}
		if got := fieldErrs(t, err)["birthDate"]; got != tc.want {
			t.Errorf("%s : %v, attendu %v", tc.date, got, tc.want)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	in := valid()
	in.Email = " Claire@Exemple.FR "
	c, err := member.Validate(in, today)
	if err != nil || c.Email == nil || *c.Email != "claire@exemple.fr" {
		t.Fatalf("adresse : %v %v", c.Email, err)
	}
	for _, e := range []string{"claire", "Claire <c@e.fr>", "a@", "a b@exemple.fr"} {
		in.Email = e
		_, err := member.Validate(in, today)
		if fieldErrs(t, err)["email"] != member.ErrEmailFormat {
			t.Errorf("%q devrait être refusée", e)
		}
	}
	in.Email = "  "
	c, err = member.Validate(in, today)
	if err != nil || c.Email != nil {
		t.Fatalf("adresse vide : %v %v", c.Email, err)
	}
}

func TestNormalizePhone(t *testing.T) {
	ok := map[string]string{
		"06 12 34 56 78":      "0612345678",
		"06.12.34.56.78":      "0612345678",
		"06-12-34-56-78":      "0612345678",
		"06 12 34 56 78":      "0612345678",
		"+32 470 12 34 56":    "+32470123456",
		"(+33) 6 12 34 56 78": "+33612345678",
	}
	for in, want := range ok {
		got, err := member.NormalizePhone(in)
		if err != nil || got != want {
			t.Errorf("%q : %q, %v ; attendu %q", in, got, err, want)
		}
	}
	for _, in := range []string{"0612", "+1", "1234567890", "abc", "+1234567890123456", "+33 (0)6 x", "06 12 34 56 78 9"} {
		if _, err := member.NormalizePhone(in); !errors.Is(err, member.ErrPhoneFormat) {
			t.Errorf("%q devrait être refusé : %v", in, err)
		}
	}
}

func TestValidateAddress(t *testing.T) {
	in := valid()
	c, err := member.Validate(in, today)
	if err != nil || c.AddressLine1 != nil || c.PostalCode != nil || c.City != nil {
		t.Fatalf("sans adresse : %+v %v", c, err)
	}

	in.AddressLine1 = "1 rue de la Paix"
	got := fieldErrs(t, func() error { _, err := member.Validate(in, today); return err }())
	if got["postalCode"] != member.ErrAddressIncomplete || got["city"] != member.ErrAddressIncomplete || got["addressLine1"] != nil {
		t.Fatalf("adresse partielle : %v", got)
	}

	in = valid()
	in.AddressLine2 = "Bâtiment B"
	got = fieldErrs(t, func() error { _, err := member.Validate(in, today); return err }())
	if len(got) != 1 || got["addressLine2"] != member.ErrAddressLine2Alone {
		t.Fatalf("complément seul : %v", got)
	}

	for _, pc := range []string{"7500", "75 001", "7500A"} {
		in = valid()
		in.AddressLine1, in.PostalCode, in.City = "1 rue de la Paix", pc, "Paris"
		got = fieldErrs(t, func() error { _, err := member.Validate(in, today); return err }())
		if got["postalCode"] != member.ErrPostalCode {
			t.Errorf("%q : %v", pc, got)
		}
	}

	in = valid()
	in.AddressLine1, in.AddressLine2, in.PostalCode, in.City = " 1 rue de la Paix ", " ", "75001", " Paris "
	c, err = member.Validate(in, today)
	if err != nil || *c.AddressLine1 != "1 rue de la Paix" || c.AddressLine2 != nil || *c.City != "Paris" {
		t.Fatalf("adresse complète : %+v %v", c, err)
	}
}

func TestValidateCumulates(t *testing.T) {
	in := valid()
	in.FirstName = ""
	in.BirthDate = season.MustDate("2026-11-02")
	in.Phone = "0612"
	got := fieldErrs(t, func() error { _, err := member.Validate(in, today); return err }())
	if len(got) != 3 || got["firstName"] == nil || got["birthDate"] == nil || got["phone"] == nil {
		t.Fatalf("erreurs cumulées : %v", got)
	}
}

func TestAge(t *testing.T) {
	cases := []struct {
		birth, day string
		want       int
	}{
		{"2015-11-01", "2026-11-01", 11},
		{"2015-11-02", "2026-11-01", 10},
		{"2008-02-29", "2026-02-28", 17},
		{"2008-02-29", "2026-03-01", 18},
		{"2026-11-01", "2026-11-01", 0},
	}
	for _, tc := range cases {
		if got := member.Age(season.MustDate(tc.birth), season.MustDate(tc.day)); got != tc.want {
			t.Errorf("%s au %s : %d, attendu %d", tc.birth, tc.day, got, tc.want)
		}
	}
}

func TestIdentityKey(t *testing.T) {
	if member.IdentityKey("Léo", "MARTIN") != member.IdentityKey("leo", " martin ") {
		t.Fatal("clés différentes pour la même identité")
	}
	if member.IdentityKey("Léo", "Martin") == member.IdentityKey("Léon", "Martin") {
		t.Fatal("clés identiques pour deux identités")
	}
}

func ids(items []member.Sortable) []int64 {
	out := make([]int64, len(items))
	for i, s := range items {
		out[i] = s.ID
	}
	return out
}

func TestSort(t *testing.T) {
	items := []member.Sortable{
		{ID: 1, FirstName: "Zoé", LastName: "Dupont"},
		{ID: 2, FirstName: "Élodie", LastName: "Martin"},
		{ID: 3, FirstName: "Claire", LastName: "Émery", IsSelf: true},
		{ID: 4, FirstName: "Élodie", LastName: "Dupont"},
		{ID: 5, FirstName: "elodie", LastName: "dupont"},
	}
	mine := slices.Clone(items)
	member.SortMine(mine)
	if got := ids(mine); !slices.Equal(got, []int64{3, 4, 5, 2, 1}) {
		t.Errorf("SortMine : %v", got)
	}
	admin := slices.Clone(items)
	member.SortAdmin(admin)
	if got := ids(admin); !slices.Equal(got, []int64{4, 5, 1, 3, 2}) {
		t.Errorf("SortAdmin : %v", got)
	}
}
