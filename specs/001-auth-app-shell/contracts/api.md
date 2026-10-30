# Contrat d'API : Authentification et shell applicatif

Préfixe `/api`, JSON, champs en camelCase anglais. Le schéma OpenAPI produit par
`frigg openapi` fait foi ; ce document fixe l'intention avant implémentation.

## Conventions

- **Accès** : chaque opération déclare `public`, `member` ou `admin`
  (`admin ⊇ member`). Sans session valide : `401`. Rôle insuffisant : `403`.
- **Erreurs** : `application/problem+json` (RFC 9457, format huma) avec `title`
  et `detail` en français ; erreurs de champ dans `errors[]` avec `location`
  (`body.email`, `body.password`…) et `message` en français.
- **Limitation de débit** : `429` avec `detail` « Trop de tentatives. Réessayez
  dans quelques minutes. » et en-tête `Retry-After`.
- **CSRF** : toute méthode non sûre est soumise à `http.CrossOriginProtection`.
- **Jetons** : toujours dans le corps des requêtes, jamais dans l'URL.

### Types partagés

```text
Account       { id: int, email: string, role: "member"|"admin",
                emailVerified: bool }
Association   { name: string }            // « Mon association » si non défini
Session       { account: Account, association: Association }
Invitation    { id: int, email: string, role: "member"|"admin",
                status: "pending"|"expired", expiresAt: datetime,
                invitedBy: string|null, createdAt: datetime }
```

## Opérations publiques (liste exhaustive — principe II)

| Opération | Méthode et chemin | Corps | Réponses |
| --- | --- | --- | --- |
| `login` | `POST /api/auth/login` | `{ email, password }` | `200 Session` (cookie posé, jeton renouvelé) · `401` « Adresse électronique ou mot de passe incorrect » · `422` · `429` |
| `request-password-reset` | `POST /api/auth/password-reset` | `{ email }` | `202` toujours (même réponse que l'adresse existe ou non) · `422` format · `429` |
| `confirm-password-reset` | `POST /api/auth/password-reset/confirm` | `{ token, password }` | `204` (époque incrémentée : toutes les sessions fermées) · `410` « Ce lien n'est plus valable » · `422` règle de mot de passe · `429` |
| `confirm-email` | `POST /api/auth/email-verification/confirm` | `{ token }` | `204` · `410` · `429` |
| `get-invitation` | `POST /api/auth/invitation/lookup` | `{ token }` | `200 { email, role }` · `410` « Cette invitation n'est plus valable » · `429` |
| `accept-invitation` | `POST /api/auth/invitation/accept` | `{ token, password }` | `200 Session` (compte créé, adresse confirmée, session ouverte) · `410` · `409` compte déjà existant · `422` · `429` |

`get-invitation` est un POST pour garder le jeton hors de l'URL.

## Opérations `member`

| Opération | Méthode et chemin | Corps | Réponses |
| --- | --- | --- | --- |
| `get-session` | `GET /api/session` | — | `200 Session` |
| `logout` | `POST /api/auth/logout` | — | `204` (session détruite, cookie expiré) |
| `resend-email-verification` | `POST /api/me/email-verification` | — | `202` · `409` adresse déjà confirmée · `429` |
| `change-password` | `PUT /api/me/password` | `{ currentPassword, newPassword }` | `204` (époque incrémentée ; la session courante reçoit la nouvelle époque, les autres sont fermées) · `422` mot de passe actuel incorrect (`body.currentPassword`) ou règle non respectée (`body.newPassword`) |

`logout` est `member` : se déconnecter suppose une session ; sans session la
réponse `401` n'a pas d'effet de bord.

## Opérations `admin`

| Opération | Méthode et chemin | Corps | Réponses |
| --- | --- | --- | --- |
| `list-invitations` | `GET /api/admin/invitations` | — | `200 { items: Invitation[] }` (en attente et expirées, plus récentes d'abord) |
| `create-invitation` | `POST /api/admin/invitations` | `{ email, role }` (`role` défaut `member`) | `201 Invitation` (e-mail mis en file) · `409` « Un compte existe déjà avec cette adresse » ou « Une invitation est déjà en attente pour cette adresse » · `422` · `429` |
| `resend-invitation` | `POST /api/admin/invitations/{id}/resend` | — | `200 Invitation` (nouveau jeton, nouvelle échéance, ancien lien invalide) · `404` |
| `cancel-invitation` | `DELETE /api/admin/invitations/{id}` | — | `204` · `404` |

## Messages de règle de mot de passe (422)

Une entrée `errors[]` par critère non respecté, à l'emplacement du champ :

- « Le mot de passe doit contenir au moins 8 caractères. »
- « Le mot de passe doit contenir au moins une lettre majuscule. »
- « Le mot de passe doit contenir au moins un chiffre. »

## Documentation de développement (hors OpenAPI)

Montées seulement si `FRIGG_API_DOCS=true` ; sinon `404`. Accès `admin` via la
fonction d'autorisation commune (anonyme `401`, membre `403`). Voir
[research.md › R18](../research.md#r18-documentation-dapi-en-développement).

| Méthode et chemin | Réponse |
| --- | --- |
| `GET /api/openapi.json` | `200` schéma OpenAPI 3.1 courant |
| `GET /api/schemas/{name}.json` | `200` schéma JSON d'un composant · `404` |
| `GET /api/docs` | `200` page Stoplight Elements (CSP assouplie sur cette seule page) |

## E-mails émis (via la file d'attente)

| Déclencheur | Objet | Lien |
| --- | --- | --- |
| `request-password-reset` (compte existant) | « Réinitialisation de votre mot de passe » | `{FRIGG_BASE_URL}/reinitialiser-mot-de-passe#<jeton>` (1 h) |
| `resend-email-verification` | « Confirmez votre adresse électronique » | `{FRIGG_BASE_URL}/confirmer-adresse#<jeton>` (48 h) |
| `create-invitation`, `resend-invitation` | « Invitation à rejoindre {association} sur Frigg » | `{FRIGG_BASE_URL}/invitation#<jeton>` (7 jours) |
