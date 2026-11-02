# Data Model: Image de production et publication à chaque release

**Feature**: [spec.md](./spec.md) | **Date**: 2026-11-02

Aucune migration : le schéma SQLite ne change pas. Cette fonctionnalité
manipule des artefacts de livraison et une réponse d'API.

## Requête sqlc ajoutée

`internal/database/queries/health.sql` :

| Nom | Type | Rôle |
| --- | --- | --- |
| `HealthCheck` | `:one` | `SELECT 1`, exécutée sur le pool de lecture par le point de santé |

## Réponse de santé (`HealthOutput`)

| Champ | Type | Règle |
| --- | --- | --- |
| `status` | chaîne, énumération `ok` | seule valeur possible en `200` |
| `Cache-Control` (en-tête) | chaîne | toujours `no-store` |

En `503`, corps de problème huma standard (`title`, `status`, `detail`), sans
autre champ.

## Release

| Attribut | Source | Règle |
| --- | --- | --- |
| `tag` | étiquette git de la release | `^v(0\|[1-9]\d*)\.(0\|[1-9]\d*)\.(0\|[1-9]\d*)(-[0-9A-Za-z.-]+)?$` |
| `version` | `tag` sans `v` | étiquette de l'image |
| `prerelease` | suffixe présent ou case « préversion » | vrai → jamais `latest` |
| `latest` | calculé | `!prerelease && tag == max(sort -V des étiquettes vX.Y.Z)` |
| `revision` | SHA du commit étiqueté | étiquette OCI `revision` |

## Image publiée

| Attribut | Valeur |
| --- | --- |
| Nom | `ghcr.io/balsakup/frigg` |
| Étiquettes | `version`, plus `latest` si `latest` |
| Plateformes | `linux/amd64`, `linux/arm64` |
| Étiquettes OCI | voir [contracts/image.md](./contracts/image.md#métadonnées-étiquettes-oci) |
| Attestations | provenance (`mode=max`), SBOM, provenance signée GitHub |

## Image locale

Même recette ; nom `frigg:dev` ; `version` = `dev-<sha court>` suffixé de
`-modifie` si l'arbre de travail contient des modifications ; aucune
attestation, jamais publiée.

## Répertoire de données de l'instance

| Chemin | Contenu | Propriétaire / mode |
| --- | --- | --- |
| `/data` | base, `-wal`, `-shm`, `.env` facultatif | `65532:65532`, `0750` |
| `/data/files` | fichiers déposés | `65532:65532`, `0750` |
| sauvegardes | fichiers `0600` créés par `frigg backup` | `65532` |

Les fichiers temporaires de SQLite restent en mémoire (`temp_store(MEMORY)`,
[research.md › R6](./research.md#r6-système-de-fichiers-en-lecture-seule--fichiers-temporaires-de-sqlite)).
