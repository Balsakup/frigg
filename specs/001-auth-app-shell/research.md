# Phase 0 — Recherche : Authentification et shell applicatif

Versions vérifiées le 2026-10-30 (`go get …@latest`, Go 1.27.3) : huma v2.40.0,
scs/v2 v2.9.0, goose v3.29.0, alexedwards/argon2id v1.0.0, go-mail v0.8.1,
godotenv v1.5.1, x/time v0.16.0, modernc.org/sqlite v1.61.0, sqlc v1.31.1
(installé). Frontend : versions déjà présentes dans `web/package.json`.

## R1. Organisation du binaire et des sous-commandes

- **Decision** : un seul `main` dans `cmd/frigg/`, avec un aiguillage manuel sur
  `os.Args[1]` (`serve`, `openapi`, `create-admin`, `backup`), et `flag.FlagSet`
  par sous-commande. Sans argument, `serve`.
- **Rationale** : quatre commandes simples, la bibliothèque standard suffit ; la
  liste des dépendances est fermée (constitution, Contraintes techniques).
- **Alternatives considered** : cobra (dépendance non listée, inutile à cette
  échelle) ; un binaire par commande (contraire au binaire unique).

## R2. Accès SQLite : deux pools et migrations

- **Decision** :
  - pool d'écriture : `SetMaxOpenConns(1)`, DSN
    `file:<path>?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)&_txlock=immediate` ;
  - pool de lecture : même DSN + `_pragma=query_only(1)`, `SetMaxOpenConns(4)` ;
  - deux jeux de requêtes sqlc (`sqlc.New(writeDB)`, `sqlc.New(readDB)`) ;
    les handlers reçoivent un petit type `Store` qui expose `Read`, `Write` et
    `WriteTx(ctx, func(*sqlc.Queries) error)` ;
  - goose embarqué (`embed.FS` dans `internal/database/migrations`), dialecte
    `sqlite3`, exécuté sur le pool d'écriture au démarrage de `serve` et de
    `create-admin`, échec = arrêt.
- **Rationale** : un seul écrivain évite `database is locked` ; `_txlock=immediate`
  (paramètre du pilote modernc) donne `BEGIN IMMEDIATE` à toute transaction ;
  `query_only` empêche une écriture accidentelle par le pool de lecture.
- **Alternatives considered** : un pool unique (verrous sous charge
  concurrente) ; `BEGIN IMMEDIATE` écrit à la main (oubli facile).

## R3. Génération SQL

- **Decision** : `sqlc.yaml` à la racine, moteur `sqlite`, schéma =
  `internal/database/migrations` (sqlc comprend les annotations goose),
  requêtes = `internal/database/queries/*.sql`, sortie =
  `internal/database/sqlc`. Binaire `sqlc` installé localement (v1.31.1) ;
  commande `go generate ./internal/database` qui l'appelle.
- **Rationale** : une seule source de vérité pour le schéma.
- **Alternatives considered** : `go tool sqlc` via la directive `tool` de
  go.mod (impose un compilateur C sur chaque poste à cause du parseur
  PostgreSQL embarqué dans sqlc ; reste possible plus tard).

## R4. API huma, routes publiques et documentation

- **Decision** :
  - `humago.NewWithPrefix(mux, "/api", cfg)` sur un `http.ServeMux` ;
  - `cfg.OpenAPIPath = ""`, `cfg.DocsPath = ""`, `cfg.SchemasPath = ""` :
    **huma publie par défaut `/openapi`, `/docs` et `/schemas` sans
    authentification**, ce qui violerait le principe II. Les routes intégrées
    sont donc toujours désactivées ; la documentation de développement est
    servie par nos propres routes, réservées aux administrateurs (voir R18) ;
    le schéma de référence reste produit par `frigg openapi` ;
  - chaque opération est déclarée via un helper `register(api, op, access, h)`
    où `access ∈ {public, member, admin}` est stocké dans
    `huma.Operation.Metadata["access"]` ; le helper refuse (panic au démarrage)
    une opération sans niveau d'accès ;
  - un middleware huma (`api.UseMiddleware`) lit ce niveau, charge le compte
    depuis la session et répond 401/403 ; la liste des opérations `public` est
    donc centralisée et vérifiable par test.
- **Rationale** : principes II et III — refus par défaut, rôle minimal déclaré
  par opération, une seule fonction d'autorisation.
- **Alternatives considered** : protéger par préfixe d'URL (`/api/admin/*`) seul
  — fragile, une route mal placée devient publique ; `huma.Operation.Security`
  seul — décoratif, ne contrôle rien sans middleware.

## R5. Sessions, expiration et invalidation des autres sessions

- **Decision** :
  - `scs.New()` avec `sqlite3store.NewWithCleanupInterval(writeDB, 30*time.Minute)` ;
    `Lifetime = 30 * 24h`, `IdleTimeout = 7 * 24h` (FR-006) ; cookie
    `frigg_session`, `Secure` (désactivable par `FRIGG_COOKIE_SECURE=false`
    pour un accès HTTP hors localhost), `HttpOnly`, `SameSite=Lax`, `Path=/` ;
  - session = `{account_id, auth_epoch}` ; `RenewToken` à la connexion,
    `Destroy` à la déconnexion ;
  - colonne `accounts.auth_epoch` incrémentée à chaque changement ou
    réinitialisation de mot de passe (et, plus tard, désactivation ou
    changement de rôle) ; le middleware compare l'époque de la session à celle
    du compte et détruit la session si elles diffèrent. La session courante
    est mise à jour avec la nouvelle époque après un changement de mot de passe
    (FR-010 : seules les *autres* sessions sont fermées) ;
  - la table `sessions` est créée par une migration goose (schéma documenté par
    sqlite3store : `token TEXT PRIMARY KEY, data BLOB NOT NULL, expiry REAL NOT NULL`
    + index sur `expiry`).
- **Rationale** : sqlite3store ne sait pas lister les sessions d'un compte ;
  l'époque évite de devoir les parcourir et couvre aussi les changements de
  rôle (cas limite de la spec) avec une seule lecture par requête, déjà
  nécessaire pour connaître le rôle.
- **Alternatives considered** : stocker l'identifiant du compte dans une table
  annexe des tokens (duplique la gestion de scs) ; rôle stocké dans la session
  (périmé après un changement de rôle).

## R6. Hachage des mots de passe et égalisation des temps

- **Decision** : `argon2id.CreateHash` avec des paramètres centralisés
  `Memory: 19*1024` (19 Mio), `Iterations: 2`, `Parallelism: 1`,
  `SaltLength: 16`, `KeyLength: 32` (recommandation OWASP). Pour une adresse
  inconnue, on compare quand même le mot de passe à un hash factice calculé au
  démarrage, afin que la durée de réponse ne révèle pas l'existence du compte.
- **Rationale** : `DefaultParams` (64 Mio, parallélisme = nombre de CPU) est
  coûteux en mémoire sur un petit serveur ; la limitation de débit (R7) borne
  le nombre de hachages simultanés.
- **Alternatives considered** : paramètres par défaut (risque de saturation
  mémoire) ; bcrypt (hors liste fermée).

## R7. Limitation de débit

- **Decision** : limiteurs `golang.org/x/time/rate` en mémoire, indexés par clé,
  avec purge périodique des clés inactives :

  | Action | Par adresse e-mail | Par IP |
  | --- | --- | --- |
  | Connexion | 5 / 15 min | 20 / min |
  | Demande de réinitialisation | 3 / h | 10 / h |
  | Renvoi du lien de confirmation | 3 / h (par compte) | 10 / h |
  | Acceptation d'invitation, confirmation, nouveau mot de passe | — | 10 / min |

  Dépassement : `429` avec le message neutre « Trop de tentatives. Réessayez
  dans quelques minutes. ». L'IP est `RemoteAddr`, sauf si
  `FRIGG_TRUSTED_PROXIES` liste le proxy : on lit alors `X-Forwarded-For`.
- **Rationale** : instance unique, un seul processus : la mémoire suffit ; un
  redémarrage remet les compteurs à zéro, ce qui est acceptable.
- **Alternatives considered** : compteurs en base (écritures à chaque tentative,
  contraire au pool d'écriture unique) ; verrouillage du compte (permet de
  bloquer un tiers en tapant son adresse).

## R8. Jetons envoyés par e-mail

- **Decision** : 32 octets de `crypto/rand`, encodés en base64url sans
  remplissage dans le lien ; seule l'empreinte SHA-256 est stockée. Table unique
  `one_time_tokens` avec `purpose ∈ {password_reset, email_verification}` ; les
  invitations portent leur propre empreinte (`invitations.token_hash`). Émettre
  un nouveau jeton invalide les précédents de même objet pour le même compte.
  Le jeton circule dans le **fragment** de l'URL (`/reinitialiser-mot-de-passe#<jeton>`)
  puis dans le corps de la requête POST, jamais dans un chemin ou une query
  string d'API (absent des journaux et des en-têtes `Referer`).
- **Rationale** : un jeton de 256 bits n'a pas besoin d'un hachage lent ;
  le fragment n'est jamais envoyé au serveur par le navigateur.
- **Alternatives considered** : argon2id sur les jetons (inutilement coûteux) ;
  jeton en query string (fuite via `Referer` et journaux de proxy).

## R9. File d'e-mails et worker

- **Decision** : table `outbound_emails` (destinataire, sujet, corps texte,
  corps HTML, état `pending|sent|failed`, tentatives, prochaine tentative,
  dernière erreur). Les messages sont rendus (`text/template` + `html/template`
  embarqués, en français) et insérés **dans la même transaction** que l'action
  qui les déclenche. Un worker lancé par `serve` se réveille toutes les
  `FRIGG_MAIL_POLL_INTERVAL` (défaut 30 s), traite par lots de 20, envoie via
  go-mail, et en cas d'échec reprogramme avec un délai exponentiel (1 min,
  2 min, 4 min… plafonné à 1 h) jusqu'à 10 tentatives, puis `failed`.
  Le `Context` d'arrêt du serveur arrête proprement le worker.
- **Rationale** : constitution (E-mails) ; l'utilisateur obtient une réponse
  immédiate même si le SMTP est indisponible (cas limite de la spec).
- **Alternatives considered** : envoi dans une goroutine sans persistance
  (perdu au redémarrage) ; bibliothèque cron (inutile, `time.Ticker` suffit).

## R10. Configuration

- **Decision** : `godotenv.Load()` (sans écraser l'environnement existant),
  puis lecture dans un struct `config.Config` validé au démarrage :

  | Variable | Obligatoire | Défaut |
  | --- | --- | --- |
  | `FRIGG_DATABASE_PATH` | non | `./data/frigg.db` |
  | `FRIGG_DATA_DIR` | non | `./data/files` |
  | `FRIGG_LISTEN_ADDR` | non | `:8080` |
  | `FRIGG_BASE_URL` | **oui** pour `serve` | — (sert aux liens des e-mails) |
  | `FRIGG_COOKIE_SECURE` | non | `true` |
  | `FRIGG_TRUSTED_PROXIES` | non | vide |
  | `FRIGG_SMTP_HOST`, `FRIGG_SMTP_PORT`, `FRIGG_SMTP_USERNAME`, `FRIGG_SMTP_PASSWORD` | `HOST` **oui** pour `serve` | port `587` |
  | `FRIGG_SMTP_TLS` | non | `starttls` (`starttls`, `tls`, `none`) |
  | `FRIGG_MAIL_FROM` | **oui** pour `serve` | — |
  | `FRIGG_MAIL_POLL_INTERVAL` | non | `30s` |
  | `FRIGG_LOG_LEVEL` | non | `info` |
  | `FRIGG_API_DOCS` | non | `false` (documentation d'API de développement, voir R18) |

  `create-admin`, `openapi` et `backup` n'exigent que ce dont ils ont besoin.
  `.env.example` documente chaque variable.
- **Rationale** : constitution (Configuration) ; échec explicite au démarrage.

## R11. Journaux

- **Decision** : `slog.New(slog.NewJSONHandler(os.Stdout, …))` ; middleware HTTP
  qui génère un identifiant de requête (16 octets aléatoires en hexadécimal),
  le place dans le contexte et dans l'en-tête `X-Request-Id`, et journalise une
  ligne par requête (méthode, chemin, statut, durée, `request_id`, `account_id`
  si connu). Messages en français, clés en anglais. Aucun corps de requête n'est
  journalisé.
- **Rationale** : constitution (Journaux, principe VII).

## R12. Service du SPA et en-têtes de sécurité

- **Decision** : `mux.Handle("/api/", …)` pour l'API (404 JSON pour une route
  API inconnue) ; tout le reste est servi depuis `web.Dist()` : fichier
  existant → servi (cache long pour `/assets/*`, fichiers à empreinte), sinon
  `index.html` (`Cache-Control: no-cache`). L'ensemble est enveloppé par
  `http.CrossOriginProtection`, puis `scs.LoadAndSave`, puis le middleware de
  journalisation. En-têtes ajoutés à toute réponse :
  `Referrer-Policy: same-origin`, `X-Content-Type-Options: nosniff`,
  `X-Frame-Options: DENY`, `X-Robots-Tag: noindex, nofollow` (application
  non indexable, doublé d'une balise `<meta name="robots">` dans le SPA), et une `Content-Security-Policy` restrictive
  (`default-src 'self'`, `style-src 'self' 'unsafe-inline'`, `img-src 'self' data:`).
  Seule exception : `/api/docs`, quand `FRIGG_API_DOCS=true` (voir R18).
- **Rationale** : principe II (les fichiers statiques ne contiennent aucune
  donnée), principe IV (polices auto-hébergées → `font-src 'self'` suffit).

## R13. Données côté frontend et garde de session

- **Decision** :
  - client `openapi-fetch` typé (`web/app/api/schema.d.ts` généré par
    `openapi-typescript` depuis `frigg openapi`), hooks `openapi-react-query` ;
  - un `QueryClient` unique ; le layout protégé a un `clientLoader` qui fait
    `queryClient.ensureQueryData(sessionQuery)` ; un 401 redirige vers
    `/connexion?suite=<chemin>` ;
  - un middleware `openapi-fetch` intercepte tout 401 ultérieur (session
    expirée) : vidage du cache et redirection avec le message « Votre session a
    expiré » ;
  - la valeur `suite` n'est acceptée que si elle commence par `/` et pas par
    `//` (FR-004).
- **Rationale** : constitution (Frontend : loaders + Query).

## R14. Espaces, URL et mémorisation

- **Decision** : l'espace se déduit de l'URL : `/admin/*` = administration, le
  reste = espace membre. Le sélecteur d'espace est un lien vers l'accueil de
  l'autre espace ; il « bascule tout seul » puisqu'il reflète l'URL. Le dernier
  espace d'un administrateur et l'état replié de la barre latérale sont stockés
  en `localStorage` (enveloppé dans try/catch) ; après connexion sans `suite`,
  un administrateur est redirigé vers `/admin` sauf si le dernier espace mémorisé
  est l'espace membre.
- **Rationale** : URL partageables et prévisibles, aucun état serveur pour une
  préférence d'affichage propre à l'appareil (spec FR-019a, FR-020).
- **Alternatives considered** : espace stocké en session serveur (ne serait plus
  « par appareil »).

## R15. Règle de mot de passe et validation

- **Decision** : la règle (≥ 8 caractères, ≥ 1 majuscule au sens
  `unicode.IsUpper`, ≥ 1 chiffre `unicode.IsDigit`) est implémentée une seule
  fois côté serveur ; chaque critère non respecté produit une entrée distincte
  dans `errors[]` du 422, à l'emplacement `body.password` (ou
  `body.newPassword`). Le frontend affiche la règle en texte d'aide statique et
  place les erreurs reçues sous le champ. `minLength: 8` figure aussi dans le
  schéma OpenAPI (tag huma), donc dans l'attribut `minLength` du champ.
- **Rationale** : principe VI — pas de schéma de validation écrit à la main côté
  client ; FR-008 satisfait par les erreurs par critère.
- **Alternatives considered** : indicateur en temps réel calculé côté client
  (duplique la règle) ; expression régulière OpenAPI (le moteur `regexp` de Go
  ne gère pas les assertions avant).

## R16. URL du frontend

- **Decision** : chemins visibles en français (chaînes destinées à l'humain),
  fichiers de route et identifiants en anglais :
  `/connexion`, `/mot-de-passe-oublie`, `/reinitialiser-mot-de-passe`,
  `/confirmer-adresse`, `/invitation`, `/` (accueil membre), `/mon-compte`,
  `/admin` (accueil administration), `/admin/comptes`. Détail dans
  [contracts/ui-routes.md](./contracts/ui-routes.md).
- **Rationale** : constitution, Conventions de langue ; cohérent avec l'artefact
  (`#tableau-de-bord`, `#mon-compte`).

## R17. Tests

- **Decision** :
  - Go : harnais `httptest` sur `server.NewHandler` (sessions et CSRF compris),
    `humatest` pour les vérifications de schéma, base SQLite fichier dans `t.TempDir()`
    migrée par goose (WAL ne fonctionne pas en mémoire partagée) ; faux
    expéditeur d'e-mails qui capture les messages de la file ;
  - **matrice d'autorisation générée** : un test parcourt toutes les opérations
    de l'OpenAPI, lit leur niveau d'accès et vérifie anonyme → 401 (sauf
    `public`), membre → 403 pour `admin`, administrateur → jamais 401/403 ;
    il échoue aussi si une opération `public` n'est pas dans la liste attendue
    (principe II). Les routes de documentation (R18), qui ne sont pas des
    opérations huma, ont leurs propres cas : drapeau inactif → 404 ; drapeau
    actif → anonyme 401, membre 403, administrateur 200 ;
  - frontend : Vitest + Testing Library pour les composants du shell ;
    Playwright + axe sur chaque écran livré, contre le binaire Go réel et une
    base jetable (le serveur Vite seul ne suffit plus dès que l'API est
    nécessaire).
- **Rationale** : constitution (Workflow et contrôles qualité).

## R18. Documentation d'API en développement

- **Decision** :
  - variable `FRIGG_API_DOCS`, `false` par défaut. Si elle vaut `true`, le
    serveur journalise au démarrage un avertissement (niveau `WARN`) :
    « documentation d'API activée : à réserver au développement » ;
  - les routes intégrées de huma restent désactivées (R4). Quand le drapeau est
    actif, trois routes maison sont montées sur le mux :
    - `GET /api/openapi.json` : `api.OpenAPI()` sérialisé ;
    - `GET /api/schemas/{name}.json` : un schéma de `components.schemas` ;
    - `GET /api/docs` : page HTML minimale qui charge Stoplight Elements et
      pointe vers `/api/openapi.json` ;
  - ces routes passent par la **même fonction d'autorisation** que l'API, avec
    le niveau `admin` : sans session → 401, membre → 403. Aucune route publique
    n'est ajoutée : pas d'amendement du principe II, et un drapeau oublié en
    production n'expose rien à un visiteur non connecté ;
  - drapeau inactif : ces trois chemins répondent 404, comme toute route API
    inconnue ;
  - CSP assouplie **uniquement** sur `/api/docs` et seulement drapeau actif :
    `script-src` et `style-src` autorisent en plus l'origine du CDN de Stoplight
    Elements (`https://unpkg.com`), version épinglée dans l'URL. Le reste de
    l'application garde la CSP stricte (principe IV : aucune ressource tierce
    dans l'application livrée aux utilisateurs) ;
  - en développement, la page s'ouvre via le proxy Vite
    (`http://localhost:5173/api/docs`) avec la session du navigateur.
- **Rationale** : outil de débogage confortable sans créer de route publique ni
  charger de ressource tierce dans l'application elle-même.
- **Alternatives considered** : routes publiques activées par drapeau (nouvelle
  route publique → amendement, et exposition totale en cas d'oubli) ; schéma
  seul sans interface (plus strict, mais moins pratique pour déboguer) ;
  interface auto-hébergée (Stoplight Elements embarqué dans le binaire :
  alourdit le binaire pour un outil de développement).
