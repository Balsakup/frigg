package api

import (
	"time"

	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Libellé affiché tant que le nom de l'association n'est pas défini.
const defaultAssociationName = "Mon association"

// timeFormat est le format des dates stockées en base (ISO 8601 UTC).
const timeFormat = "2006-01-02T15:04:05Z"

func nowString() string { return time.Now().UTC().Format(timeFormat) }

// Account est la représentation publique d'un compte.
type Account struct {
	ID            int64  `json:"id"`
	Email         string `json:"email"`
	Role          string `json:"role" enum:"member,admin"`
	EmailVerified bool   `json:"emailVerified"`
}

// Association est la représentation publique de l'association.
type Association struct {
	Name string `json:"name"`
}

// Session décrit la personne connectée et son association.
type Session struct {
	Account     Account     `json:"account"`
	Association Association `json:"association"`
}

func toAccount(a sqlc.Account) Account {
	return Account{ID: a.ID, Email: a.Email, Role: a.Role, EmailVerified: a.EmailVerifiedAt.Valid}
}

// SessionOutput est la réponse des opérations qui renvoient la session.
type SessionOutput struct {
	Body Session
}
