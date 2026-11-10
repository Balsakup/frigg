package membership

import "testing"

func TestComputeStats(t *testing.T) {
	rows := []StatRow{
		{Status: Validated, Kind: First, AmountCents: 6000, FeeID: 1, FeeLabel: "Adulte", PaymentStatus: Paid, PaidMethod: Cash, PaidAmountCents: 6000},
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

func TestComputeStatsCard(t *testing.T) {
	rows := []StatRow{
		{Status: Validated, AmountCents: 4000, PaymentStatus: Paid, PaidMethod: Card, PaidAmountCents: 4000},
		{Status: Validated, AmountCents: 6000, PaymentStatus: Paid, PaidMethod: Card, PaidAmountCents: 5000},
		{Status: Validated, AmountCents: 4000, PaymentStatus: Paid, PaidMethod: Card, PaidAmountCents: 4000, PaidTest: true},
		{Status: Validated, AmountCents: 3000, PaymentStatus: Paid, PaidMethod: Cheque, PaidAmountCents: 3000},
		{Status: Validated, AmountCents: 2000, PaymentStatus: Due},
	}
	s := ComputeStats(rows)
	if s.PaidCents != 12000 || s.PaidByMethod.Card != 9000 || s.PaidByMethod.Cheque != 3000 {
		t.Fatalf("encaissé : %d, par moyen %+v", s.PaidCents, s.PaidByMethod)
	}
	if s.TestPaidCents != 4000 {
		t.Fatalf("paiements de test : %d", s.TestPaidCents)
	}
	if s.TotalCents != 19000 || s.DueCents != 2000 {
		t.Fatalf("total %d, restant %d", s.TotalCents, s.DueCents)
	}
}

func TestComputeStatsAdjustments(t *testing.T) {
	base := int64(7500)
	rows := []StatRow{
		{Status: Validated, AmountCents: 7000, BaseCents: &base, FeeID: 1, FeeLabel: "Adulte", PaymentStatus: Paid, PaidMethod: Transfer, PaidAmountCents: 7000},
		{Status: Validated, AmountCents: 4500, FeeID: 1, FeeLabel: "Adulte", PaymentStatus: Paid, PaidMethod: Transfer, PaidAmountCents: 8000},
		{Status: Cancelled, AmountCents: 4500, FeeID: 1, FeeLabel: "Adulte", PaymentStatus: Paid, PaidMethod: Cash, PaidAmountCents: 9000},
	}
	s := ComputeStats(rows)
	if s.TheoreticalCents != 12000 || s.DiscountCents != 500 || s.SurchargeCents != 0 || s.TotalCents != 11500 {
		t.Errorf("théorique %d, remises %d, majorations %d, à payer %d", s.TheoreticalCents, s.DiscountCents, s.SurchargeCents, s.TotalCents)
	}
	if s.PaidCents != 15000 || s.DonationCents != 3500 || s.PaidByMethod.Transfer != 15000 {
		t.Errorf("encaissé %d, dons %d, par moyen %+v", s.PaidCents, s.DonationCents, s.PaidByMethod)
	}
	if len(s.ByFee) != 1 || s.ByFee[0].AmountCents != 11500 {
		t.Errorf("répartition : %+v", s.ByFee)
	}
}
