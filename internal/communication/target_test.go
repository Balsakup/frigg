package communication

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPresets(t *testing.T) {
	for _, tc := range []struct {
		situations []Situation
		want       Preset
	}{
		{[]Situation{Due, Announced, UpToDate}, PresetAll},
		{[]Situation{UpToDate}, PresetUpToDate},
		{[]Situation{Due, Announced}, PresetUnpaid},
		{[]Situation{Announced, Due, Due}, PresetUnpaid},
		{[]Situation{Due}, PresetCustom},
		{[]Situation{UpToDate, Due}, PresetCustom},
		{nil, PresetCustom},
	} {
		if got := PresetOf(tc.situations); got != tc.want {
			t.Errorf("PresetOf(%v) = %s, attendu %s", tc.situations, got, tc.want)
		}
	}
	for _, p := range []Preset{PresetAll, PresetUpToDate, PresetUnpaid} {
		if got := PresetOf(SituationsOf(p)); got != p {
			t.Errorf("aller-retour %s → %s", p, got)
		}
	}
	if SituationsOf(PresetCustom) != nil {
		t.Error("custom n'a pas de situations prédéfinies")
	}
}

func TestNormalizeSituations(t *testing.T) {
	got, err := NormalizeSituations([]Situation{UpToDate, Due, Announced, Due})
	if err != nil || !slices.Equal(got, []Situation{Announced, Due, UpToDate}) {
		t.Fatalf("normalisation : %v %v", got, err)
	}
	if EncodeSituations(got) != "announced,due,up_to_date" {
		t.Errorf("encodage : %s", EncodeSituations(got))
	}
	if !slices.Equal(DecodeSituations("announced,due"), []Situation{Announced, Due}) {
		t.Error("décodage")
	}
	if _, err := NormalizeSituations(nil); !errors.Is(err, ErrNoSituation) {
		t.Errorf("vide : %v", err)
	}
	if _, err := NormalizeSituations([]Situation{"paid"}); !errors.Is(err, ErrUnknownSituation) {
		t.Errorf("inconnue : %v", err)
	}
}

func TestFeeIDs(t *testing.T) {
	ids := NormalizeFeeIDs([]int64{3, 1, 3, 2})
	if !slices.Equal(ids, []int64{1, 2, 3}) || EncodeFeeIDs(ids) != "1,2,3" {
		t.Fatalf("tarifs : %v", ids)
	}
	if !slices.Equal(DecodeFeeIDs("1,2,3"), ids) || len(DecodeFeeIDs("")) != 0 || EncodeFeeIDs(nil) != "" {
		t.Error("aller-retour des tarifs")
	}
}

func TestTargetLabel(t *testing.T) {
	for _, tc := range []struct {
		situations []Situation
		fees       []string
		want       string
	}{
		{SituationsOf(PresetUnpaid), nil, "2026-2027 · En impayé"},
		{SituationsOf(PresetAll), nil, "2026-2027 · Tous les adhérents"},
		{[]Situation{Due}, []string{"Jeune", "Adulte"}, "2026-2027 · Personnalisée : À payer · Tarifs : Jeune, Adulte"},
		{[]Situation{Due, UpToDate}, nil, "2026-2027 · Personnalisée : À jour, À payer"},
	} {
		if got := TargetLabel("2026-2027", tc.situations, tc.fees); got != tc.want {
			t.Errorf("TargetLabel(%v, %v) = %q, attendu %q", tc.situations, tc.fees, got, tc.want)
		}
	}
	long := TargetLabel("2026-2027", []Situation{Due}, []string{strings.Repeat("é", 400)})
	if utf8.RuneCountInString(long) != TargetLabelMax || !strings.HasSuffix(long, "…") {
		t.Errorf("libellé tronqué : %d caractères", utf8.RuneCountInString(long))
	}
}

func TestValidDraftKey(t *testing.T) {
	if !ValidDraftKey("0b0e9c8a-1d2f-4a3b-9c4d-5e6f7a8b9c0d") {
		t.Error("UUID v4 refusé")
	}
	for _, bad := range []string{"", "0B0E9C8A-1D2F-4A3B-9C4D-5E6F7A8B9C0D", "0b0e9c8a-1d2f-1a3b-9c4d-5e6f7a8b9c0d", "../etc"} {
		if ValidDraftKey(bad) {
			t.Errorf("%q accepté", bad)
		}
	}
}
