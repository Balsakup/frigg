package api

import (
	"context"
	"database/sql"
	"errors"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/customfield"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Lecture et écriture des valeurs des champs personnalisés, partagées par les
// opérations des fiches et des adhésions (research.md 011 › R2, R3, R7).

// Choice est un choix d'un champ de type liste de choix.
type Choice struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
}

// FormField est la définition d'un champ utile à un formulaire, selon
// l'espace de l'opération.
type FormField struct {
	ID       int64    `json:"id"`
	Label    string   `json:"label"`
	Type     string   `json:"type" enum:"short_text,long_text,number,date,bool,choice"`
	Help     *string  `json:"help"`
	Required bool     `json:"required"`
	Editable bool     `json:"editable" doc:"Vrai : modifiable dans cet espace"`
	Choices  []Choice `json:"choices" doc:"Vide sauf pour une liste de choix"`
}

// FieldValue est un champ et sa valeur ; une seule propriété de valeur est
// renseignée, toutes sont null si le champ n'est pas renseigné (checked vaut
// alors false pour un oui/non).
type FieldValue struct {
	FormField
	Text     *string  `json:"text" doc:"Texte court ou long"`
	Number   *float64 `json:"number"`
	Date     *string  `json:"date" format:"date"`
	Checked  *bool    `json:"checked"`
	ChoiceID *int64   `json:"choiceId"`
}

// FieldInput est une valeur saisie : au plus une propriété de valeur. text
// porte un texte ou la saisie brute d'un nombre (« 12,5 ») ; aucune valeur ou
// un texte vide efface la valeur ; checked false efface un oui/non.
type FieldInput struct {
	FieldID  int64   `json:"fieldId"`
	Text     *string `json:"text,omitempty" required:"false" maxLength:"2000"`
	Date     *string `json:"date,omitempty" required:"false" maxLength:"10" doc:"AAAA-MM-JJ ; vide : efface la valeur"`
	Checked  *bool   `json:"checked,omitempty" required:"false"`
	ChoiceID *int64  `json:"choiceId,omitempty" required:"false"`
}

func toDefinition(r sqlc.CustomField, choices []customfield.Choice) customfield.Definition {
	d := customfield.Definition{
		ID:         r.ID,
		Scope:      customfield.Scope(r.Scope),
		Label:      r.Label,
		Type:       customfield.Type(r.Type),
		Visibility: customfield.Visibility(r.Visibility),
		Required:   r.Required == 1,
		Archived:   r.ArchivedAt.Valid,
		Choices:    choices,
	}
	if r.Help.Valid {
		d.Help = &r.Help.String
	}
	return d
}

// choicesByField regroupe tous les choix par champ.
func choicesByField(ctx context.Context, q *sqlc.Queries) (map[int64][]customfield.Choice, error) {
	rows, err := q.ListAllCustomFieldChoices(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64][]customfield.Choice{}
	for _, r := range rows {
		out[r.FieldID] = append(out[r.FieldID], customfield.Choice{ID: r.ID, Label: r.Label})
	}
	return out, nil
}

// loadDefinitions charge les champs actifs d'une portée lisibles par
// l'audience, dans l'ordre d'affichage. En audience Owner, les champs
// réservés sont écartés par la requête elle-même (FR-020).
func loadDefinitions(ctx context.Context, q *sqlc.Queries, scope customfield.Scope, a customfield.Audience) ([]customfield.Definition, error) {
	var rows []sqlc.CustomField
	var err error
	if a == customfield.Owner {
		rows, err = q.ListOwnerCustomFields(ctx, string(scope))
	} else {
		rows, err = q.ListActiveCustomFields(ctx, string(scope))
	}
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	choices, err := choicesByField(ctx, q)
	if err != nil {
		return nil, err
	}
	defs := make([]customfield.Definition, len(rows))
	for i, r := range rows {
		defs[i] = toDefinition(r, choices[r.ID])
	}
	return defs, nil
}

func valueOf(text, long sql.NullString, number sql.NullFloat64, date sql.NullString, b, choice sql.NullInt64) customfield.Value {
	var v customfield.Value
	switch {
	case text.Valid:
		v.Text = &text.String
	case long.Valid:
		v.Long = &long.String
	case number.Valid:
		v.Number = &number.Float64
	case date.Valid:
		v.Date = &date.String
	case b.Valid:
		t := b.Int64 == 1
		v.Bool = &t
	case choice.Valid:
		v.ChoiceID = &choice.Int64
	}
	return v
}

func memberValues(ctx context.Context, q *sqlc.Queries, memberID int64) (map[int64]customfield.Value, error) {
	rows, err := q.ListMemberFieldValues(ctx, memberID)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]customfield.Value, len(rows))
	for _, r := range rows {
		out[r.FieldID] = valueOf(r.ValueText, r.ValueLong, r.ValueNumber, r.ValueDate, r.ValueBool, r.ValueChoiceID)
	}
	return out, nil
}

func membershipValues(ctx context.Context, q *sqlc.Queries, membershipID int64) (map[int64]customfield.Value, error) {
	rows, err := q.ListMembershipFieldValues(ctx, membershipID)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]customfield.Value, len(rows))
	for _, r := range rows {
		out[r.FieldID] = valueOf(r.ValueText, r.ValueLong, r.ValueNumber, r.ValueDate, r.ValueBool, r.ValueChoiceID)
	}
	return out, nil
}

func formField(d customfield.Definition, editable bool) FormField {
	choices := make([]Choice, len(d.Choices))
	for i, c := range d.Choices {
		choices[i] = Choice{ID: c.ID, Label: c.Label}
	}
	return FormField{
		ID: d.ID, Label: d.Label, Type: string(d.Type), Help: d.Help,
		Required: d.Required, Editable: editable, Choices: choices,
	}
}

// formFields présente des définitions pour un formulaire de création.
func formFields(defs []customfield.Definition, a customfield.Audience) []FormField {
	out := make([]FormField, len(defs))
	for i, d := range defs {
		out[i] = formField(d, customfield.CanWrite(d, a))
	}
	return out
}

// fieldValues présente tous les champs de defs avec leur valeur. writable
// faux force editable à false (adhésion refusée ou annulée, FR-016).
func fieldValues(defs []customfield.Definition, current map[int64]customfield.Value, a customfield.Audience, writable bool) []FieldValue {
	out := make([]FieldValue, len(defs))
	for i, d := range defs {
		v := current[d.ID]
		fv := FieldValue{FormField: formField(d, writable && customfield.CanWrite(d, a))}
		switch {
		case v.Text != nil:
			fv.Text = v.Text
		case v.Long != nil:
			fv.Text = v.Long
		}
		fv.Number, fv.Date, fv.ChoiceID = v.Number, v.Date, v.ChoiceID
		if d.Type == customfield.TypeBool {
			checked := v.Bool != nil && *v.Bool
			fv.Checked = &checked
		}
		out[i] = fv
	}
	return out
}

// readMemberFields renvoie les champs de membre d'une fiche pour l'audience.
func readMemberFields(ctx context.Context, q *sqlc.Queries, memberID int64, a customfield.Audience) ([]FieldValue, error) {
	defs, err := loadDefinitions(ctx, q, customfield.ScopeMember, a)
	if err != nil || len(defs) == 0 {
		return []FieldValue{}, err
	}
	current, err := memberValues(ctx, q, memberID)
	if err != nil {
		return nil, err
	}
	return fieldValues(defs, current, a, true), nil
}

// readMembershipFields renvoie les champs d'adhésion pour l'audience.
func readMembershipFields(ctx context.Context, q *sqlc.Queries, membershipID int64, a customfield.Audience, writable bool) ([]FieldValue, error) {
	defs, err := loadDefinitions(ctx, q, customfield.ScopeMembership, a)
	if err != nil || len(defs) == 0 {
		return []FieldValue{}, err
	}
	current, err := membershipValues(ctx, q, membershipID)
	if err != nil {
		return nil, err
	}
	return fieldValues(defs, current, a, writable), nil
}

func toInputs(in []FieldInput) []customfield.Input {
	out := make([]customfield.Input, len(in))
	for i, f := range in {
		out[i] = customfield.Input{FieldID: f.FieldID, Text: f.Text, Date: f.Date, Checked: f.Checked, ChoiceID: f.ChoiceID}
	}
	return out
}

// customFieldDetails convertit les erreurs du domaine en détails 422.
func customFieldDetails(errs customfield.FieldErrors) []*huma.ErrorDetail {
	out := make([]*huma.ErrorDetail, len(errs))
	for i, e := range errs {
		out[i] = &huma.ErrorDetail{Location: "body." + e.Field, Message: e.Message}
	}
	return out
}

// planFields charge les définitions et les valeurs courantes puis calcule
// l'écriture ; les erreurs de saisie sont renvoyées en détails 422.
func planFields(ctx context.Context, q *sqlc.Queries, scope customfield.Scope, ownerID int64, inputs []FieldInput, a customfield.Audience) (customfield.Plan, []*huma.ErrorDetail, error) {
	defs, err := loadDefinitions(ctx, q, scope, a)
	if err != nil {
		return customfield.Plan{}, nil, err
	}
	current := map[int64]customfield.Value{}
	if ownerID != 0 {
		if scope == customfield.ScopeMember {
			current, err = memberValues(ctx, q, ownerID)
		} else {
			current, err = membershipValues(ctx, q, ownerID)
		}
		if err != nil {
			return customfield.Plan{}, nil, err
		}
	}
	plan, errs := customfield.Apply(defs, current, toInputs(inputs), a)
	if errs != nil {
		return customfield.Plan{}, customFieldDetails(errs), nil
	}
	return plan, nil, nil
}

// checkFields vérifie une saisie sans écrire ; ownerID vaut 0 pour une
// création. Sert à renvoyer en une fois les erreurs de la fiche ou de
// l'adhésion et celles des champs personnalisés.
func checkFields(ctx context.Context, q *sqlc.Queries, scope customfield.Scope, ownerID int64, inputs []FieldInput, a customfield.Audience) ([]*huma.ErrorDetail, error) {
	_, details, err := planFields(ctx, q, scope, ownerID, inputs, a)
	return details, err
}

// writeFields applique une saisie dans la transaction en cours et renvoie le
// nombre de valeurs modifiées ; une saisie invalide donne un 422.
func writeFields(ctx context.Context, q *sqlc.Queries, scope customfield.Scope, ownerID int64, inputs []FieldInput, a customfield.Audience) (int, error) {
	plan, details, err := planFields(ctx, q, scope, ownerID, inputs, a)
	if err != nil {
		return 0, err
	}
	if details != nil {
		return 0, FieldErrorDetails(details...)
	}
	now := nowString()
	for fieldID, v := range plan.Upserts {
		p := sqlc.UpsertMemberFieldValueParams{OwnerID: ownerID, FieldID: fieldID, Now: now}
		if v.Text != nil {
			p.ValueText = sql.NullString{String: *v.Text, Valid: true}
		}
		if v.Long != nil {
			p.ValueLong = sql.NullString{String: *v.Long, Valid: true}
		}
		if v.Number != nil {
			p.ValueNumber = sql.NullFloat64{Float64: *v.Number, Valid: true}
		}
		if v.Date != nil {
			p.ValueDate = sql.NullString{String: *v.Date, Valid: true}
		}
		if v.Bool != nil {
			p.ValueBool = sql.NullInt64{Int64: boolInt(*v.Bool), Valid: true}
		}
		if v.ChoiceID != nil {
			p.ValueChoiceID = sql.NullInt64{Int64: *v.ChoiceID, Valid: true}
		}
		if scope == customfield.ScopeMember {
			err = q.UpsertMemberFieldValue(ctx, p)
		} else {
			err = q.UpsertMembershipFieldValue(ctx, sqlc.UpsertMembershipFieldValueParams(p))
		}
		if err != nil {
			return 0, err
		}
	}
	for _, fieldID := range plan.Deletes {
		if scope == customfield.ScopeMember {
			err = q.DeleteMemberFieldValue(ctx, sqlc.DeleteMemberFieldValueParams{OwnerID: ownerID, FieldID: fieldID})
		} else {
			err = q.DeleteMembershipFieldValue(ctx, sqlc.DeleteMembershipFieldValueParams{OwnerID: ownerID, FieldID: fieldID})
		}
		if err != nil {
			return 0, err
		}
	}
	return plan.Changed(), nil
}

// fieldFilters lit les filtres de liste par champ personnalisé (FR-023) ;
// renvoie le JSON attendu par la requête, ou un 422 sur query.field.
func fieldFilters(ctx context.Context, q *sqlc.Queries, scope customfield.Scope, raw []string) (sql.NullString, error) {
	if len(raw) == 0 {
		return sql.NullString{}, nil
	}
	defs, err := loadDefinitions(ctx, q, scope, customfield.Admin)
	if err != nil {
		return sql.NullString{}, err
	}
	filters := make([]customfield.Filter, 0, len(raw))
	for _, r := range raw {
		f, err := customfield.ParseFilter(r, defs)
		if err != nil {
			return sql.NullString{}, FieldErrors("query.field", customfield.MsgUnknownFilter)
		}
		filters = append(filters, f)
	}
	return sql.NullString{String: customfield.FiltersJSON(filters), Valid: true}, nil
}

// withFieldErrors ajoute aux erreurs 422 d'une saisie (err) celles des
// champs personnalisés ; renvoie err inchangée si l'un ou l'autre est vide.
func withFieldErrors(err error, details []*huma.ErrorDetail) error {
	if len(details) == 0 {
		return err
	}
	var model *huma.ErrorModel
	if !errors.As(err, &model) || model.Status != 422 {
		return err
	}
	merged := make([]*huma.ErrorDetail, 0, len(model.Errors)+len(details))
	merged = append(merged, model.Errors...)
	merged = append(merged, details...)
	return FieldErrorDetails(merged...)
}
