# Contrat : construction d'une version du site

Interface entre le workflow `docs.yml` (extrait de `main`) et
`docs/.vitepress/config.mts`, qui est celle de la source construite : `main`
ou l'étiquette de la dernière release.

Ce contrat MUST rester rétrocompatible : une release construite avec une
configuration plus ancienne doit continuer à produire un site correct. Pour
ajouter une variable, on lui donne une valeur par défaut ; on ne renomme ni
ne supprime jamais une variable.

## Commande

```sh
cd docs
bun install --frozen-lockfile
bun run build          # = bun --bun vitepress build
```

Sortie : `docs/.vitepress/dist/`.

## Variables d'entrée

| Variable | Exemple | Défaut (local) | Rôle |
| --- | --- | --- | --- |
| `DOCS_BASE` | `/frigg/` ou `/frigg/main/` | `/` | `base` VitePress |
| `DOCS_VERSION` | `v1.2.0-rc.1` ou `main` | `main` | Texte du sélecteur ; `main` est affiché « main (en développement) » |
| `DOCS_RELEASE_VERSION` | `v1.2.0-rc.1` ou vide | vide | Étiquette de la release publiée à côté. Vide = pas d'entrée « Dernière release » dans le sélecteur |
| `DOCS_SITE_URL` | `https://balsakup.github.io/frigg/` | `https://balsakup.github.io/frigg/` | URL absolue de la racine du site, qui sert à construire les liens du sélecteur |

Liens du sélecteur, identiques dans les deux constructions :

- dernière release : `${DOCS_SITE_URL}` ;
- main : `${DOCS_SITE_URL}main/`.

## Échecs

La commande se termine avec un code non nul si :

- une page contient un lien interne mort (comportement par défaut de
  VitePress, `ignoreDeadLinks` non défini). Le message nomme le lien et le
  fichier : `Found dead link ./a/nope in file index.md` ;
- une page ne se compile pas.
