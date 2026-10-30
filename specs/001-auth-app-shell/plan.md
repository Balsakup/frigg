# Implementation Plan: Authentification et shell applicatif membre/administrateur

**Branch**: `001-auth-app-shell` | **Date**: 2026-10-30 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/001-auth-app-shell/spec.md`

## Summary

Poser le socle de Frigg : connexion par adresse et mot de passe, sessions
stockées en SQLite, deux rôles (`admin ⊇ member`) contrôlés par une fonction
d'autorisation unique, premier administrateur créé en ligne de commande,
invitations par e-mail, réinitialisation de mot de passe et confirmation
d'adresse via une file d'e-mails, et le shell applicatif commun (barre latérale
avec sélecteur d'espace pour les administrateurs, en-tête, fil d'Ariane, pied de
page) conforme à l'artefact de référence. Approche détaillée dans
[research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 5.9 + React 19 (frontend)

**Primary Dependencies**: huma/v2 v2.40 (humago), scs/v2 v2.9 + sqlite3store,
goose/v3 v3.29, modernc.org/sqlite v1.61, alexedwards/argon2id v1.0, go-mail
v0.8, godotenv v1.5, x/time v0.16, x/term ; React Router v8 (SPA), TanStack
Query, openapi-fetch, openapi-react-query, react-hook-form, base-ui,
lucide-react, Tailwind v4

**Storage**: SQLite (WAL, un pool d'écriture à une connexion, un pool de
lecture) ; aucun fichier déposé dans cette fonctionnalité

**Testing**: `go test` : httptest sur la chaîne complète (sessions, CSRF) et
humatest pour le schéma, base fichier temporaire par test ; Vitest + Testing
Library ; Playwright + axe contre le binaire réel

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` derrière un
reverse proxy TLS ; navigateurs récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: affichage de l'accueil < 30 s depuis l'ouverture, saisie
comprise (SC-001) ; réponses de l'API perçues comme immédiates hors hachage du
mot de passe (≈ 50–100 ms avec les paramètres argon2id retenus)

**Constraints**: refus par défaut (principe II) ; aucune ressource tierce
(polices locales, CSP `default-src 'self'`) ; RGAA ; mémoire contenue
(argon2id 19 Mio par hachage, borné par la limitation de débit)

**Scale/Scope**: une association, quelques dizaines à quelques centaines de
comptes ; 5 écrans publics, 5 écrans dans le shell + pages « bientôt
disponible » ; 14 opérations d'API

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Table `association` singleton (`CHECK id = 1`), aucun `association_id` | ✅ |
| II. Authentification partout | Niveau d'accès obligatoire par opération ; 6 opérations publiques exactement (les 4 parcours autorisés, dont l'invitation depuis la 1.3.1) ; routes de documentation intégrées de huma désactivées, documentation de développement derrière `FRIGG_API_DOCS` (désactivée par défaut) et réservée aux administrateurs ; SPA sans données | ✅ |
| III. Deux rôles | `member`/`admin`, rang comparé dans une seule fonction ; `create-admin` seul moyen de créer le premier administrateur | ✅ |
| IV. Un shell, un design system | Shell unique, sélecteur d'espace, tokens de l'artefact dans `@theme`, base-ui, lucide via `Icon`, polices Fontsource, thème clair, Playwright + axe | ✅ |
| V. Binaire unique en Go pur | `CGO_ENABLED=0`, SPA via `web.Dist()`, migrations goose embarquées bloquantes, `frigg backup` | ✅ |
| VI. API typée et SQL généré | huma → OpenAPI → `openapi-typescript` ; sqlc ; règle de mot de passe uniquement côté serveur, erreurs 422 par critère ; pas de recherche texte dans cette fonctionnalité (FTS5 non requis) | ✅ |
| VII. Sécurité | argon2id centralisé + hash factice ; cookies Secure/HttpOnly/Lax ; CrossOriginProtection ; limitation de débit ; jetons hachés à usage unique, transmis par fragment d'URL ; réponses neutres ; rien de sensible dans les journaux | ✅ |
| Contraintes techniques | Dépendances toutes dans la liste fermée ; config `.env` ; slog JSON ; file d'e-mails + worker périodique | ✅ |
| Conventions de langue | Identifiants, tables, champs JSON, valeurs d'énumération en anglais ; messages, commentaires, chemins visibles du SPA en français | ✅ |
| Workflow | Contrôles de [quickstart.md](./quickstart.md) ; matrice d'autorisation générée depuis l'OpenAPI | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance notés
pour l'implémentation : (1) le test de matrice doit échouer si une opération
publique apparaît hors de la liste de [api.md](./contracts/api.md) ; (2) le
script `bun run gen:api` est ajouté au contrôle « sans diff ».

## Project Structure

### Documentation (this feature)

```text
specs/001-auth-app-shell/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── api.md
│   ├── cli.md
│   └── ui-routes.md
├── checklists/
│   └── requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
cmd/frigg/
└── main.go                    # aiguillage serve | openapi | create-admin | backup

internal/
├── config/                    # .env + variables, validation au démarrage
├── database/
│   ├── db.go                  # ouverture des pools lecture/écriture, Store, WriteTx
│   ├── migrate.go             # goose embarqué
│   ├── generate.go            # //go:generate sqlc generate
│   ├── migrations/            # 00001_init.sql, 00002_auth_tokens.sql, 00003_outbound_emails.sql
│   ├── queries/               # accounts.sql, tokens.sql, invitations.sql, emails.sql, association.sql
│   └── sqlc/                  # code généré (commité)
├── auth/
│   ├── password.go            # paramètres argon2id, règle de mot de passe, hash factice
│   ├── tokens.go              # génération, empreinte SHA-256
│   ├── access.go              # Access (public/member/admin), Allows(role, access)
│   ├── session.go             # scs, clés de session, époque
│   └── ratelimit.go           # limiteurs par clé avec purge
├── mail/
│   ├── templates/             # *.txt.tmpl, *.html.tmpl (français)
│   ├── render.go              # rendu des messages
│   ├── outbox.go              # mise en file (dans une transaction)
│   └── worker.go              # dépilage périodique, backoff, purge
├── api/
│   ├── api.go                 # config huma (docs désactivées), register(), middleware d'accès
│   ├── errors.go              # problèmes en français, erreurs de champ
│   ├── auth.go                # login, logout, reset, confirm, invitation lookup/accept
│   ├── me.go                  # session, renvoi de confirmation, changement de mot de passe
│   ├── invitations.go         # opérations admin
│   ├── docs.go                # /api/openapi.json, /api/schemas, /api/docs (FRIGG_API_DOCS, admin)
│   └── *_test.go              # httptest (chaîne complète) + humatest + matrice d'autorisation
├── server/
│   ├── server.go              # mux, chaîne de middlewares, arrêt propre
│   ├── spa.go                 # service de web.Dist() avec repli sur index.html
│   └── middleware.go          # request id, journalisation, en-têtes de sécurité
└── cli/
    ├── createadmin.go
    ├── openapi.go
    └── backup.go

web/
├── embed.go                   # existant
├── app/
│   ├── api/                   # schema.d.ts (généré), client.ts, queries.ts
│   ├── lib/cn.ts              # existant
│   ├── shell/                 # Sidebar, SpaceSwitcher, Header, Breadcrumbs, PageHeader,
│   │                          # Footer, UserMenu, VerifyEmailBanner, navigation.ts
│   ├── ui/                    # Button, Alert, Badge, Field, Icon, ConfirmDialog (base-ui + tokens)
│   ├── routes/
│   │   ├── auth/              # login, forgot-password, reset-password, confirm-email, accept-invitation
│   │   └── app/               # layout, member/home, account, admin/layout, admin/home,
│   │                          # admin/accounts, coming-soon, not-found
│   ├── routes.ts
│   ├── root.tsx
│   └── app.css                # tokens de l'artefact dans @theme, thème clair
└── e2e/                       # parcours Playwright + axe

sqlc.yaml
.env.example
Taskfile.yml                   # go-task : build (bun puis go), dev, gen, check (contrôles de la constitution)
compose.yaml                   # services de développement : Mailpit
```

**Structure Decision**: application web en un seul module Go
(`github.com/balsakup/frigg`) ; le frontend vit dans `web/` et n'est consommé
par Go qu'à travers `web.Dist()`. Le package `internal/textfold` n'est pas créé
dans cette fonctionnalité (aucune recherche ni comparaison de texte libre ;
les adresses sont normalisées par minuscules et suppression des espaces).
Les commandes du projet sont décrites dans un `Taskfile.yml` (go-task),
lancé par `go tool task` : l'outil est déclaré dans `go.mod` (directive `tool`),
donc versionné avec le projet et sans conflit avec la commande `task` de
Taskwarrior. Il n'est pas compilé dans le binaire Frigg. La tâche `build`
enchaîne `bun run build` puis `go build`, car la compilation Go exige
`web/build/client`. Les services de développement (Mailpit) sont décrits dans
`compose.yaml` ; ce ne sont pas des dépendances d'exécution de Frigg
(principe V).

## Complexity Tracking

Aucune violation de la constitution à justifier.
