package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/customfield"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/season"
	"github.com/balsakup/frigg/internal/textfold"
)

// Messages des opérations sur les définitions de champs.
const (
	MsgCustomFieldGone     = "Ce champ n'existe plus."
	MsgCustomFieldsFull    = "30 champs actifs au plus : archivez un champ inutile."
	MsgCustomFieldTypeUsed = "Le type ne peut plus changer : des valeurs sont enregistrées."
	MsgCustomFieldOrder    = "L'ordre a changé entre-temps : rechargez la page."
	MsgCustomFieldArchived = "Ce champ est déjà archivé."
	MsgCustomFieldActive   = "Ce champ est déjà actif."
	MsgChoiceInUse         = "Le choix « %s » est utilisé : renommez-le plutôt que de le retirer."
)

// FieldDefinition est un champ personnalisé vu par l'administration.
type FieldDefinition struct {
	ID           int64    `json:"id"`
	Scope        string   `json:"scope" enum:"member,membership"`
	Label        string   `json:"label"`
	Type         string   `json:"type" enum:"short_text,long_text,number,date,bool,choice"`
	Visibility   string   `json:"visibility" enum:"admin,visible,editable"`
	Required     bool     `json:"required"`
	Help         *string  `json:"help"`
	Choices      []Choice `json:"choices"`
	Archived     bool     `json:"archived"`
	ValueCount   int64    `json:"valueCount" doc:"Nombre de valeurs enregistrées"`
	MissingCount *int64   `json:"missingCount" doc:"Champ obligatoire actif : fiches, ou adhésions en attente ou validées de la saison en cours, sans valeur ; null sinon"`
}

// CustomFieldList regroupe les champs par portée : actifs par position, puis
// archivés par libellé.
type CustomFieldList struct {
	Member     []FieldDefinition `json:"member"`
	Membership []FieldDefinition `json:"membership"`
}

type CustomFieldListOutput struct{ Body CustomFieldList }

type CustomFieldOutput struct{ Body FieldDefinition }

// MyCustomFieldList est la liste des champs d'un formulaire de l'espace membre.
type MyCustomFieldList struct {
	Member     []FormField `json:"member"`
	Membership []FormField `json:"membership"`
}

type MyCustomFieldListOutput struct{ Body MyCustomFieldList }

// CustomFieldBody est une définition saisie.
type CustomFieldBody struct {
	Label      string  `json:"label" maxLength:"200"`
	Type       string  `json:"type" enum:"short_text,long_text,number,date,bool,choice"`
	Visibility string  `json:"visibility" enum:"admin,visible,editable"`
	Required   bool    `json:"required"`
	Help       *string `json:"help,omitempty" required:"false" maxLength:"400"`
}

// CreateCustomFieldBody ajoute la portée et les libellés des choix.
type CreateCustomFieldBody struct {
	CustomFieldBody
	Scope   string   `json:"scope" enum:"member,membership"`
	Choices []string `json:"choices,omitempty" required:"false" maxItems:"60" doc:"Libellés des choix, dans l'ordre (liste de choix seulement)"`
}

// ChoiceBody est un choix saisi ; sans id, un nouveau choix.
type ChoiceBody struct {
	ID    *int64 `json:"id,omitempty" required:"false"`
	Label string `json:"label" maxLength:"200"`
}

// UpdateCustomFieldBody est la définition complète d'un champ existant.
type UpdateCustomFieldBody struct {
	CustomFieldBody
	Choices []ChoiceBody `json:"choices,omitempty" required:"false" maxItems:"60" doc:"Choix existants (avec id) et nouveaux ; un choix absent est retiré"`
}

type CreateCustomFieldInput struct{ Body CreateCustomFieldBody }

type CustomFieldIDInput struct {
	ID int64 `path:"id" doc:"Identifiant du champ"`
}

type UpdateCustomFieldInput struct {
	ID   int64 `path:"id" doc:"Identifiant du champ"`
	Body UpdateCustomFieldBody
}

// ReorderCustomFieldsBody est l'ordre complet des champs actifs d'une portée.
type ReorderCustomFieldsBody struct {
	Scope string  `json:"scope" enum:"member,membership"`
	IDs   []int64 `json:"ids" maxItems:"30"`
}

type ReorderCustomFieldsInput struct{ Body ReorderCustomFieldsBody }

// registerCustomFields enregistre les opérations des définitions de champs
// personnalisés (research.md 011 › R9, R10).
func (s *server) registerCustomFields(api huma.API) {
	op := func(id, method, path, summary string, status int) huma.Operation {
		return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{tagSettings}, DefaultStatus: status}
	}
	register(api, auth.Admin, op("list-custom-fields", http.MethodGet, "/admin/custom-fields", "Lister les champs personnalisés", 0), s.listCustomFields)
	register(api, auth.Admin, op("create-custom-field", http.MethodPost, "/admin/custom-fields", "Ajouter un champ personnalisé", http.StatusCreated), s.createCustomField)
	register(api, auth.Admin, op("reorder-custom-fields", http.MethodPut, "/admin/custom-fields/order", "Réordonner les champs personnalisés", 0), s.reorderCustomFields)
	register(api, auth.Admin, op("update-custom-field", http.MethodPut, "/admin/custom-fields/{id}", "Modifier un champ personnalisé", 0), s.updateCustomField)
	register(api, auth.Admin, op("archive-custom-field", http.MethodPost, "/admin/custom-fields/{id}/archive", "Archiver un champ personnalisé", 0), s.archiveCustomField)
	register(api, auth.Admin, op("restore-custom-field", http.MethodPost, "/admin/custom-fields/{id}/restore", "Rétablir un champ personnalisé", 0), s.restoreCustomField)
	register(api, auth.Admin, op("delete-custom-field", http.MethodDelete, "/admin/custom-fields/{id}", "Supprimer un champ personnalisé", http.StatusNoContent), s.deleteCustomField)
	register(api, auth.Member, huma.Operation{
		OperationID: "list-my-custom-fields",
		Method:      http.MethodGet,
		Path:        "/me/custom-fields",
		Summary:     "Champs personnalisés des formulaires de l'espace membre",
		Tags:        []string{tagMembers},
	}, s.listMyCustomFields)
}

// logCustomFieldAction trace une écriture réussie sur une définition, sans
// libellé ni valeur : un libellé peut être sensible (R11).
func (s *server) logCustomFieldAction(ctx context.Context, action string, fieldID int64, scope string) {
	s.Logger.InfoContext(ctx, "écriture sur un champ personnalisé",
		"action", action, "actor_id", CurrentAccount(ctx).ID, "field_id", fieldID, "scope", scope)
}

// currentSeasonID renvoie la saison qui contient la date du jour, s'il y en a une.
func (s *server) currentSeasonID(ctx context.Context, q *sqlc.Queries) (int64, bool, error) {
	rows, err := q.ListSeasons(ctx)
	if err != nil {
		return 0, false, err
	}
	seasons, err := toDomain(rows)
	if err != nil {
		return 0, false, err
	}
	today := s.today()
	for _, se := range seasons {
		if season.IsCurrent(se, today) {
			return se.ID, true, nil
		}
	}
	return 0, false, nil
}

// customFieldList construit la liste de l'administration avec ses compteurs.
func (s *server) customFieldList(ctx context.Context, q *sqlc.Queries) (CustomFieldList, error) {
	out := CustomFieldList{Member: []FieldDefinition{}, Membership: []FieldDefinition{}}
	rows, err := q.ListCustomFields(ctx)
	if err != nil {
		return out, err
	}
	choices, err := choicesByField(ctx, q)
	if err != nil {
		return out, err
	}
	counts, err := q.CountCustomFieldValues(ctx)
	if err != nil {
		return out, err
	}
	valueCount := map[int64]int64{}
	for _, c := range counts {
		valueCount[c.FieldID] = c.ValueCount
	}
	missing := map[int64]int64{}
	memberMissing, err := q.CountMembersMissingFields(ctx)
	if err != nil {
		return out, err
	}
	for _, m := range memberMissing {
		missing[m.FieldID] = m.MissingCount
	}
	seasonID, ok, err := s.currentSeasonID(ctx, q)
	if err != nil {
		return out, err
	}
	if ok {
		msMissing, err := q.CountMembershipsMissingFields(ctx, seasonID)
		if err != nil {
			return out, err
		}
		for _, m := range msMissing {
			missing[m.FieldID] = m.MissingCount
		}
	}

	var archived []FieldDefinition
	for _, r := range rows {
		fd := fieldDefinition(toDefinition(r, choices[r.ID]))
		fd.ValueCount = valueCount[r.ID]
		if n, ok := missing[r.ID]; ok {
			fd.MissingCount = &n
		}
		if fd.Archived {
			archived = append(archived, fd)
			continue
		}
		out.add(fd)
	}
	slices.SortStableFunc(archived, func(a, b FieldDefinition) int {
		return strings.Compare(textfold.Fold(a.Label), textfold.Fold(b.Label))
	})
	for _, fd := range archived {
		out.add(fd)
	}
	return out, nil
}

func (l *CustomFieldList) add(fd FieldDefinition) {
	if fd.Scope == string(customfield.ScopeMember) {
		l.Member = append(l.Member, fd)
	} else {
		l.Membership = append(l.Membership, fd)
	}
}

func fieldDefinition(d customfield.Definition) FieldDefinition {
	choices := make([]Choice, len(d.Choices))
	for i, c := range d.Choices {
		choices[i] = Choice{ID: c.ID, Label: c.Label}
	}
	return FieldDefinition{
		ID: d.ID, Scope: string(d.Scope), Label: d.Label, Type: string(d.Type),
		Visibility: string(d.Visibility), Required: d.Required, Help: d.Help,
		Choices: choices, Archived: d.Archived,
	}
}

func (s *server) listCustomFields(ctx context.Context, _ *struct{}) (*CustomFieldListOutput, error) {
	list, err := s.customFieldList(ctx, s.Store.Read)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", err)
	}
	return &CustomFieldListOutput{Body: list}, nil
}

// customFieldOutput relit un champ et ses compteurs après une écriture.
func (s *server) customFieldOutput(ctx context.Context, id int64) (*CustomFieldOutput, error) {
	list, err := s.customFieldList(ctx, s.Store.Read)
	if err != nil {
		return nil, s.internalError(ctx, "lecture du champ personnalisé impossible", err)
	}
	for _, fd := range slices.Concat(list.Member, list.Membership) {
		if fd.ID == id {
			return &CustomFieldOutput{Body: fd}, nil
		}
	}
	return nil, huma.Error404NotFound(MsgCustomFieldGone)
}

// scopeDefinitions charge tous les champs d'une portée, archivés compris,
// pour l'unicité des libellés.
func scopeDefinitions(ctx context.Context, q *sqlc.Queries, scope customfield.Scope) ([]customfield.Definition, error) {
	rows, err := q.ListCustomFieldsOfScope(ctx, string(scope))
	if err != nil {
		return nil, err
	}
	out := make([]customfield.Definition, len(rows))
	for i, r := range rows {
		out[i] = toDefinition(r, nil)
	}
	return out, nil
}

func definitionDetails(errs customfield.FieldErrors) error {
	details := make([]*huma.ErrorDetail, len(errs))
	for i, e := range errs {
		details[i] = &huma.ErrorDetail{Location: "body." + e.Field, Message: e.Message}
	}
	return FieldErrorDetails(details...)
}

func nullHelp(h *string) sql.NullString {
	if h == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *h, Valid: true}
}

func (s *server) createCustomField(ctx context.Context, in *CreateCustomFieldInput) (*CustomFieldOutput, error) {
	scope := customfield.Scope(in.Body.Scope)
	choices := make([]customfield.ChoiceInput, len(in.Body.Choices))
	for i, c := range in.Body.Choices {
		choices[i] = customfield.ChoiceInput{Label: c}
	}
	var id int64
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		existing, err := scopeDefinitions(ctx, q, scope)
		if err != nil {
			return err
		}
		def, errs := customfield.ValidateDefinition(customfield.DefinitionInput{
			Scope: scope, Label: in.Body.Label, Type: customfield.Type(in.Body.Type),
			Visibility: customfield.Visibility(in.Body.Visibility), Required: in.Body.Required,
			Help: in.Body.Help, Choices: choices,
		}, existing, 0)
		active, err := q.CountActiveCustomFields(ctx, string(scope))
		if err != nil {
			return err
		}
		if active >= customfield.MaxActive {
			errs = append(errs, customfield.FieldError{Field: "scope", Message: MsgCustomFieldsFull})
		}
		if errs != nil {
			return definitionDetails(errs)
		}
		pos, err := q.MaxCustomFieldPosition(ctx, string(scope))
		if err != nil {
			return err
		}
		row, err := q.CreateCustomField(ctx, sqlc.CreateCustomFieldParams{
			Scope: string(scope), Label: def.Label, Type: string(def.Type), Visibility: string(def.Visibility),
			Required: boolInt(def.Required), Help: nullHelp(def.Help), Position: pos + 1, Now: nowString(),
		})
		if err != nil {
			return err
		}
		id = row.ID
		for i, c := range def.Choices {
			if err := q.CreateCustomFieldChoice(ctx, sqlc.CreateCustomFieldChoiceParams{FieldID: id, Label: c.Label, Position: int64(i + 1)}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, s.txError(ctx, "création du champ personnalisé impossible", err)
	}
	s.logCustomFieldAction(ctx, "custom_field.create", id, string(scope))
	return s.customFieldOutput(ctx, id)
}

// fieldValueCount compte les valeurs enregistrées d'un champ.
func fieldValueCount(ctx context.Context, q *sqlc.Queries, id int64) (int64, error) {
	counts, err := q.CountCustomFieldValues(ctx)
	if err != nil {
		return 0, err
	}
	for _, c := range counts {
		if c.FieldID == id {
			return c.ValueCount, nil
		}
	}
	return 0, nil
}

func getCustomField(ctx context.Context, q *sqlc.Queries, id int64) (sqlc.CustomField, error) {
	row, err := q.GetCustomField(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return row, huma.Error404NotFound(MsgCustomFieldGone)
	}
	return row, err
}

func (s *server) updateCustomField(ctx context.Context, in *UpdateCustomFieldInput) (*CustomFieldOutput, error) {
	var scope string
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		row, err := getCustomField(ctx, q, in.ID)
		if err != nil {
			return err
		}
		scope = row.Scope
		if in.Body.Type != row.Type {
			n, err := fieldValueCount(ctx, q, in.ID)
			if err != nil {
				return err
			}
			if n > 0 {
				return huma.Error409Conflict(MsgCustomFieldTypeUsed)
			}
		}
		existing, err := scopeDefinitions(ctx, q, customfield.Scope(row.Scope))
		if err != nil {
			return err
		}
		current, err := q.ListCustomFieldChoices(ctx, in.ID)
		if err != nil {
			return err
		}
		known := map[int64]sqlc.CustomFieldChoice{}
		for _, c := range current {
			known[c.ID] = c
		}
		choices := make([]customfield.ChoiceInput, len(in.Body.Choices))
		var errs customfield.FieldErrors
		for i, c := range in.Body.Choices {
			choices[i] = customfield.ChoiceInput{Label: c.Label}
			if c.ID != nil {
				if _, ok := known[*c.ID]; !ok {
					errs = append(errs, customfield.FieldError{Field: choiceField(i), Message: customfield.MsgUnknownChoice})
					continue
				}
				choices[i].ID = *c.ID
			}
		}
		def, verrs := customfield.ValidateDefinition(customfield.DefinitionInput{
			Scope: customfield.Scope(row.Scope), Label: in.Body.Label, Type: customfield.Type(in.Body.Type),
			Visibility: customfield.Visibility(in.Body.Visibility), Required: in.Body.Required,
			Help: in.Body.Help, Choices: choices,
		}, existing, in.ID)
		errs = append(errs, verrs...)
		// Choix existants retirés : refusés s'ils sont utilisés (FR-006).
		kept := map[int64]bool{}
		for _, c := range def.Choices {
			if c.ID != 0 {
				kept[c.ID] = true
			}
		}
		var removed []int64
		for _, c := range current {
			if kept[c.ID] {
				continue
			}
			uses, err := q.CountChoiceUses(ctx, sql.NullInt64{Int64: c.ID, Valid: true})
			if err != nil {
				return err
			}
			if uses > 0 {
				errs = append(errs, customfield.FieldError{Field: "choices", Message: fmt.Sprintf(MsgChoiceInUse, c.Label)})
				continue
			}
			removed = append(removed, c.ID)
		}
		if errs != nil {
			return definitionDetails(errs)
		}
		if _, err := q.UpdateCustomField(ctx, sqlc.UpdateCustomFieldParams{
			Label: def.Label, Type: string(def.Type), Visibility: string(def.Visibility),
			Required: boolInt(def.Required), Help: nullHelp(def.Help), Now: nowString(), ID: in.ID,
		}); err != nil {
			return err
		}
		for _, id := range removed {
			if err := q.DeleteCustomFieldChoice(ctx, sqlc.DeleteCustomFieldChoiceParams{ID: id, FieldID: in.ID}); err != nil {
				return err
			}
		}
		for i, c := range def.Choices {
			pos := int64(i + 1)
			if c.ID != 0 {
				err = q.UpdateCustomFieldChoice(ctx, sqlc.UpdateCustomFieldChoiceParams{Label: c.Label, Position: pos, ID: c.ID, FieldID: in.ID})
			} else {
				err = q.CreateCustomFieldChoice(ctx, sqlc.CreateCustomFieldChoiceParams{FieldID: in.ID, Label: c.Label, Position: pos})
			}
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, s.txError(ctx, "modification du champ personnalisé impossible", err)
	}
	s.logCustomFieldAction(ctx, "custom_field.update", in.ID, scope)
	return s.customFieldOutput(ctx, in.ID)
}

func choiceField(i int) string { return "choices[" + strconv.Itoa(i) + "]" }

func (s *server) reorderCustomFields(ctx context.Context, in *ReorderCustomFieldsInput) (*CustomFieldListOutput, error) {
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		rows, err := q.ListActiveCustomFields(ctx, in.Body.Scope)
		if err != nil {
			return err
		}
		active := make([]int64, len(rows))
		for i, r := range rows {
			active[i] = r.ID
		}
		given := slices.Clone(in.Body.IDs)
		slices.Sort(active)
		slices.Sort(given)
		if !slices.Equal(active, given) {
			return huma.Error409Conflict(MsgCustomFieldOrder)
		}
		now := nowString()
		for i, id := range in.Body.IDs {
			if err := q.SetCustomFieldPosition(ctx, sqlc.SetCustomFieldPositionParams{Position: int64(i + 1), Now: now, ID: id}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, s.txError(ctx, "classement des champs personnalisés impossible", err)
	}
	s.logCustomFieldAction(ctx, "custom_field.reorder", 0, in.Body.Scope)
	return s.listCustomFields(ctx, nil)
}

func (s *server) archiveCustomField(ctx context.Context, in *CustomFieldIDInput) (*CustomFieldOutput, error) {
	var scope string
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		row, err := getCustomField(ctx, q, in.ID)
		if err != nil {
			return err
		}
		scope = row.Scope
		n, err := q.ArchiveCustomField(ctx, sqlc.ArchiveCustomFieldParams{Now: sql.NullString{String: nowString(), Valid: true}, ID: in.ID})
		if err != nil {
			return err
		}
		if n == 0 {
			return huma.Error409Conflict(MsgCustomFieldArchived)
		}
		return nil
	})
	if err != nil {
		return nil, s.txError(ctx, "archivage du champ personnalisé impossible", err)
	}
	s.logCustomFieldAction(ctx, "custom_field.archive", in.ID, scope)
	return s.customFieldOutput(ctx, in.ID)
}

func (s *server) restoreCustomField(ctx context.Context, in *CustomFieldIDInput) (*CustomFieldOutput, error) {
	var scope string
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		row, err := getCustomField(ctx, q, in.ID)
		if err != nil {
			return err
		}
		scope = row.Scope
		if !row.ArchivedAt.Valid {
			return huma.Error409Conflict(MsgCustomFieldActive)
		}
		active, err := q.CountActiveCustomFields(ctx, row.Scope)
		if err != nil {
			return err
		}
		if active >= customfield.MaxActive {
			return huma.Error422UnprocessableEntity(MsgCustomFieldsFull)
		}
		pos, err := q.MaxCustomFieldPosition(ctx, row.Scope)
		if err != nil {
			return err
		}
		_, err = q.RestoreCustomField(ctx, sqlc.RestoreCustomFieldParams{Position: pos + 1, Now: nowString(), ID: in.ID})
		return err
	})
	if err != nil {
		return nil, s.txError(ctx, "rétablissement du champ personnalisé impossible", err)
	}
	s.logCustomFieldAction(ctx, "custom_field.restore", in.ID, scope)
	return s.customFieldOutput(ctx, in.ID)
}

// deleteCustomField supprime un champ, ses choix et ses valeurs (US5). La
// contrainte différée de value_choice_id n'est vérifiée qu'à la validation de
// la transaction, quand valeurs et choix sont partis (data-model.md).
func (s *server) deleteCustomField(ctx context.Context, in *CustomFieldIDInput) (*struct{}, error) {
	var scope string
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		row, err := getCustomField(ctx, q, in.ID)
		if err != nil {
			return err
		}
		scope = row.Scope
		_, err = q.DeleteCustomField(ctx, in.ID)
		return err
	})
	if err != nil {
		return nil, s.txError(ctx, "suppression du champ personnalisé impossible", err)
	}
	s.logCustomFieldAction(ctx, "custom_field.delete", in.ID, scope)
	return nil, nil
}

func (s *server) listMyCustomFields(ctx context.Context, _ *struct{}) (*MyCustomFieldListOutput, error) {
	q := s.Store.Read
	members, err := loadDefinitions(ctx, q, customfield.ScopeMember, customfield.Owner)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", err)
	}
	memberships, err := loadDefinitions(ctx, q, customfield.ScopeMembership, customfield.Owner)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", err)
	}
	return &MyCustomFieldListOutput{Body: MyCustomFieldList{
		Member:     formFields(members, customfield.Owner),
		Membership: formFields(memberships, customfield.Owner),
	}}, nil
}
