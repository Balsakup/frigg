# Research: Site de documentation versionné

Décisions prises pour lever les inconnues du plan. Essais faits le 2026-11-12
dans un projet jetable (Bun 1.3.14).

**Principe directeur** (clarification du 2026-11-12) :

- la documentation est la plus simple possible : `docs/` ne contient que du
  Markdown, la configuration VitePress, `package.json` et `bun.lock` ;
- le thème par défaut est utilisé tel quel ;
- toute la logique de version et d'assemblage vit dans le workflow GitHub.

## R1. Version de VitePress

- **Decision** : `vitepress@2.0.0-alpha.21` épinglé exactement (sans `^`),
  avec `vue@3.5.45` (dépendance de pair).
- **Rationale** :
  - la dernière version stable, `1.6.4` (août 2025), dépend de Vite 5 ;
    `bun audit` y relève 9 vulnérabilités (1 élevée, 7 modérées, 1 faible :
    `vite <=6.4.2`, `esbuild <=0.24.2`, `launch-editor`) ;
  - la branche 2.0 s'appuie sur Vite 8, la même version majeure que `web/`,
    et `bun audit` n'y relève aucune vulnérabilité ;
  - essai réussi sur la 2.0.0-alpha.21 : construction avec `base`, recherche
    locale, menu de navigation vers des URL absolues, échec sur un lien
    mort ;
  - la documentation n'utilise que la configuration documentée : le risque
    d'une alpha est minime, et l'épinglage exact protège des changements
    entre deux alphas.
- **Alternatives considered** :
  - `1.6.4` stable : écartée (vulnérabilités, Vite en fin de vie, alertes
    Dependabot permanentes) ;
  - autre générateur : écarté, car VitePress est imposé par la demande.

## R2. Exécution avec Bun, sans Node

- **Decision** : les scripts de `docs/package.json` appellent
  `bun --bun vitepress …` : la CLI tourne sous le runtime Bun.
- **Rationale** : les prérequis du guide de contribution restent Go et Bun ;
  l'essai (`dev`, `build`) fonctionne. La CI utilise `oven-sh/setup-bun` avec
  le même `BUN_VERSION` que `ci.yml`.
- **Alternatives considered** : Node + npm, écarté car le projet a choisi
  Bun. `actions/setup-node` en secours si une incompatibilité apparaît en CI.

## R3. Contenu du dossier `docs/`

- **Decision** :

  ```text
  docs/
  ├── package.json          # vitepress, vue ; scripts dev, build, preview
  ├── bun.lock
  ├── .vitepress/config.mts # seule configuration
  ├── index.md              # accueil (layout: home, frontmatter)
  ├── installation/*.md
  ├── usage/*.md
  └── contributing/*.md
  ```

  `docs/` est la racine du site (pas de `srcDir`). Les seuls fichiers
  non Markdown sont la configuration, le manifeste et le verrou.
- **Rationale** : c'est la forme minimale d'un projet VitePress. `README.md`
  n'existe pas dans `docs/`, donc rien à exclure. `node_modules/` et
  `.vitepress/{cache,dist}` sont ignorés par VitePress et par `.gitignore`.
- **Alternatives considered** :
  - dossier dans `web/` : mélange les dépendances et les configurations
    Vite ;
  - dépôt séparé : casse FR-001, qui veut la documentation d'une release
    dans son étiquette.

## R4. Disposition des deux versions sur GitHub Pages

- **Decision** :
  - site : `https://balsakup.github.io/frigg/` ;
  - version « release » à la racine (`base` = `/frigg/`) ;
  - version « main » sous `/frigg/main/` (`base` = `/frigg/main/`) ;
  - aucune page ne s'appelle `main` ;
  - sans release documentée, la racine contient une page de redirection
    vers `/frigg/main/`, générée par le workflow, et le `404.html` de
    « main ».
- **Rationale** : chaque version est une construction VitePress complète et
  autonome ; seule sa `base` change. Les liens profonds restent stables
  (FR-008), et l'adresse de « main » ne change pas à la première release
  documentée.
- **Alternatives considered** : construire « main » à la racine tant qu'aucune
  release n'existe. Écartée, car les liens de « main » changeraient ensuite.

## R5. Construction des deux versions à chaque publication

- **Decision** : un workflow `docs.yml`.
  1. Il détermine la dernière release.
  2. Il extrait `main` et l'étiquette (worktree Git).
  3. Il construit chaque version avec sa propre configuration.
  4. Il assemble les sorties dans un seul artefact Pages.
  5. Il déploie avec `actions/deploy-pages`.
- **Rationale** :
  - reconstruire les deux versions garantit que l'une ne fait jamais
    disparaître l'autre (FR-015) et prend en compte une release supprimée ;
  - le déploiement Pages est atomique : un échec ne publie rien (FR-016) ;
  - chaque version est construite avec la configuration de sa source :
    l'interface entre le workflow et la configuration est un contrat stable
    ([build-env.md](./contracts/build-env.md)).
- **Alternatives considered** : une branche `gh-pages` mise à jour version par
  version. Écartée : écriture non atomique, jeton en écriture nécessaire, et
  une release supprimée resterait en ligne.

## R6. Choix de la dernière release (dans le workflow, sans script)

- **Decision** : une seule ligne de shell dans le workflow :

  ```sh
  gh release list --exclude-drafts --json tagName -q '.[].tagName' \
    | grep -E '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$' \
    | sed 's/-/~/' | sort -V | sed 's/~/-/' | tail -n1
  ```

  Remplacer le premier `-` par `~` fait passer une préversion avant la
  version stable correspondante dans `sort -V` (convention Debian). Le
  workflow vérifie ensuite que l'étiquette contient
  `docs/.vitepress/config.mts` ; sinon, la version « release » est absente
  (FR-011).
- **Rationale** : vérifié le 2026-11-12 sur `v0.0.3-alpha`, `v1.0.0-rc.2`,
  `v1.0.0`, `v1.1.0-rc.1`, `v1.1.4`, `v1.2.0`, `v1.3.0-rc.9` et
  `v1.3.0-rc.10` : l'ordre obtenu est l'ordre attendu. Aucun script dans
  `docs/`.
- **Alternatives considered** :
  - script Bun avec `Bun.semver.order` : plus exact sur des suffixes
    exotiques, mais ajoute un script et des tests dans `docs/` ;
  - « latest » au sens de GitHub : exclut les préversions, alors que la spec
    les inclut.

## R7. Sélecteur et libellé de version (configuration seule)

- **Decision** : une entrée `nav` de la configuration, au texte
  `DOCS_VERSION` (`v1.2.0-rc.1`, ou `main (en développement)`). Son menu
  contient :
  - `Dernière release (<tag>)` → `${DOCS_SITE_URL}`, absent si aucune
    release n'est documentée ;
  - `main (en développement)` → `${DOCS_SITE_URL}main/`.

  Les liens sont des URL absolues avec `target: '_self'`. Un chemin absolu
  serait préfixé par la `base` : `/frigg/main/` deviendrait
  `/frigg/frigg/main/`.
- **Rationale** : essai réussi. Le libellé est visible sur chaque page
  (FR-003) et le choix mène à l'accueil de l'autre version (FR-009). Le thème
  par défaut ajoute l'icône « lien externe », ce qui est acceptable.
- **Écarté** : garder la même page lors du changement de version, et un
  bandeau sur « main ». Ces deux fonctions demandent des composants Vue, ce
  qui est contraire au principe directeur.

## R8. Page introuvable

- **Decision** : page 404 par défaut de VitePress, avec des textes en
  français (`themeConfig.notFound`). GitHub Pages ne sert que le
  `404.html` racine, dont le lien mène à l'accueil de la version racine
  (FR-006).

## R9. Recherche

- **Decision** : `search: { provider: 'local' }` avec les libellés traduits
  dans la configuration. L'index est propre à chaque construction, donc à
  chaque version (FR-004). Pas de service externe.

## R10. Workflow : déclenchement, concurrence, sécurité

- **Decision** :
  - **déclencheurs** :
    - `push` sur `main` et `pull_request`, filtrés sur `docs/**` et
      `.github/workflows/docs.yml` ;
    - `release` (`published`) ;
    - `workflow_dispatch` ;
    - sur une PR : construction seule (liens morts), sans déploiement
      (FR-021) ;
  - **concurrence** : groupe `pages` sans annulation pour les déploiements
    (FR-017), groupe par PR avec annulation pour les PR ;
  - **permissions** : `permissions: {}` par défaut, `contents: read` pour la
    construction, `pages: write` et `id-token: write` pour le seul
    déploiement ; jeton passé uniquement à l'étape `gh release list` ;
    `persist-credentials: false` ; actions épinglées par SHA (FR-018) ;
  - **événement `release`** : GitHub exécute le workflow de l'étiquette ; le
    workflow extrait donc explicitement `main` pour la version « main ».
- **Pourquoi pas dans `ci.yml`** : ses jobs sont des checks requis ; un job
  filtré par chemins y resterait en attente sur les PR sans documentation.
- **Réglage unique** : Settings → Pages → Source « GitHub Actions ».

## R11. Mise à jour des dépendances

- **Decision** : bloc `bun` pour `/docs` dans `.github/dependabot.yml`, avec
  la même politique que `/web` (correctifs de sécurité seulement). Les mises à
  jour de version de `vitepress` sont ignorées : on les fait à la main,
  jusqu'à la 2.0 stable.

## R12. Contenu initial

- **Decision** : trois guides (FR-002), une page par sujet.
  - **Installation** : reprend le README, qui est ensuite réduit.
  - **Utilisation** : rédigé à partir des spécifications 001 à 014, sans
    captures d'écran ; le rôle requis est indiqué en tête de page
    (`**Rôle requis** : Administrateur`).
  - **Contribution** : reprend `CONTRIBUTING.md`, qui garde le CLA et
    l'ouverture d'une Pull Request.
  - **Liens vers des fichiers du dépôt hors de `docs/`** (`CLA.md`,
    `.env.example`, `specs/…`) : toujours en URL GitHub absolue. Un lien
    relatif serait vu comme un lien mort et ferait échouer la construction.
