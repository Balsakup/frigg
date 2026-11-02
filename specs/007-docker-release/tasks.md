---

description: "Liste des tâches : image de production et publication à chaque release"
---

# Tasks: Image de production et publication à chaque release

**Input**: Design documents from `specs/007-docker-release/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, cli.md, image.md, release.md), quickstart.md

**Tests**: inclus pour le code Go. La spécification ne les demande pas
explicitement, mais la constitution impose les tests d'autorisation
(principes II et III) et `go test ./...` avant fusion. Le `Dockerfile`, le
workflow et la tâche Taskfile se valident par les scénarios de
[quickstart.md](./quickstart.md).

**Organization**: une phase par user story, dans l'ordre de la spec (US1, US2,
US3 en P1 ; US4 en P2). Le `Dockerfile` de base est fondationnel : toutes les
stories en dépendent. US3 complète des fichiers créés par US1 (`release.yml`)
et US2 (`Dockerfile`) : la faire après elles.

**Conventions** (constitution 1.5.0) : identifiants, champs JSON, clés de
journal et de configuration en anglais ; commentaires, messages, sorties CLI,
descriptions de tâches et documentation en français. Commentaires des
requêtes sqlc en ASCII uniquement. Aucune dépendance Go ni frontend ajoutée.
Images de base épinglées `image:tag@sha256:…` (empreinte relevée avec
`docker buildx imagetools inspect <image:tag>`) ; actions GitHub tierces
épinglées par SHA de commit complet avec `# vX.Y.Z` en commentaire (SHA relevé
avec `gh api repos/<owner>/<repo>/commits/<tag> -q .sha`).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US4)

## Path Conventions

Un module Go à la racine (`internal/`, `cmd/frigg`) et le SPA dans `web/`.
Code existant à réutiliser :

- API : `internal/api/api.go` (`register`, `auth.Public`, `s.Logger`,
  `s.Store.Read`), constantes de tag `tag…` en tête de chaque fichier
  d'opérations ; `internal/api/access_test.go` (`allowedPublic`,
  `TestAccessMatrix` lit l'OpenAPI).
- Tests d'API : `internal/api/apitest` (`New`, `Client().Do`, `Response`
  avec `Status`, `Header`, `JSON`, `Logs` (JSON `slog` de niveau `debug`),
  `Store` (`ReadDB`, `Read`)).
- Serveur : `internal/server/middleware.go` (`requestLogger`),
  `internal/server/spa.go` (404 JSON pour `/api/*` inconnu).
- CLI : `internal/cli/serve.go` (`Serve`), `internal/cli/backup.go`,
  `internal/config/config.go` (`Load`, `RequireServe`, `invalid`),
  `cmd/frigg/main.go` (`usage`, `switch cmd`).
- Base : `internal/database/db.go` (`Open`, `dsn`),
  `internal/database/queries/*.sql`, `sqlc.yaml`.
- Commandes : `go tool task gen` (sqlc, OpenAPI, types TS), `go tool task
  test`, `go tool task check`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: contexte de construction sûr et versions de référence.

- [X] T001 Créer `.dockerignore` en liste blanche ([research.md › R4](./research.md#r4-contexte-de-construction-en-liste-blanche)) : première ligne `*`, puis `!cmd/`, `!internal/`, `!web/`, `!go.mod`, `!go.sum`, `!LICENSE`, puis réexclure `web/node_modules`, `web/build`, `web/.react-router`, `web/test-results`, `web/playwright-report`, `web/.agents` (les fichiers `*_test.go` restent dans le contexte : ils ne compilent pas dans le binaire et n'atteignent pas l'image finale) ; commentaire de tête en français expliquant que tout fichier non listé (`.env`, `data/`, `bin/`, `.git`) est exclu par défaut
- [X] T002 [P] Relever et noter dans un commentaire de tête de `Dockerfile` (créé en T003) les empreintes actuelles de `docker/dockerfile:1`, `oven/bun:1.3.14-alpine`, `golang:1.27-alpine` (version de `go.mod` : 1.27.3, utiliser `golang:1.27.3-alpine` si l'étiquette existe) et `gcr.io/distroless/static-debian13:nonroot` (repli `static-debian12:nonroot` si `debian13` n'existe pas, et le signaler dans [research.md › R1](./research.md#r1-image-dexécution--distroless-static-utilisateur-nonroot)) avec `docker buildx imagetools inspect`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: une image construisible et exécutable, utilisée par toutes les stories.

**⚠️ CRITICAL**: aucune story ne commence avant la fin de cette phase.

- [X] T003 Créer `Dockerfile` ([research.md › R2, R5](./research.md#r2-construction-multi-étapes-et-multi-architecture-sans-émulation), [contracts/image.md](./contracts/image.md)) : `# syntax=docker/dockerfile:1@sha256:…` ; commentaires en français ; étape `web` `FROM --platform=$BUILDPLATFORM oven/bun:1.3.14-alpine@sha256:… AS web`, `WORKDIR /src/web`, `COPY web/package.json web/bun.lock ./`, `RUN --mount=type=cache,target=/root/.bun/install/cache bun install --frozen-lockfile`, `COPY web/ ./`, `RUN bun run build` ; étape `build` `FROM --platform=$BUILDPLATFORM golang:1.27…-alpine@sha256:… AS build`, `ARG TARGETOS TARGETARCH`, `WORKDIR /src`, `COPY go.mod go.sum ./`, `RUN --mount=type=cache,target=/go/pkg/mod go mod download`, `COPY cmd/ cmd/`, `COPY internal/ internal/`, `COPY web/embed.go web/`, `COPY --from=web /src/web/build/client web/build/client`, `RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/frigg ./cmd/frigg`, `RUN mkdir -p /out/data/files && chmod 0750 /out/data /out/data/files` ; étape finale `FROM gcr.io/distroless/static-debian13:nonroot@sha256:…`, `ARG VERSION=dev REVISION=inconnue CREATED=`, `LABEL` `org.opencontainers.image.title="Frigg"`, `.description="Gestion d'association auto-hébergée"`, `.version="$VERSION"`, `.revision="$REVISION"`, `.created="$CREATED"`, `.source="https://github.com/Balsakup/frigg"`, `.licenses="AGPL-3.0-or-later"` ; `COPY --from=build --chown=65532:65532 /out/data /data`, `COPY --from=build /out/frigg /frigg`, `COPY LICENSE /usr/share/doc/frigg/LICENSE` ; `ENV FRIGG_DATABASE_PATH=/data/frigg.db FRIGG_DATA_DIR=/data/files FRIGG_LISTEN_ADDR=:8080` ; `USER 65532:65532`, `WORKDIR /data`, `VOLUME ["/data"]`, `EXPOSE 8080`, `ENTRYPOINT ["/frigg"]`, `CMD ["serve"]` ; aucune instruction `RUN` dans l'étape finale
- [X] T004 Construire et lancer l'image pour valider la base : `docker build -t frigg:dev .` puis `docker run --rm frigg:dev` doit s'arrêter sur `configuration invalide : FRIGG_BASE_URL est obligatoire ; …` et `docker run --rm frigg:dev help` afficher l'aide (corriger `Dockerfile` ou `.dockerignore` jusqu'à obtenir ce résultat)

**Checkpoint**: l'image se construit pour l'architecture du poste et démarre le binaire.

---

## Phase 3: User Story 1 - Publier l'image en créant une release (Priority: P1) 🎯 MVP

**Goal**: la publication d'une release `vX.Y.Z` construit, vérifie et publie `ghcr.io/balsakup/frigg:<version>` (amd64 + arm64), avec `latest` sur la plus haute version stable.

**Independent Test**: [quickstart.md › 7](./quickstart.md#7-release-us1) sur une préversion `v0.1.0-rc.1`, puis une étiquette invalide, puis `v0.1.0`.

### Implementation for User Story 1

- [X] T005 [US1] Créer `.github/workflows/release.yml`, en-tête et job `verify` ([contracts/release.md](./contracts/release.md), [research.md › R11, R12](./research.md#r11-publication--workflow-releaseyml)) : commentaire de tête en français ; `name: Release` ; `on: release: types: [published]` ; `permissions: {}` ; `concurrency: { group: release-${{ github.ref }}, cancel-in-progress: false }` ; `env: IMAGE: ghcr.io/balsakup/frigg` ; job `verify` (`runs-on: ubuntu-latest`, `timeout-minutes: 30`, `permissions: contents: read`, `outputs: version, latest, prerelease`) : `actions/checkout` (`fetch-depth: 0`, `persist-credentials: false`) ; étape `tag` en bash `set -euo pipefail` qui lit `TAG=${{ github.event.release.tag_name }}` et `RELEASE_PRERELEASE=${{ github.event.release.prerelease }}` via `env:` (jamais interpolés directement dans le script), refuse l'étiquette si elle ne correspond pas à `^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$` avec `::error::étiquette $TAG invalide : format attendu vX.Y.Z ou vX.Y.Z-suffixe` et `exit 1`, calcule `version=${TAG#v}`, `prerelease=true` si suffixe présent ou `RELEASE_PRERELEASE=true`, `highest=$(git tag -l 'v*' | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -n1)`, `latest=true` seulement si `prerelease=false` et `TAG == highest`, et écrit les trois valeurs dans `$GITHUB_OUTPUT` ; `actions/setup-go` (`go-version-file: go.mod`, cache activé) ; `oven-sh/setup-bun` (`bun-version: 1.3.14`) ; étapes `bun install --frozen-lockfile` et `bun run build` dans `web/` ; `CGO_ENABLED=0 go build ./...` ; `go vet ./...` ; `go test ./...` ; dans `web/` : `bun run typecheck`, `bun run lint`, `bun run test` ; toutes les actions épinglées par SHA
- [X] T006 [US1] Ajouter le job `image` à `.github/workflows/release.yml` : `needs: verify`, `timeout-minutes: 30`, `permissions: contents: read, packages: write, id-token: write, attestations: write`, `outputs: digest` ; `actions/checkout` (`persist-credentials: false`) ; `docker/setup-buildx-action` ; `docker/login-action` (`registry: ghcr.io`, `username: ${{ github.actor }}`, `password: ${{ secrets.GITHUB_TOKEN }}`) ; `docker/metadata-action` (`images: ${{ env.IMAGE }}`, `flavor: latest=false`, `tags:` `type=raw,value=${{ needs.verify.outputs.version }}` et `type=raw,value=latest,enable=${{ needs.verify.outputs.latest == 'true' }}`, `labels:` `org.opencontainers.image.title=Frigg`, `org.opencontainers.image.description=Gestion d'association auto-hébergée`, `org.opencontainers.image.version=${{ needs.verify.outputs.version }}`, `org.opencontainers.image.licenses=AGPL-3.0-or-later`) ; `docker/build-push-action` id `push` (`context: .`, `platforms: linux/amd64,linux/arm64`, `push: true`, `tags`/`labels` issus de metadata, `build-args: VERSION=…`, `REVISION=${{ github.sha }}`, `CREATED=${{ fromJSON(steps.meta.outputs.json).labels['org.opencontainers.image.created'] }}`, `cache-from: type=gha`, `cache-to: type=gha,mode=max`, `provenance: mode=max`, `sbom: true`) ; `actions/attest-build-provenance` (`subject-name: ${{ env.IMAGE }}`, `subject-digest: ${{ steps.push.outputs.digest }}`, `push-to-registry: true`) ; sortie `digest` = `steps.push.outputs.digest` ; aucun secret passé en `build-args`
- [X] T007 [US1] Ajouter le job `announce` à `.github/workflows/release.yml` : `needs: [verify, image]`, `permissions: contents: write`, étape bash avec `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}`, `TAG`, `VERSION`, `DIGEST` passés par `env:` : lit les notes (`gh release view "$TAG" --repo "$GITHUB_REPOSITORY" --json body -q .body`), et si elles ne contiennent pas déjà `ghcr.io/balsakup/frigg:$VERSION` (relance idempotente), les remplace par les notes existantes suivies d'une ligne vide et de `**Image** : \`ghcr.io/balsakup/frigg:$VERSION\`` puis `(\`$DIGEST\`, linux/amd64 et linux/arm64)` via `gh release edit "$TAG" --repo "$GITHUB_REPOSITORY" --notes-file -`
- [X] T008 [P] [US1] Ajouter à `CONTRIBUTING.md` une section « Publier une release » ([contracts/release.md](./contracts/release.md)) : format d'étiquette `vX.Y.Z` et préversions `vX.Y.Z-rc.N`, tableau étiquette → étiquettes d'image (dont le cas `v1.1.4` après `v1.2.0`), ce que vérifie le workflow, où suivre l'exécution (onglet Actions, notification d'échec, ligne « Image » ajoutée aux notes en cas de succès), relance par « Re-run jobs », opération unique « rendre le paquet `frigg` public » (Paramètres du paquet → Change visibility → Public), retrait d'une image publiée à la main
- [X] T009 [US1] Valider le workflow sans publier : `actionlint .github/workflows/release.yml` si disponible (`go run github.com/rhysd/actionlint/cmd/actionlint@latest`, sans l'ajouter à `go.mod`), puis tester le script de l'étape `tag` en local avec les étiquettes de [contracts/release.md › Étiquette de la release](./contracts/release.md#étiquette-de-la-release) dans un dépôt temporaire contenant des étiquettes `v1.1.3`, `v1.2.0`, `v1.3.0-rc.1`

**Checkpoint**: une release publie l'image multi-architecture ; une étiquette invalide ou un test en échec n'en publie aucune.

---

## Phase 4: User Story 2 - Exploiter Frigg à partir de l'image (Priority: P1)

**Goal**: le conteneur démarre, signale sa santé, refuse un volume non accessible en écriture, permet `create-admin`, `backup` et la mise à jour sans perte.

**Independent Test**: [quickstart.md › 4, 5 et 6](./quickstart.md#4-exploitation-durcie-us2-us3).

### Tests for User Story 2 ⚠️

> Écrire ces tests d'abord et vérifier qu'ils échouent avant l'implémentation.

- [X] T010 [P] [US2] Ajouter `"get-health": true` à `allowedPublic` dans `internal/api/access_test.go`, avec le commentaire « point de santé (amendement 1.5.0) »
- [X] T011 [P] [US2] Créer `internal/api/health_test.go` (package `api_test`, descriptions en français) ([contracts/api.md](./contracts/api.md)) : `TestHealthOK` — client anonyme, `GET /api/health` → `200`, corps exactement `{"status":"ok"}` (comparaison après décodage JSON, aucune autre clé), en-tête `Cache-Control` = `no-store`, aucun `Set-Cookie` ; `TestHealthDatabaseDown` — `env.Store.ReadDB.Close()` puis `GET /api/health` → `503`, `detail` = `Service indisponible.`, le corps ne contient ni `sql`, ni `database`, ni `closed`, et `env.Logs` contient une ligne de niveau `WARN` au message `vérification de santé en échec` ; `TestHealthLogLevel` — après un appel réussi, aucune ligne de niveau `INFO` dont `path` vaut `/api/health` (décoder chaque ligne JSON de `env.Logs.String()`) mais une ligne `DEBUG` `requête traitée` ; `TestHealthRejectsWrite` — `POST /api/health` → statut ≥ 400
- [X] T012 [P] [US2] Créer `internal/cli/healthcheck_test.go` ([contracts/cli.md › frigg healthcheck](./contracts/cli.md#frigg-healthcheck)) : tests en table de `healthURL(listenAddr string) (string, error)` — `:8080` → `http://127.0.0.1:8080/api/health`, `0.0.0.0:9000` → `http://127.0.0.1:9000/api/health`, `[::]:8080` → `http://127.0.0.1:8080/api/health`, `127.0.0.1:8080` inchangé, `[::1]:8080` → `http://[::1]:8080/api/health`, `sans-port` → erreur ; tests de `Healthcheck(ctx, url string, stderr io.Writer) error` contre un `httptest.Server` : `200` → `nil` et `stderr` vide, `503` → erreur `instance en mauvaise santé : statut HTTP 503`, serveur fermé → erreur commençant par `instance injoignable : `
- [X] T013 [P] [US2] Créer `internal/cli/datadir_test.go` ([research.md › R7](./research.md#r7-vérification-décriture-au-démarrage)) : `ensureWritableDir` crée un répertoire absent en `0750` et n'y laisse aucun fichier ; sur un répertoire en `0500` (test ignoré avec `t.Skip` si `os.Geteuid() == 0`), renvoie une erreur contenant `n'est pas accessible en écriture par l'utilisateur` et `UID 65532`

### Implementation for User Story 2

- [X] T014 [P] [US2] Créer `internal/database/queries/health.sql` avec un commentaire ASCII et `-- name: HealthCheck :one` / `SELECT 1;`, puis lancer `go tool task gen` pour régénérer `internal/database/sqlc/`
- [X] T015 [US2] Créer `internal/api/health.go` ([contracts/api.md](./contracts/api.md), [research.md › R8](./research.md#r8-point-de-santé)) : `const tagHealth = "Santé"` ; `const HealthPath = "/api/health"` (exportée pour le middleware) ; type `HealthOutput` avec `CacheControl string \`header:"Cache-Control"\`` et `Body struct { Status string \`json:"status" enum:"ok" doc:"État de l'instance"\` }` ; méthode `registerHealth(api huma.API)` qui appelle `register(api, auth.Public, huma.Operation{OperationID: "get-health", Method: http.MethodGet, Path: "/health", Summary: "État de santé de l'instance", Description: …, Tags: []string{tagHealth}, Errors: []int{503}}, s.getHealth)` ; `getHealth` exécute `s.Store.Read.HealthCheck` avec `context.WithTimeout(ctx, 2*time.Second)` ; en cas d'erreur, `s.Logger.WarnContext(ctx, "vérification de santé en échec", "error", err)` puis `huma.Error503ServiceUnavailable("Service indisponible.")` ; sinon `CacheControl: "no-store"`, `Status: "ok"` ; ajouter `s.registerHealth(api)` dans `New` de `internal/api/api.go`
- [X] T016 [US2] Modifier `requestLogger` dans `internal/server/middleware.go` ([research.md › R10](./research.md#r10-journalisation-du-point-de-santé)) : niveau `slog.LevelDebug` si `r.URL.Path == api.HealthPath && rec.status < 400`, `slog.LevelInfo` sinon, via `logger.Log(r.Context(), level, "requête traitée", attrs...)` ; commentaire en français justifiant (une vérification toutes les 30 s)
- [X] T017 [US2] Régénérer le contrat : `go tool task gen` puis vérifier que `web/app/api/openapi.json` et les types générés contiennent `get-health` et rien d'autre de nouveau ; `go test ./internal/api/...` (T010, T011) doit passer
- [X] T018 [US2] Créer `internal/cli/healthcheck.go` ([contracts/cli.md › frigg healthcheck](./contracts/cli.md#frigg-healthcheck)) : `healthURL(listenAddr)` (via `net.SplitHostPort` ; hôte vide, `0.0.0.0` ou `::` → `127.0.0.1` ; `net.JoinHostPort` pour l'IPv6 ; erreur `FRIGG_LISTEN_ADDR invalide : …`) ; `Healthcheck(ctx, url, stderr)` avec un `http.Client{Timeout: 3 * time.Second}`, erreur `instance en mauvaise santé : statut HTTP %d` sur tout statut ≠ 200 et `instance injoignable : %w` sur erreur réseau ; `HealthcheckCommand(ctx)` qui fait `config.Load()` (sans `RequireServe`), appelle `healthURL(cfg.ListenAddr)` puis `Healthcheck` ; rien n'est écrit en cas de succès
- [X] T019 [US2] Modifier `cmd/frigg/main.go` : ajouter `case "healthcheck": err = cli.HealthcheckCommand(ctx)` et la ligne `  healthcheck   vérifie l'état de santé de l'instance locale (code 0 si saine)` dans `usage` ; `go test ./internal/cli/...` (T012) doit passer
- [X] T020 [US2] Créer `internal/cli/datadir.go` : `ensureWritableDir(dir string) error` — `os.MkdirAll(dir, 0o750)`, puis `os.CreateTemp(dir, ".frigg-write-check-*")`, fermeture et `os.Remove` ; en cas d'échec, `fmt.Errorf("le répertoire %s n'est pas accessible en écriture par l'utilisateur %d : %w (en conteneur, le volume doit appartenir à l'UID 65532)", dir, os.Getuid(), err)` ; puis dans `Serve` de `internal/cli/serve.go`, juste après `RequireServe` et avant `database.Open`, appeler `ensureWritableDir(filepath.Dir(cfg.DatabasePath))` puis `ensureWritableDir(cfg.DataDir)` ; `go test ./internal/cli/...` (T013) doit passer
- [X] T021 [US2] Ajouter au `Dockerfile` (étape finale, après `EXPOSE`) : `HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --start-interval=2s --retries=3 CMD ["/frigg", "healthcheck"]` avec un commentaire français (pas de shell ni de curl dans l'image ; « sain » en moins de 30 s, « en mauvaise santé » en moins de 2 min)
- [X] T022 [US2] Ajouter à `README.md` une section « Installer avec Docker » ([contracts/image.md](./contracts/image.md), [research.md › R15](./research.md#r15-documentation)) : exemple `docker run` et exemple `compose.yaml` de production (dans le README uniquement), variables obligatoires (`FRIGG_BASE_URL`, `FRIGG_SMTP_HOST`, `FRIGG_MAIL_FROM`) et renvoi à `.env.example`, valeurs par défaut de l'image, volume `/data` et propriétaire `65532` (`chown -R 65532:65532` pour un répertoire de l'hôte ; avertissement : sans volume nommé, les données disparaissent avec le conteneur), `/data/.env` facultatif, premier administrateur `docker exec -it frigg /frigg create-admin`, sauvegarde `docker exec frigg /frigg backup backups/frigg-AAAAMMJJ.db` (créer `backups/` d'abord) puis `docker cp`, restauration (arrêter le conteneur, remplacer `frigg.db` et supprimer `frigg.db-wal`/`frigg.db-shm`, redémarrer), mise à jour (`docker pull`, recréer le conteneur sur le même volume, migrations automatiques ; sauvegarde préalable recommandée), arrêt propre (Frigg laisse 10 s aux requêtes en cours, autant que le délai par défaut de `docker stop` : recommander `--stop-timeout 20` et `stop_grace_period: 20s` dans les deux exemples, les 30 s par défaut de Kubernetes suffisent), reverse proxy TLS et `FRIGG_TRUSTED_PROXIES`, point de santé `/api/health` et sonde Kubernetes `httpGet` équivalente au `HEALTHCHECK`
- [X] T023 [US2] Reconstruire `frigg:dev` et dérouler [quickstart.md › 4 (sans les options de durcissement), 5 et 6](./quickstart.md#4-exploitation-durcie-us2-us3) : `healthy` en moins de 30 s, `/data/files` créé et accessible en écriture, `create-admin`, connexion, e-mail Mailpit, sauvegarde, arrêt propre avec `--stop-timeout 20` (code de sortie 0, jamais 137), erreurs de configuration et de volume, mise à jour sans perte

**Checkpoint**: l'image s'exploite de bout en bout et signale sa santé.

---

## Phase 5: User Story 3 - Image sécurisée et légère (Priority: P1)

**Goal**: image sans root ni shell, fonctionnelle en lecture seule sans capacités, analysée avant publication, aux dépendances épinglées et tenues à jour, sous 50 Mo.

**Independent Test**: [quickstart.md › 3 et 4 avec durcissement](./quickstart.md#3-sécurité-et-taille-us3).

### Tests for User Story 3 ⚠️

- [X] T024 [P] [US3] Ajouter à `internal/database/db_test.go` un test vérifiant que `PRAGMA temp_store` renvoie `2` (MEMORY) sur les pools d'écriture et de lecture d'une base ouverte par `Open`

### Implementation for User Story 3

- [X] T025 [US3] Ajouter `q.Add("_pragma", "temp_store(MEMORY)")` dans `dsn` de `internal/database/db.go`, avec un commentaire français ([research.md › R6](./research.md#r6-système-de-fichiers-en-lecture-seule--fichiers-temporaires-de-sqlite)) : les fichiers temporaires de SQLite restent en mémoire pour que l'instance fonctionne avec un système de fichiers racine en lecture seule ; `go test ./internal/database/...` (T024) doit passer
- [X] T026 [US3] Insérer dans le job `image` de `.github/workflows/release.yml`, avant l'étape `push`, l'analyse bloquante ([research.md › R11](./research.md#r11-publication--workflow-releaseyml)) : `docker/build-push-action` id `scan-build` (`platforms: linux/amd64`, `load: true`, `push: false`, `tags: frigg:scan`, `provenance: false`, `sbom: false`, `cache-from: type=gha`) puis `aquasecurity/trivy-action` épinglée par SHA (`image-ref: frigg:scan`, `severity: CRITICAL,HIGH`, `ignore-unfixed: true`, `exit-code: '1'`, `format: table`) ; commentaire français : un échec arrête le job avant toute publication
- [X] T027 [P] [US3] Créer `.github/dependabot.yml` ([research.md › R3](./research.md#r3-épinglage-et-mise-à-jour-des-dépendances-de-construction)) : `version: 2`, écosystèmes `docker` (répertoire `/`) et `github-actions` (répertoire `/`), `schedule: interval: weekly`, `commit-message: prefix: "⬆️"`, commentaire de tête en français ; ne pas inclure `gomod` ni `bun` (hors périmètre)
- [X] T028 [P] [US3] Ajouter à la section « Installer avec Docker » de `README.md` une sous-section « Durcissement » : `--read-only`, `--cap-drop=ALL`, `--security-opt=no-new-privileges` (et leurs équivalents `read_only`, `cap_drop`, `security_opt` en Compose), `runAsNonRoot`/`readOnlyRootFilesystem`/`allowPrivilegeEscalation: false` en Kubernetes, et la vérification de provenance `gh attestation verify oci://ghcr.io/balsakup/frigg:<version> --owner Balsakup`
- [X] T029 [US3] Valider [quickstart.md › 3](./quickstart.md#3-sécurité-et-taille-us3) et [› 4 avec durcissement](./quickstart.md#4-exploitation-durcie-us2-us3) sur `frigg:dev` reconstruite : utilisateur `65532:65532`, pas de shell, taille compressée de l'image ≤ taille compressée du binaire `/frigg` + 5 Mo et < 50 Mo (SC-003, mesures de [quickstart.md › 3](./quickstart.md#3-sécurité-et-taille-us3) ; noter les deux valeurs mesurées dans [research.md › R1](./research.md#r1-image-dexécution--distroless-static-utilisateur-nonroot)), `docker history` sans `.env` ni source, Trivy local sans `CRITICAL`/`HIGH` corrigeable, parcours complet (démarrage avec contrôle d'écriture de `/data/files`, connexion, e-mail, sauvegarde) avec `--read-only --cap-drop=ALL --security-opt=no-new-privileges`

**Checkpoint**: l'image respecte FR-012 à FR-018 et SC-003 à SC-005.

---

## Phase 6: User Story 4 - Construire l'image en local pour déboguer (Priority: P2)

**Goal**: une commande Taskfile produit `frigg:dev` avec la recette de la release, sans rien publier.

**Independent Test**: [quickstart.md › 2](./quickstart.md#2-construction-locale-us4).

### Implementation for User Story 4

- [X] T030 [US4] Ajouter la tâche `docker:build` à `Taskfile.yml` ([research.md › R14](./research.md#r14-construction-locale--tâche-dockerbuild)) : `desc: Construit l'image Docker locale frigg:dev (même recette que la release, sans publication)` ; `preconditions: - sh: command -v docker` avec `msg: Docker est requis pour construire l'image locale.` ; `vars:` `REVISION` (`sh: git rev-parse HEAD 2>/dev/null || echo inconnue`), `SHORT` (`sh: git rev-parse --short HEAD 2>/dev/null || echo inconnue`), `DIRTY` (`sh: git diff --quiet HEAD 2>/dev/null && echo "" || echo "-modifie"`), `CREATED` (`sh: date -u +%Y-%m-%dT%H:%M:%SZ`) ; `cmds: - docker build --build-arg VERSION=dev-{{.SHORT}}{{.DIRTY}} --build-arg REVISION={{.REVISION}} --build-arg CREATED={{.CREATED}} -t frigg:dev .` ; ni `--push` ni connexion au registre
- [X] T031 [P] [US4] Ajouter à `CONTRIBUTING.md` une section « Construire l'image en local » : `go tool task docker:build`, image `frigg:dev` jamais publiée, commande `docker run` minimale avec Mailpit (`docker compose up -d`) et renvoi à la section Docker du README, astuce de débogage sans shell (`docker cp`, `docker logs`, `docker inspect`)
- [X] T032 [US4] Valider [quickstart.md › 2](./quickstart.md#2-construction-locale-us4) : `go tool task --list` affiche `docker:build` et sa description ; l'étiquette `org.opencontainers.image.version` vaut `dev-<sha>` puis `dev-<sha>-modifie` après une modification non commitée ; avec un `PATH` sans Docker, le message de précondition s'affiche

**Checkpoint**: toutes les user stories sont fonctionnelles indépendamment.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T033 [P] Mettre à jour `specs/001-auth-app-shell/contracts/cli.md` : renvoi vers [contracts/cli.md](./contracts/cli.md) pour `frigg healthcheck` et le contrôle d'écriture au démarrage
- [X] T034 [P] Mettre à jour le statut de `specs/007-docker-release/spec.md` (`**Status**: Implemented`) ; vérifier que la limite de 50 Mo de SC-003 reste cohérente avec la taille mesurée en T029
- [X] T035 Lancer `go tool task check` (build `CGO_ENABLED=0`, vet, tests, `gen` sans diff, typecheck, Biome, Vitest, e2e) et corriger toute régression
- [ ] T036 Dérouler [quickstart.md › 7](./quickstart.md#7-release-us1) avec une préversion réelle `v0.1.0-rc.1` après fusion sur `main` (action du mainteneur : rendre le paquet public au premier passage), puis supprimer la préversion de test si elle n'a plus d'usage

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance.
- **Foundational (Phase 2)** : dépend de T001 et T002 ; bloque toutes les stories.
- **US1 (Phase 3)** : dépend de la phase 2.
- **US2 (Phase 4)** : dépend de la phase 2 ; indépendante de US1.
- **US3 (Phase 5)** : T026 dépend de T006 (job `image` de US1) ; T028 dépend de T022 (section README de US2) ; T029 dépend de T021 et T025.
- **US4 (Phase 6)** : dépend de la phase 2 ; indépendante des autres.
- **Polish (Phase 7)** : après toutes les stories ; T036 après fusion.

### Within Each User Story

- US2 : T010 à T013 (tests) avant T014 à T020 ; T014 avant T015 ; T015 avant T016 et T017 ; T018 avant T019 ; T021 après T019 ; T023 en dernier.
- US1 : T005 → T006 → T007 (même fichier) ; T009 après T007.
- US3 : T024 avant T025.

### Parallel Opportunities

- T002 en parallèle de T001.
- Après la phase 2, US1, US2 et US4 peuvent avancer en parallèle (fichiers distincts : `release.yml`, `internal/…`, `Taskfile.yml`).
- Dans US2 : T010, T011, T012, T013 et T014 en parallèle.
- T008 (CONTRIBUTING) en parallèle de T005 à T007 ; T031 en parallèle de T030, mais après T008 si les deux modifient `CONTRIBUTING.md` en même temps (même fichier : éviter de les lancer ensemble).
- T027 en parallèle de toute tâche d'US3.

---

## Parallel Example: User Story 2

```bash
# Tests d'abord, en parallèle :
Task: "T010 allowedPublic dans internal/api/access_test.go"
Task: "T011 tests du point de santé dans internal/api/health_test.go"
Task: "T012 tests de healthcheck dans internal/cli/healthcheck_test.go"
Task: "T013 tests de ensureWritableDir dans internal/cli/datadir_test.go"
Task: "T014 requête HealthCheck dans internal/database/queries/health.sql"
```

## Parallel Example: après la phase 2

```bash
Task: "T005-T007 workflow release.yml (US1)"
Task: "T010-T020 point de santé, healthcheck, contrôle d'écriture (US2)"
Task: "T030 tâche docker:build dans Taskfile.yml (US4)"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 et phase 2 : image construisible.
2. Phase 3 (US1) : publication à la release.
3. **STOP and VALIDATE** : préversion de test, image téléchargeable.

L'image publiée à ce stade fonctionne mais n'a ni vérification de santé ni
analyse de vulnérabilités : ne pas annoncer de version stable avant US2 et US3.

### Incremental Delivery

1. Setup + Foundational → image locale.
2. US1 → publication automatique (MVP technique).
3. US2 → exploitation complète, santé, documentation : première version
   stable possible après US3.
4. US3 → durcissement et analyse bloquante.
5. US4 → confort de développement.

### Suggested commits (gitmoji, titre court)

- `🙈 Ajoute le .dockerignore en liste blanche` (T001)
- `🐳 Ajoute l'image Docker de production` (T002 à T004)
- `👷 Publie l'image à chaque release` (T005 à T009)
- `🩺 Ajoute le point de santé et frigg healthcheck` (T010 à T021)
- `📝 Documente l'installation avec Docker` (T022, T028)
- `🔒 Durcit l'image et analyse ses vulnérabilités` (T024 à T027)
- `🔨 Ajoute la construction de l'image locale` (T030, T031)
