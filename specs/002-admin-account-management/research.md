# Recherche : Gestion des comptes par les administrateurs

Décisions techniques de la fonctionnalité 002. Elles prolongent celles de
[001 › research.md](../001-auth-app-shell/research.md), qui restent valables
(sessions, époque d'authentification, jetons, file d'e-mails, matrice
d'autorisation).

## R1. État « désactivé » d'un compte

- **Décision** : colonne `accounts.disabled_at TEXT NULL` ; `NULL` = actif. La
  date sert à l'affichage et tient lieu d'état, sans colonne `status` séparée.
- **Rationale** : un seul champ, aucune incohérence possible entre un état et
  une date ; même convention que `email_verified_at`.
- **Alternatives écartées** : colonne `status TEXT CHECK (...)` + date (deux
  sources de vérité) ; booléen `disabled` (perd la date demandée par la spec).

## R2. Fermer les sessions d'un compte désactivé

- **Décision** : la désactivation incrémente `auth_epoch` (mécanisme de 001 ›
  R5) et le middleware d'accès refuse en plus tout compte dont `disabled_at`
  n'est pas `NULL` (`401` « Votre session a expiré. », session détruite). La
  réactivation ne remet pas l'époque en arrière : les anciennes sessions restent
  mortes, la personne se reconnecte.
- **Rationale** : les données de session `scs` sont un blob encodé ; on ne peut
  pas supprimer les sessions d'un compte par une requête SQL. L'époque le fait
  déjà pour le changement de mot de passe ; le contrôle explicite de
  `disabled_at` protège même si une évolution future oubliait l'époque.
- **Alternatives écartées** : parcourir et décoder la table `sessions` (fragile,
  couplé au format interne de scs) ; table d'index session → compte (une table
  et une écriture de plus à chaque connexion, pour un gain nul).

## R3. Parcours d'authentification d'un compte désactivé

- **Décision** :
  - `login` vérifie toujours le mot de passe (temps constant), puis renvoie le
    même `401` « Adresse électronique ou mot de passe incorrect » si le compte
    est désactivé ;
  - `request-password-reset` ne crée ni jeton ni e-mail pour un compte
    désactivé et répond `202` comme pour une adresse inconnue ;
  - la désactivation supprime tous les jetons non utilisés du compte, et
    `GetValidToken` joint `accounts` pour ignorer les comptes désactivés
    (défense en profondeur : un lien déjà en file reste inutilisable) ;
  - `accept-invitation` et `create-invitation` traitent un compte désactivé
    comme un compte existant (`409`).
- **Rationale** : FR-007 et principe VII (aucune réponse ne révèle l'existence
  ni l'état d'un compte).

## R4. Changement de rôle

- **Décision** : `UPDATE accounts SET role` ; aucune action sur les sessions. Le
  middleware relit déjà le compte à chaque requête (001), donc le nouveau rôle
  s'applique à la requête suivante (FR-009, SC-003).
- **Frontend** : un `403` reçu sur `/api/admin/*` invalide la requête de
  session. `AppShell` mémorise le rôle précédent : s'il passe de `admin` à
  `member`, l'espace mémorisé repasse à « Membre » et la personne est envoyée
  sur `/` avec un message « Vos droits d'administration ont été retirés. ».
  Une promotion apparaît au prochain rafraîchissement de la session (focus de
  la fenêtre ou expiration de `staleTime`, 30 s).
- **Alternatives écartées** : forcer la déconnexion (contraire à FR-009) ;
  sondage périodique de la session (trafic inutile ; le serveur fait autorité).

## R5. Garde-fous : soi-même et dernier administrateur

- **Décision** :
  1. Le handler refuse toute action de désactivation, suppression ou
     changement de rôle dont la cible est le compte courant : `409`
     « Vous ne pouvez pas modifier votre propre compte depuis cette page. »
     (FR-013).
  2. Dans la transaction d'écriture (`BEGIN IMMEDIATE`, pool à une connexion),
     l'acteur est relu : s'il n'est plus administrateur actif, `403`.
  3. Les écritures qui peuvent retirer un administrateur actif (rétrogradation,
     désactivation ou suppression d'un administrateur actif) sont des requêtes
     conditionnelles : elles ne s'appliquent que s'il existe un **autre**
     administrateur actif (`EXISTS (... o.id <> cible)`). Zéro ligne modifiée →
     `409` « L'association doit conserver au moins un administrateur actif. »
     (FR-014).
- **Rationale** : les écritures sont sérialisées par le pool à une connexion et
  `BEGIN IMMEDIATE` ; la condition évaluée dans la même instruction que
  l'écriture rend la course « A désactive B pendant que B désactive A »
  impossible (SC-004). Le point 2 couvre l'acteur qui a passé le middleware
  juste avant d'être lui-même désactivé.
- **Code** : `409` plutôt que `403` pour ces refus métier, afin que le
  frontend ne les confonde pas avec un retrait de droits (R4).

## R6. Suppression définitive

- **Décision** : dans une transaction :
  - `DELETE FROM accounts` (garde-fou R5 pour un administrateur actif) ;
    `one_time_tokens` part en cascade, `invitations.invited_by` passe à `NULL`
    (contraintes existantes) ;
  - suppression des invitations **adressées** au compte (`invitations.email =
    adresse`, quel que soit leur état) : l'adresse est une donnée personnelle ;
  - suppression des e-mails de la file dont `recipient` est l'adresse.
  Les sessions disparaissent de fait : le middleware ne trouve plus le compte
  et détruit la session (001).
- **Rationale** : droit à l'effacement (principe VII) ; il ne reste aucune
  copie de l'adresse dans la base, hors sauvegardes antérieures.
- **Alternatives écartées** : anonymisation (aucune donnée métier à conserver,
  voir hypothèses de la spec) ; suppression douce (contraire au droit à
  l'effacement).

## R7. Recherche par adresse (FTS5)

- **Décision** : table `accounts_fts` FTS5 à contenu externe sur `email`
  (`content='accounts'`, `content_rowid='id'`,
  `tokenize="trigram remove_diacritics 1"`, catégorie « champ d'identité
  court » de la constitution 1.4.0), triggers `AFTER INSERT`, `AFTER DELETE`
  et `AFTER UPDATE OF email`, remplissage initial par la commande `'rebuild'`.
  La saisie est découpée en Go aux espaces ; les mots de moins de 3 caractères
  sont écartés ; chaque mot restant est entouré de guillemets (guillemets
  internes doublés) ; les mots sont combinés en ET : `exemple dup` →
  `"exemple" "dup"`. Fonction `database.FTSTrigramQuery`, testée ; elle
  renvoie aussi si la saisie contenait au moins un mot utilisable.
- **Règle des 3 caractères** : saisie vide → pas de filtre ; saisie non vide
  sans aucun mot d'au moins 3 caractères → `422` « Saisissez au moins
  3 caractères. » (`query.q`). Le frontend n'envoie pas `q` dans ce cas et
  affiche l'indication sous le champ (constitution, principe VI).
- **Comportement vérifié** (sqlite 3.51 ; modernc 1.61 embarque SQLite 3.54.1,
  ≥ 3.45 requis pour `remove_diacritics` avec trigram) : `pont` trouve
  `jean.dupont@exemple.fr` ; `n.dup` (fragment avec ponctuation) le trouve
  aussi ; `LEFÈ` trouve `hélène.lefèvre@asso.fr` ; `dupre` trouve
  `marc.dupré@…` ; `"exemple" "dup"` trouve les deux comptes `dup…@exemple.fr` ;
  un mot de syntaxe FTS5 entre guillemets (`"OR"`) n'a aucun effet ; les
  triggers suivent les écritures ; `integrity-check` passe.
- **Rationale** : principe VI 1.4.0 ; les administrateurs tapent un fragment
  d'adresse ou de nom, qui peut se trouver au milieu d'un mot ; le
  guillemetage neutralise la syntaxe FTS5 saisie par l'utilisateur ;
  l'expression est passée en paramètre, jamais concaténée au SQL.
- **Alternatives écartées** : `unicode61` (ne trouve que les débuts de mots,
  réservé au texte long par la constitution) ; `LIKE '%…%'` (sensible aux
  accents, interdit par le principe VI).

## R8. Liste paginée et filtres avec sqlc

- **Décision** : deux requêtes, `ListAccounts` et `CountAccounts`, avec filtres
  optionnels de la forme `(CAST(sqlc.narg(x) AS TEXT) IS NULL OR …)` ; le
  `CAST` donne des paramètres typés (`sql.NullString`, `sql.NullInt64`) au lieu
  de `interface{}`. Vérifié avec sqlc 1.31.1, y compris la sous-requête
  `MATCH` sur la table FTS5. Pagination par `LIMIT`/`OFFSET`, tri par `email`
  puis `id`.
- **Paramètres d'API** : `q` (≤ 100 caractères, voir la règle des
  3 caractères en R7), `role` (`member`|`admin`),
  `status` (`active`|`disabled`), `page` (≥ 1, défaut 1), `pageSize` (1 à 100,
  défaut 25). Une page au-delà de la dernière renvoie une liste vide avec le
  total.
- **Rationale** : quelques milliers de comptes au plus ; `OFFSET` reste
  instantané à cette échelle et permet d'afficher « page 3 sur 12 ».
- **Alternatives écartées** : pagination par curseur (pas de numéro de page,
  complexité inutile) ; SQL construit dynamiquement (interdit, principe VI).
- **Tri** : les adresses sont stockées normalisées (minuscules) ; le tri SQL
  binaire suffit pour des adresses ASCII. `internal/textfold` n'est pas créé :
  aucune comparaison de texte saisi n'est faite en Go dans cette
  fonctionnalité.

## R9. Envoi d'un lien de réinitialisation par un administrateur

- **Décision** : opération `admin` qui réutilise `issueToken` et le modèle
  d'e-mail `password-reset` de 001 (même lien, même durée de 1 h). Refus `409`
  si le compte est désactivé. Nouveau limiteur `AdminReset` : 30 envois par
  heure et par administrateur (aligné sur `InviteAdmin`).
- **Rationale** : un seul parcours de réinitialisation à maintenir ; le
  limiteur empêche d'utiliser Frigg pour inonder une boîte aux lettres.

## R10. Journalisation des actions (FR-016)

- **Décision** : une ligne `slog` niveau `INFO` par action réussie, message
  « action d'administration sur un compte », attributs `action`
  (`account.disable`, `account.enable`, `account.role_change`,
  `account.delete`, `account.password_reset_sent`), `actor_id`, `target_id`,
  et `role` pour un changement de rôle. Pas d'adresse électronique : après une
  suppression, les journaux ne doivent pas conserver l'adresse effacée.
  L'identifiant de requête est ajouté par le middleware existant.
- **Alternatives écartées** : table d'audit (historique consultable hors
  périmètre, voir hypothèses de la spec).

## R11. Contrat d'API

- **Décision** : ressource `/api/admin/accounts`, toutes opérations `admin`,
  tag `Comptes`. Actions sous forme de sous-ressources explicites
  (`/disable`, `/enable`, `/role`, `/password-reset`) plutôt qu'un `PATCH`
  générique : chaque action a ses propres garde-fous, messages et ligne de
  journal, et la matrice d'autorisation les teste une par une. Les actions
  idempotentes (désactiver un compte déjà désactivé, réattribuer le même rôle)
  renvoient `200` avec l'état courant, sans écrire ni journaliser. Détail dans
  [contracts/api.md](./contracts/api.md).

## R12. Interface : tableau sans bibliothèque

- **Décision** : pas de TanStack Table. Composants `ui/Table` (extrait du
  tableau des invitations : `<caption>`, `scope="col"`, `ScrollRegion` pour
  320 px) et `ui/Pagination` (liens précédent/suivant et « Page n sur m »,
  `aria-current`), réutilisés par les invitations, les comptes et les
  rubriques futures.
- **Rationale** : recherche, filtres, tri et pagination sont faits par le
  serveur (FTS5 imposé) ; la bibliothèque n'aurait que du rendu à faire, et la
  liste fermée de dépendances (constitution) exige une justification qu'on
  n'a pas. À reconsidérer si une spec demande tri par colonne, sélection
  multiple ou colonnes configurables.
- **Critères dans l'URL** : `?q=&role=&statut=&page=` sur `/admin/comptes`
  (FR-004) ; tant que la saisie ne compte aucun mot de 3 caractères,
  l'indication « Saisissez au moins 3 caractères. » s'affiche sous le champ
  (`aria-describedby`) et la liste n'est pas filtrée ; sinon la recherche met
  l'URL à jour après 300 ms sans frappe, avec
  `replace` pour ne pas remplir l'historique ; un changement de filtre ramène à
  la page 1. Le nombre de résultats est annoncé dans une région `role=status`.
- **Fiche** : route `/admin/comptes/:id`, fil d'Ariane « Comptes › adresse ».
  Actions dans l'en-tête de page, chacune derrière `ConfirmDialog` (FR-012) ;
  le changement de rôle utilise un `Select` base-ui dans le dialogue. Après
  suppression, retour à la liste avec le message de confirmation.

## R13. Modifier les migrations existantes

- **Décision** : pas de migration `00004`. `disabled_at` est ajoutée à la
  création de `accounts` dans `00001_init.sql`, avec `accounts_fts` et ses
  triggers juste après.
- **Rationale** : le projet n'a jamais été déployé ; la constitution 1.4.0
  autorise alors la modification des migrations existantes. Le schéma reste
  lisible en un seul endroit, sans `ALTER TABLE`.
- **Conséquence** : les bases de développement existantes doivent être
  recréées (goose considère `00001` comme déjà appliquée).
