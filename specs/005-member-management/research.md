# Recherche : Gestion des membres

Décisions techniques de la fonctionnalité 005. Elles prolongent celles de
001 à 004, qui restent valables : `register` et niveau d'accès par opération,
écritures sérialisées par `Store.WriteTx` (une connexion, `BEGIN IMMEDIATE`),
horloge injectable `Deps.Now` (date du jour à Paris par `s.today()`),
recherche FTS5 trigram via `database.FTSTrigramQuery`, comparaisons repliées
par `internal/textfold`, erreurs `422` cumulées par `FieldErrors`,
journalisation `INFO` des écritures sans texte libre.

## R1. Stockage d'une fiche

- **Décision** : table `members` (nouvelle migration `00007_members.sql`)
  avec `account_id` facultatif (`REFERENCES accounts(id) ON DELETE SET NULL`),
  `is_self`, identité (`first_name`, `last_name`, `birth_date` en
  `YYYY-MM-DD`), coordonnées facultatives (`email`, `phone`, `address_line1`,
  `address_line2`, `postal_code`, `city`) et horodatages. Contraintes `CHECK`
  sur les longueurs, `CHECK (is_self = 0 OR account_id IS NOT NULL)`, index
  unique partiel `members_one_self` sur `account_id WHERE is_self = 1`.
- **Rationale** : `account_id` nul porte directement la notion « Sans
  compte » (FR-003). L'index partiel garantit en base « une fiche “Vous” au
  plus par compte », même en cas d'écritures concurrentes. Le `CHECK` sur
  `is_self` fait échouer un `ON DELETE SET NULL` qui oublierait de retirer le
  signalement : la suppression d'un compte efface ou détache toujours ses
  fiches explicitement (R9) ; `SET NULL` n'est qu'un filet.
- **Nouvelle migration** : règle append-only par défaut, comme 004 › R12.
- **Alternatives écartées** : table de liaison compte ↔ fiche (inutile, un
  seul compte par fiche, décision du 2026-11-01) ; `ON DELETE CASCADE`
  (effacerait des fiches sans choix explicite) ; colonne `is_self` sur
  `accounts` pointant vers une fiche (référence circulaire).

## R2. Validation et nettoyage des champs

- **Décision** : nouveau package de domaine pur `internal/member`
  (`member.go`) : `Normalize` puis `Validate` sur une structure `Input`,
  erreurs de champ cumulées (même motif que `season.ValidateFee`). Règles :
  - prénom et nom : espaces de début et fin retirés, espaces intérieurs
    réduits ; non vides ; 80 caractères au plus (schéma `maxLength`) ;
  - date de naissance : format `YYYY-MM-DD` (schéma `format:"date"`), pas
    après aujourd'hui, pas plus de 120 ans avant aujourd'hui (`s.today()`) ;
  - adresse électronique : facultative, 254 caractères au plus, forme
    vérifiée par `net/mail.ParseAddress` sans nom d'affichage, enregistrée en
    minuscules ;
  - téléphone : voir R3 ;
  - adresse postale : tout ou rien. Si l'un des champs ligne 1, code postal
    ou commune est saisi, les trois sont obligatoires (complément
    facultatif). Code postal français à 5 chiffres ; lignes et commune
    100 caractères au plus.
- **Rationale** : domaine testable en table sans base (comme `season`) ; le
  serveur fait autorité (principe VI), le frontend n'utilise que les
  attributs HTML natifs (`required`, `maxLength`, `type="email"`,
  `type="date"`, `inputmode`).
- **Code postal français uniquement** : Frigg est réservée aux associations
  françaises (conventions de langue) ; une adresse à l'étranger reste
  possible plus tard en assouplissant la règle.

## R3. Téléphone

- **Décision** : l'API accepte un texte libre (chiffres, espaces, points,
  tirets, parenthèses, `+` initial) et l'enregistre normalisé : chiffres seuls
  pour un numéro français à 10 chiffres commençant par 0 (`0612345678`), ou
  `+` suivi de 8 à 15 chiffres pour un numéro international. Tout autre forme
  → `422`. Le frontend affiche un numéro français par paires
  (`06 12 34 56 78`), un numéro international tel qu'enregistré.
- **Rationale** : la recherche trigram (R6) trouve un fragment de numéro
  (`1234`) seulement si les chiffres sont contigus en base ; le format
  d'affichage est une présentation, pas une règle métier.
- **Alternatives écartées** : stockage tel que saisi (recherche par fragment
  impossible selon les séparateurs) ; bibliothèque de numéros (dépendance
  hors liste fermée pour un besoin simple).

## R4. Cloisonnement par compte (US2)

- **Décision** : les opérations de l'espace membre (`/me/members…`, accès
  `member`, donc aussi `admin` par la hiérarchie) ne prennent jamais
  d'identifiant de compte en entrée : le compte vient de la session
  (`CurrentAccount(ctx)`). Chaque requête sqlc de cet espace est conditionnée
  par `account_id = ?` (`GetMemberOfAccount`, `UpdateMemberOfAccount`,
  `DeleteMemberOfAccount` en `:execrows`) ; une fiche d'un autre compte, sans
  compte ou inexistante donne le même `404` « Ce membre n'existe plus. ».
- **Rationale** : le filtre est dans la requête SQL elle-même, pas dans une
  vérification ajoutée après coup : impossible d'oublier le contrôle dans un
  handler (FR-008, SC-002). Le `404` uniforme ne révèle pas l'existence de la
  fiche.
- **Tests** : une matrice dédiée (`members_isolation_test.go`) vérifie, pour
  chaque opération `/me/members/{id}`, qu'un compte B reçoit `404` sur une
  fiche de A et que la fiche est inchangée ; la matrice d'autorisation
  existante couvre l'accès anonyme (`401`) et les opérations `admin` (`403`
  pour un membre).

## R5. Opérations d'API

- **Décision** : deux groupes d'opérations, mêmes types de sortie.
  - Espace membre (tag `Mes membres`, accès `member`) : `list-my-members`,
    `get-my-member`, `create-my-member`, `update-my-member`,
    `delete-my-member` sous `/me/members`.
  - Administration (tag `Membres`, accès `admin`) : `list-members`,
    `get-member`, `create-member`, `update-member`, `delete-member` sous
    `/admin/members`.
  - Modification par `PUT` (formulaire toujours complet, comme 004 › R6).
- **Rationale** : deux chemins distincts rendent le niveau d'accès explicite
  par opération (principe III) et évitent un handler unique qui choisirait
  son filtre selon le rôle (vérification de rôle ad hoc interdite). Les
  handlers partagent le domaine et les fonctions de conversion.
- **Champs réservés à l'administration** : `accountId` (rattachement) n'existe
  que dans le corps des opérations `admin`. `isSelf` n'existe que dans le
  corps des opérations membre : un administrateur ne désigne pas la fiche
  d'un autre comme « Vous » ; changer ou retirer le rattachement remet
  `is_self` à 0 (US4 §3, §7).
- **Rattachement concurrent** : `update-member` reçoit `expectedAccountId`
  (rattachement affiché à l'ouverture de la fiche), obligatoire : en JSON,
  « absent » et `null` (sans compte) ne se distinguent pas une fois décodés,
  le champ est donc toujours envoyé. S'il diffère du
  rattachement relu dans la transaction : `409`, et rien n'est écrit. Les
  autres champs restent en « le dernier qui enregistre gagne », comme dans les
  fonctionnalités 002 à 004. Seul le rattachement décide de qui voit la fiche :
  l'écraser sans avertissement retirerait une fiche à une famille.
- **Propriétés inconnues** : refusées (`422`) par le schéma huma
  (`additionalProperties: false` par défaut).

## R6. Recherche (administration)

- **Décision** : table FTS5 à contenu externe `members_fts` (`first_name`,
  `last_name`, `email`, `phone`), `tokenize="trigram remove_diacritics 1"`,
  synchronisée par trois triggers (insertion, suppression, mise à jour de ces
  colonnes), comme `accounts_fts`. `q` passe par `FTSTrigramQuery` (fragments
  d'au moins 3 caractères, tous requis) ; aucun fragment valide → `422`
  `query.q` « Saisissez au moins 3 caractères. » (message existant).
- **Rationale** : principe VI : tous ces champs sont des champs d'identité
  courts (une seule catégorie par table FTS) ; même comportement que la
  recherche des comptes (002).

## R7. Tri et pagination de la liste d'administration

- **Décision** : la requête `ListMembers` renvoie toutes les fiches
  correspondant aux critères (avec l'adresse du compte par jointure) ; le
  handler trie en Go par `textfold.Fold(last_name)`, puis
  `Fold(first_name)`, puis identifiant, et découpe la page demandée
  (25 par défaut, 100 au plus). Le total est la longueur de la liste filtrée.
- **Rationale** : SQLite ne trie pas sans accents (« Élodie » après
  « Zoé ») ; la constitution impose `textfold` pour les tris et interdit les
  colonnes repliées. À l'échelle visée (SC-004 : 2 000 membres, quelques
  centaines d'octets par ligne), charger la liste filtrée coûte moins d'une
  milliseconde de tri et reste sur le pool de lecture.
- **Évolution** : si le volume l'exigeait, une collation `fold` enregistrée
  via `sqlite.RegisterCollationUtf8` (modernc, Go pur, appelant `textfold`)
  permettrait `ORDER BY … COLLATE fold LIMIT/OFFSET` ; écartée ici pour
  rester cohérent avec 004 › R2 (pas de couplage au pilote).
- **Espace membre** : 20 fiches au plus, tri en Go : fiche « Vous » d'abord,
  puis prénom replié, puis nom replié.

## R8. Doublon probable

- **Décision** : à la création, et à la modification lorsque prénom, nom ou
  date de naissance changent, le handler cherche dans la transaction les
  fiches de même `birth_date` (index `members_birth_date`) et compare
  `textfold.Fold` des prénoms et noms. S'il en trouve une (autre que la fiche
  modifiée) et que le corps ne porte pas `confirmDuplicate: true`, réponse
  `422` sur `body.confirmDuplicate` : « Un membre portant ces nom, prénom et
  date de naissance existe déjà. Confirmez pour enregistrer quand même. ». Le
  frontend affiche alors une case « Enregistrer quand même » sur le champ
  concerné ; le message ne mentionne jamais le compte de la fiche existante.
- **Rationale** : réutilise le canal d'erreurs par champ (adaptateur
  `applyProblemToForm`) sans nouveau type de réponse ; la recherche par date
  de naissance indexée limite la comparaison à une poignée de fiches. Le
  contrôle porte sur toute l'instance (FR-005), y compris depuis l'espace
  membre, sans révéler d'information au-delà de « existe déjà ».
- **Alternatives écartées** : refus strict (deux homonymes nés le même jour
  existent, par exemple des jumeaux homonymes rares mais possibles) ;
  avertissement non bloquant après enregistrement (doublon déjà créé).

## R9. Suppression d'un compte gérant des fiches (US5)

- **Décision** : `delete-account` (002) reçoit deux paramètres de requête
  facultatifs : `members` (`delete` ou `keep`) et `memberIds` (liste des
  identifiants des fiches affichées dans le dialogue). Dans la transaction,
  après les garde-fous de `adminWrite` :
  1. lecture des fiches du compte ;
  2. aucune fiche : comportement de 002 inchangé, paramètres ignorés ;
  3. sinon, `members` absent → `422` `query.members` « Choisissez le sort
     des fiches gérées par ce compte. » ; ensemble `memberIds` différent des
     fiches lues → `409` « La liste des fiches de ce compte a changé. » ;
  4. `delete` → `DeleteMembersOfAccount` ; `keep` →
     `DetachMembersOfAccount` (`account_id = NULL`, `is_self = 0`) ;
  5. suppression du compte comme en 002.
  La ligne de journal `account.delete` gagne `members_deleted` ou
  `members_kept` (nombre).
- **Rationale** : l'effacement et le détachement sont atomiques avec la
  suppression du compte. La comparaison des identifiants (et non d'un
  nombre) détecte une fiche ajoutée puis une retirée pendant que le dialogue
  est ouvert (US5 §5). `DELETE` sans corps reste conforme aux usages HTTP.
- **Dialogue** : il charge les fiches du compte par `list-members` filtré
  (`account={id}`, 20 au plus, R5) ; sans fiche, le dialogue de 002 est
  inchangé.
- **Alternatives écartées** : nouvelle opération `POST
  /admin/accounts/{id}/deletion` (doublonne `delete-account` et casse les
  écrans de 002) ; corps sur `DELETE` (mal pris en charge par certains
  clients et intermédiaires).

## R10. Plafond de 20 fiches par compte et fiche « Vous »

- **Décision** : à la création depuis l'espace membre, au rattachement
  (création ou modification par un administrateur) et à la réactivation
  (rien à faire : le rattachement est conservé), le handler compte les
  fiches du compte cible dans la transaction ; 20 atteint → `409` « Vous
  gérez déjà 20 membres. Contactez le bureau pour en ajouter. »
  (espace membre) ou « Ce compte gère déjà 20 membres. » (administration).
  Une seconde fiche « Vous » est refusée par le domaine (`422` `body.isSelf`
  « Une autre fiche vous désigne déjà. ») ; l'index unique partiel fait foi
  en cas de concurrence (violation traduite en ce même `422`).
- **Rationale** : écritures sérialisées, donc pas de course sur le
  comptage ; l'index protège la seule règle d'unicité.
- **Actions annoncées** : `list-my-members` renvoie `actions: { add, markSelf
  }` (`add` : moins de 20 fiches ; `markSelf` : aucune fiche « Vous »), comme
  les actions des tarifs (004 › R5) ; le frontend ne recalcule rien.

## R11. Âge

- **Décision** : `age` (années révolues) est calculé par le serveur avec
  `s.today()` et renvoyé dans chaque fiche ; jamais stocké.
- **Rationale** : la date du jour fait foi côté serveur (horloge injectable,
  testable) ; le frontend n'a pas de règle de calcul à dupliquer.

## R12. Écrans

- **Décision** :
  - Espace membre : `/mes-membres` (liste en cartes, une par fiche),
    `/mes-membres/nouveau` (formulaire), `/mes-membres/:id` (formulaire
    prérempli, bouton « Supprimer »).
  - Administration : `/admin/membres` (tableau, recherche et filtre
    « Rattachement » reflétés dans l'URL, pagination `ui/Pagination`),
    `/admin/membres/nouveau`, `/admin/membres/:id` (formulaire prérempli,
    bloc d'informations : compte de rattachement avec lien, dates).
  - Un composant `MemberFields` partagé (identité, coordonnées, adresse) ;
    les champs propres à chaque espace (« Il s'agit de moi », compte de
    rattachement) sont ajoutés par la page.
  - Choix du compte de rattachement : `AccountPicker`, Combobox base-ui
    alimentée par `list-accounts` (`q`, 3 caractères au moins), avec
    l'option « Sans compte ».
  - Suppression via `ConfirmDialog` `danger` ; dialogue de suppression de
    compte (002) enrichi d'un groupe de boutons radio obligatoire.
  - Les rubriques « Mes membres » et « Membres » quittent la liste
    `comingSoon` (navigation et `routes.ts`).
- **Rationale** : une dizaine de champs ne tient pas dans un dialogue lisible
  à 320 px (contrairement aux tarifs, 004 › R9) ; une page par fiche donne
  une URL partageable entre membres du bureau. Combobox imposée par le
  principe IV pour une liste filtrée.
- **Alternatives écartées** : `Select` de tous les comptes (inutilisable à
  2 000 comptes) ; formulaire en dialogue.

## R13. Journalisation

- **Décision** : une ligne `INFO` par écriture réussie : `action`
  (`member.create`, `member.update`, `member.attach`, `member.delete`),
  `actor_id`, `member_id`, `account_id` (rattachement résultant, absent si
  « Sans compte »), et `space` (`me` ou `admin`). `member.attach` est émis
  quand le rattachement change, en plus de `member.update`. Aucune donnée de
  la fiche (nom, date, coordonnées) n'est journalisée.
- **Rationale** : FR-017 ; principe VII (données personnelles hors des
  journaux) ; cohérent avec 004 › R10.

## R14. Fiches d'un compte désactivé

- **Décision** : aucune règle spécifique. Un compte désactivé n'a pas de
  session (002) : ses fiches restent rattachées et visibles de
  l'administration ; elles réapparaissent dans « Mes membres » à la
  réactivation. Un administrateur peut rattacher une fiche à un compte
  désactivé (le sélecteur l'indique « désactivé »).
- **Rationale** : la désactivation est réversible ; détacher les fiches
  ferait perdre l'information.
