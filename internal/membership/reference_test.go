package membership

import (
	"testing"

	"github.com/balsakup/frigg/internal/season"
)

func TestReferences(t *testing.T) {
	p := ReferencePrefix(season.MustDate("2026-09-01"), season.MustDate("2027-08-31"))
	if p != "ADH-2627-" {
		t.Fatalf("préfixe : %q", p)
	}
	if c := ReferencePrefix(season.MustDate("2027-01-01"), season.MustDate("2027-12-31")); c != "ADH-2727-" {
		t.Errorf("saison civile : %q", c)
	}
	cases := []struct {
		existing []string
		want     string
	}{
		{nil, "ADH-2627-0001"},
		{[]string{"ADH-2627-0041", "ADH-2627-0007"}, "ADH-2627-0042"},
		{[]string{"ADH-2627-9999"}, "ADH-2627-10000"},
		{[]string{"ADH-2627-10000", "ADH-2627-9999"}, "ADH-2627-10001"},
	}
	for _, c := range cases {
		if got := NextReference(p, c.existing); got != c.want {
			t.Errorf("%v : %q, attendu %q", c.existing, got, c.want)
		}
	}
}
