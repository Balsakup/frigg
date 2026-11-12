# Rédiger la documentation

Cette documentation est construite avec [VitePress](https://vitepress.dev) à
partir du dossier `docs/` du dépôt. Elle est versionnée avec le code : la
documentation d'une release est celle de son étiquette.

## Règle de simplicité

`docs/` ne contient que :

- des pages **Markdown** ;
- la configuration `docs/.vitepress/config.mts` (navigation, barre latérale,
  recherche, sélecteur de version) ;
- `package.json` et `bun.lock`.

Le thème par défaut est utilisé tel quel : pas de composant, de style, de
police, de script ni de test propre à la documentation.

## Écrire une page

- Une page par sujet, rangée dans le guide concerné : `installation/`,
  `usage/` ou `contributing/`. Noms de fichiers en anglais, contenu en
  français.
- Ajoutez la page à la barre latérale (`sidebar`) dans `config.mts`.
- Dans le guide d'utilisation, commencez chaque page, sous son titre, par
  `**Rôle requis** : Membre` ou `**Rôle requis** : Administrateur`.
- Liens entre pages : chemins relatifs (`./seasons`, `../installation/docker`).
- Liens vers un fichier du dépôt **hors de `docs/`** (`CLA.md`,
  `.env.example`, `specs/…`) : toujours une URL absolue
  `https://github.com/Balsakup/frigg/blob/main/<chemin>`. Un lien relatif
  serait vu comme un lien mort.
- Aucune donnée personnelle ni secret dans les exemples (domaines
  `exemple.fr`, `mon-asso.fr`).
- N'appelez aucune page ni aucun dossier `main` : ce chemin est réservé à la
  version en développement.

Une Pull Request qui change un comportement visible par un exploitant, un
administrateur ou un membre met à jour la page correspondante.

## Prévisualiser et construire

```sh
go tool task docs:dev     # http://localhost:5174, rechargement à chaque enregistrement
go tool task docs:build   # construction comme en production
```

La construction échoue sur un lien interne mort, en nommant le lien et la
page (`Found dead link ./nope in file index.md`). Le workflow
`Documentation` fait la même vérification sur chaque Pull Request qui modifie
`docs/`.

## Versions publiées

Le site <https://balsakup.github.io/frigg/> publie deux versions :

| Version | Adresse | Source |
| --- | --- | --- |
| Dernière release | `/frigg/` | Étiquette de la plus haute release publiée, préversions comprises |
| main (en développement) | `/frigg/main/` | Branche `main` |

Le sélecteur de version, dans la barre de navigation, passe de l'une à
l'autre. Tant qu'aucune release ne contient la documentation, `/frigg/`
redirige vers `/frigg/main/`.

Le workflow `.github/workflows/docs.yml` reconstruit et publie les deux
versions à chaque fusion sur `main` qui modifie `docs/`, à chaque release
publiée, et à la demande (« Run workflow »).

## Variables de construction

La configuration lit quatre variables, fournies par le workflow :
`DOCS_BASE`, `DOCS_VERSION`, `DOCS_RELEASE_VERSION` et `DOCS_SITE_URL`.
Chaque release est construite avec sa propre configuration : **ne renommez ni
ne supprimez jamais ces variables**. Pour en ajouter une, donnez-lui une
valeur par défaut. Le contrat est décrit dans
[specs/015-vitepress-docs/contracts/build-env.md](https://github.com/Balsakup/frigg/blob/main/specs/015-vitepress-docs/contracts/build-env.md).

## Réglage du dépôt

GitHub Pages doit être réglé une fois : Settings → Pages → Source « GitHub
Actions ». L'environnement `github-pages` n'autorise les déploiements que
depuis `main` et depuis les étiquettes `v*`.
