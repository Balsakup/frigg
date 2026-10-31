package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// MsgBadCredentials est identique que l'adresse soit inconnue ou le mot de passe
// erroné (FR-003).
const MsgBadCredentials = "Adresse électronique ou mot de passe incorrect."

const tagAuth = "Authentification"

// registerAuth enregistre les opérations d'authentification (contracts/api.md).
func (s *server) registerAuth(api huma.API) {
	register(api, auth.Public, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/auth/login",
		Summary:     "Se connecter",
		Tags:        []string{tagAuth},
	}, s.login)

	register(api, auth.Public, huma.Operation{
		OperationID:   "request-password-reset",
		Method:        http.MethodPost,
		Path:          "/auth/password-reset",
		Summary:       "Demander un lien de réinitialisation",
		Description:   "Réponse identique que l'adresse corresponde à un compte ou non.",
		Tags:          []string{tagAuth},
		DefaultStatus: http.StatusAccepted,
	}, s.requestPasswordReset)

	register(api, auth.Public, huma.Operation{
		OperationID:   "confirm-password-reset",
		Method:        http.MethodPost,
		Path:          "/auth/password-reset/confirm",
		Summary:       "Choisir un nouveau mot de passe",
		Tags:          []string{tagAuth},
		DefaultStatus: http.StatusNoContent,
	}, s.confirmPasswordReset)

	register(api, auth.Public, huma.Operation{
		OperationID:   "confirm-email",
		Method:        http.MethodPost,
		Path:          "/auth/email-verification/confirm",
		Summary:       "Confirmer son adresse électronique",
		Description:   "Fonctionne sans être connecté.",
		Tags:          []string{tagAuth},
		DefaultStatus: http.StatusNoContent,
	}, s.confirmEmail)

	register(api, auth.Public, huma.Operation{
		OperationID: "get-invitation",
		Method:      http.MethodPost,
		Path:        "/auth/invitation/lookup",
		Summary:     "Consulter une invitation",
		Description: "POST pour garder le jeton hors de l'URL.",
		Tags:        []string{tagAuth},
	}, s.getInvitation)

	register(api, auth.Public, huma.Operation{
		OperationID: "accept-invitation",
		Method:      http.MethodPost,
		Path:        "/auth/invitation/accept",
		Summary:     "Accepter une invitation",
		Tags:        []string{tagAuth},
	}, s.acceptInvitation)

	register(api, auth.Member, huma.Operation{
		OperationID:   "logout",
		Method:        http.MethodPost,
		Path:          "/auth/logout",
		Summary:       "Se déconnecter",
		Tags:          []string{tagAuth},
		DefaultStatus: http.StatusNoContent,
	}, s.logout)
}

// LoginInput est le corps de la connexion.
type LoginInput struct {
	Body struct {
		Email    string `json:"email" maxLength:"254" doc:"Adresse électronique"`
		Password string `json:"password" maxLength:"1024" doc:"Mot de passe"`
	}
}

func (s *server) login(ctx context.Context, in *LoginInput) (*SessionOutput, error) {
	email := auth.NormalizeEmail(in.Body.Email)
	if !s.Limits.LoginIP.Allow(clientIP(ctx)) || !s.Limits.LoginEmail.Allow(email) {
		return nil, TooManyAttempts()
	}

	account, err := s.Store.Read.GetAccountByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		auth.VerifyAgainstDummy(in.Body.Password)
		return nil, huma.Error401Unauthorized(MsgBadCredentials)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture du compte impossible", err)
	}
	// Compte désactivé : même réponse que des identifiants incorrects, après la
	// vérification du mot de passe pour garder un temps constant (principe VII).
	if !auth.VerifyPassword(in.Body.Password, account.PasswordHash) || account.DisabledAt.Valid {
		return nil, huma.Error401Unauthorized(MsgBadCredentials)
	}
	return s.openSession(ctx, account)
}

// openSession ouvre une session pour le compte et renvoie sa description.
func (s *server) openSession(ctx context.Context, account sqlc.Account) (*SessionOutput, error) {
	if err := auth.Login(ctx, s.Sessions, account.ID, account.AuthEpoch); err != nil {
		return nil, s.internalError(ctx, "ouverture de session impossible", err)
	}
	if err := s.Store.Write.UpdateLastLogin(ctx, sqlc.UpdateLastLoginParams{
		ID:          account.ID,
		LastLoginAt: sql.NullString{String: nowString(), Valid: true},
	}); err != nil {
		return nil, s.internalError(ctx, "mise à jour de la dernière connexion impossible", err)
	}
	SetRequestAccount(ctx, account.ID)
	return s.sessionOutput(ctx, account)
}

func (s *server) logout(ctx context.Context, _ *struct{}) (*struct{}, error) {
	if err := auth.Logout(ctx, s.Sessions); err != nil {
		return nil, s.internalError(ctx, "fermeture de session impossible", err)
	}
	return nil, nil
}

// TokenInput porte un jeton reçu par e-mail.
type TokenInput struct {
	Body struct {
		Token string `json:"token" maxLength:"128" doc:"Jeton reçu par e-mail"`
	}
}

type InvitationPreviewOutput struct {
	Body struct {
		Email string `json:"email"`
		Role  string `json:"role" enum:"member,admin"`
	}
}

// validInvitation renvoie l'invitation en attente et non expirée du jeton.
func validInvitation(ctx context.Context, q *sqlc.Queries, token string) (sqlc.Invitation, error) {
	inv, err := q.GetInvitationByTokenHash(ctx, auth.HashToken(token))
	if errors.Is(err, sql.ErrNoRows) {
		return inv, huma.NewError(http.StatusGone, MsgInvitationGone)
	}
	if err != nil {
		return inv, err
	}
	if inv.Status != "pending" || inv.ExpiresAt <= nowString() {
		return inv, huma.NewError(http.StatusGone, MsgInvitationGone)
	}
	return inv, nil
}

func (s *server) getInvitation(ctx context.Context, in *TokenInput) (*InvitationPreviewOutput, error) {
	if !s.Limits.TokenIP.Allow(clientIP(ctx)) {
		return nil, TooManyAttempts()
	}
	inv, err := validInvitation(ctx, s.Store.Read, in.Body.Token)
	if err != nil {
		return nil, s.txError(ctx, "lecture de l'invitation impossible", err)
	}
	out := &InvitationPreviewOutput{}
	out.Body.Email, out.Body.Role = inv.Email, inv.Role
	return out, nil
}

// PasswordTokenInput porte un jeton et le mot de passe choisi.
type PasswordTokenInput struct {
	Body struct {
		Token    string `json:"token" maxLength:"128" doc:"Jeton reçu par e-mail"`
		Password string `json:"password" maxLength:"1024" doc:"Au moins 8 caractères, dont une majuscule et un chiffre"`
	}
}

func (s *server) acceptInvitation(ctx context.Context, in *PasswordTokenInput) (*SessionOutput, error) {
	if !s.Limits.TokenIP.Allow(clientIP(ctx)) {
		return nil, TooManyAttempts()
	}
	if msgs := auth.ValidatePassword(in.Body.Password); len(msgs) > 0 {
		return nil, FieldErrors("body.password", msgs...)
	}
	hash, err := auth.HashPassword(in.Body.Password)
	if err != nil {
		return nil, s.internalError(ctx, "hachage du mot de passe impossible", err)
	}

	var account sqlc.Account
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		inv, err := validInvitation(ctx, q, in.Body.Token)
		if err != nil {
			return err
		}
		if _, err := q.GetAccountByEmail(ctx, inv.Email); err == nil {
			return huma.Error409Conflict(MsgAccountExists)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		now := nowString()
		account, err = q.CreateAccount(ctx, sqlc.CreateAccountParams{
			Email:           inv.Email,
			PasswordHash:    hash,
			Role:            inv.Role,
			EmailVerifiedAt: sql.NullString{String: now, Valid: true},
			CreatedAt:       now,
		})
		if err != nil {
			return err
		}
		return q.MarkInvitationAccepted(ctx, sqlc.MarkInvitationAcceptedParams{ID: inv.ID, Now: now})
	})
	if err != nil {
		return nil, s.txError(ctx, "acceptation de l'invitation impossible", err)
	}
	return s.openSession(ctx, account)
}

type EmailInput struct {
	Body struct {
		Email string `json:"email" maxLength:"254" doc:"Adresse électronique"`
	}
}

func (s *server) requestPasswordReset(ctx context.Context, in *EmailInput) (*struct{}, error) {
	email := auth.NormalizeEmail(in.Body.Email)
	if !auth.ValidateEmail(email) {
		return nil, FieldErrors("body.email", MsgInvalidEmail)
	}
	if !s.Limits.ResetIP.Allow(clientIP(ctx)) || !s.Limits.ResetEmail.Allow(email) {
		return nil, TooManyAttempts()
	}
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		account, err := q.GetAccountByEmail(ctx, email)
		if errors.Is(err, sql.ErrNoRows) {
			return nil // même réponse qu'un compte existant (FR-003, principe VII)
		}
		if err != nil {
			return err
		}
		if account.DisabledAt.Valid {
			return nil // compte désactivé : même réponse qu'une adresse inconnue
		}
		raw, err := issueToken(ctx, q, account.ID, purposePasswordReset, passwordResetLifetime)
		if err != nil {
			return err
		}
		return s.enqueueMail(ctx, q, "password-reset", account.Email, "/reinitialiser-mot-de-passe", raw)
	})
	if err != nil {
		return nil, s.internalError(ctx, "demande de réinitialisation impossible", err)
	}
	return nil, nil
}

func (s *server) confirmPasswordReset(ctx context.Context, in *PasswordTokenInput) (*struct{}, error) {
	if !s.Limits.TokenIP.Allow(clientIP(ctx)) {
		return nil, TooManyAttempts()
	}
	if msgs := auth.ValidatePassword(in.Body.Password); len(msgs) > 0 {
		return nil, FieldErrors("body.password", msgs...)
	}
	hash, err := auth.HashPassword(in.Body.Password)
	if err != nil {
		return nil, s.internalError(ctx, "hachage du mot de passe impossible", err)
	}
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		tok, err := consumeToken(ctx, q, in.Body.Token, purposePasswordReset)
		if err != nil {
			return err
		}
		// L'époque incrémentée ferme toutes les sessions ouvertes (FR-010).
		_, err = q.UpdatePasswordAndBumpEpoch(ctx, sqlc.UpdatePasswordAndBumpEpochParams{ID: tok.AccountID, PasswordHash: hash})
		return err
	})
	if err != nil {
		return nil, s.txError(ctx, "réinitialisation du mot de passe impossible", err)
	}
	return nil, nil
}

func (s *server) confirmEmail(ctx context.Context, in *TokenInput) (*struct{}, error) {
	if !s.Limits.TokenIP.Allow(clientIP(ctx)) {
		return nil, TooManyAttempts()
	}
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		tok, err := consumeToken(ctx, q, in.Body.Token, purposeEmailVerification)
		if err != nil {
			return err
		}
		return q.MarkEmailVerified(ctx, sqlc.MarkEmailVerifiedParams{ID: tok.AccountID, VerifiedAt: sql.NullString{String: nowString(), Valid: true}})
	})
	if err != nil {
		return nil, s.txError(ctx, "confirmation de l'adresse impossible", err)
	}
	return nil, nil
}
