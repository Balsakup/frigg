# Contrat d'API : événements et inscriptions

Mêmes conventions que [006 › api.md](../../006-memberships/contracts/api.md)
et [008 › api.md](../../008-helloasso-card-payment/contracts/api.md) :
préfixe `/api`, camelCase, `problem+json` en français avec
`errors[].location` sur les `422`, `401` sans session, `403` rôle
insuffisant, `404` uniforme pour un objet absent ou hors des fiches du
compte. Le schéma OpenAPI produit par `frigg openapi` fait foi. Nouveaux tags
OpenAPI `events` et `registrations`.

## Types

```text
LocalDateTime     = string            // "2026-12-12T14:00", heure de Paris, sans fuseau (R8) : entrées
                                      // les sorties donnent des instants (date-time UTC) et, dans
                                      // AdminEvent.local, les valeurs locales du formulaire
EventStatus       = "draft" | "published" | "cancelled"
EventPhase        = "draft" | "upcoming" | "finished" | "cancelled"   // calculée
EventOpenness     = "season_members" | "all_members"
RegistrationState = "confirmed" | "cancelled"
Ineligibility     = "not_season_member" | "already_registered"

EventBody {                                   // création et modification
  title: string,                              // 1..120
  location: string,                           // 1..200
  description: string|null,                   // ≤ 5000
  startsAt: LocalDateTime, endsAt: LocalDateTime,
  openness: EventOpenness,
  capacity: int|null,                         // 1..10000
  price: string,                              // euros saisis, « 35 » ou « 15,50 » ; "" ou « 0 » = gratuit ;
                                              // analysé par le serveur comme le montant d'un tarif (004), 0..10000 €
  registrationOpensAt: LocalDateTime|null,
  registrationClosesAt: LocalDateTime|null
}

EventSummary {
  id: int, title, location, startsAt, endsAt,
  season: SeasonRef, status: EventStatus, phase: EventPhase,
  openness, capacity: int|null, priceCents: int,
  confirmedCount: int, placesLeft: int|null,   // null : illimité
  registrationOpen: bool                       // période ouverte maintenant
}

AdminEvent = EventSummary + {
  description, registrationOpensAt, registrationClosesAt,
  cancelMessage: string|null, publishedAt, cancelledAt, cancelledBy: AccountRef|null,
  createdBy: AccountRef|null, updatedBy: AccountRef|null,
  totals: { dueCents: int, collectedCents: int, collectedTestCents: int,
            paidCount: int, paidAmountCents: int },  // inscriptions confirmées (FR-026)
  actions: { update: bool, updateDates: bool, publish: bool, cancel: bool,
             restore: bool, delete: bool, register: bool }
}

MyEvent = EventSummary + {
  description, registrationOpensAt, registrationClosesAt, cancelMessage,
  members: [{                                  // fiches gérées par le compte
    member: MemberRef,
    registration: MyRegistrationRef|null,      // inscription confirmée
    eligible: bool, reason: Ineligibility|null
  }]
}

Payment                                        // repris tel quel de 006/008
PaymentMethodInfo[]                            // idem

AdminRegistration {
  id: int, event: EventRef, member: MemberRef,  // MemberRef : id, firstName, lastName
  isSeasonMember: bool,                         // adhérent de la saison de l'événement
  eligible: bool,                               // selon l'ouverture actuelle (R7)
  contactEmail: string|null,                    // null : non joignable
  state: RegistrationState,
  registeredAt, registeredBy: AccountRef|null,
  cancelledAt, cancelledBy: AccountRef|null,
  amountCents: int, paymentReference: string|null,
  payment: Payment|null,                        // null : gratuite
  paymentNote: string|null,
  onlinePayments: OnlinePayment[], alerts: PaymentAlert[],   // détail seulement
  openAlerts: int,                              // signalements non traités (liste et détail)
  cardPaymentPending: bool,
  actions: { remove: bool, recordPayment: bool, cancelPayment: bool,
             verifyCardPayment: bool }
}

MyRegistration {
  id: int, event: EventSummary, member: MemberRef,
  state, registeredAt, amountCents, paymentReference, payment: Payment|null,
  paymentMethods: PaymentMethodInfo[],          // page de paiement (détail seulement)
  cardPayment: { status, startedAt }|null,
  actions: { cancel: bool, pay: bool, payByCard: bool }
}
MyRegistrationRef { id: int, payment: Payment|null, actions }

Due {                                           // « À régler » (R11)
  kind: "membership" | "registration", id: int,
  member: MemberRef, label: string,             // « Adhésion 2026-2027 » | titre de l'événement
  amountCents: int, reference: string,
  payment: Payment, createdAt: datetime
}
```

## Opérations `member` (espace membre)

| Opération | Méthode et chemin | Entrée | Réponses |
| --- | --- | --- | --- |
| `list-my-events` | `GET /me/events` | — | `200 EventSummary[]` (publiés ou annulés non terminés, par début croissant) |
| `get-my-event` | `GET /me/events/{id}` | — | `200 MyEvent` · `404` (brouillon, inexistant) |
| `register-my-members` | `POST /me/events/{id}/registrations` | `{ memberIds: int[] }` (1..20, distincts) | `201 RegisterResult` · `404` · `409` · `422` |
| `list-my-registrations` | `GET /me/registrations` | — | `200 MyRegistration[]` (sans `paymentMethods`) |
| `get-my-registration` | `GET /me/registrations/{id}` | — | `200 MyRegistration` · `404` |
| `cancel-my-registration` | `POST /me/registrations/{id}/cancel` | — | `200 MyRegistration` · `404` · `409` |
| `announce-my-registration-payment` | `PUT /me/registrations/{id}/payment` | `AnnounceBody` (006) | `200 MyRegistration` · `404` · `409` · `422` |
| `start-my-registration-card-payment` | `POST /me/registrations/{id}/card-payment` | — | `200 StartCardPayment` (008) · `404` · `409` · `422` · `503` |
| `confirm-my-registration-card-payment` | `POST /me/registrations/{id}/card-payment/confirm` | `ConfirmCardPaymentBody` (008) | `200 { status, registration: MyRegistration }` · `404` |
| `list-my-dues` | `GET /me/dues` | — | `200 Due[]` |

`register-my-members` :

- `201` : `RegisterResult { event: MyEvent, createdRegistrationIds: int[] }`,
  ids des inscriptions créées dans l'ordre de `memberIds` ; le frontend
  redirige vers `/mes-inscriptions/{createdRegistrationIds[0]}` pour un
  événement payant ;
- `404` : événement brouillon ou inexistant, ou une fiche que le compte ne
  gère pas (sans dire laquelle) ;
- `409` : événement annulé ou terminé, période d'inscription close
  (« Les inscriptions sont closes. »), **complet** (« Complet : il reste
  n place(s). », tout ou rien, R6) ;
- `422` sur `body.memberIds[i]` : fiche non éligible (« Lucas n'est pas
  adhérent de la saison 2026-2027. ») ou déjà inscrite.

`cancel-my-registration` : `409` si la période est close, si l'inscription
est payée ou a un paiement par carte en cours (« Cette inscription est
réglée : contactez l'association pour l'annuler. »), ou déjà annulée.

## Opérations `admin`

| Opération | Méthode et chemin | Entrée | Réponses |
| --- | --- | --- | --- |
| `list-events` | `GET /admin/events` | `season` (id \| `all`), `status` (`draft`,`published`,`cancelled`,`finished`), `when` (`upcoming` \| `past`) | `200 EventSummary[]` |
| `get-event` | `GET /admin/events/{id}` | — | `200 AdminEvent` · `404` |
| `create-event` | `POST /admin/events` | `EventBody` | `201 AdminEvent` · `422` |
| `update-event` | `PUT /admin/events/{id}` | `EventBody` | `200 AdminEvent` · `404` · `409` · `422` |
| `publish-event` | `POST /admin/events/{id}/publish` | — | `200 AdminEvent` · `404` · `409` |
| `cancel-event` | `POST /admin/events/{id}/cancel` | `{ message: string\|null }` (≤ 1000) | `200 AdminEvent` · `404` · `409` |
| `restore-event` | `POST /admin/events/{id}/restore` | — | `200 AdminEvent` · `404` · `409` |
| `delete-event` | `DELETE /admin/events/{id}` | — | `204` · `404` · `409` (« Cet événement a des inscriptions : annulez-le plutôt. ») |
| `list-event-registrations` | `GET /admin/events/{id}/registrations` | `state` (`confirmed` défaut \| `all`) | `200 AdminRegistration[]` (sans tentatives) · `404` |
| `export-event-registrations` | `GET /admin/events/{id}/registrations.csv` | — | `200 text/csv` · `404` |
| `register-member` | `POST /admin/events/{id}/registrations` | `{ memberId: int, payment: RecordPaymentBody\|null }` | `201 AdminRegistration` · `404` · `409` · `422` |
| `list-registrations` | `GET /admin/registrations` | `event`, `state`, `payment` (`due`,`announced`,`paid`,`free`), `alerts` (`open`), `q` (≥ 3 car.), `page`, `pageSize` | `200 AdminRegistrationPage` · `422` |
| `get-registration` | `GET /admin/registrations/{id}` | — | `200 AdminRegistration` · `404` |
| `remove-registration` | `POST /admin/registrations/{id}/cancel` | — | `200 AdminRegistration` · `404` · `409` |
| `record-registration-payment` | `POST /admin/registrations/{id}/payment` | `RecordPaymentBody` (006) | `200 AdminRegistration` · `404` · `409` · `422` |
| `cancel-registration-payment` | `DELETE /admin/registrations/{id}/payment` | — | `200 AdminRegistration` · `404` · `409` |
| `verify-registration-card-payment` | `POST /admin/registrations/{id}/card-payment/verify` | — | `200 AdminRegistration` · `404` · `503` |
| `resolve-registration-payment-alert` | `POST /admin/registration-alerts/{id}/resolve` | — | `200 AdminRegistration` · `404` · `409` |

Règles :

- `update-event` : `409` si l'événement est terminé et que les dates
  changent, si la capacité passe sous le nombre d'inscrits (`422` sur
  `body.capacity` avec le nombre actuel), ou si l'événement est annulé ;
  `422` champ par champ pour les validations de FR-002/FR-003.
- `publish-event` : seulement depuis `draft` et avant la fin.
- `register-member` : hors période autorisé, avant la fin ; capacité et
  éligibilité contrôlées (`409` complet, `422` sur `body.memberId`) ;
  `payment` non nul → inscription créée directement « Payée » (gratuite :
  `payment` ignoré).
- `remove-registration` : avant la fin de l'événement ; payée ou carte en
  cours acceptée (le dialogue a prévenu).
- `cancel-registration-payment` : `409` pour un paiement par carte.
- Le `q` de `list-registrations` cherche dans `members_fts` (nom, prénom) et
  `event_registrations_fts` (référence).

## Opérations modifiées

| Opération | Changement |
| --- | --- |
| `get-membership-dashboard` | `paymentAlerts` ne compte que les signalements d'adhésions (inchangé en pratique) ; nouveau `registrationPaymentAlerts: int` pour les inscriptions ; l'encart du tableau de bord renvoie vers la liste de chaque cible filtrée sur `alerts=open` |
| `resolve-payment-alert` | réservée aux signalements d'adhésion : un signalement d'inscription renvoie `404` (il a sa propre opération, `resolve-registration-payment-alert`, qui renvoie l'inscription) |
| `update-season`, `delete-season` | `409` si un événement sortirait des bornes ou si la saison a des événements (R8) |
| `get-member`, `get-my-member` | `paidUpcomingRegistrations: int` pour le dialogue de suppression (R14) |
| `receive-helloasso-notification` | inchangée en surface ; traite les tentatives d'inscription (R4) |

## Rôles déclarés

10 opérations `member`, 18 opérations `admin`, **aucune publique**. La
matrice d'autorisation automatique couvre les 28 opérations.

## E-mails (file `outbound_emails`)

| Déclencheur | Modèle | Destinataire | Contenu |
| --- | --- | --- | --- |
| `register-my-members`, `register-member` | `registration-confirmed` | adresse de contact, une par demande | événement, date, lieu, inscrits ; si payante : montant, référence, lien `/mes-inscriptions/{id}` |
| `record-registration-payment`, carte constatée | `registration-paid` | adresse de contact | montant, moyen, date, référence |
| `remove-registration` | `registration-removed` | adresse de contact | événement, membre ; rappel remboursement si payée |
| `update-event` (début, fin ou lieu, publié, non terminé) | `event-changed` | une par adresse (R10) | anciennes et nouvelles valeurs, inscrits nommés |
| `cancel-event` | `event-cancelled` | une par adresse | message, inscrits ; mention de remboursement si payée |
| `restore-event` | `event-restored` | une par adresse | date, lieu, inscrits |
| remboursement HelloAsso vérifié | `registration-removed` | adresse de contact | motif « remboursé » |

## Journaux

| Opération | Message | Attributs |
| --- | --- | --- |
| `create-event`, `update-event`, `publish-event`, `cancel-event`, `restore-event`, `delete-event` | « événement créé », « … modifié », … | `event_id`, `actor_id`, `notified` (nombre d'e-mails) |
| `register-*`, `cancel-my-registration`, `remove-registration` | « inscription enregistrée », « inscription annulée » | `event_id`, `registration_id`, `actor_id`, `payment_reference` |
| paiements | repris de 006/008 avec `registration_id` à la place de `membership_id` | |

Jamais de titre, lieu, description, nom, adresse électronique ni message
d'annulation (R15).
