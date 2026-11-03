# Contrat d'API : paiement par carte (HelloAsso)

Complète [006 › api.md](../../006-memberships/contracts/api.md) : mêmes
conventions (préfixe `/api`, camelCase, `problem+json` en français, `401`
sans session, `403` rôle insuffisant, `404` uniforme hors des fiches du
compte). Le schéma OpenAPI produit par `frigg openapi` fait foi.

## Types modifiés

```text
PaymentMethod = "card" | "cash" | "cheque" | "transfer"     // + card, en premier

Payment += { paidAmountCents: int|null,                      // carte : montant reçu
             online: OnlinePaymentRef|null }                 // carte : tentative qui a constaté
OnlinePaymentRef { orderId: int, paidAt: datetime, test: bool }

PaymentMethodInfo +=
  | { method: "card", amountCents: int, test: bool }        // proposé si disponible (R7)

MyMembership += {
  actions: { cancel, pay, payByCard: bool },
  cardPayment: { status: "pending"|"succeeded"|"failed"|"abandoned"|"expired",
                 startedAt: datetime }|null }                // dernière tentative du compte

AdminMembership += {
  onlinePayments: OnlinePayment[],                           // plus récente d'abord
  alerts: PaymentAlert[],                                    // non traités d'abord
  cardPaymentPending: bool,                                  // tentative en cours (< 60 min)
  actions: { …, verifyCardPayment: bool } }                  // cancelPayment=false si payée par carte

OnlinePayment { id: int, status: OnlineStatus, environment: "sandbox"|"production",
                amountCents: int, receivedAmountCents: int|null,
                orderId: int|null, paidAt: datetime|null, errorCode: string|null,
                createdAt: datetime, finishedAt: datetime|null,
                helloassoUrl: string|null }                  // lien vers la commande dans l'espace HelloAsso
PaymentAlert { id: int, kind: "duplicate"|"amount_mismatch"|"refund_due"|"unverified",
               onlinePaymentId: int, createdAt: datetime,
               resolvedAt: datetime|null, resolvedBy: AccountRef|null }

PaymentSettings += {
  card: { enabled: bool,                                     // choix de l'administrateur
          available: bool,                                   // proposée aux adhérents (R7)
          configured: bool,                                  // identifiants présents dans la configuration
          environment: "sandbox"|"production"|null,
          organization: string|null,                         // slug configuré
          notificationUrl: string|null,                      // à déclarer dans HelloAsso
          lastTest: { at: datetime, environment, ok: bool,
                      organizationName: string|null,
                      error: HelloAssoError|null }|null,
          failing: bool } }                                  // R9
HelloAssoError = "unauthorized" | "forbidden" | "not_found" | "unavailable" | "unexpected"

MembershipStats += { collectedByMethod.card: int, collectedTest: int }
MembershipDashboard += { paymentAlerts: int, helloassoFailing: bool }
```

## Opérations ajoutées

| Opération | Méthode et chemin | Accès | Rôle |
| --- | --- | --- | --- |
| `start-my-card-payment` | `POST /me/memberships/{id}/card-payment` | member | Lancer (ou reprendre) un paiement par carte |
| `confirm-my-card-payment` | `POST /me/memberships/{id}/card-payment/confirm` | member | Vérifier une tentative au retour de HelloAsso |
| `test-helloasso-connection` | `POST /admin/settings/payment/helloasso-test` | admin | Tester la connexion à HelloAsso |
| `verify-card-payment` | `POST /admin/memberships/{id}/card-payment/verify` | admin | Vérifier auprès de HelloAsso les tentatives en cours |
| `resolve-payment-alert` | `POST /admin/payment-alerts/{id}/resolve` | admin | Marquer un signalement comme traité |
| `receive-helloasso-notification` | `POST /webhooks/helloasso/{key}` | **public** | Recevoir une notification de HelloAsso (principe II, v1.7.0) |

### `start-my-card-payment`

Corps : aucun. Réponse `200` :

```text
{ redirectUrl: string, onlinePaymentId: int, resumed: bool }
```

- `404` : adhésion inexistante ou d'une fiche que le compte ne gère pas.
- `409` : adhésion non payable par carte (non validée, gratuite, payée,
  annulée) — `detail` explicite.
- `422` (`location: body`) : carte indisponible (désactivée, non configurée,
  test absent pour l'environnement) ou montant inférieur au minimum.
- `503` : HelloAsso injoignable ou identifiants refusés — « Le paiement par
  carte est momentanément indisponible. Choisissez un autre moyen ou
  réessayez plus tard. »

### `confirm-my-card-payment`

```text
Body { onlinePaymentId: int,
       outcome: "return" | "error" | "back",                 // d'après ?carte=retour|erreur|annule
       checkoutIntentId: int|null }                           // indicatif, comparé à la tentative
```

Réponse `200` : `{ status: OnlineStatus, membership: MyMembership }`.
Vérification R5 : seule la réponse de HelloAsso fait foi ; `outcome`
ne sert qu'à clore une tentative sans paiement (`error` → `failed`,
`back` → `abandoned`). Tentative d'un autre compte ou d'une autre adhésion
→ `404`. HelloAsso injoignable → `200` avec `status: "pending"` (la page
réessaie).

### `test-helloasso-connection`

Corps : aucun. Réponse `200` : `PaymentSettings` à jour (avec `lastTest`).
HelloAsso non configuré → `409` (« Renseignez les identifiants HelloAsso
dans la configuration de l'instance. »). Un échec de connexion n'est pas une
erreur HTTP : il est décrit dans `lastTest.error`.

### `update-payment-settings` (modifiée)

`PaymentSettingsBody += { card: { enabled: bool } }`. `card.enabled = true`
refusé en `422` sur `body.card.enabled` :

- non configuré : « Les identifiants HelloAsso ne sont pas renseignés dans
  la configuration de l'instance. » ;
- pas de test réussi pour l'environnement courant : « Testez d'abord la
  connexion à HelloAsso. »

### `verify-card-payment`

Corps : aucun. Vérifie chaque tentative `pending` (ou close sans paiement
depuis moins de 24 h) de l'adhésion. Réponse `200` : `AdminMembership`.
`503` si HelloAsso est injoignable.

### `resolve-payment-alert`

Corps : aucun. `200` : `AdminMembership` de l'adhésion concernée. Déjà
traité → `409`. Inexistant → `404`.

### `receive-helloasso-notification`

- Chemin : `{key}`, chaîne libre (aucun `pattern` huma), comparée en temps
  constant à la clé dérivée du secret (R6).
- Corps : lu brut (`RawBody`), 1 Mio au plus, surplus ignoré ; aucun schéma
  huma, donc jamais de `422` ni de `413`.
- Réponses (corps toujours vide, jamais de `problem+json`) :
  - clé absente, fausse ou mal formée → `200`, sans traitement ;
  - clé valide, notification traitée ou ignorée → `200` ;
  - clé valide, tentative connue, HelloAsso injoignable ou erreur interne →
    `503` (HelloAsso renvoie la notification) ;
  - débit dépassé → `429`.
- Effet : vérification R5 de la tentative désignée par
  `metadata.frigg.payment`, si elle existe dans l'environnement courant.
- Aucune session lue ni créée ; aucun en-tête révélant la version ou la
  configuration.

## Opérations modifiées (règles)

| Opération | Changement |
| --- | --- |
| `announce-my-payment` | `method: "card"` → `422` (« Le paiement par carte se fait en ligne. ») |
| `record-membership-payment` | `method: "card"` → `422` (constatation réservée à HelloAsso) |
| `cancel-membership-payment` | paiement par carte → `409` |
| `cancel-membership` | adhésion payée par carte → crée un signalement `refund_due` dans la même transaction |
| `get-my-membership`, `list-my-memberships` | `paymentMethods` inclut la carte si disponible ; `cardPayment`, `actions.payByCard` |
| `get-membership` | `onlinePayments`, `alerts`, `cardPaymentPending`, `actions.verifyCardPayment` |
| `list-memberships` | filtres `paidMethod` (dont `card`) et `alerts=open` ; `search` numérique → aussi égalité sur `online_payments.order_id` ; chaque élément porte `openAlerts: int` |
| `get-membership-stats` | `collectedByMethod.card`, `collectedTest` |
| `get-membership-dashboard` | `paymentAlerts`, `helloassoFailing` |

## Matrice d'autorisation

6 nouvelles opérations : 2 `member`, 3 `admin`, 1 `public`.
`allowedPublic` (`internal/api/access_test.go`) gagne
`"receive-helloasso-notification"` avec la mention de l'amendement 1.7.0.
`memberships_isolation_test.go` couvre `start-my-card-payment` et
`confirm-my-card-payment` (compte B → `404`, sans création de tentative).
