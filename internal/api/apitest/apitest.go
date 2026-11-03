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
	"github.com/balsakup/frigg/internal/communication"
	"github.com/balsakup/frigg/internal/config"
	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/helloasso/hatest"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/mail/mailtest"
	"github.com/balsakup/frigg/internal/season"
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
	// HelloAsso est le faux HelloAsso (NewWithHelloAsso), nil sinon.
	HelloAsso *hatest.Fake
	// Sweeper balaie les tentatives de paiement par carte (NewWithHelloAsso).
	Sweeper *api.OnlinePaymentSweeper
	// Files range les pièces jointes des communications (répertoire temporaire).
	Files *communication.FileStore
	// LogoFiles range les variantes du logo (répertoire temporaire).
	LogoFiles *communication.FileStore
	// AttachmentSweeper purge les pièces jointes abandonnées.
	AttachmentSweeper *api.CommunicationAttachmentSweeper
	seq               atomic.Int64
	clock             atomic.Pointer[time.Time]
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
	files, err := communication.NewFileStore(filepath.Join(t.TempDir(), "communications"))
	if err != nil {
		t.Fatal(err)
	}
	logoFiles, err := communication.NewFileStore(filepath.Join(t.TempDir(), "logo"))
	if err != nil {
		t.Fatal(err)
	}
	env := &Env{t: t, Store: store, Mail: &mailtest.Sender{}, Logs: logs, Files: files, LogoFiles: logoFiles}
	env.worker = &mail.Worker{Queries: store.Write, Sender: env.Mail, Interval: time.Hour, Logger: logger, Files: files}

	cfg := config.Config{BaseURL: "http://frigg.test", CookieSecure: false}
	for _, opt := range options {
		opt(&cfg)
	}
	srv := httptest.NewUnstartedServer(nil)
	deps := api.Deps{
		Store:     store,
		Sessions:  auth.NewSessionManager(store.WriteDB, false),
		Limits:    auth.NewLimits(),
		Config:    cfg,
		Logger:    logger,
		Now:       env.now,
		HelloAsso: api.NewHelloAssoClient(cfg, env.now),
		Files:     files,
		LogoFiles: logoFiles,
	}
	env.worker.Identity = api.MailIdentity(deps)
	env.AttachmentSweeper = api.NewCommunicationAttachmentSweeper(deps)
	if deps.HelloAsso != nil {
		env.Sweeper = api.NewOnlinePaymentSweeper(deps)
	}
	h, err := server.NewHandler(deps, fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>Frigg</title>")}})
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

// NewWithHelloAsso crée un environnement relié à un faux HelloAsso
// (environnement sandbox).
func NewWithHelloAsso(t testing.TB, options ...func(*config.Config)) *Env {
	t.Helper()
	fake := hatest.New(t)
	opts := append([]func(*config.Config){func(c *config.Config) {
		c.HelloAsso = config.HelloAsso{
			ClientID: hatest.ClientID, ClientSecret: hatest.ClientSecret, Organization: hatest.Organization,
			Environment: config.HelloAssoSandbox, APIURL: fake.URL,
		}
	}}, options...)
	env := New(t, opts...)
	env.HelloAsso = fake
	return env
}

// EnableCard active la carte et enregistre un test de connexion réussi pour
// l'environnement donné, comme après « Tester la connexion ».
func (e *Env) EnableCard(environment string) {
	e.t.Helper()
	ctx := context.Background()
	if _, err := e.Store.WriteDB.ExecContext(ctx, `UPDATE payment_settings SET card_enabled = 1,
		helloasso_test_at = ?, helloasso_test_environment = ?, helloasso_test_ok = 1 WHERE id = 1`,
		time.Now().UTC().Format(mail.TimeFormat), environment); err != nil {
		e.t.Fatal(err)
	}
}

func (e *Env) now() time.Time {
	if t := e.clock.Load(); t != nil {
		return *t
	}
	return time.Now()
}

// SetToday fixe l'horloge du serveur à midi (Paris) du jour donné, au format
// AAAA-MM-JJ.
func (e *Env) SetToday(day string) {
	e.t.Helper()
	d, err := time.ParseInLocation("2006-01-02", day, season.Paris)
	if err != nil {
		e.t.Fatal(err)
	}
	noon := d.Add(12 * time.Hour)
	e.clock.Store(&noon)
}

// Advance avance l'horloge du serveur (fixée par SetToday) de d.
func (e *Env) Advance(d time.Duration) {
	t := e.now().Add(d)
	e.clock.Store(&t)
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
// Raw est un corps envoyé tel quel par Do.
type Raw []byte

func (c *Client) Do(method, path string, body any, headers ...string) Response {
	c.t.Helper()
	var r io.Reader
	if raw, ok := body.(Raw); ok {
		r = bytes.NewReader(raw)
	} else if body != nil {
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
