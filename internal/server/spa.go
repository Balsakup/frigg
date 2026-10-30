package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
)

// spaHandler sert le SPA embarqué : fichier existant, sinon index.html pour que
// le routage côté client prenne le relais. Les chemins /api/ inconnus reçoivent
// un 404 JSON.
func spaHandler(dist fs.FS) (http.Handler, error) {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, fmt.Errorf("index.html introuvable dans le SPA embarqué : %w", err)
	}
	files := http.FileServerFS(dist)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			writeProblem(w, http.StatusNotFound, "Introuvable", "Cette ressource n'existe pas.")
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" && name != "index.html" {
			if st, err := fs.Stat(dist, name); err == nil && !st.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	}), nil
}

var inlineScriptRe = regexp.MustCompile(`(?s)<script(?:\s[^>]*)?>(.*?)</script>`)

// contentSecurityPolicy construit la CSP stricte : les scripts inline générés par
// React Router sont autorisés par leur empreinte, sans 'unsafe-inline'.
func contentSecurityPolicy(dist fs.FS) (string, error) {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return "", err
	}
	scriptSrc := []string{"'self'"}
	for _, m := range inlineScriptRe.FindAllSubmatch(index, -1) {
		if len(bytes.TrimSpace(m[1])) == 0 {
			continue
		}
		sum := sha256.Sum256(m[1])
		scriptSrc = append(scriptSrc, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
	}
	return strings.Join([]string{
		"default-src 'self'",
		"script-src " + strings.Join(scriptSrc, " "),
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"font-src 'self'",
		"connect-src 'self'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}, "; "), nil
}

func writeProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `{"status":%d,"title":%q,"detail":%q}`, status, title, detail)
}
