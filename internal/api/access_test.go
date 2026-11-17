package api_test

import (
	"context"
	"database/sql"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Opérations publiques autorisées par le principe II (contracts/api.md).
var allowedPublic = map[string]bool{
	"login":                  true,
	"request-password-reset": true,
	"confirm-password-reset": true,
	"confirm-email":          true,
	"get-invitation":         true,
	"accept-invitation":      true,
	// Point de santé (amendement 1.5.0 de la constitution).
	"get-health": true,
	// Notifications de HelloAsso (amendements 1.6.0 et 1.7.0 de la constitution).
	"receive-helloasso-notification": true,
	// Identité visuelle (amendement 1.8.0 de la constitution).
	"get-identity":      true,
	"get-identity-logo": true,
	// Accès par lien personnel (amendement 1.9.0 de la constitution).
	"open-tracking-link": true,
	// Parcours publics (amendement 1.9.0 de la constitution, 016 FR-001).
	"get-public-membership-page": true,
	"submit-public-membership":   true,
	"confirm-public-submission":  true,
	"request-tracking-link":      true,
	// Page publique d'un événement (amendement 1.9.0, 016 FR-025).
	"get-public-event":           true,
	"submit-public-registration": true,
}

var pathParamRe = regexp.MustCompile(`\{[^}]+\}`)

// TestAccessMatrix vérifie chaque opération de l'API : anonyme → 401 (sauf
// publiques), membre → 403 sur les opérations admin, administrateur jamais
// refusé. Les opérations par lien (auth.Link) refusent tout client sans lien
// ouvert, même connecté, et un lien révoqué (research.md 016 › R1, R17).
func TestAccessMatrix(t *testing.T) {
	env := apitest.New(t)
	// Les chemins sont appelés avec l'identifiant 1 : ce compte sert de cible
	// aux actions d'administration (désactivation, suppression…) pour que les
	// clients membre et administrateur restent connectés.
	env.Seed(auth.RoleMember, true)
	anonymous := env.Client()
	member := env.LoggedIn(env.Seed(auth.RoleMember, true))
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	tracked := env.TrackingClient("suivi@exemple.fr")
	revoked := env.TrackingClient("revoque@exemple.fr")
	if _, err := env.Store.Write.RevokeTrackingLinksOfEmail(context.Background(), sqlc.RevokeTrackingLinksOfEmailParams{
		Email: "revoque@exemple.fr", Now: sql.NullString{String: "2000-01-01T00:00:00Z", Valid: true},
	}); err != nil {
		t.Fatal(err)
	}

	ops := 0
	for path, item := range env.API.OpenAPI().Paths {
		for method, op := range map[string]*hOp{
			"GET": wrap(item.Get), "POST": wrap(item.Post), "PUT": wrap(item.Put),
			"PATCH": wrap(item.Patch), "DELETE": wrap(item.Delete),
		} {
			if op == nil {
				continue
			}
			ops++
			access, ok := api.AccessOf(op.Operation)
			if !ok {
				t.Errorf("%s %s : aucun niveau d'accès déclaré", method, path)
				continue
			}
			if access == auth.Public && !allowedPublic[op.OperationID] {
				t.Errorf("%s %s (%s) est publique sans figurer dans la liste du principe II", method, path, op.OperationID)
			}
			url := "/api" + pathParamRe.ReplaceAllString(path, "1")
			var body any
			if method != "GET" && method != "DELETE" {
				body = map[string]any{}
			}

			if access != auth.Public {
				if res := anonymous.Do(method, url, body); res.Status != http.StatusUnauthorized {
					t.Errorf("%s %s anonyme : statut %d, attendu 401", method, path, res.Status)
				}
			}
			if access == auth.Admin {
				if res := member.Do(method, url, body); res.Status != http.StatusForbidden {
					t.Errorf("%s %s membre : statut %d, attendu 403", method, path, res.Status)
				}
			}
			if access == auth.Link {
				for name, c := range map[string]*apitest.Client{"membre": member, "administrateur": admin, "lien révoqué": revoked} {
					if res := c.Do(method, url, body); res.Status != http.StatusUnauthorized {
						t.Errorf("%s %s %s : statut %d, attendu 401", method, path, name, res.Status)
					}
				}
				client := tracked
				if op.OperationID == "close-tracking" {
					// Retirerait l'accès du client partagé.
					client = env.TrackingClient("suivi@exemple.fr")
				}
				if res := client.Do(method, url, body); res.Status == http.StatusUnauthorized || res.Status == http.StatusForbidden {
					t.Errorf("%s %s lien : statut %d", method, path, res.Status)
				}
				continue
			}
			if strings.HasSuffix(op.OperationID, "logout") {
				continue // déconnecterait le client administrateur
			}
			if res := admin.Do(method, url, body); res.Status == http.StatusUnauthorized || res.Status == http.StatusForbidden {
				t.Errorf("%s %s administrateur : statut %d", method, path, res.Status)
			}
		}
	}
	t.Logf("%d opérations vérifiées", ops)
}
