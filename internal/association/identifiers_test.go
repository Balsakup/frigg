package association

import (
	"strings"
	"testing"
)

func TestNormalizeRNA(t *testing.T) {
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{"W751234567", "W751234567", true},
		{"w751234567", "W751234567", true},
		{"W12345678", "", false},
		{"X751234567", "", false},
		{"W75123456A", "", false},
	} {
		got, ok := NormalizeRNA(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("NormalizeRNA(%q) = %q, %v", c.in, got, ok)
		}
	}
}

func TestNormalizeSIRET(t *testing.T) {
	for _, c := range []struct{ in, want, msg string }{
		{"73282932000074", "73282932000074", ""},
		{"732 829 320 00074", "73282932000074", ""},
		{"73282932000075", "", MsgSIRETInvalid},
		{"7328293200007", "", MsgSIRETLength},
		{"7328293200007A", "", MsgSIRETLength},
		// Établissement de La Poste : somme des chiffres multiple de 5.
		{"35600000000001", "35600000000001", ""},
		{"35600000000002", "", MsgSIRETInvalid},
	} {
		got, msg := NormalizeSIRET(c.in)
		if msg != c.msg || (msg == "" && got != c.want) {
			t.Errorf("NormalizeSIRET(%q) = %q, %q", c.in, got, msg)
		}
	}
}

func TestNormalizePhone(t *testing.T) {
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{"01 23 45 67 89", "0123456789", true},
		{"01.23.45.67.89", "0123456789", true},
		{"06-12-34-56-78", "0612345678", true},
		{"+33 1 23 45 67 89", "0123456789", true},
		{"0033123456789", "0123456789", true},
		{"+262 262 12 34 56", "0262123456", true},
		{"+590 590 12 34 56", "0590123456", true},
		{"00 12 34 56 78", "", false},
		{"+44 20 7946 0958", "", false},
		{"+33 01 23 45 67 89", "", false},
		{"123", "", false},
		{"01 23 45 67 8a", "", false},
	} {
		got, ok := NormalizePhone(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("NormalizePhone(%q) = %q, %v", c.in, got, ok)
		}
	}
}

func TestValidWebURL(t *testing.T) {
	for _, c := range []struct {
		in string
		ok bool
	}{
		{"https://exemple.fr", true},
		{"http://exemple.fr/page?a=1", true},
		{"https://www.instagram.com/sentiers.du.val/", true},
		{"www.exemple.fr", false},
		{"javascript:alert(1)", false},
		{"https://", false},
		{"ftp://exemple.fr", false},
		{"https://exemple .fr", false},
		{"https://exemple.fr/" + strings.Repeat("a", 282), false},
	} {
		if got := ValidWebURL(c.in); got != c.ok {
			t.Errorf("ValidWebURL(%q) = %v", c.in, got)
		}
	}
}
