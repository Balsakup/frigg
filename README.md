# Frigg

Frigg est une application web de gestion d'association, conçue pour être
auto-hébergée : une instance par association, livrée sous forme d'un binaire
unique en Go.

## Installer avec Docker

Chaque release publie l'image `ghcr.io/balsakup/frigg:<version>` pour
`linux/amd64` et `linux/arm64` (`latest` désigne la dernière version stable).
L'image ne contient que le binaire : pas de shell, utilisateur non privilégié
`65532`, données dans le volume `/data`.

### Démarrer

```sh
docker volume create frigg-data
docker run -d --name frigg --restart unless-stopped --stop-timeout 20 \
  -p 127.0.0.1:8080:8080 -v frigg-data:/data \
  -e FRIGG_BASE_URL=https://frigg.mon-asso.fr \
  -e FRIGG_SMTP_HOST=smtp.exemple.fr -e FRIGG_SMTP_USERNAME=frigg \
  -e FRIGG_SMTP_PASSWORD=… -e FRIGG_MAIL_FROM=frigg@mon-asso.fr \
  ghcr.io/balsakup/frigg:1.0.0
```

Ou avec Docker Compose (`compose.yaml` de votre serveur, distinct de celui du
dépôt qui ne sert qu'au développement) :

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

**Configuration** : `FRIGG_BASE_URL`, `FRIGG_SMTP_HOST` et `FRIGG_MAIL_FROM`
sont obligatoires ; sans elles, le conteneur s'arrête en les nommant. Toutes
les variables sont décrites dans [`.env.example`](.env.example). L'image fixe
`FRIGG_DATABASE_PATH=/data/frigg.db`, `FRIGG_DATA_DIR=/data/files` et
`FRIGG_LISTEN_ADDR=:8080`. Un fichier `/data/.env` est lu s'il existe ; les
variables passées au conteneur priment.

**Volume** : `/data` contient la base (`frigg.db`, `frigg.db-wal`,
`frigg.db-shm`), les fichiers déposés (`files/`) et les sauvegardes
(`backups/`). Un volume nommé reçoit automatiquement les bons droits. Pour un
répertoire de l'hôte, attribuez-le d'abord à l'utilisateur de l'image :
`sudo chown -R 65532:65532 /srv/frigg`. Sans volume, les données disparaissent
avec le conteneur.

**Arrêt** : Frigg laisse jusqu'à 10 s aux requêtes en cours, soit autant que
le délai par défaut de `docker stop`. Gardez un délai d'arrêt de 20 s
(`--stop-timeout 20`, `stop_grace_period: 20s`) pour que la base soit toujours
fermée proprement.

### Premier administrateur

```sh
docker exec -it frigg /frigg create-admin
```

### Sauvegarder et restaurer

```sh
docker exec frigg /frigg backup backups/frigg-$(date +%F).db
docker cp frigg:/data/backups/frigg-$(date +%F).db .
```

La sauvegarde, lisible par son seul propriétaire, contient toute la base ;
sauvegardez aussi le répertoire `files/` du volume. Pour restaurer, arrêtez le
conteneur et remplacez la base depuis un conteneur temporaire :

```sh
docker stop frigg
docker run --rm -v frigg-data:/data -v "$PWD":/restore busybox:1.37 sh -c \
  'cp /restore/frigg-2026-11-02.db /data/frigg.db \
   && rm -f /data/frigg.db-wal /data/frigg.db-shm \
   && chown 65532:65532 /data/frigg.db && chmod 600 /data/frigg.db'
docker start frigg
```

### Mettre à jour

Faites une sauvegarde, puis remplacez le conteneur par la nouvelle version en
gardant le même volume ; les migrations s'appliquent au démarrage.

```sh
docker pull ghcr.io/balsakup/frigg:1.1.0
docker stop frigg && docker rm frigg
docker run … ghcr.io/balsakup/frigg:1.1.0   # mêmes options qu'au démarrage
```

### Reverse proxy et santé

Placez Frigg derrière un reverse proxy qui termine le TLS, et déclarez son
adresse dans `FRIGG_TRUSTED_PROXIES` pour que l'adresse des clients soit
reconnue. Le point de santé `GET /api/health` répond `200` quand l'instance
peut servir des requêtes et `503` sinon, sans aucune autre information.
L'image l'utilise pour sa vérification de santé (`docker ps` affiche
`healthy`). En Kubernetes, qui ignore cette vérification, déclarez une sonde :

```yaml
readinessProbe:
  httpGet: { path: /api/health, port: 8080 }
  periodSeconds: 30
livenessProbe:
  httpGet: { path: /api/health, port: 8080 }
  periodSeconds: 30
  failureThreshold: 3
```

SQLite impose une seule réplique : volume `ReadWriteOnce` et stratégie de
déploiement `Recreate`.

### Durcissement

L'image fonctionne avec un système de fichiers racine en lecture seule et
sans aucun privilège ; seul `/data` doit être accessible en écriture.

| Docker | Compose | Kubernetes (`securityContext`) |
| --- | --- | --- |
| `--read-only` | `read_only: true` | `readOnlyRootFilesystem: true` |
| `--cap-drop=ALL` | `cap_drop: [ALL]` | `capabilities: { drop: [ALL] }` |
| `--security-opt=no-new-privileges` | `security_opt: [no-new-privileges:true]` | `allowPrivilegeEscalation: false` |
| (image : utilisateur 65532) | | `runAsNonRoot: true`, `runAsUser: 65532`, `fsGroup: 65532` |

Chaque image publiée porte une attestation de provenance signée et un SBOM.
Pour vérifier qu'une image vient bien de ce dépôt :

```sh
gh attestation verify oci://ghcr.io/balsakup/frigg:1.0.0 --owner Balsakup
```

### Paiement par carte (HelloAsso)

Les adhérents peuvent payer leur adhésion par carte bancaire via
[HelloAsso](https://www.helloasso.com), la plateforme de paiement gratuite des
associations françaises. Cette fonction est facultative : sans les variables
ci-dessous, Frigg fonctionne normalement et ne propose pas la carte.

1. Créez un client API dans l'espace HelloAsso de l'association
   (*Mon compte › Intégrations et API*), avec le droit « Checkout ».
   Commencez par l'environnement de test, sur
   [helloasso-sandbox.com](https://www.helloasso-sandbox.com) : aucune somme
   réelle n'y est débitée.
2. Renseignez les trois premières variables ensemble (jamais dans un fichier
   commité) :

   | Variable | Rôle |
   | --- | --- |
   | `FRIGG_HELLOASSO_CLIENT_ID` | identifiant du client API |
   | `FRIGG_HELLOASSO_CLIENT_SECRET` | secret du client API |
   | `FRIGG_HELLOASSO_ORGANIZATION` | identifiant (slug) de l'association sur HelloAsso |
   | `FRIGG_HELLOASSO_ENVIRONMENT` | `sandbox` (défaut) ou `production` |

   `FRIGG_BASE_URL` doit alors être en `https://`, même en mode test :
   HelloAsso n'accepte que des adresses de retour sécurisées. En
   développement, exposez l'instance par un tunnel HTTPS (expose.dev,
   cloudflared, ngrok…) et indiquez son adresse dans `FRIGG_BASE_URL`. L'instance doit être joignable depuis
   Internet pour recevoir les notifications de paiement.
3. Redémarrez Frigg. Dans *Administration › Paramètres › Paiements*, copiez
   l'adresse de notification affichée et déclarez-la dans l'espace HelloAsso,
   puis choisissez « Tester la connexion » et activez la carte.
4. Faites un paiement d'essai avec une carte de test, puis passez en
   production : identifiants de production, `FRIGG_HELLOASSO_ENVIRONMENT=production`,
   redémarrage, nouveau test de connexion et nouvelle déclaration de l'adresse
   de notification dans l'espace HelloAsso de production.

Changer de secret change l'adresse de notification : déclarez la nouvelle
adresse dans HelloAsso. Les remboursements se font dans l'espace HelloAsso ;
Frigg signale aux administrateurs ceux qui sont à faire.

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
