package auth

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
)

const (
	sessionAccountKey = "account_id"
	sessionEpochKey   = "auth_epoch"

	SessionLifetime    = 30 * 24 * time.Hour // FR-006 : 30 jours après la connexion
	SessionIdleTimeout = 7 * 24 * time.Hour  // FR-006 : 7 jours d'inactivité
)

// NewSessionManager configure les sessions stockées en SQLite (research.md › R5).
func NewSessionManager(writeDB *sql.DB, cookieSecure bool) *scs.SessionManager {
	sm := scs.New()
	sm.Store = sqlite3store.NewWithCleanupInterval(writeDB, 30*time.Minute)
	sm.Lifetime = SessionLifetime
	sm.IdleTimeout = SessionIdleTimeout
	sm.Cookie.Name = "frigg_session"
	sm.Cookie.Secure = cookieSecure
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Path = "/"
	return sm
}

// Login ouvre une session pour le compte, avec un nouveau jeton (fixation de session).
func Login(ctx context.Context, sm *scs.SessionManager, accountID, epoch int64) error {
	if err := sm.RenewToken(ctx); err != nil {
		return err
	}
	sm.Put(ctx, sessionAccountKey, accountID)
	sm.Put(ctx, sessionEpochKey, epoch)
	return nil
}

// Logout détruit la session courante.
func Logout(ctx context.Context, sm *scs.SessionManager) error {
	return sm.Destroy(ctx)
}

// Current renvoie le compte et l'époque enregistrés dans la session.
func Current(ctx context.Context, sm *scs.SessionManager) (accountID, epoch int64, ok bool) {
	accountID = sm.GetInt64(ctx, sessionAccountKey)
	if accountID == 0 {
		return 0, 0, false
	}
	return accountID, sm.GetInt64(ctx, sessionEpochKey), true
}

// SetEpoch met à jour l'époque de la session courante après un changement de
// mot de passe, pour qu'elle reste valide alors que les autres sont fermées.
func SetEpoch(ctx context.Context, sm *scs.SessionManager, epoch int64) {
	sm.Put(ctx, sessionEpochKey, epoch)
}
