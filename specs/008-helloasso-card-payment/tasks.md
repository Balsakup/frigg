---

description: "Liste des tâches : paiement des adhésions par carte bancaire (HelloAsso)"
---

# Tasks: Paiement des adhésions par carte bancaire (HelloAsso)

**Input**: Design documents from `specs/008-helloasso-card-payment/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, helloasso.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : matrice d'autorisation (principes II et III, dont la
nouvelle opération publique), tests d'API, Playwright + axe sur chaque
parcours touché. La sécurité de la constatation (retours falsifiés,
notifications rejouées, concurrence) a ses propres tests.

**Organization**: une phase par user story, dans l'ordre de la spec (US1 et
US2 en P1, US3 en P2). US1 est testable sans l'écran de US2 : les tests
activent la carte directement en base (`payment_settings.card_enabled = 1` et
test de connexion réussi enregistré).

**Conventions** (constitution 1.7.0) : identifiants, tables, champs JSON et
valeurs en anglais ; commentaires, messages, textes d'interface, journaux et
e-mails en français. Commentaires des requêtes sqlc en ASCII uniquement.
Montants en centimes d'euro. Jamais de nom, d'e-mail, de secret, de jeton,
d'URL de redirection ni de donnée de carte dans les journaux. Aucun appel
HelloAsso à l'intérieur d'une transaction d'écriture.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US3)

## Path Conventions

Module Go à la racine (`internal/`), SPA dans `web/` (plan.md › Project
Structure). Code existant à réutiliser :

- API : `internal/api/api.go` (`register`, `Deps`, `s.today()`,
  `CurrentAccount`, `s.internalError`), `internal/api/memberships.go`
  (`notifyMembership`, `logMembershipAction`, `paymentOf`, `toMembership`,
  `snapshotOf`, `formatEuros`, `methodLabels`, `s.timestamp()`),
  `internal/api/memberships_me.go` (`ownAccount`, requêtes `…OfAccount`,
  `PaymentMethodInfo`, `MyMembershipActions`), `internal/api/memberships_admin.go`,
  `internal/api/settings.go` (`settingsFromRow`, `toPaymentSettings`,
  messages `Msg…`), `internal/api/errors.go` (`FieldErrors`).
- Domaine : `internal/membership` (`Method`, `Methods`, `Snapshot`, `CanPay`, `CanRecordPayment`, `CanCancelPayment`, `MemberActionsFor`, `AdminActionsFor`,
  `Snapshot`, `Settings.EnabledMethods`, `Stats`), `internal/season` (`Date`,
  `Today`, `FieldErrors`).
- Infrastructure : `internal/config/config.go` (`env`, `envBool`, `invalid`,
  `RequireServe`), `internal/auth/ratelimit.go` (`KeyedLimiter`, `Limits`),
  `internal/mail/worker.go` (modèle de worker périodique à horloge
  injectable), `internal/cli/serve.go`.
- Tests Go : `internal/api/apitest` (`New`, `Seed`, `LoggedIn`, `SetToday`,
  `FlushMail`, `Mail`, `Logs`, `Store`, `Client().Do`), matrice
  `internal/api/access_test.go` (`allowedPublic`),
  `internal/api/memberships_isolation_test.go`,
  `internal/api/memberships_helpers_test.go` (création d'adhésions validées).
- Frontend : `web/app/api/*` (requêtes et invalidations), `web/app/api/problem.ts`,
  `web/app/memberships/PaymentMethodInfo.tsx`,
  `web/app/routes/app/member/membership-detail.tsx`,
  `web/app/routes/app/admin/settings-payments.tsx`, composants `web/app/ui/*`,
  `Icon`. E2E : `web/e2e/test.ts`, `web/e2e/axe.ts`, `web/e2e/mailpit.ts`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: configuration, schéma et client HelloAsso communs à toutes les stories.

- [ ] T001 Ajouter à `Config` les champs `HelloAsso{ClientID, ClientSecret, Organization, Environment, APIURL}` lus depuis `FRIGG_HELLOASSO_CLIENT_ID`, `FRIGG_HELLOASSO_CLIENT_SECRET`, `FRIGG_HELLOASSO_ORGANIZATION`, `FRIGG_HELLOASSO_ENVIRONMENT` (`sandbox` par défaut, sinon `production`, toute autre valeur → erreur « FRIGG_HELLOASSO_ENVIRONMENT doit valoir sandbox ou production »), `FRIGG_HELLOASSO_API_URL` (tests) ; méthode `HelloAssoConfigured() bool` ; règle tout ou rien (« FRIGG_HELLOASSO_CLIENT_SECRET est obligatoire lorsque FRIGG_HELLOASSO_CLIENT_ID est renseigné », etc.) ; dans `RequireServe`, erreur si HelloAsso est configuré, `FRIGG_BASE_URL` ne commence pas par `https://` et `FRIGG_HELLOASSO_API_URL` est vide, in `internal/config/config.go`
- [ ] T002 [P] Tests en table de la configuration HelloAsso (aucune variable → non configuré sans erreur ; variables partielles → message explicite ; environnement inconnu ; base HTTP refusée ; base HTTP admise avec `FRIGG_HELLOASSO_API_URL`) in `internal/config/config_test.go` (nouveau)
- [ ] T003 [P] Documenter les cinq variables HelloAsso in `.env.example` : les quatre variables d'exploitation (commentées, défaut `sandbox`, rappel que le secret ne doit jamais être commité) et `FRIGG_HELLOASSO_API_URL` avec le commentaire « Réservée aux tests : remplace l'hôte de l'API HelloAsso (faux HelloAsso local). Ne pas renseigner en production. » (constitution : `.env.example` documente chaque variable)
- [ ] T004 Écrire la migration in `internal/database/migrations/00009_online_payments.sql` : (1) créer `memberships_new` identique à `memberships` de `00008` mais avec `paid_method TEXT CHECK (paid_method IN ('cash','cheque','transfer','card'))`, `paid_online_payment_id INTEGER REFERENCES online_payments (id) ON DELETE SET NULL`, `paid_amount_cents INTEGER CHECK (paid_amount_cents BETWEEN 0 AND 1000000)`, et les `CHECK` ajoutés `paid_method IS NOT 'card' OR (paid_online_payment_id IS NOT NULL AND paid_amount_cents IS NOT NULL AND paid_by IS NULL)` et `paid_online_payment_id IS NULL OR paid_method = 'card'` ; copier les lignes ; supprimer les triggers `memberships_fts_ai/ad/au` et `memberships` ; renommer ; recréer les index `memberships_one_active` (`UNIQUE (member_id, season_id) WHERE status IN ('pending','validated')`), `memberships_season_status`, `memberships_member_id`, `memberships_fee_id`, `memberships_status_requested` et les trois triggers FTS ; `INSERT INTO memberships_fts (memberships_fts) VALUES ('rebuild')` ; (2) créer `online_payments` (STRICT) selon data-model.md : `environment TEXT NOT NULL CHECK (environment IN ('sandbox','production'))`, `amount_cents INTEGER NOT NULL CHECK (amount_cents BETWEEN 1 AND 1000000)`, `status TEXT NOT NULL CHECK (status IN ('pending','succeeded','failed','abandoned','expired'))`, `redirect_url` `CHECK (redirect_url IS NULL OR length(redirect_url) <= 2000)`, `received_amount_cents` `CHECK (received_amount_cents BETWEEN 0 AND 1000000)`, `error_code` `CHECK (error_code IS NULL OR length(error_code) <= 60)`, `CHECK (status <> 'succeeded' OR (order_id IS NOT NULL AND helloasso_payment_id IS NOT NULL AND received_amount_cents IS NOT NULL AND paid_at IS NOT NULL))`, `CHECK (status = 'pending' OR finished_at IS NOT NULL)`, index `online_payments_intent` `UNIQUE (environment, checkout_intent_id) WHERE checkout_intent_id IS NOT NULL`, `online_payments_helloasso_payment` `UNIQUE (environment, helloasso_payment_id) WHERE helloasso_payment_id IS NOT NULL`, `online_payments_membership (membership_id, created_at)`, `online_payments_pending (status, created_at) WHERE status = 'pending'`, `online_payments_order (order_id)` ; (3) créer `payment_alerts` (STRICT) avec `kind TEXT NOT NULL CHECK (kind IN ('duplicate','amount_mismatch','refund_due','unverified'))`, `CHECK (resolved_by IS NULL OR resolved_at IS NOT NULL)`, `UNIQUE (online_payment_id, kind)`, index `payment_alerts_open (resolved_at, created_at)` ; (4) `ALTER TABLE payment_settings ADD COLUMN` `card_enabled INTEGER NOT NULL DEFAULT 0 CHECK (card_enabled IN (0, 1))`, `helloasso_test_at TEXT`, `helloasso_test_environment TEXT CHECK (… IN ('sandbox','production'))`, `helloasso_test_ok INTEGER CHECK (… IN (0, 1))`, `helloasso_test_error TEXT CHECK (length(…) <= 60)`, `helloasso_organization_name TEXT CHECK (length(…) <= 250)`, `helloasso_error_at TEXT`, `helloasso_error_code TEXT CHECK (length(…) <= 60)` ; section `Down` inverse
- [ ] T005 [P] Test de migration : base migrée jusqu'à `00008`, peuplée d'adhésions dans chaque état (dont payée par chèque, annulée payée, gratuite), puis `00009` : lignes identiques, `memberships_one_active` toujours actif, recherche FTS par fragment de référence fonctionnelle, `card_enabled = 0` ; migration sur base vierge, in `internal/database/migrate_00009_test.go` (nouveau, à côté de `db_test.go`)
- [ ] T006 [P] Écrire les requêtes in `internal/database/queries/online_payments.sql` : `CreateOnlinePayment`, `GetOnlinePayment`, `GetOnlinePaymentOfAccount` (jointure `memberships`/`members`, filtre `members.account_id`), `FindResumableOnlinePayment` (même adhésion, environnement, montant, `status = 'pending'`, `redirect_url IS NOT NULL`, `created_at > ?`), `SetOnlinePaymentCheckout`, `FailOnlinePayment`, `CloseOnlinePayment` (failed/abandoned/expired), `SucceedOnlinePayment`, `TouchOnlinePaymentVerified`, `ListOnlinePaymentsOfMembership`, `ListPendingOnlinePaymentsBefore` (limite 50, plus anciennes d'abord), `ListVerifiableOnlinePaymentsOfMembership`, `LatestOnlinePaymentOfMembership` ; commentaires ASCII
- [ ] T007 [P] Écrire les requêtes in `internal/database/queries/payment_alerts.sql` : `CreatePaymentAlert` (`ON CONFLICT (online_payment_id, kind) DO NOTHING`), `ListPaymentAlertsOfMembership`, `GetPaymentAlert`, `ResolvePaymentAlert` (`WHERE resolved_at IS NULL`), `CountOpenPaymentAlerts` ; commentaires ASCII
- [ ] T008 Étendre `internal/database/queries/payment_settings.sql` (`card_enabled` dans la lecture et la mise à jour, `RecordHelloAssoTest`, `RecordHelloAssoError`, `ClearHelloAssoError`) et `internal/database/queries/memberships.sql` (`RecordCardPayment` qui fixe `payment_status='paid'`, `paid_method='card'`, `paid_on`, `paid_at`, `paid_by=NULL`, `paid_amount_cents`, `paid_online_payment_id` sous condition `status='validated' AND payment_status IN ('due','announced')` ; nouvelles colonnes dans les lectures), puis lancer `sqlc generate` (code commité in `internal/database/sqlc/`)
- [ ] T009 [P] Créer le client in `internal/helloasso/client.go` : `Config` (hôte dérivé de l'environnement — `https://api.helloasso.com` ou `https://api.helloasso-sandbox.com` — ou `APIURL`), `New(cfg, *http.Client, now func() time.Time)`, jeton OAuth2 en mémoire sous `sync.Mutex` (`client_credentials`, rafraîchi 60 s avant expiration par `refresh_token`, repli sur `client_credentials`), un `401` → un renouvellement puis une seule nouvelle tentative, délai 10 s par appel, `Organization` exécutée sous un délai global de 10 s couvrant jeton et appel, erreurs `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrUnavailable`, `ErrUnexpected` sans secret ni jeton dans leur texte, `ErrorCode(err) string` (`unauthorized`, `forbidden`, `not_found`, `unavailable`, `unexpected`), `Organization(ctx) (name string, err)` (contracts/helloasso.md)
- [ ] T010 Créer in `internal/helloasso/checkout.go` : `CheckoutRequest` (montant, `itemName` tronqué à 250 caractères, payeur facultatif, trois URL, `Metadata{Payment, Membership int64; Reference string}` sérialisé sous `{"frigg": {...}}`), `CreateCheckout` (sans nouvelle tentative automatique ; refus d'une `redirectUrl` dont l'hôte n'est ni `helloasso.com`, ni `helloasso-sandbox.com` ni ses sous-domaines, ni l'hôte de `APIURL`), `GetCheckout` décodé de façon tolérante en `Checkout{ID, Metadata, Order *Order}`, `Payment{ID, AmountCents, TipCents, State, Date}`, `(Order) AuthorizedPayment() (Payment, bool)` (`State == "Authorized"`)
- [ ] T011 [P] Créer in `internal/helloasso/notification.go` : `NotificationKey(secret string) string` (32 premiers caractères hexadécimaux de `HMAC-SHA256(secret, "frigg-helloasso-notifications")`), `ParseNotification([]byte) (paymentID int64, eventType string, ok bool)` lisant `metadata.frigg.payment` (et, à défaut, `data.metadata.frigg.payment`)
- [ ] T012 Tests du client contre un faux HelloAsso `httptest` in `internal/helloasso/client_test.go` : jeton obtenu une fois et réutilisé, rafraîchi avant expiration, `401` puis succès, délai dépassé → `ErrUnavailable`, test de connexion borné à 10 s au total même si le jeton et l'appel sont lents, `403`/`404` typés, intention sans `order`, paiement `Authorized` avec contribution (`AmountCents − TipCents`), `redirectUrl` d'hôte inattendu refusée, `NotificationKey` stable, `ParseNotification` tolérant (champs inconnus, corps invalide), aucun message d'erreur ne contient le secret ou le jeton

**Checkpoint**: configuration, schéma et client prêts ; `go test ./internal/config ./internal/helloasso ./internal/database/...` passe.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: règles du domaine et câblage partagés par les trois stories.

**⚠️ CRITICAL**: aucune story ne commence avant la fin de cette phase.

- [ ] T013 Ajouter `Card Method = "card"` (libellé « carte bancaire »), `Methods = []Method{Card, Cash, Cheque, Transfer}`, `MinCardAmount = 100`, le champ `PaidMethod Method` à `Snapshot` (rempli par `snapshotOf` dans `internal/api/memberships.go`), `CanPayByCard(s Snapshot, cardAvailable bool) bool` (`CanPay(s)`, montant ≥ `MinCardAmount`, carte disponible), `PayByCard` dans `MemberActions`/`MemberActionsFor` (signature étendue d'un paramètre `cardAvailable`), `CanCancelPayment` renvoyant `false` si `s.PaidMethod == Card` (donc `AdminActionsFor.CancelPayment = false`), et `ManualMethod(m Method) bool` (vrai sauf pour `card`, utilisé par T020), in `internal/membership/membership.go`
- [ ] T014 [P] Créer in `internal/membership/online.go` : `OnlineStatus` (`pending`, `succeeded`, `failed`, `abandoned`, `expired`), `AlertKind` (`duplicate`, `amount_mismatch`, `refund_due`, `unverified`), `CardAvailability(configured, enabled bool, testedEnv string, testOK bool, currentEnv string) bool` (R7), `Verified{Authorized bool; ReceivedCents int64; PaidAt time.Time; Hint string /* return|error|back|"" */}`, `SettleOnline(attempt OnlineStatus, m Snapshot, v Verified) Settlement{NewStatus OnlineStatus; RecordPayment bool; Alert AlertKind}` appliquant le tableau de research.md › R5 (paiement autorisé l'emporte toujours, `succeeded` final)
- [ ] T015 [P] Tests en table in `internal/membership/online_test.go` : chaque ligne du tableau R5 (non autorisé + `error` → `failed`, + `back` → `abandoned`, + vide → inchangé ; autorisé sur `due`/`announced` → constatation ; montant différent → constatation + `amount_mismatch` ; déjà payée → `duplicate` ; annulée → `refund_due` ; `failed` puis autorisé → `succeeded` ; `succeeded` rejoué → aucun effet), `CardAvailability` (non configuré, désactivé, environnement testé différent, test en échec), `CanPayByCard`, `MemberActionsFor` avec carte et minimum, `CanCancelPayment` sur paiement par carte, `ManualMethod`
- [ ] T016 [P] Étendre `Stats` : `CollectedByMethod[Card]`, `CollectedTest` (adhésions payées par carte dont la tentative est `sandbox`, exclues de `Collected`) et tests in `internal/membership/stats.go` et `internal/membership/stats_test.go`
- [ ] T017 [P] Ajouter `NotificationIP *KeyedLimiter` (60 requêtes par minute et par IP) à `Limits` et à son constructeur in `internal/auth/ratelimit.go`
- [ ] T018 Ajouter `HelloAsso *helloasso.Client` (nil si non configuré) à `api.Deps` et une fonction `s.cardAvailable(ctx, q) (bool, error)` qui lit `payment_settings` et applique `membership.CardAvailability` avec `s.Config.HelloAsso` ; construire le client dans `internal/cli/serve.go` quand `cfg.HelloAssoConfigured()` ; permettre à `apitest.New` d'injecter un faux HelloAsso (`httptest`) et une configuration HelloAsso, in `internal/api/api.go`, `internal/cli/serve.go`, `internal/api/apitest/`
- [ ] T019 [P] Ajouter à `apitest` un faux HelloAsso pilotable (`env.HelloAsso.Authorize(intentID, amount, tip)`, `.Fail()`, `.Down()`, compteur d'appels) et un utilitaire `EnableCard(env)` qui fixe `card_enabled = 1` et un test réussi pour l'environnement configuré, in `internal/api/apitest/helloasso.go`
- [ ] T020 Ajouter `card` au `enum` des types huma (`PaymentMethod`, `paidMethod`), `paidAmountCents` et `online` à `Payment` dans `paymentOf`, et, au moyen de `membership.ManualMethod`, faire refuser `card` par `announce-my-payment` (`422`, « Le paiement par carte se fait en ligne. ») et `record-membership-payment` (`422`), et `cancel-membership-payment` sur paiement par carte (`409`), in `internal/api/memberships.go`, `internal/api/memberships_me.go`, `internal/api/memberships_admin.go`
- [ ] T021 Régénérer le schéma (`frigg openapi`) et les types frontend (`openapi-typescript`) ; ajouter le libellé « Carte bancaire » et l'icône `CreditCard` aux libellés de moyens in `web/app/api/schema.d.ts` et `web/app/memberships/MembershipBadges.tsx`

**Checkpoint**: domaine et câblage prêts ; `go test ./...` et `bun run typecheck` passent.

---

## Phase 3: User Story 1 - Payer mon adhésion par carte (Priority: P1) 🎯 MVP

**Goal**: le compte qui gère la fiche paie par carte chez HelloAsso ; Frigg constate le paiement après vérification, que l'adhérent revienne ou non.

**Independent Test**: carte activée en base, faux HelloAsso : valider une demande, lancer le paiement, payer, revenir → « Payée — carte bancaire » ; payer sans revenir → constaté par la notification.

### Tests for User Story 1

- [ ] T022 [P] [US1] Tests d'API du lancement in `internal/api/card_payments_test.go` : tentative `pending` au montant appliqué ; `itemName` (saison, tarif, référence, sans le prénom ni le nom du membre), `payer` (fiche « Vous » ou e-mail seul), `metadata.frigg` et URL de retour envoyés au faux HelloAsso ; second lancement < 14 min → `resumed: true` sans nouvel appel ; montant changé → nouvelle tentative ; HelloAsso injoignable → `503` et tentative `failed` (`error_code = unavailable`) ; adhésion payée, gratuite, annulée, en attente → `409` ; carte indisponible ou montant < 1 € → `422`
- [ ] T023 [P] [US1] Tests d'API de la constatation in `internal/api/card_payments_test.go` : retour `return` avec paiement autorisé → adhésion `paid`, `paid_method='card'`, `paid_amount_cents` hors contribution, e-mail `membership-paid` en file ; retour `return` alors que HelloAsso ne renvoie pas d'`order` → **non** payée ; `error` → `failed`, `back` → `abandoned` ; `metadata` d'une autre tentative → ignoré ; vérifications rapprochées (< 5 s) → un seul appel HelloAsso ; `failed` puis autorisé → `succeeded` ; doublon (déjà payée par chèque) → `duplicate` ; montant différent → `amount_mismatch` ; adhésion annulée → `refund_due`
- [ ] T024 [P] [US1] Tests de la notification in `internal/api/helloasso_webhook_test.go` (chaîne HTTP complète `httptest`) : clé valide et tentative connue → constatation, `200` ; notification et retour concurrents (goroutines) → une seule constatation et un seul e-mail ; **sans clé valide** (clé absente, fausse, mal formée comme `abc` ou 200 caractères, avec corps valide, illisible ou de plus de 1 Mio) → réponses identiques octet à octet (`200`, corps vide), aucune écriture, aucun appel au faux HelloAsso ; clé valide avec autre vente, type `Form`, tentative inconnue, corps illisible, notification rejouée → `200` vide sans constatation supplémentaire ; clé valide, tentative connue, faux HelloAsso injoignable → `503` corps vide puis, HelloAsso rétabli, renvoi → constatation ; erreur interne simulée → `503` corps vide ; aucune réponse en `problem+json`, `422` ni `413` ; débit dépassé → `429` ; aucune session ni cookie ; `Sec-Fetch-Site: cross-site` → refusée par `CrossOriginProtection`
- [ ] T025 [P] [US1] Tests du balayage (horloge injectée) in `internal/api/online_sweeper_test.go` : tentative > 60 min sans paiement → `expired` ; avec paiement autorisé → constatée ; HelloAsso injoignable → reste `pending`, puis signalement `unverified` au-delà de 24 h ; tentative `pending` d'un autre environnement que celui configuré → `expired` (`error_code = 'environment_changed'`) sans appel au faux HelloAsso ni signalement ; au plus 50 par passage
- [ ] T026 [P] [US1] Étendre `internal/api/memberships_isolation_test.go` : `start-my-card-payment` et `confirm-my-card-payment` par un autre compte, sur une fiche sans compte et sur un identifiant inexistant → `404`, aucune tentative créée ; ajouter `"receive-helloasso-notification": true` à `allowedPublic` avec la mention « amendements 1.6.0 et 1.7.0 » in `internal/api/access_test.go`

### Implementation for User Story 1

- [ ] T027 [US1] Implémenter `verifyOnlinePayment(ctx, paymentID int64, source, hint string) (membership.OnlineStatus, error)` in `internal/api/card_payments.go` : lecture, idempotence `succeeded`, limitation 5 s (`last_verified_at`), `GetCheckout` hors transaction, contrôle `metadata.frigg.payment`/`membership` et environnement, puis transaction d'écriture (`BEGIN IMMEDIATE`) avec relecture, `membership.SettleOnline`, `SucceedOnlinePayment` / `CloseOnlinePayment`, `RecordCardPayment` (`paid_on` = date du paiement à Paris), `CreatePaymentAlert`, e-mail `membership-paid` via `notifyMembership` dans la transaction ; `RecordHelloAssoError` / `ClearHelloAssoError` selon l'issue de l'appel ; journaux R14 (`paiement en ligne vérifié`, `source`, `outcome`, `order_id`)
- [ ] T028 [US1] Implémenter `start-my-card-payment` (`POST /me/memberships/{id}/card-payment`, `auth.Member`) in `internal/api/card_payments.go` : lecture `…OfAccount`, `membership.CanPayByCard(snapshot, s.cardAvailable)`, reprise via `FindResumableOnlinePayment` (14 min), sinon insertion puis `CreateCheckout` hors transaction (`returnUrl` = `{BASE_URL}/adhesions/{id}?carte=retour&tentative={n}`, `errorUrl` `…carte=erreur…`, `backUrl` `…carte=annule…`, `itemName` « Adhésion {saison} — {tarif} — {référence de paiement} », sans nom du membre), `SetOnlinePaymentCheckout` ou `FailOnlinePayment` + `503` ; réponse `{redirectUrl, onlinePaymentId, resumed}` ; journal `paiement en ligne démarré`
- [ ] T029 [US1] Implémenter `confirm-my-card-payment` (`POST /me/memberships/{id}/card-payment/confirm`, `auth.Member`, corps `{onlinePaymentId, outcome: return|error|back, checkoutIntentId}`) in `internal/api/card_payments.go` : `GetOnlinePaymentOfAccount` (sinon `404`), `verifyOnlinePayment(…, "return", outcome)`, HelloAsso injoignable → `200` `status: "pending"` ; réponse `{status, membership: MyMembership}`
- [ ] T030 [US1] Implémenter `receive-helloasso-notification` (`POST /webhooks/helloasso/{key}`, `auth.Public`, `key` chaîne libre **sans `pattern`**, corps brut `RawBody` lu jusqu'à 1 Mio, surplus ignoré, aucun schéma de corps) in `internal/api/helloasso_webhook.go` : limitation `NotificationIP` (sinon `429`) ; clé comparée en temps constant (`hmac.Equal`) avec `helloasso.NotificationKey` — si invalide, `200` vide sans autre traitement ; `ParseNotification` ; tentative inconnue ou d'un autre environnement, ou notification illisible → `200` vide et journal `notification ignorée` (debug) ; sinon `verifyOnlinePayment(…, "notification", "")` → `200` vide si la vérification aboutit, `503` vide si HelloAsso est injoignable ou en cas d'erreur interne (constitution v1.7.0, principe II) ; corps de réponse toujours vide ; enregistrer l'opération dans `New` in `internal/api/api.go`
- [ ] T031 [US1] Implémenter `OnlinePaymentSweeper` (`Run(ctx)`, `Tick(ctx)`, intervalle 5 min, clôture en `expired` (`error_code = 'environment_changed'`) des tentatives `pending` d'un autre environnement sans appel à HelloAsso, horloge injectable, sur le modèle de `mail.Worker`) in `internal/api/online_sweeper.go` et le lancer dans `internal/cli/serve.go` lorsque HelloAsso est configuré
- [ ] T032 [US1] Exposer la carte dans l'espace membre in `internal/api/memberships_me.go` : `PaymentMethodInfo` `{method: "card", amountCents, test}` en premier si `s.cardAvailable` et adhésion payable, `actions.payByCard`, `cardPayment` (dernière tentative de l'adhésion : statut, `startedAt`)
- [ ] T033 [US1] Mentionner le moyen « carte bancaire » et la date du paiement dans les gabarits in `internal/mail/templates/membership-paid.html.tmpl` et `internal/mail/templates/membership-paid.txt.tmpl` (libellé via `methodLabels` complété de `card` in `internal/api/memberships.go`)
- [ ] T034 [P] [US1] Ajouter les requêtes `useStartCardPayment` et `useConfirmCardPayment` (invalidation de `my-memberships` et de l'adhésion) in `web/app/api/memberships.ts`
- [ ] T035 [P] [US1] Créer `CardPaymentPanel` in `web/app/memberships/CardPaymentPanel.tsx` : montant formaté, texte « Vous allez être redirigé vers HelloAsso, la plateforme de paiement des associations. HelloAsso vous proposera une contribution volontaire facultative à son fonctionnement. », bouton principal « Payer {montant} par carte » relié au texte par `aria-describedby`, état de chargement, erreur `503` affichée sans quitter la page, puis `window.location.assign(redirectUrl)`
- [ ] T036 [US1] Intégrer la carte à la page de paiement in `web/app/routes/app/member/membership-detail.tsx` et `web/app/memberships/PaymentMethodInfo.tsx` : « Carte bancaire » en premier dans le `RadioGroup`, `CardPaymentPanel` à la place du bouton de confirmation, bandeau `warn` « Mode test — aucun débit réel » si `test` ; badge `warn` « Mode test » dans `web/app/memberships/MembershipBadges.tsx` pour toute adhésion payée en mode test (`payment.online.test`), visible sur cette page et dans la liste « Adhésions » du membre (`web/app/routes/app/member/memberships.tsx`, FR-015) ; au chargement avec `carte` ∈ `retour|erreur|annule` et `tentative`, appel de `confirm-my-card-payment`, retrait des paramètres (`replace`), annonce `role="status"` : « Paiement reçu, merci… » (`ok`), « Le paiement n'a pas abouti. Aucune somme n'a été débitée. » (`danger`, « Réessayer »), `pending` : « Confirmation du paiement en cours… » avec nouvel appel toutes les 3 s pendant 30 s puis « Votre paiement sera confirmé dans quelques minutes ; vous recevrez un e-mail. »
- [ ] T037 [P] [US1] Créer le faux HelloAsso de test in `web/e2e/fake-helloasso.ts` (serveur Bun : `/oauth2/token`, `GET /v5/organizations/{slug}`, `POST` et `GET` `checkout-intents`, page « Payer / Refuser / Retour » renvoyant vers `returnUrl`, `errorUrl`, `backUrl`, option d'envoi de la notification à Frigg) et le brancher dans la configuration Playwright (`FRIGG_HELLOASSO_*`, `FRIGG_HELLOASSO_API_URL`) in `web/playwright.config.ts`
- [ ] T038 [US1] Parcours Playwright + axe in `web/e2e/card-payment.spec.ts` : paiement réussi depuis le lien de l'e-mail (Mailpit), refus, abandon, paiement sans revenir sur Frigg (notification seule puis rechargement → « Payée »), bandeau mode test, parcours au clavier ; aucune violation `serious` ou `critical`

**Checkpoint**: US1 livrable seule (carte activée en base) ; SC-001 à SC-004 vérifiables.

---

## Phase 4: User Story 2 - Paramétrer HelloAsso (administration) (Priority: P1)

**Goal**: l'administrateur voit l'état de la configuration, copie l'adresse de notification, teste la connexion et active la carte.

**Independent Test**: avec identifiants (faux HelloAsso) : tester, activer, constater la carte sur une page de paiement ; sans identifiants : activation impossible.

### Tests for User Story 2

- [ ] T039 [P] [US2] Tests d'API in `internal/api/settings_test.go` : `get-payment-settings` renvoie `card` (`configured`, `environment`, `organization`, `notificationUrl` = `{BASE_URL}/api/webhooks/helloasso/{clé}`, `lastTest`, `available`, `failing`) sans jamais le secret ; `test-helloasso-connection` non configuré → `409`, succès → `lastTest.ok` et nom de l'association, échecs `unauthorized`/`forbidden`/`not_found`/`unavailable` décrits dans `lastTest.error` ; activation sans configuration ou sans test réussi pour l'environnement courant → `422` sur `body.card.enabled` ; environnement changé → `available = false` jusqu'au nouveau test, `card_enabled` conservé ; identifiants retirés → `available = false` ; `get-membership-dashboard.helloassoFailing` ; test vérifiant que ni le secret ni le jeton n'apparaissent dans les réponses et les journaux (`env.Logs`)

### Implementation for User Story 2

- [ ] T040 [US2] Étendre `PaymentSettings` et `PaymentSettingsBody` (`card: {enabled}`) et `toPaymentSettings` (bloc `card` de contracts/api.md) ; règle d'activation (FR-013) avec les messages « Les identifiants HelloAsso ne sont pas renseignés dans la configuration de l'instance. » et « Testez d'abord la connexion à HelloAsso. » in `internal/api/settings.go`
- [ ] T041 [US2] Implémenter `test-helloasso-connection` (`POST /admin/settings/payment/helloasso-test`, `auth.Admin`) : `409` si non configuré (« Renseignez les identifiants HelloAsso dans la configuration de l'instance. »), sinon `Organization(ctx)` puis `RecordHelloAssoTest` (date, environnement, résultat, nom, code d'erreur) et mise à jour de l'incident ; journal `test de connexion HelloAsso` (`ok`, `error_code`) ; réponse `PaymentSettings`, in `internal/api/settings.go`
- [ ] T042 [US2] Ajouter `helloassoFailing` (carte activée et dernier test en échec ou incident plus récent que le dernier succès) à `get-membership-dashboard` in `internal/api/memberships_admin.go`
- [ ] T043 [P] [US2] Ajouter `useTestHelloAssoConnection` (invalidation des paramètres et du tableau de bord) in `web/app/api/settings.ts`
- [ ] T044 [US2] Ajouter la carte « Carte bancaire (HelloAsso) » avant les espèces in `web/app/routes/app/admin/settings-payments.tsx` : non configuré → explication des variables `FRIGG_HELLOASSO_*` lien vers la documentation d'installation et lien vers la documentation de HelloAsso (`https://dev.helloasso.com`, nouvel onglet annoncé), interrupteur désactivé avec explication liée ; configuré → badge `Mode test` (`warn`) ou `Production` (`ok`), slug, lien vers la documentation de HelloAsso pour obtenir les identifiants et déclarer l'adresse, adresse de notification en lecture seule avec bouton « Copier » (annonce « Adresse copiée »), bouton « Tester la connexion » (chargement, résultat `ok`/`danger` avec la cause traduite), interrupteur `Switch` activable seulement après un test réussi ; erreurs `422` sur l'interrupteur via `applyProblemToForm`
- [ ] T045 [US2] Encart « Connexion à HelloAsso en échec » menant à « Paramètres › Paiements » in `web/app/routes/app/admin/home.tsx`
- [ ] T046 [US2] Parcours Playwright + axe in `web/e2e/admin-settings.spec.ts` : section carte configurée (faux HelloAsso), copie de l'adresse, test réussi puis activation, test en échec, membre refusé ; aucune violation `serious` ou `critical`

**Checkpoint**: US1 + US2 forment la fonctionnalité utilisable de bout en bout.

---

## Phase 5: User Story 3 - Suivre les paiements par carte (administration) (Priority: P2)

**Goal**: le trésorier consulte les paiements par carte, vérifie une tentative en cours, traite les signalements et suit les chiffres par moyen.

**Independent Test**: adhésions payées par carte, par chèque et non payées : filtrer par moyen, chiffres par moyen, référence HelloAsso ; annuler une adhésion payée par carte → signalement, le traiter.

### Tests for User Story 3

- [ ] T047 [P] [US3] Tests d'API in `internal/api/memberships_admin_test.go` et `internal/api/payment_alerts_test.go` : `get-membership` renvoie `onlinePayments` (plus récente d'abord, `helloassoUrl`), `alerts`, `cardPaymentPending`, `actions.verifyCardPayment` et `cancelPayment = false` pour un paiement par carte ; `verify-card-payment` constate un paiement autorisé, `503` si HelloAsso injoignable ; `cancel-membership` d'une adhésion payée par carte → signalement `refund_due` dans la même transaction ; `resolve-payment-alert` → traité (date, auteur), déjà traité → `409`, inexistant → `404` ; `list-memberships` filtres `paidMethod=card` et `alerts=open`, recherche par numéro de commande HelloAsso (égalité exacte), `openAlerts` par élément ; `get-membership-stats` `collectedByMethod.card` et `collectedTest` (paiements sandbox exclus de `collected`) ; `get-membership-dashboard.paymentAlerts`

### Implementation for User Story 3

- [ ] T048 [US3] Étendre `AdminMembership` (`onlinePayments`, `alerts`, `cardPaymentPending` = tentative `pending` de moins de 60 min, `actions.verifyCardPayment`) et la construction de `helloassoUrl` (lien vers la commande dans l'espace HelloAsso de l'environnement ; forme exacte confirmée en environnement de test, sinon lien vers l'espace de l'association) in `internal/api/memberships_admin.go`
- [ ] T049 [US3] Implémenter `verify-card-payment` (`POST /admin/memberships/{id}/card-payment/verify`, `auth.Admin`) : `verifyOnlinePayment(…, "admin", "")` sur chaque tentative de `ListVerifiableOnlinePaymentsOfMembership`, `503` si HelloAsso injoignable, réponse `AdminMembership`, in `internal/api/card_payments.go`
- [ ] T050 [US3] Créer `refund_due` dans la transaction de `cancel-membership` lorsque l'adhésion est payée par carte, in `internal/api/memberships_admin.go`
- [ ] T051 [P] [US3] Implémenter `resolve-payment-alert` (`POST /admin/payment-alerts/{id}/resolve`, `auth.Admin`) avec journal `signalement de paiement` traité, réponse `AdminMembership`, in `internal/api/payment_alerts.go` et son enregistrement dans `internal/api/api.go`
- [ ] T052 [US3] Étendre `list-memberships` (filtres `paidMethod` dont `card`, `alerts=open`, recherche numérique aussi sur `online_payments.order_id`, `openAlerts`), `get-membership-stats` (`collectedByMethod.card`, `collectedTest`) et `get-membership-dashboard` (`paymentAlerts`), avec les requêtes correspondantes in `internal/database/queries/memberships.sql` (régénérer sqlc) et `internal/api/memberships_admin.go`
- [ ] T053 [P] [US3] Ajouter `useVerifyCardPayment` et `useResolvePaymentAlert` (invalidations : adhésion, liste, chiffres, tableau de bord) in `web/app/api/memberships.ts`
- [ ] T054 [US3] Bloc « Paiements en ligne » in `web/app/routes/app/admin/membership-detail.tsx` : « Payée — carte bancaire » avec badge `warn` « Mode test » si la tentative est `sandbox` (FR-015), date et heure, montant reçu, « Commande HelloAsso n° … » en lien externe vers la commande ou, à défaut, vers l'espace HelloAsso de l'association (`helloassoUrl`) (`target="_blank"`, `rel="noopener noreferrer"`, mention « nouvel onglet » pour les lecteurs d'écran, icône `ExternalLink`) ; tableau des tentatives (état en badge, montants, cause traduite) ; bouton « Vérifier auprès de HelloAsso » (`RefreshCw`) ; signalements en alertes `warn` (`TriangleAlert`) avec « Marquer comme traité » → `Dialog` de confirmation ; dans les dialogues d'annulation et de correction, avertissement « Le remboursement se fait dans l'espace HelloAsso de l'association. » si paiement par carte reçu ou `cardPaymentPending` ; « Annuler la constatation » masqué pour un paiement par carte
- [ ] T055 [US3] Filtres « Moyen » (dont « Carte bancaire ») et « Signalements », badge du nombre de signalements par ligne, badge `warn` « Mode test » pour une adhésion payée en mode test, filtres reflétés dans l'adresse, in `web/app/routes/app/admin/memberships.tsx` ; montant encaissé par moyen et « dont paiements de test » dans les chiffres de la saison
- [ ] T056 [US3] Encart « {n} signalement(s) de paiement à traiter » menant à la liste filtrée in `web/app/routes/app/admin/home.tsx`
- [ ] T057 [US3] Parcours Playwright + axe in `web/e2e/admin-memberships.spec.ts` : adhésion payée par carte (détail, lien HelloAsso), vérification d'une tentative en cours, annulation → signalement → traitement, filtre par moyen ; aucune violation `serious` ou `critical`

**Checkpoint**: les trois stories fonctionnent indépendamment.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T058 [P] Documenter la mise en place de HelloAsso (compte, client API avec le droit « Checkout », variables, environnement de test puis production, déclaration de l'adresse de notification, instance joignable en HTTPS) in `README.md`
- [ ] T059 [P] Test transversal des journaux : un parcours complet (lancement, retour, notification, test de connexion) ne journalise ni e-mail, ni nom, ni secret, ni jeton, ni `redirect_url`, in `internal/api/logs_test.go`
- [ ] T060 Session en environnement de test HelloAsso (quickstart.md › validation manuelle, étapes 1 à 9) : confirmer unités des montants, `state` d'un paiement refusé, emplacement de `metadata` dans les notifications `Order`/`Payment`, URL d'une commande, montant minimal, et rapprocher le montant encaissé par carte d'une saison du relevé HelloAsso (SC-006) ; ajuster `internal/helloasso`, `membership.MinCardAmount`, la construction de `helloassoUrl` et `specs/008-helloasso-card-payment/contracts/helloasso.md`
- [ ] T061 Revue de conformité (plan.md › points de vigilance 1 à 5) : seule `verifyOnlinePayment` écrit `paid_method = 'card'` ; aucun appel HelloAsso dans une transaction ; réponse de notification constante ; secret confiné à `internal/config` et `internal/helloasso`
- [ ] T062 Lancer `go tool task check` (build `CGO_ENABLED=0`, `go vet`, `go test ./...`, `sqlc generate`, `frigg openapi` et types sans diff, migrations sur base vierge, `bun run typecheck`, Biome, Vitest, `bun run build`, Playwright + axe) et corriger jusqu'au vert

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance. T004 avant T005, T006, T007, T008 ; T009 avant T010 et T012 (T011 est indépendante).
- **Foundational (Phase 2)** : dépend de la phase 1 (T008 pour le code sqlc, T009–T011 pour le client). Bloque toutes les stories.
- **US1 (Phase 3)** : dépend de la phase 2. MVP.
- **US2 (Phase 4)** : dépend de la phase 2 ; indépendante de US1 côté code (la page de paiement de US1 affiche la carte dès qu'elle est disponible).
- **US3 (Phase 5)** : dépend de la phase 2 et de `verifyOnlinePayment` (T027, US1) pour `verify-card-payment` ; le reste est indépendant.
- **Polish (Phase 6)** : après les stories voulues ; T060 de préférence après US1 et US2.

### User Story Dependencies

- **US1 (P1)** : aucune autre story ; tests avec carte activée en base (`apitest.EnableCard`).
- **US2 (P1)** : aucune autre story.
- **US3 (P2)** : T049 réutilise T027 ; les signalements sont créés par US1 (constatation) et par T050.

### Within Each User Story

- Tests d'abord (ils doivent échouer), puis requêtes/domaine, puis handlers, puis frontend, puis Playwright.
- `frigg openapi` et la régénération des types après chaque ajout d'opération, avant le frontend.

### Parallel Opportunities

- Phase 1 : T002, T003 en parallèle de T004 ; T005–T007 après T004 ; T009–T011 en parallèle du schéma.
- Phase 2 : T014–T017 et T019 en parallèle après T013/T018.
- US1 : T022–T026 en parallèle ; T034, T035, T037 en parallèle des handlers.
- US2 et US3 peuvent avancer en parallèle de US1 une fois la phase 2 terminée (sauf T049).

---

## Parallel Example: User Story 1

```text
# Tests (fichiers distincts) :
T022/T023 internal/api/card_payments_test.go
T024      internal/api/helloasso_webhook_test.go
T025      internal/api/online_sweeper_test.go
T026      internal/api/memberships_isolation_test.go + access_test.go

# Frontend pendant l'implémentation des handlers :
T034 web/app/api/memberships.ts
T035 web/app/memberships/CardPaymentPanel.tsx
T037 web/e2e/fake-helloasso.ts
```

## Parallel Example: User Stories 2 et 3

```text
# Après la phase 2, une personne sur US2 :
T039 → T040 → T041 → T042 → T043/T044 → T045 → T046
# une autre sur US3 (hors T049, qui attend T027) :
T047 → T048 → T050/T051 → T052 → T053 → T054/T055/T056 → T057
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 (configuration, migration, client) puis phase 2 (domaine, câblage).
2. US1 : paiement, constatation sur retour, notification et balayage.
3. **STOP et valider** : tests d'API et Playwright avec le faux HelloAsso ;
   carte activée en base.

### Incremental Delivery

1. US1 → paiement par carte fonctionnel (activation technique).
2. US2 → activation par un administrateur depuis « Paramètres » : la
   fonctionnalité devient livrable aux associations.
3. US3 → suivi, signalements, chiffres par moyen.
4. T060 (session en environnement de test HelloAsso) avant toute mise en
   production réelle.

### Notes

- Chaque tâche se termine par ses tests verts et un commit (gitmoji, message
  court, sans co-auteur, après validation de l'utilisateur).
- Si l'on établit qu'aucune instance n'a jamais été déployée, T004 peut
  modifier `00008` au lieu de reconstruire `memberships` (constitution,
  migrations append-only) ; T005 s'adapte alors.
