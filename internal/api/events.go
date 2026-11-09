package api

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/event"
	"github.com/balsakup/frigg/internal/season"
)

// Événements et inscriptions (spec 012). Les règles vivent dans
// internal/event ; ce fichier porte les types communs de l'API.

const (
	tagEvents        = "Événements"
	tagRegistrations = "Inscriptions"
)

const (
	MsgEventGone              = "Cet événement n'existe plus."
	MsgEventChanged           = "L'événement a changé entre-temps : rechargez la page."
	MsgEventFinished          = "Cet événement est terminé : ses dates ne peuvent plus changer."
	MsgEventCancelledLocked   = "Cet événement est annulé : rétablissez-le avant de le modifier."
	MsgEventHasRegistrations  = "Cet événement a des inscriptions : annulez-le plutôt."
	MsgEventTitle             = "Saisissez un titre de 120 caractères au plus."
	MsgEventLocation          = "Saisissez un lieu de 200 caractères au plus."
	MsgEventDescription       = "5 000 caractères au plus."
	MsgEventDateTime          = "Saisissez une date et une heure valides."
	MsgEventNonexistentTime   = "Cette heure n'existe pas ce jour-là (changement d'heure)."
	MsgEventEndBeforeStart    = "La fin doit suivre le début."
	MsgEventNoSeason          = "Aucune saison ne couvre cette date : créez d'abord la saison."
	MsgEventCapacity          = "La capacité va de 1 à 10 000 places."
	MsgEventPrice             = "Le prix va de 0 à 10 000 €."
	MsgEventOpenness          = "Choisissez qui peut s'inscrire."
	MsgEventCloseBeforeOpen   = "La clôture doit suivre l'ouverture des inscriptions."
	MsgEventCloseAfterEnd     = "Les inscriptions doivent fermer au plus tard à la fin de l'événement."
	MsgCancelMessageTooLong   = "1 000 caractères au plus."
	MsgRegistrationGone       = "Cette inscription n'existe pas."
	MsgRegistrationChanged    = "L'inscription a changé entre-temps : rechargez la page."
	MsgRegistrationsClosed    = "Les inscriptions sont closes."
	MsgRegistrationPaid       = "Cette inscription est réglée : contactez l'association pour l'annuler."
	MsgNotPayableRegistration = "Cette inscription n'est pas à payer."
)

// EventWindow est la période d'inscription effective de l'espace membre.
type EventWindow struct {
	OpensAt  time.Time `json:"opensAt" doc:"Ouverture : date saisie, sinon publication"`
	ClosesAt time.Time `json:"closesAt" doc:"Clôture : date saisie, sinon début de l'événement"`
}

// EventSummary est un événement dans une liste.
type EventSummary struct {
	ID               int64        `json:"id"`
	Title            string       `json:"title"`
	Location         string       `json:"location"`
	StartsAt         time.Time    `json:"startsAt"`
	EndsAt           time.Time    `json:"endsAt"`
	Season           SeasonRef    `json:"season"`
	Status           string       `json:"status" enum:"draft,published,cancelled"`
	Phase            string       `json:"phase" enum:"draft,upcoming,finished,cancelled" doc:"« Terminé » est calculé"`
	Openness         string       `json:"openness" enum:"season_members,all_members"`
	Capacity         *int64       `json:"capacity" doc:"null : illimitée"`
	PriceCents       int64        `json:"priceCents" doc:"0 : gratuit"`
	ConfirmedCount   int64        `json:"confirmedCount"`
	PlacesLeft       *int64       `json:"placesLeft" doc:"null : illimité"`
	RegistrationOpen bool         `json:"registrationOpen" doc:"Inscription possible maintenant depuis l'espace membre"`
	Window           *EventWindow `json:"window" doc:"null : brouillon"`
}

// EventRef désigne un événement.
type EventRef struct {
	ID       int64     `json:"id"`
	Title    string    `json:"title"`
	StartsAt time.Time `json:"startsAt"`
	Status   string    `json:"status" enum:"draft,published,cancelled"`
}

func intPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}

// eventState présente un événement enregistré au domaine.
func eventState(e sqlc.Event) event.State {
	var capacity *int
	if e.Capacity.Valid {
		capacity = intPtr(e.Capacity)
	}
	return event.State{
		Status:               event.Status(e.Status),
		StartsAt:             parseTime(e.StartsAt),
		EndsAt:               parseTime(e.EndsAt),
		PublishedAt:          optionalTimeValue(e.PublishedAt),
		RegistrationOpensAt:  optionalTimeValue(e.RegistrationOpensAt),
		RegistrationClosesAt: optionalTimeValue(e.RegistrationClosesAt),
		Capacity:             capacity,
	}
}

func eventSummary(e sqlc.Event, confirmed int64, seasons map[int64]seasonInfo, now time.Time) EventSummary {
	st := eventState(e)
	out := EventSummary{
		ID: e.ID, Title: e.Title, Location: e.Location,
		StartsAt: st.StartsAt, EndsAt: st.EndsAt,
		Season: seasons[e.SeasonID].ref, Status: e.Status, Phase: string(event.PhaseOf(st, now)),
		Openness: e.Openness, Capacity: optionalInt(e.Capacity), PriceCents: e.PriceCents,
		ConfirmedCount:   confirmed,
		RegistrationOpen: event.CanRegister(st, now),
	}
	if left := event.CapacityLeft(st.Capacity, int(confirmed)); left != nil {
		n := int64(*left)
		out.PlacesLeft = &n
	}
	if e.Status != string(event.Draft) {
		opens, closes := event.Window(st)
		out.Window = &EventWindow{OpensAt: opens, ClosesAt: closes}
	}
	return out
}

func eventRef(e sqlc.Event) EventRef {
	return EventRef{ID: e.ID, Title: e.Title, StartsAt: parseTime(e.StartsAt), Status: e.Status}
}

// seasonSpans réduit l'index des saisons à leurs bornes.
func seasonSpans(index map[int64]seasonInfo) []event.SeasonSpan {
	out := make([]event.SeasonSpan, 0, len(index))
	for id, s := range index {
		out = append(out, event.SeasonSpan{ID: id, StartsOn: s.startsOn, EndsOn: s.endsOn})
	}
	return out
}

// eventFieldErrors traduit les erreurs du domaine en un seul 422.
func eventFieldErrors(err error) error {
	var fes season.FieldErrors
	if !errors.As(err, &fes) {
		return err
	}
	messages := map[error]string{
		event.ErrTooLong:         MsgEventDescription,
		event.ErrDateTime:        MsgEventDateTime,
		event.ErrNonexistentTime: MsgEventNonexistentTime,
		event.ErrEndBeforeStart:  MsgEventEndBeforeStart,
		event.ErrNoSeason:        MsgEventNoSeason,
		event.ErrCapacity:        MsgEventCapacity,
		event.ErrPrice:           MsgEventPrice,
		event.ErrOpenness:        MsgEventOpenness,
		event.ErrCloseBeforeOpen: MsgEventCloseBeforeOpen,
		event.ErrCloseAfterEnd:   MsgEventCloseAfterEnd,
	}
	details := make([]*huma.ErrorDetail, len(fes))
	for i, fe := range fes {
		msg := messages[fe.Err]
		switch fe.Field {
		case "title":
			msg = MsgEventTitle
		case "location":
			msg = MsgEventLocation
		}
		details[i] = &huma.ErrorDetail{Location: "body." + fe.Field, Message: msg}
	}
	return FieldErrorDetails(details...)
}

// logEventAction trace une écriture sur un événement, sans titre, lieu,
// description ni message (research.md 012 › R15).
func (s *server) logEventAction(ctx context.Context, action string, eventID int64, attrs ...any) {
	base := []any{"action", action, "actor_id", CurrentAccount(ctx).ID, "event_id", eventID}
	s.Logger.InfoContext(ctx, "écriture sur un événement", append(base, attrs...)...)
}

// logRegistrationAction trace une écriture sur une inscription, sans nom.
func (s *server) logRegistrationAction(ctx context.Context, action string, eventID, registrationID int64, attrs ...any) {
	base := []any{"action", action, "actor_id", CurrentAccount(ctx).ID, "event_id", eventID, "registration_id", registrationID}
	s.Logger.InfoContext(ctx, "écriture sur une inscription", append(base, attrs...)...)
}

func (s *server) registerEvents(api huma.API) {
	s.registerAdminEvents(api)
	s.registerMyEvents(api)
	s.registerMyRegistrations(api)
	s.registerAdminRegistrations(api)
}
