# Contrat d'interface : tarifs de l'adhésion

Complète [003 › ui-routes.md](../../003-season-management/contracts/ui-routes.md).
Aucune route ajoutée : la section s'insère dans la fiche d'une saison
(`/admin/saisons/:id`).

## Section « Tarifs de l'adhésion »

- Placée sous la liste de définitions de la fiche, titre `h2` « Tarifs de
  l'adhésion ». Quand la saison a au moins un tarif, bouton « Ajouter un
  tarif » (secondaire, icône `Plus`) dans l'en-tête de section si
  `actions.add`. Sans tarif, l'en-tête n'a pas de bouton : le seul bouton
  « Ajouter un tarif » de la page est celui de l'état vide.
- **Message informatif** : si `actions.edit` est vrai et qu'il n'y a aucun
  tarif, `Alert` `info` « Cette saison ne sera pas proposée à l'adhésion tant
  qu'elle n'a aucun tarif. » (FR-016).
- **État vide** : texte « Un tarif est le prix que paie un adhérent pour
  adhérer pendant la saison. » ; si `actions.add`, bouton « Ajouter un
  tarif » (principal) ; sur une saison close, texte « Aucun tarif n'a été
  défini pour cette saison. ».
- **Tableau** (`ui/Table`, `<caption>` « Tarifs de la saison {libellé} ») :
  Tarif (libellé en `th scope="row"`, description en texte secondaire
  dessous) · Montant (aligné à droite, « 30,00 € » ou « Gratuit » ; avec un montant de
  renouvellement, seconde ligne « Renouvellement 55,00 € ») · Actions
  (« Modifier » et « Supprimer », boutons icônes avec libellé accessible
  « Modifier le tarif {libellé} », présents seulement si `actions.edit` /
  `actions.delete` ; colonne absente sur une saison close).
- Plafond atteint (`actions.add` faux, saison non close) : texte « 20 tarifs
  au plus par saison. » à la place du bouton.
- À 320 px, le tableau défile dans son `ScrollRegion`.

## Dialogue d'ajout et de modification

- `Dialog` base-ui, titre « Ajouter un tarif » ou « Modifier le tarif
  {libellé} ».
- Champs (`ui/Field`) :
  - « Libellé » : `maxLength` 80, obligatoire ;
  - « Montant » : `type="text"`, `inputmode="decimal"`, aide « En euros. 0
    pour un tarif gratuit. » (pas de suffixe visuel : l'unité est dans l'aide,
    lue par les lecteurs d'écran) ; en modification, prérempli
    (« 15,50 », « 30 ») ;
  - case à cocher native « Tarif de renouvellement différent » ; cochée, elle
    affiche le champ « Montant du renouvellement » (`type="text"`,
    `inputmode="decimal"`, aide « En euros. »), et le champ « Montant » prend
    le libellé « Montant de première adhésion » ; en modification, cochée et
    préremplie si `renewalAmountCents` n'est pas nul ;
  - « Description (facultatif) » : `TextAreaField` (nouveau, sur le modèle
    de `Field`), `maxLength` 500, aide
    « Affichée sous le libellé, par exemple les conditions du tarif. ».
- Boutons « Ajouter » / « Enregistrer » (principal) et « Annuler ».
- Succès → fermeture, message « Tarif {libellé} ajouté. » / « Tarif
  {libellé} modifié. » dans la région `role="status"`. `422` → erreurs sur
  les champs ; `404`/`409` → `Alert` `danger` dans le dialogue et
  invalidation des tarifs et de la fiche.

## Suppression

- `ConfirmDialog` (`danger`) « Supprimer le tarif {libellé} ? », « Cette
  action est définitive. » ; succès → message « Tarif {libellé} supprimé. ».
- Focus rendu au déclencheur après fermeture d'un dialogue (ou au bouton
  « Ajouter un tarif » si la ligne a disparu).

## Données

- Requête `membershipFeesQuery(seasonId)` (`GET /admin/seasons/{id}/fees`),
  préchargée par le `clientLoader` de la fiche.
- Toute écriture de tarif invalide `["get", "/admin/seasons/{id}/fees"]`.
- `duplicate-season` et `delete-season` n'ont rien de plus à invalider : la
  nouvelle saison charge ses tarifs à l'ouverture de sa fiche.
