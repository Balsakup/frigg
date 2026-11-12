# Installation et exploitation

Frigg est une application web de gestion d'association, conçue pour être
auto-hébergée : **une instance par association**, livrée sous la forme d'un
binaire unique qui contient l'interface et applique lui-même les migrations
de sa base au démarrage.

Pour héberger plusieurs associations, déployez plusieurs instances, chacune
avec son propre volume de données.

## Ce dont vous avez besoin

- Un serveur capable de faire tourner des conteneurs (Docker, Podman ou
  Kubernetes), en `linux/amd64` ou `linux/arm64`.
- Un **nom de domaine** et un **reverse proxy** qui termine le TLS (Caddy,
  Traefik, nginx…) : Frigg doit être servi en `https://`.
- Un **serveur SMTP** pour envoyer les e-mails (invitations, réinitialisation
  du mot de passe, communications).
- Facultatif : un compte **HelloAsso** pour le paiement des adhésions et des
  inscriptions par carte bancaire.

## Ce que contient une instance

| Élément | Rôle |
| --- | --- |
| Image `ghcr.io/balsakup/frigg:<version>` | Le binaire, sans shell, exécuté par un utilisateur non privilégié |
| Volume `/data` | La base SQLite, les fichiers déposés et les sauvegardes |
| Variables d'environnement | Toute la configuration (voir [Configuration](./configuration)) |

## Étapes

1. [Démarrer Frigg avec Docker](./docker).
2. Renseigner la [configuration](./configuration), au minimum l'URL publique,
   le serveur SMTP et l'expéditeur des e-mails.
3. Placer l'instance derrière un [reverse proxy](./reverse-proxy).
4. Créer le [premier administrateur](./first-admin).
5. Mettre en place les [sauvegardes](./backup).
6. Facultatif : activer le [paiement par carte](./helloasso).

Les mises à jour sont décrites dans [Mettre à jour](./upgrade), et les
réglages de sécurité du conteneur dans [Durcissement](./hardening).
