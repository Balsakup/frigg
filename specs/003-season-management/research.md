# Recherche : Gestion des saisons par les administrateurs

Décisions techniques de la fonctionnalité 003. Elles prolongent celles de
[001 › research.md](../001-auth-app-shell/research.md) et
[002 › research.md](../002-admin-account-management/research.md), qui restent
valables (enregistrement des opérations par `register`, écritures sérialisées,
erreurs `problem+json` en français, matrice d'autorisation parcourant
l'OpenAPI).

## R1. Stockage d'une saison

- **Décision** : table `seasons` (migration `00004_seasons.sql`) avec
  `starts_on` et `ends_on` en `TEXT` au format `AAAA-MM-JJ` (dates civiles,
  bornes incluses), `opened_at TEXT NULL` (`NULL` = brouillon) et
  `created_at`. Contraintes : `CHECK (date(x) = x)` sur les deux dates,
  `CHECK (ends_on > starts_on)`, `UNIQUE (starts_on)`.
- **Rationale** : une saison est une période de jours, pas un instant ; le
  format ISO se trie et se compare comme du texte. `opened_at` tient lieu
  d'état enregistré, sur le modèle de `accounts.disabled_at` (002 › R1), et
  conserve la date d'ouverture demandée par la spec (Key Entities).
- **Nouvelle migration plutôt que modifier `00001`** : la table est
  indépendante des précédentes ; ajouter une migration est la règle par défaut
  et évite de recréer les bases de développement.
- **Alternatives écartées** : colonne `status` (`draft`/`open`/`closed`)
  stockée : l'état « close » dépend de la date du jour et deviendrait faux
  sans tâche planifiée ; dates en `INTEGER` (jours julien) : illisibles en
  base, sans gain à ce volume.

## R2. État « close » et saison courante calculés

- **Décision** : l'état exposé est calculé à chaque lecture :
  `closed` si `ends_on < aujourd'hui`, sinon `open` si `opened_at` est
  renseigné, sinon `draft`. La saison courante est celle pour laquelle
  `starts_on ≤ aujourd'hui ≤ ends_on`, quel que soit son état. Aucun champ
  n'est écrit à la clôture.
- **Rationale** : FR-012 (« sans action de quiconque ») et SC-004 sont
  satisfaits par construction, sans worker ni risque d'oubli ; un serveur
  arrêté le jour de la clôture ne laisse aucune saison dans un mauvais état.
- **Alternatives écartées** : tâche périodique qui écrit `closed_at` (une
  pièce mobile de plus, état faux entre minuit et son passage) ; calcul en SQL
  avec `date('now', 'localtime')` (dépend du fuseau du système hôte, non
  testable).

## R3. « Aujourd'hui » : fuseau et horloge injectable

- **Décision** : la date du jour est celle d'Europe/Paris, calculée en Go à
  partir d'une horloge injectable : nouveau champ `Deps.Now func() time.Time`
  (`time.Now` si nul). Le fuseau est chargé par `time.LoadLocation` ; le
  binaire importe `time/tzdata` (bibliothèque standard, Go pur) pour ne pas
  dépendre de la base de fuseaux du système hôte (image minimale,
  `CGO_ENABLED=0`). La date du jour est passée en paramètre aux fonctions du
  domaine et aux requêtes.
- **Rationale** : hypothèse de la spec (association française) ; un serveur
  en UTC basculerait sinon de saison à 1 h ou 2 h du matin. L'horloge
  injectable permet de tester la clôture, la saison courante et les bornes
  (FR-007, FR-016) sans attendre ni manipuler l'horloge système.
- **Alternatives écartées** : variable `FRIGG_TIMEZONE` (aucune association
  hors de France visée ; à ajouter le jour où le besoin existe) ; fuseau du
  système (non maîtrisé en conteneur).

## R4. Règles du domaine dans un package pur

- **Décision** : nouveau package `internal/season`, sans accès à la base ni
  au réseau. Il expose un type `Date` (date civile sur `time.Time` à minuit
  UTC, avec analyse et formatage `AAAA-MM-JJ`) et des fonctions pures sur la
  suite ordonnée des saisons et la date du jour :
  - `State`, `IsCurrent`, `Labels` (libellés avec désambiguïsation, R6) ;
  - `NextStart` (lendemain de la fin de la dernière saison) ;
  - `DuplicateDates` (R7) ;
  - `Actions` (actions autorisées pour une saison, R8) ;
  - `ValidateCreate`, `ValidateUpdate` (renvoient des erreurs typées par
    champ et par règle, traduites en `422` ou `409` par l'API).
  Les handlers lisent toutes les saisons dans la transaction, appellent le
  domaine, puis écrivent.
- **Rationale** : les règles (continuité, 24 mois, bornes, états) sont
  nombreuses et combinées ; des tests unitaires en table les couvrent
  exhaustivement, sans base ni HTTP. Le volume (quelques dizaines de saisons,
  hypothèse de la spec) permet de tout charger à chaque requête.
- **Alternatives écartées** : règles en SQL (triggers et `CHECK` imbriqués :
  illisibles et messages d'erreur pauvres) ; règles dans les handlers
  (difficiles à tester isolément, dupliquées entre création, modification et
  actions).

## R5. Continuité et concurrence

- **Décision** : toutes les écritures passent par `Store.WriteTx`
  (une connexion, `BEGIN IMMEDIATE`) et relisent la liste des saisons dans la
  transaction avant de valider. La création exige `startsOn` dans le corps :
  s'il existe déjà des saisons et que la valeur diffère du lendemain de la fin
  de la dernière, la réponse est `409` « La liste des saisons a changé.
  Rechargez la page. ». La duplication refuse de même une saison source qui
  n'est plus la dernière. `UNIQUE (starts_on)` reste un filet de sécurité.
- **Modification d'une date de fin** : dans la même transaction, la saison
  suivante en brouillon voit son `starts_on` décalé (FR-017). L'ordre des deux
  `UPDATE` n'a pas d'importance puisqu'aucun trigger ne contrôle les
  chevauchements ; la validation complète a lieu en Go avant l'écriture.
- **Rationale** : les écritures sérialisées rendent la course « deux
  administrateurs dupliquent en même temps » déterministe : la seconde requête
  voit la saison créée par la première et reçoit un `409` (edge case de la
  spec).
- **Alternatives écartées** : triggers SQL anti-chevauchement (défense en
  profondeur utile, mais la cascade de dates impose un ordre d'écriture fragile
  et les erreurs `RAISE(ABORT)` sont difficiles à traduire en messages de
  champ) ; verrouillage optimiste par version (inutile avec un seul écrivain).

## R6. Libellé

- **Décision** : calculé par le domaine et renvoyé par l'API (`label`).
  Même année civile → `"2027"` ; sinon `"2026-2027"` (années de début et de
  fin, `"2026-2028"` sur trois années civiles). Si plusieurs saisons ont le
  même libellé de base, chacune reçoit les mois abrégés en français :
  `"2027 (janv.–juin)"`. Seules des saisons tenant dans une même année civile
  peuvent entrer en collision (deux saisons à cheval sur le 31 décembre d'une
  même année se chevaucheraient), donc le suffixe ne porte que sur des mois
  d'une même année. Le libellé n'est jamais stocké.
- **Rationale** : une seule implémentation (le frontend ne recalcule rien) ;
  la désambiguïsation dépend des autres saisons, que seul le serveur connaît
  en entier ; une modification de dates met tous les libellés à jour.
- **Alternatives écartées** : libellé saisi (contraire à la spec) ; libellé
  stocké (à recalculer pour les autres saisons à chaque écriture) ; calcul côté
  frontend (règle dupliquée, contraire au principe VI).

## R7. Duplication

- **Décision** : la nouvelle saison commence le lendemain de la fin de la
  source. Si la source couvre des mois entiers (début le 1er, fin le dernier
  jour d'un mois), elle dure le même nombre de mois : fin = début + N mois −
  1 jour (pas de débordement de `AddDate` puisque le début est un 1er) ;
  sinon elle dure le même nombre de jours. Pour une saison annuelle, le
  résultat équivaut à un décalage d'un an. La copie peut être entièrement
  passée : dupliquer de proche en proche comble le trou laissé par une
  interruption (décision du 2026-10-31).
- **Rationale** : couvre les saisons annuelles, semestrielles et les cas
  atypiques sans surprise (le 29 février ne décale pas les saisons suivantes
  de septembre à août).
- **Alternatives écartées** : `AddDate(1, 0, 0)` sur les deux dates (crée un
  trou ou un chevauchement pour une saison qui ne dure pas un an).

## R8. Actions autorisées exposées par l'API

- **Décision** : chaque saison renvoyée porte un objet `actions` calculé par
  le domaine : `editStartsOn`, `editEndsOn`, `open`, `duplicate`, `delete`
  (booléens) et `openBlockedReason` (texte français ou `null`, par exemple
  « Ouvrez d'abord la saison 2027-2028. »). Elle porte aussi `endsOnMin` et
  `endsOnMax`, les bornes de la date de fin, utilisées comme attributs `min` et
  `max` du champ de date, et `shiftsSeasonLabel`, le libellé de la saison
  suivante en brouillon dont le début suivra une modification de la fin
  (`null` sinon), pour l'aide du dialogue de modification. La réponse de liste porte `today`, `nextStartsOn`
  (date imposée pour la prochaine création, `null` s'il n'y a aucune saison),
  les bornes `nextEndsOnMin` et `nextEndsOnMax`, et `setup` (R10).
- **Rationale** : le serveur fait autorité (principe VI) ; le frontend
  n'affiche que les actions annoncées (FR-022) sans réimplémenter la
  continuité, la position ni les états. Les attributs HTML natifs guident la
  saisie, les erreurs `422` restent la référence.
- **Alternatives écartées** : déduire les actions côté frontend à partir de
  l'état et de la position (règles dupliquées, désynchronisées à la prochaine
  évolution) ; `OPTIONS` ou en-têtes `Allow` (peu pratiques à consommer avec
  openapi-fetch).

## R9. Format des dates dans l'API

- **Décision** : champs `string` avec `format:"date"` dans les types huma
  (`startsOn`, `endsOn`, `today`…), analysés en `season.Date` ; une date
  invalide donne un `422` sur le champ. Côté frontend, `<input type="date">`
  natif (accessible, clavier, sans dépendance) et une fonction
  `formatDay("2026-09-01")` → « 1 septembre 2026 » qui construit la date en
  heure locale (`new Date(y, m - 1, d)`) : `formatDate` existant passe par
  `new Date(chaîne)`, interprété en UTC, ce qui décale d'un jour à l'ouest de
  Greenwich.
- **Alternatives écartées** : `time.Time` en JSON (ajoute une heure et un
  fuseau à une date civile) ; sélecteur de date maison (motif ARIA complexe,
  base-ui n'en fournit pas ; le champ natif suffit).

## R10. Appel à l'action du tableau de bord

- **Décision** : la réponse de `list-seasons` porte
  `setup: "missing" | "no-current" | "current-draft" | "ready"`, calculé par
  le domaine (FR-023). Le tableau de bord appelle `list-seasons` et affiche
  une `Alert` correspondante : `info` avec lien vers `/admin/saisons/nouvelle`
  pour `missing` et `no-current` ; `warning` avec lien vers la fiche de la
  saison courante pour `current-draft` (identifiant dans `currentId`) ; rien
  pour `ready`.
- **Rationale** : une seule lecture, règle portée par le serveur ; pas
  d'opération dédiée tant que le tableau de bord n'a pas d'autres indicateurs.
- **Alternatives écartées** : opération `GET /admin/dashboard` (prématurée) ;
  calcul côté frontend (règle dupliquée).

## R11. Navigation : section « Configuration »

- **Décision** : la clé de section `resources` est renommée `settings` dans
  `navigation.ts` ; le titre affiché dépend de l'espace : « Configuration »
  dans l'espace administration, « Ressources » dans l'espace membre
  (hypothèse de la spec). L'entrée `saisons` (`/admin/saisons`, icône
  `CalendarRange`) est insérée avant `comptes` ; « Aide » reste en dernier
  (FR-001).
- **Rationale** : le titre est une affaire d'espace, pas d'entrée ; la clé
  `resources` aurait désormais menti dans l'espace administration.
- **Alternatives écartées** : troisième section (deux séparateurs pour trois
  liens) ; garder la clé `resources` (nom trompeur).

## R12. Écrans

- **Décision** :
  - `/admin/saisons` : liste dans `ui/Table` (Saison · Début · Fin · État ·
    mention « Saison courante »), sans pagination ni recherche (volume) ;
    bouton de page « Nouvelle saison » ; état vide dédié (FR-021).
  - `/admin/saisons/nouvelle` : formulaire de création sur une page (cible
    des liens de l'état vide et du tableau de bord). Date de début saisie s'il
    n'y a aucune saison, sinon affichée en lecture seule (`nextStartsOn`).
  - `/admin/saisons/:id` : fiche (liste de définitions), actions en en-tête de
    page ; modification des dates dans un `Dialog` base-ui ; ouverture et
    suppression via `ConfirmDialog` (principe IV) ; « Dupliquer » sans
    confirmation (action non destructive, réversible par suppression), qui
    mène à la fiche créée.
- Formulaires `react-hook-form`, erreurs `422` affichées par l'adaptateur
  existant ; `409` affiché dans une `Alert` au-dessus du formulaire avec
  invalidation des requêtes de saisons.
- **Alternatives écartées** : création dans un dialogue (pas d'URL à cibler
  depuis le tableau de bord) ; modification sur une page dédiée (deux champs au
  plus : un dialogue suffit).

## R13. Journalisation

- **Décision** : une ligne `INFO` par écriture réussie, sur le modèle de
  `logAccountAction` : `action` (`season.create`, `season.update`,
  `season.open`, `season.duplicate`, `season.delete`), `actor_id`,
  `season_id`, et les dates concernées.
- **Rationale** : traçabilité minimale cohérente avec 002 ; les dates ne sont
  pas des données personnelles.

## R14. Requêtes sqlc

- **Décision** : `internal/database/queries/seasons.sql` : `ListSeasons`
  (tri par `starts_on`), `CreateSeason`, `UpdateSeasonDates`,
  `OpenSeason` (conditionnelle : `opened_at IS NULL`), `DeleteSeason`.
  `GetSeason` n'est pas nécessaire : la fiche a besoin de toute la suite
  (libellés, position, actions). Commentaires de requêtes en ASCII
  uniquement (sqlc échoue sur un commentaire non ASCII dans les requêtes
  SQLite).
