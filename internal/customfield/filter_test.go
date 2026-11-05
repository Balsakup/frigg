package customfield

import "testing"

func TestParseFilter(t *testing.T) {
	defs := []Definition{
		{ID: 4, Type: TypeBool},
		{ID: 7, Type: TypeChoice, Choices: []Choice{{ID: 12, Label: "S"}, {ID: 13, Label: "M"}}},
		{ID: 8, Type: TypeShortText},
		{ID: 9, Type: TypeBool, Archived: true},
	}
	ok := map[string]Filter{
		"4:true":  {FieldID: 4, Op: OpTrue},
		"4:false": {FieldID: 4, Op: OpFalse},
		"7:empty": {FieldID: 7, Op: OpEmpty},
		"7:c12":   {FieldID: 7, Op: OpChoice, ChoiceID: 12},
	}
	for raw, want := range ok {
		got, err := ParseFilter(raw, defs)
		if err != nil || got != want {
			t.Errorf("%s : %+v %v, attendu %+v", raw, got, err, want)
		}
	}
	for _, raw := range []string{"", "4", "4:oui", "x:true", "4:empty", "4:c12", "7:true", "7:c99", "8:true", "9:true", "99:true", "7:c", " 4:true"} {
		if _, err := ParseFilter(raw, defs); err == nil || err.Error() != MsgUnknownFilter {
			t.Errorf("%q accepté", raw)
		}
	}
	if FiltersJSON(nil) != "" {
		t.Error("JSON sans filtre")
	}
	if got := FiltersJSON([]Filter{{FieldID: 4, Op: OpFalse}, {FieldID: 7, Op: OpChoice, ChoiceID: 12}}); got != `[{"field":4,"op":"false"},{"field":7,"op":"choice","choice":12}]` {
		t.Errorf("JSON : %s", got)
	}
}
