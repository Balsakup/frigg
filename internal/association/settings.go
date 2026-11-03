package association

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/balsakup/frigg/internal/season"
)

// Bornes des champs (data-model.md).
const (
	NameMax      = 120
	TaglineMax   = 150
	ShortNameMax = 20
	PurposeMax   = 2000
	AddressMax   = 100
	EmailMax     = 254
)

// Messages d'erreur des champs texte (contracts/api.md).
const (
	MsgNameRequired = "Indiquez le nom de l'association."
	MsgNameTooLong  = "Le nom ne doit pas dépasser 120 caractères."
	MsgTextTooLong  = "Ce texte est trop long."
)

// Address est l'adresse du siège ; chaque élément est facultatif.
type Address struct {
	Line1, Line2, PostalCode, City *string
}

// Social regroupe les liens des réseaux sociaux (liste fermée).
type Social struct {
	Facebook, Instagram, LinkedIn, YouTube, Mastodon, Bluesky *string
}

// Settings sont les paramètres de la section « Général ». Un pointeur nil
// signifie « non renseigné ».
type Settings struct {
	Name         string
	Tagline      *string
	ShortName    *string
	Purpose      *string
	RNA          *string
	SIRET        *string
	FoundedOn    *string // AAAA-MM-JJ
	Address      Address
	ContactEmail *string
	Phone        *string // forme nationale à 10 chiffres
	Website      *string
	Social       Social
}

// FieldError est une erreur sur un champ ; Field est le chemin JSON
// (« address.postalCode », « social.instagram »).
type FieldError struct {
	Field   string
	Message string
}

var postalCodeRe = regexp.MustCompile(`^[0-9]{5}$`)

// Trim retire les espaces de début et de fin ; une chaîne vide devient nil.
func Trim(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// Normalize supprime les espaces superflus, normalise les identifiants et
// vérifie chaque champ. Toutes les erreurs sont renvoyées ensemble ; si la liste
// n'est pas vide, aucune valeur ne doit être enregistrée.
func Normalize(in Settings, today season.Date) (Settings, []FieldError) {
	var errs []FieldError
	fail := func(field, msg string) { errs = append(errs, FieldError{field, msg}) }
	maxLen := func(field string, s *string, max int) *string {
		s = Trim(s)
		if s != nil && utf8.RuneCountInString(*s) > max {
			fail(field, MsgTextTooLong)
		}
		return s
	}
	webURL := func(field string, s *string) *string {
		s = Trim(s)
		if s != nil && !ValidWebURL(*s) {
			fail(field, MsgWebURLInvalid)
		}
		return s
	}

	out := Settings{Name: strings.TrimSpace(in.Name)}
	switch n := utf8.RuneCountInString(out.Name); {
	case n == 0:
		fail("name", MsgNameRequired)
	case n > NameMax:
		fail("name", MsgNameTooLong)
	}
	out.Tagline = maxLen("tagline", in.Tagline, TaglineMax)
	out.ShortName = maxLen("shortName", in.ShortName, ShortNameMax)
	out.Purpose = maxLen("purpose", in.Purpose, PurposeMax)

	if s := Trim(in.RNA); s != nil {
		v, ok := NormalizeRNA(*s)
		if !ok {
			fail("rna", MsgRNAInvalid)
		}
		out.RNA = &v
	}
	if s := Trim(in.SIRET); s != nil {
		v, msg := NormalizeSIRET(*s)
		if msg != "" {
			fail("siret", msg)
		}
		out.SIRET = &v
	}
	if s := Trim(in.FoundedOn); s != nil {
		d, err := season.ParseDate(*s)
		switch {
		case err != nil:
			fail("foundedOn", MsgDateInvalid)
		case d.After(today):
			fail("foundedOn", MsgFoundedInFuture)
		}
		out.FoundedOn = s
	}

	out.Address.Line1 = maxLen("address.line1", in.Address.Line1, AddressMax)
	out.Address.Line2 = maxLen("address.line2", in.Address.Line2, AddressMax)
	out.Address.City = maxLen("address.city", in.Address.City, AddressMax)
	if s := Trim(in.Address.PostalCode); s != nil {
		if !postalCodeRe.MatchString(*s) {
			fail("address.postalCode", MsgPostalCode)
		}
		out.Address.PostalCode = s
	}

	if s := Trim(in.ContactEmail); s != nil {
		if !validEmail(*s) {
			fail("contactEmail", MsgEmailInvalid)
		}
		out.ContactEmail = s
	}
	if s := Trim(in.Phone); s != nil {
		v, ok := NormalizePhone(*s)
		if !ok {
			fail("phone", MsgPhoneInvalid)
			v = *s
		}
		out.Phone = &v
	}
	out.Website = webURL("website", in.Website)
	out.Social = Social{
		Facebook:  webURL("social.facebook", in.Social.Facebook),
		Instagram: webURL("social.instagram", in.Social.Instagram),
		LinkedIn:  webURL("social.linkedin", in.Social.LinkedIn),
		YouTube:   webURL("social.youtube", in.Social.YouTube),
		Mastodon:  webURL("social.mastodon", in.Social.Mastodon),
		Bluesky:   webURL("social.bluesky", in.Social.Bluesky),
	}
	return out, errs
}

// validEmail accepte une adresse seule (sans nom d'affichage), bien formée.
func validEmail(s string) bool {
	if len(s) > EmailMax {
		return false
	}
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s && a.Name == "" && strings.Contains(s[strings.LastIndex(s, "@")+1:], ".")
}
