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
	row, err := q.GetAssociation(ctx)
	if err != nil {
		return err
	}
	name := row.Name
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

// enqueueNotice rend un gabarit d'information et le met en file avec les
// requêtes de la transaction courante. path, s'il n'est pas vide, est le chemin
// du SPA vers lequel pointe le lien (sans jeton : la page exige une session).
func (s *server) enqueueNotice(ctx context.Context, q *sqlc.Queries, template, to string, data mail.Data, path string) error {
	row, err := q.GetAssociation(ctx)
	if err != nil {
		return err
	}
	name := row.Name
	data.Association = defaultAssociationName
	if name.Valid {
		data.Association = name.String
	}
	if path != "" {
		data.Link = s.Config.BaseURL + path
	}
	msg, err := mail.Render(template, data)
	if err != nil {
		return err
	}
	return mail.Enqueue(ctx, q, to, msg)
}
