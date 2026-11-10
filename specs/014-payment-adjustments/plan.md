# Implementation Plan: Remises et dons sur les paiements

**Branch**: `014-payment-adjustments` | **Date**: 2026-11-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/014-payment-adjustments/spec.md`

## Summary

Un administrateur peut fixer le **montant à payer** d'une adhésion (à la
validation, à la création, à la correction tant qu'elle n'est pas payée) et
d'une inscription (à l'inscription par un administrateur, puis par une action
« Ajuster le montant »). Ce montant est prérempli par le montant théorique et
exige un motif s'il en diffère. À la constatation d'un paiement en espèces,
par chèque ou par virement, il saisit le **montant reçu** : l'excédent est un
**don**, un montant inférieur est refusé. Les chiffres de la saison et des
événements distinguent le théorique, les remises, les majorations, le montant
à payer, l'encaissé, les dons et le reste à percevoir.

Approche : `amount_cents` devient le montant à payer, si bien que la carte,
« À régler », les e-mails et la reprise des tentatives HelloAsso suivent sans
changement. Deux colonnes facultatives, `base_amount_cents` et
`adjustment_reason`, ne sont renseignées qu'en cas d'ajustement : aucune
reprise n'est nécessaire (R1). `paid_amount_cents`, déjà présent, devient le
montant reçu de tout paiement ; la migration le renseigne pour les paiements
manuels existants (R2). Les règles vont dans `internal/membership/amount.go`,
partagé par les adhésions et les inscriptions (R3). Détails :
[research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 6.0 + React 19 (frontend)

**Primary Dependencies**: aucune nouvelle. Existantes : huma/v2, sqlc,
goose v3.29, `internal/membership` (`PaymentAfter`, `CleanReason`,
`ComputeStats`), `internal/event` (`PaymentSnapshot`), `internal/season`
(`ParseAmount`), `internal/csvfile`. Frontend : react-hook-form, TanStack
Query, `openapi-fetch`, composants `ui/*` existants.

**Storage**: SQLite ; migration `00013_payment_adjustments.sql` : deux
colonnes `ADD COLUMN … CHECK` sur `memberships` et `event_registrations`,
reprise de `paid_amount_cents` ([data-model.md](./data-model.md)).

**Testing**: `go test` : tests en table du domaine ; test de migration sur
base `00012` peuplée, `Down` compris ; tests d'API httptest (validation,
correction, création, constatation, ajustement d'inscription, carte via
`hatest`, chiffres, CSV, e-mails, isolation de la réponse membre) ; matrice
d'autorisation automatique. Vitest (`useTrackedAmount`). Playwright + axe
([R10](./research.md#r10-tests)).

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` ;
navigateurs récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: sans enjeu nouveau. Les chiffres restent calculés en
une requête par saison ou par événement.

**Constraints**: motif jamais transmis à un compte membre ni journalisé ;
montants analysés par le serveur ; pas de SQL dynamique ; commentaires sqlc
en ASCII ; migration append-only (instance déjà déployée) ; aucune route
publique

**Scale/Scope**: 1 opération ajoutée (`admin`). 9 opérations voient leur
comportement modifié (`validate-membership`, `update-membership`,
`create-membership`, `record-membership-payment`, `get-membership-stats`,
`register-member`, `record-registration-payment`,
`export-event-registrations`, `get-event`). D'autres voient seulement leur
réponse enrichie : les opérations admin de lecture des adhésions et des
inscriptions (`get-membership`, `list-memberships`, `get-registration`,
`list-registrations`, `list-event-registrations`) reçoivent
`baseAmountCents`, `adjustmentCents` et `adjustmentReason`, et toute réponse
qui contient un `Payment`, espace membre compris, reçoit `donationCents` ;
2 composants frontend (1 nouveau, 1 modifié), 8 écrans retouchés ; 4 modèles
d'e-mail (2 × txt/html)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution v1.8.0.

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Colonnes sur des tables de l'instance, sans locataire | ✅ |
| II. Authentification partout | Aucune route publique ; la nouvelle opération est sous session | ✅ |
| III. Deux rôles | `adjust-registration-amount` déclarée `admin` par `register` ; montants modifiables seulement par des opérations `admin` ; matrice automatique | ✅ |
| IV. Un shell, un design system | Composants `ui/*` et tokens existants ; dialogues existants enrichis ; RGAA (étiquettes, `aria-describedby`, `aria-live`) ; 320 px ; pas d'action destructive nouvelle | ✅ |
| V. Binaire unique en Go pur | Aucune dépendance ; migration embarquée ; un échec de `00013` empêche le démarrage | ✅ |
| VI. API typée et SQL généré | Bornes déclarées dans huma (`maxLength`) ; montants analysés par `season.ParseAmount` côté serveur ; le frontend n'analyse ni ne compare les montants ; requêtes sqlc statiques ; types frontend régénérés | ✅ |
| VII. Sécurité et RGPD | Motif réservé aux administrateurs (R9), absent des journaux (R7) ; motif protégé contre l'injection de formule dans le CSV (`SafeCell`) | ✅ |
| Contraintes techniques | Écritures en `WriteTx` ; e-mails dans la transaction de l'action | ✅ |
| Conventions de langue | Identifiants anglais (`base_amount_cents`, `adjustment_reason`, `receivedAmount`) ; textes, erreurs, e-mails en français | ✅ |
| Workflow | Nouvelle migration (append-only) ; matrice étendue ; contrôles de [quickstart.md](./quickstart.md) | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance :

1. **Un seul montant demandé** : aucun code ne doit relire le tarif ou le
   prix de l'événement pour un montant à payer figé. Tests : carte, « À
   régler » et e-mail de validation au montant ajusté.
2. **Fuite du motif** : test de réponse brute de l'espace membre (adhésion,
   inscription, « À régler ») sans `adjustmentReason`, `adjustmentCents` ni `baseAmountCents`.
3. **Reprise** : après `00013`, aucune ligne payée sans `paid_amount_cents` ;
   la constatation et son annulation maintiennent cette règle.
4. **Identité des chiffres** : `théorique − remises + majorations = à payer`,
   testée sur un jeu mixte (remise, majoration, gratuit par ajustement,
   annulée, carte en mode test).

## Project Structure

### Documentation (this feature)

```text
specs/014-payment-adjustments/
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
├── membership/
│   ├── amount.go                       # nouveau : Adjust, CheckReceived, Donation, AmountTotals
│   ├── amount_test.go                  # nouveau
│   └── stats.go                        # ComputeStats s'appuie sur AmountTotals
├── database/
│   ├── migrations/00013_payment_adjustments.sql   # nouvelle
│   ├── queries/memberships.sql         # Create/Validate/Update, Record/CancelPayment, stats
│   ├── queries/event_registrations.sql # CreateRegistration, AdjustRegistrationAmount, Record/CancelPayment
│   └── sqlc/                           # régénéré
├── mail/
│   ├── render.go                       # Data.Donation
│   └── templates/membership-paid.*, registration-paid.*
└── api/
    ├── memberships_admin.go            # DecisionBody, AdminCreateBody, RecordPaymentBody, stats, détail
    ├── memberships.go                  # Payment.donationCents, notifyMembership, paymentColumns
    ├── registrations_admin.go          # register-member, adjust-registration-amount, constatation, CSV
    ├── registration_target.go          # notifyRegistrationPaid (don)
    ├── events_admin.go                 # EventTotals via AmountTotals
    ├── events_me.go                    # createRegistrations : base et motif
    ├── errors.go                       # nouveaux messages
    └── *_test.go

web/
├── app/
│   ├── payments/AmountToPayField.tsx   # nouveau (+ useTrackedAmount et son test)
│   ├── payments/AdminPayments.tsx      # RecordPaymentDialog : montant reçu
│   ├── events/RegisterMemberDialog.tsx
│   ├── routes/app/admin/               # membership-detail, membership-new, memberships, registration-detail, event-detail
│   ├── routes/app/member/              # membership-detail, registration-detail (don)
│   └── api/openapi.json, api/schema.d.ts   # régénérés
└── e2e/payment-adjustments.spec.ts     # nouveau

internal/tools/e2eseed                  # une adhésion ajustée et un paiement avec don
```

**Structure Decision**: structure existante. Aucune table ni aucun paquet
nouveau : le domaine de paiement de `internal/membership` est étendu et sert
aux inscriptions par l'adaptateur de 012. Côté frontend, un composant de
saisie du montant à payer est partagé par les quatre formulaires qui en ont
besoin.

## Complexity Tracking

Aucune violation de la constitution v1.8.0, et aucun écart de complexité :
la fonctionnalité ajoute des colonnes et des champs facultatifs, sans table,
paquet, dépendance ni reconstruction de table.
