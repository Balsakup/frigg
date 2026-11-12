# Quickstart : valider le site de documentation

Prérequis : Bun (guide de contribution), `gh` authentifié pour le § 3.

## 1. Prévisualiser en local (US4)

```sh
go tool task docs:dev       # http://localhost:5174, rechargement à chaque enregistrement
```

Attendu :

- le sélecteur affiche `main (en développement)` et ne propose pas
  « Dernière release » ;
- une modification de `docs/index.md` apparaît sans relancer la commande ;
- la recherche d'un terme (« sauvegarde ») propose la page correspondante.

## 2. Construire comme en production

```sh
go tool task docs:build
```

Attendu : succès. Ajouter ensuite `[x](./nope)` dans une page et relancer :
la construction échoue avec `Found dead link ./nope in file …`. Retirer le
lien.

Vérifier aussi que `docs/` ne contient que du Markdown, la configuration, le
manifeste et le verrou (SC-006) :

```sh
git ls-files docs | grep -vE '\.md$|^docs/\.vitepress/config\.mts$|^docs/package\.json$|^docs/bun\.lock$'
```

Attendu : aucune ligne.

## 3. Choix de la dernière release

```sh
gh release list --exclude-drafts --json tagName -q '.[].tagName' \
  | grep -E '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$' \
  | sed 's/-/~/' | sort -V | sed 's/~/-/' | tail -n1
```

Attendu : `v0.0.3-alpha` aujourd'hui. Cette étiquette ne contient pas `docs/`,
donc seule la version « main » sera publiée.

## 4. Publication réelle (US1, US2, US3)

1. Une fois : Settings → Pages → Source « GitHub Actions », et les règles de
   l'environnement `github-pages` ([workflow.md](./contracts/workflow.md)).
2. Fusionner la fonctionnalité sur `main`. En moins de 10 min, le workflow
   « Documentation » est vert, `https://balsakup.github.io/frigg/` redirige
   vers `/frigg/main/`, et le résumé indique qu'aucune release n'est
   documentée.
3. Publier une préversion (par exemple `v0.1.0-alpha`). En moins de 10 min :
   - `/frigg/` affiche `v0.1.0-alpha` dans le sélecteur ;
   - le sélecteur mène à `/frigg/main/`, et inversement ;
   - `/frigg/nexiste-pas` affiche la page introuvable.
4. Fusionner une modification d'une page : seule `/frigg/main/` change.
5. Fusionner une modification hors `docs/` : aucune exécution
   « Documentation » n'est lancée.
6. Ouvrir une PR qui ajoute un lien mort : le job échoue en nommant le lien,
   et rien n'est publié.
