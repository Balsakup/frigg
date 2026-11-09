package event

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
)

var seasons = []SeasonSpan{
	{ID: 1, StartsOn: season.MustDate("2025-09-01"), EndsOn: season.MustDate("2026-08-31")},
	{ID: 2, StartsOn: season.MustDate("2026-09-01"), EndsOn: season.MustDate("2027-08-31")},
}

func ptr[T any](v T) *T { return &v }

func valid() Input {
	return Input{
		Title: "Assemblée générale", Location: "Salle polyvalente",
		StartsAt: "2026-12-12T14:00", EndsAt: "2026-12-12T17:00",
		Openness: SeasonMembers,
	}
}

func fieldErr(t *testing.T, err error, field string, want error) {
	t.Helper()
	var fes season.FieldErrors
	if !errors.As(err, &fes) {
		t.Fatalf("erreur %v, attendu des erreurs de champ", err)
	}
	for _, fe := range fes {
		if fe.Field == field && errors.Is(fe.Err, want) {
			return
		}
	}
	t.Fatalf("%s : %v absent de %v", field, want, err)
}

func TestValidateEvent(t *testing.T) {
	ev, err := ValidateEvent(valid(), seasons)
	if err != nil {
		t.Fatal(err)
	}
	if ev.SeasonID != 2 || ev.StartsAt.Format(time.RFC3339) != "2026-12-12T13:00:00Z" {
		t.Fatalf("événement : %+v", ev)
	}

	cases := []struct {
		name  string
		edit  func(*Input)
		field string
		err   error
	}{
		{"titre vide", func(i *Input) { i.Title = "   " }, "title", ErrRequired},
		{"titre 121", func(i *Input) { i.Title = strings.Repeat("é", 121) }, "title", ErrTooLong},
		{"lieu 201", func(i *Input) { i.Location = strings.Repeat("à", 201) }, "location", ErrTooLong},
		{"description 5001", func(i *Input) { i.Description = strings.Repeat("a", 5001) }, "description", ErrTooLong},
		{"fin = début", func(i *Input) { i.EndsAt = i.StartsAt }, "endsAt", ErrEndBeforeStart},
		{"avant la première saison", func(i *Input) { i.StartsAt, i.EndsAt = "2025-08-31T10:00", "2025-08-31T12:00" }, "startsAt", ErrNoSeason},
		{"après la dernière saison", func(i *Input) { i.StartsAt, i.EndsAt = "2027-09-01T10:00", "2027-09-01T12:00" }, "startsAt", ErrNoSeason},
		{"heure inexistante", func(i *Input) { i.StartsAt, i.EndsAt = "2027-03-28T02:30", "2027-03-28T05:00" }, "startsAt", ErrNonexistentTime},
		{"format", func(i *Input) { i.StartsAt = "12/12/2026 14:00" }, "startsAt", ErrDateTime},
		{"capacité 0", func(i *Input) { i.Capacity = ptr(0) }, "capacity", ErrCapacity},
		{"capacité 10001", func(i *Input) { i.Capacity = ptr(10001) }, "capacity", ErrCapacity},
		{"prix", func(i *Input) { i.PriceCents = 1_000_001 }, "price", ErrPrice},
		{"ouverture", func(i *Input) { i.Openness = "tous" }, "openness", ErrOpenness},
		{"clôture après la fin", func(i *Input) { i.RegistrationClosesAt = ptr("2026-12-12T18:00") }, "registrationClosesAt", ErrCloseAfterEnd},
		{"ouverture après clôture", func(i *Input) {
			i.RegistrationOpensAt, i.RegistrationClosesAt = ptr("2026-12-01T10:00"), ptr("2026-11-30T10:00")
		}, "registrationClosesAt", ErrCloseBeforeOpen},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := valid()
			c.edit(&in)
			_, err := ValidateEvent(in, seasons)
			fieldErr(t, err, c.field, c.err)
		})
	}

	for _, c := range []struct {
		name string
		edit func(*Input)
	}{
		{"titre 120 avec accents", func(i *Input) { i.Title = strings.Repeat("é", 120) }},
		{"capacité 1 et 10000", func(i *Input) { i.Capacity = ptr(10000) }},
		{"prix maximal", func(i *Input) { i.PriceCents = 1_000_000 }},
		{"heure ambiguë", func(i *Input) { i.StartsAt, i.EndsAt = "2026-10-25T02:30", "2026-10-25T05:00" }},
		{"période vide", func(i *Input) { i.RegistrationOpensAt, i.RegistrationClosesAt = ptr(""), ptr(" ") }},
	} {
		t.Run(c.name, func(t *testing.T) {
			in := valid()
			c.edit(&in)
			if _, err := ValidateEvent(in, seasons); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSeasonOfLocalDate(t *testing.T) {
	// 31 août 23 h 30 à Paris = 21 h 30 UTC : saison qui finit le 31 août.
	in := valid()
	in.StartsAt, in.EndsAt = "2026-08-31T23:30", "2026-09-01T01:00"
	ev, err := ValidateEvent(in, seasons)
	if err != nil || ev.SeasonID != 1 {
		t.Fatalf("saison = %d (%v), attendu 1", ev.SeasonID, err)
	}
}

func TestParseLocalAmbiguousTakesFirst(t *testing.T) {
	got, err := ParseLocal("2026-10-25T02:30")
	if err != nil {
		t.Fatal(err)
	}
	if got.UTC().Format(time.RFC3339) != "2026-10-25T00:30:00Z" {
		t.Fatalf("heure ambiguë : %s, attendu la première occurrence (00:30 UTC)", got.UTC())
	}
	if FormatLocal(got) != "2026-10-25T02:30" {
		t.Fatalf("FormatLocal = %s", FormatLocal(got))
	}
}

func TestSeasonBounds(t *testing.T) {
	from, to := SeasonBounds(season.MustDate("2026-09-01"), season.MustDate("2027-08-31"))
	if from.Format(time.RFC3339) != "2026-08-31T22:00:00Z" || to.Format(time.RFC3339) != "2027-08-31T22:00:00Z" {
		t.Fatalf("bornes : %s %s", from, to)
	}
}

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func published() State {
	return State{
		Status: Published, PublishedAt: ptr(at("2026-11-01T10:00:00Z")),
		StartsAt: at("2026-12-12T13:00:00Z"), EndsAt: at("2026-12-12T16:00:00Z"),
	}
}

func TestPhaseAndWindow(t *testing.T) {
	s := published()
	for _, c := range []struct {
		now  string
		want Phase
	}{
		{"2026-12-12T15:59:59Z", PhaseUpcoming},
		{"2026-12-12T16:00:00Z", PhaseFinished},
	} {
		if got := PhaseOf(s, at(c.now)); got != c.want {
			t.Errorf("phase à %s = %s, attendu %s", c.now, got, c.want)
		}
	}
	if PhaseOf(State{Status: Draft}, at("2030-01-01T00:00:00Z")) != PhaseDraft {
		t.Error("brouillon")
	}
	c := s
	c.Status = Cancelled
	if PhaseOf(c, at("2026-11-02T00:00:00Z")) != PhaseCancelled {
		t.Error("annulé")
	}

	// Période par défaut : de la publication au début.
	if WindowOpen(s, at("2026-10-31T10:00:00Z")) || !WindowOpen(s, at("2026-11-01T10:00:00Z")) ||
		!WindowOpen(s, at("2026-12-12T12:59:59Z")) || WindowOpen(s, at("2026-12-12T13:00:00Z")) {
		t.Error("période par défaut")
	}
	s.RegistrationClosesAt = ptr(at("2026-12-01T00:00:00Z"))
	if CanRegister(s, at("2026-12-02T00:00:00Z")) || !CanAdminRegister(s, at("2026-12-02T00:00:00Z")) {
		t.Error("clôture explicite")
	}
	if CanAdminRegister(s, at("2026-12-12T16:00:00Z")) {
		t.Error("inscription admin après la fin")
	}
}

func TestEligibility(t *testing.T) {
	cases := []struct {
		o               Openness
		member, already bool
		want            Ineligibility
	}{
		{SeasonMembers, true, false, Eligible},
		{SeasonMembers, false, false, NotSeasonMember},
		{AllMembers, false, false, Eligible},
		{AllMembers, false, true, AlreadyRegistered},
	}
	for _, c := range cases {
		if got := Eligibility(c.o, c.member, c.already); got != c.want {
			t.Errorf("%v : %q, attendu %q", c, got, c.want)
		}
	}
}

func TestPlaces(t *testing.T) {
	if CheckPlaces(nil, 500, 20) != nil {
		t.Error("illimité")
	}
	var full *ErrFull
	if err := CheckPlaces(ptr(3), 2, 2); !errors.As(err, &full) || full.Left != 1 {
		t.Errorf("1 place pour 2 : %v", err)
	}
	if CheckPlaces(ptr(3), 2, 1) != nil {
		t.Error("dernière place")
	}
	if *CapacityLeft(ptr(3), 5) != 0 {
		t.Error("restant négatif")
	}
	if CheckCapacityChange(ptr(1), 2) == nil || CheckCapacityChange(ptr(2), 2) != nil || CheckCapacityChange(nil, 9) != nil {
		t.Error("changement de capacité")
	}
}

func TestUnregister(t *testing.T) {
	s, now := published(), at("2026-11-10T00:00:00Z")
	r := RegSnapshot{Confirmed: true, AmountCents: 3500, PaymentStatus: membership.Due}
	if !CanUnregister(s, r, now) {
		t.Error("à payer")
	}
	r.PaymentStatus = membership.Announced
	if !CanUnregister(s, r, now) {
		t.Error("paiement annoncé")
	}
	r.CardPending = true
	if CanUnregister(s, r, now) {
		t.Error("carte en cours")
	}
	r = RegSnapshot{Confirmed: true, PaymentStatus: membership.Paid, PaidMethod: membership.Cheque}
	if CanUnregister(s, r, now) {
		t.Error("payée")
	}
	if !CanRemove(s, r, now) || CanRemove(s, r, at("2026-12-12T16:00:00Z")) {
		t.Error("retrait admin")
	}
	if CanUnregister(s, RegSnapshot{Confirmed: true}, at("2026-12-12T13:00:00Z")) {
		t.Error("période close")
	}
}

func TestPaymentSnapshotWithSettleOnline(t *testing.T) {
	due := RegSnapshot{Confirmed: true, AmountCents: 3500, PaymentStatus: membership.Due}
	v := membership.Verified{Authorized: true, ReceivedCents: 3500}
	cases := []struct {
		name      string
		r         RegSnapshot
		cancelled bool
		received  int64
		record    bool
		alert     membership.AlertKind
	}{
		{"confirmée", due, false, 3500, true, membership.NoAlert},
		{"montant différent", due, false, 3000, true, membership.AlertAmountMismatch},
		{"inscription annulée", RegSnapshot{AmountCents: 3500, PaymentStatus: membership.Due}, false, 3500, false, membership.AlertRefundDue},
		{"événement annulé", due, true, 3500, false, membership.AlertRefundDue},
		{"déjà payée", RegSnapshot{Confirmed: true, AmountCents: 3500, PaymentStatus: membership.Paid, PaidMethod: membership.Cheque}, false, 3500, false, membership.AlertDuplicate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v.ReceivedCents = c.received
			got := membership.SettleOnline(membership.OnlinePending, PaymentSnapshot(c.r, c.cancelled), v)
			if got.RecordPayment != c.record || got.Alert != c.alert {
				t.Fatalf("%+v", got)
			}
		})
	}
	paid := RegSnapshot{Confirmed: true, AmountCents: 3500, PaymentStatus: membership.Paid, PaidMethod: membership.Card}
	if !membership.RefundCancels(membership.OnlineSucceeded, PaymentSnapshot(paid, false), true) {
		t.Error("remboursement d'une inscription payée par carte")
	}
	if membership.CanCancelPayment(PaymentSnapshot(paid, false)) {
		t.Error("annulation d'une constatation par carte")
	}
}

func TestReferencePrefix(t *testing.T) {
	if got := ReferencePrefix(season.MustDate("2026-09-01"), season.MustDate("2027-08-31")); got != "EVT-2627-" {
		t.Fatalf("préfixe = %s", got)
	}
	if got := membership.NextReference("EVT-2627-", []string{"EVT-2627-0001", "EVT-2627-0009"}); got != "EVT-2627-0010" {
		t.Fatalf("suivante = %s", got)
	}
}
