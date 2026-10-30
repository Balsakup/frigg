package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

const tagMe = "Mon compte"

// registerMe enregistre les opérations sur le compte connecté.
func (s *server) registerMe(api huma.API) {
	register(api, auth.Member, huma.Operation{
		OperationID: "get-session",
		Method:      http.MethodGet,
		Path:        "/session",
		Summary:     "Session courante",
		Description: "Compte connecté et association de l'instance.",
		Tags:        []string{tagMe},
	}, s.getSession)

	register(api, auth.Member, huma.Operation{
		OperationID:   "resend-email-verification",
		Method:        http.MethodPost,
		Path:          "/me/email-verification",
		Summary:       "Renvoyer le lien de confirmation d'adresse",
		Tags:          []string{tagMe},
		DefaultStatus: http.StatusAccepted,
	}, s.resendEmailVerification)

	register(api, auth.Member, huma.Operation{
		OperationID:   "change-password",
		Method:        http.MethodPut,
		Path:          "/me/password",
		Summary:       "Changer son mot de passe",
		Description:   "Les autres sessions du compte sont fermées.",
		Tags:          []string{tagMe},
		DefaultStatus: http.StatusNoContent,
	}, s.changePassword)
}

const MsgWrongCurrentPassword = "Le mot de passe actuel est incorrect."

type ChangePasswordInput struct {
	Body struct {
		CurrentPassword string `json:"currentPassword" maxLength:"1024" doc:"Mot de passe actuel"`
		NewPassword     string `json:"newPassword" maxLength:"1024" doc:"Au moins 8 caractères, dont une majuscule et un chiffre"`
	}
}

func (s *server) changePassword(ctx context.Context, in *ChangePasswordInput) (*struct{}, error) {
	account := CurrentAccount(ctx)
	if !auth.VerifyPassword(in.Body.CurrentPassword, account.PasswordHash) {
		return nil, FieldErrors("body.currentPassword", MsgWrongCurrentPassword)
	}
	if msgs := auth.ValidatePassword(in.Body.NewPassword); len(msgs) > 0 {
		return nil, FieldErrors("body.newPassword", msgs...)
	}
	hash, err := auth.HashPassword(in.Body.NewPassword)
	if err != nil {
		return nil, s.internalError(ctx, "hachage du mot de passe impossible", err)
	}
	epoch, err := s.Store.Write.UpdatePasswordAndBumpEpoch(ctx, sqlc.UpdatePasswordAndBumpEpochParams{ID: account.ID, PasswordHash: hash})
	if err != nil {
		return nil, s.internalError(ctx, "changement du mot de passe impossible", err)
	}
	// La session courante suit la nouvelle époque ; les autres sont fermées (FR-010).
	auth.SetEpoch(ctx, s.Sessions, epoch)
	return nil, nil
}

const MsgAlreadyVerified = "Votre adresse électronique est déjà confirmée."

func (s *server) resendEmailVerification(ctx context.Context, _ *struct{}) (*struct{}, error) {
	account := CurrentAccount(ctx)
	if account.EmailVerifiedAt.Valid {
		return nil, huma.Error409Conflict(MsgAlreadyVerified)
	}
	if !s.Limits.VerifyIP.Allow(clientIP(ctx)) || !s.Limits.VerifyAccount.Allow(strconv.FormatInt(account.ID, 10)) {
		return nil, TooManyAttempts()
	}
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		raw, err := issueToken(ctx, q, account.ID, purposeEmailVerification, emailVerificationLifetime)
		if err != nil {
			return err
		}
		return s.enqueueMail(ctx, q, "email-verification", account.Email, "/confirmer-adresse", raw)
	})
	if err != nil {
		return nil, s.internalError(ctx, "envoi du lien de confirmation impossible", err)
	}
	return nil, nil
}

func (s *server) getSession(ctx context.Context, _ *struct{}) (*SessionOutput, error) {
	return s.sessionOutput(ctx, CurrentAccount(ctx))
}

func (s *server) sessionOutput(ctx context.Context, account sqlc.Account) (*SessionOutput, error) {
	name, err := s.associationName(ctx)
	if err != nil {
		return nil, err
	}
	return &SessionOutput{Body: Session{Account: toAccount(account), Association: Association{Name: name}}}, nil
}

// associationName renvoie le nom de l'association, ou « Mon association ».
func (s *server) associationName(ctx context.Context) (string, error) {
	name, err := s.Store.Read.GetAssociation(ctx)
	if err != nil {
		return "", s.internalError(ctx, "lecture de l'association impossible", err)
	}
	if !name.Valid {
		return defaultAssociationName, nil
	}
	return name.String, nil
}
