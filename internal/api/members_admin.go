package api

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/customfield"
	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/member"
	"github.com/balsakup/frigg/internal/public"
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

	register(api, auth.Admin, huma.Operation{
		OperationID: "delete-inactive-members",
		Method:      http.MethodPost,
		Path:        "/admin/members/inactive/delete",
		Summary:     "Supprimer des fiches publiques inactives",
		Description: "Chaque fiche est revérifiée : une fiche redevenue active ou disparue est ignorée et comptée dans skipped (016 FR-041).",
		Tags:        []string{tagMembers},
	}, s.deleteInactiveMembers)

	register(api, auth.Admin, huma.Operation{
		OperationID:   "send-tracking-link",
		Method:        http.MethodPost,
		Path:          "/admin/members/{id}/tracking-link",
		Summary:       "Envoyer un lien personnel de suivi",
		Description:   "Fiche sans compte avec une adresse seulement (016 FR-040).",
		Tags:          []string{tagMembers},
		DefaultStatus: http.StatusNoContent,
	}, s.sendTrackingLink)
}

const (
	// MsgNotTrackable : seule une fiche sans compte avec une adresse reçoit
	// un lien personnel (016 FR-040).
	MsgNotTrackable = "Cette fiche a un compte ou n'a pas d'adresse."
	// MsgTooManyMails : plafond d'e-mails de l'adresse atteint (R9).
	MsgTooManyMails = "Trop d'envois vers cette adresse, réessayez plus tard."
)

// sendTrackingLink envoie à la demande d'un administrateur un lien personnel
// à l'adresse d'une fiche sans compte (FR-040). Le plafond d'e-mails de
// l'adresse est partagé avec les parcours publics (research.md 016 › R9).
func (s *server) sendTrackingLink(ctx context.Context, in *MemberIDInput) (*struct{}, error) {
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		row, err := q.GetMember(ctx, in.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgMemberGone)
		}
		if err != nil {
			return err
		}
		if row.AccountID.Valid || !row.Email.Valid || row.Email.String == "" {
			return huma.Error409Conflict(MsgNotTrackable)
		}
		email := auth.NormalizeEmail(row.Email.String)
		if !s.Limits.PublicMailEmail.Allow(email) {
			h := http.Header{}
			h.Set("Retry-After", strconv.Itoa(3600))
			return huma.ErrorWithHeaders(huma.Error429TooManyRequests(MsgTooManyMails), h)
		}
		raw, err := s.issueTrackingLink(ctx, q, email)
		if err != nil {
			return err
		}
		return s.enqueueNotice(ctx, q, "tracking-link", row.Email.String, mail.Data{}, "/suivi#jeton="+raw)
	})
	if err != nil {
		return nil, s.txError(ctx, "envoi du lien personnel impossible", err)
	}
	s.Logger.InfoContext(ctx, "lien personnel envoyé", "action", "tracking_link_sent",
		"actor_id", CurrentAccount(ctx).ID, "member_id", in.ID)
	return nil, nil
}

type ListMembersInput struct {
	Q        string   `query:"q" maxLength:"100" doc:"Fragments d'au moins 3 caractères (prénom, nom, adresse, téléphone), séparés par des espaces, tous requis ; ou fragment de la valeur d'un champ personnalisé de type texte court"`
	Field    []string `query:"field,explode" maxItems:"10" doc:"Filtre par champ personnalisé oui/non ou liste de choix : <champ>:true, <champ>:false, <champ>:empty ou <champ>:c<choix> ; répétable"`
	Account  string   `query:"account" pattern:"^(none|[0-9]+)$" doc:"none : fiches sans compte ; identifiant : fiches de ce compte"`
	Origin   string   `query:"origin" enum:"public" doc:"public : fiches créées par un parcours public (016 FR-037)"`
	Inactive bool     `query:"inactive" doc:"Vrai : fiches publiques inactives seulement (016 FR-041)"`
	Page     int64    `query:"page" minimum:"1" default:"1" doc:"Numéro de page"`
	PageSize int64    `query:"pageSize" minimum:"1" maximum:"100" default:"25" doc:"Nombre de fiches par page"`
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
	if in.Origin != "" {
		params.Origin = ns(in.Origin)
	}
	rows, err := s.Store.Read.ListMembers(ctx, params)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des membres impossible", err)
	}
	inactive, err := s.inactivePublicMembers(ctx, s.Store.Read)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des fiches inactives impossible", err)
	}
	byID := make(map[int64]memberRecord, len(rows))
	order := make([]member.Sortable, 0, len(rows))
	for _, r := range rows {
		if in.Inactive && !inactive[r.ID] {
			continue
		}
		rec := fromListRow(r)
		byID[r.ID] = rec
		order = append(order, sortable(rec.Member))
	}
	member.SortAdmin(order)

	total := int64(len(order))
	start := min((in.Page-1)*in.PageSize, total)
	end := min(start+in.PageSize, total)
	today := s.today()
	items := make([]AdminMember, 0, end-start)
	for _, o := range order[start:end] {
		m := toAdminMember(byID[o.ID], today)
		m.Inactive = inactive[o.ID]
		items = append(items, m)
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
	if out.Origin == "public" {
		inactive, err := s.inactivePublicMembers(ctx, s.Store.Read)
		if err != nil {
			return nil, s.internalError(ctx, "lecture des fiches inactives impossible", err)
		}
		out.Inactive = inactive[id]
	}
	if out.CustomFields, err = readMemberFields(ctx, s.Store.Read, id, customfield.Admin); err != nil {
		return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", err)
	}
	ms, err := s.memberMemberships(ctx, id)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des adhésions impossible", err)
	}
	out.Memberships = ms
	if out.PaidUpcomingRegistrations, err = s.paidUpcoming(ctx, id); err != nil {
		return nil, s.internalError(ctx, "lecture des inscriptions impossible", err)
	}
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
		// Une adresse modifiée n'est plus confirmée par la personne (016 R4).
		if !strings.EqualFold(current.Email.String, f.Email.String) {
			if err := q.ResetMemberEmailConfirmation(ctx, in.ID); err != nil {
				return err
			}
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

// inactivePublicMembers renvoie l'ensemble des fiches publiques inactives à
// la date du jour (research.md 016 › R14). Aucune n'est supprimée sans
// décision d'un administrateur (FR-041).
func (s *server) inactivePublicMembers(ctx context.Context, q *sqlc.Queries) (map[int64]bool, error) {
	cutoff := public.InactiveCutoff(s.today())
	ids, err := q.ListInactivePublicMembers(ctx, sqlc.ListInactivePublicMembersParams{
		CutoffAt: public.CutoffInstant(cutoff).UTC().Format(timeFormat),
		CutoffOn: cutoff.String(),
	})
	if err != nil {
		return nil, err
	}
	out := make(map[int64]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// DeleteInactiveMembersInput liste les fiches à supprimer.
type DeleteInactiveMembersInput struct {
	Body struct {
		IDs []int64 `json:"ids" minItems:"1" maxItems:"500" doc:"Fiches publiques inactives à supprimer"`
	}
}

// DeleteInactiveMembersResult compte les fiches supprimées et ignorées.
type DeleteInactiveMembersResult struct {
	Deleted int64 `json:"deleted" doc:"Fiches supprimées"`
	Skipped int64 `json:"skipped" doc:"Fiches ignorées : redevenues actives, déjà supprimées ou inconnues"`
}

type DeleteInactiveMembersOutput struct{ Body DeleteInactiveMembersResult }

// deleteInactiveMembers supprime, sur décision d'un administrateur, les
// fiches publiques inactives listées (FR-041). L'inactivité est revérifiée
// dans la transaction ; la suppression suit les règles de delete-member :
// adhésions, inscriptions et valeurs de champs partent avec la fiche (006
// FR-021, 012).
func (s *server) deleteInactiveMembers(ctx context.Context, in *DeleteInactiveMembersInput) (*DeleteInactiveMembersOutput, error) {
	var out DeleteInactiveMembersResult
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		out = DeleteInactiveMembersResult{}
		inactive, err := s.inactivePublicMembers(ctx, q)
		if err != nil {
			return err
		}
		seen := make(map[int64]bool, len(in.Body.IDs))
		for _, id := range in.Body.IDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			if !inactive[id] {
				out.Skipped++
				continue
			}
			n, err := q.DeleteMember(ctx, id)
			if err != nil {
				return err
			}
			out.Deleted += n
		}
		return nil
	})
	if err != nil {
		return nil, s.txError(ctx, "suppression des fiches inactives impossible", err)
	}
	s.Logger.InfoContext(ctx, "fiches publiques inactives supprimées", "action", "inactive_members_deleted",
		"actor_id", CurrentAccount(ctx).ID, "deleted", out.Deleted, "skipped", out.Skipped)
	return &DeleteInactiveMembersOutput{Body: out}, nil
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
