package auth

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestKeyedLimiter(t *testing.T) {
	l := NewKeyedLimiter(5, 15*time.Minute)
	for i := range 5 {
		if !l.Allow("claire@exemple.fr") {
			t.Fatalf("tentative %d refusée", i+1)
		}
	}
	if l.Allow("claire@exemple.fr") {
		t.Fatal("6e tentative acceptée")
	}
	if !l.Allow("thomas@exemple.fr") {
		t.Fatal("une autre clé est bloquée")
	}
	l.purge(0)
	if len(l.entries) != 0 {
		t.Fatal("purge sans effet")
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.9, 198.51.100.7")
	if got := ClientIP(r, nil); got != "10.0.0.1" {
		t.Errorf("sans proxy de confiance : %q", got)
	}
	if got := ClientIP(r, []string{"10.0.0.1"}); got != "198.51.100.7" {
		t.Errorf("avec proxy de confiance : %q", got)
	}
}

func TestForwardedUntrusted(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	if ForwardedUntrusted(r, nil) {
		t.Error("sans X-Forwarded-For : rien à signaler")
	}
	r.Header.Set("X-Forwarded-For", "203.0.113.9")
	if !ForwardedUntrusted(r, nil) {
		t.Error("X-Forwarded-For sans proxy de confiance : à signaler")
	}
	if !ForwardedUntrusted(r, []string{"10.0.0.2"}) {
		t.Error("X-Forwarded-For d'un proxy non listé : à signaler")
	}
	if ForwardedUntrusted(r, []string{"10.0.0.1"}) {
		t.Error("X-Forwarded-For d'un proxy de confiance : rien à signaler")
	}
}

// Limiteurs des parcours publics (research.md 016 › R9) : valeurs et purge.
func TestPublicLimits(t *testing.T) {
	l := NewLimits()
	tests := []struct {
		name string
		lim  *KeyedLimiter
		n    int
	}{
		{"PublicSubmitIP", l.PublicSubmitIP, 10},
		{"PublicMailEmail", l.PublicMailEmail, 5},
		{"PublicConfirmIP", l.PublicConfirmIP, 20},
		{"PublicConfirmEmail", l.PublicConfirmEmail, 10},
		{"TrackingRequestIP", l.TrackingRequestIP, 10},
		{"TrackingOpenIP", l.TrackingOpenIP, 30},
		{"TrackingOpenEmail", l.TrackingOpenEmail, 30},
		{"TrackingUseIP", l.TrackingUseIP, 120},
		{"TrackingUseEmail", l.TrackingUseEmail, 120},
	}
	for _, tt := range tests {
		for i := range tt.n {
			if !tt.lim.Allow("cle") {
				t.Fatalf("%s : tentative %d refusée", tt.name, i+1)
			}
		}
		if tt.lim.Allow("cle") {
			t.Errorf("%s : tentative %d acceptée", tt.name, tt.n+1)
		}
		if !tt.lim.Allow("autre") {
			t.Errorf("%s : une autre clé est bloquée", tt.name)
		}
	}
	all := l.all()
	for _, tt := range tests {
		found := false
		for _, k := range all {
			found = found || k == tt.lim
		}
		if !found {
			t.Errorf("%s absent de la purge", tt.name)
		}
	}
	for _, k := range all {
		if k == nil {
			t.Fatal("limiteur nil dans la purge")
		}
	}
}
