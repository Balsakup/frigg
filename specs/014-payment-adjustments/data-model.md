# Data Model: Remises et dons sur les paiements

**Feature**: [spec.md](./spec.md) | **Research**: [research.md](./research.md)

Une seule migration, `00013_payment_adjustments.sql` (append-only : `00008`
et `00012` ne sont pas modifiées). Montants en centimes d'euro. Les
commentaires de la migration et des requêtes sont en ASCII (règle sqlc).

## memberships (006) : colonnes ajoutées

| Colonne | Type | Contraintes | Sens |
| --- | --- | --- | --- |
| `base_amount_cents` | INTEGER NULL | `CHECK (base_amount_cents IS NULL OR (base_amount_cents BETWEEN 0 AND 1000000 AND amount_cents IS NOT NULL AND base_amount_cents <> amount_cents))` | Montant théorique, seulement s'il y a un ajustement |
| `adjustment_reason` | TEXT NULL | `CHECK ((adjustment_reason IS NULL) = (base_amount_cents IS NULL))`, `CHECK (adjustment_reason IS NULL OR length(adjustment_reason) BETWEEN 1 AND 200)` | Motif de l'ajustement |

Colonnes existantes dont le sens évolue :

| Colonne | Avant | Après |
| --- | --- | --- |
| `amount_cents` | Montant appliqué (tarif) | **Montant à payer**, ajustement compris |
| `paid_amount_cents` | Montant reçu par carte uniquement | **Montant reçu**, pour tout paiement constaté |

Valeurs dérivées (jamais stockées) :

- montant théorique = `COALESCE(base_amount_cents, amount_cents)` ;
- ajustement = `amount_cents − base_amount_cents` (remise si < 0, majoration
  si > 0, 0 sans ajustement), calculé par le serveur et exposé aux
  administrateurs en `adjustmentCents` ;
- don = `paid_amount_cents − amount_cents` pour un moyen manuel, 0 pour la
  carte.

## event_registrations (012) : colonnes ajoutées

Mêmes colonnes et mêmes contraintes, `amount_cents` étant `NOT NULL` sur
cette table :

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `base_amount_cents` | INTEGER NULL | `CHECK (base_amount_cents IS NULL OR (base_amount_cents BETWEEN 0 AND 1000000 AND base_amount_cents <> amount_cents))` |
| `adjustment_reason` | TEXT NULL | identiques à `memberships` |

Le montant théorique d'une inscription est le prix de l'événement figé à
l'inscription. La contrainte existante `(payment_status IS NULL) =
(amount_cents = 0)` s'applique au montant à payer : une inscription ajustée
à 0 € n'a pas de paiement.

## Reprise de données (Up)

```text
UPDATE memberships SET paid_amount_cents = amount_cents
 WHERE payment_status = 'paid' AND paid_amount_cents IS NULL;
UPDATE event_registrations SET paid_amount_cents = amount_cents
 WHERE payment_status = 'paid' AND paid_amount_cents IS NULL;
```

Les paiements par carte ont déjà leur montant reçu. Aucun ajustement n'est
créé.

## Down

Sur chaque table, `DROP COLUMN adjustment_reason` **puis** `DROP COLUMN
base_amount_cents` : le `CHECK` du motif lit `base_amount_cents`, donc
l'ordre inverse échoue. Ensuite, `paid_amount_cents = NULL` pour les
paiements manuels (`paid_method <> 'card'`). Ordre vérifié sur SQLite 3.51.

## Requêtes (sqlc) modifiées ou ajoutées

`internal/database/queries/memberships.sql` :

- `CreateMembership`, `ValidateMembership`, `UpdateValidatedMembership` :
  paramètres `base_amount_cents` et `adjustment_reason` (nullable).
- `RecordMembershipPayment` : paramètre `paid_amount_cents`.
- `CancelMembershipPayment` : `paid_amount_cents = NULL`.
- `ListSeasonMembershipsForStats` : ajoute `base_amount_cents`.

`internal/database/queries/event_registrations.sql` :

- `CreateRegistration` : `base_amount_cents`, `adjustment_reason`.
- `AdjustRegistrationAmount` (nouvelle) : met à jour `amount_cents`,
  `base_amount_cents`, `adjustment_reason`, `payment_reference`,
  `payment_status` et `announced_method`, avec la garde `WHERE id = ? AND
  status = 'confirmed' AND payment_status IS NOT 'paid'`.
- `RecordRegistrationPayment` : paramètre `paid_amount_cents`.
- `CancelRegistrationPayment` : `paid_amount_cents = NULL`.
- `GetRegistrationNotice` : inchangée (lit déjà `paid_amount_cents`).

## Règles d'état (domaine `internal/membership/amount.go`)

```text
Adjust(base, requested, reason)
  requested absent        → amount = base, pas d'ajustement
  requested == base       → amount = base, pas d'ajustement (motif ignoré)
  requested != base
     motif vide           → ErrAdjustmentReasonRequired
     motif > 200 car.     → ErrAdjustmentReasonTooLong
     sinon                → amount = requested, base conservé, motif nettoyé

CheckReceived(due, received)
  received < due          → ErrReceivedBelowDue

Effets d'un montant à payer sur le paiement : membership.PaymentAfter (inchangé)
  0 €                     → pas de paiement, annonce effacée, référence conservée
  > 0 sans paiement       → « à payer », référence attribuée si absente
  > 0 avec paiement       → état et annonce conservés
```

Les actions autorisées sont inchangées. On ajuste une adhésion avec la
correction existante (`CanUpdate` : validée, non payée, saison non close).
On ajuste une inscription si elle est confirmée, non payée et que son
événement n'est pas annulé.

## AmountTotals (chiffres)

| Champ | Cumul sur les lignes retenues (adhésions validées, inscriptions confirmées) |
| --- | --- |
| `Theoretical` | montant théorique |
| `Discounts` | somme des remises (valeur positive) |
| `Surcharges` | somme des majorations |
| `Due` | montant à payer |
| `Collected` + `CollectedByMethod` | montants reçus des lignes payées, hors mode test |
| `Donations` | dons des lignes payées, hors mode test |
| `Remaining` + `RemainingByMethod` | montants à payer des lignes « à payer » ou « annoncé » |
| `CollectedTest` | montants reçus par carte en mode test |

Identité vérifiée par les tests : `Theoretical − Discounts + Surcharges =
Due`.
