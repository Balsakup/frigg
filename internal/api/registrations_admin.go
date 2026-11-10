package api

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/csvfile"
	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/event"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
)

// Inscriptions côté administration (US5 ; contracts/api.md 012).

// AdminRegistrationActions sont les actions proposées à un administrateur.
type AdminRegistrationActions struct {
	Remove bool `json:"remove"`
	// AdjustAmount : ajustement du montant à payer (014).
	AdjustAmount      bool `json:"adjustAmount"`
	RecordPayment     bool `json:"recordPayment"`
	CancelPayment     bool `json:"cancelPayment"`
	VerifyCardPayment bool `json:"verifyCardPayment"`
}

// AdminRegistration est une inscription vue par l'administration.
type AdminRegistration struct {
	ID             int64       `json:"id"`
	Event          EventRef    `json:"event"`
	Member         MemberRef   `json:"member"`
	IsSeasonMember bool        `json:"isSeasonMember" doc:"Adhérent de la saison de l'événement"`
	Eligible       bool        `json:"eligible" doc:"Éligible selon l'ouverture actuelle de l'événement"`
	ContactEmail   *string     `json:"contactEmail" doc:"Adresse de contact ; null : non joignable"`
	State          string      `json:"state" enum:"confirmed,cancelled"`
	RegisteredAt   time.Time   `json:"registeredAt"`
	RegisteredBy   *AccountRef `json:"registeredBy"`
	CancelledAt    *time.Time  `json:"cancelledAt"`
	CancelledBy    *AccountRef `json:"cancelledBy"`
	AmountCents    int64       `json:"amountCents" doc:"Montant à payer"`
	// Remises (014), réservées aux administrateurs (research.md 014 › R9).
	BaseAmountCents    int64                    `json:"baseAmountCents" doc:"Montant théorique : prix de l'événement figé à l'inscription"`
	AdjustmentCents    int64                    `json:"adjustmentCents" doc:"Montant à payer moins montant théorique : négatif = remise, positif = majoration, 0 sans ajustement"`
	AdjustmentReason   *string                  `json:"adjustmentReason" doc:"Motif de l'ajustement"`
	Payment            *Payment                 `json:"payment" doc:"null : gratuite"`
	PaymentNote        *string                  `json:"paymentNote"`
	OpenAlerts         int64                    `json:"openAlerts" doc:"Signalements de paiement non traités"`
	OnlinePayments     []OnlinePayment          `json:"onlinePayments" doc:"Détail seulement ; la plus récente d'abord"`
	Alerts             []PaymentAlert           `json:"alerts" doc:"Détail seulement ; non traités d'abord"`
	CardPaymentPending bool                     `json:"cardPaymentPending"`
	Actions            AdminRegistrationActions `json:"actions"`
}

type AdminRegistrationOutput struct{ Body AdminRegistration }

type AdminRegistrationListOutput struct {
	Body struct {
		Items []AdminRegistration `json:"items" doc:"Par date d'inscription"`
	}
}

// AdminRegistrationRow est une ligne de la rubrique « Inscriptions ».
type AdminRegistrationRow struct {
	ID           int64     `json:"id"`
	Event        EventRef  `json:"event"`
	Member       MemberRef `json:"member"`
	State        string    `json:"state" enum:"confirmed,cancelled"`
	RegisteredAt time.Time `json:"registeredAt"`
	AmountCents  int64     `json:"amountCents" doc:"Montant à payer"`
	// BaseAmountCents et AdjustmentCents : montant théorique et ajustement (014).
	BaseAmountCents int64    `json:"baseAmountCents"`
	AdjustmentCents int64    `json:"adjustmentCents"`
	Payment         *Payment `json:"payment"`
	OpenAlerts      int64    `json:"openAlerts"`
}

// AdminRegistrationPage est une page de la rubrique « Inscriptions ».
type AdminRegistrationPage struct {
	Items    []AdminRegistrationRow `json:"items" doc:"La plus récente d'abord"`
	Total    int64                  `json:"total"`
	Page     int64                  `json:"page"`
	PageSize int64                  `json:"pageSize"`
}

type AdminRegistrationPageOutput struct{ Body AdminRegistrationPage }

type ListEventRegistrationsInput struct {
	ID    int64  `path:"id" doc:"Identifiant de l'événement"`
	State string `query:"state" enum:"confirmed,all" default:"confirmed"`
}

type ListRegistrationsInput struct {
	Event    int64  `query:"event" minimum:"0" doc:"Inscriptions d'un événement"`
	State    string `query:"state" enum:"confirmed,cancelled"`
	Payment  string `query:"payment" enum:"due,announced,paid,free"`
	Alerts   string `query:"alerts" enum:"open" doc:"open : inscriptions portant un signalement de paiement à traiter"`
	Q        string `query:"q" maxLength:"100" doc:"Fragments d'au moins 3 caractères cherchés dans le nom, le prénom ou la référence"`
	Page     int64  `query:"page" minimum:"1" default:"1"`
	PageSize int64  `query:"pageSize" minimum:"1" maximum:"100" default:"25"`
}

type RegisterMemberInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'événement"`
	Body struct {
		MemberID int64              `json:"memberId" minimum:"1"`
		Payment  *RecordPaymentBody `json:"payment,omitempty" required:"false" doc:"Règlement déjà reçu ; ignoré si l'inscription est gratuite"`
		// Montant à payer ajusté (014).
		Amount           string `json:"amount,omitempty" required:"false" maxLength:"12" doc:"Montant à payer en euros ; vide : prix de l'événement"`
		AdjustmentReason string `json:"adjustmentReason,omitempty" required:"false" maxLength:"200" doc:"Motif de l'ajustement, obligatoire si le montant diffère du prix"`
	}
}

type AdjustRegistrationAmountInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'inscription"`
	Body struct {
		Amount           string `json:"amount,omitempty" required:"false" maxLength:"12" doc:"Montant à payer en euros ; vide : montant théorique (prix figé à l'inscription)"`
		AdjustmentReason string `json:"adjustmentReason,omitempty" required:"false" maxLength:"200" doc:"Motif de l'ajustement, obligatoire si le montant diffère du montant théorique"`
	}
}

type RecordRegistrationPaymentInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'inscription"`
	Body RecordPaymentBody
}

func (s *server) registerAdminRegistrations(api huma.API) {
	op := func(id, method, path, summary string, status int) huma.Operation {
		return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{tagRegistrations}, DefaultStatus: status}
	}
	register(api, auth.Admin, op("list-event-registrations", http.MethodGet, "/admin/events/{id}/registrations", "Inscrits d'un événement", 0), s.listEventRegistrations)
	csvOp := op("export-event-registrations", http.MethodGet, "/admin/events/{id}/registrations.csv", "Exporter les inscrits (CSV)", 0)
	csvOp.Responses = map[string]*huma.Response{"200": {
		Description: "Inscrits au format CSV (UTF-8, séparateur point-virgule)",
		Content:     map[string]*huma.MediaType{"text/csv": {Schema: &huma.Schema{Type: "string", Format: "binary"}}},
	}}
	register(api, auth.Admin, csvOp, s.exportEventRegistrations)
	register(api, auth.Admin, op("register-member", http.MethodPost, "/admin/events/{id}/registrations", "Inscrire un membre", http.StatusCreated), s.registerMember)
	register(api, auth.Admin, op("list-registrations", http.MethodGet, "/admin/registrations", "Lister les inscriptions", 0), s.listRegistrations)
	register(api, auth.Admin, op("get-registration", http.MethodGet, "/admin/registrations/{id}", "Une inscription", 0), s.getRegistration)
	register(api, auth.Admin, op("remove-registration", http.MethodPost, "/admin/registrations/{id}/cancel", "Retirer une inscription", 0), s.removeRegistration)
	register(api, auth.Admin, op("adjust-registration-amount", http.MethodPut, "/admin/registrations/{id}/amount", "Ajuster le montant à payer d'une inscription", 0), s.adjustRegistrationAmount)
	register(api, auth.Admin, op("record-registration-payment", http.MethodPost, "/admin/registrations/{id}/payment", "Constater le paiement d'une inscription", 0), s.recordRegistrationPayment)
	register(api, auth.Admin, op("cancel-registration-payment", http.MethodDelete, "/admin/registrations/{id}/payment", "Annuler la constatation du paiement", 0), s.cancelRegistrationPayment)
	register(api, auth.Admin, op("verify-registration-card-payment", http.MethodPost, "/admin/registrations/{id}/card-payment/verify", "Vérifier auprès de HelloAsso les tentatives en cours", 0), s.verifyRegistrationCardPayment)
	register(api, auth.Admin, op("resolve-registration-payment-alert", http.MethodPost, "/admin/registration-alerts/{id}/resolve", "Marquer un signalement d'inscription comme traité", 0), s.resolveRegistrationAlert)
}

// contactEmail : adresse du compte gestionnaire actif, sinon de la fiche.
func contactEmail(account, disabled, member sql.NullString) *string {
	if account.Valid && !disabled.Valid && account.String != "" {
		return &account.String
	}
	if member.Valid && member.String != "" {
		return &member.String
	}
	return nil
}

type adminRegSource struct {
	r                       sqlc.EventRegistration
	e                       sqlc.Event
	first, last, birth      string
	account, disabled, mail sql.NullString
	registeredBy            sql.NullString
	cancelledBy             sql.NullString
	isSeasonMember          bool
}

func (s *server) toAdminRegistration(src adminRegSource, openAlerts int64, today season.Date) AdminRegistration {
	r, now := src.r, s.now()
	out := AdminRegistration{
		ID: r.ID, Event: eventRef(src.e), Member: memberRef(r.MemberID, src.first, src.last, src.birth, today),
		IsSeasonMember: src.isSeasonMember,
		Eligible:       event.Eligibility(event.Openness(src.e.Openness), src.isSeasonMember, false) == event.Eligible,
		ContactEmail:   contactEmail(src.account, src.disabled, src.mail),
		State:          r.Status, RegisteredAt: parseTime(r.RegisteredAt), RegisteredBy: accountRef(r.RegisteredBy, src.registeredBy),
		CancelledAt: optionalTimeValue(r.CancelledAt), CancelledBy: accountRef(r.CancelledBy, src.cancelledBy),
		AmountCents: r.AmountCents, PaymentNote: optionalValue(r.PaymentNote), OpenAlerts: openAlerts,
		BaseAmountCents:  membership.Theoretical(r.AmountCents, basePtr(r.BaseAmountCents)),
		AdjustmentCents:  membership.AdjustmentOf(r.AmountCents, basePtr(r.BaseAmountCents)),
		AdjustmentReason: optionalValue(r.AdjustmentReason),
		OnlinePayments:   []OnlinePayment{}, Alerts: []PaymentAlert{},
	}
	snap := event.PaymentSnapshot(regSnapshot(r), src.e.Status == string(event.Cancelled))
	out.Actions = AdminRegistrationActions{
		Remove:        event.CanRemove(eventState(src.e), regSnapshot(r), now),
		AdjustAmount:  canAdjustRegistration(r, src.e),
		RecordPayment: membership.CanRecordPayment(snap),
		CancelPayment: membership.CanCancelPayment(snap),
	}
	return out
}

func (s *server) openAlertsByRegistration(ctx context.Context, q *sqlc.Queries) (map[int64]int64, error) {
	rows, err := q.ListOpenAlertCountsByRegistration(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]int64, len(rows))
	for _, r := range rows {
		out[r.RegistrationID] = r.OpenAlerts
	}
	return out, nil
}

func (s *server) listEventRegistrations(ctx context.Context, in *ListEventRegistrationsInput) (*AdminRegistrationListOutput, error) {
	q := s.Store.Read
	ev, err := q.GetEvent(ctx, in.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgEventGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'événement impossible", err)
	}
	status := ns("confirmed")
	if in.State == "all" {
		status = sql.NullString{}
	}
	rows, err := q.ListRegistrationsOfEvent(ctx, sqlc.ListRegistrationsOfEventParams{SeasonID: ev.Event.SeasonID, EventID: ev.Event.ID, Status: status})
	if err != nil {
		return nil, s.internalError(ctx, "lecture des inscriptions impossible", err)
	}
	alerts, err := s.openAlertsByRegistration(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des signalements impossible", err)
	}
	today := s.today()
	out := &AdminRegistrationListOutput{}
	out.Body.Items = make([]AdminRegistration, 0, len(rows))
	for _, r := range rows {
		item := s.toAdminRegistration(adminRegSource{
			r: r.EventRegistration, e: ev.Event, first: r.FirstName, last: r.LastName, birth: r.BirthDate,
			account: r.AccountEmail, disabled: r.AccountDisabledAt, mail: r.MemberEmail,
			registeredBy: r.RegisteredByEmail, isSeasonMember: r.IsSeasonMember,
		}, alerts[r.EventRegistration.ID], today)
		if item.Payment, err = registrationPayment(ctx, q, r.EventRegistration); err != nil {
			return nil, s.internalError(ctx, "lecture des paiements impossible", err)
		}
		out.Body.Items = append(out.Body.Items, item)
	}
	return out, nil
}

// adminRegistration lit une inscription avec ses tentatives et signalements.
func (s *server) adminRegistration(ctx context.Context, q *sqlc.Queries, id int64) (*AdminRegistrationOutput, error) {
	r, err := q.GetRegistration(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgRegistrationGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'inscription impossible", err)
	}
	alerts, err := q.ListPaymentAlertsOfRegistration(ctx, id)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des signalements impossible", err)
	}
	open := int64(0)
	for _, a := range alerts {
		if !a.ResolvedAt.Valid {
			open++
		}
	}
	out := s.toAdminRegistration(adminRegSource{
		r: r.EventRegistration, e: r.Event, first: r.FirstName, last: r.LastName, birth: r.BirthDate,
		account: r.AccountEmail, disabled: r.AccountDisabledAt, mail: r.MemberEmail,
		registeredBy: r.RegisteredByEmail, cancelledBy: r.CancelledByEmail, isSeasonMember: r.IsSeasonMember,
	}, open, s.today())
	if out.Payment, err = registrationPayment(ctx, q, r.EventRegistration); err != nil {
		return nil, s.internalError(ctx, "lecture des paiements impossible", err)
	}
	ops, err := q.ListOnlinePaymentsOfRegistration(ctx, id)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
	}
	out.OnlinePayments, out.CardPaymentPending, out.Actions.VerifyCardPayment = s.onlineDetails(ops)
	for _, a := range alerts {
		out.Alerts = append(out.Alerts, paymentAlert(a.ID, a.Kind, a.OnlinePaymentID, a.CreatedAt, a.ResolvedAt, a.ResolvedBy, a.ResolvedByEmail))
	}
	return &AdminRegistrationOutput{Body: out}, nil
}

func (s *server) getRegistration(ctx context.Context, in *RegistrationIDInput) (*AdminRegistrationOutput, error) {
	return s.adminRegistration(ctx, s.Store.Read, in.ID)
}

func (s *server) listRegistrations(ctx context.Context, in *ListRegistrationsInput) (*AdminRegistrationPageOutput, error) {
	q := s.Store.Read
	params := sqlc.ListRegistrationsParams{Status: ns(in.State), Payment: ns(in.Payment)}
	if in.Event > 0 {
		params.EventID = nid(in.Event)
	}
	if in.Alerts == "open" {
		params.OpenAlerts = nid(1)
	}
	if strings.TrimSpace(in.Q) != "" {
		fts, ok := database.FTSTrigramQuery(in.Q)
		if !ok {
			return nil, FieldErrors("query.q", MsgSearchTooShort)
		}
		params.Query = ns(fts)
	}
	rows, err := q.ListRegistrations(ctx, params)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des inscriptions impossible", err)
	}
	alerts, err := s.openAlertsByRegistration(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des signalements impossible", err)
	}
	total := int64(len(rows))
	start := min((in.Page-1)*in.PageSize, total)
	end := min(start+in.PageSize, total)
	out := AdminRegistrationPage{Items: make([]AdminRegistrationRow, 0, end-start), Total: total, Page: in.Page, PageSize: in.PageSize}
	for _, r := range rows[start:end] {
		reg := r.EventRegistration
		p, err := registrationPayment(ctx, q, reg)
		if err != nil {
			return nil, s.internalError(ctx, "lecture des paiements impossible", err)
		}
		out.Items = append(out.Items, AdminRegistrationRow{
			ID:     reg.ID,
			Event:  EventRef{ID: reg.EventID, Title: r.EventTitle, StartsAt: parseTime(r.EventStartsAt), Status: r.EventStatus},
			Member: MemberRef{ID: reg.MemberID, FirstName: r.FirstName, LastName: r.LastName},
			State:  reg.Status, RegisteredAt: parseTime(reg.RegisteredAt), AmountCents: reg.AmountCents,
			BaseAmountCents: membership.Theoretical(reg.AmountCents, basePtr(reg.BaseAmountCents)),
			AdjustmentCents: membership.AdjustmentOf(reg.AmountCents, basePtr(reg.BaseAmountCents)),
			Payment:         p, OpenAlerts: alerts[reg.ID],
		})
	}
	return &AdminRegistrationPageOutput{Body: out}, nil
}

func (s *server) registerMember(ctx context.Context, in *RegisterMemberInput) (*AdminRegistrationOutput, error) {
	var created sqlc.EventRegistration
	var adj membership.Adjustment
	var received int64
	paid := false
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		r, err := q.GetEvent(ctx, in.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgEventGone)
		}
		if err != nil {
			return err
		}
		if !event.CanAdminRegister(eventState(r.Event), s.now()) {
			return huma.Error409Conflict(MsgRegistrationsClosed)
		}
		m, err := q.GetMember(ctx, in.Body.MemberID)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgMemberGone)
		}
		if err != nil {
			return err
		}
		regs, err := s.createRegistrations(ctx, q, r.Event, []sqlc.Member{memberOfRow(m)}, "body.memberId")
		if err != nil {
			return err
		}
		created = regs[0]
		// Montant à payer fixé à l'inscription (014) : l'inscription relue
		// porte l'état de paiement qui en découle.
		if strings.TrimSpace(in.Body.Amount) != "" {
			if created, adj, err = s.applyRegistrationAmount(ctx, q, created, r.Event, in.Body.Amount, in.Body.AdjustmentReason, "body"); err != nil {
				return err
			}
			regs[0] = created
		}
		if in.Body.Payment != nil && created.PaymentStatus.Valid {
			paidOn, note, err := s.checkPayment(*in.Body.Payment, "body.payment", season.Date{}, "")
			if err != nil {
				return err
			}
			if received, err = receivedAmount(*in.Body.Payment, "body.payment", created.AmountCents); err != nil {
				return err
			}
			if err := changedRegistration(q.RecordRegistrationPayment(ctx, sqlc.RecordRegistrationPaymentParams{
				Method: ns(in.Body.Payment.Method), PaidOn: ns(paidOn), ActorID: nid(CurrentAccount(ctx).ID),
				Now: ns(s.timestamp()), Note: ns(note), ID: created.ID, PaidAmountCents: nid(received),
			})); err != nil {
				return err
			}
			paid = true
		}
		return s.notifyCreated(ctx, q, r.Event, regs)
	})
	if err != nil {
		return nil, s.txError(ctx, "inscription impossible", err)
	}
	s.logRegistrationAction(ctx, "registration.create", created.EventID, created.ID,
		append([]any{"space", "admin", "payment_reference", created.PaymentReference.String, "paid", paid, "received_cents", received}, adjustmentLog(adj)...)...)
	return s.adminRegistration(ctx, s.Store.Read, created.ID)
}

// adminRegistrationWrite relit l'inscription dans la transaction et applique fn.
func (s *server) adminRegistrationWrite(ctx context.Context, id int64, fn func(q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, cardPending bool) error) (sqlc.EventRegistration, error) {
	var reg sqlc.EventRegistration
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		row, err := q.GetRegistration(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgRegistrationGone)
		}
		if err != nil {
			return err
		}
		reg = row.EventRegistration
		ops, err := q.ListOnlinePaymentsOfRegistration(ctx, id)
		if err != nil {
			return err
		}
		_, pending, _ := s.onlineDetails(ops)
		return fn(q, row.EventRegistration, row.Event, pending)
	})
	if err != nil {
		return reg, s.txError(ctx, "écriture de l'inscription impossible", err)
	}
	return reg, nil
}

func (s *server) removeRegistration(ctx context.Context, in *RegistrationIDInput) (*AdminRegistrationOutput, error) {
	r, err := s.adminRegistrationWrite(ctx, in.ID, func(q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, _ bool) error {
		if !event.CanRemove(eventState(e), regSnapshot(r), s.now()) {
			return huma.Error409Conflict(MsgRegistrationChanged)
		}
		if err := changedRegistration(q.CancelRegistration(ctx, sqlc.CancelRegistrationParams{
			ActorID: nid(CurrentAccount(ctx).ID), Now: ns(s.timestamp()), ID: r.ID,
		})); err != nil {
			return err
		}
		// Payée par carte : le remboursement se fait dans HelloAsso (008 › R10).
		if r.PaidMethod.String == string(membership.Card) && r.PaidOnlinePaymentID.Valid {
			if err := q.CreatePaymentAlert(ctx, sqlc.CreatePaymentAlertParams{
				RegistrationID: nid(r.ID), OnlinePaymentID: r.PaidOnlinePaymentID.Int64,
				Kind: string(membership.AlertRefundDue), Now: s.timestamp(),
			}); err != nil {
				return err
			}
		}
		return s.notifyRegistrations(ctx, q, "registration-removed", e, []int64{r.ID}, mail.Data{})
	})
	if err != nil {
		return nil, err
	}
	s.logRegistrationAction(ctx, "registration.remove", r.EventID, r.ID, "space", "admin")
	return s.adminRegistration(ctx, s.Store.Read, in.ID)
}

func (s *server) recordRegistrationPayment(ctx context.Context, in *RecordRegistrationPaymentInput) (*AdminRegistrationOutput, error) {
	var received, due int64
	r, err := s.adminRegistrationWrite(ctx, in.ID, func(q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, _ bool) error {
		if !membership.CanRecordPayment(event.PaymentSnapshot(regSnapshot(r), e.Status == string(event.Cancelled))) {
			return huma.Error409Conflict(MsgNotPayableRegistration)
		}
		paidOn, note, err := s.checkPayment(in.Body, "body", season.Date{}, "")
		if err != nil {
			return err
		}
		if received, err = receivedAmount(in.Body, "body", r.AmountCents); err != nil {
			return err
		}
		due = r.AmountCents
		if err := changedRegistration(q.RecordRegistrationPayment(ctx, sqlc.RecordRegistrationPaymentParams{
			Method: ns(in.Body.Method), PaidOn: ns(paidOn), ActorID: nid(CurrentAccount(ctx).ID),
			Now: ns(s.timestamp()), Note: ns(note), ID: r.ID, PaidAmountCents: nid(received),
		})); err != nil {
			return err
		}
		return s.notifyRegistrationPaid(ctx, q, e, r.ID)
	})
	if err != nil {
		return nil, err
	}
	s.logRegistrationAction(ctx, "registration.payment_record", r.EventID, r.ID, "method", in.Body.Method,
		"received_cents", received, "donation_cents", membership.Donation(membership.Method(in.Body.Method), due, received))
	return s.adminRegistration(ctx, s.Store.Read, in.ID)
}

func (s *server) cancelRegistrationPayment(ctx context.Context, in *RegistrationIDInput) (*AdminRegistrationOutput, error) {
	r, err := s.adminRegistrationWrite(ctx, in.ID, func(q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, _ bool) error {
		if !membership.CanCancelPayment(event.PaymentSnapshot(regSnapshot(r), false)) {
			return huma.Error409Conflict(MsgRegistrationChanged)
		}
		return changedRegistration(q.CancelRegistrationPayment(ctx, sqlc.CancelRegistrationPaymentParams{Now: s.timestamp(), ID: r.ID}))
	})
	if err != nil {
		return nil, err
	}
	s.logRegistrationAction(ctx, "registration.payment_cancel", r.EventID, r.ID)
	return s.adminRegistration(ctx, s.Store.Read, in.ID)
}

func (s *server) verifyRegistrationCardPayment(ctx context.Context, in *RegistrationIDInput) (*AdminRegistrationOutput, error) {
	if _, err := s.Store.Read.GetRegistration(ctx, in.ID); errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgRegistrationGone)
	} else if err != nil {
		return nil, s.internalError(ctx, "lecture de l'inscription impossible", err)
	}
	if s.HelloAsso == nil {
		return nil, huma.Error503ServiceUnavailable(MsgCardUnavailableNow)
	}
	ops, err := s.Store.Read.ListVerifiableOnlinePaymentsOfRegistration(ctx, sqlc.ListVerifiableOnlinePaymentsOfRegistrationParams{
		RegistrationID: in.ID, Since: ns(s.now().Add(-verifiableWindow).UTC().Format(timeFormat)),
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
	return s.adminRegistration(ctx, s.Store.Read, in.ID)
}

func (s *server) resolveRegistrationAlert(ctx context.Context, in *PaymentAlertIDInput) (*AdminRegistrationOutput, error) {
	var alert sqlc.PaymentAlert
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		a, err := q.GetPaymentAlert(ctx, in.ID)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && !a.RegistrationID.Valid) {
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
		"registration_id", alert.RegistrationID.Int64, "actor_id", CurrentAccount(ctx).ID)
	return s.adminRegistration(ctx, s.Store.Read, alert.RegistrationID.Int64)
}

// Export CSV des inscrits (R13).

var paymentLabels = map[string]string{"due": "À payer", "announced": "Paiement annoncé", "paid": "Payée"}

func (s *server) exportEventRegistrations(ctx context.Context, in *EventIDInput) (*huma.StreamResponse, error) {
	q := s.Store.Read
	ev, err := q.GetEvent(ctx, in.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgEventGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'événement impossible", err)
	}
	rows, err := q.ListRegistrationsOfEvent(ctx, sqlc.ListRegistrationsOfEventParams{SeasonID: ev.Event.SeasonID, EventID: ev.Event.ID, Status: ns("confirmed")})
	if err != nil {
		return nil, s.internalError(ctx, "lecture des inscriptions impossible", err)
	}
	var buf bytes.Buffer
	w := csvfile.NewWriter(&buf)
	_ = w.Write([]string{"Nom", "Prénom", "Date de naissance", "Téléphone", "Adresse de contact", "Adhérent",
		"Date d'inscription", "Montant théorique", "Montant à payer", "Motif d'ajustement", "Paiement", "Moyen",
		"Montant reçu", "Don", "Référence"})
	for _, r := range rows {
		reg := r.EventRegistration
		contact := ""
		if c := contactEmail(r.AccountEmail, r.AccountDisabledAt, r.MemberEmail); c != nil {
			contact = *c
		}
		adherent := "non"
		if r.IsSeasonMember {
			adherent = "oui"
		}
		amount, payment, method := "", "Gratuite", ""
		theoretical, receivedCell, donation := "", "", ""
		if reg.PaymentStatus.Valid || reg.BaseAmountCents.Valid {
			theoretical = formatEuros(membership.Theoretical(reg.AmountCents, basePtr(reg.BaseAmountCents)))
			amount = formatEuros(reg.AmountCents)
		}
		if reg.PaymentStatus.Valid {
			payment = paymentLabels[reg.PaymentStatus.String]
			method = methodLabels[reg.PaidMethod.String]
			if method == "" {
				method = methodLabels[reg.AnnouncedMethod.String]
			}
		}
		if reg.PaidAmountCents.Valid {
			receivedCell = formatEuros(reg.PaidAmountCents.Int64)
			donation = formatEuros(membership.Donation(membership.Method(reg.PaidMethod.String), reg.AmountCents, reg.PaidAmountCents.Int64))
		}
		registered := parseTime(reg.RegisteredAt).In(season.Paris).Format("02/01/2006 15:04")
		birth := r.BirthDate
		if d, err := season.ParseDate(r.BirthDate); err == nil {
			birth = fmt.Sprintf("%02d/%02d/%d", d.Day(), d.Month(), d.Year())
		}
		_ = w.Write([]string{csvfile.SafeCell(r.LastName), csvfile.SafeCell(r.FirstName), birth, csvfile.SafeCell(r.Phone.String), csvfile.SafeCell(contact), adherent,
			registered, theoretical, amount, csvfile.SafeCell(reg.AdjustmentReason.String), payment, method,
			receivedCell, donation, reg.PaymentReference.String})
	}
	w.Flush()
	data := buf.Bytes()
	s.logEventAction(ctx, "event.export", ev.Event.ID, "rows", len(rows))
	return &huma.StreamResponse{Body: func(hctx huma.Context) {
		hctx.SetHeader("Content-Type", "text/csv; charset=utf-8")
		hctx.SetHeader("Content-Length", strconv.Itoa(len(data)))
		hctx.SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="inscrits-%d.csv"`, ev.Event.ID))
		hctx.SetHeader("X-Content-Type-Options", "nosniff")
		hctx.SetHeader("Cache-Control", "private, no-store")
		hctx.SetStatus(http.StatusOK)
		_, _ = hctx.BodyWriter().Write(data)
	}}, nil
}

// memberOfRow réduit une fiche lue par l'administration à ce qu'utilise
// createRegistrations.
func memberOfRow(m sqlc.GetMemberRow) sqlc.Member {
	return sqlc.Member{ID: m.ID, FirstName: m.FirstName, LastName: m.LastName, BirthDate: m.BirthDate}
}

// paidUpcoming compte les inscriptions payées d'une fiche à des événements à
// venir, pour avertir avant sa suppression (012 › R14).
func (s *server) paidUpcoming(ctx context.Context, memberID int64) (*int64, error) {
	n, err := s.Store.Read.CountPaidUpcomingRegistrationsOfMember(ctx, sqlc.CountPaidUpcomingRegistrationsOfMemberParams{
		MemberID: memberID, Now: s.now().UTC().Format(timeFormat),
	})
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// canAdjustRegistration : ajustement du montant d'une inscription confirmée,
// non payée, d'un événement non annulé (research.md 014 › R6).
func canAdjustRegistration(r sqlc.EventRegistration, e sqlc.Event) bool {
	return membership.CanUpdate(event.PaymentSnapshot(regSnapshot(r), e.Status == string(event.Cancelled)))
}

// applyRegistrationAmount fixe le montant à payer d'une inscription dans la
// transaction courante : montant théorique = prix figé à l'inscription,
// paiement recalculé comme pour une adhésion, référence EVT- attribuée si le
// paiement apparaît (research.md 014 › R6).
func (s *server) applyRegistrationAmount(ctx context.Context, q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, amount, reason, prefix string) (sqlc.EventRegistration, membership.Adjustment, error) {
	adj, err := adjust(membership.Theoretical(r.AmountCents, basePtr(r.BaseAmountCents)), amount, reason, prefix)
	if err != nil {
		return r, adj, err
	}
	ps, method := membership.PaymentAfter(adj.AmountCents, membership.PaymentStatus(r.PaymentStatus.String), membership.Method(r.AnnouncedMethod.String))
	ref := r.PaymentReference
	var status, announced sql.NullString
	if ps != membership.NoPayment {
		status = ns(string(ps))
		if !ref.Valid {
			seasons, _, err := seasonIndex(ctx, q, s.today())
			if err != nil {
				return r, adj, err
			}
			info := seasons[e.SeasonID]
			p := event.ReferencePrefix(info.startsOn, info.endsOn)
			refs, err := q.ListRegistrationReferences(ctx, p)
			if err != nil {
				return r, adj, err
			}
			ref = ns(membership.NextReference(p, refs))
		}
	}
	if method != "" {
		announced = ns(string(method))
	}
	base, why := adjustmentColumns(adj)
	if err := changedRegistration(q.AdjustRegistrationAmount(ctx, sqlc.AdjustRegistrationAmountParams{
		AmountCents: adj.AmountCents, BaseAmountCents: base, AdjustmentReason: why,
		PaymentReference: ref, PaymentStatus: status, AnnouncedMethod: announced,
		Now: s.timestamp(), ID: r.ID,
	})); err != nil {
		return r, adj, err
	}
	r.AmountCents, r.BaseAmountCents, r.AdjustmentReason = adj.AmountCents, base, why
	r.PaymentReference, r.PaymentStatus, r.AnnouncedMethod = ref, status, announced
	return r, adj, nil
}

func (s *server) adjustRegistrationAmount(ctx context.Context, in *AdjustRegistrationAmountInput) (*AdminRegistrationOutput, error) {
	var adj membership.Adjustment
	var old int64
	r, err := s.adminRegistrationWrite(ctx, in.ID, func(q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, _ bool) error {
		if !canAdjustRegistration(r, e) {
			return huma.Error409Conflict(MsgRegistrationChanged)
		}
		old = r.AmountCents
		var err error
		_, adj, err = s.applyRegistrationAmount(ctx, q, r, e, in.Body.Amount, in.Body.AdjustmentReason, "body")
		return err
	})
	if err != nil {
		return nil, err
	}
	s.logRegistrationAction(ctx, "registration.adjust", r.EventID, r.ID, append([]any{"old_amount_cents", old}, adjustmentLog(adj)...)...)
	return s.adminRegistration(ctx, s.Store.Read, in.ID)
}
