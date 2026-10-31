# Implementation Plan: Gestion des comptes par les administrateurs

**Branch**: `002-admin-account-management` | **Date**: 2026-10-31 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/002-admin-account-management/spec.md`

## Summary

Compléter la rubrique « Comptes » de l'espace administration : liste paginée
des comptes avec recherche par adresse (FTS5) et filtres par rôle et état,
fiche de compte, désactivation et réactivation, changement de rôle,
suppression définitive et envoi d'un lien de réinitialisation. La
désactivation s'appuie sur l'époque d'authentification de 001 pour fermer les
sessions ; le dernier administrateur actif est protégé par des écritures
conditionnelles sérialisées ; aucune nouvelle dépendance. Approche détaillée
dans [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 5.9 + React 19 (frontend)

**Primary Dependencies**: inchangées depuis 001 : huma/v2 (humago), scs/v2 +
sqlite3store, goose/v3, modernc.org/sqlite (FTS5), sqlc 1.31 ; React Router
v8, TanStack Query, openapi-fetch, openapi-react-query, react-hook-form,
base-ui (`Select`, `Dialog`), lucide-react, Tailwind v4. Pas de bibliothèque
de tableau ([research.md › R12](./research.md#r12-interface--tableau-sans-bibliothèque)).

**Storage**: SQLite ; `00001_init.sql` modifiée (projet jamais déployé,
constitution 1.4.0) : colonne `accounts.disabled_at`, table virtuelle FTS5
`accounts_fts` (trigram) et ses triggers

**Testing**: `go test` : httptest (chaîne complète, sessions) pour la
désactivation et le retrait de droits, humatest pour le schéma, matrice
d'autorisation étendue, test de concurrence sur le dernier administrateur ;
Vitest (correspondance URL ↔ filtres, `Pagination`) ; Playwright + axe

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` ; navigateurs
récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: liste et recherche perçues comme immédiates jusqu'à
2 000 comptes (SC-001) ; désactivation effective à la requête suivante du
compte visé (SC-002)

**Constraints**: refus par défaut (principe II) ; écritures sérialisées
(une connexion, `BEGIN IMMEDIATE`) ; aucune adresse électronique dans les
journaux d'action ; RGAA

**Scale/Scope**: quelques centaines à 2 000 comptes ; 2 écrans (liste enrichie,
fiche) ; 7 opérations d'API `admin` ; 3 composants d'interface réutilisables

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Aucune notion de locataire ajoutée | ✅ |
| II. Authentification partout | 7 opérations `admin`, aucune opération publique ajoutée ; un compte désactivé perd sa session côté serveur (middleware) | ✅ |
| III. Deux rôles | Changement de rôle via la même fonction `Allows` ; niveau d'accès déclaré par `register` pour chaque opération ; aucune vérification de rôle ad hoc (le garde-fou « acteur toujours admin actif » relit le compte en transaction et réutilise `Allows`) | ✅ |
| IV. Un shell, un design system | Pages dans le shell, `Table`/`Pagination`/`Select` sur tokens et base-ui, dialogues de confirmation pour désactiver, changer le rôle, supprimer ; 320 px ; Playwright + axe | ✅ |
| V. Binaire unique en Go pur | FTS5 fourni par modernc ; migration goose embarquée | ✅ |
| VI. API typée et SQL généré | Filtres et pagination validés par huma ; requêtes sqlc (paramètres typés par `CAST`) ; recherche via FTS5 à contenu externe `trigram remove_diacritics 1` (champ d'identité court, 1.4.0) avec triggers, sans colonne repliée ; minimum de 3 caractères indiqué dans l'interface ; expression MATCH passée en paramètre | ✅ |
| VII. Sécurité | Réponses de connexion et de réinitialisation identiques pour un compte désactivé ; jetons supprimés à la désactivation ; effacement RGPD (compte, invitations reçues, e-mails en file) ; journaux sans adresse ni jeton | ✅ |
| Contraintes techniques | Aucune dépendance ajoutée ; e-mail de réinitialisation via la file existante ; nouveau limiteur `AdminReset` | ✅ |
| Conventions de langue | Chemins API et valeurs en anglais (`active`, `disabled`) ; chemins et paramètres visibles du SPA en français (`/admin/comptes`, `statut=desactive`) ; messages en français | ✅ |
| Workflow | Matrice d'autorisation étendue ; `00001` modifiée, autorisé tant que le projet n'est pas déployé (1.4.0) ; contrôles de [quickstart.md](./quickstart.md) | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance :
(1) `00001_init.sql` est modifiée : à vérifier qu'aucune instance n'a été
déployée au moment de la fusion, et recréer les bases de développement ;
(2) `GetValidToken` change de forme (jointure) : régénérer sqlc et vérifier les
tests de 001 ; (3) la recherche trigram exige 3 caractères : l'interface
l'indique (FR-003).

## Project Structure

### Documentation (this feature)

```text
specs/002-admin-account-management/
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
├── auth/
│   └── ratelimit.go             ~ limiteur AdminReset
├── database/
│   ├── fts.go                   + FTSTrigramQuery (découpage, mots ≥ 3, guillemets)
│   ├── fts_test.go              +
│   ├── migrations/
│   │   └── 00001_init.sql       ~ disabled_at, accounts_fts (trigram), triggers
│   ├── queries/
│   │   ├── accounts.sql         ~ ListAccounts, CountAccounts, DisableAccount,
│   │   │                          EnableAccount, SetAccountRole, DeleteAccount
│   │   ├── tokens.sql           ~ GetValidToken ignore les comptes désactivés ;
│   │   │                          DeleteAllUnusedTokens
│   │   ├── invitations.sql      ~ DeleteInvitationsByEmail
│   │   └── emails.sql           ~ DeleteEmailsByRecipient
│   └── sqlc/                    ~ régénéré
├── api/
│   ├── api.go                   ~ middleware : compte désactivé → 401 ;
│   │                              enregistrement registerAccounts
│   ├── auth.go                  ~ login et request-password-reset : comptes désactivés
│   ├── accounts.go              + 7 opérations admin, garde-fous, journalisation
│   └── accounts_test.go         + parcours, garde-fous, concurrence, effacement
│                                  (access_test.go parcourt déjà l'OpenAPI :
│                                  les 7 opérations y entrent sans modification)
└── tools/e2eseed/main.go        ~ comptes de test de 002

web/app/
├── api/
│   ├── client.ts                ~ 403 sur /api/admin/* → invalidation de la session
│   ├── openapi.json, schema.d.ts ~ régénérés
├── ui/
│   ├── Table.tsx                + extrait du tableau des invitations
│   ├── Pagination.tsx           + (+ test Vitest)
│   ├── Select.tsx               + base-ui Select habillé
│   └── ConfirmDialog.tsx        ~ props tone et confirmDisabled
├── lib/
│   └── account-filters.ts       + URL ↔ paramètres d'API (+ test Vitest)
├── shell/AppShell.tsx           ~ détection admin → member, retour à l'accueil
├── routes.ts                    ~ route admin/comptes/:id
└── routes/app/admin/
    ├── accounts.tsx             ~ section Comptes + Table pour les invitations
    └── account-detail.tsx       + fiche et dialogues d'action

web/e2e/
└── admin-accounts.spec.ts       + parcours US1–US5 + axe
```

**Structure Decision**: même organisation que 001 (un module Go, SPA dans
`web/`). Les opérations de comptes vivent dans `internal/api/accounts.go`, à
côté de `invitations.go`. `FTSTrigramQuery` est placé dans `internal/database`
car il produit une expression propre au moteur SQL ; il servira aux futures
recherches (membres, recherche globale). `internal/textfold` n'est toujours
pas nécessaire (aucune comparaison de texte saisi en Go).

## Complexity Tracking

Aucune violation de la constitution à justifier.
