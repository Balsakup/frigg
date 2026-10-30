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
)

const (
	tagInvitations     = "Invitations"
	invitationLifetime = 7 * 24 * time.Hour // FR-017

	MsgAccountExists     = "Un compte existe déjà avec cette adresse."
	MsgInvitationPending = "Une invitation est déjà en attente pour cette adresse."
	MsgInvitationGone    = "Cette invitation n'est plus valable."
	MsgInvalidEmail      = "L'adresse électronique n'est pas valide."
)

// Invitation est la représentation d'une invitation en attente ou expirée.
type Invitation struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role" enum:"member,admin"`
	Status    string    `json:"status" enum:"pending,expired"`
	ExpiresAt time.Time `json:"expiresAt"`
	InvitedBy *string   `json:"invitedBy"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s *server) registerInvitations(api huma.API) {
	register(api, auth.Admin, huma.Operation{
		OperationID: "list-invitations",
		Method:      http.MethodGet,
		Path:        "/admin/invitations",
		Summary:     "Invitations en attente",
		Tags:        []string{tagInvitations},
	}, s.listInvitations)

	register(api, auth.Admin, huma.Operation{
		OperationID:   "create-invitation",
		Method:        http.MethodPost,
		Path:          "/admin/invitations",
		Summary:       "Inviter une personne",
		Tags:          []string{tagInvitations},
		DefaultStatus: http.StatusCreated,
	}, s.createInvitation)

	register(api, auth.Admin, huma.Operation{
		OperationID: "resend-invitation",
		Method:      http.MethodPost,
		Path:        "/admin/invitations/{id}/resend",
		Summary:     "Renvoyer une invitation",
		Tags:        []string{tagInvitations},
	}, s.resendInvitation)

	register(api, auth.Admin, huma.Operation{
		OperationID:   "cancel-invitation",
		Method:        http.MethodDelete,
		Path:          "/admin/invitations/{id}",
		Summary:       "Annuler une invitation",
		Tags:          []string{tagInvitations},
		DefaultStatus: http.StatusNoContent,
	}, s.cancelInvitation)
}

type InvitationListOutput struct {
	Body struct {
		Items []Invitation `json:"items"`
	}
}

func (s *server) listInvitations(ctx context.Context, _ *struct{}) (*InvitationListOutput, error) {
	rows, err := s.Store.Read.ListPendingInvitations(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des invitations impossible", err)
	}
	out := &InvitationListOutput{}
	out.Body.Items = make([]Invitation, 0, len(rows))
	for _, r := range rows {
		var by *string
		if r.InvitedByEmail.Valid {
			by = &r.InvitedByEmail.String
		}
		out.Body.Items = append(out.Body.Items, newInvitation(r.ID, r.Email, r.Role, r.ExpiresAt, r.CreatedAt, by))
	}
	return out, nil
}

type CreateInvitationInput struct {
	Body struct {
		Email string `json:"email" maxLength:"254" doc:"Adresse électronique de la personne invitée"`
		Role  string `json:"role,omitempty" enum:"member,admin" default:"member" doc:"Rôle du futur compte"`
	}
}

type InvitationOutput struct {
	Body Invitation
}

func (s *server) createInvitation(ctx context.Context, in *CreateInvitationInput) (*InvitationOutput, error) {
	admin := CurrentAccount(ctx)
	if !s.Limits.InviteAdmin.Allow(strconv.FormatInt(admin.ID, 10)) {
		return nil, TooManyAttempts()
	}
	email := auth.NormalizeEmail(in.Body.Email)
	if !auth.ValidateEmail(email) {
		return nil, FieldErrors("body.email", MsgInvalidEmail)
	}
	role := in.Body.Role
	if role == "" {
		role = auth.RoleMember
	}

	raw, hash := auth.NewToken()
	now := time.Now().UTC()
	var inv sqlc.Invitation
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		if _, err := q.GetAccountByEmail(ctx, email); err == nil {
			return huma.Error409Conflict(MsgAccountExists)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err := q.GetPendingInvitationByEmail(ctx, email); err == nil {
			return huma.Error409Conflict(MsgInvitationPending)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var err error
		inv, err = q.CreateInvitation(ctx, sqlc.CreateInvitationParams{
			Email:     email,
			Role:      role,
			TokenHash: hash,
			InvitedBy: sql.NullInt64{Int64: admin.ID, Valid: true},
			ExpiresAt: now.Add(invitationLifetime).Format(timeFormat),
			Now:       now.Format(timeFormat),
		})
		if err != nil {
			return err
		}
		return s.enqueueMail(ctx, q, "invitation", email, "/invitation", raw)
	})
	if err != nil {
		return nil, s.txError(ctx, "création de l'invitation impossible", err)
	}
	return &InvitationOutput{Body: newInvitation(inv.ID, inv.Email, inv.Role, inv.ExpiresAt, inv.CreatedAt, &admin.Email)}, nil
}

type InvitationIDInput struct {
	ID int64 `path:"id" doc:"Identifiant de l'invitation"`
}

func (s *server) resendInvitation(ctx context.Context, in *InvitationIDInput) (*InvitationOutput, error) {
	raw, hash := auth.NewToken()
	now := time.Now().UTC()
	var inv sqlc.Invitation
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		var err error
		inv, err = q.ReplaceInvitationToken(ctx, sqlc.ReplaceInvitationTokenParams{
			ID:        in.ID,
			TokenHash: hash,
			ExpiresAt: now.Add(invitationLifetime).Format(timeFormat),
			Now:       now.Format(timeFormat),
		})
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound("Cette invitation n'existe pas ou n'est plus en attente.")
		}
		if err != nil {
			return err
		}
		return s.enqueueMail(ctx, q, "invitation", inv.Email, "/invitation", raw)
	})
	if err != nil {
		return nil, s.txError(ctx, "renvoi de l'invitation impossible", err)
	}
	by, err := s.inviterEmail(ctx, inv.InvitedBy)
	if err != nil {
		return nil, err
	}
	return &InvitationOutput{Body: newInvitation(inv.ID, inv.Email, inv.Role, inv.ExpiresAt, inv.CreatedAt, by)}, nil
}

func (s *server) cancelInvitation(ctx context.Context, in *InvitationIDInput) (*struct{}, error) {
	n, err := s.Store.Write.CancelInvitation(ctx, sqlc.CancelInvitationParams{ID: in.ID, Now: nowString()})
	if err != nil {
		return nil, s.internalError(ctx, "annulation de l'invitation impossible", err)
	}
	if n == 0 {
		return nil, huma.Error404NotFound("Cette invitation n'existe pas ou n'est plus en attente.")
	}
	return nil, nil
}

func (s *server) inviterEmail(ctx context.Context, id sql.NullInt64) (*string, error) {
	if !id.Valid {
		return nil, nil
	}
	a, err := s.Store.Read.GetAccountByID(ctx, id.Int64)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture du compte invitant impossible", err)
	}
	return &a.Email, nil
}

func newInvitation(id int64, email, role, expiresAt, createdAt string, invitedBy *string) Invitation {
	exp, _ := time.Parse(timeFormat, expiresAt)
	created, _ := time.Parse(timeFormat, createdAt)
	status := "pending"
	if !exp.After(time.Now()) {
		status = "expired"
	}
	return Invitation{ID: id, Email: email, Role: role, Status: status, ExpiresAt: exp, InvitedBy: invitedBy, CreatedAt: created}
}

// txError laisse passer les erreurs huma levées dans une transaction et
// transforme les autres en erreur interne.
func (s *server) txError(ctx context.Context, msg string, err error) error {
	var se huma.StatusError
	if errors.As(err, &se) {
		return err
	}
	return s.internalError(ctx, msg, err)
}
