package membership

import "testing"

func TestComputeStats(t *testing.T) {
	rows := []StatRow{
		{Status: Validated, Kind: First, AmountCents: 6000, FeeID: 1, FeeLabel: "Adulte", PaymentStatus: Paid},
		{Status: Validated, Kind: First, AmountCents: 3000, FeeID: 2, FeeLabel: "Jeune", PaymentStatus: Due},
		{Status: Validated, Kind: Renewal, AmountCents: 4000, FeeID: 1, FeeLabel: "Adulte", PaymentStatus: Announced, AnnouncedMethod: Transfer},
		{Status: Validated, Kind: First, AmountCents: 0, FeeID: 3, FeeLabel: "Bienfaiteur"},
		{Status: Pending, FeeID: 1, FeeLabel: "Adulte"},
		{Status: Refused, FeeID: 2, FeeLabel: "Jeune"},
		{Status: Cancelled, Kind: First, AmountCents: 6000, FeeID: 1, FeeLabel: "Adulte", PaymentStatus: Paid},
	}
	s := ComputeStats(rows)
	if s.Validated != 4 || s.First != 3 || s.Renewal != 1 || s.Pending != 1 {
		t.Errorf("compteurs : %+v", s)
	}
	if s.TotalCents != 13000 || s.PaidCents != 6000 || s.DueCents != 7000 {
		t.Errorf("montants : total %d, encaissé %d, restant %d", s.TotalCents, s.PaidCents, s.DueCents)
	}
	if s.DueByMethod != (DueByMethod{None: 3000, Transfer: 4000}) {
		t.Errorf("restant par moyen : %+v", s.DueByMethod)
	}
	want := []FeeStat{
		{FeeID: 1, Label: "Adulte", Count: 2, AmountCents: 10000},
		{FeeID: 2, Label: "Jeune", Count: 1, AmountCents: 3000},
		{FeeID: 3, Label: "Bienfaiteur", Count: 1, AmountCents: 0},
	}
	if len(s.ByFee) != len(want) {
		t.Fatalf("répartition : %+v", s.ByFee)
	}
	for i := range want {
		if s.ByFee[i] != want[i] {
			t.Errorf("répartition %d : %+v, attendu %+v", i, s.ByFee[i], want[i])
		}
	}
	if empty := ComputeStats(nil); empty.ByFee == nil || empty.Validated != 0 {
		t.Errorf("saison vide : %+v", empty)
	}
}
