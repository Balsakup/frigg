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
)

// Objets des jetons à usage unique (table one_time_tokens).
const (
	purposePasswordReset     = "password_reset"
	purposeEmailVerification = "email_verification"

	passwordResetLifetime     = time.Hour      // FR-009
	emailVerificationLifetime = 48 * time.Hour // FR-011

	MsgLinkGone = "Ce lien n'est plus valable."
)

// issueToken remplace les jetons inutilisés du même objet (seul le plus récent
// est valable) et renvoie le nouveau jeton brut, à envoyer par e-mail.
func issueToken(ctx context.Context, q *sqlc.Queries, accountID int64, purpose string, lifetime time.Duration) (string, error) {
	if err := q.DeleteUnusedTokens(ctx, sqlc.DeleteUnusedTokensParams{AccountID: accountID, Purpose: purpose}); err != nil {
		return "", err
	}
	raw, hash := auth.NewToken()
	now := time.Now().UTC()
	err := q.CreateToken(ctx, sqlc.CreateTokenParams{
		AccountID: accountID,
		Purpose:   purpose,
		TokenHash: hash,
		ExpiresAt: now.Add(lifetime).Format(timeFormat),
		CreatedAt: now.Format(timeFormat),
	})
	return raw, err
}

// checkToken vérifie le jeton sans le consommer ; 410 s'il est inconnu, expiré
// ou déjà utilisé.
func checkToken(ctx context.Context, q *sqlc.Queries, raw, purpose string) error {
	_, err := q.GetValidToken(ctx, sqlc.GetValidTokenParams{TokenHash: auth.HashToken(raw), Purpose: purpose, Now: nowString()})
	if errors.Is(err, sql.ErrNoRows) {
		return huma.NewError(http.StatusGone, MsgLinkGone)
	}
	return err
}

// consumeToken vérifie le jeton et le marque comme utilisé ; 410 s'il est
// inconnu, expiré ou déjà utilisé.
func consumeToken(ctx context.Context, q *sqlc.Queries, raw, purpose string) (sqlc.OneTimeToken, error) {
	now := nowString()
	tok, err := q.GetValidToken(ctx, sqlc.GetValidTokenParams{TokenHash: auth.HashToken(raw), Purpose: purpose, Now: now})
	if errors.Is(err, sql.ErrNoRows) {
		return tok, huma.NewError(http.StatusGone, MsgLinkGone)
	}
	if err != nil {
		return tok, err
	}
	return tok, q.MarkTokenUsed(ctx, sqlc.MarkTokenUsedParams{ID: tok.ID, UsedAt: sql.NullString{String: now, Valid: true}})
}
