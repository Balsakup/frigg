---

description: "Liste des tâches : champs personnalisés des membres et des adhésions"
---

# Tasks: Champs personnalisés des membres et des adhésions

**Input**: Design documents from `specs/011-custom-fields/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : matrice d'autorisation (principes II et III), tests
d'API sur base isolée, Playwright + axe sur chaque parcours touché. La
non-divulgation des champs réservés (FR-020) a son propre test de fuite.

**Organization**: une phase par user story, dans l'ordre de priorité de la spec.
US1 (P1) livre la section « Champs personnalisés » (définitions). US2 (P1) la
saisie des champs de membre, US3 (P1) celle des champs d'adhésion. US4 (P2)
les filtres et la recherche. US5 (P3) la suppression définitive.

**Conventions** (constitution 1.8.0) : identifiants, tables, champs JSON et
valeurs d'énumération en anglais (`short_text`, `editable`…) ; commentaires,
messages, textes d'interface et journaux en français. Commentaires de la
migration et des requêtes sqlc en **ASCII uniquement** (sqlc sqlite casse sur
un commentaire non ASCII). Jamais de valeur ni de libellé de champ
personnalisé dans les journaux. Aucun test de rôle dans les handlers :
l'audience (`Admin` ou `Owner`) dépend de l'espace de l'opération (R3).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US5)

## Path Conventions

Module Go à la racine (`internal/`), SPA dans `web/` (plan.md › Project
Structure). Code existant à réutiliser :

- API : `internal/api/api.go` (`register`, `server`, `s.today()`,
  `CurrentAccount`, `s.internalError`, `s.txError`), `internal/api/errors.go`
  (`FieldErrors`, détails `huma.ErrorDetail{Location, Message}`),
  `internal/api/settings.go` (`tagSettings`), `internal/api/members.go`
  (`Member`, `AdminMember`, `MemberBody`, `logMemberAction`),
  `internal/api/members_admin.go` (`ListMembersInput`, `AdminMemberBody`,
  `listMembers`, `createMember`, `updateMember`, `memberOutput`),
  `internal/api/members_me.go` (`MyMemberBody`, `ownAccount`,
  `myMemberOutput`, `createMyMember`, `updateMyMember`),
  `internal/api/memberships_admin.go` (`AdminMembership`, `DecisionBody`,
  `AdminCreateBody`, `ListMembershipsInput`, `ns`),
  `internal/api/memberships_me.go` (`MyMembership`, `MyMembershipBody`,
  `myMembershipOutput`, `myMembershipWrite`, `changed`).
- Domaine : `internal/textfold` (`Fold`), `internal/season` (`IsCurrent`,
  `Date`), `internal/member` (modèle de `FieldError`/`FieldErrors`).
- Base : `internal/database` (`Store.Read`, `Store.WriteTx` en `BEGIN
  IMMEDIATE`, `FTSTrigramQuery`), `internal/database/queries/members.sql`
  (`ListMembers`), `internal/database/queries/memberships.sql`
  (`ListMemberships`), migration `00007_members.sql` (modèle FTS + triggers).
- Tests Go : `internal/api/apitest` (`New`, `Seed`, `LoggedIn`, `SetToday`,
  `Logs`, `Store`, `Client().Do`, `Response.JSON`),
  `internal/api/members_helpers_test.go` (`memberBody`, `myMemberPath`,
  `adminMemberPath`, `createMine`, `insertForAccount`),
  `internal/api/memberships_helpers_test.go` (`newMsFixture`, `request`,
  `validate`, `adminMemberships`, `setPayment`), `internal/api/access_test.go`
  (`TestAccessMatrix`), `internal/database/db_test.go`.
- Frontend : `web/app/routes.ts`, `web/app/routes/app/admin/SettingsPage.tsx`
  (`settingsSections`), `web/app/routes/app/admin/settings-payments.tsx`
  (modèle de section), `web/app/api/*` (`queries.ts`, `members.ts`,
  `memberships.ts`, `problem.ts` : `applyProblemToForm`), `web/app/ui/*`
  (`Button`, `Field`, `Checkbox`, `RadioGroup`, `Select`, `Badge`, `Alert`,
  `ConfirmDialog`, `Icon`), `web/app/lib/format.ts`,
  `web/app/lib/member-filters.ts`, `web/app/lib/membership-filters.ts`,
  `web/app/members/{AdminMemberForm,MyMemberForm,MemberFields}.tsx`. E2E :
  `web/e2e/test.ts`, `web/e2e/axe.ts`, `web/e2e/admin-members.spec.ts`,
  `web/e2e/memberships.spec.ts`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: schéma et requêtes communs à toutes les stories.

- [X] T001 Créer la migration `internal/database/migrations/00011_custom_fields.sql` (commentaires ASCII) selon data-model.md, tables `STRICT` : `custom_fields` (`id INTEGER PRIMARY KEY`, `scope TEXT NOT NULL CHECK (scope IN ('member','membership'))`, `label TEXT NOT NULL CHECK (length(label) BETWEEN 1 AND 80)`, `type TEXT NOT NULL CHECK (type IN ('short_text','long_text','number','date','bool','choice'))`, `visibility TEXT NOT NULL DEFAULT 'admin' CHECK (visibility IN ('admin','visible','editable'))`, `required INTEGER NOT NULL DEFAULT 0 CHECK (required IN (0,1))`, `help TEXT CHECK (help IS NULL OR length(help) BETWEEN 1 AND 200)`, `position INTEGER NOT NULL`, `archived_at TEXT`, `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`, `CHECK (required = 0 OR type <> 'bool')`), index `custom_fields_scope_position (scope, archived_at, position)` ; `custom_field_choices` (`id`, `field_id INTEGER NOT NULL REFERENCES custom_fields (id) ON DELETE CASCADE`, `label TEXT NOT NULL CHECK (length(label) BETWEEN 1 AND 80)`, `position INTEGER NOT NULL`), index `(field_id, position)` ; `member_field_values` (`id`, `member_id INTEGER NOT NULL REFERENCES members (id) ON DELETE CASCADE`, `field_id INTEGER NOT NULL REFERENCES custom_fields (id) ON DELETE CASCADE`, `value_text TEXT CHECK (value_text IS NULL OR length(value_text) BETWEEN 1 AND 200)`, `value_long TEXT CHECK (value_long IS NULL OR length(value_long) BETWEEN 1 AND 2000)`, `value_number REAL`, `value_date TEXT CHECK (value_date IS NULL OR date(value_date) = value_date)`, `value_bool INTEGER CHECK (value_bool IN (0,1))`, `value_choice_id INTEGER REFERENCES custom_field_choices (id) DEFERRABLE INITIALLY DEFERRED` (NO ACTION différé, **pas** `ON DELETE RESTRICT` que SQLite applique immédiatement même différé : voir data-model.md), `updated_at TEXT NOT NULL`, `UNIQUE (member_id, field_id)`, CHECK « exactement une des six colonnes `value_*` est non NULL » : `((value_text IS NOT NULL) + (value_long IS NOT NULL) + (value_number IS NOT NULL) + (value_date IS NOT NULL) + (value_bool IS NOT NULL) + (value_choice_id IS NOT NULL)) = 1`), index `(field_id)` et `(value_choice_id)` ; `membership_field_values` identique avec `membership_id INTEGER NOT NULL REFERENCES memberships (id) ON DELETE CASCADE` et `UNIQUE (membership_id, field_id)` ; table `member_field_values_fts USING fts5(value_text, content = 'member_field_values', content_rowid = 'id', tokenize = "trigram remove_diacritics 1")` et triggers `member_field_values_fts_ai`, `_ad`, `_au` (`AFTER UPDATE OF value_text`) entre `-- +goose StatementBegin/End`, sur le modèle de `members_fts` (00007) ; section `Down` qui supprime triggers, FTS et tables dans l'ordre inverse
- [X] T002 [P] Test de migration in `internal/database/db_test.go` (`TestCustomFieldsMigration`) : base vierge ; base migrée jusqu'à `00010` avec une fiche et une adhésion, puis `00011` ; CHECK refusant un `required = 1` sur un `bool`, une valeur avec deux colonnes renseignées, une valeur sans colonne, une date `2026-02-30` ; `DELETE` d'un choix utilisé refusé à la validation de la transaction ; suppression d'une fiche → ses valeurs supprimées et `INSERT INTO member_field_values_fts(member_field_values_fts) VALUES ('integrity-check')` sans erreur ; suppression d'un champ portant une valeur de choix → valeurs et choix supprimés sans erreur, index intègre ; `Down` puis `Up` sans erreur
- [X] T003 Créer `internal/database/queries/custom_fields.sql` (commentaires ASCII) : `ListCustomFields` (tous, `ORDER BY scope, archived_at IS NOT NULL, position`), `ListActiveCustomFields` (`scope`, actifs, par position), `ListOwnerCustomFields` (`scope`, actifs **et** `visibility <> 'admin'`), `GetCustomField`, `CreateCustomField` (`RETURNING *`), `UpdateCustomField`, `SetCustomFieldPosition`, `ArchiveCustomField`, `RestoreCustomField` (position donnée), `DeleteCustomField`, `MaxCustomFieldPosition` (`scope`), `CountActiveCustomFields` (`scope`), `ListCustomFieldChoices` (par `field_id` et position), `ListChoicesOfFields` (tous les choix des champs actifs d'une portée), `CreateCustomFieldChoice`, `UpdateCustomFieldChoice`, `DeleteCustomFieldChoice`, `CountChoiceUses` (les deux tables de valeurs), `CountCustomFieldValues` (`GROUP BY field_id`, les deux tables), `CountMembersMissingField` (fiches sans ligne pour un champ), `CountMembershipsMissingField` (adhésions `pending`/`validated` d'une saison sans ligne) ; valeurs : `ListMemberFieldValues` (`member_id`), `UpsertMemberFieldValue` (`INSERT … ON CONFLICT (member_id, field_id) DO UPDATE`, les six colonnes remises à jour), `DeleteMemberFieldValue`, et les trois équivalents `…MembershipFieldValue…` ; lancer `sqlc generate` (code commité in `internal/database/sqlc/`)

**Checkpoint**: schéma et requêtes générés ; `go build ./...` et `go test ./...` passent ; comportement inchangé.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: domaine pur et fonctions partagées de lecture et d'écriture des valeurs, utilisés par toutes les stories.

**⚠️ CRITICAL**: aucune story ne commence avant la fin de cette phase.

- [X] T004 [P] Créer `internal/customfield/definition.go` (commentaire de paquet en français : règles pures des champs personnalisés, sans base) : types `Scope` (`member`, `membership`), `Type` (`short_text`, `long_text`, `number`, `date`, `bool`, `choice`), `Visibility` (`admin`, `visible`, `editable`), `Choice{ID int64; Label string}`, `Definition{ID; Scope; Label; Type; Visibility; Required bool; Help *string; Choices []Choice; Archived bool}` ; `FieldError{Field, Message string}` et `FieldErrors` ; constantes `MaxActive = 30`, `MaxLabel = 80`, `MaxHelp = 200`, `MinChoices = 2`, `MaxChoices = 50` ; `ValidateDefinition(d DefinitionInput, existing []Definition, excludeID int64) (DefinitionInput, FieldErrors)` : libellé et choix nettoyés (espaces de bord, espaces multiples), libellé « 1 à 80 caractères » unique dans la portée **archivés compris** par `textfold.Fold` (« Un champ porte déjà ce libellé. »), aide vide → nil, « au plus 200 caractères », `choice` : « 2 à 50 choix distincts de 1 à 80 caractères » (doublons par `textfold.Fold`, erreur sur `choices[i]`), autres types : aucun choix, `Required` refusé pour `bool` (« Un champ Oui/non ne peut pas être obligatoire. »)
- [X] T005 [P] Créer `internal/customfield/value.go` : `Input{FieldID int64; Text *string; Date *string; Checked *bool; ChoiceID *int64}`, `Value{Text, Long *string; Number *float64; Date *string; Bool *bool; ChoiceID *int64}` (au plus une colonne), `IsEmpty()` ; `NormalizeValue(d Definition, in Input) (Value, error)` : au plus une propriété renseignée et compatible avec le type, sinon « Valeur incompatible avec ce champ. » ; `short_text` : espaces de bord retirés, une seule ligne, « 200 caractères au plus. » ; `long_text` : retours à la ligne conservés (`\r\n` → `\n`), « 2 000 caractères au plus. » ; `number` : `Text` saisi, espaces et espaces insécables retirés, virgule ou point décimal, `strconv.ParseFloat`, fini et `|x| < 1e12`, sinon « Saisissez un nombre. » ; `date` : `AAAA-MM-JJ` valide (« Saisissez une date valide. ») ; `bool` : `false` → valeur vide (suppression) ; `choice` : `ChoiceID` parmi `d.Choices` (« Choix inconnu. ») ; texte vide ou aucune propriété → valeur vide (effacement)
- [X] T006 [P] Créer `internal/customfield/apply.go` : `Audience` (`Admin`, `Owner`), `CanRead(d, a)` (`Owner` : `visible` ou `editable`), `CanWrite(d, a)` (`Admin` : toujours ; `Owner` : `editable`) ; `Apply(defs []Definition, current map[int64]Value, inputs []Input, a Audience) (Plan, FieldErrors)` : `defs` = champs **actifs** de la portée déjà filtrés pour l'audience ; `fieldId` en double ou inconnu de `defs` ou non `CanWrite` → « Ce champ n'existe pas ou n'est pas modifiable. » (même message, R3) ; normalisation de chaque entrée ; un champ absent de `inputs` est inchangé ; puis pour chaque champ `Required` et `CanWrite` dont la valeur finale est vide : « Ce champ est obligatoire. » ; erreurs avec `Field = "customFields.f<id>"` ; `Plan{Upserts map[int64]Value; Deletes []int64; Changed int}` sans opération pour une valeur identique
- [X] T007 [P] Tests en table in `internal/customfield/definition_test.go`, `value_test.go`, `apply_test.go` : libellé 80/81 runes avec accents, doublon « Numéro de licence » / « numero de LICENCE » (actif et archivé), `excludeID` ; choix 1/2/50/51, choix en double, choix sur un type non `choice` ; obligatoire sur `bool` ; `short_text` 200/201, saut de ligne refusé ; `long_text` 2000/2001 ; nombres « 12,5 », « 12.5 », « 1 234,5 », « -3 », « abc », « 1e13 », « NaN » ; dates `2026-02-28`, `2026-02-30`, `28/02/2026` ; `checked: false` → effacement ; choix d'un autre champ ; matrice visibilités × audiences de `CanRead`/`CanWrite` ; `Apply` : champ absent inchangé, champ `visible` écrit en `Owner` refusé, champ inconnu refusé avec le même message, obligatoire manquant en `Owner` seulement s'il est `editable`, obligatoire manquant en `Admin` pour tout champ, plusieurs erreurs ensemble, valeur identique sans opération
- [X] T008 Créer `internal/api/custom_field_values.go` : types huma de contracts/api.md › Types (`Choice`, `FormField`, `FieldValue` avec `text`, `number`, `date` `format:"date"`, `checked`, `choiceId` nullable, `FieldInput` avec `fieldId`, `text` `maxLength:"2000"`, `date` `format:"date"`, `checked`, `choiceId` facultatifs) ; `loadDefinitions(ctx, q *sqlc.Queries, scope, audience)` (requête `ListActiveCustomFields` en `Admin`, `ListOwnerCustomFields` en `Owner`, choix via `ListChoicesOfFields`) ; `memberFieldValues(ctx, q, memberID, defs)` et `membershipFieldValues(ctx, q, membershipID, defs)` → `[]FieldValue` pour **tous** les champs de `defs` dans l'ordre (non renseignés compris, `checked: false` pour un `bool` vide), `editable = CanWrite` ; `writeMemberFields(ctx, q, memberID, defs, inputs, audience) (int, error)` et `writeMembershipFields(...)` : lecture des valeurs courantes, `customfield.Apply`, conversion des `FieldErrors` en un seul `422` avec `Location: "body." + Field`, exécution des upserts et suppressions ; fonction `customFieldErrors(errs customfield.FieldErrors) []error` réutilisable pour fusionner avec les erreurs de la fiche ; `maxItems:"30"` sur les listes `customFields`
- [X] T009 Créer `internal/api/custom_fields.go` avec `func (s *server) registerCustomFields(api huma.API)` (vide à ce stade) appelée depuis `New` in `internal/api/api.go`, et la journalisation `s.logCustomFieldAction(ctx, action string, fieldID int64, scope string)` (« écriture sur un champ personnalisé », attributs `action`, `actor_id`, `field_id`, `scope`, jamais de libellé ni de valeur, R11)

**Checkpoint**: `go test ./internal/customfield/...` passe ; `go build ./...` passe.

---

## Phase 3: User Story 1 - Définir des champs personnalisés (Priority: P1) 🎯 MVP

**Goal**: un administrateur crée, modifie, réordonne, archive et rétablit des champs de membre et d'adhésion dans « Paramètres › Champs personnalisés ».

**Independent Test**: créer un champ de chaque type pour les deux portées, en modifier un, en réordonner deux, en archiver un, recharger la section et retrouver l'état attendu ; un membre est refusé.

### Tests for User Story 1

- [X] T010 [P] [US1] Tests d'API in `internal/api/custom_fields_test.go` : `list-custom-fields` vide ; `create-custom-field` de chaque type → `201`, ajouté en fin de liste, `visibility` par défaut `admin` ; libellé en double sans casse ni accents → `422 body.label` ; choix 1 et 51, choix vide, choix en double → `422 body.choices`/`body.choices[i]` ; `required` sur `bool` → `422 body.required` ; 31e champ actif → `422 body.scope` ; `update-custom-field` libellé, aide, visibilité, obligatoire, ajout et renommage de choix → `200` ; changement de type sans valeur → `200`, avec une valeur (insérée par `env.Store`) → `409` ; retrait d'un choix utilisé → `422 body.choices` ; retrait d'un choix inutilisé → `200` ; `reorder-custom-fields` complet → nouvel ordre ; liste incomplète, avec un id archivé ou d'une autre portée → `409` ; `archive-custom-field` puis `restore-custom-field` → champ en fin de liste, valeurs conservées ; archivage d'un archivé et rétablissement d'un actif → `409` ; rétablissement à 30 actifs → `422` ; `valueCount` et `missingCount` (champ de membre obligatoire, et champ d'adhésion obligatoire avec et sans saison en cours via `env.SetToday`) ; `list-my-custom-fields` en membre : seuls les champs actifs `visible`/`editable`, `editable` correct ; journaux `custom_field.create|update|reorder|archive|restore` sans libellé (`env.Logs`)
- [X] T011 [P] [US1] Vérifier que la matrice `internal/api/access_test.go` couvre les 6 opérations admin de US1 et `list-my-custom-fields` (anonyme → `401`, membre → `403` pour les opérations admin, membre accepté pour `list-my-custom-fields`) sans modifier `allowedPublic`

### Implementation for User Story 1

- [X] T012 [US1] Opérations de définition in `internal/api/custom_fields.go` (contracts/api.md › Définitions), tag `settings`, `auth.Admin` : `list-custom-fields` (`GET /admin/custom-fields` → `{member, membership}` : actifs par position puis archivés par libellé, `valueCount` via `CountCustomFieldValues`, `missingCount` pour un champ obligatoire actif : `CountMembersMissingField` ou `CountMembershipsMissingField` sur la saison `season.IsCurrent`, `null` sans saison en cours), `create-custom-field` (`POST`, `201` ; `ValidateDefinition` contre tous les champs de la portée ; `CountActiveCustomFields` < 30 sinon « 30 champs actifs au plus : archivez un champ inutile. » sur `body.scope` ; position = `MaxCustomFieldPosition + 1` ; choix créés dans l'ordre), `update-custom-field` (`PUT /admin/custom-fields/{id}` ; `409` « Le type ne peut plus changer : des valeurs sont enregistrées. » si le type change et `valueCount > 0` ; choix avec `id` renommés, sans `id` créés, absents retirés sauf `CountChoiceUses > 0` → `422 body.choices` « Le choix « … » est utilisé : renommez-le plutôt que de le retirer. » ; positions des choix réécrites), `reorder-custom-fields` (`PUT /admin/custom-fields/order`, `{scope, ids}` ; `409` « L'ordre a changé entre-temps : rechargez la page. » si `ids` ≠ ensemble des actifs de la portée), `archive-custom-field` et `restore-custom-field` (`POST …/archive`, `…/restore` ; `409` si déjà dans l'état ; restauration : limite de 30 puis position en fin de liste) ; toutes les écritures en `WriteTx` ; `404` « Ce champ n'existe plus. » ; journal par action
- [X] T013 [US1] Opération `list-my-custom-fields` in `internal/api/custom_fields.go` (`GET /me/custom-fields`, `auth.Member`, tag `members`) → `{member: FormField[], membership: FormField[]}` via `loadDefinitions(…, customfield.Owner)`
- [X] T014 [US1] Régénérer le contrat avec `go tool task gen` (`web/app/api/openapi.json`, `web/app/api/schema.d.ts`), fichiers commités
- [X] T015 [P] [US1] Requêtes et mutations in `web/app/api/custom-fields.ts` : `customFieldsQuery()` (`list-custom-fields`), `myCustomFieldsQuery()` (`list-my-custom-fields`), `useCreateCustomField`, `useUpdateCustomField`, `useReorderCustomFields`, `useArchiveCustomField`, `useRestoreCustomField`, chacune invalidant `list-custom-fields` et `list-my-custom-fields`
- [X] T016 [P] [US1] Section dans `web/app/routes/app/admin/SettingsPage.tsx` (`settingsSections` : « Champs personnalisés » `/admin/parametres/champs`, entre Apparence et Paiements) et `web/app/routes.ts` (`route("champs", "routes/app/admin/settings-custom-fields.tsx")` sous `parametres`)
- [X] T017 [US1] Dialogue `web/app/customfields/FieldDialog.tsx` (contracts/ui-routes.md › Dialogue) : `Dialog` base-ui, `react-hook-form` ; Libellé (requis, compteur 80), Type (`RadioGroup` six options, désactivé avec l'aide « Le type ne peut plus changer : des valeurs sont enregistrées. » si `valueCount > 0`), Choix (type Liste de choix : lignes éditables, « Ajouter un choix », « Retirer » ; `id` conservé pour les choix existants), Visibilité (`RadioGroup`, défaut « Réservé à l'administration », une phrase d'explication par option), Obligatoire (`Checkbox` masquée pour Oui/non), Texte d'aide (compteur 200) ; `422` par `applyProblemToForm` (dont `choices.<i>`) ; notifications « Champ ajouté. » / « Champ enregistré. »
- [X] T018 [US1] Écran `web/app/routes/app/admin/settings-custom-fields.tsx` (contracts/ui-routes.md › Contenu) : `clientLoader` (`ensureQueryData(customFieldsQuery())`) ; `SettingsPage section="Champs personnalisés"` ; deux régions `h2` « Champs des membres » et « Champs des adhésions » avec leur phrase d'explication ; `ol` des champs actifs : libellé, type, `Badge` de visibilité, `Badge` « Obligatoire », alerte `--warn` « n fiches sans valeur » / « n adhésions de la saison en cours sans valeur » si `missingCount > 0` ; boutons icône « Monter »/« Descendre » (libellé accessible incluant le libellé du champ, désactivés aux extrémités, focus conservé, annonce dans une région `status` « … déplacé en position 2 sur 4. ») ; « Modifier » (FieldDialog) ; menu « Plus » base-ui (Archiver) ; « Ajouter un champ » désactivé avec explication à 30 actifs ; états vides avec l'exemple de la licence fédérale ; partie repliable « Champs archivés (n) » sur `Collapsible` de `@base-ui/react` (`Collapsible.Trigger` avec `aria-expanded`, `Collapsible.Panel`, repliée par défaut) avec « Rétablir » ; mise en page utilisable à 320 px
- [X] T019 [US1] Scénario Playwright `web/e2e/custom-fields.spec.ts` (« définir les champs ») : états vides, création « Numéro de licence fédérale » (texte court, rempli par le membre) et « A payé sa licence fédérale » (oui/non, la case Obligatoire disparaît), doublon de libellé refusé avec message sur le champ, liste de choix à un seul choix refusée, déplacement au clavier conservé après rechargement, archivage puis rétablissement ; membre : `/admin/parametres/champs` refusé ; 320 px sans défilement horizontal ; axe sans violation `serious`/`critical`

**Checkpoint**: US1 livrable seule : les définitions sont gérées, sans effet encore sur les fiches.

---

## Phase 4: User Story 2 - Renseigner les champs de membre (Priority: P1)

**Goal**: les champs de membre apparaissent dans les fiches ; l'administration lit et écrit tout ; le compte gestionnaire voit les champs visibles et écrit les champs remplis par le membre ; les champs réservés ne lui sont jamais transmis.

**Independent Test**: avec un champ de chaque visibilité, renseigner les valeurs depuis l'administration, ouvrir la fiche depuis le compte gestionnaire et vérifier ce qui est affiché, modifiable ou absent (y compris dans la réponse réseau).

### Tests for User Story 2

- [X] T020 [P] [US2] Tests d'API in `internal/api/member_custom_fields_test.go` : `create-member` et `update-member` avec `customFields` de chaque type → valeurs relues dans `get-member` (`customFields` complet, dans l'ordre, non renseignés compris) ; nombre « 12,5 » → `number: 12.5` ; valeur non conforme → `422 body.customFields.f<id>` en même temps qu'une erreur `body.firstName` ; obligatoire manquant → `422` ; champ archivé dans `customFields` → `422` ; `get-my-member` : champs `visible` (`editable: false`) et `editable`, pas de champ réservé ; `update-my-member` d'un champ `editable` → `200` ; d'un champ `visible` ou réservé → `422` avec le message identique « Ce champ n'existe pas ou n'est pas modifiable. », valeur inchangée ; `update-my-member` sans `customFields` ne touche pas une valeur réservée saisie par l'administration ; obligatoire réservé vide ne bloque pas `update-my-member` mais bloque `update-member` ; obligatoire `editable` manquant bloque `create-my-member` ; suppression d'une fiche → valeurs supprimées ; journal `member.update` avec `custom_fields` = nombre modifié, sans valeur ; cloisonnement (FR-022) : ajouter à `TestMembersIsolation` in `internal/api/members_isolation_test.go` un `update-my-member` portant des `customFields` sur la fiche d'un autre compte → `404`, valeur de l'autre fiche inchangée
- [X] T021 [P] [US2] Test de fuite in `internal/api/custom_fields_leak_test.go` (`TestReservedFieldsNeverReachOwner`) : un champ de membre réservé (libellé distinctif « Remarque-secrète-bureau ») avec une valeur « valeur-secrete-xyz » ; en compte gestionnaire, le corps brut des réponses de `get-my-member`, `list-my-members`, `update-my-member`, `list-my-custom-fields` ne contient ni le libellé, ni la valeur, ni `"fieldId":<id>` ni `"id":<id>` du champ ; l'erreur `422` d'une écriture sur ce champ est identique (corps comparé) à celle d'un `fieldId` inexistant

### Implementation for User Story 2

- [X] T022 [US2] Fiches de l'administration in `internal/api/members.go` et `internal/api/members_admin.go` : `AdminMember.CustomFields []FieldValue` ; `AdminMemberBody.CustomFields []FieldInput` (`required:"false"`) ; `memberOutput` charge `loadDefinitions(member, Admin)` puis `memberFieldValues` ; `createMember` et `updateMember` appellent `writeMemberFields(…, customfield.Admin)` dans la même `WriteTx`, après l'écriture de la fiche, et fusionnent les erreurs de validation de la fiche (`validateMember`) et des champs en un seul `422` quand les deux échouent ; `logMemberAction` reçoit l'attribut `custom_fields` = nombre de valeurs modifiées
- [X] T023 [US2] Fiches de l'espace membre in `internal/api/members.go` et `internal/api/members_me.go` : `Member.CustomFields []FieldValue` ; `MyMemberBody.CustomFields []FieldInput` ; `myMemberOutput` charge `loadDefinitions(member, Owner)` ; `createMyMember` et `updateMyMember` appellent `writeMemberFields(…, customfield.Owner)` dans la même transaction ; `list-my-members` inchangée (pas de champs dans la liste)
- [X] T024 [US2] Régénérer le contrat avec `go tool task gen`, fichiers commités
- [X] T025 [P] [US2] Utilitaires in `web/app/customfields/form.ts` : `toFormValues(fields: FieldValue[])` → `{ f<id>: string | boolean }` (nombre affiché avec virgule via `Intl.NumberFormat("fr-FR", { useGrouping: false })`, choix en chaîne d'id, date ISO), `toInputs(fields, values)` → `FieldInput[]` limité aux champs `editable` (texte et nombre dans `text`, date, `checked`, `choiceId` numérique ou absent si vide) ; `formFieldsAsValues(defs: FormField[])` pour les formulaires de création ; tests Vitest in `web/app/customfields/form.test.ts`
- [X] T026 [P] [US2] Composant `web/app/customfields/CustomFieldInputs.tsx` (contracts/ui-routes.md › Composants) : `fieldset` « Informations complémentaires » (rien s'il n'y a aucun champ) ; par type : `input` texte (`maxLength` 200), `textarea` (`maxLength` 2000), `input inputmode="decimal"`, `input type="date"`, `Checkbox`, `Select` avec option vide « — » si facultatif ; libellé, aide et erreur reliées (`aria-describedby`), marque obligatoire visible et vocalisée ; champ non `editable` rendu en texte (`CustomFieldValues`) ; noms de champ `customFields.f<id>` pour `react-hook-form` et `applyProblemToForm`
- [X] T027 [P] [US2] Composant `web/app/customfields/CustomFieldValues.tsx` : `dl` en lecture ; nombre au format français, date « 12 mars 2026 » (fonctions de `web/app/lib/format.ts`, ajouter `formatNumber` si absent), Oui/Non, libellé du choix, « Non renseigné » en `--muted`
- [X] T028 [US2] Formulaires de fiche : `web/app/members/AdminMemberForm.tsx` (définitions de `customFieldsQuery()` filtrées sur les actifs de portée `member` à la création, `customFields` de la fiche ensuite) et `web/app/members/MyMemberForm.tsx` (`myCustomFieldsQuery()` à la création) intègrent `CustomFieldInputs` sous l'adresse et envoient `toInputs(...)` ; préchargement dans les `clientLoader` de `web/app/routes/app/admin/member-new.tsx`, `member-detail.tsx`, `web/app/routes/app/member/my-member-new.tsx`, `my-member-detail.tsx` ; invalidation de `list-custom-fields` après écriture (compteurs)
- [X] T029 [US2] Scénario Playwright in `web/e2e/custom-fields.spec.ts` (« champs de membre ») : l'administration saisit le numéro de licence de Lucas ; Claire voit et modifie le numéro, ne voit pas « Remarque du bureau » ; numéro obligatoire manquant refusé avec message sur le champ ; axe sans violation `serious`/`critical`

**Checkpoint**: US1 + US2 : SC-001, SC-003 (fiches) vérifiables.

---

## Phase 5: User Story 3 - Renseigner les champs d'adhésion (Priority: P1)

**Goal**: les champs d'adhésion sont saisis à la demande (champs remplis par le membre), à la validation, à la création directe et à tout moment sur une adhésion en attente ou validée, même payée ; ils repartent vides à chaque adhésion.

**Independent Test**: un champ réservé et un champ rempli par le membre ; déposer une demande, la valider en renseignant le champ réservé, constater le paiement, puis modifier le champ depuis l'adhésion validée.

### Tests for User Story 3

- [X] T030 [P] [US3] Tests d'API in `internal/api/membership_custom_fields_test.go` (fixture `newMsFixture`) : `create-my-membership` avec le champ `editable` → valeur enregistrée ; obligatoire `editable` manquant → `422 body.customFields.f<id>` ; champ réservé envoyé par le membre → `422` ; `get-membership` : tous les champs d'adhésion actifs ; `validate-membership` avec le champ réservé → valeur enregistrée ; obligatoire réservé manquant → `422` à la validation et à `create-membership` ; `update-membership-fields` sur une adhésion en attente, validée, puis **payée** (`setPayment`) → `200`, `status`, `amountCents`, `paymentStatus` inchangés ; sur une adhésion refusée ou annulée → `409` « Cette adhésion ne peut plus être modifiée. », valeurs toujours dans `get-membership` avec `editable: false` ; `update-my-membership-fields` : champ `editable` → `200`, champ `visible` → `422`, adhésion d'une autre fiche → `404` ; nouvelle adhésion de la même fiche pour une autre saison → champs vides ; `get-my-membership` sans champ réservé (étendre `TestReservedFieldsNeverReachOwner` in `internal/api/custom_fields_leak_test.go`)
- [X] T031 [P] [US3] Vérifier que la matrice `internal/api/access_test.go` couvre `update-membership-fields` (admin) et `update-my-membership-fields` (membre)

### Implementation for User Story 3

- [X] T032 [US3] Administration in `internal/api/memberships_admin.go` : `AdminMembership.CustomFields []FieldValue` (`editable` forcé à `false` si `status` ∈ `refused`, `cancelled`) chargés dans la sortie de `get-membership` ; `DecisionBody.CustomFields` et `AdminCreateBody.CustomFields` (`[]FieldInput`, facultatifs) écrits par `writeMembershipFields(…, customfield.Admin)` dans la transaction de `validateMembership` et `createMembership` ; nouvelle opération `update-membership-fields` (`PUT /admin/memberships/{id}/fields`, `auth.Admin`, tag `memberships`, corps `{customFields}` requis) : `404` si inconnue, `409` « Cette adhésion ne peut plus être modifiée. » hors `pending`/`validated`, n'écrit que dans `membership_field_values` (aucune colonne de `memberships` modifiée : ni état, ni tarif, ni paiement), réponse = sortie de `get-membership` ; journal de l'action `membership.fields` avec `membership_id` et `custom_fields`
- [X] T033 [US3] Espace membre in `internal/api/memberships_me.go` : `MyMembership.CustomFields []FieldValue` (audience `Owner`) chargés par `myMembershipOutput` ; `MyMembershipBody.CustomFields` écrits par `writeMembershipFields(…, customfield.Owner)` dans la transaction de `createMyMembership` ; nouvelle opération `update-my-membership-fields` (`PUT /me/memberships/{id}/fields`, `auth.Member`) : adhésion d'une fiche gérée seulement (même contrôle que `myMembershipWrite`, sinon `404`), `409` hors `pending`/`validated`, réponse `MyMembership`
- [X] T034 [US3] Régénérer le contrat avec `go tool task gen`, fichiers commités
- [X] T035 [P] [US3] Mutations in `web/app/api/memberships.ts` : `useUpdateMembershipFields()` et `useUpdateMyMembershipFields()`, invalidant l'adhésion, la liste des adhésions et `list-custom-fields`
- [X] T036 [US3] Écrans de l'administration : `web/app/routes/app/admin/membership-detail.tsx` (partie « Informations complémentaires » : formulaire `CustomFieldInputs` + « Enregistrer » pour une adhésion en attente ou validée ; champs inclus dans le formulaire de validation et envoyés dans `customFields` ; `CustomFieldValues` en lecture si refusée ou annulée) et `web/app/routes/app/admin/membership-new.tsx` (`CustomFieldInputs` des champs d'adhésion actifs)
- [X] T037 [US3] Écrans de l'espace membre : `web/app/routes/app/member/membership-request.tsx` (`CustomFieldInputs` des champs d'adhésion `editable` de `myCustomFieldsQuery()`, envoyés avec la demande) et `web/app/routes/app/member/membership-detail.tsx` (valeurs visibles ; champs `editable` modifiables via `useUpdateMyMembershipFields` tant que l'adhésion est en attente ou validée)
- [X] T038 [US3] Scénario Playwright in `web/e2e/custom-fields.spec.ts` (« champs d'adhésion ») : Claire demande l'adhésion avec « Taille de t-shirt » (exigée), sans voir « A payé sa licence fédérale » ; l'administration coche le champ à la validation, constate le paiement, décoche et recoche ; adhésion annulée en lecture seule ; axe sans violation `serious`/`critical`

**Checkpoint**: US1 à US3 : les deux exemples de la demande fonctionnent de bout en bout.

---

## Phase 6: User Story 4 - Retrouver les membres et adhésions selon un champ (Priority: P2)

**Goal**: filtrer les listes de l'administration par valeur d'un champ oui/non ou liste de choix ; retrouver un membre par un fragment d'un champ texte court.

**Independent Test**: valeurs variées sur plusieurs fiches et adhésions ; appliquer chaque filtre et la recherche ; comparer au jeu de données.

### Tests for User Story 4

- [X] T039 [P] [US4] Tests en table de `ParseFilter` in `internal/customfield/filter_test.go` : `4:true`, `4:false`, `7:empty`, `7:c12`, formats invalides, champ d'une autre portée, archivé, de type texte, choix d'un autre champ, `empty` sur un `bool`, 11 filtres
- [X] T040 [P] [US4] Tests d'API in `internal/api/custom_field_filters_test.go` : `list-members?field=<id>:true|false` sur un champ oui/non de membre ; `list-members?field=<id>:c<choix>|empty` ; deux filtres combinés au filtre `account` ; `list-memberships?season=…&field=<id>:false` (« A payé sa licence fédérale : Non ») : seules les adhésions de la saison non cochées, `total` exact ; filtre invalide → `422 query.field` ; `list-members?q=3456` trouve la fiche dont le numéro de licence est `1234567`, sans casse ni accents (« Ébène » / `ebe`) ; champ archivé → plus trouvé ni filtrable, rétabli → de nouveau trouvé ; valeur effacée → plus trouvée ; **performance (SC-004)** : `TestMemberSearchByFieldPerformance` insère 2 000 fiches avec un numéro de licence chacune (insertion groupée dans une transaction par `env.Store`), puis `list-members?q=<fragment de 4 chiffres>` doit répondre en moins de 2 s (mesure `time.Since` autour de l'appel HTTP), y compris combiné à un filtre `field`

### Implementation for User Story 4

- [X] T041 [US4] Créer `internal/customfield/filter.go` : `Filter{FieldID int64; Op string /* true|false|empty|choice */; ChoiceID int64}` ; `ParseFilter(raw string, defs []Definition) (Filter, error)` (format `^[0-9]+:(true|false|empty|c[0-9]+)$`, champ actif de la portée de type `bool` (`true`/`false`) ou `choice` (`empty`/`c<id>` parmi ses choix), sinon « Filtre inconnu ou périmé. ») ; `FiltersJSON([]Filter) string` (`[{"field":4,"op":"false"},…]`) ; `MaxFilters = 10`
- [X] T042 [US4] Requêtes in `internal/database/queries/members.sql` et `internal/database/queries/memberships.sql` (commentaires ASCII) : `ListMembers` reçoit `field_filters` (`CAST(sqlc.narg(field_filters) AS TEXT)`) parcouru par `json_each` selon research.md › R6 (cas `true`, `false`, `empty`, `choice` sur `member_field_values`) et étend la condition de recherche par `OR m.id IN (SELECT v.member_id FROM member_field_values v JOIN custom_fields cf ON cf.id = v.field_id AND cf.archived_at IS NULL WHERE v.id IN (SELECT rowid FROM member_field_values_fts WHERE member_field_values_fts MATCH CAST(sqlc.narg(query) AS TEXT)))` ; `ListMemberships` reçoit le même `field_filters` sur `membership_field_values` ; `sqlc generate`
- [X] T043 [US4] Handlers : `ListMembersInput.Field []string` (`query:"field"`, `maxItems:"10"`, `pattern` de contracts/api.md) in `internal/api/members_admin.go` et `ListMembershipsInput.Field []string` in `internal/api/memberships_admin.go` ; chargement des définitions actives de la portée (`Admin`), `ParseFilter` (erreurs → `422 query.field`), `FiltersJSON` passé à la requête ; doc du paramètre `q` complétée (« … ou valeur d'un champ texte court ») ; `go tool task gen`
- [X] T044 [P] [US4] Composant `web/app/customfields/CustomFieldFilters.tsx` : un `Select` par champ actif oui/non (« Tous », « Oui », « Non ») ou liste de choix (« Tous », chaque choix, « Non renseigné ») ; libellé visible = libellé du champ
- [X] T045 [P] [US4] Filtres d'URL in `web/app/lib/member-filters.ts` et `web/app/lib/membership-filters.ts` : paramètre répété `champ` (`4:false`, `7:c12`) lu, écrit, compté dans `hasActive…Filters`, converti en `field` dans `toApiQuery` ; tests Vitest in `web/app/lib/membership-filters.test.ts` et `web/app/lib/member-filters.test.ts`
- [X] T046 [US4] Écrans `web/app/routes/app/admin/members.tsx` (filtres de portée `member` à côté du rattachement, aide de recherche mentionnant les champs texte) et `web/app/routes/app/admin/memberships.tsx` (filtres de portée `membership` combinés à la saison, l'état et le paiement) ; nombre de résultats annoncé comme pour les filtres existants ; « Effacer les filtres » réinitialise aussi `champ`
- [X] T047 [US4] Scénario Playwright in `web/e2e/custom-fields.spec.ts` (« filtres ») : « A payé sa licence fédérale : Non » sur la saison en cours, filtre conservé dans l'URL, recherche `3456` dans « Membres » ; axe sans violation `serious`/`critical`

**Checkpoint**: SC-002 et SC-004 vérifiables.

---

## Phase 7: User Story 5 - Supprimer définitivement un champ (Priority: P3)

**Goal**: supprimer un champ et toutes ses valeurs après un dialogue qui annonce le nombre de valeurs perdues et propose l'archivage.

**Independent Test**: créer un champ, renseigner des valeurs, le supprimer en confirmant ; il ne figure plus nulle part et ses valeurs ont disparu.

### Tests for User Story 5

- [X] T048 [P] [US5] Tests d'API in `internal/api/custom_fields_test.go` : `delete-custom-field` d'un champ de membre avec valeurs (et d'un champ d'adhésion avec choix) → `204`, absent de `list-custom-fields`, des fiches et des adhésions, lignes de valeurs et de choix supprimées, `integrity-check` FTS sans erreur ; champ archivé supprimable ; inconnu → `404` ; journal `custom_field.delete` ; matrice d'autorisation couvrant `delete-custom-field`

### Implementation for User Story 5

- [X] T049 [US5] Opération `delete-custom-field` in `internal/api/custom_fields.go` (`DELETE /admin/custom-fields/{id}`, `auth.Admin`, `204`) : `DeleteCustomField` en `WriteTx` ; valeurs et choix supprimés par `ON DELETE CASCADE`, la contrainte différée de `value_choice_id` n'étant vérifiée qu'à la validation de la transaction (data-model.md) ; journal ; `go tool task gen`
- [X] T050 [US5] `useDeleteCustomField` in `web/app/api/custom-fields.ts` ; dans `web/app/routes/app/admin/settings-custom-fields.tsx`, action « Supprimer » (menu « Plus » des actifs et ligne des archivés) ouvrant `ConfirmDialog` danger « Supprimer le champ « … » ? », avec, si `valueCount > 0`, « Les n valeurs enregistrées seront définitivement perdues. Pour les conserver, archivez plutôt le champ. » et un bouton secondaire « Archiver à la place » (champ actif seulement) ; notification « Champ supprimé. »
- [X] T051 [US5] Scénario Playwright in `web/e2e/custom-fields.spec.ts` (« suppression ») : dialogue annonçant les valeurs perdues, « Archiver à la place », puis suppression confirmée ; axe sans violation `serious`/`critical`

**Checkpoint**: toutes les stories livrées.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T052 [P] Données de démonstration in `internal/tools/e2eseed/main.go` : champ de membre « Numéro de licence fédérale » (texte court, rempli par le membre) et champ d'adhésion « A payé sa licence fédérale » (oui/non, réservé), avec quelques valeurs
- [X] T053 [P] Revue de sécurité ciblée (plan.md › points de vigilance 1 à 5) : toutes les requêtes appelées en audience `Owner` passent par `ListOwnerCustomFields` ; aucun `slog` des nouveaux fichiers ne journalise libellé ni valeur (`grep -n "Logger\." internal/api/custom_field*.go internal/api/member*.go internal/api/membership*.go`) ; aucun test de rôle (`CurrentAccount(ctx).Role`) ajouté dans les handlers ; aucun SQL construit par concaténation ; corriger le cas échéant
- [X] T054 [P] Documentation utilisateur in `README.md` (ou documentation d'utilisation existante) : rubrique « Paramètres › Champs personnalisés » (champ de membre ou d'adhésion, types, visibilités, obligatoire, archivage ou suppression, filtres et recherche)
- [X] T055 [P] Ajouter à la section Assumptions de `specs/005-member-management/spec.md` et `specs/006-memberships/spec.md` une note renvoyant à la spec 011 pour les informations complémentaires des fiches et des adhésions
- [X] T056 Exécuter `go tool task check` (build `CGO_ENABLED=0`, `go vet`, `go test ./...`, `sqlc generate` et `frigg openapi` sans diff, migrations sur base vierge, `bun run typecheck`, Biome, Vitest, `bun run build`, Playwright + axe) et dérouler quickstart.md › scénarios manuels 1 à 5

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance. T001 avant T002 et T003.
- **Foundational (Phase 2)** : dépend de la phase 1. T004, T005, T006, T007 en parallèle (T006 utilise les types de T004 et T005 : écrire T004 et T005 d'abord si un seul développeur) ; T008 après T004–T006 ; T009 en parallèle de T008. Bloque toutes les stories.
- **US1 (Phase 3)** : dépend de la phase 2. MVP.
- **US2 (Phase 4)** : dépend de la phase 2 ; côté API, testable sans US1 (définitions insérées par `env.Store` dans les tests) ; écrans et Playwright après US1 (création des champs, `customFieldsQuery`, `myCustomFieldsQuery`).
- **US3 (Phase 5)** : dépend de la phase 2 ; réutilise `CustomFieldInputs`, `CustomFieldValues` et `form.ts` de US2 (T025–T027) pour les écrans.
- **US4 (Phase 6)** : dépend de la phase 2 ; ses tests supposent l'écriture des valeurs (US2 pour les fiches, US3 pour les adhésions).
- **US5 (Phase 7)** : dépend de US1 (écran de la section).
- **Polish (Phase 8)** : après les stories voulues.

### User Story Dependencies

- **US1 (P1)** : aucune autre story.
- **US2 (P1)** : backend indépendant ; frontend après US1.
- **US3 (P1)** : backend indépendant ; frontend après US2 (composants partagés).
- **US4 (P2)** : après US2 et US3 pour des données à filtrer.
- **US5 (P3)** : après US1.

### Within Each User Story

- Tests d'abord (ils doivent échouer), puis domaine, puis requêtes et handlers, puis régénération du contrat, puis frontend, puis Playwright.

### Parallel Opportunities

- Phase 1 : T002 en parallèle de T003 après T001.
- Phase 2 : T004, T005, T006, T007 en parallèle ; T009 en parallèle de T008.
- US1 : T010, T011 en parallèle ; T015 et T016 en parallèle après T014.
- US2 : T020, T021 en parallèle ; T025, T026, T027 en parallèle après T024.
- US3 : T030, T031 en parallèle ; T035 en parallèle des écrans après T034.
- US4 : T039, T040 en parallèle ; T044, T045 en parallèle après T043.
- Backend de US2 (T022–T023) et de US3 (T032–T033) peuvent avancer en parallèle du frontend de US1.

---

## Parallel Example: Phase 2

```text
T004 internal/customfield/definition.go
T005 internal/customfield/value.go
T006 internal/customfield/apply.go
T007 internal/customfield/{definition,value,apply}_test.go
```

## Parallel Example: User Story 2

```text
# Tests :
T020 internal/api/member_custom_fields_test.go
T021 internal/api/custom_fields_leak_test.go

# Frontend après régénération du contrat (T024) :
T025 web/app/customfields/form.ts (+ form.test.ts)
T026 web/app/customfields/CustomFieldInputs.tsx
T027 web/app/customfields/CustomFieldValues.tsx
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 (migration, requêtes) puis phase 2 (domaine, lecture/écriture partagées).
2. US1 : section « Champs personnalisés ».
3. **STOP et valider** : tests d'API, Playwright ; les champs se définissent sans intervention technique.

US1 seule n'a pas d'effet visible pour les membres : la première livraison
utile à l'association est **US1 + US2** (numéro de licence sur les fiches).

### Incremental Delivery

1. US1 → définitions.
2. US2 → champs de membre (numéro de licence).
3. US3 → champs d'adhésion (licence payée).
4. US4 → filtres et recherche (relances de licence).
5. US5 → suppression définitive.
6. Phase 8 → données de démonstration, revue de sécurité, documentation, contrôles complets.

### Notes

- Chaque tâche se termine par ses tests verts et un commit (gitmoji, message
  court, sans co-auteur, après validation de l'utilisateur).
- Après chaque ajout ou modification d'opération : `frigg openapi` et
  `bun run gen:api` avant le frontend ; le contrôle « sans diff » de la
  constitution l'exige.
- Les champs absents d'une écriture ne sont jamais modifiés : c'est ce qui
  protège les valeurs réservées quand un membre enregistre sa fiche (T020,
  T021).
