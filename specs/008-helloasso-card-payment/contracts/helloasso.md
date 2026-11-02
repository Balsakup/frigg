# Contrat sortant : appels de Frigg à HelloAsso

Ce que Frigg envoie à HelloAsso et ce qu'il en lit. Faits sources :
[research.md › R1](../research.md#r1-ce-que-fournit-helloasso-faits-vérifiés).
`{hôte}` = `https://api.helloasso.com` (production),
`https://api.helloasso-sandbox.com` (sandbox) ou `FRIGG_HELLOASSO_API_URL`
(tests).

## Jeton

```text
POST {hôte}/oauth2/token
Content-Type: application/x-www-form-urlencoded
grant_type=client_credentials&client_id=…&client_secret=…
  | grant_type=refresh_token&client_id=…&refresh_token=…
→ 200 { access_token, refresh_token, expires_in, token_type: "bearer" }
```

Jeton gardé en mémoire, renouvelé 60 s avant `expires_in`. Corps de réponse
jamais journalisé.

## Test de connexion

```text
GET {hôte}/v5/organizations/{slug}
Authorization: Bearer …
→ 200 { name, … }            lu : name
→ 401 → unauthorized   403 → forbidden   404 → not_found
→ réseau, délai > 10 s, 5xx → unavailable   autre → unexpected
```

## Création d'une intention

```text
POST {hôte}/v5/organizations/{slug}/checkout-intents
{
  "totalAmount": 4000, "initialAmount": 4000,
  "itemName": "Adhésion 2026-2027 — Jeune — ADH-2627-0042",
  "backUrl":   "{BASE_URL}/adhesions/42?carte=annule&tentative=7",
  "errorUrl":  "{BASE_URL}/adhesions/42?carte=erreur&tentative=7",
  "returnUrl": "{BASE_URL}/adhesions/42?carte=retour&tentative=7",
  "containsDonation": false,
  "payer": { "firstName": "Claire", "lastName": "Martin", "email": "claire@…" },
  "metadata": { "frigg": { "payment": 7, "membership": 42, "reference": "ADH-2627-0042" } }
}
→ 200 { "id": 123456, "redirectUrl": "https://www.helloasso.com/…" }
```

- `payer.firstName`/`lastName` omis si le compte n'a pas de fiche « Vous ».
- Aucune nouvelle tentative automatique sur cet appel.
- `redirectUrl` refusée si son hôte n'est pas un hôte HelloAsso attendu.

## Lecture d'une intention

```text
GET {hôte}/v5/organizations/{slug}/checkout-intents/{id}
→ 200 {
    "id": 123456,
    "metadata": { "frigg": { "payment": 7, "membership": 42, … } },
    "order": {                                  // absent tant que non autorisé
      "id": 98765,
      "payments": [ { "id": 555, "amount": 4000, "amountTip": 300,
                      "state": "Authorized", "date": "2026-11-03T18:12:00+01:00" } ]
    }
  }
```

Lu : `metadata.frigg.*`, `order.id`, le premier paiement `state =
"Authorized"` : `id`, `amount − amountTip`, `date`. Tout autre champ est
ignoré (décodage tolérant).

## Notification reçue (entrant, pour mémoire)

```text
POST {BASE_URL}/api/webhooks/helloasso/{key}
{ "eventType": "Order" | "Payment" | …, "data": { … }, "metadata": { "frigg": { "payment": 7, … } } }
```

Lu : `metadata.frigg.payment` uniquement (emplacement exact confirmé en
environnement de test, R1). Le reste du corps n'est jamais utilisé pour
décider.
