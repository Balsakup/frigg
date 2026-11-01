package server

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/balsakup/frigg/internal/api"
)

// requestLogger attribue un identifiant à chaque requête et journalise une ligne
// à la fin (research.md › R11). Aucun corps n'est journalisé.
func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		info := &api.RequestInfo{ID: hex.EncodeToString(b)}
		w.Header().Set("X-Request-Id", info.ID)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r.WithContext(api.WithRequestInfo(r.Context(), info)))

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", info.ID,
		}
		if info.AccountID != 0 {
			attrs = append(attrs, "account_id", info.AccountID)
		}
		logger.InfoContext(r.Context(), "requête traitée", attrs...)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// securityHeaders ajoute les en-têtes de sécurité à toute réponse (research.md › R12).
// csp est calculée au démarrage (empreintes des scripts inline du SPA).
func securityHeaders(csp string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Robots-Tag", "noindex, nofollow")
		if h.Get("Content-Security-Policy") == "" {
			h.Set("Content-Security-Policy", csp)
		}
		next.ServeHTTP(w, r)
	})
}
