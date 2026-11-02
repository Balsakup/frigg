# Contrat CLI : ajouts au binaire `frigg`

Complète [specs/001-auth-app-shell/contracts/cli.md](../../001-auth-app-shell/contracts/cli.md).

## `frigg healthcheck`

Vérifie l'état de santé de l'instance locale. Utilisée par le `HEALTHCHECK` de
l'image ([research.md › R9](../research.md#r9-vérification-de-santé-de-limage--frigg-healthcheck)).

- Lit la configuration (`.env` et environnement) sans exiger les variables de
  `serve`.
- URL appelée : `http://<hôte>:<port>/api/health`, où hôte et port viennent de
  `FRIGG_LISTEN_ADDR` ; un hôte vide, `0.0.0.0` ou `::` devient `127.0.0.1`.
- Délai : 3 s.

| Cas | Sortie (stderr) | Code |
| --- | --- | --- |
| `200` | aucune | 0 |
| Autre statut | `instance en mauvaise santé : statut HTTP 503` | 1 |
| Connexion impossible, délai dépassé | `instance injoignable : <cause réseau>` | 1 |
| `FRIGG_LISTEN_ADDR` illisible | `configuration invalide : FRIGG_LISTEN_ADDR …` | 1 |

## `frigg serve` — contrôle d'écriture au démarrage

Avant l'ouverture de la base, pour le répertoire de `FRIGG_DATABASE_PATH` et
pour `FRIGG_DATA_DIR` : création si absent (`0750`), puis écriture d'un fichier
témoin supprimé aussitôt
([research.md › R7](../research.md#r7-vérification-décriture-au-démarrage)).

| Cas | Sortie (stderr) | Code |
| --- | --- | --- |
| Répertoire non accessible en écriture | `le répertoire /data/files n'est pas accessible en écriture par l'utilisateur 65532 : permission denied (en conteneur, le volume doit appartenir à l'UID 65532)` | 1 |

## Aide

`frigg help` ajoute la ligne :

```text
  healthcheck   vérifie l'état de santé de l'instance locale (code 0 si saine)
```
