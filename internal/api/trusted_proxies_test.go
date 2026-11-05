package api_test

import (
	"net/http"
	"testing"

	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/config"
)

const trustedProxiesWarning = "X-Forwarded-For reçu d'un proxy absent de FRIGG_TRUSTED_PROXIES"

func countTrustedProxiesWarnings(t *testing.T, env *apitest.Env) int {
	t.Helper()
	n := 0
	for _, l := range logLines(t, env) {
		if l["msg"] == trustedProxiesWarning {
			if l["level"] != "WARN" {
				t.Errorf("niveau %v, attendu WARN", l["level"])
			}
			n++
		}
	}
	return n
}

// Derrière un proxy non déclaré, toutes les requêtes partagent l'IP du proxy
// pour la limitation de débit : Frigg l'avertit une fois dans les journaux.
func TestTrustedProxiesWarning(t *testing.T) {
	env := apitest.New(t)
	c := env.Client()
	for range 3 {
		if res := c.Do("GET", "/api/health", nil, "X-Forwarded-For", "203.0.113.9"); res.Status != http.StatusOK {
			t.Fatalf("statut %d", res.Status)
		}
	}
	if n := countTrustedProxiesWarnings(t, env); n != 1 {
		t.Errorf("%d avertissements, attendu 1", n)
	}
}

func TestTrustedProxiesNoWarning(t *testing.T) {
	t.Run("sans X-Forwarded-For", func(t *testing.T) {
		env := apitest.New(t)
		env.Client().Do("GET", "/api/health", nil)
		if n := countTrustedProxiesWarnings(t, env); n != 0 {
			t.Errorf("%d avertissements, attendu 0", n)
		}
	})
	t.Run("proxy de confiance", func(t *testing.T) {
		env := apitest.New(t, func(c *config.Config) { c.TrustedProxies = []string{"127.0.0.1", "::1"} })
		env.Client().Do("GET", "/api/health", nil, "X-Forwarded-For", "203.0.113.9")
		if n := countTrustedProxiesWarnings(t, env); n != 0 {
			t.Errorf("%d avertissements, attendu 0", n)
		}
	})
}
