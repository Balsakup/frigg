package api

import (
	"testing"
	"time"
)

func TestFrenchDateTimeRange(t *testing.T) {
	at := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	cases := []struct{ start, end, want string }{
		{"2026-12-12T13:00:00Z", "2026-12-12T16:00:00Z", "samedi 12 décembre 2026 de 14 h à 17 h"},
		{"2027-04-01T07:30:00Z", "2027-04-01T15:00:00Z", "jeudi 1er avril 2027 de 9 h 30 à 17 h"},
		{"2027-04-12T07:00:00Z", "2027-04-16T15:00:00Z", "du lundi 12 avril 2027 à 9 h au vendredi 16 avril 2027 à 17 h"},
		// Passage à l'heure d'hiver le 25 octobre 2026.
		{"2026-10-24T22:00:00Z", "2026-10-25T08:00:00Z", "dimanche 25 octobre 2026 de 0 h à 9 h"},
	}
	for _, c := range cases {
		if got := frenchDateTimeRange(at(c.start), at(c.end)); got != c.want {
			t.Errorf("%s → %s : %q, attendu %q", c.start, c.end, got, c.want)
		}
	}
}
