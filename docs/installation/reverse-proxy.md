# Reverse proxy et santé

## Reverse proxy

Placez Frigg derrière un reverse proxy qui termine le TLS, puis déclarez
l'adresse du proxy dans `FRIGG_TRUSTED_PROXIES` pour que l'adresse des
clients soit reconnue.

Sans cette déclaration, tous les clients partagent l'adresse du proxy, et donc
ses limites de tentatives (connexion, réinitialisation du mot de passe…).
Frigg le signale une fois dans ses journaux (« X-Forwarded-For reçu d'un proxy
absent de FRIGG_TRUSTED_PROXIES »), avec l'adresse à déclarer.

## Point de santé

`GET /api/health` répond `200` quand l'instance peut servir des requêtes, et
`503` sinon, sans aucune autre information.

L'image l'utilise pour sa vérification de santé : `docker ps` affiche
`healthy`. Kubernetes ignore cette vérification ; déclarez donc des sondes :

```yaml
readinessProbe:
  httpGet: { path: /api/health, port: 8080 }
  periodSeconds: 30
livenessProbe:
  httpGet: { path: /api/health, port: 8080 }
  periodSeconds: 30
  failureThreshold: 3
```

::: warning Une seule réplique
SQLite impose une seule réplique : volume `ReadWriteOnce` et stratégie de
déploiement `Recreate`.
:::
