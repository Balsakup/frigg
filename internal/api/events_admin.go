package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/config"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/event"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
)

// Événements côté administration (US1, US6 ; contracts/api.md 012).

// EventBody est un événement saisi. Les dates-heures sont locales (Paris),
// sans fuseau (research.md 012 › R8).
type EventBody struct {
	Title                string  `json:"title" maxLength:"120"`
	Location             string  `json:"location" maxLength:"200"`
	Description          *string `json:"description,omitempty" required:"false" nullable:"true" maxLength:"5000"`
	StartsAt             string  `json:"startsAt" doc:"Date et heure locales, par exemple « 2026-12-12T14:00 »" maxLength:"16"`
	EndsAt               string  `json:"endsAt" doc:"Date et heure locales" maxLength:"16"`
	Openness             string  `json:"openness" enum:"season_members,all_members"`
	Capacity             *int    `json:"capacity,omitempty" required:"false" nullable:"true" doc:"Absente ou null : illimitée"`
	Price                string  `json:"price,omitempty" required:"false" maxLength:"12" doc:"Prix en euros, par exemple « 15,50 » ; vide : gratuit"`
	RegistrationOpensAt  *string `json:"registrationOpensAt,omitempty" required:"false" nullable:"true" maxLength:"16" doc:"Absente : dès la publication"`
	RegistrationClosesAt *string `json:"registrationClosesAt,omitempty" required:"false" nullable:"true" maxLength:"16" doc:"Absente : au début de l'événement"`
}

// EventLocal reprend les dates-heures locales pour le formulaire.
type EventLocal struct {
	StartsAt             string  `json:"startsAt"`
	EndsAt               string  `json:"endsAt"`
	RegistrationOpensAt  *string `json:"registrationOpensAt"`
	RegistrationClosesAt *string `json:"registrationClosesAt"`
}

// EventTotals sont les chiffres des inscriptions confirmées (FR-026, 014).
type EventTotals struct {
	DueCents           int64 `json:"dueCents" doc:"Montant total à payer des inscriptions"`
	CollectedCents     int64 `json:"collectedCents" doc:"Montant encaissé, dons compris, hors paiements en mode test"`
	CollectedTestCents int64 `json:"collectedTestCents" doc:"Montant encaissé en mode test HelloAsso"`
	PaidCount          int64 `json:"paidCount" doc:"Inscriptions payées"`
	TheoreticalCents   int64 `json:"theoreticalCents" doc:"Montant théorique total (prix figés à l'inscription)"`
	DiscountCents      int64 `json:"discountCents" doc:"Total des remises"`
	SurchargeCents     int64 `json:"surchargeCents" doc:"Total des majorations"`
	DonationCents      int64 `json:"donationCents" doc:"Dons compris dans l'encaissé"`
	RemainingCents     int64 `json:"remainingCents" doc:"Reste à percevoir"`
}

// AdminEventActions sont les actions proposées à un administrateur.
type AdminEventActions struct {
	Update      bool `json:"update"`
	UpdateDates bool `json:"updateDates"`
	Publish     bool `json:"publish"`
	Cancel      bool `json:"cancel"`
	Restore     bool `json:"restore"`
	Delete      bool `json:"delete"`
	Register    bool `json:"register"`
}

// AdminEvent est un événement vu par l'administration.
type AdminEvent struct {
	EventSummary
	Description          *string           `json:"description"`
	RegistrationOpensAt  *time.Time        `json:"registrationOpensAt" doc:"Ouverture saisie ; null : dès la publication"`
	RegistrationClosesAt *time.Time        `json:"registrationClosesAt" doc:"Clôture saisie ; null : au début"`
	Local                EventLocal        `json:"local" doc:"Valeurs locales pour le formulaire"`
	CancelMessage        *string           `json:"cancelMessage"`
	PublishedAt          *time.Time        `json:"publishedAt"`
	CancelledAt          *time.Time        `json:"cancelledAt"`
	CancelledBy          *AccountRef       `json:"cancelledBy"`
	CreatedBy            *AccountRef       `json:"createdBy"`
	UpdatedBy            *AccountRef       `json:"updatedBy"`
	RegistrationCount    int64             `json:"registrationCount" doc:"Inscriptions, annulées comprises"`
	Totals               EventTotals       `json:"totals"`
	Actions              AdminEventActions `json:"actions"`
}

type AdminEventOutput struct{ Body AdminEvent }

type EventIDInput struct {
	ID int64 `path:"id" doc:"Identifiant de l'événement"`
}

type EventBodyInput struct {
	Body EventBody
}

type UpdateEventInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'événement"`
	Body EventBody
}

type CancelEventInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'événement"`
	Body struct {
		Message *string `json:"message,omitempty" required:"false" nullable:"true" maxLength:"1000" doc:"Message aux inscrits"`
	}
}

type ListEventsInput struct {
	Season string `query:"season" pattern:"^(all|[0-9]+)$" doc:"Identifiant de saison ou all ; absent : saison courante"`
	Status string `query:"status" enum:"draft,published,cancelled,finished" doc:"finished : publié et terminé"`
	When   string `query:"when" enum:"upcoming,past" doc:"upcoming : fin non passée ; past : terminé"`
}

type EventListOutput struct {
	Body struct {
		Items []EventSummary `json:"items" doc:"Par date de début croissante"`
	}
}

func (s *server) registerAdminEvents(api huma.API) {
	op := func(id, method, path, summary string, status int) huma.Operation {
		return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{tagEvents}, DefaultStatus: status}
	}
	register(api, auth.Admin, op("list-events", http.MethodGet, "/admin/events", "Lister les événements", 0), s.listEvents)
	register(api, auth.Admin, op("get-event", http.MethodGet, "/admin/events/{id}", "Un événement", 0), s.getEvent)
	register(api, auth.Admin, op("create-event", http.MethodPost, "/admin/events", "Créer un événement (brouillon)", http.StatusCreated), s.createEvent)
	register(api, auth.Admin, op("update-event", http.MethodPut, "/admin/events/{id}", "Modifier un événement", 0), s.updateEvent)
	register(api, auth.Admin, op("publish-event", http.MethodPost, "/admin/events/{id}/publish", "Publier un événement", 0), s.publishEvent)
	register(api, auth.Admin, op("cancel-event", http.MethodPost, "/admin/events/{id}/cancel", "Annuler un événement", 0), s.cancelEvent)
	register(api, auth.Admin, op("restore-event", http.MethodPost, "/admin/events/{id}/restore", "Rétablir un événement annulé", 0), s.restoreEvent)
	register(api, auth.Admin, op("delete-event", http.MethodDelete, "/admin/events/{id}", "Supprimer un événement sans inscription", http.StatusNoContent), s.deleteEvent)
}

func (s *server) listEvents(ctx context.Context, in *ListEventsInput) (*EventListOutput, error) {
	q, now := s.Store.Read, s.now()
	seasons, order, err := seasonIndex(ctx, q, season.Today(now))
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	var seasonID sql.NullInt64
	switch in.Season {
	case "all":
	case "":
		if id := defaultSeason(seasons, order); id != nil {
			seasonID = nid(*id)
		}
	default:
		id, _ := strconv.ParseInt(in.Season, 10, 64)
		seasonID = nid(id)
	}
	rows, err := q.ListEvents(ctx, seasonID)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des événements impossible", err)
	}
	out := &EventListOutput{}
	out.Body.Items = make([]EventSummary, 0, len(rows))
	for _, r := range rows {
		sum := eventSummary(r.Event, r.ConfirmedCount, seasons, now)
		finished := !now.Before(sum.EndsAt)
		switch {
		case in.Status == "finished" && sum.Phase != string(event.PhaseFinished):
			continue
		case in.Status != "" && in.Status != "finished" && sum.Status != in.Status:
			continue
		case in.When == "upcoming" && finished, in.When == "past" && !finished:
			continue
		}
		out.Body.Items = append(out.Body.Items, sum)
	}
	return out, nil
}

// adminEvent lit un événement vu par l'administration.
func (s *server) adminEvent(ctx context.Context, q *sqlc.Queries, id int64) (*AdminEventOutput, error) {
	r, err := q.GetEvent(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgEventGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'événement impossible", err)
	}
	now := s.now()
	seasons, _, err := seasonIndex(ctx, q, season.Today(now))
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	e, st := r.Event, eventState(r.Event)
	out := AdminEvent{
		EventSummary:         eventSummary(e, r.ConfirmedCount, seasons, now),
		Description:          optionalValue(e.Description),
		RegistrationOpensAt:  st.RegistrationOpensAt,
		RegistrationClosesAt: st.RegistrationClosesAt,
		Local: EventLocal{
			StartsAt: event.FormatLocal(st.StartsAt), EndsAt: event.FormatLocal(st.EndsAt),
			RegistrationOpensAt: localPtr(st.RegistrationOpensAt), RegistrationClosesAt: localPtr(st.RegistrationClosesAt),
		},
		CancelMessage:     optionalValue(e.CancelMessage),
		PublishedAt:       st.PublishedAt,
		CancelledAt:       optionalTimeValue(e.CancelledAt),
		CancelledBy:       accountRef(e.CancelledBy, r.CancelledByEmail),
		CreatedBy:         accountRef(e.CreatedBy, r.CreatedByEmail),
		UpdatedBy:         accountRef(e.UpdatedBy, r.UpdatedByEmail),
		RegistrationCount: r.RegistrationCount,
		Actions: AdminEventActions{
			Update:      event.CanUpdate(st),
			UpdateDates: event.CanUpdateDates(st, now),
			Publish:     event.CanPublish(st, now),
			Cancel:      event.CanCancel(st, now),
			Restore:     event.CanRestore(st, now),
			Delete:      event.CanDelete(int(r.RegistrationCount)),
			Register:    event.CanAdminRegister(st, now),
		},
	}
	regs, err := q.ListRegistrationsOfEvent(ctx, sqlc.ListRegistrationsOfEventParams{
		SeasonID: e.SeasonID, EventID: e.ID, Status: ns("confirmed"),
	})
	if err != nil {
		return nil, s.internalError(ctx, "lecture des inscriptions impossible", err)
	}
	out.Totals = eventTotals(regs)
	return &AdminEventOutput{Body: out}, nil
}

func localPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	v := event.FormatLocal(*t)
	return &v
}

// eventTotals calcule les chiffres des inscriptions confirmées d'un
// événement (research.md 014 › R3).
func eventTotals(regs []sqlc.ListRegistrationsOfEventRow) EventTotals {
	var t membership.AmountTotals
	var paid int64
	for _, reg := range regs {
		r := reg.EventRegistration
		if r.PaymentStatus.String == string(membership.Paid) {
			paid++
		}
		t.Add(membership.AmountRow{
			AmountCents: r.AmountCents, BaseCents: basePtr(r.BaseAmountCents),
			PaymentStatus: membership.PaymentStatus(r.PaymentStatus.String), AnnouncedMethod: membership.Method(r.AnnouncedMethod.String),
			PaidMethod: membership.Method(r.PaidMethod.String), PaidAmountCents: r.PaidAmountCents.Int64,
			PaidTest: reg.PaidEnvironment.String == config.HelloAssoSandbox,
		})
	}
	return EventTotals{
		DueCents: t.Due, CollectedCents: t.Collected, CollectedTestCents: t.CollectedTest, PaidCount: paid,
		TheoreticalCents: t.Theoretical, DiscountCents: t.Discounts, SurchargeCents: t.Surcharges,
		DonationCents: t.Donations, RemainingCents: t.Remaining,
	}
}

func (s *server) getEvent(ctx context.Context, in *EventIDInput) (*AdminEventOutput, error) {
	return s.adminEvent(ctx, s.Store.Read, in.ID)
}

// eventInput convertit le corps reçu : le prix est analysé comme le montant
// d'un tarif (season.ParseAmount), chaîne vide → gratuit.
func eventInput(b EventBody) (event.Input, error) {
	in := event.Input{
		Title: b.Title, Location: b.Location, StartsAt: b.StartsAt, EndsAt: b.EndsAt,
		Openness: event.Openness(b.Openness), Capacity: b.Capacity,
		RegistrationOpensAt: b.RegistrationOpensAt, RegistrationClosesAt: b.RegistrationClosesAt,
	}
	if b.Description != nil {
		in.Description = *b.Description
	}
	if strings.TrimSpace(b.Price) != "" {
		cents, err := season.ParseAmount(b.Price)
		if err != nil {
			msg := MsgAmountFormat
			if errors.Is(err, season.ErrAmountTooHigh) {
				msg = MsgEventPrice
			}
			return in, FieldErrors("body.price", msg)
		}
		in.PriceCents = cents
	}
	return in, nil
}

func nullTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return ns(t.UTC().Format(timeFormat))
}

func nullCapacity(c *int) sql.NullInt64 {
	if c == nil {
		return sql.NullInt64{}
	}
	return nid(int64(*c))
}

func (s *server) validEvent(ctx context.Context, q *sqlc.Queries, b EventBody) (event.Event, error) {
	in, err := eventInput(b)
	if err != nil {
		return event.Event{}, err
	}
	seasons, _, err := seasonIndex(ctx, q, s.today())
	if err != nil {
		return event.Event{}, err
	}
	ev, err := event.ValidateEvent(in, seasonSpans(seasons))
	if err != nil {
		return event.Event{}, eventFieldErrors(err)
	}
	return ev, nil
}

func (s *server) createEvent(ctx context.Context, in *EventBodyInput) (*AdminEventOutput, error) {
	var created sqlc.Event
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		ev, err := s.validEvent(ctx, q, in.Body)
		if err != nil {
			return err
		}
		created, err = q.CreateEvent(ctx, sqlc.CreateEventParams{
			SeasonID: ev.SeasonID, Title: ev.Title, Location: ev.Location, Description: ns(ev.Description),
			StartsAt: ev.StartsAt.Format(timeFormat), EndsAt: ev.EndsAt.Format(timeFormat), Openness: string(ev.Openness),
			Capacity: nullCapacity(ev.Capacity), PriceCents: ev.PriceCents,
			RegistrationOpensAt: nullTime(ev.RegistrationOpensAt), RegistrationClosesAt: nullTime(ev.RegistrationClosesAt),
			ActorID: nid(CurrentAccount(ctx).ID), Now: s.timestamp(),
		})
		return err
	})
	if err != nil {
		return nil, s.txError(ctx, "création de l'événement impossible", err)
	}
	s.logEventAction(ctx, "event.create", created.ID, "season_id", created.SeasonID)
	return s.adminEvent(ctx, s.Store.Read, created.ID)
}

func (s *server) updateEvent(ctx context.Context, in *UpdateEventInput) (*AdminEventOutput, error) {
	notified := 0
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		r, err := q.GetEvent(ctx, in.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgEventGone)
		}
		if err != nil {
			return err
		}
		old, st, now := r.Event, eventState(r.Event), s.now()
		if !event.CanUpdate(st) {
			return huma.Error409Conflict(MsgEventCancelledLocked)
		}
		ev, err := s.validEvent(ctx, q, in.Body)
		if err != nil {
			return err
		}
		datesChanged := !ev.StartsAt.Equal(st.StartsAt) || !ev.EndsAt.Equal(st.EndsAt)
		if datesChanged && !event.CanUpdateDates(st, now) {
			return huma.Error409Conflict(MsgEventFinished)
		}
		if err := event.CheckCapacityChange(ev.Capacity, int(r.ConfirmedCount)); err != nil {
			return FieldErrors("body.capacity", capacityBelowMessage(r.ConfirmedCount))
		}
		if err := changedEvent(q.UpdateEvent(ctx, sqlc.UpdateEventParams{
			SeasonID: ev.SeasonID, Title: ev.Title, Location: ev.Location, Description: ns(ev.Description),
			StartsAt: ev.StartsAt.Format(timeFormat), EndsAt: ev.EndsAt.Format(timeFormat), Openness: string(ev.Openness),
			Capacity: nullCapacity(ev.Capacity), PriceCents: ev.PriceCents,
			RegistrationOpensAt: nullTime(ev.RegistrationOpensAt), RegistrationClosesAt: nullTime(ev.RegistrationClosesAt),
			ActorID: nid(CurrentAccount(ctx).ID), Now: s.timestamp(), ID: old.ID,
		})); err != nil {
			return err
		}
		// Changement de date, d'heure ou de lieu d'un événement publié non
		// terminé : les inscrits sont prévenus (FR-030).
		if old.Status == string(event.Published) && !event.Finished(st, now) &&
			(datesChanged || ev.Location != old.Location) {
			updated, err := q.GetEvent(ctx, old.ID)
			if err != nil {
				return err
			}
			notified, err = s.notifyEventRegistrants(ctx, q, "event-changed", updated.Event, mail.Data{
				PreviousWhen: frenchDateTimeRange(st.StartsAt, st.EndsAt), PreviousLocation: old.Location,
			})
			return err
		}
		return nil
	})
	if err != nil {
		return nil, s.txError(ctx, "modification de l'événement impossible", err)
	}
	s.logEventAction(ctx, "event.update", in.ID, "notified", notified)
	return s.adminEvent(ctx, s.Store.Read, in.ID)
}

func capacityBelowMessage(confirmed int64) string {
	return "Il y a déjà " + strconv.FormatInt(confirmed, 10) + " inscrit(s) : la capacité ne peut pas être inférieure."
}

func changedEvent(n int64, err error) error {
	if err == nil && n == 0 {
		return huma.Error409Conflict(MsgEventChanged)
	}
	return err
}

// eventWrite relit l'événement dans la transaction et applique fn.
func (s *server) eventWrite(ctx context.Context, id int64, fn func(q *sqlc.Queries, e sqlc.Event, st event.State, now time.Time) error) error {
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		r, err := q.GetEvent(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgEventGone)
		}
		if err != nil {
			return err
		}
		return fn(q, r.Event, eventState(r.Event), s.now())
	})
	if err != nil {
		return s.txError(ctx, "écriture de l'événement impossible", err)
	}
	return nil
}

func (s *server) publishEvent(ctx context.Context, in *EventIDInput) (*AdminEventOutput, error) {
	err := s.eventWrite(ctx, in.ID, func(q *sqlc.Queries, e sqlc.Event, st event.State, now time.Time) error {
		if !event.CanPublish(st, now) {
			return huma.Error409Conflict(MsgEventChanged)
		}
		return changedEvent(q.PublishEvent(ctx, sqlc.PublishEventParams{Now: s.timestamp(), ActorID: nid(CurrentAccount(ctx).ID), ID: e.ID}))
	})
	if err != nil {
		return nil, err
	}
	s.logEventAction(ctx, "event.publish", in.ID)
	return s.adminEvent(ctx, s.Store.Read, in.ID)
}

func (s *server) cancelEvent(ctx context.Context, in *CancelEventInput) (*AdminEventOutput, error) {
	message := ""
	if in.Body.Message != nil {
		message = strings.TrimSpace(strings.ReplaceAll(*in.Body.Message, "\r\n", "\n"))
	}
	if len([]rune(message)) > event.MaxCancelMessage {
		return nil, FieldErrors("body.message", MsgCancelMessageTooLong)
	}
	notified := 0
	err := s.eventWrite(ctx, in.ID, func(q *sqlc.Queries, e sqlc.Event, st event.State, now time.Time) error {
		if !event.CanCancel(st, now) {
			return huma.Error409Conflict(MsgEventChanged)
		}
		if err := changedEvent(q.CancelEvent(ctx, sqlc.CancelEventParams{
			Message: ns(message), Now: ns(s.timestamp()), ActorID: nid(CurrentAccount(ctx).ID), ID: e.ID,
		})); err != nil {
			return err
		}
		var err error
		notified, err = s.notifyEventRegistrants(ctx, q, "event-cancelled", e, mail.Data{Message: message})
		return err
	})
	if err != nil {
		return nil, err
	}
	s.logEventAction(ctx, "event.cancel", in.ID, "notified", notified)
	return s.adminEvent(ctx, s.Store.Read, in.ID)
}

func (s *server) restoreEvent(ctx context.Context, in *EventIDInput) (*AdminEventOutput, error) {
	notified := 0
	err := s.eventWrite(ctx, in.ID, func(q *sqlc.Queries, e sqlc.Event, st event.State, now time.Time) error {
		if !event.CanRestore(st, now) {
			return huma.Error409Conflict(MsgEventChanged)
		}
		if err := changedEvent(q.RestoreEvent(ctx, sqlc.RestoreEventParams{
			ActorID: nid(CurrentAccount(ctx).ID), Now: s.timestamp(), ID: e.ID,
		})); err != nil {
			return err
		}
		var err error
		notified, err = s.notifyEventRegistrants(ctx, q, "event-restored", e, mail.Data{})
		return err
	})
	if err != nil {
		return nil, err
	}
	s.logEventAction(ctx, "event.restore", in.ID, "notified", notified)
	return s.adminEvent(ctx, s.Store.Read, in.ID)
}

func (s *server) deleteEvent(ctx context.Context, in *EventIDInput) (*struct{}, error) {
	err := s.eventWrite(ctx, in.ID, func(q *sqlc.Queries, e sqlc.Event, _ event.State, _ time.Time) error {
		n, err := q.DeleteEventWithoutRegistrations(ctx, e.ID)
		if err == nil && n == 0 {
			return huma.Error409Conflict(MsgEventHasRegistrations)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	s.logEventAction(ctx, "event.delete", in.ID)
	return nil, nil
}
