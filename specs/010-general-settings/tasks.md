---

description: "Liste des tâches : paramètres généraux de l'association"
---

# Tasks: Paramètres généraux de l'association

**Input**: Design documents from `specs/010-general-settings/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : matrice d'autorisation (principes II et III, avec deux
opérations publiques nouvelles), tests d'API sur base isolée, Playwright + axe
sur chaque parcours touché. La sécurité du logo (type vérifié sur le contenu,
bombes de pixels, réencodage) et la garantie de contraste ont leurs propres
tests.

**Organization**: une phase par user story, dans l'ordre de priorité de la spec.
US1 (P1) livre la section « Général » avec le nom et la tagline, affichés dans la
barre latérale et sur l'accueil membre. US2 (P2) y ajoute l'identité légale et
les coordonnées. US3 (P2) livre la section « Apparence » (logo, couleurs) et
l'identité visuelle hors session et dans les e-mails. US4 (P3) ajoute les réseaux
sociaux et l'encart « Contacter l'association ».

**Conventions** (constitution 1.8.0) : identifiants, tables, champs JSON et
valeurs en anglais ; commentaires, messages, textes d'interface, journaux et
e-mails en français. Commentaires de la migration et des requêtes sqlc en ASCII
uniquement. Jamais de valeur de coordonnées (adresse, e-mail, téléphone, liens)
dans les journaux : seulement les noms des champs modifiés. Aucun e-mail envoyé
dans une requête HTTP. Sans session, seules `get-identity` et
`get-identity-logo` exposent des données de l'association : nom, couleurs,
logo, et rien d'autre.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US4)

## Path Conventions

Module Go à la racine (`internal/`), SPA dans `web/` (plan.md › Project
Structure). Code existant à réutiliser :

- API : `internal/api/api.go` (`register`, `Deps`, `server`, `s.now()`,
  `s.today()`, `CurrentAccount`, `s.internalError`), `internal/api/errors.go`
  (`FieldErrors`, `FieldErrorDetails`), `internal/api/settings.go` (modèle
  d'opérations de paramètres, `tagSettings`, `MsgTextTooLong`, erreurs groupées
  par champ comme l'IBAN), `internal/api/me.go` (`sessionOutput`,
  `associationName`), `internal/api/types.go` (`Association`, `Session`,
  `defaultAssociationName`), `internal/api/mail.go` (`enqueueMail`,
  `enqueueNotice`), `internal/api/communication_attachments.go` (dépôt
  multipart `huma.MultipartFormFiles`, `MaxBodyBytes`, en-têtes de
  téléchargement), `internal/api/communications.go` (`preview-communication`).
- Domaine : `internal/communication` (`FileStore` : `Save`, `Open`, `Read`,
  noms `^[0-9a-f]{32}$` ; `SanitizeFilename`), `internal/season` (`Today`).
- Base : `internal/database` (`Store.Read`, `Store.WriteTx`, `BEGIN IMMEDIATE`),
  `internal/database/queries/association.sql`, `internal/database/queries/emails.sql`.
- Mail : `internal/mail/render.go` (`Render`, `Data`, `Message`, `Paragraphs`),
  `internal/mail/outbox.go` (`Enqueue`, `TimeFormat`), `internal/mail/worker.go`
  (`Tick`, `deliver`), `internal/mail/smtp.go`, `internal/mail/templates/*`,
  `internal/mail/mailtest`.
- CLI : `internal/cli/serve.go` (câblage du worker, `FileStore`),
  `internal/cli/createadmin.go`, `internal/tools/e2eseed/main.go`.
- Tests Go : `internal/api/apitest` (`New`, `Seed`, `LoggedIn`, `SetToday`,
  `FlushMail`, `Mail`, `Logs`, `Store`, `Client().Do`),
  `internal/api/access_test.go` (matrice, `allowedPublic`),
  `internal/database/db_test.go`.
- Frontend : `web/app/routes.ts`, `web/app/root.tsx` (`clientLoader`,
  `HydrateFallback`), `web/app/app.css` (variables `--brand` etc. en triplets
  RVB), `web/app/shell/Sidebar.tsx`, `web/app/shell/AppShell.tsx`,
  `web/app/routes/app/admin/SettingsPage.tsx` (`settingsSections`),
  `web/app/routes/app/admin/settings-index.tsx`,
  `web/app/routes/app/admin/settings-payments.tsx` (modèle de section),
  `web/app/routes/app/member/home.tsx`, `web/app/routes/auth/layout.tsx`,
  `web/app/api/*` (`queries.ts`, `settings.ts`, `problem.ts` : 422 →
  `setError`), `web/app/ui/*` (`Button`, `Field`, `Alert`, `ConfirmDialog`,
  `Icon`), `web/app/lib/format.ts`, `useBlocker` comme dans
  `web/app/routes/app/admin/communication-new.tsx`. E2E : `web/e2e/test.ts`,
  `web/e2e/axe.ts`, `web/e2e/mailpit.ts`, `web/e2e/admin-settings.spec.ts`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: schéma et requêtes communs à toutes les stories.

- [X] T001 Créer la migration `internal/database/migrations/00010_association_settings.sql` (commentaires ASCII) selon data-model.md : `ALTER TABLE association ADD COLUMN` pour chaque colonne, toutes nullable avec CHECK `IS NULL OR …` : `tagline` « `length(tagline) BETWEEN 1 AND 150` », `short_name` « `length BETWEEN 1 AND 20` », `purpose` « `length BETWEEN 1 AND 2000` », `rna` « `rna GLOB 'W[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]'` », `siret` « `length = 14 AND siret NOT GLOB '*[^0-9]*'` », `founded_on` « `date(founded_on) = founded_on` », `address_line1`, `address_line2`, `city` « `length BETWEEN 1 AND 100` », `postal_code` (5 chiffres par GLOB), `contact_email` « `length ≤ 254` », `phone` « 10 chiffres commençant par `0` » (GLOB `0[0-9]…`), `website`, `facebook_url`, `instagram_url`, `linkedin_url`, `youtube_url`, `mastodon_url`, `bluesky_url` « `length ≤ 300`, commence par `http://` ou `https://` », `primary_color`, `accent_color` « `#` + 6 hexadécimaux en minuscules » (GLOB `#[0-9a-f]…`) ; table `association_logo` (`id INTEGER PRIMARY KEY CHECK (id = 1)`, `original_name TEXT NOT NULL`, `source_type TEXT NOT NULL CHECK (source_type IN ('image/png','image/jpeg','image/webp'))`, `source_size_bytes INTEGER NOT NULL CHECK (source_size_bytes BETWEEN 1 AND 1048576)`, `web_name TEXT NOT NULL UNIQUE` et `email_name TEXT NOT NULL UNIQUE` (GLOB 32 caractères `[0-9a-f]`), `web_type`/`email_type` `CHECK IN ('image/png','image/jpeg')`, `web_size_bytes`, `web_width`, `web_height` (côté le plus long ≤ 512), `web_sha256`, `email_size_bytes CHECK (email_size_bytes BETWEEN 1 AND 102400)`, `email_width ≤ 400`, `email_height ≤ 96`, `email_sha256`, `uploaded_at TEXT NOT NULL`, `uploaded_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL`) STRICT ; `ALTER TABLE outbound_emails ADD COLUMN wrap INTEGER NOT NULL DEFAULT 0 CHECK (wrap IN (0, 1))` et `ADD COLUMN footer TEXT` ; section `Down` : `DROP TABLE association_logo`, `ALTER TABLE association DROP COLUMN` pour chaque colonne ajoutée, reconstruction de `outbound_emails` à l'identique de 00009 (colonnes `communication_id` avec sa clé étrangère et `from_name`, index `outbound_emails_status_next_idx` et `outbound_emails_communication_id`, et les deux déclencheurs de 00009 recréés entre `-- +goose StatementBegin/End`)
- [X] T002 [P] Test de migration in `internal/database/db_test.go` (`TestAssociationSettingsMigration`) : base vierge ; base migrée jusqu'à `00009` avec un nom d'association et des `outbound_emails` en attente liés à une communication (conservés, `wrap = 0`, `footer` NULL, déclencheurs de 00009 toujours actifs) ; CHECK refusant `rna = 'W12'`, `siret = '1234'`, `primary_color = '#FFF'`, `website = 'ftp://x'`, une deuxième ligne `association_logo` (`id = 2`) ; `Down` de `00010` puis `Up` de nouveau sans erreur, e-mails et déclencheurs conservés
- [X] T003 Étendre `internal/database/queries/association.sql` (commentaires ASCII) : `GetAssociation` renvoie toute la ligne (`SELECT * … WHERE id = 1`) ; `UpdateGeneralSettings` (toutes les colonnes « Général » sauf couleurs, plus `updated_at`) ; `UpdateAssociationColors` (`primary_color`, `accent_color`, `updated_at`, NULL admis) ; `GetAssociationLogo`, `UpsertAssociationLogo` (`INSERT … ON CONFLICT (id) DO UPDATE`), `DeleteAssociationLogo` ; dans `internal/database/queries/emails.sql`, ajouter `wrap` et `footer` à `EnqueueEmail` et `EnqueueCommunicationEmail` ; lancer `sqlc generate` (code commité in `internal/database/sqlc/`)
- [X] T004 Adapter les appelants de `GetAssociation` à la ligne complète (champ `Name`) : `internal/api/me.go` (`associationName`), `internal/api/mail.go`, `internal/cli/createadmin.go`, `internal/tools/e2eseed/main.go`, `internal/database/db_test.go` ; faire passer `wrap = 0`, `footer = NULL` dans `internal/mail/outbox.go` et dans l'appel `EnqueueCommunicationEmail` de `internal/api/communications.go` (comportement inchangé jusqu'à US3) ; `go build ./...` et `go test ./...` passent

**Checkpoint**: schéma et requêtes générés ; comportement de l'application inchangé.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: domaine de normalisation, opérations de paramètres et squelette des écrans partagés par toutes les stories.

**⚠️ CRITICAL**: aucune story ne commence avant la fin de cette phase.

- [X] T005 [P] Créer le paquet `internal/association` : `doc.go` (commentaire de paquet en français : règles pures des paramètres de l'association, sans base) ; `settings.go` : `Settings` (champs « Général » en `*string`, `FoundedOn *season.Date` ou `*string` ISO), `FieldError{Field, Message string}` (`Field` = chemin JSON camelCase, par exemple `address.postalCode`, `social.instagram`), fonction `Trim(*string) *string` (espaces de bord retirés, chaîne vide → nil, FR-014) et `Normalize(in Settings, today season.Date) (Settings, []FieldError)` qui, à ce stade, traite seulement `name` (obligatoire : « Indiquez le nom de l'association. », 1–120 caractères en runes : « Le nom ne doit pas dépasser 120 caractères. »), `tagline` (≤ 150), `shortName` (≤ 20), `purpose` (≤ 2000) ; les messages viennent de contracts/api.md › Messages d'erreur
- [X] T006 [P] Tests en table in `internal/association/settings_test.go` : nom vide, fait d'espaces, 120 et 121 runes (avec accents) ; tagline de 150 et 151 caractères ; chaîne vide → nil ; espaces de bord retirés ; plusieurs erreurs renvoyées ensemble
- [X] T007 Créer `internal/api/general_settings.go` : types huma de contracts/api.md (`GeneralSettings` complet avec `Address` et `SocialLinks` déclarés dès maintenant, bornes `maxLength` de data-model.md, `format:"email"` sur `contactEmail`, `pattern:"^[0-9]{5}$"` sur `address.postalCode`, `format:"date"` sur `foundedOn`, champs nullable), `func (s *server) registerGeneralSettings(api huma.API)` appelée depuis `New` in `internal/api/api.go` ; opérations `get-general-settings` (`GET /admin/settings/general`, `auth.Admin`, tag `settings`) et `update-general-settings` (`PUT`, `auth.Admin`) : `Normalize` avec `s.today()`, erreurs converties en un seul `422` par `FieldErrorDetails` avec `Location: "body." + Field` (tout-ou-rien, FR-013), écriture `UpdateGeneralSettings` en `WriteTx`, réponse normalisée ; journal `info` « paramètres généraux modifiés » avec `account_id` et la liste des **noms** de champs modifiés, sans valeurs (FR-031)
- [X] T008 [P] Sections de « Paramètres » in `web/app/routes/app/admin/SettingsPage.tsx` (`settingsSections` : « Général » `/admin/parametres/general`, « Apparence » `/admin/parametres/apparence`, « Paiements ») et `web/app/routes.ts` (`route("general", "routes/app/admin/settings-general.tsx")`, `route("apparence", "routes/app/admin/settings-appearance.tsx")` sous `parametres`) ; `web/app/routes/app/admin/settings-index.tsx` redirige vers `/admin/parametres/general` (FR-001) ; créer `settings-appearance.tsx` provisoire (`SettingsPage section="Apparence"` avec un texte « Bientôt disponible. ») remplacé en US3 ; mettre à jour `web/e2e/admin-settings.spec.ts` si elle suppose l'arrivée sur « Paiements »

**Checkpoint**: `go test ./...` et `bun run typecheck` passent ; « Paramètres » ouvre « Général ».

---

## Phase 3: User Story 1 - Renseigner le nom et la tagline (Priority: P1) 🎯 MVP

**Goal**: un administrateur modifie le nom et la tagline ; le nom s'affiche partout (barre latérale, onglet, pied de page, e-mails) et la tagline sous le nom dans la barre latérale et dans l'en-tête de l'accueil membre.

**Independent Test**: modifier nom et tagline, enregistrer, constater la barre latérale à jour sans rechargement ; se connecter en membre : tagline dans la barre latérale et sur l'accueil ; déclencher une invitation : nouveau nom dans l'e-mail.

### Tests for User Story 1

- [X] T009 [P] [US1] Tests d'API in `internal/api/general_settings_test.go` : `get-general-settings` sur une base neuve (nom saisi par `Seed`, autres champs `null`) ; `update-general-settings` nom + tagline → `200`, valeurs relues ; nom vide ou d'espaces → `422` sur `body.name`, rien d'enregistré ; tagline de 151 caractères → `422` sur `body.tagline` ; tagline vide → `null` ; `get-session` renvoie `association.tagline` ; une invitation mise en file après le renommage porte le nouveau nom (`apitest.Mail`) ; journal « paramètres généraux modifiés » contenant `name`, `tagline` et pas leurs valeurs (`env.Logs`)
- [X] T010 [P] [US1] Vérifier que la matrice `internal/api/access_test.go` couvre `get-general-settings` et `update-general-settings` (anonyme → `401`, membre → `403`) sans modifier `allowedPublic`

### Implementation for User Story 1

- [X] T011 [US1] Étendre `Association` in `internal/api/types.go` à `{name, tagline: string|null, logoUrl: string|null}` (`logoUrl` reste `null` jusqu'à US3) et `sessionOutput` in `internal/api/me.go` pour la remplir depuis `GetAssociation`
- [X] T012 [US1] Régénérer le contrat avec `go tool task gen` (`frigg openapi` → `web/app/api/openapi.json`, `bun run gen:api` → `web/app/api/schema.d.ts`), fichiers commités
- [X] T013 [P] [US1] Requêtes et mutation in `web/app/api/settings.ts` : `generalSettingsQuery`, `useUpdateGeneralSettings()` invalidant `get-general-settings` et `get-session` (FR-027)
- [X] T014 [US1] Écran `web/app/routes/app/admin/settings-general.tsx` (contracts/ui-routes.md) : `clientLoader` (`ensureQueryData`) ; `SettingsPage section="Général"` ; `react-hook-form` ; `fieldset` « Identité » : Nom (requis, `maxLength` 120), Tagline (`maxLength` 150, compteur, aide « Affichée sous le nom dans le menu et sur l'accueil des membres. »), Nom court (`maxLength` 20), Objet (`textarea`, `maxLength` 2000) ; bouton « Enregistrer » ; succès : notification « Paramètres enregistrés. » et `reset` avec les valeurs renvoyées ; `422` : erreurs sur les champs via `web/app/api/problem.ts`, alerte récapitulative, focus sur le premier champ en erreur ; `useBlocker` + `ConfirmDialog` « Quitter sans enregistrer ? » si le formulaire est modifié (FR-030)
- [X] T015 [P] [US1] Barre latérale `web/app/shell/Sidebar.tsx` : tagline sous le nom (classe `truncate`, `title` = texte complet), puis l'indication d'espace existante ; barre repliée : nom et tagline en `sr-only` (FR-023) ; mettre à jour `web/app/shell/Sidebar.test.tsx` s'il existe
- [X] T016 [P] [US1] Accueil membre `web/app/routes/app/member/home.tsx` : tagline sous le titre de l'en-tête de page quand elle est renseignée, rien sinon (FR-024)
- [X] T017 [US1] Scénario Playwright `web/e2e/general-settings.spec.ts` (« nom et tagline ») : « Paramètres » arrive sur « Général », modification du nom et de la tagline, barre latérale et titre d'onglet à jour sans rechargement, nom vide refusé avec message sur le champ, confirmation en quittant avec des modifications, connexion membre : tagline dans la barre latérale et sur l'accueil, `/admin/parametres/general` refusé ; section « Général » utilisable dans une fenêtre de 320 px sans défilement horizontal ; axe sans violation `serious`/`critical`

**Checkpoint**: US1 livrable seule ; SC-001 et SC-002 vérifiables.

---

## Phase 4: User Story 2 - Renseigner l'identité légale et les coordonnées (Priority: P2)

**Goal**: l'administrateur complète RNA, SIRET, date de création, adresse du siège, adresse électronique de contact, téléphone et site web ; chaque champ renseigné est vérifié, l'enregistrement est tout-ou-rien.

**Independent Test**: tous les champs valides enregistrés et relus à l'identique (forme normalisée) ; une valeur invalide par champ refusée avec un message sur ce champ ; plusieurs erreurs signalées ensemble sans rien enregistrer.

### Tests for User Story 2

- [X] T018 [P] [US2] Tests en table in `internal/association/identifiers_test.go` : RNA (`W751234567` ok, `w751234567` → `W751234567`, `W12345678`, `X751234567`, `W75123456A` refusés) ; SIRET (`73282932000074` ok, `732 829 320 00074` → compact, clé fausse refusée, 13 chiffres refusés, établissement de La Poste `35600000000001` accepté par la règle « somme multiple de 5 ») ; téléphone (`01 23 45 67 89`, `01.23.45.67.89`, `+33 1 23 45 67 89`, `0033123456789` → `0123456789` ; `+262 262 12 34 56` → `0262123456` ; `00 12 34 56 78`, `+44 20 7946 0958`, `123` refusés) ; URL (`https://exemple.fr` ok, `www.exemple.fr`, `javascript:alert(1)`, `https://` sans hôte, 301 caractères refusés)
- [X] T019 [P] [US2] Compléter `internal/association/settings_test.go` : date de création demain refusée, aujourd'hui acceptée (horloge de Paris) ; code postal `7500` refusé ; erreurs groupées sur `siret`, `foundedOn` et `website` en un seul appel
- [X] T020 [P] [US2] Compléter `internal/api/general_settings_test.go` : enregistrement de tous les champs de quickstart.md › scénario 2 relus normalisés ; SIRET à clé fausse + date future + site sans schéma → un `422` avec trois `errors[].location` (`body.siret`, `body.foundedOn`, `body.website`) et aucune valeur modifiée en base ; `contactEmail` mal formée → `422` huma sur `body.contactEmail` ; journal sans aucune valeur de coordonnées (`env.Logs` ne contient ni l'adresse, ni le téléphone, ni le SIRET)

### Implementation for User Story 2

- [X] T021 [P] [US2] Implémenter in `internal/association/identifiers.go` (research R2) : `NormalizeRNA` (majuscule, `^W[0-9]{9}$`), `NormalizeSIRET` (espaces retirés, 14 chiffres, clé de Luhn, exception SIREN `356000000` : somme des chiffres multiple de 5), `NormalizePhone` (espaces, points, tirets retirés ; `0[1-9]` + 8 chiffres ; `+33`/`0033` + 9 chiffres ; `+262`, `+269`, `+590`, `+594`, `+596` + 9 chiffres ramenés à `0` + 9 chiffres ; sortie à 10 chiffres), `ValidWebURL` (`net/url`, schéma `http`/`https`, hôte non vide, ≤ 300 caractères) ; messages de contracts/api.md
- [X] T022 [US2] Étendre `Normalize` in `internal/association/settings.go` : `rna`, `siret`, `foundedOn` (≤ `today` : « La date de création ne peut pas être dans le futur. »), adresse (`line1`, `line2`, `city` ≤ 100 ; `postalCode` 5 chiffres), `contactEmail` (trim, ≤ 254), `phone`, `website` ; vérifier que `update-general-settings` in `internal/api/general_settings.go` écrit tous ces champs
- [X] T023 [P] [US2] Ajouter `formatPhone(national: string): string` (« 01 23 45 67 89 ») et ses tests in `web/app/lib/format.ts` et `web/app/lib/format.test.ts`
- [X] T024 [US2] Compléter `web/app/routes/app/admin/settings-general.tsx` : `fieldset` « Identité légale » (Numéro RNA ; Numéro SIRET `inputmode="numeric"` ; Date de création `type="date"` avec `max` = aujourd'hui) et « Coordonnées » (Adresse, Complément, Code postal `inputmode="numeric"` `autocomplete="postal-code"`, Ville, Adresse électronique de contact `type="email"`, Téléphone `type="tel"` affiché avec `formatPhone`, Site web `type="url"`) ; erreurs `422` imbriquées (`address.postalCode`) reliées aux champs
- [X] T025 [US2] Compléter `web/e2e/general-settings.spec.ts` (« identité légale et coordonnées ») : saisie de quickstart.md › scénario 2 et relecture normalisée (`W751234567`, `73282932000074`, `01 23 45 67 89`), trois erreurs simultanées affichées, rechargement sans valeur du deuxième essai ; axe

**Checkpoint**: US1 + US2 fonctionnent ; SC-003 vérifiable pour les champs « Général ».

---

## Phase 5: User Story 3 - Personnaliser le logo et les couleurs (Priority: P2)

**Goal**: l'administrateur dépose un logo et choisit deux couleurs, avec aperçu ; logo et couleurs s'appliquent à toute l'interface, aux écrans hors session (avec le nom) et aux e-mails ; contraste ≥ 4,5:1 garanti ; retrait du logo et retour aux couleurs par défaut après confirmation.

**Independent Test**: déposer un logo, choisir une couleur claire, enregistrer ; se déconnecter : écran de connexion à l'identité de l'association, sans clignotement ; se connecter en membre : logo et couleurs ; demander une réinitialisation de mot de passe : e-mail avec bandeau, bouton et logo embarqué ; retirer le logo et rétablir les couleurs : retour à l'apparence de Frigg.

### Tests for User Story 3

- [X] T026 [P] [US3] Tests en table in `internal/brand/palette_test.go` : `ParseHex` (`#1F5278` → minuscules, `#fff`, `1f5278`, `#12345g` refusés) ; contraste WCAG (blanc/noir = 21, blanc/blanc = 1) ; `Derive` sur un balayage d'au moins 200 couleurs (pas régulier dans l'espace RVB, plus `#fff3b0`, `#ffff00`, `#777777`, `#000000`, `#ffffff`) : pour chaque palette, `contrast(OnBrand, Brand) ≥ 4.5`, `contrast(OnAccent, Accent) ≥ 4.5`, `contrast(BrandInk, canvas #f5f6f8) ≥ 4.5`, `contrast(AccentInk, canvas) ≥ 4.5`, et `BrandInk`/`AccentInk` ≥ 4,5 sur le fond teinté (couleur à 10 % sur `canvas`) ; `Brand` et `Accent` égaux aux couleurs choisies ; `Default` = `#1f5278` / `#c44822` (SC-004)
- [X] T027 [P] [US3] Tests in `internal/association/logo/process_test.go` avec des images générées dans le test ou rangées in `internal/association/logo/testdata/` : PNG transparent, JPEG avec segment EXIF/GPS, WebP sans perte acceptés ; SVG, GIF, PDF, PNG tronqué, texte renommé `.png` refusés ; PNG de 20 000 × 1 px et de 5 000 × 5 000 px refusés avant décodage complet ; variante `web` côté le plus long ≤ 512 ; variante `email` hauteur ≤ 96, largeur ≤ 400, ≤ 102400 octets (y compris pour une photo très détaillée) ; sortie JPEG pour une source JPEG, PNG sinon ; aucun segment EXIF (`APP1`) dans les sorties
- [X] T028 [P] [US3] Tests in `internal/mail/render_test.go` : `Render` ne renvoie plus que le contenu (pas de `<html>`) ; `Wrap` avec identité par défaut, avec palette claire (bandeau `brand`, texte `onBrand`, boutons, liens `brandInk` présents dans le HTML), avec logo (`<img src="cid:logo" alt="">` et pièce embarquée) et sans logo (aucune balise `<img>`) ; nom présent en texte dans les deux cas ; `Footer` remplace la mention par défaut ; version texte : nom en tête ; boucle sur les **9 gabarits de contenu** (`invitation`, `password-reset`, `email-verification`, `membership-validated`, `membership-refused`, `membership-cancelled`, `membership-paid`, `communication`, `test`) : `Wrap` avec une palette claire produit le bandeau et, pour ceux qui ont un bouton, un bouton aux couleurs de la palette, et aucun HTML ne contient plus `#1f5278` (SC-008)
- [X] T029 [P] [US3] Tests in `internal/mail/worker_test.go` : une ligne `wrap = 1` est habillée avec l'identité renvoyée par la fonction injectée **au moment du passage** (identité changée entre `Enqueue` et `Tick` → nouvel habillage) ; une ligne `wrap = 0` est envoyée telle quelle (avec son `from_name` éventuel) ; pour une ligne `wrap = 1`, le faux `Sender` reçoit `FromName` = nom de l'identité courante (y compris pour une invitation) et le logo embarqué quand l'identité en a un
- [X] T030 [P] [US3] Tests d'API in `internal/api/appearance_test.go` : `update-appearance` avec `#FFF3B0` → `200`, couleur stockée en minuscules, palette dérivée ; code invalide → `422` sur `body.primaryColor` ; `null` → couleur par défaut ; `preview-palette` n'écrit rien ; `upload-logo` PNG valide → `200`, deux fichiers dans le répertoire `logo/`, `association_logo` remplie, `originalName` assaini ; SVG → `422` sur `form.file`, logo précédent conservé ; PNG de 1 048 577 octets (sous la limite du corps) → `422` « Déposez une image PNG, JPEG ou WebP de 1 Mo au plus. » sur `form.file` ; corps de 2 Mio → `413` ; nouveau dépôt → anciens fichiers supprimés, ancienne URL → `404` ; `delete-logo` → `204`, répertoire vide, `204` de nouveau sans logo ; journaux « logo déposé », « logo retiré », « couleurs modifiées »
- [X] T031 [P] [US3] Tests d'API in `internal/api/identity_test.go` : `get-identity` anonyme → `200` avec **exactement** les clés `name`, `logoUrl`, `palette` (comparaison de la liste des clés JSON décodées), même avec tagline, coordonnées et SIRET renseignés ; nom jamais défini → « Mon association » ; `get-identity-logo` anonyme sur le nom courant → octets de la variante `web`, `Content-Type`, `X-Content-Type-Options: nosniff`, `Content-Disposition: inline`, `Cache-Control: public, max-age=31536000, immutable` ; sur le nom de la variante `email`, un ancien nom ou `../x` → `404` ; `get-session` renvoie `association.logoUrl` ; un e-mail mis en file après la fonctionnalité porte `wrap = 1`
- [X] T032 [US3] Ajouter `get-identity` et `get-identity-logo` à `allowedPublic` in `internal/api/access_test.go` avec le commentaire « Identité visuelle (amendement 1.8.0 de la constitution). » ; vérifier `update-appearance`, `preview-palette`, `get-appearance`, `upload-logo`, `delete-logo` → `admin`

### Implementation for User Story 3

- [X] T033 [US3] Ajouter la dépendance `golang.org/x/image` (`go get golang.org/x/image@latest`, `go mod tidy`) in `go.mod`/`go.sum` ; vérifier `CGO_ENABLED=0 go list -deps ./cmd/frigg | grep golang.org/x/image` et l'absence de paquet CGO
- [X] T034 [P] [US3] Créer le paquet `internal/brand` (research R3) : `color.go` (`RGB{R, G, B uint8}`, `ParseHex`, `Hex()`, `Triplet()` « `31 82 120` » pour les variables CSS, luminance relative WCAG, `Contrast(a, b RGB) float64`, conversions sRGB ↔ OKLCH) ; `palette.go` (`Palette{Brand, BrandInk, OnBrand, Accent, AccentInk, OnAccent RGB}`, `Default`, `Derive(primary, accent *RGB) Palette` : `on*` = blanc si contraste ≥ 4,5, sinon `ink` `#181f29` si ≥ 4,5, sinon noir ; `*Ink` = couleur assombrie par pas de luminosité OKLCH −0,02 jusqu'à un contraste ≥ 4,5 avec le plus foncé de `canvas` `#f5f6f8` et du fond teinté (couleur à 10 % sur `canvas`)) ; commentaire de paquet en français
- [X] T035 [P] [US3] Créer `internal/association/logo/process.go` (research R4) : décodeurs PNG, JPEG et `golang.org/x/image/webp` enregistrés ; `Process(r io.Reader) (Variants, error)` : lecture de 1 048 577 octets au plus et `ErrTooBig` si le fichier dépasse 1 048 576 octets, puis `image.DecodeConfig` (format parmi `png`, `jpeg`, `webp`, sinon `ErrUnsupported` ; plus de 16 000 000 pixels ou 8 000 px par côté → `ErrTooLarge`), puis `image.Decode` ; variante `web` (côté le plus long ≤ 512, sans agrandissement) et variante `email` (hauteur ≤ 96, largeur ≤ 400 ; si l'encodage dépasse 102 400 octets, réduction de 25 % et nouvel essai) redimensionnées avec `golang.org/x/image/draw.CatmullRom` ; encodage JPEG qualité 88 si la source est un JPEG (fond blanc), PNG sinon ; `Variants{SourceType string; SourceSize int64; Web, Email Encoded{Data []byte; Type string; Width, Height int}}`
- [X] T036 [US3] Habillage à l'envoi in `internal/mail` (research R6) : `render.go` : `Render` n'exécute plus `layout` et renvoie `Message{Subject, Text, HTML}` de contenu ; nouveau type `Identity{Name string; Palette brand.Palette; Logo *Attachment}` et `Wrap(msg Message, footer string, id Identity) Message` qui exécute `layout.txt.tmpl` et `layout.html.tmpl` avec le contenu déjà rendu (`template.HTML`), le nom, la palette (`.Palette.Brand.Hex` etc.) et `HasLogo` ; `templates/layout.html.tmpl` : bandeau d'en-tête en `brand` avec texte `onBrand`, logo `<img src="cid:logo" alt="" height="48">` avant le nom si `HasLogo`, liens en `brandInk`, définition d'un gabarit `button` (`{{define "button"}}` : fond `brand`, texte `onBrand`) ; remplacer le style codé en dur `background:#1f5278;color:#ffffff` des gabarits `invitation`, `password-reset`, `email-verification`, `membership-*`, `communication`, `test` par `{{template "button" …}}` ; `Message.Inline []Attachment` pour les images embarquées ; `outbox.go` : `Enqueue(ctx, q, to, msg, footer)` écrit `wrap = 1` et `footer`
- [X] T037 [US3] Worker et SMTP in `internal/mail/worker.go` et `internal/mail/smtp.go` : champ `Worker.Identity func(ctx) (Identity, error)` ; `Tick` lit l'identité une fois par passage (erreur → journal et identité par défaut, l'envoi continue) ; `deliver` applique `Wrap(msg, e.Footer.String, id)` et pose `msg.FromName = id.Name` quand `e.Wrap = 1` (le `from_name` stocké n'est lu que pour `wrap = 0`) ; `SMTPSender.Send` ajoute chaque `msg.Inline` par `m.EmbedReader("logo", …, gomail.WithFileContentType(…))` avec `Content-ID: logo`
- [X] T038 [US3] Adapter les appelants du mail in `internal/api/mail.go` (`enqueueMail`, `enqueueNotice` : `Footer` passé à `Enqueue`), `internal/api/communications.go` (envoi : `wrap = 1`, `footer` du pied de saison et `from_name` NULL dans `EnqueueCommunicationEmail` ; adapter `internal/api/communications_send_test.go` : l'expéditeur vient de l'identité à l'envoi ; `preview-communication` : `Wrap` avec l'identité courante, logo en `data:` URI à la place de `cid:logo` pour l'`iframe`) ; fonction `func (s *server) mailIdentity(ctx, q) (mail.Identity, error)` in `internal/api/identity.go` (nom, `brand.Derive` des couleurs, variante `email` lue par `FileStore.Read` avec taille et SHA-256) réutilisée par le worker
- [X] T039 [US3] Créer `internal/api/identity.go` (contracts/api.md, tag `association`) : `Palette` (six `Hex`), `Identity{Name, LogoURL *string, Palette}` ; opérations `get-identity` (`GET /identity`, `auth.Public`, `Cache-Control: no-cache`) et `get-identity-logo` (`GET /identity/logo/{name}`, `auth.Public`, `name` validé `^[0-9a-f]{32}$` et égal à `web_name` courant, sinon `404` ; octets lus par `FileStore.Read` ; en-têtes de contracts/api.md) ; `registerIdentity` appelée depuis `New` in `internal/api/api.go` ; `Deps.LogoFiles *communication.FileStore` ; `Association.logoUrl` de `get-session` rempli in `internal/api/me.go`
- [X] T040 [US3] Créer `internal/api/appearance.go` : `get-appearance`, `update-appearance` (`{primaryColor, accentColor}` `pattern:"^#[0-9a-fA-F]{6}$"` nullable, stockés en minuscules par `UpdateAssociationColors`), `preview-palette` (`GET /admin/settings/appearance/palette`, paramètres `primary`, `accent` facultatifs), `upload-logo` (`PUT /admin/settings/appearance/logo`, `huma.MultipartFormFiles` champ `file` requis, `MaxBodyBytes` = 1 048 576 + 16 384 ; `logo.Process` ; `ErrUnsupported` et `ErrTooBig` → `422` « Déposez une image PNG, JPEG ou WebP de 1 Mo au plus. », `ErrTooLarge` → « Cette image est trop grande : 8 000 pixels de côté au plus. » sur `form.file` ; écriture des deux variantes par `LogoFiles.Save`, puis `UpsertAssociationLogo` en `WriteTx` ; après commit, suppression des fichiers de l'ancien logo ; en cas d'échec de la transaction, suppression des nouveaux fichiers), `delete-logo` (`DELETE`, `204`, idempotent, fichiers supprimés après commit) ; tous `auth.Admin`, tag `settings` ; journaux research R9
- [X] T041 [US3] Câblage in `internal/cli/serve.go` : `FileStore` sur `FRIGG_DATA_DIR/logo/` passé à `api.Deps.LogoFiles` ; au démarrage, suppression des fichiers de `logo/` non référencés par `association_logo` (journal `warn` par fichier supprimé) ; `Worker.Identity` branchée sur la lecture de l'identité (même code que `mailIdentity`, exposé par une fonction de `internal/api` ou un petit adaptateur)
- [X] T042 [US3] Régénérer le contrat avec `go tool task gen` ; `web/app/api/openapi.json` et `web/app/api/schema.d.ts` commités
- [X] T043 [P] [US3] Créer `web/app/api/identity.ts` : `identityQuery` (`get-identity` ; `queryFn` qui renvoie l'identité par défaut de Frigg (nom « Frigg », palette des valeurs actuelles de `app.css`, pas de logo) en cas d'erreur réseau ou serveur, `staleTime` long) ; compléter `web/app/api/settings.ts` : `appearanceQuery`, `palettePreviewQuery(primary, accent)` (`placeholderData: keepPreviousData`), `useUpdateAppearance()`, `useUploadLogo()` (`bodySerializer` FormData), `useDeleteLogo()`, chacune invalidant `get-identity`, `get-session` et `get-appearance`
- [X] T044 [US3] Créer `web/app/shell/BrandStyle.tsx` (écrit `--brand`, `--brand-ink`, `--on-brand`, `--accent`, `--accent-ink`, `--on-accent` en triplets RVB sur `document.documentElement` depuis `identityQuery`) et l'intégrer in `web/app/root.tsx` : `clientLoader` qui appelle `queryClient.ensureQueryData(identityQuery)` avant le premier rendu, `HydrateFallback` neutre inchangé (aucun clignotement) ; ajouter `--accent-ink` (valeur par défaut) et `--color-accent-ink` in `web/app/app.css` ; test Vitest `web/app/shell/BrandStyle.test.tsx` : seules les six variables de marque sont écrites, `--ok`, `--warn`, `--danger`, `--info` restent inchangées (FR-022)
- [X] T045 [P] [US3] Écrans hors session `web/app/routes/auth/layout.tsx` : logo de l'association (`<img alt="">`, `object-contain`, hauteur 40 px) ou ses initiales, et son nom, depuis `identityQuery` ; identité Frigg si l'identité par défaut est renvoyée ; retirer le commentaire « Le nom de l'association n'est pas affiché hors connexion » (FR-028)
- [X] T046 [P] [US3] Barre latérale `web/app/shell/Sidebar.tsx` : logo (`association.logoUrl` de la session, `<img alt="">` `object-contain` dans la pastille de 36 px) à défaut initiales (FR-023)
- [X] T047 [US3] Migrer les usages de la couleur choisie in `web/app/**/*.tsx` (research R3 › Usages dans l'interface) : les 36 `text-brand` → `text-brand-ink`, `border-brand` et `border-brand/40` → `border-brand-ink` et `border-brand-ink/40`, `outline-brand` → `outline-brand-ink`, `accent-brand` → `accent-brand-ink` ; `bg-brand`, `bg-brand/90`, `bg-brand/10`, `bg-brand/5` inchangés ; mêmes règles pour `accent` ; vérifier avec la couleur `#fff3b0` (axe, US3) que menu actif, liens, cases cochées et bordures restent visibles
- [X] T048 [P] [US3] Composants in `web/app/settings/` : `ColorField.tsx` (libellé unique, `input type="color"` et champ texte hexadécimal synchronisés, erreur `422` reliée), `LogoField.tsx` (logo actuel ou initiales, champ fichier natif `accept="image/png,image/jpeg,image/webp"`, aide « PNG, JPEG ou WebP, 1 Mo au plus. », aperçu par URL d'objet libérée au démontage, bouton « Retirer le logo »), `AppearancePreview.tsx` (`aria-label="Aperçu"` ; en-tête de barre latérale avec logo, nom, tagline ; élément de menu actif ; bouton principal ; bouton d'accent ; lien ; bandeau d'e-mail miniature ; variables CSS de la palette d'aperçu appliquées **sur son conteneur seulement** ; région `role="status"` « Certains textes ont été assombris pour rester lisibles. » quand `brandInk ≠ brand` ou `accentInk ≠ accent`)
- [X] T049 [US3] Écran `web/app/routes/app/admin/settings-appearance.tsx` (remplace la version provisoire, contracts/ui-routes.md) : `clientLoader` (`appearanceQuery`) ; `SettingsPage section="Apparence"` ; blocs « Logo » et « Couleurs », aperçu à droite (collant) sur grand écran et dessous à 320 px ; palette d'aperçu demandée 150 ms après la dernière modification ; « Enregistrer » : dépôt du logo d'abord s'il a changé (arrêt et erreur sur le champ en cas de refus), puis couleurs ; un `413` du dépôt est affiché sur le champ du logo avec le message « Déposez une image PNG, JPEG ou WebP de 1 Mo au plus. » ; pas de dialogue au remplacement d'un logo (plan.md › Complexity Tracking) ; notification « Apparence enregistrée. » ; « Retirer le logo » et « Rétablir les couleurs par défaut » derrière `ConfirmDialog` (FR-017, FR-021) ; `useBlocker` en cas de modifications non enregistrées
- [X] T050 [US3] Scénario Playwright `web/e2e/appearance.spec.ts` : dépôt d'un PNG (aperçu puis barre latérale), refus d'un SVG avec logo conservé, couleur `#fff3b0` (aperçu ajusté, reste de l'interface inchangé avant l'enregistrement), enregistrement, déconnexion : écran de connexion avec logo, nom et couleurs de l'association, `GET /api/identity` limité à `name`, `logoUrl`, `palette` ; demande de réinitialisation de mot de passe et lecture Mailpit (`web/e2e/mailpit.ts`) : bandeau à la couleur, partie `inline` `cid:logo` ; retrait du logo et rétablissement des couleurs avec confirmation ; section « Apparence » et aperçu utilisables dans une fenêtre de 320 px sans défilement horizontal ; axe sans violation `serious`/`critical` sur « Apparence », l'accueil et l'écran de connexion avec la couleur claire

**Checkpoint**: US3 fonctionne au-dessus de la phase 2 et de US1 (barre latérale de T015, tagline de session de T011) ; SC-004, SC-005, SC-006, SC-008 vérifiables.

---

## Phase 6: User Story 4 - Réseaux sociaux et contact des membres (Priority: P3)

**Goal**: l'administrateur saisit les liens de six réseaux sociaux ; l'accueil membre affiche un encart « Contacter l'association » avec coordonnées et réseaux renseignés, absent si rien n'est renseigné.

**Independent Test**: renseigner adresse de contact, téléphone, Instagram et Facebook ; l'accueil membre montre l'encart avec `mailto:`, `tel:` et liens externes en `noopener noreferrer` ; tout vider : encart absent.

### Tests for User Story 4

- [X] T051 [P] [US4] Compléter `internal/association/settings_test.go` et `internal/api/general_settings_test.go` : liens `social.*` valides enregistrés ; `social.instagram = "instagram.com/x"` → `422` sur `body.social.instagram` ; lien vide → `null`
- [X] T052 [P] [US4] Tests d'API in `internal/api/identity_test.go` (ou `internal/api/association_contact_test.go`) : `get-association-contact` membre → adresse, contact, téléphone, site, réseaux, **sans** `rna`, `siret`, `foundedOn`, `purpose`, `shortName` (liste des clés vérifiée) ; anonyme → `401` (matrice `internal/api/access_test.go`)

### Implementation for User Story 4

- [X] T053 [US4] Étendre `Normalize` in `internal/association/settings.go` aux six liens `social.facebook`, `social.instagram`, `social.linkedin`, `social.youtube`, `social.mastodon`, `social.bluesky` (`ValidWebURL`, message « Saisissez une adresse web complète, commençant par https:// ») et vérifier leur écriture par `update-general-settings`
- [X] T054 [US4] Opération `get-association-contact` (`GET /association/contact`, `auth.Member`, tag `association`) in `internal/api/identity.go` : type `AssociationContact{address, contactEmail, phone, website, social}` (contracts/api.md) ; régénérer le contrat (`go tool task gen`)
- [X] T055 [P] [US4] Compléter `web/app/routes/app/admin/settings-general.tsx` : `fieldset` « Réseaux sociaux » avec six champs `type="url"` libellés « Lien Facebook », « Lien Instagram », « Lien LinkedIn », « Lien YouTube », « Lien Mastodon », « Lien Bluesky »
- [X] T056 [P] [US4] Créer `web/app/settings/ContactCard.tsx` et l'intégrer in `web/app/routes/app/member/home.tsx` (requête `get-association-contact` lue par le composant, sans `clientLoader` : un préchargement en parallèle du layout protégé déclencherait la redirection « session expirée » ; requête ajoutée in `web/app/api/identity.ts`) : titre « Contacter l'association », adresse du siège, `mailto:`, `tel:+33…` (affichage `formatPhone`), site et réseaux en liens `target="_blank" rel="noopener noreferrer"`, texte visible du réseau, icône générique `Share2` `aria-hidden`, nom accessible « {Réseau} de {nom} », mention « (nouvelle fenêtre) » en `sr-only` ; composant absent si aucune information (FR-025) ; invalidation de cette requête par `useUpdateGeneralSettings`
- [X] T057 [US4] Compléter `web/e2e/general-settings.spec.ts` (« contact ») : saisie de l'adresse de contact, du téléphone, d'Instagram et de Facebook, encart de l'accueil membre et attributs des liens, lien social invalide refusé, champs vidés → encart absent ; axe sur l'accueil membre

**Checkpoint**: toutes les stories fonctionnent ensemble.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T058 [P] Revue de sécurité ciblée (plan.md › points de vigilance 1 à 6) : aucun octet reçu écrit sur disque sans passer par `logo.Process` ; `get-identity-logo` ne sert que `web_name` ; aucune valeur de coordonnées dans les journaux (`grep` des appels `slog` des nouveaux fichiers) ; `Identity` limitée à trois clés ; `grep -rE "(text|border|outline|accent)-brand([^-]|$)" web/app` ne renvoie aucune classe hors `-brand-ink` ; corriger le cas échéant
- [X] T059 [P] Documentation utilisateur in `README.md` (ou documentation d'utilisation existante) : rubrique « Paramètres › Général » (champs, formats RNA, SIRET, téléphone) et « Apparence » (formats et taille du logo, ajustement automatique des couleurs, identité affichée sur l'écran de connexion et dans les e-mails), répertoire `logo/` inclus dans la sauvegarde du répertoire de données
- [X] T060 [P] Mettre à jour `specs/001-auth-app-shell/spec.md` d'une note renvoyant à la spec 010 pour l'identité affichée hors session (le nom et le logo de l'association remplacent ceux de Frigg)
- [X] T061 Exécuter `go tool task check` (build `CGO_ENABLED=0`, `go vet`, `go test ./...`, `sqlc generate` et `frigg openapi` sans diff, migrations sur base vierge, `bun run typecheck`, Biome, Vitest, `bun run build`, Playwright + axe) et dérouler quickstart.md › scénarios manuels 1 à 5

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance. T001 avant T002, T003 ; T003 avant T004.
- **Foundational (Phase 2)** : dépend de la phase 1. T005 avant T007 ; T006 et T008 en parallèle. Bloque toutes les stories.
- **US1 (Phase 3)** : dépend de la phase 2. MVP.
- **US2 (Phase 4)** : dépend de la phase 2 ; T022 et T024 complètent des fichiers de T005/T007 et T014 : à faire après US1.
- **US3 (Phase 5)** : dépend de la phase 2 ; indépendante de US1 et US2 côté API, domaine et mail ; T046 modifie la barre latérale (T015), T048 lit la tagline de la session (T011). T047 touche de nombreux écrans (dont la barre latérale) : après T044 et T046, hors parallélisme.
- **US4 (Phase 6)** : dépend de la phase 2 ; T055 complète l'écran de T014, T056 l'accueil de T016 et utilise `formatPhone` (T023).
- **Polish (Phase 7)** : après les stories voulues.

### User Story Dependencies

- **US1 (P1)** : aucune autre story.
- **US2 (P2)** : domaine indépendant (T018, T021) ; intégration à l'écran après US1.
- **US3 (P2)** : backend entièrement indépendant (T026–T042) ; écrans après US1 pour la barre latérale.
- **US4 (P3)** : backend indépendant ; écrans après US1 et `formatPhone` de US2.

### Within Each User Story

- Tests d'abord (ils doivent échouer), puis domaine, puis handlers, puis régénération du contrat, puis frontend, puis Playwright.

### Parallel Opportunities

- Phase 1 : T002 en parallèle de T003 après T001.
- Phase 2 : T005, T006, T008 en parallèle.
- US1 : T009, T010 en parallèle ; T013, T015, T016 en parallèle après T012.
- US2 : T018, T019, T020 en parallèle ; T021 et T023 en parallèle.
- US3 : T026–T031 en parallèle ; T034 et T035 en parallèle (après T033 pour T035) ; T043, T045, T046, T048 en parallèle après T042.
- US3 (domaine `brand`, `logo`, habillage des e-mails) peut avancer en parallèle de US1 et US2.

---

## Parallel Example: User Story 1

```text
# Tests :
T009 internal/api/general_settings_test.go
T010 internal/api/access_test.go

# Frontend après régénération du contrat (T012) :
T013 web/app/api/settings.ts
T015 web/app/shell/Sidebar.tsx
T016 web/app/routes/app/member/home.tsx
```

## Parallel Example: User Story 3

```text
# Tests :
T026 internal/brand/palette_test.go
T027 internal/association/logo/process_test.go
T028 internal/mail/render_test.go
T029 internal/mail/worker_test.go
T030 internal/api/appearance_test.go
T031 internal/api/identity_test.go

# Domaine :
T034 internal/brand/{color,palette}.go
T035 internal/association/logo/process.go

# Frontend après T042 :
T043 web/app/api/identity.ts
T045 web/app/routes/auth/layout.tsx
T046 web/app/shell/Sidebar.tsx
T048 web/app/settings/{ColorField,LogoField,AppearancePreview}.tsx
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 (migration, requêtes) puis phase 2 (normalisation, opérations, sections).
2. US1 : nom et tagline, barre latérale, accueil membre.
3. **STOP et valider** : tests d'API, Playwright ; le nom se corrige sans intervention technique.

### Incremental Delivery

1. US1 → nom et tagline.
2. US2 → identité légale et coordonnées vérifiées.
3. US3 → logo, couleurs, identité hors session et e-mails habillés à l'envoi.
4. US4 → réseaux sociaux et encart de contact.
5. Phase 7 → revue de sécurité, documentation, contrôles complets.

### Notes

- Chaque tâche se termine par ses tests verts et un commit (gitmoji, message
  court, sans co-auteur, après validation de l'utilisateur).
- Après chaque ajout d'opération : `frigg openapi` et `bun run gen:api` avant
  le frontend ; le contrôle « sans diff » de la constitution l'exige.
- US3 change le moment de l'habillage des e-mails (T036–T038) : vérifier que
  tous les tests existants de `internal/mail` et des e-mails d'API passent
  avant de poursuivre le frontend.
