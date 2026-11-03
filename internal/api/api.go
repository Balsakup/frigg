// Package api définit l'API REST typée de Frigg (huma) : opérations, contrôle
// d'accès et erreurs en français.
package api

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/communication"
	"github.com/balsakup/frigg/internal/config"
	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/helloasso"
	"github.com/balsakup/frigg/internal/season"
)

// Deps regroupe les dépendances des handlers. Elles peuvent être vides pour
// produire le schéma OpenAPI (frigg openapi).
type Deps struct {
	Store    *database.Store
	Sessions *scs.SessionManager
	Limits   *auth.Limits
	Config   config.Config
	Logger   *slog.Logger
	// Horloge injectable ; time.Now si nil (research.md 003 › R3).
	Now func() time.Time
	// HelloAsso est le client du paiement par carte ; nil si les identifiants
	// ne sont pas configurés (research.md 008 › R2).
	HelloAsso *helloasso.Client
	// Files range les pièces jointes des communications (specs/009-bulk-email).
	Files *communication.FileStore
	// LogoFiles range les variantes du logo (specs/010-general-settings).
	LogoFiles *communication.FileStore
}

// NewHelloAssoClient construit le client HelloAsso depuis la configuration, ou
// nil si elle ne le configure pas.
func NewHelloAssoClient(cfg config.Config, now func() time.Time) *helloasso.Client {
	if !cfg.HelloAssoConfigured() {
		return nil
	}
	h := cfg.HelloAsso
	return helloasso.New(helloasso.Config{
		ClientID: h.ClientID, ClientSecret: h.ClientSecret, Organization: h.Organization,
		Environment: h.Environment, APIURL: h.APIURL,
	}, nil, now)
}

type server struct {
	Deps
}

// today renvoie la date du jour à Paris, seule source de « aujourd'hui » pour
// les saisons.
func (s *server) today() season.Date {
	return season.Today(s.now())
}

// now renvoie l'instant courant de l'horloge injectable.
func (s *server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// New enregistre toutes les opérations sous le préfixe /api.
func New(mux *http.ServeMux, deps Deps) huma.API {
	cfg := huma.DefaultConfig("Frigg API", "1.0.0")
	// huma publie par défaut /openapi, /docs et /schemas sans authentification :
	// interdit par le principe II (voir research.md › R4 et R18).
	cfg.OpenAPIPath = ""
	cfg.DocsPath = ""
	cfg.SchemasPath = ""
	// Sans route /schemas, pas de champ $schema ni d'en-tête Link dans les réponses.
	cfg.CreateHooks = nil
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"session": {Type: "apiKey", In: "cookie", Name: "frigg_session"},
	}

	// Les dépôts de pièces jointes (5 Mio au plus) restent en mémoire : sans
	// cela, net/http écrit dans le répertoire temporaire, en lecture seule en
	// conteneur (spec 007).
	humago.MultipartMaxMemory = UploadMaxBody
	api := humago.NewWithPrefix(mux, "/api", cfg)
	s := &server{Deps: deps}
	api.UseMiddleware(s.accessMiddleware(api))

	s.registerHealth(api)
	s.registerAuth(api)
	s.registerMe(api)
	s.registerInvitations(api)
	s.registerAccounts(api)
	s.registerSeasons(api)
	s.registerMembershipFees(api)
	s.registerMembers(api)
	s.registerMemberships(api)
	s.registerCardPayments(api)
	s.registerPaymentAlerts(api)
	s.registerHelloAssoWebhook(api)
	s.registerCommunications(api)
	s.registerIdentity(api)
	s.registerAppearance(api)
	s.mountDocs(mux, api)
	return api
}

// register déclare une opération avec son niveau d'accès minimal, lu ensuite par
// accessMiddleware. Aucune opération ne doit être enregistrée autrement.
func register[I, O any](api huma.API, access auth.Access, op huma.Operation, h func(context.Context, *I) (*O, error)) {
	if op.Metadata == nil {
		op.Metadata = map[string]any{}
	}
	op.Metadata[accessMetadataKey] = access
	if access != auth.Public {
		op.Security = []map[string][]string{{"session": {}}}
	}
	huma.Register(api, op, h)
}

const accessMetadataKey = "access"

// AccessOf renvoie le niveau d'accès déclaré d'une opération.
func AccessOf(op *huma.Operation) (auth.Access, bool) {
	a, ok := op.Metadata[accessMetadataKey].(auth.Access)
	return a, ok
}

type ctxKey int

const (
	accountKey ctxKey = iota
	clientIPKey
)

// accessMiddleware applique le refus par défaut (principe II) et la fonction
// d'autorisation unique (principe III).
func (s *server) accessMiddleware(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		access, ok := AccessOf(ctx.Operation())
		if !ok {
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "Opération sans niveau d'accès déclaré.")
			return
		}
		r, _ := humago.Unwrap(ctx)
		ctx = huma.WithValue(ctx, clientIPKey, auth.ClientIP(r, s.Config.TrustedProxies))
		if access == auth.Public {
			next(ctx)
			return
		}

		reqCtx := ctx.Context()
		id, epoch, ok := auth.Current(reqCtx, s.Sessions)
		if !ok {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Vous devez être connecté.")
			return
		}
		account, err := s.Store.Read.GetAccountByID(reqCtx, id)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && (account.AuthEpoch != epoch || account.DisabledAt.Valid)) {
			// Compte supprimé ou désactivé, mot de passe changé ailleurs, etc. :
			// session révoquée (research.md 002 › R2).
			_ = auth.Logout(reqCtx, s.Sessions)
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Votre session a expiré.")
			return
		}
		if err != nil {
			s.Logger.ErrorContext(reqCtx, "lecture du compte impossible", "error", err)
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "Une erreur inattendue s'est produite.")
			return
		}
		SetRequestAccount(reqCtx, account.ID)
		if !auth.Allows(account.Role, access) {
			_ = huma.WriteErr(api, ctx, http.StatusForbidden, "Vous n'avez pas accès à cette ressource.")
			return
		}
		next(huma.WithValue(ctx, accountKey, account))
	}
}

// CurrentAccount renvoie le compte connecté (opérations non publiques).
func CurrentAccount(ctx context.Context) sqlc.Account {
	a, _ := ctx.Value(accountKey).(sqlc.Account)
	return a
}

func clientIP(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey).(string)
	return ip
}

// internalError journalise l'erreur et renvoie un 500 générique.
func (s *server) internalError(ctx context.Context, msg string, err error) error {
	s.Logger.ErrorContext(ctx, msg, "error", err)
	return huma.Error500InternalServerError("Une erreur inattendue s'est produite.")
}
