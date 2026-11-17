package public

import (
	"errors"
	"strings"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/member"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
)

// Messages des erreurs propres aux dépôts publics (FR-006, FR-026).
const (
	MsgEmailRequired   = "L'adresse électronique est obligatoire."
	MsgConsentRequired = "Vous devez accepter le traitement de vos données."
	MsgDeclaredKind    = "Choisissez une première adhésion ou un renouvellement."
)

var (
	// ErrEmailRequired : un dépôt public exige une adresse électronique.
	ErrEmailRequired = errors.New("adresse électronique obligatoire")
	// ErrConsentRequired : le consentement au traitement des données est
	// obligatoire (FR-007).
	ErrConsentRequired = errors.New("consentement obligatoire")
	// ErrDeclaredKind : type d'adhésion déclaré inconnu.
	ErrDeclaredKind = errors.New("type d'adhésion inconnu")
)

// Message renvoie le message français d'une erreur propre aux dépôts
// publics, ou "" pour une erreur de member.Validate.
func Message(err error) string {
	switch {
	case errors.Is(err, ErrEmailRequired):
		return MsgEmailRequired
	case errors.Is(err, ErrConsentRequired):
		return MsgConsentRequired
	case errors.Is(err, ErrDeclaredKind):
		return MsgDeclaredKind
	}
	return ""
}

// Submission est la saisie d'un formulaire public, avant nettoyage.
type Submission struct {
	FirstName, LastName string
	BirthDate           season.Date
	Email, Phone        string
	AddressLine1        string
	AddressLine2        string
	PostalCode, City    string
	// Consent : case de consentement cochée (FR-007).
	Consent bool
	// CreateAccount : la personne souhaite créer un compte après confirmation.
	CreateAccount bool
	// Website est le champ piège, invisible pour une personne (R10).
	Website string
}

// Honeypot indique que le champ piège est rempli : le dépôt vient d'un robot
// et doit être ignoré sans le révéler (research.md › R10).
func Honeypot(website string) bool {
	return website != ""
}

// Validate nettoie la saisie avec les règles des fiches (005) et exige en
// plus une adresse électronique et le consentement. Toutes les erreurs de
// champ sont renvoyées ensemble ; l'adresse est normalisée comme à la
// connexion (auth.NormalizeEmail).
func Validate(in Submission, today season.Date) (member.Clean, member.FieldErrors) {
	email := auth.NormalizeEmail(in.Email)
	c, err := member.Validate(member.Input{
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		BirthDate:    in.BirthDate,
		Email:        email,
		Phone:        in.Phone,
		AddressLine1: in.AddressLine1,
		AddressLine2: in.AddressLine2,
		PostalCode:   in.PostalCode,
		City:         in.City,
	}, today)
	var errs member.FieldErrors
	if err != nil && !errors.As(err, &errs) {
		errs = member.FieldErrors{{Field: "", Err: err}}
	}
	if strings.TrimSpace(email) == "" {
		errs = append(errs, &member.FieldError{Field: "email", Err: ErrEmailRequired})
	}
	if !in.Consent {
		errs = append(errs, &member.FieldError{Field: "consent", Err: ErrConsentRequired})
	}
	if len(errs) > 0 {
		return member.Clean{}, errs
	}
	return c, nil
}

// ValidateMembershipTarget vérifie le type d'adhésion déclaré par la
// personne : première adhésion ou renouvellement.
func ValidateMembershipTarget(declaredKind string) error {
	switch membership.Kind(declaredKind) {
	case membership.First, membership.Renewal:
		return nil
	}
	return &member.FieldError{Field: "declaredKind", Err: ErrDeclaredKind}
}
