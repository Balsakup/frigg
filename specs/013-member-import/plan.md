# Implementation Plan: Importation des membres

**Branch**: `013-member-import` | **Date**: 2026-11-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/013-member-import/spec.md`

## Summary

Les administrateurs importent des fiches membres « Sans compte » depuis un
fichier **CSV** : dépôt, correspondance des colonnes proposée d'après les
intitulés (champs personnalisés de membre compris), aperçu ligne par ligne
(à importer, en erreur, doublon), import tout ou rien, rapport et
téléchargement des lignes rejetées réimportables, modèle de fichier.

Approche : **aucun état serveur et aucune migration** (R1). Le navigateur
garde le fichier et le renvoie à chaque étape. Le serveur le relit et le
revalide entièrement à chaque fois, puis l'oublie. Deux paquets purs sont
ajoutés. `internal/csvfile` détecte l'encodage et le séparateur et lit et
écrit des CSV sûrs (R2, R3). `internal/memberimport` porte la
correspondance, la validation des lignes et la classification des doublons
(R4 à R7). Ce dernier réutilise sans les dupliquer `member.Validate`,
`member.IdentityKey` et `customfield.NormalizeValue`. L'import écrit dans une
seule `WriteTx`, où la règle de doublon et l'empreinte des champs
personnalisés sont revérifiées (R6, R8, R9). Détails :
[research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 6.0 + React 19 (frontend)

**Primary Dependencies**: aucune nouvelle. `encoding/csv` et `crypto/sha256`
(bibliothèque standard) ; `golang.org/x/text/encoding/charmap` (module
`golang.org/x/text`, déjà dépendance directe) ; existants : huma/v2
(`MultipartFormFiles`, `StreamResponse`), sqlc, `internal/member`,
`internal/customfield`, `internal/textfold`, `internal/season`. Frontend :
base-ui (`Select`, `RadioGroup`, `Collapsible`), TanStack Query
(`useMutation`), `openapi-fetch` (`bodySerializer`, `parseAs: "blob"`),
lucide-react (`Upload`, `Download`, `FileSpreadsheet`).

**Storage**: SQLite, aucune migration. Écritures dans `members` et
`member_field_values` existantes ; une requête sqlc ajoutée,
`ListMemberIdentities` ([data-model.md](./data-model.md)). Aucun fichier
écrit sur disque : le multipart reste en mémoire
(`humago.MultipartMaxMemory = UploadMaxBody`).

**Testing**: `go test` : tests en table de `internal/csvfile` et
`internal/memberimport` ; tests d'API httptest (parcours, limites, `409`
d'empreinte, doublon concurrent, atomicité, réimport des rejets, journaux,
aucun e-mail) ; matrice d'autorisation automatique. Vitest (`MappingStep` :
cible unique, champ exigé). Playwright + axe : `member-import.spec.ts`
([R13](./research.md#r13-tests)).

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` ;
navigateurs récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: aperçu de 2 000 lignes en moins de 2 s, import de
2 000 fiches en une transaction de moins de 2 s ; parcours complet de
500 lignes en moins de 5 min (SC-001), le temps étant celui de
l'administrateur et non de la machine

**Constraints**: aucune donnée personnelle conservée hors des fiches créées
(fichier jamais stocké) ni écrite dans les journaux ; pas de SQL dynamique ;
pas de nouvelle route publique ; CSV produits protégés contre l'injection de
formule ; commentaires sqlc en ASCII ; aucune règle de validation dupliquée
côté client

**Scale/Scope**: une association, quelques imports à la mise en service puis
occasionnels ; 2 000 lignes, 100 colonnes et 5 Mo par fichier ; 5 opérations
`admin` ajoutées, 1 modifiée (export des inscrits : `csvCell` → `csvfile`) ;
1 écran nouveau (assistant en 4 étapes), 1 action ajoutée à « Membres »

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution v1.8.0.

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Fiches de l'instance, sans locataire | ✅ |
| II. Authentification partout | 5 opérations sous session, rôle admin ; aucune route publique ajoutée | ✅ |
| III. Deux rôles | Rôle déclaré par `register(api, auth.Admin, …)` pour chaque opération ; matrice automatique étendue ; pas de test de rôle dans les handlers | ✅ |
| IV. Un shell, un design system | Page dans la rubrique « Membres » existante ; `ui/*` (`Alert`, `Badge`, `Button`, `Select`, `RadioGroup`, `Table`, `Pagination`), tokens `--ok`, `--warn`, `--danger`, `--info`, `--muted` ; icônes via `Icon` ; RGAA : étapes `aria-current`, focus sur le titre d'étape, `aria-live` des compteurs, dépôt par `input` étiqueté ; 320 px ; aucune action destructive, donc pas de dialogue de confirmation requis (l'aperçu obligatoire précède toute écriture) | ✅ |
| V. Binaire unique en Go pur | Aucune dépendance ajoutée (`x/text` déjà présent) ; aucune migration ; aucun fichier lu ou écrit hors base | ✅ |
| VI. API typée et SQL généré | Contrat huma (multipart typé, sorties typées, CSV en `StreamResponse`) ; validation par le serveur seul, frontend sans règle propre ; requête sqlc statique ; comparaisons d'intitulés, de choix et d'identités par `internal/textfold` ; pas de colonne repliée | ✅ |
| VII. Sécurité et RGPD | Fichier jamais conservé (FR-021 par construction) ; multipart en mémoire ; journaux sans donnée personnelle ni nom de fichier ; CSV de sortie neutralisés (`SafeCell`) ; limites de taille avant lecture (`MaxBodyBytes`, contrôle de `file.Size`) ; données collectées = celles de la fiche (005) | ✅ |
| Contraintes techniques | Écriture en une `WriteTx` (`BEGIN IMMEDIATE`) ; aucun e-mail ; lecture de l'aperçu sur le pool de lecture | ✅ |
| Conventions de langue | Identifiants anglais (`memberimport`, `fieldsVersion`, `ready`/`error`/`duplicate`) ; messages, intitulés du modèle et journaux en français | ✅ |
| Workflow | Contrôles de [quickstart.md](./quickstart.md) ; aucune migration modifiée | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance :

1. **Une seule règle de doublon** : `memberimport.Classify` et
   `checkDuplicate` (005) utilisent `member.IdentityKey` et la date ISO.
   Un test croisé vérifie qu'une ligne déclarée doublon à l'import serait
   signalée doublon par la création manuelle, et inversement.
2. **Revalidation dans la transaction** : identités et empreinte sont relues
   dans la `WriteTx`, jamais reprises de l'aperçu. Un test vérifie le doublon
   concurrent et un autre le `409` d'empreinte.
3. **Neutralisation aller-retour** : `csvfile.Decode` retire exactement ce
   que `SafeCell` ajoute, pas plus. Une apostrophe suivie d'une lettre est
   conservée (« 'Ohana »).
4. **Champs obligatoires** : l'import n'appelle pas `customfield.Apply`
   (qui les impose). Un test vérifie qu'une ligne sans valeur obligatoire
   est importée et comptée.
5. **Déplacement de `csvCell`** : l'export des inscrits (012) garde une
   sortie identique, vérifiée par son test existant.

## Project Structure

### Documentation (this feature)

```text
specs/013-member-import/
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
├── csvfile/                            # nouveau : paquet pur
│   ├── decode.go                       # encodage, séparateur, Decode, limites, erreurs
│   ├── write.go                        # SafeCell, NewWriter (BOM, ;)
│   └── *_test.go
├── memberimport/                       # nouveau : paquet pur, sans base
│   ├── mapping.go                      # Target, normalisation des intitulés, Suggest, ValidateMapping
│   ├── row.go                          # ParseDate, ParseRow, conversions des champs personnalisés
│   ├── classify.go                     # Identity, Classify, Result, Status
│   ├── template.go                     # TemplateHeaders, RejectedHeaders, RejectedRecord
│   ├── fingerprint.go                  # Fingerprint
│   └── *_test.go
├── database/
│   ├── queries/members.sql             # + ListMemberIdentities (ASCII)
│   └── sqlc/                           # régénéré
└── api/
    ├── members_import.go               # 5 opérations, lecture multipart, traduction des erreurs
    ├── members_import_test.go
    ├── registrations_admin.go          # csvCell → csvfile.SafeCell / NewWriter
    ├── api.go                          # registerMemberImport (avant /admin/members/{id})
    └── access_test.go                  # matrice : 5 opérations admin

web/
├── app/
│   ├── routes.ts                       # membres/importer avant :id
│   ├── routes/app/admin/members.tsx    # action « Importer »
│   ├── routes/app/admin/member-import.tsx   # nouveau : assistant
│   ├── members/import/                 # ImportSteps, FileStep, MappingStep, PreviewStep, ReportStep, download.ts
│   ├── routes/app/admin/event-detail.tsx    # utilise download.ts
│   ├── api/memberImport.ts             # nouveau
│   └── api/openapi.json, api/schema.d.ts    # régénérés
└── e2e/
    ├── member-import.spec.ts           # nouveau
    └── fixtures/member-import.csv      # fichier d'exemple (Windows-1252)
```

**Structure Decision**: structure existante (API Go + SPA dans `web/`). Deux
paquets purs nouveaux. `csvfile` est un outil générique, que réutilise
l'export des inscrits. `memberimport` porte les règles propres à l'import et
délègue toute validation de valeur aux paquets `member` et `customfield`.
L'API se limite à lire le multipart, appeler ces paquets, lire et écrire la
base et traduire les erreurs. Côté frontend, un assistant à état local,
sans cache de requête, car le fichier n'existe que dans le navigateur.

## Complexity Tracking

Aucune violation de la constitution v1.8.0. Écarts de complexité retenus :

| Élément | Pourquoi | Alternative plus simple écartée |
| --- | --- | --- |
| Renvoi du fichier à chaque étape (4 requêtes multipart) | Aucun stockage de données personnelles, ni table, ni balayage, ni contrôle de propriétaire ; revalidation complète garantie à l'import | Stocker le fichier avec une table d'imports et un balayage horaire : plus de code et une copie de données personnelles sur disque |
| Opération `analyze` distincte de `preview` | Afficher la correspondance avant toute validation de ligne, avec une proposition calculée par le serveur | Tout calculer dans `preview` avec une correspondance vide : l'aperçu initial ne contiendrait que des erreurs « colonne non associée » |
| Empreinte `fieldsVersion` | Détecter un champ modifié entre l'aperçu et l'import sans état serveur (FR-017) | Colonne de version sur les champs personnalisés : migration pour un seul usage |
