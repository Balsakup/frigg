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
