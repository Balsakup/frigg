# Contrat d'API : champs personnalisés

Mêmes conventions que [006 › api.md](../../006-memberships/contracts/api.md) :
préfixe `/api`, camelCase, `problem+json` en français avec `errors[].location`
sur les `422`, `401` sans session, `403` rôle insuffisant. Le schéma OpenAPI
produit par `frigg openapi` fait foi. Tag OpenAPI `settings` pour les
définitions, tags existants `members` et `memberships` pour le reste.

## Types

```text
FieldScope      = "member" | "membership"
FieldType       = "short_text" | "long_text" | "number" | "date" | "bool" | "choice"
FieldVisibility = "admin" | "visible" | "editable"

Choice { id: int, label: string }                       // label 1..80

FieldDefinition {                // administration
  id: int, scope: FieldScope, label: string,            // 1..80
  type: FieldType, visibility: FieldVisibility,
  required: bool,                                       // toujours false pour bool
  help: string|null,                                    // ≤ 200
  choices: Choice[],                                    // [] sauf type choice (2..50)
  archived: bool,
  valueCount: int,                                      // valeurs enregistrées (dialogue de suppression)
  missingCount: int|null                                // champ obligatoire : fiches, ou adhésions de la saison en cours, sans valeur ; null sinon
}

FormField {                      // définition utile à un formulaire, selon l'audience
  id: int, label: string, type: FieldType, help: string|null,
  required: bool, editable: bool, choices: Choice[]
}

FieldValue {                     // sortie : FormField + valeur typée
  ...FormField,
  text: string|null,             // short_text, long_text
  number: number|null,
  date: date|null,               // AAAA-MM-JJ
  checked: bool|null,            // bool : true/false (false si non renseigné)
  choiceId: int|null
}

FieldInput {                     // entrée ; au plus une propriété de valeur
  fieldId: int,
  text?: string,                 // short_text, long_text, ou saisie brute d'un nombre (« 12,5 »)
  date?: date,
  checked?: bool,
  choiceId?: int
}                                // aucune valeur ou text "" : efface la valeur
```

Règles communes à toute écriture de `customFields: FieldInput[]` (≤ 30
éléments, `fieldId` distincts) :

- un champ absent de la liste n'est pas modifié ;
Toutes les erreurs de valeur sont localisées par identifiant de champ,
`body.customFields.f<fieldId>` (par exemple `body.customFields.f7`), que la
valeur figure ou non dans la requête ; l'adaptateur 422 du frontend les relie
au champ `customFields.f<fieldId>` du formulaire :

- champ inconnu, archivé, d'une autre portée, ou non modifiable pour
  l'audience : « Ce champ n'existe pas ou n'est pas modifiable. » (même
  message dans tous les cas, R3) ;
- valeur non conforme : message du type (« Saisissez un nombre. »,
  « 200 caractères au plus. », « Choix inconnu. »…) ;
- champ obligatoire modifiable par l'audience et sans valeur finale :
  « Ce champ est obligatoire. » (une entrée par champ manquant) ;
- toutes les erreurs de la requête sont renvoyées ensemble, avec celles des
  autres champs de la fiche ou de l'adhésion.

## Définitions (administration, tag `settings`)

| Opération | Méthode et chemin | Rôle | Corps / paramètres | Réponse |
| --- | --- | --- | --- | --- |
| `list-custom-fields` | `GET /admin/custom-fields` | admin | — | `{ member: FieldDefinition[], membership: FieldDefinition[] }` : actifs par position, puis archivés par libellé |
| `create-custom-field` | `POST /admin/custom-fields` | admin | `{ scope, label, type, visibility, required, help, choices: string[] }` | `201 FieldDefinition` (en fin de liste) |
| `update-custom-field` | `PUT /admin/custom-fields/{id}` | admin | `{ label, type, visibility, required, help, choices: { id?: int, label }[] }` | `FieldDefinition` |
| `reorder-custom-fields` | `PUT /admin/custom-fields/order` | admin | `{ scope, ids: int[] }` (tous les actifs de la portée, dans l'ordre) | `{ member, membership }` |
| `archive-custom-field` | `POST /admin/custom-fields/{id}/archive` | admin | — | `FieldDefinition` |
| `restore-custom-field` | `POST /admin/custom-fields/{id}/restore` | admin | — | `FieldDefinition` (en fin de liste) |
| `delete-custom-field` | `DELETE /admin/custom-fields/{id}` | admin | — | `204` |

Erreurs :

- `422` : libellé vide, trop long ou déjà utilisé dans la portée
  (`body.label`) ; aide trop longue (`body.help`) ; moins de 2 ou plus de 50
  choix, choix vide ou en double (`body.choices`, `body.choices[i]`) ;
  `required` sur un `bool` (`body.required`) ; choix utilisé retiré
  (`body.choices` « Le choix « … » est utilisé : renommez-le plutôt que de le
  retirer. ») ; 31e champ actif (`body.scope` à la création, détail à la
  restauration) ;
- `409` : changement de `type` alors que des valeurs existent ; liste
  d'ordre incomplète ou périmée ; archivage d'un champ archivé ou
  rétablissement d'un champ actif ;
- `404` : champ inconnu.

## Définitions pour les formulaires du membre

| Opération | Méthode et chemin | Rôle | Réponse |
| --- | --- | --- | --- |
| `list-my-custom-fields` | `GET /me/custom-fields` | member | `{ member: FormField[], membership: FormField[] }` : actifs, `visibility <> 'admin'`, par position ; `editable = (visibility = 'editable')` |

Côté administration, les formulaires de création utilisent
`list-custom-fields` (actifs seulement, `editable = true`).

## Opérations existantes modifiées

| Opération | Modification |
| --- | --- |
| `get-member`, `create-member`, `update-member` | Sortie `AdminMember.customFields: FieldValue[]` (tous les champs de membre actifs) ; corps de création et de modification : `customFields?: FieldInput[]` (audience `Admin`) |
| `get-my-member`, `create-my-member`, `update-my-member` | Sortie `Member.customFields: FieldValue[]` (champs `visible` et `editable`) ; corps : `customFields?: FieldInput[]` (audience `Owner`) |
| `list-members` | Paramètre répété `field` (`^[0-9]+:(true\|false\|empty\|c[0-9]+)$`, ≤ 10) ; `q` trouve aussi les valeurs des champs de membre actifs de type `short_text` (R5) ; `422` `query.field` si le champ n'est pas un oui/non ou une liste de choix actif de portée `member` |
| `list-memberships` | Paramètre répété `field`, mêmes règles, portée `membership` |
| `get-membership` | Sortie `customFields: FieldValue[]` (champs d'adhésion actifs ; `editable = false` si l'adhésion est refusée ou annulée) |
| `create-membership`, `validate-membership` | Corps : `customFields?: FieldInput[]` (audience `Admin`, obligatoires contrôlés) |
| `get-my-membership`, `list-my-memberships` | `get-my-membership` : sortie `customFields: FieldValue[]` (audience `Owner`) ; `list-my-memberships` inchangée |
| `create-my-membership` | Corps : `customFields?: FieldInput[]` (audience `Owner`, obligatoires `editable` contrôlés) |

## Nouvelles opérations d'adhésion

| Opération | Méthode et chemin | Rôle | Corps | Réponse |
| --- | --- | --- | --- | --- |
| `update-membership-fields` | `PUT /admin/memberships/{id}/fields` | admin | `{ customFields: FieldInput[] }` | `AdminMembership` (sortie de `get-membership`) |
| `update-my-membership-fields` | `PUT /me/memberships/{id}/fields` | member | `{ customFields: FieldInput[] }` | `MyMembership` |

- adhésion `refused` ou `cancelled` : `409` « Cette adhésion ne peut plus
  être modifiée. » ;
- `update-my-membership-fields` sur une adhésion d'une fiche non gérée :
  `404` (comme `get-my-membership`) ;
- l'état, le tarif, le montant et le paiement ne sont jamais modifiés.

## Autorisation (matrice)

10 opérations ajoutées : 8 `admin` (`list-custom-fields`,
`create-custom-field`, `update-custom-field`, `reorder-custom-fields`,
`archive-custom-field`, `restore-custom-field`, `delete-custom-field`,
`update-membership-fields`) et 2 `member` (`list-my-custom-fields`,
`update-my-membership-fields`). Aucune opération publique.
