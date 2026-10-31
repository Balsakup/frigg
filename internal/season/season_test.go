package season

import (
	"errors"
	"testing"
	"time"
)

var today = MustDate("2026-10-31")

var opened = time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)

// seq construit une suite de saisons : chaque élément vaut "début fin" et
// « + » en suffixe marque une saison ouverte.
func seq(specs ...string) []Season {
	out := make([]Season, len(specs))
	for i, s := range specs {
		out[i] = Season{ID: int64(i + 1), StartsOn: MustDate(s[:10]), EndsOn: MustDate(s[11:21])}
		if len(s) > 21 && s[21] == '+' {
			out[i].OpenedAt = &opened
		}
	}
	return out
}

func TestStateAndCurrent(t *testing.T) {
	cases := []struct {
		name    string
		season  string
		state   State
		current bool
	}{
		{"brouillon en cours", "2026-09-01 2027-08-31", Draft, true},
		{"ouverte en cours", "2026-09-01 2027-08-31+", Open, true},
		{"ouverte finie hier", "2025-09-01 2026-10-30+", Closed, false},
		{"brouillon fini hier", "2025-09-01 2026-10-30", Closed, false},
		{"ouverte finie aujourd'hui", "2025-11-01 2026-10-31+", Open, true},
		{"ouverte commençant aujourd'hui", "2026-10-31 2027-10-30+", Open, true},
		{"ouverte future", "2027-09-01 2028-08-31+", Open, false},
	}
	for _, c := range cases {
		s := seq(c.season)[0]
		if got := StateOf(s, today); got != c.state {
			t.Errorf("%s : état %s, attendu %s", c.name, got, c.state)
		}
		if got := IsCurrent(s, today); got != c.current {
			t.Errorf("%s : courante %v, attendu %v", c.name, got, c.current)
		}
	}
}

func TestLabels(t *testing.T) {
	cases := []struct {
		name  string
		specs []string
		want  []string
	}{
		{"deux années", []string{"2026-09-01 2027-08-31"}, []string{"2026-2027"}},
		{"année civile", []string{"2027-01-01 2027-12-31"}, []string{"2027"}},
		{"trois années", []string{"2026-01-01 2028-06-30"}, []string{"2026-2028"}},
		{"homonymes", []string{"2027-01-01 2027-06-30", "2027-07-01 2027-12-31"}, []string{"2027 (janv.–juin)", "2027 (juil.–déc.)"}},
		{"une seule saison dans l'année", []string{"2026-07-01 2026-12-31", "2027-01-01 2027-12-31"}, []string{"2026", "2027"}},
		{"mois accentués", []string{"2027-02-01 2027-08-31", "2027-09-01 2027-12-31"}, []string{"2027 (févr.–août)", "2027 (sept.–déc.)"}},
	}
	for _, c := range cases {
		labels := Labels(seq(c.specs...))
		for i, want := range c.want {
			if got := labels[int64(i+1)]; got != want {
				t.Errorf("%s : libellé %q, attendu %q", c.name, got, want)
			}
		}
	}
}

func TestNextStart(t *testing.T) {
	if _, ok := NextStart(nil); ok {
		t.Error("aucune saison : pas de début imposé")
	}
	d, ok := NextStart(seq("2026-09-01 2027-08-31", "2027-09-01 2028-08-31"))
	if !ok || d.String() != "2028-09-01" {
		t.Errorf("NextStart = %s, %v", d, ok)
	}
}

func TestActions(t *testing.T) {
	type want struct {
		editStart, editEnd, open, duplicate, del bool
		blockedBy, shifts                        string
	}
	cases := []struct {
		name  string
		specs []string
		i     int
		want  want
	}{
		{"première en brouillon, seule", []string{"2026-09-01 2027-08-31"}, 0,
			want{editStart: true, editEnd: true, open: true, duplicate: true, del: true}},
		{"ouverte, seule", []string{"2026-09-01 2027-08-31+"}, 0,
			want{editEnd: true, duplicate: true}},
		{"ouverte suivie d'un brouillon", []string{"2026-09-01 2027-08-31+", "2027-09-01 2028-08-31"}, 0,
			want{editEnd: true, shifts: "2027-2028"}},
		{"ouverte suivie d'une ouverte", []string{"2026-09-01 2027-08-31+", "2027-09-01 2028-08-31+"}, 0,
			want{}},
		{"brouillon après une ouverte", []string{"2026-09-01 2027-08-31+", "2027-09-01 2028-08-31"}, 1,
			want{editEnd: true, open: true, duplicate: true, del: true}},
		{"brouillon après un brouillon", []string{"2026-09-01 2027-08-31", "2027-09-01 2028-08-31"}, 1,
			want{editEnd: true, duplicate: true, del: true, blockedBy: "2026-2027"}},
		{"brouillon non dernier", []string{"2026-09-01 2027-08-31", "2027-09-01 2028-08-31"}, 0,
			want{editStart: true, editEnd: true, open: true, shifts: "2027-2028"}},
		{"brouillon après une close", []string{"2025-09-01 2026-08-31+", "2026-09-01 2027-08-31"}, 1,
			want{editEnd: true, open: true, duplicate: true, del: true}},
		{"close non dernière", []string{"2025-09-01 2026-08-31+", "2026-09-01 2027-08-31+"}, 0,
			want{}},
		{"close dernière", []string{"2025-09-01 2026-08-31+"}, 0,
			want{duplicate: true}},
		{"close dernière jamais ouverte", []string{"2024-09-01 2025-08-31+", "2025-09-01 2026-08-31"}, 1,
			want{duplicate: true, del: true}},
		{"close dernière dont la copie serait passée", []string{"2024-09-01 2025-08-31+"}, 0,
			want{duplicate: true}},
	}
	for _, c := range cases {
		seasons := seq(c.specs...)
		a := ActionsFor(seasons, c.i, today, Labels(seasons))
		got := want{a.EditStartsOn, a.EditEndsOn, a.Open, a.Duplicate, a.Delete, a.OpenBlockedBy, ShiftsLabel(seasons, c.i, today, Labels(seasons))}
		if got != c.want {
			t.Errorf("%s : %+v, attendu %+v", c.name, got, c.want)
		}
	}
}

func TestEndsOnBounds(t *testing.T) {
	cases := []struct {
		name     string
		specs    []string
		i        int
		min, max string
	}{
		{"saison en cours", []string{"2026-09-01 2027-08-31"}, 0, "2026-10-31", "2028-08-31"},
		{"saison future", []string{"2026-09-01 2027-08-31+", "2027-09-01 2028-08-31"}, 1, "2027-09-02", "2029-08-31"},
		{"suivie d'un brouillon court", []string{"2026-09-01 2027-08-31+", "2027-09-01 2028-02-29"}, 0, "2026-10-31", "2028-02-27"},
		{"suivie d'un brouillon long", []string{"2025-09-01 2027-08-31+", "2027-09-01 2029-08-31"}, 0, "2027-08-31", "2027-08-31"},
	}
	for _, c := range cases {
		lo, hi := EndsOnBounds(seq(c.specs...), c.i, today)
		if lo.String() != c.min || hi.String() != c.max {
			t.Errorf("%s : [%s, %s], attendu [%s, %s]", c.name, lo, hi, c.min, c.max)
		}
	}
}

func TestNewSeasonEndsOnBounds(t *testing.T) {
	if _, _, _, ok := NewSeasonEndsOnBounds(nil); ok {
		t.Error("aucune saison : bornes libres")
	}
	start, lo, hi, ok := NewSeasonEndsOnBounds(seq("2026-09-01 2027-08-31"))
	if !ok || start.String() != "2027-09-01" || lo.String() != "2027-09-02" || hi.String() != "2029-08-31" {
		t.Errorf("bornes : %s %s %s %v", start, lo, hi, ok)
	}
	// Après une interruption, la prochaine saison peut être entièrement passée.
	start, lo, _, _ = NewSeasonEndsOnBounds(seq("2022-09-01 2023-08-31"))
	if start.String() != "2023-09-01" || lo.String() != "2023-09-02" {
		t.Errorf("bornes après interruption : %s %s", start, lo)
	}
}

func TestSetup(t *testing.T) {
	cases := []struct {
		name    string
		specs   []string
		setup   Setup
		current int64
	}{
		{"aucune", nil, SetupMissing, 0},
		{"aucune en cours", []string{"2025-09-01 2026-08-31+"}, SetupNoCurrent, 0},
		{"courante en brouillon", []string{"2025-09-01 2026-08-31+", "2026-09-01 2027-08-31"}, SetupCurrentDraft, 2},
		{"courante ouverte", []string{"2026-09-01 2027-08-31+", "2027-09-01 2028-08-31"}, SetupReady, 1},
	}
	for _, c := range cases {
		setup, current := SetupOf(seq(c.specs...), today)
		if setup != c.setup || current != c.current {
			t.Errorf("%s : (%s, %d), attendu (%s, %d)", c.name, setup, current, c.setup, c.current)
		}
	}
}

func fieldErr(t *testing.T, err error, field string, code error) {
	t.Helper()
	var fe *FieldError
	if !errors.As(err, &fe) || fe.Field != field || !errors.Is(err, code) {
		t.Errorf("erreur %v, attendu %s sur %s", err, code, field)
	}
}

func TestValidateCreate(t *testing.T) {
	d := MustDate
	if err := ValidateCreate(nil, d("2026-09-01"), d("2027-08-31"), today); err != nil {
		t.Errorf("première saison commencée : %v", err)
	}
	fieldErr(t, ValidateCreate(nil, d("2026-09-01"), d("2026-09-01"), today), "endsOn", ErrEndBeforeStart)
	fieldErr(t, ValidateCreate(nil, d("2026-09-01"), d("2026-10-30"), today), "endsOn", ErrEndInPast)
	fieldErr(t, ValidateCreate(nil, d("2026-09-01"), d("2028-09-01"), today), "endsOn", ErrTooLong)
	if err := ValidateCreate(nil, d("2026-09-01"), d("2028-08-31"), today); err != nil {
		t.Errorf("24 mois exactement : %v", err)
	}
	existing := seq("2026-09-01 2027-08-31")
	if err := ValidateCreate(existing, d("2027-08-01"), d("2028-08-31"), today); !errors.Is(err, ErrSeasonsChanged) {
		t.Errorf("début non imposé : %v", err)
	}
	if err := ValidateCreate(existing, d("2027-09-01"), d("2028-08-31"), today); err != nil {
		t.Errorf("début imposé : %v", err)
	}
	// Trou à combler après une interruption de plus de 24 mois : fin passée
	// acceptée pour une saison qui n'est pas la première.
	old := seq("2022-09-01 2023-08-31+")
	if err := ValidateCreate(old, d("2023-09-01"), d("2024-08-31"), today); err != nil {
		t.Errorf("saison passée comblant un trou : %v", err)
	}
	fieldErr(t, ValidateCreate(old, d("2023-09-01"), d("2023-09-01"), today), "endsOn", ErrEndBeforeStart)
	start, end, err := ValidateDuplicate(old, 0)
	if err != nil || start.String() != "2023-09-01" || end.String() != "2024-08-31" {
		t.Errorf("duplication d'une saison ancienne : %s %s %v", start, end, err)
	}
	if _, _, err := ValidateDuplicate(seq("2022-09-01 2023-08-31", "2023-09-01 2024-08-31"), 0); !errors.Is(err, ErrSeasonsChanged) {
		t.Errorf("duplication d'une saison non dernière : %v", err)
	}
}

func TestDuplicateDates(t *testing.T) {
	cases := []struct{ src, start, end string }{
		{"2026-09-01 2027-08-31", "2027-09-01", "2028-08-31"},
		{"2027-01-01 2027-06-30", "2027-07-01", "2027-12-31"},
		{"2027-04-01 2027-09-30", "2027-10-01", "2028-03-31"},
		{"2028-03-01 2029-02-28", "2029-03-01", "2030-02-28"},
		{"2026-09-15 2027-09-14", "2027-09-15", "2028-09-13"},
	}
	for _, c := range cases {
		start, end := DuplicateDates(seq(c.src)[0])
		if start.String() != c.start || end.String() != c.end {
			t.Errorf("DuplicateDates(%s) = %s %s, attendu %s %s", c.src, start, end, c.start, c.end)
		}
	}
}

func TestCanOpen(t *testing.T) {
	seasons := seq("2026-09-01 2027-08-31", "2027-09-01 2028-08-31")
	var le *LabelError
	if err := CanOpen(seasons, 1, today); !errors.As(err, &le) || !errors.Is(err, ErrPreviousDraft) || le.Label != "2026-2027" {
		t.Errorf("précédente en brouillon : %v", err)
	}
	if err := CanOpen(seasons, 0, today); err != nil {
		t.Errorf("première : %v", err)
	}
	if err := CanOpen(seq("2025-09-01 2026-08-31"), 0, today); !errors.Is(err, ErrClosed) {
		t.Errorf("close : %v", err)
	}
	if err := CanOpen(seq("2026-09-01 2027-08-31+"), 0, today); err != nil {
		t.Errorf("déjà ouverte : %v", err)
	}
}

func TestValidateUpdate(t *testing.T) {
	d := MustDate
	ptr := func(s string) *Date { v := d(s); return &v }

	p, err := ValidateUpdate(seq("2026-09-01 2027-08-31"), 0, ptr("2026-08-15"), d("2027-07-31"), today)
	if err != nil || p.StartsOn.String() != "2026-08-15" || p.EndsOn.String() != "2027-07-31" || p.NextID != 0 {
		t.Errorf("première en brouillon : %+v %v", p, err)
	}
	if _, err := ValidateUpdate(seq("2026-09-01 2027-08-31+"), 0, ptr("2026-08-15"), d("2027-08-31"), today); !errors.Is(err, ErrStartLocked) {
		t.Errorf("début d'une ouverte : %v", err)
	}
	if _, err := ValidateUpdate(seq("2026-09-01 2027-08-31+", "2027-09-01 2028-08-31"), 1, ptr("2027-08-01"), d("2028-08-31"), today); !errors.Is(err, ErrStartLocked) {
		t.Errorf("début d'une saison non première : %v", err)
	}
	if _, err := ValidateUpdate(seq("2026-09-01 2027-08-31+"), 0, ptr("2026-09-01"), d("2027-07-31"), today); err != nil {
		t.Errorf("début inchangé fourni : %v", err)
	}
	if _, err := ValidateUpdate(seq("2025-09-01 2026-08-31+"), 0, nil, d("2026-12-31"), today); !errors.Is(err, ErrClosed) {
		t.Errorf("close : %v", err)
	}
	_, err = ValidateUpdate(seq("2026-09-01 2027-08-31+"), 0, nil, d("2026-10-30"), today)
	fieldErr(t, err, "endsOn", ErrEndInPast)
	if _, err := ValidateUpdate(seq("2026-09-01 2027-08-31+"), 0, nil, d("2026-10-31"), today); err != nil {
		t.Errorf("fin aujourd'hui : %v", err)
	}
	p, err = ValidateUpdate(seq("2026-09-01 2027-08-31+", "2027-09-01 2028-08-31"), 0, nil, d("2027-08-15"), today)
	if err != nil || p.NextID != 2 || p.NextStartsOn.String() != "2027-08-16" {
		t.Errorf("décalage de la suivante : %+v %v", p, err)
	}
	if _, err := ValidateUpdate(seq("2026-09-01 2027-08-31+", "2027-09-01 2028-08-31+"), 0, nil, d("2027-08-15"), today); !errors.Is(err, ErrNextOpen) {
		t.Errorf("suivante ouverte : %v", err)
	}
	_, err = ValidateUpdate(seq("2026-09-01 2027-08-31+", "2027-09-01 2028-08-31"), 0, nil, d("2028-08-30"), today)
	fieldErr(t, err, "endsOn", ErrNextInvalid)
	var fe *FieldError
	if errors.As(err, &fe) && fe.Label != "2027-2028" {
		t.Errorf("libellé de la suivante : %q", fe.Label)
	}
	_, err = ValidateUpdate(seq("2026-09-01 2027-08-31+", "2027-09-01 2028-08-31"), 0, nil, d("2026-08-31"), today)
	fieldErr(t, err, "endsOn", ErrEndBeforeStart)
	_, err = ValidateUpdate(seq("2025-09-01 2027-08-31+", "2027-09-01 2029-08-31"), 0, nil, d("2027-06-30"), today)
	fieldErr(t, err, "endsOn", ErrNextInvalid)
}

func TestCanDelete(t *testing.T) {
	if err := CanDelete(seq("2026-09-01 2027-08-31"), 0); err != nil {
		t.Errorf("dernier brouillon : %v", err)
	}
	if err := CanDelete(seq("2026-09-01 2027-08-31+"), 0); !errors.Is(err, ErrDeleteOpened) {
		t.Errorf("ouverte : %v", err)
	}
	if err := CanDelete(seq("2025-09-01 2026-08-31+"), 0); !errors.Is(err, ErrDeleteOpened) {
		t.Errorf("close après ouverture : %v", err)
	}
	if err := CanDelete(seq("2022-09-01 2023-08-31+", "2023-09-01 2024-08-31"), 1); err != nil {
		t.Errorf("close jamais ouverte (comblement raté) : %v", err)
	}
	if err := CanDelete(seq("2026-09-01 2027-08-31", "2027-09-01 2028-08-31"), 0); !errors.Is(err, ErrDeleteLastOnly) {
		t.Errorf("brouillon non dernier : %v", err)
	}
}
