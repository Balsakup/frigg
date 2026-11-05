# Implementation Plan: Champs personnalisés des membres et des adhésions

**Branch**: `011-custom-fields` | **Date**: 2026-11-05 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/011-custom-fields/spec.md`

## Summary

Permettre aux administrateurs de définir, dans « Paramètres › Champs
personnalisés », des champs supplémentaires pour les **fiches membres**
(valeur durable, par exemple le numéro de licence fédérale) et pour les
**adhésions** (valeur propre à une saison, par exemple « A payé sa licence
fédérale »). Six types, trois visibilités (réservé, visible, rempli par le
membre), ordre, archivage et suppression. Stockage en définitions + valeurs à
**colonnes typées** dans deux tables de valeurs reliées par clé étrangère aux
fiches et aux adhésions (R1). Les règles vivent dans un domaine pur
`internal/customfield` (R2). La visibilité est appliquée selon l'**espace** de
l'opération (`/admin` → audience `Admin`, `/me` → audience `Owner`), sans
test de rôle dans les handlers ; les champs réservés sont filtrés dans les
requêtes SQL (R3). Recherche des membres par champ texte court via une table
FTS5 trigram à contenu externe (R5). Filtres de liste en nombre variable par
une requête sqlc statique qui parcourt un JSON avec `json_each` (R6). Une
opération dédiée permet de modifier les champs d'une adhésion même payée
(R8). Détails : [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 6.0 + React 19 (frontend)

**Primary Dependencies**: aucune nouvelle. Existantes : huma/v2 (query
`[]string`, `pattern`, `maxItems`), sqlc (moteur SQLite, `json_each` vérifié),
goose, `internal/textfold`, `internal/season` (`IsCurrent`). Frontend :
base-ui (`Dialog`, `RadioGroup`, `Select`, `Menu`), react-hook-form, TanStack
Query, `openapi-fetch`, lucide-react (`ListPlus`, `ArrowUp`, `ArrowDown`,
`Archive`, `ArchiveRestore`, `Trash2`).

**Storage**: SQLite ; migration `00011_custom_fields.sql` : `custom_fields`,
`custom_field_choices`, `member_field_values` (+ `member_field_values_fts`
et ses triggers), `membership_field_values`
([data-model.md](./data-model.md)).

**Testing**: `go test` : tests en table de `internal/customfield` ; tests
d'API httptest (CRUD des définitions, écriture par audience, fuite des champs
réservés, obligatoires, filtres, recherche, cascade et FTS) ; matrice
d'autorisation automatique ; migration sur base `00010` peuplée. Vitest
(`customfields/form.ts`, filtres d'URL). Playwright + axe
(`custom-fields.spec.ts`) ([R12](./research.md#r12-tests)).

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` ;
navigateurs récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: recherche par fragment de numéro de licence < 2 s sur
2 000 fiches (SC-004) : FTS trigram indexé ; listes toujours chargées
entières puis triées en Go (005 › R7) avec au plus 10 sous-requêtes `EXISTS`
indexées par filtre

**Constraints**: aucune définition ni valeur réservée transmise hors audience
`Admin` ; pas de SQL dynamique ; aucune valeur ni libellé de champ dans les
journaux ; commentaires sqlc en ASCII ; ≤ 30 champs actifs par portée ;
≤ 10 filtres par liste

**Scale/Scope**: une association, quelques milliers de fiches ; 10
opérations ajoutées, 13 modifiées ; 1 écran nouveau, 9 écrans modifiés,
4 composants partagés

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution v1.8.0.

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Définitions = données de l'instance, sans locataire | ✅ |
| II. Authentification partout | Aucune opération publique ; 10 opérations ajoutées sous session | ✅ |
| III. Deux rôles | Rôle déclaré par `register` pour chaque opération (8 `admin`, 2 `member`) ; la visibilité dépend de l'espace de l'opération et non d'un test de rôle dans les handlers (R3) ; matrice d'autorisation automatique | ✅ |
| IV. Un shell, un design system | Section intégrée à « Paramètres » ; composants `ui/*` et base-ui (`Dialog`, `RadioGroup`, `Select`, `Menu`) ; tokens (`--warn` pour les fiches incomplètes, `--muted` pour « Non renseigné ») ; `ConfirmDialog` pour la suppression ; déplacement par boutons accessibles au clavier ; `fieldset`/`legend`, aide et erreurs reliées ; 320 px | ✅ |
| V. Binaire unique en Go pur | Aucune dépendance ; migration embarquée | ✅ |
| VI. API typée et SQL généré | Bornes et `pattern` déclarés dans huma ; règles de type dans un domaine Go unique, nombre converti par le serveur (pas de règle dans le navigateur) ; requêtes sqlc statiques, filtres variables via `json_each` (R6) ; recherche via FTS5 à contenu externe trigram synchronisée par triggers, texte court seul indexé (une catégorie par table) ; unicité des libellés par `textfold` | ✅ |
| VII. Sécurité et RGPD | Champs réservés filtrés dans la requête en audience `Owner`, même `422` pour un champ inconnu ou réservé (pas d'oracle) ; isolation des fiches de 005/006 réutilisée ; valeurs supprimées avec la fiche ; journaux sans valeur ni libellé ; export RGPD futur informé (spec) | ✅ |
| Contraintes techniques | Écritures en `BEGIN IMMEDIATE` (`WriteTx`), valeurs écrites dans la même transaction que la fiche ou l'adhésion | ✅ |
| Conventions de langue | Identifiants en anglais (`custom_fields`, `visibility`, `editable`) ; textes, erreurs et journaux en français | ✅ |
| Workflow | Nouvelle migration (append-only) ; matrice étendue ; contrôles de [quickstart.md](./quickstart.md) | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance :

1. **Fuite des champs réservés** : les requêtes de l'audience `Owner` portent
   `visibility <> 'admin'` ; un test compare la réponse JSON brute de
   `get-my-member`, `get-my-membership` et `list-my-custom-fields` à
   l'identifiant et au libellé d'un champ réservé.
2. **Pas d'effacement implicite** : un champ absent de `customFields` n'est
   jamais modifié ; test : un membre enregistre sa fiche, la valeur réservée
   saisie par un administrateur est intacte.
3. **Type immuable** : refusé (`409`) dès qu'une valeur existe, ce qui garde
   la colonne `value_*` cohérente avec le type sans contrôle en base.
4. **Index FTS cohérent** : suppression d'une fiche, d'un champ, et
   effacement d'une valeur ; test `members_fts`-like avec
   `INSERT INTO member_field_values_fts(member_field_values_fts) VALUES
   ('integrity-check')`.
5. **Champs archivés** : exclus des sorties, filtres et recherche (jointure
   `archived_at IS NULL`) ; valeurs conservées et restituées au
   rétablissement (SC-005).

## Project Structure

### Documentation (this feature)

```text
specs/011-custom-fields/
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
├── customfield/                        # nouveau : domaine pur, sans base
│   ├── definition.go                   # Scope, Type, Visibility, Definition, ValidateDefinition, choix
│   ├── value.go                        # Input, Value, NormalizeValue (virgule décimale, bornes)
│   ├── apply.go                        # Audience, CanRead/CanWrite, Apply (plan d'écriture, obligatoires)
│   ├── filter.go                       # ParseFilter, JSON pour json_each
│   └── *_test.go
├── database/
│   ├── migrations/00011_custom_fields.sql   # nouvelle
│   ├── queries/custom_fields.sql       # définitions, choix, valeurs, compteurs (ASCII)
│   ├── queries/members.sql             # ListMembers : recherche par valeur, filtres
│   ├── queries/memberships.sql         # ListMemberships : filtres
│   └── sqlc/                           # régénéré
└── api/
    ├── custom_fields.go                # 7 opérations admin + list-my-custom-fields
    ├── custom_field_values.go          # chargement par audience, écriture d'un plan, FieldValue
    ├── members_admin.go, members_me.go # customFields en entrée et en sortie, filtres
    ├── memberships_admin.go            # create/validate + update-membership-fields, filtres
    ├── memberships_me.go               # create-my-membership + update-my-membership-fields
    ├── api.go                          # registerCustomFields
    └── *_test.go                       # dont custom_fields_leak_test.go

web/
├── app/
│   ├── routes.ts                       # parametres/champs
│   ├── routes/app/admin/SettingsPage.tsx              # section « Champs personnalisés »
│   ├── routes/app/admin/settings-custom-fields.tsx    # nouveau
│   ├── customfields/                   # CustomFieldInputs, CustomFieldValues, CustomFieldFilters, FieldDialog, form.ts
│   ├── members/AdminMemberForm.tsx, MyMemberForm.tsx
│   ├── routes/app/admin/members.tsx, memberships.tsx, membership-detail.tsx, membership-new.tsx
│   ├── routes/app/member/membership-request.tsx, membership-detail.tsx
│   ├── lib/member-filters.ts, lib/membership-filters.ts   # paramètre d'URL « champ »
│   ├── api/customFields.ts             # requêtes et mutations
│   └── api/openapi.json, api/schema.d.ts                  # régénérés
└── e2e/custom-fields.spec.ts           # nouveau

internal/tools/e2eseed                  # champs d'exemple (licence fédérale)
```

**Structure Decision**: structure existante (API Go + SPA dans `web/`). Un
paquet pur nouveau `internal/customfield`, sur le modèle de `internal/member`
et `internal/membership`. Les handlers existants des fiches et des adhésions
appellent deux fonctions partagées de `custom_field_values.go` (lire pour une
audience, appliquer un plan dans la transaction), pour ne pas répartir la
logique dans 13 opérations.

## Complexity Tracking

Aucune violation de la constitution v1.8.0. Écarts de complexité retenus :

| Élément | Pourquoi | Alternative plus simple écartée |
| --- | --- | --- |
| Valeurs à colonnes typées (EAV) plutôt qu'une colonne JSON | Clés étrangères vers les choix (choix utilisé non retirable), `CHECK` par type, index FTS sur le seul texte court, filtres en SQL | Colonne JSON par fiche : ni clé étrangère, ni FTS à contenu externe, filtres en Go sur toutes les fiches |
| Filtres via `json_each` dans une requête statique | Nombre variable de filtres sans SQL construit à l'exécution (principe VI) | Concaténation de conditions : interdite ; nombre fixe de paramètres : borne arbitraire, requête illisible |
| Opérations `update-membership-fields` et `update-my-membership-fields` | Saisir une information sur une adhésion payée sans toucher au tarif ni au montant, ce que `update-membership` interdit volontairement | Élargir `update-membership` : mélange correction financière et saisie, préconditions contradictoires |
| Table FTS supplémentaire `member_field_values_fts` | Retrouver un membre par son numéro de licence (FR-024) sans dénormaliser `members` | Ajouter les valeurs à `members_fts` : impossible pour une table à contenu externe sur plusieurs tables |
