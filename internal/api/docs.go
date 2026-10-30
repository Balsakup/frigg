package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
)

// Version épinglée de l'interface de documentation chargée depuis le CDN
// (exception du principe IV réservée à la documentation de développement).
const elementsURL = "https://unpkg.com/@stoplight/elements@9.0.28"

// mountDocs monte la documentation d'API de développement (research.md › R18) :
// uniquement si FRIGG_API_DOCS=true, et réservée aux administrateurs. Sans le
// drapeau, ces chemins répondent 404 comme toute route API inconnue.
func (s *server) mountDocs(mux *http.ServeMux, api huma.API) {
	if !s.Config.APIDocs {
		return
	}
	s.Logger.Warn("documentation d'API activée : à réserver au développement", "path", "/api/docs")

	mux.Handle("GET /api/openapi.json", s.adminOnly(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, api.OpenAPI())
	}))
	mux.Handle("GET /api/schemas/{name}", s.adminOnly(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSuffix(r.PathValue("name"), ".json")
		schema, ok := api.OpenAPI().Components.Schemas.Map()[name]
		if !ok {
			writeHTTPProblem(w, http.StatusNotFound, "Ce schéma n'existe pas.")
			return
		}
		writeJSON(w, schema)
	}))
	mux.Handle("GET /api/docs", s.adminOnly(func(w http.ResponseWriter, _ *http.Request) {
		// CSP assouplie sur cette seule page pour autoriser le CDN de l'interface.
		w.Header().Set("Content-Security-Policy", fmt.Sprintf(
			"default-src 'self'; script-src 'self' %[1]s; style-src 'self' 'unsafe-inline' %[1]s; img-src 'self' data:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'",
			"https://unpkg.com"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, docsPage, elementsURL)
	}))
}

const docsPage = `<!doctype html>
<html lang="fr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Documentation de l'API · Frigg</title>
  <script src="%[1]s/web-components.min.js"></script>
  <link rel="stylesheet" href="%[1]s/styles.min.css">
</head>
<body style="margin:0;height:100vh">
  <elements-api apiDescriptionUrl="/api/openapi.json" basePath="/" router="hash" layout="sidebar"></elements-api>
</body>
</html>`

// adminOnly applique, hors huma, la même fonction d'autorisation que l'API.
func (s *server) adminOnly(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id, epoch, ok := auth.Current(ctx, s.Sessions)
		if !ok {
			writeHTTPProblem(w, http.StatusUnauthorized, "Vous devez être connecté.")
			return
		}
		account, err := s.Store.Read.GetAccountByID(ctx, id)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && account.AuthEpoch != epoch) {
			writeHTTPProblem(w, http.StatusUnauthorized, "Votre session a expiré.")
			return
		}
		if err != nil {
			s.Logger.ErrorContext(ctx, "lecture du compte impossible", "error", err)
			writeHTTPProblem(w, http.StatusInternalServerError, "Une erreur inattendue s'est produite.")
			return
		}
		SetRequestAccount(ctx, account.ID)
		if !auth.Allows(account.Role, auth.Admin) {
			writeHTTPProblem(w, http.StatusForbidden, "Vous n'avez pas accès à cette ressource.")
			return
		}
		h(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeHTTPProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(huma.NewError(status, detail))
}
