---

description: "Liste des tâches : parcours publics (adhésion et inscription sans compte)"
---

# Tasks: Parcours publics (adhésion et inscription sans compte)

**Input**: Design documents from `specs/016-public-journeys/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution v1.9.0 les impose :

- matrice d'autorisation étendue au niveau `link` (principes II et III) ;
- tests d'API sur base isolée ;
- Playwright + axe sur chaque parcours touché.

Plusieurs points ont aussi leurs propres tests :

- la non-révélation (SC-005) ;
- l'isolation d'un lien (SC-007) ;
- la capacité sous concurrence (SC-006) ;
- la migration et sa sonde (R4).

**Organization**: une phase par user story, dans l'ordre de la spec :

- **US1 (P1)** : la demande d'adhésion publique, avec la confirmation, le
  rattachement à un compte existant et le rattachement d'un doublon ;
- **US2 (P1)** : le suivi et le paiement par lien, avec les notifications
  unifiées et les fiches importées ;
- **US3 (P2)** : l'événement public ;
- **US4 (P3)** : la création de compte ;
- **US5 (P3)** : la supervision, avec l'origine, les filtres et les fiches
  inactives.

**Conventions** (constitution 1.9.0) :

- Identifiants, tables, champs JSON et valeurs d'énumération en anglais
  (`public_submissions`, `via_public`, `tracking`). Commentaires, messages,
  textes d'interface, e-mails et journaux en français.
- Commentaires de la migration et des requêtes sqlc en **ASCII uniquement**.
- Jamais d'adresse électronique, de nom, de jeton ni de slug dans les journaux
  (R16).
- Aucun test de rôle ni de lien dans les handlers. Le niveau d'accès est
  déclaré par `register`, et le périmètre d'un lien passe par les requêtes
  `…OfTrackedEmail`.
- Toutes les écritures en `s.Store.WriteTx` (`BEGIN IMMEDIATE`), avec les
  e-mails insérés dans la même transaction.
- Le jeton n'apparaît **jamais** dans un chemin ni dans une requête HTTP
  envoyés au serveur : il passe toujours dans un corps JSON (R1).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US5)

## Path Conventions

Le module Go est à la racine (`internal/`) et la SPA dans `web/`
(plan.md › Project Structure). Code existant à réutiliser :

- **API** :
  - `internal/api/api.go` : `register`, `accessMiddleware`, `server`,
    `s.now()`, `s.today()`, `s.timestamp()`, `CurrentAccount`, `ownAccount`,
    `clientIP`, `s.internalError` ;
  - `internal/api/errors.go` : `FieldErrors` ;
  - `internal/api/mail.go` : `enqueueNotice` ;
  - `internal/api/memberships.go` : `seasonIndex`, `notifyMembership`,
    `noticeRecipient`, `formatEuros`, `frenchDate`, `methodLabels`, `ns`,
    `nid` ;
  - `internal/api/memberships_me.go` : `MyMembership`, annonce d'un moyen,
    annulation ;
  - `internal/api/memberships_admin.go` : `AdminMembership`,
    `getMembershipDashboard` ;
  - `internal/api/card_payments.go` : `openCheckout`, `confirmMyCardPayment`,
    `verifyOnlinePayment`, `cardItemName`, `resumeWindow` ;
  - `internal/api/pay_targets.go`, `internal/api/registration_target.go` :
    `payTarget` ;
  - `internal/api/registrations_me.go`, `internal/api/registrations_admin.go`
    (`contactEmail`) et `internal/api/event_notices.go` ;
  - `internal/api/events_admin.go`, `internal/api/events_me.go` ;
  - `internal/api/members_admin.go` ;
  - `internal/api/general_settings.go`, `internal/api/identity.go` ;
  - `internal/api/auth.go` : `acceptInvitation`, modèle de création de compte
    et de connexion ;
  - `internal/api/online_sweeper.go` : modèle de balayeur ;
  - `internal/cli/serve.go` : démarrage des balayeurs.
- **Domaine** :
  - `internal/auth` : `Access`, `Allows`, `NewToken`, `HashToken`,
    `NormalizeEmail`, `ValidateEmail`, `ValidatePassword`, `HashPassword`,
    `Login`, `Limits`, `KeyedLimiter` ;
  - `internal/member` : `Validate`, `IdentityKey`, `NormalizePhone` ;
  - `internal/customfield` : `Apply`, `Owner`, `Admin`, `CanRead`,
    `NormalizeValue` ;
  - `internal/membership` : `Snapshot`, `CanPay`, `CanPayByCard`,
    `NextReference` ;
  - `internal/event` : `Openness`, `Eligibility` ;
  - `internal/season` : `Today`, `Date`.
- **Base** :
  - `internal/database` : `Store.Read`, `Store.WriteTx` ;
  - migrations `00007_members.sql` (CHECK d'adresse), `00012_events.sql`,
    `00013_payment_adjustments.sql` ;
  - `internal/database/db_test.go` : modèle `TestCustomFieldsMigration`,
    avec `goose.NewProvider` et une base peuplée.
- **Mail** : `internal/mail/render.go` (`Data`, `Render`),
  `internal/mail/templates/*.tmpl`, `internal/mail/render_test.go`.
- **Tests Go** :
  - `internal/api/apitest` : `New`, `Seed`, `LoggedIn`, `SetToday`, `Logs`,
    `Client().Do`, `Response.JSON`, Mailpit simulé par la table
    `outbound_emails` ;
  - `internal/api/access_test.go` : `TestAccessMatrix`, `allowedPublic` ;
  - `internal/api/memberships_helpers_test.go`,
    `internal/api/events_helpers_test.go`,
    `internal/api/card_payments_test.go` (`newCardFixture`) ;
  - `internal/helloasso/hatest`.
- **Frontend** :
  - `web/app/routes.ts` ;
  - `web/app/routes/auth/layout.tsx` (habillage hors session),
    `web/app/routes/auth/accept-invitation.tsx` (mot de passe) ;
  - `web/app/api/*` : `client.ts`, `queries.ts`, `problem.ts`,
    `identity.ts` ;
  - `web/app/ui/*` : `Button`, `Field`, `Checkbox`, `RadioGroup`, `Badge`,
    `Alert`, `ConfirmDialog`, `Table`, `Icon` ;
  - `web/app/payments/PaymentPanel.tsx` : `PaymentSection` (props
    `announce`, `startCard`, `refresh`), `CardReturn` (prop `confirm`) ;
  - `web/app/customfields/*` ;
  - `web/app/routes/app/member/membership-request.tsx` : formulaire de
    demande à imiter ;
  - `web/app/lib/format.ts`.
- **E2E** : `web/e2e/test.ts`, `axe.ts`, `mailpit.ts`, `fake-helloasso.ts`,
  `memberships.spec.ts`, `card-payment.spec.ts`.
- **Documentation** : `docs/usage/*.md`, `docs/installation/*.md`
  (VitePress, 015).

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: schéma, sonde de migration et requêtes de base communes à toutes les stories.

- [X] T001 Créer `internal/database/migrations/00014_public_journeys.sql`. Elle est transactionnelle (pas de `NO TRANSACTION`), ses commentaires sont en ASCII, et elle contient les éléments suivants (data-model.md) :
  - **Table `public_submissions` (STRICT)** :
    - `id INTEGER PRIMARY KEY` ;
    - `kind TEXT NOT NULL CHECK (kind IN ('membership','registration'))` ;
    - `season_id INTEGER REFERENCES seasons (id) ON DELETE CASCADE` ;
    - `fee_id INTEGER REFERENCES membership_fees (id) ON DELETE CASCADE` ;
    - `declared_kind TEXT CHECK (declared_kind IN ('first','renewal'))` ;
    - `event_id INTEGER REFERENCES events (id) ON DELETE CASCADE` ;
    - le `CHECK` de cohérence : `(kind = 'membership' AND season_id IS NOT NULL AND fee_id IS NOT NULL AND declared_kind IS NOT NULL AND event_id IS NULL) OR (kind = 'registration' AND event_id IS NOT NULL AND season_id IS NULL AND fee_id IS NULL AND declared_kind IS NULL)` ;
    - `first_name` et `last_name` `TEXT NOT NULL CHECK (length(...) BETWEEN 1 AND 80)` ;
    - `birth_date TEXT NOT NULL CHECK (birth_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]')` ;
    - `email TEXT NOT NULL CHECK (length(email) BETWEEN 3 AND 254)` ;
    - `phone`, `address_line1`, `address_line2`, `postal_code` et `city`, avec les mêmes `CHECK` que `members` dans `00007_members.sql`, y compris le `CHECK` d'adresse tout ou rien ;
    - `custom_values TEXT NOT NULL DEFAULT '[]'` ;
    - `create_account INTEGER NOT NULL DEFAULT 0 CHECK (create_account IN (0,1))` ;
    - `consented_at TEXT NOT NULL` ;
    - `token_hash BLOB NOT NULL UNIQUE` ;
    - `expires_at TEXT NOT NULL` ;
    - `created_at TEXT NOT NULL` ;
    - l'index `public_submissions_created_at (created_at)`.
  - **Table `tracking_links` (STRICT)** :
    - `id INTEGER PRIMARY KEY` ;
    - `email TEXT NOT NULL CHECK (length(email) BETWEEN 3 AND 254)` ;
    - `token_hash BLOB NOT NULL UNIQUE` ;
    - `expires_at TEXT NOT NULL` ;
    - `revoked_at TEXT` ;
    - `last_used_at TEXT` ;
    - `created_at TEXT NOT NULL` ;
    - les index `tracking_links_email (email)` et `tracking_links_expires_at (expires_at)`.
  - **Colonnes de `members`** :
    - `ALTER TABLE members ADD COLUMN origin TEXT NOT NULL DEFAULT 'internal' CHECK (origin IN ('internal','public'))` ;
    - `email_confirmed_at TEXT` ;
    - `consented_at TEXT` ;
    - l'index `members_tracked_email ON members (email COLLATE NOCASE) WHERE account_id IS NULL AND email IS NOT NULL` ;
    - l'index `members_origin_created ON members (origin, created_at) WHERE origin = 'public'`.
  - **Colonne de `memberships`** : `ALTER TABLE memberships ADD COLUMN via_public INTEGER NOT NULL DEFAULT 0 CHECK (via_public IN (0,1)) CHECK (via_public = 0 OR (requested_by IS NULL AND created_by IS NULL AND declared_kind IS NOT NULL))`.
  - **Colonne de `event_registrations`** : `ADD COLUMN via_public INTEGER NOT NULL DEFAULT 0 CHECK (via_public IN (0,1)) CHECK (via_public = 0 OR registered_by IS NULL)`.
  - **Colonne de `online_payments`** : `ADD COLUMN via_link INTEGER NOT NULL DEFAULT 0 CHECK (via_link IN (0,1)) CHECK (via_link = 0 OR account_id IS NULL)`.
  - **Colonne d'`events`** : `ADD COLUMN public_slug TEXT CHECK (public_slug IS NULL OR (length(public_slug) = 22 AND openness = 'all_members'))`, plus `CREATE UNIQUE INDEX events_public_slug ON events (public_slug) WHERE public_slug IS NOT NULL`.
  - **Colonne d'`association`** : `ADD COLUMN public_membership_enabled INTEGER NOT NULL DEFAULT 0 CHECK (public_membership_enabled IN (0,1))`.
  - **Section `Down`** : elle supprime les index, les colonnes (`DROP COLUMN`) et les deux tables, dans l'ordre inverse.
- [X] T002 Ajouter `TestPublicJourneysMigration` à `internal/database/db_test.go`, sur le modèle de `TestCustomFieldsMigration`. Le test migre jusqu'à `00013`, peuple une fiche, une adhésion payée par carte, une inscription et un événement `all_members`, applique `00014`, puis vérifie :
  - que les données sont conservées et que les défauts valent `origin='internal'`, `via_public=0`, `via_link=0`, `public_membership_enabled=0` ;
  - que `PRAGMA foreign_key_check` est vide ;
  - **la sonde de R4** : un `UPDATE memberships SET via_public = 1` sur une ligne dont `requested_by` n'est pas `NULL` échoue sur le `CHECK`, et un `public_slug` posé sur un événement `season_members` échoue aussi ;
  - que `Down` puis `Up` fonctionnent.

  Si la sonde révèle qu'un `CHECK` multi-colonnes n'est pas accepté en `ADD COLUMN`, retirer ces `CHECK` de T001, garder l'invariant dans le domaine (T026, T028, T057) et le noter dans plan.md › Complexity Tracking.
- [X] T003 [P] Créer `internal/database/queries/public_submissions.sql` (ASCII) avec les requêtes suivantes :
  - `CreatePublicSubmission` (toutes les colonnes) ;
  - `GetPublicSubmissionByTokenHash` ;
  - `DeletePublicSubmission` ;
  - `PurgePublicSubmissions(before)`, qui supprime les dépôts dont `created_at < before` ;
  - `CountMembershipFeeForPublic(season_id, fee_id)`, qui vérifie que le tarif appartient à la saison.
- [X] T004 [P] Créer `internal/database/queries/tracking_links.sql` (ASCII) avec les requêtes suivantes :
  - `CreateTrackingLink` ;
  - `GetTrackingLinkByTokenHash` ;
  - `GetActiveTrackingLink(id, now)`, qui ne renvoie le lien que s'il n'est pas révoqué et que `expires_at > now` ;
  - `TouchTrackingLink(id, now)`, à appeler au plus une fois par minute ;
  - `RevokeTrackingLinksOfEmail(email, now)`, avec une comparaison `COLLATE NOCASE` ;
  - `PurgeTrackingLinks(before)`, qui supprime les liens dont `expires_at < before`.
- [X] T005 [P] Ajouter à `internal/database/queries/members.sql` (ASCII) les requêtes suivantes :
  - `ListTrackedMembers(email)` : fiches dont `account_id IS NULL AND email = ? COLLATE NOCASE`, triées par nom puis prénom ;
  - `GetTrackedMember(id, email)` ;
  - `CountTrackedMembers(email)` ;
  - `FindTrackedMemberByIdentity(email, birth_date)`, pour la déduplication ;
  - `FindAccountMemberByIdentity(account_id, birth_date)` ;
  - `CreatePublicMember` : fiche avec `origin='public'`, `email_confirmed_at` et `consented_at`, et `account_id` éventuel ;
  - `ResetMemberEmailConfirmation(id)`.
- [X] T006 Lancer `sqlc generate` et vérifier que `go build ./...` passe (`internal/database/sqlc/`).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: niveau d'accès `link`, limiteurs, domaine `internal/public`, ouverture d'un lien, balayeur, modèles d'e-mail et layout public. Aucune story ne peut démarrer avant.

- [X] T007 [P] Dans `internal/auth/access.go`, ajouter `Link Access = 3` avec `String()` → `"link"`. `Allows(role, Link)` renvoie toujours `false`, et le classement `Public < Member < Admin` reste inchangé. Couvrir par un test en table dans `internal/auth/access_test.go` : `Allows("admin", Link) == false`, et les cas existants inchangés.
- [X] T008 [P] Dans `internal/auth/ratelimit.go`, ajouter à `Limits` et à `NewLimits` neuf limiteurs (research.md › R9 ; principe VII : chaque parcours public par IP **et** par adresse visée) :
  - `PublicSubmitIP` : `NewKeyedLimiter(10, time.Hour)` ;
  - `PublicMailEmail` : `NewKeyedLimiter(5, time.Hour)`, clé = adresse normalisée ;
  - `PublicConfirmIP` : `NewKeyedLimiter(20, time.Hour)` ;
  - `PublicConfirmEmail` : `NewKeyedLimiter(10, time.Hour)`, clé = adresse du dépôt ;
  - `TrackingRequestIP` : `NewKeyedLimiter(10, time.Hour)` ;
  - `TrackingOpenIP` : `NewKeyedLimiter(30, time.Hour)` ;
  - `TrackingOpenEmail` : `NewKeyedLimiter(30, time.Hour)`, clé = adresse du lien ;
  - `TrackingUseIP` : `NewKeyedLimiter(120, time.Minute)` ;
  - `TrackingUseEmail` : `NewKeyedLimiter(120, time.Minute)`, clé = adresse du lien.

  Les ajouter aussi à la liste de `RunPurge`, et étendre `internal/auth/ratelimit_test.go`.
- [X] T009 [P] Créer `internal/public/limits.go` (paquet `public`, sans base) avec `ConfirmTTL = 48 * time.Hour`, `SubmissionRetention = 7 * 24 * time.Hour`, `LinkTTL = 30 * 24 * time.Hour`, `LinkPurgeGrace = 24 * time.Hour`, `InactiveAfterMonths = 12` et `MaxManagedMembers = 20` (005 FR-004). Chaque constante porte un commentaire qui renvoie à l'exigence (FR-008, FR-009, FR-013, FR-041).
- [X] T010 [P] Créer `internal/public/submission.go` :
  - `type Submission` : identité, coordonnées, adresse, consentement, `CreateAccount`, `Website` ;
  - `Honeypot(website string) bool`, vrai si la valeur n'est pas vide ;
  - `Validate(in, today) (member.Clean, FieldErrors)` : compose `member.Validate` et exige une adresse électronique (« L'adresse électronique est obligatoire. ») ainsi que le consentement (« Vous devez accepter le traitement de vos données. », sur `consent`) ;
  - `ValidateMembershipTarget(declaredKind)` : `first` ou `renewal`.

  Ajouter des tests en table dans `internal/public/submission_test.go` : adresse absente, consentement absent, piège rempli, adresse normalisée en minuscules avec `auth.NormalizeEmail`.
- [X] T011 [P] Créer `internal/public/confirm.go` (research.md › R7).
  - Il définit `type State struct` avec les champs `AccountFound`, `AccountActive`, `AccountManaged int`, `ExistingMember bool`, `ExistingActive bool` (adhésion en attente ou validée sur la saison, ou inscription confirmée à l'événement) et `TargetOpen bool`, `ClosedReason string`.
  - `Decide(State) Outcome` renvoie l'un des quatre cas suivants :
    - `Closed` si `!TargetOpen` ;
    - `Duplicate` si `ExistingMember && ExistingActive` ;
    - `Account` si `AccountFound && AccountActive && (ExistingMember || AccountManaged < MaxManagedMembers)` ;
    - `Tracking` sinon.
  - Ajouter des tests en table dans `internal/public/confirm_test.go` : compte désactivé, compte à 20 fiches et fiche existante dans un compte plein donnent respectivement `Tracking`, `Tracking` et `Account`.
- [X] T012 Étendre `accessMiddleware` dans `internal/api/api.go` pour le niveau `auth.Link` :
  - limiter avec `TrackingUseIP` par `clientIP`, avec un `429` neutre en cas de dépassement ;
  - lire `tracking_link_id` dans la session scs (`s.Sessions.GetInt64`) ;
  - sinon, ou si `GetActiveTrackingLink` ne renvoie rien, répondre `401` avec « Ce lien n'est plus valable. » ;
  - appeler `TouchTrackingLink` au plus une fois par minute ;
  - limiter avec `TrackingUseEmail` sur l'adresse du lien, juste après sa lecture, avec un `429` neutre en cas de dépassement ;
  - placer l'adresse du lien dans le contexte (nouvelle clé `trackedEmailKey`) et l'exposer par `trackedEmail(ctx) string` ;
  - **ne jamais** consulter la session de compte pour ce niveau.

  Dans `register`, pour `auth.Link`, poser `op.Security` sur le schéma `session`.
- [X] T013 Créer `internal/api/tracking_links.go` avec trois fonctions :
  - `issueTrackingLink(ctx, q, email) (raw string, err)` : il crée le lien avec `auth.NewToken`, `expires_at = now + public.LinkTTL` et l'adresse normalisée, puis journalise `tracking_link_issued` avec `tracking_link_id` seulement ;
  - `openTrackingInSession(ctx, linkID)` : `s.Sessions.RenewToken` puis `Put("tracking_link_id", id)`, sans toucher aux clés de compte ;
  - `revokeTrackingLinks(ctx, q, email)`.

  Enregistrer aussi deux opérations :
  - `open-tracking-link` (`auth.Public`, `POST /public/tracking/open`, corps `{ token }`) : limitée par `TrackingOpenIP` puis, après la lecture du lien, par `TrackingOpenEmail` sur son adresse (`429` neutre), elle renvoie `410` (« Ce lien n'est plus valable. ») si le lien est inconnu, révoqué ou expiré, et `204` sinon, avec l'accès ouvert ;
  - `close-tracking` (`auth.Link`, `POST /tracking/close`) : elle retire seulement `tracking_link_id`, avec `s.Sessions.Remove`, puis renvoie `204`.

  Ajouter `registerTracking(api)` dans `New`.
- [X] T014 Créer `internal/api/public_sweeper.go` : `PublicSweeper`, sur le modèle d'`OnlinePaymentSweeper`, avec un intervalle de 10 minutes. Son `Tick` purge les dépôts dont `created_at < now - SubmissionRetention` et les liens dont `expires_at < now - LinkPurgeGrace`, puis journalise les nombres. Le démarrer dans `internal/cli/serve.go` et l'exposer dans `internal/api/apitest/apitest.go` comme les autres balayeurs. Ajouter un test dans `internal/api/public_sweeper_test.go`, avec une horloge injectée : un dépôt de 8 jours et un lien expiré depuis 2 jours sont purgés, un dépôt de 6 jours est conservé.
- [X] T015 Étendre `internal/api/apitest/apitest.go` avec deux helpers :
  - `env.TrackingClient(email string) *Client` : il émet un lien en base, puis appelle `POST /api/public/tracking/open` avec le jeton et renvoie le client muni du cookie ;
  - `env.Mails(to string) []Mail` : il lit `outbound_emails` et extrait le lien (`Link`) et le jeton du fragment `#jeton=`.

  Étendre `TestAccessMatrix` dans `internal/api/access_test.go` :
  - pour `access == auth.Link`, l'anonyme, le membre et l'administrateur sans lien reçoivent `401` ;
  - un client `TrackingClient` ne reçoit jamais `401` ni `403` ;
  - un lien révoqué (`RevokeTrackingLinksOfEmail`) reçoit `401` ;
  - ajouter à `allowedPublic` `"open-tracking-link": true`, avec le commentaire « Accès par lien personnel (amendement 1.9.0 de la constitution). ».
- [X] T016 [P] Créer les modèles `internal/mail/templates/public-confirm.{txt,html}.tmpl` et `tracking-link.{txt,html}.tmpl`, dans le layout existant et en français :
  - **`public-confirm`** confirme une adresse et contient le lien `{{.Link}}`, valable 48 h. Ajouter les champs booléens `AccountExists` et `AccountFull` à `mail.Data` dans `internal/mail/render.go` (research.md › R7, « E-mail de confirmation du dépôt ») :
    - `AccountExists` (compte actif qui gère moins de 20 fiches) : l'e-mail ajoute « Un compte Frigg existe déjà pour cette adresse : après confirmation, votre demande apparaîtra dans votre espace membre. » ;
    - `AccountFull` (compte actif qui gère déjà 20 fiches) : l'e-mail ajoute « Votre compte gère déjà 20 fiches : cette demande ne pourra pas y être ajoutée et sera suivie par un lien personnel. » ;
    - compte désactivé ou absent : aucune mention de compte.
  - **`tracking-link`** : « Votre lien de suivi », valable 30 jours.

  Ajouter les cas dans `internal/mail/render_test.go`.
- [X] T017 [P] Créer le layout public `web/app/routes/public/layout.tsx`, qui reprend l'habillage de `routes/auth/layout.tsx` : identité par `get-identity`, carte centrée, sans barre latérale. Créer aussi les composants suivants :
  - `web/app/public/useFragmentToken.ts` : il lit `#jeton=` au premier rendu, l'efface aussitôt avec `history.replaceState` (chemin et requête conservés) et renvoie la valeur ;
  - `web/app/public/Honeypot.tsx` : un champ `website` dans un conteneur `aria-hidden="true"` hors écran, avec `tabIndex={-1}` et `autoComplete="off"` (R10) ;
  - `web/app/public/ConsentField.tsx` : la case obligatoire et le texte de FR-007, avec le nom de l'association ;
  - `web/app/public/PublicIdentityFields.tsx` : prénom, nom, date de naissance, adresse obligatoire, téléphone et adresse postale repliable, avec les règles de 005 reprises de `members/`.

  Ajouter les tests Vitest `web/app/public/useFragmentToken.test.ts` (fragment lu puis effacé, absent → `null`) et `Honeypot.test.tsx` (absent de l'arbre d'accessibilité).
- [X] T018 Ajouter dans `web/app/routes.ts` un bloc `layout("routes/public/layout.tsx", [...])`, avant le layout `app`. Il est vide pour l'instant ; chaque story y ajoute ses routes. Le placer pour que `routes/app/layout.tsx` ne redirige jamais ces chemins vers `/connexion`.

**Checkpoint** : `go test ./...` vert. Une opération `link` n'est accessible qu'avec un lien ouvert. Le layout public est rendu.

---

## Phase 3: User Story 1 - Demander une adhésion sans compte (Priority: P1) 🎯 MVP

**Goal**: une personne non connectée dépose une demande d'adhésion sur `/adherer` et confirme son adresse. La demande apparaît alors aux administrateurs avec l'origine « Publique ». Elle est rattachée d'office à un compte existant de même adresse, et un doublon peut être rattaché à une fiche existante.

**Independent Test**: activer l'accueil, déposer une demande en navigation privée, confirmer l'adresse par l'e-mail, puis valider la demande dans Admin › Adhésions (quickstart.md › scénarios 1, 2, 5, 8).

### Tests for User Story 1

- [X] T019 [P] [US1] Créer `internal/api/public_membership_test.go` pour `get-public-membership-page` et `submit-public-membership` :
  - **accueil fermé par défaut** : `open:false` et `seasons:[]`, et le dépôt renvoie `409` ;
  - **accueil ouvert** : seulement les saisons ouvertes, non closes, avec tarif, et seulement les champs « Rempli par le membre » actifs, sans aucun champ `admin_only` ni `member_visible` ;
  - **dépôt valide** : `202` avec le corps neutre exact de contracts/api.md, une ligne `public_submissions`, un e-mail `public-confirm`, et aucune fiche ni adhésion ;
  - **`422`** sur une adresse absente, l'absence de consentement ou un tarif d'une autre saison ;
  - **piège rempli** : `202` identique, sans ligne ni e-mail, avec le journal `public_submit_rejected` et `reason=honeypot` ;
  - **limite par adresse** : 6 dépôts produisent 5 e-mails et 6 réponses `202` identiques ;
  - **limite par IP** : le 11ᵉ dépôt renvoie `429`.
- [X] T020 [P] [US1] Créer `internal/api/public_nonrevelation_test.go` : pour une adresse inconnue, l'adresse d'un compte et l'adresse d'une fiche sans compte, les réponses de `submit-public-membership` sont identiques octet pour octet (statut, corps et en-têtes hors `Date` et cookie de session) (SC-005).
- [X] T021 [P] [US1] Créer `internal/api/public_confirm_test.go` pour `confirm-public-submission`, côté adhésion :
  - **jeton valide** : une fiche `origin='public'`, sans compte, avec l'adresse confirmée, `email_confirmed_at` et `consented_at` ; une adhésion `pending` avec `via_public=1`, `requested_by` et `created_by` à `NULL` et `declared_kind` ; les valeurs des champs personnalisés ; le dépôt supprimé ; un e-mail `tracking-link` ; `outcome:"tracking"` ; l'accès par lien ouvert dans la session ;
  - **jeton réutilisé ou expiré** (`SetToday` + 49 h) : `410` ;
  - **limite par adresse** : 11 confirmations pour la même adresse en une heure, depuis des IP différentes, donnent un `429` (`PublicConfirmEmail`) ;
  - **conditions revérifiées** : accueil désactivé ou saison close entre le dépôt et la confirmation donnent `closed` avec la raison ;
  - **adresse d'un compte actif** : fiche rattachée au compte, `outcome:"account"`, sans lien émis ;
  - **compte désactivé, ou compte qui gère déjà 20 fiches** : `tracking`. L'e-mail de dépôt ne mentionne pas de compte pour le compte désactivé, et porte la variante `AccountFull` pour le compte plein ; la réponse à l'écran du dépôt reste identique dans les deux cas ;
  - **même identité et même adresse** avec une demande en attente : `duplicate`, sans création ;
  - **champ archivé ou ajouté entre-temps** : la valeur est ignorée et le champ n'est pas exigé.
- [X] T022 [P] [US1] Créer `internal/api/memberships_attach_test.go` pour `attach-membership-to-member` :
  - **rattachement réussi** : l'adhésion est déplacée, l'adresse est copiée si la cible n'en a pas, les valeurs des champs de membre sont copiées là où la cible n'en a pas, et la fiche publique vide est supprimée ;
  - **`409`** si la cible a déjà une adhésion active sur la saison (message de 006 FR-003), si la demande n'est pas publique, ou si elle n'est pas en attente ;
  - **membre** : `403`.

  Tester aussi que `get-membership` renvoie `duplicateCandidates` pour une demande publique en attente dont l'identité correspond (005 FR-005).

### Implementation for User Story 1

- [X] T023 [P] [US1] Étendre `internal/database/queries/association.sql`, `GetGeneralSettings` et `UpdateGeneralSettings` avec `public_membership_enabled`. Dans `internal/api/general_settings.go` :
  - ajouter `publicMembershipEnabled: bool` en lecture et en écriture ;
  - en lecture, ajouter `publicMembershipUrl = s.Config.BaseURL + "/adherer"` ;
  - journaliser la modification (FR-004).

  Étendre `internal/api/general_settings_test.go` en conséquence.
- [X] T024 [US1] Créer `internal/api/public.go` avec `registerPublic(api)`, appelé dans `New`, et l'opération `get-public-membership-page` (`auth.Public`, `GET /public/membership`).
  - Elle renvoie `PublicMembershipPage` (contracts/api.md) : `open` vaut vrai si le paramètre est activé et qu'au moins une saison ouverte, non close, porte un tarif.
  - Les saisons sont lues avec `seasonIndex`. Chaque tarif donne `label`, `description`, `firstAmountCents` et `renewalAmountCents`.
  - Les champs viennent des définitions actives dont la visibilité est « Rempli par le membre », filtrées par `customfield.CanWrite(d, customfield.Owner)`.

  Ajouter `"get-public-membership-page"` à `allowedPublic` dans `internal/api/access_test.go`.
- [X] T025 [US1] Dans `internal/api/public.go`, ajouter `submit-public-membership` (`auth.Public`, `POST /public/membership`, corps de contracts/api.md), qui suit cet ordre :
  1. limite `PublicSubmitIP`, avec `429` neutre ;
  2. piège : réponse `202` neutre sans écriture ;
  3. `409` (« Les demandes d'adhésion ne sont pas ouvertes. ») si l'accueil est fermé ;
  4. `public.Validate`, avec erreurs `422` ;
  5. vérification que le tarif appartient à la saison ouverte ;
  6. `customfield.Apply`, avec les définitions de membre et d'adhésion et l'audience `Owner`, et les obligatoires exigés ; sérialisation canonique dans `custom_values` ;
  7. limite `PublicMailEmail` sur l'adresse normalisée. Au-delà, réponse `202` neutre sans écriture, avec le journal `reason=rate_email` ;
  8. en `WriteTx` : `CreatePublicSubmission` avec `auth.NewToken`, `expires_at = now + public.ConfirmTTL` et `consented_at = now`, puis `enqueueNotice("public-confirm", email, Data{AccountExists, AccountFull}, "/confirmer-demande#jeton="+raw)`. `AccountExists` vaut vrai seulement pour un compte actif qui gère moins de 20 fiches, `AccountFull` pour un compte actif qui en gère 20 ; un compte désactivé est traité comme une adresse sans compte.

  L'e-mail est mis en file dans tous les cas, par le même chemin. Réponse : `202` avec `{message}`. Ajouter l'opération à `allowedPublic`.
- [X] T026 [US1] Créer `internal/api/public_confirm.go` avec `confirm-public-submission` (`auth.Public`, `POST /public/submissions/confirm`, `{ token }`), limitée par `PublicConfirmIP`. Elle fait, en `WriteTx` (research.md › R7) :
  1. relire le dépôt par `HashToken`, ou renvoyer `410` (« Ce lien n'est plus valable. ») s'il est absent ou expiré ; puis consommer `PublicConfirmEmail` sur l'adresse du dépôt, avant toute écriture (`429` neutre en cas de dépassement) ;
  2. construire `public.State` : le compte par `NormalizeEmail` (actif, nombre de fiches gérées), une fiche identique (`FindTrackedMemberByIdentity` comparé par `member.IdentityKey`, ou `FindAccountMemberByIdentity`), une adhésion active, et les conditions de la cible ;
  3. appliquer `public.Decide`. Pour `Tracking` et `Account` : créer ou réutiliser la fiche (`CreatePublicMember`, avec `account_id` pour `Account`), créer l'adhésion en attente (`via_public=1`, `declared_kind`), réappliquer `custom_values` aux définitions courantes et supprimer le dépôt ;
  4. envoyer l'e-mail : pour `Tracking`, `issueTrackingLink` et le modèle `tracking-link` avec le chemin `/suivi#jeton=…` ; pour `Account`, aucun lien ;
  5. supprimer le dépôt aussi en cas de `Closed` ou de `Duplicate`.

  Après la transaction, pour `Tracking`, appeler `openTrackingInSession`. La réponse est `ConfirmOutcome`, avec `createAccount` repris du dépôt. Journaliser `public_confirm` avec `outcome` et les identifiants. Ajouter l'opération à `allowedPublic`.
- [X] T027 [US1] Dans `internal/api/memberships_admin.go` et `memberships.go`, ajouter à `AdminMembership` :
  - `viaPublic` ;
  - l'auteur affiché « Demande publique » quand `via_public = 1`, avant toute déduction depuis `requested_by` ou `created_by` ;
  - `duplicateCandidates: MemberRef[]`, calculés pour une demande publique en attente avec la requête de doublon de 005 (`members_birth_date`, `member.IdentityKey`), en excluant la fiche elle-même.

  Ajouter le filtre `origin=public` à `list-memberships` (requête sqlc statique, avec un paramètre `origin` nullable) dans `internal/database/queries/memberships.sql`.
- [X] T028 [US1] Ajouter `attach-membership-to-member` (`auth.Admin`, `POST /admin/memberships/{id}/attach`, `{ memberId }`) dans `internal/api/memberships_admin.go`, avec des requêtes `MoveMembershipToMember` et `DeleteMemberIfEmpty` dans `memberships.sql` et `members.sql`. Elle fait, en `WriteTx` :
  - vérifier que la demande est `via_public = 1` et `pending`, ou renvoyer `409` ;
  - déplacer la demande, en traduisant la violation de `memberships_one_active` en `409` avec le message de 006 FR-003 ;
  - copier l'adresse et les valeurs de champs de membre absentes de la cible ;
  - supprimer la fiche d'origine si elle ne porte plus d'adhésion ni d'inscription.

  Journaliser `membership_attached`.
- [X] T029 [P] [US1] Ajouter la section « Adhésions publiques » dans `web/app/routes/app/admin/settings-general.tsx` : l'interrupteur, l'adresse en lecture seule avec un bouton « Copier » (annonce en `aria-live`), et le rappel « Les tarifs de la saison ouverte deviennent visibles de tous. ».
- [X] T030 [US1] Créer `web/app/routes/public/membership.tsx` (`/adherer`) et `web/app/api/public.ts` (hooks `usePublicMembershipPage`, `useSubmitPublicMembership`). L'écran suit contracts/ui-routes.md › `/adherer` :
  - message de fermeture ; bandeau « Vous êtes connecté » si `get-session` répond `200`, avec le `401` silencieux ;
  - `fieldset` « Vous », « Vous joindre », « Votre adhésion » (`RadioGroup` des tarifs, type, montant en `aria-live`), puis les champs personnalisés via `web/app/customfields`, `ConsentField`, la case « Créer un compte pour suivre mes demandes » et `Honeypot` ;
  - écran « Consultez votre messagerie » avec un lien vers `/suivi/lien`.

  Les erreurs `422` sont appliquées par `applyProblemToForm`. Ajouter la route `adherer` au bloc public de `web/app/routes.ts`.
- [X] T031 [US1] Créer `web/app/routes/public/confirm.tsx` (`/confirmer-demande`). Il utilise `useFragmentToken` et appelle `confirm-public-submission` une seule fois (garde `useRef`), puis affiche les écrans de contracts/ui-routes.md selon `tracking`, `account`, `duplicate`, `closed` ou `410`. Pour `tracking`, il affiche un écran autonome « Demande enregistrée » ou « Inscription confirmée », avec « Un e-mail vous a été envoyé avec votre lien de suivi », sans dépendre de `/suivi` : US1 reste testable seule. Ajouter la route au bloc public de `web/app/routes.ts`.
- [X] T032 [US1] Dans `web/app/routes/app/admin/membership-detail.tsx` et `memberships.tsx` :
  - afficher le badge « Publique » et l'auteur « Demande publique » ;
  - ajouter le filtre « Origine : publique », reflété dans l'adresse de la page ;
  - pour une demande publique en attente, ajouter l'encart « Doublon probable » qui liste `duplicateCandidates`, avec l'action « Rattacher à cette fiche ». Elle passe par un `ConfirmDialog` qui nomme les deux fiches (FR-039) et invalide les requêtes après succès.
- [X] T033 [US1] Créer `web/e2e/public-journeys.spec.ts`, section « Adhésion publique » :
  - l'accueil est fermé, puis l'admin l'active et copie l'adresse ;
  - en contexte anonyme, dépôt puis écran neutre ;
  - l'e-mail est lu dans Mailpit (`mailpit.ts`) et le lien de confirmation suivi ; la demande est enregistrée ;
  - en admin, la demande porte le badge « Publique » et est validée ;
  - axe passe sur `/adherer` et `/confirmer-demande` ;
  - au clavier, le piège n'est jamais focalisé.

**Checkpoint**: US1 livrable seule. On reçoit des demandes d'adhésion publiques, confirmées et validables, sans exposer de données.

---

## Phase 4: User Story 2 - Suivre et payer sa demande par lien personnel (Priority: P1)

**Goal**: un lien personnel ouvre le suivi des fiches sans compte d'une adresse, y compris les fiches importées. Il permet d'annuler une demande, d'annoncer un moyen de paiement et de payer par carte. Les notifications d'adhésion et d'inscription partent vers la fiche sans compte, avec un lien neuf.

**Independent Test**: une demande publique validée et non gratuite se paie par carte depuis le lien de l'e-mail de validation, sans connexion. Une fiche importée reçoit un lien sur demande de l'admin et peut annoncer un chèque (quickstart.md › scénarios 3 et 4).

### Tests for User Story 2

- [X] T034 [P] [US2] Créer `internal/api/tracking_isolation_test.go`. Avec un lien de A, les cas suivants renvoient `404`, de forme identique à une donnée inexistante (FR-014, SC-007) :
  - une adhésion de B ;
  - une fiche de A rattachée entre-temps à un compte (005 FR-012) ;
  - une fiche de A dont l'adresse a été changée par un admin ;
  - une fiche de A supprimée.

  Une fiche de A dont l'adresse diffère seulement par la casse reste visible. Une session de membre ou d'administrateur sans lien reçoit `401`. Une session de compte munie d'un lien ne voit que le périmètre du lien.
- [X] T035 [P] [US2] Créer `internal/api/tracking_test.go` :
  - **`get-tracking`** : les fiches nommées avec leurs adhésions et leurs inscriptions, les `actions`, et `canCreateAccount` faux si un compte existe pour l'adresse ;
  - **`cancel-tracking-membership`** : annulation d'une demande en attente, `409` sur une adhésion validée ;
  - **`announce-tracking-membership-payment`** : chèque, puis virement (changement de moyen), et `409` après constatation ;
  - **`request-tracking-link`** : la même réponse `202` pour une adresse inconnue, une adresse de compte et une adresse de fiche suivie ; un seul e-mail, pour la fiche suivie ; `429` au-delà de la limite par IP ;
  - **`close-tracking`** : `get-tracking` renvoie ensuite `401`, et la session de compte éventuelle reste valide ;
  - **limites par adresse** : 121 appels `link` en une minute avec le même lien, répartis sur deux IP, donnent un `429` (`TrackingUseEmail`) ; 31 ouvertures d'un lien de la même adresse en une heure depuis des IP différentes donnent un `429` (`TrackingOpenEmail`) ;
  - **périmètre** : aucune opération par lien ne modifie l'identité ni les champs (FR-016).
- [X] T036 [P] [US2] Créer `internal/api/tracking_card_payments_test.go` avec `hatest`, sur le modèle de `card_payments_test.go` :
  - **lancement** par lien : le payeur prérempli est l'adresse du lien plus le prénom et le nom de la fiche ; `account_id` vaut `NULL` et `via_link = 1` ; les URL de retour pointent vers `/suivi/adhesions/{id}?tentative=…&carte=` ;
  - **constatation** par retour (`confirm-tracking-membership-card-payment`) et par notification (webhook), une seule fois ;
  - **mode test** : la mention est visible.
- [X] T037 [P] [US2] Créer `internal/api/recipients_test.go` (FR-035, FR-040) :
  - une fiche importée sans compte, avec une adresse, dont l'adhésion est validée, reçoit `membership-validated` à son adresse, avec un lien vers `/suivi/adhesions/{id}#jeton=` et une ligne `tracking_links` créée ;
  - une fiche gérée par un compte actif : adresse du compte, lien vers l'espace membre (inchangé) ;
  - une fiche sans compte et sans adresse ne reçoit aucun e-mail ;
  - une inscription d'une fiche sans compte reçoit `registration-confirmed` avec un lien personnel ;
  - fiche dont le compte est désactivé : **aucun** e-mail d'adhésion (006 FR-023 inchangé), mais l'e-mail d'inscription part à l'adresse de la fiche, sans lien personnel (012 inchangé).

  Les tests existants de 006 et 012 doivent rester verts.

### Implementation for User Story 2

- [X] T038 [P] [US2] Ajouter à `internal/database/queries/memberships.sql` les requêtes `ListMembershipsOfTrackedEmail(email)` et `GetMembershipOfTrackedEmail(id, email)`, qui joignent `members` avec `account_id IS NULL AND email = ? COLLATE NOCASE`. Ajouter à `internal/database/queries/event_registrations.sql` `ListRegistrationsOfTrackedEmail` et `GetRegistrationOfTrackedEmail`, et à `internal/database/queries/online_payments.sql` `GetOnlinePaymentOfTrackedEmail(id, membership_id|registration_id, email)`. Ajouter `via_link` à `CreateOnlinePayment`. Puis lancer `sqlc generate`.
- [X] T039 [US2] Créer `internal/api/recipients.go` avec `recipientOf(ctx, q, member, disabledFallback bool) (to string, tracked bool)` (research.md › R8). Il renvoie :
  - l'adresse du compte, s'il est actif ;
  - sinon, l'adresse de la fiche si elle est sans compte, avec `tracked = true` ;
  - sinon, si le compte est désactivé et que `disabledFallback` est vrai, l'adresse de la fiche avec `tracked = false` ;
  - sinon `""`.

  Les adhésions appellent `recipientOf(…, false)` (006 FR-023 : rien pour un compte désactivé), les inscriptions `recipientOf(…, true)` (comportement actuel de `contactEmail`). Aucun comportement existant ne change en dehors des fiches sans compte (FR-035).

  Ajouter `noticePath(ctx, q, to, tracked, memberPath, trackingPath)`. Si `tracked`, il émet un lien avec `issueTrackingLink` et renvoie `trackingPath + "#jeton=" + raw`, sinon `memberPath`. Remplacer `noticeRecipient` dans `notifyMembership` (`internal/api/memberships.go`), et `contactEmail` dans `internal/api/event_notices.go` et `registrations_*.go`. Les chemins de suivi sont `/suivi/adhesions/{id}` et `/suivi/inscriptions/{id}`, ou `/suivi` sans lien de paiement. Pour les e-mails groupés par adresse (012 FR-031), émettre un seul lien par adresse.
- [X] T040 [US2] Créer `internal/api/tracking.go`, enregistré dans `registerTracking`, avec les types `Tracking`, `TrackedItem` et `TrackedMembership` de contracts/api.md. `TrackedMembership` réutilise la construction de `MyMembership` sans les champs de compte. Opérations `auth.Link` :
  - `get-tracking` (`GET /tracking`) : `ListTrackedMembers` et les listes `…OfTrackedEmail`. `canCreateAccount` est faux s'il existe un compte pour l'adresse ou si l'adresse suit plus de 20 fiches ;
  - `get-tracking-membership` (`GET /tracking/memberships/{id}`) ;
  - `cancel-tracking-membership` (`POST …/cancel`) : mêmes règles que l'annulation par le compte (006 FR-009), avec `cancelled_by` à `NULL` et le journal `space=tracking` ;
  - `announce-tracking-membership-payment` (`PUT …/payment`, `{ method }`) : il réutilise la fonction interne de l'annonce de l'espace membre.

  Un objet hors périmètre renvoie `404` (`MsgMembershipGone`). Ajouter `request-tracking-link` (`auth.Public`, `POST /public/tracking/link-request`, `{ email }`) :
  - limite `TrackingRequestIP`, avec `429` ;
  - `422` si l'adresse est mal formée ;
  - si `CountTrackedMembers > 0` et que `PublicMailEmail` le permet : `issueTrackingLink`, puis le modèle `tracking-link` vers `/suivi#jeton=…` ;
  - toujours `202` avec le corps neutre.

  L'ajouter à `allowedPublic`.
- [X] T041 [US2] Dans `internal/api/card_payments.go`, faire évoluer `openCheckout(ctx, q, opID, base, req)` en `openCheckout(ctx, q, opID, base, req, payer helloasso.Payer)`. Les appelants de l'espace membre construisent le payeur comme avant (adresse du compte, fiche « Vous »), dans un helper `accountPayer(ctx, q)`. Aucune autre modification du circuit : `verifyOnlinePayment` reste inchangé.
- [X] T042 [US2] Créer `internal/api/tracking_payments.go` avec deux opérations :
  - `start-tracking-membership-card-payment` (`auth.Link`, `POST /tracking/memberships/{id}/card-payment`) : `GetMembershipOfTrackedEmail`, les mêmes contrôles que `startMyCardPayment` (`CanPay`, `CanPayByCard`, reprise sur `resumeWindow`), `CreateOnlinePayment` avec `AccountID` nul et `via_link=1`, un payeur formé de l'adresse du lien et du prénom et du nom de la fiche, et la base de retour `BaseURL + "/suivi/adhesions/{id}?tentative={op}&carte="` ;
  - `confirm-tracking-membership-card-payment` (`POST …/confirm`) : `GetOnlinePaymentOfTrackedEmail`, puis `verifyOnlinePayment(…, sourceReturn, hint)`, et la réponse `TrackedMembership`.

  Journaliser avec `online_payment_id` et `membership_id` seulement.
- [X] T043 [US2] Dans `internal/api/members_admin.go` :
  - ajouter à `AdminMember` les champs `origin`, `emailConfirmedAt`, `consentedAt` et `trackedByLink` (sans compte et avec une adresse) ;
  - dans `update-member`, appeler `ResetMemberEmailConfirmation` quand l'adresse change ;
  - ajouter `send-tracking-link` (`auth.Admin`, `POST /admin/members/{id}/tracking-link`), qui renvoie `409` (« Cette fiche a un compte ou n'a pas d'adresse. ») si la fiche a un compte ou n'a pas d'adresse, et sinon consomme `PublicMailEmail` sur l'adresse de la fiche (au-delà : `429`, « Trop d'envois vers cette adresse, réessayez plus tard. »), émet le lien, envoie le modèle `tracking-link` et renvoie `204`. Tester le plafond : après 5 envois en une heure (dépôts compris), le 6ᵉ renvoie `429`.

  Ajouter les tests dans `internal/api/members_admin_test.go`.
- [X] T044 [P] [US2] Créer `web/app/api/tracking.ts` avec les hooks `useTracking`, `useTrackedMembership`, `useCancelTrackedMembership`, `useAnnounceTrackedMembership`, `useStartTrackedMembershipCard`, `useConfirmTrackedMembershipCard`, `useOpenTrackingLink`, `useRequestTrackingLink` et `useCloseTracking`. Un `401` est traduit en état « lien expiré ».
- [X] T045 [US2] Créer `web/app/routes/public/tracking.tsx` (`/suivi`). Au montage, `useFragmentToken` puis `open-tracking-link` si un jeton est présent, et ensuite `get-tracking`. L'écran affiche les cartes par fiche (`h2` au nom de la fiche), les badges d'état réutilisés de `memberships/`, les actions « Payer » ou « Voir », et le bouton « Fermer l'accès sur cet appareil ». Si l'accès est expiré, il affiche « Ce lien n'est plus valable » avec un lien vers `/suivi/lien`. Créer aussi `web/app/routes/public/tracking-request.tsx` (`/suivi/lien`), avec un formulaire d'adresse et une réponse neutre. Ajouter les deux routes au bloc public de `web/app/routes.ts`. Dans `web/app/routes/public/confirm.tsx`, ajouter à l'écran `tracking` le bouton « Voir mon suivi » vers `/suivi` (l'accès est déjà ouvert en session par la confirmation).
- [X] T046 [US2] Créer `web/app/routes/public/tracking-membership.tsx` (`/suivi/adhesions/:id`). Il réutilise `PaymentSection`, en branchant `announce`, `startCard` et `refresh` sur les hooks de `tracking.ts`, et `CardReturn`, en branchant `confirm` sur `useConfirmTrackedMembershipCard`. Il affiche l'annulation par `ConfirmDialog` et la note « Pour corriger vos informations, contactez l'association. ». Ajouter la route au bloc public.
- [X] T047 [US2] Dans `web/app/routes/app/admin/member-detail.tsx`, afficher l'origine, « Adresse confirmée le … » et « Consentement le … ». Pour une fiche sans compte avec adresse, ajouter le bouton « Envoyer un lien de suivi ». Sous le champ d'adresse d'une fiche sans compte, afficher l'aide de FR-040 : « Cette adresse recevra les notifications et un accès au suivi et au paiement. ». Afficher la même aide dans `web/app/routes/app/admin/member-new.tsx`, tant qu'aucun compte de rattachement n'est choisi.
- [X] T048 [US2] Étendre `web/e2e/public-journeys.spec.ts`, section « Suivi et paiement » :
  - l'admin valide la demande publique ;
  - dans Mailpit, on suit le lien de validation ; l'URL ne contient plus `jeton` ;
  - paiement par carte avec `fake-helloasso.ts`, puis statut « Payée » ;
  - `/suivi/lien` donne le même message pour deux adresses ;
  - fiche importée : « Envoyer un lien de suivi », puis annonce d'un chèque ;
  - axe sur `/suivi`, `/suivi/lien` et `/suivi/adhesions/:id`.

**Checkpoint**: US1 et US2 forment un parcours d'adhésion complet sans compte. Les fiches importées paient en ligne.

---

## Phase 5: User Story 3 - S'inscrire à un événement public par son lien (Priority: P2)

**Goal**: un événement « Ouvert au public » a un lien public non devinable. Une personne extérieure s'y inscrit et confirme son adresse : l'inscription est alors confirmée et la place réservée. Elle paie ou annule par son lien personnel.

**Independent Test**: un événement payant public de 2 places. On s'y inscrit, on confirme, la place est décomptée, on paie par lien. La dernière place est disputée, et l'ancien lien ne sert plus après régénération (quickstart.md › scénario 6).

### Tests for User Story 3

- [X] T049 [P] [US3] Dans `internal/event/event_test.go`, ajouter des tests en table pour `OpennessOf("all_members", slug)` (qui donne `Public`), `StorageOf(Public)`, `Eligibility(Public, …)` (identique à `AllMembers`) et `PublicVisible` (publié, annulé ou terminé et public : vrai ; brouillon ou non public : faux).
- [X] T050 [P] [US3] Créer `internal/api/public_event_test.go` :
  - **`get-public-event`** : la liste fermée des champs de `PublicEvent`, comparée sur le JSON brut : exactement les clés de contracts/api.md, sans inscrit, identifiant interne, message d'annulation ni champ personnalisé (principe II). `closedReason` vaut `full`, `cancelled`, `finished`, `not_yet_open` ou `closed`. La réponse `404` est identique pour un brouillon, un événement non public, un slug inconnu et un slug régénéré.
  - **`submit-public-registration`** : `202` neutre, `409` avec la raison si l'événement est complet ou fermé, piège et limites comme T019.
  - **Confirmation** : inscription `confirmed` avec `via_public=1` et `registered_by` à `NULL`, prix figé, référence `EVT-`, `payment_status='due'` si l'événement est payant, et e-mail avec un lien personnel.
  - **50 confirmations concurrentes sur 3 places** : 3 inscriptions et 47 `closed` (SC-006).
  - **Même identité déjà inscrite** : `duplicate`.
  - **Compte connecté** : `openness:"public"` permet d'inscrire toute fiche gérée (FR-023).
- [X] T051 [P] [US3] Ajouter à `internal/api/events_admin_test.go` :
  - créer un événement `openness:"public"` génère `publicUrl`, qui se termine par `/e/<22 caractères>` ;
  - passer à `all_members` met `publicUrl` à `null`, et les inscriptions existantes restent ;
  - `regenerate-event-public-link` change le slug, et renvoie `409` si l'événement n'est pas public.
- [X] T052 [P] [US3] Ajouter à `internal/api/tracking_test.go` et `tracking_card_payments_test.go` les opérations d'inscription par lien : annulation selon 012 FR-013 (gratuite ; payante non payée ; refusée si le paiement par carte est en cours), annonce d'un moyen, paiement par carte avec retour sur `/suivi/inscriptions/{id}`. Vérifier aussi que le retrait par un admin envoie `registration-removed` à l'adresse confirmée, avec un lien personnel.

### Implementation for User Story 3

- [X] T053 [P] [US3] Dans `internal/event/event.go`, ajouter `Public` à `Openness`, ainsi que `OpennessOf(openness string, slug sql.NullString) Openness`, `StorageOf(Openness) (openness string, public bool)` et `PublicVisible(status, phase, openness) bool`. `Eligibility` traite `Public` comme `AllMembers`. Ajouter `NewPublicSlug() string` : 16 octets de `crypto/rand` en base64url sans remplissage, soit 22 caractères.
- [X] T054 [US3] Dans `internal/database/queries/events.sql`, ajouter `public_slug` aux lectures et écritures, ainsi que les requêtes `SetEventPublicSlug(id, slug)` et `GetEventByPublicSlug(slug)`. Dans `internal/api/events_admin.go` et `events.go` :
  - mapper l'enum API `season_members | all_members | public` (huma `enum`) avec `StorageOf` et `OpennessOf` ;
  - à la création ou à la modification vers `public`, générer un slug s'il est absent ; vers une autre ouverture, le mettre à `NULL` ;
  - ajouter `publicUrl = BaseURL + "/e/" + slug` à `AdminEvent` ;
  - ajouter `regenerate-event-public-link` (`auth.Admin`, `POST /admin/events/{id}/public-link`), avec `409` si l'événement n'est pas public et le journal `event_public_link_regenerated` sans le slug.

  Lancer `sqlc generate`.
- [X] T055 [US3] Dans `internal/api/events_me.go`, traiter `public` comme `all_members` pour la liste, le détail et `register-my-members`, en passant par `event.Eligibility`.
- [X] T056 [US3] Dans `internal/api/public.go`, ajouter deux opérations publiques :
  - `get-public-event` (`auth.Public`, `GET /public/events/{slug}`) : `GetEventByPublicSlug`, `PublicVisible`, sinon `404` (« Événement introuvable. ») identique dans tous les cas. Elle renvoie `PublicEvent`, limité aux clés de contracts/api.md : ni `cancelMessage` ni champ personnalisé (principe II) ;
  - `submit-public-registration` (`auth.Public`, `POST /public/events/{slug}/registrations`) : le même ordre que T025, avec un `404` uniforme et un `409` si l'inscription est impossible au moment du dépôt (raison de `closedReason`). Le corps n'accepte aucun champ personnalisé (FR-026), et `custom_values` vaut `'[]'`. Le dépôt est de type `kind='registration'`.

  Ajouter les deux opérations à `allowedPublic`.
- [X] T057 [US3] Dans `internal/api/public_confirm.go`, ajouter la branche `registration`, dans la même transaction. Les conditions sont un événement publié, public, dans sa période et non complet, avec la capacité comptée comme 012 FR-015. La branche :
  - déduplique sur une inscription confirmée existante ;
  - crée la fiche, ou rattache selon FR-033 ;
  - crée l'inscription `confirmed` avec `via_public=1`, le prix figé, la référence (`event.ReferencePrefix` + `NextReference`) et `payment_status='due'` si `amount > 0` ;
  - envoie l'e-mail `public-registration-confirmed`.

  Créer ce modèle dans `internal/mail/templates/public-registration-confirmed.{txt,html}.tmpl` : titre, date et lieu de l'événement, montant et référence si l'événement est payant, et lien personnel. Ajouter le cas dans `render_test.go`.
- [X] T058 [US3] Dans `internal/api/tracking.go` et `tracking_payments.go`, ajouter :
  - `get-tracking-registration` ;
  - `cancel-tracking-registration`, selon 012 FR-013, via la fonction interne de `registrations_me.go` ;
  - `announce-tracking-registration-payment` ;
  - `start-tracking-registration-card-payment`, avec l'intitulé de 012 FR-024 et le retour `/suivi/inscriptions/{id}?tentative=…&carte=` ;
  - `confirm-tracking-registration-card-payment`.

  Toutes sont `auth.Link` et utilisent `GetRegistrationOfTrackedEmail`.
- [X] T059 [P] [US3] Dans `internal/api/registrations_admin.go` et `internal/database/queries/event_registrations.sql`, ajouter `viaPublic` à `AdminRegistration` et le filtre `origin=public` à `list-registrations`.
- [X] T060 [P] [US3] Dans `web/app/events/EventForm.tsx`, ajouter l'option d'ouverture « Ouvert au public », avec l'aide « Toute personne qui reçoit le lien peut s'inscrire ». Dans `web/app/routes/app/admin/event-detail.tsx`, ajouter l'encart « Lien public » avec l'adresse, « Copier » et « Régénérer » (`ConfirmDialog`). Dans `web/app/routes/app/admin/registrations.tsx` et `registration-detail.tsx`, ajouter le badge « Publique » et le filtre d'origine.
- [X] T061 [US3] Créer `web/app/routes/public/event.tsx` (`/e/:slug`), qui suit contracts/ui-routes.md : en-tête et détails avec les formats de `web/app/events/EventDate.tsx`, encart de fermeture selon `closedReason`, et formulaire avec `PublicIdentityFields`, `ConsentField`, « Créer un compte » et `Honeypot`, sans champ personnalisé (FR-026). Le bouton est « M'inscrire » ou « M'inscrire — 35 € ». Le `404` donne « Événement introuvable ». Créer aussi `web/app/routes/public/tracking-registration.tsx` (`/suivi/inscriptions/:id`) sur le modèle de T046. Ajouter les deux routes au bloc public de `web/app/routes.ts`, et les hooks à `web/app/api/public.ts` et `tracking.ts`.
- [X] T062 [US3] Étendre `web/e2e/public-journeys.spec.ts`, section « Événement public » :
  - l'admin crée un événement public de 2 places à 5 € et copie le lien ;
  - en anonyme : inscription, confirmation par Mailpit, « 1 place restante » ;
  - la dernière place est prise par l'admin, puis une seconde confirmation affiche « complet » ;
  - après régénération, l'ancien lien affiche « Événement introuvable » ;
  - axe sur `/e/:slug` et `/suivi/inscriptions/:id`.

**Checkpoint**: US3 est livrable après US1 et US2, dont elle réutilise la confirmation et le suivi.

---

## Phase 6: User Story 4 - Créer un compte depuis un parcours public (Priority: P3)

**Goal**: depuis le dépôt (case cochée) ou depuis l'espace de suivi, la personne choisit un mot de passe. Un compte membre est créé, avec ses fiches suivies rattachées et sa fiche « Vous » désignée. Ses liens sont révoqués et elle est connectée.

**Independent Test**: un dépôt avec « Créer un compte », la confirmation, le mot de passe. La personne est connectée, sa fiche porte « Vous » et sa demande est listée dans son espace. Un ancien lien est ensuite refusé (quickstart.md › scénario 7).

### Tests for User Story 4

- [X] T063 [P] [US4] Créer `internal/api/tracking_account_test.go` pour `create-tracking-account` :
  - **création réussie** : un compte `member` avec `email_verified_at` posé, les fiches suivies rattachées, la fiche « Vous » d'office s'il n'y en a qu'une, la fiche choisie ou aucune s'il y en a plusieurs ; la session est connectée (`get-session` renvoie `200`) et `tracking_link_id` est retiré ; tous les liens de l'adresse sont révoqués (un ancien lien donne `410` à l'ouverture) ;
  - **`422`** si le mot de passe n'est pas conforme ;
  - **`409`** si un compte existe déjà pour l'adresse, ou si l'adresse suit 21 fiches ;
  - **`selfMemberId` hors périmètre** : `422`.

### Implementation for User Story 4

- [X] T064 [US4] Dans `internal/api/tracking.go`, ajouter `create-tracking-account` (`auth.Link`, `POST /tracking/account`, `{ password, selfMemberId }`). Elle s'appuie sur la création de compte d'`acceptInvitation` (`internal/api/auth.go`) : `ValidatePassword`, `HashPassword`, `CreateAccount` avec le rôle `member` et `EmailVerifiedAt = now`. En `WriteTx`, elle :
  - vérifie qu'aucun compte n'existe pour l'adresse et que `CountTrackedMembers ≤ 20` ;
  - rattache les fiches (`UPDATE members SET account_id = ? WHERE account_id IS NULL AND email = ? COLLATE NOCASE`) ;
  - pose `is_self` ;
  - appelle `RevokeTrackingLinksOfEmail`.

  Après la transaction : `s.Sessions.Remove("tracking_link_id")`, puis `auth.Login`. Elle renvoie `201` et journalise `tracking_account_created` avec `account_id` seulement.
- [X] T065 [US4] Créer `web/app/routes/public/tracking-account.tsx` (`/suivi/compte`) : le mot de passe avec la règle affichée, en reprenant le composant de `routes/auth/accept-invitation.tsx`. Il pose la question de la fiche « Vous » s'il y a plusieurs fiches, avec un `RadioGroup` et l'option « Aucune ». Après succès, il invalide la session et redirige vers `/`. Ajouter l'encart « Créer un compte » dans `tracking.tsx` quand `canCreateAccount`. Dans `web/app/routes/public/confirm.tsx`, rediriger vers `/suivi/compte` quand l'issue est `tracking` et que `createAccount` est vrai. Ajouter la route au bloc public de `web/app/routes.ts`.
- [X] T066 [US4] Étendre `web/e2e/public-journeys.spec.ts`, section « Création de compte » : dépôt avec la case cochée, confirmation, redirection vers `/suivi/compte`, mot de passe, puis connexion. La demande apparaît dans « Adhésions » avec « Vous ». Un ancien lien affiche « Ce lien n'est plus valable ». Vérifier axe sur `/suivi/compte`.

**Checkpoint**: US4 livrable après US2.

---

## Phase 7: User Story 5 - Ouvrir et superviser les parcours publics (Priority: P3)

**Goal**: l'administration distingue l'origine « Publique » dans les membres, les adhésions et les inscriptions. Les fiches publiques inactives sont signalées et supprimées sur décision humaine. Fermer l'accueil n'empêche pas le suivi des demandes en cours.

**Independent Test**: filtrer les membres sur l'origine publique, constater l'encart des fiches inactives dans le tableau de bord, les supprimer en lot. Désactiver l'accueil et vérifier qu'un lien personnel permet encore de payer (quickstart.md › scénarios 1 et 10).

### Tests for User Story 5

- [X] T067 [P] [US5] Créer `internal/public/inactive_test.go`, des tests en table de `Inactive` sur les bornes de 12 mois : créée il y a 11 ou 13 mois, demande en attente, adhésion validée d'une saison terminée il y a 6 mois, inscription à un événement terminé il y a 13 mois. Créer aussi `internal/api/members_inactive_test.go`, avec `SetToday` :
  - `list-members?inactive=true` liste exactement les fiches inactives ;
  - `get-membership-dashboard.inactivePublicCount` en donne le nombre ;
  - `delete-inactive-members` renvoie `{deleted, skipped}`, où une fiche redevenue active entre-temps est ignorée ;
  - les fiches `internal` ne sont jamais concernées ;
  - membre : `403`.
- [X] T068 [P] [US5] Ajouter à `internal/api/members_admin_test.go` les cas suivants :
  - le filtre `origin=public` se combine avec la recherche et `noAccount` ;
  - avec l'accueil désactivé, un lien personnel existant permet encore `get-tracking` et `announce-tracking-membership-payment`, tandis qu'un nouveau dépôt renvoie `409` (US5 › scénario 5).

### Implementation for User Story 5

- [X] T069 [P] [US5] Créer `internal/public/inactive.go` : `Inactive(createdAt, pending, lastValidatedSeasonEnd, lastEventEnd, cutoff) bool`, le miroir testable de la requête de R14.
- [X] T070 [US5] Dans `internal/database/queries/members.sql`, ajouter deux requêtes :
  - `ListInactivePublicMembers(cutoff)` : `origin='public' AND account_id IS NULL AND created_at < cutoff`, sans adhésion `pending`, sans adhésion `validated` dont la saison se termine après la coupure, et sans inscription `confirmed` à un événement dont la fin est postérieure à la coupure ;
  - `CountInactivePublicMembers(cutoff)`.

  Ajouter aussi les paramètres `origin` et `inactive_cutoff`, tous deux nullables, à la requête de `list-members`. Lancer `sqlc generate`.
- [X] T071 [US5] Dans `internal/api/members_admin.go` :
  - ajouter les filtres `origin` et `inactive` à `list-members` ;
  - ajouter `origin` et `inactive` à `AdminMemberSummary` ;
  - ajouter `delete-inactive-members` (`auth.Admin`, `POST /admin/members/inactive/delete`, `{ ids: int[] }` avec `minItems:1` et `maxItems:500`). En `WriteTx`, elle revérifie chaque fiche avec `ListInactivePublicMembers` et la supprime selon les règles existantes (cascade de 006 FR-021 et 012), puis renvoie `{deleted, skipped}` et journalise `inactive_members_deleted` avec les nombres.

  Dans `internal/api/memberships_admin.go`, ajouter `inactivePublicCount` à `MembershipDashboard`.
- [X] T072 [US5] Dans `web/app/routes/app/admin/members.tsx`, ajouter :
  - les filtres « Origine : publique » et « Inactives à supprimer », reflétés dans l'adresse de la page ;
  - le badge « Publique » ;
  - avec le filtre « Inactives », l'action « Supprimer les N fiches », qui passe par un `ConfirmDialog` indiquant le nombre et annonce le résultat (`deleted`, `skipped`).

  Dans `web/app/routes/app/admin/home.tsx`, ajouter l'encart « N fiches publiques inactives », qui mène au filtre.
- [X] T073 [US5] Étendre `web/e2e/public-journeys.spec.ts`, section « Supervision » : filtres d'origine sur les membres, les adhésions et les inscriptions, encart du tableau de bord avec une fiche inactive seedée, suppression en lot, et axe sur `/admin/membres` avec les filtres.

**Checkpoint**: toutes les stories sont livrées.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T074 [P] Dans `internal/tools/e2eseed`, ajouter une fiche importée sans compte (`paul@example.org`) avec une adhésion validée « À payer », un événement public publié, et une fiche publique inactive créée il y a 13 mois. Les tests Playwright s'en servent.
- [X] T075 [P] Créer `internal/api/public_logs_test.go` : rejouer les parcours de US1 à US4 avec `env.Logs()` et vérifier qu'aucune ligne de journal ne contient d'adresse électronique, de nom, de jeton (brut ou base64url de 43 caractères) ni de slug (FR-042). Vérifier aussi qu'aucune requête HTTP du journal d'accès ne porte `jeton` (point de vigilance 1 du plan).
- [X] T076 Régénérer les artefacts avec `go tool task gen` (`go generate ./...`, `go run ./cmd/frigg openapi > web/app/api/openapi.json`, `bun run gen:api`). Vérifier qu'il n'y a aucun diff après la régénération. Vérifier que `TestAccessMatrix` journalise 24 opérations de plus, dont 7 publiques et 13 `link`.
- [X] T077 [P] Créer `docs/usage/public-journeys.md` et l'ajouter à la navigation VitePress de `docs/`. La page explique :
  - comment activer les demandes publiques et diffuser l'adresse `/adherer` ;
  - comment ouvrir un événement au public et régénérer son lien ;
  - le lien de suivi des personnes sans compte, y compris les fiches importées ;
  - les fiches inactives et leur suppression.

  Mettre à jour `docs/usage/memberships.md`, `events.md` et `members.md` avec l'origine « Publique » et les e-mails envoyés aux fiches sans compte. Dans `docs/installation/upgrade.md`, signaler le changement de comportement : les fiches sans compte avec adresse reçoivent désormais les notifications.
- [X] T078 Exécuter `go tool task check` : build `CGO_ENABLED=0`, `go vet`, `go test ./...`, générations sans diff, migrations sur base vierge, `bun run typecheck`, Biome, Vitest, `bun run build`, puis Playwright et axe. Dérouler ensuite les scénarios manuels de `specs/016-public-journeys/quickstart.md` et cocher les points de vigilance 1 à 7 de plan.md.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : T001 → T002. T003, T004 et T005 sont parallèles après T001. T006 vient après T003 à T005.
- **Foundational (Phase 2)** : dépend de la Phase 1. T007 à T011, T016 et T017 sont parallèles. T012 dépend de T007 ; T013 de T012 ; T014 de T009. T015 dépend de T013. T018 dépend de T017.
- **US1 (Phase 3)** : dépend de la Phase 2.
- **US2 (Phase 4)** : dépend de la Phase 2. Son test de bout en bout (T048) suppose une demande publique validée de US1, mais les tests Go de US2 utilisent des fiches seedées et sont indépendants.
- **US3 (Phase 5)** : dépend de US1 (T025, T026 : chemin de dépôt et de confirmation) et de US2 (T039, T040, T042 : suivi et carte par lien).
- **US4 (Phase 6)** : dépend de US2 (accès par lien, `tracking.tsx`).
- **US5 (Phase 7)** : dépend de la Phase 2. T068 dépend de US2.
- **Polish (Phase 8)** : après les stories voulues.

### Within Each User Story

- Les tests d'abord : ils doivent échouer, puis on passe aux requêtes, au domaine, à l'API et à l'interface, et enfin aux tests de bout en bout.
- Lancer `sqlc generate` après toute modification de `queries/*.sql`.
- Ajouter chaque opération publique à `allowedPublic` dans la même tâche que son enregistrement.

### Parallel Opportunities

- **Phase 1** : T003, T004 et T005.
- **Phase 2** : T007, T008, T009, T010, T011, T016 et T017.
- **US1** : les tests T019 à T022 ensemble ; T023 et T029 en parallèle du reste.
- **US2** : les tests T034 à T037 ensemble ; T038 et T044 en parallèle.
- **US3** : les tests T049 à T052 ensemble ; T053, T059 et T060 en parallèle.
- **US5** : T067, T068 et T069 en parallèle. US5 (hors T068) peut avancer en parallèle de US3 et US4.
- **Polish** : T074, T075 et T077.

---

## Parallel Example: User Story 1

```text
# Tests de US1 ensemble :
Task: "T019 public_membership_test.go (page, dépôt, piège, limites)"
Task: "T020 public_nonrevelation_test.go (réponses identiques octet pour octet)"
Task: "T021 public_confirm_test.go (confirmation, FR-033, doublon, conditions)"
Task: "T022 memberships_attach_test.go (rattachement d'un doublon)"

# Puis en parallèle :
Task: "T023 general_settings.go : publicMembershipEnabled"
Task: "T029 settings-general.tsx : section Adhésions publiques"
```

## Parallel Example: User Story 2

```text
Task: "T034 tracking_isolation_test.go"
Task: "T035 tracking_test.go"
Task: "T036 tracking_card_payments_test.go"
Task: "T037 recipients_test.go"
Task: "T038 requêtes …OfTrackedEmail"
Task: "T044 web/app/api/tracking.ts"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phases 1 et 2, en commençant par la sonde de migration (T002) : elle peut modifier T001.
2. Phase 3 (US1).
3. **Stop et validation** : quickstart.md › scénarios 1, 2, 5, 8 et 9. On peut livrer : les demandes publiques arrivent à l'administration, qui les traite et encaisse hors ligne comme aujourd'hui.

### Incremental Delivery

1. US1 : on reçoit des demandes d'adhésion sans compte.
2. US2 : la personne suit et paie en ligne, et les fiches importées aussi. C'est le cœur de la valeur.
3. US3 : les événements publics.
4. US4 : la création de compte.
5. US5 : la supervision et le RGPD (fiches inactives).

Chaque incrément passe `go tool task check` avant le suivant.

---

## Notes

- **[P]** : fichiers différents, sans dépendance sur une tâche inachevée.
- **Non-révélation** : toute modification d'un parcours public impose de relancer T020 et l'équivalent pour la demande de lien (T035).
- **Jeton** : il ne doit jamais apparaître dans une URL envoyée au serveur. On le passe dans un corps JSON, jamais en paramètre de chemin ni de requête.
- **Commits** : un commit par tâche ou par groupe logique, au format gitmoji et titre court du projet.
