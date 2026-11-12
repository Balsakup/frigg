# Data Model: Site de documentation versionné

Aucune base de données ni migration : le site est statique. Les « entités »
ci-dessous sont des valeurs calculées par le workflow.

## Release (lue depuis GitHub)

| Champ | Règle |
| --- | --- |
| `tagName` | Retenue seulement si elle respecte `^v(0\|[1-9][0-9]*)\.(0\|[1-9][0-9]*)\.(0\|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$` (spécification 007) |
| brouillon | Exclu (`--exclude-drafts`) |

**Dernière release** : la plus haute des étiquettes retenues dans l'ordre
des versions ([research.md › R6](./research.md)). Elle est absente s'il n'y a
aucune étiquette valide, ou si l'étiquette ne contient pas
`docs/.vitepress/config.mts`.

| Étiquettes publiées | Dernière release |
| --- | --- |
| aucune | absente |
| `v1.0.0`, `v1.1.0-rc.1` | `v1.1.0-rc.1` |
| `v1.0.0-rc.2`, `v1.0.0` | `v1.0.0` |
| `v1.2.0`, `v1.1.4` | `v1.2.0` |
| `v1.3.0-rc.9`, `v1.3.0-rc.10` | `v1.3.0-rc.10` |
| `v0.0.1-alpha` à `v0.0.3-alpha` (sans `docs/`) | absente |

## Version de documentation

| Champ | « release » | « main » |
| --- | --- | --- |
| identifiant | étiquette (`v1.2.0-rc.1`) | `main` |
| source | étiquette Git | branche `main` |
| `base` | `/frigg/` | `/frigg/main/` |
| texte du sélecteur | `v1.2.0-rc.1` | `main (en développement)` |
| présence | si la dernière release existe | toujours |

Transitions de la racine du site :

```text
aucune release documentée ──(release publiée avec docs/)──▶ racine = release
racine = release ──(seule release documentée supprimée ou repassée en brouillon)──▶ racine = redirection vers main
racine = release vA ──(release vB > vA publiée)──▶ racine = release vB
racine = release vB ──(release vA < vB publiée)──▶ inchangée
```
