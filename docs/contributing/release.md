# Releases et image locale

## Construire l'image en local

Pour déboguer un comportement propre au conteneur, construisez l'image avec
la même recette que la release :

```sh
go tool task docker:build   # produit l'image frigg:dev, jamais publiée
```

L'étiquette `org.opencontainers.image.version` vaut `dev-<commit>`, suffixée
de `-modifie` si l'arbre de travail contient des modifications non commitées.
Pour la lancer avec Mailpit (`docker compose up -d`) :

```sh
docker run --rm --network host \
  -e FRIGG_BASE_URL=http://localhost:8080 \
  -e FRIGG_SMTP_HOST=localhost -e FRIGG_SMTP_PORT=1025 -e FRIGG_SMTP_TLS=none \
  -e FRIGG_MAIL_FROM=frigg@localhost \
  -v frigg-dev-data:/data frigg:dev
```

L'image n'a pas de shell : pour inspecter un conteneur, utilisez
`docker logs`, `docker inspect` et `docker cp <conteneur>:/data/… .`. Les
options d'exploitation sont décrites dans
[Démarrer avec Docker](../installation/docker).

## Publier une release

La publication d'une release GitHub construit et publie automatiquement
l'image `ghcr.io/balsakup/frigg` (workflow `.github/workflows/release.yml`),
puis met à jour cette documentation.

1. Vérifiez que `main` passe `go tool task check`.
2. Créez la release depuis GitHub (ou `gh release create`) avec une étiquette
   `vX.Y.Z` ; pour une préversion, `vX.Y.Z-rc.N` et la case « Set as a
   pre-release ».
3. Suivez l'exécution dans l'onglet Actions. En cas de succès, une ligne
   « Image » avec l'empreinte est ajoutée aux notes de la release ; en cas
   d'échec, GitHub vous notifie et rien n'est publié.

| Étiquette de la release | Étiquettes de l'image |
| --- | --- |
| `v1.2.0`, plus haute version stable | `1.2.0` et `latest` |
| `v1.1.4` publiée après `v1.2.0` | `1.1.4` seulement |
| `v1.3.0-rc.1`, ou toute release marquée préversion | `1.3.0-rc.1` seulement |
| `1.2.0`, `v1.2`, `release-1`… | aucune : le workflow échoue |

Avant de publier, le workflow vérifie l'étiquette, compile
(`CGO_ENABLED=0`), lance `go vet`, les tests Go, le typecheck, Biome et
Vitest, puis analyse l'image (les vulnérabilités critiques ou élevées
corrigeables sont bloquantes). L'image est publiée pour `linux/amd64` et
`linux/arm64`, avec une attestation de provenance et un SBOM. Les tests de
bout en bout ne sont pas rejoués : ils doivent passer avant la fusion.

**Première publication** : GitHub crée le paquet `frigg` en privé.
Rendez-le public une fois pour toutes (page du paquet → Package settings →
Change visibility → Public), sans quoi les exploitants ne peuvent pas
télécharger l'image.

**Relancer** : « Re-run jobs » sur l'exécution de la release reconstruit le
même code et remplace l'image sous la même étiquette ; les autres versions ne
sont pas touchées.

**Retirer une image** : supprimer une release ne supprime pas l'image. Pour
la retirer, supprimez la version correspondante dans la page du paquet.

## Documentation de la release

Chaque release publiée (préversions comprises) devient la version par défaut
de la documentation, si elle est la plus haute. Voir
[Rédiger la documentation](./documentation#versions-publiees).
