package api

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/event"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
	"github.com/balsakup/frigg/internal/textfold"
)

// Événements dans l'espace membre (US2, US3, US4 ; contracts/api.md 012).

// MyEventMember est une fiche gérée par le compte, vue depuis un événement.
type MyEventMember struct {
	Member       MemberRef          `json:"member"`
	Registration *MyRegistrationRef `json:"registration" doc:"Inscription confirmée ; null sinon"`
	Eligible     bool               `json:"eligible"`
	Reason       *string            `json:"reason" enum:"not_season_member,already_registered" doc:"Raison de l'inéligibilité"`
}

// MyRegistrationRef résume l'inscription d'une fiche.
type MyRegistrationRef struct {
	ID      int64                 `json:"id"`
	Payment *Payment              `json:"payment" doc:"null : inscription gratuite"`
	Actions MyRegistrationActions `json:"actions"`
}

// MyEvent est un événement vu depuis l'espace membre.
type MyEvent struct {
	EventSummary
	Description   *string         `json:"description"`
	CancelMessage *string         `json:"cancelMessage"`
	Members       []MyEventMember `json:"members" doc:"Fiches gérées par le compte, par prénom"`
}

type MyEventOutput struct{ Body MyEvent }

type MyEventListOutput struct {
	Body struct {
		Items []EventSummary `json:"items" doc:"Publiés ou annulés, non terminés, par date de début croissante"`
	}
}

// RegisterBody est une demande d'inscription de fiches.
type RegisterBody struct {
	MemberIDs []int64 `json:"memberIds" minItems:"1" maxItems:"20" uniqueItems:"true"`
}

type RegisterMyMembersInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'événement"`
	Body RegisterBody
}

// RegisterResult est l'événement après inscription et les inscriptions
// créées, dans l'ordre de memberIds.
type RegisterResult struct {
	Event                  MyEvent `json:"event"`
	CreatedRegistrationIDs []int64 `json:"createdRegistrationIds"`
}

type RegisterResultOutput struct{ Body RegisterResult }

func (s *server) registerMyEvents(api huma.API) {
	op := func(id, method, path, summary string, status int) huma.Operation {
		return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{tagEvents}, DefaultStatus: status}
	}
	register(api, auth.Member, op("list-my-events", http.MethodGet, "/me/events", "Événements à venir", 0), s.listMyEvents)
	register(api, auth.Member, op("get-my-event", http.MethodGet, "/me/events/{id}", "Un événement", 0), s.getMyEvent)
	register(api, auth.Member, op("register-my-members", http.MethodPost, "/me/events/{id}/registrations", "Inscrire des fiches à un événement", http.StatusCreated), s.registerMembersToEvent)
}

func (s *server) listMyEvents(ctx context.Context, _ *struct{}) (*MyEventListOutput, error) {
	q, now := s.Store.Read, s.now()
	seasons, _, err := seasonIndex(ctx, q, season.Today(now))
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	rows, err := q.ListMemberEvents(ctx, now.UTC().Format(timeFormat))
	if err != nil {
		return nil, s.internalError(ctx, "lecture des événements impossible", err)
	}
	out := &MyEventListOutput{}
	out.Body.Items = make([]EventSummary, len(rows))
	for i, r := range rows {
		out.Body.Items[i] = eventSummary(r.Event, r.ConfirmedCount, seasons, now)
	}
	return out, nil
}

func (s *server) getMyEvent(ctx context.Context, in *EventIDInput) (*MyEventOutput, error) {
	ev, err := s.myEvent(ctx, s.Store.Read, in.ID)
	if err != nil {
		return nil, err
	}
	return &MyEventOutput{Body: *ev}, nil
}

// myEvent lit un événement non brouillon et les fiches du compte.
func (s *server) myEvent(ctx context.Context, q *sqlc.Queries, id int64) (*MyEvent, error) {
	r, err := q.GetMemberEvent(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgEventGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'événement impossible", err)
	}
	now := s.now()
	today := season.Today(now)
	seasons, _, err := seasonIndex(ctx, q, today)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	e := r.Event
	out := MyEvent{
		EventSummary:  eventSummary(e, r.ConfirmedCount, seasons, now),
		Description:   optionalValue(e.Description),
		CancelMessage: optionalValue(e.CancelMessage),
		Members:       []MyEventMember{},
	}
	account := ownAccount(ctx)
	members, err := q.ListMembersOfAccount(ctx, account)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des fiches impossible", err)
	}
	seasonMembers, err := seasonMemberSet(ctx, q, e.SeasonID)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des adhésions impossible", err)
	}
	regs, err := q.ListAccountRegistrationsOfEvent(ctx, sqlc.ListAccountRegistrationsOfEventParams{EventID: e.ID, AccountID: account})
	if err != nil {
		return nil, s.internalError(ctx, "lecture des inscriptions impossible", err)
	}
	byMember := make(map[int64]sqlc.EventRegistration, len(regs))
	for _, reg := range regs {
		byMember[reg.MemberID] = reg
	}
	settings, err := s.paymentSettings(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paramètres impossible", err)
	}
	slices.SortFunc(members, func(a, b sqlc.Member) int {
		return cmp.Or(cmp.Compare(textfold.Fold(a.FirstName), textfold.Fold(b.FirstName)), cmp.Compare(a.ID, b.ID))
	})
	for _, m := range members {
		item := MyEventMember{Member: memberRef(m.ID, m.FirstName, m.LastName, m.BirthDate, today)}
		reg, registered := byMember[m.ID]
		if registered {
			st, err := s.myRegistrationState(ctx, q, reg, e, settings, now)
			if err != nil {
				return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
			}
			item.Registration = &MyRegistrationRef{ID: reg.ID, Payment: st.payment, Actions: st.actions}
		}
		reason := event.Eligibility(event.Openness(e.Openness), seasonMembers[m.ID], registered)
		item.Eligible = reason == event.Eligible
		if !item.Eligible {
			v := string(reason)
			item.Reason = &v
		}
		out.Members = append(out.Members, item)
	}
	return &out, nil
}

func seasonMemberSet(ctx context.Context, q *sqlc.Queries, seasonID int64) (map[int64]bool, error) {
	ids, err := q.ListSeasonMemberIDs(ctx, seasonID)
	if err != nil {
		return nil, err
	}
	set := make(map[int64]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

// fullMessage annonce les places restantes d'un événement complet.
func fullMessage(left int) string {
	return fmt.Sprintf("Complet : il reste %d place(s).", left)
}

// createRegistrations inscrit des fiches dans la transaction courante, après
// contrôle de l'éligibilité et des places (R6 : tout ou rien). Les erreurs
// d'éligibilité visent location+"[i]" (location seul pour body.memberId).
// L'appelant envoie la confirmation (notifyCreated).
func (s *server) createRegistrations(ctx context.Context, q *sqlc.Queries, e sqlc.Event, members []sqlc.Member, location string) ([]sqlc.EventRegistration, error) {
	seasonMembers, err := seasonMemberSet(ctx, q, e.SeasonID)
	if err != nil {
		return nil, err
	}
	seasons, _, err := seasonIndex(ctx, q, s.today())
	if err != nil {
		return nil, err
	}
	info := seasons[e.SeasonID]
	var details []*huma.ErrorDetail
	for i, m := range members {
		n, err := q.CountMemberConfirmedRegistration(ctx, sqlc.CountMemberConfirmedRegistrationParams{EventID: e.ID, MemberID: m.ID})
		if err != nil {
			return nil, err
		}
		loc := location
		if location != "body.memberId" {
			loc = fmt.Sprintf("%s[%d]", location, i)
		}
		switch event.Eligibility(event.Openness(e.Openness), seasonMembers[m.ID], n > 0) {
		case event.AlreadyRegistered:
			details = append(details, &huma.ErrorDetail{Location: loc, Message: m.FirstName + " est déjà inscrit(e)."})
		case event.NotSeasonMember:
			details = append(details, &huma.ErrorDetail{Location: loc,
				Message: fmt.Sprintf("%s n'est pas adhérent(e) de la saison %s.", m.FirstName, info.ref.Label)})
		}
	}
	if len(details) > 0 {
		return nil, FieldErrorDetails(details...)
	}
	confirmed, err := q.CountConfirmedRegistrations(ctx, e.ID)
	if err != nil {
		return nil, err
	}
	var full *event.ErrFull
	if err := event.CheckPlaces(intPtr(e.Capacity), int(confirmed), len(members)); errors.As(err, &full) {
		return nil, huma.Error409Conflict(fullMessage(full.Left))
	}
	var refs []string
	prefix := event.ReferencePrefix(info.startsOn, info.endsOn)
	if e.PriceCents > 0 {
		if refs, err = q.ListRegistrationReferences(ctx, prefix); err != nil {
			return nil, err
		}
	}
	actor := nid(CurrentAccount(ctx).ID)
	created := make([]sqlc.EventRegistration, 0, len(members))
	for _, m := range members {
		params := sqlc.CreateRegistrationParams{
			EventID: e.ID, MemberID: m.ID, ActorID: actor, Now: s.timestamp(), AmountCents: e.PriceCents,
		}
		if e.PriceCents > 0 {
			ref := membership.NextReference(prefix, refs)
			refs = append(refs, ref)
			params.PaymentReference, params.PaymentStatus = ns(ref), ns(string(membership.Due))
		}
		r, err := q.CreateRegistration(ctx, params)
		if err != nil {
			return nil, err
		}
		created = append(created, r)
	}
	return created, nil
}

// notifyCreated envoie l'e-mail de confirmation des inscriptions créées.
func (s *server) notifyCreated(ctx context.Context, q *sqlc.Queries, e sqlc.Event, created []sqlc.EventRegistration) error {
	ids := make([]int64, len(created))
	for i, r := range created {
		ids[i] = r.ID
	}
	return s.notifyRegistrations(ctx, q, "registration-confirmed", e, ids, mail.Data{})
}

func (s *server) registerMembersToEvent(ctx context.Context, in *RegisterMyMembersInput) (*RegisterResultOutput, error) {
	account := ownAccount(ctx)
	var created []sqlc.EventRegistration
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		r, err := q.GetMemberEvent(ctx, in.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgEventGone)
		}
		if err != nil {
			return err
		}
		if !event.CanRegister(eventState(r.Event), s.now()) {
			return huma.Error409Conflict(MsgRegistrationsClosed)
		}
		owned, err := q.ListMembersOfAccount(ctx, account)
		if err != nil {
			return err
		}
		byID := make(map[int64]sqlc.Member, len(owned))
		for _, m := range owned {
			byID[m.ID] = m
		}
		members := make([]sqlc.Member, len(in.Body.MemberIDs))
		for i, id := range in.Body.MemberIDs {
			m, ok := byID[id]
			if !ok {
				return huma.Error404NotFound(MsgMemberGone)
			}
			members[i] = m
		}
		if created, err = s.createRegistrations(ctx, q, r.Event, members, "body.memberIds"); err != nil {
			return err
		}
		return s.notifyCreated(ctx, q, r.Event, created)
	})
	if err != nil {
		return nil, s.txError(ctx, "inscription impossible", err)
	}
	out := RegisterResult{CreatedRegistrationIDs: make([]int64, len(created))}
	for i, r := range created {
		out.CreatedRegistrationIDs[i] = r.ID
		s.logRegistrationAction(ctx, "registration.create", r.EventID, r.ID, "space", "me",
			"payment_reference", r.PaymentReference.String)
	}
	ev, err := s.myEvent(ctx, s.Store.Read, in.ID)
	if err != nil {
		return nil, err
	}
	out.Event = *ev
	return &RegisterResultOutput{Body: out}, nil
}
