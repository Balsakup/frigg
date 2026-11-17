package public

import (
	"errors"
	"testing"

	"github.com/balsakup/frigg/internal/member"
	"github.com/balsakup/frigg/internal/season"
)

func TestValidate(t *testing.T) {
	today := season.MustDate("2026-11-17")
	valid := Submission{
		FirstName: "Claire", LastName: "Martin", BirthDate: season.MustDate("1985-01-01"),
		Email: "claire@exemple.fr", Consent: true,
	}
	tests := []struct {
		name   string
		edit   func(*Submission)
		fields map[string]error
	}{
		{"valide", func(*Submission) {}, nil},
		{"adresse absente", func(s *Submission) { s.Email = "  " }, map[string]error{"email": ErrEmailRequired}},
		{"adresse mal formée", func(s *Submission) { s.Email = "claire" }, map[string]error{"email": member.ErrEmailFormat}},
		{"consentement absent", func(s *Submission) { s.Consent = false }, map[string]error{"consent": ErrConsentRequired}},
		{"tout manque", func(s *Submission) { s.FirstName, s.Email, s.Consent = "", "", false }, map[string]error{
			"firstName": member.ErrRequired, "email": ErrEmailRequired, "consent": ErrConsentRequired,
		}},
		{"adresse postale incomplète", func(s *Submission) { s.City = "Lyon" }, map[string]error{
			"addressLine1": member.ErrAddressIncomplete, "postalCode": member.ErrAddressIncomplete,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid
			tt.edit(&in)
			_, errs := Validate(in, today)
			if len(errs) != len(tt.fields) {
				t.Fatalf("erreurs = %v, attendu %v", errs, tt.fields)
			}
			for _, fe := range errs {
				if want, ok := tt.fields[fe.Field]; !ok || !errors.Is(fe, want) {
					t.Errorf("%s : %v, attendu %v", fe.Field, fe.Err, want)
				}
			}
		})
	}
}

func TestValidateNormalizesEmail(t *testing.T) {
	c, errs := Validate(Submission{
		FirstName: " Claire ", LastName: "Martin", BirthDate: season.MustDate("1985-01-01"),
		Email: "  Claire.Martin@Exemple.FR ", Consent: true,
	}, season.MustDate("2026-11-17"))
	if errs != nil {
		t.Fatal(errs)
	}
	if c.Email == nil || *c.Email != "claire.martin@exemple.fr" {
		t.Fatalf("adresse = %v, attendu claire.martin@exemple.fr", c.Email)
	}
	if c.FirstName != "Claire" {
		t.Errorf("prénom = %q", c.FirstName)
	}
}

func TestHoneypot(t *testing.T) {
	for website, want := range map[string]bool{"": false, "https://robot.example": true, " ": true} {
		if got := Honeypot(website); got != want {
			t.Errorf("Honeypot(%q) = %v, attendu %v", website, got, want)
		}
	}
}

func TestValidateMembershipTarget(t *testing.T) {
	for kind, ok := range map[string]bool{"first": true, "renewal": true, "": false, "autre": false} {
		err := ValidateMembershipTarget(kind)
		if (err == nil) != ok {
			t.Errorf("ValidateMembershipTarget(%q) = %v", kind, err)
		}
		if err != nil && Message(err) != MsgDeclaredKind {
			t.Errorf("message de %q : %q", kind, Message(err))
		}
	}
}

func TestMessage(t *testing.T) {
	_, errs := Validate(Submission{FirstName: "A", LastName: "B", BirthDate: season.MustDate("1985-01-01")}, season.MustDate("2026-11-17"))
	got := map[string]string{}
	for _, fe := range errs {
		got[fe.Field] = Message(fe)
	}
	if got["email"] != MsgEmailRequired || got["consent"] != MsgConsentRequired {
		t.Fatalf("messages = %v", got)
	}
	if Message(member.ErrRequired) != "" {
		t.Error("une erreur de member.Validate n'a pas de message propre")
	}
}
