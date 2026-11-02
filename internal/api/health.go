package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
)

const tagHealth = "Santé"

// HealthPath est le chemin du point de santé, que le journal des requêtes
// traite à part (specs/007-docker-release, research.md › R10).
const HealthPath = "/api/health"

// healthTimeout borne la requête de vérification : une base bloquée doit
// rendre l'instance « en mauvaise santé », pas faire attendre la sonde.
const healthTimeout = 2 * time.Second

// HealthOutput est la réponse du point de santé. Elle ne porte aucune donnée de
// l'association, ni version, ni configuration (amendement 1.5.0, principe II).
type HealthOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         struct {
		Status string `json:"status" enum:"ok" doc:"État de l'instance"`
	}
}

// registerHealth enregistre le point de santé, seule opération publique hors
// des parcours d'authentification.
func (s *server) registerHealth(api huma.API) {
	register(api, auth.Public, huma.Operation{
		OperationID: "get-health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "État de santé de l'instance",
		Description: "Indique si l'instance peut servir des requêtes (serveur démarré, base accessible). " +
			"Destiné aux vérifications de santé du conteneur, d'un orchestrateur ou d'un reverse proxy.",
		Tags:   []string{tagHealth},
		Errors: []int{http.StatusServiceUnavailable},
	}, s.getHealth)
}

func (s *server) getHealth(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, healthTimeout)
	defer cancel()
	if _, err := s.Store.Read.HealthCheck(ctx); err != nil {
		// La cause reste dans les journaux : la réponse n'en dit rien.
		s.Logger.WarnContext(ctx, "vérification de santé en échec", "error", err)
		return nil, huma.Error503ServiceUnavailable("Service indisponible.")
	}
	out := &HealthOutput{CacheControl: "no-store"}
	out.Body.Status = "ok"
	return out, nil
}
