# Membres

**Rôle requis** : Membre pour *Mes membres*, Administrateur pour la rubrique
*Membres*

Une **fiche de membre** décrit une personne de l'association. Elle est
distincte du compte qui permet de se connecter : un compte gère une ou
plusieurs fiches (un parent et ses enfants), et une fiche peut exister sans
compte.

## Contenu d'une fiche

| Champ | Règle |
| --- | --- |
| Prénom, nom | Obligatoires, 80 caractères au plus chacun |
| Date de naissance | Obligatoire, ni future ni antérieure de plus de 120 ans |
| Adresse électronique, téléphone, adresse postale | Facultatifs, vérifiés s'ils sont renseignés |
| Informations complémentaires | [Champs personnalisés](./custom-fields) de l'association |

Frigg signale un **doublon probable** (même prénom, même nom, même date de
naissance, sans tenir compte des majuscules ni des accents) et demande une
confirmation avant d'enregistrer.

## Mes membres (espace membre)

Chaque compte, membre ou administrateur, gère ses fiches dans *Mes membres* :

- il peut les consulter, en créer, les modifier et les supprimer ;
- une fiche créée ici est rattachée à son compte, et seul un administrateur
  peut changer ce rattachement ;
- une fiche par compte peut être marquée comme la sienne (« Vous ») ; elle
  apparaît en premier ;
- un compte gère au plus 20 fiches.

Un compte ne voit jamais les fiches qu'il ne gère pas.

## Membres (administration)

La rubrique *Membres* liste toutes les fiches, triées par nom puis prénom.

- **Recherche** : tout fragment d'au moins 3 caractères du prénom, du nom, de
  l'adresse électronique, du téléphone ou d'un champ personnalisé de type
  texte court, sans tenir compte des majuscules ni des accents.
- **Filtres** : fiches « Sans compte », et valeurs des champs personnalisés
  oui/non et liste de choix.
- Un administrateur peut créer, modifier et supprimer n'importe quelle fiche,
  et choisir, changer ou retirer le compte qui la gère.
- Pour enregistrer de nombreuses fiches d'un coup, utilisez
  l'[importation](./member-import).

## Supprimer

La suppression d'une fiche demande confirmation et efface réellement ses
données.

Supprimer un **compte** qui gère des fiches impose un choix explicite :
supprimer ces fiches avec le compte, ou les conserver sans compte.
