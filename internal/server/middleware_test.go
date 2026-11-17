package server

import "testing"

func TestRedactPath(t *testing.T) {
	for in, want := range map[string]string{
		"/api/webhooks/helloasso/cle-secrete":                     "/api/webhooks/helloasso/…",
		"/api/public/events/AAAAAAAAAAAAAAAAAAAAAA":               "/api/public/events/…",
		"/api/public/events/AAAAAAAAAAAAAAAAAAAAAA/registrations": "/api/public/events/…/registrations",
		"/e/AAAAAAAAAAAAAAAAAAAAAA":                               "/e/…",
		"/api/public/membership":                                  "/api/public/membership",
		"/evenements/3":                                           "/evenements/3",
	} {
		if got := redactPath(in); got != want {
			t.Errorf("redactPath(%q) = %q, attendu %q", in, got, want)
		}
	}
}
