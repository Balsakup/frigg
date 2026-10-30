package api

import (
	"context"

	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/mail"
)

// enqueueMail rend le gabarit et met l'e-mail en file avec les requêtes de la
// transaction courante. path est le chemin du SPA, le jeton va dans le fragment
// de l'URL (research.md › R8).
func (s *server) enqueueMail(ctx context.Context, q *sqlc.Queries, template, to, path, token string) error {
	name, err := q.GetAssociation(ctx)
	if err != nil {
		return err
	}
	association := defaultAssociationName
	if name.Valid {
		association = name.String
	}
	msg, err := mail.Render(template, mail.Data{
		Association: association,
		Link:        s.Config.BaseURL + path + "#" + token,
	})
	if err != nil {
		return err
	}
	return mail.Enqueue(ctx, q, to, msg)
}
