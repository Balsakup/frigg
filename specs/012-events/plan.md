# Implementation Plan: Événements de l'association

**Branch**: `012-events` | **Date**: 2026-11-09 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/012-events/spec.md`

## Summary

Les administrateurs créent des **événements** rattachés à une saison
(brouillon → publié → annulé, « Terminé » calculé), avec une ouverture par
événement (adhérents de la saison ou toutes les fiches), une capacité et un
prix facultatifs. Les comptes inscrivent les fiches qu'ils gèrent, sans liste
d'attente, et paient les inscriptions payantes par les **moyens de paiement
existants**, carte HelloAsso comprise. La rubrique « À régler » regroupe
adhésions et inscriptions à payer.

Stockage : deux tables nouvelles, `events` et `event_registrations` ; cette
dernière reprend les colonnes de paiement de `memberships` (R1). Pour un seul
circuit de paiement par carte, `online_payments` et `payment_alerts` sont
**reconstruites** dans une migration `NO TRANSACTION`, avec une cible
adhésion **ou** inscription (R2, sonde validée avec sqlc et goose sur base
peuplée). Le domaine pur `internal/event` porte les règles ; il présente une
inscription au domaine de paiement de `internal/membership` par un
adaptateur, si bien qu'aucune règle de paiement n'est dupliquée (R3). Le
circuit de carte de 008 est généralisé à une « cible » sans nouvelle route
publique (R4). Détails : [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 6.0 + React 19 (frontend)

**Primary Dependencies**: aucune nouvelle. Existantes : huma/v2, sqlc
(moteur SQLite ; `ALTER TABLE … RENAME` et `PRAGMA` en migration vérifiés),
goose v3.29 (`-- +goose NO TRANSACTION` vérifié), `internal/season`
(`Paris`, `Today`), `internal/membership` (paiement, `SettleOnline`,
`NextReference`), `internal/communication` (`BuildAudience`),
`internal/helloasso`, `internal/textfold`. Frontend : base-ui (`Dialog`,
`RadioGroup`, `Select`, `Combobox`, `Collapsible`, `Menu`), react-hook-form,
TanStack Query, `openapi-fetch`, lucide-react (`Calendar`, `CalendarPlus`,
`MapPin`, `Users`, `Download`, `Ban`, `RotateCcw`).

**Storage**: SQLite ; migration `00012_events.sql` : `events`,
`event_registrations` (+ `event_registrations_fts` et ses triggers),
reconstruction de `online_payments` et `payment_alerts`
([data-model.md](./data-model.md)).

**Testing**: `go test` : tests en table de `internal/event` ; test de
migration sur base `00011` peuplée ; tests d'API httptest (cycle de vie,
inscriptions, concurrence, éligibilité, paiement manuel et par carte via
`hatest`, notification, remboursement, « À régler », CSV, regroupement des
e-mails, isolation) ; matrice d'autorisation automatique. Vitest (dates,
formulaire). Playwright + axe (`events.spec.ts`) ([R16](./research.md#r16-tests)).

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` ;
navigateurs récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: liste et export de 500 inscrits < 2 s (SC-007) :
requête indexée `(event_id, status)`, jointure fiches et compte ; 50
inscriptions simultanées sans dépassement de capacité (SC-004), sérialisées
par l'écrivain unique en `BEGIN IMMEDIATE`

**Constraints**: aucun brouillon ni fiche non gérée transmis à un compte
membre ; pas de SQL dynamique ; pas de nouvelle route publique ; aucune
donnée personnelle transmise à HelloAsso ni écrite dans les journaux ;
commentaires sqlc en ASCII ; migration append-only ; e-mails dans la
transaction de l'action

**Scale/Scope**: une association, quelques centaines d'événements et
quelques milliers d'inscriptions par saison ; 27 opérations ajoutées (10
`member`, 17 `admin`), 7 modifiées ; 11 écrans nouveaux, 4 composants de
paiement extraits ; 6 modèles d'e-mail

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution v1.8.0.

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Événements et inscriptions = données de l'instance, sans locataire | ✅ |
| II. Authentification partout | 27 opérations sous session ; **aucune route publique ajoutée** : les notifications HelloAsso des inscriptions arrivent sur l'adresse existante, qui reste limitée à déclencher une vérification (R4) | ✅ |
| III. Deux rôles | Rôle déclaré par `register` pour chaque opération (10 `member`, 17 `admin`) ; l'isolation des fiches passe par les requêtes « … OfAccount » existantes ; pas de test de rôle dans les handlers ; matrice automatique | ✅ |
| IV. Un shell, un design system | Rubriques déjà présentes dans la navigation ; composants `ui/*`, base-ui, tokens (`--ok`, `--warn`, `--danger`, `--info`, `--muted`) ; `ConfirmDialog` pour annuler, supprimer, retirer, annuler une constatation ; 320 px ; RGAA (fieldset, raisons reliées, `aria-live`) | ✅ |
| V. Binaire unique en Go pur | Aucune dépendance ; migration embarquée ; un échec de `00012` empêche le démarrage | ✅ |
| VI. API typée et SQL généré | Bornes déclarées dans huma, montants analysés par le serveur (comme les tarifs) ; requêtes sqlc statiques ; recherche par FTS5 à contenu externe trigram (`event_registrations_fts` pour la référence, `members_fts` pour les noms) ; dates locales converties côté serveur | ✅ |
| VII. Sécurité et RGPD | Inscriptions supprimées avec la fiche (cascade) ; intitulé HelloAsso sans nom du membre ; CSV protégé contre l'injection de formule ; journaux sans titre, lieu, nom ni adresse ; 404 uniforme hors des fiches du compte | ✅ |
| Contraintes techniques | Écritures en `WriteTx` (`BEGIN IMMEDIATE`) ; e-mails insérés dans la même transaction ; balayage des tentatives existant réutilisé | ✅ |
| Conventions de langue | Identifiants anglais (`events`, `openness`, `season_members`) ; textes, erreurs, journaux et e-mails en français | ✅ |
| Workflow | Nouvelle migration (append-only, `00008` non modifiée) ; matrice étendue ; contrôles de [quickstart.md](./quickstart.md) | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance :

1. **Migration de reconstruction** : `PRAGMA foreign_keys = OFF` avant
   `BEGIN`, `ON` en dernier statement ; test sur base peuplée qui vérifie la
   conservation des lignes, `foreign_key_check` vide, `foreign_keys = 1` et
   la clé `memberships.paid_online_payment_id` encore active. Un
   `Down` symétrique est fourni et testé.
2. **Un seul chemin de constatation** : `verifyOnlinePayment` reste l'unique
   fonction qui constate une carte ; la cible des métadonnées HelloAsso doit
   correspondre à celle de la tentative (adhésion ou inscription), test
   croisé (métadonnées d'adhésion sur une tentative d'inscription → rien).
3. **Capacité** : contrôle et insertion dans la même transaction ; test de
   concurrence ; une demande multi-fiches est tout ou rien.
4. **Brouillons et fiches non gérées** : requêtes `/me` filtrées par
   `status <> 'draft'` et par compte ; test de réponse brute.
5. **Paiement d'un événement annulé** : adaptateur → `Cancelled` ; un
   paiement par carte reçu produit `refund_due`, jamais « Payée ».

## Project Structure

### Documentation (this feature)

```text
specs/012-events/
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
├── event/                              # nouveau : domaine pur, sans base
│   ├── event.go                        # Event, Status, Phase, Openness, ValidateEvent, dates locales
│   ├── registration.go                 # Eligible, RegistrationOpen, Can*, CapacityLeft
│   ├── payment.go                      # PaymentSnapshot (adaptateur vers membership.Snapshot)
│   ├── reference.go                    # ReferencePrefix EVT-AAAA-
│   └── *_test.go
├── helloasso/checkout.go               # Metadata.Registration, Membership omitempty
├── database/
│   ├── migrations/00012_events.sql     # nouvelle (NO TRANSACTION)
│   ├── queries/events.sql              # événements (ASCII)
│   ├── queries/event_registrations.sql # inscriptions, paiement, FTS, « À régler »
│   ├── queries/online_payments.sql     # cible inscription
│   ├── queries/payment_alerts.sql      # cible inscription
│   ├── queries/seasons.sql             # contrôles des événements
│   └── sqlc/                           # régénéré
├── mail/templates/                     # registration-*, event-* (txt + html)
└── api/
    ├── events_admin.go                 # 8 opérations événements
    ├── events_me.go                    # list/get-my-event, register-my-members
    ├── registrations_admin.go          # inscrits, CSV, register-member, paiement
    ├── registrations_me.go             # mes inscriptions, annulation, paiement, list-my-dues
    ├── event_notices.go                # e-mails regroupés par adresse
    ├── card_payments.go                # cible (payTarget) adhésion | inscription
    ├── online_sweeper.go, helloasso_webhook.go, payment_alerts.go   # cible
    ├── seasons.go, members*.go         # contrôles et compteur de suppression
    ├── api.go                          # registerEvents, registerRegistrations
    └── *_test.go

web/
├── app/
│   ├── routes.ts, shell/navigation.ts  # rubriques livrées
│   ├── routes/app/member/              # events, event-detail, registrations, registration-detail, dues
│   ├── routes/app/admin/               # events, event-new, event-edit, event-detail, registrations, registration-detail
│   ├── events/                         # EventForm, EventDate, EventBadges, RegisterMembers, RegisterMemberDialog
│   ├── payments/                       # PaymentPanel, RecordPaymentDialog, OnlinePayments, PaymentAlerts (extraits de 006/008)
│   ├── routes/app/member/membership-detail.tsx, admin/membership-detail.tsx   # utilisent payments/*
│   ├── api/events.ts, api/registrations.ts
│   └── api/openapi.json, api/schema.d.ts   # régénérés
└── e2e/events.spec.ts                  # nouveau

internal/tools/e2eseed                  # événements d'exemple
```

**Structure Decision**: structure existante (API Go + SPA dans `web/`). Un
paquet pur nouveau `internal/event`, sur le modèle de `internal/membership`,
dont il réutilise le domaine de paiement par adaptateur. Côté API, le
circuit de carte de 008 est paramétré par une cible plutôt que copié. Côté
frontend, les composants de paiement des adhésions sont extraits dans
`payments/` pour servir aux deux objets payables.

## Complexity Tracking

Aucune violation de la constitution v1.8.0. Écarts de complexité retenus :

| Élément | Pourquoi | Alternative plus simple écartée |
| --- | --- | --- |
| Reconstruction de `online_payments` et `payment_alerts` en migration `NO TRANSACTION` | Une tentative de carte et un signalement doivent pouvoir viser une inscription ; SQLite ne sait pas rendre une colonne `NOT NULL` facultative sans reconstruire la table | Tables `event_online_payments`/`event_payment_alerts` parallèles : migration triviale mais deux circuits de constatation à maintenir, contraire à l'unicité du chemin de constatation de 008 |
| Interface interne `payTarget` dans `card_payments.go` | Seul point de variation entre adhésion et inscription dans la constatation, le remboursement et les e-mails | Dupliquer les ~400 lignes du circuit carte pour les inscriptions |
| Adaptateur `event.PaymentSnapshot` vers `membership.Snapshot` | Réutiliser `CanPay`, `SettleOnline`, `RefundCancels` déjà testés | Extraire un paquet `internal/payment` : déplacement de code stable sans besoin immédiat ; envisageable au troisième objet payable |
| Extraction des composants de paiement frontend dans `payments/` | La page de paiement et les panneaux d'administration servent aux adhésions et aux inscriptions | Copier les composants : divergence garantie à la prochaine évolution des moyens de paiement |
