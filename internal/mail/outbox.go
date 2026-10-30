package mail

import (
	"context"
	"time"

	"github.com/balsakup/frigg/internal/database/sqlc"
)

// TimeFormat est le format des dates stockées en base (ISO 8601 UTC).
const TimeFormat = "2006-01-02T15:04:05Z"

// Enqueue met le message en file. À appeler avec les requêtes de la transaction
// de l'action qui le déclenche, pour que l'e-mail n'existe que si elle aboutit.
func Enqueue(ctx context.Context, q *sqlc.Queries, to string, msg Message) error {
	return q.EnqueueEmail(ctx, sqlc.EnqueueEmailParams{
		Recipient: to,
		Subject:   msg.Subject,
		BodyText:  msg.Text,
		BodyHtml:  msg.HTML,
		Now:       time.Now().UTC().Format(TimeFormat),
	})
}
