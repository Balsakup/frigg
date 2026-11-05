package customfield

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Input est une valeur saisie : au plus une propriété renseignée. Text porte
// un texte ou la saisie brute d'un nombre (« 12,5 »). Aucune propriété, ou
// un texte vide, efface la valeur.
type Input struct {
	FieldID  int64
	Text     *string
	Date     *string
	Checked  *bool
	ChoiceID *int64
}

// Value est une valeur normalisée : au plus une propriété renseignée, celle
// de la colonne du type (data-model.md › Validation). Une valeur vide
// signifie « non renseigné » (ligne absente).
type Value struct {
	Text     *string // short_text
	Long     *string // long_text
	Number   *float64
	Date     *string
	Bool     *bool
	ChoiceID *int64
}

// IsEmpty indique une valeur non renseignée.
func (v Value) IsEmpty() bool {
	return v.Text == nil && v.Long == nil && v.Number == nil && v.Date == nil && v.Bool == nil && v.ChoiceID == nil
}

// Equal compare deux valeurs normalisées.
func (v Value) Equal(o Value) bool {
	return eqPtr(v.Text, o.Text) && eqPtr(v.Long, o.Long) && eqPtr(v.Number, o.Number) &&
		eqPtr(v.Date, o.Date) && eqPtr(v.Bool, o.Bool) && eqPtr(v.ChoiceID, o.ChoiceID)
}

func eqPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// MaxNumber borne les nombres en valeur absolue.
const MaxNumber = 1e12

var errValue = errors.New("valeur invalide")

// ValueError porte le message à afficher pour une valeur refusée.
type ValueError struct{ Message string }

func (e *ValueError) Error() string { return e.Message }

func fail(msg string) error { return &ValueError{Message: msg} }

// NormalizeValue vérifie une valeur saisie selon le type du champ (FR-014).
func NormalizeValue(d Definition, in Input) (Value, error) {
	set := 0
	for _, ok := range []bool{in.Text != nil, in.Date != nil, in.Checked != nil, in.ChoiceID != nil} {
		if ok {
			set++
		}
	}
	if set > 1 {
		return Value{}, fail(MsgIncompatible)
	}
	if set == 0 {
		return Value{}, nil
	}
	switch d.Type {
	case TypeShortText:
		if in.Text == nil {
			return Value{}, fail(MsgIncompatible)
		}
		s := strings.TrimSpace(*in.Text)
		if s == "" {
			return Value{}, nil
		}
		if strings.ContainsAny(s, "\r\n") {
			return Value{}, fail(MsgShortOneLine)
		}
		if utf8.RuneCountInString(s) > MaxShort {
			return Value{}, fail(MsgShortTooLong)
		}
		return Value{Text: &s}, nil
	case TypeLongText:
		if in.Text == nil {
			return Value{}, fail(MsgIncompatible)
		}
		s := strings.TrimSpace(strings.ReplaceAll(*in.Text, "\r\n", "\n"))
		if s == "" {
			return Value{}, nil
		}
		if utf8.RuneCountInString(s) > MaxLong {
			return Value{}, fail(MsgLongTooLong)
		}
		return Value{Long: &s}, nil
	case TypeNumber:
		if in.Text == nil {
			return Value{}, fail(MsgIncompatible)
		}
		if strings.TrimSpace(*in.Text) == "" {
			return Value{}, nil
		}
		n, err := ParseNumber(*in.Text)
		if err != nil {
			return Value{}, fail(MsgNumber)
		}
		return Value{Number: &n}, nil
	case TypeDate:
		if in.Date == nil {
			return Value{}, fail(MsgIncompatible)
		}
		if *in.Date == "" {
			return Value{}, nil
		}
		t, err := time.Parse(time.DateOnly, *in.Date)
		if err != nil || t.Format(time.DateOnly) != *in.Date {
			return Value{}, fail(MsgDate)
		}
		s := *in.Date
		return Value{Date: &s}, nil
	case TypeBool:
		if in.Checked == nil {
			return Value{}, fail(MsgIncompatible)
		}
		if !*in.Checked {
			return Value{}, nil
		}
		b := true
		return Value{Bool: &b}, nil
	case TypeChoice:
		if in.ChoiceID == nil {
			return Value{}, fail(MsgIncompatible)
		}
		for _, c := range d.Choices {
			if c.ID == *in.ChoiceID {
				id := c.ID
				return Value{ChoiceID: &id}, nil
			}
		}
		return Value{}, fail(MsgUnknownChoice)
	}
	return Value{}, fail(MsgIncompatible)
}

// ParseNumber lit un nombre saisi en français ou en anglais : espaces (y
// compris insécables) ignorés, virgule ou point décimal ; fini et borné.
func ParseNumber(s string) (float64, error) {
	s = strings.Map(func(r rune) rune {
		switch r {
		case ' ', ' ', ' ', '\t':
			return -1
		case ',':
			return '.'
		}
		return r
	}, s)
	if s == "" || strings.ContainsAny(s, "eExXpP_") {
		return 0, errValue
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) >= MaxNumber {
		return 0, errValue
	}
	return n, nil
}
