---

description: "Liste des tâches : envoi d'e-mails groupés aux adhérents"
---

# Tasks: Envoi d'e-mails groupés aux adhérents

**Input**: Design documents from `specs/009-bulk-email/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : matrice d'autorisation (principes II et III), tests
d'API sur base isolée, Playwright + axe sur chaque parcours touché. La sécurité
des fichiers (type vérifié sur le contenu, chemins, échappement) a ses propres
tests.

**Organization**: une phase par user story, dans l'ordre de priorité de la spec
(US1 en P1 ; US2 et US3 en P2 ; US4 en P3). US1 livre la relance des impayés
avec les trois cibles prédéfinies et un détail minimal de la communication
envoyée ; US2 ajoute les pièces jointes ; US3 la cible personnalisée (situations
et tarifs) ; US4 la liste et le suivi d'envoi par destinataire.

**Conventions** (constitution 1.7.0) : identifiants, tables, champs JSON et
valeurs en anglais ; commentaires, messages, textes d'interface, journaux et
e-mails en français. Commentaires des requêtes sqlc en ASCII uniquement. Jamais
d'adresse, d'objet, de texte de message ni de nom de fichier dans les journaux.
Aucun e-mail envoyé dans une requête HTTP : tout passe par `outbound_emails`
dans la transaction de l'action.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US4)

## Path Conventions

Module Go à la racine (`internal/`), SPA dans `web/` (plan.md › Project
Structure). Code existant à réutiliser :

- API : `internal/api/api.go` (`register`, `Deps`, `server`, `s.now()`,
  `s.today()`, `CurrentAccount`, `s.internalError`), `internal/api/errors.go`
  (`FieldErrors`), `internal/api/mail.go` (`enqueueNotice`,
  `defaultAssociationName`), `internal/api/memberships_admin.go` (modèle
  d'opérations `admin`, helper `op(...)`), `internal/api/online_sweeper.go`
  (modèle de balayage périodique à horloge injectable),
  `internal/api/accounts.go` (`delete-account`, appel à
  `DeleteEmailsByRecipient`), `internal/api/seasons.go` (état d'une saison,
  libellé).
- Base : `internal/database` (`Store.Read`, `Store.Write`, transaction
  `BEGIN IMMEDIATE` utilisée par les handlers existants),
  `internal/database/queries/emails.sql`, `internal/database/queries/memberships.sql`.
- Mail : `internal/mail/render.go` (`Render`, `Data`), `internal/mail/outbox.go`
  (`Enqueue`, `TimeFormat`), `internal/mail/worker.go`, `internal/mail/smtp.go`,
  `internal/mail/mailtest`.
- Domaine : `internal/membership` (`PaymentStatus` : `due`, `announced`, `paid`).
- Tests Go : `internal/api/apitest` (`New`, `Seed`, `LoggedIn`, `SetToday`,
  `Advance`, `FlushMail`, `Mail`, `Logs`, `Store`, `Client().Do`),
  `internal/api/access_test.go` (matrice, `allowedPublic`),
  `internal/api/memberships_helpers_test.go` (création d'adhésions validées).
- Frontend : `web/app/routes.ts`, `web/app/shell/navigation.ts` (entrée
  `communication` déjà présente), `web/app/api/*` (modèle requêtes et
  invalidations), `web/app/api/problem.ts` (erreurs 422 → `setError`),
  `web/app/api/seasons.ts`, `web/app/api/membership-fees.ts`, composants
  `web/app/ui/*` (`Button`, `Field`, `Select`, `RadioGroup`, `Table`,
  `Pagination`, `Badge`, `Alert`, `ConfirmDialog`, `ScrollRegion`, `Icon`),
  `web/app/shell/PageHeader.tsx`, `web/app/lib/format.ts`. E2E :
  `web/e2e/test.ts`, `web/e2e/axe.ts`, `web/e2e/mailpit.ts`,
  `internal/tools/e2eseed`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: schéma, requêtes et squelette du domaine communs à toutes les stories.

- [X] T001 Créer la migration `internal/database/migrations/00009_communications.sql` (STRICT, commentaires ASCII) selon data-model.md : table `communications` (`draft_key TEXT NOT NULL UNIQUE CHECK (length(draft_key) = 36)`, `author_id INTEGER REFERENCES accounts (id) ON DELETE SET NULL`, `season_id INTEGER NOT NULL REFERENCES seasons (id) ON DELETE RESTRICT`, `situations TEXT NOT NULL` limité par CHECK à une liste triée non vide de `up_to_date,announced,due` — valeurs admises : `announced`, `announced,due`, `announced,due,up_to_date`, `announced,up_to_date`, `due`, `due,up_to_date`, `up_to_date` —, `fee_ids TEXT NOT NULL DEFAULT ''`, `target_label TEXT NOT NULL CHECK (length(target_label) BETWEEN 1 AND 300)`, `subject` 1–150, `body` 1–10000, `members_count >= 1`, `unreachable_count >= 0`, `recipients_count >= 1`, `created_at TEXT NOT NULL`, index `communications_created_at (created_at DESC, id DESC)`) ; table `communication_attachments` (`draft_key` longueur 36, `uploaded_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL` (jamais CASCADE : les pièces d'une communication envoyée doivent survivre à la suppression du compte de leur auteur), `communication_id INTEGER REFERENCES communications (id) ON DELETE CASCADE`, `stored_name TEXT NOT NULL UNIQUE` avec CHECK GLOB 32 caractères `[0-9a-f]`, `original_name` 1–150, `content_type` CHECK IN les 7 types de data-model.md, `size_bytes INTEGER NOT NULL CHECK (size_bytes BETWEEN 1 AND 5242880)`, `sha256` 64 hex, `created_at`, index `(draft_key, uploaded_by)`, `(communication_id)`, `communication_attachments_drafts (created_at) WHERE communication_id IS NULL`) ; table `communication_recipients` (`communication_id NOT NULL … ON DELETE CASCADE`, `email TEXT CHECK (email IS NULL OR length(email) BETWEEN 3 AND 254)`, `members_count >= 1`, `outbound_email_id INTEGER UNIQUE REFERENCES outbound_emails (id) ON DELETE SET NULL`, `status TEXT NOT NULL CHECK (status IN ('pending','sent','failed'))`, `sent_at TEXT`, index `(communication_id, status)` et `(email)`) ; `ALTER TABLE outbound_emails ADD COLUMN communication_id INTEGER REFERENCES communications (id) ON DELETE SET NULL` et `ADD COLUMN from_name TEXT CHECK (from_name IS NULL OR length(from_name) <= 120)`, index `outbound_emails_communication_id` ; déclencheurs `outbound_emails_status_to_recipient` (`AFTER UPDATE OF status ON outbound_emails WHEN new.communication_id IS NOT NULL` → `status = new.status, sent_at = new.sent_at` sur la ligne `outbound_email_id = new.id`) et `outbound_emails_pending_deleted` (`BEFORE DELETE ON outbound_emails WHEN old.status = 'pending' AND old.communication_id IS NOT NULL` → `status = 'failed'`), chacun entre `-- +goose StatementBegin/End` ; section `Down` complète : suppression des déclencheurs et des index, puis reconstruction de `outbound_emails` sans `communication_id` ni `from_name` (`CREATE TABLE outbound_emails_old` à l'identique de `00003`, copie des lignes, `DROP TABLE outbound_emails`, renommage, recréation de `outbound_emails_status_next_idx`) — SQLite refuse `DROP COLUMN` sur une colonne porteuse d'une clé étrangère —, puis suppression des trois tables ; le test T002 applique `Down` puis `Up` de nouveau
- [X] T002 [P] Test de migration in `internal/database/db_test.go` (`TestCommunicationsMigration`) : application sur base vierge et sur une base migrée jusqu'à `00008` contenant des `outbound_emails` (conservés, `communication_id` et `from_name` NULL) ; `situations` invalide (`due,announced` non trié, vide, `paid`) refusée ; `stored_name` hors format refusé ; déclencheurs : passage `pending → sent` recopié avec `sent_at`, `pending → failed` recopié, suppression d'un e-mail en attente → destinataire `failed`, suppression d'un e-mail envoyé → destinataire inchangé et `outbound_email_id` NULL ; `Down` de `00009` puis `Up` de nouveau sans erreur, e-mails conservés
- [X] T003 [P] Écrire les requêtes in `internal/database/queries/communications.sql` (commentaires ASCII) : `ListCommunicationAudience` et `ListCommunicationAudienceForFees` (adhésions `status = 'validated'` de `sqlc.arg(season_id)`, jointures `members`, `accounts` (LEFT), `membership_fees` ; colonne calculée `situation` = `CASE WHEN m.amount_cents = 0 OR m.payment_status = 'paid' THEN 'up_to_date' WHEN m.payment_status = 'announced' THEN 'announced' ELSE 'due' END` ; filtre sur `situation IN (sqlc.slice(situations))`, et pour la seconde variante `m.fee_id IN (sqlc.slice(fee_ids))` ; colonnes `member_id`, `first_name`, `last_name`, `fee_label`, `account_email`, `member_email` ; tri `last_name, first_name, member_id`), `CountFeesInSeason` (`id IN (sqlc.slice(fee_ids)) AND season_id = ?`), `GetCommunicationByDraftKey`, `CreateCommunication`, `GetCommunication` (avec saison et auteur), `ListCommunications` (pagination `LIMIT/OFFSET`, compteurs `pending/sent/failed` agrégés depuis `communication_recipients`, nombre de pièces), `CountCommunications`, `CreateCommunicationRecipient`, `ListCommunicationRecipients` (tri : `failed`, `pending`, `sent` puis `email`), `CountCommunicationDelivery`, `EraseRecipientEmail` (`UPDATE … SET email = NULL WHERE lower(email) = lower(?)`) ; pièces : `CreateCommunicationAttachment`, `ListDraftAttachments` (`draft_key`, `uploaded_by`, `communication_id IS NULL`), `GetDraftAttachment` (`id`, `uploaded_by`, `communication_id IS NULL`), `DeleteDraftAttachment`, `AttachDraftToCommunication`, `ListCommunicationAttachments`, `GetCommunicationAttachment` (`id`, `communication_id`), `ListExpiredDraftAttachments` (`communication_id IS NULL AND created_at < ?`, limite 100), `ListAllStoredNames`
- [X] T004 [P] Ajouter `EnqueueCommunicationEmail` in `internal/database/queries/emails.sql` (comme `EnqueueEmail` avec `communication_id` et `from_name`, `RETURNING id`) et inclure `communication_id`, `from_name` dans `ListDueEmails` (déjà `SELECT *`) ; puis lancer `sqlc generate` (code commité in `internal/database/sqlc/`)
- [X] T005 [P] Créer le paquet `internal/communication` avec `doc.go` (commentaire de paquet en français : règles pures des communications, sans base) et `limits.go` : `MaxFiles = 5`, `MaxFileBytes = 5 << 20`, `MaxTotalBytes = 10 << 20`, `SubjectMax = 150`, `BodyMax = 10000`, `TargetLabelMax = 300`, expression `DraftKeyPattern` (`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

**Checkpoint**: schéma et requêtes générés ; `go test ./internal/database/...` passe.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: règles de cible et d'audience, rendu du message et câblage partagés par toutes les stories.

**⚠️ CRITICAL**: aucune story ne commence avant la fin de cette phase.

- [X] T006 [P] Implémenter in `internal/communication/target.go` : type `Situation` (`UpToDate = "up_to_date"`, `Announced = "announced"`, `Due = "due"`), `Situations` (ordre d'affichage), `Preset` (`all`, `up_to_date`, `unpaid`, `custom`), `SituationsOf(Preset) []Situation`, `PresetOf([]Situation) Preset`, `NormalizeSituations([]Situation) ([]Situation, error)` (dédoublonnage, tri alphabétique pour le stockage, erreur si vide ou inconnue), `EncodeSituations`/`DecodeSituations` (liste séparée par des virgules), `EncodeFeeIDs`/`DecodeFeeIDs`, `TargetLabel(seasonLabel string, p Preset, situations []Situation, feeLabels []string) string` (« 2026-2027 · En impayé », « 2026-2027 · Personnalisée : À payer · Tarifs : Jeune, Adulte », tronqué à 300 caractères), libellés français `SituationLabel` (« À jour », « Paiement annoncé », « À payer ») et `PresetLabel` (« Tous les adhérents », « À jour », « En impayé », « Personnalisée »)
- [X] T007 [P] Implémenter in `internal/communication/audience.go` : `Candidate{MemberID, FirstName, LastName, FeeLabel string/int64, Situation, AccountEmail, MemberEmail string}`, `Target{Candidate, ContactEmail string}`, `Recipient{Email string, MemberIDs []int64}`, `Audience{Members, Unreachable []Target, Recipients []Recipient}` avec `MembersCount()` (joignables + non joignables), `BuildAudience([]Candidate) Audience` : adresse du compte si non vide, sinon adresse de la fiche, sinon non joignable ; dédoublonnage sur `strings.ToLower(strings.TrimSpace(email))`, première graphie conservée, ordre d'entrée préservé (research R2)
- [X] T008 [P] Tests en table in `internal/communication/target_test.go` et `internal/communication/audience_test.go` : chaque préréglage ↔ situations, `PresetOf` d'un sous-ensemble quelconque → `custom`, normalisation (doublons, ordre, vide, valeur inconnue), libellés de cible ; audience : compte prioritaire sur fiche, fiche sans compte, aucune adresse, un parent pour deux enfants → un destinataire avec deux `MemberIDs`, même adresse en casse différente sur deux fiches sans compte → un destinataire, compteurs
- [X] T009 Étendre le rendu in `internal/mail/render.go` : champ `Data.Footer` (pied de message facultatif), `Data.Body` (texte saisi) et fonction de gabarit HTML `paragraphs` qui découpe le texte en paragraphes (lignes vides) et lignes, échappe chaque segment avec `template.HTMLEscapeString`, rend cliquables les seules URL `http://` et `https://` (expression bornée, `rel="noopener"`, `href` échappé), et renvoie `template.HTML` (research R9) ; adapter `templates/layout.txt.tmpl` et `templates/layout.html.tmpl` pour afficher `{{.Footer}}` à la place de « Message envoyé automatiquement par Frigg pour … Merci de ne pas y répondre. » lorsque `Footer` est renseigné ; ajouter `Message.FromName` et `Message.Attachments []Attachment{Name, ContentType, StoredName string; Size int64; SHA256 string}` (rempli par le worker, US2)
- [X] T010 Créer les gabarits `internal/mail/templates/communication.txt.tmpl` (objet = `{{.Subject}}` via un nouveau champ `Data.Subject`, corps = `{{.Body}}` brut) et `internal/mail/templates/communication.html.tmpl` (`{{paragraphs .Body}}`) ; pied « Vous recevez ce message en tant qu'adhérent de {association} pour la saison {saison}. » fourni par l'appelant via `Footer`
- [X] T011 [P] Tests de rendu in `internal/mail/render_test.go` : `communication` avec `<script>alert(1)</script>`, `<b>`, `javascript:alert(1)`, `https://exemple.test/a?b=1&c=2`, paragraphes et retours à la ligne ; version texte inchangée ; pied de communication présent et « Merci de ne pas y répondre » absent ; gabarits existants inchangés (pied par défaut)
- [X] T012 Dans `internal/mail/smtp.go`, utiliser `m.FromFormat(msg.FromName, s.cfg.From)` quand `msg.FromName` n'est pas vide (sinon `m.From` inchangé) ; dans `internal/mail/worker.go`, passer `e.FromName` au `Message` ; test in `internal/mail/worker_test.go` (le faux `Sender` reçoit `FromName`)
- [X] T013 Créer `internal/api/communications.go` : constante `tagCommunications = "communications"`, helper `op(...)` sur le modèle de `memberships_admin.go`, `func (s *server) registerCommunications(api huma.API)` (vide pour l'instant), appelé depuis `New` in `internal/api/api.go` ; types huma partagés de contracts/api.md : `Situation` (enum `up_to_date`, `announced`, `due`), `TargetPreset` (enum `all`, `up_to_date`, `unpaid`, `custom`), `Target{SeasonID int64; Situations []Situation minItems 1 maxItems 3 uniqueItems; FeeIDs []int64 maxItems 50}`, `AudienceMember`, `Audience`, `Attachment`, `Recipient`, `CommunicationSummary`, `Communication`, `Preview` (champs camelCase)
- [X] T014 Implémenter in `internal/api/communications.go` le service interne `func (s *server) audienceFor(ctx, q *sqlc.Queries, t Target) (communication.Audience, seasonRow, []feeRow, error)` : saison inexistante ou en brouillon → `FieldErrors` sur `seasonId` (« Choisissez une saison ouverte ou close. ») ; tarifs hors saison (`CountFeesInSeason` ≠ nombre demandé) → `feeIds` (« Un tarif choisi n'appartient pas à cette saison. ») ; appel de `ListCommunicationAudience` ou `ListCommunicationAudienceForFees` puis `communication.BuildAudience` ; fonction `toAudience(...)` vers le type huma (`preset` = `PresetOf`, `targetLabel` = `TargetLabel`)
- [X] T015 [P] Route de la rubrique in `web/app/routes.ts` : retirer `"communication"` de `adminComingSoon` et ajouter `route("communication", "routes/app/admin/communications-layout.tsx", [index("routes/app/admin/communications.tsx"), route("nouvelle", "routes/app/admin/communication-new.tsx"), route(":id", "routes/app/admin/communication-detail.tsx")])` ; créer `web/app/routes/app/admin/communications-layout.tsx` (`Outlet`, fil d'Ariane « Communication ») et un index provisoire `web/app/routes/app/admin/communications.tsx` (en-tête de page « Communication », introduction de contracts/ui-routes.md, action « Nouvelle communication ») — la liste arrive en US4 ; mettre à jour `web/app/shell/navigation.test.ts` si elle vérifie les rubriques « bientôt »
- [X] T016 [P] Ajouter `formatBytes(n: number): string` (« 850 octets », « 12 Ko », « 1,2 Mo », virgule française) et ses tests in `web/app/lib/format.ts` et `web/app/lib/format.test.ts`

**Checkpoint**: domaine et câblage prêts ; `go test ./...` et `bun run typecheck` passent.

---

## Phase 3: User Story 1 - Relancer les adhérents en impayé (Priority: P1) 🎯 MVP

**Goal**: un administrateur choisit une saison et une cible prédéfinie (« En impayé » par défaut), voit le décompte et les non joignables, rédige, prévisualise, confirme et envoie ; chaque adresse reçoit un exemplaire individuel.

**Independent Test**: saison avec adhésions payées, gratuites, annoncées, à payer, en attente : cibler « En impayé », vérifier décompte et listes, envoyer, constater dans Mailpit un e-mail par adresse de contact des seuls adhérents annoncés ou à payer.

### Tests for User Story 1

- [X] T017 [P] [US1] Tests d'API de l'audience in `internal/api/communications_audience_test.go` (aide de jeu de données dans `internal/api/communications_helpers_test.go` réutilisant `memberships_helpers_test.go` : la saison et les membres de quickstart.md › validation manuelle) : `situations=announced&situations=due` → Léo, Inès, Paul joignables, Hugo non joignable, `recipientsCount = 2`, `membersCount = 4` ; adhésion gratuite et payée → `up_to_date` ; tentative carte échouée → `due` ; demandes en attente, refusées, annulées exclues ; compte désactivé : adresse conservée ; saison en brouillon ou inexistante → `422` sur `query.seasonId` ; situation inconnue → `422` ; cible vide → `200` compteurs à 0 ; volume (SC-002) : saison de 1 000 adhésions → réponse en moins de 2 s sur la machine de test
- [X] T018 [P] [US1] Tests d'API de l'envoi in `internal/api/communications_send_test.go` : `201`, communication enregistrée (`targetLabel`, compteurs), un `outbound_emails` par adresse distincte avec `communication_id`, `from_name` = nom de l'association, objet saisi, pied de saison, aucun autre destinataire dans le message (`apitest.Mail`) ; même `draftKey` renvoyé deux fois (y compris en parallèle, deux goroutines) → une seule communication, `200` au second appel, aucun e-mail ajouté ; décompte modifié entre audience et envoi (paiement constaté) → `409` avec `audience` à jour, rien d'enregistré ; cible sans destinataire → `422` sur `body.target` ; objet vide, objet de 151 caractères, texte de 10 001 caractères, `draftKey` mal formée → `422` sur le champ ; `preview-communication` renvoie objet, texte et HTML échappé sans rien écrire ; journal « communication envoyée » sans adresse ni objet (`env.Logs`) ; volume (SC-004) : saison de 500 adhésions à payer avec 500 adresses distinctes → `201` en moins de 2 s sur la machine de test, 500 `outbound_emails` et 500 destinataires créés
- [X] T019 [P] [US1] Vérifier que la matrice `internal/api/access_test.go` couvre les nouvelles opérations sans modification de `allowedPublic` (anonyme → `401`, membre → `403`, administrateur jamais `401/403`) ; ajouter si besoin un cas explicite pour `send-communication` et `get-communication-audience`

### Implementation for User Story 1

- [X] T020 [US1] Opération `get-communication-audience` (`GET /admin/communications/audience`, `auth.Admin`) in `internal/api/communications.go` : paramètres de requête `seasonId` (requis), `situations` (requis, tableau `explode`), `feeIds` (facultatif) ; appel de `audienceFor` sur `Store.Read` ; réponse `Audience` (contracts/api.md)
- [X] T021 [US1] Opération `preview-communication` (`POST /admin/communications/preview`, `auth.Admin`) in `internal/api/communications.go` : corps `{seasonId, subject (1..150, espaces de bord retirés), body (1..10000)}`, rendu `mail.Render("communication", …)` avec nom de l'association et pied de saison, réponse `Preview{subject, text, html}`, aucune écriture
- [X] T022 [US1] Opération `send-communication` (`POST /admin/communications`, `auth.Admin`, `DefaultStatus` 201) in `internal/api/communications.go` selon research R7 : transaction `BEGIN IMMEDIATE` ; `GetCommunicationByDraftKey` → renvoi `200` de l'existante ; `audienceFor` ; `recipientsCount` ≠ `expectedRecipients` → `409` `problem+json` avec extension `audience` (type d'erreur dédié implémentant `huma.StatusError` et sérialisant `audience`) ; aucun destinataire → `422` « Aucun destinataire ne correspond à cette cible. » ; `CreateCommunication` (situations et tarifs encodés, `target_label`, compteurs, `author_id`) ; pour chaque destinataire : rendu unique du message (une seule fois hors boucle), `EnqueueCommunicationEmail` (`communication_id`, `from_name`), `CreateCommunicationRecipient` (`status = 'pending'`, `outbound_email_id`, `members_count`) ; commit ; journal `info` « communication envoyée » (`communication_id`, `author_id`, `recipients`, `attachments`) ; réponse `Communication`
- [X] T023 [US1] Opération `get-communication` (`GET /admin/communications/{id}`, `auth.Admin`) in `internal/api/communications.go` : `404` « Communication introuvable. » ; réponse `Communication` avec cible décodée, saison, auteur (`null` si supprimé), pièces (liste vide en US1), destinataires et compteurs `delivery`
- [X] T024 [US1] Régénérer le contrat avec `go tool task gen` (`go generate`, `go run ./cmd/frigg openapi > web/app/api/openapi.json`, `bun run gen:api`) ; `web/app/api/openapi.json` et `web/app/api/schema.d.ts` commités
- [X] T025 [P] [US1] Créer `web/app/api/communications.ts` : `audienceQuery(target)` (clé dépendant de `seasonId`, `situations` triées, `feeIds` triés ; `placeholderData: keepPreviousData`), `communicationQuery(id)`, `usePreviewCommunication()`, `useSendCommunication()` (invalide `list-communications` et place la réponse dans le cache de `get-communication`) ; lecture de l'extension `audience` d'un `409` via `web/app/api/problem.ts`
- [X] T026 [P] [US1] Composant `web/app/communications/AudienceSummary.tsx` : région `role="status"` `aria-live="polite"` « {n} adhérents ciblés · {m} destinataires », alerte `warn` « {k} adhérents n'ont aucune adresse et ne recevront pas ce message. » si `unreachableCount > 0`, indicateur de recalcul sans effacer l'ancien décompte, boutons « Voir les adhérents ciblés » / « Voir les non joignables » ouvrant un `Dialog` base-ui avec `ui/Table` dans `ScrollRegion` (Nom, Prénom, Situation, Tarif, Adresse de contact)
- [X] T027 [P] [US1] Composant `web/app/communications/MessagePreview.tsx` : `Dialog` « Aperçu du message » avec objet, liste des pièces (prop facultative), `iframe` `sandbox=""` `srcdoc={html}` `title="Aperçu du message"`, hauteur fixe défilante, rendu dès 320 px
- [X] T028 [US1] Écran `web/app/routes/app/admin/communication-new.tsx` (contracts/ui-routes.md) : `clientLoader` préchargeant les saisons (`ensureQueryData`) ; `react-hook-form` ; clé de préparation `crypto.randomUUID()` créée au montage (`useState` initialisé une fois) ; section « Destinataires » (`Select` saisons ouvertes et closes, courante par défaut ; `RadioGroup` « Tous les adhérents », « À jour », « En impayé » par défaut) ; `AudienceSummary` ; section « Message » (objet `maxLength` 150 avec compteur, message `maxLength` 10 000 avec compteur et aide sur les liens) ; actions « Aperçu » et « Envoyer » ; `ConfirmDialog` « Envoyer cette communication ? » rappelant objet, `targetLabel`, nombre de destinataires, bouton « Envoyer à {n} destinataires » ; sur `409` : alerte `warn` « Le nombre de destinataires a changé : {nouveau} au lieu de {ancien}. » dans le dialogue et confirmation au nouveau nombre ; erreurs `422` sur les champs ; succès : notification annoncée « Communication envoyée à {n} destinataires. » et navigation vers `/admin/communication/{id}` ; `useBlocker` « Abandonner cette communication ? » si le formulaire est modifié
- [X] T029 [US1] Écran minimal `web/app/routes/app/admin/communication-detail.tsx` : `clientLoader` (`communicationQuery`), en-tête (objet, « Envoyée le … par … »), blocs « Cible » (saison, `targetLabel`, adhérents ciblés, non joignables, destinataires) et « Message » (`whitespace-pre-wrap`) ; `404` → page « introuvable » existante — la liste des destinataires arrive en US4
- [X] T030 [US1] Scénario Playwright `web/e2e/communications.spec.ts` (« relance des impayés ») : jeu de données via un nouveau mode `communication` de `internal/tools/e2eseed/main.go` (à côté du mode `carte` et de `seedMemberships`) créant exactement les membres de quickstart.md › validation manuelle : saison 2026-2027 ouverte, tarifs « Jeune » et « Adulte », compte Claire (`claire@example.test`) gérant Léo et Inès Martin (Jeune, à payer), compte Paul (`paul@example.test`, Adulte, paiement annoncé par chèque), Anne Petit sans compte avec `anne@example.test` (Adulte, payée), Hugo Roux sans compte ni adresse (Jeune, à payer), compte Zoé Blanc (Jeune, gratuite à 0 €), compte Marc Noir (Adulte, demande en attente), cible « En impayé », décompte annoncé, dialogue des non joignables, aperçu, confirmation, redirection ; Mailpit (`web/e2e/mailpit.ts`) : un message pour `claire@example.test`, un pour `paul@example.test`, expéditeur au nom de l'association, aucun autre ; axe sans violation `serious`/`critical` sur la préparation et le détail

**Checkpoint**: US1 livrable seule ; SC-001, SC-002, SC-003 vérifiables.

---

## Phase 4: User Story 2 - Joindre des documents à une communication (Priority: P2)

**Goal**: l'administrateur joint jusqu'à 5 fichiers vérifiés, en retire, les voit dans l'aperçu et la confirmation ; chaque destinataire les reçoit intacts ; l'historique permet de les télécharger ; les dépôts abandonnés sont purgés.

**Independent Test**: joindre deux fichiers valides, tenter un fichier trop gros puis un type refusé, envoyer, vérifier dans Mailpit les deux documents identiques, les télécharger depuis le détail.

### Tests for User Story 2

- [X] T031 [P] [US2] Tests en table in `internal/communication/attachment_test.go` avec des fichiers minimaux générés dans le test (`archive/zip` pour DOCX, DOCM, XLSX, ODT, ODS, ZIP quelconque ; en-têtes PDF, PNG, JPEG ; binaire `MZ` nommé `.pdf` ; conteneur OLE `D0 CF 11 E0` pour `.doc`) : 7 types acceptés avec le bon type MIME, refus des autres ; `[Content_Types].xml` de plus de 64 Kio refusé sans tout décompresser ; `SanitizeFilename` (`../../etc/passwd` → `passwd`, `C:\dossier\a.pdf` → `a.pdf`, caractères de contrôle retirés, nom accentué conservé, 200 caractères → 150 avec extension conservée, vide → `piece-jointe`) ; `NewStoredName` produit 32 hex, `ValidStoredName` refuse tout le reste
- [X] T032 [P] [US2] Tests du stockage in `internal/communication/store_test.go` : écriture atomique (aucun fichier temporaire restant), taille et SHA-256 renvoyés, `Open` d'un nom invalide refusé sans accès disque, `Remove` idempotent, `Orphans(known, olderThan)`
- [X] T033 [P] [US2] Tests d'API des pièces in `internal/api/communication_attachments_test.go` (multipart construit avec `mime/multipart` ; `apitest` avec `DataDir` temporaire) : dépôt PDF → `201` et fichier présent sous `communications/` ; type refusé, fichier vide → `422` sur `body.file` avec les messages de contracts/api.md ; 5 Mio + 1 octet → `413` ou `422` « Le fichier dépasse 5 Mo. » ; 6ᵉ fichier → `422` ; total > 10 Mio (trois fichiers de 4 Mio) → `422` avec le total déjà joint ; `draftKey` déjà envoyée → `422` ; suppression par un autre administrateur ou après envoi → `404` ; envoi : pièces rattachées, `outbound_emails` porteurs de `communication_id` ; fichier altéré sur disque avant envoi → `422` « Une pièce jointe est illisible… » ; téléchargement : en-têtes `Content-Type`, `Content-Disposition` avec `filename*=UTF-8''` pour « Convocation AG été.pdf », `nosniff`, `no-store`, contenu identique ; pièce de préparation ou d'une autre communication → `404` ; journaux sans nom de fichier
- [X] T034 [P] [US2] Tests du worker in `internal/mail/worker_test.go` : e-mail avec `communication_id` → le faux `Sender` reçoit les pièces (noms d'origine, types, contenu identique), liste des pièces lue une seule fois par lot ; fichier manquant ou SHA-256 différent → `failed` immédiat sans nouvelle tentative, journal sans adresse ; e-mails sans communication inchangés ; débit (SC-004) : 500 e-mails de communication portant une pièce de 5 Mio, faux `Sender` instantané → tous `sent` en 25 appels de `Tick` (lots de 20, soit ≈ 13 min avec l'intervalle par défaut de 30 s), pièce lue depuis le disque sans charger l'ensemble des fichiers en mémoire
- [X] T035 [P] [US2] Tests du balayage in `internal/api/communication_sweeper_test.go` (horloge `env.Advance`) : pièce de préparation de 25 h supprimée (ligne et fichier), de 23 h conservée, pièce rattachée conservée, fichier orphelin de plus d'une heure supprimé, fichier orphelin récent conservé

### Implementation for User Story 2

- [X] T036 [P] [US2] Implémenter in `internal/communication/attachment.go` (research R4, R5) : constantes des 7 types acceptés, `DetectType(r io.ReaderAt, size int64) (string, error)` (`http.DetectContentType` sur les 512 premiers octets pour PDF/JPEG/PNG ; pour une archive ZIP, `zip.NewReader` : ODF si première entrée `mimetype` stockée sans compression et valant `application/vnd.oasis.opendocument.text` ou `…spreadsheet` ; OOXML si `[Content_Types].xml` (lecture limitée à 64 Kio décompressés) déclare `…wordprocessingml.document.main+xml` ou `…spreadsheetml.sheet.main+xml` et aucun type `macroEnabled` ; sinon `ErrUnsupportedType`), `SanitizeFilename`, `NewStoredName()` (`crypto/rand`, 16 octets en hex), `ValidStoredName`, `CheckLimits(existingCount int, existingBytes, newBytes int64) error` (erreurs typées `ErrTooManyFiles`, `ErrFileTooLarge`, `ErrTotalTooLarge`)
- [X] T037 [P] [US2] Implémenter in `internal/communication/store.go` : `FileStore{Dir string}` (racine `FRIGG_DATA_DIR/communications`, créée en `0750`), `Save(r io.Reader, max int64) (stored string, size int64, sha string, err error)` (fichier temporaire du même répertoire, `0640`, copie bornée, `fsync`, renommage ; dépassement → `ErrFileTooLarge` et nettoyage), `Open(stored) (*os.File, error)` et `Remove(stored) error` (refus si `!ValidStoredName`), `Verify(stored, size, sha) error`, `Orphans(known map[string]bool, olderThan time.Time) ([]string, error)`
- [X] T038 [US2] Câblage : `Deps.Files *communication.FileStore` in `internal/api/api.go` ; création dans `internal/cli/serve.go` à partir de `cfg.DataDir` ; option `DataDir` temporaire (`t.TempDir()`) dans `internal/api/apitest/apitest.go`
- [X] T039 [US2] Opération `upload-communication-attachment` (`POST /admin/communications/attachments`, `auth.Admin`, `DefaultStatus` 201, `MaxBodyBytes: 5<<20 + 64<<10`) in `internal/api/communication_attachments.go` : `RawBody huma.MultipartFormFiles[struct{ DraftKey string \`form:"draftKey" pattern:"…"\`; File huma.FormFile \`form:"file" required:"true"\` }]` ; `draftKey` déjà utilisée par une communication → `422` ; `FileStore.Save` borné à `MaxFileBytes` ; `DetectType` sur le fichier enregistré (refus → suppression du fichier et `422`) ; dans `BEGIN IMMEDIATE`, `ListDraftAttachments` + `CheckLimits` puis `CreateCommunicationAttachment` (en cas d'échec : suppression du fichier) ; messages exacts de contracts/api.md (total formaté par `formatMegabytes`, « 7,2 Mo ») ; traduction du `413` de huma en « Le fichier dépasse 5 Mo. » ; journal « pièce jointe déposée » (`attachment_id`, `size`, `type`)
- [X] T040 [US2] Opération `delete-communication-attachment` (`DELETE /admin/communications/attachments/{id}`, `auth.Admin`, 204) in `internal/api/communication_attachments.go` : `GetDraftAttachment` (`id`, auteur courant, non rattachée) sinon `404` uniforme ; suppression de la ligne puis du fichier après commit (échec journalisé, rattrapé par le balayage)
- [X] T041 [US2] Étendre `send-communication` in `internal/api/communications.go` : dans la transaction, `ListDraftAttachments(draftKey, auteur)`, revérification de `CheckLimits` et `FileStore.Verify` de chaque fichier (`422` « Une pièce jointe est illisible ; retirez-la et ajoutez-la de nouveau. ») ; `AttachDraftToCommunication` ; `attachments` dans la réponse et dans `get-communication`
- [X] T042 [US2] Opération `download-communication-attachment` (`GET /admin/communications/{id}/attachments/{attachmentId}`, `auth.Admin`) in `internal/api/communication_attachments.go` : `GetCommunicationAttachment` sinon `404` ; réponse en flux (`huma.StreamResponse`) avec `Content-Type`, `Content-Length`, `Content-Disposition: attachment; filename="<repli ASCII>"; filename*=UTF-8''<url.PathEscape>`, `X-Content-Type-Options: nosniff`, `Cache-Control: private, no-store` ; fichier manquant → `500` générique journalisé
- [X] T043 [US2] Pièces jointes dans le worker : interface `AttachmentStore{Open(stored string) (io.ReadCloser, error)}` et champ `Worker.Attachments` in `internal/mail/worker.go` ; pour un lot, lecture de `ListCommunicationAttachments` par `communication_id` (cache local au lot) ; ouverture et contrôle taille/SHA-256 ; transmission via `Message.Attachments` ; erreur de fichier → `MarkEmailFailed` immédiat ; in `internal/mail/smtp.go`, `m.AttachReader(name, r, gomail.WithFileContentType(...))` pour chaque pièce ; câblage du `FileStore` dans `internal/cli/serve.go`
- [X] T044 [US2] Balayage `CommunicationAttachmentSweeper` in `internal/api/communication_sweeper.go` (modèle `online_sweeper.go`, intervalle 15 min, horloge injectable) : `ListExpiredDraftAttachments(now - 24 h)` → suppression ligne puis fichier ; seconde passe `FileStore.Orphans(ListAllStoredNames, now - 1 h)` ; journaux sans nom d'origine ; lancement dans `internal/cli/serve.go`
- [X] T045 [US2] Régénérer le contrat (`openapi.json`, `schema.d.ts`) comme en T024 ; ajouter à `web/app/api/communications.ts` `useUploadAttachment()` (`client.POST` avec `bodySerializer` construisant un `FormData` `draftKey` + `file`) et `useDeleteAttachment()`
- [X] T046 [P] [US2] Composant `web/app/communications/AttachmentsField.tsx` : bouton « Ajouter des fichiers » lié à `<input type="file" multiple accept=".pdf,.jpg,.jpeg,.png,.docx,.xlsx,.odt,.ods">` visuellement masqué mais focalisable ; aide (formats, 5 fichiers, 5 Mo, 10 Mo au total) ; un dépôt par fichier ; liste (icône selon type via `Icon`, nom, `formatBytes`, état « Envoi… », erreur `danger` sous le fichier refusé avec bouton de fermeture, bouton « Retirer » `aria-label="Retirer {nom}"`) ; total « {n} fichiers · {x} sur 10 Mo » ; expose `uploading: boolean` et la liste des pièces acceptées
- [X] T047 [US2] Intégrer `AttachmentsField` (section « Pièces jointes ») dans `web/app/routes/app/admin/communication-new.tsx` : « Envoyer » désactivé pendant un dépôt ; pièces listées dans `MessagePreview` et dans le `ConfirmDialog`
- [X] T048 [US2] Bloc « Pièces jointes » dans `web/app/routes/app/admin/communication-detail.tsx` : liens `<a href="/api/admin/communications/{id}/attachments/{aid}" download>` avec nom et taille
- [X] T049 [US2] Scénario Playwright « pièces jointes » in `web/e2e/communications.spec.ts` : ajout d'un PDF et d'un ODS, refus d'un exécutable renommé `.pdf` et d'un fichier de 6 Mo (messages visibles), retrait, envoi ; Mailpit : pièce présente avec le nom d'origine et la même taille ; téléchargement depuis le détail ; axe

**Checkpoint**: US1 + US2 fonctionnent ; SC-006 vérifiable.

---

## Phase 5: User Story 3 - Choisir librement la cible (Priority: P2)

**Goal**: cible personnalisée par situations de paiement et par tarifs ; décompte mis à jour à chaque changement.

**Independent Test**: comparer le décompte de chaque cible prédéfinie et d'une cible personnalisée (« À payer » + tarif « Adulte ») aux adhésions attendues.

### Tests for User Story 3

- [X] T050 [P] [US3] Compléter `internal/api/communications_audience_test.go` : restriction à un tarif, à deux tarifs, tarif d'une autre saison → `422` sur `query.feeIds`, plus de 50 tarifs → `422`, situations dupliquées acceptées et normalisées ; `targetLabel` et `preset = custom` ; envoi avec cible personnalisée : `situations` et `fee_ids` stockés triés, cible décodée identique dans `get-communication`
- [X] T051 [P] [US3] Tests de composant in `web/app/ui/Checkbox.test.tsx` : coche au clavier (Espace), état `aria-checked`, groupe avec légende accessible

### Implementation for User Story 3

- [X] T052 [P] [US3] Composant `web/app/ui/Checkbox.tsx` : `Checkbox` et `CheckboxGroup` sur `@base-ui/react` (`Checkbox.Root`, `Checkbox.Indicator`, `CheckboxGroup`), habillés avec les tokens Tailwind (focus visible, état désactivé), icône `Check` via `Icon`, légende de groupe (`fieldset`/`legend` ou `aria-labelledby`), intégrable à `react-hook-form` (`value`/`onValueChange`)
- [X] T053 [US3] Dans `web/app/routes/app/admin/communication-new.tsx` : option « Personnalisée » du `RadioGroup` révélant les groupes « Situation de paiement » (À jour, Paiement annoncé, À payer ; au moins une requise, message sinon) et « Tarifs » (tarifs de la saison via `web/app/api/membership-fees.ts`, aucun coché = tous, aide le précisant) ; passage d'un préréglage à « Personnalisée » pré-coche ses situations ; changement de saison → tarifs réinitialisés ; le décompte suit chaque changement (clé de requête)
- [X] T054 [US3] Scénario Playwright « cibles » in `web/e2e/communications.spec.ts` : « Tous les adhérents » (6 ciblés, 4 destinataires, 1 non joignable), « À jour » (Anne, Zoé), « Personnalisée » À payer + Adulte → 0, envoi impossible avec message ; navigation au clavier dans les cases ; axe

**Checkpoint**: US1 à US3 fonctionnent indépendamment.

---

## Phase 6: User Story 4 - Consulter l'historique des communications (Priority: P3)

**Goal**: liste paginée des communications envoyées et suivi de l'état d'envoi de chaque destinataire.

**Independent Test**: envoyer deux communications, vérifier ordre et colonnes de la liste, puis le détail avec un destinataire en échec et un en attente.

### Tests for User Story 4

- [X] T055 [P] [US4] Tests d'API in `internal/api/communications_history_test.go` : `list-communications` triée de la plus récente à la plus ancienne, pagination (`page`, `perPage` 1..100, `total`), compteurs `delivery` et `attachmentsCount`, auteur supprimé → `null` ; `get-communication` : destinataires triés `failed`, `pending`, `sent` ; worker (`env.FlushMail` avec faux SMTP en échec) → destinataire `failed` ; purge `PurgeSentEmails` → état `sent` conservé ; communication inexistante → `404` ; FR-020 : le schéma OpenAPI ne comporte aucune opération `PUT`, `PATCH` ni `DELETE` sur `/admin/communications/{id}`

### Implementation for User Story 4

- [X] T056 [US4] Opération `list-communications` (`GET /admin/communications`, `auth.Admin`) in `internal/api/communications.go` : `page` (≥ 1, défaut 1), `perPage` (1..100, défaut 25), réponse `{items: CommunicationSummary[], total}` ; compléter `get-communication` avec `recipients` triés et `delivery`
- [X] T057 [US4] Régénérer le contrat comme en T024 ; ajouter `communicationsQuery(page)` à `web/app/api/communications.ts` ; dans `communicationQuery`, `refetchInterval: 15_000` tant que `delivery.pending > 0`
- [X] T058 [P] [US4] Composant `web/app/communications/DeliveryBadge.tsx` : badges `ok` « Envoyé », `info` « En attente », `danger` « En échec » (`ui/Badge`) et résumé « {s} envoyés · {p} en attente · {f} en échec »
- [X] T059 [US4] Remplacer l'index provisoire `web/app/routes/app/admin/communications.tsx` par l'historique : `clientLoader` (`communicationsQuery`), `ui/Table` (Date, Objet lien vers le détail, Cible, Auteur, Destinataires, Envoi avec `DeliveryBadge`), `ui/Pagination` reflétée dans `?page=`, état vide (icône `Mail`, « Aucune communication pour l'instant », texte et bouton « Nouvelle communication »)
- [X] T060 [US4] Bloc « Destinataires » dans `web/app/routes/app/admin/communication-detail.tsx` : résumé `DeliveryBadge`, tableau dans `ScrollRegion` (Adresse — « Adresse supprimée » en `muted` si `null` —, Adhérents, État, Envoyé le), échecs en tête
- [X] T061 [US4] Scénario Playwright « historique » in `web/e2e/communications.spec.ts` : deux communications listées dans l'ordre, état vide sur base sans communication, détail avec états « Envoyé » après relève Mailpit ; axe sur la liste et le détail ; passage à 320 px de large sur la préparation, la liste et le détail (aucun défilement horizontal de la page, tableaux dans `ScrollRegion`)

**Checkpoint**: les quatre stories fonctionnent ; SC-005 vérifiable.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T062 Effacement RGPD : dans `delete-account` (`internal/api/accounts.go`), à côté de `DeleteEmailsByRecipient`, appeler `EraseRecipientEmail` avec l'adresse du compte dans la même transaction ; test in `internal/api/accounts_delete_test.go` (adresse `NULL` dans l'historique, e-mail de communication en attente supprimé → destinataire `failed`, suppression du compte de l'administrateur auteur → communication et pièces jointes conservées, `author_id` et `uploaded_by` à NULL, fichier toujours téléchargeable)
- [X] T063 [P] Revue de sécurité ciblée (plan.md › points de vigilance 1 à 5) : aucun `content_type` écrit depuis l'en-tête multipart, aucun chemin construit hors `FileStore`, aucune donnée personnelle dans les journaux (`grep` des appels `slog` des nouveaux fichiers) ; corriger le cas échéant
- [X] T064 [P] Documentation utilisateur in `README.md` (ou documentation d'utilisation existante) : rubrique « Communication », cibles, adresse de contact retenue, limites et formats des pièces jointes, purge des dépôts abandonnés, sauvegarde du répertoire `communications/` avec le répertoire de données
- [X] T065 Exécuter `go tool task check` (build `CGO_ENABLED=0`, `go vet`, `go test ./...`, `sqlc generate` et `frigg openapi` sans diff, migrations sur base vierge, `bun run typecheck`, Biome, Vitest, `bun run build`, Playwright + axe) et dérouler quickstart.md › validation manuelle (étapes 1 à 11)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance. T001 avant T002, T003, T004 ; T005 indépendante.
- **Foundational (Phase 2)** : dépend de la phase 1 (code sqlc de T003/T004). T009 avant T010, T011, T012 ; T013 avant T014. Bloque toutes les stories.
- **US1 (Phase 3)** : dépend de la phase 2. MVP.
- **US2 (Phase 4)** : dépend de la phase 2 ; T041, T047, T048 modifient des fichiers créés en US1 (T022, T028, T029) : à faire après eux.
- **US3 (Phase 5)** : dépend de la phase 2 ; T053 modifie l'écran de T028.
- **US4 (Phase 6)** : dépend de la phase 2 ; T056 complète `get-communication` (T023), T060 l'écran de T029.
- **Polish (Phase 7)** : après les stories voulues.

### User Story Dependencies

- **US1 (P1)** : aucune autre story ; l'API d'audience accepte déjà situations et tarifs (utilisés par US3 côté écran).
- **US2 (P2)** : côté API et domaine, indépendante (T031–T040, T042–T044) ; intégration à l'envoi et aux écrans après US1.
- **US3 (P2)** : écran seulement, après US1.
- **US4 (P3)** : API indépendante ; écrans après US1.

### Within Each User Story

- Tests d'abord (ils doivent échouer), puis domaine/requêtes, puis handlers, puis régénération du contrat, puis frontend, puis Playwright.

### Parallel Opportunities

- Phase 1 : T002, T003, T004 après T001 ; T005 à tout moment.
- Phase 2 : T006, T007, T008 en parallèle ; T011 et T012 après T009 ; T015, T016 en parallèle du backend.
- US1 : T017, T018, T019 en parallèle ; T025, T026, T027 en parallèle des handlers une fois T024 fait.
- US2 : T031–T035 en parallèle ; T036, T037 en parallèle ; T046 en parallèle des handlers.
- US2 (domaine, fichiers, worker, balayage) et US4 (API) peuvent avancer en parallèle de US1.

---

## Parallel Example: User Story 1

```text
# Tests (fichiers distincts) :
T017 internal/api/communications_audience_test.go
T018 internal/api/communications_send_test.go
T019 internal/api/access_test.go

# Frontend après régénération du contrat (T024) :
T025 web/app/api/communications.ts
T026 web/app/communications/AudienceSummary.tsx
T027 web/app/communications/MessagePreview.tsx
```

## Parallel Example: User Story 2

```text
# Tests :
T031 internal/communication/attachment_test.go
T032 internal/communication/store_test.go
T033 internal/api/communication_attachments_test.go
T034 internal/mail/worker_test.go
T035 internal/api/communication_sweeper_test.go

# Domaine :
T036 internal/communication/attachment.go
T037 internal/communication/store.go
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 (migration, requêtes) puis phase 2 (cible, audience, rendu, câblage).
2. US1 : audience, aperçu, envoi idempotent, détail minimal, écran de préparation.
3. **STOP et valider** : tests d'API, Playwright avec Mailpit ; relance des impayés de bout en bout.

### Incremental Delivery

1. US1 → relance des impayés et cibles prédéfinies.
2. US2 → pièces jointes (convocations, comptes rendus).
3. US3 → cible personnalisée par situations et tarifs.
4. US4 → historique et suivi des envois.
5. Phase 7 → effacement RGPD, revue de sécurité, documentation, contrôles complets.

### Notes

- Chaque tâche se termine par ses tests verts et un commit (gitmoji, message
  court, sans co-auteur, après validation de l'utilisateur).
- Après chaque ajout d'opération : `frigg openapi` et `bun run gen:api` avant
  le frontend ; le contrôle « sans diff » de la constitution l'exige.
