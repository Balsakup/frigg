package api

import "testing"

func TestFormatEuros(t *testing.T) {
	cases := map[int64]string{
		0:         "0,00 €",
		4000:      "40,00 €",
		1550:      "15,50 €",
		125000:    "1 250,00 €",
		100000000: "1 000 000,00 €",
	}
	for cents, want := range cases {
		if got := formatEuros(cents); got != want {
			t.Errorf("%d : %q, attendu %q", cents, got, want)
		}
	}
}
