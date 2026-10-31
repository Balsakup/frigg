// Package apitest monte l'API complète (sessions et protection CSRF comprises)
// sur un serveur httptest, avec une base temporaire et un faux expéditeur.
// Constitution 1.4.0 : httptest sur la chaîne complète dès que sessions, cookies
// ou CSRF sont en jeu ; humatest reste réservé aux vérifications de schéma.
package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/config"
	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/mail/mailtest"
	"github.com/balsakup/frigg/internal/server"
)

// Password est le mot de passe des comptes créés par Seed.
const Password = "Motdepasse1"

// Env est un environnement de test complet.
type Env struct {
	t      testing.TB
	URL    string
	Store  *database.Store
	Mail   *mailtest.Sender
	API    huma.API
	Logs   *LogBuffer
	worker *mail.Worker
	seq    atomic.Int64
}

// LogBuffer capture les journaux pour vérifier qu'aucune donnée sensible n'y figure.
type LogBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *LogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *LogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// New crée la base, applique les migrations et démarre le serveur de test.
// Les options modifient la configuration avant le démarrage.
func New(t testing.TB, options ...func(*config.Config)) *Env {
	t.Helper()
	ctx := context.Background()
	store, err := database.Open(filepath.Join(t.TempDir(), "frigg.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	if err := database.Migrate(ctx, store.WriteDB); err != nil {
		t.Fatal(err)
	}

	logs := &LogBuffer{}
	logger := slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	env := &Env{t: t, Store: store, Mail: &mailtest.Sender{}, Logs: logs}
	env.worker = &mail.Worker{Queries: store.Write, Sender: env.Mail, Interval: time.Hour, Logger: logger}

	cfg := config.Config{BaseURL: "http://frigg.test", CookieSecure: false}
	for _, opt := range options {
		opt(&cfg)
	}
	srv := httptest.NewUnstartedServer(nil)
	h, err := server.NewHandler(api.Deps{
		Store:    store,
		Sessions: auth.NewSessionManager(store.WriteDB, false),
		Limits:   auth.NewLimits(),
		Config:   cfg,
		Logger:   logger,
	}, fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>Frigg</title>")}})
	if err != nil {
		t.Fatal(err)
	}
	srv.Config.Handler = h
	srv.Start()
	t.Cleanup(srv.Close)
	env.URL = srv.URL
	env.API = h.API
	return env
}

// Seed crée un compte avec le mot de passe Password et renvoie son adresse.
func (e *Env) Seed(role string, verified bool) string {
	e.t.Helper()
	email := fmt.Sprintf("compte%d@exemple.fr", e.seq.Add(1))
	hash, err := auth.HashPassword(Password)
	if err != nil {
		e.t.Fatal(err)
	}
	now := time.Now().UTC().Format(mail.TimeFormat)
	params := sqlc.CreateAccountParams{Email: email, PasswordHash: hash, Role: role, CreatedAt: now}
	if verified {
		params.EmailVerifiedAt.String, params.EmailVerifiedAt.Valid = now, true
	}
	if _, err := e.Store.Write.CreateAccount(context.Background(), params); err != nil {
		e.t.Fatal(err)
	}
	return email
}

// FlushMail envoie les e-mails en file vers le faux expéditeur.
func (e *Env) FlushMail() {
	e.worker.Tick(context.Background())
}

// Client est un client HTTP avec ses propres cookies (un navigateur).
type Client struct {
	t    testing.TB
	env  *Env
	http *http.Client
}

// Client renvoie un nouveau client sans session. Il n'envoie ni Origin ni
// Sec-Fetch-Site, comme un client non navigateur.
func (e *Env) Client() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{t: e.t, env: e, http: &http.Client{Jar: jar}}
}

// LoggedIn renvoie un client connecté avec le compte email.
func (e *Env) LoggedIn(email string) *Client {
	c := e.Client()
	res := c.Do("POST", "/api/auth/login", map[string]string{"email": email, "password": Password})
	if res.Status != http.StatusOK {
		e.t.Fatalf("connexion de %s : statut %d, %s", email, res.Status, res.Body)
	}
	return c
}

// Response est une réponse décodable.
type Response struct {
	Status int
	Header http.Header
	Body   string
}

// JSON décode le corps dans v.
func (r Response) JSON(t testing.TB, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(r.Body), v); err != nil {
		t.Fatalf("décodage de %q : %v", r.Body, err)
	}
}

// Problem est le format d'erreur de l'API.
type Problem struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Errors []struct {
		Location string `json:"location"`
		Message  string `json:"message"`
	} `json:"errors"`
}

// Do envoie une requête ; body est encodé en JSON s'il n'est pas nil.
func (c *Client) Do(method, path string, body any, headers ...string) Response {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.env.URL+path, r)
	if err != nil {
		c.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return Response{Status: res.StatusCode, Header: res.Header, Body: string(b)}
}
