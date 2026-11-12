# Environnement de développement

## Prérequis

- Go, dans la version indiquée par
  [`go.mod`](https://github.com/Balsakup/frigg/blob/main/go.mod) ;
- [Bun](https://bun.sh) ;
- Docker, pour Mailpit et les tests de bout en bout.

Les commandes du projet passent par [Task](https://taskfile.dev), déclaré
comme outil dans `go.mod` : `go tool task <tâche>`, sans rien installer de
plus.

## Démarrer

```sh
cp .env.example .env
docker compose up -d   # Mailpit : SMTP sur :1025, interface sur http://localhost:8025
go tool task dev       # API sur :8080, frontend sur :5173
```

Ouvrez `http://localhost:5173`. Le serveur Vite redirige `/api` vers le
serveur Go, si bien que le navigateur reste sur une seule origine, comme en
production.

Créez un premier administrateur :

```sh
go run ./cmd/frigg create-admin
```

## Commandes utiles

| Commande | Rôle |
| --- | --- |
| `go tool task dev` | API et frontend en parallèle |
| `go tool task build` | Compile le binaire unique `bin/frigg` (frontend compris) |
| `go tool task gen` | Régénère le code sqlc, le schéma OpenAPI et les types TypeScript |
| `go tool task test` | Tests Go et tests unitaires du frontend |
| `go tool task e2e` | Tests de bout en bout (prérequis : `docker compose up -d`) |
| `go tool task check` | Tous les contrôles exigés avant fusion |
| `go tool task docs:dev` | Prévisualise cette documentation |

## Architecture

- **Backend** : Go, API REST typée (huma), SQLite (sans CGO), requêtes SQL
  générées par sqlc, migrations goose appliquées au démarrage.
- **Frontend** (`web/`) : React et React Router en mode SPA, Tailwind CSS,
  types générés depuis le schéma OpenAPI. Il est compilé puis intégré au
  binaire.

Les principes d'architecture sont décrits dans
[la constitution](https://github.com/Balsakup/frigg/blob/main/.specify/memory/constitution.md).
Toute contribution doit les respecter.
