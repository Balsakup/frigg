package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api/apitest"
)

// logLines décode les lignes JSON de slog capturées par l'environnement.
func logLines(t *testing.T, env *apitest.Env) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(env.Logs.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("ligne de journal illisible %q : %v", line, err)
		}
		lines = append(lines, m)
	}
	return lines
}

// Le point de santé répond sans session, sans cookie ni donnée.
func TestHealthOK(t *testing.T) {
	env := apitest.New(t)
	res := env.Client().Do("GET", "/api/health", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("statut %d, attendu 200 : %s", res.Status, res.Body)
	}
	var body map[string]any
	res.JSON(t, &body)
	if len(body) != 1 || body["status"] != "ok" {
		t.Errorf(`corps %s, attendu {"status":"ok"}`, res.Body)
	}
	if got := res.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, attendu no-store", got)
	}
	if got := res.Header.Get("Set-Cookie"); got != "" {
		t.Errorf("cookie posé par le point de santé : %q", got)
	}
}

// Base indisponible : 503 sans détail interne, cause journalisée en warn.
func TestHealthDatabaseDown(t *testing.T) {
	env := apitest.New(t)
	if err := env.Store.ReadDB.Close(); err != nil {
		t.Fatal(err)
	}
	res := env.Client().Do("GET", "/api/health", nil)
	if res.Status != http.StatusServiceUnavailable {
		t.Fatalf("statut %d, attendu 503 : %s", res.Status, res.Body)
	}
	var p apitest.Problem
	res.JSON(t, &p)
	if p.Detail != "Service indisponible." {
		t.Errorf("detail = %q", p.Detail)
	}
	for _, leak := range []string{"sql", "database", "closed"} {
		if strings.Contains(strings.ToLower(res.Body), leak) {
			t.Errorf("le corps révèle un détail interne (%q) : %s", leak, res.Body)
		}
	}
	found := false
	for _, l := range logLines(t, env) {
		if l["msg"] == "vérification de santé en échec" && l["level"] == "WARN" {
			found = true
		}
	}
	if !found {
		t.Errorf("aucune ligne warn « vérification de santé en échec » :\n%s", env.Logs.String())
	}
}

// Un appel réussi n'est journalisé qu'au niveau debug.
func TestHealthLogLevel(t *testing.T) {
	env := apitest.New(t)
	if res := env.Client().Do("GET", "/api/health", nil); res.Status != http.StatusOK {
		t.Fatalf("statut %d", res.Status)
	}
	debug := false
	for _, l := range logLines(t, env) {
		if l["path"] != "/api/health" {
			continue
		}
		switch l["level"] {
		case "INFO":
			t.Errorf("appel réussi journalisé en info : %v", l)
		case "DEBUG":
			debug = l["msg"] == "requête traitée"
		}
	}
	if !debug {
		t.Errorf("aucune ligne debug « requête traitée » pour /api/health")
	}
}

// Le point de santé n'accepte aucune écriture.
func TestHealthRejectsWrite(t *testing.T) {
	env := apitest.New(t)
	res := env.Client().Do("POST", "/api/health", map[string]any{})
	if res.Status < 400 {
		t.Errorf("POST /api/health : statut %d, attendu une erreur", res.Status)
	}
}
