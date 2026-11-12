# Contrat : workflow `docs.yml`

## Déclencheurs

| Événement | Filtre | Effet |
| --- | --- | --- |
| `pull_request` | chemins `docs/**`, `.github/workflows/docs.yml` | Construit la version de la PR (liens morts bloquants) ; **pas de déploiement** |
| `push` | branche `main`, mêmes chemins | Construit les deux versions et déploie |
| `release` | `types: [published]` (préversions comprises) | Construit les deux versions et déploie |
| `workflow_dispatch` | | Construit les deux versions et déploie (relance manuelle) |

`permissions: {}` au niveau du workflow ; chaque job déclare les siennes.

Concurrence :

- déploiements : `group: pages`, `cancel-in-progress: false` ;
- PR : `group: docs-pr-<numéro>`, `cancel-in-progress: true`.

## Job `build` : « Construit la documentation »

Permissions : `contents: read`.

1. Extrait le dépôt dans `src`, avec `persist-credentials: false` et
   `fetch-depth: 0`. `ref` vaut `main` pour les événements `release` et
   `workflow_dispatch`, sinon la référence de l'événement.
2. Installe Bun (`BUN_VERSION`, aligné sur `ci.yml`).
3. **Hors PR**, détermine la dernière release. Cette étape est la seule à
   recevoir `GH_TOKEN`.
   - Commande de [research.md › R6](../research.md#r6-choix-de-la-dernière-release-dans-le-workflow-sans-script).
   - Si l'étiquette ne contient pas `docs/.vitepress/config.mts`
     (`git cat-file -e`), une annotation `::notice::` le signale et la
     release est ignorée.
   - Sortie : `tag`, ou vide.
4. Construit « main » (ou la PR) selon [build-env.md](./build-env.md) :
   `DOCS_BASE=/frigg/main/`, `DOCS_VERSION=main`,
   `DOCS_RELEASE_VERSION=<tag>`.
5. Hors PR, si `tag` n'est pas vide : `git worktree add` de l'étiquette dans
   `release`, puis construction avec `DOCS_BASE=/frigg/`,
   `DOCS_VERSION=<tag>`, `DOCS_RELEASE_VERSION=<tag>`.
6. Hors PR, assemble `site/` :
   - avec release : sortie « release » à la racine, sortie « main » dans
     `site/main/` ; échec si la sortie « release » contient déjà `main/` ;
   - sans release : sortie « main » dans `site/main/`, plus
     `site/index.html` (page de redirection en français vers
     `/frigg/main/`, `meta refresh` et lien visible) et `site/404.html`
     (copie de celui de « main »).
7. Écrit dans `$GITHUB_STEP_SUMMARY` les versions publiées.
8. Hors PR : `actions/configure-pages`, puis `actions/upload-pages-artifact`
   (`path: site`).

## Job `deploy` : « Publie sur GitHub Pages »

- `needs: build`, seulement hors PR.
- Permissions : `pages: write`, `id-token: write`.
- Environnement : `github-pages`, URL = sortie de `actions/deploy-pages`.

## Garanties

- Un échec à n'importe quelle étape de `build` empêche `deploy` : le site
  précédent reste en ligne et l'exécution apparaît en échec dans Actions.
- Aucune étape de construction ne reçoit de jeton.
- Actions tierces épinglées par SHA, avec la version en commentaire.
- Durée cible : moins de 10 minutes (SC-001, SC-002).

## Réglage du dépôt (une fois)

- Settings → Pages → Source : **GitHub Actions**.
- Dans les règles de l'environnement `github-pages`, n'autoriser les
  déploiements que depuis `main` et depuis les étiquettes `v*`.
