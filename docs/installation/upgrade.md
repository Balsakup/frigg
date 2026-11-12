# Mettre à jour

1. Faites une [sauvegarde](./backup).
2. Remplacez le conteneur par la nouvelle version, en gardant le même volume.

```sh
docker pull ghcr.io/balsakup/frigg:1.1.0
docker stop frigg && docker rm frigg
docker run … ghcr.io/balsakup/frigg:1.1.0   # mêmes options qu'au démarrage
```

Avec Docker Compose, modifiez la version de l'image dans `compose.yaml`, puis
lancez `docker compose up -d`.

Les migrations de la base s'appliquent automatiquement au démarrage. Si l'une
d'elles échoue, Frigg refuse de démarrer : restaurez la sauvegarde et
reprenez l'ancienne version.

Consultez les notes de la release sur
[GitHub](https://github.com/Balsakup/frigg/releases) avant chaque mise à
jour. La documentation de la version installée est celle de la
[dernière release](https://balsakup.github.io/frigg/), choisie dans le
sélecteur de version.
