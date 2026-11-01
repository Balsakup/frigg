# Implementation Plan: Adhésions

**Branch**: `006-memberships` | **Date**: 2026-11-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/006-memberships/spec.md`

## Summary

Introduire l'adhésion d'une fiche membre (005) à une saison (003) à un tarif
(004). L'espace membre gagne « Adhésions » : demande (saison, membre, tarif,
type déclaré, montant affiché), suivi par saison, annulation d'une demande en
attente, puis paiement hors ligne : choix d'un moyen (espèces, chèque,
virement) et affichage de ses informations avec la référence de paiement.
L'administration gagne « Adhésions » (liste filtrée, chiffres de la saison,
validation avec type et tarif retenus et montant figé, refus motivé,
création directe, correction, annulation, constatation du paiement) et
« Paramètres » (moyens de paiement). Une table `memberships` porte
l'adhésion et son paiement, avec un index unique partiel « une adhésion en
cours par fiche et saison » ; une table singleton `payment_settings` porte
les moyens. Les règles d'état vivent dans un nouveau domaine pur
`internal/membership`. Les décisions sont envoyées par e-mail via la file
existante. Approche détaillée dans [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 5.9 + React 19 (frontend)

**Primary Dependencies**: aucune nouvelle. Backend : huma/v2, sqlc, goose,
`internal/textfold`, `internal/mail` (file `outbound_emails`). Frontend :
base-ui (`RadioGroup`, `Switch`, `Combobox`, `Dialog`), react-hook-form,
TanStack Query, lucide-react (`CreditCard`, `Settings`, `Copy`, `Check`).

**Storage**: SQLite ; nouvelle migration `00008_memberships.sql` (tables
`memberships` et `payment_settings`, table FTS `memberships_fts` et ses
triggers, index), voir
[data-model.md](./data-model.md)

**Testing**: `go test` : tests en table de `internal/membership`
(transitions, actions, montant, aide, références, IBAN, chiffres) ; tests
d'API (humatest / httptest) des deux espaces, matrice de cloisonnement
(`memberships_isolation_test.go`), concurrence sur l'index unique, e-mails mis
en file, horloge injectée ; matrice d'autorisation (automatique via
l'OpenAPI) ; Vitest (formatage) ; Playwright + axe (`memberships.spec.ts`,
`admin-memberships.spec.ts`, `admin-settings.spec.ts`)

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` ;
navigateurs récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: SC-001/SC-002 (demande < 2 min, décision < 30 s) :
une requête par écran, liste et chiffres calculés en Go sur les adhésions
d'une saison (quelques milliers au plus,
[R10](./research.md#r10-liste-recherche-et-chiffres-administration))

**Constraints**: refus par défaut (principe II) ; cloisonnement par compte
dans les requêtes SQL ; écritures sérialisées et transitions relues dans la
transaction ; date du jour à Paris via `s.today()` ; e-mails mis en file dans
la transaction ; ni données personnelles, ni motifs, ni coordonnées bancaires
dans les journaux ; RGAA ; commentaires des requêtes sqlc en ASCII

**Scale/Scope**: 7 écrans (3 membre, 4 administration) + 2 fiches et
2 dialogues modifiés + tableau de bord ; 19 opérations d'API (6 `member`,
13 `admin`) + 6 opérations modifiées ; 2 tables ; 1 package de domaine ;
4 gabarits d'e-mail

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Paramètres de paiement en table singleton (`id = 1`) ; aucune notion de locataire | ✅ |
| II. Authentification partout | 6 opérations `member`, 13 `admin` ; aucune route publique : le lien de l'e-mail mène à une page protégée et la connexion ramène sur l'URL (`suite`) | ✅ |
| III. Deux rôles | Niveau déclaré par `register` ; chemins distincts `/me/memberships` et `/admin/memberships` ; aucun test de rôle dans les handlers ; l'administrateur dispose de l'espace membre par la hiérarchie | ✅ |
| IV. Un shell, un design system | Rubriques existantes activées, entrée « Paramètres » ajoutée à la section Configuration ; composants `ui/*` existants ; `RadioGroup`, `Switch`, `Combobox`, `Dialog` base-ui ; icônes via `Icon` ; annulations et suppressions confirmées ; 320 px ; Playwright + axe | ✅ |
| V. Binaire unique en Go pur | Aucune dépendance ; migration goose embarquée ; gabarits d'e-mail embarqués ; IBAN vérifié en Go standard | ✅ |
| VI. API typée et SQL généré | Énumérations, longueurs et dates dans huma ; règles dans le domaine ; `422` par champ ; requêtes sqlc ; recherche FTS5 trigram à contenu externe sur la référence (`memberships_fts`, nouvelle) et sur la fiche (`members_fts`, existante) ; tri replié par `textfold` ; le frontend lit `actions`, `amountCents`, `amounts`, `paymentMethods` de l'API sans recalculer de montant | ✅ |
| VII. Sécurité et RGPD | Cloisonnement par fiche du compte dans le SQL, `404` uniforme ; aucune donnée personnelle nouvelle (l'IBAN est celui de l'association) ; aucun encaissement ; journaux sans nom, motif ni IBAN ([R12](./research.md#r12-journalisation)) ; effacement d'une fiche par l'administrateur emporte ses adhésions | ✅ |
| Contraintes techniques | E-mails différés via `outbound_emails`, insérés dans la transaction ; aucune tâche planifiée ; aucune dépendance | ✅ |
| Conventions de langue | Tables, chemins, champs et valeurs en anglais (`memberships`, `pending`, `transfer`, `paymentReference`) ; textes, e-mails et messages en français | ✅ |
| Workflow | Matrice d'autorisation étendue + matrice de cloisonnement ; nouvelle migration (append-only) ; contrôles de [quickstart.md](./quickstart.md) | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance :
(1) toute requête de l'espace membre joint `members` et filtre
`members.account_id = ?` (requêtes nommées `…OfAccount`), y compris
l'annonce du paiement ; (2) chaque transition relit l'adhésion dans la
transaction et passe par `membership.ActionsFor` avant d'écrire ;
(3) l'e-mail est mis en file dans la même transaction que la décision, le
destinataire étant relu à ce moment ; (4) les informations de paiement ne
sont exposées aux comptes que pour une adhésion à payer de leurs fiches, et
seulement pour les moyens activés ; (5) les `CHECK` de cohérence de
`memberships` doivent rester alignés sur le domaine : tout nouvel état passe
par une nouvelle migration.

## Project Structure

### Documentation (this feature)

```text
specs/006-memberships/
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

Fichiers ajoutés (+) ou modifiés (~) :

```text
internal/
├── membership/
│   ├── membership.go                + Status, Kind, AppliedAmount, transitions, ActionsFor
│   ├── hint.go                      + PreviousSeasonHint
│   ├── reference.go                 + ReferencePrefix, NextReference
│   ├── payment.go                   + Method, PaymentSettings, ValidatePaymentSettings,
│   │                                  NormalizeIBAN, ValidIBAN, ValidBIC
│   ├── stats.go                     + Stats d'une saison
│   └── *_test.go                    + tests en table
├── database/
│   ├── migrations/
│   │   └── 00008_memberships.sql    + memberships, payment_settings, index
│   ├── queries/
│   │   ├── memberships.sql          + …OfAccount (espace membre), admin, offre,
│   │   │                              chiffres, aide, référence, tarif utilisé
│   │   └── payment_settings.sql     + Get/Update
│   └── sqlc/                        ~ régénéré
├── mail/
│   ├── render.go                    ~ Data : Member, Season, Amount, Reference, Reason
│   └── templates/                   + membership-{validated,refused,cancelled,paid}.{txt,html}.tmpl
├── api/
│   ├── api.go                       ~ registerMemberships, registerSettings
│   ├── memberships.go               + types, conversions, transitions communes,
│   │                                  e-mails (enqueueNotice), journal
│   ├── memberships_me.go            + 6 opérations member
│   ├── memberships_admin.go         + 11 opérations admin
│   ├── settings.go                  + 2 opérations admin
│   ├── membership_fees.go           ~ inUse, delete refusé si utilisé
│   ├── members_me.go                ~ delete refusé si adhésion en cours
│   ├── members_admin.go             ~ get-member : memberships ; delete : journal
│   ├── memberships_me_test.go       +
│   ├── memberships_admin_test.go    +
│   ├── memberships_isolation_test.go +
│   └── settings_test.go             +
└── tools/e2eseed/main.go            ~ saison ouverte, tarifs, fiches, paramètres

web/app/
├── api/
│   ├── memberships.ts               + requêtes et invalidations des deux espaces
│   ├── settings.ts                  +
│   ├── openapi.json, schema.d.ts    ~ régénérés
├── lib/format.ts (+ test)           ~ libellés d'état, de type, de moyen ; formatIban
├── shell/navigation.ts              ~ « Adhésions » livrées, « Paramètres »
├── routes.ts                        ~ nouvelles routes
├── ui/
│   ├── RadioGroup.tsx               + habillage base-ui
│   ├── Switch.tsx                   + habillage base-ui
│   └── CopyButton.tsx               + copie + annonce aria-live
├── memberships/
│   ├── MembershipStatusBadges.tsx   + badges d'état et de paiement
│   ├── FeeChoice.tsx                + RadioGroup de tarifs + montant calculé
│   ├── PaymentMethodInfo.tsx        + informations d'un moyen
│   └── MemberPicker.tsx             + Combobox sur list-members
└── routes/app/
    ├── member/
    │   ├── memberships.tsx          + liste par saison
    │   ├── membership-request.tsx   + demande
    │   ├── membership-detail.tsx    + détail, annulation, paiement
    │   └── my-member-detail.tsx     ~ section « Adhésions », 409 à la suppression
    └── admin/
        ├── memberships.tsx          + filtres, chiffres, tableau
        ├── membership-new.tsx       + création directe
        ├── membership-detail.tsx    + détail et dialogues de décision
        ├── settings.tsx             + moyens de paiement
        ├── member-detail.tsx        ~ section « Adhésions », dialogue de suppression
        ├── account-detail.tsx       ~ libellé « et leurs adhésions »
        ├── SeasonFees.tsx           ~ « Supprimer » masqué si inUse
        └── home.tsx                 ~ encarts demandes en attente et paramètres

web/e2e/
├── memberships.spec.ts              + US1, US3, US4
├── admin-memberships.spec.ts        + US2, US6, US7, US8
└── admin-settings.spec.ts           + US5
```

**Structure Decision**: même organisation que 005. Le domaine
`internal/membership` est pur, comme `internal/season` et
`internal/member`. Les handlers sont séparés par espace
(`memberships_me.go`, `memberships_admin.go`) pour que chaque fichier n'utilise
que les requêtes de son espace ; `memberships.go` regroupe conversions,
transitions et e-mails communs. Les paramètres ont leur propre fichier
(`settings.go`), point d'entrée des réglages à venir. Côté frontend, les
composants partagés par les deux espaces vont dans `app/memberships/` ; les
habillages base-ui génériques (`RadioGroup`, `Switch`, `CopyButton`) dans
`app/ui/`.

## Complexity Tracking

Aucune violation de la constitution à justifier.
