package api

import (
	"context"
	"database/sql"
	"errors"

	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/membership"
)

// Cible d'un paiement par carte : une adhésion ou une inscription à un
// événement (research.md 012 › R4). C'est le seul point de variation du
// circuit de constatation, qui reste unique (008 › R5).
type payTarget interface {
	// snapshot présente la cible au domaine de paiement.
	snapshot() membership.Snapshot
	// paidByAttempt : la cible a été réglée par cette tentative.
	paidByAttempt(opID int64) bool
	recordCard(ctx context.Context, q *sqlc.Queries, paidOn, now string, amount, opID int64) error
	cancelRefunded(ctx context.Context, q *sqlc.Queries, now string, opID int64) error
	notifyPaid(ctx context.Context, q *sqlc.Queries) error
	notifyCancelled(ctx context.Context, q *sqlc.Queries) error
	// alertTarget renvoie les colonnes de cible d'un signalement.
	alertTarget() (membershipID, registrationID sql.NullInt64)
	logAttrs() []any
}

var errUnknownTarget = errors.New("tentative sans cible")

// loadTarget relit la cible d'une tentative avec q (dans la transaction).
func (s *server) loadTarget(ctx context.Context, q *sqlc.Queries, op sqlc.OnlinePayment) (payTarget, error) {
	switch {
	case op.MembershipID.Valid:
		r, err := q.GetMembership(ctx, op.MembershipID.Int64)
		if err != nil {
			return nil, err
		}
		seasons, _, err := seasonIndex(ctx, q, s.today())
		if err != nil {
			return nil, err
		}
		return &membershipTarget{s: s, m: r.Membership, snap: snapshotOf(r.Membership, seasons)}, nil
	case op.RegistrationID.Valid:
		return s.loadRegistrationTarget(ctx, q, op.RegistrationID.Int64)
	}
	return nil, errUnknownTarget
}

// targetAttrs : attributs de journal de la cible d'une tentative, sans lecture.
func targetAttrs(op sqlc.OnlinePayment) []any {
	if op.RegistrationID.Valid {
		return []any{"registration_id", op.RegistrationID.Int64}
	}
	return []any{"membership_id", op.MembershipID.Int64}
}

type membershipTarget struct {
	s    *server
	m    sqlc.Membership
	snap membership.Snapshot
}

func (t *membershipTarget) snapshot() membership.Snapshot { return t.snap }

func (t *membershipTarget) paidByAttempt(opID int64) bool {
	return t.m.PaidOnlinePaymentID.Valid && t.m.PaidOnlinePaymentID.Int64 == opID
}

func (t *membershipTarget) recordCard(ctx context.Context, q *sqlc.Queries, paidOn, now string, amount, opID int64) error {
	_, err := q.RecordCardPayment(ctx, sqlc.RecordCardPaymentParams{
		PaidOn: ns(paidOn), Now: ns(now), PaidAmountCents: nid(amount), OnlinePaymentID: nid(opID), ID: t.m.ID,
	})
	return err
}

func (t *membershipTarget) cancelRefunded(ctx context.Context, q *sqlc.Queries, now string, opID int64) error {
	_, err := q.CancelRefundedMembership(ctx, sqlc.CancelRefundedMembershipParams{
		Reason: ns(MsgRefundedReason), Now: ns(now), ID: t.m.ID, OnlinePaymentID: nid(opID),
	})
	return err
}

func (t *membershipTarget) notifyPaid(ctx context.Context, q *sqlc.Queries) error {
	return t.s.notifyMembership(ctx, q, "membership-paid", t.m.ID, false)
}

func (t *membershipTarget) notifyCancelled(ctx context.Context, q *sqlc.Queries) error {
	return t.s.notifyMembership(ctx, q, "membership-cancelled", t.m.ID, false)
}

func (t *membershipTarget) alertTarget() (sql.NullInt64, sql.NullInt64) {
	return nid(t.m.ID), sql.NullInt64{}
}

func (t *membershipTarget) logAttrs() []any { return []any{"membership_id", t.m.ID} }
