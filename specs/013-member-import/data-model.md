# Modèle de données : importation des membres

## Persistance

**Aucune migration.** L'importation est sans état côté serveur
([research.md › R1](./research.md#r1-importation-sans-état-côté-serveur)) :
elle n'écrit que dans les tables existantes `members` (005) et
`member_field_values` (011), avec les mêmes contraintes.

| Table | Écriture | Valeurs propres à l'import |
| --- | --- | --- |
| `members` | `CreateMember` existante | `account_id` NULL, `is_self` 0, `created_at` = `updated_at` = instant de l'import |
| `member_field_values` | `UpsertMemberFieldValue` existante | une ligne par valeur non vide |

**Requête sqlc ajoutée** (`internal/database/queries/members.sql`,
commentaire ASCII) :

```sql
-- name: ListMemberIdentities :many
-- Identites de toutes les fiches, pour la detection des doublons a l'import.
SELECT id, first_name, last_name, birth_date FROM members;
```

Les définitions de champs personnalisés sont lues par `loadDefinitions`
existant (portée membre, audience administrateur, champs actifs).

## Objets en mémoire (paquets purs)

### `internal/csvfile`

| Type / fonction | Rôle |
| --- | --- |
| `Table{Headers []string; Rows []Row; Encoding string; Separator rune}` | fichier décodé |
| `Row{Line int; Cells []string; Extra bool}` | `Line` : numéro d'enregistrement (intitulés = 1) ; `Extra` : cellules non vides au-delà de l'en-tête |
| `Decode(data []byte, limits Limits) (Table, error)` | encodage, séparateur, lecture, lignes vides ignorées, apostrophe de neutralisation retirée |
| `Limits{MaxBytes, MaxRows, MaxColumns}` | 5 Mio, 2 000, 100 |
| `ErrEmpty`, `ErrBinary`, `ErrMalformed{Line}`, `ErrTooLarge`, `ErrTooManyRows`, `ErrTooManyColumns`, `ErrNoData` | refus du fichier, traduits en `422 form.file` |
| `SafeCell(string) string` | neutralisation de formule (déplacée de `api.csvCell`) |
| `NewWriter(io.Writer) *csv.Writer` | BOM UTF-8, `;` |

### `internal/memberimport`

**Target** (chaîne) : `""` (ne pas importer), `firstName`, `lastName`,
`birthDate`, `email`, `phone`, `addressLine1`, `addressLine2`, `postalCode`,
`city`, `field:<id>`.

| Type / fonction | Rôle |
| --- | --- |
| `Suggest(headers []string, defs []customfield.Definition) []Target` | proposition (R4) |
| `ValidateMapping(m []Target, columns int, defs) error` | longueur, cibles connues et uniques, trois champs exigés ; erreurs par index |
| `ParseDate(string) (season.Date, error)` | formats de R5 |
| `ParseRow(row csvfile.Row, m []Target, defs, today) Parsed` | une ligne |
| `Parsed{Line; Input member.Clean; Fields map[int64]customfield.Value; Errors []CellError; MissingRequired []int64}` | résultat d'une ligne |
| `CellError{Column int; Message string}` | `Column` : index de la colonne (−1 : ligne entière, cellules excédentaires) |
| `Identity{ID int64; FirstName, LastName string; BirthDate string}` | fiche existante |
| `Classify(rows []Parsed, existing []Identity) []Result` | statut et doublon (R6) |
| `Result{Parsed; Status Status; DuplicateOfMember int64; DuplicateOfLine int}` | ligne classée |
| `Status` | `ready`, `error`, `duplicate` |
| `Fingerprint(defs) string` | version des champs (R8) |
| `TemplateHeaders(defs) []string` | intitulés du modèle (R7) |
| `RejectedHeaders(t csvfile.Table) []string` / `RejectedRecord(r Result, t) []string` | colonnes d'origine + « Raison du rejet » |

### Règles de validation (rappel)

| Donnée | Règle | Source |
| --- | --- | --- |
| Prénom, nom | 1 à 80 caractères après réduction des espaces | 005 FR-001 (`member.Validate`) |
| Date de naissance | formats R5 ; ni future ni antérieure à 120 ans | 005 FR-001 |
| Adresse électronique, téléphone | formats de la fiche ; téléphone normalisé | 005 FR-002 |
| Adresse postale | tout ou rien ; code postal sur 5 chiffres ; complément seulement avec une adresse | 005 FR-002 |
| Champ personnalisé | type et limites du champ, sans troncature | 011 FR-003, FR-014 |
| Champ personnalisé obligatoire | non exigé ; compté dans `missingRequired` | 013 FR-013 |
| Doublon | prénom + nom repliés (`member.IdentityKey`) + date | 005 FR-005 |

## États d'une ligne

```text
                ┌── erreurs de cellule ou cellules excédentaires ──► error
ligne non vide ─┤
                └── valide ─┬── identité en base ─────────────────► duplicate (fiche)
                            ├── identité sur une ligne ready plus haut ► duplicate (ligne)
                            └── sinon ────────────────────────────► ready
```

À l'import, la classification est recalculée dans la transaction : une
ligne `ready` de l'aperçu peut devenir `duplicate` (compteur
`newDuplicates`), jamais l'inverse sauf suppression d'une fiche entre-temps
(la ligne est alors importée, conformément à l'état courant).

## Effets d'un import confirmé

- N fiches « Sans compte » créées (N = lignes `ready` à la confirmation),
  avec leurs valeurs de champs personnalisés ; identifiants contigus.
- Aucune autre table modifiée ; aucun e-mail, compte, invitation ni
  adhésion.
- Une ligne de journal `member.import` (R10).
