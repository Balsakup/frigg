package season

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	d, err := ParseDate("2026-09-01")
	if err != nil {
		t.Fatal(err)
	}
	if d.String() != "2026-09-01" {
		t.Errorf("String() = %q", d.String())
	}
	for _, bad := range []string{"2026-02-30", "2026-9-1", "", "2026-13-01", "01/09/2026"} {
		if _, err := ParseDate(bad); err == nil {
			t.Errorf("ParseDate(%q) aurait dû échouer", bad)
		}
	}
}

func TestDateArithmetic(t *testing.T) {
	cases := []struct {
		got, want string
	}{
		{MustDate("2026-03-01").AddDays(-1).String(), "2026-02-28"},
		{MustDate("2028-03-01").AddDays(-1).String(), "2028-02-29"},
		{MustDate("2026-12-31").AddDays(1).String(), "2027-01-01"},
		{MustDate("2026-09-01").AddMonths(12).String(), "2027-09-01"},
		{MustDate("2027-01-01").AddMonths(-24).String(), "2025-01-01"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("obtenu %s, attendu %s", c.got, c.want)
		}
	}
	a, b := MustDate("2026-09-01"), MustDate("2026-09-02")
	if !a.Before(b) || !b.After(a) || a.Equal(b) || !a.Equal(MustDate("2026-09-01")) {
		t.Error("comparaisons incorrectes")
	}
	if a.Compare(b) != -1 || b.Compare(a) != 1 || a.Compare(a) != 0 {
		t.Error("Compare incorrect")
	}
	if !MustDate("2026-09-01").IsFirstOfMonth() || MustDate("2026-09-02").IsFirstOfMonth() {
		t.Error("IsFirstOfMonth incorrect")
	}
	if !MustDate("2028-02-29").IsLastOfMonth() || MustDate("2028-02-28").IsLastOfMonth() {
		t.Error("IsLastOfMonth incorrect")
	}
}

func TestToday(t *testing.T) {
	cases := []struct {
		now  string
		want string
	}{
		{"2026-10-31T23:30:00Z", "2026-11-01"}, // 00 h 30 à Paris (heure d'hiver)
		{"2026-10-31T22:30:00Z", "2026-10-31"},
		{"2026-06-30T22:30:00Z", "2026-07-01"}, // heure d'été
	}
	for _, c := range cases {
		now, _ := time.Parse(time.RFC3339, c.now)
		if got := Today(now).String(); got != c.want {
			t.Errorf("Today(%s) = %s, attendu %s", c.now, got, c.want)
		}
	}
}

func TestMonthsCovered(t *testing.T) {
	cases := []struct {
		start, end string
		n          int
		ok         bool
	}{
		{"2026-09-01", "2027-08-31", 12, true},
		{"2027-01-01", "2027-06-30", 6, true},
		{"2028-03-01", "2029-02-28", 12, true},
		{"2026-09-15", "2027-09-14", 0, false},
		{"2026-09-01", "2027-08-30", 0, false},
	}
	for _, c := range cases {
		n, ok := MonthsCovered(MustDate(c.start), MustDate(c.end))
		if n != c.n || ok != c.ok {
			t.Errorf("MonthsCovered(%s, %s) = (%d, %v), attendu (%d, %v)", c.start, c.end, n, ok, c.n, c.ok)
		}
	}
}
