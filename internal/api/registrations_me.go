package api

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/config"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/event"
	"github.com/balsakup/frigg/internal/helloasso"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
)

// Inscriptions dans l'espace membre : « Mes inscriptions », annulation,
// paiement et « À régler » (US3, US4 ; contracts/api.md 012).

// MyRegistrationActions sont les actions proposées au compte gestionnaire.
type MyRegistrationActions struct {
	Cancel    bool `json:"cancel"`
	Pay       bool `json:"pay"`
	PayByCard bool `json:"payByCard"`
}

// MyRegistration est une inscription d'une fiche gérée par le compte.
type MyRegistration struct {
	ID             int64                 `json:"id"`
	Event          EventSummary          `json:"event"`
	Member         MemberRef             `json:"member"`
	State          string                `json:"state" enum:"confirmed,cancelled"`
	RegisteredAt   time.Time             `json:"registeredAt"`
	CancelledAt    *time.Time            `json:"cancelledAt"`
	AmountCents    int64                 `json:"amountCents" doc:"Prix figé à l'inscription ; 0 : gratuite"`
	Payment        *Payment              `json:"payment" doc:"null : inscription gratuite"`
	PaymentMethods []PaymentMethodInfo   `json:"paymentMethods" doc:"Moyens activés ; vide s'il n'y a rien à payer ou dans la liste"`
	CardPayment    *CardPaymentState     `json:"cardPayment" doc:"Dernière tentative de paiement par carte"`
	Actions        MyRegistrationActions `json:"actions"`
}

type MyRegistrationOutput struct{ Body MyRegistration }

type MyRegistrationListOutput struct {
	Body struct {
		Items []MyRegistration `json:"items" doc:"À venir puis passées"`
	}
}

type RegistrationIDInput struct {
	ID int64 `path:"id" doc:"Identifiant de l'inscription"`
}

type AnnounceRegistrationInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'inscription"`
	Body AnnounceBody
}

type ConfirmRegistrationCardInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'inscription"`
	Body ConfirmCardPaymentBody
}

// ConfirmRegistrationCard est l'état de la tentative après vérification.
type ConfirmRegistrationCard struct {
	Status       string         `json:"status" enum:"pending,succeeded,failed,abandoned,expired,refunded"`
	Registration MyRegistration `json:"registration"`
}

type ConfirmRegistrationCardOutput struct{ Body ConfirmRegistrationCard }

// Due est une somme à régler par le compte (R11).
type Due struct {
	Kind        string    `json:"kind" enum:"membership,registration"`
	ID          int64     `json:"id"`
	Member      MemberRef `json:"member"`
	Label       string    `json:"label" doc:"« Adhésion 2026-2027 » ou titre de l'événement"`
	AmountCents int64     `json:"amountCents"`
	Payment     Payment   `json:"payment"`
	CreatedAt   time.Time `json:"createdAt"`
}

type DueListOutput struct {
	Body struct {
		Items []Due `json:"items" doc:"Par date"`
	}
}

func (s *server) registerMyRegistrations(api huma.API) {
	op := func(id, method, path, summary string) huma.Operation {
		return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{tagRegistrations}}
	}
	register(api, auth.Member, op("list-my-registrations", http.MethodGet, "/me/registrations", "Mes inscriptions"), s.listMyRegistrations)
	register(api, auth.Member, op("get-my-registration", http.MethodGet, "/me/registrations/{id}", "Une inscription"), s.getMyRegistration)
	register(api, auth.Member, op("cancel-my-registration", http.MethodPost, "/me/registrations/{id}/cancel", "Annuler une inscription"), s.cancelMyRegistration)
	register(api, auth.Member, op("announce-my-registration-payment", http.MethodPut, "/me/registrations/{id}/payment", "Choisir le moyen de paiement d'une inscription"), s.announceMyRegistrationPayment)
	register(api, auth.Member, op("start-my-registration-card-payment", http.MethodPost, "/me/registrations/{id}/card-payment", "Payer une inscription par carte"), s.startMyRegistrationCardPayment)
	register(api, auth.Member, op("confirm-my-registration-card-payment", http.MethodPost, "/me/registrations/{id}/card-payment/confirm", "Vérifier un paiement par carte au retour de HelloAsso"), s.confirmMyRegistrationCardPayment)
	register(api, auth.Member, op("list-my-dues", http.MethodGet, "/me/dues", "À régler : adhésions et inscriptions"), s.listMyDues)
}

// registrationPayment est le paiement d'une inscription payante.
func registrationPayment(ctx context.Context, q *sqlc.Queries, r sqlc.EventRegistration) (*Payment, error) {
	if !r.PaymentStatus.Valid || !r.PaymentReference.Valid {
		return nil, nil
	}
	p := &Payment{
		Reference:       r.PaymentReference.String,
		Status:          r.PaymentStatus.String,
		AnnouncedMethod: optionalValue(r.AnnouncedMethod),
		PaidMethod:      optionalValue(r.PaidMethod),
		PaidOn:          optionalValue(r.PaidOn),
		PaidAmountCents: optionalInt(r.PaidAmountCents),
		DonationCents:   membership.Donation(membership.Method(r.PaidMethod.String), r.AmountCents, r.PaidAmountCents.Int64),
	}
	if r.PaidOnlinePaymentID.Valid {
		op, err := q.GetOnlinePayment(ctx, r.PaidOnlinePaymentID.Int64)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if err == nil {
			p.Online = &OnlinePaymentRef{OrderID: op.OrderID.Int64, PaidAt: parseTime(op.PaidAt.String), Test: op.Environment == config.HelloAssoSandbox}
		}
	}
	return p, nil
}

// registrationState rassemble paiement, actions et dernière tentative.
type registrationState struct {
	payment *Payment
	actions MyRegistrationActions
	card    *CardPaymentState
	methods []PaymentMethodInfo
	snap    event.RegSnapshot
}

func (s *server) myRegistrationState(ctx context.Context, q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event,
	settings membership.Settings, now time.Time) (registrationState, error) {
	var out registrationState
	var err error
	if out.payment, err = registrationPayment(ctx, q, r); err != nil {
		return out, err
	}
	out.snap = regSnapshot(r)
	if r.PaymentStatus.Valid {
		op, err := q.LatestOnlinePaymentOfRegistration(ctx, r.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return out, err
		}
		if err == nil {
			out.card = &CardPaymentState{Status: op.Status, StartedAt: parseTime(op.CreatedAt)}
			out.snap.CardPending = op.Status == string(membership.OnlinePending) && parseTime(op.CreatedAt).After(now.Add(-pendingWindow))
		}
	}
	snap := event.PaymentSnapshot(out.snap, e.Status == string(event.Cancelled))
	a := membership.MemberActionsFor(snap, settings.CardAvailable)
	out.actions = MyRegistrationActions{Cancel: event.CanUnregister(eventState(e), out.snap, now), Pay: a.Pay, PayByCard: a.PayByCard}
	out.methods = []PaymentMethodInfo{}
	if a.PayByCard {
		amount := snap.AmountCents
		out.methods = append(out.methods, PaymentMethodInfo{Method: string(membership.Card), AmountCents: &amount, Test: settings.CardTest})
	}
	if a.Pay {
		out.methods = append(out.methods, paymentMethodInfos(settings)...)
	}
	return out, nil
}

func (s *server) toMyRegistration(ctx context.Context, q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, first, last string,
	confirmed int64, seasons map[int64]seasonInfo, settings membership.Settings, now time.Time, withMethods bool) (MyRegistration, error) {
	st, err := s.myRegistrationState(ctx, q, r, e, settings, now)
	if err != nil {
		return MyRegistration{}, err
	}
	out := MyRegistration{
		ID: r.ID, Event: eventSummary(e, confirmed, seasons, now),
		Member: MemberRef{ID: r.MemberID, FirstName: first, LastName: last},
		State:  r.Status, RegisteredAt: parseTime(r.RegisteredAt), CancelledAt: optionalTimeValue(r.CancelledAt),
		AmountCents: r.AmountCents, Payment: st.payment, CardPayment: st.card, Actions: st.actions,
		PaymentMethods: []PaymentMethodInfo{},
	}
	if withMethods {
		out.PaymentMethods = st.methods
	}
	return out, nil
}

func (s *server) listMyRegistrations(ctx context.Context, _ *struct{}) (*MyRegistrationListOutput, error) {
	q, now := s.Store.Read, s.now()
	rows, err := q.ListRegistrationsOfAccount(ctx, ownAccount(ctx))
	if err != nil {
		return nil, s.internalError(ctx, "lecture des inscriptions impossible", err)
	}
	seasons, _, err := seasonIndex(ctx, q, season.Today(now))
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	settings, err := s.paymentSettings(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paramètres impossible", err)
	}
	out := &MyRegistrationListOutput{}
	out.Body.Items = make([]MyRegistration, 0, len(rows))
	for _, r := range rows {
		confirmed, err := q.CountConfirmedRegistrations(ctx, r.Event.ID)
		if err != nil {
			return nil, s.internalError(ctx, "lecture des inscriptions impossible", err)
		}
		item, err := s.toMyRegistration(ctx, q, r.EventRegistration, r.Event, r.FirstName, r.LastName, confirmed, seasons, settings, now, false)
		if err != nil {
			return nil, s.internalError(ctx, "lecture des paiements impossible", err)
		}
		out.Body.Items = append(out.Body.Items, item)
	}
	// À venir (par date croissante) puis passées (la plus récente d'abord).
	slices.SortStableFunc(out.Body.Items, func(a, b MyRegistration) int {
		pa, pb := !now.Before(a.Event.EndsAt), !now.Before(b.Event.EndsAt)
		switch {
		case pa != pb && pa:
			return 1
		case pa != pb:
			return -1
		case pa:
			return b.Event.StartsAt.Compare(a.Event.StartsAt)
		default:
			return cmp.Or(a.Event.StartsAt.Compare(b.Event.StartsAt), cmp.Compare(a.ID, b.ID))
		}
	})
	return out, nil
}

// myRegistrationOutput lit une inscription d'une fiche gérée par le compte.
func (s *server) myRegistrationOutput(ctx context.Context, q *sqlc.Queries, id int64) (*MyRegistrationOutput, error) {
	r, err := q.GetRegistrationOfAccount(ctx, sqlc.GetRegistrationOfAccountParams{ID: id, AccountID: ownAccount(ctx)})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgRegistrationGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'inscription impossible", err)
	}
	now := s.now()
	seasons, _, err := seasonIndex(ctx, q, season.Today(now))
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	settings, err := s.paymentSettings(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paramètres impossible", err)
	}
	confirmed, err := q.CountConfirmedRegistrations(ctx, r.Event.ID)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des inscriptions impossible", err)
	}
	out, err := s.toMyRegistration(ctx, q, r.EventRegistration, r.Event, r.FirstName, r.LastName, confirmed, seasons, settings, now, true)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paiements impossible", err)
	}
	return &MyRegistrationOutput{Body: out}, nil
}

func (s *server) getMyRegistration(ctx context.Context, in *RegistrationIDInput) (*MyRegistrationOutput, error) {
	return s.myRegistrationOutput(ctx, s.Store.Read, in.ID)
}

// registrationLoader relit une inscription et son événement dans le périmètre
// de l'appelant (compte ou lien personnel) ; sql.ErrNoRows hors périmètre.
type registrationLoader func(q *sqlc.Queries) (sqlc.EventRegistration, sqlc.Event, error)

// registrationWrite relit dans la transaction une inscription du périmètre de
// l'appelant et applique fn.
func (s *server) registrationWrite(ctx context.Context, load registrationLoader, fn func(q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, st registrationState) error) (sqlc.EventRegistration, error) {
	var reg sqlc.EventRegistration
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		r, e, err := load(q)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgRegistrationGone)
		}
		if err != nil {
			return err
		}
		reg = r
		settings, err := s.paymentSettings(ctx, q)
		if err != nil {
			return err
		}
		st, err := s.myRegistrationState(ctx, q, r, e, settings, s.now())
		if err != nil {
			return err
		}
		return fn(q, r, e, st)
	})
	if err != nil {
		return reg, s.txError(ctx, "écriture de l'inscription impossible", err)
	}
	return reg, nil
}

// myRegistrationWrite relit dans la transaction une inscription d'une fiche du
// compte et applique fn.
func (s *server) myRegistrationWrite(ctx context.Context, id int64, fn func(q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, st registrationState) error) (sqlc.EventRegistration, error) {
	return s.registrationWrite(ctx, func(q *sqlc.Queries) (sqlc.EventRegistration, sqlc.Event, error) {
		r, err := q.GetRegistrationOfAccount(ctx, sqlc.GetRegistrationOfAccountParams{ID: id, AccountID: ownAccount(ctx)})
		return r.EventRegistration, r.Event, err
	}, fn)
}

// checkUnregister applique 012 FR-013 à l'annulation d'une inscription par la
// personne (compte ou lien personnel).
func (s *server) checkUnregister(r sqlc.EventRegistration, e sqlc.Event, st registrationState) error {
	if event.CanUnregister(eventState(e), st.snap, s.now()) {
		return nil
	}
	switch {
	case r.Status != "confirmed":
		return huma.Error409Conflict(MsgRegistrationChanged)
	case st.snap.PaymentStatus == membership.Paid || st.snap.CardPending:
		return huma.Error409Conflict(MsgRegistrationPaid)
	default:
		return huma.Error409Conflict(MsgRegistrationsClosed)
	}
}

// checkRegistrationAnnounce vérifie l'annonce d'un moyen de paiement manuel
// pour une inscription (012 FR-022).
func (s *server) checkRegistrationAnnounce(ctx context.Context, q *sqlc.Queries, e sqlc.Event, st registrationState, method string) error {
	m := membership.Method(method)
	if !membership.ManualMethod(m) {
		return FieldErrors("body.method", MsgCardIsOnline)
	}
	if !membership.CanPay(event.PaymentSnapshot(st.snap, e.Status == string(event.Cancelled))) {
		return huma.Error409Conflict(MsgNotPayableRegistration)
	}
	settings, err := s.paymentSettings(ctx, q)
	if err != nil {
		return err
	}
	if !settings.Enabled(m) {
		return FieldErrors("body.method", MsgMethodUnavailable)
	}
	return nil
}

func changedRegistration(n int64, err error) error {
	if err == nil && n == 0 {
		return huma.Error409Conflict(MsgRegistrationChanged)
	}
	return err
}

func (s *server) cancelMyRegistration(ctx context.Context, in *RegistrationIDInput) (*MyRegistrationOutput, error) {
	r, err := s.myRegistrationWrite(ctx, in.ID, func(q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, st registrationState) error {
		if err := s.checkUnregister(r, e, st); err != nil {
			return err
		}
		return changedRegistration(q.CancelRegistration(ctx, sqlc.CancelRegistrationParams{
			ActorID: nid(CurrentAccount(ctx).ID), Now: ns(s.timestamp()), ID: r.ID,
		}))
	})
	if err != nil {
		return nil, err
	}
	s.logRegistrationAction(ctx, "registration.cancel", r.EventID, r.ID, "space", "me")
	return s.myRegistrationOutput(ctx, s.Store.Read, in.ID)
}

func (s *server) announceMyRegistrationPayment(ctx context.Context, in *AnnounceRegistrationInput) (*MyRegistrationOutput, error) {
	r, err := s.myRegistrationWrite(ctx, in.ID, func(q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, st registrationState) error {
		if err := s.checkRegistrationAnnounce(ctx, q, e, st, in.Body.Method); err != nil {
			return err
		}
		return changedRegistration(q.AnnounceRegistrationPaymentOfAccount(ctx, sqlc.AnnounceRegistrationPaymentOfAccountParams{
			Method: ns(in.Body.Method), Now: s.timestamp(), ID: r.ID, AccountID: ownAccount(ctx),
		}))
	})
	if err != nil {
		return nil, err
	}
	s.logRegistrationAction(ctx, "registration.payment_announce", r.EventID, r.ID, "method", in.Body.Method)
	return s.myRegistrationOutput(ctx, s.Store.Read, in.ID)
}

// cardItemNameForEvent : intitulé transmis à HelloAsso, sans donnée
// personnelle du membre (FR-024).
func cardItemNameForEvent(title, reference string) string {
	return helloasso.TruncateItemName(fmt.Sprintf("Événement %s — %s", title, reference))
}

func (s *server) startMyRegistrationCardPayment(ctx context.Context, in *RegistrationIDInput) (*StartCardPaymentOutput, error) {
	q, account := s.Store.Read, ownAccount(ctx)
	r, err := q.GetRegistrationOfAccount(ctx, sqlc.GetRegistrationOfAccountParams{ID: in.ID, AccountID: account})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgRegistrationGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'inscription impossible", err)
	}
	return s.startRegistrationCard(ctx, q, r.EventRegistration, r.Event, cardOrigin{
		account: account, returnPath: "/mes-inscriptions/%d",
		payer:    func() (helloasso.Payer, error) { return s.accountPayer(ctx, q) },
		logAttrs: []any{"actor_id", account.Int64},
	})
}

// startRegistrationCard lance ou reprend le paiement par carte d'une
// inscription déjà relue dans le périmètre de l'appelant (compte ou lien
// personnel, 016 › R11).
func (s *server) startRegistrationCard(ctx context.Context, q *sqlc.Queries, reg sqlc.EventRegistration, e sqlc.Event, origin cardOrigin) (*StartCardPaymentOutput, error) {
	settings, err := s.paymentSettings(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paramètres impossible", err)
	}
	snap := event.PaymentSnapshot(regSnapshot(reg), e.Status == string(event.Cancelled))
	if !membership.CanPay(snap) {
		return nil, huma.Error409Conflict(MsgNotPayableRegistration)
	}
	if !membership.CanPayByCard(snap, settings.CardAvailable) {
		return nil, FieldErrors("body", MsgCardUnavailable)
	}

	env, amount, now := s.HelloAsso.Environment(), reg.AmountCents, s.now()
	resumable, err := q.FindResumableRegistrationOnlinePayment(ctx, sqlc.FindResumableRegistrationOnlinePaymentParams{
		RegistrationID: reg.ID, Environment: env, AmountCents: amount,
		Since: now.Add(-resumeWindow).UTC().Format(timeFormat),
	})
	if err == nil {
		return &StartCardPaymentOutput{Body: StartCardPayment{RedirectURL: resumable.RedirectUrl.String, OnlinePaymentID: resumable.ID, Resumed: true}}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
	}
	payer, err := origin.payer()
	if err != nil {
		return nil, err
	}
	op, err := s.Store.Write.CreateOnlinePayment(ctx, sqlc.CreateOnlinePaymentParams{
		RegistrationID: nid(reg.ID), AccountID: origin.account, ViaLink: boolInt(origin.viaLink),
		Environment: env, AmountCents: amount, Now: s.timestamp(),
	})
	if err != nil {
		return nil, s.internalError(ctx, "création du paiement en ligne impossible", err)
	}
	req := helloasso.CheckoutRequest{
		AmountCents: amount,
		ItemName:    cardItemNameForEvent(e.Title, reg.PaymentReference.String),
		Metadata:    helloasso.Metadata{Payment: op.ID, Registration: reg.ID, Reference: reg.PaymentReference.String},
	}
	base := fmt.Sprintf("%s"+origin.returnPath+"?tentative=%d&carte=", s.Config.BaseURL, reg.ID, op.ID)
	redirect, err := s.openCheckout(ctx, op.ID, base, req, payer)
	if err != nil {
		return nil, err
	}
	s.Logger.InfoContext(ctx, "paiement en ligne démarré", append([]any{
		"registration_id", reg.ID, "event_id", e.ID, "online_payment_id", op.ID,
		"environment", env, "amount_cents", amount,
	}, origin.logAttrs...)...)
	return &StartCardPaymentOutput{Body: StartCardPayment{RedirectURL: redirect, OnlinePaymentID: op.ID}}, nil
}

func (s *server) confirmMyRegistrationCardPayment(ctx context.Context, in *ConfirmRegistrationCardInput) (*ConfirmRegistrationCardOutput, error) {
	op, err := s.Store.Read.GetOnlinePaymentOfRegistrationAccount(ctx, sqlc.GetOnlinePaymentOfRegistrationAccountParams{
		ID: in.Body.OnlinePaymentID, RegistrationID: in.ID, AccountID: ownAccount(ctx),
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
	out, err := s.myRegistrationOutput(ctx, s.Store.Read, in.ID)
	if err != nil {
		return nil, err
	}
	return &ConfirmRegistrationCardOutput{Body: ConfirmRegistrationCard{Status: string(status), Registration: out.Body}}, nil
}

func (s *server) listMyDues(ctx context.Context, _ *struct{}) (*DueListOutput, error) {
	q, account := s.Store.Read, ownAccount(ctx)
	today := s.today()
	seasons, _, err := seasonIndex(ctx, q, today)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	out := &DueListOutput{}
	out.Body.Items = []Due{}
	memberships, err := q.ListMembershipsOfAccount(ctx, sqlc.ListMembershipsOfAccountParams{AccountID: account})
	if err != nil {
		return nil, s.internalError(ctx, "lecture des adhésions impossible", err)
	}
	for _, r := range memberships {
		m := r.Membership
		if !membership.CanPay(snapshotOf(m, seasons)) {
			continue
		}
		p := paymentOf(m)
		if p == nil {
			continue
		}
		out.Body.Items = append(out.Body.Items, Due{
			Kind: "membership", ID: m.ID, Member: memberRef(m.MemberID, r.FirstName, r.LastName, r.BirthDate, today),
			Label: "Adhésion " + seasons[m.SeasonID].ref.Label, AmountCents: m.AmountCents.Int64, Payment: *p,
			CreatedAt: parseTime(m.RequestedAt),
		})
	}
	regs, err := q.ListDueRegistrationsOfAccount(ctx, account)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des inscriptions impossible", err)
	}
	for _, r := range regs {
		p, err := registrationPayment(ctx, q, r.EventRegistration)
		if err != nil {
			return nil, s.internalError(ctx, "lecture des paiements impossible", err)
		}
		if p == nil {
			continue
		}
		out.Body.Items = append(out.Body.Items, Due{
			Kind: "registration", ID: r.EventRegistration.ID,
			Member: MemberRef{ID: r.EventRegistration.MemberID, FirstName: r.FirstName, LastName: r.LastName},
			Label:  r.EventTitle, AmountCents: r.EventRegistration.AmountCents, Payment: *p,
			CreatedAt: parseTime(r.EventRegistration.RegisteredAt),
		})
	}
	slices.SortStableFunc(out.Body.Items, func(a, b Due) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out, nil
}
