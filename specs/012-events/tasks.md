---

description: "Liste des tâches : événements de l'association"
---

# Tasks: Événements de l'association

**Input**: Design documents from `specs/012-events/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : matrice d'autorisation (principes II et III), tests
d'API sur base isolée, Playwright + axe sur chaque parcours touché. La
migration de reconstruction (R2) et la capacité sous concurrence (R6) ont
leurs propres tests.

**Organization**: une phase par user story, dans l'ordre de la spec. US1 (P1)
livre la création et la publication, US2 (P1) la consultation dans l'espace
membre, US3 (P1) l'inscription, US4 (P1) le paiement et « À régler », US5
(P2) le suivi des inscrits par l'administration, US6 (P2) l'annulation et la
suppression.

**Conventions** (constitution 1.8.0) : identifiants, tables, champs JSON et
valeurs d'énumération en anglais (`season_members`, `confirmed`…) ;
commentaires, messages, textes d'interface, e-mails et journaux en français.
Commentaires de la migration et des requêtes sqlc en **ASCII uniquement**
(sqlc sqlite casse sur un commentaire non ASCII). Jamais de titre, lieu,
description, nom de membre, adresse électronique ni message d'annulation
dans les journaux (R15). Aucun test de rôle dans les handlers : le rôle est
déclaré par `register` ; l'isolation des fiches passe par les requêtes
filtrées par compte. Toutes les écritures en `s.Store.WriteTx` (`BEGIN
IMMEDIATE`), e-mails insérés dans la même transaction.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US6)

## Path Conventions

Module Go à la racine (`internal/`), SPA dans `web/` (plan.md › Project
Structure). Code existant à réutiliser :

- API : `internal/api/api.go` (`register`, `server`, `s.now()`, `s.today()`,
  `s.timestamp()`, `CurrentAccount`, `ownAccount`, `s.internalError`),
  `internal/api/errors.go` (`FieldErrors`), `internal/api/memberships.go`
  (`seasonIndex`, `seasonInfo`, `SeasonRef`, `notifyMembership`,
  `noticeRecipient`, `formatEuros`, `frenchDate`, `methodLabels`, `ns`,
  `nid`), `internal/api/memberships_me.go` (`MyMembership`, `AnnounceBody`,
  `announce-my-payment`), `internal/api/memberships_admin.go`
  (`AdminMembership`, `RecordPaymentBody`, `record-membership-payment`,
  `cancel-membership-payment`, `withOnlineAdmin`), `internal/api/card_payments.go`
  (`startMyCardPayment`, `confirmMyCardPayment`, `verifyCardPayment`,
  `verifyOnlinePayment`, `settleOnlinePayment`, `settleRefund`,
  `cardItemName`, `resumeWindow`, `verifiableWindow`), `internal/api/online_sweeper.go`
  (`OnlinePaymentSweeper`), `internal/api/helloasso_webhook.go`,
  `internal/api/payment_alerts.go`, `internal/api/membership_fees.go`
  (analyse du montant en euros saisi, `Amount string`), `internal/api/seasons.go`,
  `internal/api/members_admin.go`, `internal/api/members_me.go`.
- Domaine : `internal/membership` (`Snapshot`, `CanPay`, `CanPayByCard`,
  `CanCancelPayment`, `SettleOnline`, `RefundCancels`, `NextReference`,
  `PaymentAfter`, `Method`, `PaymentStatus`, `MinCardAmount`),
  `internal/season` (`Paris`, `Today`, `Date`, `ParseDate`, `FieldErrors`),
  `internal/communication` (`Candidate`, `BuildAudience`),
  `internal/helloasso` (`Metadata`, `TruncateItemName`), `internal/textfold`.
- Base : `internal/database` (`Store.Read`, `Store.WriteTx`,
  `FTSTrigramQuery`), migration `00008_memberships.sql` (modèle de
  `memberships`, `online_payments`, `payment_alerts`, `memberships_fts`),
  `internal/database/queries/{memberships,online_payments,payment_alerts,members,seasons}.sql`.
- Mail : `internal/mail/render.go` (`Data`), `internal/mail/templates/membership-*.tmpl`
  (modèles), `internal/mail/render_test.go`.
- Tests Go : `internal/api/apitest` (`New`, `Seed`, `LoggedIn`, `SetToday`,
  `Logs`, `Store`, `Client().Do`, `Response.JSON`),
  `internal/api/memberships_helpers_test.go` (`newMsFixture`, `request`,
  `validate`, `setPayment`), `internal/api/card_payments_test.go`
  (`newCardFixture`, `cardPath`), `internal/helloasso/hatest/fake.go`,
  `internal/api/access_test.go` (`TestAccessMatrix`, `allowedPublic`),
  `internal/database/db_test.go`.
- Frontend : `web/app/routes.ts` (`memberComingSoon`, `adminComingSoon`),
  `web/app/shell/navigation.ts` (`comingSoon`), `web/app/api/*`
  (`client.ts` `$api`, `queries.ts`, `memberships.ts`, `problem.ts`
  `applyProblemToForm`), `web/app/ui/*` (`Button`, `Field`, `Checkbox`,
  `RadioGroup`, `Select`, `Badge`, `Alert`, `ConfirmDialog`, `Table`,
  `Pagination`, `Icon`), `web/app/lib/format.ts`,
  `web/app/memberships/{CardPaymentPanel,PaymentMethodInfo,MembershipBadges,MemberPicker}.tsx`,
  `web/app/routes/app/member/membership-detail.tsx`,
  `web/app/routes/app/admin/{membership-detail,FeeDialog}.tsx`. E2E :
  `web/e2e/test.ts`, `web/e2e/axe.ts`, `web/e2e/mailpit.ts`,
  `web/e2e/fake-helloasso.ts`, `web/e2e/memberships.spec.ts`,
  `web/e2e/card-payment.spec.ts`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: schéma, requêtes et métadonnées HelloAsso communs à toutes les stories.

- [X] T001 Créer la migration `internal/database/migrations/00012_events.sql` (commentaires ASCII), première ligne `-- +goose NO TRANSACTION`, section Up encadrée par `PRAGMA foreign_keys = OFF;` + `BEGIN;` … `COMMIT;` + `PRAGMA foreign_keys = ON;` (research.md › R2), tables `STRICT` selon data-model.md : `events` (`id INTEGER PRIMARY KEY`, `season_id INTEGER NOT NULL REFERENCES seasons (id) ON DELETE RESTRICT`, `title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 120)`, `location TEXT NOT NULL CHECK (length(location) BETWEEN 1 AND 200)`, `description TEXT CHECK (description IS NULL OR length(description) BETWEEN 1 AND 5000)`, `starts_at TEXT NOT NULL`, `ends_at TEXT NOT NULL`, `CHECK (ends_at > starts_at)`, `openness TEXT NOT NULL DEFAULT 'season_members' CHECK (openness IN ('season_members','all_members'))`, `capacity INTEGER CHECK (capacity IS NULL OR capacity BETWEEN 1 AND 10000)`, `price_cents INTEGER NOT NULL DEFAULT 0 CHECK (price_cents BETWEEN 0 AND 1000000)`, `registration_opens_at TEXT`, `registration_closes_at TEXT CHECK (registration_closes_at IS NULL OR registration_closes_at <= ends_at)`, `CHECK (registration_opens_at IS NULL OR registration_closes_at IS NULL OR registration_closes_at > registration_opens_at)`, `status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','cancelled'))`, `cancel_message TEXT CHECK (cancel_message IS NULL OR length(cancel_message) BETWEEN 1 AND 1000)`, `published_at TEXT`, `cancelled_at TEXT`, `cancelled_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL`, `created_by`/`updated_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL`, `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`, `CHECK (status <> 'cancelled' OR cancelled_at IS NOT NULL)`, `CHECK (status = 'draft' OR published_at IS NOT NULL)`), index `events_starts_at (starts_at)`, `events_season (season_id, starts_at)`, `events_status (status, starts_at)` ; `event_registrations` (`id`, `event_id INTEGER NOT NULL REFERENCES events (id) ON DELETE RESTRICT`, `member_id INTEGER NOT NULL REFERENCES members (id) ON DELETE CASCADE`, `status TEXT NOT NULL CHECK (status IN ('confirmed','cancelled'))`, `registered_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL`, `registered_at TEXT NOT NULL`, `cancelled_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL`, `cancelled_at TEXT`, `amount_cents INTEGER NOT NULL CHECK (amount_cents BETWEEN 0 AND 1000000)`, `payment_reference TEXT UNIQUE CHECK (payment_reference IS NULL OR length(payment_reference) BETWEEN 8 AND 16)`, `payment_status TEXT CHECK (payment_status IN ('due','announced','paid'))`, `announced_method TEXT CHECK (announced_method IN ('cash','cheque','transfer'))`, `paid_method TEXT CHECK (paid_method IN ('cash','cheque','transfer','card'))`, `paid_on TEXT CHECK (paid_on IS NULL OR date(paid_on) = paid_on)`, `paid_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL`, `paid_at TEXT`, `payment_note TEXT CHECK (payment_note IS NULL OR length(payment_note) BETWEEN 1 AND 200)`, `paid_online_payment_id INTEGER REFERENCES online_payments (id) ON DELETE SET NULL`, `paid_amount_cents INTEGER CHECK (paid_amount_cents BETWEEN 0 AND 1000000)`, `updated_at TEXT NOT NULL`, et les CHECK de data-model.md : `CHECK ((payment_status IS NULL) = (amount_cents = 0))`, `CHECK (payment_status IS NULL OR payment_reference IS NOT NULL)`, `CHECK ((payment_status IS 'paid') = (paid_method IS NOT NULL AND paid_on IS NOT NULL))`, `CHECK (payment_status IS NOT 'announced' OR announced_method IS NOT NULL)`, `CHECK (paid_method IS NOT 'card' OR (paid_online_payment_id IS NOT NULL AND paid_amount_cents IS NOT NULL AND paid_by IS NULL))`, `CHECK (paid_online_payment_id IS NULL OR paid_method = 'card')`, `CHECK ((status = 'cancelled') = (cancelled_at IS NOT NULL))`), index `event_registrations_one_confirmed` UNIQUE `(event_id, member_id) WHERE status = 'confirmed'`, `event_registrations_event (event_id, status)`, `event_registrations_member (member_id)`, `event_registrations_payment (payment_status) WHERE payment_status IN ('due','announced')` ; FTS `event_registrations_fts USING fts5(payment_reference, content = 'event_registrations', content_rowid = 'id', tokenize = "trigram remove_diacritics 1")` et triggers `_ai`, `_ad`, `_au` (`AFTER UPDATE OF payment_reference`) copiés de `memberships_fts` (00008) ; **reconstruction** de `online_payments` (`online_payments_new` avec toutes les colonnes et CHECK de 00008, `membership_id INTEGER REFERENCES memberships (id) ON DELETE CASCADE` devenue facultative, nouvelle `registration_id INTEGER REFERENCES event_registrations (id) ON DELETE CASCADE`, `CHECK ((membership_id IS NULL) <> (registration_id IS NULL))` ; `INSERT … SELECT` des colonnes existantes ; `DROP TABLE online_payments` ; `ALTER TABLE online_payments_new RENAME TO online_payments` ; index de 00008 recréés + `online_payments_registration (registration_id, created_at)`) puis de `payment_alerts` à l'identique (`membership_id` facultative, `registration_id` nouvelle `ON DELETE CASCADE`, même CHECK d'exclusivité, index recréés + `payment_alerts_registration (registration_id)`) ; section Down symétrique (même encadrement `PRAGMA`/`BEGIN`) : reconstruction des deux tables à l'état 00008 en supprimant d'abord les lignes visant une inscription, puis suppression des triggers, FTS, `event_registrations` et `events`
- [X] T002 [P] Test de migration in `internal/database/db_test.go` (`TestEventsMigration`) : base migrée jusqu'à `00011` avec saison, tarif, fiche, adhésion validée **payée par carte** (`online_payments` réussie référencée par `memberships.paid_online_payment_id`) et un `payment_alerts` ouvert ; `Up` ; lignes de `online_payments`, `payment_alerts` et `memberships.paid_online_payment_id` identiques ; `PRAGMA foreign_key_check` vide ; `PRAGMA foreign_keys` = 1 sur `store.WriteDB` ; une tentative avec `membership_id` **et** `registration_id`, ou sans aucun, refusée ; une inscription payante sans référence, une deuxième inscription confirmée de la même fiche, une capacité 0 refusées ; deux inscriptions dont une annulée acceptées ; suppression d'une fiche → inscriptions, tentatives et signalements supprimés ; `INSERT INTO event_registrations_fts(event_registrations_fts) VALUES ('integrity-check')` sans erreur ; `Down` puis `Up` sans erreur et `foreign_keys` = 1
- [X] T003 [P] Créer `internal/database/queries/events.sql` (commentaires ASCII) : `CreateEvent` (`RETURNING *`), `UpdateEvent`, `GetEvent` (avec libellés des comptes `created_by`, `updated_by`, `cancelled_by`), `ListEvents` (filtres facultatifs `season_id`, `status`, `upcoming`/`past` comparés à `sqlc.arg(now)`, avec `confirmed_count` par sous-requête), `ListMemberEvents` (`status IN ('published','cancelled') AND ends_at > now`, par `starts_at`), `GetMemberEvent` (`status <> 'draft'`), `PublishEvent`, `CancelEvent`, `RestoreEvent` (chacune `:execrows` conditionnée par l'état attendu), `DeleteEventWithoutRegistrations` (`:execrows`, `AND NOT EXISTS (SELECT 1 FROM event_registrations WHERE event_id = id)`), `CountEventsOutsideSeason` (`season_id` et nouvelles bornes, comparaison sur la date locale fournie en paramètres UTC calculés côté Go), `CountEventsOfSeason`
- [X] T004 [P] Créer `internal/database/queries/event_registrations.sql` (commentaires ASCII) : `CreateRegistration` (`RETURNING *`), `CountConfirmedRegistrations` (`event_id`), `ListRegistrationReferences` (préfixe, pour `NextReference`), `GetRegistration` (jointures événement, fiche, compte gestionnaire `accounts.email`/`disabled_at`, adresse de la fiche), `GetRegistrationOfAccount` (`id`, `account_id` : fiche gérée par le compte), `ListRegistrationsOfEvent` (`state` `confirmed`|`all`, jointure fiche + compte + adhésion validée à la saison de l'événement `is_season_member`), `ListConfirmedRegistrationsOfMembers` (`event_id`, ids de fiches du compte), `ListRegistrationsOfAccount` (fiches du compte, avec événement), `ListRegistrations` (filtres `event_id`, `status`, `payment`, recherche `members_fts` OU `event_registrations_fts` via `MATCH`, comme `ListMemberships`), `CancelRegistration` (`:execrows`, `status = 'confirmed'`), `AnnounceRegistrationPayment`, `RecordRegistrationPayment`, `CancelRegistrationPayment` (rétablit `announced` ou `due` comme 006), `RecordRegistrationCardPayment`, `CancelRefundedRegistration`, `ListDueRegistrationsOfAccount` (`payment_status IN ('due','announced')`, inscription confirmée, événement non annulé), `RegistrationTotals` (`event_id` : sommes dues, encaissées hors test, encaissées en test, nombre payées), `CountPaidUpcomingRegistrationsOfMember` (`member_id`, `now`), `ListNoticeRecipientsOfEvent` (inscriptions confirmées avec prénom, nom, e-mail du compte actif et e-mail de la fiche)
- [X] T005 Adapter `internal/database/queries/online_payments.sql` et `internal/database/queries/payment_alerts.sql` (commentaires ASCII) : `CreateOnlinePayment` et `CreatePaymentAlert` prennent `membership_id` et `registration_id` facultatifs ; ajouter `FindResumableRegistrationOnlinePayment`, `GetOnlinePaymentOfRegistrationAccount`, `ListVerifiableOnlinePaymentsOfRegistration`, `ListOnlinePaymentsOfRegistration`, `ListPaymentAlertsOfRegistration`, `ListOpenAlertCountsByRegistration` ; `ListOpenAlertCounts` filtré sur `membership_id IS NOT NULL` ; `CountOpenPaymentAlerts` remplacée par `CountOpenMembershipPaymentAlerts` (`membership_id IS NOT NULL`) et `CountOpenRegistrationPaymentAlerts` (`registration_id IS NOT NULL`), pour que le compteur actuel du tableau de bord ne mélange pas les deux cibles ; lancer `sqlc generate` (code commité in `internal/database/sqlc/`) et corriger la compilation des appels existants (`MembershipID` devenu `sql.NullInt64`) in `internal/api/card_payments.go`, `internal/api/online_sweeper.go`, `internal/api/helloasso_webhook.go`, `internal/api/payment_alerts.go`, `internal/api/memberships_admin.go`
- [X] T006 [P] Métadonnées HelloAsso in `internal/helloasso/checkout.go` : `Metadata.Membership` en `json:"membership,omitempty"`, nouveau `Registration int64 \`json:"registration,omitempty"\`` ; test in `internal/helloasso/client_test.go` : relecture d'une intention ancienne (sans `registration`) et d'une intention d'inscription ; `internal/helloasso/hatest/fake.go` restitue `registration`

**Checkpoint**: schéma et requêtes générés ; `go build ./...` et `go test ./...` passent ; comportement inchangé pour les adhésions.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: domaine pur, généralisation du circuit de carte à une cible, squelette des opérations et des e-mails, utilisés par toutes les stories.

**⚠️ CRITICAL**: aucune story ne commence avant la fin de cette phase.

- [X] T007 [P] Créer `internal/event/event.go` (commentaire de paquet en français : règles pures des événements, sans base) : `Status` (`draft`, `published`, `cancelled`), `Phase` (`draft`, `upcoming`, `finished`, `cancelled`), `Openness` (`season_members`, `all_members`), constantes `MaxTitle = 120`, `MaxLocation = 200`, `MaxDescription = 5000`, `MaxCapacity = 10000`, `MaxPriceCents = 1000000`, `MaxCancelMessage = 1000` ; `ParseLocal(s string) (time.Time, error)` : format `2006-01-02T15:04` en `season.Paris`, heure inexistante (passage à l'heure d'été) refusée (« Cette heure n'existe pas ce jour-là (changement d'heure). »), heure ambiguë → première occurrence ; `FormatLocal(t)` ; `Input{Title, Location, Description string; StartsAt, EndsAt string; Openness; Capacity *int; PriceCents int64; RegistrationOpensAt, RegistrationClosesAt *string}` ; `SeasonSpan{ID int64; StartsOn, EndsOn season.Date}` ; `ValidateEvent(in Input, seasons []SeasonSpan) (Event, season.FieldErrors)` : espaces de bord retirés, titre « 1 à 120 caractères », lieu « 1 à 200 caractères », description vide → nil sinon « 5 000 caractères au plus », fin > début (« La fin doit suivre le début. » sur `endsAt`), saison contenant la date locale du début sinon « Aucune saison ne couvre cette date. » sur `startsAt`, capacité 1–10 000, prix 0–10 000 €, ouverture < clôture, clôture ≤ fin ; `PhaseOf(ev Event, now time.Time) Phase`
- [X] T008 [P] Créer `internal/event/registration.go` : `Ineligibility` (`not_season_member`, `already_registered`) ; `Eligible(o Openness, isSeasonMember, alreadyRegistered bool) (bool, Ineligibility)` ; `RegistrationWindow(ev, now) (opens, closes time.Time, open bool)` (ouverture nulle = publication, clôture nulle = début) ; `CanRegister(ev, now)` (publié, non terminé, période ouverte) ; `CanAdminRegister(ev, now)` (publié, avant la fin) ; `CapacityLeft(capacity *int, confirmed int) *int` ; `CheckPlaces(capacity *int, confirmed, requested int) error` (tout ou rien, `ErrFull{Left int}`) ; `RegSnapshot{Confirmed bool; Payment membership.PaymentStatus; PaidMethod membership.Method; CardPending bool}` ; `CanUnregister(ev, r, now)` (FR-013 : confirmée, période ouverte, non payée, pas de carte en cours) ; `CanRemove(ev, r, now)` (confirmée, avant la fin) ; `CanUpdate(ev, now)`, `CanUpdateDates(ev, now)` (non terminé), `CanPublish`, `CanCancel`, `CanRestore` (annulé, avant la fin), `CanDelete(hasRegistrations bool)` ; `CheckCapacityChange(capacity *int, confirmed int) error` (« Au moins n inscrits : la capacité ne peut pas être inférieure. »)
- [X] T009 [P] Créer `internal/event/payment.go` et `internal/event/reference.go` : `PaymentSnapshot(r RegSnapshot, amountCents int64, eventCancelled bool) membership.Snapshot` (`Status = membership.Validated` si `r.Confirmed && !eventCancelled`, sinon `membership.Cancelled` ; `SeasonClosed = false` ; `PaymentStatus`, `AmountCents`, `PaidMethod` recopiés) ; `ReferencePrefix(starts, ends season.Date) string` = `EVT-` + deux chiffres de chaque année + `-` (« EVT-2627- »)
- [X] T010 [P] Tests en table in `internal/event/event_test.go`, `registration_test.go`, `payment_test.go`, `reference_test.go` : titre 120/121 runes avec accents, lieu 200/201, description 5000/5001, fin = début, début avant la première saison et après la dernière, début le 31 août 23 h 30 (heure de Paris) rattaché à la saison qui finit le 31 août, `2027-03-28T02:30` refusé (heure inexistante), `2026-10-25T02:30` accepté, capacité 0/1/10000/10001, prix 1 000 001, clôture après la fin, ouverture après clôture ; `PhaseOf` aux bornes ; période par défaut ; éligibilité des deux ouvertures ; `CheckPlaces` 3 places / 2 demandées / 1 restante ; `CanUnregister` payée, carte en cours, période close ; `PaymentSnapshot` × `membership.SettleOnline` : inscription confirmée → `RecordPayment`, annulée → `AlertRefundDue`, événement annulé → `AlertRefundDue`, déjà payée → `AlertDuplicate`, montant différent → `AlertAmountMismatch` ; `ReferencePrefix` 2026-09-01/2027-08-31 → `EVT-2627-`
- [X] T011 Généraliser le circuit de carte in `internal/api/card_payments.go` (research.md › R4), **sans changement de comportement** pour les adhésions : interface interne `payTarget` (`snapshot() membership.Snapshot`, `paidByAttempt(opID int64) bool`, `recordCard(ctx, q, paidOn, amount, opID) error`, `cancelRefunded(ctx, q, opID) error`, `notifyPaid(ctx, q) error`, `notifyCancelled(ctx, q) error`, `logAttrs() []any`) ; `loadTarget(ctx, q, op sqlc.OnlinePayment) (payTarget, error)` qui choisit selon `op.MembershipID`/`op.RegistrationID` ; implémentation `membershipTarget` (code actuel déplacé de `settleOnlinePayment` et `settleRefund`) ; `verifyOnlinePayment` vérifie que la cible des métadonnées (`Membership` ou `Registration`) est celle de la tentative ; `settleOnlinePayment`, `settleRefund` et les signalements passent par `payTarget` ; même adaptation in `internal/api/online_sweeper.go` et `internal/api/helloasso_webhook.go` (le webhook ne connaît pas la cible) ; `go test ./internal/api/ -run 'Card|HelloAsso|Sweeper|Alert|Refund'` reste vert
- [X] T012 [P] Test croisé in `internal/api/card_payments_test.go` (`TestCardMetadataTargetMismatch`) : une intention dont les métadonnées visent une autre adhésion, ou une inscription (`Registration` non nul), n'est jamais constatée sur la tentative d'une adhésion (journal « intention HelloAsso sans rapport avec la tentative »)
- [X] T013 Créer le squelette in `internal/api/events_admin.go` (`func (s *server) registerAdminEvents(api huma.API)`, tag `events`), `internal/api/events_me.go` (`registerMyEvents`), `internal/api/registrations_admin.go` (`registerAdminRegistrations`, tag `registrations`), `internal/api/registrations_me.go` (`registerMyRegistrations`), appelés depuis `New` in `internal/api/api.go` ; types huma communs in `internal/api/events_admin.go` selon contracts/api.md › Types (`LocalDateTime` : `string` avec `pattern:"^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}$"`, `EventBody` avec `title` `minLength:"1" maxLength:"120"`, `location` `minLength:"1" maxLength:"200"`, `description` `maxLength:"5000"` nullable, `openness` `enum:"season_members,all_members"`, `capacity` `minimum:"1" maximum:"10000"` nullable, `price` `string` `maxLength:"12"`, `registrationOpensAt`/`registrationClosesAt` nullables ; `EventSummary`, `AdminEvent`, `MyEvent`, `MemberRef`, `EventRef`) ; fonctions `eventSummary(…)`, `seasonSpans(seasonIndex)` ; journalisation `s.logEventAction(ctx, action string, eventID int64, attrs ...any)` et `s.logRegistrationAction(ctx, action string, eventID, registrationID int64, attrs ...any)` (attributs `action`, `actor_id`, `event_id`, `registration_id`, jamais de titre, lieu ni nom, R15)
- [X] T014 Créer `internal/api/event_notices.go` : `s.noticeEventRecipients(ctx, q, eventID) (communication.Audience, error)` (via `ListNoticeRecipientsOfEvent` → `communication.Candidate` → `communication.BuildAudience`, R10) ; `s.notifyRegistration(ctx, q, template string, registrationID int64) error` (adresse de contact : compte gestionnaire actif, sinon adresse de la fiche, sinon rien) ; `s.notifyEventRegistrants(ctx, q, template string, eventID int64, extra mail.Data) (int, error)` (un e-mail par destinataire, membres nommés) ; ajouter à `mail.Data` in `internal/mail/render.go` les champs `Event`, `When` (plage formatée en français, heure de Paris), `Location`, `PreviousWhen`, `PreviousLocation`, `Members []string`, `Message` ; fonction `frenchDateTimeRange(start, end time.Time) string` in `internal/api/event_notices.go` (« samedi 12 décembre 2026 de 14 h à 17 h », « du lundi 13 au vendredi 17 avril 2027 ») avec test en table in `internal/api/event_notices_internal_test.go`

**Checkpoint**: `go test ./internal/event/... ./internal/helloasso/...` passe ; les tests de paiement des adhésions restent verts ; `go build ./...` passe.

---

## Phase 3: User Story 1 - Créer et publier un événement (Priority: P1) 🎯 MVP

**Goal**: un administrateur crée, modifie et publie des événements dans la rubrique « Événements » ; les dates des saisons restent cohérentes.

**Independent Test**: créer un événement, vérifier qu'il reste invisible tant qu'il est en brouillon, le publier, le modifier, le retrouver dans « À venir » ou « Passés » ; un membre est refusé.

### Tests for User Story 1

- [X] T015 [P] [US1] Tests d'API in `internal/api/events_admin_test.go` : `list-events` vide ; `create-event` → `201`, `status: draft`, `phase: draft`, saison déduite, `priceCents` analysé depuis `price: "15,50"` → 1550, `""` → 0 ; fin avant début → `422 body.endsAt` ; début hors saison → `422 body.startsAt` ; heure inexistante → `422 body.startsAt` ; titre 121 caractères → `422 body.title` ; `publish-event` → `published` ; republier → `409` ; `update-event` titre et description sans e-mail ; lieu modifié avec deux inscriptions confirmées de deux fiches gérées par le même compte (insérées par `env.Store`) → **un** e-mail `event-changed` nommant les deux ; événement terminé (`env.SetToday`) : changement de dates → `409`, titre → `200` ; capacité sous le nombre d'inscrits → `422 body.capacity` ; nouveau prix → inscriptions existantes inchangées ; `list-events` filtres `season`, `status`, `when` et ordre ; `get-event` `actions` selon l'état ; journaux `event.create|update|publish` sans titre ni lieu (`env.Logs`)
- [X] T016 [P] [US1] Tests in `internal/api/seasons_update_test.go` (fichier qui couvre déjà `update-season` et `delete-season`, helpers `seasonPath`, `expectProblem`) : `update-season` qui exclurait un événement (inséré par `env.Store`) → `409` avec le nombre d'événements ; `delete-season` d'une saison avec événement → `409` explicite ; sans événement → comportement inchangé (tests existants verts)
- [X] T017 [P] [US1] Vérifier que la matrice `internal/api/access_test.go` couvre les opérations admin de US1 (anonyme → `401`, membre → `403`) sans modifier `allowedPublic`

### Implementation for User Story 1

- [X] T018 [US1] Opérations in `internal/api/events_admin.go` (contracts/api.md › Opérations `admin`), `auth.Admin` : `list-events` (`GET /admin/events`, `season` défaut saison courante, `status` dont `finished` calculé, `when`), `get-event` (`GET /admin/events/{id}`, `totals` à zéro tant que US4/US5 ne sont pas livrées, `actions`), `create-event` (`POST /admin/events`, `201` ; prix analysé par `season.ParseAmount` (`internal/season/fee.go`, même règle que le montant d'un tarif), chaîne vide → 0, erreur sur `body.price` ; même analyse dans `update-event` ; `event.ValidateEvent` avec `seasonSpans` ; instants stockés en UTC), `update-event` (`PUT /admin/events/{id}` ; `CanUpdate`, `CanUpdateDates` si début ou fin changent, `CheckCapacityChange` avec `CountConfirmedRegistrations` dans la transaction ; si publié, non terminé et début, fin ou lieu changés : `notifyEventRegistrants("event-changed", PreviousWhen, PreviousLocation)` dans la transaction), `publish-event` (`POST /admin/events/{id}/publish` ; `PublishEvent` `:execrows` = 0 → `409`) ; `404` « Cet événement n'existe plus. » ; journal par action avec `notified`
- [X] T019 [US1] Contrôles des saisons in `internal/api/seasons.go` : `update-season` refusé (`409` « n événement(s) sortiraient de la saison : modifiez-les d'abord. ») via `CountEventsOutsideSeason` ; `delete-season` refusé (`409` « Cette saison a des événements. ») via `CountEventsOfSeason`, dans la transaction existante
- [X] T020 [P] [US1] Modèle d'e-mail `internal/mail/templates/event-changed.txt.tmpl` et `.html.tmpl` (objet « Changement : <événement> », anciennes et nouvelles date/heure et lieu, liste des inscrits concernés) sur le modèle de `membership-validated.*` ; cas ajouté in `internal/mail/render_test.go`
- [X] T021 [US1] Régénérer le contrat avec `go tool task gen` (`web/app/api/openapi.json`, `web/app/api/schema.d.ts`), fichiers commités
- [X] T022 [P] [US1] Requêtes et mutations in `web/app/api/events.ts` : `eventsQuery(filters)`, `eventQuery(id)`, `useCreateEvent`, `useUpdateEvent`, `usePublishEvent` (invalidation explicite de `list-events`, `get-event`, `list-my-events`, `get-my-event`)
- [X] T023 [P] [US1] Composants in `web/app/events/` : `EventDate.tsx` (plage de dates `Intl.DateTimeFormat("fr-FR", { timeZone: "Europe/Paris" })`, « sam. 12 déc. · 14 h – 17 h », plusieurs jours « du lun. 13 au ven. 17 avr. »), `EventBadges.tsx` (« Brouillon » `--muted`, « Publié » `--ok`, « Annulé » `--danger`, « Terminé » `--info`, « Complet » `--warn`, « Gratuit »/prix) ; tests Vitest `web/app/events/EventDate.test.ts` (même jour, plusieurs jours, passage à l'heure d'hiver)
- [X] T024 [US1] Formulaire `web/app/events/EventForm.tsx` (contracts/ui-routes.md › Formulaire d'événement) : `react-hook-form` ; titre (compteur 120), lieu, description (`textarea`, compteur 5 000), début et fin (`input type="datetime-local"`), `RadioGroup` « Qui peut s'inscrire ? », capacité (facultative, `type="number" min="1" max="10000"`), prix (texte « En euros. Laisser vide pour un événement gratuit. »), `Collapsible` base-ui « Période d'inscription » ; **aucune déduction de la saison côté client** (principe VI : la règle de rattachement reste au serveur) : aide statique sous le début « L'événement est rattaché à la saison qui contient sa date de début. », et en modification la saison actuelle lue dans `AdminEvent.season` ; une date hors saison est signalée par le `422` sur `startsAt` ; en modification avec inscrits : avertissements « Les inscrits seront prévenus » (début, fin ou lieu modifiés) et « Le nouveau prix s'appliquera aux prochaines inscriptions » ; `422` par `applyProblemToForm`
- [X] T025 [US1] Écrans admin et routes : `web/app/routes/app/admin/events.tsx` (filtres saison et état en paramètres d'URL, régions « À venir » et « Passés », `Table`, inscrits « 12 / 20 », état vide avec explication et bouton « Créer un événement »), `event-new.tsx`, `event-edit.tsx` (formulaire `EventForm`), `event-detail.tsx` (en-tête, badges, actions « Publier » et « Modifier », section Informations ; sections Chiffres et Inscrits ajoutées par US5) ; `web/app/routes.ts` : `route("evenements", …layout, [index, route("nouveau"), route(":id"), route(":id/modifier")])` sous `admin`, `evenements` retiré de `adminComingSoon` ; `web/app/shell/navigation.ts` : `comingSoon` retiré de `gestion-evenements`
- [X] T026 [US1] Scénario Playwright `web/e2e/events.spec.ts` (« créer et publier ») : état vide, création « Assemblée générale », fin avant début refusée avec message sur le champ, publication, modification ; membre : `/admin/evenements` refusé ; 320 px sans défilement horizontal ; axe sans violation `serious`/`critical`

**Checkpoint**: US1 livrable seule : agenda géré par l'administration, sans effet visible encore côté membre.

---

## Phase 4: User Story 2 - Consulter les événements à venir (Priority: P1)

**Goal**: un compte voit les événements publiés à venir et leur détail.

**Independent Test**: deux événements publiés à venir, un brouillon, un annulé (via `env.Store`) et un passé ; vérifier depuis un compte membre ce qui est listé, dans quel ordre, et que le brouillon renvoie `404`.

### Tests for User Story 2

- [X] T027 [P] [US2] Tests d'API in `internal/api/events_me_test.go` : `list-my-events` : publiés et annulés non terminés seulement, par début croissant, `placesLeft`, `registrationOpen` ; `get-my-event` d'un brouillon ou inexistant → `404` identique ; réponse brute de `list-my-events` sans le titre d'un brouillon ; `members` : fiches du compte uniquement, `eligible`/`reason` (`not_season_member` pour une fiche sans adhésion validée à la saison, `already_registered` pour une inscription confirmée insérée par `env.Store`) ; ouverture `all_members` → toutes éligibles
- [X] T028 [P] [US2] Vérifier la matrice `internal/api/access_test.go` pour `list-my-events` et `get-my-event` (anonyme → `401`, membre accepté)

### Implementation for User Story 2

- [X] T029 [US2] Opérations in `internal/api/events_me.go`, `auth.Member` : `list-my-events` (`GET /me/events`, `ListMemberEvents`), `get-my-event` (`GET /me/events/{id}`, `GetMemberEvent`, fiches via `ListMyMembers` existante, adhésions validées de la saison, inscriptions confirmées via `ListConfirmedRegistrationsOfMembers`, `event.Eligible`) ; `MyRegistrationRef` nul tant que US3 n'est pas livrée
- [X] T030 [US2] Régénérer le contrat avec `go tool task gen`, fichiers commités
- [X] T031 [P] [US2] Requêtes in `web/app/api/events.ts` : `myEventsQuery()`, `myEventQuery(id)`
- [X] T032 [US2] Écrans membre : `web/app/routes/app/member/events.tsx` (contracts/ui-routes.md › `/evenements` : cartes `ul`, date en bloc, `EventDate`, lieu, `EventBadges`, places restantes, état vide « Aucun événement à venir pour le moment. ») et `web/app/routes/app/member/event-detail.tsx` (en-tête, description en texte avec retours à la ligne conservés, encart « Inscriptions » en lecture : période, ouverture, fiches avec badge « Inscrit » ou raison d'inéligibilité ; alerte `--danger` et message pour un événement annulé) ; `web/app/routes.ts` : `route("evenements", …, [index, route(":id")])`, `evenements` retiré de `memberComingSoon` ; `web/app/shell/navigation.ts` : `comingSoon` retiré de `evenements`
- [X] T033 [US2] Scénario Playwright in `web/e2e/events.spec.ts` (« consulter ») : liste ordonnée, brouillon absent et `/evenements/<id>` → « Page introuvable », événement annulé signalé ; axe et 320 px

**Checkpoint**: agenda visible des membres (US1 + US2).

---

## Phase 5: User Story 3 - Inscrire ses membres à un événement (Priority: P1)

**Goal**: un compte inscrit des fiches éligibles à un événement gratuit, voit « Mes inscriptions » et annule pendant la période.

**Independent Test**: sur un événement gratuit réservé aux adhérents, inscrire la fiche adhérente, constater le refus pour l'autre, l'e-mail, la place décomptée et « Mes inscriptions », puis annuler et vérifier que la place est libérée.

### Tests for User Story 3

- [X] T034 [P] [US3] Tests d'API in `internal/api/registrations_me_test.go` : `register-my-members` d'une fiche éligible → `201`, `confirmed`, e-mail `registration-confirmed` à l'adresse de contact (compte actif ; fiche sans compte avec adresse ; aucune adresse → aucun e-mail) ; deux fiches en une demande → un seul e-mail nommant les deux ; fiche non adhérente sur `season_members` → `422 body.memberIds[0]` ; `all_members` → acceptée ; fiche d'un autre compte → `404` sans dire laquelle ; déjà inscrite → `422` ; période close (`env.SetToday`) ou événement annulé, brouillon (`404`), terminé → `409` ; complet → `409` « Complet : il reste 0 place(s). » ; 2 places restantes et 3 fiches demandées → `409`, rien d'inscrit ; `cancel-my-registration` → `cancelled`, place libérée, réinscription possible (nouvelle ligne) ; annulation période close → `409` ; `list-my-registrations` à venir puis passées, fiches du compte seulement ; `get-my-registration` d'un autre compte → `404` ; journaux `registration.create|cancel` sans nom
- [X] T035 [P] [US3] Test de concurrence in `internal/api/registrations_concurrency_test.go` : 50 comptes, chacun une fiche éligible, 50 goroutines sur un événement de 3 places → exactement 3 `201` et 47 `409`, `CountConfirmedRegistrations` = 3 (SC-004)
- [X] T036 [P] [US3] Vérifier la matrice `internal/api/access_test.go` pour `register-my-members`, `list-my-registrations`, `get-my-registration`, `cancel-my-registration`

### Implementation for User Story 3

- [X] T037 [US3] Opération `register-my-members` in `internal/api/events_me.go` (`POST /me/events/{id}/registrations`, `{ memberIds: int[] }` `minItems:"1" maxItems:"20" uniqueItems:"true"`, `201 RegisterResult { event: MyEvent, createdRegistrationIds: int[] }`, ids dans l'ordre de `memberIds`) : dans `WriteTx`, relecture de l'événement (`GetMemberEvent`), `event.CanRegister`, fiches du compte (sinon `404`), éligibilité de chaque fiche (`422 body.memberIds[i]` « Lucas n'est pas adhérent de la saison 2026-2027. » / « Lucas est déjà inscrit. »), `CountConfirmedRegistrations` + `event.CheckPlaces` (R6), puis `CreateRegistration` pour chaque fiche avec `amount_cents` = prix courant (événement gratuit : sans référence ni état de paiement ; payant : géré par US4, refusé `409` « Inscription payante bientôt disponible » tant que US4 n'est pas livrée), `notifyRegistration("registration-confirmed")` regroupé ; journal par inscription
- [X] T038 [US3] Opérations in `internal/api/registrations_me.go`, `auth.Member` : `list-my-registrations` (`GET /me/registrations`), `get-my-registration` (`GET /me/registrations/{id}`, `404` hors des fiches du compte), `cancel-my-registration` (`POST /me/registrations/{id}/cancel` ; `event.CanUnregister` sinon `409` avec le message de contracts/api.md ; `CancelRegistration` `:execrows`) ; type `MyRegistration` (sans `paymentMethods` dans la liste) ; `MyEvent.members[].registration` rempli in `internal/api/events_me.go`
- [X] T039 [P] [US3] Modèle d'e-mail `internal/mail/templates/registration-confirmed.txt.tmpl` et `.html.tmpl` (événement, date, lieu, inscrits ; bloc conditionnel montant, référence et lien de paiement pour US4) ; cas in `internal/mail/render_test.go`
- [X] T040 [US3] Régénérer le contrat avec `go tool task gen`, fichiers commités
- [X] T041 [P] [US3] Requêtes et mutations in `web/app/api/registrations.ts` : `myRegistrationsQuery()`, `myRegistrationQuery(id)`, `useRegisterMyMembers(eventId)`, `useCancelMyRegistration` (invalidation de `get-my-event`, `list-my-events`, `list-my-registrations`, `get-my-registration`)
- [X] T042 [US3] Composant `web/app/events/RegisterMembers.tsx` intégré à `web/app/routes/app/member/event-detail.tsx` : `fieldset` « Qui inscrire ? » (une `Checkbox` par fiche, désactivée avec raison reliée par `aria-describedby` si non éligible, badge « Inscrit » et lien « Voir l'inscription »), bouton « Inscrire », encart « Complet » `--warn` ; après succès : notification et focus sur le titre ; après `409` complet : annonce `aria-live` des places restantes
- [X] T043 [US3] Écrans `web/app/routes/app/member/registrations.tsx` (régions « À venir » et « Passées », `Table` en cartes sous 640 px : membre, événement, date, état ; état vide avec lien vers « Événements à venir ») et `web/app/routes/app/member/registration-detail.tsx` (récapitulatif ; « Annuler mon inscription » par `ConfirmDialog` si `actions.cancel`) ; `web/app/routes.ts` : `route("mes-inscriptions", …, [index, route(":id")])`, retrait de `memberComingSoon` ; `web/app/shell/navigation.ts` : `comingSoon` retiré de `mes-inscriptions`
- [X] T044 [US3] Scénario Playwright in `web/e2e/events.spec.ts` (« s'inscrire ») : fiche non adhérente désactivée avec raison, inscription, e-mail reçu (`mailpit.ts`), « Mes inscriptions », annulation par dialogue, réinscription ; axe et 320 px

**Checkpoint**: événements gratuits complets de bout en bout (US1 à US3).

---

## Phase 6: User Story 4 - Payer une inscription (Priority: P1)

**Goal**: les inscriptions payantes reçoivent une référence et se paient par espèces, chèque, virement ou carte ; « À régler » regroupe adhésions et inscriptions.

**Independent Test**: sur un événement payant, inscrire une fiche, vérifier la page de paiement et « À régler », annoncer un chèque ; avec la carte en mode test (`hatest`), payer une seconde inscription et vérifier la constatation automatique.

### Tests for User Story 4

- [X] T045 [P] [US4] Tests d'API in `internal/api/registration_payments_test.go` : inscription payante → `amountCents` figé, `paymentReference` `EVT-2627-0001` puis `-0002`, `payment.status: due`, e-mail avec montant, référence et lien `/mes-inscriptions/{id}` ; changement de prix ensuite → montant figé inchangé ; `get-my-registration` → `paymentMethods` activés ; `announce-my-registration-payment` chèque → `announced` ; moyen désactivé → `422` ; inscription annulée ou événement annulé → `409` ; `cancel-my-registration` d'une inscription payée → `409` « Cette inscription est réglée : contactez l'association pour l'annuler. » ; `list-my-dues` : adhésion à payer + deux inscriptions (« À payer » et « Paiement annoncé »), sans inscription gratuite, payée, annulée ni d'un événement annulé, ni d'une fiche d'un autre compte ; journaux sans nom
- [X] T046 [P] [US4] Tests de carte in `internal/api/registration_card_payments_test.go` (sur le modèle de `card_payments_test.go`, `hatest`) : `start-my-registration-card-payment` → intention avec `Metadata.Registration`, intitulé `Événement <titre> — EVT-…` sans nom du membre, adresses de retour `/mes-inscriptions/{id}?tentative=…&carte=` ; reprise d'une tentative de moins de 14 min ; montant < 1 € → `422` ; `confirm-my-registration-card-payment` `return` → `Payée`, e-mail `registration-paid` ; constatation par notification seule (`receive-helloasso-notification`) ; notification et retour concurrents → une seule constatation ; paiement reçu sur une inscription annulée → `refund_due` ; double paiement → `duplicate` ; remboursement vérifié → tentative `refunded`, inscription annulée, e-mail `registration-removed` ; balayage `OnlinePaymentSweeper.Tick` d'une tentative d'inscription expirée
- [X] T047 [P] [US4] Vérifier la matrice `internal/api/access_test.go` pour `announce-my-registration-payment`, `start-my-registration-card-payment`, `confirm-my-registration-card-payment`, `list-my-dues`

### Implementation for User Story 4

- [X] T048 [US4] Inscription payante in `internal/api/events_me.go` (`register-my-members`, retrait du refus provisoire de T037) : pour un prix > 0, référence `event.ReferencePrefix(saison)` + `membership.NextReference(prefix, ListRegistrationReferences)` dans la transaction, `payment_status = 'due'` ; `registration-confirmed` avec montant, référence et lien ; `createdRegistrationIds` de `RegisterResult` (défini en T037) permet au frontend de rediriger vers la première inscription créée ; test in `internal/api/registration_payments_test.go` (T045) : `createdRegistrationIds` contient exactement les inscriptions créées, dans l'ordre
- [X] T049 [US4] Cible inscription in `internal/api/card_payments.go` : `registrationTarget` implémentant `payTarget` (`event.PaymentSnapshot`, `RecordRegistrationCardPayment`, `CancelRefundedRegistration` avec motif « remboursé », `notifyRegistration("registration-paid"|"registration-removed")`, attributs `registration_id`) ; `loadTarget` la choisit quand `op.RegistrationID` est renseigné
- [X] T050 [US4] Opérations in `internal/api/registrations_me.go`, `auth.Member` : `get-my-registration` complété (`payment`, `paymentMethods` via `s.paymentSettings`, `cardPayment`, `actions` via `membership.MemberActionsFor(event.PaymentSnapshot(…))` et `event.CanUnregister`), `announce-my-registration-payment` (`PUT /me/registrations/{id}/payment`, `AnnounceBody` de 006, `membership.CanPay`), `start-my-registration-card-payment` (`POST …/card-payment`, copie paramétrée de `startMyCardPayment` : `FindResumableRegistrationOnlinePayment`, `CreateOnlinePayment` avec `registration_id`, intitulé `helloasso.TruncateItemName("Événement " + titre + " — " + référence)`, `Metadata{Payment, Registration, Reference}`, retours `/mes-inscriptions/{id}`), `confirm-my-registration-card-payment` (`POST …/card-payment/confirm`, `GetOnlinePaymentOfRegistrationAccount` puis `verifyOnlinePayment`) ; extraire dans `card_payments.go` le code commun aux deux `start…` pour ne pas le dupliquer
- [X] T051 [US4] Opération `list-my-dues` in `internal/api/registrations_me.go` (`GET /me/dues`, `auth.Member`) : adhésions payables du compte (requête existante des adhésions du compte filtrée par `membership.CanPay`) + `ListDueRegistrationsOfAccount`, fusion triée par date de création, type `Due` de contracts/api.md
- [X] T052 [P] [US4] Modèle d'e-mail `internal/mail/templates/registration-paid.txt.tmpl` et `.html.tmpl` (montant, moyen, date, référence, événement) ; cas in `internal/mail/render_test.go`
- [X] T053 [US4] Régénérer le contrat avec `go tool task gen`, fichiers commités
- [X] T054 [US4] Extraire la page de paiement de `web/app/routes/app/member/membership-detail.tsx` dans `web/app/payments/PaymentPanel.tsx` (choix du moyen, `PaymentMethodInfo`, `CardPaymentPanel`, traitement du retour `?tentative=&carte=`), paramétrée par la cible (`{ payment, paymentMethods, cardPayment, actions, announce, startCard, confirmCard }`) ; `membership-detail.tsx` l'utilise sans changement visible ; `web/e2e/memberships.spec.ts` et `web/e2e/card-payment.spec.ts` restent verts
- [X] T055 [P] [US4] Requêtes et mutations in `web/app/api/registrations.ts` : `useAnnounceRegistrationPayment`, `useStartRegistrationCardPayment`, `useConfirmRegistrationCardPayment`, `myDuesQuery()` ; invalidation de `list-my-dues`, `get-my-registration`, `list-my-registrations`
- [X] T056 [US4] Écrans : `web/app/routes/app/member/registration-detail.tsx` complété (récapitulatif montant et référence, `PaymentPanel`, encart « Pour annuler, contactez l'association. » si réglée) ; `RegisterMembers.tsx` : bouton « Inscrire et payer 70 € » (prix × fiches cochées) et redirection vers `/mes-inscriptions/{createdRegistrationIds[0]}` ; `web/app/routes/app/member/registrations.tsx` : colonne paiement (`MembershipBadges`) et action « Payer » ; `web/app/routes/app/member/dues.tsx` (contracts/ui-routes.md › `/a-regler` : total, lignes, bouton « Payer » vers `/adhesions/:id` ou `/mes-inscriptions/:id`, état vide « Rien à régler. ») ; `web/app/routes.ts` : `route("a-regler", …)`, retrait de `memberComingSoon` ; `web/app/shell/navigation.ts` : `comingSoon` retiré de `a-regler`
- [X] T057 [US4] Scénarios Playwright in `web/e2e/events.spec.ts` (« payer ») et `web/e2e/card-payment.spec.ts` (inscription par carte avec `fake-helloasso.ts`) : inscription payante, référence affichée, « À régler », annonce d'un chèque, paiement par carte et retour « Payée », mention « Mode test » ; axe et 320 px

**Checkpoint**: événements payants de bout en bout côté membre (US1 à US4) ; constatation manuelle encore absente côté administration.

---

## Phase 7: User Story 5 - Suivre les inscrits et les paiements d'un événement (Priority: P2)

**Goal**: l'administration voit les inscrits et les totaux, inscrit, retire, constate les paiements, exporte et recherche les inscriptions.

**Independent Test**: sur un événement payant avec des inscrits, ajouter une inscription marquée payée, constater un paiement annoncé, retirer une inscription, puis exporter la liste et vérifier son contenu et les totaux.

### Tests for User Story 5

- [X] T058 [P] [US5] Tests d'API in `internal/api/registrations_admin_test.go` : `list-event-registrations` (confirmées par défaut, `state=all`, `isSeasonMember`, `eligible` faux après passage en `season_members`, `contactEmail` nul pour un non joignable) ; `get-event.totals` (dû, encaissé, encaissé en test exclu, nombre payées) ; `register-member` hors période → `201`, avec `payment` → « Payée » ; complet → `409` ; non éligible → `422 body.memberId` ; `remove-registration` → `cancelled`, e-mail `registration-removed`, payée acceptée ; après la fin → `409` ; `record-registration-payment` → « Payée », e-mail `registration-paid` ; `cancel-registration-payment` manuelle → état précédent, carte → `409` ; `verify-registration-card-payment` ; `list-registrations` filtres `event`, `state`, `payment`, `alerts=open` (seules les inscriptions portant un signalement non traité), `q` « luc » (nom) et « EVT-26 » (référence), `q` de 2 caractères → `422`, `openAlerts` par ligne ; `get-registration` avec `onlinePayments` et `alerts` ; `resolve-payment-alert` d'un signalement d'inscription → disparaît de `alerts=open` ; `get-membership-dashboard` : `paymentAlerts` ne compte que les adhésions, `registrationPaymentAlerts` les inscriptions ; `get-member`/`get-my-member` → `paidUpcomingRegistrations` ; journaux sans nom
- [X] T059 [P] [US5] Test d'export et de volume in `internal/api/registrations_export_test.go` : **SC-007** : événement de 500 inscriptions confirmées (fiches et inscriptions insérées par `env.Store` en une transaction), `list-event-registrations`, `get-event` (totaux) et `export-event-registrations` répondent chacun en moins de 2 s (borne vérifiée par `time.Since`, test ignoré avec `testing.Short()`), export de 501 lignes (en-tête compris) ; `EXPLAIN QUERY PLAN` de `ListRegistrationsOfEvent` utilise `event_registrations_event` ; `export-event-registrations` → `text/csv; charset=utf-8`, BOM UTF-8, séparateur `;`, en-têtes de US5 §6, accents intacts, cellule `=1+1` saisie dans un nom préfixée d'une apostrophe, `Content-Disposition` `inscrits-<id>.csv` ; membre → `403`
- [X] T060 [P] [US5] Vérifier la matrice `internal/api/access_test.go` pour les 9 opérations admin de US5

### Implementation for User Story 5

- [X] T061 [US5] Opérations in `internal/api/registrations_admin.go`, `auth.Admin` (contracts/api.md) : `list-event-registrations` (`GET /admin/events/{id}/registrations`, `state`), `register-member` (`POST /admin/events/{id}/registrations`, `{ memberId, payment: RecordPaymentBody|null }`, `event.CanAdminRegister`, éligibilité, `CheckPlaces`, référence si payante, constatation immédiate si `payment`, e-mail `registration-confirmed`), `list-registrations` (`GET /admin/registrations`, recherche `FTSTrigramQuery` ≥ 3 caractères sinon `422 q`, filtre `alerts` `enum:"open"` et compteur `openAlerts` par ligne via `ListOpenAlertCountsByRegistration`, comme `alerts` de `list-memberships`, pagination comme `list-memberships`), `get-registration` (`GET /admin/registrations/{id}`, tentatives et signalements comme `withOnlineAdmin`), `remove-registration` (`POST /admin/registrations/{id}/cancel`, `event.CanRemove`, e-mail `registration-removed`), `record-registration-payment` et `cancel-registration-payment` (`POST`/`DELETE /admin/registrations/{id}/payment`, règles de 006 via `event.PaymentSnapshot`), `verify-registration-card-payment` (`POST /admin/registrations/{id}/card-payment/verify`, `ListVerifiableOnlinePaymentsOfRegistration` puis `verifyOnlinePayment`) ; `get-event.totals` via `RegistrationTotals` in `internal/api/events_admin.go`
- [X] T062 [US5] Export `export-event-registrations` in `internal/api/registrations_admin.go` (`GET /admin/events/{id}/registrations.csv`, R13) : `encoding/csv` avec `Comma = ';'`, BOM, colonnes nom, prénom, date de naissance, téléphone, adresse de contact, adhérent (oui/non), date d'inscription, montant dû, état du paiement, moyen, référence ; neutralisation des cellules commençant par `=`, `+`, `-`, `@` ; réponse huma en flux binaire avec `Content-Type` et `Content-Disposition` déclarés dans l'OpenAPI
- [X] T063 [US5] Opérations modifiées : `internal/api/payment_alerts.go` (`resolve-payment-alert` accepte un signalement d'inscription), tableau de bord des adhésions in `internal/api/memberships_admin.go` (`MembershipDashboard.paymentAlerts` via `CountOpenMembershipPaymentAlerts`, nouveau champ `registrationPaymentAlerts` via `CountOpenRegistrationPaymentAlerts`), `internal/api/members_admin.go` et `internal/api/members_me.go` (`paidUpcomingRegistrations` via `CountPaidUpcomingRegistrationsOfMember`)
- [X] T064 [P] [US5] Modèle d'e-mail `internal/mail/templates/registration-removed.txt.tmpl` et `.html.tmpl` (événement, membre, motif facultatif « remboursé », rappel que le remboursement se fait hors de Frigg si l'inscription était payée) ; cas in `internal/mail/render_test.go`
- [X] T065 [US5] Régénérer le contrat avec `go tool task gen`, fichiers commités
- [X] T066 [US5] Extraire de `web/app/routes/app/admin/membership-detail.tsx` les composants `web/app/payments/RecordPaymentDialog.tsx`, `web/app/payments/OnlinePayments.tsx`, `web/app/payments/PaymentAlerts.tsx` ; `membership-detail.tsx` les utilise sans changement visible ; `web/e2e/admin-memberships.spec.ts` reste vert
- [X] T067 [P] [US5] Requêtes et mutations in `web/app/api/registrations.ts` : `eventRegistrationsQuery(id, state)`, `registrationsQuery(filters)`, `registrationQuery(id)`, `useRegisterMember`, `useRemoveRegistration`, `useRecordRegistrationPayment`, `useCancelRegistrationPayment`, `useVerifyRegistrationCardPayment`, `exportEventRegistrations(id)` (client `openapi-fetch`, `parseAs: "blob"`, téléchargement par lien objet) ; invalidation de `get-event`, `list-event-registrations`, `list-registrations`, `get-registration`, `get-membership-dashboard`
- [X] T068 [US5] Écran `web/app/routes/app/admin/event-detail.tsx` complété : section Chiffres (inscrits / capacité, total dû, total encaissé, « dont mode test »), section Inscrits (bascule « Afficher les annulées », `Table` nom, prénom, adhérent, badge `--warn` « Non éligible », date, inscrit par, paiement, actions « Constater le paiement » `RecordPaymentDialog` et « Retirer » `ConfirmDialog` avec rappel de remboursement si payée ; « non joignable » signalé), boutons « Inscrire un membre » (`web/app/events/RegisterMemberDialog.tsx` : `MemberPicker`, « Déjà payé » + moyen et date si payant) et « Exporter (CSV) »
- [X] T069 [US5] Écrans `web/app/routes/app/admin/registrations.tsx` (filtres événement `Combobox`, état, paiement, case « Signalements à traiter » (`alerts=open`, paramètre d'URL) ; recherche avec message « 3 caractères minimum » ; `Table` paginée avec badge `--danger` « n signalement(s) » si `openAlerts > 0`) ; encart du tableau de bord in `web/app/routes/app/admin/home.tsx` : un lien par cible non nulle, « n signalement(s) sur les adhésions » vers `/admin/adhesions?season=all&alerts=open` et « n signalement(s) sur les inscriptions » vers `/admin/inscriptions?alerts=open` et `web/app/routes/app/admin/registration-detail.tsx` (calqué sur `membership-detail.tsx` avec `payments/*`, « Vérifier auprès de HelloAsso », « Retirer l'inscription ») ; `web/app/routes.ts` : `route("inscriptions", …, [index, route(":id")])` sous `admin`, retrait de `adminComingSoon` ; `web/app/shell/navigation.ts` : `comingSoon` retiré de `inscriptions` ; dialogues de suppression de fiche in `web/app/routes/app/admin/member-detail.tsx` et `web/app/routes/app/member/my-member-detail.tsx` : mention « n inscription(s) payée(s) à un événement à venir » si `paidUpcomingRegistrations > 0`
- [X] T070 [US5] Scénario Playwright `web/e2e/admin-events.spec.ts` : inscrits et totaux, inscription manuelle « Déjà payé — espèces », constatation d'un chèque, retrait avec rappel de remboursement, export CSV téléchargé, recherche « EVT-26 » dans « Inscriptions », signalement d'une inscription (paiement par carte reçu sur une inscription annulée via `fake-helloasso.ts`) visible depuis l'encart du tableau de bord puis traité ; axe et 320 px

**Checkpoint**: gestion complète des inscrits et des paiements par l'administration.

---

## Phase 8: User Story 6 - Annuler ou supprimer un événement (Priority: P2)

**Goal**: l'administration annule (avec message) ou rétablit un événement en prévenant les inscrits, et supprime un événement sans inscription.

**Independent Test**: annuler un événement publié avec des inscrits payés et non payés, vérifier les e-mails en file, l'affichage côté membre et la disparition de « À régler » ; rétablir ; supprimer un brouillon sans inscription.

### Tests for User Story 6

- [X] T071 [P] [US6] Tests d'API in `internal/api/events_lifecycle_test.go` : `cancel-event` avec message → `cancelled`, inscriptions conservées, **un** e-mail `event-cancelled` par adresse de contact (Claire avec deux fiches : un seul), message repris ; message de 1 001 caractères → `422 body.message` ; brouillon ou terminé → `409` ; après annulation : `list-my-dues` sans ces inscriptions, `announce-my-registration-payment` → `409`, paiement par carte reçu → signalement `refund_due` ; `restore-event` → `published`, paiements intacts, e-mail `event-restored` ; après la fin → `409` ; `delete-event` d'un brouillon → `204` ; avec une inscription, même annulée → `409` « Cet événement a des inscriptions : annulez-le plutôt. » ; journaux `event.cancel|restore|delete` avec `notified`, sans message
- [X] T072 [P] [US6] Vérifier la matrice `internal/api/access_test.go` pour `cancel-event`, `restore-event`, `delete-event`

### Implementation for User Story 6

- [X] T073 [US6] Opérations in `internal/api/events_admin.go`, `auth.Admin` : `cancel-event` (`POST /admin/events/{id}/cancel`, `{ message: string|null }` `maxLength:"1000"`, `event.CanCancel`, `CancelEvent` `:execrows`, `notifyEventRegistrants("event-cancelled", Message)`), `restore-event` (`POST /admin/events/{id}/restore`, `event.CanRestore`, `notifyEventRegistrants("event-restored")`), `delete-event` (`DELETE /admin/events/{id}`, `204`, `DeleteEventWithoutRegistrations` = 0 ligne → `409`) ; `get-event` : nombre et montant des inscriptions payées pour le dialogue d'annulation
- [X] T074 [P] [US6] Modèles d'e-mail `internal/mail/templates/event-cancelled.*.tmpl` (message de l'association, inscrits, mention « l'association reviendra vers vous pour le remboursement » si une inscription du destinataire est payée) et `internal/mail/templates/event-restored.*.tmpl` (date, lieu, inscrits) ; cas in `internal/mail/render_test.go`
- [X] T075 [US6] Régénérer le contrat avec `go tool task gen`, fichiers commités
- [X] T076 [P] [US6] Mutations in `web/app/api/events.ts` : `useCancelEvent`, `useRestoreEvent`, `useDeleteEvent` (invalidation de `list-events`, `get-event`, `list-my-events`, `get-my-event`, `list-my-dues`, `list-my-registrations`)
- [X] T077 [US6] Menu « Plus » base-ui in `web/app/routes/app/admin/event-detail.tsx` : « Annuler l'événement » (`ConfirmDialog` avec `textarea` « Message aux inscrits (facultatif) », nombre de personnes prévenues, encart « n inscriptions payées (total) : le remboursement se fait hors de Frigg ou dans l'espace HelloAsso »), « Rétablir », « Supprimer » (seulement sans inscription, retour à la liste après succès)
- [X] T078 [US6] Scénario Playwright in `web/e2e/admin-events.spec.ts` (« annuler et supprimer ») : annulation avec message, e-mail reçu, badge « Annulé » côté membre et « À régler » vidé, rétablissement, suppression refusée puis suppression d'un brouillon ; axe et 320 px

**Checkpoint**: toutes les user stories livrées.

---

## Phase 9: Polish & Cross-Cutting Concerns

- [X] T079 [P] Données de démonstration in `internal/tools/e2eseed/main.go` : une assemblée générale gratuite réservée aux adhérents, un stage payant ouvert à toutes les fiches avec capacité, quelques inscriptions dont une payée par chèque
- [X] T080 [P] Revue de sécurité ciblée (plan.md › points de vigilance 1 à 5) : toutes les requêtes `/me` filtrent `status <> 'draft'` et par compte ; aucun `slog` des nouveaux fichiers ne journalise titre, lieu, description, message, nom ni adresse (`grep -n "Logger\." internal/api/event*.go internal/api/registration*.go internal/api/card_payments.go`) ; aucun test de rôle (`CurrentAccount(ctx).Role`) ajouté ; aucun SQL construit par concaténation ; intitulé HelloAsso sans nom du membre ; corriger le cas échéant
- [X] T081 [P] Documentation utilisateur in `README.md` (ou documentation d'utilisation existante) : rubriques « Événements » et « Inscriptions » (ouverture, capacité sans liste d'attente, prix et moyens de paiement, annulation et remboursements hors de Frigg) et « À régler » côté membre
- [X] T082 [P] Ajouter à la section Assumptions de `specs/006-memberships/spec.md` et `specs/008-helloasso-card-payment/spec.md` une note renvoyant à la spec 012 (moyens de paiement et paiement par carte communs aux inscriptions)
- [X] T083 Exécuter `go tool task check` (build `CGO_ENABLED=0`, `go vet`, `go test ./...`, `sqlc generate` et `frigg openapi` sans diff, migrations sur base vierge, `bun run typecheck`, Biome, Vitest, `bun run build`, Playwright + axe) et dérouler quickstart.md › scénarios manuels 1 à 6

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance. T001 avant T002, T003, T004 ; T005 après T003 et T004 (un seul `sqlc generate`) ; T006 indépendante.
- **Foundational (Phase 2)** : dépend de la phase 1. T007, T008, T009, T010 en parallèle (T008 et T009 utilisent les types de T007 : écrire T007 d'abord si un seul développeur) ; T011 après T005 et T006 ; T012 après T011 ; T013 et T014 en parallèle de T011. Bloque toutes les stories.
- **US1 (Phase 3)** : dépend de la phase 2. MVP.
- **US2 (Phase 4)** : dépend de la phase 2 ; côté API, testable sans US1 (événements insérés par `env.Store`) ; écrans après les composants de US1 (T023).
- **US3 (Phase 5)** : dépend de US2 (`get-my-event`, page de l'événement).
- **US4 (Phase 6)** : dépend de US3 (inscription) et de T011 (cible).
- **US5 (Phase 7)** : dépend de US3 ; les tests de paiement de US5 supposent US4.
- **US6 (Phase 8)** : dépend de US1 ; ses tests de paiement supposent US4.
- **Polish (Phase 9)** : après les stories voulues.

### User Story Dependencies

- **US1 (P1)** : aucune autre story.
- **US2 (P1)** : backend indépendant ; frontend après US1 (composants d'événement).
- **US3 (P1)** : après US2.
- **US4 (P1)** : après US3.
- **US5 (P2)** : après US3 (US4 pour la partie paiement).
- **US6 (P2)** : après US1 (US4 pour les vérifications de paiement).

### Within Each User Story

- Tests d'abord (ils doivent échouer), puis domaine, puis requêtes et handlers, puis régénération du contrat, puis frontend, puis Playwright.

### Parallel Opportunities

- Phase 1 : T002, T003, T004 en parallèle après T001 ; T006 à tout moment.
- Phase 2 : T007, T008, T009, T010 en parallèle ; T013 et T014 en parallèle de T011.
- US1 : T015, T016, T017 en parallèle ; T020 en parallèle des handlers ; T022 et T023 en parallèle après T021.
- US2 : T027, T028 en parallèle ; backend de US2 en parallèle du frontend de US1.
- US3 : T034, T035, T036 en parallèle ; T039 en parallèle des handlers ; T041 après T040.
- US4 : T045, T046, T047 en parallèle ; T052 en parallèle ; T054 (extraction frontend) peut commencer dès la phase 2.
- US5 : T058, T059, T060 en parallèle ; T064 en parallèle ; T066 (extraction) peut commencer dès la phase 2.
- US6 : backend (T071–T075) en parallèle de US5 une fois US4 livrée.

---

## Parallel Example: Phase 2

```text
T007 internal/event/event.go
T008 internal/event/registration.go
T009 internal/event/payment.go + reference.go
T010 internal/event/*_test.go
T013 squelette internal/api/events_*.go, registrations_*.go
T014 internal/api/event_notices.go
```

## Parallel Example: User Story 4

```text
# Tests :
T045 internal/api/registration_payments_test.go
T046 internal/api/registration_card_payments_test.go
T047 internal/api/access_test.go

# En parallèle des handlers :
T052 internal/mail/templates/registration-paid.*
T054 web/app/payments/PaymentPanel.tsx (extraction)
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 (migration de reconstruction, requêtes, métadonnées) puis phase 2
   (domaine, cible de paiement par carte, squelettes, e-mails regroupés).
2. US1 : création et publication.
3. **STOP et valider** : test de migration sur base peuplée, tests d'API,
   Playwright ; les paiements des adhésions fonctionnent toujours.

US1 seule n'a pas d'effet visible pour les membres : la première livraison
utile est **US1 + US2 + US3** (événements gratuits avec inscription). Les
événements payants arrivent avec **US4**.

### Incremental Delivery

1. US1 → agenda géré par l'administration.
2. US2 → agenda visible des membres.
3. US3 → inscriptions gratuites.
4. US4 → inscriptions payantes et « À régler ».
5. US5 → suivi des inscrits, constatation des paiements, export.
6. US6 → annulation, rétablissement, suppression.
7. Phase 9 → données de démonstration, revue de sécurité, documentation, contrôles complets.

### Notes

- Chaque tâche se termine par ses tests verts et un commit (gitmoji, message
  court, sans co-auteur, après validation de l'utilisateur).
- Après chaque ajout ou modification d'opération : `frigg openapi` et
  `bun run gen:api` avant le frontend ; le contrôle « sans diff » de la
  constitution l'exige.
- La migration `00012` est la tâche la plus risquée : ne pas commencer les
  stories avant que T002 soit vert, et ne jamais la modifier après un
  déploiement.
- T011 est un refactor sans changement de comportement : l'exécuter seul,
  suite complète verte avant et après.
