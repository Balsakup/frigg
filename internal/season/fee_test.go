package season

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestParseAmount(t *testing.T) {
	ok := []struct {
		in   string
		want int64
	}{
		{"30", 3000},
		{"15,5", 1550},
		{"15.50", 1550},
		{"0", 0},
		{"1 000", 100000},
		{"1 000,00", 100000},
		{"1 000", 100000},
		{"10000", 1000000},
		{" 12 ", 1200},
		{"0030", 3000},
		{"00000000030", 3000},
		{"0,05", 5},
	}
	for _, tt := range ok {
		got, err := ParseAmount(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("ParseAmount(%q) = %d, %v ; attendu %d", tt.in, got, err, tt.want)
		}
	}
	bad := []struct {
		in   string
		want error
	}{
		{"-1", ErrAmountFormat},
		{"1,234", ErrAmountFormat},
		{"abc", ErrAmountFormat},
		{"", ErrAmountFormat},
		{"1,", ErrAmountFormat},
		{",5", ErrAmountFormat},
		{"1.2.3", ErrAmountFormat},
		{"10000,01", ErrAmountTooHigh},
		{"12345678", ErrAmountTooHigh},
		{"99999999999999999999", ErrAmountTooHigh},
	}
	for _, tt := range bad {
		_, err := ParseAmount(tt.in)
		var fe *FieldError
		if !errors.As(err, &fe) || fe.Field != "amount" || !errors.Is(err, tt.want) {
			t.Errorf("ParseAmount(%q) = %v ; attendu amount/%v", tt.in, err, tt.want)
		}
	}
}

// fieldCodes renvoie les couples « champ/erreur » d'une erreur de validation.
func fieldCodes(t *testing.T, err error) []string {
	t.Helper()
	var fes FieldErrors
	if !errors.As(err, &fes) {
		t.Fatalf("FieldErrors attendu, obtenu %v", err)
	}
	var out []string
	for _, fe := range fes {
		out = append(out, fe.Field+"/"+fe.Err.Error())
	}
	return out
}

func code(field string, err error) string { return field + "/" + err.Error() }

func TestValidateFee(t *testing.T) {
	existing := []Fee{
		{ID: 1, Label: "Plein tarif", AmountCents: 3000},
		{ID: 2, Label: "Tarif réduit", AmountCents: 1550},
	}

	in, err := ValidateFee(existing, 0, FeeDraft{Label: "  Membre d'honneur ", Amount: "0", Description: new("  Bureau  ")})
	if err != nil {
		t.Fatalf("création valide refusée : %v", err)
	}
	if in.Label != "Membre d'honneur" || in.AmountCents != 0 || in.Description == nil || *in.Description != "Bureau" {
		t.Errorf("valeurs nettoyées inattendues : %+v", in)
	}

	in, err = ValidateFee(existing, 0, FeeDraft{Label: "Famille", Amount: "50", Description: new("   ")})
	if err != nil || in.Description != nil {
		t.Errorf("description vide : attendu nil, obtenu %+v, %v", in, err)
	}
	if in, _ := ValidateFee(existing, 0, FeeDraft{Label: "Famille", Amount: "50"}); in.Description != nil {
		t.Error("description absente : attendu nil")
	}

	cases := []struct {
		name          string
		id            int64
		label, amount string
		renewal       *string
		want          []string
	}{
		{"libellé vide", 0, "   ", "10", nil, []string{code("label", ErrLabelEmpty)}},
		{"libellé pris", 0, "tarif REDUIT", "10", nil, []string{code("label", ErrLabelTaken)}},
		{"libellé pris par un autre", 1, "  tarif reduit", "10", nil, []string{code("label", ErrLabelTaken)}},
		{"vide et montant invalide", 0, "", "abc", nil, []string{code("label", ErrLabelEmpty), code("amount", ErrAmountFormat)}},
		{"pris et montant trop élevé", 0, "Plein Tarif", "20000", nil, []string{code("label", ErrLabelTaken), code("amount", ErrAmountTooHigh)}},
		{"renouvellement vide", 0, "Adulte", "85", new(""), []string{code("renewalAmount", ErrRenewalRequired)}},
		{"renouvellement blanc", 0, "Adulte", "85", new("   "), []string{code("renewalAmount", ErrRenewalRequired)}},
		{"renouvellement invalide", 0, "Adulte", "85", new("abc"), []string{code("renewalAmount", ErrAmountFormat)}},
		{"renouvellement trop élevé", 0, "Adulte", "85", new("20000"), []string{code("renewalAmount", ErrAmountTooHigh)}},
		{"trois champs en erreur", 0, "", "abc", new(""), []string{code("label", ErrLabelEmpty), code("amount", ErrAmountFormat), code("renewalAmount", ErrRenewalRequired)}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateFee(existing, tt.id, FeeDraft{Label: tt.label, Amount: tt.amount, RenewalAmount: tt.renewal})
			if got := fieldCodes(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("erreurs = %v, attendu %v", got, tt.want)
			}
		})
	}

	in, err = ValidateFee(existing, 0, FeeDraft{Label: "Adulte compétition", Amount: "85", RenewalAmount: new("55")})
	if err != nil || in.RenewalAmountCents == nil || *in.RenewalAmountCents != 5500 {
		t.Errorf("renouvellement : %+v, %v", in, err)
	}
	if in, _ := ValidateFee(existing, 0, FeeDraft{Label: "Adulte", Amount: "85"}); in.RenewalAmountCents != nil {
		t.Error("sans renouvellement : attendu nil")
	}
	if _, err := ValidateFee(existing, 0, FeeDraft{Label: "Atypique", Amount: "40", RenewalAmount: new("60")}); err != nil {
		t.Errorf("renouvellement supérieur au montant principal : %v", err)
	}
	if _, err := ValidateFee(existing, 1, FeeDraft{Label: "PLEIN TARIF", Amount: "32"}); err != nil {
		t.Errorf("garder son propre libellé : %v", err)
	}

	full := make([]Fee, MaxFees)
	for i := range full {
		full[i] = Fee{ID: int64(i + 1), Label: "Tarif " + strings.Repeat("x", i+1)}
	}
	if _, err := ValidateFee(full, 0, FeeDraft{Amount: "abc"}); !errors.Is(err, ErrTooManyFees) {
		t.Errorf("21e tarif : attendu ErrTooManyFees avant les champs, obtenu %v", err)
	}
	if _, err := ValidateFee(full, 3, FeeDraft{Label: "Nouveau nom", Amount: "10"}); err != nil {
		t.Errorf("modification avec 20 tarifs : %v", err)
	}
}

func TestFeeActionsFor(t *testing.T) {
	cases := []struct {
		state State
		count int
		want  FeeActions
	}{
		{Draft, 0, FeeActions{Add: true, Edit: true, Delete: true}},
		{Open, MaxFees - 1, FeeActions{Add: true, Edit: true, Delete: true}},
		{Open, MaxFees, FeeActions{Add: false, Edit: true, Delete: true}},
		{Closed, 2, FeeActions{}},
	}
	for _, tt := range cases {
		if got := FeeActionsFor(tt.state, tt.count); got != tt.want {
			t.Errorf("FeeActionsFor(%s, %d) = %+v, attendu %+v", tt.state, tt.count, got, tt.want)
		}
	}
}

func TestSortFees(t *testing.T) {
	fees := []Fee{
		{ID: 1, Label: "Famille", AmountCents: 1000},
		{ID: 2, Label: "Plein tarif", AmountCents: 3000},
		{ID: 3, Label: "élève", AmountCents: 1000},
		{ID: 4, Label: "Gratuit", AmountCents: 0},
		{ID: 5, Label: "Eleve", AmountCents: 1000},
	}
	SortFees(fees)
	var ids []int64
	for _, f := range fees {
		ids = append(ids, f.ID)
	}
	if want := []int64{2, 3, 5, 1, 4}; !slices.Equal(ids, want) {
		t.Errorf("ordre = %v, attendu %v", ids, want)
	}
}
