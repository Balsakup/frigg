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
	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

const (
	tagAccounts = "Comptes"

	MsgAccountGone     = "Ce compte n'existe plus."
	MsgSelfAction      = "Vous ne pouvez pas modifier votre propre compte depuis cette page."
	MsgLastAdmin       = "L'association doit conserver au moins un administrateur actif."
	MsgAccountDisabled = "Ce compte est désactivé. Réactivez-le avant d'envoyer un lien."
	MsgSearchTooShort  = "Saisissez au moins 3 caractères."
)

// AdminAccount est la représentation d'un compte pour les administrateurs.
type AdminAccount struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Role          string     `json:"role" enum:"member,admin"`
	Status        string     `json:"status" enum:"active,disabled"`
	EmailVerified bool       `json:"emailVerified"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastLoginAt   *time.Time `json:"lastLoginAt"`
	DisabledAt    *time.Time `json:"disabledAt"`
	IsSelf        bool       `json:"isSelf" doc:"Compte de l'administrateur connecté"`
}

func newAdminAccount(a sqlc.Account, currentID int64) AdminAccount {
	created, _ := time.Parse(timeFormat, a.CreatedAt)
	status := "active"
	if a.DisabledAt.Valid {
		status = "disabled"
	}
	return AdminAccount{
		ID:            a.ID,
		Email:         a.Email,
		Role:          a.Role,
		Status:        status,
		EmailVerified: a.EmailVerifiedAt.Valid,
		CreatedAt:     created,
		LastLoginAt:   optionalTime(a.LastLoginAt),
		DisabledAt:    optionalTime(a.DisabledAt),
		IsSelf:        a.ID == currentID,
	}
}

func optionalTime(v sql.NullString) *time.Time {
	if !v.Valid {
		return nil
	}
	t, err := time.Parse(timeFormat, v.String)
	if err != nil {
		return nil
	}
	return &t
}

func (s *server) registerAccounts(api huma.API) {
	register(api, auth.Admin, huma.Operation{
		OperationID: "list-accounts",
		Method:      http.MethodGet,
		Path:        "/admin/accounts",
		Summary:     "Comptes de l'association",
		Tags:        []string{tagAccounts},
	}, s.listAccounts)

	register(api, auth.Admin, huma.Operation{
		OperationID: "get-account",
		Method:      http.MethodGet,
		Path:        "/admin/accounts/{id}",
		Summary:     "Fiche d'un compte",
		Tags:        []string{tagAccounts},
	}, s.getAccount)

	register(api, auth.Admin, huma.Operation{
		OperationID: "disable-account",
		Method:      http.MethodPost,
		Path:        "/admin/accounts/{id}/disable",
		Summary:     "Désactiver un compte",
		Tags:        []string{tagAccounts},
	}, s.disableAccount)

	register(api, auth.Admin, huma.Operation{
		OperationID: "enable-account",
		Method:      http.MethodPost,
		Path:        "/admin/accounts/{id}/enable",
		Summary:     "Réactiver un compte",
		Tags:        []string{tagAccounts},
	}, s.enableAccount)

	register(api, auth.Admin, huma.Operation{
		OperationID: "set-account-role",
		Method:      http.MethodPut,
		Path:        "/admin/accounts/{id}/role",
		Summary:     "Changer le rôle d'un compte",
		Tags:        []string{tagAccounts},
	}, s.setAccountRole)

	register(api, auth.Admin, huma.Operation{
		OperationID:   "delete-account",
		Method:        http.MethodDelete,
		Path:          "/admin/accounts/{id}",
		Summary:       "Supprimer définitivement un compte",
		Tags:          []string{tagAccounts},
		DefaultStatus: http.StatusNoContent,
	}, s.deleteAccount)

	register(api, auth.Admin, huma.Operation{
		OperationID:   "send-account-password-reset",
		Method:        http.MethodPost,
		Path:          "/admin/accounts/{id}/password-reset",
		Summary:       "Envoyer un lien de réinitialisation du mot de passe",
		Tags:          []string{tagAccounts},
		DefaultStatus: http.StatusAccepted,
	}, s.sendAccountPasswordReset)
}

type ListAccountsInput struct {
	Q        string `query:"q" maxLength:"100" doc:"Fragments d'adresse d'au moins 3 caractères, séparés par des espaces, tous requis"`
	Role     string `query:"role" enum:"member,admin" doc:"Filtre sur le rôle"`
	Status   string `query:"status" enum:"active,disabled" doc:"Filtre sur l'état"`
	Page     int64  `query:"page" minimum:"1" default:"1" doc:"Numéro de page"`
	PageSize int64  `query:"pageSize" minimum:"1" maximum:"100" default:"25" doc:"Nombre de comptes par page"`
}

// AdminAccountPage est une page de la liste des comptes.
type AdminAccountPage struct {
	Items    []AdminAccount `json:"items"`
	Total    int64          `json:"total"`
	Page     int64          `json:"page"`
	PageSize int64          `json:"pageSize"`
}

type AdminAccountPageOutput struct {
	Body AdminAccountPage
}

func (s *server) listAccounts(ctx context.Context, in *ListAccountsInput) (*AdminAccountPageOutput, error) {
	var query, role sql.NullString
	var disabled sql.NullInt64
	if in.Q != "" {
		q, ok := database.FTSTrigramQuery(in.Q)
		if !ok {
			return nil, FieldErrors("query.q", MsgSearchTooShort)
		}
		query = sql.NullString{String: q, Valid: true}
	}
	if in.Role != "" {
		role = sql.NullString{String: in.Role, Valid: true}
	}
	switch in.Status {
	case "active":
		disabled = sql.NullInt64{Int64: 0, Valid: true}
	case "disabled":
		disabled = sql.NullInt64{Int64: 1, Valid: true}
	}

	rows, err := s.Store.Read.ListAccounts(ctx, sqlc.ListAccountsParams{
		Query:    query,
		Role:     role,
		Disabled: disabled,
		Limit:    in.PageSize,
		Offset:   (in.Page - 1) * in.PageSize,
	})
	if err != nil {
		return nil, s.internalError(ctx, "lecture des comptes impossible", err)
	}
	total, err := s.Store.Read.CountAccounts(ctx, sqlc.CountAccountsParams{Query: query, Role: role, Disabled: disabled})
	if err != nil {
		return nil, s.internalError(ctx, "comptage des comptes impossible", err)
	}

	current := CurrentAccount(ctx).ID
	out := &AdminAccountPageOutput{Body: AdminAccountPage{Total: total, Page: in.Page, PageSize: in.PageSize}}
	out.Body.Items = make([]AdminAccount, 0, len(rows))
	for _, r := range rows {
		out.Body.Items = append(out.Body.Items, newAdminAccount(sqlc.Account{
			ID:              r.ID,
			Email:           r.Email,
			Role:            r.Role,
			EmailVerifiedAt: r.EmailVerifiedAt,
			DisabledAt:      r.DisabledAt,
			CreatedAt:       r.CreatedAt,
			LastLoginAt:     r.LastLoginAt,
		}, current))
	}
	return out, nil
}

type AccountIDInput struct {
	ID int64 `path:"id" doc:"Identifiant du compte"`
}

type AdminAccountOutput struct {
	Body AdminAccount
}

func (s *server) getAccount(ctx context.Context, in *AccountIDInput) (*AdminAccountOutput, error) {
	return s.accountOutput(ctx, in.ID)
}

// accountOutput relit le compte après une écriture validée.
func (s *server) accountOutput(ctx context.Context, id int64) (*AdminAccountOutput, error) {
	a, err := s.Store.Read.GetAccountByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgAccountGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture du compte impossible", err)
	}
	return &AdminAccountOutput{Body: newAdminAccount(a, CurrentAccount(ctx).ID)}, nil
}

// adminWrite exécute fn dans une transaction d'écriture après les garde-fous
// communs aux actions sur un compte (research.md 002 › R5) : pas d'action sur
// soi-même (sauf allowSelf), acteur relu dans la transaction (il a pu être
// désactivé ou rétrogradé depuis le passage du middleware), cible existante.
// Les écritures sont sérialisées (une connexion, BEGIN IMMEDIATE) ; les
// requêtes qui peuvent retirer un administrateur actif ne s'appliquent que
// s'il en reste un autre, ce qui rend les actions concurrentes sûres (FR-014).
func (s *server) adminWrite(ctx context.Context, targetID int64, allowSelf bool, fn func(q *sqlc.Queries, target sqlc.Account) error) error {
	actorID := CurrentAccount(ctx).ID
	if !allowSelf && targetID == actorID {
		return huma.Error409Conflict(MsgSelfAction)
	}
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		actor, err := q.GetAccountByID(ctx, actorID)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && actor.DisabledAt.Valid) {
			return huma.Error401Unauthorized("Votre session a expiré.")
		}
		if err != nil {
			return err
		}
		if !auth.Allows(actor.Role, auth.Admin) {
			return huma.Error403Forbidden("Vous n'avez pas accès à cette ressource.")
		}
		target, err := q.GetAccountByID(ctx, targetID)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgAccountGone)
		}
		if err != nil {
			return err
		}
		return fn(q, target)
	})
	if err != nil {
		return s.txError(ctx, "action sur le compte impossible", err)
	}
	return nil
}

// logAccountAction trace une action d'administration réussie (FR-016). Jamais
// d'adresse électronique : après une suppression, les journaux ne doivent pas
// la conserver (research.md 002 › R10).
func (s *server) logAccountAction(ctx context.Context, action string, targetID int64, attrs ...any) {
	attrs = append([]any{"action", action, "actor_id", CurrentAccount(ctx).ID, "target_id", targetID}, attrs...)
	s.Logger.InfoContext(ctx, "action d'administration sur un compte", attrs...)
}

func (s *server) disableAccount(ctx context.Context, in *AccountIDInput) (*AdminAccountOutput, error) {
	changed := false
	err := s.adminWrite(ctx, in.ID, false, func(q *sqlc.Queries, target sqlc.Account) error {
		if target.DisabledAt.Valid {
			return nil
		}
		// L'époque incrémentée ferme toutes les sessions du compte (research.md 002 › R2).
		n, err := q.DisableAccount(ctx, sqlc.DisableAccountParams{ID: target.ID, Now: sql.NullString{String: nowString(), Valid: true}})
		if err != nil {
			return err
		}
		if n == 0 {
			return huma.Error409Conflict(MsgLastAdmin)
		}
		changed = true
		return q.DeleteAllUnusedTokens(ctx, target.ID)
	})
	if err != nil {
		return nil, err
	}
	if changed {
		s.logAccountAction(ctx, "account.disable", in.ID)
	}
	return s.accountOutput(ctx, in.ID)
}

func (s *server) enableAccount(ctx context.Context, in *AccountIDInput) (*AdminAccountOutput, error) {
	changed := false
	// Un compte désactivé n'a pas de session : la cible ne peut pas être soi-même.
	err := s.adminWrite(ctx, in.ID, true, func(q *sqlc.Queries, target sqlc.Account) error {
		n, err := q.EnableAccount(ctx, target.ID)
		changed = n > 0
		return err
	})
	if err != nil {
		return nil, err
	}
	if changed {
		s.logAccountAction(ctx, "account.enable", in.ID)
	}
	return s.accountOutput(ctx, in.ID)
}

type SetAccountRoleInput struct {
	ID   int64 `path:"id" doc:"Identifiant du compte"`
	Body struct {
		Role string `json:"role" enum:"member,admin" doc:"Nouveau rôle"`
	}
}

func (s *server) setAccountRole(ctx context.Context, in *SetAccountRoleInput) (*AdminAccountOutput, error) {
	role := in.Body.Role
	changed := false
	err := s.adminWrite(ctx, in.ID, false, func(q *sqlc.Queries, target sqlc.Account) error {
		if target.Role == role {
			return nil
		}
		n, err := q.SetAccountRole(ctx, sqlc.SetAccountRoleParams{ID: target.ID, Role: role})
		if err != nil {
			return err
		}
		if n == 0 {
			return huma.Error409Conflict(MsgLastAdmin)
		}
		changed = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	if changed {
		s.logAccountAction(ctx, "account.role_change", in.ID, "role", role)
	}
	return s.accountOutput(ctx, in.ID)
}

// DeleteAccountInput précise le sort des fiches gérées par le compte
// (research.md 005 › R9).
type DeleteAccountInput struct {
	AccountIDInput
	Members   string  `query:"members" enum:"delete,keep" doc:"Sort des fiches gérées par le compte : delete (effacées) ou keep (conservées sans compte) ; obligatoire s'il en gère"`
	MemberIDs []int64 `query:"memberIds" explode:"false" doc:"Identifiants des fiches affichées dans le dialogue de confirmation"`
}

const (
	MsgMembersChoiceRequired = "Choisissez le sort des fiches gérées par ce compte."
	MsgMembersChanged        = "La liste des fiches de ce compte a changé."
)

// sameIDs compare deux ensembles d'identifiants.
func sameIDs(a, b []int64) bool {
	set := make(map[int64]bool, len(a))
	for _, id := range a {
		set[id] = true
	}
	seen := make(map[int64]bool, len(b))
	for _, id := range b {
		if !set[id] {
			return false
		}
		seen[id] = true
	}
	return len(seen) == len(set)
}

func (s *server) deleteAccount(ctx context.Context, in *DeleteAccountInput) (*struct{}, error) {
	var membersAttr []any
	err := s.adminWrite(ctx, in.ID, false, func(q *sqlc.Queries, target sqlc.Account) error {
		// Fiches gérées par le compte : effacées ou détachées avant lui, dans la
		// même transaction (FR-012b).
		account := sql.NullInt64{Int64: target.ID, Valid: true}
		ids, err := q.ListMemberIDsOfAccount(ctx, account)
		if err != nil {
			return err
		}
		if len(ids) > 0 {
			if in.Members == "" {
				return FieldErrors("query.members", MsgMembersChoiceRequired)
			}
			if !sameIDs(ids, in.MemberIDs) {
				return huma.Error409Conflict(MsgMembersChanged)
			}
			if in.Members == "delete" {
				n, err := q.DeleteMembersOfAccount(ctx, account)
				if err != nil {
					return err
				}
				membersAttr = []any{"members_deleted", n}
			} else {
				n, err := q.DetachMembersOfAccount(ctx, sqlc.DetachMembersOfAccountParams{Now: nowString(), AccountID: account})
				if err != nil {
					return err
				}
				membersAttr = []any{"members_kept", n}
			}
		}
		// Jetons supprimés en cascade, invitations envoyées conservées sans auteur.
		n, err := q.DeleteAccount(ctx, target.ID)
		if err != nil {
			return err
		}
		if n == 0 {
			return huma.Error409Conflict(MsgLastAdmin)
		}
		// Droit à l'effacement : aucune copie de l'adresse ne reste en base
		// (research.md 002 › R6).
		if err := q.DeleteInvitationsByEmail(ctx, target.Email); err != nil {
			return err
		}
		return q.DeleteEmailsByRecipient(ctx, target.Email)
	})
	if err != nil {
		return nil, err
	}
	s.logAccountAction(ctx, "account.delete", in.ID, membersAttr...)
	return nil, nil
}

func (s *server) sendAccountPasswordReset(ctx context.Context, in *AccountIDInput) (*struct{}, error) {
	if !s.Limits.AdminReset.Allow(strconv.FormatInt(CurrentAccount(ctx).ID, 10)) {
		return nil, TooManyAttempts()
	}
	err := s.adminWrite(ctx, in.ID, false, func(q *sqlc.Queries, target sqlc.Account) error {
		if target.DisabledAt.Valid {
			return huma.Error409Conflict(MsgAccountDisabled)
		}
		raw, err := issueToken(ctx, q, target.ID, purposePasswordReset, passwordResetLifetime)
		if err != nil {
			return err
		}
		return s.enqueueMail(ctx, q, "password-reset", target.Email, "/reinitialiser-mot-de-passe", raw)
	})
	if err != nil {
		return nil, err
	}
	s.logAccountAction(ctx, "account.password_reset_sent", in.ID)
	return nil, nil
}
