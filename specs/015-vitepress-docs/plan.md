# Implementation Plan: Site de documentation versionné

**Branch**: `015-vitepress-docs` | **Date**: 2026-11-12 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/015-vitepress-docs/spec.md`

## Summary

Publier la documentation de Frigg sur GitHub Pages
(`https://balsakup.github.io/frigg/`), en deux versions, de la façon la plus
simple possible.

- **Sources** :
  - le dossier `docs/` ne contient que des pages Markdown (trois guides en
    français), une configuration VitePress (`.vitepress/config.mts`),
    `package.json` et `bun.lock` ;
  - thème par défaut, sans composant, style ni test ;
  - la configuration lit quatre variables d'environnement (`base`, libellé de
    version, release voisine, URL du site) pour produire le sélecteur de
    version (un menu `nav`).
- **Workflow `docs.yml`** :
  - il détermine la dernière release par une ligne de shell (`sort -V`,
    préversions comprises) ;
  - il construit `main` (sous `/frigg/main/`) et cette étiquette (à la
    racine) ;
  - il assemble les deux sorties, ou ajoute une redirection racine sans
    release, et déploie de façon atomique ;
  - sur une PR, il construit seulement : les liens morts sont bloquants.
- **Fichiers existants** : README et `CONTRIBUTING.md` sont allégés et
  renvoient au site.

Approche détaillée dans [research.md](./research.md).

## Technical Context

**Language/Version**: Markdown ; configuration TypeScript (`config.mts`) exécutée par Bun 1.4.4 (`bun --bun vitepress`) ; GitHub Actions (YAML, bash)

**Primary Dependencies** :

- `vitepress@2.0.0-alpha.21` et `vue@3.5.45`, épinglés exactement, seules
  dépendances de `docs/` ([R1](./research.md#r1-version-de-vitepress)) ;
- nouvelles actions : `actions/configure-pages`,
  `actions/upload-pages-artifact` et `actions/deploy-pages`, épinglées par
  SHA ;
- aucune nouvelle dépendance Go, aucune dans `web/`.

**Storage**: N/A (site statique)

**Testing** :

- aucune suite de tests propre à la documentation (clarification du
  2026-11-12) ;
- la construction VitePress fait office de contrôle : liens morts et pages
  invalides bloquants, en local et sur chaque PR ;
- validation manuelle selon [quickstart.md](./quickstart.md).

**Target Platform**: GitHub Pages ; runners `ubuntu-latest`

**Project Type**: site de documentation statique + chaîne de publication, en marge de l'application web

**Performance Goals**: publication en moins de 10 min après l'événement (SC-001, SC-002) ; une construction prend environ 5 s

**Constraints** :

- `docs/` limité au Markdown, à la configuration, au manifeste et au verrou
  (FR-005, SC-006) ;
- aucun secret dans les étapes de construction ; permissions minimales ;
  déploiement atomique ;
- contrat de construction rétrocompatible
  ([build-env.md](./contracts/build-env.md)) ;
- textes en français, noms de fichiers et variables en anglais.

**Scale/Scope** :

- environ 35 pages Markdown, 1 fichier de configuration ;
- 1 workflow, 1 bloc Dependabot ;
- 2 tâches Taskfile (`docs:dev`, `docs:build`) ;
- README, `CONTRIBUTING.md` et modèle de PR mis à jour.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Le site est indépendant de toute instance et ne contient aucune donnée d'association (FR-019) | ✅ |
| II. Authentification partout | Aucune route ajoutée au binaire ; le site public est hébergé hors de toute instance | ✅ |
| III. Deux rôles | Aucune opération d'API ; le guide documente le rôle requis, sans rien changer aux autorisations | ✅ |
| IV. Un shell, un design system | Ce principe régit l'application servie aux utilisateurs, qui n'est pas modifiée. Le site de documentation n'est pas l'application : il utilise le thème par défaut de VitePress, sans reprendre le design system et sans vérification RGAA (clarification du 2026-11-12, hypothèses de la spec) | ✅ |
| V. Binaire unique en Go pur | Rien n'est ajouté au binaire ni à `web/embed.go` ; `.dockerignore` est en liste blanche, donc `docs/` est exclu de l'image | ✅ |
| VI. API typée et SQL généré | Non concerné | ✅ |
| VII. Sécurité et RGPD | Aucune donnée personnelle ; workflow avec `permissions: {}` par défaut, jeton limité à une étape, actions épinglées par SHA, déploiement OIDC limité à l'environnement `github-pages` | ✅ |
| Contraintes techniques (listes fermées) | Elles visent le backend et `web/`. `docs/` ajoute deux dépendances de construction, non livrées : voir Complexity Tracking | ⚠️ justifié |
| Conventions de langue | Contenu et messages du workflow en français ; noms de fichiers, variables `DOCS_*` et jobs en anglais | ✅ |
| Workflow et contrôles qualité | `ci.yml` et `go tool task check` inchangés ; la documentation a son contrôle dans `docs.yml` | ✅ |

**Après la Phase 1** : la conception ne change rien et aucun amendement
n'est nécessaire. Gate franchie.

## Project Structure

### Documentation (this feature)

```text
specs/015-vitepress-docs/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── build-env.md     # Variables d'une construction (contrat rétrocompatible)
│   ├── site.md          # Adresses, arborescence, éléments de page
│   └── workflow.md      # docs.yml : déclencheurs, jobs, garanties
├── checklists/requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
docs/
├── package.json             # vitepress, vue ; scripts dev (port 5174), build, preview
├── bun.lock
├── .vitepress/config.mts    # base, sélecteur de version, nav, sidebar, recherche, libellés FR
├── index.md                 # accueil (layout: home)
├── installation/*.md
├── usage/*.md
└── contributing/*.md

.github/workflows/docs.yml   # nouveau
.github/dependabot.yml       # + bloc bun /docs
.github/pull_request_template.md  # + case « documentation à jour »
Taskfile.yml                 # + docs:dev, docs:build
README.md                    # allégé, lien vers le site
CONTRIBUTING.md              # allégé, lien vers le site
.gitignore                   # docs/node_modules, docs/.vitepress/{dist,cache}
```

**Structure Decision** : un projet `docs/` minimal, à côté de `web/` et du
code Go ([R3](./research.md#r3-contenu-du-dossier-docs)). Toute la logique de
version et d'assemblage est dans `docs.yml`.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
| --- | --- | --- |
| Projet `docs/` avec deux dépendances hors des listes fermées (VitePress, Vue) | VitePress est imposé par la demande ; ces dépendances ne servent qu'à construire le site et ne sont ni dans le binaire ni chargées par l'application | Les ajouter dans `web/` mélangerait deux configurations Vite et ferait entrer Vue dans un frontend React à la liste fermée |
| VitePress en version alpha (2.0.0-alpha.21) | La 1.6.4 stable traîne 9 vulnérabilités connues via Vite 5 et esbuild ; la 2.0 n'en a aucune | Rester en 1.6.4 imposerait des alertes permanentes ; le risque de l'alpha est contenu par l'épinglage exact et l'usage de la seule configuration |
| Workflow séparé `docs.yml` | Les jobs de `ci.yml` sont des checks requis ; un job filtré par chemins y resterait en attente sur les PR sans documentation | — |
