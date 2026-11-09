package api

import (
	"context"
	"database/sql"

	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/event"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
)

// Cible « inscription » du circuit de paiement par carte (012 › R4).

// MsgRefundedRegistration est le motif annoncé quand une inscription réglée
// par carte est annulée après un remboursement dans HelloAsso.
const MsgRefundedRegistration = "Paiement remboursé dans HelloAsso."

type registrationTarget struct {
	s  *server
	r  sqlc.EventRegistration
	ev sqlc.Event
}

func (s *server) loadRegistrationTarget(ctx context.Context, q *sqlc.Queries, id int64) (payTarget, error) {
	row, err := q.GetRegistration(ctx, id)
	if err != nil {
		return nil, err
	}
	return &registrationTarget{s: s, r: row.EventRegistration, ev: row.Event}, nil
}

// regSnapshot présente une inscription au domaine (sans tentative en cours).
func regSnapshot(r sqlc.EventRegistration) event.RegSnapshot {
	return event.RegSnapshot{
		Confirmed:     r.Status == "confirmed",
		AmountCents:   r.AmountCents,
		PaymentStatus: membership.PaymentStatus(r.PaymentStatus.String),
		PaidMethod:    membership.Method(r.PaidMethod.String),
	}
}

func (t *registrationTarget) snapshot() membership.Snapshot {
	return event.PaymentSnapshot(regSnapshot(t.r), t.ev.Status == string(event.Cancelled))
}

func (t *registrationTarget) paidByAttempt(opID int64) bool {
	return t.r.PaidOnlinePaymentID.Valid && t.r.PaidOnlinePaymentID.Int64 == opID
}

func (t *registrationTarget) recordCard(ctx context.Context, q *sqlc.Queries, paidOn, now string, amount, opID int64) error {
	_, err := q.RecordRegistrationCardPayment(ctx, sqlc.RecordRegistrationCardPaymentParams{
		PaidOn: ns(paidOn), Now: ns(now), PaidAmountCents: nid(amount), OnlinePaymentID: nid(opID), ID: t.r.ID,
	})
	return err
}

func (t *registrationTarget) cancelRefunded(ctx context.Context, q *sqlc.Queries, now string, opID int64) error {
	_, err := q.CancelRefundedRegistration(ctx, sqlc.CancelRefundedRegistrationParams{
		Now: ns(now), ID: t.r.ID, OnlinePaymentID: nid(opID),
	})
	return err
}

func (t *registrationTarget) notifyPaid(ctx context.Context, q *sqlc.Queries) error {
	return t.s.notifyRegistrationPaid(ctx, q, t.ev, t.r.ID)
}

func (t *registrationTarget) notifyCancelled(ctx context.Context, q *sqlc.Queries) error {
	return t.s.notifyRegistrations(ctx, q, "registration-removed", t.ev, []int64{t.r.ID},
		mail.Data{Reason: MsgRefundedRegistration})
}

func (t *registrationTarget) alertTarget() (sql.NullInt64, sql.NullInt64) {
	return sql.NullInt64{}, nid(t.r.ID)
}

func (t *registrationTarget) logAttrs() []any {
	return []any{"registration_id", t.r.ID, "event_id", t.ev.ID}
}

// notifyRegistrationPaid relit l'inscription payée pour l'e-mail de
// constatation (montant, moyen, date).
func (s *server) notifyRegistrationPaid(ctx context.Context, q *sqlc.Queries, ev sqlc.Event, id int64) error {
	r, err := q.GetRegistrationNotice(ctx, id)
	if err != nil {
		return err
	}
	data := mail.Data{Method: methodLabels[r.PaidMethod.String], Amount: formatEuros(r.AmountCents)}
	if r.PaidAmountCents.Valid {
		data.Amount = formatEuros(r.PaidAmountCents.Int64)
	}
	if d, err := season.ParseDate(r.PaidOn.String); err == nil && r.PaidOn.Valid {
		data.PaidOn = frenchDate(d)
	}
	return s.notifyRegistrations(ctx, q, "registration-paid", ev, []int64{id}, data)
}
