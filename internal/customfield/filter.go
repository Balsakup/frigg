package customfield

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
)

// MaxFilters est le nombre maximal de filtres par champ dans une liste.
const MaxFilters = 10

// Opérations de filtre (research.md 011 › R6).
const (
	OpTrue   = "true"
	OpFalse  = "false"
	OpEmpty  = "empty"
	OpChoice = "choice"
)

// Filter retient les fiches ou adhésions selon la valeur d'un champ oui/non
// ou liste de choix.
type Filter struct {
	FieldID  int64  `json:"field"`
	Op       string `json:"op"`
	ChoiceID int64  `json:"choice,omitempty"`
}

var filterRe = regexp.MustCompile(`^([0-9]+):(true|false|empty|c([0-9]+))$`)

var errFilter = errors.New(MsgUnknownFilter)

// ParseFilter lit un filtre « <champ>:<valeur> » (true, false, empty ou
// c<choix>) ; defs contient les champs actifs de la portée de la liste.
func ParseFilter(raw string, defs []Definition) (Filter, error) {
	m := filterRe.FindStringSubmatch(raw)
	if m == nil {
		return Filter{}, errFilter
	}
	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return Filter{}, errFilter
	}
	var def *Definition
	for i := range defs {
		if defs[i].ID == id {
			def = &defs[i]
		}
	}
	if def == nil || def.Archived {
		return Filter{}, errFilter
	}
	f := Filter{FieldID: id}
	switch def.Type {
	case TypeBool:
		if m[2] != OpTrue && m[2] != OpFalse {
			return Filter{}, errFilter
		}
		f.Op = m[2]
	case TypeChoice:
		if m[2] == OpEmpty {
			f.Op = OpEmpty
			break
		}
		if m[3] == "" {
			return Filter{}, errFilter
		}
		choice, err := strconv.ParseInt(m[3], 10, 64)
		if err != nil {
			return Filter{}, errFilter
		}
		known := false
		for _, c := range def.Choices {
			known = known || c.ID == choice
		}
		if !known {
			return Filter{}, errFilter
		}
		f.Op, f.ChoiceID = OpChoice, choice
	default:
		return Filter{}, errFilter
	}
	return f, nil
}

// FiltersJSON encode les filtres pour la requête SQL (json_each) ; chaîne
// vide s'il n'y en a aucun.
func FiltersJSON(filters []Filter) string {
	if len(filters) == 0 {
		return ""
	}
	b, _ := json.Marshal(filters)
	return string(b)
}
