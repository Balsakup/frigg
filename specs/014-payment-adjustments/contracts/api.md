# Contrat d'API : Remises et dons sur les paiements

Source de vérité : le schéma OpenAPI généré (`frigg openapi`). Ce document
décrit les écarts avec 006, 008 et 012. Toutes les opérations exigent une
session. Erreurs de champ : 422 huma avec `errors[].location`.

## Nouvelle opération

| Operation ID | Méthode et chemin | Rôle | Résumé |
| --- | --- | --- | --- |
| `adjust-registration-amount` | `PUT /admin/registrations/{id}/amount` | admin | Ajuster le montant à payer d'une inscription |

Corps :

```json
{ "amount": "20", "adjustmentReason": "bénévole encadrant" }
```

- `amount` : texte, `maxLength 12`, en euros (« 20 », « 15,50 »). Vide :
  montant théorique (prix figé à l'inscription).
- `adjustmentReason` : texte, `maxLength 200`, obligatoire si le montant
  diffère du théorique.

Réponse `200` : `AdminRegistration`. Erreurs :

- `404` : inscription inexistante ;
- `409` (`MsgRegistrationChanged`) : inscription annulée, payée, ou
  événement annulé ;
- `422` sur `body.amount` (format, trop élevé) ou sur
  `body.adjustmentReason` (manquant, trop long).

## Corps modifiés (champs facultatifs ajoutés)

| Type | Champs ajoutés | Utilisé par |
| --- | --- | --- |
| `DecisionBody` | `amount?: string` (maxLength 12), `adjustmentReason?: string` (maxLength 200) | `validate-membership` (via `ValidateBody`), `update-membership` |
| `AdminCreateBody` | `amount?`, `adjustmentReason?` | `create-membership` |
| `RegisterMemberInput.Body` | `amount?`, `adjustmentReason?` | `register-member` |
| `RecordPaymentBody` | `receivedAmount?: string` (maxLength 12) | `record-membership-payment`, `record-registration-payment`, `create-membership.paid`, `register-member.payment` |

Règles communes :

- `amount` absent ou vide : montant théorique ; égal au théorique :
  `adjustmentReason` ignoré.
- `receivedAmount` absent ou vide : montant à payer ; inférieur au montant à
  payer : 422 sur `<location>.receivedAmount`, avec le message
  « Le montant reçu ne peut pas être inférieur au montant à payer. Ajustez
  d'abord le montant à payer. ».
- Messages de champ (`errors.go`) :
  - `MsgAdjustmentReasonRequired` : « Indiquez le motif de l'ajustement. » ;
  - `MsgAdjustmentReasonTooLong` : « Le motif ne doit pas dépasser 200
    caractères. » ;
  - `MsgReceivedBelowDue` : voir ci-dessus ;
  - format et plafond : messages existants des montants (tarifs).
- `update-membership` sur une adhésion payée : `409 MsgPaidCorrection`
  (règle existante).

## Réponses modifiées

| Type | Champs ajoutés | Visible par |
| --- | --- | --- |
| `Payment` (partagé) | `donationCents: int64` (0 sans don) ; `paidAmountCents` désormais renseigné pour tout paiement constaté | membre et admin |
| `AdminMembership` | `baseAmountCents: int64` (montant théorique, toujours renseigné pour une adhésion validée), `adjustmentCents: int64` (montant à payer − théorique : négatif = remise, positif = majoration, 0 sans ajustement), `adjustmentReason: string \| null` | admin |
| `AdminMembershipRow` | `baseAmountCents: int64 \| null` (null tant que non validée), `adjustmentCents: int64` | admin |
| `AdminRegistration` | `baseAmountCents: int64`, `adjustmentCents: int64`, `adjustmentReason: string \| null`, `actions.adjustAmount: bool` | admin |
| `AdminRegistrationRow` | `baseAmountCents: int64`, `adjustmentCents: int64` | admin |
| `MembershipStats` | `theoreticalCents`, `discountCents`, `surchargeCents`, `donationCents` | admin |
| `EventTotals` | `theoreticalCents`, `discountCents`, `surchargeCents`, `donationCents`, `remainingCents` | admin |

Sens des champs existants, sans changement de nom :

- `amountCents` (adhésion, inscription) : montant à payer ;
- `MembershipStats.totalCents` : montant total à payer ; `paidCents` :
  encaissé, dons compris ; `dueCents` : restant à percevoir ;
- `EventTotals.dueCents` : montant total à payer ; `collectedCents` :
  encaissé, dons compris.

Le type `Membership` (espace membre) et les inscriptions de l'espace membre
ne contiennent **ni** `baseAmountCents`, **ni** `adjustmentCents`, **ni**
`adjustmentReason` (R9).

Une demande d'adhésion faite depuis l'espace membre (`POST /me/memberships`)
n'accepte pas de montant : un champ `amount` envoyé est refusé (422, champ
inconnu) ou sans effet, et le montant reste fixé à la validation par un
administrateur. Un
test vérifie la réponse brute.

## Export CSV des inscrits (`export-event-registrations`)

Colonnes, dans l'ordre : Nom, Prénom, Date de naissance, Téléphone, Adresse
de contact, Adhérent, Date d'inscription, **Montant théorique**, **Montant à
payer** (anciennement « Montant dû »), **Motif d'ajustement**, Paiement,
Moyen, **Montant reçu**, **Don**, Référence. Le motif passe par
`csvfile.SafeCell`. Une inscription gratuite sans ajustement a des montants
vides, comme aujourd'hui.

## E-mails

`membership-paid` et `registration-paid` (txt et html) : « Montant : » donne
le montant reçu ; une ligne « Dont X € de don : merci pour votre soutien ! »
s'ajoute s'il y a un don.
