# Quickstart: valider l'image et la release

**Feature**: [spec.md](./spec.md) | **Contrats** : [image](./contracts/image.md),
[release](./contracts/release.md), [API](./contracts/api.md), [CLI](./contracts/cli.md)

Prérequis : Docker 25 ou plus récent (pour `--start-interval`), Go et Bun pour
les tests, `gh` pour les vérifications de release.

## 1. Tests automatisés

```sh
go tool task gen      # OpenAPI et types régénérés : get-health ajouté
git diff --exit-code internal/database/sqlc web/app/api
go test ./internal/api/... ./internal/cli/... ./internal/server/...
```

Attendu : matrice d'accès verte avec `get-health` publique ; tests de santé
(`200`, `503`, journaux) et de `healthcheck` verts ; contrôle d'écriture au
démarrage couvert.

## 2. Construction locale (US4)

```sh
go tool task --list | grep docker:build    # description en français
go tool task docker:build
docker image inspect frigg:dev \
  --format '{{index .Config.Labels "org.opencontainers.image.version"}} {{.Config.User}}'
```

Attendu : `dev-<sha>` (suffixé `-modifie` si l'arbre est modifié) et
`65532:65532`. Aucune image `ghcr.io/…` n'a été poussée.

Sans Docker dans le `PATH` : `Docker est requis pour construire l'image
locale.`

## 3. Sécurité et taille (US3)

```sh
docker run --rm --entrypoint sh frigg:dev        # échoue : pas de shell
docker save frigg:dev | gzip | wc -c             # taille compressée de l'image
docker create --name frigg-bin frigg:dev && docker cp frigg-bin:/frigg - | gzip | wc -c; docker rm frigg-bin
# SC-003 : image ≤ binaire compressé + 5 Mo, et image < 50 Mo
docker history --no-trunc frigg:dev              # ni .env, ni source, ni jeton
```

Analyse de vulnérabilités locale (facultatif) :

```sh
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy image --severity CRITICAL,HIGH --ignore-unfixed frigg:dev
```

## 4. Exploitation durcie (US2, US3)

```sh
docker network create frigg-net
docker run -d --name frigg-mail --network frigg-net \
  -p 127.0.0.1:8025:8025 axllent/mailpit:v1.32.0
docker volume create frigg-data
docker run -d --name frigg --network frigg-net --stop-timeout 20 \
  --read-only --cap-drop=ALL --security-opt=no-new-privileges \
  -p 127.0.0.1:8080:8080 -v frigg-data:/data \
  -e FRIGG_BASE_URL=http://localhost:8080 -e FRIGG_COOKIE_SECURE=false \
  -e FRIGG_SMTP_HOST=frigg-mail -e FRIGG_SMTP_PORT=1025 -e FRIGG_SMTP_TLS=none \
  -e FRIGG_MAIL_FROM=frigg@localhost \
  frigg:dev
```

| Vérification | Commande | Attendu |
| --- | --- | --- |
| Santé | `docker inspect -f '{{.State.Health.Status}}' frigg` | `healthy` en moins de 30 s |
| Point de santé | `curl -si localhost:8080/api/health` | `200`, `{"status":"ok"}`, `Cache-Control: no-store` |
| Journaux | `docker logs frigg` | aucune ligne `info` pour `/api/health` |
| Admin | `docker exec -it frigg /frigg create-admin` | saisie masquée, compte créé |
| Connexion | navigateur sur `http://localhost:8080` | connexion réussie |
| E-mail | réinitialisation du mot de passe, puis Mailpit `:8025` | e-mail reçu |
| Sauvegarde | `docker exec frigg /frigg backup sauvegarde.db` puis `docker cp frigg:/data/sauvegarde.db .` | fichier `0600` dans le volume, récupéré |
| Données | `docker exec frigg /frigg healthcheck` puis `docker logs frigg` | démarrage sans erreur d'écriture : `/data/files` créé et accessible |
| Arrêt | `time docker stop frigg` | journal `arrêt du serveur`, sortie avec le code 0 (pas de `SIGKILL`, code 137), moins de 20 s |

## 5. Cas d'erreur (US2)

```sh
# Variable obligatoire manquante
docker run --rm frigg:dev
# → configuration invalide : FRIGG_BASE_URL est obligatoire ; …

# Répertoire de l'hôte appartenant à root
mkdir -p /tmp/frigg-root && sudo chown root:root /tmp/frigg-root
docker run --rm -v /tmp/frigg-root:/data -e FRIGG_BASE_URL=http://x \
  -e FRIGG_SMTP_HOST=x -e FRIGG_MAIL_FROM=x@x frigg:dev
# → le répertoire /data n'est pas accessible en écriture par l'utilisateur 65532 : …
```

Base inaccessible : `docker exec frigg /frigg healthcheck; echo $?` renvoie
`0` sur le conteneur de l'étape 4. Le passage à `503` quand la base ne répond
plus est couvert par le test d'API (base fermée) ; avec les paramètres de
`HEALTHCHECK`, trois échecs espacés de 30 s font passer le conteneur à
`unhealthy` en moins de 2 minutes (SC-009).

## 6. Mise à jour sans perte (SC-006)

1. Sur le conteneur de l'étape 4, créer quelques données (saison, membre).
2. `docker stop frigg && docker rm frigg`.
3. Relancer la commande de l'étape 4 avec une image plus récente
   (`frigg:dev` reconstruite après un changement, ou une version publiée).
4. Vérifier que les données sont intactes et que les migrations ont été
   appliquées (journaux).

## 7. Release (US1)

Sur une préversion de test, par exemple `v0.1.0-rc.1` :

```sh
gh release create v0.1.0-rc.1 --prerelease --title "v0.1.0-rc.1" --notes "Test"
gh run watch                                   # < 20 min (SC-001)
docker pull ghcr.io/balsakup/frigg:0.1.0-rc.1  # sans authentification (après passage en public)
docker buildx imagetools inspect ghcr.io/balsakup/frigg:0.1.0-rc.1   # amd64 + arm64
gh attestation verify oci://ghcr.io/balsakup/frigg:0.1.0-rc.1 --owner Balsakup
gh release view v0.1.0-rc.1                    # notes : image et empreinte
```

Attendu : `latest` absent ou inchangé. Puis :

- étiquette invalide (`gh release create test-1`) → job `verify` en échec,
  rien de publié ;
- release stable `v0.1.0` → `0.1.0` et `latest` désignent la même empreinte.
