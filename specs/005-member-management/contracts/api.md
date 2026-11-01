# Contrat d'API : Gestion des membres

Mêmes conventions que [002 › api.md](../../002-admin-account-management/contracts/api.md)
et [004 › api.md](../../004-membership-fees/contracts/api.md) (préfixe `/api`,
camelCase, `problem+json` en français, CSRF, `401` sans session, `403` rôle
insuffisant). Le schéma OpenAPI produit par `frigg openapi` fait foi.

## Types

```text
Member { id: int,
         firstName: string, lastName: string,
         birthDate: date,                 // "2015-04-12"
         age: int,                        // calculé avec la date du jour (Paris)
         email: string|null, phone: string|null,      // phone normalisé : "0612345678", "+32470123456"
         address: PostalAddress|null,
         isSelf: bool,
         createdAt: datetime, updatedAt: datetime }

PostalAddress { line1: string, line2: string|null, postalCode: string, city: string }

AdminMember = Member + { account: MemberAccount|null }   // null = « Sans compte »
MemberAccount { id: int, email: string, disabled: bool }

MyMemberList { items: Member[],                 // « Vous » d'abord, puis prénom, puis nom (repliés)
               actions: { add: bool, markSelf: bool } }

AdminMemberPage { items: AdminMember[],          // nom, puis prénom (repliés), puis id
                  total: int, page: int, pageSize: int }

MemberBody { firstName: string,                // maxLength 80
             lastName: string,                 // maxLength 80
             birthDate: date,                  // format date
             email?: string,                   // maxLength 254 ; vide = aucune
             phone?: string,                   // maxLength 30 ; vide = aucun
             addressLine1?: string,            // maxLength 100
             addressLine2?: string,            // maxLength 100
             postalCode?: string,              // maxLength 10
             city?: string,                    // maxLength 100
             confirmDuplicate?: bool }         // vrai = enregistrer malgré un doublon probable

MyMemberBody    = MemberBody + { isSelf?: bool }          // espace membre seulement
AdminMemberBody = MemberBody + { accountId: int|null }    // administration seulement ; null = Sans compte ; obligatoire
AdminMemberUpdateBody = AdminMemberBody + { expectedAccountId: int|null }
                                 // update-member seulement, obligatoire : rattachement affiché à l'ouverture
```

`isSelf` et `accountId` n'existent que dans le corps de leur espace
([R5](../research.md#r5-opérations-dapi)).
Les corps refusent toute propriété inconnue (`additionalProperties: false`,
comportement par défaut de huma, déjà appliqué à tous les corps existants) :
`accountId` envoyé à `/me/members` ou `isSelf` envoyé à `/admin/members` donne
un `422`.

## Opérations `member` (tag `Mes membres`)

Le compte est celui de la session ; aucune opération ne reçoit d'identifiant
de compte.

| Opération | Méthode et chemin | Entrée | Réponses |
| --- | --- | --- | --- |
| `list-my-members` | `GET /api/me/members` | — | `200 MyMemberList` |
| `get-my-member` | `GET /api/me/members/{id}` | — | `200 Member` · `404` |
| `create-my-member` | `POST /api/me/members` | `MyMemberBody` | `201 Member` · `409` plafond · `422` |
| `update-my-member` | `PUT /api/me/members/{id}` | `MyMemberBody` | `200 Member` · `404` · `422` |
| `delete-my-member` | `DELETE /api/me/members/{id}` | — | `204` · `404` |

Une fiche d'un autre compte, sans compte ou inexistante donne toujours `404`
« Ce membre n'existe plus. » ([R4](../research.md#r4-cloisonnement-par-compte-us2)).

## Opérations `admin` (tag `Membres`)

| Opération | Méthode et chemin | Entrée | Réponses |
| --- | --- | --- | --- |
| `list-members` | `GET /api/admin/members` | `q`, `account`, `page`, `pageSize` | `200 AdminMemberPage` · `422` |
| `get-member` | `GET /api/admin/members/{id}` | — | `200 AdminMember` · `404` |
| `create-member` | `POST /api/admin/members` | `AdminMemberBody` | `201 AdminMember` · `409` plafond · `422` |
| `update-member` | `PUT /api/admin/members/{id}` | `AdminMemberUpdateBody` | `200 AdminMember` · `404` · `409` plafond ou rattachement modifié · `422` |
| `delete-member` | `DELETE /api/admin/members/{id}` | — | `204` · `404` |

Paramètres de `list-members` :

| Paramètre | Valeurs | Rôle |
| --- | --- | --- |
| `q` | ≤ 100 caractères | fragments d'au moins 3 caractères (prénom, nom, adresse, téléphone), tous requis |
| `account` | `none` ou identifiant de compte | fiches sans compte, ou fiches d'un compte (dialogue de suppression de compte) |
| `page` | ≥ 1, défaut 1 | |
| `pageSize` | 1 à 100, défaut 25 | |

## Opération modifiée : `delete-account` (002)

`DELETE /api/admin/accounts/{id}?members=delete|keep&memberIds=3,8,12`

| Situation | Réponse |
| --- | --- |
| compte sans fiche | comportement de 002, paramètres ignorés |
| fiches présentes, `members` absent | `422` `query.members` |
| `memberIds` ≠ fiches actuelles du compte | `409` « La liste des fiches de ce compte a changé. » |
| `members=delete` | fiches effacées, puis compte supprimé (`204`) |
| `members=keep` | fiches « Sans compte », `isSelf` faux, puis compte supprimé (`204`) |

Les autres réponses de 002 (`404`, `409` soi-même, `409` dernier
administrateur) sont inchangées et prioritaires.

## Erreurs de champ (`422`)

| Emplacement | Cas | `message` |
| --- | --- | --- |
| `body.firstName` | vide après nettoyage | « Saisissez le prénom. » |
| `body.lastName` | vide après nettoyage | « Saisissez le nom. » |
| `body.birthDate` | dans le futur | « La date de naissance ne peut pas être dans le futur. » |
| `body.birthDate` | plus de 120 ans | « Vérifiez la date de naissance : elle remonte à plus de 120 ans. » |
| `body.email` | forme invalide | « Saisissez une adresse électronique valide (par exemple nom@exemple.fr). » |
| `body.phone` | forme invalide | « Saisissez un numéro à 10 chiffres (06 12 34 56 78) ou au format international (+32 …). » |
| `body.addressLine1` · `body.postalCode` · `body.city` | adresse partielle (champ manquant) | « Complétez l'adresse : ligne d'adresse, code postal et commune. » (sur chaque champ manquant) |
| `body.addressLine2` | complément sans ligne d'adresse | « Saisissez d'abord la ligne d'adresse. » |
| `body.postalCode` | pas 5 chiffres | « Saisissez un code postal à 5 chiffres. » |
| `body.isSelf` | une autre fiche du compte est déjà « Vous » | « Une autre fiche vous désigne déjà. » |
| `body.accountId` | compte inexistant | « Ce compte n'existe plus. » |
| `body.confirmDuplicate` | doublon probable non confirmé | « Un membre portant ces nom, prénom et date de naissance existe déjà. Confirmez pour enregistrer quand même. » |
| `query.q` | aucun fragment de 3 caractères | « Saisissez au moins 3 caractères. » (existant) |
| `query.members` | fiches présentes sans choix | « Choisissez le sort des fiches gérées par ce compte. » |

Les longueurs (`maxLength`) et le format de date sont validés par huma avant
le handler (traductions existantes). Toutes les erreurs de champ d'une requête
sont renvoyées ensemble ; le doublon n'est vérifié que si aucune autre erreur
de champ n'est présente.

## Autres erreurs

| Statut | Cas | `detail` |
| --- | --- | --- |
| `404` | fiche absente, supprimée, ou hors du compte (espace membre) | « Ce membre n'existe plus. » |
| `409` | plafond (espace membre) | « Vous gérez déjà 20 membres. Contactez le bureau pour en ajouter. » |
| `409` | plafond (administration) | « Ce compte gère déjà 20 membres. » |
| `409` | `update-member` : rattachement actuel ≠ `expectedAccountId` | « Le rattachement de ce membre a changé entre-temps. Vérifiez la fiche avant d'enregistrer. » |
| `409` | `memberIds` périmé (`delete-account`) | « La liste des fiches de ce compte a changé. » |

## Effets et journal

| Opération | Effet | Journal (`INFO`) |
| --- | --- | --- |
| `create-*` | fiche créée | `member.create` (+ `member.attach` si rattachée, administration) |
| `update-*` | fiche modifiée ; changement de rattachement → `isSelf` faux | `member.update` (+ `member.attach` si le rattachement change) |
| `delete-*` | fiche effacée | `member.delete` |
| `delete-account` | voir ci-dessus | `account.delete` + `members_deleted` ou `members_kept` |

Attributs : `action`, `actor_id`, `member_id`, `account_id` (si rattachée),
`space` (`me` ou `admin`) ; aucune donnée de la fiche
([R13](../research.md#r13-journalisation)).
