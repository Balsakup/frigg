# Modèle de données : paiement par carte (HelloAsso)

Complète [006 › data-model.md](../006-memberships/data-model.md). Décisions :
[research.md](./research.md). Migration unique `00009_online_payments.sql`.
Commentaires SQL des requêtes sqlc en ASCII. Montants en centimes d'euro,
dates en texte ISO 8601 UTC (sauf `paid_on`, date à Paris).

## Configuration (hors base)

Variables `FRIGG_HELLOASSO_*` ([R2](./research.md#r2-configuration-de-linstance)) :
identifiant et secret du client, slug de l'association, environnement
(`sandbox` | `production`). Le secret et les jetons OAuth2 ne sont **jamais**
écrits en base (FR-014) ; les jetons vivent en mémoire du processus.

## Table `memberships` (reconstruite)

Reconstruction à l'identique ([R12](./research.md#r12-migration)), avec :

| Colonne | Changement |
| --- | --- |
| `announced_method` | `CHECK (announced_method IN ('cash','cheque','transfer'))` inchangé : la carte ne s'annonce pas |
| `paid_method` | `CHECK (paid_method IN ('cash','cheque','transfer','card'))` |
| `paid_online_payment_id` | **nouvelle** : `INTEGER NULL REFERENCES online_payments(id) ON DELETE SET NULL` — tentative qui a constaté le paiement |
| `paid_amount_cents` | **nouvelle** : `INTEGER NULL CHECK (paid_amount_cents BETWEEN 0 AND 1000000)` — montant reçu, renseigné pour un paiement par carte (hors contribution volontaire) |

`CHECK` ajoutés :

- `paid_method IS NOT 'card' OR (paid_online_payment_id IS NOT NULL AND paid_amount_cents IS NOT NULL AND paid_by IS NULL)` ;
- `paid_online_payment_id IS NULL OR paid_method = 'card'`.

Les autres colonnes, `CHECK`, index (`memberships_one_active`…) et la table
FTS `memberships_fts` avec ses trois triggers sont recréés à l'identique ;
`memberships_fts` est reconstruite (`'rebuild'`).

Note : la référence circulaire `memberships.paid_online_payment_id` →
`online_payments` → `memberships` est permise par SQLite (contrôle des clés
étrangères à l'écriture) ; la migration crée `online_payments` après la
nouvelle `memberships`, les deux dans la même transaction goose.

## Table `online_payments`

Une ligne par tentative de paiement par carte
([R4](./research.md#r4-tentatives-de-paiement-en-ligne)).

| Colonne | Type | Contraintes | Rôle |
| --- | --- | --- | --- |
| `id` | `INTEGER` | `PRIMARY KEY` | Identifiant, transmis dans `metadata.frigg.payment` |
| `membership_id` | `INTEGER` | `NOT NULL REFERENCES memberships(id) ON DELETE CASCADE` | Adhésion payée |
| `account_id` | `INTEGER` | `NULL REFERENCES accounts(id) ON DELETE SET NULL` | Compte qui a lancé le paiement |
| `environment` | `TEXT` | `NOT NULL CHECK (environment IN ('sandbox','production'))` | Environnement HelloAsso au lancement |
| `amount_cents` | `INTEGER` | `NOT NULL CHECK (amount_cents BETWEEN 1 AND 1000000)` | Montant demandé (montant appliqué au lancement) |
| `status` | `TEXT` | `NOT NULL CHECK (status IN ('pending','succeeded','failed','abandoned','expired'))` | État de la tentative |
| `checkout_intent_id` | `INTEGER` | `NULL` | Intention HelloAsso ; `NULL` tant que la création n'a pas abouti |
| `redirect_url` | `TEXT` | `NULL CHECK (redirect_url IS NULL OR length(redirect_url) <= 2000)` | Page de paiement HelloAsso (valable 15 min), pour la reprise |
| `order_id` | `INTEGER` | `NULL` | Commande HelloAsso (référence affichée aux administrateurs) |
| `helloasso_payment_id` | `INTEGER` | `NULL` | Paiement HelloAsso autorisé |
| `received_amount_cents` | `INTEGER` | `NULL CHECK (received_amount_cents BETWEEN 0 AND 1000000)` | Montant reçu hors contribution volontaire |
| `paid_at` | `TEXT` | `NULL` | Date et heure du paiement chez HelloAsso |
| `error_code` | `TEXT` | `NULL CHECK (error_code IS NULL OR length(error_code) <= 60)` | Cause d'un échec (`unavailable`, `unauthorized`, code renvoyé par HelloAsso…) |
| `created_at` | `TEXT` | `NOT NULL` | Lancement |
| `finished_at` | `TEXT` | `NULL` | Passage à un état final |
| `last_verified_at` | `TEXT` | `NULL` | Dernière vérification auprès de HelloAsso (limitation R5) |
| `updated_at` | `TEXT` | `NOT NULL` | Dernière modification |

Table `STRICT`. `CHECK` :

- `status <> 'succeeded' OR (order_id IS NOT NULL AND helloasso_payment_id IS NOT NULL AND received_amount_cents IS NOT NULL AND paid_at IS NOT NULL)` ;
- `status = 'pending' OR finished_at IS NOT NULL`.

Index :

- `online_payments_intent` : `UNIQUE (environment, checkout_intent_id) WHERE checkout_intent_id IS NOT NULL` ;
- `online_payments_helloasso_payment` : `UNIQUE (environment, helloasso_payment_id) WHERE helloasso_payment_id IS NOT NULL` (un paiement HelloAsso n'est rattaché qu'une fois, FR-008a) ;
- `online_payments_membership` sur `(membership_id, created_at)` (reprise, historique) ;
- `online_payments_pending` sur `(status, created_at) WHERE status = 'pending'` (balayage R8) ;
- `online_payments_order` sur `order_id` (recherche par référence HelloAsso : égalité exacte sur un numéro, pas de recherche textuelle).

### Transitions

```text
            ┌──────── retour « erreur » ───────▶ failed ─────┐
            │                                                 │
pending ────┼──────── retour « annulé » ───────▶ abandoned ──┤  paiement autorisé
            │                                                 │  découvert plus tard
            ├──────── > 60 min sans paiement ──▶ expired ────┤  (R5, règle 7)
            │                                                 ▼
            └──────── paiement autorisé vérifié ───────────▶ succeeded (final)
création HelloAsso impossible : pending ─▶ failed (error_code)
```

`succeeded` est le seul état réellement final. Les règles vivent dans
`internal/membership` (`SettleOnline`), testées en table.

## Table `payment_alerts`

Signalements à traiter par un administrateur
([R10](./research.md#r10-signalements)).

| Colonne | Type | Contraintes | Rôle |
| --- | --- | --- | --- |
| `id` | `INTEGER` | `PRIMARY KEY` | Identifiant |
| `membership_id` | `INTEGER` | `NOT NULL REFERENCES memberships(id) ON DELETE CASCADE` | Adhésion concernée |
| `online_payment_id` | `INTEGER` | `NOT NULL REFERENCES online_payments(id) ON DELETE CASCADE` | Tentative concernée |
| `kind` | `TEXT` | `NOT NULL CHECK (kind IN ('duplicate','amount_mismatch','refund_due','unverified'))` | Nature |
| `created_at` | `TEXT` | `NOT NULL` | Création |
| `resolved_at` | `TEXT` | `NULL` | Traitement |
| `resolved_by` | `INTEGER` | `NULL REFERENCES accounts(id) ON DELETE SET NULL` | Administrateur qui l'a traité |

Table `STRICT`. `CHECK (resolved_by IS NULL OR resolved_at IS NOT NULL)`.

Index : `UNIQUE (online_payment_id, kind)` (idempotence) ;
`payment_alerts_open` sur `(resolved_at, created_at)` (tableau de bord, liste).

| Nature | Créé quand | Libellé |
| --- | --- | --- |
| `duplicate` | paiement autorisé sur une adhésion déjà payée | Paiement en double — remboursement à effectuer dans HelloAsso |
| `amount_mismatch` | montant reçu ≠ montant appliqué | Montant reçu différent du montant attendu |
| `refund_due` | paiement sur adhésion annulée, ou annulation d'une adhésion payée par carte | Paiement par carte reçu — remboursement à effectuer dans HelloAsso |
| `unverified` | tentative impossible à vérifier plus de 24 h après son expiration | Paiement par carte à vérifier dans HelloAsso |

## Table `payment_settings` (colonnes ajoutées)

`ALTER TABLE … ADD COLUMN` (pas de reconstruction) :

| Colonne | Type | Contraintes | Rôle |
| --- | --- | --- | --- |
| `card_enabled` | `INTEGER` | `NOT NULL DEFAULT 0 CHECK (card_enabled IN (0,1))` | Carte activée par un administrateur |
| `helloasso_test_at` | `TEXT` | `NULL` | Date du dernier test de connexion |
| `helloasso_test_environment` | `TEXT` | `NULL CHECK (… IN ('sandbox','production'))` | Environnement testé |
| `helloasso_test_ok` | `INTEGER` | `NULL CHECK (… IN (0,1))` | Résultat |
| `helloasso_test_error` | `TEXT` | `NULL CHECK (length(…) <= 60)` | Code d'erreur du test |
| `helloasso_organization_name` | `TEXT` | `NULL CHECK (length(…) <= 250)` | Nom de l'association renvoyé par HelloAsso |
| `helloasso_error_at` | `TEXT` | `NULL` | Dernier incident d'appel (R9), remis à `NULL` au premier succès |
| `helloasso_error_code` | `TEXT` | `NULL CHECK (length(…) <= 60)` | Code de cet incident |

Pas de `CHECK` liant `card_enabled` à la configuration (elle est hors base) :
la règle d'activation est appliquée par le domaine et l'API (FR-013), et la
disponibilité recalculée à chaque lecture (R7).

## Domaine `internal/membership` (ajouts)

| Élément | Rôle |
| --- | --- |
| `Card Method = "card"` | Nouveau moyen ; `Methods = {Card, Cash, Cheque, Transfer}` |
| `MinCardAmount = 100` | Montant minimal payable par carte (centimes) |
| `CardAvailability(cfgConfigured, enabled, testedEnv, testOK, currentEnv) bool` | Règle R7 |
| `ActionsFor` | + `PayByCard`, `cancelPayment=false` si `paid_method = card` |
| `OnlineStatus`, `AlertKind` | Énumérations ci-dessus |
| `SettleOnline(Snapshot, Verified) Settlement` | Décision R5 : nouvel état de tentative, constatation ou non, signalement éventuel |
| `Stats` | `collectedByMethod.card`, `collectedTest` |

## Domaine `internal/helloasso` (nouveau, sans base)

| Type | Rôle |
| --- | --- |
| `Config` | identifiants, slug, environnement, hôte (dérivé ou surchargé) |
| `Client` | jeton en mémoire, `Organization`, `CreateCheckout`, `GetCheckout` |
| `CheckoutRequest` | montant, intitulé, payeur, URL, `Metadata{Payment, Membership, Reference}` |
| `Checkout` | `ID`, `Metadata`, `Order *Order` |
| `Order` | `ID`, `Payments []Payment` |
| `Payment` | `ID`, `AmountCents`, `TipCents`, `State`, `Date` ; `Authorized()` |
| `NotificationKey(secret) string` | clé de l'adresse de notification (R6) |
