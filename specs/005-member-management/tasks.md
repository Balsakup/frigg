---

description: "Liste des tâches : gestion des membres"
---

# Tasks: Gestion des membres

**Input**: Design documents from `specs/005-member-management/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : tests d'autorisation (principes II et III), tests
d'API, Playwright + axe sur chaque parcours touché (Workflow). Le cloisonnement
par compte (US2) a sa propre matrice de tests. Les règles du domaine
(nettoyage, date, téléphone, adresse, âge, tri) sont couvertes par des tests en
table.

**Organization**: une phase par user story, dans l'ordre de priorité de la spec
(US1, US2, US3 en P1 ; US4, US5 en P2).

**Conventions** (constitution 1.4.0) : identifiants, tables, champs JSON et
valeurs en anglais ; commentaires, messages, textes d'interface en français.
Commentaires des requêtes sqlc en ASCII uniquement (sqlc échoue sur un
commentaire non ASCII). Nouvelle migration `00007_members.sql` (append-only).
Aucune donnée de fiche (nom, date, coordonnées) dans les journaux.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US5)

## Path Conventions

Un module Go à la racine (`internal/`) et le SPA dans `web/` (voir plan.md ›
Project Structure). Code existant à réutiliser : `internal/api/api.go`
(`register`, `s.today()`), `internal/api/errors.go` (`FieldErrors(location,
messages...)`, `s.internalError`), `internal/api/accounts.go` (`adminWrite`,
`CurrentAccount(ctx)`, `MsgAccountGone`, `MsgSearchTooShort`, pagination de
`listAccounts`), `internal/api/seasons.go` (`optionalString`),
`internal/api/membership_fees.go` (motif de cumul des `FieldError` du domaine
en un seul `422`, journalisation), `internal/database/fts.go`
(`FTSTrigramQuery`), `internal/textfold` (`Fold`), `internal/season`
(`Date`, `Today`), helpers de test `internal/api/apitest` (`New`, `Seed`,
`LoggedIn`, `SetToday`, `Client().Do`, `Response.JSON`, `LogBuffer`) ;
frontend : `web/app/api/accounts.ts` et `web/app/routes/app/admin/accounts.tsx`
(liste filtrée et paginée par l'URL), `web/app/routes/app/admin/account-detail.tsx`
(dialogue de suppression), `web/app/api/problem.ts` (`applyProblemToForm`,
`isProblem`), `web/app/ui/` (`Field`, `Select`, `Table`, `Pagination`,
`Badge`, `Alert`, `ConfirmDialog`, `ScrollRegion`, `Button`, `Icon`) ; e2e :
`web/e2e/test.ts`, `web/e2e/axe.ts`. Régénération : `go tool task gen`
(sqlc, OpenAPI, types TypeScript).

**Date de référence des exemples** : « aujourd'hui » = 2026-11-01
(Europe/Paris) dans les tests Go, via `env.SetToday`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: table, recherche plein texte et requêtes SQL

- [X] T001 Créer `internal/database/migrations/00007_members.sql` (commentaires français) : `CREATE TABLE members (id INTEGER PRIMARY KEY, account_id INTEGER REFERENCES accounts(id) ON DELETE SET NULL, is_self INTEGER NOT NULL DEFAULT 0 CHECK (is_self IN (0, 1)), first_name TEXT NOT NULL CHECK (length(first_name) BETWEEN 1 AND 80), last_name TEXT NOT NULL CHECK (length(last_name) BETWEEN 1 AND 80), birth_date TEXT NOT NULL CHECK (birth_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'), email TEXT CHECK (email IS NULL OR length(email) BETWEEN 3 AND 254), phone TEXT CHECK (phone IS NULL OR length(phone) BETWEEN 9 AND 16), address_line1 TEXT CHECK (address_line1 IS NULL OR length(address_line1) BETWEEN 1 AND 100), address_line2 TEXT CHECK (address_line2 IS NULL OR length(address_line2) BETWEEN 1 AND 100), postal_code TEXT CHECK (postal_code IS NULL OR postal_code GLOB '[0-9][0-9][0-9][0-9][0-9]'), city TEXT CHECK (city IS NULL OR length(city) BETWEEN 1 AND 100), created_at TEXT NOT NULL, updated_at TEXT NOT NULL, CHECK (is_self = 0 OR account_id IS NOT NULL), CHECK ((address_line1 IS NULL) = (postal_code IS NULL) AND (postal_code IS NULL) = (city IS NULL) AND (address_line2 IS NULL OR address_line1 IS NOT NULL))) STRICT;` ; index `members_account_id` (`account_id`), `members_birth_date` (`birth_date`), `CREATE UNIQUE INDEX members_one_self ON members (account_id) WHERE is_self = 1;` ; table `members_fts USING fts5(first_name, last_name, email, phone, content = 'members', content_rowid = 'id', tokenize = "trigram remove_diacritics 1")` et triggers `members_fts_ai`, `members_fts_ad`, `members_fts_au` (`AFTER UPDATE OF first_name, last_name, email, phone`) entre `-- +goose StatementBegin/End`, sur le modèle exact de `accounts_fts` dans `00001_init.sql` ; commentaires : « Fiches membre : personnes de l'association (research.md 005 › R1). », « NULL = fiche sans compte, gérée par le seul bureau. », « Une fiche "Vous" au plus par compte (R10). », « Recherche par fragment, sans casse ni accents (R6). » ; `-- +goose Down` supprimant triggers, `members_fts` puis `members`
- [X] T002 Créer `internal/database/queries/members.sql` (commentaires ASCII uniquement) :
  - espace membre, toutes conditionnées par `account_id = sqlc.arg(account_id)` : `ListMembersOfAccount :many` (`SELECT * … WHERE account_id = ?`), `GetMemberOfAccount :one` (`WHERE id = ? AND account_id = ?`), `CreateMemberForAccount :one` (insertion de toutes les colonnes avec `account_id` et `is_self`, `RETURNING *`), `UpdateMemberOfAccount :execrows` (toutes les colonnes de données + `is_self` + `updated_at`, `WHERE id = ? AND account_id = ?`), `DeleteMemberOfAccount :execrows` (`WHERE id = ? AND account_id = ?`) ;
  - administration : `ListMembers :many` (`SELECT m.*, a.email AS account_email, a.disabled_at AS account_disabled_at FROM members m LEFT JOIN accounts a ON a.id = m.account_id WHERE (CAST(sqlc.narg(query) AS TEXT) IS NULL OR m.id IN (SELECT rowid FROM members_fts WHERE members_fts MATCH CAST(sqlc.narg(query) AS TEXT))) AND (CAST(sqlc.narg(no_account) AS INTEGER) IS NULL OR m.account_id IS NULL) AND (CAST(sqlc.narg(account_id) AS INTEGER) IS NULL OR m.account_id = CAST(sqlc.narg(account_id) AS INTEGER))`, sans `ORDER BY` ni `LIMIT` : tri et page en Go, R7), `GetMember :one` (même jointure, `WHERE m.id = ?`), `CreateMember :one` (avec `sqlc.narg(account_id)`, `is_self` = 0), `UpdateMember :execrows` (données + `account_id = sqlc.narg(account_id)` + `is_self = CASE WHEN account_id IS sqlc.narg(account_id) THEN is_self ELSE 0 END` + `updated_at`, `WHERE id = ?`), `DeleteMember :execrows` ;
  - communes : `CountMembersOfAccount :one`, `ListMembersByBirthDate :many` (`SELECT id, first_name, last_name FROM members WHERE birth_date = ?`), `DeleteMembersOfAccount :execrows`, `DetachMembersOfAccount :execrows` (`UPDATE members SET account_id = NULL, is_self = 0, updated_at = ? WHERE account_id = ?`), `ListMemberIDsOfAccount :many` ;
  - lancer `go generate ./internal/database` et vérifier les types générés (`sqlc.Member` avec `AccountID sql.NullInt64`, `IsSelf int64`, champs facultatifs en `sql.NullString`) ; si sqlc refuse `IS sqlc.narg(...)`, remplacer par `CASE WHEN account_id IS NOT DISTINCT FROM …` ou par un paramètre booléen `keep_self` calculé en Go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: domaine `internal/member`, types et conversions de l'API, formatage et composants partagés du frontend

**⚠️ CRITICAL**: aucune story ne peut commencer avant la fin de cette phase

### Domaine (`internal/member`)

- [X] T003 [P] Écrire `internal/member/member_test.go` (tests en table, doivent échouer avant T004 ; aujourd'hui = 2026-11-01) : **noms** — `"  Jean   Pierre "` → `"Jean Pierre"` ; vide ou espaces seuls → `FieldError{firstName|lastName, ErrRequired}` ; **date de naissance** — `2026-11-01` OK, `2026-11-02` → `ErrBirthFuture`, `1906-11-01` OK (120 ans pile), `1906-10-31` → `ErrBirthTooOld` ; **adresse électronique** — `" Claire@Exemple.FR "` → `"claire@exemple.fr"` ; `"claire"`, `"Claire <c@e.fr>"`, `"a@"` → `ErrEmailFormat` ; vide → absente ; **téléphone** (`NormalizePhone`) — `"06 12 34 56 78"`, `"06.12.34.56.78"`, `"06-12-34-56-78"` → `"0612345678"` ; `"+32 470 12 34 56"` → `"+32470123456"` ; `"(+33) 6 12 34 56 78"` → `"+33612345678"` ; `"+33 (0)6"` → `ErrPhoneFormat` ; `"0612"`, `"+1"`, `"1234567890"` (10 chiffres sans 0 initial), `"abc"`, `"+1234567890123456"` (16 chiffres) → `ErrPhoneFormat` ; vide → absent ; **adresse** — trois champs vides → aucune adresse ; `line1` seul → `ErrAddressIncomplete` sur `postalCode` et `city` ; `line2` seul → `ErrAddressLine2Alone` sur `addressLine2` et pas d'autre erreur ; `postalCode "7500"` ou `"75 001"` → `ErrPostalCode` ; **cumul** — prénom vide + date future + téléphone invalide → trois erreurs dans un seul `FieldErrors` ; **âge** (`Age(birth, today)`) — né le 2015-11-01 → 11 ; né le 2015-11-02 → 10 ; né le 2008-02-29 au 2026-02-28 → 17 et au 2026-03-01 → 18 ; **IdentityKey** — `("Léo", "MARTIN")` égal à `("leo", " martin ")` ; **SortMine** — fiche `IsSelf` d'abord, puis prénom replié (« Élodie » avant « Zoé »), puis nom replié, puis id ; **SortAdmin** — nom replié, prénom replié, id
- [X] T004 Implémenter `internal/member/member.go` (commentaire de package : « Règles des fiches membre (research.md 005 › R2, R3, R7, R11). ») : constantes `MaxPerAccount = 20`, `MaxName = 80`, `MaxEmail = 254`, `MaxAddress = 100`, `MaxAge = 120` ; erreurs sentinelles `ErrRequired`, `ErrBirthFuture`, `ErrBirthTooOld`, `ErrEmailFormat`, `ErrPhoneFormat`, `ErrAddressIncomplete`, `ErrAddressLine2Alone`, `ErrPostalCode` ; `type FieldError struct { Field string; Err error }` et `type FieldErrors []*FieldError` (implémente `error`) ; `type Input struct { FirstName, LastName string; BirthDate season.Date; Email, Phone, AddressLine1, AddressLine2, PostalCode, City string }` ; `type Clean struct { FirstName, LastName string; BirthDate season.Date; Email, Phone, AddressLine1, AddressLine2, PostalCode, City *string }` ; `Validate(in Input, today season.Date) (Clean, error)` : noms par `strings.Join(strings.Fields(s), " ")`, champs facultatifs nettoyés (vide → `nil`), règles du data-model (date ≤ today et ≥ today − 120 ans ; `net/mail.ParseAddress` dont `Name` est vide et `Address` égal à la saisie nettoyée, puis minuscules ; `NormalizePhone` ; adresse tout ou rien ; code postal `^[0-9]{5}$`), toutes les erreurs cumulées ; `NormalizePhone(s string) (string, error)` : retirer espaces (y compris insécables), points, tirets et parenthèses ; puis si commence par `+` : `^\+[0-9]{8,15}$`, sinon `^0[0-9]{9}$`, sinon `ErrPhoneFormat` ; `Age(birth, today season.Date) int` (années révolues ; un 29 février est atteint le 1er mars les années non bissextiles) ; `IdentityKey(first, last string) string` (`textfold.Fold(first) + "\x00" + textfold.Fold(last)`) ; `SortMine` et `SortAdmin` sur un type `Sortable { ID int64; FirstName, LastName string; IsSelf bool }` (ou fonctions de comparaison génériques utilisées avec `slices.SortFunc`) ; faire passer T003 ; ajouter des méthodes sur `season.Date` si nécessaire (soustraction d'années) dans `internal/season/date.go` avec leurs tests

### API (types et conversions communs)

- [X] T005 Créer `internal/api/members.go` : constantes de tags `tagMyMembers = "Mes membres"`, `tagMembers = "Membres"` ; messages (contrat › Erreurs) : `MsgMemberGone = "Ce membre n'existe plus."`, `MsgFirstNameRequired = "Saisissez le prénom."`, `MsgLastNameRequired = "Saisissez le nom."`, `MsgBirthFuture = "La date de naissance ne peut pas être dans le futur."`, `MsgBirthTooOld = "Vérifiez la date de naissance : elle remonte à plus de 120 ans."`, `MsgEmailFormat = "Saisissez une adresse électronique valide (par exemple nom@exemple.fr)."`, `MsgPhoneFormat = "Saisissez un numéro à 10 chiffres (06 12 34 56 78) ou au format international (+32 …)."`, `MsgAddressIncomplete = "Complétez l'adresse : ligne d'adresse, code postal et commune."`, `MsgAddressLine2Alone = "Saisissez d'abord la ligne d'adresse."`, `MsgPostalCode = "Saisissez un code postal à 5 chiffres."`, `MsgSelfTaken = "Une autre fiche vous désigne déjà."`, `MsgDuplicate = "Un membre portant ces nom, prénom et date de naissance existe déjà. Confirmez pour enregistrer quand même."`, `MsgMyMembersFull = "Vous gérez déjà 20 membres. Contactez le bureau pour en ajouter."`, `MsgAccountMembersFull = "Ce compte gère déjà 20 membres."` ; types de sortie `Member`, `PostalAddress`, `MemberAccount`, `AdminMember` (embarque `Member` + `Account *MemberAccount`), `MyMemberList { Items []Member; Actions MyMemberActions{Add, MarkSelf bool} }`, `AdminMemberPage` (champs JSON exacts de `contracts/api.md`) ; type d'entrée commun `MemberBody` : `FirstName string \`json:"firstName" maxLength:"80"\``, `LastName` idem, `BirthDate string \`json:"birthDate" format:"date"\``, `Email string \`json:"email,omitempty" maxLength:"254"\``, `Phone` (`maxLength:"30"`), `AddressLine1`, `AddressLine2`, `City` (`maxLength:"100"`), `PostalCode` (`maxLength:"10"`), `ConfirmDuplicate bool \`json:"confirmDuplicate,omitempty"\`` avec `doc` en français ; fonctions : `toMember(row, today)` (calcule `age` via `member.Age`, `address` nul si `address_line1` nul), `toAdminMember`, `memberFieldErrors(err) error` (traduit `member.FieldErrors` en un seul `FieldErrors` huma aux emplacements `body.<champ>` ; sentinelles → messages ci-dessus), `(s *server) validateMember(body MemberBody) (member.Clean, error)` (date via `season.ParseDate`, erreur → `body.birthDate` « Saisissez une date valide. » existant `MsgInvalidDate`), `checkDuplicate(ctx, q, clean, excludeID int64, confirmed bool) error` (`ListMembersByBirthDate`, compare `member.IdentityKey`, renvoie `FieldErrors("body.confirmDuplicate", MsgDuplicate)` si trouvé et non confirmé), `checkAccountCapacity(ctx, q, accountID int64, msg string) error` (`CountMembersOfAccount` ≥ `member.MaxPerAccount` → `huma.Error409Conflict(msg)`), `isUniqueSelfViolation(err) bool` (détecte la violation de `members_one_self` : message contenant `UNIQUE constraint failed: members.account_id`), `(s *server) logMemberAction(ctx, action string, memberID int64, accountID *int64, space string, attrs ...any)` (attributs `action`, `actor_id`, `member_id`, `account_id` si non nul, `space`), `func (s *server) registerMembers(api huma.API)` appelant `s.registerMyMembers(api)` et `s.registerAdminMembers(api)` (définies dans les phases suivantes ; laisser des corps vides compilables)
- [X] T006 Enregistrer `s.registerMembers(api)` dans `New` de `internal/api/api.go`, après `registerMembershipFees`

### Frontend partagé

- [X] T007 [P] Ajouter à `web/app/lib/format.ts` : `formatPhone(phone: string)` (numéro français à 10 chiffres → paires séparées par des espaces insécables fins ou espaces simples « 06 12 34 56 78 » ; sinon tel quel) et `formatAge(age: number)` (« 1 an », « 11 ans », « moins d'un an » pour 0) ; tests dans `web/app/lib/format.test.ts`
- [X] T008 [P] Créer `web/app/members/MemberFields.tsx` : composant contrôlé par `react-hook-form` (`register`, `formState.errors`) rendant trois `fieldset` avec `legend` (« Identité », « Coordonnées », « Adresse postale ») et les champs du contrat `ui-routes.md` › Formulaire (`Field` existant : `maxLength` 80/254/100/5, `required` sur prénom, nom, date ; `type="date"`, `type="email"`, `type="tel"` + `inputmode="tel"` + aide « 10 chiffres, ou format international (+32 …) », code postal `inputmode="numeric"` ; aide d'adresse « Laissez vide ou complétez les trois champs obligatoires. ») ; noms de champs identiques aux clés JSON du `MemberBody` pour que `applyProblemToForm` place les erreurs `body.<champ>` ; prop `autocompleteSelf?: boolean` (ajoute `autocomplete` `given-name`, `family-name`, `email`, `tel`, `street-address`… seulement si vrai) ; types issus de `schema.d.ts` (aucune redéfinition à la main)
- [X] T009 [P] Créer `web/app/members/DuplicateConfirm.tsx` : si `errors.confirmDuplicate` existe, affiche le message dans une `Alert` `warning` et une case « Enregistrer quand même » liée au champ `confirmDuplicate` du formulaire ; focus placé sur la case à son apparition ; sinon ne rend rien
- [X] T010 Régénérer (`go tool task gen`) puis créer `web/app/api/members.ts` sur le modèle de `web/app/api/accounts.ts` : `myMemberListQuery()`, `myMemberQuery(id)`, `memberListQuery(filters: { q?: string; account?: string; page?: number })`, `memberQuery(id)`, clés d'invalidation `myMemberListKey`, `memberListKey`

**Checkpoint**: le domaine passe ses tests, l'API compile avec les types communs, les composants partagés sont prêts

---

## Phase 3: User Story 1 - Gérer mes membres depuis mon compte (Priority: P1) 🎯 MVP

**Goal**: un titulaire de compte liste, crée, modifie et supprime ses fiches depuis « Mes membres », avec la fiche « Vous » et le contrôle de doublon

**Independent Test**: avec un compte membre sans fiche, constater l'état vide, créer trois fiches (dont « Vous »), en modifier une, en supprimer une ; la rubrique reflète chaque action

### Tests for User Story 1

- [X] T011 [P] [US1] Écrire `internal/api/members_me_test.go` (aujourd'hui = 2026-11-01, membre connecté via `env.LoggedIn(env.Seed(auth.RoleMember, true))`) : liste vide → `items: []`, `actions: {add: true, markSelf: true}` ; création de « Léo Martin » `2015-04-12` → `201`, `age` 11, `isSelf` faux, `address` nul ; création avec `isSelf: true` → `actions.markSelf` faux ensuite ; seconde fiche `isSelf: true` → `422` `body.isSelf` `MsgSelfTaken` ; tri de la liste (« Vous » d'abord puis « Élodie » avant « Zoé ») ; `PUT` avec téléphone `"06.12.34.56.78"` → `phone` `"0612345678"` ; `PUT` qui décoche `isSelf` → `markSelf` redevient vrai ; `DELETE` → `204` puis `GET` → `404` `MsgMemberGone` ; `422` cumulés (prénom vide, date `2026-11-02`, téléphone `"0612"`, adresse partielle) dans une seule réponse aux emplacements du contrat ; doublon : seconde création « léo MARTIN » même date → `422` `body.confirmDuplicate` `MsgDuplicate`, puis avec `confirmDuplicate: true` → `201` ; le doublon est aussi détecté contre une fiche d'un **autre** compte, sans que le message contienne l'adresse de ce compte ; `PUT` sans changement d'identité sur une fiche en doublon → pas de `422` doublon ; 21e création → `409` `MsgMyMembersFull` et `actions.add` faux à 20 ; journal : `member.create`, `member.update`, `member.delete` avec `space=me`, `actor_id`, `member_id`, `account_id`, sans prénom ni nom (vérifier via `LogBuffer` que `"Martin"` n'apparaît pas)

### Implementation for User Story 1

- [X] T012 [US1] Créer `internal/api/members_me.go` : `registerMyMembers` (accès `auth.Member`, tag `tagMyMembers`) avec `list-my-members` (`GET /me/members`), `get-my-member` (`GET /me/members/{id}`), `create-my-member` (`POST /me/members`, `DefaultStatus: 201`), `update-my-member` (`PUT /me/members/{id}`), `delete-my-member` (`DELETE /me/members/{id}`, `DefaultStatus: 204`) ; corps `MyMemberBody` = `MemberBody` + `IsSelf bool \`json:"isSelf,omitempty"\`` ; le compte vient **uniquement** de `CurrentAccount(ctx).ID` ; n'utiliser que les requêtes `…OfAccount` / `…ForAccount` (R4) ; création et modification dans `s.Store.WriteTx` : `validateMember` → si `IsSelf`, vérifier qu'aucune autre fiche du compte n'est « Vous » (`422` `body.isSelf`) → `checkDuplicate` (à la modification, seulement si `IdentityKey` ou date changent, en excluant la fiche) → à la création `checkAccountCapacity(…, MsgMyMembersFull)` → écriture ; violation de `members_one_self` traduite en `422` `body.isSelf` ; `:execrows` = 0 → `404` `MsgMemberGone` ; liste triée par `member.SortMine` avec `actions` (`add` = moins de 20, `markSelf` = aucune fiche `is_self`) ; journalisation `member.create|update|delete` avec `space="me"` ; faire passer T011
- [X] T013 [US1] Activer la rubrique : retirer `comingSoon: true` de `mes-membres` dans `web/app/shell/navigation.ts`, retirer `"mes-membres"` de `memberComingSoon` dans `web/app/routes.ts` et y ajouter `route("mes-membres", "routes/app/member/my-members-layout.tsx", [index("routes/app/member/my-members.tsx"), route("nouveau", "routes/app/member/my-member-new.tsx"), route(":id", "routes/app/member/my-member-detail.tsx")])`
- [X] T014 [P] [US1] Créer `web/app/routes/app/member/my-members-layout.tsx` (fil d'Ariane « Mes membres », `Outlet`, sur le modèle de `accounts-layout.tsx`) et `web/app/routes/app/member/my-members.tsx` : `clientLoader` avec `ensureQueryData(myMemberListQuery())` ; en-tête « Mes membres » + introduction + bouton « Ajouter un membre » (icône `UserPlus`) si `actions.add`, sinon texte « 20 membres au plus par compte. Contactez le bureau pour en ajouter. » ; état vide (texte du contrat + bouton principal) ; liste `<ul>` de cartes (lien `h2` « {prénom} {nom} » vers `/mes-membres/{id}`, `formatAge`, `Badge` « Vous ») ; notification de retour (`location.state.flash`, région `aria-live`) comme `accounts.tsx`
- [X] T015 [US1] Créer `web/app/routes/app/member/my-member-new.tsx` : formulaire `react-hook-form` avec `MemberFields`, case « Il s'agit de moi » (`isSelf`) si `actions.markSelf` ; la cocher préremplit `email` vide avec l'adresse du compte (`get-session`) et active `autocompleteSelf` ; `DuplicateConfirm` ; soumission `create-my-member`, erreurs via `applyProblemToForm`, `409` dans une `Alert` `danger` ; succès → invalidation `myMemberListKey`, retour à `/mes-membres` avec flash « Membre ajouté. »
- [X] T016 [US1] Créer `web/app/routes/app/member/my-member-detail.tsx` : `clientLoader` `ensureQueryData(myMemberQuery(id))` ; `404` → rendu de la page « Page introuvable » existante (`routes/app/not-found.tsx` ou mécanisme utilisé par `account-detail.tsx`) ; même formulaire prérempli (le champ téléphone reçoit la valeur normalisée de l'API, sans mise en forme) ; case « Il s'agit de moi » visible si la fiche est « Vous » ou si `markSelf` (liste préchargée) ; bouton « Enregistrer » (`update-my-member`), bouton « Supprimer » (danger, `Trash2`) avec `ConfirmDialog` « Supprimer {prénom} {nom} ? » / « La fiche et ses informations seront effacées définitivement. » ; `404` en écriture → `Alert` + invalidation de la liste ; succès → retour à la liste avec flash « Modifications enregistrées. » / « Membre supprimé. »
- [X] T017 [US1] Écrire `web/e2e/my-members.spec.ts` (test « mes membres (005) ») : membre connecté ; état vide et axe ; ajout de sa fiche avec « Il s'agit de moi » (adresse préremplie) ; ajout de « Léo Martin » 2015-04-12 → carte « Léo Martin », « 11 ans » (ou âge calculé à la date d'exécution), la fiche « Vous » est la première ; modification du téléphone `06.12.34.56.78` → valeur `06 12 34 56 78` affichée ; erreurs de champ (prénom vide, code postal `7500`) affichées sur les champs ; doublon « léo martin » même date → case « Enregistrer quand même » puis création ; suppression avec dialogue → « Membre supprimé. » ; axe sur la liste, le formulaire et le dialogue ; parcours au clavier de l'ajout

**Checkpoint**: « Mes membres » est utilisable de bout en bout (MVP)

---

## Phase 4: User Story 2 - Ne jamais voir ni modifier les membres d'un autre compte (Priority: P1)

**Goal**: garantir le cloisonnement des fiches par compte, côté serveur, quel que soit le moyen d'accès

**Independent Test**: créer une fiche avec le compte A, puis, connecté avec B, tenter de la lister, de l'ouvrir, de la modifier et de la supprimer par l'interface et par des requêtes directes

### Tests for User Story 2

- [X] T018 [P] [US2] Écrire `internal/api/members_isolation_test.go` : comptes membres A et B, administrateur C ; A crée une fiche `fa` ; une fiche sans compte `fn` est insérée directement par une requête sqlc `CreateMember` (en attendant US4) ; pour chaque cible (`fa`, `fn`, identifiant inexistant 99999) et chaque opération de B (`GET`, `PUT` avec corps valide, `DELETE` sur `/api/me/members/{id}`) → `404` et `detail` exactement `MsgMemberGone` (réponses identiques, corps comparés) ; après ces tentatives, `fa` est inchangée (relue par A : mêmes champs, même `updatedAt`) ; `GET /api/me/members` de B ne contient ni `fa` ni `fn` ; C (administrateur) dans l'espace membre ne voit pas `fa` dans `/api/me/members` ; un corps `PUT /api/me/members/{id}` (fiche de B) contenant un champ `accountId` → `422`, et le rattachement de la fiche ne change pas ; B membre → `403` sur `GET /api/admin/members` et `GET /api/admin/members/{fa}`
- [X] T019 [P] [US2] Vérifier dans `internal/api/access_test.go` que les 5 opérations `Mes membres` sont parcourues par la matrice (anonyme → 401, membre accepté, administrateur accepté : `404` sur `/me/members/1` compte comme accepté) ; ajouter des corps minimaux valides si la matrice en exige pour `POST`/`PUT`

### Implementation for User Story 2

- [X] T020 [US2] Relire `internal/api/members_me.go` et `internal/database/queries/members.sql` : aucune requête de l'espace membre sans condition `account_id = ?` ; `MyMemberBody` n'a pas de champ `accountId` ; vérifier dans `web/app/api/openapi.json` régénéré que `MyMemberBody` et `AdminMemberBody` ont `additionalProperties: false` ; corriger jusqu'au vert de T018
- [X] T021 [US2] Compléter `web/e2e/my-members.spec.ts` (test « cloisonnement (005) ») : le membre A crée une fiche et relève son URL ; connecté en membre B, ouvrir cette URL → « Page introuvable » ; la liste de B ne contient pas la fiche ; B ouvre `/admin/membres` → « Accès refusé »

**Checkpoint**: le cloisonnement est démontré par les tests API et e2e

---

## Phase 5: User Story 3 - Retrouver et consulter tous les membres (Priority: P1)

**Goal**: un administrateur liste, recherche, filtre et consulte toutes les fiches

**Independent Test**: avec des fiches réparties sur plusieurs comptes, ouvrir la rubrique, rechercher un fragment de nom, paginer, ouvrir une fiche

### Tests for User Story 3

- [X] T022 [P] [US3] Écrire `internal/api/members_admin_test.go` (partie consultation) : jeu de fiches sur deux comptes + une sans compte (insertion via `/api/me/members` et `CreateMember`) ; `GET /api/admin/members` → tri « Dupont Élodie », « Dupont Zoé », « Émery … », « Martin … » (nom puis prénom repliés, « Émery » avant « Martin ») ; `account` renseigné (`id`, `email`, `disabled`) ou `null` ; `total`, `page`, `pageSize` ; `pageSize=2&page=2` renvoie les 3e et 4e ; `q=elod` trouve « Élodie » ; `q=dupo zoe` (deux fragments) ne renvoie que Zoé Dupont ; `q=3456` trouve un téléphone `0612345678` ; `q=ex` → `422` `query.q` `MsgSearchTooShort` ; `account=none` ne renvoie que la fiche sans compte ; `account={id}` ne renvoie que les fiches de ce compte ; `GET /api/admin/members/{id}` → `AdminMember` complet ; inexistant → `404` `MsgMemberGone` ; compte désactivé (via `disable-account`) → `account.disabled` vrai, fiche toujours listée ; **volume (SC-004)** : `TestListMembersVolume` (ignoré avec `-short`) insère 2 000 fiches réparties sur 200 comptes en une transaction (`CreateMember` en boucle, noms accentués variés), puis vérifie que `GET /api/admin/members?q=dupo&page=3` et la liste complète répondent chacune en moins de 500 ms, et que le tri est correct sur un échantillon ; `BenchmarkListMembers` dans le même fichier pour suivre l'évolution (évolution prévue si le temps se dégrade : collation `fold`, research.md › R7)

### Implementation for User Story 3

- [X] T023 [US3] Créer `internal/api/members_admin.go` avec `registerAdminMembers` (accès `auth.Admin`, tag `tagMembers`) et, pour cette story, `list-members` (`GET /admin/members`, entrée `Q string query:"q" maxLength:"100"`, `Account string query:"account" pattern:"^(none|[0-9]+)$"`, `Page`, `PageSize` comme `ListAccountsInput`) et `get-member` (`GET /admin/members/{id}`) ; `list-members` : `FTSTrigramQuery` (échec → `422` `query.q`), `ListMembers` sur `s.Store.Read`, conversion, tri `member.SortAdmin`, découpe `[(page-1)*size : page*size]` bornée, `total` = longueur filtrée ; faire passer T022
- [X] T024 [US3] Activer la rubrique : retirer `comingSoon: true` de `membres` dans `web/app/shell/navigation.ts`, retirer `"membres"` de `adminComingSoon` dans `web/app/routes.ts` et ajouter sous `prefix("admin")` : `route("membres", "routes/app/admin/members-layout.tsx", [index("routes/app/admin/members.tsx"), route("nouveau", "routes/app/admin/member-new.tsx"), route(":id", "routes/app/admin/member-detail.tsx")])` (créer `member-new.tsx` comme espace réservé renvoyant vers la liste jusqu'à US4, ou n'ajouter cette route qu'en T029)
- [X] T025 [P] [US3] Créer `web/app/routes/app/admin/members-layout.tsx` (fil d'Ariane « Membres ») et `web/app/routes/app/admin/members.tsx` sur le modèle de la liste de `accounts.tsx` : critères `q`, `account`, `page` lus et écrits dans l'URL ; champ « Rechercher un membre » avec l'indication des 3 caractères ; `Select` « Rattachement » (« Tous » / « Sans compte ») ; `Table` `<caption>` « Membres de l'association » : Nom (lien « {nom} {prénom} » `th scope="row"`) · Âge (`formatAge`) · Compte (adresse + `Badge` « désactivé », ou « Sans compte » en texte secondaire) dans `ScrollRegion` ; total (« 128 membres ») ; `Pagination` ; aucun résultat → message et bouton « Effacer la recherche » ; flash de retour
- [X] T026 [US3] Créer `web/app/routes/app/admin/member-detail.tsx` (version consultation) : `clientLoader` `ensureQueryData(memberQuery(id))`, `404` → « Page introuvable » ; titre « {prénom} {nom} » ; liste de définitions : « Compte de rattachement » (lien `/admin/comptes/{id}` + « désactivé », ou « Sans compte »), « Fiche du titulaire » (« Oui » si `isSelf`), « Date de naissance » (`formatDay`) et âge, coordonnées (`formatPhone`), adresse, « Créée le », « Modifiée le » (`formatDate`)
- [X] T027 [US3] Écrire `web/e2e/admin-members.spec.ts` (test « membres (005) », exécuté en série) : préparer des fiches via l'API (membre connecté) ; administrateur, « Membres » : liste triée, recherche `mart` filtre et l'URL contient `q=mart` ; recherche `ma` → indication des 3 caractères ; ouverture d'une fiche → compte de rattachement avec lien ; axe sur la liste et la fiche

**Checkpoint**: l'administration consulte et retrouve toutes les fiches

---

## Phase 6: User Story 4 - Créer, modifier et supprimer n'importe quel membre (Priority: P2)

**Goal**: un administrateur crée des fiches (avec ou sans compte), les modifie, change ou retire leur rattachement et les supprime

**Independent Test**: créer une fiche rattachée à un compte existant et la retrouver dans « Mes membres » de ce compte ; créer une fiche sans compte puis lui rattacher un compte ; modifier puis supprimer une fiche créée par un membre

### Tests for User Story 4

- [X] T028 [P] [US4] Compléter `internal/api/members_admin_test.go` (partie écriture) : `POST /api/admin/members` avec `accountId` d'un membre → `201`, visible dans `/api/me/members` de ce compte ; avec `accountId: null` → `account` nul ; `accountId` inexistant → `422` `body.accountId` `MsgAccountGone` ; `PUT` changeant `accountId` d'une fiche « Vous » vers un autre compte → `isSelf` faux, disparaît de l'ancien compte, apparaît dans le nouveau ; `PUT` avec `accountId: null` → « Sans compte », `isSelf` faux ; `PUT` sans changement de compte sur une fiche « Vous » → `isSelf` conservé ; rattachement à un compte qui gère déjà 20 fiches → `409` `MsgAccountMembersFull` ; rattachement à un compte désactivé → accepté ; doublon et `confirmDuplicate` comme côté membre ; `DELETE` → `204`, absente de la liste et de `/api/me/members` du compte ; `isSelf` dans le corps admin rejeté (propriété inconnue) ; `PUT` avec `expectedAccountId` différent du rattachement actuel (modifié entre-temps par un autre `PUT`) → `409` `MsgMemberAccountChanged` et fiche inchangée ; `expectedAccountId` absent → `422` (champ obligatoire : « absent » et `null` ne se distinguent pas) ; journal : `member.create` + `member.attach` (création rattachée), `member.update` + `member.attach` (changement de compte, `account_id` résultant absent si sans compte), `member.delete`, tous avec `space=admin`, sans données de fiche

### Implementation for User Story 4

- [X] T029 [US4] Compléter `internal/api/members_admin.go` : `create-member` (`POST /admin/members`, `201`), `update-member` (`PUT /admin/members/{id}`), `delete-member` (`DELETE /admin/members/{id}`, `204`) ; corps `AdminMemberBody` = `MemberBody` + `AccountID *int64 \`json:"accountId" nullable:"true" required:"true"\`` (présent, `null` = sans compte) et, pour la seule modification, `AdminMemberUpdateBody` = `AdminMemberBody` + `ExpectedAccountID *int64 \`json:"expectedAccountId" nullable:"true" required:"true"\`` ; message `MsgMemberAccountChanged = "Le rattachement de ce membre a changé entre-temps. Vérifiez la fiche avant d'enregistrer."` ; à la modification, relire la fiche (`GetMember`) et comparer `account_id` à `ExpectedAccountID` (écart → `409`, rien n'est écrit) ; dans `WriteTx` : `validateMember` → compte cible relu (`GetAccountByID`, absent → `422` `body.accountId` `MsgAccountGone`) → `checkDuplicate` (modification : seulement si identité changée) → `checkAccountCapacity(…, MsgAccountMembersFull)` si le compte cible change ou à la création avec compte → `CreateMember` / `UpdateMember` (la requête remet `is_self` à 0 si le compte change) ; `:execrows` = 0 → `404` ; journal `member.create|update|delete` + `member.attach` quand le rattachement change, `space="admin"` ; faire passer T028
- [X] T030 [P] [US4] Créer `web/app/members/AccountPicker.tsx` : `Combobox` de `@base-ui/react` étiqueté « Compte de rattachement » ; option fixe « Sans compte » en tête ; saisie d'au moins 3 caractères → requête `list-accounts` (`q`, `pageSize` 10) via `$api`/`openapi-react-query`, sans filtrage côté client ; options « {adresse} » (+ « désactivé ») ; message annoncé « Saisissez au moins 3 caractères. » en dessous du seuil et « {n} compte(s) » ; valeur `number | null` intégrée à `react-hook-form` (`Controller`) sous le nom `accountId` pour que l'erreur `body.accountId` s'y affiche ; prop `initialAccount?: { id: number; email: string }` pour afficher la valeur courante
- [X] T031 [US4] Créer `web/app/routes/app/admin/member-new.tsx` : `MemberFields` + `AccountPicker` (valeur initiale « Sans compte ») + `DuplicateConfirm` ; `create-member` ; `409` → `Alert` ; succès → invalidations `memberListKey` et `myMemberListKey`, retour à `/admin/membres` avec flash « Membre ajouté. » ; ajouter la route si elle n'a pas été créée en T024
- [X] T032 [US4] Faire évoluer `web/app/routes/app/admin/member-detail.tsx` : conserver le bloc d'informations de T026 au-dessus, puis le formulaire prérempli (`MemberFields`, `AccountPicker` initialisé, aide « Le signalement “Vous” sera retiré. » quand le compte choisi diffère d'une fiche `isSelf`, `DuplicateConfirm`) ; « Enregistrer » (`update-member`, avec `expectedAccountId` = `account?.id ?? null` lu au chargement) ; sur `409` `MsgMemberAccountChanged` : `Alert`, rechargement de la fiche, saisies non enregistrées conservées ; « Supprimer » avec `ConfirmDialog` danger (textes du contrat) ; `404`/`409` → `Alert` + invalidations ; succès → retour à la liste avec flash
- [X] T033 [US4] Compléter `web/e2e/admin-members.spec.ts` : créer « Jeanne Petit » sans compte → filtre « Sans compte » la montre ; lui rattacher le compte du membre via le Combobox (saisie de 3 caractères, choix au clavier) ; se connecter avec ce membre → elle apparaît dans « Mes membres » ; modifier sa date de naissance ; la supprimer (dialogue) → « Membre supprimé. » ; axe sur le formulaire avec le Combobox ouvert et sur le dialogue

**Checkpoint**: l'administration gère toutes les fiches et leur rattachement

---

## Phase 7: User Story 5 - Choisir le sort des fiches à la suppression d'un compte (Priority: P2)

**Goal**: la suppression d'un compte gérant des fiches impose un choix explicite (effacer ou conserver sans compte), de façon atomique, en refusant une liste périmée

**Independent Test**: supprimer un compte gérant des fiches avec chacune des deux options ; vérifier l'effacement ou la conservation « Sans compte » ; constater le refus si une fiche est ajoutée pendant que le dialogue est ouvert

### Tests for User Story 5

- [X] T034 [P] [US5] Compléter `internal/api/accounts_delete_test.go` : compte sans fiche → `204` sans paramètre (comportement 002 inchangé) ; compte avec deux fiches dont « Vous », sans `members` → `422` `query.members` « Choisissez le sort des fiches gérées par ce compte. » et compte toujours présent ; `members=keep&memberIds=` des deux fiches → `204`, fiches présentes avec `account` nul et `isSelf` faux ; `members=delete&memberIds=…` → `204`, fiches absentes (`GET /api/admin/members/{id}` → `404`) ; `memberIds` incomplet ou contenant un identifiant en trop → `409` « La liste des fiches de ce compte a changé. » et rien n'est modifié ; garde-fous 002 (soi-même, dernier administrateur) prioritaires même avec fiches ; journal `account.delete` avec `members_kept=2` ou `members_deleted=2`

### Implementation for User Story 5

- [X] T035 [US5] Modifier `internal/api/accounts.go` : ajouter au type d'entrée de `delete-account` (nouveau `DeleteAccountInput` embarquant `AccountIDInput`) `Members string \`query:"members" enum:"delete,keep"\`` et `MemberIDs []int64 \`query:"memberIds" explode:"false"\`` avec `doc` français ; messages `MsgMembersChoiceRequired = "Choisissez le sort des fiches gérées par ce compte."`, `MsgMembersChanged = "La liste des fiches de ce compte a changé."` ; dans la fonction passée à `adminWrite`, **avant** `DeleteAccount` : `ListMemberIDsOfAccount` ; aucune fiche → inchangé ; sinon `Members` vide → `FieldErrors("query.members", MsgMembersChoiceRequired)` ; ensemble `MemberIDs` ≠ ensemble lu → `huma.Error409Conflict(MsgMembersChanged)` ; `delete` → `DeleteMembersOfAccount` ; `keep` → `DetachMembersOfAccount` ; journal `account.delete` enrichi de `members_deleted` ou `members_kept` ; mettre à jour `specs/002-admin-account-management/contracts/api.md` par un renvoi vers `specs/005-member-management/contracts/api.md` ; faire passer T034
- [X] T036 [US5] Modifier le dialogue de suppression de `web/app/routes/app/admin/account-detail.tsx` : à l'ouverture, charger `memberListQuery({ account: String(account.id) })` (`pageSize` 20) ; s'il y a des fiches : paragraphe « Ce compte gère {n} fiche(s) de membre : » + liste des noms, puis `RadioGroup` base-ui sans valeur initiale (« Supprimer aussi ces fiches » / « Conserver ces fiches sans compte ») ; la requête `delete-account` envoie toujours `memberIds` (identifiants affichés) et `members` seulement si un choix est fait ; le `422` `query.members` du serveur est affiché sur le groupe (erreur reliée par `aria-describedby`, focus sur le premier bouton radio), sans validation côté client (constitution, principe VI) ; sur `409` `MsgMembersChanged` : dialogue maintenu ouvert, liste rechargée, `Alert` avec le message ; succès : invalidation aussi de `memberListKey`
- [X] T037 [US5] Compléter `web/e2e/admin-accounts-delete.spec.ts` (test « fiches du compte (005) ») : un membre crée deux fiches ; l'administrateur supprime ce compte → le dialogue liste les deux noms ; confirmer sans choix → erreur sur le groupe ; choisir « Conserver ces fiches sans compte » → les fiches apparaissent dans « Membres » filtré « Sans compte » ; second compte avec une fiche, « Supprimer aussi ces fiches » → la fiche n'existe plus ; axe sur le dialogue enrichi

**Checkpoint**: toutes les user stories sont fonctionnelles

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T038 [P] Vérifier dans `internal/api/access_test.go` que les 5 opérations `Membres` sont parcourues (anonyme → 401, membre → 403, administrateur accepté) et que `delete-account` avec ses nouveaux paramètres reste couvert ; compléter les corps minimaux si nécessaire
- [X] T039 [P] Vérifier que les fichiers e2e qui créent des fiches (`my-members.spec.ts`, `admin-members.spec.ts`, `admin-accounts-delete.spec.ts`) n'interfèrent pas entre eux (comptes et noms distincts, ou exécution en série comme `admin-seasons.spec.ts`) et que `navigation.spec.ts` ne s'attend plus à « Bientôt disponible » sur « Mes membres » et « Membres »
- [X] T040 [P] Vérifier le respect de 320 px de large sur les six écrans et le dialogue de suppression de compte (Playwright, viewport 320) et le parcours complet au clavier
- [X] T041 Lancer `go tool task check` (build `CGO_ENABLED=0`, vet, tests Go, génération sans diff, migrations sur base vierge et sur une base de développement existante, typecheck, Biome, Vitest, build, Playwright + axe) et corriger jusqu'au vert
- [ ] T042 Dérouler les parcours manuels de `specs/005-member-management/quickstart.md` (dont la vérification des journaux : aucune donnée de fiche)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : T001 puis T002 (sqlc lit la migration)
- **Foundational (Phase 2)** : après la phase 1 ; T003 → T004 ; T005 après T004 ; T006 après T005 ; T007, T008, T009 en parallèle ; T010 après T006 (génération de l'OpenAPI)
- **US1 (Phase 3)** : après la phase 2
- **US2 (Phase 4)** : après T012 (opérations de l'espace membre) ; T018 et T019 en parallèle ; T021 après T016
- **US3 (Phase 5)** : après la phase 2 ; indépendante de US1 côté API (les tests insèrent des fiches par `CreateMember`), mais T022 utilise `/api/me/members` pour une partie du jeu de données : si US1 n'est pas faite, insérer toutes les fiches par sqlc
- **US4 (Phase 6)** : après US3 (même fichier `members_admin.go`, même écran de fiche)
- **US5 (Phase 7)** : T034 et T035 (API) **juste après T012**, avant toute fusion de US1 : sinon `delete-account` renvoie une erreur 500 pour un compte qui a une fiche « Vous » (`CHECK`), et `ON DELETE SET NULL` détache les autres fiches sans demander quoi en faire. T036 et T037 (écran) après US3 (le dialogue utilise `list-members?account=`)
- **Polish (Phase 8)** : après toutes les stories

### User Story Dependencies

- **US1** : aucune autre story
- **US2** : vérifie US1 (pas de code propre hors corrections)
- **US3** : aucune autre story
- **US4** : étend US3
- **US5** : API (T034, T035) après US1, sans dépendre de US3 ; écran (T036, T037) après US3, qui fournit `list-members`

### Within Each User Story

- Tests d'API écrits d'abord et en échec, puis implémentation jusqu'au vert
- API avant écrans ; régénération des types (`go tool task gen`) après chaque changement d'opération
- e2e en dernier

### Parallel Opportunities

- Phase 2 : T003 (tests du domaine) avec T007, T008, T009 (frontend)
- US1 : T011 en parallèle de T013/T014 une fois l'OpenAPI régénérée
- US2 : T018 et T019
- US3 et US1 peuvent avancer en parallèle après la phase 2 (fichiers distincts : `members_me.go` / `members_admin.go`, routes `member/` / `admin/`)
- US4 : T028 et T030
- US5 : T034 en parallèle de US2 (fichier `accounts_delete_test.go`)
- Polish : T038, T039, T040

---

## Parallel Example: User Story 1

```text
# Après la phase 2 :
Task: "T011 [US1] Tests d'API de l'espace membre dans internal/api/members_me_test.go"
Task: "T013 [US1] Activer la rubrique dans web/app/shell/navigation.ts et web/app/routes.ts"
Task: "T014 [US1] Liste « Mes membres » dans web/app/routes/app/member/my-members.tsx"
```

## Parallel Example: User Stories 1 et 3

```text
# Deux développeurs après la phase 2 :
Dev A: T011 → T012 → T013 → T014 → T015 → T016 → T017   (espace membre)
Dev B: T022 → T023 → T024 → T025 → T026 → T027          (administration, consultation)
```

---

## Implementation Strategy

### MVP First (User Story 1 + 2)

1. Phase 1 et phase 2
2. Phase 3 (US1) : « Mes membres » de bout en bout
3. Phase 4 (US2) : le cloisonnement est démontré — **condition de mise en service** de US1 (spec, US2 « Why this priority »)
4. T034 et T035 (US5, API) : `delete-account` traite les fiches. Jusqu'à l'arrivée du dialogue (T036), un administrateur qui supprime un compte gérant des fiches voit la suppression refusée (`422` `query.members`, affiché comme une erreur dans le dialogue de 002) : aucune perte silencieuse
5. **STOP and VALIDATE** : parcours 1 à 3 de `quickstart.md`

### Incremental Delivery

1. Setup + Foundational
2. US1 + US2 + API de US5 → livrable (les familles saisissent leurs fiches)
3. US3 → le bureau consulte et recherche
4. US4 → le bureau crée, corrige et rattache
5. US5 (écran) → le dialogue de suppression d'un compte propose le choix ; l'API correspondante a été livrée avec US1 (MVP, étape 4)

### Notes

- Commit après chaque tâche ou groupe logique (gitmoji, message court, validation de l'utilisateur avant commit)
- Commentaires des requêtes sqlc en ASCII uniquement
- `s.today()` pour toute date du jour, jamais `time.Now`
- Aucune donnée de fiche dans les journaux
