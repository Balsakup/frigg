---

description: "Liste des tâches : gestion des comptes par les administrateurs"
---

# Tasks: Gestion des comptes par les administrateurs

**Input**: Design documents from `specs/002-admin-account-management/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (api.md, ui-routes.md), quickstart.md

**Tests**: inclus. La spécification ne les demande pas explicitement, mais la
constitution les impose : tests d'autorisation (principes II et III), tests
d'API sur la chaîne complète dès que les sessions sont en jeu, Playwright + axe
sur chaque parcours touché (Workflow).

**Organization**: une phase par user story, dans l'ordre de priorité de la spec
(US1 et US2 en P1, US3 et US4 en P2, US5 en P3).

**Conventions** (constitution 1.4.0) : identifiants, tables, champs JSON et
valeurs d'énumération en anglais ; commentaires, messages, textes d'interface
et chemins visibles du SPA en français. Le projet n'a jamais été déployé :
`00001_init.sql` est modifiée, pas de nouvelle migration.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: parallélisable (fichiers différents, aucune dépendance sur une tâche inachevée)
- **[Story]**: user story de rattachement (US1 à US5)

## Path Conventions

Un module Go à la racine (`internal/`) et le SPA dans `web/` (voir plan.md ›
Project Structure). Helpers de test existants : `internal/api/apitest`
(`New`, `Seed(role, verified)`, `LoggedIn(email)`, `Client().Do(...)`,
`FlushMail`, `LogBuffer`).

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: aucune dépendance ni outil nouveau (plan.md › Technical Context)

- [X] T001 Vérifier qu'aucune instance de Frigg n'a été déployée (condition de la constitution 1.4.0 pour modifier `internal/database/migrations/00001_init.sql`) et supprimer la base de développement locale désignée par `FRIGG_DATABASE_PATH` dans `.env` (goose considère `00001` comme déjà appliquée)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: schéma, état « désactivé », type `AdminAccount`, enregistrement des opérations, composants partagés et jeu de données e2e

**⚠️ CRITICAL**: aucune story ne peut commencer avant la fin de cette phase

### Backend

- [X] T002 Modifier `internal/database/migrations/00001_init.sql` : ajouter `disabled_at TEXT` (« NULL = compte actif ; date de désactivation sinon ») dans `CREATE TABLE accounts` après `last_login_at` ; après `accounts`, créer `CREATE VIRTUAL TABLE accounts_fts USING fts5(email, content='accounts', content_rowid='id', tokenize="trigram remove_diacritics 1")` et les triggers `accounts_fts_ai` (`AFTER INSERT ON accounts` : `INSERT INTO accounts_fts (rowid, email) VALUES (new.id, new.email)`), `accounts_fts_ad` (`AFTER DELETE ON accounts` : `INSERT INTO accounts_fts (accounts_fts, rowid, email) VALUES ('delete', old.id, old.email)`), `accounts_fts_au` (`AFTER UPDATE OF email ON accounts` : suppression de l'ancienne entrée puis insertion de la nouvelle), chacun entre `-- +goose StatementBegin` et `-- +goose StatementEnd`, avec un commentaire français citant le principe VI ; dans `Down`, supprimer les trois triggers et `accounts_fts` avant `accounts`
- [X] T003 [P] Écrire `internal/database/fts_test.go` (doit échouer avant T004), tests unitaires de `FTSTrigramQuery` uniquement : renvoie `("", false)` pour `""`, `"  "`, `"du"`, `"a b"` ; `("\"pont\"", true)` pour `" pont "` ; `("\"exemple\" \"dup\"", true)` pour `"exemple  dup"` ; écarte les mots de moins de 3 caractères (`"de pont"` → `"\"pont\""`) en comptant les caractères Unicode (`"hél"` est gardé) ; double les guillemets internes (`a"bc` → `"a""bc"`) ; laisse `OR`, `NEAR`, `*` à l'intérieur des guillemets
- [X] T004 Implémenter `internal/database/fts.go` : `func FTSTrigramQuery(input string) (query string, ok bool)` — découpage aux espaces Unicode (`strings.Fields`), mots de moins de 3 runes écartés, chaque mot entouré de `"` avec `"` internes doublés, mots joints par une espace (ET implicite FTS5) ; commentaire français renvoyant à research.md › R7 et à la constitution 1.4.0 (tokenizer trigram, champs d'identité courts)
- [X] T005 Ajouter dans `internal/database/queries/accounts.sql` : `ListAccounts :many` et `CountAccounts :one` (colonnes `id, email, role, email_verified_at, disabled_at, created_at, last_login_at` ; filtres `(CAST(sqlc.narg(query) AS TEXT) IS NULL OR a.id IN (SELECT rowid FROM accounts_fts WHERE accounts_fts MATCH CAST(sqlc.narg(query) AS TEXT)))`, `(CAST(sqlc.narg(role) AS TEXT) IS NULL OR a.role = CAST(sqlc.narg(role) AS TEXT))`, `(CAST(sqlc.narg(disabled) AS INTEGER) IS NULL OR (a.disabled_at IS NOT NULL) = CAST(sqlc.narg(disabled) AS INTEGER))` ; `ListAccounts` trié `ORDER BY a.email, a.id` avec `LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset)`) ; puis `go generate ./internal/database` et vérifier que les paramètres générés sont `sql.NullString` / `sql.NullInt64` (research.md › R8)
- [X] T005b Ajouter à `internal/database/fts_test.go` un test sur base migrée (`database.Open` sur un fichier de `t.TempDir()`) : insérer `jean.dupont@exemple.fr`, `hélène.lefèvre@asso.fr`, `marc.dupré@exemple.fr`, puis vérifier via `ListAccounts` + `FTSTrigramQuery` que `pont` → dupont, `n.dup` → dupont, `LEFÈ` → lefèvre, `dupre` → dupré, `exemple dup` → dupont et dupré ; modifier une adresse et en supprimer une, puis vérifier que l'index suit (triggers)
- [X] T006 Ajouter dans `internal/api/api.go`, dans `accessMiddleware`, le refus des comptes désactivés : après la lecture du compte, si `account.DisabledAt.Valid`, `auth.Logout` puis `401` « Votre session a expiré. » (même branche que l'époque périmée, research.md › R2) ; ajouter l'appel `s.registerAccounts(api)` après `s.registerInvitations(api)`
- [X] T007 Créer `internal/api/accounts.go` : constante `tagAccounts = "Comptes"`, messages `MsgAccountGone = "Ce compte n'existe plus."`, `MsgSelfAction = "Vous ne pouvez pas modifier votre propre compte depuis cette page."`, `MsgLastAdmin = "L'association doit conserver au moins un administrateur actif."`, `MsgAccountDisabled = "Ce compte est désactivé. Réactivez-le avant d'envoyer un lien."` ; type `AdminAccount` (`id`, `email`, `role` `enum:"member,admin"`, `status` `enum:"active,disabled"`, `emailVerified`, `createdAt`, `lastLoginAt *time.Time`, `disabledAt *time.Time`, `isSelf`) ; `newAdminAccount(a sqlc.Account, currentID int64) AdminAccount` ; `func (s *server) registerAccounts(api huma.API)` vide pour l'instant ; helper `logAccountAction(ctx, action string, actorID, targetID int64, attrs ...any)` qui écrit en `INFO` « action d'administration sur un compte » avec `action`, `actor_id`, `target_id` et jamais d'adresse électronique (research.md › R10)
- [X] T008 [P] Ajouter le limiteur `AdminReset: NewKeyedLimiter(30, time.Hour)` dans la struct `Limits` et `NewLimits()` de `internal/auth/ratelimit.go`, et l'inclure dans la boucle de purge (research.md › R9)

### Frontend

- [X] T009 [P] Créer `web/app/ui/Table.tsx` en extrayant le tableau des invitations de `web/app/routes/app/admin/accounts.tsx` : props `caption` (visible ou `sr-only`), `columns: { key, label, className? }[]`, `children` (lignes), en-têtes `scope="col"`, enveloppe `ScrollRegion` étiquetée par le caption (320 px sans défilement de page) ; remplacer le tableau des invitations par ce composant sans changement visuel
- [X] T010 [P] Créer `web/app/ui/Select.tsx` sur `@base-ui/react/select`, habillé des tokens (bordure `line`, focus visible, popup `surface`), props `label`, `value`, `onValueChange`, `items: { value, label }[]`, `name?` ; utilisable au clavier et étiqueté pour les lecteurs d'écran
- [X] T011 [P] Ajouter une prop facultative `tone?: "danger" | "primary"` (défaut `danger`) à `web/app/ui/ConfirmDialog.tsx` : icône et bouton de confirmation `primary` sans teinte d'alerte pour les actions non destructives (changement de rôle, envoi de lien) ; ajouter aussi `confirmDisabled?: boolean`

### Jeu de données e2e

- [X] T012 [P] Ajouter dans `internal/tools/e2eseed/main.go` les comptes de 002 : `admin-comptes@exemple.fr`, `admin-desactivation@exemple.fr`, `admin-role@exemple.fr`, `admin-suppression@exemple.fr`, `admin-lien@exemple.fr` (admins vérifiés, un par fichier e2e : limite de 5 connexions par adresse et tests en parallèle), `membre-a-desactiver@exemple.fr`, `membre-desactive@exemple.fr` (créé puis désactivé : `disabled_at` renseigné), `membre-a-promouvoir@exemple.fr`, `admin-a-retrograder@exemple.fr` (admin), `admin-a-supprimer@exemple.fr` (admin, avec une invitation en attente envoyée vers `invite-par-supprime@exemple.fr`), `membre-lien@exemple.fr`, `hélène.lefèvre@exemple.fr`, et 30 comptes `membre-liste-NN@exemple.fr` (NN = 01 à 30) pour la pagination ; mot de passe commun `Password`

**Checkpoint**: `go tool task gen` sans erreur, `go test ./...` vert (dont T003), comptes de 001 inchangés.

---

## Phase 3: User Story 1 - Consulter et retrouver les comptes (Priority: P1) 🎯 MVP

**Goal**: liste paginée, recherche par fragment (trigram), filtres rôle/état, fiche d'un compte

**Independent Test**: avec le jeu de données, ouvrir `/admin/comptes`, vérifier les colonnes et le total, rechercher `pont` / `LEFÈ`, combiner les filtres, paginer, ouvrir une fiche ; en membre, l'API répond 403

### Tests for User Story 1

- [X] T013 [P] [US1] Écrire `internal/api/accounts_list_test.go` (httptest via `apitest`) : `GET /api/admin/accounts` renvoie `items`, `total`, `page`, `pageSize` (défaut 25) triés par adresse ; `q=pont` filtre par sous-chaîne sans accents ; `q=du` → 422 avec `errors[].location = "query.q"` et le message « Saisissez au moins 3 caractères. » ; `q` vide → pas de filtre ; `role=admin` et `status=disabled` se combinent avec `q` ; `pageSize=101` et `page=0` → 422 ; page au-delà de la dernière → `items: []` et `total` correct ; `isSelf` vrai pour le compte connecté ; `GET /api/admin/accounts/{id}` → 200, id inconnu → 404 « Ce compte n'existe plus. » ; membre → 403 et anonyme → 401 sur les deux opérations
- [X] T013b [P] [US1] Écrire dans `internal/api/accounts_list_test.go` un test `TestListAccountsScale` : insérer 2 000 comptes en une transaction via le Store de `apitest` (adresses `prenom-NNNN.nom@exemple.fr`), puis vérifier que `GET /api/admin/accounts?q=nom-1234` et `?page=80` répondent chacun en moins de 200 ms et renvoient les bons résultats (SC-001) ; ignoré avec `testing.Short()`
- [X] T014 [P] [US1] Écrire `web/app/lib/account-filters.test.ts` (Vitest) : `parseFilters(URLSearchParams)` → `{ q, role, status, page }` avec `role=membre|admin` → `member|admin`, `statut=actif|desactive` → `active|disabled`, valeurs inconnues ignorées, `page` invalide → 1 ; `toApiQuery(filters)` n'inclut pas `q` s'il ne contient aucun mot de 3 caractères ; `toSearchParams(filters)` omet les valeurs par défaut ; `hasSearchableTerm("de pont") === true`, `hasSearchableTerm("du") === false`
- [X] T015 [P] [US1] Écrire `web/app/ui/Pagination.test.tsx` (Vitest + Testing Library) : « Page 2 sur 5 », liens Précédente/Suivante avec les bons `href`, `aria-current="page"`, Précédente absente en page 1, composant non rendu si une seule page, `nav` étiquetée « Pagination »

### Implementation for User Story 1

- [X] T016 [US1] Implémenter dans `internal/api/accounts.go` les opérations `list-accounts` (`GET /admin/accounts`, accès `auth.Admin`, entrée `q` `query:"q" maxLength:"100"`, `role` `enum:"member,admin"`, `status` `enum:"active,disabled"`, `page` `minimum:"1" default:"1"`, `pageSize` `minimum:"1" maximum:"100" default:"25"` ; `database.FTSTrigramQuery` : saisie non vide et `ok == false` → `FieldErrors("query.q", "Saisissez au moins 3 caractères.")` ; `ListAccounts` + `CountAccounts` sur `s.Store.Read` ; sortie `AdminAccountPage`) et `get-account` (`GET /admin/accounts/{id}`, 404 `MsgAccountGone`) ; régénérer avec `go tool task gen`
- [X] T017 [P] [US1] Créer `web/app/lib/account-filters.ts` (`parseFilters`, `toApiQuery`, `toSearchParams`, `hasSearchableTerm`) selon contracts/ui-routes.md › Routes, pour faire passer T014
- [X] T018 [P] [US1] Créer `web/app/ui/Pagination.tsx` (liens React Router, `nav aria-label="Pagination"`, « Page n sur m », Précédente/Suivante, `aria-current="page"`) pour faire passer T015
- [X] T019 [US1] Ajouter la section « Comptes » en tête de `web/app/routes/app/admin/accounts.tsx` : `clientLoader` qui précharge `list-accounts` depuis `parseFilters(url.searchParams)` via `queryClient.ensureQueryData` ; barre de critères (champ `type="search"` « Rechercher une adresse » avec indication « Saisissez au moins 3 caractères. » liée par `aria-describedby` tant que `hasSearchableTerm` est faux, mise à jour de l'URL après 300 ms avec `replace` ; `Select` Rôle et État ; bouton « Effacer les critères » si un critère est actif ; tout changement de filtre remet `page` à 1) ; `Table` caption « Comptes de l'association » avec colonnes Adresse (lien vers `/admin/comptes/:id`, « (vous) » si `isSelf`), Rôle (`Badge` via `roleLabel`), État (`Badge` `ok` « Actif » / `muted` « Désactivé »), Adresse confirmée (Oui/Non), Créé le, Dernière connexion (« Jamais » si nulle) via `formatDate` ; `Pagination` ; région `role="status"` annonçant « N comptes » ou « Aucun compte ne correspond à ces critères. » avec bouton « Effacer les critères » ; mettre à jour l'introduction de la page (« Gérez les comptes de l'association et invitez de nouvelles personnes. ») ; la section Invitations reste en dessous
- [X] T020 [US1] Créer `web/app/routes/app/admin/account-detail.tsx` et l'enregistrer dans `web/app/routes.ts` (`route("comptes/:id", "routes/app/admin/account-detail.tsx")` dans le layout admin) : `clientLoader` qui précharge `get-account` (404 → rendu de « Page introuvable » du shell) ; `handle.crumb` « Comptes » vers `/admin/comptes` puis l'adresse ; `Page` avec titre = adresse, introduction = badges rôle et état ; liste de définitions (rôle, état et « depuis le … » si désactivé, adresse confirmée, créé le, dernière connexion) ; si `isSelf`, note « Vous ne pouvez pas modifier votre propre compte ici. Utilisez “Mon compte”. » avec lien vers `/mon-compte` ; zone d'actions vide prête pour US2 à US5
- [X] T021 [US1] Écrire `web/e2e/admin-accounts-list.spec.ts` (Playwright + `axe.ts`, compte `admin-comptes@exemple.fr`) : tri par adresse ; total et pagination vérifiés avec `q=membre-liste` (30 comptes, stable malgré les autres fichiers e2e qui modifient des comptes en parallèle) : « 30 comptes », page 2 sur 2 avec `pageSize` par défaut ; `du` affiche l'indication et ne filtre pas ; `pont` puis `LEFÈ` filtrent ; filtres combinés puis « Effacer les critères » ; rechargement qui conserve les critères de l'URL ; ouverture d'une fiche et fil d'Ariane ; sa propre fiche affiche la note ; axe sans violation `serious`/`critical` sur la liste et la fiche, y compris en 320 px sans défilement horizontal de la page

**Checkpoint**: US1 utilisable seule — quickstart scénarios 1 à 3.

---

## Phase 4: User Story 2 - Désactiver et réactiver un compte (Priority: P1)

**Goal**: désactivation qui ferme les sessions et bloque la connexion et la réinitialisation ; réactivation

**Independent Test**: session membre ouverte dans un client ; un admin la désactive ; la requête suivante du membre → 401 ; reconnexion → message d'identifiants incorrects ; réactivation → connexion possible

### Tests for User Story 2

- [X] T022 [P] [US2] Écrire `internal/api/accounts_disable_test.go` (httptest via `apitest`) : `POST /api/admin/accounts/{id}/disable` → 200 `status: "disabled"`, `disabledAt` renseigné ; la session déjà ouverte du compte reçoit 401 « Votre session a expiré. » sur `GET /api/session` ; `POST /api/auth/login` avec le bon mot de passe → 401 « Adresse électronique ou mot de passe incorrect » ; `POST /api/auth/password-reset` → 202 et aucun e-mail en file ; un jeton de réinitialisation émis avant la désactivation → 410 sur `confirm-password-reset` ; désactiver un compte déjà désactivé → 200 sans nouvelle ligne de journal ; sa propre cible → 409 `MsgSelfAction` ; `enable` → 200 `status: "active"` et connexion possible avec l'ancien mot de passe, l'ancienne session restant fermée ; `POST /api/admin/invitations` sur l'adresse d'un compte désactivé → 409 « Un compte existe déjà avec cette adresse. » ; après réactivation d'un compte non confirmé, `emailVerified` reste `false` ; une ligne de journal `INFO` par action avec `action` (`account.disable`, `account.enable`), `actor_id`, `target_id` et sans `@` ; membre → 403, anonyme → 401
- [X] T023 [P] [US2] Écrire `internal/api/accounts_guard_test.go` : avec exactement deux administrateurs A et B, A désactive B puis B (session encore ouverte côté client) tente de désactiver A → 401 ou 403 et A reste actif ; test concurrent : deux goroutines lancent en parallèle « A désactive B » et « B désactive A » (sessions ouvertes avant), au plus une réussit, et `SELECT count(*) FROM accounts WHERE role = 'admin' AND disabled_at IS NULL` vaut au moins 1 ; un seul administrateur actif plus un administrateur désactivé : réactiver ce dernier fonctionne

### Implementation for User Story 2

- [X] T024 [US2] Ajouter dans `internal/database/queries/accounts.sql` : `DisableAccount :execrows` (`UPDATE accounts SET disabled_at = sqlc.arg(now), auth_epoch = auth_epoch + 1 WHERE accounts.id = sqlc.arg(id) AND accounts.disabled_at IS NULL AND (accounts.role <> 'admin' OR EXISTS (SELECT 1 FROM accounts o WHERE o.role = 'admin' AND o.disabled_at IS NULL AND o.id <> sqlc.arg(id)))`) et `EnableAccount :execrows` (`disabled_at = NULL` si non NULL) ; dans `internal/database/queries/tokens.sql`, ajouter `DeleteAllUnusedTokens :exec` (`account_id`, `used_at IS NULL`) et modifier `GetValidToken` pour joindre `accounts` et exiger `accounts.disabled_at IS NULL` (colonnes renvoyées inchangées : `one_time_tokens.*`) ; `go generate ./internal/database`
- [X] T025 [US2] Modifier `internal/api/auth.go` : dans `login`, après `auth.VerifyPassword` réussi, renvoyer le même `401 MsgBadCredentials` si `account.DisabledAt.Valid` ; dans `requestPasswordReset`, retourner `nil` sans jeton ni e-mail si le compte est désactivé (commentaire : même réponse qu'une adresse inconnue, principe VII)
- [X] T026 [US2] Ajouter dans `internal/api/accounts.go` le helper transactionnel `s.adminWrite(ctx, targetID, fn func(q, actor, target sqlc.Account) error)` : refuse `targetID == CurrentAccount(ctx).ID` avec `409 MsgSelfAction` (paramètre pour le désactiver sur `enable`) ; dans `s.Store.WriteTx`, relit l'acteur (`GetAccountByID`) et renvoie 401 s'il est désactivé ou absent, 403 si `!auth.Allows(actor.Role, auth.Admin)` ; relit la cible (404 `MsgAccountGone`) ; erreurs via `s.txError` ; puis les opérations `disable-account` (`POST /admin/accounts/{id}/disable` : déjà désactivé → renvoyer l'état sans écrire ; sinon `DisableAccount`, 0 ligne → `409 MsgLastAdmin` ; `DeleteAllUnusedTokens` ; `logAccountAction("account.disable")`) et `enable-account` (`POST /admin/accounts/{id}/enable`, sans garde-fou « soi-même » ; déjà actif → état inchangé ; `logAccountAction("account.enable")`) renvoyant `AdminAccount` relu ; `go tool task gen`
- [X] T027 [US2] Ajouter les actions « Désactiver » (compte actif : `ConfirmDialog` `tone="danger"`, texte « {adresse} sera déconnecté de tous ses appareils et ne pourra plus se connecter. Vous pourrez réactiver le compte. ») et « Réactiver » (compte désactivé : bouton direct, sans dialogue) dans `web/app/routes/app/admin/account-detail.tsx`, masquées si `isSelf` ; après succès : `Alert` `success` + région `role="status"`, invalidation de `get-account` et `list-accounts` ; `409`/`404` : `Alert` `danger` avec le `detail` du serveur, `404` rafraîchit la fiche
- [X] T028 [US2] Écrire `web/e2e/admin-accounts-disable.spec.ts` : contexte A connecté en `membre-a-desactiver@exemple.fr`, contexte B en `admin-desactivation@exemple.fr` désactive le compte via la fiche ; A navigue → page de connexion ; reconnexion de A → message d'identifiants incorrects ; B réactive → A se reconnecte ; vérifier le focus rendu après fermeture du dialogue et axe sur la fiche avec le dialogue ouvert

**Checkpoint**: US1 + US2 — quickstart scénarios 4, 5 et 10.

---

## Phase 5: User Story 3 - Changer le rôle d'un compte (Priority: P2)

**Goal**: promotion et rétrogradation, effectives à la requête suivante, sans déconnexion

**Independent Test**: promouvoir un membre et constater l'accès admin ; rétrograder un admin connecté et constater le retour à l'espace membre à sa navigation suivante

### Tests for User Story 3

- [X] T029 [P] [US3] Écrire `internal/api/accounts_role_test.go` : `PUT /api/admin/accounts/{id}/role` `{ "role": "admin" }` sur un membre → 200, sa session existante obtient 200 sur `GET /api/admin/accounts` ; rétrogradation d'un admin → sa session existante reçoit 403 sur les opérations admin et 200 sur `GET /api/session` avec `role: "member"` ; même rôle → 200 sans ligne de journal ; `role: "owner"` → 422 ; cible = soi-même → 409 `MsgSelfAction` ; garde-fou « dernier administrateur » testé au niveau de la requête (inatteignable par l'API seule, puisque l'acteur est lui-même un administrateur actif) : avec un seul admin actif X, `SetAccountRole(id: X, role: "member")` sur le Store de `apitest` → 0 ligne, et rétrograder un admin désactivé → 1 ligne ; journal `account.role_change` avec `role`

### Implementation for User Story 3

- [X] T030 [US3] Ajouter dans `internal/database/queries/accounts.sql` `SetAccountRole :execrows` (`UPDATE accounts SET role = sqlc.arg(role) WHERE accounts.id = sqlc.arg(id) AND accounts.role <> sqlc.arg(role) AND (sqlc.arg(role) = 'admin' OR accounts.disabled_at IS NOT NULL OR EXISTS (SELECT 1 FROM accounts o WHERE o.role = 'admin' AND o.disabled_at IS NULL AND o.id <> sqlc.arg(id)))`) ; `go generate ./internal/database`
- [X] T031 [US3] Implémenter dans `internal/api/accounts.go` l'opération `set-account-role` (`PUT /admin/accounts/{id}/role`, corps `{ role }` `enum:"member,admin"` requis) via `adminWrite` : même rôle → état inchangé sans écriture ; 0 ligne → `409 MsgLastAdmin` ; `logAccountAction("account.role_change", ..., "role", role)` ; `go tool task gen`
- [X] T032 [US3] Ajouter l'action « Changer le rôle » dans `web/app/routes/app/admin/account-detail.tsx` : `ConfirmDialog` `tone="primary"`, titre « Changer le rôle de {adresse} », `Select` Membre/Administrateur initialisé au rôle actuel, bouton « Enregistrer » (`confirmDisabled` si inchangé) ; masquée si `isSelf` ; mêmes retours et invalidations que T027
- [X] T033 [US3] Gérer le retrait des droits en cours de navigation : dans `web/app/api/client.ts`, middleware qui, sur une réponse `403` dont le chemin commence par `/api/admin/`, invalide `sessionQuery` ; dans `web/app/shell/AppShell.tsx`, mémoriser le rôle précédent (`useRef`) et, s'il passe de `admin` à `member`, `writePreference("last-space", "member")`, naviguer vers `/` et afficher l'`Alert` `info` « Vos droits d'administration ont été retirés. » (annoncée par `role="status"`)
- [X] T034 [US3] Écrire `web/e2e/admin-accounts-role.spec.ts` : B (`admin-role@exemple.fr`) promeut `membre-a-promouvoir@exemple.fr`, qui se connecte et voit le sélecteur d'espace ; A connecté en `admin-a-retrograder@exemple.fr` sur `/admin/comptes`, B le rétrograde, A clique sur un lien de l'espace administration → arrive sur `/` avec le message, sans sélecteur d'espace ; axe sur le dialogue de rôle

**Checkpoint**: quickstart scénarios 6 et 7.

---

## Phase 6: User Story 4 - Supprimer définitivement un compte (Priority: P2)

**Goal**: effacement RGPD du compte et de ses traces nominatives

**Independent Test**: supprimer `admin-a-supprimer@exemple.fr` ; il disparaît de la liste ; son invitation en attente affiche « Compte supprimé » ; son adresse peut être réinvitée ; la connexion échoue avec le message générique

### Tests for User Story 4

- [X] T035 [P] [US4] Écrire `internal/api/accounts_delete_test.go` : `DELETE /api/admin/accounts/{id}` → 204 ; `GET` → 404 ; sa session existante → 401 ; `one_time_tokens` du compte supprimés ; invitations qu'il a envoyées conservées avec `invitedBy: null` dans `list-invitations` ; invitations **adressées** à son adresse supprimées (tous états) ; `outbound_emails` dont `recipient` est son adresse supprimés ; `POST /api/admin/invitations` avec son adresse → 201 ; connexion → 401 `MsgBadCredentials` ; cible = soi-même → 409 ; dernier admin actif : avec un seul admin actif X, `DeleteAccount(X)` sur le Store → 0 ligne (même méthode que T029) ; ligne de journal `account.delete` sans l'adresse ; membre → 403

### Implementation for User Story 4

- [X] T036 [P] [US4] Ajouter `DeleteAccount :execrows` dans `internal/database/queries/accounts.sql` (`DELETE FROM accounts WHERE accounts.id = sqlc.arg(id) AND (accounts.role <> 'admin' OR accounts.disabled_at IS NOT NULL OR EXISTS (SELECT 1 FROM accounts o WHERE o.role = 'admin' AND o.disabled_at IS NULL AND o.id <> sqlc.arg(id)))`), `DeleteInvitationsByEmail :exec` dans `internal/database/queries/invitations.sql` et `DeleteEmailsByRecipient :exec` dans `internal/database/queries/emails.sql` ; `go generate ./internal/database`
- [X] T037 [US4] Implémenter `delete-account` (`DELETE /admin/accounts/{id}`, `DefaultStatus: 204`) dans `internal/api/accounts.go` via `adminWrite` : `DeleteAccount` (0 ligne → `409 MsgLastAdmin`), puis `DeleteInvitationsByEmail` et `DeleteEmailsByRecipient` avec l'adresse de la cible, dans la même transaction ; `logAccountAction("account.delete")` ; `go tool task gen`
- [X] T038 [US4] Ajouter l'action « Supprimer » dans `web/app/routes/app/admin/account-detail.tsx` : `ConfirmDialog` `tone="danger"`, titre « Supprimer définitivement {adresse} ? », texte « Cette action est irréversible. » et, si le compte est actif, un bouton « Désactiver plutôt » qui ferme ce dialogue et ouvre celui de désactivation ; masquée si `isSelf` ; après succès, invalider `list-accounts` et `list-invitations`, puis naviguer vers `/admin/comptes` avec le message « Le compte {adresse} a été supprimé. » (état de navigation affiché par la liste dans une `Alert` `success` + `role="status"`)
- [X] T039 [P] [US4] Afficher « Compte supprimé » quand `invitedBy` est `null` dans la colonne de l'auteur des invitations de `web/app/routes/app/admin/accounts.tsx`
- [X] T040 [US4] Écrire `web/e2e/admin-accounts-delete.spec.ts` : connecté en `admin-suppression@exemple.fr`, supprimer `admin-a-supprimer@exemple.fr` depuis sa fiche ; retour à la liste avec le message ; invitation de `invite-par-supprime@exemple.fr` affichée avec « Compte supprimé » ; réinviter `admin-a-supprimer@exemple.fr` réussit ; axe sur le dialogue de suppression

**Checkpoint**: quickstart scénario 8.

---

## Phase 7: User Story 5 - Envoyer un lien de réinitialisation à une personne (Priority: P3)

**Goal**: un admin déclenche l'e-mail de réinitialisation standard pour un compte actif

**Independent Test**: depuis la fiche de `membre-lien@exemple.fr`, envoyer le lien, le recevoir dans Mailpit, définir un nouveau mot de passe, se connecter

### Tests for User Story 5

- [X] T041 [P] [US5] Écrire `internal/api/accounts_reset_test.go` : `POST /api/admin/accounts/{id}/password-reset` → 202 et un e-mail « Réinitialisation de votre mot de passe » en file vers l'adresse du compte, dont le jeton fonctionne avec `confirm-password-reset` (puis 410 à la seconde utilisation) ; un second envoi invalide le premier jeton ; compte désactivé → 409 `MsgAccountDisabled` ; cible = soi-même → 409 `MsgSelfAction` ; 31ᵉ envoi dans l'heure par le même admin → 429 ; id inconnu → 404 ; journal `account.password_reset_sent` ; membre → 403

### Implementation for User Story 5

- [X] T042 [US5] Implémenter `send-account-password-reset` (`POST /admin/accounts/{id}/password-reset`, `DefaultStatus: 202`) dans `internal/api/accounts.go` : `s.Limits.AdminReset.Allow(adminID)` sinon `TooManyAttempts()` ; dans `s.Store.WriteTx`, relire la cible (404), refuser si `DisabledAt.Valid` (`409 MsgAccountDisabled`), `issueToken(ctx, q, id, purposePasswordReset, passwordResetLifetime)` puis `s.enqueueMail(ctx, q, "password-reset", email, "/reinitialiser-mot-de-passe", raw)` ; `logAccountAction("account.password_reset_sent")` ; cible = soi-même → `409 MsgSelfAction` (son propre mot de passe se gère depuis « Mon compte » ou « Mot de passe oublié ») ; `go tool task gen`
- [X] T043 [US5] Ajouter l'action « Envoyer un lien de réinitialisation » (compte actif uniquement, masquée si `isSelf`) dans `web/app/routes/app/admin/account-detail.tsx` : `ConfirmDialog` `tone="primary"`, texte « Un e-mail de réinitialisation sera envoyé à {adresse}. », bouton « Envoyer » ; succès → « Lien de réinitialisation envoyé à {adresse}. » ; `429` → `Alert` `danger` avec le `detail`
- [X] T044 [US5] Écrire `web/e2e/admin-accounts-reset.spec.ts` (helper `mailpit.ts`) : connecté en `admin-lien@exemple.fr`, envoyer le lien à `membre-lien@exemple.fr`, ouvrir le lien reçu dans un contexte neuf, définir un nouveau mot de passe, se connecter ; action absente sur la fiche de `membre-desactive@exemple.fr`

**Checkpoint**: quickstart scénario 9.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T045 [P] Vérifier que `internal/api/access_test.go` (matrice dérivée de l'OpenAPI) couvre les 7 nouvelles opérations sans modification, et qu'aucune n'est publique ; compléter uniquement si une opération échappe à la matrice
- [X] T046 [P] Mettre à jour le commentaire d'en-tête de `internal/api/apitest/apitest.go` (« Constitution 1.3.2 » → « Constitution 1.4.0 »)
- [X] T047 [P] Comparer `specs/002-admin-account-management/contracts/api.md` au schéma OpenAPI généré `web/app/api/openapi.json` (chemins, méthodes, paramètres, statuts) et corriger le contrat en cas d'écart ; ne pas modifier les documents de 001
- [X] T048 Lancer `go tool task check` (build `CGO_ENABLED=0`, vet, tests Go, `sqlc generate` et `gen:api` sans diff, typecheck, Biome, Vitest, build, Playwright + axe) et corriger jusqu'au vert
- [X] T049 Dérouler manuellement les scénarios 1 à 12 de `specs/002-admin-account-management/quickstart.md` (dont 11 : journaux sans adresse, et 12 : clavier seul, 320 px, lecteur d'écran)
- [X] T050 Retirer le rapport d'impact (commentaire HTML en tête) de `.specify/memory/constitution.md` avant le commit de l'amendement 1.4.0

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** : aucune dépendance
- **Foundational (Phase 2)** : dépend de la phase 1 ; bloque toutes les stories
- **US1 (Phase 3)** : dépend de la phase 2
- **US2 à US5 (Phases 4 à 7)** : dépendent de la phase 2 pour l'API ; leurs actions d'interface s'ajoutent à la fiche créée par US1 (T020) ; US3, US4 et US5 réutilisent le helper `adminWrite` introduit en US2 (T026)
- **Polish (Phase 8)** : après les stories voulues

### User Story Dependencies

```text
Setup → Foundational ── US1 ── US2 ─┬─ US3
                                    ├─ US4
                                    └─ US5
```

US3, US4 et US5 sont indépendantes entre elles une fois `adminWrite` (T026)
disponible ; elles touchent toutes `accounts.go` et `account-detail.tsx`, donc
à enchaîner si un seul développeur, ou à fusionner avec soin en parallèle.

### Within Each User Story

- Tests rédigés d'abord ; ils doivent échouer avant l'implémentation
- Requêtes SQL et `go generate` avant les opérations d'API
- Opérations d'API, puis `go tool task gen` (OpenAPI + types TypeScript), puis écrans
- Test e2e + axe en dernier

### Parallel Opportunities

- Phase 2 : T003, T008, T009, T010, T011, T012 en parallèle ; T004 après T003 ; T005 après T002 ; T005b après T004 et T005 ; T006 et T007 après T005
- US1 : T013, T013b, T014, T015 en parallèle ; T017 et T018 en parallèle de T016
- US4 : T035, T036 et T039 en parallèle
- Polish : T045, T046, T047 en parallèle

---

## Parallel Example: Phase 2

```bash
Task: "T003 internal/database/fts_test.go"
Task: "T008 internal/auth/ratelimit.go (AdminReset)"
Task: "T009 web/app/ui/Table.tsx"
Task: "T010 web/app/ui/Select.tsx"
Task: "T011 web/app/ui/ConfirmDialog.tsx (tone)"
Task: "T012 internal/tools/e2eseed/main.go"
```

## Parallel Example: User Story 1

```bash
# Tests en parallèle :
Task: "T013 + T013b internal/api/accounts_list_test.go"
Task: "T014 web/app/lib/account-filters.test.ts"
Task: "T015 web/app/ui/Pagination.test.tsx"

# Puis T016 (API) en parallèle de T017 et T018, puis T019 → T020 → T021
```

---

## Implementation Strategy

### MVP First

1. Phases 1 et 2 (schéma, recherche trigram, composants)
2. Phase 3 (US1) : les administrateurs voient et retrouvent les comptes
3. **STOP and VALIDATE** : quickstart scénarios 1 à 3, `go tool task check`

### Incremental Delivery

1. Socle + US1 → visibilité sur les comptes
2. US2 → mesure de sécurité principale (départs, comptes compromis)
3. US3 → changements de mandat du bureau
4. US4 → droit à l'effacement
5. US5 → aide aux bénévoles qui n'arrivent plus à se connecter
