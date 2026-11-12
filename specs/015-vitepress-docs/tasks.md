---

description: "Liste des tâches : site de documentation versionné"
---

# Tasks: Site de documentation versionné

**Input**: Design documents from `specs/015-vitepress-docs/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/
(build-env.md, site.md, workflow.md), quickstart.md

**Tests** : aucun test automatisé propre à la documentation (clarification
du 2026-11-12). La construction VitePress sert de contrôle, avec liens morts
bloquants. La validation est manuelle (quickstart.md).

**Organization** : une phase par user story.

- **US1 (P1)** : contenu et lecture.
- **US2 (P1)** : publication automatique.
- **US3 (P2)** : sélecteur de version.
- **US4 (P3)** : prévisualisation et contrôle des PR.

**Règle de simplicité** (FR-005, SC-006) : `docs/` ne contient que des
fichiers `.md`, `docs/.vitepress/config.mts`, `docs/package.json` et
`docs/bun.lock`.

- Aucun composant Vue, CSS, police, script ni test dans `docs/` ; thème par
  défaut.
- Toute logique de version et d'assemblage va dans
  `.github/workflows/docs.yml`.

**Conventions** :

- contenu et messages du workflow en **français** ; noms de fichiers,
  variables `DOCS_*` et jobs en **anglais** ;
- liens vers un fichier du dépôt hors de `docs/` (`CLA.md`, `LICENSE`,
  `.env.example`, `specs/…`) **toujours en URL absolue**
  `https://github.com/Balsakup/frigg/blob/main/<chemin>`. Un lien relatif
  serait vu comme un lien mort et ferait échouer la construction ;
- aucune donnée personnelle ni secret dans les exemples (domaines
  `exemple.fr`, `mon-asso.fr`).

## Format: `[ID] [P?] [Story] Description`

- **[P]** : parallélisable (fichiers différents, aucune dépendance sur une
  tâche inachevée)
- **[Story]** : user story de rattachement (US1 à US4)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Créer `docs/package.json` :
  - `"name": "docs"`, `"private": true`, `"type": "module"`,
    `"license": "AGPL-3.0-or-later"` ;
  - scripts : `"dev": "bun --bun vitepress dev --port 5174"`,
    `"build": "bun --bun vitepress build"`,
    `"preview": "bun --bun vitepress preview"` ;
  - `devDependencies` exactes : `"vitepress": "2.0.0-alpha.21"`,
    `"vue": "3.5.45"` ;
  - lancer `cd docs && bun install` pour produire `docs/bun.lock`.
- [X] T002 [P] Ajouter à `.gitignore` :
  `docs/node_modules/`, `docs/.vitepress/dist/`, `docs/.vitepress/cache/`.
- [X] T003 [P] Ajouter à `Taskfile.yml` deux tâches décrites en français :
  - `docs:dev` (« Prévisualise la documentation (http://localhost:5174) »),
    `dir: docs`, commandes `bun install --frozen-lockfile` puis
    `bun run dev` ;
  - `docs:build` (« Construit la documentation comme en production »),
    `dir: docs`, commandes `bun install --frozen-lockfile` puis
    `bun run build`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL** : aucune story ne commence avant la fin de cette phase.

- [X] T004 Créer `docs/.vitepress/config.mts` (`defineConfig` de
  `vitepress`) :
  - **variables** de contracts/build-env.md, lues en tête de fichier :
    `const base = process.env.DOCS_BASE ?? '/'`,
    `const version = process.env.DOCS_VERSION ?? 'main'`,
    `const release = process.env.DOCS_RELEASE_VERSION ?? ''`,
    `const siteUrl = process.env.DOCS_SITE_URL ?? 'https://balsakup.github.io/frigg/'` ;
  - **site** : `base`, `lang: 'fr-FR'`, `title: 'Frigg'`,
    `description: "Documentation de Frigg, gestion d'association auto-hébergée"`,
    `cleanUrls: true`. Ne pas définir `ignoreDeadLinks`, pour que les liens
    morts restent bloquants ;
  - **`themeConfig`** :
    - `nav` : les trois guides et le libellé de version, ajoutés en US1 ;
      l'entrée « Dernière release » du sélecteur est ajoutée en US3 ;
    - libellés en français : `outline: { label: 'Sur cette page' }`,
      `docFooter: { prev: 'Page précédente', next: 'Page suivante' }`,
      `darkModeSwitchLabel: 'Apparence'`, `sidebarMenuLabel: 'Menu'`,
      `returnToTopLabel: 'Retour en haut'`,
      `skipToContentLabel: 'Aller au contenu'`,
      `notFound: { title: 'Page introuvable', quote: "Cette page n'existe pas dans cette version de la documentation.", linkLabel: "Revenir à l'accueil", linkText: "Revenir à l'accueil" }` ;
    - `socialLinks: [{ icon: 'github', link: 'https://github.com/Balsakup/frigg' }]`.
- [X] T005 Créer `docs/index.md`. Frontmatter uniquement :
  - `layout: home` ;
  - `hero` : `name: Frigg`, `text: Gestion d'association auto-hébergée`,
    `tagline` d'une phrase ;
  - `features` à trois entrées, chacune avec un `link` :
    - « Installer et exploiter » → `/installation/` ;
    - « Utiliser Frigg » → `/usage/` ;
    - « Contribuer » → `/contributing/`.

  Les liens restent morts jusqu'à US1 : ne pas lancer `go tool task
  docs:build` avant la fin de T007 à T011.

**Checkpoint** : configuration et accueil en place.

---

## Phase 3: User Story 1 - Consulter la documentation de la version installée (Priority: P1) 🎯 MVP

**Goal** : un site complet en français (trois guides), avec la version
affichée et une recherche.

**Independent Test** : `DOCS_VERSION=v9.9.9 go tool task docs:build` passe,
puis `cd docs && bun run preview`. Vérifier que :

- la barre de navigation affiche `v9.9.9` sur l'accueil et sur une page de
  chaque guide (dans le menu sur mobile) ;
- sans variable, elle affiche `main (en développement)` ;
- la recherche de « sauvegarde » propose la page Sauvegarder et restaurer ;
- chaque guide est accessible depuis la navigation.

- [X] T006 [US1] Dans `docs/.vitepress/config.mts`, compléter `themeConfig` :
  - `nav` : `{ text: 'Installation', link: '/installation/' }`,
    `{ text: 'Utilisation', link: '/usage/' }`,
    `{ text: 'Contribution', link: '/contributing/' }`, puis le libellé de
    version (FR-003, data-model.md › Version de documentation), en dernière
    position :
    `{ text: version === 'main' ? 'main (en développement)' : version, items: [{ text: 'main (en développement)', link: \`${siteUrl}main/\`, target: '_self' }] }`.
    Le lien est une URL absolue : un chemin absolu serait préfixé par `base`
    (research.md › R7). L'entrée « Dernière release » est ajoutée en US3
    (T016) ;
  - `sidebar` par préfixe (`'/installation/'`, `'/usage/'`,
    `'/contributing/'`), avec les pages dans l'ordre de contracts/site.md ›
    Arborescence ;
  - `search: { provider: 'local', options: { translations } }`, avec les
    libellés en français :
    - `button` : `buttonText: 'Rechercher'`,
      `buttonAriaLabel: 'Rechercher'` ;
    - `modal` : `noResultsText: 'Aucun résultat pour'`,
      `resetButtonTitle: 'Effacer la recherche'`, et `footer` avec
      `selectText: 'pour sélectionner'`, `navigateText: 'pour naviguer'`,
      `closeText: 'pour fermer'`.
- [X] T007 [P] [US1] Rédiger le guide d'installation dans
  `docs/installation/` : `index.md`, `docker.md`, `configuration.md`,
  `first-admin.md`, `backup.md`, `upgrade.md`, `reverse-proxy.md`,
  `hardening.md`, `helloasso.md` et `email.md`.
  - Reprendre sans perte les sections « Installer avec Docker » à « Paiement
    par carte (HelloAsso) » de `README.md`.
  - `configuration.md` documente **chaque** variable de `.env.example` :
    nom, obligatoire ou facultative, valeur par défaut, rôle.
  - `email.md` décrit le SMTP et la file d'envoi (états `pending`, `sent`,
    `failed`, nouvelles tentatives), d'après la section « Communication » du
    README et `specs/009-bulk-email/spec.md`.
- [X] T008 [P] [US1] Rédiger `docs/usage/index.md` : rôles membre et
  administrateur (un administrateur peut tout ce que peut un membre), menu
  selon le rôle, recherche globale (Ctrl K). Rédiger aussi :
  - `docs/usage/account.md`, d'après `specs/001-auth-app-shell/spec.md` :
    connexion, mot de passe oublié, vérification de l'adresse, invitation ;
  - `docs/usage/admin-accounts.md`, d'après
    `specs/002-admin-account-management/spec.md` ;
  - `docs/usage/settings.md`, d'après
    `specs/010-general-settings/spec.md` et la section « Paramètres de
    l'association » du README.

  Chaque page commence sous son titre par `**Rôle requis** : Membre` ou
  `**Rôle requis** : Administrateur`.
- [X] T009 [P] [US1] Rédiger, à partir des spécifications citées (parcours et
  règles métier visibles par l'utilisateur, renvois vers les pages liées,
  ligne `**Rôle requis**` en tête) :
  - `docs/usage/seasons.md` (`specs/003-season-management/spec.md`) ;
  - `docs/usage/fees.md` (`specs/004-membership-fees/spec.md`) ;
  - `docs/usage/members.md` (`specs/005-member-management/spec.md`) ;
  - `docs/usage/custom-fields.md` (`specs/011-custom-fields/spec.md`) ;
  - `docs/usage/member-import.md` (`specs/013-member-import/spec.md`).
- [X] T010 [P] [US1] Rédiger, de la même façon :
  - `docs/usage/memberships.md` (`specs/006-memberships/spec.md`), y compris
    le renouvellement déclaré par un membre d'une association antérieure à
    Frigg, puis validé par un administrateur ;
  - `docs/usage/card-payment.md` (`specs/008-helloasso-card-payment/spec.md`,
    côté membre et côté administrateur) ;
  - `docs/usage/adjustments.md` (`specs/014-payment-adjustments/spec.md`) ;
  - `docs/usage/events.md` (`specs/012-events/spec.md` et la section
    « Événements et inscriptions » du README) ;
  - `docs/usage/bulk-email.md` (`specs/009-bulk-email/spec.md`).
- [X] T011 [P] [US1] Rédiger le guide de contribution dans
  `docs/contributing/` :
  - `index.md` : « Licence et CLA » et « Proposer une modification » de
    `CONTRIBUTING.md`, avec des liens absolus vers `CLA.md` et `LICENSE` ;
  - `setup.md` : « Mettre en place l'environnement » ;
  - `workflow.md` : cycle Spec Kit (spécification, plan, tâches,
    implémentation), conventions de langue de la constitution, commits
    (gitmoji, message court en français) ;
  - `checks.md` : contrôles avant fusion (`go tool task check` et la section
    « Workflow de développement et contrôles qualité » de la constitution) ;
  - `release.md` : « Construire l'image en local » et « Publier une
    release » ;
  - `documentation.md` est rédigé en US4 (T018). Ajouter dès maintenant le
    fichier avec son seul titre, pour que la barre latérale ne contienne pas
    de lien mort.
- [X] T012 [US1] Lancer `go tool task docs:build` : aucun lien mort. Puis
  `cd docs && DOCS_VERSION=v9.9.9 bun run build && bun run preview` et
  vérifier l'Independent Test ci-dessus, libellé de version compris. Lancer enfin la commande SC-006 du
  quickstart § 2 : aucune ligne.

**Checkpoint** : le contenu est complet et construit sans erreur.

---

## Phase 4: User Story 2 - Publication automatique à chaque release et à chaque fusion (Priority: P1)

**Goal** : chaque fusion sur `main` qui touche la documentation et chaque
release publiée reconstruisent et publient les deux versions de façon
atomique.

**Independent Test** : après fusion, le workflow « Documentation » publie le
site (quickstart.md § 4).

- [X] T013 [US2] Créer `.github/workflows/docs.yml`, conforme à
  contracts/workflow.md.
  - **En-tête** :
    - commentaire en français, comme `release.yml`, qui résume le
      fonctionnement et renvoie à
      `specs/015-vitepress-docs/contracts/workflow.md` ;
    - `name: Documentation`, `permissions: {}` ;
    - `env` : `BUN_VERSION: 1.4.4` (commentaire « garder alignée sur
      ci.yml ») et `SITE_URL: https://balsakup.github.io/frigg/`.
  - **Déclencheurs** : `push` sur `main` et `pull_request`, tous deux avec
    `paths: [docs/**, .github/workflows/docs.yml]` ; `release` avec
    `types: [published]` ; `workflow_dispatch`.
  - **Concurrence** :
    `group: ${{ github.event_name == 'pull_request' && format('docs-pr-{0}', github.event.pull_request.number) || 'pages' }}`
    et `cancel-in-progress: ${{ github.event_name == 'pull_request' }}`.
  - **Job `build`** (« Construit la documentation »), `permissions:
    contents: read`, `timeout-minutes: 15` :
    1. `actions/checkout` (même SHA que `ci.yml`) avec `path: src`,
       `fetch-depth: 0` et `persist-credentials: false`. `ref` vaut
       `${{ (github.event_name == 'release' || github.event_name == 'workflow_dispatch') && 'main' || github.ref }}` ;
    2. `oven-sh/setup-bun` (même SHA que `ci.yml`) avec
       `bun-version: ${{ env.BUN_VERSION }}` ;
    3. étape `id: release` « Détermine la dernière release », `if:
       github.event_name != 'pull_request'`, seule à recevoir
       `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}`. Elle lance la commande de
       research.md › R6 avec `--repo "$GITHUB_REPOSITORY"`, puis :
       - si `tag` n'est pas vide et que `git -C src cat-file -e "$tag:docs/.vitepress/config.mts"`
         échoue : `echo "::notice::$tag ne contient pas la documentation : seule la version main est publiée"`
         et `tag=` ;
       - écrire `tag=$tag` dans `$GITHUB_OUTPUT` ;
    4. « Construit la version main » : `working-directory: src/docs`, `env`
       `DOCS_BASE: /frigg/main/`, `DOCS_VERSION: main`,
       `DOCS_RELEASE_VERSION: ${{ steps.release.outputs.tag }}` et
       `DOCS_SITE_URL: ${{ env.SITE_URL }}`. Commandes
       `bun install --frozen-lockfile` puis `bun run build` ;
    5. « Construit la version release », `if: steps.release.outputs.tag != ''` :
       `git -C src worktree add ../release "$TAG"`, puis dans
       `release/docs` : `bun install --frozen-lockfile` et `bun run build`,
       avec `DOCS_BASE: /frigg/`, `DOCS_VERSION` et `DOCS_RELEASE_VERSION`
       égaux à l'étiquette, et `DOCS_SITE_URL: ${{ env.SITE_URL }}`. Passer
       l'étiquette par `env: TAG:`, jamais par interpolation dans le
       script ;
    6. « Assemble le site », `if: github.event_name != 'pull_request'`, bash
       `set -euo pipefail` :
       - avec release : copier `release/docs/.vitepress/dist/.` dans
         `site/`, échouer avec `::error::` si `site/main` existe déjà, puis
         copier `src/docs/.vitepress/dist/.` dans `site/main/` ;
       - sans release : copier la sortie main dans `site/main/`, écrire
         `site/index.html` (page HTML `lang="fr"`, titre « Documentation de
         Frigg », `<meta http-equiv="refresh" content="0; url=main/">`, lien
         visible « Accéder à la documentation (version en développement) »)
         et copier `site/main/404.html` dans `site/404.html` ;
       - ajouter au `$GITHUB_STEP_SUMMARY` « Versions publiées : main » et,
         selon le cas, « , <tag> » ou « (aucune release documentée) » ;
    7. `actions/configure-pages`, puis `actions/upload-pages-artifact` avec
       `path: site`, tous deux avec `if: github.event_name != 'pull_request'`.
  - **Job `deploy`** (« Publie sur GitHub Pages ») : `needs: build`,
    `if: github.event_name != 'pull_request'`, `permissions` (`pages: write`,
    `id-token: write`), `environment` (`name: github-pages`,
    `url: ${{ steps.deployment.outputs.page_url }}`), `timeout-minutes: 10`,
    étape unique `actions/deploy-pages` avec `id: deployment`.
  - **Épinglage** : pour `actions/configure-pages`,
    `actions/upload-pages-artifact` et `actions/deploy-pages`, récupérer le
    SHA de la dernière version avec
    `gh api repos/actions/<nom>/releases/latest -q .tag_name`, puis
    `gh api repos/actions/<nom>/commits/<tag> -q .sha`. Épingler avec la
    version en commentaire.
- [X] T014 [P] [US2] Dans `.github/dependabot.yml`, ajouter un bloc :
  - `package-ecosystem: bun`, `directory: /docs` ;
  - même politique que `/web` : `schedule.interval: weekly`,
    `open-pull-requests-limit: 0`, `commit-message.prefix: "⬆️"` ;
  - `ignore: [{ dependency-name: vitepress }]`, avec le commentaire « alpha
    épinglée exactement, mise à jour manuelle jusqu'à la 2.0 stable (spec
    015, research.md › R11) ».
- [X] T015 [US2] Vérifier en local :
  - le classement des versions de research.md › R6 sur les 8 étiquettes de
    référence. La commande suivante MUST afficher exactement `v0.0.3-alpha`,
    `v1.0.0-rc.2`, `v1.0.0`, `v1.1.0-rc.1`, `v1.1.4`, `v1.2.0`,
    `v1.3.0-rc.9`, `v1.3.0-rc.10`, dans cet ordre :
    ```sh
    printf '%s\n' v1.2.0 v1.0.0 v1.3.0-rc.10 v1.1.0-rc.1 v0.0.3-alpha v1.1.4 v1.0.0-rc.2 v1.3.0-rc.9 release-1 v1.0 \
      | grep -E '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$' \
      | sed 's/-/~/' | sort -V | sed 's/~/-/'
    ```
    `release-1` et `v1.0` sont écartés. Le filtre et le tri MUST être
    identiques, caractère pour caractère, à ceux de l'étape 3 de T013 ;
  - la commande du quickstart § 3 affiche `v0.0.3-alpha` ;
  - `actionlint .github/workflows/docs.yml`, si l'outil est installé, ne
    signale rien. Sinon, relire le YAML contre contracts/workflow.md.

**Checkpoint** : le workflow est prêt ; il publiera dès la fusion.

---

## Phase 5: User Story 3 - Passer de la version publiée à la version en développement (Priority: P2)

**Goal** : le sélecteur de version, présent sur chaque page, mène à l'accueil
de l'autre version. Le libellé et l'entrée « main » existent depuis T006 ;
cette phase ajoute l'entrée « Dernière release ».

**Independent Test** : `cd docs && DOCS_VERSION=v9.9.9 DOCS_RELEASE_VERSION=v9.9.9 bun run build && bun run preview`.
Le sélecteur affiche `v9.9.9` et propose « Dernière release (v9.9.9) » et
« main (en développement) ». Sans variable, il affiche « main (en
développement) » sans entrée release.

- [X] T016 [US3] Dans `docs/.vitepress/config.mts`, ajouter en tête des
  `items` du libellé de version (T006) l'entrée
  `{ text: \`Dernière release (${release})\`, link: siteUrl, target: '_self' }`,
  seulement si `release` n'est pas vide (par exemple
  `...(release ? [{ … }] : [])`). Le lien est une URL absolue, comme celui de
  l'entrée « main » (research.md › R7).
- [X] T017 [US3] Vérifier l'Independent Test ci-dessus dans les deux cas,
  puis `go tool task docs:build`.

**Checkpoint** : les deux versions sont reliées.

---

## Phase 6: User Story 4 - Rédiger et prévisualiser la documentation en local (Priority: P3)

**Goal** : un contributeur prévisualise ses modifications ; une PR qui casse
la documentation est signalée.

**Independent Test** : `go tool task docs:dev` applique une modification sans
relance ; sur une PR, un lien mort fait échouer « Construit la
documentation ».

- [X] T018 [P] [US4] Rédiger `docs/contributing/documentation.md` :
  - emplacement (`docs/`, une page par sujet, noms de fichiers en anglais,
    titres en français) ;
  - règle de simplicité : seulement du Markdown et `config.mts`, thème par
    défaut ;
  - liens en URL absolue vers les fichiers hors de `docs/` ;
  - ligne `**Rôle requis**` en tête des pages d'utilisation ;
  - commandes `go tool task docs:dev` et `go tool task docs:build` ;
  - règle FR-002a : mettre à jour la page dans la même Pull Request ;
  - les deux versions publiées, leurs adresses, et le nom de page `main`
    interdit ;
  - variables `DOCS_*` à ne jamais renommer (lien absolu vers
    `specs/015-vitepress-docs/contracts/build-env.md`).
- [X] T019 [P] [US4] Dans `.github/pull_request_template.md`, ajouter sous
  « Vérifications » la case `- [ ] La documentation (docs/) est à jour si le
  comportement visible change`.
- [X] T020 [US4] Vérifier :
  - le déclencheur `pull_request` de `docs.yml` (T013) : sur une PR, seules
    les étapes 1, 2 et 4 du job `build` s'exécutent, et `deploy` est
    ignoré ;
  - en local : ajouter `[x](./nope)` dans une page, lancer
    `go tool task docs:build`, constater l'échec `Found dead link ./nope in
    file …`, puis retirer le lien.

**Checkpoint** : contribuer à la documentation est outillé et contrôlé.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T021 [P] Réduire `README.md` :
  - présentation de Frigg ;
  - lien vers `https://balsakup.github.io/frigg/` en tête ;
  - démarrage rapide (`docker run` minimal, `docker exec -it frigg /frigg
    create-admin`) ;
  - liens vers les guides Installation, Utilisation et Contribution ;
  - sections « Contribuer » et « Licence » conservées ;
  - supprimer les sections reprises en T007, T008 et T010 (FR-022).
- [X] T022 [P] Réduire `CONTRIBUTING.md` : garder « Licence et CLA » et
  « Proposer une modification », et remplacer les autres sections par des
  liens vers les pages `https://balsakup.github.io/frigg/contributing/…`
  (FR-022).
- [X] T023 [P] Corriger les liens vers des sections disparues de `README.md`
  et de `CONTRIBUTING.md`, trouvés avec `grep -rn "README.md#\|CONTRIBUTING.md#"`
  hors de `specs/`.
- [X] T024 Relire les trois guides contre le code :
  - variables de `.env.example` ;
  - sous-commandes de `go run ./cmd/frigg --help` ;
  - libellés d'interface cités, retrouvés par `grep` dans `web/app/` ;
  - aucune donnée personnelle ni secret.
- [X] T025 Lancer `go tool task docs:build` et `go tool task check` : les
  deux passent, et `check` est inchangé par la fonctionnalité. Dérouler le
  quickstart §§ 1 à 3.
- [ ] T026 Après fusion, le mainteneur règle Pages (Source « GitHub
  Actions ») et l'environnement `github-pages` (déploiements limités à
  `main` et aux étiquettes `v*`), puis déroule le quickstart § 4.

---

## Dependencies & Execution Order

- **Setup (T001–T003)** : aucune dépendance ; T002 et T003 en parallèle
  après T001.
- **Foundational (T004–T005)** : dépend de T001.
- **US1 (T006–T012)** : dépend de Foundational. Les pages T007 à T011 sont
  en parallèle entre elles, après ou pendant T006. T012 vient en dernier.
- **US2 (T013–T015)** : dépend de Foundational. Elle peut avancer en
  parallèle de US1, mais ne publie un site utile qu'une fois US1 livrée.
- **US3 (T016–T017)** : dépend de T006, qui crée le libellé de version que
  T016 complète. `config.mts` est modifié séquentiellement par T004, T006
  puis T016.
- **US4 (T018–T020)** : T018 dépend de T011 (fichier créé) ; T020 dépend de
  T013.
- **Polish (T021–T026)** : T021 à T023 dépendent de US1 (contenu repris) ;
  T025 et T026 dépendent de tout.

## Parallel Example: User Story 1

```bash
Task: "Guide d'installation dans docs/installation/ (T007)"
Task: "Pages compte, comptes administrateurs, paramètres (T008)"
Task: "Pages saisons, tarifs, membres, champs personnalisés, importation (T009)"
Task: "Pages adhésions, paiement par carte, remises, événements, e-mails (T010)"
Task: "Guide de contribution dans docs/contributing/ (T011)"
```

## Implementation Strategy

### MVP (US1 + US2)

Les deux sont P1 : un site non publié ne sert à personne, une publication
sans contenu n'apporte rien.

1. Setup, puis Foundational.
2. US1 (contenu), puis US2 (workflow).
3. Fusion, puis quickstart § 4. Jusqu'à la prochaine release, la racine
   redirige vers la version « main ».

### Incremental Delivery

1. US3 : le sélecteur ne touche qu'une entrée de `config.mts`.
2. US4 : page de contribution et modèle de PR.
3. Polish : README et CONTRIBUTING allégés.

## Notes

- Commits : gitmoji, message court en français, validation par le mainteneur
  avant chaque commit.
- Le contrat `DOCS_*` (contracts/build-env.md) est figé dès la première
  release qui contient `docs/`. Toute évolution ajoute une variable avec une
  valeur par défaut.
- Si `bun --bun vitepress` échoue avec Bun 1.4.4 en CI, ajouter
  `actions/setup-node` au job `build` et retirer `--bun` des scripts
  (research.md › R2).
