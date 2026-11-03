package mail

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/balsakup/frigg/internal/brand"
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
	// Files lit les pièces jointes des communications (specs/009-bulk-email) ;
	// nil : un e-mail de communication portant des pièces échoue.
	Files AttachmentFiles
	// Identity renvoie l'identité visuelle en vigueur (nom, couleurs, logo),
	// lue une fois par passage ; nil : identité de Frigg sans nom.
	Identity func(ctx context.Context) (Identity, error)
}

// AttachmentFiles lit un fichier enregistré en vérifiant sa taille et son
// empreinte (communication.FileStore).
type AttachmentFiles interface {
	Read(stored string, size int64, sha256 string) ([]byte, error)
}

// errAttachment marque un échec définitif lié aux pièces jointes : fichier
// manquant ou altéré, inutile de réessayer.
var errAttachment = errors.New("pièce jointe illisible")

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
	if len(due) > 0 {
		id := w.identity(ctx)
		// Pièces jointes lues une fois par communication et par lot.
		cache := map[int64][]Attachment{}
		for _, e := range due {
			if ctx.Err() != nil {
				return
			}
			w.deliver(ctx, e, id, cache)
		}
	}
	if err := w.Queries.PurgeSentEmails(ctx, sql.NullString{String: now.Add(-sentRetention).Format(TimeFormat), Valid: true}); err != nil {
		w.Logger.Error("purge des e-mails envoyés impossible", "error", err)
	}
}

// identity lit l'identité visuelle ; en cas d'erreur, l'envoi continue avec
// l'identité par défaut.
func (w *Worker) identity(ctx context.Context) Identity {
	def := Identity{Palette: brand.Default}
	if w.Identity == nil {
		return def
	}
	id, err := w.Identity(ctx)
	if err != nil {
		w.Logger.Error("lecture de l'identité de l'association impossible", "error", err)
		return def
	}
	return id
}

func (w *Worker) deliver(ctx context.Context, e sqlc.OutboundEmail, id Identity, cache map[int64][]Attachment) {
	msg := Message{Subject: e.Subject, Text: e.BodyText, HTML: e.BodyHtml, FromName: e.FromName.String, Footer: e.Footer.String}
	var err error
	if e.Wrap == 1 {
		// Nom, couleurs et logo en vigueur à l'envoi ; le nom d'expéditeur
		// stocké n'est lu que pour les lignes antérieures (wrap = 0).
		msg, err = Wrap(msg, id)
	}
	if err == nil && e.CommunicationID.Valid {
		msg.Attachments, err = w.attachments(ctx, e.CommunicationID.Int64, cache)
	}
	if err == nil {
		err = w.Sender.Send(ctx, e.Recipient, msg)
	}
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
	if attempts >= maxAttempts || errors.Is(err, errAttachment) {
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

// attachments charge les pièces jointes d'une communication.
func (w *Worker) attachments(ctx context.Context, communicationID int64, cache map[int64][]Attachment) ([]Attachment, error) {
	if a, ok := cache[communicationID]; ok {
		return a, nil
	}
	rows, err := w.Queries.ListCommunicationAttachments(ctx, sql.NullInt64{Int64: communicationID, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]Attachment, 0, len(rows))
	for _, r := range rows {
		if w.Files == nil {
			return nil, errAttachment
		}
		data, err := w.Files.Read(r.StoredName, r.SizeBytes, r.Sha256)
		if err != nil {
			return nil, fmt.Errorf("%w : %v", errAttachment, err)
		}
		out = append(out, Attachment{Name: r.OriginalName, ContentType: r.ContentType, Data: data})
	}
	cache[communicationID] = out
	return out, nil
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
