package auth

import (
	"context"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// KeyedLimiter limite le débit par clé (adresse, compte ou IP), en mémoire :
// une seule instance, un seul processus (research.md › R7).
type KeyedLimiter struct {
	limit rate.Limit
	burst int

	mu      sync.Mutex
	entries map[string]*limiterEntry
}

type limiterEntry struct {
	lim  *rate.Limiter
	last time.Time
}

// NewKeyedLimiter autorise n événements par période pour chaque clé.
func NewKeyedLimiter(n int, period time.Duration) *KeyedLimiter {
	return &KeyedLimiter{
		limit:   rate.Every(period / time.Duration(n)),
		burst:   n,
		entries: map[string]*limiterEntry{},
	}
}

// Allow consomme un jeton pour la clé et indique si l'action est autorisée.
func (l *KeyedLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[key]
	if !ok {
		e = &limiterEntry{lim: rate.NewLimiter(l.limit, l.burst)}
		l.entries[key] = e
	}
	e.last = time.Now()
	return e.lim.Allow()
}

// purge oublie les clés inactives depuis plus longtemps que maxIdle.
func (l *KeyedLimiter) purge(maxIdle time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, e := range l.entries {
		if time.Since(e.last) > maxIdle {
			delete(l.entries, k)
		}
	}
}

// Limits regroupe les limiteurs des parcours sensibles (FR-007).
type Limits struct {
	LoginEmail    *KeyedLimiter
	LoginIP       *KeyedLimiter
	ResetEmail    *KeyedLimiter
	ResetIP       *KeyedLimiter
	VerifyAccount *KeyedLimiter
	VerifyIP      *KeyedLimiter
	TokenIP       *KeyedLimiter
	InviteAdmin   *KeyedLimiter
	AdminReset    *KeyedLimiter
}

// NewLimits crée les limiteurs avec les valeurs de research.md › R7.
func NewLimits() *Limits {
	return &Limits{
		LoginEmail:    NewKeyedLimiter(5, 15*time.Minute),
		LoginIP:       NewKeyedLimiter(20, time.Minute),
		ResetEmail:    NewKeyedLimiter(3, time.Hour),
		ResetIP:       NewKeyedLimiter(10, time.Hour),
		VerifyAccount: NewKeyedLimiter(3, time.Hour),
		VerifyIP:      NewKeyedLimiter(10, time.Hour),
		TokenIP:       NewKeyedLimiter(10, time.Minute),
		InviteAdmin:   NewKeyedLimiter(30, time.Hour),
		AdminReset:    NewKeyedLimiter(30, time.Hour), // research.md 002 › R9
	}
}

// RunPurge purge périodiquement les clés inactives jusqu'à l'annulation de ctx.
func (l *Limits) RunPurge(ctx context.Context) {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for _, k := range []*KeyedLimiter{l.LoginEmail, l.LoginIP, l.ResetEmail, l.ResetIP, l.VerifyAccount, l.VerifyIP, l.TokenIP, l.InviteAdmin, l.AdminReset} {
				k.purge(2 * time.Hour)
			}
		}
	}
}

// ClientIP renvoie l'IP du client ; X-Forwarded-For n'est lu que si la
// connexion provient d'un proxy de confiance.
func ClientIP(r *http.Request, trustedProxies []string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if !slices.Contains(trustedProxies, host) {
		return host
	}
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return host
	}
	// La dernière adresse est celle ajoutée par notre proxy.
	parts := strings.Split(xff, ",")
	return strings.TrimSpace(parts[len(parts)-1])
}
