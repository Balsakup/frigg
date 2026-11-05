package customfield

import "testing"

func TestVisibility(t *testing.T) {
	cases := []struct {
		v                     Visibility
		ownerRead, ownerWrite bool
	}{
		{VisibilityAdmin, false, false},
		{VisibilityVisible, true, false},
		{VisibilityEditable, true, true},
	}
	for _, c := range cases {
		d := Definition{Visibility: c.v}
		if !CanRead(d, Admin) || !CanWrite(d, Admin) {
			t.Errorf("%s : l'administration doit tout lire et écrire", c.v)
		}
		if CanRead(d, Owner) != c.ownerRead || CanWrite(d, Owner) != c.ownerWrite {
			t.Errorf("%s : Owner lecture=%v écriture=%v", c.v, CanRead(d, Owner), CanWrite(d, Owner))
		}
	}
}

func TestApply(t *testing.T) {
	licence := Definition{ID: 1, Type: TypeShortText, Visibility: VisibilityEditable}
	visible := Definition{ID: 2, Type: TypeShortText, Visibility: VisibilityVisible}
	reserved := Definition{ID: 3, Type: TypeShortText, Visibility: VisibilityAdmin, Required: true}
	paid := Definition{ID: 4, Type: TypeBool, Visibility: VisibilityAdmin}
	all := []Definition{licence, visible, reserved, paid}
	ownerDefs := []Definition{licence, visible}

	t.Run("champ absent inchangé", func(t *testing.T) {
		current := map[int64]Value{1: {Text: new("111")}}
		plan, errs := Apply(ownerDefs, current, nil, Owner)
		if errs != nil || plan.Changed() != 0 {
			t.Fatalf("%v %+v", errs, plan)
		}
	})
	t.Run("écriture, effacement et valeur identique", func(t *testing.T) {
		current := map[int64]Value{1: {Text: new("111")}, 3: {Text: new("note")}, 4: {Bool: new(true)}}
		plan, errs := Apply(all, current, []Input{
			{FieldID: 1, Text: new("111")},
			{FieldID: 2, Text: new("nouveau")},
			{FieldID: 4, Checked: new(false)},
		}, Admin)
		if errs != nil {
			t.Fatal(errs)
		}
		if plan.Changed() != 2 || *plan.Upserts[2].Text != "nouveau" || len(plan.Deletes) != 1 || plan.Deletes[0] != 4 {
			t.Fatalf("%+v", plan)
		}
	})
	t.Run("même message pour inconnu, visible et en double", func(t *testing.T) {
		_, errs := Apply(ownerDefs, nil, []Input{
			{FieldID: 2, Text: new("x")},
			{FieldID: 3, Text: new("x")},
			{FieldID: 99, Text: new("x")},
			{FieldID: 1, Text: new("a")},
			{FieldID: 1, Text: new("b")},
		}, Owner)
		for _, id := range []int64{2, 3, 99} {
			if !hasField(errs, ErrorField(id), MsgUnknownField) {
				t.Errorf("champ %d : %v", id, errs)
			}
		}
		if !hasField(errs, ErrorField(1), MsgUnknownField) {
			t.Errorf("doublon accepté : %v", errs)
		}
	})
	t.Run("obligatoire réservé : exigé de l'administration seulement", func(t *testing.T) {
		if _, errs := Apply(ownerDefs, nil, []Input{{FieldID: 1, Text: new("1")}}, Owner); errs != nil {
			t.Fatalf("membre bloqué : %v", errs)
		}
		_, errs := Apply(all, nil, []Input{{FieldID: 1, Text: new("1")}}, Admin)
		if !hasField(errs, ErrorField(3), MsgRequired) {
			t.Fatalf("obligatoire non exigé : %v", errs)
		}
	})
	t.Run("obligatoire rempli par le membre", func(t *testing.T) {
		req := licence
		req.Required = true
		_, errs := Apply([]Definition{req}, nil, nil, Owner)
		if !hasField(errs, ErrorField(1), MsgRequired) {
			t.Fatalf("%v", errs)
		}
		if _, errs := Apply([]Definition{req}, map[int64]Value{1: {Text: new("x")}}, nil, Owner); errs != nil {
			t.Fatalf("valeur existante : %v", errs)
		}
	})
	t.Run("plusieurs erreurs ensemble, sans doublon d'erreur", func(t *testing.T) {
		num := Definition{ID: 5, Type: TypeNumber, Required: true}
		_, errs := Apply([]Definition{num, reserved}, nil, []Input{{FieldID: 5, Text: new("abc")}}, Admin)
		if len(errs) != 2 || !hasField(errs, ErrorField(5), MsgNumber) || !hasField(errs, ErrorField(3), MsgRequired) {
			t.Fatalf("%v", errs)
		}
	})
}
