# Démarrer avec Docker

Chaque release publie l'image `ghcr.io/balsakup/frigg:<version>` pour
`linux/amd64` et `linux/arm64`. `latest` désigne la dernière version stable ;
préférez une version précise pour maîtriser les mises à jour.

L'image ne contient que le binaire : pas de shell, utilisateur non privilégié
`65532`, données dans le volume `/data`.

## Avec `docker run`

```sh
docker volume create frigg-data
docker run -d --name frigg --restart unless-stopped --stop-timeout 20 \
  -p 127.0.0.1:8080:8080 -v frigg-data:/data \
  -e FRIGG_BASE_URL=https://frigg.mon-asso.fr \
  -e FRIGG_SMTP_HOST=smtp.exemple.fr -e FRIGG_SMTP_USERNAME=frigg \
  -e FRIGG_SMTP_PASSWORD=… -e FRIGG_MAIL_FROM=frigg@mon-asso.fr \
  ghcr.io/balsakup/frigg:1.0.0
```

## Avec Docker Compose

Le fichier `compose.yaml` ci-dessous est celui de votre serveur. Il est
distinct de celui du dépôt, qui ne sert qu'au développement.

```yaml
services:
  frigg:
    image: ghcr.io/balsakup/frigg:1.0.0
    restart: unless-stopped
    stop_grace_period: 20s
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - frigg-data:/data
    env_file: frigg.env # FRIGG_BASE_URL, FRIGG_SMTP_*, FRIGG_MAIL_FROM…
    read_only: true
    cap_drop: [ALL]
    security_opt: [no-new-privileges:true]

volumes:
  frigg-data:
```

## Configuration minimale

`FRIGG_BASE_URL`, `FRIGG_SMTP_HOST` et `FRIGG_MAIL_FROM` sont obligatoires ;
sans elles, le conteneur s'arrête en les nommant. Toutes les variables sont
décrites dans [Configuration](./configuration).

L'image fixe `FRIGG_DATABASE_PATH=/data/frigg.db`,
`FRIGG_DATA_DIR=/data/files` et `FRIGG_LISTEN_ADDR=:8080`. Un fichier
`/data/.env` est lu s'il existe ; les variables passées au conteneur priment.

## Volume `/data`

`/data` contient :

- la base (`frigg.db`, `frigg.db-wal`, `frigg.db-shm`) ;
- les fichiers déposés (`files/`) ;
- les sauvegardes (`backups/`).

Un volume nommé reçoit automatiquement les bons droits. Pour un répertoire de
l'hôte, attribuez-le d'abord à l'utilisateur de l'image :

```sh
sudo chown -R 65532:65532 /srv/frigg
```

::: warning
Sans volume, les données disparaissent avec le conteneur.
:::

## Arrêt

Frigg laisse jusqu'à 10 s aux requêtes en cours, soit autant que le délai par
défaut de `docker stop`. Gardez un délai d'arrêt de 20 s (`--stop-timeout 20`,
`stop_grace_period: 20s`) pour que la base soit toujours fermée proprement.

## Étape suivante

Créez le [premier administrateur](./first-admin).
