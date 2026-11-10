---

description: "Liste des tâches : importation des membres"
---

# Tasks: Importation des membres

**Input**: Design documents from `specs/013-member-import/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : matrice d'autorisation (principes II et III), tests
d'API sur base isolée, Playwright + axe sur chaque parcours touché. Les
paquets purs `internal/csvfile` et `internal/memberimport` ont des tests en
table (research.md › R13).

**Organization**: une phase par user story, dans l'ordre de la spec. US1 (P1)
livre l'import de bout en bout pour les champs de la fiche. US2 (P1) ajoute
l'explication et le téléchargement des lignes rejetées. US3 (P2) ajoute les
champs personnalisés. US4 (P3) ajoute le modèle de fichier.

**Conventions** (constitution 1.8.0) : identifiants, champs JSON et valeurs
d'énumération en anglais (`ready`, `error`, `duplicate`, `fieldsVersion`) ;
commentaires, messages, intitulés du modèle et journaux en français.
Commentaires des requêtes sqlc en **ASCII uniquement** (sqlc sqlite casse sur
un commentaire non ASCII). **Aucune migration.** Le fichier n'est jamais
écrit sur disque ni conservé (research.md › R1). Les journaux ne contiennent
jamais de nom de fichier, d'intitulé, de valeur de cellule ni de donnée
personnelle (R10). Aucun test de rôle dans les handlers : le rôle est déclaré
par `register(api, auth.Admin, …)`. Toute écriture passe par
`s.Store.WriteTx` (`BEGIN IMMEDIATE`). Aucune règle de validation n'est
dupliquée côté frontend.

## Format: `[ID] [P?] [Story] Description`

- **[P]** : parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]** : user story de rattachement (US1 à US4)

## Path Conventions

Module Go à la racine (`internal/`), SPA dans `web/` (plan.md › Project
Structure). Code existant à réutiliser :

- **API** :
  - `internal/api/api.go` : `register`, `server`, `s.today()`, `s.now()`,
    `CurrentAccount`, `s.internalError`, `s.txError`, `UploadMaxBody`,
    `humago.MultipartMaxMemory` ;
  - `internal/api/errors.go` : `FieldErrors`, `FieldErrorDetails` ;
  - `internal/api/members.go` : `checkDuplicate`, `logMemberAction`,
    `fieldsOf`, `nowString`, et l'appel de `registerAdminMembers` à la
    ligne 302 ;
  - `internal/api/members_admin.go` : `createMember`, modèle de l'écriture ;
  - `internal/api/custom_field_values.go` : `loadDefinitions`, `boolInt`,
    `UpsertMemberFieldValue` dans `writeFields` ;
  - `internal/api/registrations_admin.go` : `csvCell` et
    `exportEventRegistrations`, modèle de la réponse CSV en
    `huma.StreamResponse` ;
  - `internal/api/communication_attachments.go` : `UploadAttachmentInput`,
    modèle de `huma.MultipartFormFiles` et de `MaxBodyBytes`.
- **Domaine** :
  - `internal/member` : `Input`, `Clean`, `Validate`, `FieldError`,
    `FieldErrors`, `IdentityKey`, les constantes `Max*` ;
  - `internal/customfield` : `Definition`, `Choice`, `Input`, `Value`,
    `NormalizeValue`, `ValueError`, `ParseNumber`, les types
    `TypeShortText`… et `Admin` ;
  - `internal/season` : `Date`, `ParseDate` ;
  - `internal/textfold` : `Fold`.
- **Base** : `internal/database/queries/members.sql` et
  `internal/database/sqlc/`.
- **Tests Go** :
  - `internal/api/apitest` : `New`, `Seed`, `LoggedIn`, `SetToday`, `Logs`,
    `Store`, `Raw`, `Client().Do`, `Response.JSON` ;
  - `internal/api/members_helpers_test.go` : `insertOrphan`, `memberLogs` ;
  - `internal/api/communications_helpers_test.go` : `upload`, modèle d'un
    corps multipart ;
  - `internal/api/access_test.go` : `TestAccessMatrix`, matrice
    automatique.
- **Frontend** :
  - `web/app/routes.ts` ;
  - `web/app/routes/app/admin/members.tsx` ;
  - `web/app/routes/app/admin/event-detail.tsx` : `onExport` avec
    `parseAs: "blob"` ;
  - `web/app/api/communications.ts` : `uploadAttachment` avec
    `bodySerializer` et `FormData` ;
  - `web/app/api/problem.ts`, `web/app/api/client.ts`,
    `web/app/lib/member-filters.ts` (`rattachement=sans-compte`) ;
  - `web/app/ui/*` : `Alert`, `Badge`, `Button`, `Select`, `RadioGroup`,
    `Table`, `Pagination`, `Icon`.
- **E2E** : `web/e2e/test.ts`, `web/e2e/axe.ts`,
  `web/e2e/admin-members.spec.ts`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: outil CSV commun et requête d'identités.

- [X] T001 Créer `internal/csvfile/write.go` (commentaire de paquet en français : lecture et écriture de fichiers CSV pour les tableurs, sans base) : déplacer `csvCell` de `internal/api/registrations_admin.go` en `SafeCell(v string) string` (préfixe `'` si la valeur commence par `=`, `+`, `-`, `@`, tabulation ou retour chariot) ; `NewWriter(w io.Writer) *csv.Writer` qui écrit d'abord la marque d'ordre des octets UTF-8 `\xef\xbb\xbf` et règle `Comma = ';'` ; remplacer dans `internal/api/registrations_admin.go` les appels à `csvCell` par `csvfile.SafeCell` et la création du writer par `csvfile.NewWriter` ; le test d'export des inscrits existant (`internal/api/registrations_export_test.go`) reste vert sans modification
- [X] T002 [P] Ajouter à `internal/database/queries/members.sql` la requête `-- name: ListMemberIdentities :many` avec le commentaire ASCII `-- Identites de toutes les fiches, pour la detection des doublons a l'import.` et `SELECT id, first_name, last_name, birth_date FROM members;` ; lancer `sqlc generate` et commiter `internal/database/sqlc/`

**Checkpoint**: `go build ./...` et `go test ./internal/api/ -run Export` passent.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: lecture du CSV, correspondance, validation et classification des lignes (champs de la fiche), squelette de l'API et du frontend, utilisés par toutes les stories.

**⚠️ CRITICAL**: aucune story ne commence avant la fin de cette phase.

- [X] T003 [P] Créer `internal/csvfile/decode.go` (research.md › R2) :
  - **Types** : `Limits{MaxBytes int64; MaxRows, MaxColumns int}` et `DefaultLimits = Limits{MaxBytes: 5 << 20, MaxRows: 2000, MaxColumns: 100}` ; `Table{Headers []string; Rows []Row; Encoding string; Separator rune}`, avec `Encoding` valant `"utf-8"` ou `"windows-1252"` ; `Row{Line int; Cells []string; Extra bool}`.
  - **Erreurs** : `ErrEmpty`, `ErrBinary`, `ErrTooLarge`, `ErrTooManyRows`, `ErrTooManyColumns`, `ErrNoData`, et `MalformedError{Line int}` (implémente `error`).
  - **`Decode(data []byte, l Limits) (Table, error)`**, dans l'ordre :
    1. taille > `MaxBytes` → `ErrTooLarge` ; contenu vide ou réduit à des espaces → `ErrEmpty` ;
    2. octet NUL → `ErrBinary` ;
    3. BOM UTF-8 retirée ; `utf8.Valid` → `"utf-8"`, sinon décodage par `charmap.Windows1252.NewDecoder()` (`golang.org/x/text/encoding/charmap`) → `"windows-1252"` ;
    4. séparateur : compte de `;`, `,` et `\t` hors guillemets sur la première ligne logique ; le plus fréquent l'emporte, `;` en cas d'égalité ou d'absence ;
    5. lecture par `encoding/csv` (`LazyQuotes = false`, `FieldsPerRecord = -1`) ; `csv.ParseError` → `MalformedError{Line: numéro d'enregistrement}` ;
    6. plus de `MaxColumns` intitulés → `ErrTooManyColumns` ;
    7. lignes dont toutes les cellules sont vides après `strings.TrimSpace` : ignorées ;
    8. `Line` = numéro d'enregistrement, la ligne d'intitulés valant 1 ; lignes courtes complétées de `""` jusqu'au nombre d'intitulés ; cellules excédentaires non vides → `Extra = true`, tronquées au nombre d'intitulés ;
    9. plus de `MaxRows` lignes non vides → `ErrTooManyRows` ; aucune ligne de données → `ErrNoData` ;
    10. chaque cellule : `strings.TrimSpace`, et retrait d'une apostrophe initiale suivie de `=`, `+`, `-`, `@`, tabulation ou retour chariot (inverse exact de `SafeCell`, aucun autre cas) ;
    11. intitulés : `strings.TrimSpace`.
- [X] T004 [P] Tests en table dans `internal/csvfile/csvfile_test.go` :
  - **Encodage** : UTF-8 sans BOM, avec BOM, Windows-1252 avec « é », « è », « ç » et « œ » ;
  - **Séparateur** : `;`, `,`, tabulation ; égalité → `;` ;
  - **Guillemets** : valeur entre guillemets contenant `;` et un saut de ligne (numéro de ligne suivant correct) ; guillemet non refermé → `MalformedError` avec la ligne ;
  - **Refus** : octet NUL (début d'un fichier zip `PK\x03\x04`) → `ErrBinary` ; en-tête seul → `ErrNoData` ; 2 000 lignes acceptées, 2 001 refusées ; 101 colonnes refusées ; 5 Mio + 1 octet refusé ;
  - **Lignes** : lignes vides ignorées sans décaler `Line` ; ligne courte complétée ; ligne longue avec cellules vides acceptée sans `Extra`, avec une cellule non vide → `Extra` ;
  - **Apostrophe** : aller-retour `SafeCell` puis `Decode` sur `+32470…`, `=SOMME(A1)`, `-5` ; « 'Ohana » conservé ; espaces de bord retirés.
- [X] T005 [P] Créer `internal/memberimport/mapping.go` (commentaire de paquet en français : règles de l'importation des membres, sans base ni HTTP ; research.md › R4) :
  - **Types et constantes** : `type Target string` ; `None Target = ""`, `FirstName = "firstName"`, `LastName = "lastName"`, `BirthDate = "birthDate"`, `Email = "email"`, `Phone = "phone"`, `AddressLine1 = "addressLine1"`, `AddressLine2 = "addressLine2"`, `PostalCode = "postalCode"`, `City = "city"` ; `FieldTarget(id int64) Target` (`"field:<id>"`) et `(t Target) FieldID() (int64, bool)` ; `StandardTargets` dans l'ordre de la fiche, avec leurs libellés français « Prénom », « Nom », « Date de naissance », « Adresse électronique », « Téléphone », « Adresse », « Complément d'adresse », « Code postal », « Commune » ; `RequiredTargets = {FirstName, LastName, BirthDate}`.
  - **`normalizeHeader(h string) (full, short string)`** : `textfold.Fold`, retrait de `*`, puis suppression de tout caractère non lettre et non chiffre (`unicode.IsLetter`/`IsDigit`) ; `short` est calculé de même après retrait d'une parenthèse finale `\s*\([^()]*\)\s*$`.
  - **Synonymes** : table exacte de research.md › R4.
  - **`Suggest(headers []string, defs []customfield.Definition) []Target`** : pour chaque colonne, dans l'ordre, synonyme sur `full` puis `short`, puis libellé normalisé d'un champ personnalisé sur `full` puis `short` ; une cible déjà prise par une colonne précédente n'est pas reproposée.
  - **`ColumnLabels(headers []string) []string`** : intitulé, ou « Colonne N » (N à partir de 1) s'il est vide ou en double après `textfold.Fold`.
  - **`ValidateMapping(m []Target, columns int, defs []customfield.Definition) []MappingError`** : `MappingError{Index int; Message string}`, `Index` valant −1 pour une erreur globale. Contrôles et messages :
    - longueur ≠ `columns` → « La correspondance ne couvre pas toutes les colonnes. » ;
    - cible inconnue, ou champ personnalisé absent de `defs` (archivé ou supprimé) → « Ce champ n'existe plus ou n'est pas importable. » ;
    - cible déjà associée à la colonne j → « Ce champ est déjà associé à la colonne « <label j> ». » ;
    - champ exigé absent → « Associez une colonne au champ « Date de naissance ». », une erreur par champ manquant.
- [X] T006 [P] Créer `internal/memberimport/row.go` (research.md › R5), partie fiche :
  - **`ParseDate(s string) (season.Date, error)`** : formats `J/M/AAAA`, `J-M-AAAA`, `J.M.AAAA` (jour et mois sur 1 ou 2 chiffres) et `AAAA-MM-JJ` ; date calendaire vérifiée. Erreurs exportées :
    - `ErrTwoDigitYear` : « Indiquez l'année sur 4 chiffres (JJ/MM/AAAA). » ;
    - `ErrDateFormat` : « Date non reconnue : utilisez le format JJ/MM/AAAA. » ;
    - `ErrDateInvalid` : « Cette date n'existe pas. ».
  - **Types** : `CellError{Column int; Message string}` (`Column` vaut −1 pour la ligne entière) ; `Parsed{Line int; Input member.Clean; Fields map[int64]customfield.Value; Errors []CellError; MissingRequired []int64; Raw RawIdentity}` ; `RawIdentity{FirstName, LastName, BirthDate string}`, valeurs brutes des colonnes associées.
  - **`ParseRow(row csvfile.Row, m []Target, defs []customfield.Definition, today season.Date) Parsed`** :
    1. `row.Extra` → erreur « Cette ligne a plus de colonnes que la ligne d'intitulés. » ;
    2. date de naissance vide → « La date de naissance est obligatoire. », sinon `ParseDate` ;
    3. `member.Validate` sur le `member.Input` construit, chaque `member.FieldError` rattachée à la colonne de son champ ; `ErrAddressIncomplete` rattachée à la colonne du champ en cause si elle est associée, sinon à la colonne `AddressLine1`, `PostalCode` ou `City` associée, sinon −1 ;
    4. messages en français repris des erreurs de `member` : « Le prénom est obligatoire. », « Code postal : 5 chiffres attendus. », « Adresse électronique invalide. », « Téléphone invalide : 10 chiffres ou format international (+32…). », « Adresse incomplète : indiquez l'adresse, le code postal et la commune. », « Complément sans adresse. », « Date de naissance dans le futur. », « Date de naissance trop ancienne (plus de 120 ans). » ; prénom ou nom de plus de 80 caractères → « 80 caractères au plus. » (vérifié avant `member.Validate`) ;
    5. **toutes** les erreurs de la ligne sont collectées ;
    6. les cibles `field:<id>` sont ignorées tant que T037 n'est pas faite.
- [X] T007 [P] Créer `internal/memberimport/classify.go` (research.md › R6) :
  - **Types** : `Status` (`StatusReady = "ready"`, `StatusError = "error"`, `StatusDuplicate = "duplicate"`) ; `Identity{ID int64; FirstName, LastName, BirthDate string}` ; `Result{Parsed; Status Status; DuplicateOfMember *Identity; DuplicateOfLine int}`.
  - **`Classify(rows []Parsed, existing []Identity) []Result`** : la clé est `member.IdentityKey(first, last) + "\x00" + date ISO` ;
    - erreurs → `error`, sans occuper de clé ;
    - clé présente en base → `duplicate` avec la fiche ;
    - clé d'une ligne `ready` plus haut → `duplicate` avec son `Line` ;
    - sinon `ready`.
  - **`Counts(results) (ready, errors, duplicates int)`**.
- [X] T008 [P] Créer `internal/memberimport/fingerprint.go` : `Fingerprint(defs []customfield.Definition) string`, SHA-256 d'une sérialisation canonique (identifiant, type, libellé, puis identifiants et libellés des choix, triés par identifiant), dont on garde les 16 premiers octets en hexadécimal minuscule (32 caractères) (research.md › R8)
- [X] T009 [P] Tests en table dans `internal/memberimport/mapping_test.go`, `row_test.go`, `classify_test.go` et `fingerprint_test.go` :
  - **Correspondance** :
    - intitulés « Prénom », « NOM », « Date de naissance * », « Date de naissance (JJ/MM/AAAA) », « Tél. », « E-mail », « Courriel », « CP », « Ville », « Complément d'adresse » → cibles attendues ;
    - « Remarques », « Raison du rejet » → `None` ;
    - deux colonnes « Nom » → seule la première est proposée ;
    - `ColumnLabels` sur un intitulé vide et sur des intitulés en double ;
    - `ValidateMapping` : longueur, cible inconnue, doublon de cible avec le libellé de la colonne j, trois champs exigés manquants → trois erreurs.
  - **`ParseDate`** : `3/4/2012`, `03-04-2012`, `03.04.2012`, `2012-04-03` → 3 avril 2012 ; `3/4/12` → `ErrTwoDigitYear` ; `31/02/2012` → `ErrDateInvalid` ; `2012/04/03` → `ErrDateFormat`.
  - **`ParseRow`** :
    - ligne valide avec adresse complète ;
    - erreurs multiples (nom vide, adresse électronique invalide, code postal `1000`) → 3 `CellError` sur les bonnes colonnes ;
    - téléphone `06 12 34 56 78` → `0612345678` ;
    - `Extra` → erreur de ligne ;
    - nom de 81 caractères.
  - **`Classify`** : doublon en base ; doublon dans le fichier ; ligne en erreur puis ligne identique valide → `ready` ; casse, accents et espaces ignorés (« Lefèvre  Léa » ≡ « lefevre lea ») ; même identité avec une autre date → `ready`.
  - **Correspondance avec 005** : une ligne `duplicate` en base aurait été refusée par `checkDuplicate` (mêmes clés `member.IdentityKey`).
  - **`Fingerprint`** : stable quel que soit l'ordre des `defs` ; change si un libellé de choix change.
- [X] T010 Créer `internal/api/members_import.go` (research.md › R11) :
  - **Enregistrement** : `func (s *server) registerMemberImport(api huma.API)`, appelée dans `internal/api/members.go` **avant** `s.registerAdminMembers(api)` ; tag `tagMembers`, rôle `auth.Admin` pour chaque opération.
  - **Messages** : `MsgImportNotCSV` « Ce fichier n'est pas un fichier CSV lisible. Dans votre tableur, choisissez « Enregistrer sous » puis le format CSV. » ; `MsgImportMalformed` « Guillemet non refermé à la ligne %d : vérifiez cette ligne dans votre tableur. » ; `MsgImportNoData` « Le fichier ne contient aucune ligne de membre sous la ligne d'intitulés. » ; `MsgImportTooLarge` « Le fichier dépasse 5 Mo. » ; `MsgImportTooManyRows` « Le fichier dépasse 2 000 lignes : découpez-le en plusieurs fichiers. » ; `MsgImportTooManyColumns` « Le fichier dépasse 100 colonnes. » ; `MsgImportEmpty` « Le fichier est vide. » ; `MsgImportNothing` « Aucune ligne à importer. » ; `MsgImportFieldsChanged` « Les champs personnalisés ont changé depuis l'aperçu. L'aperçu a été recalculé. ».
  - **Types huma** (contracts/api.md › Types) : `ImportColumn`, `ImportTargetOption`, `ImportFile`, `ImportCellError`, `ImportRow` (`status` `enum:"ready,error,duplicate"`), `ImportMemberRef{ID int64; Label string}`, `ImportMissing`, `ImportCounts`, `ImportAnalysis`, `ImportPreview`, `ImportReport`.
  - **Entrées multipart** (`huma.MultipartFormFiles`) :
    - `ImportFileForm{File huma.FormFile \`form:"file" required:"true"\`}` ;
    - `ImportMappingForm`, qui ajoute `Mapping string \`form:"mapping" required:"true" maxLength:"4096"\`` ;
    - `ImportRunForm`, qui ajoute `FieldsVersion string \`form:"fieldsVersion" required:"true" pattern:"^[0-9a-f]{32}$"\``.
  - **Helpers** :
    - `s.readImportFile(ctx, f huma.FormFile) (csvfile.Table, error)` : `f.Size > csvfile.DefaultLimits.MaxBytes` → `FieldErrors("form.file", MsgImportTooLarge)` ; lecture bornée par `io.LimitReader` ; `csvfile.Decode` ; chaque erreur traduite en `FieldErrors("form.file", …)` ;
    - `parseMapping(raw string, t csvfile.Table, defs) ([]memberimport.Target, error)` : JSON invalide → `form.mapping` ; `MappingError` → `form.mapping[i]` (`form.mapping` si −1), via `FieldErrorDetails` ;
    - `s.importDefinitions(ctx, q *sqlc.Queries) ([]customfield.Definition, error)` : `loadDefinitions(ctx, q, customfield.ScopeMember, customfield.Admin)`, champs actifs seulement ;
    - `toIdentities([]sqlc.ListMemberIdentitiesRow) []memberimport.Identity` ;
    - `s.classifyImport(ctx, q, t, m, defs) []memberimport.Result`, qui appelle `ParseRow` sur chaque ligne avec `s.today()` puis `Classify` avec `q.ListMemberIdentities`.
- [X] T011 [P] Helpers de test dans `internal/api/members_import_helpers_test.go` :
  - `importForm(t, fields map[string]string, name string, data []byte) (body []byte, contentType string)` ;
  - `analyzeImport(t, c, data)`, `previewImport(t, c, data, mapping []string)`, `runImport(t, c, data, mapping []string, version string)` et `rejectedImport(t, c, data, mapping []string)`, qui renvoient `apitest.Response` (modèle : `upload` dans `internal/api/communications_helpers_test.go`, `apitest.Raw`) ;
  - `csvBytes(lines ...string) []byte`, lignes jointes par `\r\n` ;
  - `win1252(s string) []byte`, via `charmap.Windows1252.NewEncoder()` ;
  - `standardMapping` pour l'en-tête `Prénom;Nom;Date de naissance;Mail;Téléphone`.
- [X] T012 [P] Créer `web/app/members/import/download.ts` :
  - `downloadBlob(blob: Blob, filename: string)`, qui crée l'URL d'objet, clique sur un lien temporaire puis révoque l'URL ;
  - remplacer le code équivalent de `onExport` dans `web/app/routes/app/admin/event-detail.tsx` par `downloadBlob`, sans changement de comportement.

**Checkpoint**: `go test ./internal/csvfile/... ./internal/memberimport/...` passe ; `go build ./...` passe.

---

## Phase 3: User Story 1 - Importer les membres d'un tableur (Priority: P1) 🎯 MVP

**Goal**: un administrateur dépose un CSV, ajuste la correspondance, voit l'aperçu et crée les fiches « Sans compte » d'un bloc.

**Independent Test**: avec un fichier de quelques lignes valides, réaliser l'import de bout en bout et vérifier que chaque ligne a donné une fiche « Sans compte » avec les bonnes valeurs ; un membre est refusé.

### Tests for User Story 1

- [X] T013 [P] [US1] Tests d'API dans `internal/api/members_import_test.go` :
  - **`TestImportAnalyze`** :
    - fichier Windows-1252 séparé par `;` → `encoding: "windows-1252"`, `separator: ";"`, `rowCount` exact, `samples` avec accents corrects (au plus 3, d'au plus 80 caractères), `suggested` attendu ;
    - `targets` dans l'ordre de la fiche, `requiredForImport` sur les trois champs exigés ;
    - `fieldsVersion` de 32 caractères hexadécimaux.
  - **`TestImportFileRejected`** (`422 form.file`, avec le message) : fichier vide, en-tête seul, octet NUL (`PK\x03\x04…`, classeur renommé), guillemet non refermé (ligne citée), 2 001 lignes, 101 colonnes, 5 Mio + 1 octet.
  - **`TestImportPreview`** : compteurs exacts ; une ligne par ligne non vide, dans l'ordre, avec `line` égal au numéro de ligne du tableur ; **aucune fiche créée** (`ListMemberIdentities` inchangé).
  - **`TestImportMappingErrors`** : JSON invalide → `form.mapping` ; cible inconnue → `form.mapping[2]` ; même cible sur deux colonnes → `form.mapping[3]` avec le libellé de la première ; date de naissance non associée → `form.mapping` avec « Associez une colonne au champ « Date de naissance ». ».
  - **`TestImportRun`** :
    - `201` avec `created` égal au nombre de lignes `ready` ;
    - fiches créées avec `account_id` NULL et `is_self` 0, valeurs nettoyées (téléphone normalisé, adresse électronique en minuscules), visibles par `GET /api/admin/members?account=none` ;
    - aucun e-mail en file ;
    - une ligne de journal `member.import` avec `created`, `errors`, `duplicates`, `first_member_id`, `last_member_id`, sans prénom, nom, adresse électronique ni nom de fichier (`memberLogs` avec ces valeurs interdites).
  - **`TestImportNothing`** : toutes les lignes sont des doublons → `422` « Aucune ligne à importer. » et aucune fiche.
  - **`TestImportFieldsVersion`** : `fieldsVersion` différent de l'empreinte courante → `409` avec `MsgImportFieldsChanged` et aucune fiche.
  - **`TestImportDuplicateSincePreview`** : fiche créée par `POST /api/admin/members` entre l'aperçu et l'import → `newDuplicates: 1`, fiche non dupliquée.
  - **`TestImportConcurrent`** : deux `runImport` simultanés du même fichier (goroutines) → chaque identité existe exactement une fois ; la somme des `created` est égale au nombre de lignes.
  - **`TestImportNoFileKept`** (FR-021) : après `analyze`, `preview`, `rejected` et `run`, le répertoire temporaire (`t.Setenv("TMPDIR", t.TempDir())` avant l'appel) et le répertoire de la base de test ne contiennent aucun fichier nouveau.
  - **`TestImportAtomic`** : échec provoqué au milieu de la transaction → aucune fiche créée. Par exemple, un trigger temporaire créé dans le test sur `env.Store` : `CREATE TRIGGER fail_import BEFORE INSERT ON members WHEN new.last_name = 'Boom' BEGIN SELECT RAISE(ABORT, 'boom'); END`.
- [X] T014 [P] [US1] Vérifier dans `internal/api/access_test.go` que la matrice automatique couvre les cinq opérations d'import (anonyme → 401, membre → 403, administrateur : tout statut sauf 401 et 403, par exemple `415` ou `422` sur un corps non multipart). Adapter le corps envoyé aux opérations multipart si la matrice ne les atteint pas avec `map[string]any{}`.

### Implementation for User Story 1

- [X] T015 [US1] Implémenter `analyze-member-import` (`POST /admin/members/import/analyze`, `MaxBodyBytes = UploadMaxBody`) dans `internal/api/members_import.go` :
  - réponse : `ImportAnalysis{file, columns, suggested, targets, fieldsVersion}` ;
  - `columns` : `index`, `header`, `label` (`ColumnLabels`) et `samples` (3 premières valeurs non vides de la colonne, tronquées à 80 runes) ;
  - `targets` : `StandardTargets`, puis les champs personnalisés actifs dans l'ordre défini (`fieldType` renseigné) ;
  - la proposition provient de `memberimport.Suggest`, l'empreinte de `memberimport.Fingerprint` ;
  - lecture sur `s.Store.Read`.
- [X] T016 [US1] Implémenter `preview-member-import` (`POST /admin/members/import/preview`) dans `internal/api/members_import.go` :
  - réponse : `ImportPreview{file, counts, missingRequired: [], incompleteCount: 0, rows, fieldsVersion}` ;
  - `rows[i]` : `line` ; `status` ; `firstName`, `lastName`, `birthDate` bruts ; `errors` avec `column`, `columnLabel` et `message` (`column` et `columnLabel` nuls pour −1) ;
  - doublon en base : `duplicateOfMember` avec le libellé « <Nom> <Prénom> (<JJ/MM/AAAA>) » ; doublon dans le fichier : `duplicateOfLine` ;
  - lecture sur `s.Store.Read`.
- [X] T017 [US1] Implémenter `run-member-import` (`POST /admin/members/import`, `DefaultStatus: 201`) dans `internal/api/members_import.go` (research.md › R9) :
  1. lecture et décodage du fichier, puis `parseMapping` **hors transaction**, sur la base des définitions lues en lecture ;
  2. `s.Store.WriteTx` : `importDefinitions(q)` puis `Fingerprint` ≠ `fieldsVersion` → `huma.Error409Conflict(MsgImportFieldsChanged)` ;
  3. `classifyImport` dans la transaction avec `q.ListMemberIdentities` ;
  4. aucune ligne `ready` → `huma.Error422UnprocessableEntity(MsgImportNothing)` ;
  5. pour chaque ligne `ready`, `q.CreateMember` (`AccountID` invalide, valeurs de `fieldsOf(clean)`, `Now: nowString()`) en mémorisant le premier et le dernier identifiant ;
  6. `newDuplicates` = lignes `ready` selon un classement préalable sur le pool de lecture, devenues `duplicate` dans la transaction ;
  7. après validation, journal `s.Logger.InfoContext(ctx, "importation de membres", "action", "member.import", "actor_id", CurrentAccount(ctx).ID, "created", n, "errors", e, "duplicates", d, "new_duplicates", nd, "missing_required", 0, "first_member_id", first, "last_member_id", last)` (R10) ;
  8. réponse `ImportReport`.
- [X] T018 [P] [US1] Régénérer le contrat : `go run ./cmd/frigg openapi` → `web/app/api/openapi.json`, puis `bun run gen:api` dans `web/` → `web/app/api/schema.d.ts` (sans retouche manuelle).
- [X] T019 [US1] Créer `web/app/api/memberImport.ts` :
  - `analyzeImport(file: File)`, `previewImport(file: File, mapping: string[])` et `runImport(file: File, mapping: string[], fieldsVersion: string)` via `client.POST` avec `bodySerializer` construisant un `FormData` (champ `file` avec `file.name`, `mapping` en `JSON.stringify`, `fieldsVersion`), sur le modèle de `uploadAttachment` dans `web/app/api/communications.ts` ;
  - en cas d'erreur, levée d'un `Problem` avec `status` ;
  - types exportés depuis `components["schemas"]` (`ImportAnalysis`, `ImportPreview`, `ImportReport`, `ImportRow`, `ImportTargetOption`).
- [X] T020 [P] [US1] Créer `web/app/members/import/ImportSteps.tsx` (contracts/ui-routes.md › Assistant) :
  - liste `ol` avec `aria-label="Étapes de l'import"` : « Fichier », « Colonnes », « Aperçu », « Rapport » ;
  - étape courante avec `aria-current="step"` et style `--brand`, étapes passées avec une coche (`Icon` `Check`) ;
  - lisible à 320 px.
- [X] T021 [P] [US1] Créer `web/app/members/import/FileStep.tsx` (ui-routes.md › 1. Fichier) :
  - **Introduction** : texte d'aide ; `Collapsible` base-ui « Comment enregistrer mon tableur en CSV ? » (Excel : Fichier › Enregistrer sous › CSV UTF-8 ; LibreOffice : Fichier › Enregistrer sous › Texte CSV) ; rappel des champs exigés, des formats de date, des fiches créées « Sans compte » sans e-mail et des personnes déjà présentes ignorées.
  - **Dépôt** : `input type="file" accept=".csv,text/csv"` étiqueté « Fichier CSV », habillé en zone de glisser-déposer (`dragover` et `drop` en complément) ; à la sélection, `onFile(file)`.
  - **États** : `busy` affiché ; erreur `422 form.file` dans une `Alert` `--danger` reliée par `aria-describedby`, focus sur l'input.
  - **Modèle** : emplacement du lien « Télécharger le modèle » réservé par une prop facultative `templateAction` (livrée en US4).
- [X] T022 [P] [US1] Créer `web/app/members/import/MappingStep.tsx` (ui-routes.md › 2. Colonnes) :
  - **Rappel du fichier** : nom, nombre de lignes, encodage et séparateur en texte secondaire.
  - **Tableau** (`Table`, cartes sous 640 px), une ligne par colonne : `label`, `samples` en `--muted`, et `Select` base-ui étiqueté « Champ Frigg pour la colonne <label> ».
  - **Options du `Select`** : « Ne pas importer », groupe « Fiche » (libellés suivis de « * » si `requiredForImport`), groupe « Informations complémentaires » (champs personnalisés, s'il y en a) ; une cible prise par une autre colonne est désactivée et suivie de « (colonne « <label> ») ».
  - **Résumé** : « N colonnes sur M associées ».
  - **Champ exigé non associé** : `Alert` `--warn` qui le nomme ; bouton « Voir l'aperçu » désactivé, la raison reliée par `aria-describedby`.
  - **Erreurs serveur** : `form.mapping[i]` affichée sous le `Select` i, `form.mapping` dans l'alerte.
  - **Actions** : « Changer de fichier » et « Voir l'aperçu ».
  - Ce contrôle sert seulement l'affichage (principe VI) : il ne s'appuie que sur `requiredForImport` et `targets` renvoyés par `analyze` et sur l'état de la correspondance, sans aucune liste de champs ni règle codée en dur. Le serveur reste l'autorité (`422`).
- [X] T023 [P] [US1] Créer `web/app/members/import/PreviewStep.tsx` (ui-routes.md › 3. Aperçu), version de base :
  - **Compteurs** : trois `Badge`-statistiques « N à importer » (`--ok`), « N en erreur » (`--danger`), « N doublons » (`--warn`) dans une région `aria-live="polite"`.
  - **Tableau** paginé côté client par 50 (`Pagination`) : Ligne, Nom, Prénom, Naissance, État (`Badge`), Détail (liste des `errors` « <columnLabel> : <message> »).
  - **Actions** : « Modifier les colonnes » ; bouton principal « Importer N membres » (`busy` pendant l'import), absent si `counts.ready = 0` et remplacé par une `Alert` `--warn` « Aucune ligne à importer : corrigez le fichier ou les colonnes. ».
  - **Prévu pour US2** : un emplacement pour le filtre et pour l'action « Télécharger les lignes rejetées ».
- [X] T024 [P] [US1] Créer `web/app/members/import/ReportStep.tsx` (ui-routes.md › 4. Rapport) :
  - `Alert` `--ok` « N membres importés. » ;
  - liste des lignes en erreur, des doublons (dont « N personnes ajoutées entre-temps » si `newDuplicates > 0`) et des fiches à compléter ;
  - actions : lien « Voir les membres sans compte » vers `/admin/membres?rattachement=sans-compte`, et « Importer un autre fichier ».
- [X] T025 [US1] Créer `web/app/routes/app/admin/member-import.tsx` et l'enregistrer dans `web/app/routes.ts` : `route("importer", "routes/app/admin/member-import.tsx")` **avant** `route(":id", …)` sous `membres`.
  - **Page** : titre « Importer des membres », fil d'Ariane Administration › Membres › Importer.
  - **État local** : `step`, `file`, `analysis`, `mapping`, `preview`, `report`. Les appels passent par `useMutation`, sans cache de requête.
  - **Changement d'étape** : focus sur le `h2` de l'étape (`ref` + `tabIndex={-1}`).
  - **`409` à l'import** : message `--warn` annoncé, nouvel appel de `previewImport` avec le même fichier et la même correspondance, puis maintien à l'étape 3.
  - **Après succès** : invalidation des requêtes de la liste des membres (même clé que `memberListQuery` dans `web/app/routes/app/admin/members.tsx`) et des requêtes de champs personnalisés (`customFieldsQuery`), puis message de succès annoncé.
- [X] T026 [US1] Ajouter l'action « Importer » (`Button` secondaire, `Icon` `Upload`, lien vers `/admin/membres/importer`) à l'en-tête de page de `web/app/routes/app/admin/members.tsx`
- [X] T027 [US1] Test de bout en bout `web/e2e/member-import.spec.ts` avec la fixture `web/e2e/fixtures/member-import.csv` (contenu de `exemple.csv` du quickstart, fichier commité encodé en Windows-1252) :
  - **Parcours** : depuis « Membres », « Importer » ; dépôt via `setInputFiles` ; correspondance proposée ; associer une colonne ; « Voir l'aperçu » ; « Importer 2 membres » ; rapport ; « Voir les membres sans compte » liste les fiches.
  - **Accessibilité** : `checkA11y` (`web/e2e/axe.ts`) à chaque étape, sans violation `serious` ni `critical`.
  - **Clavier** : parcours complet au clavier sur l'étape 2.

**Checkpoint**: US1 livrable seule : import de bout en bout pour les champs de la fiche ; `go test ./...` et l'e2e passent.

---

## Phase 4: User Story 2 - Comprendre et corriger les lignes rejetées (Priority: P1)

**Goal**: l'aperçu explique chaque ligne rejetée, se filtre par état, et les lignes rejetées se téléchargent puis se réimportent après correction.

**Independent Test**: importer un fichier mêlant lignes valides, invalides de chaque sorte et doublons ; vérifier l'aperçu, le rapport, le fichier des lignes rejetées, puis réimporter ce fichier corrigé.

### Tests for User Story 2

- [X] T028 [P] [US2] Tests d'API dans `internal/api/members_import_test.go` :
  - **`TestImportPreviewReasons`** : une ligne aux trois erreurs → trois `errors` avec `columnLabel` ; doublon en base → `duplicateOfMember.id` et libellé ; doublon dans le fichier → `duplicateOfLine` ; ligne longue → erreur de ligne (`column` nul).
  - **`TestImportRejectedCSV`** :
    - en-têtes `Content-Type: text/csv; charset=utf-8` et `Content-Disposition` avec `lignes-rejetees.csv` ;
    - corps commençant par la BOM, séparateur `;` ;
    - intitulés d'origine suivis de « Raison du rejet » ; une ligne par ligne `error` ou `duplicate`, dans l'ordre ;
    - raisons « <intitulé> : <message> » jointes par « ; », « Doublon de la fiche <libellé> », « Doublon de la ligne N » ;
    - cellule `+32…` neutralisée en `'+32…`.
  - **`TestImportRejectedRoundTrip`** (SC-005) :
    1. on corrige dans le fichier rejeté les cellules en erreur ;
    2. `analyze` propose « Raison du rejet » en `None` ;
    3. `preview` : les lignes corrigées sont `ready`, aucune nouvelle erreur, et les doublons restent doublons ;
    4. `run` crée exactement les lignes corrigées, avec le téléphone `+32470123456` intact.

### Implementation for User Story 2

- [X] T029 [US2] Créer `internal/memberimport/template.go` (partie rejets) :
  - `RejectedHeaders(t csvfile.Table) []string` : intitulés d'origine suivis de « Raison du rejet » ;
  - `RejectedRecord(r Result, t csvfile.Table, labels []string) []string` : cellules d'origine passées par `csvfile.SafeCell`, puis la raison. Raisons : messages « <label> : <message> » joints par « ; » (sans préfixe pour une erreur de ligne), « Doublon de la fiche <Nom> <Prénom> (JJ/MM/AAAA) » ou « Doublon de la ligne N » ;
  - tests en table dans `internal/memberimport/template_test.go`.
- [X] T030 [US2] Implémenter `export-member-import-rejected` (`POST /admin/members/import/rejected`, réponse `text/csv` déclarée comme pour `export-event-registrations`) dans `internal/api/members_import.go` :
  - `huma.StreamResponse` qui écrit `csvfile.NewWriter`, puis `RejectedHeaders` et un `RejectedRecord` par ligne `error` ou `duplicate` ;
  - en-têtes `Content-Type: text/csv; charset=utf-8` et `Content-Disposition: attachment; filename="lignes-rejetees.csv"` ;
  - mêmes `422` que `preview` ;
  - régénérer `web/app/api/openapi.json` et `web/app/api/schema.d.ts`.
- [X] T031 [US2] Ajouter `rejectedImport(file: File, mapping: string[]): Promise<Blob>` dans `web/app/api/memberImport.ts` (`parseAs: "blob"` et `bodySerializer`)
- [X] T032 [US2] Compléter `web/app/members/import/PreviewStep.tsx` :
  - **Filtre** : `RadioGroup` base-ui « Afficher » avec les options Toutes, À importer, En erreur et Doublons, chacune suivie de son nombre ; par défaut « En erreur » s'il y en a, sinon « Toutes » ; la pagination revient à 1 à chaque changement.
  - **Doublons** : détail « Déjà dans Frigg : » avec un lien `Link` vers `/admin/membres/{id}` (libellé du membre), ou « Même personne qu'à la ligne N ».
  - **Lignes rejetées** : action « Télécharger les lignes rejetées » (`Icon` `Download`, `busy`) si `errors + duplicates > 0`, par `downloadBlob(blob, "lignes-rejetees.csv")`.
- [X] T033 [US2] Ajouter « Télécharger les lignes rejetées » à `web/app/members/import/ReportStep.tsx` (si erreurs ou doublons). La route `web/app/routes/app/admin/member-import.tsx` conserve `file` et `mapping` après l'import pour ce téléchargement.
- [X] T034 [US2] Étendre `web/e2e/member-import.spec.ts` :
  - le filtre « En erreur » affiche 3 lignes avec leurs raisons ;
  - lien du doublon vers la fiche existante ;
  - téléchargement des lignes rejetées (`page.waitForEvent("download")`), avec les intitulés d'origine et « Raison du rejet » ;
  - `checkA11y` sur l'aperçu filtré.

**Checkpoint**: US1 + US2 : un fichier imparfait s'importe en plusieurs passes sans ressaisie.

---

## Phase 5: User Story 3 - Importer les informations complémentaires (Priority: P2)

**Goal**: les colonnes peuvent alimenter les champs personnalisés de membre actifs, avec leurs règles ; les valeurs obligatoires manquantes sont comptées sans bloquer.

**Independent Test**: définir un champ de chaque type, importer un fichier contenant une colonne par champ, et vérifier les valeurs enregistrées et les erreurs signalées pour des valeurs hors limites.

### Tests for User Story 3

- [X] T035 [P] [US3] Tests en table dans `internal/memberimport/row_test.go` et `mapping_test.go` :
  - **Proposition** : `Suggest` associe une colonne au libellé d'un champ personnalisé (« Numéro de licence », « Taille (cm) » avec parenthèse conservée, puis « Catégorie (Poussin, Benjamin) » par la forme courte) ; une colonne synonyme d'un champ de la fiche l'emporte sur un champ personnalisé de même libellé.
  - **`ParseRow`, valeurs acceptées** :
    - texte court de 200 runes accepté ;
    - nombre « 12,5 » → 12.5 ;
    - date `03/04/2012` → `2012-04-03` ;
    - oui/non : « Oui », « x », « VRAI », « 1 » → coché ; « non », « faux », « 0 » → vide ;
    - choix « poussin » → identifiant de « Poussin », accents et casse ignorés.
  - **`ParseRow`, valeurs refusées** :
    - texte court de 201 runes → erreur, sans troncature ;
    - nombre « douze » → erreur ;
    - oui/non « peut-être » → erreur ;
    - choix « Cadet » → erreur listant les choix (10 au plus puis « … »).
  - **Champs obligatoires** : un champ obligatoire vide alimente `MissingRequired` sans erreur.
- [X] T036 [P] [US3] Tests d'API dans `internal/api/members_import_test.go` :
  - **`TestImportCustomFields`** :
    - `analyze` propose les champs personnalisés actifs quelle que soit leur visibilité, et non les archivés ;
    - `preview` renvoie `missingRequired` (`fieldId`, `label`, `count`) et `incompleteCount` ;
    - chaque `ImportRow` `ready` porte dans `missingRequired` les identifiants des champs obligatoires sans valeur, et ce tableau est vide pour les lignes `error` et `duplicate` ;
    - `run` enregistre les valeurs, lues ensuite par `GET /api/admin/members/{id}` (`customFields`) ;
    - le rapport et le journal reprennent `missing_required`.
  - **`TestImportArchivedTarget`** : un champ archivé entre `analyze` et `preview` donne `form.mapping[i]` « Ce champ n'existe plus ou n'est pas importable. » ; un champ modifié entre `preview` et `run` donne `409`.

### Implementation for User Story 3

- [X] T037 [US3] Compléter `ParseRow` dans `internal/memberimport/row.go` pour les cibles `field:<id>` (research.md › R5) :
  - **Conversion en `customfield.Input`** :
    - texte court, texte long et nombre : `Text` ;
    - date : `ParseDate`, puis `Date` au format ISO ;
    - oui/non : valeur repliée dans `oui|vrai|1|x` → `Checked=true`, dans `non|faux|0` → `Checked=false`, sinon « Valeur non reconnue : indiquez oui ou non. » ;
    - liste de choix : choix dont `textfold.Fold(label)` égale la valeur repliée, sinon « Choix non reconnu. Choix possibles : A, B, C… ».
  - **Validation** : chaque valeur passe par `customfield.NormalizeValue`, et un `customfield.ValueError` devient un `CellError` sur la colonne. Une valeur non vide est rangée dans `Parsed.Fields`. Un champ `Required` sans valeur, associé ou non, alimente `MissingRequired`. On n'appelle **pas** `customfield.Apply`.
- [X] T038 [US3] Mettre à jour `internal/api/members_import.go` :
  - `preview` : chaque `ImportRow` `ready` reçoit `missingRequired` (identifiants des champs de `Parsed.MissingRequired`, tableau vide sinon) ; `missingRequired` est agrégé par champ (libellé) sur les lignes `ready`, et `incompleteCount` compte les lignes `ready` dont `MissingRequired` n'est pas vide ;
  - `run` : pour chaque fiche créée, `q.UpsertMemberFieldValue` pour chaque valeur de `Parsed.Fields`, avec les colonnes typées et `boolInt` comme dans `writeFields` (`internal/api/custom_field_values.go`) et `Now: nowString()` ;
  - rapport et journal : `missingRequired`, `incompleteCount` et `missing_required` ;
  - régénérer le contrat si un type change.
- [X] T039 [US3] Compléter les écrans :
  - `web/app/members/import/MappingStep.tsx` : groupe « Informations complémentaires » du `Select`, avec le type entre parenthèses (« oui/non », « date », « liste ») en `--muted` ;
  - `web/app/members/import/PreviewStep.tsx` : `Alert` `--info` « N fiches importées n'auront pas de valeur pour un champ obligatoire : <libellé> (n), … Vous pourrez les compléter ensuite. » si `incompleteCount > 0` ;
  - `web/app/members/import/ReportStep.tsx` : ligne « N fiches à compléter ».
- [X] T040 [US3] Étendre `web/e2e/member-import.spec.ts` :
  - préparation : création par l'API des champs « Numéro de licence » (texte court, obligatoire) et « Catégorie » (liste Poussin, Benjamin, Minime) ;
  - associer « Licence » au champ ;
  - vérifier l'alerte « fiches à compléter » puis la valeur « Poussin » sur la fiche importée.

**Checkpoint**: US1 à US3 : les informations propres à l'association sont reprises.

---

## Phase 6: User Story 4 - Partir d'un modèle de fichier (Priority: P3)

**Goal**: télécharger un modèle CSV dont chaque colonne est reconnue d'office.

**Independent Test**: télécharger le modèle, le remplir de quelques lignes et constater que toutes les colonnes sont reconnues à l'étape de correspondance.

### Tests for User Story 4

- [X] T041 [P] [US4] Tests :
  - **Tests en table** dans `internal/memberimport/template_test.go` : `TemplateHeaders` donne « Prénom * », « Nom * », « Date de naissance * (JJ/MM/AAAA) », « Adresse électronique », « Téléphone », « Adresse », « Complément d'adresse », « Code postal », « Commune », puis les champs personnalisés dans l'ordre défini (« Licence », « Catégorie (Poussin, Benjamin, Minime) », « Bénévole (oui/non) », « Arrivée (JJ/MM/AAAA) », « Taille (nombre) », choix limités à 10 puis « … ») ; `Suggest(TemplateHeaders(defs), defs)` associe **chaque** colonne à sa cible.
  - **Test d'API `TestImportTemplate`** dans `internal/api/members_import_test.go` : `GET /api/admin/members/import/template` → `200`, `text/csv; charset=utf-8`, `Content-Disposition` avec `modele-import-membres.csv`, BOM, une seule ligne, aucune donnée de membre ; `analyze` de ce modèle suivi de deux lignes → toutes les colonnes ont une cible.

### Implementation for User Story 4

- [X] T042 [US4] Implémenter `TemplateHeaders(defs []customfield.Definition) []string` dans `internal/memberimport/template.go` (research.md › R7), puis l'opération `get-member-import-template` (`GET /admin/members/import/template`, `huma.StreamResponse`, `csvfile.NewWriter`, en-têtes de T030 avec `modele-import-membres.csv`) dans `internal/api/members_import.go`. Régénérer ensuite `web/app/api/openapi.json` et `web/app/api/schema.d.ts`.
- [X] T043 [US4] Ajouter `downloadTemplate(): Promise<Blob>` dans `web/app/api/memberImport.ts`, puis passer à `FileStep` (dans `web/app/routes/app/admin/member-import.tsx`) une `templateAction` « Télécharger le modèle » (`Icon` `Download`, `busy`) qui appelle `downloadBlob(blob, "modele-import-membres.csv")`.
- [X] T044 [US4] Étendre `web/e2e/member-import.spec.ts` : télécharger le modèle depuis l'étape 1 ; ses intitulés contiennent « Prénom * » et « Date de naissance * (JJ/MM/AAAA) ».

**Checkpoint**: toutes les stories livrées.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T045 [P] Test Vitest `web/app/members/import/MappingStep.test.tsx` :
  - une cible prise est désactivée dans les autres `Select` ;
  - l'alerte de champ exigé et la désactivation de « Voir l'aperçu » ;
  - une erreur `form.mapping[1]` s'affiche sous le bon `Select`.
- [X] T046 [P] Vérifier la conformité RGAA sur `web/app/routes/app/admin/member-import.tsx` :
  - focus sur le titre à chaque étape ;
  - annonces `aria-live` des compteurs et du rapport ;
  - libellés des `Select` ;
  - `prefers-reduced-motion` respecté par la zone de dépôt ;
  - lisibilité à 320 px (test Playwright avec un viewport de 320 px dans `web/e2e/member-import.spec.ts`).
- [X] T047 Performance (plan.md › Performance Goals) : test `TestImportLarge` dans `internal/api/members_import_test.go`, ignoré par `testing.Short()`, sur 2 000 lignes valides. L'aperçu et l'import doivent chacun prendre moins de 2 s ; 2 000 fiches créées.
- [ ] T048 Dérouler [quickstart.md](./quickstart.md) :
  - scénarios manuels 1 à 4 avec `go tool task dev` ;
  - corriger les écarts dans le code ou dans les documents de la fonctionnalité.
- [X] T049 Contrôles de la constitution, avec `go tool task check` :
  - `CGO_ENABLED=0 go build` ;
  - `go vet` et `go test ./...` ;
  - `sqlc generate`, `frigg openapi` et `bun run gen:api` sans diff ;
  - dans `web/` : Biome, `bun run typecheck`, Vitest et `bun run build` ;
  - Playwright avec axe sur `member-import.spec.ts` et `admin-members.spec.ts`.

---

## Notes d'implémentation

Écarts par rapport aux tâches, retenus pendant l'implémentation :

- **T006** : les erreurs de fiche gardent l'erreur de `member` (`CellError.Err`
  et `Field`) ; l'API les traduit avec `memberMessages`, les messages du
  formulaire de fiche, au lieu de nouveaux messages dupliqués. Seules les
  erreurs propres à l'import (dates, colonnes en trop, longueurs, oui/non,
  choix) ont leur texte dans `memberimport`.
- **T005** : `ValidateMapping` reçoit les libellés de colonne (pour le message
  « déjà associé à la colonne « … » ») au lieu du seul nombre de colonnes.
- **T017** : `newDuplicates` repose sur le champ facultatif `expectedReady`
  envoyé par le navigateur (research.md › R6, contracts/api.md) ; un
  classement préalable sur le pool de lecture ne voyait pas les fiches créées
  entre l'aperçu et la confirmation.
- **T008** : l'empreinte inclut aussi le caractère obligatoire des champs,
  dont dépendent les fiches à compléter.
- **T022** : `ui/Select` gagne les groupes, les options désactivées avec une
  mention, un message d'erreur relié et un libellé réservé aux lecteurs
  d'écran.
- **T027** : l'e2e tourne sur un huitième serveur dédié (`import`, port 8098),
  les champs personnalisés étant globaux ; un test couvre le champ archivé
  pendant l'aperçu (retour à l'étape « Colonnes »).
- **T037** : fait avec T006 (`parseField`), les tests de T035 et T036 passant
  dès la phase 2.
- **T048** : scénarios du quickstart couverts par l'e2e et les tests d'API ;
  le déroulé manuel dans un navigateur reste à faire par l'utilisateur.

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance.
- **Foundational (Phase 2)** : dépend de la phase 1 (T001 pour `csvfile`, T002 pour `ListMemberIdentities`) ; bloque toutes les stories.
- **US1 (Phase 3)** : dépend de la phase 2.
- **US2 (Phase 4)** : dépend de US1, car elle étend l'aperçu, le rapport et la route.
- **US3 (Phase 5)** : dépend de US1. Elle est indépendante de US2, hormis les fichiers partagés (`row.go`, `members_import.go`, `PreviewStep.tsx`).
- **US4 (Phase 6)** : dépend de US1, et de T029 pour `template.go` si US2 est faite avant ; sinon T042 crée le fichier.
- **Polish (Phase 7)** : après les stories retenues.

### User Story Dependencies

```text
Setup ─► Foundational ─► US1 (MVP) ─┬─► US2
                                    ├─► US3
                                    └─► US4
```

### Within Each User Story

- Les tests sont écrits avant l'implémentation et doivent échouer au départ.
- Côté backend : paquet pur, puis opération, puis contrat régénéré, puis frontend, puis e2e.
- `internal/api/members_import.go` est partagé par toutes les opérations : ses tâches sont séquentielles.

### Parallel Opportunities

- **Phase 1** : T002 en parallèle de T001.
- **Phase 2** : T003 à T009, T011 et T012 sont des fichiers distincts. T010 attend T003, T005 à T008.
- **US1** : T013 et T014 (tests) en parallèle ; T020 à T024 (composants) en parallèle une fois T019 fait.
- **US3** : T035 et T036 en parallèle.
- **Après US1** : US3 et US4 en parallèle par deux personnes, en coordonnant les modifications de `members_import.go`.

---

## Parallel Example: Phase 2

```text
T003 internal/csvfile/decode.go
T004 internal/csvfile/csvfile_test.go
T005 internal/memberimport/mapping.go
T006 internal/memberimport/row.go
T007 internal/memberimport/classify.go
T008 internal/memberimport/fingerprint.go
T011 internal/api/members_import_helpers_test.go
T012 web/app/members/import/download.ts
```

## Parallel Example: User Story 1

```text
T020 web/app/members/import/ImportSteps.tsx
T021 web/app/members/import/FileStep.tsx
T022 web/app/members/import/MappingStep.tsx
T023 web/app/members/import/PreviewStep.tsx
T024 web/app/members/import/ReportStep.tsx
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phases 1 et 2 : outil CSV, règles pures testées.
2. Phase 3 : analyse, aperçu, import, assistant.
3. **Stop and validate** : scénario manuel 1 du quickstart sur un fichier sans champs personnalisés.

### Incremental Delivery

1. US1 : le bureau importe son registre (identité et coordonnées).
2. US2 : les fichiers imparfaits deviennent exploitables en plusieurs passes.
3. US3 : reprise des numéros de licence, catégories, etc.
4. US4 : modèle pour les associations parties du papier.

### Notes

- Ne jamais écrire le fichier déposé sur disque, ni le journaliser.
- Ne jamais dupliquer une règle de `member.Validate` ou de
  `customfield.NormalizeValue` : les appeler.
- La règle de doublon passe uniquement par `member.IdentityKey`, et la date
  par sa forme ISO.
- Proposer un commit après chaque groupe logique, une fois validé par l'utilisateur (gitmoji, message court).
