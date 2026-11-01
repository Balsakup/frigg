---

description: "Liste des tâches : adhésions"
---

# Tasks: Adhésions

**Input**: Design documents from `specs/006-memberships/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : tests d'autorisation (principes II et III), tests
d'API, Playwright + axe sur chaque parcours touché (Workflow). Le cloisonnement
par compte a sa propre matrice (`memberships_isolation_test.go`). Les règles du
domaine (transitions, actions, montant, aide, référence, IBAN, chiffres) sont
couvertes par des tests en table.

**Organization**: une phase par user story, dans l'ordre de priorité de la spec
(US1 à US5 en P1 ; US6, US7 en P2 ; US8 en P3). US4 (payer) s'appuie sur la
table des paramètres créée en phase 1 ; son écran de paramétrage est livré par
US5, qui peut être faite avant ou en parallèle.

**Conventions** (constitution 1.4.0) : identifiants, tables, champs JSON et
valeurs en anglais ; commentaires, messages, textes d'interface et e-mails en
français. Commentaires des requêtes sqlc en ASCII uniquement (sqlc échoue sur
un commentaire non ASCII). Nouvelle migration `00008_memberships.sql`
(append-only). Jamais de nom, de motif, de commentaire ni de coordonnées
bancaires dans les journaux. Montants en centimes d'euro.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US8)

## Path Conventions

Un module Go à la racine (`internal/`) et le SPA dans `web/` (voir plan.md ›
Project Structure). Code existant à réutiliser :

- API : `internal/api/api.go` (`register`, `s.today()`, `CurrentAccount(ctx)`,
  `s.internalError`), `internal/api/errors.go` (`FieldErrors(location,
  messages...)`, `FieldErrorDetails`), `internal/api/members.go`
  (`toMember`, `memberRecord`, `logMemberAction`, `MsgMemberGone`),
  `internal/api/members_me.go` (`ownAccount(ctx)`, motif des requêtes
  `…OfAccount`), `internal/api/members_admin.go`,
  `internal/api/membership_fees.go` (`toFeeOutput`, `seasonState`,
  `nullInt64`, `nullString`, `MsgSearchTooShort` via `accounts.go`),
  `internal/api/seasons.go` (`buildSeasons`, `optionalString`, libellés
  calculés), `internal/api/mail.go` (`enqueueMail`, `defaultAssociationName`),
  `internal/database/fts.go` (`FTSTrigramQuery`), `internal/textfold`
  (`Fold`), `internal/season` (`Date`, `ParseDate`, `MustDate`, `StateOf`,
  `Open`, `Closed`, `Draft`, `Fee`, `FieldError`, `FieldErrors`).
- Tests Go : `internal/api/apitest` (`New`, `Seed`, `LoggedIn`, `SetToday`,
  `FlushMail`, `Mail` (faux expéditeur `mailtest.Sender`), `Logs`
  (`LogBuffer`), `Store`, `Client().Do`, `Response.JSON`), matrice
  d'autorisation automatique `internal/api/access_test.go`.
- Frontend : `web/app/api/members.ts`, `web/app/api/membership-fees.ts`,
  `web/app/api/seasons.ts` (motif des requêtes et invalidations),
  `web/app/api/problem.ts` (`applyProblemToForm`, `isProblem`),
  `web/app/routes/app/admin/members.tsx` (liste filtrée et paginée par
  l'URL), `web/app/routes/app/admin/account-detail.tsx` (RadioGroup base-ui
  inline, dialogue), `web/app/members/AccountPicker.tsx` (Combobox base-ui),
  `web/app/ui/` (`Field`, `Select`, `Table`, `Pagination`, `Badge`, `Alert`,
  `ConfirmDialog`, `ScrollRegion`, `Button`, `Icon`), `web/app/lib/format.ts`
  (`formatAmount`, `formatDay`, `formatAge`), `web/app/shell/Page.tsx`.
- e2e : `web/e2e/test.ts`, `web/e2e/axe.ts`, `web/e2e/mailpit.ts`,
  `internal/tools/e2eseed/main.go`.
- Régénération : `go tool task gen` (sqlc, OpenAPI, types TypeScript) ;
  contrôles : `go tool task check`.

**Date de référence des exemples** : « aujourd'hui » = 2026-11-01
(Europe/Paris) dans les tests Go, via `env.SetToday`. Jeu de données type :
saison 2026-2027 (2026-09-01 → 2027-08-31) ouverte, tarifs « Adulte »
(6000, renouvellement 4000), « Jeune » (3000), « Bienfaiteur » (0).

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: tables et requêtes SQL

- [X] T001 Créer `internal/database/migrations/00008_memberships.sql` (commentaires français, références « research.md 006 › R1 / R7 ») :
  - `CREATE TABLE memberships (id INTEGER PRIMARY KEY, member_id INTEGER NOT NULL REFERENCES members (id) ON DELETE CASCADE, season_id INTEGER NOT NULL REFERENCES seasons (id) ON DELETE RESTRICT, fee_id INTEGER NOT NULL REFERENCES membership_fees (id) ON DELETE RESTRICT, status TEXT NOT NULL CHECK (status IN ('pending', 'validated', 'refused', 'cancelled')), declared_kind TEXT CHECK (declared_kind IN ('first', 'renewal')), kind TEXT CHECK (kind IN ('first', 'renewal')), amount_cents INTEGER CHECK (amount_cents BETWEEN 0 AND 1000000), reason TEXT CHECK (reason IS NULL OR length(reason) BETWEEN 1 AND 500), requested_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL, created_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL, requested_at TEXT NOT NULL, decided_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL, decided_at TEXT, cancelled_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL, cancelled_at TEXT, payment_reference TEXT UNIQUE CHECK (payment_reference IS NULL OR length(payment_reference) BETWEEN 8 AND 16), payment_status TEXT CHECK (payment_status IN ('due', 'announced', 'paid')), announced_method TEXT CHECK (announced_method IN ('cash', 'cheque', 'transfer')), paid_method TEXT CHECK (paid_method IN ('cash', 'cheque', 'transfer')), paid_on TEXT CHECK (paid_on IS NULL OR date(paid_on) = paid_on), paid_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL, paid_at TEXT, payment_note TEXT CHECK (payment_note IS NULL OR length(payment_note) BETWEEN 1 AND 200), updated_at TEXT NOT NULL, …) STRICT;` avec les `CHECK` de table : `status <> 'pending' OR (kind IS NULL AND amount_cents IS NULL)` ; `status <> 'validated' OR (kind IS NOT NULL AND amount_cents IS NOT NULL)` ; `payment_status IS NULL OR (status IN ('validated', 'cancelled') AND amount_cents > 0 AND payment_reference IS NOT NULL)` (une adhésion annulée garde son état de paiement, data-model) ; `status <> 'validated' OR amount_cents = 0 OR payment_status IS NOT NULL` ; `(payment_status IS 'paid') = (paid_method IS NOT NULL AND paid_on IS NOT NULL)` ; `payment_status IS NOT 'announced' OR announced_method IS NOT NULL` ; `status <> 'refused' OR reason IS NOT NULL` ; `requested_by IS NULL OR declared_kind IS NOT NULL`
  - index : `CREATE UNIQUE INDEX memberships_one_active ON memberships (member_id, season_id) WHERE status IN ('pending', 'validated');`, `memberships_season_status (season_id, status)`, `memberships_member_id (member_id)`, `memberships_fee_id (fee_id)`, `memberships_status_requested (status, requested_at)`
  - recherche par référence (principe VI, R10) : `CREATE VIRTUAL TABLE memberships_fts USING fts5(payment_reference, content = 'memberships', content_rowid = 'id', tokenize = "trigram remove_diacritics 1");` et triggers `memberships_fts_ai` (`AFTER INSERT`), `memberships_fts_ad` (`AFTER DELETE`, insertion `'delete'`), `memberships_fts_au` (`AFTER UPDATE OF payment_reference` : `'delete'` de l'ancienne valeur puis insertion de la nouvelle) entre `-- +goose StatementBegin/End`, sur le modèle exact de `members_fts` dans `00007_members.sql` ; commentaire « Recherche par fragment de référence de paiement, sans casse (R10). »
  - `CREATE TABLE payment_settings (id INTEGER PRIMARY KEY CHECK (id = 1), cash_enabled INTEGER NOT NULL DEFAULT 0 CHECK (cash_enabled IN (0, 1)), cash_instructions TEXT CHECK (cash_instructions IS NULL OR length(cash_instructions) BETWEEN 1 AND 1000), cheque_enabled INTEGER NOT NULL DEFAULT 0 CHECK (cheque_enabled IN (0, 1)), cheque_payee TEXT CHECK (cheque_payee IS NULL OR length(cheque_payee) BETWEEN 1 AND 120), cheque_address TEXT CHECK (cheque_address IS NULL OR length(cheque_address) BETWEEN 1 AND 300), cheque_instructions TEXT CHECK (… BETWEEN 1 AND 1000), transfer_enabled INTEGER NOT NULL DEFAULT 0 CHECK (transfer_enabled IN (0, 1)), transfer_holder TEXT CHECK (… BETWEEN 1 AND 120), transfer_iban TEXT CHECK (transfer_iban IS NULL OR (length(transfer_iban) BETWEEN 15 AND 34 AND transfer_iban GLOB '[A-Z][A-Z][0-9][0-9]*')), transfer_bic TEXT CHECK (transfer_bic IS NULL OR length(transfer_bic) IN (8, 11)), transfer_instructions TEXT CHECK (… BETWEEN 1 AND 1000), updated_at TEXT NOT NULL, CHECK (cash_enabled = 0 OR cash_instructions IS NOT NULL), CHECK (cheque_enabled = 0 OR cheque_payee IS NOT NULL), CHECK (transfer_enabled = 0 OR (transfer_holder IS NOT NULL AND transfer_iban IS NOT NULL))) STRICT;` puis `INSERT INTO payment_settings (id, updated_at) VALUES (1, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'));`
  - `-- +goose Down` : `DROP TABLE payment_settings;`, les trois triggers `memberships_fts_*`, `DROP TABLE memberships_fts;`, `DROP TABLE memberships;`
- [X] T002 Créer `internal/database/queries/memberships.sql` (commentaires ASCII uniquement) :
  - **espace membre**, toutes jointes à `members m ON m.id = ms.member_id` et conditionnées par `m.account_id = sqlc.arg(account_id)` : `ListMembershipsOfAccount :many` (colonnes de `memberships`, `m.first_name`, `m.last_name`, `m.birth_date`, colonnes du tarif `f.label, f.amount_cents AS fee_amount_cents, f.renewal_amount_cents AS fee_renewal_amount_cents, f.description AS fee_description` ; filtre facultatif `CAST(sqlc.narg(member_id) AS INTEGER) IS NULL OR ms.member_id = …`), `GetMembershipOfAccount :one` (même projection, `WHERE ms.id = ? AND m.account_id = ?`), `CancelMembershipOfAccount :execrows` (`UPDATE memberships SET status = 'cancelled', cancelled_by = ?, cancelled_at = ?, updated_at = ? WHERE id = ? AND status = 'pending' AND member_id IN (SELECT id FROM members WHERE account_id = ?)`), `AnnouncePaymentOfAccount :execrows` (`SET payment_status = 'announced', announced_method = ?, updated_at = ? WHERE id = ? AND status = 'validated' AND payment_status IN ('due', 'announced') AND member_id IN (SELECT id FROM members WHERE account_id = ?)`)
  - **communes** : `CreateMembership :one` (toutes les colonnes d'insertion, `RETURNING id`), `GetMembership :one` (même projection que l'espace membre + `m.account_id`, `a.email AS account_email`, `a.disabled_at AS account_disabled_at` par `LEFT JOIN accounts a ON a.id = m.account_id`, + adresses des auteurs par `LEFT JOIN accounts` aliasés `rb`, `cb`, `db`, `xb`, `pb` sur `requested_by`, `created_by`, `decided_by`, `cancelled_by`, `paid_by`), `ListMemberships :many` (projection de liste + filtres `sqlc.narg` : `season_id`, `status`, `payment` (`'free'` = `status = 'validated' AND amount_cents = 0`, sinon `payment_status = ?`), `member_id`, `query` FTS : `CAST(sqlc.narg(query) AS TEXT) IS NULL OR ms.id IN (SELECT rowid FROM memberships_fts WHERE memberships_fts MATCH CAST(sqlc.narg(query) AS TEXT)) OR ms.member_id IN (SELECT rowid FROM members_fts WHERE members_fts MATCH CAST(sqlc.narg(query) AS TEXT))` (référence **ou** fiche, R10) ; sans `ORDER BY` ni `LIMIT` : tri et page en Go, R10), `ValidateMembership :execrows` (`SET status = 'validated', fee_id, kind, amount_cents, payment_reference, payment_status, decided_by, decided_at, updated_at WHERE id = ? AND status = 'pending'`), `RefuseMembership :execrows` (`SET status = 'refused', reason, decided_by, decided_at, updated_at WHERE id = ? AND status = 'pending'`), `UpdateValidatedMembership :execrows` (`SET fee_id, kind, amount_cents, payment_reference, payment_status, announced_method, updated_at WHERE id = ? AND status = 'validated' AND payment_status IS NOT 'paid'`), `CancelMembership :execrows` (`SET status = 'cancelled', reason, cancelled_by, cancelled_at, updated_at WHERE id = ? AND status = 'validated'`), `RecordMembershipPayment :execrows` (`SET payment_status = 'paid', paid_method, paid_on, paid_by, paid_at, payment_note, updated_at WHERE id = ? AND status = 'validated' AND payment_status IN ('due', 'announced')`), `CancelMembershipPayment :execrows` (`SET payment_status = CASE WHEN announced_method IS NULL THEN 'due' ELSE 'announced' END, paid_method = NULL, paid_on = NULL, paid_by = NULL, paid_at = NULL, payment_note = NULL, updated_at = ? WHERE id = ? AND payment_status = 'paid'`)
  - **aides** : `LastPaymentReference :one` (`SELECT COALESCE(MAX(payment_reference), '') FROM memberships WHERE payment_reference LIKE ? || '%' AND length(payment_reference) = (SELECT MAX(length(payment_reference)) FROM memberships WHERE payment_reference LIKE ? || '%')` : le domaine déduit le numéro suivant), `CountValidatedInSeason :one`, `HasValidatedMembership :one` (`member_id`, `season_id`), `ListSeasonMembershipsForStats :many` (`season_id` ; `status`, `kind`, `amount_cents`, `fee_id`, `f.label`, `payment_status`, `announced_method`), `CountPendingMemberships :one`, `CountMembershipsOfFee :one`, `CountActiveMembershipsOfMember :one` (`status IN ('pending', 'validated')`), `ListMembershipsOfMember :many` (administration : `id`, `season_id`, `status`, `payment_reference`, `payment_status`, `announced_method`, `paid_method`, `paid_on`), `ListActiveSeasonsOfMembers :many` (`member_id, season_id` des adhésions `pending` ou `validated` des fiches d'un compte, pour l'offre)
- [X] T003 [P] Créer `internal/database/queries/payment_settings.sql` (commentaires ASCII) : `GetPaymentSettings :one` (`SELECT * FROM payment_settings WHERE id = 1`), `UpdatePaymentSettings :exec` (toutes les colonnes + `updated_at`, `WHERE id = 1`) ; lancer `go generate ./internal/database` et vérifier les types générés (`sqlc.Membership`, `sqlc.PaymentSetting`, champs facultatifs en `sql.NullString` / `sql.NullInt64`) ; si sqlc refuse une construction (`IS 'paid'` dans une requête, double `LIKE`), la remplacer par une forme équivalente sans changer le comportement

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: domaine `internal/membership`, types et assistants communs de l'API, e-mails, formatage et composants partagés du frontend

**⚠️ CRITICAL**: aucune story ne peut commencer avant la fin de cette phase

### Domaine (`internal/membership`)

- [X] T004 [P] Écrire `internal/membership/membership_test.go` (tests en table, en échec avant T005) : **AppliedAmount** — tarif 6000/4000 : `first` → 6000, `renewal` → 4000 ; tarif 3000 sans renouvellement : `renewal` → 3000 ; tarif 0 → 0 ; **transitions** (`CanCancelRequest`, `CanValidate`, `CanRefuse`, `CanUpdate`, `CanCancel`, `CanAnnounce`, `CanRecordPayment`, `CanCancelPayment`) pour chaque combinaison `status` × `payment_status` × état de saison (`open`, `closed`) : valider ou refuser `pending` admis même saison close ; corriger seulement `validated`, non `paid`, saison non close ; annuler (administration) seulement `validated`, saison non close ; annoncer seulement `validated` avec `due`/`announced` ; constater `due`/`announced`, saison close admise ; annuler la constatation seulement `paid` ; **ActionsFor** — `MemberActions{Cancel, Pay}` et `AdminActions{Validate, Refuse, Update, Cancel, RecordPayment, CancelPayment}` correspondent aux transitions ; `Pay` faux pour une adhésion gratuite ; **PaymentAfterAmount(old, amount, ref)** — 0 → `nil` (pas de paiement), >0 sans référence → `due` et demande de référence, >0 avec paiement `announced` conservé ; **CleanReason** — `"  doublon  "` → `"doublon"`, vide → `ErrReasonRequired`, 501 caractères → refusé
- [X] T005 Créer `internal/membership/membership.go` : types `Status` (`Pending = "pending"`, `Validated`, `Refused`, `Cancelled`), `Kind` (`First = "first"`, `Renewal = "renewal"`), `PaymentStatus` (`Due = "due"`, `Announced`, `Paid`), `Method` (`Cash = "cash"`, `Cheque = "cheque"`, `Transfer = "transfer"`) ; `AppliedAmount(fee season.Fee, k Kind) int64` ; fonctions de transition et `ActionsFor` (T004) ; `CleanReason` (espaces de début et fin retirés, intérieurs réduits, 1 à 500 caractères) ; erreurs `ErrReasonRequired`, `ErrTransition` ; constante `MaxReason = 500`, `MaxNote = 200` ; faire passer T004
- [X] T006 [P] Écrire `internal/membership/reference_test.go` puis `internal/membership/reference.go` : `ReferencePrefix(starts, ends season.Date) string` → `"ADH-2627-"` pour 2026-09-01/2027-08-31, `"ADH-2727-"` pour 2027-01-01/2027-12-31 ; `NextReference(prefix, last string) string` → `last` vide → `ADH-2627-0001` ; `ADH-2627-0041` → `ADH-2627-0042` ; `ADH-2627-9999` → `ADH-2627-10000` (14 caractères) ; numéro sur 4 chiffres au moins (R4)
- [X] T007 [P] Écrire `internal/membership/hint_test.go` puis `internal/membership/hint.go` : `PreviousSeasonHint(hasPrevious bool, previousValidatedCount int64, memberValidatedInPrevious bool) Hint` avec `Hint` = `"yes" | "no" | "unknown"` : pas de saison précédente → `unknown` ; saison précédente sans adhésion validée → `unknown` ; avec adhésions validées mais pas pour ce membre → `no` ; membre validé → `yes` (R6)
- [X] T008 [P] Écrire `internal/membership/payment_test.go` puis `internal/membership/payment.go` : `NormalizeIBAN` (`" fr76 3000 6000 0112 3456 7890 189 "` → `"FR7630006000011234567890189"`) ; `ValidIBAN` (clé modulo 97 ISO 13616 : l'exemple ci-dessus valide ; dernier chiffre modifié → invalide ; `"FR76"` trop court ; caractères non alphanumériques → invalide ; longueur 15 à 34) ; `FormatIBAN` (groupes de 4) ; `NormalizeBIC` + `ValidBIC` (`"BNPAFRPP"`, `"BNPAFRPPXXX"` valides ; `"BNPAFRP"`, `"BNPA1RPP"` invalides : 4 lettres banque, 2 lettres pays, 2 alphanumériques, 3 alphanumériques facultatifs) ; `ValidatePaymentSettings(s Settings) (Settings, error)` : textes nettoyés (vides → absents), longueurs (instructions 1 000, ordre et titulaire 120, adresse 300 ; les retours à la ligne sont conservés dans les textes multilignes), et seulement si le moyen est activé : espèces → instructions obligatoires (`cash.instructions`), chèque → ordre (`cheque.payee`), virement → titulaire (`transfer.holder`) et IBAN (`transfer.iban`, absent → `ErrIBANRequired`, invalide → `ErrIBANInvalid`) ; un BIC saisi est validé même si le moyen est désactivé ; un IBAN saisi invalide est refusé même désactivé ; erreurs cumulées dans `season.FieldErrors` (emplacements `cash.instructions`, `cheque.payee`, `transfer.holder`, `transfer.iban`, `transfer.bic`) ; `EnabledMethods(s Settings) []Method`
- [X] T009 [P] Écrire `internal/membership/stats_test.go` puis `internal/membership/stats.go` : `ComputeStats(rows []StatRow) Stats` où seules les adhésions `validated` comptent : `Validated`, `First`, `Renewal`, `Pending` (adhésions `pending`), `ByFee` (nombre et somme par tarif, ordre du montant total décroissant puis libellé replié), `TotalCents` (somme des montants appliqués), `PaidCents` (`paid`), `DueCents` (`due` + `announced`), `DueByMethod` (`none` = `due`, sinon moyen annoncé) ; cas de test : jeu mixte (validées payées, annoncées, dues, gratuites, refusées, annulées payées) → aucune annulée ni refusée dans les totaux ; `TotalCents = PaidCents + DueCents` + gratuites (0)

### API commune

- [X] T010 Créer `internal/api/memberships.go` :
  - constantes `tagMyMemberships = "Mes adhésions"`, `tagMemberships = "Adhésions"`, `tagSettings = "Paramètres"` et messages du contrat (`MsgMembershipGone = "Cette adhésion n'existe plus."`, `MsgMembershipActive = "Ce membre a déjà une adhésion en cours pour cette saison."`, `MsgMembershipChanged = "Cette adhésion a changé entre-temps."`, `MsgPaidCorrection = "Annulez d'abord la constatation du paiement."`, `MsgSeasonClosedAction = "Cette saison est close."`, `MsgSeasonNotOffered = "Cette saison n'est pas ouverte à l'adhésion."`, `MsgFeeNotInSeason = "Choisissez un tarif de cette saison."`, `MsgReasonRequired = "Indiquez le motif."`, `MsgMethodUnavailable = "Ce moyen de paiement n'est plus proposé. Choisissez-en un autre."`, `MsgReceivedFuture = "La date de réception ne peut pas être dans le futur."`, `MsgReceivedBeforeValidation = "La date de réception ne peut pas précéder la validation de l'adhésion."`, `MsgFeeInUse`, `MsgMemberHasMembership`, et ceux des paramètres) ;
  - types de sortie du contrat (`Membership`, `MyMembership`, `AdminMembership`, `AdminMembershipRow`, `Payment`, `MembershipFeeRef`, `SeasonRef`, `MemberRef`, `AccountRef`, `PaymentMethodInfo` avec champs facultatifs selon `method`, `MembershipOffer`, `MyMembershipList`, `AdminMembershipPage`, `MembershipStats`, `MembershipDashboard`) avec énumérations huma (`enum:"pending,validated,refused,cancelled"`, etc.) ;
  - conversions depuis les lignes sqlc ; `amountCents` = montant figé si `amount_cents` non nul (`amountFixed: true`), sinon `AppliedAmount(tarif actuel, declared_kind)` ; `toFeeRef(fee)` remplit `amounts: { first: AppliedAmount(fee, First), renewal: AppliedAmount(fee, Renewal) }` (principe VI : le frontend ne recalcule aucun montant) ;
  - `formatEuros(cents int64) string` (« 40,00 € », « 1 250,00 € » avec espace insécable fine comme `formatAmount` du frontend, « Gratuit » pour 0 non utilisé dans les e-mails) et son test en table dans `internal/api/memberships_test.go`, utilisé par les e-mails dès US2 ;
  - assistants : `loadSeasons(ctx, q)` (suite des saisons + états via `buildSeasons`/`season.StateOf`), `offeredSeason(seasons, fees, id)` (ouverte, non close, au moins un tarif), `feeOfSeason(fees, seasonID, feeID)`, `assignReference(ctx, q, season, current)` (R4 : `ReferencePrefix` + `LastPaymentReference` + `NextReference`), `activeConflict(err)` (violation de `memberships_one_active` → `409 MsgMembershipActive`), `transitionError` (`:execrows` = 0 après relecture → `409 MsgMembershipChanged`) ;
  - `logMembershipAction(ctx, action string, m sqlc.Membership, space string, attrs ...any)` : `action`, `actor_id`, `membership_id`, `member_id`, `season_id`, `space`, + `status`/`kind`/`method`/`amount_cents` selon les attributs passés ; jamais de motif, commentaire, nom ;
  - `registerMemberships(api)` appelant `registerMyMemberships`, `registerAdminMemberships`, et `registerSettings` ; ajouter l'appel `s.registerMemberships(api)` dans `internal/api/api.go` après `s.registerMembers(api)`
- [X] T011 [P] Étendre `internal/mail/render.go` : `Data` gagne `Member string`, `Season string`, `Amount string` (déjà formaté « 40,00 € »), `Reference string`, `Reason string`, `Method string` ; créer les gabarits `internal/mail/templates/membership-validated.{txt,html}.tmpl` (objet « Adhésion de {Member} validée » ; texte : saison, montant ; si `Link` : « Pour payer l'adhésion, ouvrez ce lien : {Link} » et la référence ; sinon « Aucun paiement n'est demandé. »), `membership-refused` (objet « Demande d'adhésion de {Member} refusée », motif), `membership-cancelled` (objet « Adhésion de {Member} annulée », motif), `membership-paid` (objet « Paiement de l'adhésion de {Member} reçu », montant, moyen) sur le modèle de `invitation.*.tmpl` ; ajouter dans `internal/api/mail.go` `enqueueNotice(ctx, q, template, to string, data mail.Data, path string)` (remplit `Association`, `Link = s.Config.BaseURL + path` si `path` non vide) et `membershipRecipient(ctx, q, memberID)` (adresse du compte qui gère la fiche, `""` si sans compte ou désactivé) ; test de rendu dans `internal/mail/render_test.go` (les quatre gabarits, avec et sans lien)
- [X] T012 Régénérer (`go tool task gen`) après T010 pour publier les types dans `web/app/api/openapi.json` et `web/app/api/schema.d.ts` (les opérations sont ajoutées par les stories ; relancer après chacune)

### Frontend commun

- [X] T013 [P] Étendre `web/app/lib/format.ts` (+ `format.test.ts`) : `membershipStatusLabel` (« En attente de validation », « Validée », « Refusée », « Annulée »), `membershipKindLabel` (« Première adhésion », « Renouvellement »), `paymentMethodLabel` (« Espèces », « Chèque », « Virement »), `paymentStatusLabel(payment)` (« À payer », « Paiement annoncé — {moyen en minuscules} », « Payée le {formatDay} »), `formatIban` (groupes de 4), `previousSeasonLabel` (« Oui », « Non », « Inconnu »)
- [X] T014 [P] Créer `web/app/ui/RadioGroup.tsx` (habillage de `RadioGroup` et `Radio` de `@base-ui/react`, `fieldset`/`legend`, description et erreur reliées par `aria-describedby`, options avec libellé et contenu secondaire), en extrayant le style du RadioGroup inline de `web/app/routes/app/admin/account-detail.tsx` qui l'utilise ensuite ; `web/app/ui/Switch.tsx` (habillage de `Switch` base-ui avec libellé) ; `web/app/ui/CopyButton.tsx` (bouton secondaire `size="sm"`, icône `Copy` puis `Check`, `navigator.clipboard.writeText`, annonce « Copié. » dans une région `aria-live="polite"`, libellé accessible « Copier {quoi} »)
- [X] T015 [P] Créer `web/app/memberships/MembershipBadges.tsx` : badge d'état (`warn` en attente, `ok` validée, `danger` refusée, neutre annulée) et badge de paiement (`warn` à payer, `info` annoncé, `ok` payée ou « Gratuite ») à partir des libellés de T013

**Checkpoint**: domaine testé, types publiés, composants prêts

---

## Phase 3: User Story 1 - Demander l'adhésion d'un de mes membres (Priority: P1) 🎯 MVP

**Goal**: un titulaire de compte dépose une demande (saison, membre, tarif, type déclaré) en voyant le montant correspondant

**Independent Test**: avec une saison ouverte portant des tarifs et un compte gérant deux fiches, déposer une demande pour une fiche, la retrouver en attente, constater que la même fiche ne peut plus demander pour cette saison

### Tests for User Story 1

- [X] T016 [P] [US1] Écrire `internal/api/memberships_me_test.go` (partie demande ; membre connecté via `env.LoggedIn(env.Seed(auth.RoleMember, true))`, fiches créées par `/api/me/members`, saison et tarifs insérés par sqlc ou par l'API admin) : `get-membership-offer` sans fiche → `members: []`, `canRequest` faux ; sans saison ouverte → `seasons: []` ; saison en brouillon, close, ou ouverte sans tarif → absente ; avec saison 2026-2027 et deux fiches → saison et tarifs, `availableSeasonIds` des deux fiches ; `create-my-membership` (`memberId`, `seasonId`, `feeId` « Adulte », `declaredKind: "renewal"`) → `201`, `status: "pending"`, `amountCents: 4000`, `amountFixed: false`, `payment: null` ; ensuite l'offre n'inclut plus la saison pour cette fiche ; seconde demande même fiche et saison → `409 MsgMembershipActive` ; deux demandes concurrentes (deux goroutines) → un `201` et un `409` ; saison non offerte → `422 body.seasonId` ; tarif d'une autre saison → `422 body.feeId` ; fiche d'un autre compte → `404 MsgMemberGone` ; après annulation ou refus (insérés par sqlc), nouvelle demande acceptée ; montant d'une demande en attente recalculé si le tarif est modifié (`amountCents` suit le tarif) ; journal `membership.request` sans prénom ni nom (vérifier `env.Logs`)
- [X] T017 [P] [US1] Étendre `internal/api/membership_fees_test.go` : tarif utilisé par une adhésion (même annulée) → `delete-membership-fee` `409` « Ce tarif est utilisé par au moins une adhésion : il ne peut pas être supprimé. », `list-membership-fees` → `inUse: true` ; tarif libre → `inUse: false`, suppression `204` ; et `internal/api/members_me_test.go` : `delete-my-member` sur une fiche avec adhésion `pending` ou `validated` → `409` « Ce membre a une adhésion en cours. Annulez la demande ou contactez le bureau. » ; avec adhésion `refused` ou `cancelled` seulement → `204` et adhésions supprimées

### Implementation for User Story 1

- [X] T018 [US1] Créer `internal/api/memberships_me.go` : `registerMyMemberships` (accès `auth.Member`, tag `tagMyMemberships`) avec `get-membership-offer` (`GET /me/memberships/offer`) et `create-my-membership` (`POST /me/memberships`, `DefaultStatus: 201`, corps `MyMembershipBody{MemberID int64, SeasonID int64, FeeID int64, DeclaredKind string enum:"first,renewal"}`) ; le compte vient **uniquement** de `ownAccount(ctx)` ; l'offre : saisons offertes (plus récente d'abord) avec tarifs triés (`season.SortFees`), fiches du compte (`ListMembersOfAccount`, triées par `member.SortMine`) avec `availableSeasonIds` (saisons offertes moins `ListActiveSeasonsOfMembers`), `canRequest` ; création dans `s.Store.WriteTx` : `GetMemberOfAccount` (absent → `404 MsgMemberGone`) → `loadSeasons` + `offeredSeason` (`422 body.seasonId`) → `feeOfSeason` (`422 body.feeId`) → `CreateMembership` (`status='pending'`, `declared_kind`, `requested_by` = compte, `requested_at`, `updated_at`) avec `activeConflict` ; réponse `get-my-membership` équivalente ; `logMembershipAction("membership.request", …, "me")` ; faire passer T016
- [X] T019 [US1] Règles ajoutées (R9) : dans `internal/api/membership_fees.go`, `MembershipFee` gagne `Amounts FeeAmounts \`json:"amounts"\`` (même calcul que `toFeeRef`, pour la création directe T059) et `InUse bool \`json:"inUse"\`` (rempli par `CountMembershipsOfFee` ou une jointure dans `ListMembershipFees` à ajouter dans `internal/database/queries/membership_fees.sql`, commentaires ASCII) et `deleteMembershipFee` vérifie `CountMembershipsOfFee` dans la transaction → `409 MsgFeeInUse` ; dans `internal/api/members_me.go`, `deleteMyMember` vérifie `CountActiveMembershipsOfMember` → `409 MsgMemberHasMembership` ; faire passer T017 ; `go tool task gen`
- [X] T020 [P] [US1] Créer `web/app/api/memberships.ts` : `membershipOfferQuery()`, `myMembershipListQuery(memberId?)`, `myMembershipQuery(id)`, mutations `useCreateMyMembership`, `useCancelMyMembership`, `useAnnounceMyPayment` invalidant `myMembershipListKey`, `membershipOfferKey`, `myMembershipKey(id)` et `myMemberListKey` (fichier `members.ts`) ; puis requêtes admin ajoutées par les stories suivantes
- [X] T021 [US1] Activer la rubrique : retirer `comingSoon: true` de `adhesions` dans `web/app/shell/navigation.ts`, retirer `"adhesions"` de la liste des pages « bientôt disponibles » de l'espace membre dans `web/app/routes.ts` et ajouter `route("adhesions", "routes/app/member/memberships-layout.tsx", [index("routes/app/member/memberships.tsx"), route("demande", "routes/app/member/membership-request.tsx"), route(":id", "routes/app/member/membership-detail.tsx")])`
- [X] T022 [US1] Créer `web/app/routes/app/member/memberships-layout.tsx` (fil d'Ariane « Adhésions », `Outlet`, modèle `my-members-layout.tsx`) et une première version de `web/app/routes/app/member/memberships.tsx` : `clientLoader` (`ensureQueryData` de l'offre et de la liste) ; en-tête « Adhésions », introduction « Les adhésions de vos membres à l'association. », bouton « Demander une adhésion » (icône `CreditCard`) si `canRequest` ; `Alert` `info` « Ajoutez d'abord un membre pour demander son adhésion. » (lien « Mes membres ») si aucune fiche ; « Les adhésions ne sont pas ouvertes pour le moment. » si aucune saison ; notification de retour (`location.state.flash`, `aria-live`)
- [X] T023 [US1] Créer `web/app/memberships/FeeChoice.tsx` (`RadioGroup` des tarifs : libellé, « 60,00 € · Renouvellement 40,00 € » ou « Gratuit », description ; prop `kind` : le récapitulatif `aria-live="polite"` « Montant : 40,00 € » affiche `fee.amounts[kind]` tel que renvoyé par l'API, sans aucun calcul côté client) et `web/app/routes/app/member/membership-request.tsx` : `react-hook-form` ; « Saison » (`Select` si plusieurs, texte sinon) ; « Membre » (`RadioGroup` des fiches dont `availableSeasonIds` contient la saison ; note « {prénom} a déjà une adhésion en cours pour cette saison. » pour les autres ; `?membre=` présélectionne) ; `FeeChoice` ; « Était-il adhérent la saison passée ? » (`RadioGroup` « Non, première adhésion » / « Oui, renouvellement ») ; rappel « Votre demande sera validée par le bureau ; vous recevrez ensuite les informations de paiement. » ; « Envoyer la demande » / « Annuler » ; erreurs `422` par `applyProblemToForm`, `409` en `Alert` `danger` + rechargement de l'offre ; succès → `/adhesions` avec flash « Demande envoyée. »
- [X] T024 [US1] Masquer « Supprimer » d'un tarif `inUse` dans `web/app/routes/app/admin/SeasonFees.tsx` (avec le texte « Utilisé par des adhésions » en secondaire) ; afficher le `409` de suppression de fiche en `Alert` dans `web/app/routes/app/member/my-member-detail.tsx`
- [X] T025 [US1] Écrire `web/e2e/memberships.spec.ts` (test « demande d'adhésion (006) ») : membre du jeu e2e (T074) ; page « Adhésions » et axe ; demande pour une fiche, tarif « Adulte », « Oui, renouvellement » → récapitulatif « 40,00 € » ; envoi → flash « Demande envoyée. » ; la fiche n'est plus proposée pour cette saison ; axe sur le formulaire ; parcours au clavier

**Checkpoint**: les familles déposent des demandes

---

## Phase 4: User Story 2 - Valider ou refuser les demandes (administration) (Priority: P1)

**Goal**: l'administrateur traite les demandes : validation avec type et tarif retenus (montant figé, référence), refus motivé, e-mail de décision

**Independent Test**: avec trois demandes en attente, en valider une sans changement, une en changeant tarif et type, en refuser une avec motif ; vérifier état, type retenu, montant, e-mails

### Tests for User Story 2

- [X] T026 [P] [US2] Écrire `internal/api/memberships_admin_test.go` (partie décisions ; administrateur via `env.LoggedIn(env.Seed(auth.RoleAdmin, true))`) : `list-memberships?season=all&status=pending` → demandes en attente des plus anciennes aux plus récentes, `total` ; `get-membership` → `member` (âge), `account`, `seasonFees`, `declaredKind`, `previousSeason` (`unknown` sans saison précédente ; avec saison 2025-2026 close portant une adhésion validée d'un autre membre → `no` ; du même membre → `yes`), `actions.validate` et `refuse` vrais ; `validate-membership` sans changement → `validated`, `kind = declaredKind`, `amountCents` figé (`amountFixed: true`), `payment.reference = "ADH-2627-0001"`, `payment.status = "due"` ; seconde validation → `ADH-2627-0002` ; validation avec autre tarif et autre type → montant recalculé, `declaredKind` conservé ; tarif d'une autre saison → `422 body.feeId` ; validation d'un tarif gratuit → `payment: null` ; `refuse-membership` sans motif ou motif blanc → `422 body.reason MsgReasonRequired` ; avec motif → `refused`, `reason` ; décision sur une adhésion déjà décidée → `409 MsgMembershipChanged` ; saison devenue close (`env.SetToday("2027-09-01")`) → validation et refus admis ; modifier le tarif après validation ne change pas `amountCents` ; e-mails (`env.FlushMail()`, `env.Mail`) : validation → `membership-validated` au compte de la fiche avec le lien `{BaseURL}/adhesions/{id}` et la référence ; gratuite → sans lien ; refus → motif dans le corps ; fiche sans compte ou compte désactivé → aucun e-mail ; journal `membership.validate`/`refuse` sans motif (chercher le texte du motif dans `env.Logs`) ; `get-membership-dashboard` → `pendingCount` = nombre de demandes en attente toutes saisons (2 puis 1 après une validation, 0 après le refus) ; `seasonFees[].amounts` = `{first: 6000, renewal: 4000}` pour « Adulte », `{first: 3000, renewal: 3000}` pour « Jeune »

### Implementation for User Story 2

- [X] T027 [US2] Créer `internal/api/memberships_admin.go` : `registerAdminMemberships` (accès `auth.Admin`, tag `tagMemberships`) avec `list-memberships` (`GET /admin/memberships`, paramètres `season` (`all` ou identifiant ; absent = saison courante, sinon plus récente ouverte, sinon plus récente), `status` (`enum`), `payment` (`enum:"due,announced,paid,free"`), `member`, `q` (`maxLength:"100"` ; `FTSTrigramQuery` puis paramètre `query` de `ListMemberships`, qui cherche dans la référence et dans la fiche ; aucun fragment → `422 query.q MsgSearchTooShort`), `page` (défaut 1), `pageSize` (1 à 100, défaut 25) ; tri en Go : `pending` d'abord par `requested_at` croissant, puis `textfold.Fold(last_name)`, `Fold(first_name)`, id ; réponse avec `seasonId`), `get-membership-dashboard` (`GET /admin/memberships/dashboard` : `pendingCount` par `CountPendingMemberships`, `paymentSetupNeeded` = aucun moyen activé dans `GetPaymentSettings` et au moins une saison offerte), `get-membership` (`GET /admin/memberships/{id}`, `previousSeason` par `PreviousSeasonHint` : saison précédente = celle dont `ends_on` = veille de `starts_on`, `CountValidatedInSeason`, `HasValidatedMembership`), `validate-membership` (`POST /admin/memberships/{id}/validate`, `DecisionBody{FeeID int64, Kind string enum:"first,renewal"}`) et `refuse-membership` (`POST …/refuse`, `ReasonBody{Reason string maxLength:"500"}`) ; écritures dans `WriteTx` : `GetMembership` (absent → `404`) → `ActionsFor` (refus → `409 MsgMembershipChanged`) → `feeOfSeason` → `AppliedAmount` → si montant > 0 `assignReference` et `payment_status='due'` → `ValidateMembership` (`:execrows` = 0 → `409`) → `enqueueNotice("membership-validated", …, "/adhesions/{id}" si montant > 0)` si destinataire ; refus : `CleanReason` → `RefuseMembership` → `enqueueNotice("membership-refused")` ; `logMembershipAction` ; faire passer T026 ; `go tool task gen`
- [X] T028 [P] [US2] Ajouter à `web/app/api/memberships.ts` : `membershipPageQuery(params)`, `membershipQuery(id)`, `membershipDashboardQuery()`, mutations `useValidateMembership`, `useRefuseMembership` invalidant `membershipPageKey`, `membershipKey(id)`, `membershipDashboardKey`, `membershipStatsKey`
- [X] T029 [US2] Activer la rubrique d'administration : retirer `comingSoon: true` de `gestion-adhesions` dans `web/app/shell/navigation.ts`, l'ôter de la liste « bientôt disponibles » de `web/app/routes.ts` et ajouter `route("admin/adhesions", "routes/app/admin/memberships-layout.tsx", [index("routes/app/admin/memberships.tsx"), route("nouvelle", "routes/app/admin/membership-new.tsx"), route(":id", "routes/app/admin/membership-detail.tsx")])` (la route `nouvelle` pointe sur un fichier créé en US6 : l'ajouter à ce moment si l'on livre US2 seule)
- [X] T030 [US2] Créer `web/app/routes/app/admin/memberships-layout.tsx` et une première version de `web/app/routes/app/admin/memberships.tsx` : si `membershipDashboardQuery().pendingCount > 0`, texte « {n} demande(s) en attente » avec le lien « Afficher les demandes en attente » (`?season=all&status=pending`), quels que soient les filtres (US2 §1) ; filtres dans l'URL (`Select` « Saison » alimenté par `seasonListQuery()` + « Toutes », `Select` « État », recherche « Nom ou référence » avec l'indication des 3 caractères comme `members.tsx`) ; tableau `ui/Table` (`<caption>` « Adhésions » ; Membre en `th scope="row"` lien vers `/admin/adhesions/{id}` · Saison si « Toutes » · Tarif · Type (retenu, sinon déclaré + « déclaré ») · Montant · État (`MembershipBadges`)) dans `ScrollRegion` ; pagination et total ; aucun résultat : message et « Effacer la recherche »
- [X] T031 [US2] Créer `web/app/routes/app/admin/membership-detail.tsx` : liste de définitions (membre avec lien `/admin/membres/{id}`, âge, compte de rattachement avec lien ou « Sans compte », saison, tarif, type déclaré, type retenu, montant avec « calculé avec le tarif actuel » si `!amountFixed`, date de la demande et auteur, décision et auteur, motif) ; aide « Adhérent la saison précédente dans Frigg : Oui / Non / Inconnu » avec explication de « Inconnu » (« la saison précédente n'est pas gérée dans Frigg ») ; bouton « Valider » → dialogue (`Dialog` base-ui) avec `FeeChoice` sur `seasonFees` (présélection du tarif choisi), `RadioGroup` du type (présélection du type déclaré), montant `fee.amounts[kind]` lu dans `seasonFees`, « Valider l'adhésion » ; bouton « Refuser » → dialogue avec `textarea` « Motif » (requis, `maxLength` 500), « Refuser la demande » ; `409` → `Alert` `danger` + rechargement ; succès → notification « Adhésion validée. » / « Demande refusée. »
- [X] T032 [US2] Écrire `web/e2e/admin-memberships.spec.ts` (test « décisions (006) ») : demande créée par l'API avec le compte membre ; en administrateur, liste filtrée « En attente », ouverture, aide « Inconnu », validation avec changement de type → montant affiché, référence ; refus d'une seconde demande sans motif (erreur) puis avec motif ; Mailpit : e-mail de validation avec lien `/adhesions/`, e-mail de refus avec le motif ; axe sur la liste, le détail et les deux dialogues

**Checkpoint**: les demandes aboutissent (MVP fonctionnel avec US1 et US3)

---

## Phase 5: User Story 3 - Suivre et annuler mes demandes (Priority: P1)

**Goal**: le compte voit les adhésions de ses fiches par saison, annule une demande en attente ; le cloisonnement est démontré

**Independent Test**: avec des adhésions dans chaque état, vérifier leur affichage, annuler une demande en attente, constater qu'une adhésion validée n'est pas annulable ; un autre compte reçoit `404`

### Tests for User Story 3

- [X] T033 [P] [US3] Compléter `internal/api/memberships_me_test.go` : `list-my-memberships` → sections par saison (la plus récente d'abord), adhésions triées par prénom replié, chaque état avec `reason` pour refusée/annulée, `actions.cancel` vrai seulement pour `pending` ; filtre `?member=` (fiche d'un autre compte → `404 MsgMemberGone`) ; `get-my-membership` → `200` ; `cancel-my-membership` sur `pending` → `cancelled`, `cancelledAt` ; sur `validated` ou `refused` → `409 MsgMembershipChanged` ; journal `membership.cancel` avec `space=me`
- [X] T034 [P] [US3] Écrire `internal/api/memberships_isolation_test.go` : comptes A et B (membres) ; adhésions d'une fiche de A, d'une fiche sans compte et identifiant inexistant ; pour `get-my-membership`, `cancel-my-membership`, `announce-my-payment` (`{method:"transfer"}`) avec la session de B → `404 MsgMembershipGone` et adhésion inchangée (relue par sqlc) ; `create-my-membership` avec une fiche de A → `404 MsgMemberGone` ; `list-my-memberships` de B ne contient aucune adhésion de A ; un administrateur dans l'espace membre ne voit que les adhésions de ses propres fiches

### Implementation for User Story 3

- [X] T035 [US3] Ajouter dans `internal/api/memberships_me.go` : `list-my-memberships` (`GET /me/memberships`, paramètre `member` facultatif vérifié par `GetMemberOfAccount`), `get-my-membership` (`GET /me/memberships/{id}`, `GetMembershipOfAccount`, `404 MsgMembershipGone`), `cancel-my-membership` (`POST /me/memberships/{id}/cancel` : `GetMembershipOfAccount` → `ActionsFor` → `CancelMembershipOfAccount` (`:execrows` = 0 → `409`) → réponse relue) ; `MyMembership.actions` par `membership.ActionsFor` ; `paymentMethods` vide à ce stade (rempli en US4) ; faire passer T033 et T034 ; `go tool task gen`
- [X] T036 [US3] Compléter `web/app/routes/app/member/memberships.tsx` : une section par saison (`h2` « Saison {libellé} »), cartes `<ul>` : « {prénom} {nom} » (lien `/adhesions/{id}`), tarif, type, montant, `MembershipBadges`, motif d'un refus ou d'une annulation ; une seule colonne à 320 px
- [X] T037 [US3] Créer `web/app/routes/app/member/membership-detail.tsx` : `clientLoader` `ensureQueryData(myMembershipQuery(id))`, `404` → page « Page introuvable » (mécanisme de `my-member-detail.tsx`) ; liste de définitions (membre, saison, tarif, type déclaré puis « {type} (retenu par le bureau) » s'il diffère, montant, état, date, motif) ; `actions.cancel` → bouton « Annuler la demande » + `ConfirmDialog` « Annuler la demande d'adhésion de {prénom} ? » ; succès → `/adhesions` avec flash « Demande annulée. »
- [X] T038 [US3] Ajouter la section « Adhésions » à `web/app/routes/app/member/my-member-detail.tsx` (`myMembershipListQuery(id)`) : adhésions de la fiche par saison avec badges et lien ; pour chaque saison de l'offre où la fiche est disponible, « Non adhérent » + lien « Demander l'adhésion » (`/adhesions/demande?membre={id}`)
- [X] T039 [US3] Compléter `web/e2e/memberships.spec.ts` (test « suivi (006) ») : liste par saison avec une demande en attente et une adhésion validée (créée par l'API admin) ; annulation de la demande avec dialogue → « Annulée » ; aucune action d'annulation sur la validée ; section « Adhésions » de la fiche ; un second compte ouvrant `/adhesions/{id}` → « Page introuvable » ; axe

**Checkpoint**: MVP (US1 + US2 + US3) : demande, décision, suivi, cloisonnement démontré

---

## Phase 6: User Story 4 - Payer une adhésion validée (Priority: P1)

**Goal**: le compte choisit un moyen de paiement, voit ses informations et la référence ; l'administrateur constate le paiement

**Independent Test**: avec les trois moyens paramétrés (insérés par sqlc si US5 n'est pas faite), valider une demande, ouvrir la page de paiement, choisir chaque moyen, confirmer le virement, puis constater le paiement côté administration

### Tests for User Story 4

- [X] T040 [P] [US4] Compléter `internal/api/memberships_me_test.go` (paramètres insérés par `UpdatePaymentSettings`) : adhésion validée à payer → `get-my-membership` renvoie `payment` (`reference`, `status: "due"`), `actions.pay` vrai, `paymentMethods` = moyens activés uniquement avec leurs champs (`cash.instructions` ; `cheque.payee/address/instructions` ; `transfer.holder/iban/bic/instructions`) ; adhésion en attente, refusée, payée ou gratuite → `paymentMethods: []`, `actions.pay` faux ; `announce-my-payment` `transfer` → `announced`, `announcedMethod: "transfer"` ; puis `cheque` → remplacé ; moyen désactivé → `422 body.method MsgMethodUnavailable` ; adhésion en attente ou payée → `409` ; journal `membership.payment_announce` sans IBAN (chercher l'IBAN dans `env.Logs`)
- [X] T041 [P] [US4] Compléter `internal/api/memberships_admin_test.go` : `record-membership-payment` (`method`, `receivedOn`, `note`) sur `due` et sur `announced` → `paid`, `paidMethod`, `paidOn`, `paidBy`, `paymentNote` ; `receivedOn` futur → `422 body.receivedOn MsgReceivedFuture` ; antérieur à `decidedAt` (date à Paris) → `422 MsgReceivedBeforeValidation` ; `note` de 201 caractères → `422` ; déjà payée → `409` ; saison close → admis ; `cancel-membership-payment` → retour à `announced` (moyen annoncé conservé) ou à `due` ; e-mail `membership-paid` au compte ; journal `membership.payment_record` / `payment_cancel` sans commentaire

### Implementation for User Story 4

- [X] T042 [US4] Ajouter dans `internal/api/memberships_me.go` : `announce-my-payment` (`PUT /me/memberships/{id}/payment`, `AnnounceBody{Method string enum:"cash,cheque,transfer"}` : `GetMembershipOfAccount` → `ActionsFor` (`Pay`) → `GetPaymentSettings` + `EnabledMethods` (moyen absent → `422 body.method`) → `AnnouncePaymentOfAccount` (`:execrows` = 0 → `409`)) ; remplir `paymentMethods` de `MyMembership` (seulement si `actions.pay`, seulement les moyens activés, IBAN normalisé) ; faire passer T040
- [X] T043 [US4] Ajouter dans `internal/api/memberships_admin.go` : `record-membership-payment` (`POST /admin/memberships/{id}/payment`, `RecordPaymentBody{Method string enum:"cash,cheque,transfer", ReceivedOn string format:"date", Note string maxLength:"200" omitempty}` : `receivedOn` ≤ `s.today()` et ≥ date de `decided_at` convertie à Paris ; `RecordMembershipPayment` ; `enqueueNotice("membership-paid")`) et `cancel-membership-payment` (`DELETE /admin/memberships/{id}/payment`, `CancelMembershipPayment`) ; `AdminMembership` expose `paidBy`, `paymentNote` et les actions `recordPayment`, `cancelPayment` ; faire passer T041 ; `go tool task gen`
- [X] T044 [P] [US4] Créer `web/app/memberships/PaymentMethodInfo.tsx` : rendu des informations d'un moyen (Espèces : instructions ; Chèque : « À l'ordre de {ordre} », adresse d'envoi, instructions, « Inscrivez la référence {réf.} au dos du chèque. » ; Virement : titulaire, IBAN via `formatIban`, BIC, instructions, « Indiquez la référence {réf.} dans le libellé du virement. ») ; textes multilignes en texte brut avec la classe `whitespace-pre-line` (jamais `dangerouslySetInnerHTML`) ; `CopyButton` à côté de l'IBAN (copie sans espaces), du BIC et de la référence
- [X] T045 [US4] Ajouter la section « Paiement » (`h2`) à `web/app/routes/app/member/membership-detail.tsx`, placée en premier quand `actions.pay` : montant et référence en évidence ; `RadioGroup` « Comment souhaitez-vous payer ? » des `paymentMethods` (présélection du moyen annoncé) ; `PaymentMethodInfo` du moyen sélectionné ; bouton « Confirmer ce moyen de paiement » (« Changer de moyen » si déjà annoncé) → `useAnnounceMyPayment`, notification « Moyen de paiement enregistré. » ; moyen annoncé absent des moyens activés → `Alert` `warn` « Ce moyen n'est plus proposé. Choisissez-en un autre. » ; aucun moyen → `Alert` `info` « Contactez le bureau pour régler cette adhésion. » ; payée → « Payée le {date} ({moyen}). » ; gratuite → « Adhésion gratuite : aucun paiement n'est demandé. » ; bouton « Payer » sur les cartes de `memberships.tsx` si `actions.pay` (lien vers `/adhesions/{id}`)
- [X] T046 [US4] Ajouter à `web/app/routes/app/admin/membership-detail.tsx` : bloc « Paiement » (référence, état, moyen annoncé, moyen constaté, date, auteur, commentaire) ; bouton « Marquer comme payée » → dialogue (moyen `RadioGroup` présélectionné sur le moyen annoncé, « Date de réception » `type="date"` défaut aujourd'hui, « Commentaire » 200) → notification « Paiement enregistré. » ; bouton « Annuler la constatation » → `ConfirmDialog` ; mutations ajoutées à `web/app/api/memberships.ts`
- [X] T047 [US4] Compléter `web/e2e/memberships.spec.ts` (test « paiement (006) ») : adhésion validée par l'API ; ouverture déconnectée de `/adhesions/{id}` → connexion → retour sur la page ; choix de chaque moyen et vérification des informations ; « Copier » annonce « Copié. » ; confirmation du virement → badge « Paiement annoncé — virement » ; en administrateur, « Marquer comme payée » → « Payée le … » ; Mailpit : e-mail « Paiement … reçu » ; axe sur la section paiement et le dialogue

**Checkpoint**: le cycle complet demande → décision → paiement fonctionne

---

## Phase 7: User Story 5 - Paramétrer les moyens de paiement (administration) (Priority: P1)

**Goal**: l'administrateur active et renseigne espèces, chèque et virement dans « Paramètres »

**Independent Test**: activer et renseigner chaque moyen, constater les informations sur une page de paiement, désactiver un moyen et constater sa disparition

### Tests for User Story 5

- [X] T048 [P] [US5] Écrire `internal/api/settings_test.go` : `get-payment-settings` initial → trois moyens désactivés, champs nuls ; `update-payment-settings` avec virement activé sans titulaire → `422 body.transfer.holder` « Indiquez le titulaire du compte. » ; sans IBAN → `422 body.transfer.iban` « Saisissez l'IBAN. » ; IBAN à clé fausse → « Cet IBAN n'est pas valide. Vérifiez-le sur votre relevé d'identité bancaire. » ; BIC `BNPAFRP` → `422 body.transfer.bic` « Saisissez un BIC de 8 ou 11 caractères. » ; chèque activé sans ordre → « Indiquez l'ordre des chèques. » ; espèces activées sans instructions → « Indiquez où et quand régler en espèces. » ; erreurs cumulées dans une seule réponse ; enregistrement valide → IBAN normalisé, retours à la ligne des instructions conservés ; moyen désactivé → informations conservées ; journal `settings.payment_update` avec la liste des moyens activés et sans IBAN ni texte ; `get-membership-dashboard` → `paymentSetupNeeded` vrai si aucun moyen activé et une saison ouverte avec tarif, faux sinon

### Implementation for User Story 5

- [X] T049 [US5] Créer `internal/api/settings.go` : `registerSettings` (accès `auth.Admin`, tag `tagSettings`) avec `get-payment-settings` (`GET /admin/settings/payment`) et `update-payment-settings` (`PUT /admin/settings/payment`, corps `PaymentSettingsBody{Cash{Enabled bool, Instructions string maxLength:"1000"}, Cheque{Enabled bool, Payee string maxLength:"120", Address string maxLength:"300", Instructions string maxLength:"1000"}, Transfer{Enabled bool, Holder string maxLength:"120", IBAN string json:"iban" maxLength:"50", BIC string json:"bic" maxLength:"20", Instructions string maxLength:"1000"}}`) ; validation par `membership.ValidatePaymentSettings`, erreurs converties en `422` `body.{moyen}.{champ}` avec les messages du contrat ; `UpdatePaymentSettings` dans `WriteTx` ; journal `settings.payment_update` (`actor_id`, `enabled` = moyens activés) ; faire passer T048 ; `go tool task gen`
- [X] T050 [US5] Vérifier que `get-membership-dashboard` (livré en T027) suit les paramètres : `paymentSetupNeeded` passe à faux dès qu'un moyen est activé par `update-payment-settings` et redevient vrai quand tous sont désactivés (partie tableau de bord de T048) ; corriger `internal/api/memberships_admin.go` si besoin
- [X] T051 [P] [US5] Créer `web/app/api/settings.ts` (`paymentSettingsQuery()`, `useUpdatePaymentSettings` invalidant `paymentSettingsKey` et `membershipDashboardKey`) ; ajouter l'entrée de navigation `{ id: "parametres", label: "Paramètres", icon: Settings, path: "/admin/parametres", space: "admin", section: "settings" }` entre `comptes` et `aide-admin` dans `web/app/shell/navigation.ts` (mettre à jour `navigation.test.ts`) et la route `route("admin/parametres", "routes/app/admin/settings.tsx")` dans `web/app/routes.ts`
- [X] T052 [US5] Créer `web/app/routes/app/admin/settings.tsx` : en-tête « Paramètres », introduction « Réglages de l'association. » ; section « Moyens de paiement » (`h2`) avec « Les informations des moyens activés sont présentées aux adhérents au moment de payer. » ; un `fieldset` par moyen (`legend` « Espèces », « Chèque », « Virement ») avec `Switch` « Proposer ce moyen » et les champs du contrat (`textarea` pour instructions et adresse ; IBAN avec `autocomplete="off"` et `spellcheck={false}`), `required` seulement si le moyen est activé ; « Enregistrer » → notification « Paramètres enregistrés. » ; erreurs `422` sur les champs (`body.transfer.iban` → champ `transfer.iban` du formulaire via `applyProblemToForm`)
- [X] T053 [US5] Ajouter à `web/app/routes/app/admin/home.tsx` : `clientLoader` précharge `membershipDashboardQuery()` (erreur ignorée comme pour les saisons) ; `Alert` `info` « {n} demande(s) d'adhésion en attente. » + lien « Traiter les demandes » (`/admin/adhesions?season=all&status=pending`) si `pendingCount > 0` ; `Alert` `warn` « Aucun moyen de paiement n'est proposé aux adhérents. » + lien « Paramétrer » (`/admin/parametres`) si `paymentSetupNeeded`
- [X] T054 [US5] Écrire `web/e2e/admin-settings.spec.ts` : en administrateur sur une base où aucun moyen n'est activé (réinitialiser par l'API au début du test) : encart du tableau de bord ; « Paramètres » : activer le virement sans IBAN → erreur sur le champ ; IBAN `FR76 3000 6000 0112 3456 7890 189` → enregistré ; activer le chèque avec un ordre ; l'encart disparaît ; désactiver le chèque → il n'apparaît plus sur une page de paiement du membre ; axe sur la page ; parcours au clavier des interrupteurs

**Checkpoint**: toutes les stories P1 sont livrées

---

## Phase 8: User Story 6 - Enregistrer une adhésion directement (administration) (Priority: P2)

**Goal**: l'administrateur crée une adhésion validée pour toute fiche, avec règlement déjà reçu facultatif ; la fiche d'administration montre les adhésions

**Independent Test**: créer une adhésion pour une fiche sans compte et pour une fiche rattachée ; elles sont validées ; la seconde apparaît chez le compte, avec l'e-mail de validation

### Tests for User Story 6

- [X] T055 [P] [US6] Compléter `internal/api/memberships_admin_test.go` : `create-membership` (`memberId`, `seasonId`, `feeId`, `kind`) → `201`, `validated`, `declaredKind: null`, `createdBy` = administrateur, référence, `due` ; avec `paid` → `paid` directement ; `paid.receivedOn` antérieur à aujourd'hui mais dans la saison (2026-10-20) → accepté ; avant le début de la saison (2026-08-31) → `422 body.paid.receivedOn` « La date de réception ne peut pas précéder le début de la saison. » ; futur → `422 body.paid.receivedOn MsgReceivedFuture` ; fiche avec adhésion en cours → `409 MsgMembershipActive` ; saison close → `422 body.seasonId` ; fiche inexistante → `404` ; fiche sans compte → aucun e-mail ; fiche rattachée → e-mail `membership-validated` (lien si `due`, aucun lien si payée ni gratuite) ; `get-member` (005) expose `memberships` ; `delete-member` supprime les adhésions et journalise `memberships_deleted` ; `delete-account` avec `members=delete` supprime aussi les adhésions, avec `members=keep` les conserve (`requestedBy` nul si le compte était le demandeur)

### Implementation for User Story 6

- [X] T056 [US6] Ajouter `create-membership` (`POST /admin/memberships`, `DefaultStatus: 201`, `AdminCreateBody{MemberID, SeasonID, FeeID int64, Kind string enum:"first,renewal", Paid *RecordPaymentBody}`) dans `internal/api/memberships_admin.go` : `GetMember` (absent → `404 MsgMemberGone`) → `offeredSeason` → `feeOfSeason` → `AppliedAmount` → référence si montant > 0 → `CreateMembership` (`status='validated'`, `created_by`, `decided_by`, `decided_at`, `requested_at` = maintenant) avec `activeConflict` → si `Paid` et montant > 0 : `receivedOn` ≤ `s.today()` et ≥ date de début de la saison (règle de la création directe, FR-022d ; **pas** la borne « date de validation » de T043, qui interdirait un chèque reçu avant la saisie du bulletin), message `MsgReceivedBeforeSeason = "La date de réception ne peut pas précéder le début de la saison."`, moyen et commentaire comme T043, puis `RecordMembershipPayment` → e-mail → journaux `membership.create` (+ `membership.payment_record`) ; faire passer la partie création de T055
- [X] T057 [US6] Dans `internal/api/members_admin.go` : `AdminMember` gagne `Memberships []MemberMembership{ID, Season SeasonRef, Status, Payment *Payment}` (`ListMembershipsOfMember` + libellés de saison) ; `deleteMember` compte les adhésions avant suppression et ajoute `memberships_deleted` au journal `member.delete` ; vérifier que `delete-account` (`members=delete`) supprime les adhésions par cascade (aucun code si la cascade suffit) ; faire passer T055 ; `go tool task gen`
- [X] T058 [P] [US6] Créer `web/app/memberships/MemberPicker.tsx` : Combobox base-ui sur `list-members?q=` (3 caractères au moins, indication du minimum), options « {nom} {prénom} — {âge} ans — {adresse du compte ou Sans compte} », sur le modèle de `web/app/members/AccountPicker.tsx` ; prop `initialMember` pour la présélection
- [X] T059 [US6] Créer `web/app/routes/app/admin/membership-new.tsx` : `MemberPicker` (présélection `?membre=` via `get-member`), « Saison » (`Select` des saisons ouvertes non closes avec tarif, à partir de `seasonListQuery()` et `membershipFeeListQuery`), `FeeChoice` (montants `amounts` de `list-membership-fees`), `RadioGroup` du type ; case « Règlement déjà reçu » révélant « Moyen » (`RadioGroup`), « Date de réception » (`type="date"`, défaut aujourd'hui), « Commentaire » ; erreurs par `applyProblemToForm` (`body.paid.receivedOn` → champ `paid.receivedOn`) ; `409` → `Alert` avec lien vers l'adhésion existante (trouvée par `list-memberships?member=&season=`) ; succès → `/admin/adhesions/{id}` avec notification « Adhésion enregistrée. » ; bouton « Créer une adhésion » dans l'en-tête de `web/app/routes/app/admin/memberships.tsx`
- [X] T060 [US6] Dans `web/app/routes/app/admin/member-detail.tsx` : section « Adhésions » (`memberships` de `get-member` : saison, badges, lien `/admin/adhesions/{id}`) et bouton « Créer une adhésion » (`/admin/adhesions/nouvelle?membre={id}`) ; dialogue de suppression : « Ses {n} adhésion(s) seront aussi supprimées : » + liste (saison, état) ; dans `web/app/routes/app/admin/account-detail.tsx`, libellé de l'option « Supprimer aussi ces fiches et leurs adhésions »
- [X] T061 [US6] Compléter `web/e2e/admin-memberships.spec.ts` (test « création directe (006) ») : depuis la fiche d'une fiche sans compte, « Créer une adhésion » (membre présélectionné), « Règlement déjà reçu » chèque → détail « Payée » ; la fiche liste l'adhésion ; dialogue de suppression d'une fiche avec adhésion : texte des adhésions ; axe sur le formulaire

---

## Phase 9: User Story 7 - Annuler ou corriger une adhésion validée (administration) (Priority: P2)

**Goal**: l'administrateur corrige tarif et type d'une adhésion validée non payée ou l'annule avec motif

**Independent Test**: corriger le tarif d'une adhésion validée (montant recalculé) ; annuler une adhésion validée ; le membre peut de nouveau adhérer pour la saison

### Tests for User Story 7

- [X] T062 [P] [US7] Compléter `internal/api/memberships_admin_test.go` : `update-membership` (`feeId`, `kind`) sur validée `due` → montant recalculé avec le tarif actuel ; sur validée `announced` → moyen annoncé conservé ; de payant à gratuit → `payment: null`, référence conservée en base ; de gratuit à payant → référence attribuée, `due` ; sur payée → `409 MsgPaidCorrection` ; saison close → `409 MsgSeasonClosedAction` ; `cancel-membership` sans motif → `422` ; avec motif → `cancelled`, `cancelledBy`, `reason`, e-mail `membership-cancelled` ; saison close → `409` ; après annulation, `create-my-membership` pour la même fiche et la même saison → `201` ; les chiffres (T064) n'incluent plus l'adhésion annulée

### Implementation for User Story 7

- [X] T063 [US7] Ajouter dans `internal/api/memberships_admin.go` : `update-membership` (`PUT /admin/memberships/{id}`, `DecisionBody` : `ActionsFor` (`Update` ; payée → `409 MsgPaidCorrection` ; saison close → `409 MsgSeasonClosedAction`) → `feeOfSeason` → `AppliedAmount` → `PaymentAfterAmount` (référence par `assignReference` si besoin) → `UpdateValidatedMembership` → journal `membership.update`) et `cancel-membership` (`POST /admin/memberships/{id}/cancel`, `ReasonBody` : `CleanReason` → `ActionsFor` (`Cancel`) → `CancelMembership` → `enqueueNotice("membership-cancelled")` → journal `membership.cancel` avec `space=admin`) ; faire passer T062 ; `go tool task gen`
- [X] T064 [US7] Ajouter à `web/app/routes/app/admin/membership-detail.tsx` : bouton « Corriger » (même dialogue que « Valider », intitulé « Corriger l'adhésion ») ; bouton « Annuler l'adhésion » → `ConfirmDialog` `danger` avec `textarea` « Motif » obligatoire, et si payée le texte « Un éventuel remboursement se fait en dehors de Frigg. » ; adhésion d'une saison close : aucune action hormis celles du paiement ; mutations dans `web/app/api/memberships.ts`
- [X] T065 [US7] Compléter `web/e2e/admin-memberships.spec.ts` (test « correction et annulation (006) ») : correction du tarif d'une adhésion due → nouveau montant ; annulation avec motif → « Annulée » ; le membre voit le motif ; axe sur les dialogues

---

## Phase 10: User Story 8 - Suivre les chiffres de l'adhésion (administration) (Priority: P3)

**Goal**: chiffres de la saison (adhésions, premières, renouvellements, tarifs, encaissé, restant), filtres paiement et recherche par référence

**Independent Test**: avec des adhésions de chaque type et état sur une saison, vérifier que les compteurs ne prennent en compte que les validées et correspondent aux données

### Tests for User Story 8

- [X] T066 [P] [US8] Compléter `internal/api/memberships_admin_test.go` : `get-membership-stats?season=` sur un jeu mixte (2 premières adhésions dont une payée par chèque, 1 renouvellement annoncé par virement, 1 gratuite, 1 en attente, 1 refusée, 1 annulée payée) → `validated` 4, `first` 3, `renewal` 1, `pending` 1, `byFee` exact, `totalCents`, `paidCents`, `dueCents`, `dueByMethod` ; saison inexistante → `404` ; `list-memberships?payment=due|announced|paid|free` ; `q=adh-2627-000` → adhésions dont la référence contient ce fragment ; `q=0002` → l'adhésion `ADH-2627-0002` ; une référence modifiée n'est jamais attribuée deux fois et une adhésion supprimée (avec sa fiche) n'est plus trouvée (synchronisation de `memberships_fts`) ; `q=mar` → fiches correspondantes ; `q=ab` → `422 query.q` ; sans `season` → saison courante, `seasonId` renvoyé

### Implementation for User Story 8

- [X] T067 [US8] Ajouter `get-membership-stats` (`GET /admin/memberships/stats`, `season` obligatoire, `ListSeasonMembershipsForStats` + `membership.ComputeStats`, saison absente → `404`) dans `internal/api/memberships_admin.go` ; vérifier le filtre `payment` et la recherche par référence de `list-memberships` (T027) ; faire passer T066 ; `go tool task gen`
- [X] T068 [US8] Compléter `web/app/routes/app/admin/memberships.tsx` : `Select` « Paiement » (« Tous », « À payer », « Paiement annoncé », « Payée », « Gratuite ») dans l'URL ; colonne « Paiement » (badge + référence) ; bloc « Chiffres de la saison » (si une saison est filtrée) en liste de définitions en grille : adhésions validées (dont premières adhésions, renouvellements), demandes en attente, total attendu, encaissé, restant à percevoir avec détail « non choisi / espèces / chèque / virement » ; tableau « Répartition par tarif » (`<caption>`) ; `membershipStatsQuery(season)` dans `web/app/api/memberships.ts`
- [X] T069 [US8] Compléter `web/e2e/admin-memberships.spec.ts` (test « chiffres (006) ») : chiffres affichés cohérents avec les adhésions créées dans le test ; filtre « À payer » ; recherche par référence ; axe

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: vérifications transverses et données de test

- [X] T070 [P] Vérifier que `internal/api/access_test.go` couvre automatiquement les 19 nouvelles opérations (anonyme → 401 ; membre → 403 sur les 13 opérations `admin` ; administrateur accepté partout) et l'ajuster si la matrice a besoin de valeurs de chemin ou de corps valides pour les nouvelles routes
- [X] T071 [P] Ajouter un test de journalisation transverse dans `internal/api/memberships_admin_test.go` : après un parcours complet (demande, validation, annonce, constatation, refus, annulation, paramètres), `env.Logs` ne contient ni prénom, ni nom, ni motif, ni commentaire, ni IBAN, ni BIC, ni instructions
- [X] T072 [P] Relire les textes des quatre e-mails et des encarts (orthographe, accords « {n} demande(s) », montants produits par `formatEuros` (T010) identiques à ceux de `formatAmount` du frontend : « 40,00 € »)
- [X] T073 Mettre à jour `web/e2e/navigation.spec.ts` et `web/app/shell/navigation.test.ts` : « Adhésions » n'est plus « bientôt disponible » dans les deux espaces ; « Paramètres » présent dans la section Configuration de l'administration, absent pour un membre
- [X] T074 Étendre `internal/tools/e2eseed/main.go` (R13) : saison ouverte contenant la date du jour avec les tarifs « Adulte » (6000, renouvellement 4000), « Jeune » (3000), « Bienfaiteur » (0) ; fiches pour le compte membre ; une fiche sans compte ; paramètres de paiement avec les trois moyens activés (IBAN `FR7630006000011234567890189`, BIC `BNPAFRPP`) ; un compte membre dédié par test e2e d'adhésions (limite de connexions par adresse)
- [X] T075 Exécuter `go tool task check` (build `CGO_ENABLED=0`, `go vet`, `go test`, génération sans diff, migrations sur base vierge, typecheck, Biome, Vitest, build, Playwright + axe) et corriger jusqu'au vert
- [ ] T076 Dérouler les parcours manuels 1 à 11 de `specs/006-memberships/quickstart.md` et cocher les points vérifiés

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : T001 puis T002 et T003 (sqlc lit la migration)
- **Foundational (Phase 2)** : après la phase 1 ; T004 → T005 ; T006, T007, T008, T009 en parallèle de T004/T005 (fichiers distincts du même package) ; T010 après T005 à T009 ; T011 en parallèle ; T012 après T010 ; T013, T014, T015 en parallèle (frontend)
- **US1 (Phase 3)** : après la phase 2
- **US2 (Phase 4)** : après la phase 2 ; ses tests insèrent les demandes par sqlc ou par l'API de US1 (si US1 n'est pas faite, insérer par sqlc)
- **US3 (Phase 5)** : après T018 (même fichier `memberships_me.go`) ; T034 (cloisonnement) est la condition de mise en service de l'espace membre
- **US4 (Phase 6)** : après US2 (adhésions validées) et T035 (`get-my-membership`) ; les paramètres sont insérés par sqlc tant que US5 n'est pas faite
- **US5 (Phase 7)** : après la phase 2 ; indépendante des autres côté API ; T050 et T053 utilisent `get-membership-dashboard`, livré en T027 (US2)
- **US6 (Phase 8)** : après US2 (même fichier `memberships_admin.go`, dialogue de décision et `FeeChoice`)
- **US7 (Phase 9)** : après US2 et T046 (bloc paiement du détail)
- **US8 (Phase 10)** : après US2 (liste) ; T066 utilise des adhésions payées : après US4 côté API
- **Polish (Phase 11)** : après toutes les stories ; T074 peut être fait dès la phase 2 (requis par les e2e T025 et suivants)

### User Story Dependencies

- **US1** : aucune autre story
- **US2** : aucune (données insérées par sqlc), naturellement après US1
- **US3** : étend l'espace membre de US1
- **US4** : après US2 et US3 ; utilise les paramètres (US5 ou insertion directe)
- **US5** : aucune autre story
- **US6, US7, US8** : étendent l'administration de US2

### Within Each User Story

- Tests d'API écrits d'abord et en échec, puis implémentation jusqu'au vert
- API avant écrans ; régénération des types (`go tool task gen`) après chaque changement d'opération
- e2e en dernier

### Parallel Opportunities

- Phase 2 : T004, T006, T007, T008, T009 (domaine), T011 (e-mails), T013, T014, T015 (frontend)
- US1 et US5 peuvent avancer en parallèle après la phase 2 (`memberships_me.go` / `settings.go`)
- US2 en parallèle de US1 côté API (fichiers `memberships_admin.go` / `memberships_me.go`) ; la seule écriture partagée est `web/app/api/memberships.ts` (sections distinctes)
- Dans chaque story : les tests [P] en parallèle de la préparation des écrans (T020, T028, T044, T051, T058)
- Polish : T070, T071, T072

---

## Parallel Example: User Story 1

```text
# Après la phase 2 :
Task: "T016 [US1] Tests d'API de la demande dans internal/api/memberships_me_test.go"
Task: "T017 [US1] Tests des règles ajoutées dans internal/api/membership_fees_test.go et members_me_test.go"
Task: "T020 [US1] Requêtes frontend dans web/app/api/memberships.ts"
```

## Parallel Example: User Stories 1, 2 et 5

```text
# Trois développeurs après la phase 2 :
Dev A: T016 → T018 → T019 → T021 → T022 → T023 → T024 → T025   (demande, espace membre)
Dev B: T026 → T027 → T028 → T029 → T030 → T031 → T032          (décisions, administration)
Dev C: T048 → T049 → T050 → T051 → T052 → T053 → T054          (paramètres)
```

---

## Implementation Strategy

### MVP First (User Stories 1 à 3)

1. Phase 1 et phase 2 (avec T074 pour les e2e)
2. Phase 3 (US1) : demandes
3. Phase 4 (US2) : décisions et e-mails
4. Phase 5 (US3) : suivi, annulation, **cloisonnement démontré** (condition de mise en service)
5. **STOP and VALIDATE** : parcours 2 à 4 de `quickstart.md`

À ce stade, une adhésion validée affiche sa référence mais pas encore les
moyens de paiement : livrer US4 et US5 avant d'ouvrir les adhésions aux
familles si le paiement doit passer par Frigg.

### Incremental Delivery

1. Setup + Foundational
2. US1 + US2 + US3 → les demandes aboutissent (MVP)
3. US5 puis US4 → paiement hors ligne de bout en bout
4. US6 → bulletins papier et fiches sans compte
5. US7 → corrections et départs
6. US8 → chiffres pour l'assemblée générale

### Notes

- Commit après chaque tâche ou groupe logique (gitmoji, message court, validation de l'utilisateur avant commit)
- Commentaires des requêtes sqlc en ASCII uniquement
- `s.today()` pour toute date du jour, jamais `time.Now`
- E-mails mis en file dans la transaction de l'action, destinataire relu à ce moment
- Aucune donnée personnelle, aucun motif ni coordonnée bancaire dans les journaux
