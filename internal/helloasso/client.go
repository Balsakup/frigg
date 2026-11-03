// Package helloasso est le client de l'API HelloAsso utilisé pour le paiement
// des adhésions par carte (research.md 008 › R1, R3). Il ne connaît ni la base
// ni les adhésions. Le secret et les jetons ne quittent jamais ce paquet : ils
// ne figurent dans aucune erreur ni aucun journal.
package helloasso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Environnements.
const (
	Sandbox    = "sandbox"
	Production = "production"
)

const (
	productionHost = "https://api.helloasso.com"
	sandboxHost    = "https://api.helloasso-sandbox.com"
	// callTimeout borne chaque appel ; testTimeout borne le test de connexion
	// entier (jeton compris), dont le résultat est attendu en moins de 10 s.
	callTimeout = 10 * time.Second
	testTimeout = 10 * time.Second
	// tokenMargin : le jeton est renouvelé avant son expiration.
	tokenMargin = 60 * time.Second
	maxBody     = 1 << 20
)

// Erreurs typées, sans aucune donnée sensible.
var (
	ErrUnauthorized = errors.New("identifiants HelloAsso refusés")
	ErrForbidden    = errors.New("droits HelloAsso insuffisants")
	ErrNotFound     = errors.New("ressource HelloAsso introuvable")
	ErrUnavailable  = errors.New("HelloAsso injoignable")
	ErrUnexpected   = errors.New("réponse inattendue de HelloAsso")
)

// ErrorCode traduit une erreur du client en code stable (contracts/api.md,
// HelloAssoError).
func ErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, ErrForbidden):
		return "forbidden"
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrUnavailable):
		return "unavailable"
	}
	return "unexpected"
}

// Config est la configuration du client.
type Config struct {
	ClientID     string
	ClientSecret string
	Organization string
	Environment  string
	// APIURL remplace l'hôte (tests) ; vide : hôte de l'environnement.
	APIURL string
}

// Host renvoie l'hôte de l'API.
func (c Config) Host() string {
	if c.APIURL != "" {
		return strings.TrimRight(c.APIURL, "/")
	}
	if c.Environment == Production {
		return productionHost
	}
	return sandboxHost
}

// Client parle à HelloAsso pour une association. Sûr en concurrence.
type Client struct {
	cfg  Config
	http *http.Client
	now  func() time.Time

	mu           sync.Mutex
	accessToken  string
	refreshToken string
	expiresAt    time.Time
}

// New construit un client. httpClient et now peuvent être nil.
func New(cfg Config, httpClient *http.Client, now func() time.Time) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: callTimeout}
	}
	if now == nil {
		now = time.Now
	}
	return &Client{cfg: cfg, http: httpClient, now: now}
}

// Environment renvoie l'environnement configuré.
func (c *Client) Environment() string { return c.cfg.Environment }

// OrganizationSlug renvoie le slug configuré.
func (c *Client) OrganizationSlug() string { return c.cfg.Organization }

// Organization teste la connexion : jeton puis lecture de l'association. Le
// tout est borné à 10 s. Renvoie le nom de l'association chez HelloAsso.
func (c *Client) Organization(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, testTimeout)
	defer cancel()
	var out struct {
		Name string `json:"name"`
	}
	if err := c.do(ctx, http.MethodGet, "/v5/organizations/"+url.PathEscape(c.cfg.Organization), nil, &out, true); err != nil {
		return "", err
	}
	return out.Name, nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// token renvoie un jeton valide, en le renouvelant si besoin. force invalide
// le jeton courant (après un 401).
func (c *Client) token(ctx context.Context, force bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if force {
		c.accessToken = ""
	}
	if c.accessToken != "" && c.now().Before(c.expiresAt.Add(-tokenMargin)) {
		return c.accessToken, nil
	}
	if c.refreshToken != "" {
		form := url.Values{"grant_type": {"refresh_token"}, "client_id": {c.cfg.ClientID}, "refresh_token": {c.refreshToken}}
		if err := c.fetchToken(ctx, form); err == nil {
			return c.accessToken, nil
		} else if errors.Is(err, ErrUnavailable) {
			return "", err
		}
		c.refreshToken = ""
	}
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {c.cfg.ClientID}, "client_secret": {c.cfg.ClientSecret}}
	if err := c.fetchToken(ctx, form); err != nil {
		return "", err
	}
	return c.accessToken, nil
}

// fetchToken appelle le point de jeton ; appelé sous c.mu.
func (c *Client) fetchToken(ctx context.Context, form url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Host()+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return ErrUnexpected
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return ErrUnavailable
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, maxBody))
	switch {
	case res.StatusCode == http.StatusOK:
	case res.StatusCode == http.StatusBadRequest || res.StatusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case res.StatusCode >= 500:
		return ErrUnavailable
	default:
		return fmt.Errorf("%w (jeton, statut %d)", ErrUnexpected, res.StatusCode)
	}
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil || tr.AccessToken == "" {
		return ErrUnexpected
	}
	c.accessToken = tr.AccessToken
	c.refreshToken = tr.RefreshToken
	expires := tr.ExpiresIn
	if expires <= 0 {
		expires = 1799
	}
	c.expiresAt = c.now().Add(time.Duration(expires) * time.Second)
	return nil
}

// do envoie une requête authentifiée. retry autorise une seule nouvelle
// tentative après un 401 (jeton expiré côté HelloAsso).
func (c *Client) do(ctx context.Context, method, path string, in, out any, retry bool) error {
	tok, err := c.token(ctx, false)
	if err != nil {
		return err
	}
	status, body, err := c.send(ctx, method, path, in, tok)
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized && retry {
		if tok, err = c.token(ctx, true); err != nil {
			return err
		}
		if status, body, err = c.send(ctx, method, path, in, tok); err != nil {
			return err
		}
	}
	switch {
	case status >= 200 && status < 300:
	case status == http.StatusUnauthorized:
		return ErrUnauthorized
	case status == http.StatusForbidden:
		return ErrForbidden
	case status == http.StatusNotFound:
		return ErrNotFound
	case status >= 500 || status == http.StatusTooManyRequests:
		return ErrUnavailable
	default:
		return fmt.Errorf("%w (statut %d)", ErrUnexpected, status)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return ErrUnexpected
	}
	return nil
}

func (c *Client) send(ctx context.Context, method, path string, in any, tok string) (int, []byte, error) {
	var reader io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return 0, nil, ErrUnexpected
		}
		reader = strings.NewReader(string(b))
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.Host()+path, reader)
	if err != nil {
		return 0, nil, ErrUnexpected
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return 0, nil, ErrUnavailable
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody))
	if err != nil {
		return 0, nil, ErrUnavailable
	}
	return res.StatusCode, body, nil
}
