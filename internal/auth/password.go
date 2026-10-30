// Package auth regroupe les briques d'authentification : mots de passe, jetons,
// niveaux d'accès, sessions et limitation de débit.
package auth

import (
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/alexedwards/argon2id"
)

// hashParams centralise les paramètres argon2id (recommandation OWASP,
// research.md › R6) : 19 Mio par hachage pour ménager les petits serveurs.
var hashParams = &argon2id.Params{
	Memory:      19 * 1024,
	Iterations:  2,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

// Messages de la règle de mot de passe (FR-008), une entrée par critère.
const (
	MsgPasswordTooShort = "Le mot de passe doit contenir au moins 8 caractères."
	MsgPasswordNoUpper  = "Le mot de passe doit contenir au moins une lettre majuscule."
	MsgPasswordNoDigit  = "Le mot de passe doit contenir au moins un chiffre."
	PasswordMinLength   = 8
	emailMinLength      = 3
	emailMaxLength      = 254
)

// HashPassword renvoie l'empreinte argon2id (format PHC) du mot de passe.
func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, hashParams)
}

// VerifyPassword compare un mot de passe à son empreinte.
func VerifyPassword(password, hash string) bool {
	ok, err := argon2id.ComparePasswordAndHash(password, hash)
	return err == nil && ok
}

var dummyHash = sync.OnceValue(func() string {
	h, err := HashPassword("mot de passe factice pour égaliser les temps")
	if err != nil {
		panic(err)
	}
	return h
})

// VerifyAgainstDummy effectue un hachage complet sans compte réel, pour que la
// durée de réponse ne révèle pas l'existence d'une adresse.
func VerifyAgainstDummy(password string) {
	_ = VerifyPassword(password, dummyHash())
}

// ValidatePassword renvoie les critères non respectés (vide si conforme).
func ValidatePassword(password string) []string {
	var msgs []string
	if utf8.RuneCountInString(password) < PasswordMinLength {
		msgs = append(msgs, MsgPasswordTooShort)
	}
	hasUpper, hasDigit := false, false
	for _, r := range password {
		hasUpper = hasUpper || unicode.IsUpper(r)
		hasDigit = hasDigit || unicode.IsDigit(r)
	}
	if !hasUpper {
		msgs = append(msgs, MsgPasswordNoUpper)
	}
	if !hasDigit {
		msgs = append(msgs, MsgPasswordNoDigit)
	}
	return msgs
}

// NormalizeEmail retire les espaces en début et fin et passe en minuscules (FR-012).
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail vérifie le format d'une adresse déjà normalisée. La preuve
// réelle reste la confirmation par e-mail.
func ValidateEmail(email string) bool {
	n := utf8.RuneCountInString(email)
	if n < emailMinLength || n > emailMaxLength || strings.Count(email, "@") != 1 {
		return false
	}
	local, domain, _ := strings.Cut(email, "@")
	return local != "" && domain != "" && !strings.ContainsAny(email, " \t\r\n")
}
