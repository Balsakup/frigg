# Frigg

Frigg est une application web de gestion d'association, conçue pour être
auto-hébergée : une instance par association, livrée sous forme d'un binaire
unique en Go.

📖 **Documentation : <https://balsakup.github.io/frigg/>**

## Démarrage rapide

Chaque release publie l'image `ghcr.io/balsakup/frigg:<version>` pour
`linux/amd64` et `linux/arm64`.

```sh
docker volume create frigg-data
docker run -d --name frigg --restart unless-stopped --stop-timeout 20 \
  -p 127.0.0.1:8080:8080 -v frigg-data:/data \
  -e FRIGG_BASE_URL=https://frigg.mon-asso.fr \
  -e FRIGG_SMTP_HOST=smtp.exemple.fr -e FRIGG_MAIL_FROM=frigg@mon-asso.fr \
  ghcr.io/balsakup/frigg:1.0.0
docker exec -it frigg /frigg create-admin
```

Configuration, reverse proxy, sauvegardes, mises à jour et paiement par
carte : voir le guide
[Installation et exploitation](https://balsakup.github.io/frigg/installation/).

## Documentation

- [Installation et exploitation](https://balsakup.github.io/frigg/installation/) :
  Docker, configuration, sauvegarde, mise à jour, HelloAsso.
- [Utilisation](https://balsakup.github.io/frigg/usage/) : saisons, membres,
  adhésions, paiements, événements, communication.
- [Contribution](https://balsakup.github.io/frigg/contributing/) :
  environnement de développement, cycle de développement, releases.

La documentation de la dernière release est publiée à la racine du site ; la
version en développement (branche `main`) est sous
[`/frigg/main/`](https://balsakup.github.io/frigg/main/).

## Contribuer

Les contributions sont les bienvenues. Consultez le
[guide de contribution](CONTRIBUTING.md) avant d'ouvrir une Pull Request.

## Licence

Frigg est un logiciel libre distribué sous la licence
[GNU Affero General Public License v3.0 ou ultérieure](LICENSE)
(AGPL-3.0-or-later).

Vous pouvez l'utiliser, le modifier et le redistribuer, y compris à des fins
commerciales. Si vous mettez une version modifiée de Frigg à disposition
d'utilisateurs par le réseau, vous devez leur donner accès à son code source
sous la même licence.

Les contributions externes sont soumises à un
[accord de licence de contribution](CLA.md).
