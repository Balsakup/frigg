# Contrat d'API : Adhésions

Mêmes conventions que [005 › api.md](../../005-member-management/contracts/api.md)
(préfixe `/api`, camelCase, `problem+json` en français, CSRF, `401` sans
session, `403` rôle insuffisant). Le schéma OpenAPI produit par
`frigg openapi` fait foi. Montants en centimes d'euro.

## Types

```text
MembershipStatus = "pending" | "validated" | "refused" | "cancelled"
MembershipKind   = "first" | "renewal"
PaymentStatus    = "due" | "announced" | "paid"          // absent (null) = pas de paiement
PaymentMethod    = "cash" | "cheque" | "transfer"
PreviousSeason   = "yes" | "no" | "unknown"

MembershipFeeRef { id: int, label: string, amountCents: int,
                   renewalAmountCents: int|null, description: string|null,
                   amounts: { first: int, renewal: int } }   // montant appliqué selon le type, calculé par le serveur
SeasonRef        { id: int, label: string, state: "draft"|"open"|"closed" }
MemberRef        { id: int, firstName: string, lastName: string, age: int }

Payment { reference: string,
          status: PaymentStatus,
          announcedMethod: PaymentMethod|null,
          paidMethod: PaymentMethod|null, paidOn: date|null }

Membership { id: int,
             member: MemberRef, season: SeasonRef, fee: MembershipFeeRef,   // tarif actuel
             status: MembershipStatus,
             declaredKind: MembershipKind|null,
             kind: MembershipKind|null,                // type retenu (validée ou annulée après validation)
             amountCents: int,                         // montant appliqué si fixé, sinon montant calculé avec le tarif actuel et le type déclaré
             amountFixed: bool,
             payment: Payment|null,                    // null : non validée ou gratuite
             reason: string|null,
             requestedAt: datetime, decidedAt: datetime|null, cancelledAt: datetime|null }

MyMembership = Membership + {
             actions: { cancel: bool, pay: bool },
             paymentMethods: PaymentMethodInfo[] }     // moyens activés ; vide si rien à payer

PaymentMethodInfo =
    { method: "cash",     instructions: string }
  | { method: "cheque",   payee: string, address: string|null, instructions: string|null }
  | { method: "transfer", holder: string, iban: string, bic: string|null, instructions: string|null }

AdminMembership = Membership + {
             account: { id: int, email: string, disabled: bool }|null,   // compte qui gère la fiche
             requestedBy: AccountRef|null, createdBy: AccountRef|null,
             decidedBy: AccountRef|null, cancelledBy: AccountRef|null,
             paidBy: AccountRef|null, paymentNote: string|null,
             previousSeason: PreviousSeason,
             seasonFees: MembershipFeeRef[],          // tarifs de la saison, pour valider ou corriger
             actions: { validate, refuse, update, cancel, recordPayment, cancelPayment: bool } }
AccountRef { id: int, email: string }

MembershipOffer {
  seasons: [{ id, label, fees: MembershipFeeRef[] }],   // ouvertes, non closes, avec tarif ; plus récente d'abord
  members: [{ id, firstName, lastName, isSelf, availableSeasonIds: int[] }],
  canRequest: bool }                                     // au moins un couple (fiche, saison) disponible

MyMembershipList { seasons: [{ season: SeasonRef, items: MyMembership[] }] }   // saison la plus récente d'abord ; prénom replié

AdminMembershipPage { seasonId: int|null,                // saison retenue (null = toutes)
                      items: AdminMembershipRow[], total, page, pageSize }
AdminMembershipRow { id, member: MemberRef, season: SeasonRef, fee: { id, label },
                     status, kind|null, declaredKind|null, amountCents, payment: Payment|null,
                     requestedAt }

MembershipStats { seasonId: int,
                  validated: int, first: int, renewal: int, pending: int,
                  byFee: [{ feeId, label, count, amountCents }],
                  totalCents: int, paidCents: int,
                  dueCents: int,
                  dueByMethod: { none: int, cash: int, cheque: int, transfer: int } }  // centimes

MembershipDashboard { pendingCount: int, paymentSetupNeeded: bool }

PaymentSettings {
  cash:     { enabled: bool, instructions: string|null },
  cheque:   { enabled: bool, payee: string|null, address: string|null, instructions: string|null },
  transfer: { enabled: bool, holder: string|null, iban: string|null, bic: string|null, instructions: string|null },
  updatedAt: datetime }
```

## Corps

```text
MyMembershipBody   { memberId: int, seasonId: int, feeId: int, declaredKind: MembershipKind }
AnnounceBody       { method: PaymentMethod }
AdminCreateBody    { memberId: int, seasonId: int, feeId: int, kind: MembershipKind,
                     paid?: RecordPaymentBody }
DecisionBody       { feeId: int, kind: MembershipKind }          // validate-membership, update-membership
ReasonBody         { reason: string }                             // maxLength 500
RecordPaymentBody  { method: PaymentMethod, receivedOn: date, note?: string }   // note maxLength 200
PaymentSettingsBody = PaymentSettings sans updatedAt               // PUT complet ; textes maxLength : voir data-model
```

## Opérations `member` (tag `Mes adhésions`)

Le compte est celui de la session. Une adhésion d'une fiche gérée par un
autre compte, sans compte ou inexistante donne toujours `404` « Cette
adhésion n'existe plus. ».

| Opération | Méthode et chemin | Entrée | Réponses |
| --- | --- | --- | --- |
| `get-membership-offer` | `GET /api/me/memberships/offer` | — | `200 MembershipOffer` |
| `list-my-memberships` | `GET /api/me/memberships` | `member?` (fiche du compte) | `200 MyMembershipList` · `404` fiche |
| `get-my-membership` | `GET /api/me/memberships/{id}` | — | `200 MyMembership` · `404` |
| `create-my-membership` | `POST /api/me/memberships` | `MyMembershipBody` | `201 MyMembership` · `404` fiche · `409` déjà adhérent · `422` |
| `cancel-my-membership` | `POST /api/me/memberships/{id}/cancel` | — | `200 MyMembership` · `404` · `409` |
| `announce-my-payment` | `PUT /api/me/memberships/{id}/payment` | `AnnounceBody` | `200 MyMembership` · `404` · `409` · `422` |

## Opérations `admin` (tag `Adhésions`)

| Opération | Méthode et chemin | Entrée | Réponses |
| --- | --- | --- | --- |
| `list-memberships` | `GET /api/admin/memberships` | `season`, `status`, `payment`, `member`, `q`, `page`, `pageSize` | `200 AdminMembershipPage` · `422` |
| `get-membership-stats` | `GET /api/admin/memberships/stats` | `season` (obligatoire) | `200 MembershipStats` · `404` saison |
| `get-membership-dashboard` | `GET /api/admin/memberships/dashboard` | — | `200 MembershipDashboard` |
| `get-membership` | `GET /api/admin/memberships/{id}` | — | `200 AdminMembership` · `404` |
| `create-membership` | `POST /api/admin/memberships` | `AdminCreateBody` | `201 AdminMembership` · `404` fiche · `409` · `422` |
| `validate-membership` | `POST /api/admin/memberships/{id}/validate` | `DecisionBody` | `200 AdminMembership` · `404` · `409` · `422` |
| `refuse-membership` | `POST /api/admin/memberships/{id}/refuse` | `ReasonBody` | `200 AdminMembership` · `404` · `409` · `422` |
| `update-membership` | `PUT /api/admin/memberships/{id}` | `DecisionBody` | `200 AdminMembership` · `404` · `409` · `422` |
| `cancel-membership` | `POST /api/admin/memberships/{id}/cancel` | `ReasonBody` | `200 AdminMembership` · `404` · `409` · `422` |
| `record-membership-payment` | `POST /api/admin/memberships/{id}/payment` | `RecordPaymentBody` | `200 AdminMembership` · `404` · `409` · `422` |
| `cancel-membership-payment` | `DELETE /api/admin/memberships/{id}/payment` | — | `200 AdminMembership` · `404` · `409` |

Paramètres de `list-memberships` :

| Paramètre | Valeurs | Rôle |
| --- | --- | --- |
| `season` | identifiant, `all` ; absent = défaut ([R10](../research.md#r10-liste-recherche-et-chiffres-administration)) | saison |
| `status` | `pending`, `validated`, `refused`, `cancelled` | état |
| `payment` | `due`, `announced`, `paid`, `free` | état du paiement (adhésions validées) |
| `member` | identifiant de fiche | adhésions d'une fiche (fiche d'administration) |
| `q` | ≤ 100 caractères | fragments ≥ 3 caractères cherchés dans la référence de paiement (`memberships_fts`) ou dans la fiche (`members_fts` : prénom, nom, adresse, téléphone) |
| `page`, `pageSize` | ≥ 1 ; 1 à 100, défaut 25 | |

## Opérations `admin` (tag `Paramètres`)

| Opération | Méthode et chemin | Entrée | Réponses |
| --- | --- | --- | --- |
| `get-payment-settings` | `GET /api/admin/settings/payment` | — | `200 PaymentSettings` |
| `update-payment-settings` | `PUT /api/admin/settings/payment` | `PaymentSettingsBody` | `200 PaymentSettings` · `422` |

## Opérations modifiées

| Opération | Changement |
| --- | --- |
| `list-membership-fees` (004) | chaque `MembershipFee` gagne `inUse: bool` et `amounts: { first, renewal }` |
| `delete-membership-fee` (004) | `409` si le tarif est utilisé |
| `delete-my-member` (005) | `409` si la fiche porte une adhésion `pending` ou `validated` |
| `get-member` (005) | `AdminMember` gagne `memberships: [{ id, season: SeasonRef, status, payment: Payment|null }]` |
| `delete-member` (005) | supprime les adhésions de la fiche |
| `delete-account` (005, `members=delete`) | supprime aussi les adhésions des fiches |

## Erreurs de champ (`422`)

| Emplacement | Cas | `message` |
| --- | --- | --- |
| `body.seasonId` | saison non ouverte, close ou sans tarif | « Cette saison n'est pas ouverte à l'adhésion. » |
| `body.feeId` | tarif absent ou d'une autre saison | « Choisissez un tarif de cette saison. » |
| `body.reason` | vide après nettoyage | « Indiquez le motif. » |
| `body.method` | moyen désactivé (espace membre) | « Ce moyen de paiement n'est plus proposé. Choisissez-en un autre. » |
| `body.receivedOn` · `body.paid.receivedOn` | dans le futur | « La date de réception ne peut pas être dans le futur. » |
| `body.receivedOn` | avant la validation | « La date de réception ne peut pas précéder la validation de l'adhésion. » |
| `body.paid.receivedOn` | création directe : avant le début de la saison | « La date de réception ne peut pas précéder le début de la saison. » |
| `body.cash.instructions` | moyen activé sans instructions | « Indiquez où et quand régler en espèces. » |
| `body.cheque.payee` | moyen activé sans ordre | « Indiquez l'ordre des chèques. » |
| `body.transfer.holder` | moyen activé sans titulaire | « Indiquez le titulaire du compte. » |
| `body.transfer.iban` | absent (activé) | « Saisissez l'IBAN. » |
| `body.transfer.iban` | format ou clé invalide | « Cet IBAN n'est pas valide. Vérifiez-le sur votre relevé d'identité bancaire. » |
| `body.transfer.bic` | format invalide | « Saisissez un BIC de 8 ou 11 caractères. » |
| `query.q` | aucun fragment de 3 caractères | « Saisissez au moins 3 caractères. » (existant) |

## Autres erreurs

| Statut | Cas | `detail` |
| --- | --- | --- |
| `404` | adhésion absente ou hors du compte | « Cette adhésion n'existe plus. » |
| `404` | fiche absente ou hors du compte | « Ce membre n'existe plus. » (existant) |
| `409` | adhésion `pending` ou `validated` existante pour la fiche et la saison | « Ce membre a déjà une adhésion en cours pour cette saison. » |
| `409` | transition impossible dans l'état actuel | « Cette adhésion a changé entre-temps. » |
| `409` | correction d'une adhésion payée | « Annulez d'abord la constatation du paiement. » |
| `409` | action sur une saison close (correction, annulation) ; une création sur une saison close donne `422 body.seasonId` | « Cette saison est close. » |
| `409` | tarif utilisé (`delete-membership-fee`) | « Ce tarif est utilisé par au moins une adhésion : il ne peut pas être supprimé. » |
| `409` | fiche avec adhésion en cours (`delete-my-member`) | « Ce membre a une adhésion en cours. Annulez la demande ou contactez le bureau. » |

## Effets, e-mails et journal

| Opération | Effet | E-mail (compte actif de la fiche) | Journal (`INFO`) |
| --- | --- | --- | --- |
| `create-my-membership` | `pending` | — | `membership.request` |
| `cancel-my-membership` | `cancelled` | — | `membership.cancel` |
| `validate-membership` | `validated`, type, montant figé, référence et `due` si non gratuite | `membership-validated` (lien `/adhesions/{id}` si non gratuite) | `membership.validate` |
| `refuse-membership` | `refused`, motif | `membership-refused` | `membership.refuse` |
| `create-membership` | `validated` (+ `paid` si `paid` fourni) | `membership-validated` (lien si `due`) | `membership.create` (+ `membership.payment_record`) |
| `update-membership` | tarif, type, montant recalculé | — | `membership.update` |
| `cancel-membership` | `cancelled`, motif | `membership-cancelled` | `membership.cancel` |
| `announce-my-payment` | `announced`, moyen | — | `membership.payment_announce` |
| `record-membership-payment` | `paid` | `membership-paid` | `membership.payment_record` |
| `cancel-membership-payment` | `announced` ou `due` | — | `membership.payment_cancel` |
| `update-payment-settings` | paramètres remplacés | — | `settings.payment_update` (moyens activés) |

Attributs de journal : voir [R12](../research.md#r12-journalisation) ; jamais
de motif, de commentaire, de nom ni de coordonnées bancaires.
