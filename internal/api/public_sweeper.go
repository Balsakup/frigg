package api

import (
	"context"
	"time"

	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/public"
)

// PublicSweeper purge les dépôts publics non confirmés au-delà de
// public.SubmissionRetention et les liens personnels expirés depuis plus de
// public.LinkPurgeGrace (research.md 016 › R3, FR-008, FR-013).
type PublicSweeper struct {
	s        *server
	Interval time.Duration
}

// NewPublicSweeper crée le balayage (intervalle de 10 minutes).
func NewPublicSweeper(deps Deps) *PublicSweeper {
	return &PublicSweeper{s: &server{Deps: deps}, Interval: 10 * time.Minute}
}

// Run balaie à intervalle régulier jusqu'à l'annulation de ctx.
func (w *PublicSweeper) Run(ctx context.Context) {
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

// Tick supprime les dépôts et les liens périmés, puis journalise leur nombre.
func (w *PublicSweeper) Tick(ctx context.Context) {
	s := w.s
	now := s.now().UTC()
	var submissions, links int64
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		var err error
		submissions, err = q.PurgePublicSubmissions(ctx, now.Add(-public.SubmissionRetention).Format(timeFormat))
		if err != nil {
			return err
		}
		links, err = q.PurgeTrackingLinks(ctx, now.Add(-public.LinkPurgeGrace).Format(timeFormat))
		return err
	})
	if err != nil {
		s.Logger.ErrorContext(ctx, "purge des dépôts publics et des liens personnels impossible", "error", err)
		return
	}
	if submissions > 0 || links > 0 {
		s.Logger.InfoContext(ctx, "dépôts publics et liens personnels purgés",
			"action", "public_purge", "submissions", submissions, "tracking_links", links)
	}
}
