---

description: "Liste des tâches : remises et dons sur les paiements"
---

# Tasks: Remises et dons sur les paiements

**Input**: Design documents from `specs/014-payment-adjustments/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : matrice d'autorisation (principes II et III), tests
d'API sur base isolée, test de migration sur base peuplée, Playwright + axe
sur chaque parcours touché.

**Organization**: une phase par user story, dans l'ordre de la spec. US1 (P1)
livre la remise à la validation et à la création d'une adhésion, US2 (P1) le
montant reçu et le don sur les adhésions, US3 (P2) l'ajustement d'une
adhésion déjà validée, US4 (P2) les remises et dons des inscriptions, US5
(P3) les chiffres.

**Conventions** (constitution 1.8.0) : identifiants, colonnes et champs JSON
en anglais (`base_amount_cents`, `adjustment_reason`, `receivedAmount`) ;
commentaires, messages, textes d'interface, e-mails et journaux en français.
Commentaires de la migration et des requêtes sqlc en **ASCII uniquement**
(sqlc sqlite casse sur un commentaire non ASCII). Le **texte du motif n'est
jamais journalisé** (R7) et n'est **jamais renvoyé à un compte membre**
(R9). Le frontend n'analyse ni ne compare les montants : il envoie le texte
saisi, et le serveur l'analyse avec `season.ParseAmount` (R4). Aucun test de
rôle dans les handlers : le rôle est déclaré par `register`. Toutes les
écritures en `s.Store.WriteTx`, e-mails insérés dans la même transaction.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US5)

## Path Conventions

Module Go à la racine (`internal/`), SPA dans `web/` (plan.md › Project
Structure). Code existant à réutiliser :

- Domaine : `internal/membership/membership.go` (`PaymentAfter`,
  `CleanReason`, `cleanText`, `ManualMethod`, `Card`, `CanUpdate`,
  `CanRecordPayment`), `internal/membership/stats.go` (`StatRow`,
  `ComputeStats`), `internal/event/payment.go` (`PaymentSnapshot`),
  `internal/season/fee.go` (`ParseAmount`, `MaxAmountCents`,
  `ErrAmountFormat`, `ErrAmountTooHigh`).
- API : `internal/api/memberships_admin.go` (`DecisionBody`, `ValidateBody`,
  `AdminCreateBody`, `RecordPaymentBody`, `decide`, `validateMembership`,
  `updateMembership`, `createMembership`, `checkPayment`,
  `recordMembershipPayment`, `getMembershipStats`, `adminMembership`),
  `internal/api/memberships.go` (`Payment`, `paymentOf`, `paymentColumns`,
  `notifyMembership`, `logMembershipAction`, `formatEuros`),
  `internal/api/registrations_admin.go` (`AdminRegistration`,
  `AdminRegistrationActions`, `toAdminRegistration`, `registerMember`,
  `adminRegistrationWrite`, `recordRegistrationPayment`,
  `exportEventRegistrations`), `internal/api/events_me.go`
  (`createRegistrations`), `internal/api/events_admin.go` (`EventTotals`,
  `addTotals`), `internal/api/registration_target.go`
  (`notifyRegistrationPaid`), `internal/api/membership_fees.go` (traduction
  des erreurs de `ParseAmount` en messages de champ).
- Tests : `internal/database/db_test.go` (`TestEventsMigration` : modèle de
  test `UpTo`/`DownTo` sur base peuplée), `internal/api/access_test.go`
  (`TestAccessMatrix`), `internal/api/memberships_helpers_test.go`,
  `internal/api/events_helpers_test.go`, `internal/api/apitest`, `hatest`
  (faux HelloAsso), `internal/mail/render_test.go`.
- Frontend : `web/app/memberships/FeeChoice.tsx` (`AmountSummary`),
  `web/app/payments/AdminPayments.tsx` (`RecordPaymentDialog`,
  `DialogShell`), `web/app/lib/format` (`formatAmount`, `formatMoney`),
  `web/e2e/axe.ts`.

---

## Phase 1: Setup

**Purpose**: partir d'un état vert.

- [X] T001 Vérifier que `go tool task check` passe sur `main` avant toute modification (référence pour les régressions)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: schéma, requêtes et domaine communs à toutes les stories.

**⚠️ CRITICAL**: aucune story ne commence avant la fin de cette phase.

- [X] T002 Créer `internal/database/migrations/00013_payment_adjustments.sql` (goose, commentaires ASCII). **Up** : sur `memberships`, `ALTER TABLE memberships ADD COLUMN base_amount_cents INTEGER CHECK (base_amount_cents IS NULL OR (base_amount_cents BETWEEN 0 AND 1000000 AND amount_cents IS NOT NULL AND base_amount_cents <> amount_cents))`, puis `ALTER TABLE memberships ADD COLUMN adjustment_reason TEXT CHECK ((adjustment_reason IS NULL) = (base_amount_cents IS NULL)) CHECK (adjustment_reason IS NULL OR length(adjustment_reason) BETWEEN 1 AND 200)`. Sur `event_registrations`, mêmes colonnes avec `CHECK (base_amount_cents IS NULL OR (base_amount_cents BETWEEN 0 AND 1000000 AND base_amount_cents <> amount_cents))` et les deux mêmes `CHECK` sur `adjustment_reason`. Puis la reprise : `UPDATE memberships SET paid_amount_cents = amount_cents WHERE payment_status = 'paid' AND paid_amount_cents IS NULL;` et la même requête sur `event_registrations`. **Down** : pour chaque table, `UPDATE … SET paid_amount_cents = NULL WHERE payment_status = 'paid' AND paid_method <> 'card'`, puis `DROP COLUMN adjustment_reason` **avant** `DROP COLUMN base_amount_cents` (ordre obligatoire, data-model.md › Down)
- [X] T003 Ajouter `TestPaymentAdjustmentsMigration` dans `internal/database/db_test.go`, sur le modèle de `TestEventsMigration` : `UpTo(12)`, puis insérer une adhésion payée en espèces, une adhésion payée par carte (avec `online_payments` et `paid_amount_cents` différent du montant), une adhésion validée due, une inscription payée par virement et une inscription due ; puis `UpTo(13)`. Vérifier : chaque paiement manuel a `paid_amount_cents = amount_cents`, la carte garde son montant, les lignes non payées restent à `NULL`. Vérifier que les `CHECK` refusent un `adjustment_reason` sans `base_amount_cents`, un `base_amount_cents` égal à `amount_cents` et un motif de 201 caractères. Enfin `DownTo(12)` : les colonnes ont disparu, et `paid_amount_cents` est `NULL` pour les paiements manuels mais conservé pour la carte
- [X] T004 Modifier `internal/database/queries/memberships.sql` (commentaires ASCII) : ajouter `base_amount_cents = sqlc.narg(base_amount_cents)` et `adjustment_reason = sqlc.narg(adjustment_reason)` à `CreateMembership` (colonnes et valeurs), `ValidateMembership` et `UpdateValidatedMembership` ; ajouter `paid_amount_cents = sqlc.arg(paid_amount_cents)` à `RecordMembershipPayment` ; ajouter `paid_amount_cents = NULL` à `CancelMembershipPayment` ; ajouter `ms.base_amount_cents` au `SELECT` de `ListSeasonMembershipsForStats`
- [X] T005 [P] Modifier `internal/database/queries/event_registrations.sql` (commentaires ASCII) : `base_amount_cents` et `adjustment_reason` (narg) dans `CreateRegistration` ; nouvelle requête `-- name: AdjustRegistrationAmount :execrows` qui met à jour `amount_cents`, `base_amount_cents`, `adjustment_reason`, `payment_reference`, `payment_status`, `announced_method` et `updated_at`, avec `WHERE id = sqlc.arg(id) AND status = 'confirmed' AND payment_status IS NOT 'paid'` ; `paid_amount_cents = sqlc.arg(paid_amount_cents)` dans `RecordRegistrationPayment` ; `paid_amount_cents = NULL` dans `CancelRegistrationPayment`
- [X] T006 Régénérer sqlc (`go tool task gen` ou `sqlc generate`) et corriger les appels Go qui ne compilent plus (`internal/api/*.go`, `internal/tools/e2eseed/main.go`) en passant pour l'instant `BaseAmountCents`/`AdjustmentReason` vides et `PaidAmountCents: nid(<montant à payer>)` dans les constatations manuelles, afin que `go build ./...` et `go test ./...` passent sans changer le comportement
- [X] T007 [P] Créer `internal/membership/amount.go` (R3) : constante `MaxAdjustmentReason = 200` ; erreurs `ErrAdjustmentReasonRequired`, `ErrAdjustmentReasonTooLong`, `ErrReceivedBelowDue` ; type `Adjustment { AmountCents int64; BaseCents *int64; Reason string }` ; `Adjust(base int64, requested *int64, reason string) (Adjustment, error)` (requête nil ou égale à base → `{AmountCents: base}` sans base ni motif ; sinon motif nettoyé par `cleanText`, vide → `ErrAdjustmentReasonRequired`, plus de 200 runes → `ErrAdjustmentReasonTooLong`, sinon `{AmountCents: *requested, BaseCents: &base, Reason}`) ; `Theoretical(amount int64, base *int64) int64` ; `CheckReceived(due, received int64) error` ; `Donation(m Method, due, received int64) int64` (`received − due` si `ManualMethod(m)` et `received > due`, sinon 0) ; types `AmountRow` (`AmountCents`, `BaseCents *int64`, `PaymentStatus`, `AnnouncedMethod`, `PaidMethod`, `PaidAmountCents`, `PaidTest`) et `AmountTotals` (`Theoretical`, `Discounts`, `Surcharges`, `Due`, `Collected`, `CollectedByMethod PaidByMethod`, `CollectedTest`, `Donations`, `Remaining`, `RemainingByMethod DueByMethod`) avec `(*AmountTotals).Add(AmountRow)` qui suit le tableau « AmountTotals » de data-model.md (le montant reçu d'une ligne payée est `PaidAmountCents`, sans repli : la reprise de T002 et les constatations garantissent qu'une ligne payée a toujours un montant reçu, R2) ; ajouter aussi `Adjustment(amount int64, base *int64) int64` (`amount − *base`, 0 si base nil)
- [X] T008 [P] Créer `internal/membership/amount_test.go` : tests en table de `Adjust` (absent, égal avec motif ignoré, remise, majoration, 0 €, motif vide, motif d'espaces, 200 et 201 runes), `CheckReceived`, `Donation` (espèces avec don, virement sans don, carte avec écart → 0) et `AmountTotals.Add` sur un jeu mixte (remise, majoration, gratuit par ajustement, payée avec don, carte en mode test, due annoncée par chèque). Vérifier l'identité `Theoretical − Discounts + Surcharges == Due`
- [X] T009 Refaire `ComputeStats` dans `internal/membership/stats.go` sur `AmountTotals` : `StatRow` reçoit `BaseCents *int64` ; les champs actuels de `Stats` (`TotalCents` = `Due`, `PaidCents` = `Collected`, `DueCents` = `Remaining`, `DueByMethod`, `PaidByMethod`, `TestPaidCents`) gardent leur valeur ; ajouter `TheoreticalCents`, `DiscountCents`, `SurchargeCents`, `DonationCents` ; `FeeStat.AmountCents` cumule le montant à payer. `internal/membership/stats_test.go` doit rester vert, et recevoir un cas avec remise et don
- [X] T010 [P] Ajouter dans `internal/api/errors.go` : `MsgAdjustmentReasonRequired = "Indiquez le motif de l'ajustement."`, `MsgAdjustmentReasonTooLong = "Le motif ne doit pas dépasser 200 caractères."`, `MsgReceivedBelowDue = "Le montant reçu ne peut pas être inférieur au montant à payer. Ajustez d'abord le montant à payer."`. Créer dans `internal/api/payment_amounts.go` les helpers partagés : `parseOptionalAmount(text, location string) (*int64, error)` (vide → nil ; erreurs de `season.ParseAmount` traduites en `FieldErrors(location, …)` avec les mêmes messages que les tarifs dans `membership_fees.go`) ; `adjustmentError(err error, location string) error` (traduit `ErrAdjustmentReasonRequired`/`TooLong` en 422 sur `<location>.adjustmentReason`) ; `receivedAmount(body RecordPaymentBody, location string, due int64) (int64, error)` (vide → due ; sinon `ParseAmount`, puis `CheckReceived` → 422 sur `<location>.receivedAmount` avec `MsgReceivedBelowDue`)
- [X] T011 [P] Ajouter `Donation string` à `mail.Data` dans `internal/mail/render.go`, puis dans `internal/mail/templates/membership-paid.txt.tmpl`, `membership-paid.html.tmpl`, `registration-paid.txt.tmpl` et `registration-paid.html.tmpl`, la ligne `{{if .Donation}}Dont {{.Donation}} de don : merci pour votre soutien !{{end}}` sous « Montant » (en `<br>` dans le html). Étendre `TestRenderMembershipTemplates` et `TestRenderEventTemplates` dans `internal/mail/render_test.go` (avec et sans don)
- [X] T012 Ajouter `DonationCents int64 \`json:"donationCents" doc:"Don : excédent du montant reçu sur le montant à payer"\`` au type partagé `Payment` dans `internal/api/memberships.go`. `paymentOf` le calcule avec `membership.Donation`. Faire de même pour le `Payment` des inscriptions, qui est construit dans `internal/api/registrations_admin.go`/`registrations_me.go`. Mettre à jour la doc de `PaidAmountCents` : « Montant reçu ».

**Checkpoint**: `go tool task check` vert ; la migration et le domaine sont testés ; le comportement visible n'a pas changé.

---

## Phase 3: User Story 1 - Accorder une remise à la validation d'une adhésion (Priority: P1) 🎯 MVP

**Goal**: l'administrateur fixe le montant à payer, avec un motif, à la validation et à la création d'une adhésion ; le membre et HelloAsso ne voient que ce montant.

**Independent Test**: valider une demande avec 70 € au lieu de 75 € et un motif. Le membre voit 70 € dans « À régler », dans l'e-mail et sur la carte ; l'administration voit le montant théorique et le motif ; la réponse membre ne contient pas le motif.

### Tests for User Story 1

- [X] T013 [P] [US1] Créer `internal/api/membership_adjustments_test.go` (httptest, base isolée) : validation sans `amount` → montant du tarif, aucun ajustement ; validation `amount: "70"` sans motif → 422 `body.adjustmentReason` ; avec motif → `amountCents 7000`, `baseAmountCents 7500`, `adjustmentReason` ; `amount: "75"` avec motif → pas d'ajustement ; `amount: "0"` avec motif → `payment` null, pas de référence ; `amount: "abc"` → 422 `body.amount` ; création admin avec ajustement et `paid` ; e-mail de validation en file au montant ajusté ; journal sans le texte du motif (capturer le logger, vérifier `adjusted=true` et l'absence du motif)
- [X] T014 [P] [US1] Dans le même fichier, test d'isolation : `GET /api/me/memberships/{id}` et la liste « À régler » d'un compte membre, en réponse brute, ne contiennent ni `baseAmountCents`, ni `adjustmentCents`, ni `adjustmentReason`, et donnent `amountCents 7000`. Une demande `POST /api/me/memberships` qui envoie un champ `amount` est refusée (422) ou sans effet : après validation sans `amount`, le montant est celui du tarif (FR-008). Test de paiement par carte avec `hatest` : l'intention HelloAsso porte 7000, et la constatation ne crée pas de signalement `amount_mismatch`

### Implementation for User Story 1

- [X] T015 [US1] Dans `internal/api/memberships_admin.go`, ajouter à `DecisionBody` et `AdminCreateBody` les champs `Amount string \`json:"amount,omitempty" required:"false" maxLength:"12" doc:"Montant à payer en euros ; vide : montant théorique"\`` et `AdjustmentReason string \`json:"adjustmentReason,omitempty" required:"false" maxLength:"200"\``. Faire renvoyer `membership.Adjustment` par `decide` (montant théorique = `membership.AppliedAmount`, puis `parseOptionalAmount(body.Amount, "body.amount")` et `membership.Adjust`, erreurs traduites par `adjustmentError(err, "body")`)
- [X] T016 [US1] Brancher l'ajustement dans `validateMembership` et `createMembership` (`internal/api/memberships_admin.go`) : `AmountCents` = montant à payer, `BaseAmountCents` et `AdjustmentReason` transmis aux requêtes, `paymentColumns` appelé avec le montant à payer. Dans `createMembership`, le bloc `paid` utilise `receivedAmount(*in.Body.Paid, "body.paid", amount)` pour `PaidAmountCents`. Journaux : ajouter `"base_amount_cents"` (montant théorique) et `"adjusted"` aux actions `membership.validate` et `membership.create`, jamais le motif
- [X] T017 [US1] Ajouter à `AdminMembership` (`internal/api/memberships_admin.go`) `BaseAmountCents int64 \`json:"baseAmountCents" doc:"Montant théorique"\`` et `AdjustmentCents int64 \`json:"adjustmentCents" doc:"Montant à payer moins montant théorique : négatif = remise, positif = majoration"\`` et `AdjustmentReason *string \`json:"adjustmentReason"\``, renseignés dans `adminMembership` (`membership.Theoretical`, `membership.Adjustment`) ; ajouter `BaseAmountCents *int64` (null tant que l'adhésion n'est pas validée) et `AdjustmentCents int64` à `AdminMembershipRow`, renseignés dans `listMemberships`. Ne rien ajouter au type partagé `Membership` (R9)
- [X] T018 [US1] Régénérer l'OpenAPI et les types frontend (`go tool task gen` : `frigg openapi` puis `bun run gen:api`), qui mettent à jour `web/app/api/openapi.json` et `web/app/api/schema.d.ts`
- [X] T019 [P] [US1] Créer `web/app/payments/useTrackedAmount.ts` : hook qui reçoit `theoreticalCents` et une valeur initiale `{ amountCents, adjustmentReason }`. Il gère `text`, `reason` et `tracked` : suivi au départ s'il n'y a pas de motif initial ; tant qu'il est suivi, `text` = `formatAmount` du théorique sans le symbole (« 75,00 ») et se recalcule quand `theoreticalCents` change ; il n'est plus suivi après une saisie ; `reset()` rétablit le suivi. Il expose `payload()`, qui renvoie `{}` quand le champ est suivi, sinon `{ amount: text, adjustmentReason: reason }`. Créer `web/app/payments/useTrackedAmount.test.ts` (Vitest : recalcul, arrêt après saisie, `reset`, valeur initiale ajustée)
- [X] T020 [US1] Créer `web/app/payments/AmountToPayField.tsx` selon contracts/ui-routes.md › AmountToPayField : champ « Montant à payer » (`inputMode="decimal"`), aide « Montant théorique : X € » reliée par `aria-describedby`, bouton « Revenir au montant théorique » (vrai `<button>`, visible seulement si le champ n'est pas suivi, qui rend le focus au champ), zone « Motif de l'ajustement » (200 caractères, compteur) visible si le champ n'est pas suivi et annoncée par `aria-live="polite"`, erreurs 422 `amount` et `adjustmentReason` sur les champs. Composants `ui/*` et tokens seulement
- [X] T021 [US1] Dans `web/app/routes/app/admin/membership-detail.tsx` › `DecisionDialog`, remplacer `AmountSummary` par `AmountToPayField` (théorique = `fee.amounts[kind]`), et fusionner `payload()` dans le corps de `validate-membership`. Dans `MembershipView`, bloc « Montant » : le montant à payer ; s'il y a un ajustement (`adjustmentCents !== 0`), la ligne « Théorique X € · Remise Y € » (`adjustmentCents < 0`) ou « Majoration Y € », avec `baseAmountCents` et la valeur absolue de `adjustmentCents` fournis par l'API (aucun calcul de montant côté client), puis le motif
- [X] T022 [US1] Dans `web/app/routes/app/admin/membership-new.tsx`, remplacer `AmountSummary` par `AmountToPayField` sous le choix du tarif et du type, et fusionner `payload()` dans le corps de `create-membership`

**Checkpoint**: US1 livrable seule : remise à la validation et à la création, visible côté administration, appliquée côté membre et carte.

---

## Phase 4: User Story 2 - Constater un paiement supérieur au montant à payer (Priority: P1)

**Goal**: saisir le montant reçu à la constatation d'un paiement d'adhésion, l'excédent étant un don ; refuser un montant inférieur.

**Independent Test**: constater 80 € sur une adhésion à 45 € → payée, don de 35 €, e-mail avec le don ; 40 € → 422 ; annuler la constatation → montant reçu effacé.

### Tests for User Story 2

- [X] T023 [P] [US2] Créer `internal/api/membership_payments_received_test.go` : constatation sans `receivedAmount` → `paidAmountCents` = montant à payer, `donationCents 0` ; `receivedAmount: "80"` sur 45 € → `paidAmountCents 8000`, `donationCents 3500`, e-mail `membership-paid` en file avec « 80,00 € » et « 35,00 € de don » ; `receivedAmount: "40"` → 422 `body.receivedAmount` ; annulation de la constatation → `paidAmountCents` null en base ; réponse membre de l'adhésion avec `donationCents 3500` ; journal avec `received_cents` et `donation_cents`

### Implementation for User Story 2

- [X] T024 [US2] Ajouter à `RecordPaymentBody` (`internal/api/memberships_admin.go`) `ReceivedAmount string \`json:"receivedAmount,omitempty" required:"false" maxLength:"12" doc:"Montant reçu en euros ; vide : montant à payer"\``. Dans `recordMembershipPayment`, appeler `receivedAmount(in.Body, "body", w.m.AmountCents.Int64)` et passer `PaidAmountCents` à `RecordMembershipPayment`. Journal `membership.payment_record` : ajouter `"received_cents"` et `"donation_cents"`
- [X] T025 [US2] Dans `notifyMembership` (`internal/api/memberships.go`), renseigner `data.Donation = formatEuros(don)` quand `membership.Donation(...) > 0`
- [X] T026 [US2] Régénérer l'OpenAPI et les types frontend (`go tool task gen`)
- [X] T027 [US2] Dans `web/app/payments/AdminPayments.tsx` › `RecordPaymentDialog`, ajouter la prop `amountCents` et le champ « Montant reçu » (prérempli avec `formatAmount(amountCents)` sans symbole, aide « Un montant supérieur au montant à payer est enregistré comme un don. », transmis en `receivedAmount` seulement s'il a été modifié, erreur 422 `receivedAmount` sur le champ). Passer `amountCents` depuis `web/app/routes/app/admin/membership-detail.tsx` › `PaymentDialog`
- [X] T028 [US2] Ajouter « Montant reçu » au bloc « déjà payé » de `web/app/routes/app/admin/membership-new.tsx` (`paid.receivedAmount`). Tant que l'administrateur n'a pas modifié « Montant reçu », ce champ recopie le texte courant de `AmountToPayField` (`text` de `useTrackedAmount`, donc le théorique formaté si le champ est suivi) ; dès qu'il est modifié, il garde sa valeur. Il n'est transmis que s'il a été modifié, et le serveur applique sinon le montant à payer
- [X] T029 [US2] Afficher le paiement reçu et le don : dans `web/app/routes/app/admin/membership-detail.tsx`, « Reçu X € » et, si `donationCents > 0`, « dont Y € de don » pour tout moyen (la ligne actuelle de `paidAmountCents` n'était affichée que pour la carte) ; dans `web/app/routes/app/member/membership-detail.tsx`, « Reçu X € » et « dont Y € de don, merci ! ». Aucun montant théorique ni motif côté membre

**Checkpoint**: US1 et US2 fonctionnent indépendamment. Le MVP couvre les cas réels de remise et de don sur les adhésions.

---

## Phase 5: User Story 3 - Ajuster le montant d'une adhésion déjà validée (Priority: P2)

**Goal**: ajuster, ou retirer l'ajustement, d'une adhésion validée non payée via la correction existante.

**Independent Test**: corriger une adhésion validée non payée avec un nouveau montant et un motif ; revenir au théorique ; passer à 0 € puis revenir à un montant positif avec la même référence ; une adhésion payée renvoie 409.

### Tests for User Story 3

- [X] T030 [P] [US3] Ajouter à `internal/api/membership_adjustments_test.go` : `update-membership` avec `amount` et motif → ajustement enregistré, référence conservée ; changement de tarif avec le même `amount` → montant conservé, `baseAmountCents` recalculé ; `amount` vide → ajustement et motif effacés ; passage à 0 € → `payment` null et référence conservée en base, puis retour à 75 € → `payment.status due` avec la même référence ; paiement annoncé puis ajustement → annonce conservée ; adhésion payée → 409 `MsgPaidCorrection` ; adhésion non payée d'une saison close → 409 `MsgSeasonClosedAction` (règle existante, spec › Saison close), alors que la constatation d'un paiement avec `receivedAmount` reste possible ; tentative de carte en cours, puis ajustement → la tentative n'est plus reprise (une nouvelle intention porte le nouveau montant)

### Implementation for User Story 3

- [X] T031 [US3] Brancher `decide` (déjà enrichi en T015) dans `updateMembership` (`internal/api/memberships_admin.go`) : `AmountCents`, `BaseAmountCents` et `AdjustmentReason` transmis à `UpdateValidatedMembership`, `paymentColumns` avec le montant à payer. Journal `membership.update` : ajouter `"old_amount_cents"`, `"base_amount_cents"` et `"adjusted"`
- [X] T032 [US3] Dans `web/app/routes/app/admin/membership-detail.tsx` › `DecisionDialog` en mode correction, initialiser `useTrackedAmount` avec `amountCents` et `adjustmentReason` actuels (champ non suivi s'il y a un ajustement), pour que le changement de tarif conserve le montant saisi et que « Revenir au montant théorique » retire l'ajustement

**Checkpoint**: US3 testable seule sur une adhésion validée existante.

---

## Phase 6: User Story 4 - Remises et dons sur les inscriptions aux événements (Priority: P2)

**Goal**: mêmes règles pour les inscriptions : montant fixé à l'inscription par un administrateur, action « Ajuster le montant », montant reçu et don, export enrichi.

**Independent Test**: inscrire un membre à 20 € au lieu de 35 € avec un motif ; ajuster une autre inscription à 0 € puis à 35 € (même référence `EVT-`) ; constater 40 € en espèces (don de 5 €) ; exporter le CSV avec les nouvelles colonnes.

### Tests for User Story 4

- [X] T033 [P] [US4] Créer `internal/api/registration_adjustments_test.go` : `register-member` avec `amount: "20"` et motif → `amountCents 2000`, `baseAmountCents 3500`, `adjustmentCents -1500` ; `register-member` sur un événement gratuit avec `amount: "10"`, motif et `payment` → référence attribuée et inscription payée ; `register-member` sur un événement payant avec `amount: "0"`, motif et `payment` → inscription gratuite, paiement ignoré ; sans motif → 422 `body.adjustmentReason` ; `adjust-registration-amount` à 0 → `payment` null, référence conservée en base ; retour à 35 € → due avec la même référence ; inscription d'un événement gratuit ajustée à 10 € → référence `EVT-` attribuée ; inscription payée → 409 ; événement annulé → 409 ; constatation `receivedAmount: "40"` sur 35 € → `donationCents 500` et e-mail `registration-paid` avec le don ; `receivedAmount` inférieur → 422 ; annulation de la constatation → `paid_amount_cents` null ; changement de prix de l'événement → montants existants inchangés ; réponse brute de l'espace membre (inscription, « À régler ») sans `baseAmountCents`, `adjustmentCents` ni `adjustmentReason` ; CSV avec les en-têtes `Montant théorique`, `Montant à payer`, `Motif d'ajustement`, `Montant reçu`, `Don`, et un motif commençant par `=` neutralisé par `SafeCell`
- [X] T034 [P] [US4] Dans `internal/api/access_test.go` › `TestAccessMatrix`, vérifier que `adjust-registration-amount` (`PUT /admin/registrations/{id}/amount`) est couverte : lancer le test et confirmer que l'opération figure dans les cas exécutés. Si la matrice liste les opérations à la main, l'y ajouter (anonyme 401, membre 403, admin accepté) ; si elle les découvre depuis l'OpenAPI, ne rien ajouter et le noter dans le message de commit

### Implementation for User Story 4

- [X] T035 [US4] Créer dans `internal/api/registrations_admin.go` le helper `applyRegistrationAmount(ctx, q, r sqlc.EventRegistration, e sqlc.Event, requested *int64, reason, location string) (sqlc.EventRegistration, error)` : théorique = `membership.Theoretical(r.AmountCents, base)` ; `membership.Adjust` ; état et annonce par `membership.PaymentAfter` ; si le paiement existe et qu'il n'y a pas de référence, attribuer la suivante via `event.ReferencePrefix` (saison de l'événement), `ListRegistrationReferences` et `membership.NextReference`, comme `createRegistrations` dans `internal/api/events_me.go` ; puis `AdjustRegistrationAmount`
- [X] T036 [US4] Ajouter l'opération `adjust-registration-amount` (`PUT /admin/registrations/{id}/amount`, `auth.Admin`, résumé « Ajuster le montant à payer d'une inscription ») dans `registerAdminRegistrations` (`internal/api/registrations_admin.go`), avec le corps `{ amount string maxLength 12, adjustmentReason string maxLength 200 }`. Le handler passe par `adminRegistrationWrite`, refuse avec `409 MsgRegistrationChanged` si `!membership.CanUpdate(event.PaymentSnapshot(regSnapshot(r), e.Status == string(event.Cancelled)))`, puis appelle `applyRegistrationAmount`. Journal `registration.adjust` avec `base_amount_cents`, `old_amount_cents` et `amount_cents`, sans motif
- [X] T037 [US4] Dans `registerMember` (`internal/api/registrations_admin.go`), ajouter au corps `Amount` et `AdjustmentReason` (mêmes tags que T015). Après `createRegistrations`, si `amount` est fourni, appeler `applyRegistrationAmount` dans la même transaction, puis appliquer le paiement éventuel avec `receivedAmount(*in.Body.Payment, "body.payment", montant à payer)` transmis à `RecordRegistrationPayment`. Si l'inscription est gratuite après l'ajustement, le paiement fourni est ignoré, comme aujourd'hui (`created.PaymentStatus.Valid` relu après l'ajustement). L'ajustement est accepté aussi pour un événement gratuit (montant positif → référence attribuée)
- [X] T038 [US4] Dans `recordRegistrationPayment` (`internal/api/registrations_admin.go`), utiliser `receivedAmount(in.Body, "body", r.AmountCents)` et passer `PaidAmountCents` ; journal avec `received_cents` et `donation_cents`. Dans `notifyRegistrationPaid` (`internal/api/registration_target.go`), renseigner `data.Donation`
- [X] T039 [US4] Ajouter à `AdminRegistration` `BaseAmountCents int64`, `AdjustmentCents int64` (mêmes tags que T017), `AdjustmentReason *string` et `Actions.AdjustAmount bool` (même condition que T036), et à `AdminRegistrationRow` `BaseAmountCents int64` et `AdjustmentCents int64`, renseignés dans `toAdminRegistration` et `listRegistrations` (`internal/api/registrations_admin.go`). Rien côté espace membre (R9)
- [X] T040 [US4] Dans `exportEventRegistrations` (`internal/api/registrations_admin.go`), passer aux colonnes de contracts/api.md › Export CSV : Nom, Prénom, Date de naissance, Téléphone, Adresse de contact, Adhérent, Date d'inscription, Montant théorique, Montant à payer, Motif d'ajustement (`csvfile.SafeCell`), Paiement, Moyen, Montant reçu, Don, Référence. Une inscription gratuite sans ajustement garde des montants vides ; une inscription gratuite par ajustement affiche son théorique et son motif
- [X] T041 [US4] Régénérer l'OpenAPI et les types frontend (`go tool task gen`)
- [X] T042 [US4] Dans `web/app/routes/app/admin/registration-detail.tsx`, ajouter l'action « Ajuster le montant » (si `actions.adjustAmount`) qui ouvre un `DialogShell` avec `AmountToPayField` (théorique = `baseAmountCents`, valeur initiale = `amountCents` et `adjustmentReason`) et appelle `adjust-registration-amount`, puis invalide les requêtes de l'inscription et de l'événement. Si `cardPaymentPending` est vrai, le dialogue affiche l'avertissement de paiement par carte en cours (même texte et même composant que la correction d'une adhésion, `membership-detail.tsx` › `DecisionDialog`) avant la confirmation (spec US3 sc. 4). Afficher le montant théorique, la remise ou la majoration (`adjustmentCents`), le motif, le montant reçu et le don comme en T021/T029 ; passer `amountCents` à `RecordPaymentDialog`
- [X] T043 [US4] Dans `web/app/events/RegisterMemberDialog.tsx`, pour tout événement (payant ou gratuit), ajouter `AmountToPayField` (théorique = prix de l'événement, 0 € s'il est gratuit) ; proposer le bloc « déjà payé » pour tout événement, avec « Montant reçu » (préremplissage comme en T028), sans comparer les montants côté client : le serveur ignore le paiement si l'inscription est gratuite après ajustement ; dans `web/app/routes/app/admin/event-detail.tsx`, passer `amountCents` à `RecordPaymentDialog`
- [X] T044 [P] [US4] Dans `web/app/routes/app/member/registration-detail.tsx`, afficher « Reçu X € » et « dont Y € de don, merci ! » pour une inscription payée, sans montant théorique ni motif

**Checkpoint**: les inscriptions suivent les mêmes règles que les adhésions ; US4 testable sur un événement payant.

---

## Phase 7: User Story 5 - Chiffres fidèles à la réalité (Priority: P3)

**Goal**: chiffres de la saison et totaux d'un événement avec théorique, remises, majorations, à payer, encaissé dont dons, reste à percevoir.

**Independent Test**: sur un jeu mixte, vérifier chaque total de `get-membership-stats` et de `get-event`, ainsi que l'identité théorique − remises + majorations = à payer.

### Tests for User Story 5

- [X] T045 [P] [US5] Créer `internal/api/payment_figures_test.go` : saison avec une adhésion ajustée à 70 € (théorique 75 €) payée 70 €, une majoration à 50 € (théorique 45 €) due annoncée par chèque, une adhésion payée 80 € pour 45 € (don de 35 €), une adhésion gratuite par ajustement, une annulée et une payée par carte en mode test. Vérifier `theoreticalCents`, `discountCents`, `surchargeCents`, `totalCents`, `paidCents`, `donationCents`, `dueCents`, `dueByMethod.cheque`, `collectedByMethod`, `collectedTest` et `byFee[].amountCents`. Même logique pour `get-event` › `totals` (`theoreticalCents`, `discountCents`, `surchargeCents`, `dueCents`, `collectedCents`, `donationCents`, `remainingCents`, `collectedTestCents`)

### Implementation for User Story 5

- [X] T046 [US5] Dans `getMembershipStats` (`internal/api/memberships_admin.go`), passer `BaseCents` à `StatRow` depuis `r.BaseAmountCents`, et ajouter à `MembershipStats` `TheoreticalCents`, `DiscountCents`, `SurchargeCents` et `DonationCents` (tags JSON de contracts/api.md, docs en français)
- [X] T047 [US5] Dans `internal/api/events_admin.go`, réécrire `addTotals` sur `membership.AmountTotals` (`AmountRow` construit depuis l'inscription et `paidEnvironment == "sandbox"`). Ajouter à `EventTotals` `TheoreticalCents`, `DiscountCents`, `SurchargeCents`, `DonationCents` et `RemainingCents`, sans changer le sens de `DueCents` (total à payer), `CollectedCents`, `CollectedTestCents` et `PaidCount`
- [X] T048 [US5] Régénérer l'OpenAPI et les types frontend (`go tool task gen`)
- [X] T049 [P] [US5] Dans `web/app/routes/app/admin/memberships.tsx` › chiffres, afficher les tuiles « Montant théorique », « Remises », « Majorations » (si non nul), « Total à payer » (`totalCents`), « Encaissé » (détail par moyen inchangé, puis « dont dons X € » si non nul) et « Reste à percevoir » (`dueCents`)
- [X] T050 [P] [US5] Dans `web/app/routes/app/admin/event-detail.tsx`, afficher les totaux : théorique, remises, majorations (si non nul), total à payer, encaissé dont dons, reste à percevoir

**Checkpoint**: toutes les stories sont fonctionnelles.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [ ] T051 [P] Ajouter dans `internal/tools/e2eseed/main.go` une adhésion validée avec remise et motif, une adhésion payée avec don et une inscription ajustée, pour les tests de bout en bout (exemples fictifs uniquement)
- [X] T052 Créer `web/e2e/payment-adjustments.spec.ts` (Playwright + `web/e2e/axe.ts`) : validation d'une demande avec remise (suivi du montant au changement de type, motif obligatoire, affichage du théorique et du motif), constatation avec don, ajustement d'une inscription, et vérification côté membre (montant ajusté, aucun motif). Aucune violation axe `serious` ni `critical` sur les dialogues ouverts
- [X] T053 [P] Mettre à jour les tests Playwright existants que les libellés changent (`web/e2e/admin-memberships.spec.ts`, `web/e2e/admin-events.spec.ts`, `web/e2e/memberships.spec.ts`, `web/e2e/events.spec.ts` : « Montant dû » → « Montant à payer », nouveaux champs des dialogues)
- [X] T054 Vérifier qu'aucun fichier versionné ne mentionne une association ou une personne réelle (`git grep` sur les noms des données de production), et que les exemples restent fictifs
- [X] T055 Lancer `go tool task check` (build `CGO_ENABLED=0`, vet, tests, `sqlc generate`/`frigg openapi`/`bun run gen:api` sans diff, migrations sur base vierge, typecheck, Biome, Vitest, build web), puis `go tool task e2e`
- [ ] T056 Dérouler les scénarios manuels de `specs/014-payment-adjustments/quickstart.md` (1 à 5)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance.
- **Foundational (Phase 2)** : après Setup ; bloque toutes les stories.
  - T002 → T003, T004, T005 → T006 ; T007 → T008, T009 ; T010, T011, T012 après T006.
- **US1 (Phase 3)** : après Phase 2.
- **US2 (Phase 4)** : après Phase 2 ; indépendante de US1 côté backend. Côté frontend, T028 réutilise le champ de T022 (membership-new) : faire US1 avant si les deux sont menées en séquence.
- **US3 (Phase 5)** : après T015 (US1, `decide` enrichi) et T019/T020 (composant).
- **US4 (Phase 6)** : après Phase 2, T010 et T012 ; réutilise `AmountToPayField` (T020) et `RecordPaymentDialog` modifié (T027).
- **US5 (Phase 7)** : après Phase 2 (T009) ; les données de test passent par l'API de US1, US2 et US4 (ou s'insèrent directement en base).
- **Polish (Phase 8)** : après les stories retenues.

### Within Each User Story

- Tests écrits d'abord et en échec, puis le backend, la régénération OpenAPI, et le frontend.
- Les tâches de régénération (T018, T026, T041, T048) précèdent les tâches frontend de leur story.

### Parallel Opportunities

- Phase 2 : T005 en parallèle de T004 ; T007/T008 en parallèle des requêtes ; T010, T011 et T012 en parallèle une fois T006 terminé.
- Dans chaque story, les tests [P] se lancent ensemble.
- US4 (backend inscriptions) et US5 (chiffres) peuvent avancer en parallèle de US2/US3, sur des fichiers distincts.
- Frontend : T044, T049 et T050 touchent des fichiers distincts.

---

## Parallel Example: User Story 1

```bash
# Tests en parallèle :
Task: "T013 Tests d'API de validation et création avec ajustement dans internal/api/membership_adjustments_test.go"
Task: "T014 Tests d'isolation membre et de carte au montant ajusté (même fichier, à rédiger à la suite de T013)"

# Frontend en parallèle du backend :
Task: "T019 Hook useTrackedAmount et son test dans web/app/payments/useTrackedAmount.ts"
```

## Parallel Example: User Story 4

```bash
Task: "T033 Tests d'ajustement des inscriptions dans internal/api/registration_adjustments_test.go"
Task: "T034 Matrice d'autorisation dans internal/api/access_test.go"
Task: "T044 Don côté membre dans web/app/routes/app/member/registration-detail.tsx"
```

---

## Implementation Strategy

### MVP First (US1 + US2)

1. Phase 1 et Phase 2.
2. US1 : remise à la validation et à la création → valider avec le scénario 1 du quickstart.
3. US2 : montant reçu et don → valider avec le scénario 2.
4. **STOP et VALIDER** : les trois cas réels (remise, membre qui donne plus, virement qui couvre davantage) sont pris en charge pour les adhésions.

### Incremental Delivery

1. Fondations → US1 → US2 (MVP, déployable).
2. US3 → ajustement après validation.
3. US4 → inscriptions aux événements.
4. US5 → chiffres complets.
5. Polish → e2e, `task check`, quickstart.

### Après déploiement

Corriger les paiements existants dont le commentaire signale un écart, selon
`quickstart.md` › Correction des paiements existants.
