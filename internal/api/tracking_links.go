package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/public"
)

// Tags OpenAPI des parcours publics (contracts/api.md de 016).
const (
	tagPublic   = "public"
	tagTracking = "tracking"
)

// MsgLinkInvalid est la réponse unique à un lien personnel inconnu, révoqué
// ou expiré, et à une opération par lien sans lien ouvert en session.
const MsgLinkInvalid = "Ce lien n'est plus valable."

// trackingLinkSessionKey range dans la session scs l'identifiant du lien
// personnel ouvert, à côté des clés de compte qu'il ne touche jamais.
const trackingLinkSessionKey = "tracking_link_id"

// linkTouchEvery espace les mises à jour de last_used_at d'un lien.
const linkTouchEvery = time.Minute

// registerTracking enregistre l'ouverture et la fermeture de l'accès par lien
// personnel (research.md 016 › R1).
func (s *server) registerTracking(api huma.API) {
	register(api, auth.Public, huma.Operation{
		OperationID:   "open-tracking-link",
		Method:        http.MethodPost,
		Path:          "/public/tracking/open",
		Summary:       "Ouvrir un lien personnel de suivi",
		Description:   "Le jeton passe dans le corps, jamais dans l'URL. Ouvre l'accès par lien dans la session.",
		Tags:          []string{tagPublic},
		DefaultStatus: http.StatusNoContent,
	}, s.openTrackingLink)

	register(api, auth.Link, huma.Operation{
		OperationID:   "close-tracking",
		Method:        http.MethodPost,
		Path:          "/tracking/close",
		Summary:       "Fermer l'accès par lien personnel",
		Description:   "Retire seulement l'accès par lien : une session de compte reste ouverte.",
		Tags:          []string{tagTracking},
		DefaultStatus: http.StatusNoContent,
	}, s.closeTracking)
}

func (s *server) openTrackingLink(ctx context.Context, in *TokenInput) (*struct{}, error) {
	if !s.Limits.TrackingOpenIP.Allow(clientIP(ctx)) {
		return nil, TooManyAttempts()
	}
	link, err := s.Store.Read.GetTrackingLinkByTokenHash(ctx, auth.HashToken(in.Body.Token))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.NewError(http.StatusGone, MsgLinkInvalid)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture du lien personnel impossible", err)
	}
	// Adresse connue seulement après la lecture du jeton (R9).
	if !s.Limits.TrackingOpenEmail.Allow(link.Email) {
		return nil, TooManyAttempts()
	}
	if link.RevokedAt.Valid || link.ExpiresAt <= s.timestamp() {
		return nil, huma.NewError(http.StatusGone, MsgLinkInvalid)
	}
	if err := s.openTrackingInSession(ctx, link.ID); err != nil {
		return nil, s.internalError(ctx, "ouverture de l'accès par lien impossible", err)
	}
	s.Logger.InfoContext(ctx, "lien personnel ouvert", "action", "tracking_link_opened", "tracking_link_id", link.ID)
	return nil, nil
}

func (s *server) closeTracking(ctx context.Context, _ *struct{}) (*struct{}, error) {
	s.Sessions.Remove(ctx, trackingLinkSessionKey)
	return nil, nil
}

// issueTrackingLink crée un lien personnel pour l'adresse, valable
// public.LinkTTL, et renvoie le jeton brut à placer dans l'e-mail
// (/suivi#jeton=…). À appeler dans la transaction qui met l'e-mail en file.
func (s *server) issueTrackingLink(ctx context.Context, q *sqlc.Queries, email string) (raw string, err error) {
	raw, hash := auth.NewToken()
	now := s.now().UTC()
	id, err := q.CreateTrackingLink(ctx, sqlc.CreateTrackingLinkParams{
		Email:     auth.NormalizeEmail(email),
		TokenHash: hash,
		ExpiresAt: now.Add(public.LinkTTL).Format(timeFormat),
		CreatedAt: now.Format(timeFormat),
	})
	if err != nil {
		return "", err
	}
	s.Logger.InfoContext(ctx, "lien personnel émis", "action", "tracking_link_issued", "tracking_link_id", id)
	return raw, nil
}

// openTrackingInSession ouvre l'accès par lien dans la session, avec un
// nouveau jeton de session (fixation), sans toucher aux clés de compte.
func (s *server) openTrackingInSession(ctx context.Context, linkID int64) error {
	if err := s.Sessions.RenewToken(ctx); err != nil {
		return err
	}
	s.Sessions.Put(ctx, trackingLinkSessionKey, linkID)
	return nil
}

// revokeTrackingLinks révoque tous les liens personnels de l'adresse, par
// exemple à la création d'un compte pour elle (FR-013).
func (s *server) revokeTrackingLinks(ctx context.Context, q *sqlc.Queries, email string) (int64, error) {
	return q.RevokeTrackingLinksOfEmail(ctx, sqlc.RevokeTrackingLinksOfEmailParams{
		Email: auth.NormalizeEmail(email), Now: ns(s.timestamp()),
	})
}

// linkAccess accorde le niveau auth.Link quand la session porte un lien
// personnel valide (research.md 016 › R1, R9). Le lien est relu à chaque
// requête : une révocation ou une expiration prend effet aussitôt. La session
// de compte n'est jamais consultée.
func (s *server) linkAccess(api huma.API, ctx huma.Context, next func(huma.Context)) {
	reqCtx := ctx.Context()
	tooMany := func() {
		ctx.SetHeader("Retry-After", strconv.Itoa(60))
		_ = huma.WriteErr(api, ctx, http.StatusTooManyRequests, MsgTooManyAttempts)
	}
	if !s.Limits.TrackingUseIP.Allow(clientIP(reqCtx)) {
		tooMany()
		return
	}
	id := s.Sessions.GetInt64(reqCtx, trackingLinkSessionKey)
	if id == 0 {
		_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, MsgLinkInvalid)
		return
	}
	now := s.now().UTC().Format(timeFormat)
	link, err := s.Store.Read.GetActiveTrackingLink(reqCtx, sqlc.GetActiveTrackingLinkParams{ID: id, Now: now})
	if errors.Is(err, sql.ErrNoRows) {
		_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, MsgLinkInvalid)
		return
	}
	if err != nil {
		s.Logger.ErrorContext(reqCtx, "lecture du lien personnel impossible", "error", err)
		_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "Une erreur inattendue s'est produite.")
		return
	}
	if !s.Limits.TrackingUseEmail.Allow(link.Email) {
		tooMany()
		return
	}
	if !link.LastUsedAt.Valid || s.now().Sub(parseTime(link.LastUsedAt.String)) >= linkTouchEvery {
		// Simple trace d'usage : un échec n'empêche pas la requête.
		if err := s.Store.WriteTx(reqCtx, func(q *sqlc.Queries) error {
			return q.TouchTrackingLink(reqCtx, sqlc.TouchTrackingLinkParams{ID: link.ID, Now: ns(now)})
		}); err != nil {
			s.Logger.WarnContext(reqCtx, "mise à jour de l'usage du lien impossible", "tracking_link_id", link.ID, "error", err)
		}
	}
	next(huma.WithValue(ctx, trackedEmailKey, link.Email))
}
