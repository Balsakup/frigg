# Implementation Plan: Gestion des membres

**Branch**: `005-member-management` | **Date**: 2026-11-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/005-member-management/spec.md`

## Summary

Introduire les fiches membre (personnes de l'association) rattachées à au plus
un compte. L'espace membre gagne « Mes membres » (liste, ajout, fiche
modifiable, suppression), strictement cloisonné au compte de la session par
des requêtes SQL conditionnées sur `account_id`. L'administration gagne
« Membres » : liste de toutes les fiches avec recherche FTS5 trigram, tri sans
accents en Go, filtre « Sans compte », création, modification, rattachement
via un Combobox de comptes, suppression. La suppression d'un compte (002)
impose de choisir entre effacer ses fiches et les conserver sans compte, de
façon atomique et en détectant une liste de fiches périmée. Règles de
validation, doublon probable et âge dans un nouveau domaine pur
`internal/member`. Approche détaillée dans [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 5.9 + React 19 (frontend)

**Primary Dependencies**: aucune nouvelle. Backend : huma/v2, sqlc, goose,
`internal/textfold`, `net/mail` (bibliothèque standard) pour l'adresse
électronique. Frontend : base-ui (`Combobox`, `Dialog`, `RadioGroup`),
react-hook-form, TanStack Query, lucide-react (`UserPlus`, `Trash2`).

**Storage**: SQLite ; nouvelle migration `00007_members.sql` (table
`members`, table FTS `members_fts` et ses triggers, index), voir
[data-model.md](./data-model.md)

**Testing**: `go test` : tests en table de `internal/member` ; tests d'API
(humatest / httptest) des deux espaces, matrice de cloisonnement dédiée
(`members_isolation_test.go`), suppression de compte avec fiches, horloge
injectée pour l'âge et la date de naissance ; matrice d'autorisation
(automatique via l'OpenAPI) ; Vitest (`formatPhone`) ; Playwright + axe
(`my-members.spec.ts`, `admin-members.spec.ts`,
`admin-accounts-delete.spec.ts` étendu) ; test de volume à 2 000 fiches
(SC-004)

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` ; navigateurs
récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: SC-004 : recherche et affichage de la liste
d'administration immédiats pour 2 000 membres (liste filtrée chargée et triée
en Go, [R7](./research.md#r7-tri-et-pagination-de-la-liste-dadministration)) ;
20 fiches au plus par compte

**Constraints**: refus par défaut (principe II) ; cloisonnement dans les
requêtes SQL ; écritures sérialisées (une connexion, `BEGIN IMMEDIATE`) ; date
du jour à Paris via `Deps.Now` ; données personnelles absentes des journaux ;
RGAA ; commentaires des requêtes sqlc en ASCII

**Scale/Scope**: 6 écrans (3 par espace) + 1 dialogue modifié ;
10 opérations d'API (5 `member`, 5 `admin`) + `delete-account` enrichie ;
1 table + 1 table FTS ; 1 package de domaine

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Fiches de l'instance, rattachées aux comptes ; aucune notion de locataire | ✅ |
| II. Authentification partout | 5 opérations `member`, 5 `admin` ; aucune opération publique ajoutée | ✅ |
| III. Deux rôles | Niveau déclaré par `register` ; deux groupes de chemins (`/me/members`, `/admin/members`) plutôt qu'un handler qui teste le rôle ; l'administrateur accède à « Mes membres » par la hiérarchie ([R5](./research.md#r5-opérations-dapi)) | ✅ |
| IV. Un shell, un design system | Rubriques existantes activées ; `Table`, `Field`, `Alert`, `Badge`, `Pagination`, `ConfirmDialog` existants ; `Combobox` et `RadioGroup` base-ui ; icônes via `Icon` ; suppression confirmée ; 320 px ; Playwright + axe | ✅ |
| V. Binaire unique en Go pur | Aucune dépendance ; migration goose embarquée | ✅ |
| VI. API typée et SQL généré | Longueurs et format de date dans huma ; nettoyage et règles dans le domaine ; erreurs `422` par champ ; requêtes sqlc ; recherche FTS5 externe trigram sur des champs d'identité ; tri et doublon par `textfold`, sans colonne repliée ; le frontend lit `age` et `actions` de l'API | ✅ |
| VII. Sécurité et RGPD | Cloisonnement par compte dans le SQL, `404` uniforme ([R4](./research.md#r4-cloisonnement-par-compte-us2)) ; données minimales (identité, coordonnées) ; effacement réel des fiches et choix explicite à la suppression d'un compte ([R9](./research.md#r9-suppression-dun-compte-gérant-des-fiches-us5)) ; aucune donnée de fiche journalisée ([R13](./research.md#r13-journalisation)) ; message de doublon sans révéler le compte | ✅ |
| Contraintes techniques | Aucune dépendance, aucune tâche planifiée, aucun e-mail | ✅ |
| Conventions de langue | Table, chemins, champs et actions en anglais (`members`, `/me/members`, `birthDate`, `member.attach`) ; textes et messages en français | ✅ |
| Workflow | Matrice d'autorisation étendue + matrice de cloisonnement ; migration append-only ; contrôles de [quickstart.md](./quickstart.md) | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance :
(1) aucune requête de l'espace membre ne doit lire ou écrire une fiche sans
condition `account_id = ?` : les requêtes sont distinctes de celles de
l'administration et nommées `…OfAccount` ; (2) `delete-account` traite les
fiches dans la même transaction que la suppression du compte, avant elle ;
(3) tout changement de rattachement remet `is_self` à 0 dans la même requête
d'écriture ; (4) la date du jour vient de `s.today()`, jamais de `time.Now` ;
(5) le tri en Go suppose une liste filtrée de taille associative ; une
collation `fold` est l'évolution prévue si le volume l'exigeait (R7).

## Project Structure

### Documentation (this feature)

```text
specs/005-member-management/
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
├── member/
│   ├── member.go                    + Input, Normalize, Validate (erreurs cumulées),
│   │                                  NormalizePhone, Age, IdentityKey (textfold),
│   │                                  Sort*, constantes (MaxPerAccount…)
│   └── member_test.go               + tests en table
├── database/
│   ├── migrations/
│   │   └── 00007_members.sql        + table members, members_fts, triggers, index
│   ├── queries/
│   │   └── members.sql              + requêtes …OfAccount (espace membre),
│   │                                  List/Get/Create/Update/Delete (admin),
│   │                                  CountMembersOfAccount, ListMembersByBirthDate,
│   │                                  DeleteMembersOfAccount, DetachMembersOfAccount
│   └── sqlc/                        ~ régénéré
└── api/
    ├── api.go                       ~ enregistrement registerMembers
    ├── members.go                   + types, conversion, validation commune, doublon,
    │                                  plafond, journalisation
    ├── members_me.go                + 5 opérations member
    ├── members_admin.go             + 5 opérations admin
    ├── members_me_test.go           + parcours espace membre, « Vous », plafond
    ├── members_admin_test.go        + liste, recherche, tri, filtre, rattachement
    ├── members_isolation_test.go    + matrice de cloisonnement (US2)
    ├── accounts.go                  ~ deleteAccount : paramètres members, memberIds
    └── accounts_delete_test.go      ~ cas avec fiches (delete, keep, 422, 409)

web/app/
├── api/
│   ├── members.ts                   + requêtes et invalidations des deux espaces
│   ├── openapi.json, schema.d.ts    ~ régénérés
├── lib/
│   ├── format.ts                    ~ formatPhone, formatAge
│   └── format.test.ts               ~
├── shell/navigation.ts              ~ « Mes membres » et « Membres » livrées
├── routes.ts                        ~ routes membres, retrait de comingSoon
├── members/
│   ├── MemberFields.tsx             + champs communs (identité, coordonnées, adresse)
│   ├── DuplicateConfirm.tsx         + case « Enregistrer quand même »
│   └── AccountPicker.tsx            + Combobox base-ui des comptes + « Sans compte »
└── routes/app/
    ├── member/
    │   ├── my-members-layout.tsx    + fil d'Ariane
    │   ├── my-members.tsx           + liste en cartes, état vide
    │   ├── my-member-new.tsx        +
    │   └── my-member-detail.tsx     + formulaire, suppression
    └── admin/
        ├── members-layout.tsx       +
        ├── members.tsx              + tableau, recherche, filtre, pagination
        ├── member-new.tsx           +
        ├── member-detail.tsx        + informations, formulaire, suppression
        └── account-detail.tsx       ~ dialogue de suppression : fiches et choix

web/e2e/
├── my-members.spec.ts               + US1, US2
├── admin-members.spec.ts            + US3, US4
└── admin-accounts-delete.spec.ts    ~ US5
```

**Structure Decision**: même organisation que 002 à 004. Le domaine
`internal/member` est pur (aucune dépendance à la base), comme
`internal/season`. Les handlers sont séparés par espace (`members_me.go`,
`members_admin.go`) pour que chaque fichier n'utilise que les requêtes de son
espace ; `members.go` regroupe ce qui leur est commun. Côté frontend, les
composants partagés par les deux espaces vont dans `app/members/`.

## Complexity Tracking

Aucune violation de la constitution à justifier.
