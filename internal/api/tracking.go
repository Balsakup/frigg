package api

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/public"
	"github.com/balsakup/frigg/internal/season"
)

// Espace de suivi par lien personnel (016, US2 ; contracts/api.md). Toutes
// les lectures et écritures passent par les requêtes …OfTrackedEmail : une
// donnée hors du périmètre du lien est indiscernable d'une donnée inexistante
// (FR-014). Aucune opération ne modifie l'identité, les coordonnées ni les
// champs personnalisés (FR-016).

// TrackedActions sont les actions proposées pour un élément suivi.
type TrackedActions struct {
	Cancel bool `json:"cancel" doc:"Annuler la demande ou l'inscription"`
	Pay    bool `json:"pay" doc:"Un moyen de paiement est proposé"`
}

// TrackedItem est une adhésion ou une inscription d'une fiche suivie (FR-015).
type TrackedItem struct {
	Kind          string         `json:"kind" enum:"membership,registration"`
	ID            int64          `json:"id"`
	Member        MemberRef      `json:"member"`
	Label         string         `json:"label" doc:"« Adhésion 2026-2027 — Adulte » ou titre de l'événement"`
	Status        string         `json:"status" enum:"pending,validated,refused,cancelled,confirmed" doc:"États de 006 (adhésion) ou de 012 (inscription)"`
	AmountCents   *int64         `json:"amountCents" doc:"Montant à payer ; null si rien n'est à payer"`
	PaymentStatus *string        `json:"paymentStatus" enum:"due,announced,paid"`
	Reference     *string        `json:"reference"`
	Reason        *string        `json:"reason" doc:"Motif d'un refus ou d'une annulation"`
	Actions       TrackedActions `json:"actions"`
}

// Tracking est l'espace de suivi d'une adresse.
type Tracking struct {
	Email            string        `json:"email"`
	Members          []MemberRef   `json:"members" doc:"Fiches suivies, par nom puis prénom"`
	Items            []TrackedItem `json:"items" doc:"Par fiche, adhésions puis inscriptions"`
	CanCreateAccount bool          `json:"canCreateAccount" doc:"Aucun compte pour l'adresse et au plus 20 fiches suivies"`
}

// TrackedMembership est une adhésion vue par lien personnel : les mêmes
// informations de paiement que dans l'espace membre, sans champ du compte.
type TrackedMembership struct {
	Membership
	Actions        MyMembershipActions `json:"actions"`
	PaymentMethods []PaymentMethodInfo `json:"paymentMethods" doc:"Moyens activés ; vide s'il n'y a rien à payer ; la carte en premier"`
	CardPayment    *CardPaymentState   `json:"cardPayment" doc:"Dernière tentative de paiement par carte"`
}

// TrackedRegistration est une inscription vue par lien personnel : les mêmes
// informations que dans l'espace membre (paiement, moyens, tentatives).
type TrackedRegistration struct {
	MyRegistration
}

type TrackingOutput struct{ Body Tracking }
type TrackedMembershipOutput struct{ Body TrackedMembership }
type TrackedRegistrationOutput struct{ Body TrackedRegistration }

func (s *server) registerTrackingSpace(api huma.API) {
	register(api, auth.Link, huma.Operation{
		OperationID: "get-tracking",
		Method:      http.MethodGet,
		Path:        "/tracking",
		Summary:     "Espace de suivi par lien personnel",
		Tags:        []string{tagTracking},
	}, s.getTracking)

	register(api, auth.Link, huma.Operation{
		OperationID: "get-tracking-membership",
		Method:      http.MethodGet,
		Path:        "/tracking/memberships/{id}",
		Summary:     "Une adhésion suivie par lien personnel",
		Tags:        []string{tagTracking},
	}, s.getTrackingMembership)

	register(api, auth.Link, huma.Operation{
		OperationID: "cancel-tracking-membership",
		Method:      http.MethodPost,
		Path:        "/tracking/memberships/{id}/cancel",
		Summary:     "Annuler une demande d'adhésion en attente",
		Tags:        []string{tagTracking},
	}, s.cancelTrackingMembership)

	register(api, auth.Link, huma.Operation{
		OperationID: "announce-tracking-membership-payment",
		Method:      http.MethodPut,
		Path:        "/tracking/memberships/{id}/payment",
		Summary:     "Choisir le moyen de paiement d'une adhésion suivie",
		Tags:        []string{tagTracking},
	}, s.announceTrackingMembershipPayment)

	register(api, auth.Link, huma.Operation{
		OperationID: "get-tracking-registration",
		Method:      http.MethodGet,
		Path:        "/tracking/registrations/{id}",
		Summary:     "Une inscription suivie par lien personnel",
		Tags:        []string{tagTracking},
	}, s.getTrackingRegistration)

	register(api, auth.Link, huma.Operation{
		OperationID: "cancel-tracking-registration",
		Method:      http.MethodPost,
		Path:        "/tracking/registrations/{id}/cancel",
		Summary:     "Annuler une inscription suivie (012 FR-013)",
		Tags:        []string{tagTracking},
	}, s.cancelTrackingRegistration)

	register(api, auth.Link, huma.Operation{
		OperationID: "announce-tracking-registration-payment",
		Method:      http.MethodPut,
		Path:        "/tracking/registrations/{id}/payment",
		Summary:     "Choisir le moyen de paiement d'une inscription suivie",
		Tags:        []string{tagTracking},
	}, s.announceTrackingRegistrationPayment)

	register(api, auth.Link, huma.Operation{
		OperationID:   "create-tracking-account",
		Method:        http.MethodPost,
		Path:          "/tracking/account",
		Summary:       "Créer un compte depuis le suivi par lien",
		Description:   "Rattache les fiches suivies, révoque les liens de l'adresse et ouvre une session de compte (016 FR-031, FR-032).",
		Tags:          []string{tagTracking},
		DefaultStatus: http.StatusCreated,
	}, s.createTrackingAccount)

	register(api, auth.Public, huma.Operation{
		OperationID:   "request-tracking-link",
		Method:        http.MethodPost,
		Path:          "/public/tracking/link-request",
		Summary:       "Recevoir un lien personnel de suivi",
		Description:   "La réponse est identique quelle que soit l'adresse. Un e-mail n'est envoyé qu'à une adresse de fiche suivie par lien.",
		Tags:          []string{tagPublic},
		DefaultStatus: http.StatusAccepted,
	}, s.requestTrackingLink)
}

func (s *server) getTracking(ctx context.Context, _ *struct{}) (*TrackingOutput, error) {
	q, email, now := s.Store.Read, trackedEmail(ctx), s.now()
	today := season.Today(now)
	members, err := q.ListTrackedMembers(ctx, ns(email))
	if err != nil {
		return nil, s.internalError(ctx, "lecture des fiches impossible", err)
	}
	out := Tracking{Email: email, Members: make([]MemberRef, 0, len(members)), Items: []TrackedItem{}}
	rank := make(map[int64]int, len(members))
	for i, m := range members {
		rank[m.ID] = i
		out.Members = append(out.Members, memberRef(m.ID, m.FirstName, m.LastName, m.BirthDate, today))
	}
	seasons, _, err := seasonIndex(ctx, q, today)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	settings, err := s.paymentSettings(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paramètres impossible", err)
	}
	ms, err := q.ListMembershipsOfTrackedEmail(ctx, ns(email))
	if err != nil {
		return nil, s.internalError(ctx, "lecture des adhésions impossible", err)
	}
	for _, r := range ms {
		m := toMembership(r.Membership, r.MembershipFee, r.FirstName, r.LastName, r.BirthDate, seasons, today)
		a := membership.MemberActionsFor(snapshotOf(r.Membership, seasons), settings.CardAvailable)
		item := TrackedItem{
			Kind: "membership", ID: m.ID, Member: m.Member,
			Label:  "Adhésion " + m.Season.Label + " — " + m.Fee.Label,
			Status: m.Status, Reason: m.Reason,
			Actions: TrackedActions{Cancel: a.Cancel, Pay: a.Pay || a.PayByCard},
		}
		if m.AmountCents > 0 {
			item.AmountCents = &m.AmountCents
		}
		if m.Payment != nil {
			item.PaymentStatus, item.Reference = &m.Payment.Status, &m.Payment.Reference
		}
		out.Items = append(out.Items, item)
	}
	regs, err := q.ListRegistrationsOfTrackedEmail(ctx, ns(email))
	if err != nil {
		return nil, s.internalError(ctx, "lecture des inscriptions impossible", err)
	}
	for _, r := range regs {
		reg := r.EventRegistration
		st, err := s.myRegistrationState(ctx, q, reg, r.Event, settings, now)
		if err != nil {
			return nil, s.internalError(ctx, "lecture des paiements impossible", err)
		}
		item := TrackedItem{
			Kind: "registration", ID: reg.ID,
			Member: MemberRef{ID: reg.MemberID, FirstName: r.FirstName, LastName: r.LastName},
			Label:  r.Event.Title, Status: reg.Status,
			Actions: TrackedActions{Cancel: st.actions.Cancel, Pay: st.actions.Pay || st.actions.PayByCard},
		}
		if reg.AmountCents > 0 {
			item.AmountCents = &reg.AmountCents
		}
		if st.payment != nil {
			item.PaymentStatus, item.Reference = &st.payment.Status, &st.payment.Reference
		}
		out.Items = append(out.Items, item)
	}
	slices.SortStableFunc(out.Items, func(a, b TrackedItem) int { return cmp.Compare(rank[a.Member.ID], rank[b.Member.ID]) })

	acc, err := lookupEmailAccount(ctx, q, email)
	if err != nil {
		return nil, s.internalError(ctx, "lecture du compte impossible", err)
	}
	out.CanCreateAccount = acc == nil && len(members) > 0 && len(members) <= public.MaxManagedMembers
	return &TrackingOutput{Body: out}, nil
}

// trackedMembershipOutput relit une adhésion dans le périmètre du lien.
func (s *server) trackedMembershipOutput(ctx context.Context, q *sqlc.Queries, id int64) (*TrackedMembershipOutput, error) {
	r, err := q.GetMembershipOfTrackedEmail(ctx, sqlc.GetMembershipOfTrackedEmailParams{ID: id, Email: ns(trackedEmail(ctx))})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgMembershipGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'adhésion impossible", err)
	}
	today := s.today()
	seasons, _, err := seasonIndex(ctx, q, today)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	settings, err := s.paymentSettings(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paramètres impossible", err)
	}
	m := toMembership(r.Membership, r.MembershipFee, r.FirstName, r.LastName, r.BirthDate, seasons, today)
	if err := withOnline(ctx, q, &m, r.Membership); err != nil {
		return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
	}
	latest, err := latestCardPayment(ctx, q, r.Membership)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
	}
	mine := myMembership(m, snapshotOf(r.Membership, seasons), settings, latest)
	return &TrackedMembershipOutput{Body: TrackedMembership{
		Membership: mine.Membership, Actions: mine.Actions, PaymentMethods: mine.PaymentMethods, CardPayment: mine.CardPayment,
	}}, nil
}

func (s *server) getTrackingMembership(ctx context.Context, in *MembershipIDInput) (*TrackedMembershipOutput, error) {
	return s.trackedMembershipOutput(ctx, s.Store.Read, in.ID)
}

// trackedMembershipWrite exécute fn sur une adhésion du périmètre du lien,
// relue dans la transaction avec son état.
func (s *server) trackedMembershipWrite(ctx context.Context, id int64, fn func(q *sqlc.Queries, m sqlc.Membership, snap membership.Snapshot) error) (sqlc.Membership, error) {
	var current sqlc.Membership
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		r, err := q.GetMembershipOfTrackedEmail(ctx, sqlc.GetMembershipOfTrackedEmailParams{ID: id, Email: ns(trackedEmail(ctx))})
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgMembershipGone)
		}
		if err != nil {
			return err
		}
		seasons, _, err := seasonIndex(ctx, q, s.today())
		if err != nil {
			return err
		}
		current = r.Membership
		return fn(q, r.Membership, snapshotOf(r.Membership, seasons))
	})
	if err != nil {
		return current, s.txError(ctx, "écriture de l'adhésion impossible", err)
	}
	return current, nil
}

// logTrackedMembershipAction trace une écriture par lien personnel, sans
// adresse ni nom (research.md 016 › R16).
func (s *server) logTrackedMembershipAction(ctx context.Context, action string, m sqlc.Membership, attrs ...any) {
	base := []any{
		"action", action, "membership_id", m.ID, "member_id", m.MemberID, "season_id", m.SeasonID, "space", "tracking",
	}
	s.Logger.InfoContext(ctx, "écriture sur une adhésion", append(base, attrs...)...)
}

func (s *server) cancelTrackingMembership(ctx context.Context, in *MembershipIDInput) (*TrackedMembershipOutput, error) {
	m, err := s.trackedMembershipWrite(ctx, in.ID, func(q *sqlc.Queries, m sqlc.Membership, snap membership.Snapshot) error {
		if !membership.CanCancelRequest(snap) {
			return huma.Error409Conflict(MsgMembershipChanged)
		}
		return changed(q.CancelMembershipOfTrackedEmail(ctx, sqlc.CancelMembershipOfTrackedEmailParams{
			Now: ns(s.timestamp()), ID: m.ID, Email: ns(trackedEmail(ctx)),
		}))
	})
	if err != nil {
		return nil, err
	}
	s.logTrackedMembershipAction(ctx, "membership.cancel", m)
	return s.trackedMembershipOutput(ctx, s.Store.Read, in.ID)
}

func (s *server) announceTrackingMembershipPayment(ctx context.Context, in *AnnounceMyPaymentInput) (*TrackedMembershipOutput, error) {
	m, err := s.trackedMembershipWrite(ctx, in.ID, func(q *sqlc.Queries, m sqlc.Membership, snap membership.Snapshot) error {
		if err := s.checkAnnounce(ctx, q, snap, in.Body.Method); err != nil {
			return err
		}
		return changed(q.AnnouncePaymentOfTrackedEmail(ctx, sqlc.AnnouncePaymentOfTrackedEmailParams{
			Method: ns(in.Body.Method), Now: s.timestamp(), ID: m.ID, Email: ns(trackedEmail(ctx)),
		}))
	})
	if err != nil {
		return nil, err
	}
	s.logTrackedMembershipAction(ctx, "membership.payment_announce", m, "method", in.Body.Method)
	return s.trackedMembershipOutput(ctx, s.Store.Read, in.ID)
}

// trackedRegistrationOutput relit une inscription dans le périmètre du lien.
func (s *server) trackedRegistrationOutput(ctx context.Context, q *sqlc.Queries, id int64) (*TrackedRegistrationOutput, error) {
	r, err := q.GetRegistrationOfTrackedEmail(ctx, sqlc.GetRegistrationOfTrackedEmailParams{ID: id, Email: ns(trackedEmail(ctx))})
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
	return &TrackedRegistrationOutput{Body: TrackedRegistration{MyRegistration: out}}, nil
}

func (s *server) getTrackingRegistration(ctx context.Context, in *RegistrationIDInput) (*TrackedRegistrationOutput, error) {
	return s.trackedRegistrationOutput(ctx, s.Store.Read, in.ID)
}

// trackedRegistrationWrite exécute fn sur une inscription du périmètre du
// lien, relue dans la transaction avec son état.
func (s *server) trackedRegistrationWrite(ctx context.Context, id int64, fn func(q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, st registrationState) error) (sqlc.EventRegistration, error) {
	return s.registrationWrite(ctx, func(q *sqlc.Queries) (sqlc.EventRegistration, sqlc.Event, error) {
		r, err := q.GetRegistrationOfTrackedEmail(ctx, sqlc.GetRegistrationOfTrackedEmailParams{ID: id, Email: ns(trackedEmail(ctx))})
		return r.EventRegistration, r.Event, err
	}, fn)
}

// logTrackedRegistrationAction trace une écriture par lien personnel sur une
// inscription, sans adresse ni nom (research.md 016 › R16).
func (s *server) logTrackedRegistrationAction(ctx context.Context, action string, r sqlc.EventRegistration, attrs ...any) {
	base := []any{"action", action, "event_id", r.EventID, "registration_id", r.ID, "member_id", r.MemberID, "space", "tracking"}
	s.Logger.InfoContext(ctx, "écriture sur une inscription", append(base, attrs...)...)
}

func (s *server) cancelTrackingRegistration(ctx context.Context, in *RegistrationIDInput) (*TrackedRegistrationOutput, error) {
	r, err := s.trackedRegistrationWrite(ctx, in.ID, func(q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, st registrationState) error {
		if err := s.checkUnregister(r, e, st); err != nil {
			return err
		}
		return changedRegistration(q.CancelRegistrationOfTrackedEmail(ctx, sqlc.CancelRegistrationOfTrackedEmailParams{
			Now: ns(s.timestamp()), ID: r.ID, Email: ns(trackedEmail(ctx)),
		}))
	})
	if err != nil {
		return nil, err
	}
	s.logTrackedRegistrationAction(ctx, "registration.cancel", r)
	return s.trackedRegistrationOutput(ctx, s.Store.Read, in.ID)
}

func (s *server) announceTrackingRegistrationPayment(ctx context.Context, in *AnnounceRegistrationInput) (*TrackedRegistrationOutput, error) {
	r, err := s.trackedRegistrationWrite(ctx, in.ID, func(q *sqlc.Queries, r sqlc.EventRegistration, e sqlc.Event, st registrationState) error {
		if err := s.checkRegistrationAnnounce(ctx, q, e, st, in.Body.Method); err != nil {
			return err
		}
		return changedRegistration(q.AnnounceRegistrationPaymentOfTrackedEmail(ctx, sqlc.AnnounceRegistrationPaymentOfTrackedEmailParams{
			Method: ns(in.Body.Method), Now: s.timestamp(), ID: r.ID, Email: ns(trackedEmail(ctx)),
		}))
	})
	if err != nil {
		return nil, err
	}
	s.logTrackedRegistrationAction(ctx, "registration.payment_announce", r, "method", in.Body.Method)
	return s.trackedRegistrationOutput(ctx, s.Store.Read, in.ID)
}

// requestTrackingLink envoie un lien personnel à une adresse de fiche suivie
// par lien (FR-017). La réponse est toujours la même (SC-005) : adresse
// inconnue, adresse d'un compte, plafond d'e-mails atteint ou envoi.
func (s *server) requestTrackingLink(ctx context.Context, in *EmailInput) (*PublicMessageOutput, error) {
	if !s.Limits.TrackingRequestIP.Allow(clientIP(ctx)) {
		return nil, TooManyAttempts()
	}
	email := auth.NormalizeEmail(in.Body.Email)
	if !auth.ValidateEmail(email) {
		return nil, FieldErrors("body.email", MsgInvalidEmail)
	}
	outcome := "none"
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		n, err := q.CountTrackedMembers(ctx, ns(email))
		if err != nil || n == 0 {
			return err
		}
		if !s.Limits.PublicMailEmail.Allow(email) {
			outcome = "rate_email"
			return nil
		}
		raw, err := s.issueTrackingLink(ctx, q, email)
		if err != nil {
			return err
		}
		outcome = "sent"
		return s.enqueueNotice(ctx, q, "tracking-link", email, mail.Data{}, "/suivi#jeton="+raw)
	})
	if err != nil {
		return nil, s.internalError(ctx, "demande de lien personnel impossible", err)
	}
	s.Logger.InfoContext(ctx, "demande de lien personnel", "action", "tracking_link_request", "outcome", outcome)
	return publicAccepted(), nil
}

const (
	// MsgTrackingAccountExists : un compte existe déjà pour l'adresse du lien
	// (research.md 016 › R12). L'appelant détient la messagerie.
	MsgTrackingAccountExists = "Un compte existe déjà pour cette adresse. Connectez-vous avec cette adresse."
	// MsgTrackingTooManyMembers : plus de 20 fiches suivies (FR-031).
	MsgTrackingTooManyMembers = "Votre adresse suit plus de 20 fiches. Contactez l'association pour créer votre compte."
	// MsgTrackingNoMember : aucune fiche suivie, donc aucun dépôt confirmé
	// (FR-034).
	MsgTrackingNoMember = "Aucune fiche n'est suivie avec ce lien."
	// MsgSelfMemberOutOfScope : la fiche « Vous » doit être une fiche suivie.
	MsgSelfMemberOutOfScope = "Cette fiche ne fait pas partie de votre suivi."
)

// CreateTrackingAccountInput porte le mot de passe et la fiche « Vous ».
type CreateTrackingAccountInput struct {
	Body struct {
		Password     string `json:"password" maxLength:"1024" doc:"Au moins 8 caractères, dont une majuscule et un chiffre"`
		SelfMemberID *int64 `json:"selfMemberId,omitempty" nullable:"true" required:"false" doc:"Fiche « Vous » parmi les fiches suivies ; null : aucune. Ignoré avec une seule fiche, marquée d'office"`
	}
}

// createTrackingAccount crée un compte membre pour l'adresse du lien
// (research.md 016 › R12) : adresse confirmée, fiches suivies rattachées,
// fiche « Vous » désignée, liens de l'adresse révoqués, puis connexion.
func (s *server) createTrackingAccount(ctx context.Context, in *CreateTrackingAccountInput) (*SessionOutput, error) {
	email := trackedEmail(ctx)
	hash, err := s.hashNewPassword(ctx, in.Body.Password)
	if err != nil {
		return nil, err
	}
	var account sqlc.Account
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		if _, err := q.GetAccountByEmail(ctx, email); err == nil {
			return huma.Error409Conflict(MsgTrackingAccountExists)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		members, err := q.ListTrackedMembers(ctx, ns(email))
		if err != nil {
			return err
		}
		switch {
		case len(members) == 0:
			return huma.Error409Conflict(MsgTrackingNoMember)
		case len(members) > public.MaxManagedMembers:
			return huma.Error409Conflict(MsgTrackingTooManyMembers)
		}
		self := in.Body.SelfMemberID
		if len(members) == 1 {
			self = &members[0].ID
		} else if self != nil && !slices.ContainsFunc(members, func(m sqlc.Member) bool { return m.ID == *self }) {
			return FieldErrors("body.selfMemberId", MsgSelfMemberOutOfScope)
		}
		now := s.timestamp()
		account, err = q.CreateAccount(ctx, sqlc.CreateAccountParams{
			Email:           email,
			PasswordHash:    hash,
			Role:            auth.RoleMember,
			EmailVerifiedAt: ns(now),
			CreatedAt:       now,
		})
		if err != nil {
			return err
		}
		if _, err := q.AttachTrackedMembers(ctx, sqlc.AttachTrackedMembersParams{AccountID: nid(account.ID), Now: now, Email: ns(email)}); err != nil {
			return err
		}
		if self != nil {
			if _, err := q.SetSelfMemberOfAccount(ctx, sqlc.SetSelfMemberOfAccountParams{Now: now, ID: *self, AccountID: nid(account.ID)}); err != nil {
				return err
			}
		}
		_, err = s.revokeTrackingLinks(ctx, q, email)
		return err
	})
	if err != nil {
		return nil, s.txError(ctx, "création du compte impossible", err)
	}
	s.Sessions.Remove(ctx, trackingLinkSessionKey)
	s.Logger.InfoContext(ctx, "compte créé depuis le suivi par lien", "action", "tracking_account_created", "account_id", account.ID)
	return s.openSession(ctx, account)
}
