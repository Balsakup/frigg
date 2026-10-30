package mail

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/balsakup/frigg/internal/database/sqlc"
)

const (
	batchSize     = 20
	maxAttempts   = 10
	maxBackoff    = time.Hour
	sentRetention = 30 * 24 * time.Hour
)

// Worker dépile la file d'e-mails à intervalle régulier (research.md › R9).
type Worker struct {
	Queries  *sqlc.Queries // pool d'écriture
	Sender   Sender
	Interval time.Duration
	Logger   *slog.Logger
	Now      func() time.Time
}

// Run traite la file jusqu'à l'annulation de ctx.
func (w *Worker) Run(ctx context.Context) {
	t := time.NewTicker(w.Interval)
	defer t.Stop()
	for {
		w.Tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Tick traite un lot de messages dus et purge les anciens messages envoyés.
func (w *Worker) Tick(ctx context.Context) {
	now := w.now()
	due, err := w.Queries.ListDueEmails(ctx, sqlc.ListDueEmailsParams{
		Now:      now.Format(TimeFormat),
		MaxCount: batchSize,
	})
	if err != nil {
		w.Logger.Error("lecture de la file d'e-mails impossible", "error", err)
		return
	}
	for _, e := range due {
		if ctx.Err() != nil {
			return
		}
		w.deliver(ctx, e)
	}
	if err := w.Queries.PurgeSentEmails(ctx, sql.NullString{String: now.Add(-sentRetention).Format(TimeFormat), Valid: true}); err != nil {
		w.Logger.Error("purge des e-mails envoyés impossible", "error", err)
	}
}

func (w *Worker) deliver(ctx context.Context, e sqlc.OutboundEmail) {
	err := w.Sender.Send(ctx, e.Recipient, Message{Subject: e.Subject, Text: e.BodyText, HTML: e.BodyHtml})
	now := w.now()
	if err == nil {
		if err := w.Queries.MarkEmailSent(ctx, sqlc.MarkEmailSentParams{
			ID:     e.ID,
			SentAt: sql.NullString{String: now.Format(TimeFormat), Valid: true},
		}); err != nil {
			w.Logger.Error("mise à jour d'un e-mail envoyé impossible", "email_id", e.ID, "error", err)
		}
		w.Logger.Info("e-mail envoyé", "email_id", e.ID)
		return
	}

	attempts := e.Attempts + 1
	lastErr := sql.NullString{String: err.Error(), Valid: true}
	if attempts >= maxAttempts {
		_ = w.Queries.MarkEmailFailed(ctx, sqlc.MarkEmailFailedParams{ID: e.ID, Attempts: attempts, LastError: lastErr})
		w.Logger.Error("envoi d'e-mail abandonné", "email_id", e.ID, "attempts", attempts, "error", err)
		return
	}
	_ = w.Queries.MarkEmailRetry(ctx, sqlc.MarkEmailRetryParams{
		ID:            e.ID,
		Attempts:      attempts,
		NextAttemptAt: now.Add(Backoff(attempts)).Format(TimeFormat),
		LastError:     lastErr,
	})
	w.Logger.Warn("envoi d'e-mail échoué, nouvelle tentative prévue", "email_id", e.ID, "attempts", attempts, "error", err)
}

// Backoff renvoie le délai avant la tentative suivante : 1 min, 2 min, 4 min…
// plafonné à 1 h.
func Backoff(attempts int64) time.Duration {
	d := time.Minute << (attempts - 1)
	if d <= 0 || d > maxBackoff {
		return maxBackoff
	}
	return d
}

func (w *Worker) now() time.Time {
	if w.Now != nil {
		return w.Now().UTC()
	}
	return time.Now().UTC()
}
