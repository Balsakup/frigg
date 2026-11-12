# Contribuer à Frigg

Merci de votre intérêt pour Frigg ! Ce guide explique comment proposer une
contribution.

## Licence et CLA

Frigg est distribué sous la licence
[GNU Affero General Public License v3.0 ou ultérieure](https://github.com/Balsakup/frigg/blob/main/LICENSE)
(AGPL-3.0-or-later).

Avant de fusionner votre première Pull Request, nous vous demandons
d'accepter l'[accord de licence de contribution](https://github.com/Balsakup/frigg/blob/main/CLA.md)
(CLA). En résumé :

- vous restez titulaire des droits d'auteur sur votre code ;
- vous accordez au mainteneur une licence large qui lui permet notamment
  d'utiliser vos contributions dans l'offre d'hébergement de Frigg, y compris
  sous d'autres conditions que l'AGPL ;
- vous garantissez avoir le droit de contribuer ce code. **Si vous contribuez
  dans le cadre de votre travail, demandez l'accord de votre employeur.**

Le résumé ci-dessus n'a pas de valeur juridique : seul le texte du CLA fait
foi.

À l'ouverture de votre première Pull Request, le robot CLA Assistant publie
un commentaire. Pour accepter le CLA, répondez par le commentaire exact :

```
J'ai lu le CLA de Frigg et je l'accepte.
```

L'acceptation est enregistrée une fois pour toutes. Pour relancer la
vérification, commentez `recheck`.

Si vous reprenez du code d'un autre projet, indiquez-le dans la description
de la Pull Request en précisant son auteur et sa licence.

## Proposer une modification

1. Créez une branche à partir de `main`.
2. Ajoutez des tests pour toute modification de comportement.
3. Mettez à jour la [documentation](./documentation) si le comportement
   visible change.
4. Vérifiez que `go tool task check` passe (voir
   [Contrôles avant fusion](./checks)).
5. Rédigez vos messages de commit en français (voir
   [Cycle de développement](./workflow#commits)).
6. Ouvrez une Pull Request en décrivant le problème résolu et la solution.

## Pour aller plus loin

- [Mettre en place l'environnement](./setup)
- [Cycle de développement](./workflow)
- [Rédiger la documentation](./documentation)
- [Publier une release](./release)
