---

description: "Liste des tâches : gestion des saisons par les administrateurs"
---

# Tasks: Gestion des saisons par les administrateurs

**Input**: Design documents from `specs/003-season-management/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : tests d'autorisation (principes II et III), tests
d'API, Playwright + axe sur chaque parcours touché (Workflow). Les règles de
dates, nombreuses et combinées, sont couvertes par des tests en table du
package `internal/season`.

**Organization**: une phase par user story, dans l'ordre de priorité de la spec
(US1 et US2 en P1, US3 et US4 en P2, US5 en P3).

**Conventions** (constitution 1.4.0) : identifiants, tables, champs JSON et
valeurs d'énumération en anglais ; commentaires, messages, textes d'interface
et chemins visibles du SPA en français. Commentaires des requêtes sqlc en
ASCII uniquement (sqlc échoue sur un commentaire non ASCII). Nouvelle
migration `00004_seasons.sql` (append-only).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US5)

## Path Conventions

Un module Go à la racine (`internal/`) et le SPA dans `web/` (voir plan.md ›
Project Structure). Helpers de test existants : `internal/api/apitest`
(`New`, `Seed(role, verified)`, `LoggedIn(email)`, `Client().Do(...)`,
`LogBuffer`) ; e2e : `web/e2e/test.ts` (`login`, `test`, `expect`) et
`web/e2e/axe.ts` (`expectNoBlockingA11yViolations`). Régénération : `go tool
task gen` (sqlc, OpenAPI, types TypeScript).

**Date de référence des exemples** : « aujourd'hui » = 2026-10-31 (Europe/Paris)
dans les tests Go, via l'horloge injectée.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: fuseau horaire embarqué ; aucune dépendance ajoutée

- [X] T001 Ajouter `import _ "time/tzdata"` dans `cmd/frigg/main.go` avec le commentaire « Base des fuseaux embarquée : la date du jour des saisons est celle d'Europe/Paris, même sur un système sans tzdata (research.md 003 › R3). »

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: table, requêtes, package de domaine (dates, états, libellés, actions), horloge injectable, type `Season` de l'API, renommage de la section de navigation, compte e2e

**⚠️ CRITICAL**: aucune story ne peut commencer avant la fin de cette phase

### Base de données

- [X] T002 Créer `internal/database/migrations/00004_seasons.sql` : `CREATE TABLE seasons (id INTEGER PRIMARY KEY, starts_on TEXT NOT NULL UNIQUE CHECK (date(starts_on) = starts_on), ends_on TEXT NOT NULL CHECK (date(ends_on) = ends_on), opened_at TEXT, created_at TEXT NOT NULL, CHECK (ends_on > starts_on)) STRICT;` avec commentaires français (« Dates civiles AAAA-MM-JJ, bornes incluses. », « opened_at : NULL = brouillon ; l'état close est calculé à partir de la date du jour (research.md 003 › R2). ») ; `-- +goose Down` : `DROP TABLE seasons;`
- [X] T003 Créer `internal/database/queries/seasons.sql` (commentaires ASCII) : `ListSeasons :many` (`SELECT * FROM seasons ORDER BY starts_on`), `CreateSeason :one` (`INSERT ... (starts_on, ends_on, created_at) VALUES (sqlc.arg(starts_on), sqlc.arg(ends_on), sqlc.arg(created_at)) RETURNING *`), `UpdateSeasonDates :exec` (`UPDATE seasons SET starts_on = sqlc.arg(starts_on), ends_on = sqlc.arg(ends_on) WHERE id = sqlc.arg(id)`), `OpenSeason :execrows` (`UPDATE seasons SET opened_at = sqlc.arg(now) WHERE id = sqlc.arg(id) AND opened_at IS NULL`), `DeleteSeason :execrows` (`DELETE FROM seasons WHERE id = sqlc.arg(id)`) ; lancer `go generate ./internal/database` et vérifier le type `sqlc.Season` généré

### Domaine (`internal/season`)

- [X] T004 [P] Écrire `internal/season/date_test.go` (doit échouer avant T005) : `ParseDate("2026-09-01")` OK, `ParseDate("2026-02-30")`, `"2026-9-1"`, `""` → erreur ; `String()` → `"2026-09-01"` ; `AddDays(-1)` franchit mois et années ; `AddMonths(12)` depuis un 1er du mois ; `Before`/`After`/`Equal` ; `Today(now time.Time)` renvoie la date à Europe/Paris (`2026-10-31T23:30:00Z` → `2026-11-01`, `2026-10-31T22:30:00Z` → `2026-10-31` en heure d'hiver) ; `MonthsCovered(start, end)` renvoie `(12, true)` pour 2026-09-01 → 2027-08-31, `(6, true)` pour 2027-01-01 → 2027-06-30, `(0, false)` pour 2026-09-15 → 2027-09-14
- [X] T005 Implémenter `internal/season/date.go` : package `season` (commentaire de package français : « Règles métier des saisons, sans base ni HTTP (research.md 003 › R4). ») ; type `Date struct{ t time.Time }` (minuit UTC), `ParseDate(string) (Date, error)` (format `2006-01-02`, refus si la reformation diffère de l'entrée), `MustDate`, `String()`, `AddDays(n)`, `AddMonths(n)` (via `time.AddDate`, documenté comme sûr seulement depuis un 1er du mois), `Before`, `After`, `Equal`, `Compare`, `Year()`, `Month()`, `Day()`, `IsFirstOfMonth()`, `IsLastOfMonth()`, `MonthsCovered(start, end Date) (int, bool)` ; `var Paris = mustLoad("Europe/Paris")` et `Today(now time.Time) Date` ; faire passer T004
- [X] T006 [P] Écrire `internal/season/season_test.go` (tests en table, doivent échouer avant T007), avec `today = 2026-10-31` : **State** — `ends_on` hier → `Closed` (brouillon ou ouverte), `ends_on` aujourd'hui → état enregistré, `opened_at` renseigné → `Open`, sinon `Draft` ; **IsCurrent** — vrai si `starts_on ≤ today ≤ ends_on`, bornes incluses ; **Labels** — 2026-09-01 → 2027-08-31 : `"2026-2027"` ; 2027-01-01 → 2027-12-31 : `"2027"` ; 2026-01-01 → 2028-06-30 : `"2026-2028"` ; deux saisons 2027-01-01 → 2027-06-30 et 2027-07-01 → 2027-12-31 : `"2027 (janv.–juin)"` et `"2027 (juil.–déc.)"` (abréviations : janv., févr., mars, avr., mai, juin, juil., août, sept., oct., nov., déc. ; tiret demi-cadratin) ; une seule saison d'une année : pas de suffixe ; **NextStart** — aucune saison → absent ; sinon lendemain de la dernière fin ; **Actions** (grille de data-model.md › Actions autorisées) — `editStartsOn` seulement pour la première saison en brouillon ; `editEndsOn` pour brouillon ou ouverte sans suivante ouverte, faux pour close ; `open` pour brouillon dont la précédente est absente, ouverte ou close ; `openBlockedReason` = « Ouvrez d'abord la saison {libellé}. » quand la précédente est en brouillon ; `duplicate` pour la dernière saison quel que soit son état ; `delete` pour la dernière saison en brouillon seulement ; **Bornes de fin** (data-model.md › Bornes de saisie) — `endsOnMin` = le plus tardif de `today`, `startsOn + 1 jour` et, si suivante en brouillon, `N.ends_on − 24 mois` ; `endsOnMax` = le plus précoce de `startsOn + 24 mois − 1 jour` et, si suivante en brouillon, `N.ends_on − 2 jours` ; `editEndsOn` faux si `endsOnMin > endsOnMax` ; **ShiftsLabel** — libellé de la saison suivante si elle est en brouillon et que `editEndsOn` est vrai (sa date de début suivra la fin), sinon vide ; **Setup** — aucune saison → `missing` ; aucune ne contient `today` → `no-current` ; courante en brouillon → `current-draft` ; courante ouverte → `ready`
- [X] T007 Implémenter `internal/season/season.go` : type `Season { ID int64; StartsOn, EndsOn Date; OpenedAt *time.Time }` (suite triée par `StartsOn`) ; type `State` (`Draft = "draft"`, `Open = "open"`, `Closed = "closed"`) ; `StateOf(s, today)`, `IsCurrent(s, today)`, `Labels(seasons) map[int64]string` (research.md › R6), `NextStart(seasons) (Date, bool)`, `type Actions { EditStartsOn, EditEndsOn, Open, Duplicate, Delete bool; OpenBlockedReason string }`, `ActionsFor(seasons, i, today, labels) Actions`, `EndsOnBounds(seasons, i, today) (min, max Date)`, `ShiftsLabel(seasons, i, today, labels) string` (saison suivante dont le début suivra une modification de la fin), `NewSeasonEndsOnBounds(seasons, today) (min, max Date, ok bool)` (bornes de la fin d'une création : début imposé = `NextStart`), `type Setup string` (`missing`, `no-current`, `current-draft`, `ready`) et `SetupOf(seasons, today) (Setup, currentID int64)` ; constante `MaxMonths = 24` ; faire passer T006

### API

- [X] T008 Ajouter dans `internal/api/api.go` le champ `Now func() time.Time` à `Deps` (commentaire : « Horloge injectable ; time.Now si nil (research.md 003 › R3). ») et la méthode `func (s *server) today() season.Date { now := time.Now; if s.Now != nil { now = s.Now }; return season.Today(now()) }` ; ajouter l'appel `s.registerSeasons(api)` après `s.registerAccounts(api)`
- [X] T009 Ajouter à `internal/api/apitest/apitest.go` une horloge réglable : champ `clock atomic.Pointer[time.Time]` dans `Env`, `Deps.Now` qui renvoie l'heure fixée ou `time.Now()` sinon, et `func (e *Env) SetToday(day string)` qui fixe l'horloge à midi (Europe/Paris) du jour donné (`"2026-10-31"`)
- [X] T010 Créer `internal/api/seasons.go` : `tagSeasons = "Saisons"` ; messages `MsgSeasonGone = "Cette saison n'existe plus."`, `MsgSeasonsChanged = "La liste des saisons a changé. Rechargez la page."`, `MsgSeasonClosed = "Une saison close ne peut plus être modifiée."`, `MsgSeasonStartLocked = "La date de début de cette saison ne peut pas être modifiée."`, `MsgNextSeasonOpen = "La saison suivante est déjà ouverte : la date de fin ne peut plus changer."`, `MsgSeasonClosedOpen = "Une saison close ne peut plus être ouverte."`, `MsgDeleteDraftOnly = "Seule une saison en brouillon peut être supprimée."`, `MsgDeleteLastOnly = "Seule la dernière saison peut être supprimée."`, et messages de champ `MsgInvalidDate = "Saisissez une date valide."`, `MsgEndBeforeStart = "La date de fin doit être postérieure à la date de début."`, `MsgEndInPast = "La date de fin ne peut pas être passée."`, `MsgSeasonTooLong = "Une saison ne peut pas durer plus de 24 mois."`, `MsgNextSeasonInvalid` (format « Cette date rendrait la saison %s invalide. ») ; types `SeasonActions` (`editStartsOn`, `editEndsOn`, `open`, `duplicate`, `delete`, `openBlockedReason *string`), `Season` (`id`, `label`, `startsOn` `format:"date"`, `endsOn` `format:"date"`, `status` `enum:"draft,open,closed"`, `isCurrent`, `openedAt *time.Time`, `createdAt time.Time`, `endsOnMin`, `endsOnMax` `format:"date"`, `shiftsSeasonLabel *string`, `actions`), `SeasonList` (`items`, `today`, `currentId *int64`, `nextStartsOn`, `nextEndsOnMin`, `nextEndsOnMax` `*string` `format:"date"`, `setup` `enum:"missing,no-current,current-draft,ready"`) — voir contracts/api.md › Types ; helpers `toDomain([]sqlc.Season) []season.Season`, `buildSeasons(rows, today) []Season` (labels, actions et bornes calculés une fois pour la suite), `buildSeasonList(rows, today) SeasonList` (items de la plus récente à la plus ancienne) ; `logSeasonAction(ctx, action string, seasonID int64, attrs ...any)` en `INFO` « action d'administration sur une saison » avec `action`, `actor_id`, `season_id` (research.md › R13) ; `func (s *server) registerSeasons(api huma.API)` vide pour l'instant

### Frontend

- [X] T011 [P] Ajouter dans `web/app/lib/format.ts` `formatDay(value: string)` (« 2026-09-01 » → « 1 septembre 2026 », date construite en heure locale `new Date(y, m - 1, d)`, commentaire : `formatDate` lit une chaîne ISO en UTC et décalerait le jour, research.md 003 › R9) et `seasonStatusLabel(status: "draft" | "open" | "closed")` (« Brouillon », « Ouverte », « Close ») ; créer `web/app/lib/format.test.ts` (Vitest) : `formatDay("2026-09-01") === "1 septembre 2026"` quel que soit `process.env.TZ` (tester avec `America/New_York` via `vi.stubEnv` ou en vérifiant `getDate()`), libellés d'état
- [X] T012 [P] Renommer la section `resources` en `settings` dans `web/app/shell/navigation.ts` (type `section: "main" | "settings"`, toutes les entrées concernées) ; dans `web/app/shell/Sidebar.tsx`, titre de la section du bas `space === "admin" ? "Configuration" : "Ressources"` (identifiant `nav-secondaire-titre` inchangé) ; mettre à jour `web/app/shell/navigation.test.ts` (`entriesFor("admin", "settings")` contient `comptes`)

### Jeu de données e2e

- [X] T013 [P] Ajouter dans `internal/tools/e2eseed/main.go` le compte administrateur vérifié `admin-saisons@exemple.fr` (mot de passe commun) ; aucune saison n'est créée par le seed (les tests partent d'une base sans saison, quickstart.md › Jeu de données)

**Checkpoint**: `go tool task gen` sans erreur, `go test ./...` vert (dont T004 et T006), navigation renommée.

---

## Phase 3: User Story 1 - Créer la première saison et consulter les saisons (Priority: P1) 🎯 MVP

**Goal**: rubrique « Saisons » dans « Configuration », état vide, création, liste, fiche en lecture

**Independent Test**: base sans saison ; ouvrir `/admin/saisons`, constater l'état vide, créer une saison, la retrouver dans la liste avec libellé, état et dates, ouvrir sa fiche ; en membre, l'API répond 403

### Tests for User Story 1

- [X] T014 [P] [US1] Ajouter à `internal/season/season_test.go` les cas de `ValidateCreate(seasons, startsOn, endsOn, today) error` : aucune saison, début passé et fin future → OK ; fin ≤ début → erreur champ `endsOn` `ErrEndBeforeStart` ; fin < today → `ErrEndInPast` ; durée > 24 mois (2026-09-01 → 2028-09-01) → `ErrTooLong`, 2026-09-01 → 2028-08-31 → OK ; saisons existantes et `startsOn` ≠ `NextStart` → `ErrSeasonsChanged` ; `startsOn` = `NextStart` → OK
- [X] T015 [P] [US1] Écrire `internal/api/seasons_list_test.go` (httptest via `apitest`, `env.SetToday("2026-10-31")`) : `GET /api/admin/seasons` sur base vide → `items: []`, `today: "2026-10-31"`, `currentId: null`, `nextStartsOn: null`, `setup: "missing"` ; `POST /api/admin/seasons` `{startsOn: "2026-09-01", endsOn: "2027-08-31"}` → 201, `label: "2026-2027"`, `status: "draft"`, `isCurrent: true`, `actions.open: true`, `actions.delete: true`, `actions.editStartsOn: true` ; ensuite la liste a `nextStartsOn: "2027-09-01"` et `setup: "current-draft"` ; 2e création avec `startsOn: "2027-09-01"`, `endsOn: "2028-08-31"` → 201 et liste triée de la plus récente à la plus ancienne ; création avec `startsOn` ≠ début imposé → 409 `MsgSeasonsChanged` ; `endsOn: "2026-10-30"` → 422 `errors[].location = "body.endsOn"` « La date de fin ne peut pas être passée. » ; fin ≤ début → 422 ; 25 mois → 422 ; `startsOn: "2026-02-30"` → 422 sur `body.startsOn` ; deux saisons semestrielles de 2027 (base neuve) → libellés « 2027 (janv.–juin) » et « 2027 (juil.–déc.) » ; `GET /api/admin/seasons/{id}` → 200, id inconnu → 404 « Cette saison n'existe plus. » ; ligne de journal `season.create` avec `actor_id` et `season_id` ; membre → 403, anonyme → 401 sur les trois opérations
- [X] T016 [P] [US1] Ajouter à `web/app/shell/navigation.test.ts` : `entriesFor("admin", "settings").map(e => e.id)` vaut `["saisons", "comptes", "aide-admin"]` ; l'entrée `saisons` pointe vers `/admin/saisons` et n'est pas `comingSoon`

### Implementation for User Story 1

- [X] T017 [US1] Implémenter dans `internal/season/season.go` `ValidateCreate` et le type `FieldError { Field string; Code error }` avec les erreurs sentinelles `ErrInvalidDate`, `ErrEndBeforeStart`, `ErrEndInPast`, `ErrTooLong`, `ErrNextInvalid`, `ErrSeasonsChanged` (les règles métier restent indépendantes des messages, traduits par l'API) ; faire passer T014
- [X] T018 [US1] Implémenter dans `internal/api/seasons.go` : `list-seasons` (`GET /admin/seasons`, `auth.Admin`, `ListSeasons` sur `s.Store.Read`, sortie `SeasonList`) ; `get-season` (`GET /admin/seasons/{id}`, lit toute la suite, 404 `MsgSeasonGone`) ; `create-season` (`POST /admin/seasons`, `DefaultStatus: 201`, corps `{ startsOn string format:"date", endsOn string format:"date" }` ; dans `s.Store.WriteTx` : `ListSeasons`, `season.ValidateCreate` avec `s.today()`, `CreateSeason` avec `created_at = nowString()`) ; traduction commune `seasonError(err)` : `FieldError` → `FieldErrors("body."+field, message)`, `ErrSeasonsChanged` → `409 MsgSeasonsChanged` ; `logSeasonAction("season.create", id, "starts_on", …, "ends_on", …)` ; réponse relue hors transaction ; puis `go tool task gen` ; faire passer T015
- [X] T019 [P] [US1] Ajouter l'entrée de navigation dans `web/app/shell/navigation.ts` : `{ id: "saisons", label: "Saisons", icon: CalendarRange, path: "/admin/saisons", space: "admin", section: "settings" }` insérée avant `comptes` (import `CalendarRange` de `lucide-react`) ; faire passer T016
- [X] T020 [US1] Créer `web/app/api/seasons.ts` sur le modèle de `web/app/api/accounts.ts` : `seasonListKey`, `seasonListQuery()` (`list-seasons`), `seasonQuery(id)` (`get-season`), `invalidateSeasons(qc)` qui invalide la liste et toutes les fiches ; types `Season`, `SeasonList` depuis `components["schemas"]`
- [X] T021 [US1] Déclarer les routes dans `web/app/routes.ts`, dans le layout admin : `route("saisons", "routes/app/admin/seasons-layout.tsx", [index("routes/app/admin/seasons.tsx"), route("nouvelle", "routes/app/admin/season-new.tsx"), route(":id", "routes/app/admin/season-detail.tsx")])` (`nouvelle` avant `:id`) ; créer `web/app/routes/app/admin/seasons-layout.tsx` sur le modèle de `accounts-layout.tsx` (`handle.crumb` « Saisons » vers `/admin/saisons`, `<Outlet />`)
- [X] T022 [US1] Créer `web/app/routes/app/admin/seasons.tsx` (contracts/ui-routes.md › Liste) : `clientLoader` qui précharge `seasonListQuery` ; `Page` titre « Saisons », introduction « Périodes d'activité de l'association, qui servent aussi d'exercice comptable. », action « Nouvelle saison » (lien `Button` vers `/admin/saisons/nouvelle`) seulement s'il existe au moins une saison ; `Table` caption « Saisons de l'association, de la plus récente à la plus ancienne », colonnes Saison (lien vers `/admin/saisons/:id` dans un `th scope="row"`, badge `brand` « Saison courante » si `isCurrent`), Début, Fin (`formatDay`), État (`Badge` : `warning` Brouillon, `ok` Ouverte, `muted` Close via `seasonStatusLabel`) ; état vide : titre `h2` « Aucune saison pour le moment », texte « Une saison délimite une période d'activité : adhésions, événements et comptes de l'année s'y rattachent. Créez la première pour mettre en place Frigg. », bouton principal « Créer la première saison » vers `/admin/saisons/nouvelle`
- [X] T023 [US1] Créer `web/app/routes/app/admin/SeasonDatesFields.tsx` : champs `Field` `type="date"` contrôlés par `react-hook-form` (`register`), props `showStartsOn`, `startsOnReadOnly?: { value: string; hint: string }`, `endsOnMin`, `endsOnMax?`, `endsOnHint?`, `errors` ; « Date de début » (aide « Elle peut être passée si la saison a déjà commencé. » quand saisissable), « Date de fin » (aide « Dernier jour inclus. ») ; attributs natifs `min`/`max`/`required`, aucune validation dupliquée côté client
- [X] T024 [US1] Créer `web/app/routes/app/admin/season-new.tsx` (contracts/ui-routes.md › Création) : `handle.crumb` « Nouvelle saison » ; `clientLoader` qui précharge `seasonListQuery` ; titre « Nouvelle saison » ; `SeasonDatesFields` : début saisissable si `nextStartsOn` est nul, sinon en lecture seule avec l'aide « Le lendemain de la fin de la saison {libellé de la plus récente}. » ; bornes de fin `nextEndsOnMin`/`nextEndsOnMax` ou `today` ; soumission `$api.useMutation("post", "/admin/seasons")` avec `startsOn` (imposé ou saisi) ; erreurs 422 via l'adaptateur existant de `react-hook-form` ; 409 → `Alert` `danger` au-dessus du formulaire et `invalidateSeasons` ; succès → `invalidateSeasons`, navigation vers `/admin/saisons/{id}` avec le message « Saison {libellé} créée. » dans la région `role="status"` (même mécanisme que les messages de `account-detail.tsx`) ; boutons « Créer la saison » (principal) et « Annuler » (lien vers `/admin/saisons`)
- [X] T025 [US1] Créer `web/app/routes/app/admin/season-detail.tsx` (contracts/ui-routes.md › Fiche), sur le modèle de `account-detail.tsx` : `parseID`, `clientLoader` qui précharge `seasonQuery(id)` avec `.catch(() => null)` ; 404 ou id invalide → `NotFound` ; `handle.crumb` = libellé ou « Saison introuvable » ; `Page` titre « Saison {libellé} », introduction = badges d'état et « Saison courante » ; liste de définitions Début, Fin (`formatDay`), État, Ouverte le (`formatDate`, si renseigné), Créée le ; zone d'actions d'en-tête vide prête pour US2 et US3 ; `Alert` `info` avec `actions.openBlockedReason` s'il est non nul
- [X] T026 [US1] Créer `web/e2e/admin-seasons.spec.ts` avec `test.describe.configure({ mode: "serial" })` (base partagée : l'état global des saisons impose l'ordre) et le compte `admin-saisons@exemple.fr` ; dates calculées depuis la date du jour par un helper local : saison de référence = du 1er du mois courant au dernier jour du 12e mois (début + 12 mois − 1 jour), donc en mois entiers et contenant aujourd'hui, ce qui donne un décalage d'un an exact à la duplication (T032) ; fin passée = veille de la date du jour ; bloc US1 : la barre latérale montre la section « Configuration » avec « Saisons » avant « Comptes » ; `/admin/saisons` affiche l'état vide et « Créer la première saison », sans bouton « Nouvelle saison » ; axe sans violation `serious`/`critical` ; création d'une saison contenant aujourd'hui → fiche « Saison … », badge « Brouillon » et « Saison courante » ; fin passée → message « La date de fin ne peut pas être passée. » sur le champ ; retour à la liste : une ligne, badge « Saison courante » ; axe sur la liste, le formulaire et la fiche, y compris à 320 px sans défilement horizontal de la page

**Checkpoint**: US1 utilisable seule — quickstart scénarios 1, 3, 4, 5.

---

## Phase 4: User Story 2 - Préparer et ouvrir la saison suivante (Priority: P1)

**Goal**: duplication de la dernière saison, création manuelle à date de début imposée, ouverture dans l'ordre, plusieurs saisons ouvertes

**Independent Test**: avec 2026-2027 ouverte, dupliquer → 2027-2028 en brouillon aux bonnes dates ; l'ouvrir → deux saisons ouvertes, la courante reste 2026-2027

### Tests for User Story 2

- [X] T027 [P] [US2] Ajouter à `internal/season/season_test.go` les cas de `DuplicateDates(src Season) (Date, Date)` : 2026-09-01 → 2027-08-31 donne 2027-09-01 → 2028-08-31 ; 2027-01-01 → 2027-06-30 donne 2027-07-01 → 2027-12-31 ; 2027-04-01 → 2027-09-30 donne 2027-10-01 → 2028-03-31 ; 2028-03-01 → 2029-02-28 (mois entiers, traverse un 29 février) donne 2029-03-01 → 2030-02-28 ; 2026-09-15 → 2027-09-14 (pas de mois entiers, 365 jours) donne 2027-09-15 → 2028-09-13 (même nombre de jours) ; et de `CanOpen` : précédente brouillon → refus avec libellé de la précédente ; close → refus
- [X] T028 [P] [US2] Écrire `internal/api/seasons_open_test.go` (httptest, `SetToday("2026-10-31")`) : saison 2026-2027 créée ; `POST /api/admin/seasons/{id}/open` → 200 `status: "open"`, `openedAt` renseigné, `actions.delete: false`, liste `setup: "ready"` ; rouvrir → 200 inchangé, pas de nouvelle ligne de journal ; `POST /api/admin/seasons/{id}/duplicate` → 201, `label: "2027-2028"`, `startsOn: "2027-09-01"`, `endsOn: "2028-08-31"`, `status: "draft"` ; dupliquer 2026-2027 (n'est plus la dernière) → 409 `MsgSeasonsChanged` ; dupliquer 2027-2028 → 2028-2029 ; ouvrir 2028-2029 → 409 « Ouvrez d'abord la saison 2027-2028. » ; ouvrir 2027-2028 (début futur) → 200 et deux saisons `open`, `currentId` = 2026-2027 ; `SetToday` après la fin d'une saison en brouillon puis `open` → 409 `MsgSeasonClosedOpen` ; **concurrence** : deux goroutines `POST …/duplicate` sur la dernière saison → exactement une 201 et une 409, une seule saison ajoutée ; lignes de journal `season.open` et `season.duplicate` ; membre → 403, anonyme → 401

### Implementation for User Story 2

- [X] T029 [US2] Implémenter dans `internal/season/season.go` `DuplicateDates` (research.md › R7 : mois entiers via `MonthsCovered` puis `start.AddMonths(n).AddDays(-1)` depuis un 1er du mois, sinon même nombre de jours) et `CanOpen(seasons, i, today, labels) error` (`ErrClosed`, `ErrPreviousDraft` portant le libellé), en cohérence avec `ActionsFor` ; faire passer T027
- [X] T030 [US2] Implémenter dans `internal/api/seasons.go` : `open-season` (`POST /admin/seasons/{id}/open` ; `WriteTx` : `ListSeasons`, cible absente → 404, `season.CanOpen` → 409 avec message traduit, déjà ouverte → renvoyer l'état sans écrire, sinon `OpenSeason` avec `nowString()` ; `logSeasonAction("season.open")`) et `duplicate-season` (`POST /admin/seasons/{id}/duplicate`, `DefaultStatus: 201` ; `WriteTx` : `ListSeasons`, cible absente → 404, cible ≠ dernière → 409 `MsgSeasonsChanged`, `season.DuplicateDates`, `CreateSeason` ; `logSeasonAction("season.duplicate", newID, "source_id", id)`) ; `go tool task gen` ; faire passer T028
- [X] T031 [US2] Ajouter à `web/app/routes/app/admin/season-detail.tsx` les actions d'en-tête selon `actions` : « Ouvrir la saison » (`Button` principal) → `ConfirmDialog` `tone="primary"`, titre « Ouvrir la saison {libellé} ? », texte « Les adhésions et inscriptions pourront s'y rattacher. Une saison ouverte ne peut plus être supprimée et seule sa date de fin restera modifiable. », bouton « Ouvrir la saison » ; « Dupliquer » (`Button` secondaire, icône `Copy`) sans confirmation → `duplicate-season`, puis navigation vers la nouvelle fiche avec le message « Saison {libellé} créée en brouillon. » ; succès d'ouverture → message « Saison {libellé} ouverte. » ; 409 → message d'erreur dans la région `role="status"` et `invalidateSeasons` ; focus rendu au déclencheur après fermeture du dialogue
- [X] T032 [US2] Ajouter à `web/e2e/admin-seasons.spec.ts` (après le bloc US1, même session de données) : ouvrir la saison courante (dialogue, confirmation, badge « Ouverte », « Supprimer » absent) ; « Dupliquer » → fiche de la saison suivante en « Brouillon », dates décalées d'un an ; « Dupliquer » encore → la fiche de la troisième affiche « Ouvrez d'abord la saison … » et aucun bouton « Ouvrir la saison » ; revenir à la deuxième et l'ouvrir → la liste montre deux saisons « Ouverte » et une seule « Saison courante » ; « Nouvelle saison » → date de début en lecture seule ; axe sur la fiche et le dialogue

**Checkpoint**: quickstart scénarios 6 à 10 et 16.

---

## Phase 5: User Story 3 - Corriger ou supprimer une saison (Priority: P2)

**Goal**: modification des dates selon l'état et la position (décalage de la suivante en brouillon), suppression du dernier brouillon

**Independent Test**: modifier la fin d'un brouillon puis d'une saison ouverte ; la date de début d'une saison ouverte n'est pas modifiable ; supprimer le dernier brouillon ; une saison close n'offre aucune action de modification

### Tests for User Story 3

- [X] T033 [P] [US3] Ajouter à `internal/season/season_test.go` les cas de `ValidateUpdate(seasons, i, startsOn *Date, endsOn Date, today) (Plan, error)` (`Plan` = nouvelles dates de la cible et éventuel nouveau début de la suivante) : première saison en brouillon, début et fin modifiés → OK ; `startsOn` fourni pour une saison qui n'est pas la première ou qui est ouverte → `ErrStartLocked` ; saison close → `ErrClosed` ; fin < today → `ErrEndInPast` ; fin = today sur une saison ouverte → OK ; suivante en brouillon → `Plan` décale son début au lendemain ; suivante ouverte → `ErrNextOpen` ; nouvelle fin qui rendrait la suivante vide (≥ `N.ends_on − 1 jour`) ou longue de plus de 24 mois → `FieldError{endsOn, ErrNextInvalid}` portant le libellé de la suivante ; et de `CanDelete` : brouillon et dernière → OK ; ouverte ou close → `ErrDeleteDraftOnly` ; brouillon non dernière → `ErrDeleteLastOnly`
- [X] T034 [P] [US3] Écrire `internal/api/seasons_update_test.go` (httptest, `SetToday("2026-10-31")`) : première saison en brouillon, `PATCH /api/admin/seasons/{id}` `{startsOn: "2026-08-15", endsOn: "2027-07-31"}` → 200, dates et libellé à jour ; après ouverture, `PATCH` avec `startsOn` → 409 `MsgSeasonStartLocked` ; la saison ouverte suivie d'un brouillon a `shiftsSeasonLabel: "2027-2028"` (et `null` dès que la suivante est ouverte) ; `PATCH {endsOn: "2027-08-15"}` sur cette saison → 200 et le brouillon commence le 2027-08-16 (relu par `GET`) ; `endsOn: "2026-10-30"` → 422 `body.endsOn` « La date de fin ne peut pas être passée. » ; suivante ouverte → 409 `MsgNextSeasonOpen` ; saison close (`SetToday` après sa fin) → 409 `MsgSeasonClosed` ; `DELETE /api/admin/seasons/{id}` sur le dernier brouillon → 204 puis 404 ; sur une saison ouverte → 409 `MsgDeleteDraftOnly` ; sur un brouillon non dernier → 409 `MsgDeleteLastOnly` ; lignes de journal `season.update` et `season.delete` ; membre → 403, anonyme → 401

### Implementation for User Story 3

- [X] T035 [US3] Implémenter dans `internal/season/season.go` `ValidateUpdate`, le type `Plan` et `CanDelete`, en cohérence avec `ActionsFor` et `EndsOnBounds` ; faire passer T033
- [X] T036 [US3] Implémenter dans `internal/api/seasons.go` : `update-season` (`PATCH /admin/seasons/{id}`, corps `{ startsOn *string format:"date" omitempty, endsOn string format:"date" }` ; `WriteTx` : `ListSeasons`, 404, `season.ValidateUpdate`, `UpdateSeasonDates` sur la cible puis, si le plan le prévoit, sur la suivante ; erreurs via `seasonError` (409 pour `ErrClosed`, `ErrStartLocked`, `ErrNextOpen`) ; `logSeasonAction("season.update", id, "starts_on", …, "ends_on", …)`) et `delete-season` (`DELETE /admin/seasons/{id}`, `DefaultStatus: 204` ; `WriteTx` : `ListSeasons`, 404, `season.CanDelete` → 409, `DeleteSeason` ; `logSeasonAction("season.delete")`) ; `go tool task gen` ; faire passer T034
- [X] T037 [US3] Ajouter à `web/app/routes/app/admin/season-detail.tsx` : « Modifier les dates » (`Button` secondaire, si `editEndsOn`) → `Dialog` base-ui titré « Modifier les dates de la saison {libellé} » contenant `SeasonDatesFields` (début seulement si `editStartsOn`, bornes `endsOnMin`/`endsOnMax`, aide « La saison {shiftsSeasonLabel} commencera le lendemain. » si `shiftsSeasonLabel` est non nul — aucune déduction côté client, principe VI) ; `$api.useMutation("patch", "/admin/seasons/{id}")`, erreurs 422 sur les champs, 409 dans une `Alert` du dialogue, succès → fermeture, `invalidateSeasons`, message « Dates de la saison {libellé} enregistrées. » ; « Supprimer » (`Button` `danger-secondary`, si `delete`) → `ConfirmDialog` (`danger`) « Supprimer la saison {libellé} ? », « Cette action est définitive. », succès → `/admin/saisons` avec le message « Saison {libellé} supprimée. »
- [X] T038 [US3] Ajouter à `web/e2e/admin-seasons.spec.ts` (après le bloc US2, qui laisse : 1re saison courante ouverte, 2e ouverte, 3e en brouillon) : sur la fiche de la **1re** saison, « Modifier les dates » est **absent** (la suivante est ouverte, FR-017) ; sur la fiche de la **2e** saison (ouverte, suivie du brouillon), « Modifier les dates » ouvre un dialogue qui n'affiche que « Date de fin », dont l'attribut `min` vaut le lendemain de son début (début futur, donc postérieur à la date du jour), avec l'aide « La saison {libellé de la 3e} commencera le lendemain. » ; avancer sa fin d'un mois → la fiche de la 3e affiche le nouveau début ; supprimer la 3e (dernier brouillon : dialogue, confirmation) → retour à la liste et message « Saison … supprimée. » ; « Supprimer » absent sur les saisons ouvertes ; axe sur le dialogue de modification

**Checkpoint**: quickstart scénarios 11 à 13.

---

## Phase 6: User Story 4 - Clôture automatique et saison courante (Priority: P2)

**Goal**: état close et saison courante calculés à chaque lecture, sans action

**Independent Test**: avec une saison finie hier et la suivante en cours, la première est close et la seconde courante, sans aucune action

### Tests for User Story 4

- [X] T039 [US4] Écrire `internal/api/seasons_clock_test.go` (httptest) : saisons 2025-09-01 → 2026-08-31 (ouverte) et 2026-09-01 → 2027-08-31 (ouverte) créées avec `SetToday("2026-08-01")` ; `SetToday("2026-08-31")` → la première est `open` et `isCurrent` ; `SetToday("2026-09-01")` → la première est `closed`, toutes ses `actions` à `false`, la seconde `isCurrent`, `setup: "ready"` ; brouillon dont la fin passe (`SetToday` au lendemain) → `closed` et `actions.delete: false` ; `SetToday` après la fin de la dernière → `currentId: null`, `setup: "no-current"`, la dernière close garde `actions.duplicate: true` ; vérifier que `Deps.Now` est la seule source de la date du jour (aucune lecture SQL de `date('now')`)

### Implementation for User Story 4

- [X] T040 [US4] Relire `internal/api/seasons.go` et `internal/season/season.go` pour garantir que toute lecture de « aujourd'hui » passe par `s.today()` (jamais `time.Now()` direct ni `date('now')` en SQL) et que chaque opération recalcule état, saison courante et actions sur la suite relue ; corriger les écarts révélés par T039

**Checkpoint**: quickstart scénario 14.

---

## Phase 7: User Story 5 - Être guidé depuis le tableau de bord (Priority: P3)

**Goal**: encart de mise en place selon `setup`

**Independent Test**: base sans saison → encart et lien ; créer et ouvrir une saison en cours → l'encart disparaît

### Implementation for User Story 5

- [X] T041 [US5] Modifier `web/app/routes/app/admin/home.tsx` : `clientLoader` qui précharge `seasonListQuery` (erreur ignorée : le tableau de bord reste affiché) ; au-dessus de la section de bienvenue, `Alert` selon `setup` (contracts/ui-routes.md › Tableau de bord) : `missing` → `info` « Créez la première saison pour mettre en place Frigg. » + lien « Créer la saison » vers `/admin/saisons/nouvelle` ; `no-current` → `info` « Aucune saison n'est en cours. » + lien « Créer la saison suivante » ; `current-draft` → `warning` « La saison {libellé de currentId} a commencé mais n'est pas encore ouverte. » + lien « Voir la saison » vers `/admin/saisons/{currentId}` ; `ready` → rien ; texte de bienvenue : « Créez les saisons de l'association depuis la rubrique « Saisons » et invitez les personnes depuis « Comptes ». Les autres rubriques s'ouvriront au fil des prochaines versions. »
- [X] T042 [US5] Compléter `web/e2e/admin-seasons.spec.ts` : **en tête du fichier**, avant le bloc US1, `/admin` affiche « Créez la première saison pour mettre en place Frigg. » et le lien mène à `/admin/saisons/nouvelle` ; **après la création** du bloc US1, `/admin` affiche l'encart « … n'est pas encore ouverte » avec le lien vers la fiche ; **après l'ouverture** du bloc US2, `/admin` n'affiche plus d'encart de saison ; axe sur le tableau de bord avec encart

**Checkpoint**: quickstart scénario 2 et fin du scénario 4.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T043 [P] Vérifier que les autres fichiers e2e qui visitent `/admin` (`web/e2e/navigation.spec.ts`, `web/e2e/keyboard.spec.ts`) restent verts avec ou sans encart de saison (exécution parallèle, état des saisons non déterministe pour eux) ; ajuster leurs sélecteurs si besoin sans dépendre de l'encart
- [X] T044 [P] Vérifier dans `internal/api/access_test.go` que les 7 opérations `Saisons` sont bien parcourues par la matrice d'autorisation (anonyme → 401, membre → 403, administrateur accepté) ; compléter les corps de requête minimaux si la matrice en fournit par opération
- [X] T045 Lancer `go tool task check` (build `CGO_ENABLED=0`, vet, tests Go, génération sans diff, migrations sur base vierge, typecheck, Biome, Vitest, build, Playwright + axe) et corriger jusqu'au vert
- [X] T046 Dérouler les scénarios manuels de `specs/003-season-management/quickstart.md` (1 à 18) et cocher les écarts constatés dans ce fichier de tâches

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance
- **Foundational (Phase 2)** : après le Setup ; bloque toutes les stories
- **US1 (Phase 3)** : après la phase 2 ; crée la fiche, les routes et le fichier e2e enrichis par les stories suivantes
- **US2, US3 (Phases 4–5)** : après US1 (actions ajoutées à `season-detail.tsx`, blocs ajoutés au fichier e2e en série)
- **US4 (Phase 6)** : après la phase 2 côté domaine ; ses tests API supposent les opérations d'US1 à US3 (création, ouverture)
- **US5 (Phase 7)** : après US1 (et US2 pour le dernier contrôle e2e)
- **Polish (Phase 8)** : après les stories voulues

### User Story Dependencies

```text
Setup → Foundational ── US1 ─┬─ US2 ─┬─ US3
                             │       └─ US4 (tests API)
                             └─ US5 (après US2 pour l'e2e complet)
```

US2 et US3 touchent `internal/season/season.go`, `internal/api/seasons.go`,
`season-detail.tsx` et `admin-seasons.spec.ts` : à enchaîner.

### Within Each User Story

- Tests rédigés d'abord ; ils doivent échouer avant l'implémentation
- Domaine (`internal/season`) avant les opérations d'API
- Opérations d'API, puis `go tool task gen` (OpenAPI + types TypeScript), puis écrans
- Bloc e2e + axe en dernier, à sa place dans le fichier en série

### Parallel Opportunities

- Phase 2 : T004, T006, T011, T012, T013 en parallèle ; T005 après T004 ; T007 après T005 et T006 ; T003 après T002 ; T008 → T009 → T010 après T003 et T007
- US1 : T014, T015, T016 en parallèle ; T019 en parallèle de T017–T018 ; T020 après T018 ; T021 → T022, T023 → T024, T025
- US2 : T027 et T028 en parallèle
- US3 : T033 et T034 en parallèle
- Polish : T043 et T044 en parallèle

---

## Parallel Example: Phase 2

```bash
Task: "T004 internal/season/date_test.go"
Task: "T006 internal/season/season_test.go"
Task: "T011 web/app/lib/format.ts (formatDay) + format.test.ts"
Task: "T012 web/app/shell/navigation.ts + Sidebar.tsx (section settings)"
Task: "T013 internal/tools/e2eseed/main.go (admin-saisons)"
```

## Parallel Example: User Story 1

```bash
# Tests en parallèle :
Task: "T014 internal/season/season_test.go (ValidateCreate)"
Task: "T015 internal/api/seasons_list_test.go"
Task: "T016 web/app/shell/navigation.test.ts"

# Puis T017 → T018 (API) en parallèle de T019 (navigation), puis T020 → T021 → T022…T025 → T026
```

---

## Implementation Strategy

### MVP First

1. Phases 1 et 2 (table, domaine, horloge, navigation)
2. Phase 3 (US1) : les administrateurs créent et consultent les saisons
3. **STOP and VALIDATE** : quickstart scénarios 1, 3, 4, 5 ; `go tool task check`

### Incremental Delivery

1. Socle + US1 → première saison créée, l'outil est « mis en place »
2. US2 → passage d'une saison à l'autre et pré-inscriptions
3. US3 → correction des erreurs de saisie
4. US4 → garantie de la clôture automatique (tests d'horloge)
5. US5 → prise en main guidée depuis le tableau de bord
