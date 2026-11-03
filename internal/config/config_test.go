package config

import (
	"strings"
	"testing"
)

// clearEnv neutralise les variables FRIGG_* et le .env éventuel du dépôt.
func clearEnv(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	for _, k := range []string{
		"FRIGG_HELLOASSO_CLIENT_ID", "FRIGG_HELLOASSO_CLIENT_SECRET", "FRIGG_HELLOASSO_ORGANIZATION",
		"FRIGG_HELLOASSO_ENVIRONMENT", "FRIGG_HELLOASSO_API_URL", "FRIGG_BASE_URL",
		"FRIGG_SMTP_HOST", "FRIGG_MAIL_FROM",
	} {
		t.Setenv(k, "")
	}
	t.Setenv("FRIGG_SMTP_HOST", "localhost")
	t.Setenv("FRIGG_MAIL_FROM", "frigg@localhost")
}

func TestHelloAssoConfig(t *testing.T) {
	full := map[string]string{
		"FRIGG_HELLOASSO_CLIENT_ID":     "id",
		"FRIGG_HELLOASSO_CLIENT_SECRET": "secret",
		"FRIGG_HELLOASSO_ORGANIZATION":  "mon-asso",
	}
	tests := []struct {
		name       string
		env        map[string]string
		baseURL    string
		configured bool
		loadErr    string
		serveErr   string
	}{
		{name: "aucune variable", baseURL: "http://localhost:5173"},
		{name: "complet", env: full, baseURL: "https://frigg.example", configured: true},
		{name: "partiel", env: map[string]string{"FRIGG_HELLOASSO_CLIENT_ID": "id"}, baseURL: "https://frigg.example",
			loadErr: "FRIGG_HELLOASSO_CLIENT_SECRET est obligatoire lorsque FRIGG_HELLOASSO_CLIENT_ID est renseigné"},
		{name: "environnement inconnu", env: merge(full, map[string]string{"FRIGG_HELLOASSO_ENVIRONMENT": "prod"}),
			baseURL: "https://frigg.example", loadErr: "FRIGG_HELLOASSO_ENVIRONMENT doit valoir sandbox ou production"},
		{name: "base HTTP refusée, même en mode test", env: full, baseURL: "http://localhost:5173", configured: true,
			serveErr: "FRIGG_BASE_URL doit commencer par https://"},
		{name: "base HTTP admise avec un faux HelloAsso",
			env:     merge(full, map[string]string{"FRIGG_HELLOASSO_API_URL": "http://localhost:9090/"}),
			baseURL: "http://localhost:5173", configured: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("FRIGG_BASE_URL", tt.baseURL)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			cfg, err := Load()
			if tt.loadErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.loadErr) {
					t.Fatalf("Load : erreur %v, attendu %q", err, tt.loadErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load : %v", err)
			}
			if cfg.HelloAssoConfigured() != tt.configured {
				t.Errorf("HelloAssoConfigured = %v", cfg.HelloAssoConfigured())
			}
			if tt.configured && tt.env["FRIGG_HELLOASSO_ENVIRONMENT"] == "" && cfg.HelloAsso.Environment != HelloAssoSandbox {
				t.Errorf("environnement par défaut = %q", cfg.HelloAsso.Environment)
			}
			err = cfg.RequireServe()
			if tt.serveErr == "" && err != nil {
				t.Errorf("RequireServe : %v", err)
			}
			if tt.serveErr != "" && (err == nil || !strings.Contains(err.Error(), tt.serveErr)) {
				t.Errorf("RequireServe : erreur %v, attendu %q", err, tt.serveErr)
			}
			if strings.Contains(cfg.HelloAsso.APIURL, "/ ") || strings.HasSuffix(cfg.HelloAsso.APIURL, "/") {
				t.Errorf("APIURL non normalisée : %q", cfg.HelloAsso.APIURL)
			}
		})
	}
}

func merge(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}
