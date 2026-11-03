// Package hatest fournit un faux HelloAsso pilotable pour les tests Go
// (research.md 008 › R15). Il n'est jamais compilé dans le binaire.
package hatest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Identifiants attendus par le faux service.
const (
	ClientID     = "client-test"
	ClientSecret = "secret-test-ne-pas-journaliser"
	Organization = "asso-test"
	OrgName      = "Association de test"
)

// Intent est une intention connue du faux service.
type Intent struct {
	ID          int64
	Body        map[string]any
	Metadata    json.RawMessage
	OrderID     int64
	PaymentID   int64
	AmountCents int64
	TipCents    int64
	State       string
	Date        time.Time
}

// Fake est un faux HelloAsso.
type Fake struct {
	*httptest.Server
	t testing.TB

	mu          sync.Mutex
	down        bool
	tokenStatus int
	expiresIn   int64
	tokens      int
	calls       map[string]int
	intents     map[int64]*Intent
	nextID      int64
	// Delay retarde chaque réponse (test des délais).
	Delay time.Duration
	// RedirectHost remplace l'hôte des pages de paiement renvoyées.
	RedirectHost string
	// Unauthorized401Once fait échouer le prochain appel API en 401.
	unauthorizedOnce bool
	revoked          bool
}

// New démarre un faux HelloAsso, arrêté à la fin du test.
func New(t testing.TB) *Fake {
	f := &Fake{t: t, calls: map[string]int{}, intents: map[int64]*Intent{}, nextID: 1000, expiresIn: 1799}
	f.Server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.Close)
	return f
}

// Down rend le service injoignable (réponses 503) ou le rétablit.
func (f *Fake) Down(down bool) { f.mu.Lock(); f.down = down; f.mu.Unlock() }

// RejectCredentials révoque les identifiants : 400 sur le jeton et 401 sur
// les appels portant un jeton déjà émis.
func (f *Fake) RejectCredentials() {
	f.mu.Lock()
	f.tokenStatus = http.StatusBadRequest
	f.revoked = true
	f.mu.Unlock()
}

// ExpireTokensIn fixe la durée de vie des jetons émis.
func (f *Fake) ExpireTokensIn(s int64) { f.mu.Lock(); f.expiresIn = s; f.mu.Unlock() }

// UnauthorizedOnce fait répondre 401 au prochain appel API.
func (f *Fake) UnauthorizedOnce() { f.mu.Lock(); f.unauthorizedOnce = true; f.mu.Unlock() }

// Tokens renvoie le nombre de jetons émis.
func (f *Fake) Tokens() int { f.mu.Lock(); defer f.mu.Unlock(); return f.tokens }

// Calls renvoie le nombre d'appels à une route (« POST checkout »,
// « GET checkout », « GET organization »).
func (f *Fake) Calls(route string) int { f.mu.Lock(); defer f.mu.Unlock(); return f.calls[route] }

// TotalCalls renvoie le nombre total d'appels API (hors jeton).
func (f *Fake) TotalCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		n += c
	}
	return n
}

// LastIntent renvoie la dernière intention créée.
func (f *Fake) LastIntent() *Intent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.intents[f.nextID]
}

// Intent renvoie une intention.
func (f *Fake) Intent(id int64) *Intent { f.mu.Lock(); defer f.mu.Unlock(); return f.intents[id] }

// Authorize marque une intention payée (commande et paiement autorisé).
func (f *Fake) Authorize(id, amountCents, tipCents int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	in := f.intents[id]
	if in == nil {
		f.t.Fatalf("intention %d inconnue", id)
	}
	in.OrderID, in.PaymentID = id+500000, id+900000
	in.AmountCents, in.TipCents, in.State = amountCents, tipCents, "Authorized"
	in.Date = time.Date(2026, 11, 3, 17, 12, 0, 0, time.UTC)
}

// Refund marque le paiement d'une intention remboursé.
func (f *Fake) Refund(id int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.intents[id].State = "Refunded"
}

// SetMetadata remplace les métadonnées renvoyées pour une intention.
func (f *Fake) SetMetadata(id int64, raw string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.intents[id].Metadata = json.RawMessage(raw)
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	if f.Delay > 0 {
		select {
		case <-time.After(f.Delay):
		case <-r.Context().Done():
			return
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.down {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if r.URL.Path == "/oauth2/token" {
		f.token(w, r)
		return
	}
	if f.revoked || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer tok-") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if f.unauthorizedOnce {
		f.unauthorizedOnce = false
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	prefix := "/v5/organizations/" + Organization
	switch {
	case r.Method == http.MethodGet && r.URL.Path == prefix:
		f.calls["GET organization"]++
		writeJSON(w, map[string]any{"name": OrgName, "organizationSlug": Organization})
	case strings.HasPrefix(r.URL.Path, "/v5/organizations/") && !strings.HasPrefix(r.URL.Path, prefix):
		w.WriteHeader(http.StatusNotFound)
	case r.Method == http.MethodPost && r.URL.Path == prefix+"/checkout-intents":
		f.calls["POST checkout"]++
		f.create(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, prefix+"/checkout-intents/"):
		f.calls["GET checkout"]++
		id, _ := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, prefix+"/checkout-intents/"), 10, 64)
		f.read(w, id)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *Fake) token(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if f.tokenStatus != 0 {
		w.WriteHeader(f.tokenStatus)
		return
	}
	ok := r.Form.Get("client_id") == ClientID &&
		((r.Form.Get("grant_type") == "client_credentials" && r.Form.Get("client_secret") == ClientSecret) ||
			(r.Form.Get("grant_type") == "refresh_token" && strings.HasPrefix(r.Form.Get("refresh_token"), "ref-")))
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	f.tokens++
	writeJSON(w, map[string]any{
		"access_token": fmt.Sprintf("tok-%d", f.tokens), "refresh_token": fmt.Sprintf("ref-%d", f.tokens),
		"expires_in": f.expiresIn, "token_type": "bearer",
	})
}

func (f *Fake) create(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	f.nextID++
	id := f.nextID
	meta, _ := json.Marshal(body["metadata"])
	f.intents[id] = &Intent{ID: id, Body: body, Metadata: meta}
	host := f.URL
	if f.RedirectHost != "" {
		host = f.RedirectHost
	}
	writeJSON(w, map[string]any{"id": id, "redirectUrl": fmt.Sprintf("%s/pay/%d", host, id)})
}

func (f *Fake) read(w http.ResponseWriter, id int64) {
	in := f.intents[id]
	if in == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	out := map[string]any{"id": in.ID, "metadata": in.Metadata, "redirectUrl": fmt.Sprintf("%s/pay/%d", f.URL, id)}
	if in.State == "Authorized" || in.State == "Refunded" {
		out["order"] = map[string]any{
			"id": in.OrderID,
			"payments": []map[string]any{{
				"id": in.PaymentID, "amount": in.AmountCents, "amountTip": in.TipCents,
				"state": in.State, "date": in.Date.Format(time.RFC3339),
			}},
		}
	}
	writeJSON(w, out)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
