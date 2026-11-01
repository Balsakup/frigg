# Contrat d'interface : gestion des membres

Complète [002 › ui-routes.md](../../002-admin-account-management/contracts/ui-routes.md).
Les rubriques « Mes membres » (espace membre) et « Membres » (administration)
ne sont plus « bientôt disponibles » : retirées de `comingSoon` dans
`app/shell/navigation.ts` et des listes de `routes.ts`.

## Routes

| Route | Écran | Données |
| --- | --- | --- |
| `/mes-membres` | liste de mes membres | `list-my-members` |
| `/mes-membres/nouveau` | ajout | `list-my-members` (actions), `get-session` (adresse du compte) |
| `/mes-membres/:id` | fiche modifiable | `get-my-member` ; `404` → « Page introuvable » |
| `/admin/membres` | liste de tous les membres | `list-members` (`q`, `account=none`, `page` dans l'URL) |
| `/admin/membres/nouveau` | ajout | — |
| `/admin/membres/:id` | fiche modifiable | `get-member` ; `404` → « Page introuvable » |

Fil d'Ariane : « Mes membres › Ajouter un membre » / « Mes membres ›
{prénom} {nom} » ; « Membres › … » côté administration.

## « Mes membres » (`/mes-membres`)

- En-tête de page : titre « Mes membres », introduction « Les personnes que
  vous inscrivez à l'association : vous-même, vos enfants… », bouton
  « Ajouter un membre » (icône `UserPlus`) si `actions.add`.
- **État vide** : « Un membre est une personne que vous inscrivez à
  l'association. Commencez par vous ajouter, ou ajoutez vos enfants. » et
  bouton principal « Ajouter un membre ».
- **Liste** : `<ul>` de cartes, ordre de l'API. Chaque carte : lien « {prénom}
  {nom} » vers la fiche (titre `h2`), âge (« 11 ans »), badge « Vous » si
  `isSelf`. Une seule colonne à 320 px.
- Plafond atteint : texte « 20 membres au plus par compte. Contactez le
  bureau pour en ajouter. » à la place du bouton.

## Formulaire d'un membre (ajout et fiche, deux espaces)

Composant `MemberFields` (`ui/Field`, `react-hook-form`), groupes `fieldset` :

- **Identité** : « Prénom » (`maxLength` 80, requis, `autocomplete="given-name"`
  seulement si « Il s'agit de moi ») ; « Nom » (idem, `family-name`) ;
  « Date de naissance » (`type="date"`, requis).
- **Coordonnées** (facultatives) : « Adresse électronique » (`type="email"`,
  `maxLength` 254) ; « Téléphone » (`type="tel"`, `inputmode="tel"`, aide
  « 10 chiffres, ou format international (+32 …) »).
- **Adresse postale** (facultative, aide « Laissez vide ou complétez les trois
  champs obligatoires. ») : « Adresse », « Complément d'adresse », « Code
  postal » (`inputmode="numeric"`, `maxLength` 5), « Commune ».
- Champs propres à l'espace membre : case « Il s'agit de moi » si
  `actions.markSelf` ou si la fiche est déjà « Vous ». La cocher à l'ajout
  préremplit l'adresse électronique vide avec celle du compte.
- Champs propres à l'administration : « Compte de rattachement »
  (`AccountPicker`, ci-dessous).
- **Doublon** : sur une erreur `body.confirmDuplicate`, l'erreur s'affiche au
  pied du formulaire avec une case « Enregistrer quand même » ; la cocher et
  renvoyer envoie `confirmDuplicate: true`. Le focus est placé sur la case.
- Boutons : « Ajouter le membre » / « Enregistrer » (principal),
  « Annuler » (retour à la liste). Sur la fiche, « Supprimer » (danger,
  icône `Trash2`) ouvre un `ConfirmDialog` « Supprimer {prénom} {nom} ? »,
  texte « La fiche et ses informations seront effacées définitivement. ».
- Après enregistrement : retour à la liste avec notification « Membre
  ajouté. » / « Modifications enregistrées. » / « Membre supprimé. »
  (annoncée aux lecteurs d'écran).
- `404` / `409` en écriture : `Alert` `danger` avec le message de l'API ;
  la requête de liste est invalidée.

## « Membres » (`/admin/membres`)

- En-tête : titre « Membres », introduction « Toutes les personnes de
  l'association. », bouton « Ajouter un membre ».
- **Recherche** : champ « Rechercher un membre » (nom, prénom, adresse,
  téléphone) avec l'indication des 3 caractères minimum comme dans
  « Comptes » ; filtre `Select` « Rattachement » : « Tous » / « Sans compte ».
  Les critères sont dans l'URL (`?q=&account=none&page=`).
- **Tableau** (`ui/Table`, `<caption>` « Membres de l'association ») : Nom
  (lien « {nom} {prénom} » en `th scope="row"`) · Âge · Compte (adresse du
  compte, badge « désactivé » le cas échéant, ou « Sans compte » en texte
  secondaire). `ScrollRegion` à 320 px.
- Pagination et total (« 128 membres ») comme la liste des comptes ;
  aucun résultat : message et bouton « Effacer la recherche ».

## Fiche d'administration (`/admin/membres/:id`)

- Bloc d'informations au-dessus du formulaire (liste de définitions) :
  « Compte de rattachement » (lien vers `/admin/comptes/{id}`, ou « Sans
  compte »), « Fiche du titulaire » (« Oui » si `isSelf`), « Créée le »,
  « Modifiée le ».
- **AccountPicker** : Combobox base-ui « Compte de rattachement », saisie
  d'au moins 3 caractères interrogeant `list-accounts?q=` ; options : adresse
  (+ « désactivé ») ; option fixe « Sans compte » en tête. Changer le compte
  d'une fiche « Vous » affiche l'aide « Le signalement “Vous” sera retiré. ».
- « Enregistrer » envoie `expectedAccountId` (rattachement lu au chargement) ;
  sur `409` « Le rattachement de ce membre a changé entre-temps… », `Alert`
  `danger`, fiche rechargée, saisies non enregistrées conservées.

## Dialogue de suppression d'un compte (002, modifié)

Quand le compte gère au moins une fiche (`list-members?account={id}&pageSize=20`) :

- texte supplémentaire « Ce compte gère {n} fiche(s) de membre : » suivi de
  la liste des noms ;
- groupe de boutons radio obligatoire, sans choix par défaut :
  « Supprimer aussi ces fiches » / « Conserver ces fiches sans compte » ;
  le bouton de confirmation reste actif ; valider sans choix affiche l'erreur
  `query.members` sur le groupe ;
- la requête envoie `members` et `memberIds` ; sur `409` « La liste des
  fiches de ce compte a changé. », le dialogue reste ouvert, recharge la
  liste et affiche l'alerte.

Sans fiche, le dialogue de 002 est inchangé.

## Accessibilité

- Chaque `fieldset` a sa `legend` ; erreurs reliées par `aria-describedby`.
- Le Combobox annonce le nombre de résultats et l'indication des
  3 caractères.
- Parcours clavier complet sur les six écrans et les deux dialogues ;
  Playwright + axe sans violation `serious` ou `critical`.
