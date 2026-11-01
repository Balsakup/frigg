// Package member porte les règles des fiches membre, sans base ni HTTP
// (research.md 005 › R2, R3, R7, R11) : nettoyage et validation des champs,
// âge, identité repliée pour les doublons et ordres d'affichage.
package member

import (
	"cmp"
	"errors"
	"net/mail"
	"regexp"
	"slices"
	"strings"

	"github.com/balsakup/frigg/internal/season"
	"github.com/balsakup/frigg/internal/textfold"
)

const (
	// MaxPerAccount est le nombre maximal de fiches gérées par un compte (FR-004).
	MaxPerAccount = 20
	MaxName       = 80
	MaxEmail      = 254
	MaxAddress    = 100
	// MaxAge borne la date de naissance dans le passé (FR-001).
	MaxAge = 120
)

var (
	ErrRequired          = errors.New("champ obligatoire")
	ErrBirthFuture       = errors.New("date de naissance dans le futur")
	ErrBirthTooOld       = errors.New("date de naissance trop ancienne")
	ErrEmailFormat       = errors.New("adresse électronique invalide")
	ErrPhoneFormat       = errors.New("téléphone invalide")
	ErrAddressIncomplete = errors.New("adresse incomplète")
	ErrAddressLine2Alone = errors.New("complément sans adresse")
	ErrPostalCode        = errors.New("code postal invalide")
)

// FieldError est une erreur rattachée à un champ (nom JSON du corps).
type FieldError struct {
	Field string
	Err   error
}

func (e *FieldError) Error() string { return e.Field + " : " + e.Err.Error() }
func (e *FieldError) Unwrap() error { return e.Err }

// FieldErrors regroupe toutes les erreurs de champ d'une saisie (FR-015).
type FieldErrors []*FieldError

func (e FieldErrors) Error() string {
	msgs := make([]string, len(e))
	for i, fe := range e {
		msgs[i] = fe.Error()
	}
	return strings.Join(msgs, " ; ")
}

// Input est une fiche saisie, avant nettoyage.
type Input struct {
	FirstName, LastName string
	BirthDate           season.Date
	Email, Phone        string
	AddressLine1        string
	AddressLine2        string
	PostalCode, City    string
}

// Clean est une fiche nettoyée et valide ; les champs facultatifs vides sont nil.
type Clean struct {
	FirstName, LastName string
	BirthDate           season.Date
	Email, Phone        *string
	AddressLine1        *string
	AddressLine2        *string
	PostalCode, City    *string
}

// squash retire les espaces de début et de fin et réduit les espaces
// intérieurs (insécables compris).
func squash(s string) string { return strings.Join(strings.Fields(s), " ") }

func optional(s string) *string {
	if s = squash(s); s == "" {
		return nil
	}
	return &s
}

var postalCodeRe = regexp.MustCompile(`^[0-9]{5}$`)

// Validate nettoie la saisie et renvoie toutes ses erreurs de champ ensemble.
func Validate(in Input, today season.Date) (Clean, error) {
	var errs FieldErrors
	add := func(field string, err error) { errs = append(errs, &FieldError{Field: field, Err: err}) }

	c := Clean{FirstName: squash(in.FirstName), LastName: squash(in.LastName), BirthDate: in.BirthDate}
	if c.FirstName == "" {
		add("firstName", ErrRequired)
	}
	if c.LastName == "" {
		add("lastName", ErrRequired)
	}
	switch {
	case in.BirthDate.After(today):
		add("birthDate", ErrBirthFuture)
	case in.BirthDate.Before(today.AddYears(-MaxAge)):
		add("birthDate", ErrBirthTooOld)
	}

	if e := strings.TrimSpace(in.Email); e != "" {
		a, err := mail.ParseAddress(e)
		if err != nil || a.Name != "" || a.Address != e {
			add("email", ErrEmailFormat)
		} else {
			lower := strings.ToLower(e)
			c.Email = &lower
		}
	}

	if strings.TrimSpace(in.Phone) != "" {
		p, err := NormalizePhone(in.Phone)
		if err != nil {
			add("phone", err)
		} else {
			c.Phone = &p
		}
	}

	c.AddressLine1, c.AddressLine2 = optional(in.AddressLine1), optional(in.AddressLine2)
	c.City = optional(in.City)
	if pc := strings.TrimSpace(in.PostalCode); pc != "" {
		c.PostalCode = &pc
	}
	if c.AddressLine1 != nil || c.PostalCode != nil || c.City != nil {
		if c.AddressLine1 == nil {
			add("addressLine1", ErrAddressIncomplete)
		}
		if c.PostalCode == nil {
			add("postalCode", ErrAddressIncomplete)
		} else if !postalCodeRe.MatchString(*c.PostalCode) {
			add("postalCode", ErrPostalCode)
		}
		if c.City == nil {
			add("city", ErrAddressIncomplete)
		}
	} else if c.AddressLine2 != nil {
		add("addressLine2", ErrAddressLine2Alone)
	}

	if len(errs) > 0 {
		return Clean{}, errs
	}
	return c, nil
}

var (
	phoneSeparators = strings.NewReplacer(" ", "", " ", "", " ", "", ".", "", "-", "", "(", "", ")", "")
	frenchPhoneRe   = regexp.MustCompile(`^0[0-9]{9}$`)
	intlPhoneRe     = regexp.MustCompile(`^\+[0-9]{8,15}$`)
)

// NormalizePhone retire les séparateurs et renvoie un numéro français à
// 10 chiffres (0612345678) ou international (+32470123456) (R3).
func NormalizePhone(s string) (string, error) {
	p := phoneSeparators.Replace(strings.TrimSpace(s))
	if frenchPhoneRe.MatchString(p) || intlPhoneRe.MatchString(p) {
		return p, nil
	}
	return "", ErrPhoneFormat
}

// Age renvoie l'âge en années révolues à la date today. Une personne née un
// 29 février prend un an le 1er mars les années non bissextiles.
func Age(birth, today season.Date) int {
	age := today.Year() - birth.Year()
	if today.Month() < birth.Month() || (today.Month() == birth.Month() && today.Day() < birth.Day()) {
		age--
	}
	return max(age, 0)
}

// IdentityKey replie prénom et nom pour la détection des doublons (FR-005).
func IdentityKey(first, last string) string {
	return textfold.Fold(first) + "\x00" + textfold.Fold(last)
}

// Sortable est ce qu'il faut d'une fiche pour la trier.
type Sortable struct {
	ID                  int64
	FirstName, LastName string
	IsSelf              bool
}

// SortMine trie « Mes membres » : la fiche « Vous » d'abord, puis par prénom
// et nom repliés (FR-009).
func SortMine(items []Sortable) {
	slices.SortStableFunc(items, func(a, b Sortable) int {
		if a.IsSelf != b.IsSelf {
			if a.IsSelf {
				return -1
			}
			return 1
		}
		return cmp.Or(
			cmp.Compare(textfold.Fold(a.FirstName), textfold.Fold(b.FirstName)),
			cmp.Compare(textfold.Fold(a.LastName), textfold.Fold(b.LastName)),
			cmp.Compare(a.ID, b.ID),
		)
	})
}

// SortAdmin trie la liste d'administration par nom puis prénom repliés (FR-010).
func SortAdmin(items []Sortable) {
	slices.SortStableFunc(items, func(a, b Sortable) int {
		return cmp.Or(
			cmp.Compare(textfold.Fold(a.LastName), textfold.Fold(b.LastName)),
			cmp.Compare(textfold.Fold(a.FirstName), textfold.Fold(b.FirstName)),
			cmp.Compare(a.ID, b.ID),
		)
	})
}
