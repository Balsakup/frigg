# Contrat d'API : Tarifs de l'adhésion

Mêmes conventions que [003 › api.md](../../003-season-management/contracts/api.md)
(préfixe `/api`, camelCase, `problem+json` en français, CSRF, `401` sans
session, `403` rôle insuffisant). Le schéma OpenAPI produit par `frigg openapi`
fait foi.

## Types

```text
MembershipFee { id: int, label: string, amountCents: int,
                renewalAmountCents: int|null,   // null = même montant pour toute adhésion
                description: string|null,
                createdAt: datetime, updatedAt: datetime }

MembershipFeeActions { add: bool, edit: bool, delete: bool }

MembershipFeeList { items: MembershipFee[],   // montant décroissant, puis libellé
                    actions: MembershipFeeActions }

MembershipFeeBody { label: string,             // 1 à 80 caractères (maxLength 80)
                    amount: string,            // "15,50", "15.5", "1 000"
                    hasRenewalAmount?: bool,   // case « Tarif de renouvellement différent » ; absent = faux
                    renewalAmount?: string,    // obligatoire si hasRenewalAmount, ignoré sinon
                    description?: string }     // 500 caractères au plus ; absente ou vide = aucune
```

## Opérations `admin` (tag `Tarifs`)

Aucune opération publique ni `member` n'est ajoutée.

| Opération | Méthode et chemin | Entrée | Réponses |
| --- | --- | --- | --- |
| `list-membership-fees` | `GET /api/admin/seasons/{id}/fees` | — | `200 MembershipFeeList` · `404` saison |
| `create-membership-fee` | `POST /api/admin/seasons/{id}/fees` | `MembershipFeeBody` | `201 MembershipFee` · `404` · `409` · `422` |
| `update-membership-fee` | `PUT /api/admin/seasons/{id}/fees/{feeId}` | `MembershipFeeBody` | `200 MembershipFee` · `404` · `409` · `422` |
| `delete-membership-fee` | `DELETE /api/admin/seasons/{id}/fees/{feeId}` | — | `204` · `404` · `409` |

Un `feeId` qui n'appartient pas à la saison `{id}` donne `404`.

## Erreurs de champ (`422`)

| Emplacement | Cas | `message` |
| --- | --- | --- |
| `body.label` | vide après nettoyage | « Saisissez un libellé. » |
| `body.label` | plus de 80 caractères (schéma `maxLength`) | « au plus 80 caractères attendus » (traduction huma existante) |
| `body.label` | même libellé replié qu'un autre tarif de la saison | « Un tarif porte déjà ce nom dans cette saison. » |
| `body.amount` | format invalide, négatif, plus de deux décimales | « Saisissez un montant en euros, avec au plus deux décimales (par exemple 15,50). » |
| `body.amount` | supérieur à 10 000 € | « Le montant ne peut pas dépasser 10 000 €. » |
| `body.renewalAmount` | `hasRenewalAmount` vrai et montant vide | « Saisissez le montant du renouvellement. » |
| `body.renewalAmount` | format invalide | « Saisissez un montant en euros, avec au plus deux décimales (par exemple 15,50). » |
| `body.renewalAmount` | supérieur à 10 000 € | « Le montant ne peut pas dépasser 10 000 €. » |
| `body.description` | plus de 500 caractères (schéma `maxLength`) | « au plus 500 caractères attendus » (traduction huma existante) |

Les longueurs maximales sont déclarées dans le schéma (`maxLength`, compté en
caractères) et validées par huma avant le handler ; le domaine nettoie ensuite
les espaces et contrôle le libellé vide. Le frontend reprend `maxLength` en
attribut HTML natif.

Toutes les erreurs de champ d'une requête sont renvoyées ensemble, dans un
seul `422` avec une entrée par champ (FR-017) : un libellé vide et un montant
invalide donnent deux entrées, `body.label` et `body.amount`. L'unicité est
contrôlée dans la transaction, avec les autres règles de champ.

## Refus métier (`404`, `409`)

| Cas | Statut | `detail` |
| --- | --- | --- |
| Saison inexistante | `404` | « Cette saison n'existe plus. » |
| Tarif inexistant ou d'une autre saison | `404` | « Ce tarif n'existe plus. » |
| Écriture sur une saison close | `409` | « Une saison close ne peut plus être modifiée. » |
| Ajout au-delà de 20 tarifs | `409` | « Une saison ne peut pas compter plus de 20 tarifs. » |

## Effets de bord

| Opération | Effet |
| --- | --- |
| `duplicate-season` (existante) | recopie tous les tarifs de la source dans la nouvelle saison, même transaction ; journal : `fees_copied` |
| `delete-season` (existante) | tarifs supprimés en cascade |
| écriture sur un tarif | ligne `INFO` (`action` `fee.create`/`fee.update`/`fee.delete`, `actor_id`, `season_id`, `fee_id`, `amount_cents`) — [R10](../research.md#r10-journalisation) |

## Changements d'opérations existantes

Contrat inchangé pour les saisons ; seuls les effets de bord ci-dessus
s'ajoutent à `duplicate-season` et `delete-season`.
