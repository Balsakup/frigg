package customfield

import (
	"errors"
	"maps"
	"strconv"
)

// Audience est le point de vue d'une opération : l'espace administration ou
// l'espace du compte qui gère la fiche (research.md › R3). Elle découle de
// l'espace de l'opération, jamais d'un test de rôle dans un handler.
type Audience int

const (
	Admin Audience = iota
	Owner
)

// CanRead indique si l'audience voit le champ.
func CanRead(d Definition, a Audience) bool {
	return a == Admin || d.Visibility == VisibilityVisible || d.Visibility == VisibilityEditable
}

// CanWrite indique si l'audience modifie le champ.
func CanWrite(d Definition, a Audience) bool {
	return a == Admin || d.Visibility == VisibilityEditable
}

// ErrorField renvoie le chemin d'erreur d'un champ (« customFields.f7 »).
func ErrorField(id int64) string { return "customFields.f" + strconv.FormatInt(id, 10) }

// Plan est le résultat d'une écriture : valeurs à enregistrer et à effacer.
type Plan struct {
	Upserts map[int64]Value
	Deletes []int64
}

// Changed est le nombre de valeurs modifiées.
func (p Plan) Changed() int { return len(p.Upserts) + len(p.Deletes) }

// Apply calcule l'écriture des valeurs saisies (R4). defs contient les champs
// actifs de la portée, déjà limités à ceux que l'audience peut lire ;
// current les valeurs enregistrées par identifiant de champ. Un champ absent
// d'inputs n'est pas modifié. Un champ obligatoire que l'audience peut
// remplir doit avoir une valeur finale.
func Apply(defs []Definition, current map[int64]Value, inputs []Input, a Audience) (Plan, FieldErrors) {
	byID := make(map[int64]Definition, len(defs))
	for _, d := range defs {
		byID[d.ID] = d
	}
	plan := Plan{Upserts: map[int64]Value{}}
	final := make(map[int64]Value, len(current))
	maps.Copy(final, current)
	var errs FieldErrors
	seen := map[int64]bool{}
	for _, in := range inputs {
		d, ok := byID[in.FieldID]
		if !ok || seen[in.FieldID] || !CanWrite(d, a) {
			errs = append(errs, FieldError{ErrorField(in.FieldID), MsgUnknownField})
			continue
		}
		seen[in.FieldID] = true
		v, err := NormalizeValue(d, in)
		if err != nil {
			msg := MsgIncompatible
			if ve, ok := errors.AsType[*ValueError](err); ok {
				msg = ve.Message
			}
			errs = append(errs, FieldError{ErrorField(d.ID), msg})
			continue
		}
		old := current[d.ID]
		final[d.ID] = v
		switch {
		case v.Equal(old):
		case v.IsEmpty():
			plan.Deletes = append(plan.Deletes, d.ID)
		default:
			plan.Upserts[d.ID] = v
		}
	}
	for _, d := range defs {
		if d.Required && CanWrite(d, a) && final[d.ID].IsEmpty() && !hasError(errs, d.ID) {
			errs = append(errs, FieldError{ErrorField(d.ID), MsgRequired})
		}
	}
	if len(errs) > 0 {
		return Plan{}, errs
	}
	return plan, nil
}

func hasError(errs FieldErrors, id int64) bool {
	f := ErrorField(id)
	for _, e := range errs {
		if e.Field == f {
			return true
		}
	}
	return false
}
