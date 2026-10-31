# Modèle de données : Gestion des tarifs de l'adhésion

Complète [003 › data-model.md](../003-season-management/data-model.md).
Décisions : [research.md](./research.md).

## Table `membership_fees` (migration `00005_membership_fees.sql`)

| Colonne | Type | Contraintes | Rôle |
| --- | --- | --- | --- |
| `id` | `INTEGER` | `PRIMARY KEY` | Identifiant |
| `season_id` | `INTEGER` | `NOT NULL`, `REFERENCES seasons(id) ON DELETE CASCADE` | Saison propriétaire |
| `label` | `TEXT` | `NOT NULL`, `CHECK (length(label) BETWEEN 1 AND 80)` | Libellé, sans espaces de début et de fin |
| `amount_cents` | `INTEGER` | `NOT NULL`, `CHECK (amount_cents BETWEEN 0 AND 1000000)` | Montant en centimes d'euro |
| `renewal_amount_cents` | `INTEGER` | `NULL`, `CHECK (renewal_amount_cents IS NULL OR renewal_amount_cents BETWEEN 0 AND 1000000)` | Montant de renouvellement en centimes ; `NULL` = même montant pour toute adhésion (migration `00006_membership_fee_renewal.sql`, [R12](./research.md#r12-montant-de-renouvellement)) |
| `description` | `TEXT` | `NULL`, `CHECK (description IS NULL OR length(description) BETWEEN 1 AND 500)` | Description facultative |
| `created_at` | `TEXT` | `NOT NULL` | Date et heure de création (UTC) |
| `updated_at` | `TEXT` | `NOT NULL` | Date et heure de dernière modification (UTC) |

Table `STRICT`. Index `membership_fees_season_id` sur `season_id` (lecture par
saison, cascade). `length()` de SQLite compte les caractères, comme le
contrôle de l'API.

Aucun index unique sur le libellé : l'unicité sans casse ni accents est
vérifiée par le domaine dans la transaction ([R2](./research.md#r2-unicité-du-libellé-sans-casse-ni-accents)).

## Règles (contrôlées par `internal/season/fee.go`)

| Règle | Détail | Exigence |
| --- | --- | --- |
| Libellé | 80 caractères au plus (schéma) ; espaces de début et fin retirés ; non vide après nettoyage | FR-001 |
| Description | 500 caractères au plus (schéma) ; espaces de début et fin retirés ; vide → `NULL` | FR-001 |
| Montant | texte analysé : chiffres, `,` ou `.` décimal, ≤ 2 décimales, espaces et zéros de tête ignorés ; 0 à 10 000 € (plus de 5 chiffres entiers significatifs → trop élevé) | FR-002 |
| Renouvellement | `hasRenewalAmount` vrai : `renewalAmount` obligatoire, mêmes règles que le montant ; faux : `NULL` ; aucune relation imposée avec le montant principal | FR-002a |
| Unicité | `textfold.Fold(label)` distinct de celui des autres tarifs de la saison | FR-003 |
| Plafond | au plus 20 tarifs par saison (contrôlé à la création) | FR-004 |
| Éditabilité | saison dont l'état calculé n'est pas `closed` | FR-007 à FR-009 |

## Attributs calculés (jamais stockés)

| Attribut | Calcul | Exigences |
| --- | --- | --- |
| ordre | `amount_cents` décroissant, puis `textfold.Fold(label)`, puis `id` | FR-005 |
| `actions.edit`, `actions.delete` | état de la saison ≠ `closed` | FR-007 à FR-009 |
| `actions.add` | état ≠ `closed` et moins de 20 tarifs | FR-004, FR-006 |

## Cycle de vie

```text
 saison brouillon ou ouverte : ajouter · modifier · supprimer (dernier compris)
 saison close               : lecture seule
 dupliquer la saison        : tous les tarifs recopiés dans la nouvelle saison
 supprimer la saison        : tarifs supprimés en cascade
```

## Relations futures

Les adhésions (fonctionnalité ultérieure) référenceront un tarif de leur
saison et conserveront le type retenu (renouvellement déclaré par l'adhérent
et validé par un administrateur, ou première adhésion) et le montant payé ; la suppression d'un tarif devra alors
vérifier qu'aucune adhésion n'y est rattachée.
