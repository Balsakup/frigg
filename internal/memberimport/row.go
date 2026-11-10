package memberimport

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/balsakup/frigg/internal/csvfile"
	"github.com/balsakup/frigg/internal/customfield"
	"github.com/balsakup/frigg/internal/member"
	"github.com/balsakup/frigg/internal/season"
	"github.com/balsakup/frigg/internal/textfold"
)

// Erreurs propres à l'import ; leur texte est le message affiché. Les
// erreurs de member (ErrPostalCode…) sont traduites par l'API avec les
// messages du formulaire de fiche, sans duplication.
var (
	ErrTwoDigitYear   = errors.New("Indiquez l'année sur 4 chiffres (JJ/MM/AAAA).")
	ErrDateFormat     = errors.New("Date non reconnue : utilisez le format JJ/MM/AAAA.")
	ErrDateInvalid    = errors.New("Cette date n'existe pas.")
	ErrBirthRequired  = errors.New("La date de naissance est obligatoire.")
	ErrExtraColumns   = errors.New("Cette ligne a plus de colonnes que la ligne d'intitulés.")
	ErrBoolValue      = errors.New("Valeur non reconnue : indiquez oui ou non.")
)

// maxListedChoices borne les choix rappelés par un message d'erreur.
const maxListedChoices = 10

// LengthError signale une valeur trop longue, refusée sans troncature.
type LengthError struct{ Max int }

func (e *LengthError) Error() string { return fmt.Sprintf("%d caractères au plus.", e.Max) }

// ChoiceError signale un choix inconnu et rappelle les choix possibles.
type ChoiceError struct{ Choices []string }

func (e *ChoiceError) Error() string {
	list := e.Choices
	suffix := ""
	if len(list) > maxListedChoices {
		list, suffix = list[:maxListedChoices], "…"
	}
	return "Choix non reconnu. Choix possibles : " + strings.Join(list, ", ") + suffix
}

var (
	dmyRe = regexp.MustCompile(`^(\d{1,2})[/.-](\d{1,2})[/.-](\d+)$`)
	isoRe = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
)

// ParseDate lit une date J/M/AAAA, J-M-AAAA, J.M.AAAA ou AAAA-MM-JJ.
func ParseDate(s string) (season.Date, error) {
	s = strings.TrimSpace(s)
	var y, m, d string
	if g := isoRe.FindStringSubmatch(s); g != nil {
		y, m, d = g[1], g[2], g[3]
	} else if g := dmyRe.FindStringSubmatch(s); g != nil {
		d, m, y = g[1], g[2], g[3]
		if len(y) == 2 {
			return season.Date{}, ErrTwoDigitYear
		}
		if len(y) != 4 {
			return season.Date{}, ErrDateFormat
		}
	} else {
		return season.Date{}, ErrDateFormat
	}
	yy, _ := strconv.Atoi(y)
	mm, _ := strconv.Atoi(m)
	dd, _ := strconv.Atoi(d)
	t := time.Date(yy, time.Month(mm), dd, 0, 0, 0, 0, time.UTC)
	if t.Year() != yy || int(t.Month()) != mm || t.Day() != dd {
		return season.Date{}, ErrDateInvalid
	}
	return season.ParseDate(t.Format(time.DateOnly))
}

// CellError est une erreur d'une ligne. Column vaut -1 pour la ligne
// entière ; Field est le champ de member en cause (« postalCode »), vide
// pour une autre erreur.
type CellError struct {
	Column int
	Field  string
	Err    error
}

// RawIdentity sont les valeurs brutes des colonnes d'identité, pour
// l'affichage.
type RawIdentity struct {
	FirstName, LastName, BirthDate string
}

// Parsed est une ligne lue : fiche nettoyée si elle est valide, valeurs des
// champs personnalisés, erreurs et champs obligatoires sans valeur.
type Parsed struct {
	Line            int
	Input           member.Clean
	Fields          map[int64]customfield.Value
	Errors          []CellError
	MissingRequired []int64
	Raw             RawIdentity
}

// maxLengths reprend les limites du formulaire de fiche (MemberBody).
var maxLengths = map[Target]int{
	FirstName: member.MaxName, LastName: member.MaxName, Email: member.MaxEmail, Phone: 30,
	AddressLine1: member.MaxAddress, AddressLine2: member.MaxAddress, PostalCode: 10, City: member.MaxAddress,
}

// ParseRow lit et valide une ligne avec les règles de la fiche (member) et
// des champs personnalisés (customfield) ; toutes les erreurs sont
// collectées. Un champ personnalisé obligatoire sans valeur n'est pas une
// erreur (FR-013) : customfield.Apply n'est pas utilisé.
func ParseRow(row csvfile.Row, m []Target, defs []customfield.Definition, today season.Date) Parsed {
	p := Parsed{Line: row.Line, Fields: map[int64]customfield.Value{}}
	col := map[Target]int{}
	for i, t := range m {
		if t != None {
			col[t] = i
		}
	}
	cell := func(t Target) string {
		if i, ok := col[t]; ok && i < len(row.Cells) {
			return row.Cells[i]
		}
		return ""
	}
	column := func(t Target) int {
		if i, ok := col[t]; ok {
			return i
		}
		return -1
	}
	add := func(column int, field string, err error) {
		p.Errors = append(p.Errors, CellError{Column: column, Field: field, Err: err})
	}
	p.Raw = RawIdentity{FirstName: cell(FirstName), LastName: cell(LastName), BirthDate: cell(BirthDate)}

	if row.Extra {
		add(-1, "", ErrExtraColumns)
	}

	tooLong := map[Target]bool{}
	for _, s := range StandardTargets {
		if max, ok := maxLengths[s.Target]; ok && utf8.RuneCountInString(cell(s.Target)) > max {
			add(column(s.Target), string(s.Target), &LengthError{Max: max})
			tooLong[s.Target] = true
		}
	}

	birth, birthOK := today, false
	if raw := cell(BirthDate); raw == "" {
		add(column(BirthDate), string(BirthDate), ErrBirthRequired)
	} else if d, err := ParseDate(raw); err != nil {
		add(column(BirthDate), string(BirthDate), err)
	} else {
		birth, birthOK = d, true
	}

	clean, err := member.Validate(member.Input{
		FirstName: cell(FirstName), LastName: cell(LastName), BirthDate: birth,
		Email: cell(Email), Phone: cell(Phone),
		AddressLine1: cell(AddressLine1), AddressLine2: cell(AddressLine2),
		PostalCode: cell(PostalCode), City: cell(City),
	}, today)
	if fes, ok := errors.AsType[member.FieldErrors](err); ok {
		for _, fe := range fes {
			t := Target(fe.Field)
			if tooLong[t] || (t == BirthDate && !birthOK) {
				continue
			}
			add(errorColumn(t, fe.Err, column), fe.Field, fe.Err)
		}
	}
	p.Input = clean

	for _, d := range defs {
		i, mapped := col[FieldTarget(d.ID)]
		raw := ""
		if mapped {
			raw = row.Cells[i]
		}
		v, err := parseField(d, raw)
		if err != nil {
			add(i, "", err)
			continue
		}
		if v.IsEmpty() {
			if d.Required {
				p.MissingRequired = append(p.MissingRequired, d.ID)
			}
			continue
		}
		p.Fields[d.ID] = v
	}
	if len(p.Errors) > 0 {
		p.Input = member.Clean{}
		// Erreurs de ligne d'abord, puis dans l'ordre des colonnes.
		slices.SortStableFunc(p.Errors, func(a, b CellError) int { return cmp.Compare(a.Column, b.Column) })
	}
	return p
}

// errorColumn rattache une erreur de fiche à sa colonne : une adresse
// incomplète porte sur le champ manquant, non associé, donc sur une colonne
// d'adresse associée, à défaut sur la ligne.
func errorColumn(t Target, err error, column func(Target) int) int {
	if c := column(t); c >= 0 || !errors.Is(err, member.ErrAddressIncomplete) {
		return c
	}
	for _, alt := range []Target{AddressLine1, PostalCode, City} {
		if c := column(alt); c >= 0 {
			return c
		}
	}
	return -1
}

var (
	boolTrue  = map[string]bool{"oui": true, "vrai": true, "1": true, "x": true}
	boolFalse = map[string]bool{"non": true, "faux": true, "0": true}
)

// parseField convertit une cellule pour un champ personnalisé puis la
// normalise avec les règles de la saisie manuelle (customfield).
func parseField(d customfield.Definition, raw string) (customfield.Value, error) {
	if strings.TrimSpace(raw) == "" {
		return customfield.Value{}, nil
	}
	var in customfield.Input
	switch d.Type {
	case customfield.TypeDate:
		date, err := ParseDate(raw)
		if err != nil {
			return customfield.Value{}, err
		}
		iso := date.String()
		in.Date = &iso
	case customfield.TypeBool:
		folded := textfold.Fold(raw)
		var checked bool
		switch {
		case boolTrue[folded]:
			checked = true
		case boolFalse[folded]:
		default:
			return customfield.Value{}, ErrBoolValue
		}
		in.Checked = &checked
	case customfield.TypeChoice:
		folded := textfold.Fold(raw)
		labels := make([]string, len(d.Choices))
		for i, c := range d.Choices {
			labels[i] = c.Label
			if textfold.Fold(c.Label) == folded {
				id := c.ID
				in.ChoiceID = &id
			}
		}
		if in.ChoiceID == nil {
			return customfield.Value{}, &ChoiceError{Choices: labels}
		}
	default:
		in.Text = &raw
	}
	return customfield.NormalizeValue(d, in)
}
