# syntax=docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e

# Image de production de Frigg (specs/007-docker-release).
#
# Les images de base sont épinglées par empreinte pour des constructions
# reproductibles ; Dependabot propose leurs mises à jour. Pour relever une
# empreinte à la main : docker buildx imagetools inspect <image:tag>.
#
# Construction locale : go tool task docker:build

# --- Frontend : compilé une seule fois, quelle que soit l'architecture cible.
FROM --platform=$BUILDPLATFORM oven/bun:1.4.4-alpine@sha256:4cf82f5686b196504fe119db3fcd9185b191f82ece82c40680e597228ec6070a AS web
WORKDIR /src/web
COPY web/package.json web/bun.lock ./
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

# --- Binaire : compilation croisée en Go pur, sans émulation.
FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18 AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/
COPY --from=web /src/web/build/client web/build/client
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/frigg ./cmd/frigg
# Répertoire de données livré vide, attribué à l'utilisateur d'exécution : un
# volume nommé vide hérite de ces droits au premier montage. backups/ accueille
# les sauvegardes (`frigg backup backups/<fichier>`) : sans shell, l'exploitant
# ne pourrait pas le créer depuis le conteneur.
RUN mkdir -p /out/data/files /out/data/backups \
    && chmod 0750 /out/data /out/data/files /out/data/backups

# --- Image d'exécution : ni shell, ni gestionnaire de paquets, utilisateur
# non privilégié (65532). Aucune instruction RUN dans cette étape.
FROM gcr.io/distroless/static-debian13:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329

ARG VERSION=dev
ARG REVISION=inconnue
ARG CREATED=
LABEL org.opencontainers.image.title="Frigg" \
      org.opencontainers.image.description="Gestion d'association auto-hébergée" \
      org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.revision="$REVISION" \
      org.opencontainers.image.created="$CREATED" \
      org.opencontainers.image.source="https://github.com/Balsakup/frigg" \
      org.opencontainers.image.licenses="AGPL-3.0-or-later"

COPY --from=build --chown=65532:65532 /out/data /data
COPY --from=build /out/frigg /frigg
COPY LICENSE /usr/share/doc/frigg/LICENSE

# Valeurs par défaut propres au conteneur ; les autres variables (URL publique,
# SMTP, expéditeur) sont fournies par l'exploitant.
ENV FRIGG_DATABASE_PATH=/data/frigg.db \
    FRIGG_DATA_DIR=/data/files \
    FRIGG_LISTEN_ADDR=:8080

USER 65532:65532
WORKDIR /data
VOLUME ["/data"]
EXPOSE 8080

# Vérification de santé par le binaire lui-même (l'image n'a ni shell ni
# curl) : « sain » quelques secondes après le démarrage grâce au sondage
# rapproché, « en mauvaise santé » en moins de 2 minutes après une panne.
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --start-interval=2s --retries=3 \
    CMD ["/frigg", "healthcheck"]

ENTRYPOINT ["/frigg"]
CMD ["serve"]
