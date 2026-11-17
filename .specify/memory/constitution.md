# Frigg Constitution

## Core Principles

### I. Une association, une instance

Frigg gère **une seule association par instance**.

- Le modèle de données MUST NOT contenir de notion de locataire (pas de colonne
  `association_id`, pas de filtrage multi-organisation).
- Les paramètres de l'association (nom, logo, couleurs, saisons…) sont des
  données singleton de l'instance.
- Héberger plusieurs associations signifie déployer plusieurs instances, chacune
  avec son propre binaire, sa propre base SQLite et son propre répertoire de
  données.

**Rationale** : supprimer le multi-tenant élimine une classe entière de fuites de
données entre organisations et garde le code et les requêtes simples.

### II. Refus par défaut, exceptions publiques énumérées

L'application entière est protégée : refus par défaut. Une personne accède aux
données par une session ou, pour un périmètre borné, par un lien personnel.

- Toute route de l'API et toute page du frontend MUST exiger une session valide,
  sauf :
  - les parcours qui permettent d'en obtenir une : connexion, réinitialisation
    du mot de passe, vérification de l'adresse électronique, acceptation d'une
    invitation envoyée par un administrateur, création d'un compte à partir
    d'une adresse confirmée par un parcours public. Il n'y a jamais
    d'inscription libre : un tel compte naît seulement d'une demande
    d'adhésion ou d'une inscription confirmée ;
  - le point de santé, en lecture seule, destiné aux vérifications de santé du
    conteneur, d'un orchestrateur ou d'un reverse proxy. Il indique uniquement
    si l'instance peut servir des requêtes et MUST NOT exposer de donnée de
    l'association, de version, de configuration ni de détail d'erreur interne ;
  - l'adresse de réception des notifications du prestataire de paiement en
    ligne (HelloAsso, voir principe V). Elle accepte uniquement les requêtes
    `POST`, est exemptée de la protection CSRF faute d'origine navigateur, et
    son seul effet est de déclencher une vérification de l'état du paiement
    auprès du prestataire : le contenu reçu MUST NOT suffire à constater un
    paiement ni à modifier une donnée sans cette vérification. L'adresse porte
    une clé dérivée du secret d'accès au prestataire, connue seulement de lui
    et de l'instance :
    - toute requête sans la clé valide (absente, fausse ou mal formée) reçoit
      une réponse identique (`200`, corps vide), sans traitement ni appel au
      prestataire ;
    - une requête portant la clé valide reçoit la réponse attendue par le
      protocole du prestataire : `200` lorsque la notification est traitée ou
      ignorée, `503` en cas d'erreur passagère de Frigg ou d'indisponibilité
      du prestataire pendant la vérification, afin qu'il la renvoie ;
    - dans tous les cas, la réponse MUST NOT révéler de donnée de
      l'association, de version, de configuration ni de détail d'erreur
      interne ;
  - l'**identité visuelle** de l'association, en lecture seule (`GET`), pour
    habiller les écrans hors session ci-dessus. Elle se limite au nom, aux
    couleurs de l'association et au fichier du logo :
    - aucune autre donnée de l'association (tagline, objet, identité légale,
      coordonnées, réseaux sociaux, saisons, tarifs, membres…) ni aucune donnée
      de version ou de configuration MUST NOT être ajoutée à cette réponse ; une
      extension de son contenu exige un amendement ;
    - le logo est servi sous son nom généré, avec son type validé, sans révéler
      son nom d'origine ni les métadonnées de dépôt ;
  - les **parcours publics d'adhésion et d'inscription**, chacun limité à ce
    qui suit :
    - la **page d'adhésion publique**, en lecture seule. Elle est servie
      seulement si un administrateur a activé l'accueil des demandes
      publiques, sinon elle se limite à un message de fermeture. Elle se
      limite à l'identité visuelle, aux saisons ouvertes non closes portant un
      tarif, à leurs tarifs (libellé, montants, description) et aux
      définitions des champs personnalisés remplis par le membre ;
    - la **page publique d'un événement**, en lecture seule, atteinte
      uniquement par un lien non devinable, qu'un administrateur diffuse et
      peut régénérer. Elle n'existe que pour un événement non brouillon dont
      l'ouverture est « Ouvert au public », et se limite à ses titre, dates,
      lieu, description, prix, période d'inscription et places restantes.
      Aucun inscrit n'y figure, et tout autre cas reçoit la même réponse qu'un
      lien inconnu. Aucun listing public des événements n'est autorisé ;
    - le **dépôt** d'une demande d'adhésion ou d'une inscription. Il ne
      produit aucun effet visible (ni fiche, ni adhésion, ni inscription, ni
      place réservée) tant que l'adresse électronique saisie n'a pas été
      confirmée par le lien reçu à cette adresse ;
    - la **confirmation** de l'adresse d'un dépôt, par ce lien ;
    - la **demande d'un lien personnel**, dont la réponse est toujours la
      même ;
- Le **lien personnel** est la seule forme d'accès sans session à des données
  de membres :
  - c'est un secret aléatoire, envoyé uniquement par e-mail, à durée de vie
    limitée, révocable et stocké haché ;
  - il donne accès à un périmètre fermé : les fiches sans compte dont
    l'adresse électronique est celle pour laquelle il a été émis, et leurs
    adhésions et inscriptions. Ce périmètre est réévalué à chaque requête ;
  - il permet seulement de consulter ce périmètre, d'annuler une demande ou
    une inscription selon les règles applicables aux comptes, de choisir un
    moyen de paiement, de lancer un paiement par carte et de créer un
    compte. Il MUST NOT permettre de modifier une identité, des coordonnées
    ou une valeur de champ, ni de constater un paiement ;
  - toute donnée hors de ce périmètre reçoit la même réponse qu'une donnée
    inexistante.
- Toute réponse d'un parcours public MUST NOT révéler de donnée de
  l'association, d'un membre, de version ou de configuration au-delà de ce que
  la liste ci-dessus autorise ; une extension de son contenu exige un
  amendement.
- Les pages publiques MUST rester exclues de l'indexation par les moteurs de
  recherche.
- Cette liste d'exceptions est explicite et centralisée dans le code ; une
  nouvelle route publique MUST faire l'objet d'un amendement de cette
  constitution.
- Le contrôle est appliqué côté serveur (middleware) ; masquer un élément dans
  l'interface ne constitue jamais une protection.
- Les fichiers statiques du frontend embarqué peuvent être servis sans session
  (le shell de connexion doit pouvoir se charger), mais ils MUST NOT contenir de
  données de l'association : l'identité visuelle est obtenue à l'exécution par
  la route ci-dessus. Les fichiers déposés, logo excepté, sont servis
  uniquement à une session autorisée.

**Rationale** : les données d'une association (membres, coordonnées, paiements)
sont personnelles ; aucune ne doit être lisible sans identification. Le nom, le
logo et les couleurs sont en revanche des informations que l'association affiche
publiquement sur tous ses supports : les montrer sur l'écran de connexion ne
divulgue rien, et la liste fermée de ces champs empêche toute dérive. De même,
les tarifs d'une saison ouverte et un événement que l'association choisit
d'ouvrir au public sont des informations qu'elle diffuse elle-même. Une
personne extérieure doit pouvoir adhérer ou s'inscrire sans qu'un
administrateur lui crée d'abord un accès. La confirmation de l'adresse
empêche qu'un dépôt laisse une trace exploitable. Le lien personnel s'appuie
sur la possession de la messagerie, comme la réinitialisation du mot de
passe, et son périmètre fermé borne ce qu'il expose.

### III. Deux rôles : l'administrateur est un membre

Il existe exactement deux rôles : **membre** et **administrateur**.

- Un administrateur est un membre doté de droits d'administration : tout ce
  qu'un membre peut faire, un administrateur MUST pouvoir le faire.
- Les permissions sont modélisées comme une hiérarchie (`admin ⊇ member`) et vérifiées par une fonction d'autorisation unique côté serveur ;
  les vérifications ad hoc de rôle dispersées dans les handlers sont interdites.
- Chaque opération de l'API déclare son niveau d'accès : public (exceptions du
  principe II), lien personnel, membre ou administrateur. Une opération sans
  niveau déclaré est un défaut bloquant.
- L'accès par lien personnel (principe II) n'est pas un rôle. Il est vérifié
  par la même fonction d'autorisation, n'ouvre aucune rubrique de l'espace
  membre ni de l'administration, et une session de membre n'élargit pas le
  périmètre d'une opération par lien.
- Le premier administrateur est créé exclusivement par la commande CLI
  `frigg create-admin`.

**Rationale** : un modèle à deux niveaux strictement inclusifs est simple à
raisonner, à tester et à expliquer aux bénévoles qui administrent l'association.

### IV. Un seul shell, un seul design system

Le layout est identique pour un membre et pour un administrateur ; seuls les
éléments de menu changent selon le rôle.

- Le rendu de référence est l'artefact
  <https://claude.ai/artifact/8i3r6G5Lmo2LrppTLzUQcQ> : barre latérale fixe
  repliable en icônes, en-tête avec recherche globale et notifications, fil
  d'Ariane, en-tête de page (titre, introduction, actions), pied de page légal.
- Les composants (boutons, alertes, badges, champs, tableaux, dialogues de
  confirmation, notifications) MUST reprendre les variantes et les tokens de
  couleur de cet artefact (`--brand`, `--accent`, `--ink`, `--muted`, `--ok`,
  `--warn`, `--danger`, `--info`…) ; aucune couleur codée en dur hors tokens.
- Les composants interactifs (dialogues, menus, listes déroulantes, combobox,
  popovers, notifications) MUST être construits sur les primitives accessibles de
  `base-ui`, habillées avec les tokens Tailwind ; on ne réimplémente pas un motif
  ARIA à la main quand base-ui le fournit.
- Les icônes proviennent de `lucide-react` via un composant `Icon` unique
  (`strokeWidth` 1,8, `aria-hidden` par défaut).
- Les polices (Atkinson Hyperlegible, Bricolage Grotesque) sont auto-hébergées via
  Fontsource et intégrées au binaire. L'application servie aux utilisateurs ne
  charge aucune ressource depuis un CDN tiers. Seule exception : la
  documentation d'API de développement, désactivée par défaut et réservée aux
  administrateurs, qui peut charger son interface depuis un CDN avec une version
  épinglée.
- Le thème est clair uniquement. Les couleurs de l'association sont lues depuis
  l'API et appliquées en variables CSS, sans bibliothèque de thème ; les teintes
  de texte dérivées MUST garantir un contraste d'au moins 4,5:1. Les couleurs
  sémantiques restent fixes.
- L'accessibilité vise la conformité **RGAA** : liens d'évitement, navigation
  au clavier, focus visible, rôles ARIA corrects, `prefers-reduced-motion`
  respecté. Une régression d'accessibilité est un défaut bloquant.
- L'interface est rédigée en français ; elle fonctionne dès 320 px de large.
- Les actions destructives MUST passer par un dialogue de confirmation.

**Rationale** : un shell unique évite deux applications à maintenir, et des
utilisateurs bénévoles de tous âges ont besoin d'une interface lisible et
prévisible.

### V. Binaire unique en Go pur

Frigg se livre sous la forme d'un **binaire unique**, compilé avec
`CGO_ENABLED=0`.

- Aucune dépendance nécessitant CGO n'est admise (d'où `modernc.org/sqlite`).
  Une dépendance dont le `go.mod` référence un paquet CGO est acceptée seulement
  si ce paquet n'est pas compilé dans le binaire (vérifié par
  `CGO_ENABLED=0 go list -deps`).
- Le frontend compilé est intégré au binaire via `embed` (`web/embed.go`) ; le
  binaire ne lit aucun fichier d'interface sur le disque à l'exécution.
- Les migrations `goose` sont embarquées et appliquées automatiquement au
  démarrage ; un échec de migration MUST empêcher le démarrage du serveur.
- Les seules dépendances d'exécution externes autorisées sont : le fichier
  SQLite, le répertoire de données des fichiers déposés, le fichier `.env`
  (facultatif), un serveur SMTP et, de façon facultative, l'API de paiement
  en ligne **HelloAsso**.
- HelloAsso est activé uniquement si ses identifiants sont fournis par la
  configuration (variables d'environnement) ; ils ne sont jamais saisis dans
  l'interface. Leur absence MUST NOT empêcher le démarrage, et une
  indisponibilité de HelloAsso MUST NOT empêcher les autres fonctions de
  Frigg : seul le paiement par carte devient indisponible. Le secret d'accès
  MUST NOT être stocké en base, renvoyé au navigateur ni journalisé.
- La commande `frigg backup` produit une sauvegarde cohérente à chaud de la base
  (`VACUUM INTO`) ; la sauvegarde complète d'une instance = ce fichier + le
  répertoire de données.

**Rationale** : une association n'a pas d'équipe d'exploitation ; installer,
mettre à jour et sauvegarder Frigg doit rester trivial.

### VI. Contrat d'API typé et SQL généré

- L'API REST est définie avec `huma/v2` (adaptateur `humago`) sur `net/http`.
  Le schéma OpenAPI généré (`frigg openapi`) est la source de vérité du contrat
  entre backend et frontend ; le frontend MUST en générer ses types
  (`openapi-typescript`) et ne jamais les redéfinir à la main.
- La validation des entrées est déclarée dans les types huma ; aucune donnée non
  validée n'atteint la couche base de données. Le serveur fait autorité : le
  frontend MUST NOT dupliquer ces règles dans des schémas écrits à la main. Il
  s'appuie sur les attributs HTML natifs et affiche les erreurs 422 de huma sur
  les champs concernés (`errors[].location`). Si un schéma client devient
  nécessaire, il est généré depuis l'OpenAPI.
- Tout accès SQL passe par des requêtes écrites dans `internal/database/queries`
  et générées par `sqlc` dans `internal/database/sqlc`. Le schéma lu par sqlc
  est celui des migrations goose. Pas d'ORM, pas de SQL construit par
  concaténation de chaînes.
- Le code généré (`sqlc`, OpenAPI) est commité et MUST être à jour : le
  régénérer ne doit produire aucun diff.
- **Recherche insensible à la casse et aux accents** :
  - côté base, la recherche textuelle MUST utiliser des tables **FTS5 à contenu
    externe** (`content='<table>'`, `content_rowid='id'`) synchronisées par des
    triggers définis dans les migrations. Aucune colonne « repliée » dupliquée
    n'est ajoutée aux tables métier ;
  - le tokenizer dépend de la nature du champ indexé :
    - **champs d'identité courts** (adresses électroniques, noms, prénoms,
      téléphones, identifiants) : `tokenize="trigram remove_diacritics 1"`,
      qui trouve toute sous-chaîne d'au moins 3 caractères ; en dessous de
      3 caractères, l'interface MUST indiquer le minimum requis plutôt que
      d'afficher une liste vide ;
    - **texte long en langage naturel** (descriptions, messages, contenus) :
      `tokenize="unicode61 remove_diacritics 2"`, recherche par mots et
      préfixes ;
    - une table FTS5 n'indexe que des champs d'une même catégorie ;
  - côté Go, les comparaisons de texte saisi (unicité, tri, rapprochement)
    MUST passer par le package `internal/textfold`.

**Rationale** : le typage de bout en bout (SQL → Go → OpenAPI → frontend) fait
échouer les incohérences à la compilation plutôt qu'en production ; FTS5 offre
la recherche sans accents sans dénormaliser le schéma. Sur une adresse ou un
nom, on tape un fragment (« pont », « lefe ») : trigram le trouve, alors que
unicode61, adapté au texte long, ne trouve que des débuts de mots.

### VII. Sécurité des comptes et des données personnelles

- Les mots de passe sont hachés avec **argon2id** via `alexedwards/argon2id`,
  avec des paramètres centralisés en un seul endroit ; ils ne sont jamais
  stockés, journalisés ni renvoyés en clair. La saisie en CLI passe par
  `golang.org/x/term` (sans écho).
- Les sessions sont gérées par `scs/v2` avec stockage `sqlite3store` ; la table
  `sessions` est créée par une migration goose. Les cookies MUST être `Secure`,
  `HttpOnly` et `SameSite=Lax` (ou plus strict). La session est renouvelée à la
  connexion et détruite à la déconnexion.
- **CSRF** : toutes les requêtes modifiant l'état MUST traverser
  `http.CrossOriginProtection` (bibliothèque standard), en plus de `SameSite`.
- **Limitation de débit** : la connexion, la demande de réinitialisation et le
  renvoi du lien de vérification MUST être limités par IP et par compte via
  `golang.org/x/time/rate` ; un dépassement renvoie `429` sans révéler si le
  compte existe. Les dépôts publics, les confirmations d'adresse, les demandes
  de lien personnel et les accès par lien personnel MUST être limités de même,
  par IP et par adresse électronique visée. Le nombre d'e-mails envoyés à une
  même adresse par ces parcours MUST être plafonné, pour qu'un tiers ne puisse
  pas en inonder une messagerie.
- Les jetons envoyés par e-mail sont aléatoires, à durée de vie limitée et
  stockés hachés. Ceux de vérification, de réinitialisation, d'invitation et
  de confirmation d'un dépôt public sont à usage unique. Le lien personnel
  (principe II) reste valable jusqu'à son expiration ou sa révocation, et il
  est révoqué dès qu'un compte est créé pour son adresse.
- Les réponses aux parcours d'authentification et aux parcours publics MUST NOT
  révéler si une adresse électronique possède un compte, une fiche ou une
  demande.
- Les formulaires publics MUST comporter un piège à robots invisible des
  personnes, y compris au lecteur d'écran. Ils MUST NOT dépendre d'un service
  tiers de vérification (captcha), qui transmettrait des données des
  visiteurs à un tiers.
- Les journaux MUST NOT contenir de mots de passe, de jetons, de cookies de
  session ni le contenu des e-mails.
- Les fichiers déposés sont stockés sous un nom généré (jamais le nom fourni par
  l'utilisateur), leur type et leur taille sont validés, et tout chemin est
  résolu à l'intérieur du répertoire de données (pas de traversée `..`).
- Les données personnelles des membres sont traitées dans le respect du RGPD :
  ne collecter que le nécessaire et permettre leur export et leur suppression.
  Un formulaire public MUST recueillir un consentement explicite, dont la date
  est conservée. Les données d'un dépôt non confirmé MUST être supprimées
  automatiquement dans le délai fixé par sa spécification.

**Rationale** : Frigg stocke des données personnelles de bénévoles et
d'adhérents ; une fuite engage la responsabilité de l'association. Ouvrir des
formulaires sans session expose l'instance aux robots, à l'énumération
d'adresses et à l'usage de Frigg pour harceler une messagerie : ces règles en
bornent chaque effet.

## Contraintes techniques

**Backend (Go 1.27)** — liste fermée ; ajouter une dépendance directe exige une
justification dans le plan de la fonctionnalité.

| Rôle | Choix |
| --- | --- |
| Serveur HTTP et routage | `net/http` (bibliothèque standard) |
| API REST typée, validation, OpenAPI | `huma/v2` + adaptateur `humago` |
| Tests d'API | `humatest` (fourni avec huma) + `net/http/httptest` |
| Base de données | SQLite via `modernc.org/sqlite` (Go pur, FTS5 inclus) |
| Accès aux données | `sqlc` (`internal/database/queries` → `internal/database/sqlc`) |
| Migrations | `goose/v3`, embarquées, appliquées au démarrage |
| Sessions | `scs/v2` + `scs/sqlite3store` |
| Hachage des mots de passe | `alexedwards/argon2id` |
| Protection CSRF | `http.CrossOriginProtection` (bibliothèque standard) |
| Limitation de débit | `golang.org/x/time/rate` |
| Journaux | `log/slog` (bibliothèque standard), sortie JSON |
| Configuration | fichier `.env` chargé par `joho/godotenv` |
| E-mails | `go-mail` via SMTP, depuis une table d'attente |
| Saisie de mot de passe en CLI | `golang.org/x/term` |
| Normalisation Unicode (sans accents) | `golang.org/x/text/unicode/norm` dans `internal/textfold` |
| Frontend embarqué | `embed` (`web/embed.go`) |

**Commandes du binaire** : au minimum le serveur, `frigg openapi` (export du
schéma), `frigg create-admin` (création d'un administrateur) et `frigg backup`
(sauvegarde de la base).

**Exploitation de SQLite** :

- Chaque connexion active `journal_mode=WAL`, `busy_timeout`, `foreign_keys=ON`
  et `synchronous=NORMAL` via les paramètres `_pragma=` de la chaîne de
  connexion.
- Les écritures passent par un pool limité à une connexion
  (`SetMaxOpenConns(1)`) ; les lectures utilisent un pool séparé.
- Les transactions d'écriture démarrent en `BEGIN IMMEDIATE`.

**E-mails** : aucun e-mail n'est envoyé de façon synchrone dans une requête HTTP.
Les messages sont insérés dans une table d'attente (dans la même transaction que
l'action qui les déclenche) et dépilés par un worker périodique interne au
binaire, à intervalle configurable, avec nouvelles tentatives et état
(`pending`, `sent`, `failed`) consultable.

**Fichiers** : les fichiers déposés (logo, documents, attestations) sont stockés
sur disque dans un répertoire de données configurable ; la base ne conserve que
leurs métadonnées (nom généré, nom d'origine, type, taille, propriétaire).

**Configuration** : toute la configuration passe par des variables
d'environnement, éventuellement chargées depuis un fichier `.env`. Les variables
déjà présentes dans l'environnement priment sur le `.env`. Le démarrage MUST
échouer avec un message explicite si une variable obligatoire manque ; une
variable facultative absente (identifiants HelloAsso, par exemple) désactive
seulement la fonction qui en dépend. Un fichier
`.env.example` documente chaque variable ; le `.env` n'est jamais commité.

**Journaux** : `slog` avec un handler JSON sur la sortie standard ; chaque requête
HTTP porte un identifiant de requête repris dans ses lignes de journal.

**Frontend** (dossier `web/`, gestionnaire de paquets **Bun**) — liste fermée,
mêmes règles d'ajout que pour le backend.

| Rôle | Choix |
| --- | --- |
| Framework et routage | React 19 + React Router v8 en mode SPA (`ssr: false`) |
| Styles | Tailwind CSS v4 (tokens de l'artefact déclarés dans `@theme`) |
| Composition de classes | `clsx` + `tailwind-merge` (fonction `cn` locale) |
| Types de l'API | `openapi-typescript` (depuis `frigg openapi`) |
| Client HTTP | `openapi-fetch` |
| Données serveur | `@tanstack/react-query` + `openapi-react-query` |
| Formulaires | `react-hook-form` |
| Composants accessibles | `@base-ui/react` |
| Icônes | `lucide-react` |
| Polices | Fontsource (Atkinson Hyperlegible, Bricolage Grotesque) |
| Lint et formatage | Biome |
| Tests unitaires et composants | Vitest + Testing Library |
| Tests de bout en bout et accessibilité | Playwright + `@axe-core/playwright` |

Règles :

- **Build** : `bun run build` produit `web/build/client`, seul répertoire intégré
  par `web/embed.go`. Le serveur Go sert `index.html` pour toute route du SPA
  inconnue côté API. Aucun runtime Node n'est utilisé en production.
- **Données** : les `clientLoader` préchargent via
  `queryClient.ensureQueryData` ; les composants lisent avec `useQuery` ; chaque
  mutation invalide explicitement les requêtes concernées. Aucun appel `fetch`
  direct hors du client `openapi-fetch`.
- **Formulaires** : `react-hook-form` ; un adaptateur unique convertit les
  erreurs 422 de huma en `setError` (voir principe VI).
- **Recherche globale** (Ctrl K) : Combobox de base-ui alimenté par l'API (FTS5),
  sans filtrage côté client.
- **Développement** : le serveur Vite redirige `/api` vers le serveur Go afin
  que le navigateur reste sur une seule origine (cookies et CSRF identiques à la
  production).

## Conventions de langue

Frigg est destinée uniquement à des associations françaises ; aucune
internationalisation n'est prévue.

- **Anglais** pour les identifiants : variables, fonctions, types, packages,
  fichiers, noms de tables et de colonnes, chemins et champs JSON de l'API,
  valeurs d'énumération (`admin`, `member`, `pending`…), codes d'erreur, clés de
  configuration et de journal.
- **Français** pour tout le reste : commentaires, textes de l'interface,
  e-mails, messages d'erreur renvoyés par l'API, messages de journal, sorties
  des commandes CLI, descriptions des tests, et la documentation du projet
  (constitution, spécifications, plans).

## Workflow de développement et contrôles qualité

- Chaque fonctionnalité suit le cycle Spec Kit : spécification → plan → tâches
  → implémentation. Le plan MUST inclure une vérification explicite de
  conformité à cette constitution (« Constitution Check »).
- Avant toute fusion, les contrôles suivants MUST passer :
  - `go build` avec `CGO_ENABLED=0` ;
  - `go vet ./...` et `go test ./...` ;
  - `sqlc generate`, `frigg openapi` et la génération des types
    `openapi-typescript` sans diff ;
  - les migrations s'appliquent sur une base vierge ;
  - dans `web/` : `bun run typecheck`, Biome sans erreur, Vitest, et `bun run
    build` ;
  - Playwright avec axe sur les parcours touchés : aucune violation
    d'accessibilité de niveau `serious` ou `critical`.
- Les tests d'API utilisent une base SQLite isolée par test, migrée par goose :
  `humatest` pour les opérations isolées et le schéma, `httptest` sur la chaîne
  HTTP complète dès que les sessions, les cookies ou la protection CSRF sont en
  jeu (humatest appelle l'API huma sans les middlewares HTTP qui l'entourent).
- Les règles d'autorisation (principes II et III) MUST être couvertes par des
  tests automatisés :
  - pour chaque opération non publique, un accès anonyme est refusé ;
  - un membre est refusé lorsque l'opération est réservée aux administrateurs ;
  - un administrateur est accepté pour toute opération accessible aux membres ;
  - pour chaque opération par lien personnel, un lien expiré, révoqué ou
    inconnu est refusé, et une donnée hors du périmètre du lien reçoit la même
    réponse qu'une donnée inexistante ;
  - chaque opération publique ne renvoie que les données que le principe II
    autorise.
- Les migrations sont append-only : une migration déjà publiée n'est jamais
  modifiée ; on en ajoute une nouvelle. « Publiée » signifie déployée sur au
  moins une instance (y compris de recette) ; tant que le projet n'a jamais été
  déployé, les migrations existantes peuvent être modifiées plutôt que
  complétées. Le premier déploiement met fin à cette exception.
- Toute complexité qui s'écarte de ces principes est documentée dans la section
  « Complexity Tracking » du plan, avec l'alternative plus simple écartée et la
  raison.

## Governance

- Cette constitution prévaut sur toute autre pratique ou convention du projet.
  En cas de conflit entre une spécification, un plan ou du code et la
  constitution, c'est la constitution qui s'applique ou qui doit être amendée.
- **Amendement** : toute modification passe par `/speckit-constitution`, est
  décrite dans le rapport d'impact en tête de fichier, puis commitée avec un
  message `docs: amend constitution to vX.Y.Z`. Les spécifications et plans en
  cours sont réévalués au regard de l'amendement.
- **Versionnage sémantique** :
  - MAJOR : suppression ou redéfinition incompatible d'un principe ;
  - MINOR : ajout d'un principe ou d'une section, ou extension notable d'une
    règle ;
  - PATCH : clarifications et corrections de formulation sans effet sur les
    règles.
- **Conformité** : chaque revue de code vérifie le respect des principes ; une
  violation non justifiée dans « Complexity Tracking » bloque la fusion.

**Version**: 1.9.0 | **Ratified**: 2026-10-30 | **Last Amended**: 2026-11-17
