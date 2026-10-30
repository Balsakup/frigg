# Quickstart : valider l'authentification et le shell

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[cli.md](./contracts/cli.md), [ui-routes.md](./contracts/ui-routes.md).

## Prérequis

- Go 1.27, Bun, `sqlc` (v1.31+), Docker (ou Podman) avec Compose.
- go-task, via `go tool task` (déclaré dans `go.mod`, aucune installation).
- Navigateurs Playwright : `cd web && bunx playwright install chromium`.

## Mise en place

```sh
cp .env.example .env     # puis renseigner au minimum :
# FRIGG_BASE_URL=http://localhost:5173
# FRIGG_SMTP_HOST=localhost
# FRIGG_SMTP_PORT=1025
# FRIGG_SMTP_TLS=none
# FRIGG_MAIL_FROM=frigg@localhost

docker compose up -d            # Mailpit : SMTP :1025, interface http://localhost:8025
go tool task build              # bun build puis go build → bin/frigg
./bin/frigg create-admin        # nom de l'association, adresse, mot de passe
go tool task dev                # API sur :8080 + SPA sur :5173 (/api relayé vers :8080)
```

## Contrôles automatisés (constitution)

```sh
go tool task check
```

La tâche `check` enchaîne : `go build` (`CGO_ENABLED=0`), `go vet`, `go test`,
`sqlc generate` et `gen:api` sans diff, puis dans `web/` `typecheck`, `lint`,
`test`, `build` et `e2e`.

Résultat attendu : tout passe, aucun diff. `go test` inclut la matrice
d'autorisation (chaque opération : anonyme → 401 sauf publiques, membre → 403
sur les opérations `admin`, administrateur jamais refusé).

## Scénarios manuels

| # | Parcours (spec) | Étapes | Résultat attendu |
| --- | --- | --- | --- |
| 1 | US3 — premier administrateur | `create-admin` sur base vierge | Nom de l'association demandé ; compte créé ; relancer avec la même adresse échoue |
| 2 | US1 — connexion | Ouvrir `http://localhost:5173/admin/comptes` non connecté | Redirection vers `/connexion?suite=/admin/comptes`, puis retour sur la page après connexion |
| 3 | US1 — échec | Mauvais mot de passe, puis adresse inconnue | Même message dans les deux cas ; 6ᵉ essai rapide → message de limitation |
| 4 | US2 — espaces | Connecté en administrateur | Arrivée sur `/admin`, sélecteur visible ; bascule vers l'espace membre ; recharger → dernier espace conservé |
| 5 | US4 — invitation | `/admin/comptes` : inviter `thomas@exemple.fr` (membre) ; ouvrir le lien dans Mailpit (fenêtre privée) | Mot de passe choisi → connecté sur `/`, sans sélecteur d'espace ni rubrique d'administration ; le lien rouvert est refusé |
| 6 | US2 — refus | En membre, ouvrir `/admin` | « Accès refusé » dans le shell ; l'API répond 403 |
| 7 | US5 — mot de passe oublié | `/mot-de-passe-oublie`, suivre le lien | Nouveau mot de passe accepté ; l'ancienne session (autre navigateur) est renvoyée vers la connexion |
| 8 | US6 — confirmation | Compte non confirmé (créé en base de test) | Alerte en tête de page ; « Renvoyer le lien » ; lien suivi → alerte disparue |
| 9 | US7 — Mon compte | Changer le mot de passe avec un mot de passe actuel faux, puis juste | Erreur sur le champ ; puis succès, autres sessions fermées |
| 10 | Shell | Largeur 320 px, navigation au clavier seul | Pas de défilement horizontal ; menu en surimpression fermé par Échap ; liens d'évitement utilisables |
| 11 | SMTP indisponible | `docker compose stop mailpit`, demander une réinitialisation, `docker compose start mailpit` | Réponse immédiate ; e-mail reçu après la reprise (≤ intervalle du worker + délai de nouvelle tentative) |
| 12 | Principe II | `curl -i http://localhost:8080/api/openapi`, `/api/openapi.json` et `/api/docs` avec `FRIGG_API_DOCS` absent | 404 partout |
| 13 | Documentation de dev | `FRIGG_API_DOCS=true`, relancer ; `curl -i …/api/docs` sans session ; ouvrir `http://localhost:5173/api/docs` en membre puis en administrateur | Avertissement au démarrage ; 401 sans session, 403 en membre, documentation affichée en administrateur |
