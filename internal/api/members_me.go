package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/member"
)

// Espace membre (« Mes membres »). Le compte vient uniquement de la session et
// chaque requête est conditionnée par account_id (research.md 005 › R4) : une
// fiche d'un autre compte est indiscernable d'une fiche inexistante.

func (s *server) registerMyMembers(api huma.API) {
	register(api, auth.Member, huma.Operation{
		OperationID: "list-my-members",
		Method:      http.MethodGet,
		Path:        "/me/members",
		Summary:     "Mes membres",
		Tags:        []string{tagMyMembers},
	}, s.listMyMembers)

	register(api, auth.Member, huma.Operation{
		OperationID: "get-my-member",
		Method:      http.MethodGet,
		Path:        "/me/members/{id}",
		Summary:     "Une fiche de mes membres",
		Tags:        []string{tagMyMembers},
	}, s.getMyMember)

	register(api, auth.Member, huma.Operation{
		OperationID:   "create-my-member",
		Method:        http.MethodPost,
		Path:          "/me/members",
		Summary:       "Ajouter un membre à mon compte",
		Tags:          []string{tagMyMembers},
		DefaultStatus: http.StatusCreated,
	}, s.createMyMember)

	register(api, auth.Member, huma.Operation{
		OperationID: "update-my-member",
		Method:      http.MethodPut,
		Path:        "/me/members/{id}",
		Summary:     "Modifier une fiche de mes membres",
		Tags:        []string{tagMyMembers},
	}, s.updateMyMember)

	register(api, auth.Member, huma.Operation{
		OperationID:   "delete-my-member",
		Method:        http.MethodDelete,
		Path:          "/me/members/{id}",
		Summary:       "Supprimer une fiche de mes membres",
		Tags:          []string{tagMyMembers},
		DefaultStatus: http.StatusNoContent,
	}, s.deleteMyMember)
}

// MyMemberActions liste les actions possibles dans « Mes membres » (R10).
type MyMemberActions struct {
	Add      bool `json:"add" doc:"Moins de 20 fiches"`
	MarkSelf bool `json:"markSelf" doc:"Aucune fiche ne désigne encore le titulaire"`
}

// MyMemberList est la liste des fiches du compte connecté.
type MyMemberList struct {
	Items   []Member        `json:"items" doc:"La fiche « Vous » d'abord, puis par prénom et nom"`
	Actions MyMemberActions `json:"actions"`
}

// MyMemberBody est une fiche saisie dans l'espace membre.
type MyMemberBody struct {
	MemberBody
	IsSelf bool `json:"isSelf,omitempty" required:"false" doc:"Case « Il s'agit de moi »"`
}

type MyMemberListOutput struct {
	Body MyMemberList
}

type MemberOutput struct {
	Body Member
}

type MemberIDInput struct {
	ID int64 `path:"id" doc:"Identifiant de la fiche"`
}

type CreateMyMemberInput struct {
	Body MyMemberBody
}

type UpdateMyMemberInput struct {
	ID   int64 `path:"id" doc:"Identifiant de la fiche"`
	Body MyMemberBody
}

func ownAccount(ctx context.Context) sql.NullInt64 {
	return sql.NullInt64{Int64: CurrentAccount(ctx).ID, Valid: true}
}

func (s *server) listMyMembers(ctx context.Context, _ *struct{}) (*MyMemberListOutput, error) {
	rows, err := s.Store.Read.ListMembersOfAccount(ctx, ownAccount(ctx))
	if err != nil {
		return nil, s.internalError(ctx, "lecture des fiches impossible", err)
	}
	byID := make(map[int64]sqlc.Member, len(rows))
	order := make([]member.Sortable, len(rows))
	markSelf := true
	for i, r := range rows {
		byID[r.ID] = r
		order[i] = sortable(r)
		if r.IsSelf == 1 {
			markSelf = false
		}
	}
	member.SortMine(order)
	today := s.today()
	items := make([]Member, len(order))
	for i, o := range order {
		items[i] = toMember(byID[o.ID], today)
	}
	return &MyMemberListOutput{Body: MyMemberList{
		Items:   items,
		Actions: MyMemberActions{Add: len(rows) < member.MaxPerAccount, MarkSelf: markSelf},
	}}, nil
}

func (s *server) myMemberOutput(ctx context.Context, id int64) (*MemberOutput, error) {
	r, err := s.Store.Read.GetMemberOfAccount(ctx, sqlc.GetMemberOfAccountParams{ID: id, AccountID: ownAccount(ctx)})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgMemberGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de la fiche impossible", err)
	}
	return &MemberOutput{Body: toMember(r, s.today())}, nil
}

func (s *server) getMyMember(ctx context.Context, in *MemberIDInput) (*MemberOutput, error) {
	return s.myMemberOutput(ctx, in.ID)
}

// checkSelf refuse une seconde fiche « Vous » sur le compte (R10).
func checkSelf(ctx context.Context, q *sqlc.Queries, account sql.NullInt64, excludeID int64) error {
	rows, err := q.ListMembersOfAccount(ctx, account)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if r.IsSelf == 1 && r.ID != excludeID {
			return FieldErrors("body.isSelf", MsgSelfTaken)
		}
	}
	return nil
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func (s *server) createMyMember(ctx context.Context, in *CreateMyMemberInput) (*MemberOutput, error) {
	account := ownAccount(ctx)
	c, err := validateMember(in.Body.MemberBody, s.today())
	if err != nil {
		return nil, err
	}
	var row sqlc.Member
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		if in.Body.IsSelf {
			if err := checkSelf(ctx, q, account, 0); err != nil {
				return err
			}
		}
		if err := checkDuplicate(ctx, q, c, 0, in.Body.ConfirmDuplicate); err != nil {
			return err
		}
		if err := checkAccountCapacity(ctx, q, account.Int64, MsgMyMembersFull); err != nil {
			return err
		}
		f := fieldsOf(c)
		row, err = q.CreateMemberForAccount(ctx, sqlc.CreateMemberForAccountParams{
			AccountID: account, IsSelf: boolInt(in.Body.IsSelf),
			FirstName: f.FirstName, LastName: f.LastName, BirthDate: f.BirthDate,
			Email: f.Email, Phone: f.Phone, AddressLine1: f.AddressLine1, AddressLine2: f.AddressLine2,
			PostalCode: f.Postal, City: f.City, Now: nowString(),
		})
		return selfTaken(err)
	})
	if err != nil {
		return nil, s.txError(ctx, "création de la fiche impossible", err)
	}
	s.logMemberAction(ctx, "member.create", row.ID, account, "me")
	return &MemberOutput{Body: toMember(row, s.today())}, nil
}

func (s *server) updateMyMember(ctx context.Context, in *UpdateMyMemberInput) (*MemberOutput, error) {
	account := ownAccount(ctx)
	c, err := validateMember(in.Body.MemberBody, s.today())
	if err != nil {
		return nil, err
	}
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		current, err := q.GetMemberOfAccount(ctx, sqlc.GetMemberOfAccountParams{ID: in.ID, AccountID: account})
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgMemberGone)
		}
		if err != nil {
			return err
		}
		if in.Body.IsSelf {
			if err := checkSelf(ctx, q, account, in.ID); err != nil {
				return err
			}
		}
		if identityChanged(c, current) {
			if err := checkDuplicate(ctx, q, c, in.ID, in.Body.ConfirmDuplicate); err != nil {
				return err
			}
		}
		f := fieldsOf(c)
		n, err := q.UpdateMemberOfAccount(ctx, sqlc.UpdateMemberOfAccountParams{
			IsSelf:    boolInt(in.Body.IsSelf),
			FirstName: f.FirstName, LastName: f.LastName, BirthDate: f.BirthDate,
			Email: f.Email, Phone: f.Phone, AddressLine1: f.AddressLine1, AddressLine2: f.AddressLine2,
			PostalCode: f.Postal, City: f.City, Now: nowString(),
			ID: in.ID, AccountID: account,
		})
		if err == nil && n == 0 {
			return huma.Error404NotFound(MsgMemberGone)
		}
		return selfTaken(err)
	})
	if err != nil {
		return nil, s.txError(ctx, "modification de la fiche impossible", err)
	}
	s.logMemberAction(ctx, "member.update", in.ID, account, "me")
	return s.myMemberOutput(ctx, in.ID)
}

func (s *server) deleteMyMember(ctx context.Context, in *MemberIDInput) (*struct{}, error) {
	account := ownAccount(ctx)
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		if _, err := q.GetMemberOfAccount(ctx, sqlc.GetMemberOfAccountParams{ID: in.ID, AccountID: account}); errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgMemberGone)
		} else if err != nil {
			return err
		}
		// Une fiche dont l'adhésion est en cours reste (research.md 006 › R9).
		active, err := q.CountActiveMembershipsOfMember(ctx, in.ID)
		if err != nil {
			return err
		}
		if active > 0 {
			return huma.Error409Conflict(MsgMemberHasMembership)
		}
		n, err := q.DeleteMemberOfAccount(ctx, sqlc.DeleteMemberOfAccountParams{ID: in.ID, AccountID: account})
		if err == nil && n == 0 {
			return huma.Error404NotFound(MsgMemberGone)
		}
		return err
	})
	if err != nil {
		return nil, s.txError(ctx, "suppression de la fiche impossible", err)
	}
	s.logMemberAction(ctx, "member.delete", in.ID, account, "me")
	return nil, nil
}
