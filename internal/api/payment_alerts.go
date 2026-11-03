package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Signalements de paiement (research.md 008 › R10).

const (
	MsgAlertGone     = "Ce signalement n'existe pas."
	MsgAlertResolved = "Ce signalement a déjà été traité."
)

type PaymentAlertIDInput struct {
	ID int64 `path:"id" doc:"Identifiant du signalement"`
}

func (s *server) registerPaymentAlerts(api huma.API) {
	register(api, auth.Admin, huma.Operation{
		OperationID: "resolve-payment-alert",
		Method:      http.MethodPost,
		Path:        "/admin/payment-alerts/{id}/resolve",
		Summary:     "Marquer un signalement de paiement comme traité",
		Tags:        []string{tagMemberships},
	}, s.resolvePaymentAlert)
}

func (s *server) resolvePaymentAlert(ctx context.Context, in *PaymentAlertIDInput) (*AdminMembershipOutput, error) {
	var alert sqlc.PaymentAlert
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		a, err := q.GetPaymentAlert(ctx, in.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgAlertGone)
		}
		if err != nil {
			return err
		}
		alert = a
		n, err := q.ResolvePaymentAlert(ctx, sqlc.ResolvePaymentAlertParams{Now: ns(s.timestamp()), ActorID: nid(CurrentAccount(ctx).ID), ID: a.ID})
		if err == nil && n == 0 {
			return huma.Error409Conflict(MsgAlertResolved)
		}
		return err
	})
	if err != nil {
		return nil, s.txError(ctx, "traitement du signalement impossible", err)
	}
	s.Logger.InfoContext(ctx, "signalement de paiement traité", "alert_id", alert.ID, "kind", alert.Kind,
		"membership_id", alert.MembershipID, "actor_id", CurrentAccount(ctx).ID)
	return s.adminMembership(ctx, s.Store.Read, alert.MembershipID)
}
