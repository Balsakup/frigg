package auth

import (
	"slices"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		password string
		want     []string
	}{
		{"Motdepasse1", nil},
		{"Élan2026", nil},
		{"Court1A", []string{MsgPasswordTooShort}},
		{"motdepasse1", []string{MsgPasswordNoUpper}},
		{"Motdepasse", []string{MsgPasswordNoDigit}},
		{"abc", []string{MsgPasswordTooShort, MsgPasswordNoUpper, MsgPasswordNoDigit}},
	}
	for _, tt := range tests {
		if got := ValidatePassword(tt.password); !slices.Equal(got, tt.want) {
			t.Errorf("ValidatePassword(%q) = %v, attendu %v", tt.password, got, tt.want)
		}
	}
}

func TestNormalizeAndValidateEmail(t *testing.T) {
	if got := NormalizeEmail("  Claire@Exemple.FR "); got != "claire@exemple.fr" {
		t.Errorf("NormalizeEmail = %q", got)
	}
	for email, want := range map[string]bool{
		"claire@exemple.fr": true,
		"a@b":               true,
		"sans-arobase":      false,
		"@exemple.fr":       false,
		"claire@":           false,
		"a@b@c":             false,
		"cl aire@x.fr":      false,
	} {
		if got := ValidateEmail(email); got != want {
			t.Errorf("ValidateEmail(%q) = %v, attendu %v", email, got, want)
		}
	}
}

func TestHashAndVerify(t *testing.T) {
	hash, err := HashPassword("Motdepasse1")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("Motdepasse1", hash) {
		t.Error("le bon mot de passe est refusé")
	}
	if VerifyPassword("Motdepasse2", hash) {
		t.Error("un mauvais mot de passe est accepté")
	}
}
