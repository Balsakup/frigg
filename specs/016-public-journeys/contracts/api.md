# Contrat d'API : parcours publics

Ce contrat reprend les conventions de
[012 › api.md](../../012-events/contracts/api.md) :

- préfixe `/api` et champs en camelCase ;
- erreurs en `problem+json`, en français, avec `errors[].location` sur les
  `422` ;
- `404` uniforme pour un objet absent ou hors périmètre.

Le schéma OpenAPI produit par `frigg openapi` fait foi.

Niveaux d'accès (principe III, v1.9.0) :

- `public` : sans session, limité aux exceptions du principe II ;
- `link` : accès par lien personnel en session ([R1](../research.md)). Sans
  accès valide, la réponse est `401`, même pour un compte connecté ;
- `member` et `admin` : inchangés.

Nouveaux tags OpenAPI : `public`, `tracking`.

## Types

```text
PublicIdentity = Identity                     // get-identity existant (nom, couleurs, logo)

PublicFee { id: int, label: string, description: string|null,
            firstAmountCents: int, renewalAmountCents: int|null }   // 004
PublicSeason { id: int, label: string, fees: PublicFee[] }
PublicCustomField = CustomFieldDefinition     // champs « Rempli par le membre » actifs,
                                              // sans visibilité ni ordre d'administration

PublicMembershipPage {
  open: bool,                                 // FR-005
  seasons: PublicSeason[],                    // vide si !open
  memberFields: PublicCustomField[], membershipFields: PublicCustomField[]
}

PublicEvent {                                 // FR-025, principe II : liste fermée
  title, location, description: string|null, startsAt, endsAt,
  priceCents: int, registrationOpensAt, registrationClosesAt,
  placesLeft: int|null,
  // États dérivés, sans donnée supplémentaire :
  registrationOpen: bool,
  closedReason: "cancelled" | "finished" | "full" | "not_yet_open" | "closed" | null
}                                             // ni message d'annulation ni champ personnalisé

IdentityInput {                               // 005 FR-001, FR-002
  firstName, lastName, birthDate: "AAAA-MM-JJ",
  email: string,                              // obligatoire
  phone: string|null, address: { line1, line2|null, postalCode, city }|null
}

SubmissionCommon = IdentityInput + {
  customFields: { [fieldKey]: value },        // même forme que 011
  consent: true,                              // obligatoire
  createAccount: bool,
  website: string                             // piège : doit être vide (R10)
}

ConfirmOutcome {
  outcome: "tracking" | "account" | "duplicate" | "closed",
  kind: "membership" | "registration",
  reason: string|null,                        // si closed : message à afficher
  createAccount: bool                         // tracking : proposer l'écran de mot de passe
}

TrackedItem {                                 // FR-015
  kind: "membership" | "registration",
  id: int, member: MemberRef,
  label: string,                              // « Adhésion 2026-2027 — Tarif plein » ou titre de l'événement
  status: string,                             // états de 006 ou de 012
  amountCents: int|null, paymentStatus: "due"|"announced"|"paid"|null,
  reference: string|null, reason: string|null,
  actions: { cancel: bool, pay: bool }
}

Tracking { email: string, items: TrackedItem[], canCreateAccount: bool }
```

`TrackedMembership` et `TrackedRegistration` reprennent `MyMembership` et
`MyRegistration`, sans champ réservé au compte (`requestedBy`, etc.). Ils
portent les mêmes informations de paiement (moyens activés, instructions,
montant à payer, référence, tentatives de carte).

## Opérations publiques (7)

Toutes sont soumises à `CrossOriginProtection` pour les `POST` et aux
limiteurs de [R9](../research.md).

| OperationID | Méthode et chemin | Corps | Réponses |
| --- | --- | --- | --- |
| `get-public-membership-page` | `GET /public/membership` | — | `200 PublicMembershipPage` |
| `submit-public-membership` | `POST /public/membership` | `SubmissionCommon + { seasonId, feeId, declaredKind: "first"\|"renewal" }` | `202` corps neutre ; `409` accueil fermé ; `422` saisie ; `429` |
| `get-public-event` | `GET /public/events/{slug}` | — | `200 PublicEvent` ; `404` uniforme (brouillon, non public, slug inconnu) |
| `submit-public-registration` | `POST /public/events/{slug}/registrations` | `SubmissionCommon` sans `customFields` (FR-026) | `202` neutre ; `404` ; `409` inscription impossible (raison) ; `422` ; `429` |
| `confirm-public-submission` | `POST /public/submissions/confirm` | `{ token }` | `200 ConfirmOutcome` (ouvre l'accès par lien si `tracking`) ; `410` lien invalide ou expiré ; `429` |
| `request-tracking-link` | `POST /public/tracking/link-request` | `{ email }` | `202` neutre, toujours ; `422` adresse mal formée ; `429` |
| `open-tracking-link` | `POST /public/tracking/open` | `{ token }` | `204` (accès en session) ; `410` lien expiré, révoqué ou inconnu ; `429` |

Le corps neutre de `202` vaut toujours
`{ "message": "Si l'adresse est valide, un e-mail vient de vous être envoyé." }`.
Il est identique pour le piège, la limite par adresse, l'adresse d'un compte
et l'adresse inconnue (SC-005).

## Opérations par lien personnel (13)

| OperationID | Méthode et chemin | Effet |
| --- | --- | --- |
| `get-tracking` | `GET /tracking` | Espace de suivi (`Tracking`) |
| `get-tracking-membership` | `GET /tracking/memberships/{id}` | `TrackedMembership` |
| `cancel-tracking-membership` | `POST /tracking/memberships/{id}/cancel` | Annule une demande en attente (006 FR-009) |
| `announce-tracking-membership-payment` | `PUT /tracking/memberships/{id}/payment` | `{ method }` : annonce un moyen (006 FR-022b) |
| `start-tracking-membership-card-payment` | `POST /tracking/memberships/{id}/card-payment` | `StartCardPayment` (008) |
| `confirm-tracking-membership-card-payment` | `POST /tracking/memberships/{id}/card-payment/confirm` | `ConfirmCardPaymentBody` → état vérifié |
| `get-tracking-registration` | `GET /tracking/registrations/{id}` | `TrackedRegistration` |
| `cancel-tracking-registration` | `POST /tracking/registrations/{id}/cancel` | 012 FR-013 |
| `announce-tracking-registration-payment` | `PUT /tracking/registrations/{id}/payment` | 012 FR-022 |
| `start-tracking-registration-card-payment` | `POST /tracking/registrations/{id}/card-payment` | 012 FR-023 |
| `confirm-tracking-registration-card-payment` | `POST /tracking/registrations/{id}/card-payment/confirm` | idem |
| `create-tracking-account` | `POST /tracking/account` | `{ password, selfMemberId: int\|null }` → `201` connecté ; `409` compte existant ou plus de 20 fiches ; `422` mot de passe (001 FR-008) |
| `close-tracking` | `POST /tracking/close` | `204` : retire l'accès par lien de la session |

Toute ressource hors du périmètre du lien renvoie `404` (FR-014). Aucune
opération par lien ne modifie une identité, des coordonnées ou une valeur de
champ (FR-016).

## Opérations d'administration ajoutées (4)

| OperationID | Méthode et chemin | Effet |
| --- | --- | --- |
| `regenerate-event-public-link` | `POST /admin/events/{id}/public-link` | Nouveau slug ; `409` si l'événement n'est pas « Ouvert au public » (FR-024) |
| `attach-membership-to-member` | `POST /admin/memberships/{id}/attach` | `{ memberId }` ; `409` si la demande n'est pas publique en attente ou si 006 FR-003 l'interdit (FR-038) |
| `send-tracking-link` | `POST /admin/members/{id}/tracking-link` | `204` ; `409` si la fiche a un compte ou n'a pas d'adresse (FR-040) ; `429` au-delà du plafond d'e-mails de l'adresse (`PublicMailEmail`, R9) |
| `delete-inactive-members` | `POST /admin/members/inactive/delete` | `{ ids: int[] }` (1..500) → `{ deleted: int, skipped: int }` (FR-041) |

## Opérations modifiées

| OperationID | Modification |
| --- | --- |
| `get-general-settings`, `update-general-settings` | + `publicMembershipEnabled: bool` ; en lecture, + `publicMembershipUrl: string` (FR-004) |
| `create-event`, `update-event`, `get-event`, `list-events` | `openness` accepte `"public"` ; `AdminEvent` + `publicUrl: string\|null` (FR-023, FR-024) |
| `get-my-event`, `list-my-events`, `register-my-members` | `openness: "public"` traité comme `"all_members"` (FR-023) |
| `list-members` | filtres `origin=public` et `inactive=true` ; `AdminMemberSummary` + `origin`, `inactive: bool` |
| `get-member`, `update-member` | `AdminMember` + `origin`, `emailConfirmedAt`, `consentedAt`, `trackedByLink: bool` ; modifier l'adresse remet `emailConfirmedAt` à `null` |
| `list-memberships`, `get-membership` | filtre `origin=public` ; `AdminMembership` + `viaPublic`, `duplicateCandidates: MemberRef[]` pour une demande publique en attente (FR-037, FR-038) |
| `list-registrations`, `get-registration` | filtre `origin=public` ; + `viaPublic` |
| `get-membership-dashboard` | + `inactivePublicCount: int` (FR-041) |
| `start-my-card-payment`, `start-my-registration-card-payment` | inchangées en contrat ; `openCheckout` reçoit le payeur explicitement ([R11](../research.md)) |

## Récapitulatif

24 opérations ajoutées :

| Accès | Nombre |
| --- | --- |
| `public` | 7 |
| `link` | 13 |
| `admin` | 4 |

16 opérations sont modifiées.
