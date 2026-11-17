# Implementation Plan: Parcours publics (adhésion et inscription sans compte)

**Branch**: `016-public-journeys` | **Date**: 2026-11-17 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/016-public-journeys/spec.md`

## Summary

Une personne extérieure peut demander une adhésion depuis `/adherer`, ou
s'inscrire à un événement « Ouvert au public » depuis son lien public, sans
compte. Le parcours se déroule ainsi :

1. Son **dépôt** reste invisible tant que son adresse n'est pas confirmée.
2. La **confirmation** crée en une transaction la fiche, puis la demande
   (006) ou l'inscription (012). Elle rattache d'office la fiche à un compte
   existant de même adresse.
3. Un **lien personnel**, envoyé par e-mail, ouvre ensuite le suivi et le
   paiement sans compte, carte HelloAsso comprise. Il vaut pour toute fiche
   sans compte de cette adresse, y compris les fiches importées.
4. La personne peut créer un compte à tout moment.
5. L'administration repère l'origine « Publique », rattache un doublon à une
   fiche existante et supprime les fiches publiques inactives.

Approche technique :

- **Accès par lien** : un quatrième niveau d'accès, `link`, hors hiérarchie
  des rôles. Le jeton passe par le fragment d'URL, puis il est échangé contre
  une clé dans la session scs existante. Son périmètre est recalculé à chaque
  requête (R1).
- **Base de données** : une migration `00014` sans reconstruction, qui ajoute
  deux tables (`public_submissions`, `tracking_links`) et des colonnes (R3 à
  R6).
- **« Ouvert au public »** : il est stocké comme `all_members` plus un
  `public_slug`, pour éviter de reconstruire `events` (R5).
- **Paiement** : le circuit de carte de 008 et 012 est réutilisé tel quel. Le
  payeur devient un paramètre (R11).
- **Notifications** : un destinataire unique, `recipientOf`, remplace les
  règles divergentes de 006 et 012 (R8).
- **Domaine** : un nouveau paquet pur, `internal/public`, porte la validation
  des dépôts et la décision de confirmation (R7).

Détails : [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 6.0 + React 19 (frontend)

**Primary Dependencies**: aucune nouvelle.

- **Backend** : huma/v2, scs/v2 (clé de session `tracking_link_id`), sqlc,
  goose, `golang.org/x/time/rate` (`KeyedLimiter`), `internal/auth`
  (`NewToken`, `NormalizeEmail`, `ValidatePassword`, `Login`),
  `internal/member` (`Validate`, `IdentityKey`), `internal/customfield`
  (`Apply`, audience `Owner`), `internal/membership` (domaine de paiement),
  `internal/event`, `internal/helloasso`, `internal/mail`.
- **Frontend** : base-ui, react-hook-form, TanStack Query, `openapi-fetch`,
  et les composants `payments/` extraits en 012.

**Storage**: SQLite. La migration `00014_public_journeys.sql` est
transactionnelle et n'utilise que `ADD COLUMN`, `CREATE TABLE` et
`CREATE INDEX` ([data-model.md](./data-model.md)).

**Testing**:

- **`go test`** :
  - tests en table de `internal/public` et des extensions
    d'`internal/event` et `internal/auth` ;
  - test de migration sur une base `00013` peuplée, avec la sonde
    `ADD COLUMN … CHECK` multi-colonnes ;
  - tests d'API httptest : dépôts, confirmation, concurrence, lien,
    isolation, non-révélation, limites, carte par lien avec `hatest`,
    création de compte, rattachement, inactives ;
  - matrice d'accès étendue au niveau `link`.
- **Vitest** : lecture et effacement du fragment, formulaire public.
- **Playwright + axe** : `public-journeys.spec.ts`, avec Mailpit
  ([R17](./research.md#r17-tests)).

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` ;
navigateurs récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**:

- Les SC-001 à SC-003 (5 et 3 minutes) dépendent surtout de la personne. Les
  pages publiques ne font qu'une ou deux requêtes indexées.
- 50 confirmations simultanées sont sérialisées par l'écrivain unique en
  `BEGIN IMMEDIATE` (SC-006).
- Le périmètre d'un lien est servi par l'index partiel
  `members_tracked_email`.

**Constraints**:

- Seules les 7 opérations publiques autorisées par le principe II v1.9.0.
- Le jeton n'apparaît jamais dans un chemin ou une requête HTTP envoyés au
  serveur.
- Les réponses publiques sont identiques quelle que soit l'adresse.
- Aucune donnée personnelle dans les journaux.
- Ni captcha ni service tiers.
- La migration est append-only, et les commentaires sqlc sont en ASCII.
- Les e-mails sont insérés dans la transaction de l'action.

**Scale/Scope**:

- Une association, quelques centaines de dépôts par saison.
- 24 opérations ajoutées (7 `public`, 13 `link`, 4 `admin`) et 16
  modifiées.
- 8 écrans publics et 8 écrans d'administration modifiés.
- 3 nouveaux modèles d'e-mail.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution v1.9.0 (amendée le 2026-11-17 pour cette fonctionnalité).

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Dépôts, liens et colonnes ajoutées sont des données de l'instance, sans locataire | ✅ |
| II. Refus par défaut, exceptions énumérées | Les 7 opérations publiques correspondent une à une à la liste du principe II : page d'adhésion, page d'événement par slug non devinable, deux dépôts, confirmation, demande de lien, ouverture d'un lien. La création de compte est une opération `link`, donc postérieure à une adresse confirmée. Le contenu public se limite aux champs autorisés (`PublicMembershipPage`, `PublicEvent`). Aucun listing d'événements. Pas de corps révélateur : `202` neutre, `404` uniforme. `noindex` existant. Liste centralisée dans `allowedPublic` | ✅ |
| III. Rôles et niveaux d'accès | Chaque opération déclare `public`, `link`, `member` ou `admin` via `register`. `link` est hors hiérarchie (`Allows` toujours faux) et accordé seulement par `accessMiddleware`. Aucun contrôle dans les handlers : périmètre par requêtes `…OfTrackedEmail` | ✅ |
| IV. Un shell, un design system | Layout public calqué sur les écrans hors session ; `ui/*`, tokens, base-ui ; composants `payments/` réutilisés ; `ConfirmDialog` pour annuler, régénérer, rattacher et supprimer en lot ; 320 px ; piège invisible au lecteur d'écran | ✅ |
| V. Binaire unique en Go pur | Aucune dépendance ; migration embarquée ; balayeur dans le processus ; HelloAsso facultatif (carte masquée sans configuration) | ✅ |
| VI. API typée et SQL généré | Types huma (bornes, enums `openness`, `outcome`) ; requêtes sqlc statiques (inactives, périmètre) ; JSON des champs produit et relu par `internal/customfield`, sans règle dupliquée ; types frontend régénérés | ✅ |
| VII. Sécurité et RGPD | Jetons `NewToken` hachés ; confirmation à usage unique (48 h) ; lien réutilisable (30 j), révoqué à la création du compte ; 9 limiteurs, chaque parcours limité par IP et par adresse visée, avec un plafond d'e-mails par adresse ; non-révélation testée octet pour octet ; piège sans captcha ; consentement daté ; dépôts purgés à 7 jours ; journaux sans adresse ni jeton ; `CrossOriginProtection` sur tous les `POST` publics | ✅ |
| Contraintes techniques | `WriteTx` (`BEGIN IMMEDIATE`) pour la confirmation, le rattachement et la création de compte ; e-mails dans la même transaction | ✅ |
| Conventions de langue | Identifiants anglais (`public_submissions`, `tracking_links`, `via_public`) ; textes, erreurs, journaux et e-mails en français | ✅ |
| Workflow | Nouvelle migration `00014` (append-only) ; matrice d'accès étendue (anonyme, membre, admin sans lien → 401 sur `link`) ; contrôles de [quickstart.md](./quickstart.md) | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance pour
l'implémentation :

1. **Jeton hors URL serveur** : le SPA lit `#jeton=` puis l'efface avec
   `history.replaceState` avant tout autre appel. Un test Vitest vérifie
   l'effacement. Un test de journal d'accès vérifie qu'aucune requête ne
   porte le jeton.
2. **Session partagée** : ouvrir un lien ne touche pas aux clés de compte. La
   création d'un compte retire `tracking_link_id` avant `auth.Login`.
   `close-tracking` ne détruit pas une session de compte.
3. **Périmètre réévalué** : toutes les requêtes `link` passent par
   `…OfTrackedEmail` (`account_id IS NULL AND email = ? COLLATE NOCASE`).
   Les tests couvrent une fiche rattachée et une adresse modifiée entre deux
   requêtes.
4. **Non-révélation** : le chemin « adresse de compte » du dépôt ne fait
   aucune lecture conditionnelle observable. L'e-mail est mis en file dans
   tous les cas, par le même code, avec une variante de modèle. Le test
   compare les réponses brutes.
5. **Destinataire unique** : `recipientOf` remplace `noticeRecipient` et
   `contactEmail`. Les tests existants de 006 et 012 restent verts. Une fiche
   sans compte avec adresse reçoit désormais les e-mails d'adhésion (changement
   de comportement documenté dans la spec, Assumptions).
6. **Jeton dans `outbound_emails`** : même régime que les invitations
   ([R8](./research.md)). La purge existante des e-mails envoyés n'est pas
   allongée.
7. **Sonde de migration** : vérifier `ADD COLUMN` avec un `CHECK` qui
   référence une autre colonne (modernc, sqlc). À défaut, l'invariant est
   garanti par le domaine et testé, et la sonde est documentée dans
   Complexity Tracking.

Vérification (T078) : les sept points sont confirmés dans le code et couverts
par des tests (`useFragmentToken.test.ts`, `public_logs_test.go`,
`tracking_links_test.go`, `tracking_isolation_test.go`,
`public_nonrevelation_test.go`, `recipients_test.go`, `db_test.go`).

## Project Structure

### Documentation (this feature)

```text
specs/016-public-journeys/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── api.md
│   └── ui-routes.md
├── checklists/
│   └── requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
internal/
├── public/                                 # nouveau : domaine pur, sans base
│   ├── submission.go                       # Validate*, Honeypot, consentement
│   ├── confirm.go                          # ConfirmDecision → Outcome
│   ├── inactive.go                         # Inactive, InactiveAfter
│   ├── limits.go                           # durées (ConfirmTTL, LinkTTL…)
│   └── *_test.go
├── auth/
│   ├── access.go                           # Access.Link, Allows
│   └── ratelimit.go                        # 9 limiteurs (R9)
├── event/event.go                          # Openness.Public, OpennessOf, PublicVisible
├── database/
│   ├── migrations/00014_public_journeys.sql
│   ├── queries/public_submissions.sql      # nouveau (ASCII)
│   ├── queries/tracking_links.sql          # nouveau
│   ├── queries/members.sql                 # …OfTrackedEmail, inactives, origine
│   ├── queries/memberships.sql             # via_public, rattachement, …OfTrackedEmail
│   ├── queries/event_registrations.sql     # via_public, …OfTrackedEmail
│   ├── queries/events.sql                  # public_slug
│   ├── queries/online_payments.sql         # via_link
│   ├── queries/association.sql             # public_membership_enabled
│   └── sqlc/                               # régénéré
├── mail/templates/                         # public-confirm, tracking-link, public-registration-confirmed
└── api/
    ├── api.go                              # accessMiddleware : niveau link ; registerPublic, registerTracking
    ├── public.go                           # 7 opérations publiques
    ├── public_confirm.go                   # confirmation transactionnelle (R7)
    ├── tracking.go                         # suivi, annulations, annonces, compte, close
    ├── tracking_payments.go                # carte par lien (R11)
    ├── tracking_links.go                   # émission, ouverture, révocation
    ├── public_sweeper.go                   # purge des dépôts et des liens
    ├── recipients.go                       # recipientOf (R8)
    ├── card_payments.go                    # openCheckout(payer)
    ├── memberships*.go, registrations*.go  # via_public, attach, filtres, notifications
    ├── members_admin.go                    # origine, inactives, send-tracking-link
    ├── events_admin.go                     # openness public, regenerate-event-public-link
    ├── general_settings.go                 # publicMembershipEnabled
    └── *_test.go                           # access_test (link), tracking_isolation_test, public_*_test

internal/cli/serve.go                       # démarrage du PublicSweeper avec les autres balayeurs
internal/api/apitest/apitest.go             # PublicSweeper exposé aux tests

web/
├── app/
│   ├── routes.ts                           # layout public + 8 routes
│   ├── routes/public/                      # layout, membership, event, confirm, tracking*, tracking-account
│   ├── public/                             # PublicIdentityForm, Honeypot, ConsentField, useFragmentToken
│   ├── payments/                           # client « par lien » injecté dans PaymentPanel
│   ├── routes/app/admin/                   # settings-general, event-*, members, member-detail, membership-detail, home
│   ├── api/public.ts, api/tracking.ts
│   └── api/openapi.json, api/schema.d.ts   # régénérés
└── e2e/public-journeys.spec.ts             # nouveau

internal/tools/e2eseed                      # fiche importée avec adresse, événement public
```

**Structure Decision** : on garde la structure existante (API Go + SPA dans
`web/`).

- **Domaine** : un nouveau paquet pur, `internal/public`, sur le modèle
  d'`internal/event`.
- **API** : les opérations publiques et par lien sont dans des fichiers
  dédiés. Elles appellent les fonctions internes déjà utilisées par l'espace
  membre (paiement, annulation). Seules la résolution du périmètre et celle
  du payeur changent.
- **Frontend** : un layout public et un dossier `public/` pour les
  composants des formulaires sans session.

## Complexity Tracking

Aucune violation de la constitution v1.9.0. Écarts de complexité retenus :

| Élément | Pourquoi | Alternative plus simple écartée |
| --- | --- | --- |
| Niveau d'accès `link` hors hiérarchie | Le lien donne un accès borné à des fiches, sans être un rôle (principe III) ; il doit rester vérifié au même endroit que les rôles | Vérifier le jeton dans chaque handler : contrôle dispersé, interdit par le principe III |
| Jeton en fragment, échangé contre une clé de session | Le jeton ne doit apparaître dans aucun journal ni en-tête `Referer`, et les retours de HelloAsso doivent reprendre l'accès sans nouveau jeton | Jeton dans l'URL de chaque appel : fuite dans les journaux et `Referer` |
| Table `public_submissions` séparée | Un dépôt non confirmé ne doit exister dans aucune requête métier (FR-008) | Fiche et adhésion « non confirmées » : filtre à ajouter partout, risque de fuite et de place réservée |
| `public_slug` à la place d'une valeur `public` dans le `CHECK` d'`openness` | Évite de reconstruire `events`, référencée par `event_registrations`, en migration `NO TRANSACTION` | Reconstruction de `events` (modèle 012 › R2) : sûre mais lourde, pour une valeur d'enum |
| Destinataire unique `recipientOf` | FR-035 et FR-040 imposent la même règle aux adhésions et aux inscriptions ; 006 et 012 divergeaient | Ajouter un cas « lien » à chacune des deux fonctions : deux règles à maintenir en parallèle |
