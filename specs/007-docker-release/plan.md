# Implementation Plan: Image de production et publication à chaque release

**Branch**: `007-docker-release` | **Date**: 2026-11-02 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/007-docker-release/spec.md`

## Summary

Livrer Frigg sous forme d'image de conteneur `ghcr.io/balsakup/frigg`,
publiée automatiquement à chaque release GitHub. Un `Dockerfile` multi-étapes
construit le SPA (Bun) et le binaire `CGO_ENABLED=0` (compilé pour
`amd64` et `arm64` sans émulation) et les dépose dans une image distroless
`static:nonroot` : pas de shell, utilisateur 65532, compatible avec un
système de fichiers en lecture seule (fichiers temporaires SQLite en mémoire).
Un workflow `release.yml` vérifie l'étiquette, rejoue compilation et tests,
analyse l'image (Trivy), la publie avec provenance et SBOM, ne déplace
`latest` que vers la plus haute version stable, puis annote la release. Le
binaire gagne un point de santé public `GET /api/health` (amendement 1.5.0 du
principe II), une commande `frigg healthcheck` utilisée par le `HEALTHCHECK`
de l'image et un contrôle d'écriture du répertoire de données au démarrage.
`go tool task docker:build` construit la même image en local (`frigg:dev`).
Approche détaillée dans [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 ; Bun 1.3 pour le SPA ; Dockerfile
(BuildKit, syntaxe `docker/dockerfile:1`) ; GitHub Actions (YAML, bash)

**Primary Dependencies**: aucune nouvelle dépendance Go ni frontend. Images
de construction : `oven/bun`, `golang:1.27-alpine` ; image d'exécution
`gcr.io/distroless/static-debian13:nonroot` ; actions `actions/checkout`,
`actions/setup-go`, `oven-sh/setup-bun`, `docker/setup-buildx-action`,
`docker/login-action`, `docker/metadata-action`, `docker/build-push-action`,
`aquasecurity/trivy-action`, `actions/attest-build-provenance` ; toutes
épinglées par empreinte ou SHA ([R3](./research.md#r3-épinglage-et-mise-à-jour-des-dépendances-de-construction))

**Storage**: SQLite inchangé (aucune migration) ; pragma
`temp_store(MEMORY)` ajouté à la chaîne de connexion ; requête sqlc
`HealthCheck` ; volume `/data` dans l'image ([data-model.md](./data-model.md))

**Testing**: `go test` : opération `get-health` (200, 503 base fermée,
journaux `debug`/`warn`, méthode d'écriture refusée) et matrice d'autorisation mise à
jour ; `frigg healthcheck` contre un `httptest.Server` (sain, 503,
injoignable, résolution de `FRIGG_LISTEN_ADDR`) ; contrôle d'écriture
(répertoire en `0500`, ignoré en root) ; validation manuelle de l'image et de
la release selon [quickstart.md](./quickstart.md)

**Target Platform**: conteneurs Linux `amd64` et `arm64` (Docker 25+,
Podman, Kubernetes) ; runners GitHub `ubuntu-latest`

**Project Type**: application web (API Go + SPA intégrée au binaire) et
chaîne de livraison

**Performance Goals**: publication < 20 min (SC-001) ; `healthy` < 30 s
après démarrage, `unhealthy` < 2 min (SC-009) ; image locale < 5 min après
une première construction (SC-007, caches BuildKit)

**Constraints**: image compressée ≤ binaire compressé + 5 Mo et < 50 Mo (SC-003) ; aucune vulnérabilité
`CRITICAL`/`HIGH` corrigeable ; aucun secret dans l'image ni les journaux ;
`GITHUB_TOKEN` seul, permissions minimales par job ; point de santé sans
aucune donnée ; textes en français, identifiants en anglais

**Scale/Scope**: 1 opération d'API, 1 sous-commande, 1 requête sqlc,
1 middleware ajusté, 1 pragma ; `Dockerfile`, `.dockerignore`,
2 fichiers GitHub (`release.yml`, `dependabot.yml`), 1 tâche Taskfile ;
2 documents (README, CONTRIBUTING)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Une image = une instance ; un volume `/data` par instance ; aucune notion de locataire | ✅ |
| II. Authentification partout | Seule nouvelle route publique : `get-health`, autorisée par l'amendement 1.5.0 ; ajoutée à `allowedPublic` (liste centralisée testée) ; réponse sans donnée, version, configuration ni détail d'erreur ; tout le reste inchangé | ✅ |
| III. Deux rôles | `get-health` déclarée par `register` avec `auth.Public` ; aucune vérification de rôle ad hoc | ✅ |
| IV. Un shell, un design system | Aucun changement d'interface ; l'image ne charge aucune ressource externe (SPA intégré) | ✅ |
| V. Binaire unique en Go pur | L'image ne contient que le binaire `CGO_ENABLED=0` ; SPA et migrations restent intégrés ; dépendances d'exécution inchangées (SQLite, répertoire de données, `.env` facultatif, SMTP) ; `create-admin` et `backup` disponibles dans le conteneur ; aucune dépendance Go ajoutée | ✅ |
| VI. API typée et SQL généré | Opération huma avec sortie typée ; `HealthCheck` écrite dans `internal/database/queries` et générée par sqlc ; OpenAPI et types TypeScript régénérés et commités | ✅ |
| VII. Sécurité et RGPD | Utilisateur non root, pas de shell, lecture seule, capacités retirées ; analyse de vulnérabilités bloquante ; contexte en liste blanche (pas de `.env`) ; journaux de santé sans secret ; sauvegardes toujours en `0600` | ✅ |
| Contraintes techniques | `slog` JSON inchangé (niveau `debug` pour la santé réussie) ; configuration par variables d'environnement, valeurs par défaut de l'image modifiables ; pragmas existants conservés, `temp_store(MEMORY)` ajouté ([R6](./research.md#r6-système-de-fichiers-en-lecture-seule--fichiers-temporaires-de-sqlite)) | ✅ |
| Workflow et qualité | La release rejoue `go build` (`CGO_ENABLED=0`), `go vet`, `go test`, `typecheck`, Biome, Vitest et `bun run build` ; les tests de bout en bout restent exigés avant fusion, pas à la release (hypothèse de la spec) | ✅ |

**Re-check après Phase 1** : la conception (contrats, modèle de données) ne
modifie aucun statut. Aucune entrée en « Complexity Tracking ».

## Project Structure

### Documentation (this feature)

```text
specs/007-docker-release/
├── plan.md              # Ce fichier
├── research.md          # Phase 0 : décisions R1 à R15
├── data-model.md        # Phase 1 : requête, réponse, release, image, volume
├── quickstart.md        # Phase 1 : scénarios de validation
├── contracts/
│   ├── api.md           # GET /api/health
│   ├── cli.md           # frigg healthcheck, contrôle d'écriture
│   ├── image.md         # interface de l'image pour les exploitants
│   └── release.md       # déclencheur, étiquettes, jobs
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 (/speckit-tasks)
```

### Source Code (repository root)

```text
Dockerfile                              # nouveau : web → build → distroless
.dockerignore                           # nouveau : liste blanche
Taskfile.yml                            # + docker:build
.github/
├── dependabot.yml                      # nouveau : docker + github-actions
└── workflows/
    └── release.yml                     # nouveau : verify → image → announce
cmd/frigg/main.go                       # + healthcheck, aide
internal/
├── api/
│   ├── health.go                       # nouveau : get-health
│   ├── health_test.go                  # nouveau
│   └── access_test.go                  # allowedPublic + get-health
├── cli/
│   ├── healthcheck.go                  # nouveau : frigg healthcheck
│   ├── healthcheck_test.go             # nouveau
│   ├── serve.go                        # contrôle d'écriture avant ouverture
│   ├── datadir.go                      # nouveau : ensureWritableDir
│   └── datadir_test.go                 # nouveau
├── server/middleware.go                # niveau debug pour /api/health réussi
└── database/
    ├── db.go                           # + temp_store(MEMORY)
    ├── queries/health.sql              # nouveau : HealthCheck
    └── sqlc/                           # régénéré
web/app/api/                            # openapi.json et types régénérés
README.md                               # + Installer avec Docker
CONTRIBUTING.md                         # + Publier une release, image locale
```

**Structure Decision**: structure existante conservée (API Go dans
`internal/`, SPA dans `web/`). Les fichiers de livraison (`Dockerfile`,
`.dockerignore`) sont à la racine, contexte de construction = dépôt ; le
workflow rejoint `cla.yml` dans `.github/workflows/`.

## Complexity Tracking

Aucune violation de la constitution à justifier.
