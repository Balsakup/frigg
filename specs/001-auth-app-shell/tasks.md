---

description: "Liste des tâches : authentification et shell applicatif membre/administrateur"
---

# Tasks: Authentification et shell applicatif membre/administrateur

**Input**: Design documents from `specs/001-auth-app-shell/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, cli.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : tests d'autorisation automatisés (principes II et III),
tests d'API, et Playwright + axe sur chaque parcours touché (Workflow).

**Organization**: une phase par user story. Ordre d'exécution : US3 (créer le
premier administrateur) passe avant US1, car on ne peut pas se connecter sans
compte ; les numéros US suivent ceux de la spec.

**Conventions** (constitution) : identifiants, tables, champs JSON et valeurs
d'énumération en anglais ; commentaires, messages, textes d'interface et chemins
visibles du SPA en français.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US7)

## Path Conventions

Un module Go à la racine (`cmd/`, `internal/`) et le SPA dans `web/` (voir
plan.md › Project Structure).

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: dépendances, outils et configuration du dépôt

- [X] T001 Ajouter les dépendances Go dans `go.mod` : `go get github.com/danielgtaylor/huma/v2@v2.40.0 github.com/alexedwards/scs/v2@v2.9.0 github.com/alexedwards/scs/sqlite3store github.com/pressly/goose/v3@v3.29.0 modernc.org/sqlite@v1.61.0 github.com/alexedwards/argon2id@v1.0.0 github.com/wneessen/go-mail@v0.8.1 github.com/joho/godotenv@v1.5.1 golang.org/x/time@v0.16.0 golang.org/x/term`, puis `go get -tool github.com/go-task/task/v3/cmd/task@latest` (directive `tool`, appel via `go tool task`)
- [X] T002 [P] Créer `sqlc.yaml` à la racine : `version: "2"`, `engine: sqlite`, `schema: internal/database/migrations`, `queries: internal/database/queries`, sortie Go `internal/database/sqlc`, `package: sqlc`, `emit_empty_slices: true`
- [X] T003 [P] Créer `.env.example` documentant en français chaque variable de research.md › R10 (`FRIGG_DATABASE_PATH`, `FRIGG_DATA_DIR`, `FRIGG_LISTEN_ADDR`, `FRIGG_BASE_URL`, `FRIGG_COOKIE_SECURE`, `FRIGG_TRUSTED_PROXIES`, `FRIGG_SMTP_HOST`, `FRIGG_SMTP_PORT`, `FRIGG_SMTP_USERNAME`, `FRIGG_SMTP_PASSWORD`, `FRIGG_SMTP_TLS`, `FRIGG_MAIL_FROM`, `FRIGG_MAIL_POLL_INTERVAL`, `FRIGG_LOG_LEVEL`, `FRIGG_API_DOCS`) avec des valeurs adaptées à Mailpit (`localhost:1025`, `FRIGG_SMTP_TLS=none`) ; créer `.gitignore` à la racine avec `.env`, `/data/`, `/bin/`
- [X] T004 [P] Créer `Taskfile.yml` (go-task v3) avec les tâches : `web:build` (`bun install --frozen-lockfile && bun run build` dans `web/`), `build` (dépend de `web:build` ; `CGO_ENABLED=0 go build -o bin/frigg ./cmd/frigg`), `gen` (dépend de `web:build` ; `go generate ./...`, `go run ./cmd/frigg openapi > web/app/api/openapi.json`, `bun run gen:api` dans `web/`), `dev` (lance en parallèle `go run ./cmd/frigg serve` et `bun run dev`), `test` (`go test ./...` puis `bun run test`), `e2e` (dépend de `build` ; `bun run e2e`), `check` (enchaîne `build`, `go vet ./...`, `go test ./...`, `gen` puis `git diff --exit-code internal/database/sqlc web/app/api`, et dans `web/` `typecheck`, `lint`, `test`, `e2e`)
- [X] T005 [P] Ajouter dans `web/package.json` le script `"gen:api": "openapi-typescript app/api/openapi.json -o app/api/schema.d.ts"`
- [X] T006 [P] Remplacer le contenu de `web/app/app.css` par les tokens de l'artefact, thème clair uniquement (supprimer `dark:` et `color-scheme: dark`) : variables `:root` en canaux RGB `--brand: 31 82 120`, `--brand-ink: 31 82 120`, `--on-brand: 255 255 255`, `--accent: 196 72 34`, `--on-accent: 255 255 255`, `--ink: 24 31 41`, `--muted: 82 90 103`, `--line: 221 225 231`, `--line-strong: 128 135 147`, `--canvas: 245 246 248`, `--surface: 255 255 255`, `--ok: 21 112 68`, `--warn: 134 80 0`, `--danger: 180 28 28`, `--info: 29 90 170`, `--sidebar-w: 18rem`, `--sidebar-w-collapsed: 4.5rem` ; dans `@theme`, couleurs Tailwind correspondantes (`--color-brand: rgb(var(--brand))`, `--color-brand-ink`, `--color-on-brand`, `--color-accent`, `--color-on-accent`, `--color-ink`, `--color-muted`, `--color-line`, `--color-line-strong`, `--color-canvas`, `--color-surface`, `--color-ok`, `--color-warn`, `--color-danger`, `--color-info`) et polices déjà déclarées ; `body` en `bg-canvas text-ink`, 16 px, interligne 1.5 ; `:focus-visible { outline: 3px solid rgb(var(--brand-ink)); outline-offset: 2px }` ; règle `prefers-reduced-motion` qui coupe transitions et animations

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: socle partagé par toutes les stories (base, sessions, autorisation, e-mails, serveur, client API, composants, e2e)

**⚠️ CRITICAL**: aucune story ne peut commencer avant la fin de cette phase

### Backend : configuration et base

- [X] T007 Implémenter `internal/config/config.go` : struct `Config` (champs de research.md › R10, durées en `time.Duration`, `APIDocs bool`), `Load()` qui appelle `godotenv.Load()` en ignorant l'absence de `.env` sans écraser l'environnement, applique les défauts, et `RequireServe()` qui renvoie une erreur `configuration invalide : FRIGG_BASE_URL est obligatoire` (idem `FRIGG_SMTP_HOST`, `FRIGG_MAIL_FROM`) ; `FRIGG_SMTP_TLS` limité à `starttls|tls|none`
- [X] T008 Écrire `internal/database/migrations/00001_init.sql` (annotations goose `-- +goose Up/Down`, tables `STRICT`) : `association` (`id INTEGER PRIMARY KEY CHECK (id = 1)`, `name TEXT` NULL tant que non défini, `updated_at TEXT NOT NULL`) + `INSERT` de la ligne `(1, NULL, now)` ; `accounts` (`id INTEGER PRIMARY KEY`, `email TEXT NOT NULL UNIQUE`, `password_hash TEXT NOT NULL`, `role TEXT NOT NULL CHECK (role IN ('member','admin'))`, `email_verified_at TEXT` NULL = adresse non confirmée, `auth_epoch INTEGER NOT NULL DEFAULT 1`, `created_at TEXT NOT NULL`, `last_login_at TEXT`) ; `sessions` (`token TEXT PRIMARY KEY`, `data BLOB NOT NULL`, `expiry REAL NOT NULL`) + `CREATE INDEX sessions_expiry_idx ON sessions(expiry)`
- [X] T009 [P] Écrire `internal/database/migrations/00002_auth_tokens.sql` : `one_time_tokens` (`id INTEGER PRIMARY KEY`, `account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE`, `purpose TEXT NOT NULL CHECK (purpose IN ('password_reset','email_verification'))`, `token_hash BLOB NOT NULL UNIQUE`, `expires_at TEXT NOT NULL`, `used_at TEXT`, `created_at TEXT NOT NULL`) + index `(account_id, purpose)` ; `invitations` (`id INTEGER PRIMARY KEY`, `email TEXT NOT NULL`, `role TEXT NOT NULL CHECK (role IN ('member','admin'))`, `token_hash BLOB NOT NULL UNIQUE`, `invited_by INTEGER REFERENCES accounts(id) ON DELETE SET NULL`, `status TEXT NOT NULL CHECK (status IN ('pending','accepted','cancelled'))`, `expires_at TEXT NOT NULL`, `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`) + `CREATE UNIQUE INDEX invitations_pending_email_idx ON invitations(email) WHERE status = 'pending'`
- [X] T010 [P] Écrire `internal/database/migrations/00003_outbound_emails.sql` : `outbound_emails` (`id INTEGER PRIMARY KEY`, `recipient TEXT NOT NULL`, `subject TEXT NOT NULL`, `body_text TEXT NOT NULL`, `body_html TEXT NOT NULL`, `status TEXT NOT NULL CHECK (status IN ('pending','sent','failed'))`, `attempts INTEGER NOT NULL DEFAULT 0`, `next_attempt_at TEXT NOT NULL`, `last_error TEXT`, `created_at TEXT NOT NULL`, `sent_at TEXT`) + index `(status, next_attempt_at)`
- [X] T011 [P] Écrire `internal/database/queries/association.sql` (`GetAssociation`, `SetAssociationName`) et `internal/database/queries/accounts.sql` (`CreateAccount`, `GetAccountByEmail`, `GetAccountByID`, `UpdateLastLogin`, `UpdatePasswordAndBumpEpoch` qui fait `auth_epoch = auth_epoch + 1` et renvoie la nouvelle époque, `MarkEmailVerified`)
- [X] T012 [P] Écrire `internal/database/queries/tokens.sql` (`DeleteUnusedTokens(account_id, purpose)`, `CreateToken`, `GetValidToken(token_hash, purpose, now)` avec `used_at IS NULL AND expires_at > now`, `MarkTokenUsed`) et `internal/database/queries/emails.sql` (`EnqueueEmail`, `ListDueEmails(now, limit)` sur `status = 'pending' AND next_attempt_at <= now`, `MarkEmailSent`, `MarkEmailRetry(attempts, next_attempt_at, last_error)`, `MarkEmailFailed`, `PurgeSentEmails(before)`)
- [X] T013 Implémenter `internal/database/db.go` (`Open(path)` → `*Store` avec pool d'écriture `SetMaxOpenConns(1)` et pool de lecture `SetMaxOpenConns(4)` + `_pragma=query_only(1)`, DSN de research.md › R2 avec `_txlock=immediate`, champs `Read`/`Write *sqlc.Queries`, `WriteTx(ctx, func(*sqlc.Queries) error) error`, `Close()`), `internal/database/migrate.go` (`//go:embed migrations/*.sql`, `goose.SetBaseFS`, dialecte `sqlite3`, `Migrate(ctx, db)`), `internal/database/generate.go` (`//go:generate sqlc generate -f ../../sqlc.yaml`) ; lancer `go generate ./internal/database` et commiter `internal/database/sqlc/`
- [X] T014 [P] Test `internal/database/db_test.go` : base fichier dans `t.TempDir()`, migrations appliquées deux fois sans erreur, `PRAGMA journal_mode` = `wal`, `foreign_keys` = 1, écriture refusée sur le pool de lecture, ligne `association` id 1 présente

### Backend : authentification et autorisation

- [X] T015 [P] Implémenter `internal/auth/password.go` : paramètres argon2id centralisés (`Memory: 19*1024`, `Iterations: 2`, `Parallelism: 1`, `SaltLength: 16`, `KeyLength: 32`), `HashPassword`, `VerifyPassword`, `VerifyAgainstDummy` (hash factice calculé une fois), `ValidatePassword(pw) []string` renvoyant exactement « Le mot de passe doit contenir au moins 8 caractères. », « Le mot de passe doit contenir au moins une lettre majuscule. » (`unicode.IsUpper`), « Le mot de passe doit contenir au moins un chiffre. » (`unicode.IsDigit`) ; `NormalizeEmail` (espaces retirés, minuscules) ; `ValidateEmail` (3 à 254 caractères, un seul `@`, domaine non vide)
- [X] T016 [P] Test `internal/auth/password_test.go` : chaque critère isolément et cumulés, majuscule accentuée acceptée (`É`), normalisation `"  Claire@Exemple.FR "` → `claire@exemple.fr`, hachage puis vérification
- [X] T017 [P] Implémenter `internal/auth/tokens.go` : `NewToken() (raw string, hash []byte)` (32 octets `crypto/rand`, base64url sans remplissage, empreinte SHA-256) et `HashToken(raw) []byte`
- [X] T018 [P] Implémenter `internal/auth/access.go` : `type Access` (`Public`, `Member`, `Admin`), constantes de rôle `RoleMember = "member"`, `RoleAdmin = "admin"`, `Allows(role string, a Access) bool` par rang (`member = 1`, `admin = 2`) — fonction d'autorisation unique (principe III) ; test `internal/auth/access_test.go`
- [X] T019 [P] Implémenter `internal/auth/session.go` : `NewSessionManager(writeDB, cfg)` (`sqlite3store.NewWithCleanupInterval(db, 30*time.Minute)`, `Lifetime = 30*24h`, `IdleTimeout = 7*24h`, cookie `frigg_session`, `Secure = cfg.CookieSecure`, `HttpOnly`, `SameSite=Lax`, `Path=/`), `Login(ctx, sm, accountID, epoch)` (`RenewToken` puis clés `account_id`, `auth_epoch`), `Logout` (`Destroy`), `Current(ctx) (id, epoch int64, ok bool)`, `SetEpoch(ctx, epoch)`
- [X] T020 [P] Implémenter `internal/auth/ratelimit.go` : `KeyedLimiter` (map protégée par mutex de `*rate.Limiter` + dernière utilisation, purge des clés inactives toutes les 10 min, `Allow(key) bool`) et `Limits` avec les valeurs de research.md › R7 (connexion 5/15 min par adresse et 20/min par IP ; réinitialisation 3/h par adresse et 10/h par IP ; renvoi de confirmation 3/h par compte et 10/h par IP ; jetons publics 10/min par IP) ; `ClientIP(r, trustedProxies)` (`RemoteAddr`, `X-Forwarded-For` seulement si le proxy est de confiance) ; test `internal/auth/ratelimit_test.go`

### Backend : e-mails

- [X] T021 [P] Créer `internal/mail/templates/layout.html.tmpl` et `layout.txt.tmpl` (en français, nom de l'association en en-tête) et `internal/mail/render.go` (`//go:embed templates`, `Render(name string, data any) (Message, error)` qui produit `Subject`, `Text`, `HTML` ; `html/template` pour le HTML, `text/template` pour le texte)
- [X] T022 Implémenter `internal/mail/outbox.go` : `Enqueue(ctx, q *sqlc.Queries, to string, msg Message)` à appeler dans la transaction de l'action (`status = 'pending'`, `next_attempt_at = now`)
- [X] T023 Implémenter `internal/mail/worker.go` et `internal/mail/smtp.go` : interface `Sender`, `SMTPSender` (go-mail, TLS selon `FRIGG_SMTP_TLS`, expéditeur `FRIGG_MAIL_FROM`), `Worker.Run(ctx)` (`time.Ticker` à `FRIGG_MAIL_POLL_INTERVAL`, lots de 20, succès → `sent`, échec → nouvelle tentative à 1 min, 2 min, 4 min… plafonnée à 1 h, `failed` à la 10ᵉ tentative, `last_error` sans le corps, purge des `sent` de plus de 30 jours, arrêt sur `ctx.Done()`), journaux slog en français
- [X] T024 [P] Créer `internal/mail/mailtest/sender.go` (faux `Sender` qui capture les messages, extraction du jeton d'un lien) et test `internal/mail/worker_test.go` (succès, nouvelles tentatives avec délais croissants, passage en `failed`, purge)

### Backend : API et serveur

- [X] T025 Implémenter `internal/api/errors.go` : titres d'erreur en français (surcharge de `huma.NewError` : 400 « Requête invalide », 401 « Connexion requise », 403 « Accès refusé », 404 « Introuvable », 409 « Conflit », 410 « Lien expiré », 422 « Données invalides », 429 « Trop de tentatives », 500 « Erreur interne ») ; traduction en français des messages de validation de huma (variables `Msg*` du package `github.com/danielgtaylor/huma/v2/validation`) ; helpers `FieldErrors(location string, messages ...string) error` (une entrée `errors[]` par message) et `TooManyAttempts()` (`detail` « Trop de tentatives. Réessayez dans quelques minutes. », en-tête `Retry-After`)
- [X] T026 Implémenter `internal/api/api.go` : `Deps` (Store, SessionManager, Limits, Config, Logger), `New(mux *http.ServeMux, deps Deps) huma.API` via `humago.NewWithPrefix(mux, "/api", cfg)` avec `cfg.OpenAPIPath = ""`, `cfg.DocsPath = ""`, `cfg.SchemasPath = ""` ; `register[I, O any](api, access auth.Access, op huma.Operation, h func(context.Context, *I) (*O, error))` qui écrit `op.Metadata["access"]` et `panic` si le niveau est absent ; middleware huma qui lit ce niveau, laisse passer `Public`, sinon lit la session (`auth.Current`), charge le compte (`Read.GetAccountByID`), détruit la session et répond 401 si le compte n'existe plus ou si `auth_epoch` diffère, répond 403 si `!auth.Allows(role, access)`, et place le compte dans le contexte (`CurrentAccount(ctx)`) ; `New` doit fonctionner avec `deps` vides pour `frigg openapi`
- [X] T027 [P] Implémenter `internal/server/middleware.go` : identifiant de requête (16 octets hexadécimaux, contexte + en-tête `X-Request-Id`), journalisation slog JSON d'une ligne par requête (méthode, chemin, statut, durée, `request_id`, `account_id` si connu ; jamais de corps), en-têtes `Referrer-Policy: same-origin`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy: default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:`
- [X] T028 [P] Implémenter `internal/server/spa.go` : sert `web.Dist()` ; chemin sous `/api/` → 404 `application/problem+json` ; fichier existant → servi (`Cache-Control: public, max-age=31536000, immutable` pour `/assets/`) ; sinon `index.html` avec `Cache-Control: no-cache`
- [X] T029 Implémenter `internal/server/server.go` : `NewHandler(deps)` (mux, `api.New`, SPA en repli sur `/`, chaîne `http.CrossOriginProtection` → `sm.LoadAndSave` → journalisation → en-têtes) et `Run(ctx, cfg, handler)` (`http.Server` avec `ReadHeaderTimeout`, arrêt propre sur annulation du contexte)
- [X] T030 Implémenter `cmd/frigg/main.go` et `internal/cli/serve.go` : aiguillage sur `os.Args[1]` (`serve` par défaut et `openapi` ; `create-admin` et `backup` sont branchés par T040 et T084 ; sinon aide en français listant les commandes disponibles et code 2) ; `serve` = `config.Load` + `RequireServe`, logger slog JSON (`FRIGG_LOG_LEVEL`), `database.Open`, `Migrate` (échec → message et code 1), session manager, `mail.Worker` en goroutine, `server.Run` avec `signal.NotifyContext(SIGINT, SIGTERM)`
- [X] T031 [P] Implémenter `internal/cli/openapi.go` : construit l'API sans base (`api.New` avec deps vides) et écrit `api.OpenAPI()` en JSON indenté (OpenAPI 3.1) sur la sortie standard
- [X] T032 Créer le harnais de test `internal/api/apitest/apitest.go` : `New(t)` crée une base dans `t.TempDir()`, applique les migrations, monte `server.NewHandler` complet (sessions + CSRF) sur `httptest.NewServer`, fournit un faux expéditeur, `Seed(role, verified bool) (email, password)`, `Client(t)` avec cookie jar (sans `Origin` ni `Sec-Fetch-Site`, comme un client non navigateur), `Login(t, email, password)` ; `humatest` pour les assertions sur le schéma (constitution 1.3.2 : `httptest` sur la chaîne complète dès que sessions, cookies ou CSRF sont en jeu)
- [X] T033 Écrire `internal/api/access_test.go` (matrice d'autorisation générée, principe II) : parcourt `api.OpenAPI().Paths`, lit `Metadata["access"]` de chaque opération ; vérifie que l'ensemble des opérations `public` est inclus dans `{login, request-password-reset, confirm-password-reset, confirm-email, get-invitation, accept-invitation}` ; pour chaque opération non publique (paramètres de chemin factices, corps `{}`) : anonyme → 401 ; membre → 403 si `admin` ; administrateur → ni 401 ni 403

### Frontend : client, composants, routes

- [X] T034 [P] Créer `web/app/api/client.ts` : `createClient<paths>({ baseUrl: "" })` (openapi-fetch, types de `./schema.d.ts`), middleware qui, sur un 401 hors `get-session` et hors routes publiques, vide le cache et redirige vers `/connexion?expiree=1&suite=<chemin courant>` ; `web/app/api/query-client.ts` (instance unique de `QueryClient`) ; `web/app/api/queries.ts` (`$api = createQueryHooks(client)` d'openapi-react-query) ; `web/app/api/problem.ts` (`applyProblemToForm(problem, setError)` : chaque `errors[].location` `body.<champ>` → `setError(<champ>)`, le reste → `setError("root")`)
- [X] T035 [P] Créer `web/app/lib/storage.ts` (`readPreference(key)` / `writePreference(key, value)` sur `localStorage`, chaque accès dans un try/catch) et `web/app/lib/redirect.ts` (`safeRedirect(value)` : accepte seulement une valeur qui commence par `/` et pas par `//`, sinon `null`) + test `web/app/lib/redirect.test.ts`
- [X] T036 [P] Créer `web/app/ui/Icon.tsx` (enveloppe lucide-react : `strokeWidth={1.8}`, `aria-hidden` par défaut, tailles `h-5 w-5`) et `web/app/ui/Button.tsx` (variantes de l'artefact `primary`, `secondary`, `tertiary`, `link`, `danger`, `danger-secondary` ; tailles `sm` `md` `lg` ; prop `iconOnly` ; état `aria-disabled` ; classes composées avec `cn`)
- [X] T037 [P] Créer `web/app/ui/Alert.tsx` (types `success`, `info`, `warning`, `danger` avec bordure et fond des tokens `ok`/`info`/`warn`/`danger`, préfixe lu par les lecteurs d'écran « Succès : », « Information : », « Avertissement : », « Erreur : ») et `web/app/ui/Field.tsx` (libellé, texte d'aide, message d'erreur relié par `aria-describedby`, `aria-invalid`, mention « (obligatoire) »)
- [X] T038 Mettre à jour `web/app/root.tsx` : `QueryClientProvider`, titre par défaut « Frigg », `HydrateFallback` « Chargement… », `ErrorBoundary` en français ; déclarer dans `web/app/routes.ts` le layout public `routes/auth/layout.tsx` uniquement ; chaque story ajoute ses routes dans `routes.ts` en même temps que leurs fichiers, pour que `typecheck` et `build` passent à chaque checkpoint ; créer `web/app/routes/auth/layout.tsx` (gabarit centré `max-w-[30rem]`, nom « Frigg » en tête — aucune donnée de l'association hors connexion, principe II)
- [X] T039 Créer l'outillage e2e : `internal/tools/e2eseed/main.go` (programme hors binaire Frigg : migre une base donnée en argument et crée l'association « Les Sentiers du Val », un administrateur, un membre confirmé et un membre non confirmé, mots de passe connus) ; `web/e2e/global-setup.ts` (base temporaire, lance `e2eseed`) ; `web/e2e/mailpit.ts` (lecture du dernier message d'une adresse via l'API Mailpit `http://localhost:8025/api/v1`, extraction du lien) ; `web/playwright.config.ts` : `globalSetup`, `webServer` = `../bin/frigg serve` sur `:8090` avec la base temporaire et `FRIGG_SMTP_HOST=localhost`, `FRIGG_SMTP_PORT=1025`, `FRIGG_SMTP_TLS=none`, `FRIGG_MAIL_POLL_INTERVAL=1s`, `baseURL` `http://localhost:8090` (prérequis : `docker compose up -d`)

**Checkpoint**: socle prêt ; `go test ./...` et `bun run typecheck` passent

---

## Phase 3: User Story 3 - Initialiser l'instance avec un premier administrateur (Priority: P1)

**Goal**: créer le premier administrateur (et le nom de l'association) en ligne de commande

**Independent Test**: sur une base vierge, `frigg create-admin` crée le compte ; une seconde exécution avec la même adresse échoue (quickstart scénario 1)

- [X] T040 [US3] Implémenter `internal/cli/createadmin.go` : fonction testable `CreateAdmin(ctx, store, in Input) error` (dans `WriteTx` : enregistre `association.name` si NULL, refuse une adresse existante avec `un compte existe déjà avec l'adresse …`, applique `ValidatePassword` (une ligne par critère), crée le compte `role = 'admin'` avec `email_verified_at = now`) et commande interactive (`term.IsTerminal` sinon `create-admin doit être lancé dans un terminal` ; invites « Nom de l'association : » seulement si non défini, « Adresse électronique : », « Mot de passe : » et « Confirmez le mot de passe : » via `term.ReadPassword` ; `les deux mots de passe ne correspondent pas` ; option `-email` ; migrations appliquées d'abord ; succès `Administrateur … créé.`) ; brancher `create-admin` dans `cmd/frigg/main.go`
- [X] T041 [P] [US3] Test `internal/cli/createadmin_test.go` : création sur base vierge (rôle admin, adresse confirmée, nom enregistré), nom non redemandé s'il existe, adresse en double refusée sans modification, mot de passe non conforme refusé

**Checkpoint**: un administrateur peut être créé

---

## Phase 4: User Story 1 - Se connecter et accéder à son espace (Priority: P1) 🎯 MVP

**Goal**: connexion, déconnexion, expiration, et shell commun affiché après connexion

**Independent Test**: se connecter, arriver sur l'accueil dans le shell, se déconnecter, constater qu'aucune page n'est accessible (quickstart scénarios 2 et 3)

### Tests for User Story 1

- [X] T042 [P] [US1] Test `internal/api/auth_test.go` : connexion réussie (cookie posé, `Session` renvoyée, `last_login_at` mis à jour) ; mauvais mot de passe et adresse inconnue → même 401 « Adresse électronique ou mot de passe incorrect » ; adresse avec majuscules et espaces reconnue ; 6ᵉ tentative en moins de 15 min pour une adresse → 429 ; déconnexion → `get-session` 401 ; POST `login` avec `Sec-Fetch-Site: cross-site` → 403, avec `Origin: https://autre.exemple` (sans `Sec-Fetch-Site`) → 403, avec `Sec-Fetch-Site: same-origin` → accepté (une requête sans ces deux en-têtes est acceptée par `http.CrossOriginProtection` : ne pas la tester comme refusée)
- [X] T043 [P] [US1] Test `internal/auth/session_test.go` : `Lifetime` = 30 jours, `IdleTimeout` = 7 jours, cookie `Secure`, `HttpOnly`, `SameSite=Lax` ; époque différente → session rejetée par le middleware

### Implementation for User Story 1

- [X] T044 [US1] Implémenter dans `internal/api/auth.go` l'opération `login` (`POST /api/auth/login`, `Public`, corps `{ email, password }`) : limites par adresse et par IP, `NormalizeEmail`, compte introuvable → `VerifyAgainstDummy` puis 401, mauvais mot de passe → 401 (message identique), sinon `auth.Login` + `UpdateLastLogin` et réponse `Session` ; et `logout` (`POST /api/auth/logout`, `Member`, 204, `auth.Logout`)
- [X] T045 [US1] Implémenter dans `internal/api/me.go` l'opération `get-session` (`GET /api/session`, `Member`) : `Session { account: { id, email, role, emailVerified }, association: { name } }`, nom « Mon association » si NULL
- [X] T046 [US1] Régénérer `web/app/api/openapi.json` et `web/app/api/schema.d.ts` (`go tool task gen`) et ajouter `sessionQuery` dans `web/app/api/queries.ts`
- [X] T047 [US1] Créer `web/app/routes/auth/login.tsx` : react-hook-form (adresse `type="email"` `autocomplete="email"`, mot de passe `autocomplete="current-password"`), alerte d'erreur pour le 401, alerte « Votre session a expiré. Reconnectez-vous. » si `?expiree=1`, lien « Mot de passe oublié ? » vers `/mot-de-passe-oublie`, après succès : mise en cache de la session puis navigation vers `safeRedirect(suite)` ou l'accueil ; `clientLoader` qui redirige vers l'accueil si une session existe déjà ; ajouter la route `/connexion` dans `web/app/routes.ts`
- [X] T048 [US1] Créer `web/app/routes/app/layout.tsx` : `clientLoader` → `queryClient.ensureQueryData(sessionQuery)`, 401 → `redirect("/connexion?suite=" + chemin)` ; rend le shell autour de `<Outlet />` ; ajouter le layout protégé dans `web/app/routes.ts`
- [X] T049 [P] [US1] Créer `web/app/shell/SkipLinks.tsx` (« Aller au contenu », « Aller à la navigation »), `web/app/shell/Header.tsx` (bouton de la barre latérale seul, FR-024), `web/app/shell/Footer.tsx` (« © {année} {association} · Propulsé par Frigg v{version} », version injectée par `define` Vite depuis `FRIGG_VERSION`, défaut `dev`)
- [X] T050 [P] [US1] Créer `web/app/shell/Breadcrumbs.tsx` et `web/app/shell/PageHeader.tsx` (titre `h1` avec `tabIndex={-1}` focalisé à chaque navigation, introduction, zone d'actions) et `web/app/shell/usePageMeta.ts` (titre du document « {page} · {association} » à partir du `handle` des routes)
- [X] T051 [US1] Créer `web/app/shell/Sidebar.tsx` (identité : monogramme des initiales du nom de l'association sur fond `brand`, nom, « Espace membre » ; emplacement de la navigation rempli en US2) et `web/app/shell/UserMenu.tsx` (menu base-ui : adresse, rôle « Membre » ou « Administrateur », « Mon compte » vers `/mon-compte`, « Se déconnecter » → `logout`, vidage du cache, navigation vers `/connexion`) et `web/app/shell/AppShell.tsx` (assemble barre latérale, en-tête, contenu `id="contenu"`, pied de page)
- [X] T052 [P] [US1] Créer `web/app/routes/app/member/home.tsx` (accueil de l'espace membre, `handle` titre « Accueil ») et `web/app/routes/app/not-found.tsx` (« Page introuvable » dans le shell, lien vers l'accueil) ; créer `web/app/routes/app/coming-soon.tsx` avec un texte générique « Bientôt disponible » ; ajouter dans `web/app/routes.ts` `/`, `/mon-compte` (provisoirement vers `coming-soon.tsx`) et la route `*` ; supprimer la démo du scaffold `web/app/welcome/` et `web/app/routes/home.tsx`
- [X] T053 [US1] Remplacer `web/e2e/home.spec.ts` par `web/e2e/login.spec.ts` : redirection vers `/connexion?suite=…` puis retour sur la page demandée, message d'échec identique, déconnexion puis bouton « Précédent » → connexion, page introuvable ; `expectNoBlockingA11yViolations` sur la connexion, l'accueil et la page introuvable

**Checkpoint**: MVP — connexion et shell fonctionnels

---

## Phase 5: User Story 2 - Navigation adaptée au rôle (Priority: P1)

**Goal**: menus par espace, sélecteur d'espace pour les administrateurs, barre latérale repliable et mobile, page « Accès refusé »

**Independent Test**: comparer les menus membre et administrateur ; ouvrir `/admin` en membre (quickstart scénarios 4 et 6)

- [X] T054 [P] [US2] Créer `web/app/shell/navigation.ts` : liste unique des entrées de contracts/ui-routes.md › Navigation (`id`, `label`, icône lucide — `House`, `Users`, `CreditCard`, `Calendar`, `ClipboardCheck`, `Coins`, `Mail`, `CircleHelp`, `BookOpen`, `Shield` —, `path`, `space: "member" | "admin"`, `section: "main" | "resources"`) et `spaceFromPath(pathname)` (`/admin*` → `admin`) + test `web/app/shell/navigation.test.ts`
- [X] T055 [US2] Compléter `web/app/shell/Sidebar.tsx` : navigation principale et « Ressources » filtrées sur l'espace courant, `aria-current="page"` sur l'entrée active, libellé de l'espace sous le nom ; repli en icônes au-delà de 1024 px (largeur `--sidebar-w-collapsed`, infobulle base-ui `Tooltip` par icône, état mémorisé via `writePreference("sidebar-collapsed")`) ; en dessous de 1024 px, tiroir base-ui `Dialog` en surimpression fermé par le bouton « Fermer le menu », Échap ou clic extérieur
- [X] T056 [P] [US2] Créer `web/app/shell/SpaceSwitcher.tsx` : affiché seulement si `role === "admin"`, deux liens « Admin » (`/admin`) et « Membre » (`/`), état actif dérivé de l'URL, mémorisation du dernier espace via `writePreference("last-space")`
- [X] T057 [US2] Créer `web/app/routes/app/admin/layout.tsx` (si `role !== "admin"` : composant `web/app/shell/AccessDenied.tsx` « Accès refusé » dans le shell, sinon `<Outlet />`), `web/app/routes/app/admin/home.tsx` (« Tableau de bord ») ; compléter `web/app/routes/app/coming-soon.tsx` (titre repris de `navigation.ts`) ; ajouter dans `web/app/routes.ts` `/admin`, son layout et les routes des rubriques métier
- [X] T058 [US2] Mettre à jour `web/app/routes/auth/login.tsx` : sans `suite`, un administrateur va vers `/admin` sauf si `readPreference("last-space") === "member"`, un membre vers `/`
- [X] T059 [US2] Écrire `web/e2e/navigation.spec.ts` : membre sans sélecteur ni rubrique d'administration, `/admin` en membre → « Accès refusé » ; administrateur sur `/admin` à la connexion, bascule d'espace, dernier espace conservé après rechargement, repli mémorisé ; tiroir mobile à 320 px fermé par Échap ; axe sur les deux accueils, « Accès refusé » et « Bientôt disponible »

**Checkpoint**: navigation par rôle complète

---

## Phase 6: User Story 4 - Inviter une personne à créer son compte (Priority: P2)

**Goal**: invitations par e-mail depuis la rubrique « Comptes », acceptation publique

**Independent Test**: inviter une adresse, suivre le lien reçu dans Mailpit, choisir un mot de passe, arriver connecté dans le bon espace ; lien rouvert refusé (quickstart scénario 5)

### Tests for User Story 4

- [X] T060 [P] [US4] Test `internal/api/invitations_test.go` : création (e-mail en file, `status = 'pending'`, échéance + 7 jours), adresse avec compte existant → 409 « Un compte existe déjà avec cette adresse », invitation en attente existante → 409 « Une invitation est déjà en attente pour cette adresse », liste (en attente et expirées, plus récentes d'abord), renvoi (nouveau jeton, ancien jeton → 410), annulation ; `get-invitation` et `accept-invitation` : compte créé avec le rôle prévu et `email_verified_at` renseigné, session ouverte, jeton réutilisé → 410, invitation expirée ou annulée → 410, règle de mot de passe → 422

### Implementation for User Story 4

- [X] T061 [P] [US4] Écrire `internal/database/queries/invitations.sql` (`CreateInvitation`, `GetPendingInvitationByEmail`, `GetInvitationByTokenHash`, `ListPendingInvitations` avec l'adresse de l'invitant par jointure, `ReplaceInvitationToken(token_hash, expires_at)`, `CancelInvitation`, `MarkInvitationAccepted`) et régénérer `internal/database/sqlc/`
- [X] T062 [P] [US4] Créer `internal/mail/templates/invitation.html.tmpl` et `invitation.txt.tmpl` (objet « Invitation à rejoindre {association} sur Frigg », lien `{FRIGG_BASE_URL}/invitation#<jeton>`, validité 7 jours)
- [X] T063 [US4] Implémenter `internal/api/invitations.go` : `list-invitations` (`GET /api/admin/invitations`), `create-invitation` (`POST /api/admin/invitations`, `{ email, role }`, `role` défaut `member`, 201), `resend-invitation` (`POST /api/admin/invitations/{id}/resend`), `cancel-invitation` (`DELETE /api/admin/invitations/{id}`, 204), tous `Admin` ; statut renvoyé `pending` ou `expired` (calculé) ; e-mail mis en file dans la même transaction
- [X] T064 [US4] Ajouter dans `internal/api/auth.go` `get-invitation` (`POST /api/auth/invitation/lookup`, `Public`, `{ token }` → `{ email, role }`) et `accept-invitation` (`POST /api/auth/invitation/accept`, `Public`, `{ token, password }`) : limite 10/min par IP, 410 « Cette invitation n'est plus valable », dans une transaction création du compte + `MarkInvitationAccepted`, puis `auth.Login` et réponse `Session`
- [X] T065 [US4] Régénérer les types (`go tool task gen`) ; créer `web/app/ui/ConfirmDialog.tsx` (base-ui `AlertDialog`, titre, texte, bouton de confirmation `danger`, « Annuler ») et `web/app/routes/app/admin/accounts.tsx` (« Comptes » : formulaire adresse + rôle « Membre » par défaut / « Administrateur », liste des invitations avec badge « En attente » ou « Expirée », actions « Renvoyer » et « Annuler » avec confirmation, messages de succès dans une région `role="status"`) ; ajouter `/admin/comptes` dans `web/app/routes.ts`
- [X] T066 [US4] Créer `web/app/routes/auth/accept-invitation.tsx` : jeton lu dans `location.hash` puis retiré de l'URL (`history.replaceState`), `get-invitation` au chargement, adresse affichée en lecture seule, mot de passe (`autocomplete="new-password"`, règle en texte d'aide) + confirmation vérifiée côté interface, erreurs 422 via `applyProblemToForm`, après succès navigation vers `/admin` ou `/` selon le rôle ; 410 → message et invitation à contacter l'association ; ajouter `/invitation` dans `web/app/routes.ts`
- [X] T067 [US4] Écrire `web/e2e/invitation.spec.ts` : administrateur invite une adresse, lien lu via `mailpit.ts`, acceptation dans un nouveau contexte navigateur, arrivée sur `/` sans sélecteur, lien rouvert refusé, annulation d'une invitation ; axe sur « Comptes » et l'acceptation

**Checkpoint**: des comptes membres peuvent être ouverts depuis l'interface

---

## Phase 7: User Story 5 - Réinitialiser un mot de passe oublié (Priority: P2)

**Goal**: demande de lien, choix d'un nouveau mot de passe, fermeture des sessions

**Independent Test**: demander un lien, le suivre, définir un nouveau mot de passe, se connecter ; l'ancienne session est fermée (quickstart scénario 7)

- [X] T068 [P] [US5] Test `internal/api/password_reset_test.go` : réponse 202 identique pour une adresse connue ou inconnue (e-mail en file seulement si le compte existe), lien valable 1 h, nouveau jeton → l'ancien est invalide, confirmation → `auth_epoch` incrémenté et autre session rejetée, jeton réutilisé ou expiré → 410, règle de mot de passe → 422 sur `body.password`, limites 3/h par adresse → 429
- [X] T069 [P] [US5] Créer `internal/mail/templates/password-reset.html.tmpl` et `password-reset.txt.tmpl` (objet « Réinitialisation de votre mot de passe », lien `{FRIGG_BASE_URL}/reinitialiser-mot-de-passe#<jeton>`, validité 1 heure, « Si vous n'êtes pas à l'origine de cette demande, ignorez ce message. »)
- [X] T070 [US5] Ajouter dans `internal/api/auth.go` `request-password-reset` (`POST /api/auth/password-reset`, `Public`, toujours 202 ; si le compte existe : `DeleteUnusedTokens` + `CreateToken` `purpose = 'password_reset'` expirant à +1 h + e-mail en file, dans une transaction) et `confirm-password-reset` (`POST /api/auth/password-reset/confirm`, `Public`, `{ token, password }` : `GetValidToken`, `ValidatePassword`, `UpdatePasswordAndBumpEpoch`, `MarkTokenUsed`, 204 ; 410 « Ce lien n'est plus valable »)
- [X] T071 [US5] Régénérer les types ; créer `web/app/routes/auth/forgot-password.tsx` (adresse, message neutre « Si un compte existe pour cette adresse, un lien de réinitialisation vient d'être envoyé. ») et `web/app/routes/auth/reset-password.tsx` (jeton lu dans le fragment puis retiré de l'URL, nouveau mot de passe + confirmation, erreurs 422 par critère, succès → lien vers `/connexion`, 410 → message et lien « Demander un nouveau lien ») ; ajouter `/mot-de-passe-oublie` et `/reinitialiser-mot-de-passe` dans `web/app/routes.ts`
- [X] T072 [US5] Écrire `web/e2e/password-reset.spec.ts` : demande, lien via Mailpit, nouveau mot de passe, ancienne session (second contexte) renvoyée vers la connexion, connexion avec le nouveau mot de passe ; axe sur les deux écrans

**Checkpoint**: un oubli de mot de passe ne demande plus d'intervention

---

## Phase 8: User Story 6 - Confirmer son adresse électronique (Priority: P2)

**Goal**: alerte permanente tant que l'adresse n'est pas confirmée, renvoi du lien, confirmation

**Independent Test**: avec le membre non confirmé du jeu de données e2e, voir l'alerte, renvoyer le lien, le suivre, l'alerte disparaît (quickstart scénario 8)

- [X] T073 [P] [US6] Test `internal/api/email_verification_test.go` : renvoi → e-mail en file et anciens liens invalides, adresse déjà confirmée → 409, 4ᵉ renvoi en une heure → 429, confirmation valable 48 h sans être connecté, jeton réutilisé ou expiré → 410
- [X] T074 [P] [US6] Créer `internal/mail/templates/email-verification.html.tmpl` et `email-verification.txt.tmpl` (objet « Confirmez votre adresse électronique », lien `{FRIGG_BASE_URL}/confirmer-adresse#<jeton>`, validité 48 heures)
- [X] T075 [US6] Implémenter `resend-email-verification` dans `internal/api/me.go` (`POST /api/me/email-verification`, `Member`, 202, limites 3/h par compte et 10/h par IP, `purpose = 'email_verification'` expirant à +48 h) et `confirm-email` dans `internal/api/auth.go` (`POST /api/auth/email-verification/confirm`, `Public`, limite 10/min par IP, `{ token }`, `MarkEmailVerified` + `MarkTokenUsed`, 204 ; 410)
- [X] T076 [US6] Régénérer les types ; créer `web/app/shell/VerifyEmailBanner.tsx` (alerte `warning` en tête du contenu si `emailVerified === false`, sans `role="alert"`, texte de l'artefact, bouton « Renvoyer le lien » avec état `aria-disabled` pendant l'envoi, résultat dans `role="status"`) et l'intégrer dans `web/app/shell/AppShell.tsx` ; créer `web/app/routes/auth/confirm-email.tsx` (confirmation automatique au chargement depuis le fragment, invalidation de `sessionQuery`, succès ou 410 avec lien vers l'accueil) ; ajouter `/confirmer-adresse` dans `web/app/routes.ts`
- [X] T077 [US6] Écrire `web/e2e/email-verification.spec.ts` : alerte visible pour le membre non confirmé, renvoi, lien via Mailpit, alerte disparue ; axe sur l'accueil avec alerte et sur la confirmation

**Checkpoint**: les adresses peuvent être confirmées

---

## Phase 9: User Story 7 - Gérer son mot de passe depuis « Mon compte » (Priority: P3)

**Goal**: changer son mot de passe en fournissant l'actuel

**Independent Test**: changer son mot de passe, se reconnecter avec le nouveau ; les autres sessions sont fermées (quickstart scénario 9)

- [X] T078 [P] [US7] Test `internal/api/change_password_test.go` : mot de passe actuel faux → 422 sur `body.currentPassword` « Le mot de passe actuel est incorrect. », règle non respectée → 422 sur `body.newPassword`, succès → 204, session courante conservée (nouvelle époque), autre session rejetée
- [X] T079 [US7] Implémenter `change-password` dans `internal/api/me.go` (`PUT /api/me/password`, `Member`, `{ currentPassword, newPassword }` : `VerifyPassword`, `ValidatePassword`, `UpdatePasswordAndBumpEpoch`, `auth.SetEpoch` sur la session courante, 204)
- [X] T080 [US7] Régénérer les types ; créer `web/app/routes/app/account.tsx` (« Mon compte » : adresse, rôle, statut de confirmation ; formulaire mot de passe actuel `autocomplete="current-password"`, nouveau `autocomplete="new-password"` + confirmation, règle en texte d'aide, erreurs via `applyProblemToForm`, succès annoncé dans `role="status"`) ; dans `web/app/routes.ts`, faire pointer `/mon-compte` vers `account.tsx` au lieu de `coming-soon.tsx`
- [X] T081 [US7] Écrire `web/e2e/account.spec.ts` : erreur sur le mot de passe actuel, succès, second contexte déconnecté, reconnexion ; axe sur « Mon compte »

**Checkpoint**: toutes les user stories sont fonctionnelles

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: commandes et outils transverses, vérification finale

- [X] T082 [P] Implémenter `internal/api/docs.go` (research.md › R18) : si `cfg.APIDocs`, monter `GET /api/openapi.json`, `GET /api/schemas/{name}`, `GET /api/docs` (page HTML qui charge Stoplight Elements depuis unpkg avec version épinglée et pointe vers `/api/openapi.json`), protégées par `auth.Allows(role, auth.Admin)` (401/403 sinon), CSP assouplie sur `/api/docs` uniquement (`script-src` et `style-src` + `https://unpkg.com`) ; avertissement `WARN` « documentation d'API activée : à réserver au développement » au démarrage ; drapeau inactif → 404
- [X] T083 [P] Test `internal/api/docs_test.go` : drapeau inactif → 404 sur les trois chemins ; actif → anonyme 401, membre 403, administrateur 200 ; CSP stricte partout ailleurs
- [X] T084 [P] Implémenter `internal/cli/backup.go` (`frigg backup <fichier>` : refuse un fichier existant `le fichier … existe déjà`, `VACUUM INTO`, rappel en sortie de sauvegarder `FRIGG_DATA_DIR` à part), branché dans `cmd/frigg/main.go`, + test `internal/cli/backup_test.go`
- [X] T085 [P] Écrire `web/e2e/keyboard.spec.ts` : parcours complet au clavier seul (liens d'évitement, menu du compte, sélecteur d'espace, déconnexion) et absence de défilement horizontal à 320 px sur chaque écran du shell (SC-005, SC-007)
- [X] T086 Vérifier qu'aucun e-mail, jeton, mot de passe ni cookie n'apparaît dans les journaux pendant `go test ./...` (capture du handler slog dans `internal/api/apitest`) et corriger si besoin
- [X] T087 Exécuter `go tool task check` et les scénarios 1 à 13 de `specs/001-auth-app-shell/quickstart.md` ; corriger tout écart

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance
- **Foundational (Phase 2)** : dépend de la phase 1 ; bloque toutes les stories
- **US3 (Phase 3)** : dépend de la phase 2
- **US1 (Phase 4)** : dépend de la phase 2 ; les tests e2e utilisent `e2eseed` (T039), pas US3
- **US2 (Phase 5)** : dépend de US1 (shell, `login.tsx`, layout protégé)
- **US4, US5, US6 (Phases 6 à 8)** : dépendent de US1 (connexion, shell) ; US4 utilise aussi l'espace administration de US2 ; US5 et US6 sont indépendantes entre elles et de US4
- **US7 (Phase 9)** : dépend de US1
- **Polish (Phase 10)** : après les stories voulues ; T082–T084 peuvent démarrer dès la phase 2

### User Story Dependencies

```text
Setup → Foundational ─┬─ US3
                      └─ US1 ─┬─ US2 ── US4
                              ├─ US5
                              ├─ US6
                              └─ US7
```

### Within Each User Story

- Tests rédigés d'abord ; ils doivent échouer avant l'implémentation
- Requêtes SQL et gabarits d'e-mail avant les opérations d'API
- Opérations d'API, puis régénération des types (`go tool task gen`), puis écrans
- Test e2e + axe en dernier

### Parallel Opportunities

- Phase 1 : T002 à T006 en parallèle après T001
- Phase 2 : T009–T012 ; T014–T020 ; T021 et T024 ; T027, T028, T031 ; T034–T037
- Dans chaque story, les tâches [P] (tests, requêtes, gabarits, composants indépendants)
- Après US1 : US5, US6 et US7 en parallèle ; US2 puis US4 sur une autre branche de travail

---

## Parallel Example: User Story 4

```bash
# Tests, requêtes et gabarit en parallèle :
Task: "T060 Test internal/api/invitations_test.go"
Task: "T061 Requêtes internal/database/queries/invitations.sql"
Task: "T062 Gabarits internal/mail/templates/invitation.*.tmpl"

# Puis en séquence : T063 → T064 → T065 → T066 → T067
```

## Parallel Example: Phase 2

```bash
Task: "T015 internal/auth/password.go"
Task: "T017 internal/auth/tokens.go"
Task: "T018 internal/auth/access.go"
Task: "T019 internal/auth/session.go"
Task: "T020 internal/auth/ratelimit.go"
Task: "T036 web/app/ui/Icon.tsx + Button.tsx"
Task: "T037 web/app/ui/Alert.tsx + Field.tsx"
```

---

## Implementation Strategy

### MVP First

1. Phases 1 et 2 (socle)
2. Phase 3 (US3) : on peut créer un administrateur
3. Phase 4 (US1) : connexion et shell
4. **STOP and VALIDATE** : quickstart scénarios 1 à 3, `go tool task check`

### Incremental Delivery

1. Socle + US3 + US1 → MVP utilisable par l'administrateur
2. US2 → navigation par rôle et espaces
3. US4 → ouverture de comptes membres depuis l'interface
4. US5 et US6 → autonomie des utilisateurs (mot de passe oublié, confirmation)
5. US7 → confort « Mon compte »
6. Polish → documentation de dev, sauvegarde, audit clavier, validation complète

---

## Notes

- [P] = fichiers différents, aucune dépendance sur une tâche inachevée
- Après toute modification d'opération d'API : `go tool task gen` puis commiter `web/app/api/openapi.json` et `schema.d.ts`
- Les tests e2e exigent `docker compose up -d` (Mailpit) et un binaire à jour (`go tool task e2e` s'en charge)
- Commiter après chaque tâche ou groupe logique ; s'arrêter à chaque checkpoint pour valider la story
