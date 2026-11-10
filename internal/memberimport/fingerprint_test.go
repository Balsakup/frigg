package memberimport

import (
	"regexp"
	"slices"
	"testing"
)

func TestFingerprint(t *testing.T) {
	defs := defsFixture()
	fp := Fingerprint(defs)
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(fp) {
		t.Fatalf("empreinte %q", fp)
	}
	reversed := slices.Clone(defs)
	slices.Reverse(reversed)
	if Fingerprint(reversed) != fp {
		t.Error("dépend de l'ordre des champs")
	}
	changed := defsFixture()
	changed[1].Choices[0].Label = "Poussins"
	if Fingerprint(changed) == fp {
		t.Error("insensible au libellé d'un choix")
	}
	if Fingerprint(defs[1:]) == fp {
		t.Error("insensible à un champ archivé")
	}
	if Fingerprint(nil) == fp {
		t.Error("vide identique")
	}
}
