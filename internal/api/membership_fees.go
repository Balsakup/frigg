package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/season"
)

const (
	tagFees = "Tarifs"

	MsgFeeGone       = "Ce tarif n'existe plus."
	MsgTooManyFees   = "Une saison ne peut pas compter plus de 20 tarifs."
	MsgFeeLabelEmpty = "Saisissez un libellé."
	MsgFeeLabelTaken = "Un tarif porte déjà ce nom dans cette saison."
	MsgAmountFormat  = "Saisissez un montant en euros, avec au plus deux décimales (par exemple 15,50)."
	MsgAmountTooHigh = "Le montant ne peut pas dépasser 10 000 €."

	MsgRenewalRequired = "Saisissez le montant du renouvellement."
)

// MembershipFee est un tarif de l'adhésion d'une saison.
type MembershipFee struct {
	ID                 int64     `json:"id"`
	Label              string    `json:"label"`
	AmountCents        int64     `json:"amountCents" doc:"Montant en centimes d'euro"`
	RenewalAmountCents *int64    `json:"renewalAmountCents" doc:"Montant de renouvellement en centimes ; null : même montant pour toute adhésion"`
	Description        *string   `json:"description"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// MembershipFeeActions liste les actions autorisées sur les tarifs d'une
// saison (research.md 004 › R5) : le frontend n'affiche que celles-ci.
type MembershipFeeActions struct {
	Add    bool `json:"add"`
	Edit   bool `json:"edit"`
	Delete bool `json:"delete"`
}

// MembershipFeeList est la liste des tarifs d'une saison.
type MembershipFeeList struct {
	Items   []MembershipFee      `json:"items" doc:"Du montant le plus élevé au plus faible, puis par libellé"`
	Actions MembershipFeeActions `json:"actions"`
}

// MembershipFeeBody est un tarif saisi ; le montant est analysé par le serveur
// (research.md 004 › R3).
type MembershipFeeBody struct {
	Label            string `json:"label" maxLength:"80"`
	Amount           string `json:"amount" doc:"Montant en euros, par exemple « 15,50 »"`
	HasRenewalAmount bool   `json:"hasRenewalAmount,omitempty" required:"false" doc:"Case « Tarif de renouvellement différent » ; absente : faux"`
	RenewalAmount    string `json:"renewalAmount,omitempty" required:"false" doc:"Montant du renouvellement en euros ; obligatoire si hasRenewalAmount, ignoré sinon"`
	Description      string `json:"description,omitempty" required:"false" maxLength:"500" doc:"Absente ou vide : aucune description"`
}

func toFee(r sqlc.MembershipFee) season.Fee {
	var d *string
	if r.Description.Valid {
		d = &r.Description.String
	}
	var renewal *int64
	if r.RenewalAmountCents.Valid {
		renewal = &r.RenewalAmountCents.Int64
	}
	return season.Fee{ID: r.ID, Label: r.Label, AmountCents: r.AmountCents, RenewalAmountCents: renewal, Description: d}
}

func toFees(rows []sqlc.MembershipFee) []season.Fee {
	out := make([]season.Fee, len(rows))
	for i, r := range rows {
		out[i] = toFee(r)
	}
	return out
}

func toFeeOutput(r sqlc.MembershipFee) MembershipFee {
	created, _ := time.Parse(timeFormat, r.CreatedAt)
	updated, _ := time.Parse(timeFormat, r.UpdatedAt)
	f := toFee(r)
	return MembershipFee{
		ID:                 r.ID,
		Label:              r.Label,
		AmountCents:        r.AmountCents,
		RenewalAmountCents: f.RenewalAmountCents,
		Description:        f.Description,
		CreatedAt:          created,
		UpdatedAt:          updated,
	}
}

func nullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func nullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// feeError traduit une erreur du domaine en réponse de l'API : toutes les
// erreurs de champ dans un seul 422 (FR-017).
func feeError(err error) error {
	var fes season.FieldErrors
	if errors.As(err, &fes) {
		details := make([]*huma.ErrorDetail, len(fes))
		for i, fe := range fes {
			msg := map[error]string{
				season.ErrLabelEmpty:      MsgFeeLabelEmpty,
				season.ErrLabelTaken:      MsgFeeLabelTaken,
				season.ErrAmountFormat:    MsgAmountFormat,
				season.ErrAmountTooHigh:   MsgAmountTooHigh,
				season.ErrRenewalRequired: MsgRenewalRequired,
			}[fe.Err]
			details[i] = &huma.ErrorDetail{Location: "body." + fe.Field, Message: msg}
		}
		return FieldErrorDetails(details...)
	}
	if errors.Is(err, season.ErrTooManyFees) {
		return huma.Error409Conflict(MsgTooManyFees)
	}
	return err
}

// logFeeAction trace une écriture réussie sur un tarif, sans son libellé
// (research.md 004 › R10).
func (s *server) logFeeAction(ctx context.Context, action string, seasonID, feeID int64, attrs ...any) {
	attrs = append([]any{"action", action, "actor_id", CurrentAccount(ctx).ID, "season_id", seasonID, "fee_id", feeID}, attrs...)
	s.Logger.InfoContext(ctx, "action d'administration sur un tarif", attrs...)
}

// seasonState renvoie l'état de la saison id dans la suite, ou une erreur 404.
func seasonState(rows []sqlc.Season, id int64, today season.Date) (season.State, error) {
	seasons, err := toDomain(rows)
	if err != nil {
		return "", err
	}
	i := indexOf(seasons, id)
	if i < 0 {
		return "", huma.Error404NotFound(MsgSeasonGone)
	}
	return season.StateOf(seasons[i], today), nil
}

// feeWrite exécute fn dans une transaction d'écriture sur les tarifs relus
// d'une saison qui n'est pas close (research.md 004 › R5).
func (s *server) feeWrite(ctx context.Context, seasonID int64, fn func(q *sqlc.Queries, fees []season.Fee) error) error {
	today := s.today()
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		rows, err := q.ListSeasons(ctx)
		if err != nil {
			return err
		}
		state, err := seasonState(rows, seasonID, today)
		if err != nil {
			return err
		}
		if state == season.Closed {
			return huma.Error409Conflict(MsgSeasonClosed)
		}
		feeRows, err := q.ListMembershipFees(ctx, seasonID)
		if err != nil {
			return err
		}
		return feeError(fn(q, toFees(feeRows)))
	})
	if err != nil {
		return s.txError(ctx, "écriture du tarif impossible", err)
	}
	return nil
}

func (s *server) registerMembershipFees(api huma.API) {
	register(api, auth.Admin, huma.Operation{
		OperationID: "list-membership-fees",
		Method:      http.MethodGet,
		Path:        "/admin/seasons/{id}/fees",
		Summary:     "Tarifs de l'adhésion d'une saison",
		Tags:        []string{tagFees},
	}, s.listMembershipFees)

	register(api, auth.Admin, huma.Operation{
		OperationID:   "create-membership-fee",
		Method:        http.MethodPost,
		Path:          "/admin/seasons/{id}/fees",
		Summary:       "Ajouter un tarif à une saison",
		Tags:          []string{tagFees},
		DefaultStatus: http.StatusCreated,
	}, s.createMembershipFee)

	register(api, auth.Admin, huma.Operation{
		OperationID: "update-membership-fee",
		Method:      http.MethodPut,
		Path:        "/admin/seasons/{id}/fees/{feeId}",
		Summary:     "Modifier un tarif",
		Tags:        []string{tagFees},
	}, s.updateMembershipFee)

	register(api, auth.Admin, huma.Operation{
		OperationID:   "delete-membership-fee",
		Method:        http.MethodDelete,
		Path:          "/admin/seasons/{id}/fees/{feeId}",
		Summary:       "Supprimer un tarif",
		Tags:          []string{tagFees},
		DefaultStatus: http.StatusNoContent,
	}, s.deleteMembershipFee)
}

type MembershipFeeListOutput struct {
	Body MembershipFeeList
}

func (s *server) listMembershipFees(ctx context.Context, in *SeasonIDInput) (*MembershipFeeListOutput, error) {
	rows, err := s.Store.Read.ListSeasons(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	state, err := seasonState(rows, in.ID, s.today())
	if err != nil {
		return nil, s.txError(ctx, "saisons illisibles", err)
	}
	feeRows, err := s.Store.Read.ListMembershipFees(ctx, in.ID)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des tarifs impossible", err)
	}
	byID := make(map[int64]sqlc.MembershipFee, len(feeRows))
	for _, r := range feeRows {
		byID[r.ID] = r
	}
	fees := toFees(feeRows)
	season.SortFees(fees)
	items := make([]MembershipFee, len(fees))
	for i, f := range fees {
		items[i] = toFeeOutput(byID[f.ID])
	}
	a := season.FeeActionsFor(state, len(fees))
	return &MembershipFeeListOutput{Body: MembershipFeeList{
		Items:   items,
		Actions: MembershipFeeActions{Add: a.Add, Edit: a.Edit, Delete: a.Delete},
	}}, nil
}

type MembershipFeeOutput struct {
	Body MembershipFee
}

type CreateMembershipFeeInput struct {
	ID   int64 `path:"id" doc:"Identifiant de la saison"`
	Body MembershipFeeBody
}

// draft renvoie le tarif saisi : description absente si le champ est vide,
// montant de renouvellement seulement si la case est cochée.
func (b MembershipFeeBody) draft() season.FeeDraft {
	d := season.FeeDraft{Label: b.Label, Amount: b.Amount}
	if b.HasRenewalAmount {
		d.RenewalAmount = &b.RenewalAmount
	}
	if b.Description != "" {
		d.Description = &b.Description
	}
	return d
}

func (s *server) createMembershipFee(ctx context.Context, in *CreateMembershipFeeInput) (*MembershipFeeOutput, error) {
	var row sqlc.MembershipFee
	err := s.feeWrite(ctx, in.ID, func(q *sqlc.Queries, fees []season.Fee) error {
		fee, err := season.ValidateFee(fees, 0, in.Body.draft())
		if err != nil {
			return err
		}
		row, err = q.CreateMembershipFee(ctx, sqlc.CreateMembershipFeeParams{
			SeasonID:           in.ID,
			Label:              fee.Label,
			AmountCents:        fee.AmountCents,
			RenewalAmountCents: nullInt64(fee.RenewalAmountCents),
			Description:        nullString(fee.Description),
			Now:                nowString(),
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	s.logFeeAction(ctx, "fee.create", in.ID, row.ID, "amount_cents", row.AmountCents)
	return &MembershipFeeOutput{Body: toFeeOutput(row)}, nil
}

type MembershipFeeIDInput struct {
	ID    int64 `path:"id" doc:"Identifiant de la saison"`
	FeeID int64 `path:"feeId" doc:"Identifiant du tarif"`
}

type UpdateMembershipFeeInput struct {
	MembershipFeeIDInput
	Body MembershipFeeBody
}

func (s *server) updateMembershipFee(ctx context.Context, in *UpdateMembershipFeeInput) (*MembershipFeeOutput, error) {
	var cents int64
	err := s.feeWrite(ctx, in.ID, func(q *sqlc.Queries, fees []season.Fee) error {
		if !slices.ContainsFunc(fees, func(f season.Fee) bool { return f.ID == in.FeeID }) {
			return huma.Error404NotFound(MsgFeeGone)
		}
		fee, err := season.ValidateFee(fees, in.FeeID, in.Body.draft())
		if err != nil {
			return err
		}
		cents = fee.AmountCents
		_, err = q.UpdateMembershipFee(ctx, sqlc.UpdateMembershipFeeParams{
			ID:                 in.FeeID,
			SeasonID:           in.ID,
			Label:              fee.Label,
			AmountCents:        fee.AmountCents,
			RenewalAmountCents: nullInt64(fee.RenewalAmountCents),
			Description:        nullString(fee.Description),
			Now:                nowString(),
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	s.logFeeAction(ctx, "fee.update", in.ID, in.FeeID, "amount_cents", cents)
	rows, err := s.Store.Read.ListMembershipFees(ctx, in.ID)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des tarifs impossible", err)
	}
	i := slices.IndexFunc(rows, func(r sqlc.MembershipFee) bool { return r.ID == in.FeeID })
	if i < 0 {
		return nil, huma.Error404NotFound(MsgFeeGone)
	}
	return &MembershipFeeOutput{Body: toFeeOutput(rows[i])}, nil
}

func (s *server) deleteMembershipFee(ctx context.Context, in *MembershipFeeIDInput) (*struct{}, error) {
	err := s.feeWrite(ctx, in.ID, func(q *sqlc.Queries, _ []season.Fee) error {
		n, err := q.DeleteMembershipFee(ctx, sqlc.DeleteMembershipFeeParams{ID: in.FeeID, SeasonID: in.ID})
		if err == nil && n == 0 {
			return huma.Error404NotFound(MsgFeeGone)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	s.logFeeAction(ctx, "fee.delete", in.ID, in.FeeID)
	return nil, nil
}
