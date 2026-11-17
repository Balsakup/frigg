package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/event"
	"github.com/balsakup/frigg/internal/public"
)

// Page publique d'un événement « Ouvert au public » et dépôt d'une
// inscription (016, US3 ; FR-023 à FR-027). La page ne transmet que les
// informations propres à l'événement (principe II) : ni inscrit, ni
// identifiant interne, ni message d'annulation, ni champ personnalisé.

// PublicEventPathPrefix est le préfixe des adresses publiques d'un événement :
// le segment suivant est son lien public, que les journaux masquent
// (research.md 016 › R16).
const PublicEventPathPrefix = "/api/public/events/"

// MsgPublicEventNotFound est la réponse unique d'un lien public inconnu, d'un
// lien régénéré, d'un brouillon et d'un événement non public (FR-025).
const MsgPublicEventNotFound = "Événement introuvable."

// publicClosedMessages expliquent pourquoi une inscription publique est
// impossible (US3 scénarios 4 et 7).
var publicClosedMessages = map[event.ClosedReason]string{
	event.ReasonCancelled: "Cet événement est annulé.",
	event.ReasonFinished:  "Cet événement est terminé.",
	event.ReasonClosed:    MsgRegistrationsClosed,
	event.ReasonNotYet:    "Les inscriptions ne sont pas encore ouvertes.",
	event.ReasonFull:      "Cet événement est complet.",
}

// PublicEvent est la page publique d'un événement : liste fermée (principe II).
type PublicEvent struct {
	Title                string    `json:"title"`
	Location             string    `json:"location"`
	Description          *string   `json:"description"`
	StartsAt             time.Time `json:"startsAt"`
	EndsAt               time.Time `json:"endsAt"`
	PriceCents           int64     `json:"priceCents" doc:"0 : gratuit"`
	RegistrationOpensAt  time.Time `json:"registrationOpensAt" doc:"Ouverture des inscriptions : date saisie, sinon publication"`
	RegistrationClosesAt time.Time `json:"registrationClosesAt" doc:"Clôture des inscriptions : date saisie, sinon début"`
	PlacesLeft           *int64    `json:"placesLeft" doc:"null : capacité illimitée"`
	RegistrationOpen     bool      `json:"registrationOpen" doc:"Le formulaire d'inscription est proposé"`
	ClosedReason         *string   `json:"closedReason" enum:"cancelled,finished,full,not_yet_open,closed" doc:"Raison de l'absence de formulaire ; null si l'inscription est possible"`
}

type PublicEventOutput struct{ Body PublicEvent }

type PublicEventInput struct {
	Slug string `path:"slug" doc:"Lien public de l'événement"`
}

type SubmitPublicRegistrationInput struct {
	Slug string `path:"slug" doc:"Lien public de l'événement"`
	// Body ne porte aucun champ personnalisé (FR-026).
	Body PublicSubmissionBody
}

func (s *server) registerPublicEvents(api huma.API) {
	register(api, auth.Public, huma.Operation{
		OperationID: "get-public-event",
		Method:      http.MethodGet,
		Path:        "/public/events/{slug}",
		Summary:     "Page publique d'un événement",
		Description: "Informations propres à l'événement, sans inscrit. La réponse 404 est identique pour un lien inconnu, régénéré, un brouillon ou un événement non public.",
		Tags:        []string{tagPublic},
	}, s.getPublicEvent)

	register(api, auth.Public, huma.Operation{
		OperationID:   "submit-public-registration",
		Method:        http.MethodPost,
		Path:          "/public/events/{slug}/registrations",
		Summary:       "Déposer une inscription publique à un événement",
		Description:   "La réponse est identique quelle que soit l'adresse. Aucune place n'est réservée avant la confirmation de l'adresse.",
		Tags:          []string{tagPublic},
		DefaultStatus: http.StatusAccepted,
	}, s.submitPublicRegistration)
}

// publicEventBySlug lit un événement par son lien public. Tout événement qui
// n'a pas de page publique donne la même erreur qu'un lien inconnu (FR-025).
func (s *server) publicEventBySlug(ctx context.Context, q *sqlc.Queries, slug string) (sqlc.GetEventByPublicSlugRow, error) {
	r, err := q.GetEventByPublicSlug(ctx, ns(slug))
	if errors.Is(err, sql.ErrNoRows) {
		return r, huma.Error404NotFound(MsgPublicEventNotFound)
	}
	if err != nil {
		return r, s.internalError(ctx, "lecture de l'événement impossible", err)
	}
	e := r.Event
	phase := event.PhaseOf(eventState(e), s.now())
	if !event.PublicVisible(event.Status(e.Status), phase, event.OpennessOf(e.Openness, e.PublicSlug)) {
		return r, huma.Error404NotFound(MsgPublicEventNotFound)
	}
	return r, nil
}

func (s *server) getPublicEvent(ctx context.Context, in *PublicEventInput) (*PublicEventOutput, error) {
	r, err := s.publicEventBySlug(ctx, s.Store.Read, in.Slug)
	if err != nil {
		return nil, err
	}
	e, st, now := r.Event, eventState(r.Event), s.now()
	opens, closes := event.Window(st)
	out := PublicEvent{
		Title: e.Title, Location: e.Location, Description: optionalValue(e.Description),
		StartsAt: st.StartsAt, EndsAt: st.EndsAt, PriceCents: e.PriceCents,
		RegistrationOpensAt: opens, RegistrationClosesAt: closes,
	}
	if left := event.CapacityLeft(st.Capacity, int(r.ConfirmedCount)); left != nil {
		n := int64(*left)
		out.PlacesLeft = &n
	}
	if reason := event.PublicClosedReason(st, int(r.ConfirmedCount), now); reason != event.NotClosed {
		v := string(reason)
		out.ClosedReason = &v
	} else {
		out.RegistrationOpen = true
	}
	return &PublicEventOutput{Body: out}, nil
}

func (s *server) submitPublicRegistration(ctx context.Context, in *SubmitPublicRegistrationInput) (*PublicMessageOutput, error) {
	const kind = "registration"
	b := in.Body
	if !s.Limits.PublicSubmitIP.Allow(clientIP(ctx)) {
		s.logPublicRejected(ctx, kind, "rate_ip")
		return nil, TooManyAttempts()
	}
	if public.Honeypot(b.Website) {
		s.logPublicRejected(ctx, kind, "honeypot")
		return publicAccepted(), nil
	}
	r, err := s.publicEventBySlug(ctx, s.Store.Read, in.Slug)
	if err != nil {
		s.logPublicRejected(ctx, kind, "not_found")
		return nil, err
	}
	e := r.Event
	if reason := event.PublicClosedReason(eventState(e), int(r.ConfirmedCount), s.now()); reason != event.NotClosed {
		s.logPublicRejected(ctx, kind, string(reason))
		return nil, huma.Error409Conflict(publicClosedMessages[reason])
	}
	clean, details := validatePublicIdentity(b, s.today())
	if len(details) > 0 {
		s.logPublicRejected(ctx, kind, "invalid")
		return nil, FieldErrorDetails(details...)
	}

	// Plafond d'e-mails par adresse : au-delà, la réponse reste la même.
	email := *clean.Email
	if !s.Limits.PublicMailEmail.Allow(email) {
		s.logPublicRejected(ctx, kind, "rate_email")
		return publicAccepted(), nil
	}
	raw, hash := auth.NewToken()
	now := s.now().UTC()
	f := fieldsOf(clean)
	var id int64
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		acc, err := lookupEmailAccount(ctx, q, email)
		if err != nil {
			return err
		}
		id, err = q.CreatePublicSubmission(ctx, sqlc.CreatePublicSubmissionParams{
			Kind:          kind,
			EventID:       nid(e.ID),
			FirstName:     f.FirstName,
			LastName:      f.LastName,
			BirthDate:     f.BirthDate,
			Email:         email,
			Phone:         f.Phone,
			AddressLine1:  f.AddressLine1,
			AddressLine2:  f.AddressLine2,
			PostalCode:    f.Postal,
			City:          f.City,
			CustomValues:  "[]",
			CreateAccount: boolInt(b.CreateAccount),
			ConsentedAt:   now.Format(timeFormat),
			TokenHash:     hash,
			ExpiresAt:     now.Add(public.ConfirmTTL).Format(timeFormat),
			CreatedAt:     now.Format(timeFormat),
		})
		if err != nil {
			return err
		}
		data := confirmNotice(acc)
		data.Member = clean.FirstName + " " + clean.LastName
		data.Event = e.Title
		return s.enqueueNotice(ctx, q, "public-confirm", email, data, "/confirmer-demande#jeton="+raw)
	})
	if err != nil {
		return nil, s.internalError(ctx, "enregistrement du dépôt public impossible", err)
	}
	s.Logger.InfoContext(ctx, "dépôt public enregistré", "action", "public_submit", "kind", kind,
		"submission_id", id, "event_id", e.ID)
	return publicAccepted(), nil
}
