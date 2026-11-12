# Sauvegarder et restaurer

Une sauvegarde complète d'une instance, c'est :

1. **la base**, sauvegardée à chaud par `frigg backup` ;
2. **le répertoire `files/`** du volume, qui contient notamment les pièces
   jointes des communications (`files/communications/`) et le logo de
   l'association (`files/logo/`).

## Sauvegarder

```sh
docker exec frigg /frigg backup backups/frigg-$(date +%F).db
docker cp frigg:/data/backups/frigg-$(date +%F).db .
```

La sauvegarde est cohérente même si l'instance est en cours d'utilisation.
Elle contient toute la base et n'est lisible que par son propriétaire :
conservez-la en lieu sûr, car elle contient les données personnelles des
membres.

## Restaurer

Arrêtez le conteneur, puis remplacez la base depuis un conteneur temporaire :

```sh
docker stop frigg
docker run --rm -v frigg-data:/data -v "$PWD":/restore busybox:1.37 sh -c \
  'cp /restore/frigg-2026-11-02.db /data/frigg.db \
   && rm -f /data/frigg.db-wal /data/frigg.db-shm \
   && chown 65532:65532 /data/frigg.db && chmod 600 /data/frigg.db'
docker start frigg
```

Restaurez aussi le répertoire `files/` correspondant à la même date.
