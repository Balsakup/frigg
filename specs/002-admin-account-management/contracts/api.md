# Contrat d'API : Gestion des comptes par les administrateurs

Mêmes conventions que [001 › api.md](../../001-auth-app-shell/contracts/api.md)
(préfixe `/api`, camelCase, `problem+json` en français, CSRF, `401` sans
session, `403` rôle insuffisant). Le schéma OpenAPI produit par `frigg openapi`
fait foi.

## Type

```text
AdminAccount { id: int, email: string, role: "member"|"admin",
               status: "active"|"disabled", emailVerified: bool,
               createdAt: datetime, lastLoginAt: datetime|null,
               disabledAt: datetime|null, isSelf: bool }

AdminAccountPage { items: AdminAccount[], total: int, page: int, pageSize: int }
```

## Opérations `admin` (tag `Comptes`)

Aucune opération publique ni `member` n'est ajoutée.

| Opération | Méthode et chemin | Entrée | Réponses |
| --- | --- | --- | --- |
| `list-accounts` | `GET /api/admin/accounts` | requête : `q?` (≤ 100 ; sous-chaînes d'au moins 3 caractères séparées par des espaces, toutes requises), `role?` (`member`\|`admin`), `status?` (`active`\|`disabled`), `page?` (≥ 1, défaut 1), `pageSize?` (1–100, défaut 25) | `200 AdminAccountPage` (tri par adresse) · `422` (dont `query.q` « Saisissez au moins 3 caractères. ») |
| `get-account` | `GET /api/admin/accounts/{id}` | — | `200 AdminAccount` · `404` |
| `disable-account` | `POST /api/admin/accounts/{id}/disable` | — | `200 AdminAccount` (déjà désactivé : inchangé) · `404` · `409` soi-même · `409` dernier administrateur actif |
| `enable-account` | `POST /api/admin/accounts/{id}/enable` | — | `200 AdminAccount` (déjà actif : inchangé) · `404` |
| `set-account-role` | `PUT /api/admin/accounts/{id}/role` | `{ role: "member"\|"admin" }` | `200 AdminAccount` (même rôle : inchangé) · `404` · `409` soi-même · `409` dernier administrateur actif · `422` |
| `delete-account` | `DELETE /api/admin/accounts/{id}` | `members`, `memberIds` (005) | `204` · `404` · `409` soi-même · `409` dernier administrateur actif · `422`/`409` fiches gérées, voir [005 › api.md](../../005-member-management/contracts/api.md#opération-modifiée--delete-account-002) |
| `send-account-password-reset` | `POST /api/admin/accounts/{id}/password-reset` | — | `202` (e-mail mis en file) · `404` · `409` soi-même · `409` compte désactivé · `429` |

`enable-account` n'a pas de garde-fou « soi-même » : un compte désactivé ne
peut pas avoir de session.

## Messages

| Cas | Statut | `detail` |
| --- | --- | --- |
| Compte inexistant | `404` | « Ce compte n'existe plus. » |
| Action sur soi-même | `409` | « Vous ne pouvez pas modifier votre propre compte depuis cette page. » |
| Dernier administrateur actif | `409` | « L'association doit conserver au moins un administrateur actif. » |
| Lien de réinitialisation pour un compte désactivé | `409` | « Ce compte est désactivé. Réactivez-le avant d'envoyer un lien. » |
| Acteur rétrogradé ou désactivé pendant la requête | `403` / `401` | messages existants de 001 |

## Effets de bord

| Opération | Effet |
| --- | --- |
| `disable-account` | `disabled_at`, époque +1 (sessions fermées), jetons non utilisés supprimés |
| `delete-account` | compte, jetons, invitations adressées à l'adresse et e-mails en file pour l'adresse supprimés ; `invited_by` des invitations envoyées → `null` |
| `send-account-password-reset` | jeton `password_reset` (1 h, remplace les précédents) + e-mail « Réinitialisation de votre mot de passe » (modèle de 001) |
| toute action qui modifie | ligne de journal `INFO` (`action`, `actor_id`, `target_id`) — voir [research.md › R10](../research.md#r10-journalisation-des-actions-fr-016) |

## Changements d'opérations existantes

| Opération | Changement |
| --- | --- |
| middleware d'accès | compte désactivé → `401` « Votre session a expiré. » et session détruite |
| `login` | compte désactivé → `401` « Adresse électronique ou mot de passe incorrect » (après vérification du mot de passe) |
| `request-password-reset` | compte désactivé → `202` sans jeton ni e-mail |
| `confirm-password-reset`, `confirm-email` | jeton d'un compte désactivé → `410` |
| `list-invitations` | inchangée ; `invitedBy: null` est affiché « Compte supprimé » par le frontend |
