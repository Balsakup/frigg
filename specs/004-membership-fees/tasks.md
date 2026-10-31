---

description: "Liste des tâches : gestion des tarifs de l'adhésion"
---

# Tasks: Gestion des tarifs de l'adhésion

**Input**: Design documents from `specs/004-membership-fees/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : tests d'autorisation (principes II et III), tests
d'API, Playwright + axe sur chaque parcours touché (Workflow). Les règles
(montant, nettoyage, unicité repliée, plafond, actions) sont couvertes par des
tests en table.

**Organization**: une phase par user story, dans l'ordre de priorité de la spec
(US1 et US2 en P1, US3 en P2, US4 en P3), puis la phase 8, évolution décidée
après essai (montant de renouvellement).

**Conventions** (constitution 1.4.0) : identifiants, tables, champs JSON et
valeurs en anglais ; commentaires, messages, textes d'interface en français.
Commentaires des requêtes sqlc en ASCII uniquement (sqlc échoue sur un
commentaire non ASCII). Nouvelle migration `00005_membership_fees.sql`
(append-only). Montants en centimes entiers (`int64`), jamais en flottant côté
Go.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US4)

## Path Conventions

Un module Go à la racine (`internal/`) et le SPA dans `web/` (voir plan.md ›
Project Structure). Code existant à réutiliser : `internal/api/seasons.go`
(`seasonWrite`, `toDomain`, `logSeasonAction`, `MsgSeasonGone`,
`MsgSeasonClosed`, `s.today()`, `nowString()`), `internal/api/errors.go`
(`FieldErrors(location, messages...)`), `internal/season` (`StateOf`,
`Closed`, `FieldError`), helpers de test `internal/api/apitest` (`New`,
`Seed`, `LoggedIn`, `SetToday`, `Client().Do`, `LogBuffer`) et
`internal/api/seasons_helpers_test.go` ; frontend : `web/app/api/seasons.ts`,
`web/app/api/problem.ts` (`applyProblemToForm`, `isProblem`),
`web/app/ui/ConfirmDialog.tsx`, le `EditDatesDialog` de
`web/app/routes/app/admin/season-detail.tsx` comme modèle de dialogue ;
e2e : `web/e2e/test.ts`, `web/e2e/axe.ts`. Régénération : `go tool task gen`
(sqlc, OpenAPI, types TypeScript).

**Date de référence des exemples** : « aujourd'hui » = 2026-10-31
(Europe/Paris) dans les tests Go, via `env.SetToday`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: package de repli du texte prévu par la constitution

- [X] T001 [P] Écrire `internal/textfold/textfold_test.go` (tests en table, doivent échouer avant T002) : `Fold("Tarif réduit") == Fold("tarif reduit") == Fold("  TARIF  RÉDUIT ")` ; `Fold("Élève")` = `"eleve"` ; `Fold("Ça")` = `"ca"` ; `Fold("")` = `""` ; espaces insécables (` `, ` `) traités comme des espaces ; `Fold("Plein tarif") != Fold("Plein-tarif")`
- [X] T002 Créer `internal/textfold/textfold.go` : commentaire de package « Repli du texte saisi pour les comparaisons sans casse ni accents (constitution, principe VI). » ; `func Fold(s string) string` : découpe par `strings.Fields` (supprime espaces de début et de fin et réduit les espaces intérieurs, insécables compris : remplacer d'abord ` ` et ` ` par un espace), décomposition `norm.NFD` de `golang.org/x/text/unicode/norm`, suppression des runes `unicode.Mn`, `strings.ToLower` ; passer `golang.org/x/text` en dépendance directe (`go mod tidy`) ; vérifier `CGO_ENABLED=0 go list -deps ./cmd/frigg` ; faire passer T001

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: table, requêtes, règles du domaine, types et enregistrement de l'API, formatage des montants

**⚠️ CRITICAL**: aucune story ne peut commencer avant la fin de cette phase

### Base de données

- [X] T003 Créer `internal/database/migrations/00005_membership_fees.sql` : `CREATE TABLE membership_fees (id INTEGER PRIMARY KEY, season_id INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE, label TEXT NOT NULL CHECK (length(label) BETWEEN 1 AND 80), amount_cents INTEGER NOT NULL CHECK (amount_cents BETWEEN 0 AND 1000000), description TEXT CHECK (description IS NULL OR length(description) BETWEEN 1 AND 500), created_at TEXT NOT NULL, updated_at TEXT NOT NULL) STRICT;` puis `CREATE INDEX membership_fees_season_id ON membership_fees (season_id);` avec commentaires français (« Tarifs de l'adhésion d'une saison (research.md 004 › R1). », « Montant en centimes d'euro. », « Unicité du libellé sans casse ni accents contrôlée par le domaine (R2). ») ; `-- +goose Down` : `DROP TABLE membership_fees;`
- [X] T004 Créer `internal/database/queries/membership_fees.sql` (commentaires ASCII) : `ListMembershipFees :many` (`SELECT * FROM membership_fees WHERE season_id = sqlc.arg(season_id) ORDER BY id`) ; `CreateMembershipFee :one` (`INSERT INTO membership_fees (season_id, label, amount_cents, description, created_at, updated_at) VALUES (sqlc.arg(season_id), sqlc.arg(label), sqlc.arg(amount_cents), sqlc.narg(description), sqlc.arg(now), sqlc.arg(now)) RETURNING *`) ; `UpdateMembershipFee :execrows` (`UPDATE membership_fees SET label = …, amount_cents = …, description = sqlc.narg(description), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id) AND season_id = sqlc.arg(season_id)`) ; `DeleteMembershipFee :execrows` (`DELETE … WHERE id = sqlc.arg(id) AND season_id = sqlc.arg(season_id)`) ; `CopyMembershipFees :execrows` (`INSERT INTO membership_fees (season_id, label, amount_cents, description, created_at, updated_at) SELECT sqlc.arg(target_id), label, amount_cents, description, sqlc.arg(now), sqlc.arg(now) FROM membership_fees WHERE season_id = sqlc.arg(source_id)`) ; lancer `go generate ./internal/database` et vérifier le type `sqlc.MembershipFee` (`Description sql.NullString`)

### Domaine (`internal/season/fee.go`)

- [X] T005 [P] Écrire `internal/season/fee_test.go` (tests en table, doivent échouer avant T006) : **ParseAmount** — `"30"` → 3000, `"15,5"` → 1550, `"15.50"` → 1550, `"0"` → 0, `"1 000"` → 100000, `"1 000,00"` → 100000, `"10000"` → 1000000, `" 12 "` → 1200 ; `"-1"`, `"1,234"`, `"abc"`, `""`, `"1,"`, `",5"`, `"1.2.3"` → `FieldError{amount, ErrAmountFormat}` ; `"0030"` → 3000, `"00000000030"` → 3000 (zéros de tête ignorés) ; `"10000,01"`, `"12345678"`, `"99999999999999999999"` → `FieldError{amount, ErrAmountTooHigh}` ; **ValidateFee** — libellé et description débarrassés des espaces de début et de fin ; libellé vide après nettoyage → `FieldError{label, ErrLabelEmpty}` ; description vide après nettoyage → absente (`nil`) ; libellé dont `textfold.Fold` égale celui d'un autre tarif de la saison → `FieldError{label, ErrLabelTaken}` ; même libellé que le tarif modifié lui-même → OK ; **erreurs cumulées** (FR-017) — libellé vide et montant `"abc"` → `FieldErrors` contenant `label/ErrLabelEmpty` et `amount/ErrAmountFormat` ; libellé déjà pris et montant `"20000"` → `label/ErrLabelTaken` et `amount/ErrAmountTooHigh` ; création avec 20 tarifs existants → `ErrTooManyFees` (avant tout contrôle de champ) ; modification avec 20 tarifs → OK ; **FeeActions** — `Draft`/`Open` et 19 tarifs → `{Add: true, Edit: true, Delete: true}` ; 20 tarifs → `Add: false` ; `Closed` → tout faux ; **SortFees** — montant décroissant, puis libellé replié (« élève » avant « Famille »), puis id
- [X] T006 Implémenter `internal/season/fee.go` (commentaire : « Règles des tarifs de l'adhésion (research.md 004 › R4). ») : constantes `MaxFees = 20`, `MaxAmountCents = 1_000_000`, `MaxFeeLabel = 80`, `MaxFeeDescription = 500` ; type `Fee { ID int64; Label string; AmountCents int64; Description *string }` ; erreurs sentinelles `ErrAmountFormat`, `ErrAmountTooHigh`, `ErrLabelEmpty`, `ErrLabelTaken`, `ErrTooManyFees` ; `ParseAmount(s string) (int64, error)` (retire espaces, ` `, ` ` ; remplace `,` par `.` ; motif `^\d+(\.\d{1,2})?$`, sinon `ErrAmountFormat` ; zéros de tête de la partie entière retirés, puis plus de 5 chiffres restants → `ErrAmountTooHigh` sans conversion, ce qui exclut tout débordement ; calcul entier, sans `strconv.ParseFloat` ; résultat > `MaxAmountCents` → `ErrAmountTooHigh`) ; type `FieldErrors []*FieldError` (implémente `error`, messages joints) ; `type FeeInput { Label string; AmountCents int64; Description *string }` ; `ValidateFee(existing []Fee, id int64, label, amount string, description *string) (FeeInput, error)` (`id` nul = création) : à la création, plafond d'abord (`ErrTooManyFees`) ; puis nettoyage du libellé et de la description, `ParseAmount`, et unicité via `textfold.Fold` si le libellé n'est pas vide ; **toutes** les erreurs de champ sont cumulées et renvoyées ensemble dans un `FieldErrors` (FR-017) ; `type FeeActions { Add, Edit, Delete bool }` et `FeeActionsFor(state State, count int) FeeActions` ; `SortFees([]Fee)` ; erreurs de champ via le `FieldError` existant (`Field` = `label`, `amount`) ; faire passer T005

### API

- [X] T007 Créer `internal/api/membership_fees.go` : `tagFees = "Tarifs"` ; messages `MsgFeeGone = "Ce tarif n'existe plus."`, `MsgTooManyFees = "Une saison ne peut pas compter plus de 20 tarifs."`, `MsgFeeLabelEmpty = "Saisissez un libellé."`, `MsgFeeLabelTaken = "Un tarif porte déjà ce nom dans cette saison."`, `MsgAmountFormat = "Saisissez un montant en euros, avec au plus deux décimales (par exemple 15,50)."`, `MsgAmountTooHigh = "Le montant ne peut pas dépasser 10 000 €."` ; types (contracts/api.md › Types) `MembershipFee` (`id`, `label`, `amountCents int64`, `description *string`, `createdAt`, `updatedAt time.Time`), `MembershipFeeActions` (`add`, `edit`, `delete`), `MembershipFeeList` (`items`, `actions`), `MembershipFeeBody` (`label string maxLength:"80"`, `amount string`, `description string maxLength:"500" required:"false"`) ; helpers `toFee(sqlc.MembershipFee) season.Fee`, `toFeeOutput(sqlc.MembershipFee) MembershipFee` ; dans `internal/api/errors.go`, helper `FieldErrorDetails(details ...*huma.ErrorDetail) error` (422 « Les données envoyées sont invalides. » avec une entrée par détail, emplacements possiblement différents ; `FieldErrors` existant inchangé) ; `feeError(err) error` qui traduit `season.FieldErrors` en un seul 422 via `FieldErrorDetails` (une entrée `body.label` / `body.amount` par erreur, message traduit) et `ErrTooManyFees` → `409 MsgTooManyFees` ; `logFeeAction(ctx, action string, seasonID, feeID int64, attrs ...any)` en `INFO` « action d'administration sur un tarif » avec `action`, `actor_id`, `season_id`, `fee_id` (sans libellé, research.md › R10) ; helper `feeWrite(ctx, seasonID, fn func(q *sqlc.Queries, state season.State, fees []season.Fee) error) error` sur le modèle de `seasonWrite` : `WriteTx`, `ListSeasons` + `toDomain`, saison absente → 404 `MsgSeasonGone`, état via `season.StateOf(…, s.today())` ; `Closed` → 409 `MsgSeasonClosed` ; `ListMembershipFees` ; `func (s *server) registerMembershipFees(api huma.API)` vide pour l'instant
- [X] T008 Ajouter l'appel `s.registerMembershipFees(api)` après `s.registerSeasons(api)` dans `internal/api/api.go`

### Frontend

- [X] T009 [P] Ajouter dans `web/app/lib/format.ts` `formatAmount(cents: number)` (`0` → « Gratuit », sinon `Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" })` sur `cents / 100`) et `amountInput(cents: number)` (valeur de préremplissage du champ : « 30 » si entier, sinon « 15,50 ») ; compléter `web/app/lib/format.test.ts` : `formatAmount(1550)` correspond à `/^15,50\s€$/`, `formatAmount(0) === "Gratuit"`, `amountInput(3000) === "30"`, `amountInput(1550) === "15,50"`, `amountInput(5) === "0,05"`

**Checkpoint**: `go tool task gen` sans erreur, `go test ./...` vert (T001, T005), migration appliquée sur base vierge.

---

## Phase 3: User Story 1 - Définir les tarifs d'une saison (Priority: P1) 🎯 MVP

**Goal**: section « Tarifs de l'adhésion » dans la fiche, état vide, ajout, liste triée

**Independent Test**: saison en brouillon sans tarif ; constater l'état vide, ajouter « Plein tarif » 30 et « Tarif réduit » 15,50 avec description, les retrouver triés ; en membre, l'API répond 403

### Tests for User Story 1

- [X] T010 [P] [US1] Écrire `internal/api/membership_fees_test.go` (httptest via `apitest`, `env.SetToday("2026-10-31")`, saison 2026-09-01 → 2027-08-31 créée par l'API) : `GET /api/admin/seasons/{id}/fees` → `items: []`, `actions: {add: true, edit: true, delete: true}` ; `POST …/fees` `{label: "Plein tarif", amount: "30"}` → 201, `amountCents: 3000`, `description: null` ; `{label: "  Tarif réduit ", amount: "15,50", description: "Étudiants"}` → 201, `label: "Tarif réduit"` ; `{label: "Membre d'honneur", amount: "0"}` → 201 ; la liste est triée Plein tarif, Tarif réduit, Membre d'honneur ; erreurs 422 (`errors[].location`, messages de contracts/api.md) : `label: "plein TARIF"` → `body.label` `MsgFeeLabelTaken` ; `label: "   "` → `body.label` `MsgFeeLabelEmpty` ; libellé de 81 caractères → `body.label` ; `amount: "-5"`, `"12,345"` → `body.amount` `MsgAmountFormat` ; `"20000"` → `body.amount` `MsgAmountTooHigh` ; description de 501 caractères → `body.description` ; `{label: "", amount: "abc"}` → un seul 422 avec deux entrées, `body.label` `MsgFeeLabelEmpty` et `body.amount` `MsgAmountFormat` (FR-017) ; saison inconnue → 404 `MsgSeasonGone` ; 21e tarif → 409 `MsgTooManyFees` et `actions.add: false` à 20 ; **concurrence** : deux goroutines `POST` du même libellé → une 201 et une 422, un seul tarif en base ; ligne de journal `fee.create` avec `season_id`, `fee_id`, `amount_cents` et sans le libellé ; membre → 403, anonyme → 401 sur liste et création

### Implementation for User Story 1

- [X] T011 [US1] Implémenter dans `internal/api/membership_fees.go` : `list-membership-fees` (`GET /admin/seasons/{id}/fees`, `auth.Admin` ; lecture sur `s.Store.Read` : `ListSeasons`, saison absente → 404, `ListMembershipFees`, `season.SortFees`, `FeeActionsFor(StateOf(…, s.today()), len)`) et `create-membership-fee` (`POST /admin/seasons/{id}/fees`, `DefaultStatus: 201` ; `feeWrite` : `ValidateFee(fees, 0, label, amount, description)` (toutes les erreurs de champ en un seul 422), `CreateMembershipFee` avec les valeurs de `FeeInput` avec `now = nowString()` ; `logFeeAction("fee.create", …, "amount_cents", cents)`) ; puis `go tool task gen` ; faire passer T010
- [X] T012 [P] [US1] Créer `web/app/api/membership-fees.ts` sur le modèle de `web/app/api/seasons.ts` : types `MembershipFee`, `MembershipFeeList` depuis `components["schemas"]` ; `membershipFeesQuery(seasonId)` (`$api.queryOptions("get", "/admin/seasons/{id}/fees", …)`) ; `invalidateMembershipFees(qc)` qui invalide `["get", "/admin/seasons/{id}/fees"]`
- [X] T013 [US1] Créer `web/app/routes/app/admin/FeeDialog.tsx` (contracts/ui-routes.md › Dialogue), sur le modèle de `EditDatesDialog` : `Dialog` base-ui, props `seasonId`, `fee?: MembershipFee` (absent = ajout), `trigger`, `onDone(message)` ; titre « Ajouter un tarif » ou « Modifier le tarif {libellé} » ; champs `Field` « Libellé » (`maxLength={80}`, `required`), « Montant » (`type="text"`, `inputMode="decimal"`, suffixe visuel « € » `aria-hidden`, aide « 0 pour un tarif gratuit. », prérempli par `amountInput` en modification), « Description » facultative (`textarea`, `maxLength={500}`, aide « Affichée sous le libellé, par exemple les conditions du tarif. ») ; aucune validation client au-delà des attributs natifs ; ajout via `$api.useMutation("post", "/admin/seasons/{id}/fees")` ; erreurs 422 via `applyProblemToForm(error, setError, ["label", "amount", "description"])` ; `404`/`409` → `Alert` `danger` dans le dialogue et `invalidateMembershipFees` + `invalidateSeasons` ; succès → fermeture, `invalidateMembershipFees`, `onDone("Tarif {libellé} ajouté.")` ; boutons « Ajouter » / « Enregistrer » (principal) et « Annuler »
- [X] T014 [US1] Créer `web/app/routes/app/admin/SeasonFees.tsx` (contracts/ui-routes.md › Section) : props `season: Season`, `onMessage` ; `useQuery(membershipFeesQuery(season.id))` ; `section` étiquetée par un `h2` « Tarifs de l'adhésion » ; quand la saison a au moins un tarif : bouton « Ajouter un tarif » (secondaire, icône `Plus` via `Icon`) dans l'en-tête de section si `actions.add`, sinon si `actions.edit` texte « 20 tarifs au plus par saison. » ; sans tarif, aucun bouton dans l'en-tête (un seul bouton « Ajouter un tarif » sur la page, celui de l'état vide) ; état vide : « Un tarif est le prix que paie un adhérent pour adhérer pendant la saison. » et bouton principal « Ajouter un tarif » si `actions.add`, ou « Aucun tarif n'a été défini pour cette saison. » sinon ; `Table` caption « Tarifs de la saison {libellé} », colonnes Tarif (`th scope="row"`, description en texte secondaire dessous), Montant (aligné à droite, `formatAmount`) ; erreur de chargement → `Alert` `danger` « Impossible de charger les tarifs. Rechargez la page. »
- [X] T015 [US1] Modifier `web/app/routes/app/admin/season-detail.tsx` : le `clientLoader` précharge aussi `membershipFeesQuery(id)` (`.catch(() => null)`) ; `SeasonView` rend `<SeasonFees season={season} onMessage={…} />` sous la liste de définitions, avec les messages de succès dans la région `role="status"` existante
- [X] T016 [US1] Ajouter à `web/e2e/admin-seasons.spec.ts` un second test « tarifs (004) » dans le `describe` en série (après le test des saisons, qui laisse la 1re saison courante ouverte et la 2e ouverte et dernière ; compte `admin-saisons@exemple.fr`) : ouvrir la fiche de la 2e saison ; section « Tarifs de l'adhésion », état vide et bouton « Ajouter un tarif » ; axe ; ajouter « Plein tarif » 30 puis « Tarif réduit » 15,50 avec la description « Étudiants et demandeurs d'emploi », puis « Membre d'honneur » 0 → tableau dans l'ordre Plein tarif « 30,00 € », Tarif réduit « 15,50 € » et sa description, Membre d'honneur « Gratuit » ; ajouter « plein TARIF » → erreur « Un tarif porte déjà ce nom dans cette saison. » sur le champ Libellé ; montant « 12,345 » → erreur sur le champ Montant ; axe avec le dialogue ouvert et à 320 px sans défilement horizontal de la page

**Checkpoint**: US1 utilisable seule — quickstart scénarios 2, 3, 4.

---

## Phase 4: User Story 2 - Reprendre les tarifs en dupliquant la saison (Priority: P1)

**Goal**: la duplication recopie les tarifs ; la création manuelle n'en crée aucun

**Independent Test**: saison portant deux tarifs ; dupliquer ; la nouvelle porte les mêmes tarifs ; en modifier un, l'original est inchangé

### Tests for User Story 2

- [X] T017 [P] [US2] Écrire `internal/api/seasons_duplicate_test.go` (httptest, `SetToday("2026-10-31")`) : saison 2026-2027 avec « Plein tarif » (3000) et « Tarif réduit » (1550, description « Étudiants ») ; `POST /api/admin/seasons/{id}/duplicate` → 201 ; `GET /api/admin/seasons/{newId}/fees` → mêmes libellés, montants et descriptions, identifiants différents ; la ligne de journal `season.duplicate` porte `fees_copied: 2` ; ajouter un tarif à la copie (`POST`) → la liste de la source est inchangée (l'indépendance en modification est vérifiée par T020 et T024) ; dupliquer une saison sans tarif → copie sans tarif, `fees_copied: 0` ; `POST /api/admin/seasons` (création manuelle après la dernière) → saison sans tarif ; copie entièrement passée (comblement : saison close dupliquée avec `SetToday` postérieur) → tarifs recopiés et `actions` toutes à `false`

### Implementation for User Story 2

- [X] T018 [US2] Modifier `duplicateSeason` dans `internal/api/seasons.go` : dans la même fonction de transaction, après `CreateSeason`, appeler `q.CopyMembershipFees(ctx, sqlc.CopyMembershipFeesParams{TargetID: row.ID, SourceID: in.ID, Now: nowString()})` et conserver le nombre de lignes ; ajouter `"fees_copied", n` aux attributs de `logSeasonAction("season.duplicate", …)` (research.md › R7) ; faire passer T017
- [X] T019 [US2] Compléter le test « tarifs (004) » de `web/e2e/admin-seasons.spec.ts` : depuis la fiche de la 2e saison (dernière, portant les trois tarifs), « Dupliquer » → la fiche de la nouvelle saison affiche les trois mêmes tarifs dans le même ordre

**Checkpoint**: quickstart scénarios 8 et 9.

---

## Phase 5: User Story 3 - Modifier ou supprimer un tarif (Priority: P2)

**Goal**: modification complète et suppression confirmée tant que la saison n'est pas close ; lecture seule sur une saison close

**Independent Test**: sur une saison en brouillon, modifier le montant d'un tarif puis en supprimer un autre ; sur une saison close, aucune action n'est proposée

### Tests for User Story 3

- [X] T020 [P] [US3] Ajouter à `internal/api/membership_fees_test.go` : `PUT /api/admin/seasons/{id}/fees/{feeId}` `{label: "Plein tarif", amount: "32", description: "Adultes"}` → 200, `amountCents: 3200`, `updatedAt` postérieur ou égal à `createdAt` ; sans `description` → `description: null` ; renommer un tarif avec le libellé replié d'un autre (« tarif REDUIT ») → 422 `body.label` ; garder son propre libellé en changeant la casse → 200 ; même chose sur une saison **ouverte** (`POST …/open`) → 200 (FR-008) ; `DELETE …/fees/{feeId}` → 204 puis `PUT` et `DELETE` du même tarif → 404 `MsgFeeGone` ; supprimer le dernier tarif d'une saison ouverte → 204 ; `feeId` d'une autre saison → 404 ; saison close (`SetToday` au lendemain de sa fin) : liste avec `actions` toutes à `false`, `POST`, `PUT`, `DELETE` → 409 `MsgSeasonClosed` ; suppression d'une saison en brouillon portant des tarifs (`DELETE /api/admin/seasons/{id}`) → 204 et plus aucune ligne `membership_fees` pour cette saison (requête directe sur `env.Store`, vérifie `foreign_keys=ON`) ; lignes de journal `fee.update` (avec `amount_cents`) et `fee.delete` ; membre → 403, anonyme → 401 sur `PUT` et `DELETE`

### Implementation for User Story 3

- [X] T021 [US3] Implémenter dans `internal/api/membership_fees.go` : `update-membership-fee` (`PUT /admin/seasons/{id}/fees/{feeId}`, entrée `MembershipFeeBody` ; `feeWrite` : tarif absent de `fees` → 404 `MsgFeeGone`, `ValidateFee(fees, feeID, label, amount, description)`, `UpdateMembershipFee` avec les valeurs de `FeeInput` ; réponse relue ; `logFeeAction("fee.update", …, "amount_cents", cents)`) et `delete-membership-fee` (`DELETE /admin/seasons/{id}/fees/{feeId}`, `DefaultStatus: 204` ; `feeWrite` : `DeleteMembershipFee`, 0 ligne → 404 `MsgFeeGone` ; `logFeeAction("fee.delete", …)`) ; `go tool task gen` ; faire passer T020
- [X] T022 [US3] Compléter `web/app/routes/app/admin/FeeDialog.tsx` pour la modification : mutation `$api.useMutation("put", "/admin/seasons/{id}/fees/{feeId}")` quand `fee` est fourni, valeurs initiales (libellé, `amountInput(fee.amountCents)`, description), message « Tarif {libellé} modifié. »
- [X] T023 [US3] Compléter `web/app/routes/app/admin/SeasonFees.tsx` : colonne Actions (en-tête « Actions » visuellement masqué) présente seulement si `actions.edit` ou `actions.delete` ; par ligne, bouton icône `Pencil` « Modifier le tarif {libellé} » (nom accessible) qui ouvre `FeeDialog` en modification, et bouton icône `Trash2` « Supprimer le tarif {libellé} » → `ConfirmDialog` (`danger`) « Supprimer le tarif {libellé} ? », « Cette action est définitive. » ; suppression via `$api.useMutation("delete", "/admin/seasons/{id}/fees/{feeId}")`, succès → `invalidateMembershipFees`, message « Tarif {libellé} supprimé. », focus rendu au bouton « Ajouter un tarif » (la ligne a disparu) ; `404`/`409` → message d'erreur dans la région `role="status"` et invalidation des tarifs et de la fiche
- [X] T024 [US3] Compléter le test « tarifs (004) » de `web/e2e/admin-seasons.spec.ts` : sur la fiche de la saison dupliquée (brouillon), modifier « Plein tarif » en 32 → « 32,00 € » ; vérifier sur la fiche de la 2e saison que « Plein tarif » vaut toujours « 30,00 € » ; sur la saison dupliquée, supprimer « Membre d'honneur » (dialogue, confirmation) → la ligne disparaît et le message « Tarif Membre d'honneur supprimé. » s'affiche ; axe avec le dialogue de confirmation

**Checkpoint**: quickstart scénarios 5, 6, 10, 12.

---

## Phase 6: User Story 4 - Être prévenu qu'une saison sans tarif ne sera pas proposée (Priority: P3)

**Goal**: message informatif sur la fiche d'une saison non close sans tarif ; ouverture non bloquée

**Independent Test**: saison non close sans tarif → message ; ajouter un tarif → message disparu ; l'ouverture reste proposée

### Tests for User Story 4

- [X] T025 [P] [US4] Ajouter à `internal/api/membership_fees_test.go` : une saison en brouillon sans tarif dont la précédente est ouverte garde `actions.open: true` dans `GET /api/admin/seasons/{id}` et `POST …/open` → 200 (FR-015, règles de 003 inchangées)

### Implementation for User Story 4

- [X] T026 [US4] Compléter `web/app/routes/app/admin/SeasonFees.tsx` : si `actions.edit` est vrai et que `items` est vide, `Alert` `info` « Cette saison ne sera pas proposée à l'adhésion tant qu'elle n'a aucun tarif. » en tête de section (research.md › R8), avant l'état vide ; rien sur une saison close
- [X] T027 [US4] Compléter le test « tarifs (004) » de `web/e2e/admin-seasons.spec.ts` : au début du test (2e saison sans tarif, ouverte), le message « Cette saison ne sera pas proposée à l'adhésion tant qu'elle n'a aucun tarif. » est visible ; après le premier ajout, il disparaît

**Checkpoint**: quickstart scénarios 1, 6 et 7.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T028 [P] Vérifier dans `internal/api/access_test.go` que les 4 opérations `Tarifs` sont parcourues par la matrice d'autorisation (anonyme → 401, membre → 403, administrateur accepté, y compris avec `{feeId}` inexistant → 404 accepté) ; compléter si la matrice demande des paramètres de chemin ou des corps minimaux
- [X] T029 [P] Vérifier que `admin-seasons.spec.ts` reste le seul fichier e2e à créer des saisons et que les autres (`navigation.spec.ts`, `keyboard.spec.ts`) ne dépendent ni des saisons ni des tarifs
- [X] T030 Lancer `go tool task check` (build `CGO_ENABLED=0`, vet, tests Go, génération sans diff, migrations sur base vierge, typecheck, Biome, Vitest, build, Playwright + axe) et corriger jusqu'au vert
- [X] T031 Dérouler les scénarios manuels de `specs/004-membership-fees/quickstart.md` (1 à 14) et consigner les écarts dans ce fichier de tâches

---

## Phase 8: Évolution — montant de renouvellement (FR-002a, 2026-10-31)

**Goal**: un tarif peut porter un montant de renouvellement distinct (case « Tarif de renouvellement différent »), recopié par la duplication ([R12](./research.md#r12-montant-de-renouvellement))

**Independent Test**: ajouter « Adulte compétition » 85 € avec renouvellement 55 € ; les deux montants s'affichent ; case cochée sans montant → erreur sur le champ ; la duplication recopie les deux montants

- [X] T032 [US1] Créer `internal/database/migrations/00006_membership_fee_renewal.sql` : `ALTER TABLE membership_fees ADD COLUMN renewal_amount_cents INTEGER CHECK (renewal_amount_cents IS NULL OR renewal_amount_cents BETWEEN 0 AND 1000000);` (commentaire français : « Montant de renouvellement en centimes ; NULL = même montant pour toute adhésion (research.md 004 › R12). ») ; `-- +goose Down` : `ALTER TABLE membership_fees DROP COLUMN renewal_amount_cents;` ; ajouter `renewal_amount_cents` (`sqlc.narg(renewal_amount_cents)`) à `CreateMembershipFee`, `UpdateMembershipFee` et `CopyMembershipFees` dans `internal/database/queries/membership_fees.sql` ; `go generate ./internal/database`
- [X] T033 [P] [US1] Compléter `internal/season/fee_test.go` : `ValidateFee` avec `FeeDraft{…, RenewalAmount: new("55")}` → `RenewalAmountCents` = 5500 ; `RenewalAmount: nil` → `nil` ; `new("")` ou `new("   ")` → `FieldError{renewalAmount, ErrRenewalRequired}` ; `new("abc")` → `renewalAmount/ErrAmountFormat` ; `new("20000")` → `renewalAmount/ErrAmountTooHigh` ; renouvellement supérieur au montant principal → OK ; erreurs cumulées avec le libellé et le montant principal
- [X] T034 [US1] Modifier `internal/season/fee.go` : `ValidateFee(existing []Fee, id int64, d FeeDraft) (FeeInput, error)` avec `FeeDraft { Label, Amount string; RenewalAmount, Description *string }` (`RenewalAmount` nil = pas de montant distinct) ; `FeeInput` et `Fee` gagnent `RenewalAmountCents *int64` ; erreur `ErrRenewalRequired` ; les erreurs de `ParseAmount` sur le renouvellement sont rattachées au champ `renewalAmount` ; faire passer T033
- [X] T035 [P] [US1] Compléter `internal/api/membership_fees_test.go` et `internal/api/seasons_duplicate_test.go` : `POST` `{label, amount: "85", hasRenewalAmount: true, renewalAmount: "55"}` → `renewalAmountCents: 5500` ; sans `hasRenewalAmount` → `null` (même si `renewalAmount` est fourni) ; `hasRenewalAmount: true` sans montant → 422 `body.renewalAmount` « Saisissez le montant du renouvellement. » ; `PUT` avec `hasRenewalAmount: false` → `renewalAmountCents: null` ; la duplication recopie `renewalAmountCents`
- [X] T036 [US1] Modifier `internal/api/membership_fees.go` : `MembershipFeeBody` gagne `HasRenewalAmount bool json:"hasRenewalAmount,omitempty" required:"false"` et `RenewalAmount string json:"renewalAmount,omitempty" required:"false"` ; `MembershipFee` gagne `RenewalAmountCents *int64 json:"renewalAmountCents"` ; message `MsgRenewalRequired = "Saisissez le montant du renouvellement."` ; `feeError` traduit le champ `renewalAmount` (format et plafond avec les messages du montant) ; création et modification passent `renewal_amount_cents` ; `go tool task gen` ; faire passer T035
- [X] T037 [P] [US1] Ajouter dans `web/app/routes/app/admin/FeeDialog.tsx` la case native « Tarif de renouvellement différent » (`register("hasRenewalAmount")`, `watch` pour afficher le champ) ; cochée : champ « Montant du renouvellement » (`inputMode="decimal"`, aide « En euros. », erreur `body.renewalAmount`) et libellé « Montant de première adhésion » pour le montant principal ; préremplissage depuis `renewalAmountCents` via `amountInput` ; corps envoyé : `hasRenewalAmount` et `renewalAmount` ; `applyProblemToForm` reçoit aussi `renewalAmount`
- [X] T038 [P] [US1] Afficher dans `web/app/routes/app/admin/SeasonFees.tsx`, colonne Montant, une seconde ligne « Renouvellement {formatAmount(renewalAmountCents)} » (texte secondaire) quand `renewalAmountCents` n'est pas nul
- [X] T039 [US1] Compléter le test « tarifs (004) » de `web/e2e/admin-seasons.spec.ts` : ajouter « Adulte compétition » 85 avec renouvellement 55 → ligne avec « 85,00 € » et « Renouvellement 55,00 € » ; case cochée sans montant → erreur « Saisissez le montant du renouvellement. » ; après duplication, la copie montre les deux montants ; ajuster les comptes de lignes des étapes suivantes
- [X] T040 Relancer les contrôles de T030 (`go vet`, `go test`, génération sans diff, typecheck, Biome, Vitest, Playwright + axe)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance ; `textfold` est requis par T006
- **Foundational (Phase 2)** : après le Setup ; bloque toutes les stories
- **US1 (Phase 3)** : après la phase 2 ; crée la section, le dialogue et le test e2e enrichis ensuite
- **US2 (Phase 4)** : côté API, seulement après la phase 2 et T011 (création de tarifs pour le jeu de test) ; côté e2e, après T016
- **US3 (Phase 5)** : après US1 (`FeeDialog.tsx`, `SeasonFees.tsx`) ; son e2e suit celui d'US2 (saison dupliquée)
- **US4 (Phase 6)** : après T014 ; son e2e s'insère au début du test « tarifs (004) »
- **Polish (Phase 7)** : après les stories voulues

### User Story Dependencies

```text
Setup → Foundational ── US1 ─┬─ US2 ── US3 (e2e)
                             ├─ US3 (API et écran)
                             └─ US4
```

US1, US3 et US4 touchent `SeasonFees.tsx` ; US1 à US4 complètent le même test
e2e en série : à enchaîner dans l'ordre des phases.

### Within Each User Story

- Tests rédigés d'abord ; ils doivent échouer avant l'implémentation
- Domaine (`internal/season/fee.go`) avant les opérations d'API
- Opérations d'API, puis `go tool task gen` (OpenAPI + types TypeScript), puis écrans
- Bloc e2e + axe en dernier

### Parallel Opportunities

- Phase 1 : T001 en parallèle de la lecture du reste ; T002 après T001
- Phase 2 : T005 et T009 en parallèle ; T004 après T003 ; T006 après T002 et T005 ; T007 → T008 après T004 et T006
- US1 : T010 en parallèle de T012 ; T011 après T010 ; T013 → T014 → T015 → T016
- US2 : T017 en parallèle des tâches frontend d'US1
- US3 : T020 en parallèle de T022–T023
- US4 : T025 en parallèle de T026
- Polish : T028 et T029 en parallèle

---

## Parallel Example: Phase 2

```bash
Task: "T005 internal/season/fee_test.go"
Task: "T009 web/app/lib/format.ts (formatAmount, amountInput) + format.test.ts"
```

## Parallel Example: User Story 1

```bash
Task: "T010 internal/api/membership_fees_test.go"
Task: "T012 web/app/api/membership-fees.ts"
# Puis T011 (API) → go tool task gen → T013 → T014 → T015 → T016
```

---

## Implementation Strategy

### MVP First

1. Phases 1 et 2 (textfold, table, domaine, types d'API)
2. Phase 3 (US1) : les administrateurs définissent les tarifs d'une saison
3. **STOP and VALIDATE** : quickstart scénarios 2 à 4 ; `go tool task check`

### Incremental Delivery

1. Socle + US1 → tarifs saisis et consultés
2. US2 → tarifs repris d'une saison à l'autre
3. US3 → corrections et suppressions, lecture seule des saisons closes
4. US4 → rappel des saisons sans tarif
