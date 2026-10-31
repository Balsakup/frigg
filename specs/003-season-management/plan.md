# Implementation Plan: Gestion des saisons par les administrateurs

**Branch**: `003-season-management` | **Date**: 2026-10-31 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/003-season-management/spec.md`

## Summary

Ajouter la rubrique « Saisons » à l'espace administration, dans la section du
bas de la barre latérale renommée « Configuration » : liste, état vide,
création, fiche, modification des dates, ouverture, duplication et suppression,
plus un encart de mise en place sur le tableau de bord. Les saisons sont
stockées avec leurs dates civiles et leur date d'ouverture ; l'état « close »,
la saison courante, le libellé et les actions autorisées sont calculés à chaque
lecture par un package de domaine pur (`internal/season`) à partir de la date
du jour à Europe/Paris, fournie par une horloge injectable. Les écritures
sérialisées relisent la suite des saisons pour garantir la continuité sans
chevauchement ni trou. Aucune nouvelle dépendance. Approche détaillée dans
[research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 5.9 + React 19 (frontend)

**Primary Dependencies**: inchangées depuis 002 : huma/v2 (humago), goose/v3,
modernc.org/sqlite, sqlc 1.31 ; React Router v8, TanStack Query,
openapi-fetch, openapi-react-query, react-hook-form, base-ui (`Dialog`),
lucide-react (`CalendarRange`), Tailwind v4. `time/tzdata` (bibliothèque
standard) est importé pour embarquer le fuseau Europe/Paris
([R3](./research.md#r3-aujourdhui--fuseau-et-horloge-injectable)).

**Storage**: SQLite ; nouvelle migration `00004_seasons.sql` (table `seasons`,
voir [data-model.md](./data-model.md))

**Testing**: `go test` : tests en table du package `internal/season` ; tests
d'API (humatest / httptest) avec horloge injectée, dont une duplication
concurrente ; matrice d'autorisation (automatique via l'OpenAPI) ; Vitest
(`formatDay`, navigation) ; Playwright + axe (`admin-seasons.spec.ts`)

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` ; navigateurs
récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: affichage immédiat de la liste et de la fiche ; volume
de quelques dizaines de saisons, toutes chargées à chaque requête

**Constraints**: refus par défaut (principe II) ; écritures sérialisées (une
connexion, `BEGIN IMMEDIATE`) ; date du jour à Europe/Paris quel que soit le
fuseau du serveur ; RGAA ; commentaires des requêtes sqlc en ASCII

**Scale/Scope**: 3 écrans (liste, création, fiche) + encart du tableau de
bord ; 7 opérations d'API `admin` ; 1 table ; 1 package de domaine

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Saisons = données de l'instance, aucune notion de locataire | ✅ |
| II. Authentification partout | 7 opérations `admin`, aucune opération publique ajoutée | ✅ |
| III. Deux rôles | Niveau d'accès déclaré par `register` pour chaque opération ; aucune vérification de rôle ad hoc | ✅ |
| IV. Un shell, un design system | Pages dans le shell ; `Table`, `Badge`, `Alert`, `Field`, `Button` existants ; `Dialog` base-ui pour la modification ; `ConfirmDialog` pour l'ouverture et la suppression (destructive) ; icône via `Icon` ; champ de date natif ; 320 px ; Playwright + axe | ✅ |
| V. Binaire unique en Go pur | `time/tzdata` est en Go pur ; migration goose embarquée | ✅ |
| VI. API typée et SQL généré | Dates validées par huma (`format: date`) puis par le domaine ; erreurs `422` par champ ; requêtes sqlc ; le frontend ne réimplémente aucune règle : actions, bornes, libellés et `setup` viennent de l'API ([R8](./research.md#r8-actions-autorisées-exposées-par-lapi)) ; attributs HTML natifs `min`/`max` | ✅ |
| VII. Sécurité | Aucune donnée personnelle ; journaux sans adresse (identifiants seulement) | ✅ |
| Contraintes techniques | Aucune dépendance ajoutée ; aucune tâche planifiée (clôture calculée) | ✅ |
| Conventions de langue | Chemins API, champs et valeurs en anglais (`draft`, `open`, `closed`, `no-current`) ; chemins du SPA en français (`/admin/saisons/nouvelle`) ; libellés, messages et journaux en français | ✅ |
| Workflow | Matrice d'autorisation étendue ; nouvelle migration append-only ; contrôles de [quickstart.md](./quickstart.md) | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance :
(1) `Deps.Now` doit être utilisé par toutes les lectures de « aujourd'hui »
des saisons, jamais `time.Now` directement ; (2) `formatDate` existant décale
les dates civiles à l'ouest de Greenwich : utiliser `formatDay` pour les dates
de saison ; (3) le renommage de la clé `resources` en `settings` touche
`Sidebar.tsx` et `navigation.test.ts`.

## Project Structure

### Documentation (this feature)

```text
specs/003-season-management/
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
cmd/frigg/main.go                ~ import _ "time/tzdata"

internal/
├── season/
│   ├── date.go                  + type Date (analyse, formatage, AddDays, AddMonths)
│   ├── season.go                + Season, State, IsCurrent, Labels, NextStart,
│   │                              DuplicateDates, Actions, bornes, Setup,
│   │                              ValidateCreate, ValidateUpdate
│   ├── date_test.go             +
│   └── season_test.go           + tests en table
├── database/
│   ├── migrations/
│   │   └── 00004_seasons.sql    + table seasons
│   ├── queries/
│   │   └── seasons.sql          + ListSeasons, CreateSeason, UpdateSeasonDates,
│   │                              OpenSeason, DeleteSeason
│   └── sqlc/                    ~ régénéré
├── api/
│   ├── api.go                   ~ Deps.Now ; enregistrement registerSeasons
│   ├── seasons.go               + 7 opérations admin, traduction des erreurs du
│   │                              domaine, journalisation
│   ├── seasons_test.go          + parcours, refus, horloge, concurrence
│   │                              (access_test.go parcourt déjà l'OpenAPI)
│   └── apitest/apitest.go       ~ option d'horloge
└── cli/serve.go                 ~ (rien si Deps.Now nul = time.Now)

web/app/
├── api/
│   ├── seasons.ts               + clés et requêtes (liste, fiche)
│   ├── openapi.json, schema.d.ts ~ régénérés
├── lib/
│   ├── format.ts                ~ formatDay (date civile), seasonStatusLabel
│   └── format.test.ts           +
├── shell/
│   ├── navigation.ts            ~ clé settings, entrée saisons
│   ├── navigation.test.ts       ~
│   └── Sidebar.tsx              ~ titre de section selon l'espace
├── routes.ts                    ~ admin/saisons, nouvelle, :id
└── routes/app/admin/
    ├── home.tsx                 ~ encart de mise en place (setup)
    ├── seasons-layout.tsx       + fil d'Ariane « Saisons »
    ├── seasons.tsx              + liste et état vide
    ├── season-new.tsx           + création
    ├── season-detail.tsx        + fiche, dialogues d'action
    └── SeasonDatesFields.tsx    + champs de dates partagés création / modification

web/e2e/
└── admin-seasons.spec.ts        + parcours US1–US5 + axe (en série)
```

**Structure Decision**: même organisation que 001 et 002. Les règles métier
vivent dans `internal/season`, premier package de domaine du projet : il ne
dépend ni de la base ni de huma, ce qui permet de tester exhaustivement les
combinaisons de dates et d'états. Les handlers de `internal/api/seasons.go`
font le lien (lecture de la suite en transaction, appel du domaine, écriture,
traduction des erreurs en `409`/`422`). `internal/textfold` n'est pas
nécessaire (aucun texte saisi).

## Complexity Tracking

Aucune violation de la constitution à justifier.
