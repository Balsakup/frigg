package api

import (
	"context"
	"errors"
	"time"

	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/membership"
)

// OnlinePaymentSweeper vérifie une fois chaque tentative restée en cours au
// delà de sa durée de validité (research.md 008 › R8) : constatée si payée,
// expirée sinon. Il rattrape une notification perdue sans sondage fréquent.
type OnlinePaymentSweeper struct {
	s        *server
	Interval time.Duration
}

// unverifiedAfter : signalement si HelloAsso reste injoignable aussi longtemps
// après l'expiration d'une tentative.
const unverifiedAfter = 24 * time.Hour

// NewOnlinePaymentSweeper crée le balayage (intervalle de 5 minutes).
func NewOnlinePaymentSweeper(deps Deps) *OnlinePaymentSweeper {
	return &OnlinePaymentSweeper{s: &server{Deps: deps}, Interval: 5 * time.Minute}
}

// Run balaie à intervalle régulier jusqu'à l'annulation de ctx.
func (w *OnlinePaymentSweeper) Run(ctx context.Context) {
	t := time.NewTicker(w.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.Tick(ctx)
		}
	}
}

// Tick traite au plus 50 tentatives expirées.
func (w *OnlinePaymentSweeper) Tick(ctx context.Context) {
	s := w.s
	if s.HelloAsso == nil {
		return
	}
	now := s.now()
	ops, err := s.Store.Read.ListPendingOnlinePaymentsBefore(ctx, now.Add(-pendingWindow).UTC().Format(timeFormat))
	if err != nil {
		s.Logger.ErrorContext(ctx, "lecture des paiements en ligne impossible", "error", err)
		return
	}
	for _, op := range ops {
		if ctx.Err() != nil {
			return
		}
		w.sweep(ctx, op, now)
	}
}

func (w *OnlinePaymentSweeper) sweep(ctx context.Context, op sqlc.OnlinePayment, now time.Time) {
	s := w.s
	stamp := ns(s.timestamp())
	// Tentative d'un autre environnement (bascule test ↔ réel) ou sans
	// intention : caduque, sans appel à HelloAsso.
	if op.Environment != s.HelloAsso.Environment() || !op.CheckoutIntentID.Valid {
		code := "environment_changed"
		if op.Environment == s.HelloAsso.Environment() {
			code = "no_checkout"
		}
		if _, err := s.Store.Write.CloseOnlinePayment(ctx, sqlc.CloseOnlinePaymentParams{
			Status: string(membership.OnlineExpired), ErrorCode: ns(code), Now: stamp, ID: op.ID,
		}); err != nil {
			s.Logger.ErrorContext(ctx, "clôture du paiement en ligne impossible", "error", err)
		}
		return
	}
	status, err := s.verifyOnlinePayment(ctx, op.ID, sourceSweep, membership.HintNone)
	switch {
	case errors.Is(err, errHelloAssoUnavailable):
		if now.Sub(parseTime(op.CreatedAt)) > pendingWindow+unverifiedAfter {
			if err := s.Store.Write.CreatePaymentAlert(ctx, sqlc.CreatePaymentAlertParams{
				MembershipID: op.MembershipID, OnlinePaymentID: op.ID, Kind: string(membership.AlertUnverified), Now: s.timestamp(),
			}); err != nil {
				s.Logger.ErrorContext(ctx, "création du signalement impossible", "error", err)
			}
		}
	case err != nil:
		s.Logger.ErrorContext(ctx, "vérification du paiement impossible", "error", err, "online_payment_id", op.ID)
	case status == membership.OnlinePending:
		if _, err := s.Store.Write.CloseOnlinePayment(ctx, sqlc.CloseOnlinePaymentParams{
			Status: string(membership.OnlineExpired), Now: stamp, ID: op.ID,
		}); err != nil {
			s.Logger.ErrorContext(ctx, "clôture du paiement en ligne impossible", "error", err)
		} else {
			s.Logger.InfoContext(ctx, "paiement en ligne vérifié", "online_payment_id", op.ID,
				"membership_id", op.MembershipID, "source", sourceSweep, "outcome", string(membership.OnlineExpired))
		}
	}
}
