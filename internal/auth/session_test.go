package auth_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

func TestSessionManagerSettings(t *testing.T) {
	sm := auth.NewSessionManager(nil, true)
	if sm.Lifetime != 30*24*time.Hour || sm.IdleTimeout != 7*24*time.Hour {
		t.Fatalf("durées : %v / %v", sm.Lifetime, sm.IdleTimeout)
	}
	if !sm.Cookie.Secure || !sm.Cookie.HttpOnly || sm.Cookie.SameSite != http.SameSiteLaxMode || sm.Cookie.Name != "frigg_session" {
		t.Fatalf("cookie : %+v", sm.Cookie)
	}
}

func TestSessionRevokedWhenEpochChanges(t *testing.T) {
	env := apitest.New(t)
	email := env.Seed(auth.RoleMember, true)
	c := env.LoggedIn(email)

	a, _ := env.Store.Read.GetAccountByEmail(t.Context(), email)
	if _, err := env.Store.Write.UpdatePasswordAndBumpEpoch(t.Context(), sqlc.UpdatePasswordAndBumpEpochParams{ID: a.ID, PasswordHash: a.PasswordHash}); err != nil {
		t.Fatal(err)
	}
	if res := c.Do("GET", "/api/session", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("session après changement d'époque : %d, attendu 401", res.Status)
	}
}
