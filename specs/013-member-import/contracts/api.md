# Contrat d'API : importation des membres

Mêmes conventions que [005 › api.md](../../005-member-management/contracts/api.md) :
préfixe `/api`, camelCase, `problem+json` en français avec
`errors[].location` sur les `422`, `401` sans session, `403` rôle
insuffisant. Le schéma OpenAPI produit par `frigg openapi` fait foi. Tag
OpenAPI `members`. Les cinq opérations exigent le rôle **admin**.

Les `POST` sont en `multipart/form-data`, `MaxBodyBytes = UploadMaxBody`
(5 Mio + 64 Kio). Le fichier n'est jamais conservé
([research.md › R1](../research.md#r1-importation-sans-état-côté-serveur)).

## Types

```text
ImportTarget = "" | "firstName" | "lastName" | "birthDate" | "email" | "phone"
             | "addressLine1" | "addressLine2" | "postalCode" | "city"
             | "field:<id>"                     // champ personnalisé de membre actif

ImportColumn {
  index: int,                 // 0..99
  header: string,             // intitulé tel que lu ; "" si vide
  label: string,              // intitulé, ou « Colonne 3 » s'il est vide ou en double
  samples: string[]           // ≤ 3 premières valeurs non vides, ≤ 80 caractères chacune
}

ImportTargetOption {
  target: ImportTarget,       // sans ""
  label: string,              // « Prénom », « Numéro de licence »…
  requiredForImport: bool,    // true pour firstName, lastName, birthDate
  fieldType: string|null      // type 011 pour un champ personnalisé
}

ImportFile {                  // fichier lu
  encoding: "utf-8" | "windows-1252",
  separator: ";" | "," | "\t",
  rowCount: int               // lignes de données non vides
}

ImportCellError {
  column: int|null,           // index ; null : ligne entière
  columnLabel: string|null,
  message: string             // français, explique la correction
}

ImportRow {
  line: int,                  // numéro de ligne dans le tableur (intitulés = 1)
  status: "ready" | "error" | "duplicate",
  firstName: string, lastName: string,   // valeurs brutes des colonnes associées, pour l'affichage
  birthDate: string,
  errors: ImportCellError[],             // vide sauf "error"
  duplicateOfMember: { id: int, label: string } | null,   // « Dupont Marie (12/03/2012) »
  duplicateOfLine: int | null,
  missingRequired: int[]                 // champs obligatoires sans valeur (ready seulement)
}

ImportMissing { fieldId: int, label: string, count: int }

ImportCounts { ready: int, errors: int, duplicates: int }
```

## `GET /admin/members/import/template` — `get-member-import-template`

`200 text/csv; charset=utf-8`,
`Content-Disposition: attachment; filename="modele-import-membres.csv"`.
BOM UTF-8, séparateur `;`, une ligne d'intitulés ([R7](../research.md#r7-modèle-de-fichier)).

## `POST /admin/members/import/analyze` — `analyze-member-import`

Formulaire : `file` (obligatoire).

`200` :

```text
ImportAnalysis {
  file: ImportFile,
  columns: ImportColumn[],
  suggested: ImportTarget[],          // même longueur que columns
  targets: ImportTargetOption[],      // champs de la fiche, puis champs personnalisés actifs dans l'ordre défini
  fieldsVersion: string
}
```

`422` sur `form.file` : fichier vide, illisible (octet NUL, guillemet non
refermé — ligne indiquée), sans ligne de données, plus de 5 Mo, plus de
2 000 lignes, plus de 100 colonnes. Messages :

- « Ce fichier n'est pas un fichier CSV lisible. Dans votre tableur,
  choisissez « Enregistrer sous » puis le format CSV. »
- « Guillemet non refermé à la ligne 18 : vérifiez cette ligne dans votre
  tableur. »
- « Le fichier ne contient aucune ligne de membre sous la ligne
  d'intitulés. »
- « Le fichier dépasse 5 Mo. » / « Le fichier dépasse 2 000 lignes :
  découpez-le en plusieurs fichiers. » / « Le fichier dépasse 100
  colonnes. »

## `POST /admin/members/import/preview` — `preview-member-import`

Formulaire : `file`, `mapping` (chaîne JSON : `ImportTarget[]`, ≤ 4 Kio).

`200` :

```text
ImportPreview {
  file: ImportFile,
  counts: ImportCounts,
  missingRequired: ImportMissing[],   // parmi les lignes ready ; vide si aucun
  incompleteCount: int,               // lignes ready auxquelles manque au moins une valeur obligatoire
  rows: ImportRow[],                  // toutes les lignes non vides, dans l'ordre du fichier
  fieldsVersion: string
}
```

`422` : erreurs de `form.file` (comme `analyze`) ; erreurs de
correspondance sur `form.mapping` (JSON invalide, longueur différente du
nombre de colonnes) ou `form.mapping[i]` (cible inconnue, champ archivé,
cible déjà associée à la colonne j) ; champ exigé absent sur
`form.mapping` : « Associez une colonne au champ « Date de naissance ». »

## `POST /admin/members/import` — `run-member-import`

Formulaire : `file`, `mapping`, `fieldsVersion` (32 caractères
hexadécimaux), `expectedReady` facultatif (nombre de lignes `ready`
affichées par l'aperçu, 0 à 2 000).

`201` :

```text
ImportReport {
  created: int,
  errors: int,
  duplicates: int,
  newDuplicates: int,                 // max(expectedReady - created, 0) : lignes ready à l'aperçu devenues doublons
  missingRequired: ImportMissing[],
  incompleteCount: int
}
```

- `409` « Les champs personnalisés ont changé depuis l'aperçu. L'aperçu a
  été recalculé. » si l'empreinte diffère.
- `422` comme `preview` ; `422` sans `location` « Aucune ligne à
  importer. » si aucune ligne n'est `ready` à la confirmation.
- Tout ou rien : en cas d'erreur interne, `500` et aucune fiche créée.

## `POST /admin/members/import/rejected` — `export-member-import-rejected`

Formulaire : `file`, `mapping`.

`200 text/csv; charset=utf-8`,
`Content-Disposition: attachment; filename="lignes-rejetees.csv"`. BOM
UTF-8, séparateur `;`. Intitulés d'origine (dans l'ordre du fichier) puis
« Raison du rejet » ; une ligne par ligne `error` ou `duplicate`, cellules
d'origine passées par `SafeCell`, raison = messages joints par « ; »
préfixés de l'intitulé de colonne, ou « Doublon de la fiche Dupont Marie »
/ « Doublon de la ligne 12 ». Mêmes `422` que `preview`. La colonne
« Raison du rejet » est proposée en « Ne pas importer » à la réimportation
(aucun synonyme).

## Autorisation

| Opération | Anonyme | Membre | Admin |
| --- | --- | --- | --- |
| les cinq ci-dessus | 401 | 403 | ✅ |

Ajoutées à la matrice automatique d'autorisation.
