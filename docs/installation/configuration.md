# Configuration

Toute la configuration passe par des **variables d'environnement**. Elles
peuvent aussi être lues depuis un fichier `.env` (dans l'image Docker :
`/data/.env`). Les variables déjà présentes dans l'environnement priment sur
ce fichier.

Si une variable obligatoire manque, Frigg refuse de démarrer et la nomme. Une
variable facultative absente désactive seulement la fonction qui en dépend.

Le fichier
[`.env.example`](https://github.com/Balsakup/frigg/blob/main/.env.example) du
dépôt reprend toutes ces variables avec des valeurs de développement.

## Serveur et stockage

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `FRIGG_BASE_URL` | oui | — | URL publique de Frigg (`https://frigg.mon-asso.fr`), utilisée dans les liens des e-mails |
| `FRIGG_DATABASE_PATH` | non | `./data/frigg.db` (image : `/data/frigg.db`) | Chemin du fichier SQLite |
| `FRIGG_DATA_DIR` | non | `./data/files` (image : `/data/files`) | Répertoire des fichiers déposés, à sauvegarder avec la base |
| `FRIGG_LISTEN_ADDR` | non | `:8080` | Adresse d'écoute du serveur HTTP |
| `FRIGG_COOKIE_SECURE` | non | `true` | Cookies de session `Secure`. À laisser à `true`, sauf accès HTTP hors `localhost` |
| `FRIGG_TRUSTED_PROXIES` | non | vide | Adresses IP des reverse proxies de confiance, séparées par des virgules : `X-Forwarded-For` n'est lu que pour elles (voir [Reverse proxy](./reverse-proxy)) |

## E-mails

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `FRIGG_SMTP_HOST` | oui | — | Serveur SMTP |
| `FRIGG_SMTP_PORT` | non | `587` | Port du serveur SMTP |
| `FRIGG_SMTP_USERNAME` | non | vide | Identifiant SMTP |
| `FRIGG_SMTP_PASSWORD` | non | vide | Mot de passe SMTP |
| `FRIGG_SMTP_TLS` | non | `starttls` | `starttls`, `tls` ou `none` |
| `FRIGG_MAIL_FROM` | oui | — | Adresse d'expédition ; les réponses arrivent à cette adresse |
| `FRIGG_MAIL_POLL_INTERVAL` | non | `30s` | Intervalle de dépilage de la file d'e-mails (voir [E-mails](./email)) |

## Journaux et outils

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `FRIGG_LOG_LEVEL` | non | `info` | `debug`, `info`, `warn` ou `error`. Les journaux sont écrits en JSON sur la sortie standard |
| `FRIGG_API_DOCS` | non | `false` | Documentation d'API de développement (`/api/docs`), réservée aux administrateurs. **Ne jamais activer en production** |

## Paiement par carte (HelloAsso, facultatif)

Renseignez les trois premières variables ensemble, ou aucune : sans elles, le
paiement par carte est simplement indisponible. Voir
[Paiement par carte](./helloasso).

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `FRIGG_HELLOASSO_CLIENT_ID` | avec les deux suivantes | — | Identifiant du client API |
| `FRIGG_HELLOASSO_CLIENT_SECRET` | avec les deux autres | — | Secret du client API. Ne jamais le commiter |
| `FRIGG_HELLOASSO_ORGANIZATION` | avec les deux précédentes | — | Identifiant (slug) de l'association sur HelloAsso |
| `FRIGG_HELLOASSO_ENVIRONMENT` | non | `sandbox` | `sandbox` (test, aucun débit réel) ou `production` |
| `FRIGG_HELLOASSO_API_URL` | non | — | Réservée aux tests : remplace l'hôte de l'API HelloAsso. Ne pas renseigner en production |
