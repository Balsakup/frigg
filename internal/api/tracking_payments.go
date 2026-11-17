package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/helloasso"
)

// Paiement par carte depuis un lien personnel (016 › R11, FR-019 à FR-021).
// Seules la résolution de la cible, celle du payeur et l'adresse de retour
// changent : la constatation reste celle de 008 (verifyOnlinePayment).

// ConfirmTrackedCardPayment est l'état de la tentative après vérification.
type ConfirmTrackedCardPayment struct {
	Status     string            `json:"status" enum:"pending,succeeded,failed,abandoned,expired,refunded"`
	Membership TrackedMembership `json:"membership"`
}

type ConfirmTrackedCardPaymentOutput struct{ Body ConfirmTrackedCardPayment }

// ConfirmTrackedRegistrationCard est l'état de la tentative après
// vérification, pour une inscription suivie.
type ConfirmTrackedRegistrationCard struct {
	Status       string              `json:"status" enum:"pending,succeeded,failed,abandoned,expired,refunded"`
	Registration TrackedRegistration `json:"registration"`
}

type ConfirmTrackedRegistrationCardOutput struct {
	Body ConfirmTrackedRegistrationCard
}

func (s *server) registerTrackingPayments(api huma.API) {
	register(api, auth.Link, huma.Operation{
		OperationID: "start-tracking-membership-card-payment",
		Method:      http.MethodPost,
		Path:        "/tracking/memberships/{id}/card-payment",
		Summary:     "Lancer le paiement par carte d'une adhésion suivie (HelloAsso)",
		Tags:        []string{tagTracking},
	}, s.startTrackingMembershipCardPayment)

	register(api, auth.Link, huma.Operation{
		OperationID: "confirm-tracking-membership-card-payment",
		Method:      http.MethodPost,
		Path:        "/tracking/memberships/{id}/card-payment/confirm",
		Summary:     "Vérifier un paiement par carte au retour de HelloAsso",
		Tags:        []string{tagTracking},
	}, s.confirmTrackingMembershipCardPayment)

	register(api, auth.Link, huma.Operation{
		OperationID: "start-tracking-registration-card-payment",
		Method:      http.MethodPost,
		Path:        "/tracking/registrations/{id}/card-payment",
		Summary:     "Lancer le paiement par carte d'une inscription suivie (HelloAsso)",
		Tags:        []string{tagTracking},
	}, s.startTrackingRegistrationCardPayment)

	register(api, auth.Link, huma.Operation{
		OperationID: "confirm-tracking-registration-card-payment",
		Method:      http.MethodPost,
		Path:        "/tracking/registrations/{id}/card-payment/confirm",
		Summary:     "Vérifier le paiement par carte d'une inscription au retour de HelloAsso",
		Tags:        []string{tagTracking},
	}, s.confirmTrackingRegistrationCardPayment)
}

func (s *server) startTrackingRegistrationCardPayment(ctx context.Context, in *RegistrationIDInput) (*StartCardPaymentOutput, error) {
	q, email := s.Store.Read, trackedEmail(ctx)
	r, err := q.GetRegistrationOfTrackedEmail(ctx, sqlc.GetRegistrationOfTrackedEmailParams{ID: in.ID, Email: ns(email)})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgRegistrationGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'inscription impossible", err)
	}
	// Payeur : adresse du lien, prénom et nom de la fiche, rien d'autre (FR-020).
	payer := helloasso.Payer{Email: email, FirstName: r.FirstName, LastName: r.LastName}
	return s.startRegistrationCard(ctx, q, r.EventRegistration, r.Event, cardOrigin{
		viaLink: true, returnPath: "/suivi/inscriptions/%d",
		payer: func() (helloasso.Payer, error) { return payer, nil },
	})
}

func (s *server) confirmTrackingRegistrationCardPayment(ctx context.Context, in *ConfirmRegistrationCardInput) (*ConfirmTrackedRegistrationCardOutput, error) {
	op, err := s.Store.Read.GetOnlinePaymentOfTrackedEmail(ctx, sqlc.GetOnlinePaymentOfTrackedEmailParams{
		ID: in.Body.OnlinePaymentID, RegistrationID: nid(in.ID), Email: ns(trackedEmail(ctx)),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgOnlinePaymentGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture du paiement en ligne impossible", err)
	}
	status, err := s.confirmCard(ctx, op, in.Body.Outcome)
	if err != nil {
		return nil, err
	}
	out, err := s.trackedRegistrationOutput(ctx, s.Store.Read, in.ID)
	if err != nil {
		return nil, err
	}
	return &ConfirmTrackedRegistrationCardOutput{Body: ConfirmTrackedRegistrationCard{Status: string(status), Registration: out.Body}}, nil
}

func (s *server) startTrackingMembershipCardPayment(ctx context.Context, in *MembershipIDInput) (*StartCardPaymentOutput, error) {
	q, email := s.Store.Read, trackedEmail(ctx)
	r, err := q.GetMembershipOfTrackedEmail(ctx, sqlc.GetMembershipOfTrackedEmailParams{ID: in.ID, Email: ns(email)})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgMembershipGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'adhésion impossible", err)
	}
	// Payeur : adresse du lien, prénom et nom de la fiche, rien d'autre (FR-020).
	payer := helloasso.Payer{Email: email, FirstName: r.FirstName, LastName: r.LastName}
	return s.startMembershipCard(ctx, q, r.Membership, r.MembershipFee.Label, cardOrigin{
		viaLink: true, returnPath: "/suivi/adhesions/%d",
		payer: func() (helloasso.Payer, error) { return payer, nil },
	})
}

func (s *server) confirmTrackingMembershipCardPayment(ctx context.Context, in *ConfirmCardPaymentInput) (*ConfirmTrackedCardPaymentOutput, error) {
	op, err := s.Store.Read.GetOnlinePaymentOfTrackedEmail(ctx, sqlc.GetOnlinePaymentOfTrackedEmailParams{
		ID: in.Body.OnlinePaymentID, MembershipID: nid(in.ID), Email: ns(trackedEmail(ctx)),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgOnlinePaymentGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture du paiement en ligne impossible", err)
	}
	status, err := s.confirmCard(ctx, op, in.Body.Outcome)
	if err != nil {
		return nil, err
	}
	out, err := s.trackedMembershipOutput(ctx, s.Store.Read, in.ID)
	if err != nil {
		return nil, err
	}
	return &ConfirmTrackedCardPaymentOutput{Body: ConfirmTrackedCardPayment{Status: string(status), Membership: out.Body}}, nil
}
