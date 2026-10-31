# Implementation Plan: Gestion des tarifs de l'adhésion

**Branch**: `004-membership-fees` | **Date**: 2026-10-31 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/004-membership-fees/spec.md`

## Summary

Ajouter une section « Tarifs de l'adhésion » à la fiche d'une saison :
liste triée, état vide, message informatif pour une saison sans tarif, ajout
et modification dans un dialogue, suppression confirmée, lecture seule sur une
saison close. Les tarifs sont stockés dans une table `membership_fees`
rattachée à la saison (montant en centimes, cascade à la suppression de la
saison) et recopiés par `duplicate-season` dans la même transaction. Les règles
(analyse du montant, unicité du libellé sans casse ni accents, plafond de
20, éditabilité selon l'état calculé de la saison) vivent dans le domaine pur
`internal/season` ; la comparaison des libellés passe par le nouveau package
`internal/textfold` prévu par la constitution. Approche détaillée dans
[research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 5.9 + React 19 (frontend)

**Primary Dependencies**: inchangées depuis 003, sauf `golang.org/x/text`
(`unicode/norm`) qui passe d'indirect à direct pour `internal/textfold` ; elle
figure dans la liste fermée de la constitution
([R2](./research.md#r2-unicité-du-libellé-sans-casse-ni-accents)). Frontend :
base-ui (`Dialog`), react-hook-form, TanStack Query, lucide-react (`Plus`,
`Pencil`, `Trash2`) ; `Intl.NumberFormat` natif pour les montants.

**Storage**: SQLite ; nouvelles migrations `00005_membership_fees.sql` (table
`membership_fees`) et `00006_membership_fee_renewal.sql` (montant de
renouvellement, [R12](./research.md#r12-montant-de-renouvellement)), voir
[data-model.md](./data-model.md)

**Testing**: `go test` : tests en table de `internal/textfold` et de
`internal/season/fee.go` ; tests d'API (humatest / httptest) avec horloge
injectée (saison close), duplication, cascade, ajouts concurrents ; matrice
d'autorisation (automatique via l'OpenAPI) ; Vitest (`formatAmount`,
`amountInput`) ; Playwright + axe (nouveau test dans `admin-seasons.spec.ts`, exécuté en série)

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` ; navigateurs
récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: affichage immédiat de la fiche ; 20 tarifs au plus par
saison, tous chargés à chaque requête

**Constraints**: refus par défaut (principe II) ; écritures sérialisées (une
connexion, `BEGIN IMMEDIATE`) ; date du jour à Europe/Paris via `Deps.Now` ;
montants en centimes entiers ; RGAA ; commentaires des requêtes sqlc en ASCII

**Scale/Scope**: 1 section dans un écran existant + 2 dialogues ; 4 opérations
d'API `admin` ; 1 table ; 2 opérations existantes enrichies (duplication,
suppression par cascade) ; 1 package utilitaire (`textfold`)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Tarifs rattachés aux saisons de l'instance, aucune notion de locataire | ✅ |
| II. Authentification partout | 4 opérations `admin`, aucune opération publique ajoutée | ✅ |
| III. Deux rôles | Niveau d'accès déclaré par `register` pour chaque opération ; aucune vérification de rôle ad hoc | ✅ |
| IV. Un shell, un design system | Section dans la fiche existante ; `Table`, `Alert`, `Field`, `Button` existants ; `Dialog` base-ui pour l'ajout et la modification ; `ConfirmDialog` `danger` pour la suppression ; icônes via `Icon` ; 320 px ; Playwright + axe | ✅ |
| V. Binaire unique en Go pur | `golang.org/x/text` est en Go pur ; migration goose embarquée | ✅ |
| VI. API typée et SQL généré | Longueurs déclarées dans huma (`maxLength`) ; montant analysé par le serveur ([R3](./research.md#r3-saisie-et-format-du-montant)) ; erreurs `422` par champ ; requêtes sqlc ; unicité par `internal/textfold`, sans colonne repliée ; le frontend ne réimplémente aucune règle (actions et montants en centimes viennent de l'API) | ✅ |
| VII. Sécurité | Aucune donnée personnelle ; libellés libres non journalisés ([R10](./research.md#r10-journalisation)) | ✅ |
| Contraintes techniques | Dépendance de la liste fermée rendue directe, justifiée en R2 ; aucune tâche planifiée | ✅ |
| Conventions de langue | Table, chemins, champs et actions en anglais (`membership_fees`, `/fees`, `amountCents`, `fee.create`) ; textes et messages en français | ✅ |
| Workflow | Matrice d'autorisation étendue ; nouvelle migration append-only ; contrôles de [quickstart.md](./quickstart.md) | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance :
(1) toute écriture de tarif relit la saison et calcule son état avec
`s.today()`, jamais `time.Now` ; (2) la cascade suppose `foreign_keys=ON`,
déjà dans la chaîne de connexion : un test d'API le vérifie ; (3) la copie des
tarifs doit se faire dans la même fonction de transaction que la création de
la saison dupliquée ; (4) `amountCents` n'est jamais converti en flottant
côté Go.

## Project Structure

### Documentation (this feature)

```text
specs/004-membership-fees/
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
go.mod                               ~ golang.org/x/text en dépendance directe

internal/
├── textfold/
│   ├── textfold.go                  + Fold (espaces, NFD, diacritiques, minuscules)
│   └── textfold_test.go             +
├── season/
│   ├── fee.go                       + ParseAmount, ValidateFee (erreurs cumulées),
│   │                                  FeeActionsFor, SortFees, constantes
│   └── fee_test.go                  + tests en table
├── database/
│   ├── migrations/
│   │   ├── 00005_membership_fees.sql + table membership_fees
│   │   └── 00006_membership_fee_renewal.sql + montant de renouvellement
│   ├── queries/
│   │   └── membership_fees.sql      + List, Create, Update, Delete, Copy
│   └── sqlc/                        ~ régénéré
└── api/
    ├── api.go                       ~ enregistrement registerMembershipFees
    ├── membership_fees.go           + 4 opérations admin, traduction des erreurs,
    │                                  journalisation
    ├── membership_fees_test.go      + parcours, refus, saison close, plafond,
    │                                  concurrence, cascade
    ├── seasons.go                   ~ duplicateSeason recopie les tarifs
    └── seasons_duplicate_test.go    + tarifs recopiés et indépendants

web/app/
├── api/
│   ├── membership-fees.ts           + requête et invalidation
│   ├── openapi.json, schema.d.ts    ~ régénérés
├── lib/
│   ├── format.ts                    ~ formatAmount, amountInput
│   └── format.test.ts               ~
├── ui/
│   ├── Field.tsx                    ~ TextAreaField (description)
│   └── Button.tsx                   ~ accepte ref (focus après suppression)
└── routes/app/admin/
    ├── season-detail.tsx            ~ préchargement et rendu de SeasonFees
    ├── SeasonFees.tsx               + section, tableau, état vide, message
    └── FeeDialog.tsx                + dialogue d'ajout et de modification

web/e2e/
└── admin-seasons.spec.ts            ~ test « tarifs (004) » à la suite, en série
```

**Structure Decision**: même organisation que 003. Les règles des tarifs
rejoignent `internal/season`, dont elles dépendent (état de la saison) ;
`internal/textfold` est créé comme prévu par la constitution et servira aux
prochaines comparaisons de texte saisi. Les handlers de
`internal/api/membership_fees.go` réutilisent le motif de `seasonWrite`
(relecture en transaction, appel du domaine, écriture, traduction en
`404`/`409`/`422`).

## Complexity Tracking

Aucune violation de la constitution à justifier.
