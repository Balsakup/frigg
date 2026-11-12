# Feature Specification: Site de documentation versionné

**Feature Branch**: `015-vitepress-docs`

**Created**: 2026-11-12

**Status**: Draft

**Input**: User description: "Mise en place de la documentation avec VitePress,
la documentation devra être hébergée sur Github Pages. La documentation sera
générée pour 2 versions, les versions released sur Github (pre-release inclut,
on prend la dernière version) et main"

## Vocabulaire

- **Site de documentation** : le site web statique et public qui présente la
  documentation de Frigg, distinct de l'application et de toute instance.
- **Release** : une publication GitHub du dépôt `Balsakup/frigg` associée à
  une étiquette `vX.Y.Z` ou `vX.Y.Z-suffixe` (voir spécification 007). Les
  préversions (pre-releases) sont des releases à part entière pour la
  documentation.
- **Dernière release** : parmi les releases publiées (préversions comprises,
  brouillons exclus), celle dont l'étiquette est la plus haute selon l'ordre
  des versions (`1.0.0` > `1.0.0-rc.2` > `0.9.1`).
- **Version « release »** : la documentation telle qu'elle existe dans le code
  de la dernière release.
- **Version « main »** : la documentation telle qu'elle existe sur la branche
  principale, qui décrit ce qui n'est pas encore publié.
- **Sélecteur de version** : l'entrée de la barre de navigation qui indique
  la version consultée et permet d'ouvrir l'autre.
- **Mainteneur** : la personne qui fusionne sur la branche principale et crée
  les releases.
- **Lecteur** : toute personne qui consulte la documentation (exploitant,
  administrateur d'association, membre, contributeur).

## Clarifications

### Session 2026-11-12

- Q: Quel contenu le site couvre-t-il à sa mise en ligne ? → A: Trois guides :
  installation et exploitation (repris du README), utilisation pour les
  administrateurs et les membres, et contribution (repris de
  `CONTRIBUTING.md`).
- Q: La documentation suit-elle les exigences de l'application
  (accessibilité vérifiée, identité visuelle, composants) ? → A: Non. La
  documentation doit être la plus simple possible : uniquement des fichiers
  Markdown et la configuration du générateur, avec son thème par défaut, sans
  composant, style ni test dédié.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Consulter la documentation de la version installée (Priority: P1)

Un exploitant qui installe ou met à jour Frigg ouvre l'adresse publique de la
documentation. Il arrive directement sur la documentation de la dernière
release, celle qui correspond à l'image qu'il va déployer. Il voit quel
numéro de version il consulte, navigue entre les pages et recherche un terme.

**Why this priority** : c'est la raison d'être du site ; aujourd'hui la seule
documentation est le README, qui suit la branche principale et peut décrire
des comportements absents de la version publiée.

**Independent Test** : après publication d'une release contenant la
documentation, ouvrir l'adresse racine du site et vérifier que le numéro de
version affiché est celui de la release et que le contenu correspond au code
de cette étiquette.

**Acceptance Scenarios**:

1. **Given** les releases `v1.0.0` et `v1.1.0-rc.1` sont publiées, **When** un
   lecteur ouvre l'adresse racine du site, **Then** il consulte la
   documentation de `v1.1.0-rc.1`, et ce numéro est affiché dans la barre de
   navigation.
2. **Given** la documentation de la dernière release, **When** le lecteur
   recherche un terme présent dans une page, **Then** cette page apparaît dans
   les résultats.

---

### User Story 2 - Publication automatique à chaque release et à chaque fusion (Priority: P1)

Le mainteneur ne fait rien de particulier pour publier la documentation : une
fusion sur la branche principale met à jour la version « main » ; la
publication d'une release met à jour la version « release ». En cas d'échec,
le site précédent reste en ligne et le mainteneur est notifié par GitHub.

**Why this priority** : une documentation publiée à la main finit toujours
par diverger du code ; sans automatisation, la US1 n'est pas tenable.

**Independent Test** : modifier une page sur la branche principale, fusionner,
puis vérifier que la version « main » du site affiche la modification sans
que la version « release » ne change ; publier ensuite une préversion et
vérifier que la version « release » affiche son contenu.

**Acceptance Scenarios**:

1. **Given** une modification de la documentation fusionnée sur la branche
   principale, **When** la publication se termine, **Then** la version « main »
   affiche la modification et la version « release » est inchangée.
2. **Given** une nouvelle release `v1.2.0` publiée, **When** la publication se
   termine, **Then** la version « release » affiche le contenu de `v1.2.0` et
   la version « main » reste celle de la branche principale.
3. **Given** un correctif `v1.1.4` publié après `v1.2.0`, **When** la
   publication se termine, **Then** la version « release » reste celle de
   `v1.2.0`.
4. **Given** une erreur dans la documentation (lien interne cassé, page qui ne
   se construit pas), **When** la publication s'exécute, **Then** elle échoue
   sans rien publier, le site précédent reste accessible et l'échec est
   visible dans l'onglet Actions.
5. **Given** une modification qui ne touche pas la documentation, **When**
   elle est fusionnée sur la branche principale, **Then** le site publié est
   inchangé.

---

### User Story 3 - Passer de la version publiée à la version en développement (Priority: P2)

Un contributeur ou un exploitant curieux veut savoir ce qui arrive dans la
prochaine version. Depuis n'importe quelle page, il ouvre le sélecteur de
version et choisit « main (en développement) ». Il arrive sur l'accueil de la
version « main », dont le libellé lui rappelle qu'il s'agit d'une
documentation en cours de développement.

**Why this priority** : utile pour préparer une mise à jour ou contribuer,
mais la majorité des lecteurs n'a besoin que de la version publiée.

**Independent Test** : depuis une page de la version « release », choisir
« main » dans le sélecteur et vérifier l'arrivée sur l'accueil de la version
« main », dont le libellé indique « main (en développement) ».

**Acceptance Scenarios**:

1. **Given** un lecteur sur une page de la version « release », **When** il
   choisit « main (en développement) » dans le sélecteur, **Then** il arrive
   sur l'accueil de la version « main ».
2. **Given** un lecteur sur la version « main », **When** il choisit la
   dernière release dans le sélecteur, **Then** il arrive sur l'accueil de la
   version « release ».
3. **Given** un lecteur sur la version « main », **When** il regarde la barre
   de navigation, **Then** le libellé de version indique « main (en
   développement) ».

---

### User Story 4 - Rédiger et prévisualiser la documentation en local (Priority: P3)

Un contributeur modifie une page de documentation dans le dépôt et la
prévisualise sur son poste, avec rechargement à chaque enregistrement, avant
d'ouvrir une Pull Request. Les contrôles de la Pull Request lui signalent une
documentation qui ne se construit pas ou un lien interne cassé.

**Why this priority** : facilite les contributions, mais le site peut exister
sans cette commande.

**Independent Test** : lancer la commande de prévisualisation documentée dans
le guide de contribution, modifier une page et voir la modification sans
relancer la commande.

**Acceptance Scenarios**:

1. **Given** le dépôt cloné et les prérequis du guide de contribution
   installés, **When** le contributeur lance la commande de prévisualisation,
   **Then** le site s'ouvre localement et se met à jour à chaque
   enregistrement.
2. **Given** une Pull Request qui introduit un lien interne cassé dans la
   documentation, **When** les contrôles s'exécutent, **Then** ils échouent en
   nommant la page et le lien.

---

### Edge Cases

- **Aucune release ne contient encore la documentation** (cas actuel : les
  préversions `v0.0.1-alpha` à `v0.0.3-alpha` sont antérieures au site) : seule
  la version « main » est publiée ; l'adresse racine mène à la version
  « main », et le sélecteur ne propose pas de version « release » jusqu'à la
  première release qui contient la documentation.
- **Aucune release publiée** : même comportement que ci-dessus.
- **Release en brouillon** : ignorée ; elle n'est pas une release publiée.
- **Préversion plus récente qu'une version stable** (`v1.1.0-rc.1` après
  `v1.0.0`) : la préversion est la dernière release ; le sélecteur affiche son
  numéro complet, suffixe compris, pour que le lecteur sache qu'il s'agit
  d'une préversion.
- **Release supprimée ou repassée en brouillon** : à la publication suivante,
  la version « release » est recalculée et redevient celle de la plus haute
  release encore publiée.
- **Publications simultanées** (fusion sur la branche principale pendant la
  publication d'une release) : les publications ne s'écrasent pas
  mutuellement ; une fois toutes terminées, chaque version affiche son contenu
  le plus récent.
- **Adresse inconnue** : le site affiche la page « introuvable » par défaut du
  générateur, avec un lien vers l'accueil du site.

## Requirements *(mandatory)*

### Functional Requirements

**Contenu et lecture**

- **FR-001** : Le dépôt MUST contenir les sources de la documentation, rédigées
  en français, versionnées avec le code : la documentation d'une release est
  celle présente dans le code de son étiquette.
- **FR-002** : Le site MUST couvrir, dès sa mise en ligne, trois guides
  accessibles depuis l'accueil et la navigation principale :
  - **Installation et exploitation** (exploitants) : installation avec Docker,
    configuration (chaque variable d'environnement), premier administrateur,
    sauvegarde et restauration, mise à jour, activation de HelloAsso ; ce
    contenu reprend celui du README ;
  - **Utilisation** (administrateurs et membres) : une page par domaine
    fonctionnel déjà livré (connexion et compte, comptes administrateurs,
    paramètres de l'association, saisons, cotisations, membres et champs
    personnalisés, importation des membres, adhésions, paiements par carte,
    remises et dons, événements et inscriptions, e-mails groupés), chacune
    indiquant le rôle requis pour les actions décrites ;
  - **Contribution** (contributeurs) : environnement de développement,
    workflow Spec Kit, contrôles qualité, CLA ; ce contenu reprend celui de
    `CONTRIBUTING.md`.
- **FR-002a** : Une fonctionnalité qui modifie un comportement visible par un
  exploitant, un administrateur ou un membre MUST mettre à jour la page
  correspondante dans la même Pull Request, afin que la version « main » reste
  fidèle au code.
- **FR-003** : La barre de navigation de chaque page MUST afficher la version
  consultée : le numéro complet de la release (suffixe de préversion compris)
  ou « main (en développement) ».
- **FR-004** : Le site MUST proposer une recherche plein texte limitée à la
  version consultée, utilisable sans service externe ni compte.
- **FR-005** : Les sources de la documentation MUST se limiter à des fichiers
  Markdown et à la configuration du générateur (plus le manifeste de
  dépendances et son verrou, nécessaires à l'installation). Le thème par
  défaut est utilisé tel quel : aucun composant, style, police, script ni
  test propre à la documentation.
- **FR-006** : Une adresse inconnue MUST afficher la page « introuvable » par
  défaut du générateur, avec un lien vers l'accueil du site.

**Versions**

- **FR-007** : Le site MUST publier exactement deux versions : la version
  « release » (dernière release, préversions comprises, brouillons exclus,
  ordre des versions) et la version « main ».
- **FR-008** : L'adresse racine du site MUST mener à la version « release » ;
  la version « main » MUST être servie sous un chemin distinct et stable
  (`/main/`), de sorte que les liens vers l'une ou l'autre restent
  valides d'une publication à l'autre.
- **FR-009** : Le sélecteur de version MUST être présent sur chaque page et
  mener à l'accueil de l'autre version.
- **FR-010** : *(supprimée le 2026-11-12 : le bandeau est remplacé par le
  libellé « main (en développement) » de FR-003)*
- **FR-011** : Tant qu'aucune release ne contient la documentation, l'adresse
  racine MUST mener à la version « main » et le sélecteur MUST NOT proposer de
  version « release ».
- **FR-012** : Le site MUST NOT conserver d'anciennes versions : une nouvelle
  dernière release remplace la précédente.

**Publication**

- **FR-013** : Le site MUST être hébergé sur GitHub Pages, à l'adresse du
  dépôt, et accessible publiquement sans authentification.
- **FR-014** : La publication MUST se déclencher automatiquement à chaque
  fusion sur la branche principale qui modifie la documentation ou sa
  configuration, et à chaque publication d'une release (préversion comprise) ;
  le mainteneur MUST pouvoir la relancer manuellement.
- **FR-015** : Chaque publication MUST reconstruire les deux versions à partir
  de leur source (branche principale et étiquette de la dernière release),
  afin que l'une ne fasse jamais disparaître l'autre.
- **FR-016** : La publication MUST échouer sans rien publier si l'une des
  versions ne se construit pas ou contient un lien interne cassé ; le site
  précédent MUST rester en ligne et l'échec MUST être visible dans l'onglet
  Actions du dépôt et notifié par GitHub.
- **FR-017** : Deux publications concurrentes MUST être sérialisées : la
  dernière à se terminer publie l'état le plus récent des deux versions.
- **FR-018** : La publication MUST respecter les règles de sécurité des
  workflows existants (spécification 007) : permissions minimales, actions
  tierces épinglées par empreinte, aucun secret exposé au contenu construit.
- **FR-019** : Le site MUST NOT contenir de donnée d'une association ni de
  donnée personnelle : il est statique, public et indépendant de toute
  instance.

**Contribution**

- **FR-020** : Une commande documentée dans le guide de contribution MUST
  permettre de prévisualiser le site en local avec rechargement automatique,
  et une autre de le construire comme en production.
- **FR-021** : Les contrôles des Pull Requests MUST construire la
  documentation et échouer sur une erreur de construction ou un lien interne
  cassé, lorsque la Pull Request modifie la documentation.
- **FR-022** : Le README et `CONTRIBUTING.md` MUST renvoyer vers le site de
  documentation ; le contenu repris n'est plus maintenu à deux endroits.
  `CONTRIBUTING.md` conserve un résumé (CLA, ouverture d'une Pull Request),
  car GitHub l'affiche aux contributeurs.

### Key Entities

- **Version de documentation** : identifiant (« main » ou numéro de release),
  source (branche principale ou étiquette), chemin de publication sur le site.
- **Page** : fichier Markdown, titre, contenu ; même arborescence dans les
  deux versions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001** : Moins de 10 minutes après la publication d'une release, la
  version « release » du site affiche son numéro et son contenu.
- **SC-002** : Moins de 10 minutes après une fusion qui modifie la
  documentation, la version « main » affiche la modification.
- **SC-003** : Aucune action manuelle du mainteneur n'est nécessaire pour
  publier la documentation, hors création de la release.
- **SC-004** : 100 % des pages affichent la version consultée, et passer à
  l'autre version demande un seul choix dans le sélecteur.
- **SC-005** : Un lecteur trouve la page d'installation depuis l'adresse
  racine en moins de 30 secondes, par la navigation ou la recherche.
- **SC-006** : Le dossier de la documentation ne contient que des fichiers
  Markdown, un fichier de configuration du générateur, le manifeste de
  dépendances et son verrou.
- **SC-007** : Une publication en échec laisse le site précédent accessible
  dans 100 % des cas.
- **SC-008** : Chaque domaine fonctionnel livré (liste de FR-002) a sa page
  dans le guide d'utilisation ; un administrateur trouve comment réaliser une
  tâche courante (créer une saison, enregistrer une adhésion) en moins de
  2 minutes depuis l'accueil.

## Assumptions

- Le site est un projet distinct de l'application : il n'est pas intégré au
  binaire ni servi par une instance. Les règles de la constitution propres à
  l'application servie ne s'y appliquent pas : authentification, design
  system, polices auto-hébergées, conformité RGAA vérifiée par des tests.
  L'accessibilité et l'affichage sur mobile sont ceux du thème par défaut du
  générateur. Son contenu étant public et sans donnée d'association, cela ne
  crée aucune route publique au sens du principe II.
- Le site est rédigé en français uniquement, comme le reste de la
  documentation du projet ; aucune traduction n'est prévue.
- L'adresse du site est celle fournie par GitHub Pages pour le dépôt ; un nom
  de domaine personnalisé est hors périmètre et pourra être ajouté plus tard.
- Le dépôt est public et GitHub Pages peut y être activé par le mainteneur
  (réglage unique, documenté dans le plan).
- Les sources de la documentation suivent le gestionnaire de paquets déjà
  utilisé par le projet (Bun) ; l'outil de génération demandé est VitePress.
- Le format des étiquettes de release est celui imposé par la spécification
  007 ; une étiquette hors format n'est jamais une release publiée valide.
- La documentation de l'API REST (schéma OpenAPI) n'est pas intégrée au site
  dans cette fonctionnalité.
- Les anciennes versions ne sont pas archivées : seules la dernière release et
  la branche principale sont consultables (FR-012).
- Le rôle requis par une page du guide d'utilisation est indiqué en texte, en
  tête de page.
