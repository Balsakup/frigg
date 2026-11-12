# Champs personnalisés

**Rôle requis** : Administrateur pour définir les champs ; Membre pour remplir
ceux qui lui sont ouverts

Les champs personnalisés ajoutent aux fiches et aux adhésions les
informations propres à votre association. Ils se définissent dans
*Configuration › Paramètres › Champs personnalisés*.

## Deux portées

| Portée | Exemple | Durée de vie de la valeur |
| --- | --- | --- |
| **Champ de membre** | Numéro de licence fédérale | Attachée à la fiche, conservée d'une saison à l'autre |
| **Champ d'adhésion** | « A payé sa licence fédérale » | Attachée à une seule adhésion ; repart vide à chaque saison |

Chaque portée compte au plus 30 champs actifs.

## Définir un champ

- **Libellé** : obligatoire, 80 caractères au plus, unique dans sa portée.
- **Type** : texte court (200 caractères), texte long (2 000 caractères),
  nombre, date, oui/non, ou liste de choix (2 à 50 choix, une seule valeur
  retenue).
- **Texte d'aide** facultatif (200 caractères au plus).
- **Obligatoire** ou facultatif. Un oui/non ne peut pas être obligatoire.
- **Visibilité** :

  | Visibilité | Le membre… |
  | --- | --- |
  | Réservé à l'administration (par défaut) | ne voit jamais le champ |
  | Visible par le membre | le voit, en lecture seule |
  | Rempli par le membre | le voit et le remplit |

Les champs se réordonnent à la souris ou au clavier.

## Règles à connaître

- Un champ obligatoire n'est exigé que de la personne qui peut le remplir, au
  prochain enregistrement du formulaire. La section indique, pour chaque champ
  obligatoire, combien de fiches ou d'adhésions de la saison en cours n'ont
  pas encore de valeur.
- Le **type** ne change plus dès qu'une valeur est enregistrée. Un choix peut
  être ajouté ou renommé, mais pas retiré s'il est utilisé.
- **Archiver** retire le champ des formulaires en conservant ses valeurs, qui
  réapparaissent s'il est rétabli.
- **Supprimer** efface le champ et toutes ses valeurs, après une confirmation
  qui indique le nombre de valeurs perdues.

## Exploiter les valeurs

- Les listes *Membres* et *Adhésions* se filtrent sur les champs oui/non et
  liste de choix, par exemple les adhésions de la saison dont la licence n'est
  pas payée.
- La recherche des membres trouve un fragment d'au moins 3 caractères d'un
  champ texte court, par exemple une partie du numéro de licence.
- Les champs archivés ne sont ni filtrables ni recherchés.
