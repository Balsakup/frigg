package api_test

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Ouverture d'un lien personnel (research.md 016 › R1) : jeton dans le
// corps, réponse unique pour un lien inconnu, expiré ou révoqué, session de
// compte conservée et fermeture limitée à l'accès par lien.
func TestOpenTrackingLink(t *testing.T) {
	ctx := context.Background()
	env := apitest.New(t)
	issue := func(email string, expiresIn time.Duration) string {
		t.Helper()
		raw, hash := auth.NewToken()
		now := time.Now().UTC()
		if _, err := env.Store.Write.CreateTrackingLink(ctx, sqlc.CreateTrackingLinkParams{
			Email: email, TokenHash: hash,
			ExpiresAt: now.Add(expiresIn).Format("2006-01-02T15:04:05Z"),
			CreatedAt: now.Format("2006-01-02T15:04:05Z"),
		}); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	open := func(c *apitest.Client, token string) apitest.Response {
		t.Helper()
		return c.Do("POST", "/api/public/tracking/open", map[string]string{"token": token})
	}

	unknown := open(env.Client(), "inconnu")
	expired := open(env.Client(), issue("a@exemple.fr", -time.Minute))
	revokedToken := issue("b@exemple.fr", time.Hour)
	if _, err := env.Store.Write.RevokeTrackingLinksOfEmail(ctx, sqlc.RevokeTrackingLinksOfEmailParams{
		Email: "B@Exemple.fr", Now: sql.NullString{String: "2000-01-01T00:00:00Z", Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	revoked := open(env.Client(), revokedToken)
	for name, res := range map[string]apitest.Response{"inconnu": unknown, "expiré": expired, "révoqué": revoked} {
		if res.Status != http.StatusGone || !strings.Contains(res.Body, "Ce lien n'est plus valable.") {
			t.Errorf("lien %s : %d %s", name, res.Status, res.Body)
		}
	}

	// Un compte connecté ouvre un lien : sa session de compte reste ouverte,
	// et fermer l'accès par lien ne la ferme pas.
	email := env.Seed(auth.RoleMember, true)
	c := env.LoggedIn(email)
	if res := open(c, issue("c@exemple.fr", time.Hour)); res.Status != http.StatusNoContent {
		t.Fatalf("ouverture : %d %s", res.Status, res.Body)
	}
	if res := c.Do("GET", "/api/session", nil); res.Status != http.StatusOK {
		t.Fatalf("session de compte perdue : %d", res.Status)
	}
	var touched int
	if err := env.Store.ReadDB.QueryRow(`SELECT count(*) FROM tracking_links WHERE email = 'c@exemple.fr' AND last_used_at IS NULL`).Scan(&touched); err != nil || touched != 1 {
		t.Fatalf("lien utilisé avant tout accès : %d (%v)", touched, err)
	}
	if res := c.Do("POST", "/api/tracking/close", nil); res.Status != http.StatusNoContent {
		t.Fatalf("fermeture : %d %s", res.Status, res.Body)
	}
	if err := env.Store.ReadDB.QueryRow(`SELECT count(*) FROM tracking_links WHERE email = 'c@exemple.fr' AND last_used_at IS NOT NULL`).Scan(&touched); err != nil || touched != 1 {
		t.Fatalf("usage du lien non enregistré : %d (%v)", touched, err)
	}
	if res := c.Do("POST", "/api/tracking/close", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("accès par lien après fermeture : %d", res.Status)
	}
	if res := c.Do("GET", "/api/session", nil); res.Status != http.StatusOK {
		t.Fatalf("session de compte fermée avec le lien : %d", res.Status)
	}

	logs := env.Logs.String()
	for _, secret := range []string{"c@exemple.fr", revokedToken} {
		if strings.Contains(logs, secret) {
			t.Errorf("journaux : %q présent", secret)
		}
	}
	if n := publicLogs(t, env, "tracking_link_opened"); n != 1 {
		t.Errorf("journal tracking_link_opened : %d", n)
	}
}

// publicLogs compte les lignes de journal d'une action des parcours publics
// et vérifie qu'aucune ne contient d'adresse électronique (R16).
func publicLogs(t *testing.T, env *apitest.Env, action string) int {
	t.Helper()
	n := 0
	for _, line := range strings.Split(env.Logs.String(), "\n") {
		if !strings.Contains(line, `"action":"`+action+`"`) {
			continue
		}
		n++
		if strings.Contains(line, "@") {
			t.Errorf("adresse électronique dans le journal : %s", line)
		}
	}
	return n
}
