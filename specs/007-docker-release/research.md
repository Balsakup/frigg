# Research: Image de production et publication à chaque release

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Date**: 2026-11-02

Chaque section suit le format Décision / Justification / Alternatives écartées.

## R1. Image d'exécution : distroless `static`, utilisateur `nonroot`

**Décision** : l'étape finale part de `gcr.io/distroless/static-debian13:nonroot`,
épinglée par empreinte (`@sha256:…`). Elle ne contient que le binaire
`/frigg`, la licence (`/usr/share/doc/frigg/LICENSE`) et le répertoire
`/data` appartenant à l'UID/GID `65532` (`nonroot`). Aucune instruction `RUN`
dans l'étape finale.

**Justification** :

- l'image `static` fournit exactement ce qu'il faut à un binaire
  `CGO_ENABLED=0` : certificats des autorités de certification (SMTP en TLS),
  `/etc/passwd` avec l'utilisateur `nonroot` (65532), `/tmp`, sans shell ni
  gestionnaire de paquets (FR-013, US3 scénario 2) ;
- elle pèse environ 1 Mo compressé et est suivie par les scanners de
  vulnérabilités (base Debian), contrairement à `scratch` où il faudrait
  copier et maintenir soi-même les certificats et `/etc/passwd` ;
- les fuseaux horaires sont déjà intégrés au binaire (`_ "time/tzdata"` dans
  `cmd/frigg/main.go`) : aucune dépendance à la base tzdata du système ;
- l'UID numérique fixe permet `USER 65532:65532`, vérifiable par
  `runAsNonRoot` dans Kubernetes (FR-012).

**Alternatives écartées** :

- `scratch` : quelques centaines de Ko de moins, mais certificats et
  utilisateur à gérer à la main, et invisibles des scanners ;
- `alpine` : shell et `apk` présents, surface d'attaque inutile (FR-013) ;
- `static-debian12` : utilisable en repli si `debian13` n'était pas encore
  publié au moment de l'implémentation ; la variante est fixée par
  l'empreinte de toute façon.

**Mesures (2026-11-02, `frigg:dev`, amd64)** : `static-debian13:nonroot`
disponible ; image compressée 8,3 Mo, binaire `/frigg` compressé 7,5 Mo, soit
0,8 Mo de plus (SC-003 : ≤ 5 Mo et < 50 Mo) ; 21 Mo non compressée. Trivy
0.70.0 : aucune vulnérabilité `CRITICAL`/`HIGH` corrigeable (Debian 13.7 et
modules Go du binaire).

## R2. Construction multi-étapes et multi-architecture sans émulation

**Décision** : `Dockerfile` à trois étapes, toutes les étapes de construction
s'exécutant sur la plateforme de la machine (`FROM --platform=$BUILDPLATFORM`) :

1. `web` (`oven/bun:<version>-alpine@sha256:…`) : `bun install
   --frozen-lockfile` puis `bun run build` → `web/build/client` ;
2. `build` (`golang:1.27-alpine@sha256:…`) : `go mod download`, copie de
   `cmd/`, `internal/`, `web/embed.go` et du SPA compilé, puis
   `CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath
   -buildvcs=false -ldflags="-s -w" -o /out/frigg ./cmd/frigg` ; préparation
   de `/out/data/files` ;
3. image finale (R1) : `COPY --from=build`.

Les caches de modules et de compilation passent par des montages de cache
BuildKit (`--mount=type=cache`).

**Justification** :

- le SPA est indépendant de l'architecture : il n'est construit qu'une fois ;
- Go compile nativement pour `arm64` depuis `amd64` : aucun besoin de QEMU,
  la construction `amd64` + `arm64` reste rapide (SC-001, FR-007) ;
- l'étape finale n'exécute aucune commande, donc aucune émulation non plus ;
- `-trimpath` et `-buildvcs=false` rendent le binaire indépendant du chemin de
  construction et de l'absence de `.git` dans le contexte ; `-s -w` retire
  les tables de symboles (taille, SC-003) ;
- outils et code source restent dans les étapes intermédiaires (FR-013,
  US3 scénario 6).

**Alternatives écartées** :

- construire le binaire hors de Docker puis le copier : la recette locale et
  la recette de release divergeraient (FR-024) ;
- `ko` : ne sait pas construire le SPA Bun, outil supplémentaire.

## R3. Épinglage et mise à jour des dépendances de construction

**Décision** : toutes les images de base (`docker/dockerfile`, `oven/bun`,
`golang`, `distroless`) sont épinglées par empreinte, avec l'étiquette lisible
à côté (`image:tag@sha256:…`). Toutes les actions GitHub tierces sont
épinglées par SHA de commit avec la version en commentaire. Un fichier
`.github/dependabot.yml` propose chaque semaine la mise à jour des
écosystèmes `docker` et `github-actions`.

**Justification** : FR-016 exige des constructions reproductibles ; les
étiquettes d'actions peuvent être réécrites lors d'une compromission de chaîne
d'approvisionnement (tj-actions/changed-files en 2025). Sans mise à jour
automatique, les empreintes figées vieilliraient et l'analyse de
vulnérabilités (R6) finirait par bloquer toute release ; Dependabot est déjà
prévu par le CLA (`allowlist`). Les empreintes et SHA exacts sont relevés au
moment de l'implémentation (`docker buildx imagetools inspect`,
`gh api repos/<owner>/<action>/commits/<tag>`).

**Alternatives écartées** : étiquettes mobiles (`golang:1.27`) sans
empreinte : non reproductible ; Renovate : plus puissant mais configuration
plus lourde pour un projet à un mainteneur.

## R4. Contexte de construction en liste blanche

**Décision** : `.dockerignore` commence par `*` puis réautorise seulement
`cmd/`, `internal/`, `web/` (hors `web/node_modules`, `web/build`,
`web/.react-router`, `web/test-results`, `web/playwright-report`), `go.mod`,
`go.sum` et `LICENSE`.

**Justification** : une liste blanche exclut par construction `.env`, `data/`,
`bin/`, `.git`, `specs/`, `.claude/` et tout fichier ajouté plus tard (FR-017).
Elle réduit aussi le contexte envoyé au moteur de construction.

**Alternative écartée** : liste noire : il suffit d'oublier un fichier
sensible pour qu'il entre dans le contexte.

## R5. Configuration par défaut, volume et répertoire de travail

**Décision** : l'image fixe :

| Élément | Valeur |
| --- | --- |
| `ENTRYPOINT` / `CMD` | `["/frigg"]` / `["serve"]` |
| `USER` | `65532:65532` |
| `WORKDIR` | `/data` |
| `VOLUME` | `/data` |
| `EXPOSE` | `8080` |
| `FRIGG_DATABASE_PATH` | `/data/frigg.db` |
| `FRIGG_DATA_DIR` | `/data/files` |
| `FRIGG_LISTEN_ADDR` | `:8080` |

**Justification** :

- forme exec : `frigg` est le PID 1 et reçoit directement `SIGTERM`, déjà géré
  par `signal.NotifyContext` et l'arrêt en 10 s de `server.Run` (FR-022) ;
  `frigg` ne crée aucun processus enfant, donc pas besoin d'un init ;
  `docker stop` envoie `SIGKILL` au bout de 10 s par défaut, soit le même
  délai que l'arrêt interne : un Dockerfile ne peut pas fixer ce délai, la
  documentation recommande donc `--stop-timeout 20` (`stop_grace_period: 20s`
  en Compose) pour que la base soit toujours fermée proprement ; le délai par
  défaut de Kubernetes (30 s) suffit ;
- `CMD ["serve"]` permet `docker run … balsakup/frigg create-admin` ou
  `backup` en remplaçant seulement l'argument (FR-019) ;
- `WORKDIR /data` : un chemin relatif passé à `frigg backup` atterrit dans le
  volume ; un `.env` éventuellement déposé par l'exploitant dans le volume est
  lu par `godotenv`, l'environnement du conteneur gardant la priorité ;
  l'image elle-même n'en contient aucun (FR-021) ;
- `/data` et `/data/files` existent dans l'image avec le propriétaire 65532 :
  un volume nommé vide hérite de ces droits au premier montage ;
- `FRIGG_BASE_URL`, `FRIGG_SMTP_HOST` et `FRIGG_MAIL_FROM` restent sans valeur
  par défaut : `RequireServe` arrête le conteneur avec le message existant
  (US2 scénario 2).

**Alternatives écartées** : `WORKDIR /` (les sauvegardes relatives échoueraient
sur un système de fichiers en lecture seule) ; variables `PUID`/`PGID`
appliquées par un script d'entrée (exige un shell et un démarrage en root).

## R6. Système de fichiers en lecture seule : fichiers temporaires de SQLite

**Décision** : la chaîne de connexion ajoute `_pragma=temp_store(MEMORY)`
(fonction `dsn` de `internal/database/db.go`), pour tous les environnements.

**Justification** : SQLite peut créer des fichiers temporaires (tris,
index temporaires) dans `SQLITE_TMPDIR`, `TMPDIR`, `/var/tmp`, `/usr/tmp`,
`/tmp` puis le répertoire courant. Avec `--read-only`, seul le repli sur le
répertoire courant fonctionnerait, ce qui est fragile. En mémoire, le
comportement ne dépend plus du système de fichiers et l'image n'a besoin
d'aucun `tmpfs` (FR-014, SC-005). La base d'une association reste petite :
l'usage mémoire supplémentaire est négligeable. WAL et `-shm` restent à côté de
la base, dans `/data`.

**Alternatives écartées** : imposer `--tmpfs /tmp` dans la documentation
(contredit FR-014 : seul le volume doit être accessible en écriture) ;
`TMPDIR=/data/tmp` (répertoire absent d'un montage lié, repli silencieux).

## R7. Vérification d'écriture au démarrage

**Décision** : avant d'ouvrir la base, `cli.Serve` crée au besoin
(`MkdirAll`, `0o750`) le répertoire de la base et `FRIGG_DATA_DIR`, puis y
crée et supprime un fichier témoin (`os.CreateTemp`). En cas d'échec, le
démarrage s'arrête avec :
`le répertoire <chemin> n'est pas accessible en écriture par l'utilisateur
<uid> : <cause> (en conteneur, le volume doit appartenir à l'UID 65532)`.

**Justification** : FR-023 et cas limite « volume appartenant à un autre
utilisateur ». Sans ce contrôle, SQLite échouerait avec un message peu clair,
et `FRIGG_DATA_DIR` (qui n'est aujourd'hui créé par aucun code) ne le serait
qu'au premier dépôt de fichier. Aucun parcours de dépôt de fichier n'existe
encore : la spec (US2, US3, SC-005) vérifie donc ce contrôle d'écriture et
non un dépôt.

**Alternative écartée** : contrôle dans un script d'entrée (pas de shell).

## R8. Point de santé

**Décision** : opération huma publique `get-health`, `GET /api/health`,
tag « Santé », déclarée avec `register(api, auth.Public, …)` :

- exécute `SELECT 1` sur le pool de lecture via une requête sqlc
  `HealthCheck` (`internal/database/queries/health.sql`), avec un délai de
  2 s ;
- succès : `200`, `{"status":"ok"}`, `Cache-Control: no-store` ;
- échec : `503` problème huma « Service indisponible. » sans détail ; la cause
  est journalisée au niveau `warn` (`vérification de santé en échec`) ;
- ajout de `get-health` à `allowedPublic` dans
  `internal/api/access_test.go`, la liste centralisée qui fait échouer les
  tests si une opération publique n'est pas autorisée par le principe II.

**Justification** :

- passer par `register` garde le refus par défaut et la déclaration de niveau
  d'accès (principe III), et l'opération apparaît dans l'OpenAPI (principe VI) ;
- `GET` sans corps : exempté par `CrossOriginProtection`, et `scs` ne lit la
  base que si un cookie est présent : la vérification de santé n'en envoie pas ;
- aucune limitation de débit : les limites existantes sont appliquées
  explicitement par les opérations d'authentification, la nouvelle opération
  ne consomme donc aucun quota (cas limite) ;
- seule la base est vérifiée : le worker d'e-mails réessaie seul (cas limite
  SMTP) ;
- pendant les migrations, le serveur HTTP n'écoute pas encore : la
  vérification échoue, couverte par le délai de démarrage (R9).

**Alternatives écartées** :

- handler `net/http` brut hors de huma : sortirait de la matrice
  d'autorisation automatique et de l'OpenAPI ;
- `PingContext` : ne garantit pas qu'une requête aboutit ;
- exposer version ou durée de fonctionnement : interdit par l'amendement 1.5.0.

## R9. Vérification de santé de l'image : `frigg healthcheck`

**Décision** : nouvelle sous-commande `frigg healthcheck` (pas de dépendance) :
lit la configuration (`config.Load`, sans `RequireServe`), déduit l'URL de
`FRIGG_LISTEN_ADDR` (hôte vide, `0.0.0.0` ou `::` → `127.0.0.1`), appelle
`GET /api/health` avec un délai de 3 s, termine avec le code `0` sur un `200`,
`1` sinon, en écrivant la raison sur la sortie d'erreur. Dans le `Dockerfile` :

```text
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s \
  --start-interval=2s --retries=3 CMD ["/frigg", "healthcheck"]
```

**Justification** : l'image n'a ni shell ni `curl` (FR-032). `--start-interval`
fait passer le conteneur à « sain » quelques secondes après le démarrage, et
trois échecs espacés de 30 s le signalent « en mauvaise santé » en moins de
2 minutes (SC-009). Kubernetes ignore `HEALTHCHECK` : la documentation donne
la sonde `httpGet` équivalente sur `/api/health`.

**Alternatives écartées** : binaire `curl` statique copié dans l'image (outil
superflu, FR-013) ; vérification TCP seule (ne voit pas une base inaccessible).

## R10. Journalisation du point de santé

**Décision** : `requestLogger` (`internal/server/middleware.go`) journalise au
niveau `debug` les requêtes `GET /api/health` de statut inférieur à 400, au
niveau `info` toutes les autres comme aujourd'hui. Le handler journalise un
échec en `warn` avec la cause.

**Justification** : FR-031 ; une vérification toutes les 30 s produirait près
de 3 000 lignes par jour au niveau `info`.

## R11. Publication : workflow `release.yml`

**Décision** : `.github/workflows/release.yml`, déclenché par
`release: types: [published]`, avec `permissions: {}` au niveau du workflow et
des permissions minimales par job :

1. **`verify`** (`contents: read`) : checkout de l'étiquette
   (`fetch-depth: 0`), contrôle du format (R12), calcul de `version`,
   `prerelease` et `latest` (R12), puis contrôles : `bun install
   --frozen-lockfile`, `bun run build`, `CGO_ENABLED=0 go build ./...`,
   `go vet ./...`, `go test ./...`, et dans `web/` `bun run typecheck`,
   `bun run lint`, `bun run test` (FR-006, SC-008) ;
2. **`image`** (`contents: read`, `packages: write`, `id-token: write`,
   `attestations: write`), après `verify` :
   - `docker/setup-buildx-action`, `docker/login-action` sur `ghcr.io` avec
     `GITHUB_TOKEN` ;
   - `docker/metadata-action` : étiquettes `type=raw,value=<version>` et
     `latest` seulement si `latest=true` (`flavor: latest=false` sinon),
     étiquettes OCI `version`, `revision`, `created`, `source`, `title`,
     `description`, `licenses=AGPL-3.0-or-later` (FR-008) ;
   - construction `linux/amd64` chargée localement, analysée par Trivy
     (`severity: CRITICAL,HIGH`, `ignore-unfixed: true`, `exit-code: 1`) :
     un échec arrête le job avant toute publication (FR-015) ;
   - construction et publication `linux/amd64,linux/arm64` (cache
     `type=gha`, donc rapide), `provenance: mode=max`, `sbom: true` ;
   - `actions/attest-build-provenance` avec `push-to-registry: true`
     (attestation signée vérifiable par `gh attestation verify`, FR-009) ;
3. **`announce`** (`contents: write`), après `image` : ajoute aux notes de la
   release le nom complet de l'image et son empreinte.

`concurrency: release-${{ github.ref }}` et `timeout-minutes: 30`.

**Justification** :

- `GITHUB_TOKEN` avec `packages: write` suffit pour le registre du dépôt :
  aucun secret à créer ni à faire tourner (FR-011) ; il n'est jamais passé en
  argument de construction ;
- un checkout de l'étiquette garantit FR-002 ;
- l'analyse porte sur l'image `amd64` : le binaire et la base distroless sont
  les mêmes sources pour `arm64`, seules les architectures diffèrent ;
- un échec est visible dans l'onglet Actions et notifié au mainteneur par
  GitHub ; un succès est visible directement dans la release (job
  `announce`) ;
- une relance (« Re-run jobs ») reconstruit la même étiquette sans toucher
  aux autres versions (cas limite).

**Alternatives écartées** :

- déclenchement sur `push` d'étiquette : publierait sans release, contraire à
  la demande ;
- jeton personnel (PAT) : secret à longue durée de vie, inutile ;
- Docker Hub : la demande vise le registre GitHub ;
- tests de bout en bout Playwright à la release : lourds (navigateurs, Mailpit),
  exclus par hypothèse de la spec.

## R12. Règles d'étiquetage et `latest`

**Décision** : dans `verify`, un script bash :

- refuse toute étiquette ne correspondant pas à
  `^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$`
  avec le message `étiquette <tag> invalide : format attendu vX.Y.Z ou
  vX.Y.Z-suffixe` (FR-005) ;
- `version` = étiquette sans `v` ;
- `prerelease` = suffixe présent **ou** release marquée préversion ;
- `latest` = non `prerelease` **et** l'étiquette est la plus haute des
  étiquettes stables du dépôt (`git tag -l 'v*'` filtré sur `vX.Y.Z`, trié
  par `sort -V`).

**Justification** : `docker/metadata-action` avec `latest=auto` déplacerait
`latest` sur un correctif d'une ancienne branche (`v1.1.4` publié après
`v1.2.0`), contraire à FR-004. Une incohérence entre suffixe et case
« préversion » est traitée prudemment : jamais `latest`.

**Alternative écartée** : interroger le registre pour connaître la plus haute
version publiée : appel réseau supplémentaire ; les étiquettes git suffisent.

## R13. Visibilité du paquet

**Décision** : au premier passage, GitHub crée le paquet `frigg` en privé ; le
mainteneur le rend public une fois dans les réglages du paquet (procédure
décrite dans `CONTRIBUTING.md`). L'étiquette OCI `source` rattache le paquet
au dépôt.

**Justification** : la visibilité d'un paquet ne se règle pas par l'API avec
`GITHUB_TOKEN` ; c'est une opération unique (FR-010).

## R14. Construction locale : tâche `docker:build`

**Décision** : tâche `docker:build` dans `Taskfile.yml` :

- `desc` en français ;
- `preconditions` : `command -v docker`, message « Docker est requis pour
  construire l'image locale. » ;
- variables : révision courte (`git rev-parse --short HEAD`) et marqueur
  `-modifie` si l'arbre de travail n'est pas propre ;
- `docker build --build-arg VERSION=dev-<rev>[-modifie] --build-arg
  REVISION=<sha> -t frigg:dev .`

**Justification** : même `Dockerfile` que la release (FR-024), architecture
du poste par défaut, aucune connexion au registre ni `--push` (FR-025), nom
`frigg:dev` impossible à confondre avec `ghcr.io/balsakup/frigg:<version>`.
Les arguments `VERSION` et `REVISION` alimentent les étiquettes OCI du
`Dockerfile` (US4 scénario 3), écrasées en CI par `docker/metadata-action`.

## R15. Documentation

**Décision** :

- `README.md`, nouvelle section « Installer avec Docker » : exemple
  `docker run` et `compose.yaml` de production (dans la documentation, pas
  dans le dépôt), variables obligatoires, volume et propriétaire `65532`,
  premier administrateur (`docker exec -it frigg /frigg create-admin`),
  sauvegarde et restauration, mise à jour, durcissement (`--read-only`,
  `--cap-drop=ALL`, `--security-opt=no-new-privileges`), reverse proxy et
  `FRIGG_TRUSTED_PROXIES`, sonde Kubernetes, vérification de l'attestation ;
- `CONTRIBUTING.md` : « Publier une release » (format d'étiquette,
  préversions, `latest`, rendre le paquet public la première fois, relance)
  et « Construire l'image en local » (`go tool task docker:build`) (FR-026,
  FR-027).

Le `compose.yaml` du dépôt reste réservé au développement (hypothèse de la
spec).
