package customfield

import (
	"strings"
	"testing"
)

func TestNormalizeValue(t *testing.T) {
	short := Definition{ID: 1, Type: TypeShortText}
	long := Definition{ID: 2, Type: TypeLongText}
	number := Definition{ID: 3, Type: TypeNumber}
	date := Definition{ID: 4, Type: TypeDate}
	boolean := Definition{ID: 5, Type: TypeBool}
	choice := Definition{ID: 6, Type: TypeChoice, Choices: []Choice{{ID: 10, Label: "S"}, {ID: 11, Label: "M"}}}

	errCases := []struct {
		name string
		d    Definition
		in   Input
		msg  string
	}{
		{"texte court de 201", short, Input{Text: new(strings.Repeat("é", 201))}, MsgShortTooLong},
		{"texte court sur deux lignes", short, Input{Text: new("a\nb")}, MsgShortOneLine},
		{"texte long de 2001", long, Input{Text: new(strings.Repeat("a", 2001))}, MsgLongTooLong},
		{"nombre abc", number, Input{Text: new("abc")}, MsgNumber},
		{"nombre 1e13", number, Input{Text: new("1e13")}, MsgNumber},
		{"nombre trop grand", number, Input{Text: new("1000000000000")}, MsgNumber},
		{"NaN", number, Input{Text: new("NaN")}, MsgNumber},
		{"date impossible", date, Input{Date: new("2026-02-30")}, MsgDate},
		{"date française", date, Input{Date: new("28/02/2026")}, MsgDate},
		{"choix d'un autre champ", choice, Input{ChoiceID: new(int64(99))}, MsgUnknownChoice},
		{"deux propriétés", short, Input{Text: new("a"), Checked: new(true)}, MsgIncompatible},
		{"date sur un texte", short, Input{Date: new("2026-01-01")}, MsgIncompatible},
		{"texte sur un oui/non", boolean, Input{Text: new("oui")}, MsgIncompatible},
	}
	for _, c := range errCases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NormalizeValue(c.d, c.in)
			if err == nil || err.Error() != c.msg {
				t.Fatalf("erreur « %s » attendue, obtenu %v", c.msg, err)
			}
		})
	}

	numbers := map[string]float64{"12,5": 12.5, "12.5": 12.5, "1 234,5": 1234.5, "1 234": 1234, "-3": -3, "0": 0}
	for in, want := range numbers {
		v, err := NormalizeValue(number, Input{Text: new(in)})
		if err != nil || v.Number == nil || *v.Number != want {
			t.Errorf("nombre %q : %v %v, attendu %v", in, v.Number, err, want)
		}
	}

	if v, err := NormalizeValue(short, Input{Text: new("  1234567 ")}); err != nil || *v.Text != "1234567" {
		t.Errorf("texte court nettoyé : %v %v", v.Text, err)
	}
	if v, err := NormalizeValue(long, Input{Text: new("a\r\nb")}); err != nil || *v.Long != "a\nb" {
		t.Errorf("texte long : %v %v", v.Long, err)
	}
	if v, err := NormalizeValue(short, Input{Text: new(strings.Repeat("é", 200))}); err != nil || v.IsEmpty() {
		t.Errorf("200 caractères refusés : %v", err)
	}
	if v, err := NormalizeValue(date, Input{Date: new("2026-02-28")}); err != nil || *v.Date != "2026-02-28" {
		t.Errorf("date : %v %v", v.Date, err)
	}
	if v, err := NormalizeValue(boolean, Input{Checked: new(true)}); err != nil || v.Bool == nil || !*v.Bool {
		t.Errorf("oui : %v %v", v.Bool, err)
	}
	if v, err := NormalizeValue(choice, Input{ChoiceID: new(int64(11))}); err != nil || *v.ChoiceID != 11 {
		t.Errorf("choix : %v %v", v.ChoiceID, err)
	}
	for name, c := range map[string]struct {
		d  Definition
		in Input
	}{
		"oui/non décoché": {boolean, Input{Checked: new(false)}},
		"texte vide":      {short, Input{Text: new("  ")}},
		"nombre vide":     {number, Input{Text: new("")}},
		"date vide":       {date, Input{Date: new("")}},
		"aucune valeur":   {choice, Input{}},
	} {
		if v, err := NormalizeValue(c.d, c.in); err != nil || !v.IsEmpty() {
			t.Errorf("%s : effacement attendu, obtenu %+v %v", name, v, err)
		}
	}
}
