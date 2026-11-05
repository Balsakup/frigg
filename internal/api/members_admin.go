package api

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/customfield"
	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/member"
)

// Administration (« Membres ») : toutes les fiches, rattachement compris.

func (s *server) registerAdminMembers(api huma.API) {
	register(api, auth.Admin, huma.Operation{
		OperationID: "list-members",
		Method:      http.MethodGet,
		Path:        "/admin/members",
		Summary:     "Lister les membres",
		Tags:        []string{tagMembers},
	}, s.listMembers)

	register(api, auth.Admin, huma.Operation{
		OperationID: "get-member",
		Method:      http.MethodGet,
		Path:        "/admin/members/{id}",
		Summary:     "Fiche d'un membre",
		Tags:        []string{tagMembers},
	}, s.getMember)

	register(api, auth.Admin, huma.Operation{
		OperationID:   "create-member",
		Method:        http.MethodPost,
		Path:          "/admin/members",
		Summary:       "Créer un membre",
		Tags:          []string{tagMembers},
		DefaultStatus: http.StatusCreated,
	}, s.createMember)

	register(api, auth.Admin, huma.Operation{
		OperationID: "update-member",
		Method:      http.MethodPut,
		Path:        "/admin/members/{id}",
		Summary:     "Modifier un membre",
		Tags:        []string{tagMembers},
	}, s.updateMember)

	register(api, auth.Admin, huma.Operation{
		OperationID:   "delete-member",
		Method:        http.MethodDelete,
		Path:          "/admin/members/{id}",
		Summary:       "Supprimer un membre",
		Tags:          []string{tagMembers},
		DefaultStatus: http.StatusNoContent,
	}, s.deleteMember)
}

type ListMembersInput struct {
	Q        string   `query:"q" maxLength:"100" doc:"Fragments d'au moins 3 caractères (prénom, nom, adresse, téléphone), séparés par des espaces, tous requis ; ou fragment de la valeur d'un champ personnalisé de type texte court"`
	Field    []string `query:"field,explode" maxItems:"10" doc:"Filtre par champ personnalisé oui/non ou liste de choix : <champ>:true, <champ>:false, <champ>:empty ou <champ>:c<choix> ; répétable"`
	Account  string `query:"account" pattern:"^(none|[0-9]+)$" doc:"none : fiches sans compte ; identifiant : fiches de ce compte"`
	Page     int64  `query:"page" minimum:"1" default:"1" doc:"Numéro de page"`
	PageSize int64  `query:"pageSize" minimum:"1" maximum:"100" default:"25" doc:"Nombre de fiches par page"`
}

// AdminMemberPage est une page de la liste des membres.
type AdminMemberPage struct {
	Items    []AdminMember `json:"items" doc:"Par nom, puis prénom (sans casse ni accents)"`
	Total    int64         `json:"total"`
	Page     int64         `json:"page"`
	PageSize int64         `json:"pageSize"`
}

type AdminMemberPageOutput struct {
	Body AdminMemberPage
}

type AdminMemberOutput struct {
	Body AdminMember
}

// AdminMemberBody est une fiche saisie par l'administration.
type AdminMemberBody struct {
	MemberBody
	AccountID *int64 `json:"accountId" nullable:"true" required:"true" doc:"Compte qui gère la fiche ; null : sans compte"`
}

// AdminMemberUpdateBody ajoute le rattachement affiché à l'ouverture de la
// fiche, pour refuser d'écraser celui choisi entre-temps par un autre
// administrateur (research.md 005 › R5).
type AdminMemberUpdateBody struct {
	AdminMemberBody
	ExpectedAccountID *int64 `json:"expectedAccountId" nullable:"true" required:"true" doc:"Rattachement affiché à l'ouverture de la fiche ; null : sans compte"`
}

type CreateMemberInput struct {
	Body AdminMemberBody
}

type UpdateMemberInput struct {
	ID   int64 `path:"id" doc:"Identifiant de la fiche"`
	Body AdminMemberUpdateBody
}

// listMembers charge toutes les fiches filtrées, les trie sans accents et
// découpe la page en Go (research.md 005 › R7).
func (s *server) listMembers(ctx context.Context, in *ListMembersInput) (*AdminMemberPageOutput, error) {
	var params sqlc.ListMembersParams
	if in.Q != "" {
		q, ok := database.FTSTrigramQuery(in.Q)
		if !ok {
			return nil, FieldErrors("query.q", MsgSearchTooShort)
		}
		params.Query = sql.NullString{String: q, Valid: true}
	}
	switch in.Account {
	case "":
	case "none":
		params.NoAccount = sql.NullInt64{Int64: 1, Valid: true}
	default:
		id, _ := strconv.ParseInt(in.Account, 10, 64)
		params.AccountID = sql.NullInt64{Int64: id, Valid: true}
	}
	filters, err := fieldFilters(ctx, s.Store.Read, customfield.ScopeMember, in.Field)
	if err != nil {
		return nil, s.txError(ctx, "lecture des champs personnalisés impossible", err)
	}
	params.FieldFilters = filters
	rows, err := s.Store.Read.ListMembers(ctx, params)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des membres impossible", err)
	}
	byID := make(map[int64]memberRecord, len(rows))
	order := make([]member.Sortable, len(rows))
	for i, r := range rows {
		rec := fromListRow(r)
		byID[r.ID] = rec
		order[i] = sortable(rec.Member)
	}
	member.SortAdmin(order)

	total := int64(len(order))
	start := min((in.Page-1)*in.PageSize, total)
	end := min(start+in.PageSize, total)
	today := s.today()
	items := make([]AdminMember, 0, end-start)
	for _, o := range order[start:end] {
		items = append(items, toAdminMember(byID[o.ID], today))
	}
	return &AdminMemberPageOutput{Body: AdminMemberPage{Items: items, Total: total, Page: in.Page, PageSize: in.PageSize}}, nil
}

func (s *server) memberOutput(ctx context.Context, id int64) (*AdminMemberOutput, error) {
	r, err := s.Store.Read.GetMember(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgMemberGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de la fiche impossible", err)
	}
	out := toAdminMember(fromGetRow(r), s.today())
	if out.CustomFields, err = readMemberFields(ctx, s.Store.Read, id, customfield.Admin); err != nil {
		return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", err)
	}
	ms, err := s.memberMemberships(ctx, id)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des adhésions impossible", err)
	}
	out.Memberships = ms
	return &AdminMemberOutput{Body: out}, nil
}

func (s *server) getMember(ctx context.Context, in *MemberIDInput) (*AdminMemberOutput, error) {
	return s.memberOutput(ctx, in.ID)
}

func nullID(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

// checkTargetAccount vérifie que le compte de rattachement existe.
func checkTargetAccount(ctx context.Context, q *sqlc.Queries, id *int64) error {
	if id == nil {
		return nil
	}
	_, err := q.GetAccountByID(ctx, *id)
	if errors.Is(err, sql.ErrNoRows) {
		return FieldErrors("body.accountId", MsgAccountGone)
	}
	return err
}

func (s *server) createMember(ctx context.Context, in *CreateMemberInput) (*AdminMemberOutput, error) {
	c, err := validateMember(in.Body.MemberBody, s.today())
	if err != nil {
		details, ferr := checkFields(ctx, s.Store.Read, customfield.ScopeMember, 0, in.Body.CustomFields, customfield.Admin)
		if ferr != nil {
			return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", ferr)
		}
		return nil, withFieldErrors(err, details)
	}
	account := nullID(in.Body.AccountID)
	var id int64
	var changed int
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		if err := checkTargetAccount(ctx, q, in.Body.AccountID); err != nil {
			return err
		}
		if err := checkDuplicate(ctx, q, c, 0, in.Body.ConfirmDuplicate); err != nil {
			return err
		}
		if account.Valid {
			if err := checkAccountCapacity(ctx, q, account.Int64, MsgAccountMembersFull); err != nil {
				return err
			}
		}
		f := fieldsOf(c)
		id, err = q.CreateMember(ctx, sqlc.CreateMemberParams{
			AccountID: account,
			FirstName: f.FirstName, LastName: f.LastName, BirthDate: f.BirthDate,
			Email: f.Email, Phone: f.Phone, AddressLine1: f.AddressLine1, AddressLine2: f.AddressLine2,
			PostalCode: f.Postal, City: f.City, Now: nowString(),
		})
		if err != nil {
			return err
		}
		changed, err = writeFields(ctx, q, customfield.ScopeMember, id, in.Body.CustomFields, customfield.Admin)
		return err
	})
	if err != nil {
		return nil, s.txError(ctx, "création de la fiche impossible", err)
	}
	s.logMemberAction(ctx, "member.create", id, account, "admin", "custom_fields", changed)
	if account.Valid {
		s.logMemberAction(ctx, "member.attach", id, account, "admin")
	}
	return s.memberOutput(ctx, id)
}

func (s *server) updateMember(ctx context.Context, in *UpdateMemberInput) (*AdminMemberOutput, error) {
	c, err := validateMember(in.Body.MemberBody, s.today())
	if err != nil {
		details, ferr := checkFields(ctx, s.Store.Read, customfield.ScopeMember, in.ID, in.Body.CustomFields, customfield.Admin)
		if ferr != nil {
			return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", ferr)
		}
		return nil, withFieldErrors(err, details)
	}
	account := nullID(in.Body.AccountID)
	attached := false
	changed := 0
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		row, err := q.GetMember(ctx, in.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgMemberGone)
		}
		if err != nil {
			return err
		}
		current := fromGetRow(row).Member
		// Rattachement modifié par un autre administrateur depuis l'affichage
		// de la fiche (research.md 005 › R5).
		if nullID(in.Body.ExpectedAccountID) != current.AccountID {
			return huma.Error409Conflict(MsgMemberAccountChanged)
		}
		if err := checkTargetAccount(ctx, q, in.Body.AccountID); err != nil {
			return err
		}
		if identityChanged(c, current) {
			if err := checkDuplicate(ctx, q, c, in.ID, in.Body.ConfirmDuplicate); err != nil {
				return err
			}
		}
		attached = account != current.AccountID
		isSelf := current.IsSelf
		if attached {
			isSelf = 0
			if account.Valid {
				if err := checkAccountCapacity(ctx, q, account.Int64, MsgAccountMembersFull); err != nil {
					return err
				}
			}
		}
		f := fieldsOf(c)
		_, err = q.UpdateMember(ctx, sqlc.UpdateMemberParams{
			AccountID: account, IsSelf: isSelf,
			FirstName: f.FirstName, LastName: f.LastName, BirthDate: f.BirthDate,
			Email: f.Email, Phone: f.Phone, AddressLine1: f.AddressLine1, AddressLine2: f.AddressLine2,
			PostalCode: f.Postal, City: f.City, Now: nowString(),
			ID: in.ID,
		})
		if err != nil {
			return err
		}
		changed, err = writeFields(ctx, q, customfield.ScopeMember, in.ID, in.Body.CustomFields, customfield.Admin)
		return err
	})
	if err != nil {
		return nil, s.txError(ctx, "modification de la fiche impossible", err)
	}
	s.logMemberAction(ctx, "member.update", in.ID, account, "admin", "custom_fields", changed)
	if attached {
		s.logMemberAction(ctx, "member.attach", in.ID, account, "admin")
	}
	return s.memberOutput(ctx, in.ID)
}

func (s *server) deleteMember(ctx context.Context, in *MemberIDInput) (*struct{}, error) {
	var account sql.NullInt64
	var memberships int
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		row, err := q.GetMember(ctx, in.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgMemberGone)
		}
		if err != nil {
			return err
		}
		account = row.AccountID
		// Les adhésions de la fiche sont supprimées avec elle (FR-021).
		ms, err := q.ListMembershipsOfMember(ctx, in.ID)
		if err != nil {
			return err
		}
		memberships = len(ms)
		_, err = q.DeleteMember(ctx, in.ID)
		return err
	})
	if err != nil {
		return nil, s.txError(ctx, "suppression de la fiche impossible", err)
	}
	s.logMemberAction(ctx, "member.delete", in.ID, account, "admin", "memberships_deleted", memberships)
	return nil, nil
}

// memberMemberships liste les adhésions d'une fiche, la saison la plus récente
// d'abord (FR-010).
func (s *server) memberMemberships(ctx context.Context, memberID int64) ([]MemberMembership, error) {
	q := s.Store.Read
	rows, err := q.ListMembershipsOfMember(ctx, memberID)
	if err != nil {
		return nil, err
	}
	seasons, order, err := seasonIndex(ctx, q, s.today())
	if err != nil {
		return nil, err
	}
	rank := make(map[int64]int, len(order))
	for i, id := range order {
		rank[id] = i
	}
	slices.SortFunc(rows, func(a, b sqlc.Membership) int {
		return cmp.Or(cmp.Compare(rank[b.SeasonID], rank[a.SeasonID]), cmp.Compare(b.ID, a.ID))
	})
	out := make([]MemberMembership, len(rows))
	for i, r := range rows {
		out[i] = MemberMembership{ID: r.ID, Season: seasons[r.SeasonID].ref, Status: r.Status, Payment: paymentOf(r)}
	}
	return out, nil
}
