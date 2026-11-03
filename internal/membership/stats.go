package membership

import (
	"cmp"
	"slices"

	"github.com/balsakup/frigg/internal/textfold"
)

// StatRow est une adhésion d'une saison, réduite à ce que comptent les
// chiffres (research.md 006 › R10).
type StatRow struct {
	Status          Status
	Kind            Kind
	AmountCents     int64
	FeeID           int64
	FeeLabel        string
	PaymentStatus   PaymentStatus
	AnnouncedMethod Method
	// PaidMethod et PaidAmountCents : moyen constaté et, pour la carte,
	// montant reçu hors contribution volontaire.
	PaidMethod      Method
	PaidAmountCents int64
	// PaidTest : payée par carte en mode test (environnement sandbox).
	PaidTest bool
}

// PaidByMethod détaille le montant encaissé par moyen constaté (008).
type PaidByMethod struct {
	Card, Cash, Cheque, Transfer int64
}

// FeeStat est la répartition d'un tarif.
type FeeStat struct {
	FeeID       int64
	Label       string
	Count       int
	AmountCents int64
}

// DueByMethod détaille le restant à percevoir par moyen annoncé.
type DueByMethod struct {
	None, Cash, Cheque, Transfer int64
}

// Stats sont les chiffres d'une saison. Seules les adhésions validées entrent
// dans les comptes ; Pending compte les demandes en attente.
type Stats struct {
	Validated, First, Renewal, Pending int
	ByFee                              []FeeStat
	TotalCents, PaidCents, DueCents    int64
	DueByMethod                        DueByMethod
	// PaidByMethod détaille PaidCents ; TestPaidCents compte à part les
	// paiements par carte en mode test, exclus de PaidCents.
	PaidByMethod  PaidByMethod
	TestPaidCents int64
}

// ComputeStats calcule les chiffres d'une saison.
func ComputeStats(rows []StatRow) Stats {
	var s Stats
	byFee := map[int64]*FeeStat{}
	for _, r := range rows {
		if r.Status == Pending {
			s.Pending++
			continue
		}
		if r.Status != Validated {
			continue
		}
		s.Validated++
		if r.Kind == Renewal {
			s.Renewal++
		} else {
			s.First++
		}
		f := byFee[r.FeeID]
		if f == nil {
			f = &FeeStat{FeeID: r.FeeID, Label: r.FeeLabel}
			byFee[r.FeeID] = f
		}
		f.Count++
		f.AmountCents += r.AmountCents
		s.TotalCents += r.AmountCents
		switch r.PaymentStatus {
		case Paid:
			received := r.AmountCents
			if r.PaidMethod == Card {
				received = r.PaidAmountCents
			}
			if r.PaidTest {
				s.TestPaidCents += received
				break
			}
			s.PaidCents += received
			switch r.PaidMethod {
			case Card:
				s.PaidByMethod.Card += received
			case Cash:
				s.PaidByMethod.Cash += received
			case Cheque:
				s.PaidByMethod.Cheque += received
			case Transfer:
				s.PaidByMethod.Transfer += received
			}
		case Due, Announced:
			s.DueCents += r.AmountCents
			switch r.AnnouncedMethod {
			case Cash:
				s.DueByMethod.Cash += r.AmountCents
			case Cheque:
				s.DueByMethod.Cheque += r.AmountCents
			case Transfer:
				s.DueByMethod.Transfer += r.AmountCents
			default:
				s.DueByMethod.None += r.AmountCents
			}
		}
	}
	s.ByFee = make([]FeeStat, 0, len(byFee))
	for _, f := range byFee {
		s.ByFee = append(s.ByFee, *f)
	}
	slices.SortFunc(s.ByFee, func(a, b FeeStat) int {
		return cmp.Or(
			cmp.Compare(b.AmountCents, a.AmountCents),
			cmp.Compare(textfold.Fold(a.Label), textfold.Fold(b.Label)),
			cmp.Compare(a.FeeID, b.FeeID),
		)
	})
	return s
}
