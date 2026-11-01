package membership

import "testing"

func TestPreviousSeasonHint(t *testing.T) {
	cases := []struct {
		name              string
		hasPrevious       bool
		previousValidated int64
		memberValidated   bool
		want              Hint
	}{
		{"pas de saison précédente", false, 0, false, HintUnknown},
		{"saison précédente hors Frigg", true, 0, false, HintUnknown},
		{"autre membre adhérent", true, 3, false, HintNo},
		{"membre adhérent", true, 3, true, HintYes},
	}
	for _, c := range cases {
		if got := PreviousSeasonHint(c.hasPrevious, c.previousValidated, c.memberValidated); got != c.want {
			t.Errorf("%s : %q, attendu %q", c.name, got, c.want)
		}
	}
}
