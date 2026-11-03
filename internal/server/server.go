// Package server assemble le serveur HTTP : API, SPA embarqué et middlewares.
package server

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/api"
)

// Handler est le gestionnaire HTTP complet, avec l'API pour les tests.
type Handler struct {
	http.Handler
	API huma.API
}

// NewHandler construit la chaîne : protection CSRF → sessions → journalisation →
// en-têtes de sécurité → API ou SPA.
func NewHandler(deps api.Deps, dist fs.FS) (*Handler, error) {
	mux := http.NewServeMux()
	humaAPI := api.New(mux, deps)

	spa, err := spaHandler(dist)
	if err != nil {
		return nil, err
	}
	mux.Handle("/", spa)

	csp, err := contentSecurityPolicy(dist)
	if err != nil {
		return nil, err
	}

	var h http.Handler = securityHeaders(csp, mux)
	h = truncateNotificationBody(h)
	h = requestLogger(deps.Logger, h)
	h = deps.Sessions.LoadAndSave(h)

	cop := http.NewCrossOriginProtection()
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, http.StatusForbidden, "Accès refusé", "Requête d'origine croisée refusée.")
	}))
	h = cop.Handler(h)

	return &Handler{Handler: h, API: humaAPI}, nil
}

// Run écoute sur addr jusqu'à l'annulation de ctx, puis arrête proprement le
// serveur en laissant 10 s aux requêtes en cours.
func Run(ctx context.Context, addr string, h http.Handler, logger *slog.Logger) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("serveur démarré", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	logger.Info("arrêt du serveur")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// truncateNotificationBody limite le corps lu d'une notification HelloAsso :
// le surplus est ignoré plutôt que refusé, pour que la réponse ne dépende pas
// du contenu reçu (constitution v1.7.0, principe II).
func truncateNotificationBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, api.NotificationPathPrefix) && r.Body != nil {
			r.Body = struct {
				io.Reader
				io.Closer
			}{io.LimitReader(r.Body, api.NotificationMaxBody), r.Body}
			r.ContentLength = -1
		}
		next.ServeHTTP(w, r)
	})
}
