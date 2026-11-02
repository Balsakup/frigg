# Feature Specification: Image de production et publication à chaque release

**Feature Branch**: `007-docker-release`

**Created**: 2026-11-02

**Status**: Implemented

**Input**: User description: "Image docker pour la prod. Créer une release sur Github build et publie sur Github l'image docker (balsakup/frigg:<version>). Ajout d'une commande dans taskfile pour builder l'image en local pour debug. L'image doit être sécurisée et légère"

## Vocabulaire

- **Image** : l'image de conteneur de production de Frigg, qui contient le
  binaire unique (frontend intégré) et rien d'autre que ce qui est strictement
  nécessaire à son exécution.
- **Release** : une publication GitHub du dépôt `Balsakup/frigg`, associée à
  une étiquette de version de la forme `vX.Y.Z` (version sémantique),
  éventuellement suivie d'un suffixe de préversion (`vX.Y.Z-rc.1`).
- **Version** : le numéro de la release sans le préfixe `v` (`1.2.0`), qui sert
  d'étiquette à l'image publiée.
- **Registre** : le registre de conteneurs de GitHub associé au dépôt, où
  l'image est publiée sous le nom `balsakup/frigg`.
- **Mainteneur** : la personne qui crée les releases du projet.
- **Exploitant** : le bénévole ou prestataire qui installe et met à jour une
  instance Frigg pour une association.
- **Répertoire de données de l'instance** : l'emplacement persistant qui
  contient la base SQLite et le répertoire des fichiers déposés (principe V).
- **Point de santé** : point d'accès HTTP public, sans session, qui indique
  seulement si l'instance est en état de servir des requêtes. Il est consulté
  par la vérification de santé de l'image, un orchestrateur ou un reverse
  proxy.

## Clarifications

### Session 2026-11-02

- Q: Faut-il un point de santé, alors que le principe II interdit toute route
  publique non listée ? → A: Oui. La constitution est amendée pour ajouter le
  point de santé à la liste des exceptions du principe II, et l'image
  intègre une vérification de santé.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Publier l'image en créant une release (Priority: P1)

Le mainteneur crée une release sur GitHub (par exemple `v1.2.0`). Sans autre
action de sa part, l'image correspondant exactement au code de cette release
est construite, vérifiée puis publiée dans le registre sous
`balsakup/frigg:1.2.0`. En cas de succès, la release indique l'image publiée ;
en cas d'échec, le mainteneur est notifié par GitHub et voit l'exécution en
erreur dans l'onglet Actions du dépôt.

**Why this priority**: c'est la raison d'être de la fonctionnalité : sans
publication automatique, aucun exploitant ne peut récupérer une image officielle
et chaque mise en production dépend d'une manipulation manuelle sur le poste du
mainteneur.

**Independent Test**: créer une release de test sur un dépôt (ou une
préversion), puis vérifier que l'image `balsakup/frigg:<version>` apparaît dans
le registre, qu'elle se télécharge sans authentification et qu'elle démarre.

**Acceptance Scenarios**:

1. **Given** une release `v1.2.0` est créée, **When** la publication se
   termine, **Then** l'image `balsakup/frigg:1.2.0` est disponible dans le
   registre et l'étiquette `latest` désigne cette même image.
2. **Given** une release `v1.3.0-rc.1` marquée comme préversion est créée,
   **When** la publication se termine, **Then** l'image
   `balsakup/frigg:1.3.0-rc.1` est disponible et l'étiquette `latest` désigne
   toujours la dernière version stable.
3. **Given** une release dont le code ne compile pas ou dont les tests
   automatisés échouent, **When** la publication s'exécute, **Then** aucune
   image n'est publiée, la release ne mentionne aucune image, et l'exécution
   en échec est visible dans l'onglet Actions et notifiée au mainteneur.
4. **Given** une release dont l'étiquette ne respecte pas le format `vX.Y.Z`
   (avec suffixe de préversion facultatif), **When** la publication s'exécute,
   **Then** elle échoue avec un message explicite et rien n'est publié.
5. **Given** une image publiée, **When** un exploitant consulte ses
   métadonnées, **Then** il y lit la version, la révision du code source,
   l'adresse du dépôt et la licence.

---

### User Story 2 - Exploiter Frigg à partir de l'image (Priority: P1)

Un exploitant déploie Frigg sur un serveur en lançant l'image, en lui
fournissant sa configuration par variables d'environnement et un volume
persistant pour les données. Il peut ensuite créer le premier administrateur,
sauvegarder la base et mettre à jour vers une nouvelle version sans perdre de
données.

**Why this priority**: une image publiée qui ne permet pas d'exploiter une
instance de bout en bout (premier administrateur, sauvegarde, mise à jour) n'a
aucune valeur pour une association.

**Independent Test**: lancer l'image avec la configuration minimale et un
volume vide, créer un administrateur, se connecter, vérifier que le
répertoire des fichiers déposés existe et est accessible en écriture, faire
une sauvegarde, puis relancer une image d'une version plus récente sur le même
volume et vérifier que les données sont intactes.

**Acceptance Scenarios**:

1. **Given** un volume vide et les variables obligatoires fournies, **When**
   l'exploitant lance l'image sans argument, **Then** le serveur démarre,
   applique les migrations et répond sur le port documenté.
2. **Given** une variable obligatoire manquante, **When** l'exploitant lance
   l'image, **Then** le conteneur s'arrête avec le message explicite déjà prévu
   par Frigg.
3. **Given** une instance lancée depuis l'image, **When** l'exploitant exécute
   la commande de création d'administrateur dans le conteneur en mode
   interactif, **Then** le mot de passe est saisi sans écho et l'administrateur
   est créé dans la base persistante.
4. **Given** une instance en fonctionnement, **When** l'exploitant exécute la
   commande de sauvegarde, **Then** le fichier de sauvegarde est écrit dans le
   volume persistant, lisible par son seul propriétaire, et récupérable depuis
   l'hôte.
5. **Given** une instance avec des données, **When** l'exploitant remplace
   l'image par une version plus récente en conservant le même volume, **Then**
   les migrations s'appliquent au démarrage et toutes les données (base et
   fichiers déposés) sont conservées.
6. **Given** l'exploitant a besoin de connaître la version déployée, **When** il
   inspecte l'image ou le conteneur en cours d'exécution, **Then** la version
   est lisible sans ouvrir de session dans le conteneur.
7. **Given** une instance démarrée et sa base accessible, **When** le moteur de
   conteneurs exécute la vérification de santé intégrée à l'image, **Then** le
   conteneur est signalé « sain ».
8. **Given** une instance dont la base est devenue inaccessible, **When** la
   vérification de santé s'exécute, **Then** le conteneur est signalé « en
   mauvaise santé ».
9. **Given** un client sans session, **When** il interroge le point de santé,
   **Then** il obtient uniquement un état (sain ou non), sans aucune donnée de
   l'association, ni version, ni détail d'erreur interne.

---

### User Story 3 - Image sécurisée et légère (Priority: P1)

L'exploitant fait tourner une image qui réduit au minimum la surface d'attaque :
elle ne s'exécute pas en super-utilisateur, ne contient ni shell ni gestionnaire
de paquets ni outils superflus, et accepte d'être lancée avec un système de
fichiers en lecture seule hors du volume de données. Elle reste petite, pour
être téléchargée rapidement même sur une connexion modeste ou un petit serveur.

**Why this priority**: c'est une exigence explicite de la demande ; une instance
Frigg expose des données personnelles d'adhérents (principe VII) et est souvent
hébergée sur un matériel modeste par des bénévoles.

**Independent Test**: inspecter l'image publiée (utilisateur d'exécution,
contenu, taille, rapport d'analyse de vulnérabilités) et la lancer avec un
système de fichiers racine en lecture seule et sans privilèges supplémentaires.

**Acceptance Scenarios**:

1. **Given** l'image publiée, **When** on inspecte son utilisateur d'exécution,
   **Then** il s'agit d'un utilisateur non privilégié, jamais du
   super-utilisateur.
2. **Given** l'image publiée, **When** on tente d'y ouvrir un shell, **Then**
   aucun shell n'est disponible.
3. **Given** l'image lancée avec un système de fichiers racine en lecture
   seule, toutes les capacités système retirées et l'interdiction d'acquérir de
   nouveaux privilèges, **When** Frigg fonctionne normalement (démarrage avec
   contrôle d'écriture du répertoire des fichiers déposés, connexion, envoi
   d'e-mails, sauvegarde), **Then** aucune opération n'échoue
   faute de droits, tant que le volume de données est accessible en écriture.
4. **Given** une analyse de vulnérabilités est effectuée lors de la
   publication, **When** une vulnérabilité de gravité critique ou élevée
   corrigeable est détectée, **Then** la publication échoue et l'image n'est pas
   publiée.
5. **Given** l'image publiée, **When** on mesure sa taille, **Then** elle
   respecte le plafond défini dans les critères de succès.
6. **Given** l'image publiée, **When** on examine son contenu et son historique
   de construction, **Then** on n'y trouve aucun secret (fichier `.env`, jeton,
   identifiant du registre), aucun code source ni dépendance de construction.

---

### User Story 4 - Construire l'image en local pour déboguer (Priority: P2)

Un développeur lance une commande du gestionnaire de tâches du projet pour
construire l'image sur son poste, à partir de son code courant (y compris non
commité), sans publier quoi que ce soit ni avoir besoin d'identifiants du
registre. Il peut ensuite la lancer pour reproduire un problème propre à
l'exécution en conteneur.

**Why this priority**: utile pour diagnostiquer et pour valider une
modification de l'image avant une release, mais non indispensable pour publier
et exploiter.

**Independent Test**: sur un poste disposant d'un moteur de conteneurs, lancer
la commande, puis démarrer l'image obtenue avec la configuration de
développement (Mailpit) et se connecter à Frigg.

**Acceptance Scenarios**:

1. **Given** un poste disposant d'un moteur de conteneurs, **When** le
   développeur lance la commande de construction locale, **Then** une image
   étiquetée de façon reconnaissable comme locale (jamais comme une version
   publiée) est produite, selon la même recette que l'image publiée.
2. **Given** la commande de construction locale, **When** elle s'exécute,
   **Then** rien n'est envoyé au registre.
3. **Given** des modifications locales non commitées, **When** le développeur
   construit l'image, **Then** l'image contient ces modifications et ses
   métadonnées indiquent qu'il s'agit d'une construction de développement.
4. **Given** la liste des commandes du gestionnaire de tâches, **When** le
   développeur l'affiche, **Then** la commande de construction locale y figure
   avec une description en français.

---

### Edge Cases

- Deux releases sont créées en peu de temps : chaque version est publiée sous
  sa propre étiquette, et `latest` désigne la plus haute version stable, pas la
  dernière publiée chronologiquement (par exemple un correctif `v1.1.4` publié
  après `v1.2.0` ne déplace pas `latest`).
- Une publication est relancée manuellement pour une release déjà publiée :
  l'image est reconstruite à partir du même code et remplace la précédente sous
  la même étiquette de version ; aucune autre version n'est touchée.
- Une release est supprimée après publication : l'image déjà publiée reste dans
  le registre ; son retrait est une action manuelle du mainteneur.
- Le volume de données monté appartient à un autre utilisateur que celui de
  l'image : le démarrage échoue avec un message explicite indiquant que le
  répertoire de données n'est pas accessible en écriture, plutôt qu'un échec
  silencieux ou tardif.
- L'exploitant lance l'image sans volume : Frigg démarre, mais la documentation
  avertit que les données seront perdues à la suppression du conteneur.
- Le serveur hôte est une machine ARM (Raspberry Pi, petit VPS ARM) : l'image
  publiée fonctionne sans configuration particulière.
- Le conteneur reçoit un signal d'arrêt : Frigg s'arrête proprement
  (connexions en cours terminées, base fermée). Frigg laisse jusqu'à 10 s aux
  requêtes en cours, soit autant que le délai par défaut de `docker stop` :
  la documentation recommande un délai d'arrêt de 20 s (`--stop-timeout 20`,
  `stop_grace_period: 20s` en Compose ; les 30 s par défaut de Kubernetes
  suffisent).
- La construction locale est lancée sans moteur de conteneurs installé : la
  commande échoue avec un message explicite.
- Le serveur SMTP est injoignable : le point de santé reste « sain », car la
  file d'e-mails réessaie d'elle-même ; seule l'indisponibilité de la base
  rend l'instance « en mauvaise santé ».
- Le point de santé est interrogé toutes les quelques secondes : les journaux
  ne sont pas saturés par ces appels et le point de santé ne consomme pas le
  quota de limitation de débit des parcours d'authentification.
- Pendant le démarrage (migrations en cours), le point de santé ne répond pas
  encore « sain » ; la vérification de santé de l'image laisse un délai de
  démarrage avant de compter les échecs.

## Requirements *(mandatory)*

### Functional Requirements

**Publication**

- **FR-001**: La création d'une release GitHub MUST déclencher automatiquement
  la construction et la publication de l'image, sans action manuelle
  supplémentaire.
- **FR-002**: L'image MUST être construite à partir du code exact de
  l'étiquette de la release.
- **FR-003**: L'image MUST être publiée dans le registre de conteneurs de
  GitHub sous le nom `balsakup/frigg`, avec pour étiquette la version sans
  préfixe `v`.
- **FR-004**: Pour une release stable (non préversion), l'étiquette `latest`
  MUST désigner l'image de la plus haute version stable publiée. Une préversion
  MUST NOT déplacer `latest`.
- **FR-005**: La publication MUST échouer sans rien publier si l'étiquette de la
  release ne respecte pas le format `vX.Y.Z` (suffixe de préversion admis).
- **FR-006**: La publication MUST être précédée de la compilation et des tests
  automatisés du projet ; un échec MUST empêcher toute publication.
- **FR-007**: L'image MUST être publiée pour les architectures `amd64` et
  `arm64` sous une même étiquette.
- **FR-008**: L'image MUST porter des métadonnées standard lisibles sans la
  lancer : version, révision du code source, date de construction, adresse du
  dépôt, licence (AGPL-3.0) et description.
- **FR-009**: L'image publiée MUST être accompagnée d'une attestation de
  provenance (qui l'a construite, à partir de quelle révision) et d'un
  inventaire de ses composants, vérifiables par l'exploitant.
- **FR-010**: L'image MUST être téléchargeable sans authentification.
- **FR-011**: Les identifiants nécessaires à la publication MUST être limités au
  strict nécessaire (écriture dans le registre du dépôt uniquement) et MUST NOT
  apparaître dans les journaux de construction ni dans l'image.

**Sécurité et légèreté de l'image**

- **FR-012**: L'image MUST s'exécuter sous un utilisateur non privilégié avec
  un identifiant numérique fixe et documenté.
- **FR-013**: L'image MUST contenir uniquement le binaire Frigg et les
  fichiers indispensables à son exécution (certificats d'autorités de
  certification pour le SMTP en TLS, données de fuseaux horaires, licence) :
  ni shell, ni gestionnaire de paquets, ni outils de compilation, ni code
  source.
- **FR-014**: L'image MUST fonctionner avec un système de fichiers racine en
  lecture seule, sans aucune capacité système et sans possibilité d'acquérir de
  nouveaux privilèges ; seul le répertoire de données de l'instance a besoin
  d'être accessible en écriture.
- **FR-015**: La publication MUST inclure une analyse de vulnérabilités de
  l'image ; une vulnérabilité de gravité critique ou élevée pour laquelle un
  correctif existe MUST bloquer la publication.
- **FR-016**: Les images de base utilisées pour construire l'image MUST être
  épinglées sur une empreinte précise, afin que deux constructions du même code
  utilisent les mêmes composants.
- **FR-017**: Le contexte de construction MUST exclure les fichiers sensibles
  ou inutiles (`.env`, bases et données locales, binaires compilés, dépendances
  installées, historique git).
- **FR-018**: L'image MUST rester sous le plafond de taille fixé par SC-003.

**Exécution**

- **FR-019**: Lancée sans argument, l'image MUST démarrer le serveur Frigg ;
  les autres commandes du binaire (création d'administrateur, sauvegarde,
  export du schéma) MUST rester accessibles en passant leur nom en argument.
- **FR-020**: L'image MUST fournir des valeurs par défaut pour le chemin de la
  base, le répertoire des fichiers déposés (tous deux sous un unique
  répertoire de données déclaré comme volume) et l'adresse d'écoute ; toute
  autre configuration passe par les variables d'environnement existantes,
  qui priment sur ces valeurs par défaut.
- **FR-021**: L'image MUST NOT embarquer de fichier `.env` ni de valeur de
  configuration propre à une instance (URL publique, SMTP, expéditeur).
- **FR-022**: L'image MUST exposer le port d'écoute par défaut et transmettre
  les signaux d'arrêt au binaire pour un arrêt propre.
- **FR-023**: Lorsque le répertoire de données n'est pas accessible en
  écriture, le démarrage MUST échouer avec un message explicite en français.

**Santé**

- **FR-028**: Frigg MUST exposer un point de santé accessible sans session, qui
  répond « sain » lorsque le serveur est démarré et que la base est accessible
  en lecture, et « en mauvaise santé » sinon, avec un code de statut HTTP
  distinct pour chaque cas.
- **FR-029**: La réponse du point de santé MUST NOT contenir de donnée de
  l'association, de version, de configuration ni de détail d'erreur interne ;
  la cause d'un échec est consignée dans les journaux du serveur uniquement.
- **FR-030**: Le point de santé MUST accepter uniquement la lecture et ne
  modifier aucun état.
- **FR-031**: Les appels réussis au point de santé MUST NOT être journalisés au
  niveau `info` ; un échec MUST l'être au niveau `warn` ou supérieur.
- **FR-032**: L'image MUST déclarer une vérification de santé qui interroge le
  point de santé sans recourir à un shell ni à un outil absent de l'image, avec
  un délai de démarrage, un intervalle et un nombre d'essais documentés.
- **FR-033**: Avant l'implémentation, la constitution MUST être amendée
  (version MINOR) pour ajouter le point de santé à la liste explicite des
  routes publiques du principe II ; la route MUST figurer dans la liste
  centralisée des exceptions dans le code et être couverte par les tests
  d'autorisation (accès anonyme accepté, aucune donnée exposée).

**Construction locale**

- **FR-024**: Le gestionnaire de tâches du projet MUST offrir une commande,
  décrite en français, qui construit l'image en local à partir du code courant,
  selon la même recette que l'image publiée, pour l'architecture du poste.
- **FR-025**: La commande de construction locale MUST NOT publier quoi que ce
  soit ni exiger d'identifiants du registre, et l'image produite MUST porter
  une étiquette locale distincte de toute version publiée.

**Documentation**

- **FR-026**: La documentation du projet MUST décrire le déploiement à partir
  de l'image : variables obligatoires, volume de données et propriétaire
  attendu, création du premier administrateur, sauvegarde et restauration,
  mise à jour vers une nouvelle version, et options de durcissement
  recommandées (lecture seule, capacités retirées).
- **FR-027**: La documentation MUST décrire la procédure de release pour le
  mainteneur et la commande de construction locale pour les développeurs.

### Key Entities

- **Release** : version sémantique, statut stable ou préversion, révision du
  code source ; déclenche une publication.
- **Image publiée** : nom (`balsakup/frigg`), étiquettes (version, `latest`
  éventuellement), architectures, métadonnées, attestation de provenance,
  inventaire des composants, rapport d'analyse de vulnérabilités.
- **Image locale** : même recette que l'image publiée, étiquette locale,
  métadonnées de développement, jamais publiée.
- **Répertoire de données de l'instance** : volume persistant contenant la base
  SQLite, le répertoire des fichiers déposés et, le cas échéant, les
  sauvegardes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Entre la création d'une release et la disponibilité de l'image
  dans le registre, il s'écoule moins de 20 minutes, sans aucune intervention
  du mainteneur.
- **SC-002**: Un exploitant qui suit la documentation déploie une instance
  fonctionnelle (serveur démarré, premier administrateur créé, connexion
  réussie) en moins de 15 minutes à partir d'un serveur disposant d'un moteur
  de conteneurs.
- **SC-003**: La taille compressée de l'image ne dépasse pas la taille
  compressée du binaire Frigg de plus de 5 Mo, et reste sous 50 Mo.
- **SC-004**: 100 % des images publiées s'exécutent sous un utilisateur non
  privilégié, sans shell, et ne présentent aucune vulnérabilité critique ou
  élevée corrigeable au moment de leur publication.
- **SC-005**: Le parcours complet (démarrage avec contrôle d'écriture du
  répertoire des fichiers déposés, connexion, envoi d'e-mail, sauvegarde)
  fonctionne à 100 % avec un système de fichiers racine
  en lecture seule et sans aucune capacité système.
- **SC-006**: Une mise à jour d'une version à la suivante en conservant le
  volume ne perd aucune donnée (base et fichiers déposés identiques avant et
  après, hors migrations attendues).
- **SC-007**: Un développeur obtient une image locale avec une seule commande,
  en moins de 5 minutes sur un poste de développement courant après une
  première construction.
- **SC-008**: Aucune release dont les tests échouent n'aboutit à une image
  publiée.
- **SC-009**: Un conteneur lancé avec une configuration valide est signalé
  « sain » en moins de 30 secondes après son démarrage, et une base devenue
  inaccessible le fait signaler « en mauvaise santé » en moins de 2 minutes.

## Assumptions

- Le registre est celui de GitHub (`ghcr.io`), rattaché au dépôt
  `Balsakup/frigg` ; le nom complet de l'image est donc
  `ghcr.io/balsakup/frigg:<version>`. Le paquet est public, cohérent avec la
  licence AGPL-3.0 du projet.
- Le déclencheur est la publication d'une release GitHub (et non le simple
  envoi d'une étiquette) ; le mainteneur crée l'étiquette au moment de créer la
  release. Une relance manuelle de la publication pour une release existante
  est possible.
- Les seules étiquettes publiées sont la version complète et `latest` ; les
  étiquettes flottantes du type `1` ou `1.2` sont hors périmètre de cette
  version.
- Les tests exécutés avant publication sont les contrôles automatisés rapides
  exigés par la constitution (compilation `CGO_ENABLED=0`, vet, tests Go,
  typecheck, lint et tests du frontend). Les tests de bout en bout
  Playwright, plus lourds, sont supposés exécutés avant la fusion sur `main` et
  ne sont pas rejoués lors de la release.
- La publication des binaires seuls en pièces jointes de la release est hors
  périmètre ; seule l'image de conteneur est publiée.
- L'image n'ayant ni shell ni outil HTTP, la vérification de santé est
  effectuée par le binaire Frigg lui-même (par exemple une commande dédiée qui
  interroge le point de santé local) ; le mécanisme exact est choisi au plan.
- Le point de santé ne vérifie que la disponibilité du serveur et de la base :
  le SMTP et l'espace disque ne sont pas pris en compte.
- Dépendance : l'amendement de la constitution exigé par FR-033 a été fait le
  2026-11-02 (version 1.5.0, principe II).
- Aucun parcours de dépôt de fichier n'existe encore dans Frigg : les
  scénarios vérifient seulement que le répertoire des fichiers déposés est
  créé et accessible en écriture. Le dépôt lui-même sera couvert par la
  fonctionnalité qui l'introduira.
- Aucune commande `frigg version` n'est ajoutée : la version est portée par les
  métadonnées de l'image (FR-008), lisibles sans lancer de session dans le
  conteneur. Une telle commande pourra faire l'objet d'une évolution ultérieure.
- La terminaison TLS est assurée par un reverse proxy devant le conteneur ;
  l'image écoute en HTTP sur le port interne. `FRIGG_TRUSTED_PROXIES` permet
  déjà de lire l'adresse du client.
- Le fichier `compose.yaml` actuel reste réservé au développement (Mailpit) ;
  un exemple de déploiement en production peut figurer dans la documentation
  sans modifier ce fichier.
- La construction locale cible l'architecture du poste du développeur ; la
  construction multi-architecture n'est faite que lors de la publication.
- Le format de licence, le CLA et les conventions de commit existants ne sont
  pas modifiés par cette fonctionnalité.
