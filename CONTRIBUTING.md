# Contribuer à Frigg

Merci de votre intérêt pour Frigg ! Ce guide explique comment proposer une
contribution.

## Licence et CLA

Frigg est distribué sous la licence
[GNU Affero General Public License v3.0 ou ultérieure](LICENSE)
(AGPL-3.0-or-later).

Avant de fusionner votre première Pull Request, nous vous demandons d'accepter
l'[accord de licence de contribution](CLA.md) (CLA). En résumé :

- vous restez titulaire des droits d'auteur sur votre code ;
- vous accordez au mainteneur une licence large qui lui permet notamment
  d'utiliser vos contributions dans l'offre d'hébergement de Frigg, y compris
  sous d'autres conditions que l'AGPL ;
- vous garantissez avoir le droit de contribuer ce code. **Si vous contribuez
  dans le cadre de votre travail, demandez l'accord de votre employeur.**

Le résumé ci-dessus n'a pas de valeur juridique : seul le texte du
[CLA](CLA.md) fait foi.

À l'ouverture de votre première Pull Request, le robot CLA Assistant publie un
commentaire. Pour accepter le CLA, répondez par le commentaire exact :

```
J'ai lu le CLA de Frigg et je l'accepte.
```

L'acceptation est enregistrée une fois pour toutes. Pour relancer la
vérification, commentez `recheck`.

Si vous reprenez du code d'un autre projet, indiquez-le dans la description de
la Pull Request en précisant son auteur et sa licence.

## Mettre en place l'environnement

Prérequis : Go (version indiquée dans `go.mod`), [Bun](https://bun.sh) et
Docker (pour les tests de bout en bout).

```sh
go tool task dev     # API sur :8080, frontend sur :5173
go tool task test    # tests Go et tests unitaires du frontend
go tool task check   # contrôles exigés avant fusion
```

Les principes d'architecture du projet sont décrits dans
[la constitution](.specify/memory/constitution.md). Toute contribution doit les
respecter.

## Proposer une modification

1. Créez une branche à partir de `main`.
2. Ajoutez des tests pour toute modification de comportement.
3. Vérifiez que `go tool task check` passe.
4. Rédigez vos messages de commit en français : un
   [gitmoji](https://gitmoji.dev) suivi d'un titre court à l'indicatif, par
   exemple `🔒 Limite les essais du mot de passe actuel`.
5. Ouvrez une Pull Request en décrivant le problème résolu et la solution.

## Construire l'image en local

Pour déboguer un comportement propre au conteneur, construisez l'image avec la
même recette que la release :

```sh
go tool task docker:build   # produit l'image frigg:dev, jamais publiée
```

L'étiquette `org.opencontainers.image.version` vaut `dev-<commit>`, suffixée de
`-modifie` si l'arbre de travail contient des modifications non commitées.
Pour la lancer avec Mailpit (`docker compose up -d`) :

```sh
docker run --rm --network host \
  -e FRIGG_BASE_URL=http://localhost:8080 \
  -e FRIGG_SMTP_HOST=localhost -e FRIGG_SMTP_PORT=1025 -e FRIGG_SMTP_TLS=none \
  -e FRIGG_MAIL_FROM=frigg@localhost \
  -v frigg-dev-data:/data frigg:dev
```

L'image n'a pas de shell : pour inspecter un conteneur, utilisez `docker logs`,
`docker inspect` et `docker cp <conteneur>:/data/… .`. Les options
d'exploitation sont décrites dans la section
[Installer avec Docker](README.md#installer-avec-docker) du README.

## Publier une release

La publication d'une release GitHub construit et publie automatiquement
l'image `ghcr.io/balsakup/frigg` (workflow `.github/workflows/release.yml`).

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

Avant de publier, le workflow vérifie l'étiquette, compile (`CGO_ENABLED=0`),
lance `go vet`, les tests Go, le typecheck, Biome et Vitest, puis analyse
l'image (vulnérabilités critiques ou élevées corrigeables bloquantes). L'image
est publiée pour `linux/amd64` et `linux/arm64`, avec une attestation de
provenance et un SBOM. Les tests de bout en bout ne sont pas rejoués : ils
doivent passer avant la fusion.

**Première publication** : GitHub crée le paquet `frigg` en privé. Rendez-le
public une fois pour toutes (page du paquet → Package settings → Change
visibility → Public), sans quoi les exploitants ne peuvent pas télécharger
l'image.

**Relancer** : « Re-run jobs » sur l'exécution de la release reconstruit le
même code et remplace l'image sous la même étiquette ; les autres versions ne
sont pas touchées.

**Retirer une image** : supprimer une release ne supprime pas l'image. Pour la
retirer, supprimez la version correspondante dans la page du paquet.
