# Contrat du processus de release

Approche : [research.md › R11, R12, R13](../research.md).

## Déclencheur

Publication d'une release GitHub (`release: published`), y compris une
préversion. Un brouillon ne déclenche rien tant qu'il n'est pas publié.

## Étiquette de la release

| Étiquette | Résultat |
| --- | --- |
| `v1.2.0` (plus haute stable) | `1.2.0` + `latest` |
| `v1.1.4` publiée après `v1.2.0` | `1.1.4` seulement |
| `v1.3.0-rc.1` | `1.3.0-rc.1` seulement |
| `v1.3.0` marquée « préversion » | `1.3.0` seulement |
| `1.2.0`, `v1.2`, `v01.2.0`, `release-1` | échec : `étiquette <tag> invalide : format attendu vX.Y.Z ou vX.Y.Z-suffixe` |

## Étapes et effets

| Job | Permissions | Échec → |
| --- | --- | --- |
| `verify` : format, version, contrôles Go et frontend | `contents: read` | rien n'est publié |
| `image` : construction, analyse de vulnérabilités, publication, attestations | `contents: read`, `packages: write`, `id-token: write`, `attestations: write` | rien n'est publié si l'analyse échoue |
| `announce` : image et empreinte ajoutées aux notes de la release | `contents: write` | l'image est publiée, la note manque (relancer le job) |

Contrôles de `verify` : `bun install --frozen-lockfile`, `bun run build`,
`CGO_ENABLED=0 go build ./...`, `go vet ./...`, `go test ./...`,
`bun run typecheck`, `bun run lint`, `bun run test`.

Analyse : gravités `CRITICAL` et `HIGH`, vulnérabilités sans correctif
ignorées ; toute autre détection fait échouer le job.

## Ajout aux notes de la release

```markdown
**Image** : `ghcr.io/balsakup/frigg:1.2.0`
(`sha256:…`, linux/amd64 et linux/arm64)
```

## Relance

« Re-run jobs » sur l'exécution de la release : reconstruit le même code et
remplace l'image sous la même étiquette ; `latest` est recalculé selon la même
règle.

## Opération unique

Au premier passage, rendre le paquet `frigg` public (Paramètres du paquet →
Change visibility → Public).
