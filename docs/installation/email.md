# E-mails

Frigg envoie des e-mails pour les invitations, la vérification de l'adresse,
la réinitialisation du mot de passe, les paiements, les événements et les
[communications](../usage/bulk-email) des administrateurs.

## Serveur SMTP

Renseignez au minimum `FRIGG_SMTP_HOST` et `FRIGG_MAIL_FROM` (voir
[Configuration](./configuration#e-mails)). Le nom de l'association apparaît
comme expéditeur, et les réponses arrivent à l'adresse `FRIGG_MAIL_FROM`.

## File d'envoi

Aucun e-mail n'est envoyé pendant l'action qui le déclenche : il est d'abord
placé dans une file d'attente, puis envoyé en arrière-plan.

- La file est dépilée toutes les `FRIGG_MAIL_POLL_INTERVAL` (30 s par
  défaut), à raison de 20 e-mails par passage.
- Chaque e-mail a un état : **en attente** (`pending`), **envoyé** (`sent`)
  ou **en échec** (`failed`).
- En cas d'erreur du serveur SMTP, l'envoi est retenté après 1 min, puis
  2 min, 4 min… (au plus 1 h d'écart). Après 10 tentatives, l'e-mail passe en
  échec et l'erreur est journalisée.
- L'historique d'une communication indique, pour chaque destinataire, si
  l'e-mail est parti, en attente ou en échec.

Le contenu des e-mails n'est jamais écrit dans les journaux.

## Tester en développement

Le `compose.yaml` du dépôt démarre [Mailpit](https://mailpit.axllent.org),
qui capture les e-mails sans les envoyer (`FRIGG_SMTP_HOST=localhost`,
`FRIGG_SMTP_PORT=1025`, `FRIGG_SMTP_TLS=none`) ; l'interface est sur
`http://localhost:8025`.
