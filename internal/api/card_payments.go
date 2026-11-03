package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/helloasso"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
)

// Paiement des adhésions par carte via HelloAsso (research.md 008 › R4, R5).

const (
	MsgCardUnavailable    = "Le paiement par carte n'est pas proposé pour cette adhésion."
	MsgCardUnavailableNow = "Le paiement par carte est momentanément indisponible. Choisissez un autre moyen ou réessayez plus tard."
	MsgNotPayableByCard   = "Cette adhésion n'est pas à payer."
	MsgOnlinePaymentGone  = "Ce paiement n'existe pas."
	// MsgRefundedReason est le motif de l'annulation d'une adhésion dont le
	// paiement par carte a été remboursé dans HelloAsso.
	MsgRefundedReason = "Paiement remboursé dans HelloAsso."
)

const (
	// resumeWindow : une page de paiement HelloAsso reste valable 15 minutes ;
	// on la reprend pendant 14 minutes (R4).
	resumeWindow = 14 * time.Minute
	// verifyThrottle : au plus une relecture HelloAsso par tentative (R5).
	verifyThrottle = 5 * time.Second
	// pendingWindow : au-delà, une tentative en cours est considérée expirée (R8).
	pendingWindow = 60 * time.Minute
	// verifiableWindow : une tentative close sans paiement reste vérifiable.
	verifiableWindow = 24 * time.Hour
)

// errHelloAssoUnavailable : HelloAsso n'a pas pu être interrogé ; la
// vérification sera rejouée (notification renvoyée, page qui réessaie,
// balayage).
var errHelloAssoUnavailable = errors.New("HelloAsso injoignable")

// Sources d'une vérification (journaux, R14).
const (
	sourceReturn       = "return"
	sourceNotification = "notification"
	sourceAdmin        = "admin"
	sourceSweep        = "sweep"
)

// OnlinePayment est une tentative de paiement par carte (administration).
type OnlinePayment struct {
	ID                  int64      `json:"id"`
	Status              string     `json:"status" enum:"pending,succeeded,failed,abandoned,expired,refunded"`
	Environment         string     `json:"environment" enum:"sandbox,production"`
	AmountCents         int64      `json:"amountCents"`
	ReceivedAmountCents *int64     `json:"receivedAmountCents"`
	OrderID             *int64     `json:"orderId"`
	PaidAt              *time.Time `json:"paidAt"`
	ErrorCode           *string    `json:"errorCode"`
	CreatedAt           time.Time  `json:"createdAt"`
	FinishedAt          *time.Time `json:"finishedAt"`
	HelloAssoURL        *string    `json:"helloassoUrl" doc:"Espace HelloAsso de l'association (commande ou accueil)"`
}

// PaymentAlert est un signalement de paiement.
type PaymentAlert struct {
	ID              int64       `json:"id"`
	Kind            string      `json:"kind" enum:"duplicate,amount_mismatch,refund_due,unverified"`
	OnlinePaymentID int64       `json:"onlinePaymentId"`
	CreatedAt       time.Time   `json:"createdAt"`
	ResolvedAt      *time.Time  `json:"resolvedAt"`
	ResolvedBy      *AccountRef `json:"resolvedBy"`
}

// StartCardPayment est la page de paiement HelloAsso vers laquelle rediriger.
type StartCardPayment struct {
	RedirectURL     string `json:"redirectUrl"`
	OnlinePaymentID int64  `json:"onlinePaymentId"`
	Resumed         bool   `json:"resumed" doc:"Tentative en cours reprise"`
}

type StartCardPaymentOutput struct{ Body StartCardPayment }

// ConfirmCardPaymentBody décrit le retour du navigateur depuis HelloAsso.
type ConfirmCardPaymentBody struct {
	OnlinePaymentID  int64  `json:"onlinePaymentId" minimum:"1"`
	Outcome          string `json:"outcome" enum:"return,error,back" doc:"D'après le paramètre carte=retour|erreur|annule"`
	CheckoutIntentID int64  `json:"checkoutIntentId,omitempty" required:"false" doc:"Indicatif : seule la relecture chez HelloAsso fait foi"`
}

type ConfirmCardPaymentInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'adhésion"`
	Body ConfirmCardPaymentBody
}

// ConfirmCardPayment est l'état de la tentative après vérification.
type ConfirmCardPayment struct {
	Status     string       `json:"status" enum:"pending,succeeded,failed,abandoned,expired,refunded"`
	Membership MyMembership `json:"membership"`
}

type ConfirmCardPaymentOutput struct{ Body ConfirmCardPayment }

func (s *server) registerCardPayments(api huma.API) {
	register(api, auth.Member, huma.Operation{
		OperationID: "start-my-card-payment",
		Method:      http.MethodPost,
		Path:        "/me/memberships/{id}/card-payment",
		Summary:     "Lancer un paiement par carte (HelloAsso)",
		Tags:        []string{tagMyMemberships},
	}, s.startMyCardPayment)

	register(api, auth.Member, huma.Operation{
		OperationID: "confirm-my-card-payment",
		Method:      http.MethodPost,
		Path:        "/me/memberships/{id}/card-payment/confirm",
		Summary:     "Vérifier un paiement par carte au retour de HelloAsso",
		Tags:        []string{tagMyMemberships},
	}, s.confirmMyCardPayment)

	register(api, auth.Admin, huma.Operation{
		OperationID: "verify-card-payment",
		Method:      http.MethodPost,
		Path:        "/admin/memberships/{id}/card-payment/verify",
		Summary:     "Vérifier auprès de HelloAsso les paiements par carte en cours",
		Tags:        []string{tagMemberships},
	}, s.verifyCardPayment)
}

// cardItemName est l'intitulé transmis à HelloAsso : saison, tarif et
// référence, sans aucune donnée personnelle du membre (FR-004).
func cardItemName(seasonLabel, feeLabel, reference string) string {
	return helloasso.TruncateItemName(fmt.Sprintf("Adhésion %s — %s — %s", seasonLabel, feeLabel, reference))
}

func (s *server) startMyCardPayment(ctx context.Context, in *MembershipIDInput) (*StartCardPaymentOutput, error) {
	q, account := s.Store.Read, ownAccount(ctx)
	r, err := q.GetMembershipOfAccount(ctx, sqlc.GetMembershipOfAccountParams{ID: in.ID, AccountID: account})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgMembershipGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'adhésion impossible", err)
	}
	seasons, _, err := seasonIndex(ctx, q, s.today())
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	settings, err := s.paymentSettings(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paramètres impossible", err)
	}
	m := r.Membership
	snap := snapshotOf(m, seasons)
	if !membership.CanPay(snap) {
		return nil, huma.Error409Conflict(MsgNotPayableByCard)
	}
	if !membership.CanPayByCard(snap, settings.CardAvailable) {
		return nil, FieldErrors("body", MsgCardUnavailable)
	}

	env, amount, now := s.HelloAsso.Environment(), m.AmountCents.Int64, s.now()
	resumable, err := q.FindResumableOnlinePayment(ctx, sqlc.FindResumableOnlinePaymentParams{
		MembershipID: m.ID, Environment: env, AmountCents: amount,
		Since: now.Add(-resumeWindow).UTC().Format(timeFormat),
	})
	if err == nil {
		return &StartCardPaymentOutput{Body: StartCardPayment{RedirectURL: resumable.RedirectUrl.String, OnlinePaymentID: resumable.ID, Resumed: true}}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
	}

	op, err := s.Store.Write.CreateOnlinePayment(ctx, sqlc.CreateOnlinePaymentParams{
		MembershipID: m.ID, AccountID: account, Environment: env, AmountCents: amount, Now: s.timestamp(),
	})
	if err != nil {
		return nil, s.internalError(ctx, "création du paiement en ligne impossible", err)
	}

	req := helloasso.CheckoutRequest{
		AmountCents: amount,
		ItemName:    cardItemName(seasons[m.SeasonID].ref.Label, r.MembershipFee.Label, m.PaymentReference.String),
		Payer:       &helloasso.Payer{Email: CurrentAccount(ctx).Email},
		Metadata:    helloasso.Metadata{Payment: op.ID, Membership: m.ID, Reference: m.PaymentReference.String},
	}
	if self, err := q.GetSelfMemberOfAccount(ctx, account); err == nil {
		req.Payer.FirstName, req.Payer.LastName = self.FirstName, self.LastName
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, s.internalError(ctx, "lecture de la fiche impossible", err)
	}
	base := fmt.Sprintf("%s/adhesions/%d?tentative=%d&carte=", s.Config.BaseURL, m.ID, op.ID)
	req.ReturnURL, req.ErrorURL, req.BackURL = base+"retour", base+"erreur", base+"annule"

	intent, err := s.HelloAsso.CreateCheckout(ctx, req)
	if err != nil {
		code := helloasso.ErrorCode(err)
		if errors.Is(err, helloasso.ErrRedirectHost) {
			code = "redirect_host"
		}
		_ = s.Store.Write.FailOnlinePayment(ctx, sqlc.FailOnlinePaymentParams{ErrorCode: ns(code), Now: ns(s.timestamp()), ID: op.ID})
		s.recordHelloAssoError(ctx, "create_checkout", err)
		return nil, huma.Error503ServiceUnavailable(MsgCardUnavailableNow)
	}
	if err := s.Store.Write.SetOnlinePaymentCheckout(ctx, sqlc.SetOnlinePaymentCheckoutParams{
		CheckoutIntentID: nid(intent.ID), RedirectUrl: ns(intent.RedirectURL), Now: s.timestamp(), ID: op.ID,
	}); err != nil {
		return nil, s.internalError(ctx, "enregistrement du paiement en ligne impossible", err)
	}
	s.clearHelloAssoError(ctx)
	s.Logger.InfoContext(ctx, "paiement en ligne démarré",
		"membership_id", m.ID, "online_payment_id", op.ID, "checkout_intent_id", intent.ID,
		"environment", env, "amount_cents", amount, "actor_id", account.Int64)
	return &StartCardPaymentOutput{Body: StartCardPayment{RedirectURL: intent.RedirectURL, OnlinePaymentID: op.ID}}, nil
}

func (s *server) confirmMyCardPayment(ctx context.Context, in *ConfirmCardPaymentInput) (*ConfirmCardPaymentOutput, error) {
	op, err := s.Store.Read.GetOnlinePaymentOfAccount(ctx, sqlc.GetOnlinePaymentOfAccountParams{
		ID: in.Body.OnlinePaymentID, MembershipID: in.ID, AccountID: ownAccount(ctx),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgOnlinePaymentGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture du paiement en ligne impossible", err)
	}
	hint := map[string]string{"return": membership.HintReturn, "error": membership.HintError, "back": membership.HintBack}[in.Body.Outcome]
	status, err := s.verifyOnlinePayment(ctx, op.ID, sourceReturn, hint)
	if errors.Is(err, errHelloAssoUnavailable) {
		status, err = membership.OnlineStatus(op.Status), nil
	}
	if err != nil {
		return nil, s.internalError(ctx, "vérification du paiement impossible", err)
	}
	out, err := s.myMembershipOutput(ctx, s.Store.Read, in.ID)
	if err != nil {
		return nil, err
	}
	return &ConfirmCardPaymentOutput{Body: ConfirmCardPayment{Status: string(status), Membership: out.Body}}, nil
}

func (s *server) verifyCardPayment(ctx context.Context, in *MembershipIDInput) (*AdminMembershipOutput, error) {
	if _, err := s.Store.Read.GetMembership(ctx, in.ID); errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgMembershipGone)
	} else if err != nil {
		return nil, s.internalError(ctx, "lecture de l'adhésion impossible", err)
	}
	if s.HelloAsso == nil {
		return nil, huma.Error503ServiceUnavailable(MsgCardUnavailableNow)
	}
	ops, err := s.Store.Read.ListVerifiableOnlinePaymentsOfMembership(ctx, sqlc.ListVerifiableOnlinePaymentsOfMembershipParams{
		MembershipID: in.ID, Since: ns(s.now().Add(-verifiableWindow).UTC().Format(timeFormat)),
	})
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
	}
	for _, op := range ops {
		if _, err := s.verifyOnlinePayment(ctx, op.ID, sourceAdmin, membership.HintNone); errors.Is(err, errHelloAssoUnavailable) {
			return nil, huma.Error503ServiceUnavailable(MsgCardUnavailableNow)
		} else if err != nil {
			return nil, s.internalError(ctx, "vérification du paiement impossible", err)
		}
	}
	return s.adminMembership(ctx, s.Store.Read, in.ID)
}

// verifyOnlinePayment est le seul chemin qui constate un paiement par carte
// (research.md 008 › R5) : relecture authentifiée de l'intention chez
// HelloAsso hors transaction, contrôle du rattachement, puis décision par le
// domaine après relecture en transaction d'écriture. Idempotente.
func (s *server) verifyOnlinePayment(ctx context.Context, id int64, source, hint string) (membership.OnlineStatus, error) {
	op, err := s.Store.Read.GetOnlinePayment(ctx, id)
	if err != nil {
		return "", err
	}
	status := membership.OnlineStatus(op.Status)
	// Un paiement réussi n'est relu que pour y chercher un remboursement, sur
	// notification ou à la demande d'un administrateur.
	refundCheck := status == membership.OnlineSucceeded && (source == sourceNotification || source == sourceAdmin)
	if (status == membership.OnlineSucceeded && !refundCheck) || status == membership.OnlineRefunded ||
		s.HelloAsso == nil || op.Environment != s.HelloAsso.Environment() || !op.CheckoutIntentID.Valid {
		return status, nil
	}

	var v membership.Verified
	throttled := op.LastVerifiedAt.Valid && s.now().Sub(parseTime(op.LastVerifiedAt.String)) < verifyThrottle
	if throttled {
		// Vérifiée il y a moins de 5 s sans paiement : seul l'indice du
		// navigateur peut clore la tentative.
		if hint == membership.HintNone || hint == membership.HintReturn {
			return status, nil
		}
	} else {
		if err := s.Store.Write.TouchOnlinePaymentVerified(ctx, sqlc.TouchOnlinePaymentVerifiedParams{Now: ns(s.timestamp()), ID: op.ID}); err != nil {
			return "", err
		}
		ch, err := s.HelloAsso.GetCheckout(ctx, op.CheckoutIntentID.Int64)
		if err != nil {
			s.recordHelloAssoError(ctx, "get_checkout", err)
			return status, fmt.Errorf("%w : %s", errHelloAssoUnavailable, helloasso.ErrorCode(err))
		}
		s.clearHelloAssoError(ctx)
		if ch.Metadata == nil || ch.Metadata.Payment != op.ID || ch.Metadata.Membership != op.MembershipID {
			s.Logger.WarnContext(ctx, "intention HelloAsso sans rapport avec la tentative",
				"online_payment_id", op.ID, "checkout_intent_id", op.CheckoutIntentID.Int64, "source", source)
			return status, nil
		}
		if refundCheck {
			if ch.Order != nil {
				if p, ok := ch.Order.PaymentByID(op.HelloassoPaymentID.Int64); ok && p.Refunded() {
					return s.settleRefund(ctx, op.ID, source)
				}
			}
			return status, nil
		}
		if ch.Order != nil {
			if p, ok := ch.Order.AuthorizedPayment(); ok {
				v = membership.Verified{Authorized: true, ReceivedCents: p.ReceivedCents(), PaidAt: p.Date}
				v.Hint = hint
				return s.settleOnlinePayment(ctx, op.ID, source, v, ch.Order.ID, p.ID)
			}
		}
	}
	v.Hint = hint
	return s.settleOnlinePayment(ctx, op.ID, source, v, 0, 0)
}

// settleOnlinePayment applique la décision du domaine dans une transaction
// d'écriture, après relecture de la tentative et de l'adhésion.
func (s *server) settleOnlinePayment(ctx context.Context, id int64, source string, v membership.Verified, orderID, paymentID int64) (membership.OnlineStatus, error) {
	var result membership.OnlineStatus
	var settled membership.Settlement
	var membershipID int64
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		op, err := q.GetOnlinePayment(ctx, id)
		if err != nil {
			return err
		}
		membershipID = op.MembershipID
		r, err := q.GetMembership(ctx, op.MembershipID)
		if err != nil {
			return err
		}
		seasons, _, err := seasonIndex(ctx, q, s.today())
		if err != nil {
			return err
		}
		current := membership.OnlineStatus(op.Status)
		settled = membership.SettleOnline(current, snapshotOf(r.Membership, seasons), v)
		result = settled.NewStatus
		if settled.NewStatus == current && !settled.RecordPayment && settled.Alert == membership.NoAlert {
			return nil
		}
		now := s.timestamp()
		switch settled.NewStatus {
		case current:
		case membership.OnlineSucceeded:
			paidAt := v.PaidAt
			if paidAt.IsZero() {
				paidAt = s.now()
			}
			if _, err := q.SucceedOnlinePayment(ctx, sqlc.SucceedOnlinePaymentParams{
				OrderID: nid(orderID), HelloassoPaymentID: nid(paymentID), ReceivedAmountCents: nid(v.ReceivedCents),
				PaidAt: ns(paidAt.UTC().Format(timeFormat)), Now: ns(now), ID: op.ID,
			}); err != nil {
				return err
			}
			if settled.RecordPayment {
				if _, err := q.RecordCardPayment(ctx, sqlc.RecordCardPaymentParams{
					PaidOn: ns(season.Today(paidAt).String()), Now: ns(now), PaidAmountCents: nid(v.ReceivedCents),
					OnlinePaymentID: nid(op.ID), ID: op.MembershipID,
				}); err != nil {
					return err
				}
				if err := s.notifyMembership(ctx, q, "membership-paid", op.MembershipID, false); err != nil {
					return err
				}
			}
		default:
			code := sql.NullString{}
			if settled.NewStatus == membership.OnlineFailed {
				code = ns("payment_error")
			}
			if _, err := q.CloseOnlinePayment(ctx, sqlc.CloseOnlinePaymentParams{
				Status: string(settled.NewStatus), ErrorCode: code, Now: ns(now), ID: op.ID,
			}); err != nil {
				return err
			}
		}
		if settled.Alert != membership.NoAlert {
			return q.CreatePaymentAlert(ctx, sqlc.CreatePaymentAlertParams{
				MembershipID: op.MembershipID, OnlinePaymentID: op.ID, Kind: string(settled.Alert), Now: now,
			})
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	attrs := []any{"online_payment_id", id, "membership_id", membershipID, "source", source, "outcome", string(result)}
	if orderID != 0 {
		attrs = append(attrs, "order_id", orderID)
	}
	s.Logger.InfoContext(ctx, "paiement en ligne vérifié", attrs...)
	if settled.Alert != membership.NoAlert {
		s.Logger.WarnContext(ctx, "signalement de paiement", "kind", string(settled.Alert), "online_payment_id", id, "membership_id", membershipID)
	}
	return result, nil
}

// settleRefund constate le remboursement vérifié d'un paiement réussi : la
// tentative passe à « remboursée », l'adhésion qu'elle a réglée est annulée si
// elle ne l'est pas déjà, et ses signalements sont clos.
func (s *server) settleRefund(ctx context.Context, id int64, source string) (membership.OnlineStatus, error) {
	var membershipID int64
	cancelled := false
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		op, err := q.GetOnlinePayment(ctx, id)
		if err != nil {
			return err
		}
		membershipID = op.MembershipID
		current := membership.OnlineStatus(op.Status)
		if current != membership.OnlineSucceeded {
			return nil
		}
		r, err := q.GetMembership(ctx, op.MembershipID)
		if err != nil {
			return err
		}
		seasons, _, err := seasonIndex(ctx, q, s.today())
		if err != nil {
			return err
		}
		now := ns(s.timestamp())
		m := r.Membership
		paidByIt := m.PaidOnlinePaymentID.Valid && m.PaidOnlinePaymentID.Int64 == op.ID
		if _, err := q.RefundOnlinePayment(ctx, sqlc.RefundOnlinePaymentParams{Now: now, ID: op.ID}); err != nil {
			return err
		}
		if membership.RefundCancels(current, snapshotOf(m, seasons), paidByIt) {
			if _, err := q.CancelRefundedMembership(ctx, sqlc.CancelRefundedMembershipParams{
				Reason: ns(MsgRefundedReason), Now: now, ID: m.ID, OnlinePaymentID: nid(op.ID),
			}); err != nil {
				return err
			}
			cancelled = true
			if err := s.notifyMembership(ctx, q, "membership-cancelled", m.ID, false); err != nil {
				return err
			}
		}
		return q.ResolveAlertsOfOnlinePayment(ctx, sqlc.ResolveAlertsOfOnlinePaymentParams{Now: now, OnlinePaymentID: op.ID})
	})
	if err != nil {
		return "", err
	}
	s.Logger.InfoContext(ctx, "paiement en ligne remboursé", "online_payment_id", id, "membership_id", membershipID,
		"source", source, "membership_cancelled", cancelled)
	return membership.OnlineRefunded, nil
}

// recordHelloAssoError mémorise un incident d'appel (tableau de bord, R9).
func (s *server) recordHelloAssoError(ctx context.Context, operation string, err error) {
	code := helloasso.ErrorCode(err)
	s.Logger.WarnContext(ctx, "appel HelloAsso en échec", "operation", operation, "error_code", code)
	if e := s.Store.Write.RecordHelloAssoError(ctx, sqlc.RecordHelloAssoErrorParams{Now: ns(s.timestamp()), ErrorCode: ns(code)}); e != nil {
		s.Logger.ErrorContext(ctx, "enregistrement de l'incident HelloAsso impossible", "error", e)
	}
}

func (s *server) clearHelloAssoError(ctx context.Context) {
	if err := s.Store.Write.ClearHelloAssoError(ctx); err != nil {
		s.Logger.ErrorContext(ctx, "effacement de l'incident HelloAsso impossible", "error", err)
	}
}

// helloAssoSpaceURL renvoie l'espace HelloAsso de l'association
// (research.md 008 › R13 ; forme exacte à confirmer en environnement de test).
func (s *server) helloAssoSpaceURL(environment string) *string {
	if s.HelloAsso == nil {
		return nil
	}
	host := "https://admin.helloasso.com/"
	if environment == helloasso.Sandbox {
		host = "https://admin.helloasso-sandbox.com/"
	}
	u := host + url.PathEscape(s.HelloAsso.OrganizationSlug())
	return &u
}

// withOnlineAdmin complète une adhésion vue par l'administration avec ses
// tentatives et ses signalements.
func (s *server) withOnlineAdmin(ctx context.Context, q *sqlc.Queries, out *AdminMembership, m sqlc.Membership) error {
	ops, err := q.ListOnlinePaymentsOfMembership(ctx, m.ID)
	if err != nil {
		return err
	}
	out.OnlinePayments = make([]OnlinePayment, 0, len(ops))
	cutoff := s.now().Add(-pendingWindow)
	since := s.now().Add(-verifiableWindow)
	for _, op := range ops {
		o := OnlinePayment{
			ID: op.ID, Status: op.Status, Environment: op.Environment, AmountCents: op.AmountCents,
			ReceivedAmountCents: optionalInt(op.ReceivedAmountCents), OrderID: optionalInt(op.OrderID),
			PaidAt: optionalTimeValue(op.PaidAt), ErrorCode: optionalValue(op.ErrorCode),
			CreatedAt: parseTime(op.CreatedAt), FinishedAt: optionalTimeValue(op.FinishedAt),
		}
		if op.OrderID.Valid {
			o.HelloAssoURL = s.helloAssoSpaceURL(op.Environment)
		}
		out.OnlinePayments = append(out.OnlinePayments, o)
		if op.Status == string(membership.OnlinePending) && o.CreatedAt.After(cutoff) {
			out.CardPaymentPending = true
		}
		// Une tentative réussie reste vérifiable : un remboursement a pu être
		// fait dans HelloAsso.
		verifiable := op.CheckoutIntentID.Valid && (op.Status == string(membership.OnlinePending) ||
			op.Status == string(membership.OnlineSucceeded) ||
			(op.Status != string(membership.OnlineRefunded) && op.FinishedAt.Valid && parseTime(op.FinishedAt.String).After(since)))
		if verifiable && s.HelloAsso != nil && op.Environment == s.HelloAsso.Environment() {
			out.Actions.VerifyCardPayment = true
		}
	}
	alerts, err := q.ListPaymentAlertsOfMembership(ctx, m.ID)
	if err != nil {
		return err
	}
	out.Alerts = make([]PaymentAlert, 0, len(alerts))
	for _, a := range alerts {
		out.Alerts = append(out.Alerts, PaymentAlert{
			ID: a.ID, Kind: a.Kind, OnlinePaymentID: a.OnlinePaymentID, CreatedAt: parseTime(a.CreatedAt),
			ResolvedAt: optionalTimeValue(a.ResolvedAt), ResolvedBy: accountRef(a.ResolvedBy, a.ResolvedByEmail),
		})
	}
	return nil
}
