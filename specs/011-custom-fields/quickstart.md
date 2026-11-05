# Quickstart : valider les champs personnalisés

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[ui-routes.md](./contracts/ui-routes.md). Modèle : [data-model.md](./data-model.md).
Prérequis : ceux de [006 › quickstart.md](../006-memberships/quickstart.md)
(`create-admin`, `go tool task dev`), une saison en cours avec un tarif, un
compte membre « Claire » qui gère la fiche de son fils « Lucas ».

## Contrôles automatisés (constitution)

```sh
go tool task check
```

Attendu en plus de 010 :

- `00011_custom_fields.sql` s'applique sur une base vierge et sur une base
  `00010` peuplée ;
- `sqlc generate`, `frigg openapi` et `bun run gen:api` sans diff ;
- matrice d'autorisation : 10 opérations ajoutées (8 `admin`, 2 `member`),
  aucune publique ;
- tests en table de `internal/customfield` ;
- test de fuite : en membre, les réponses de `get-my-member`,
  `get-my-membership` et `list-my-custom-fields` ne contiennent ni l'id ni le
  libellé d'un champ réservé ;
- test de cohérence FTS après suppression d'une fiche et d'un champ ;
- Playwright + axe : `custom-fields.spec.ts` sans violation `serious` ni
  `critical`.

## Scénarios manuels

### 1. Définir les champs (US1)

1. Administrateur → « Paramètres » → « Champs personnalisés ». **Attendu** :
   deux états vides avec les exemples de la licence fédérale.
2. Champs des membres → « Ajouter un champ » : « Numéro de licence
   fédérale », Texte court, Rempli par le membre, aide « 7 chiffres ».
   **Attendu** : champ listé, notification.
3. Champs des adhésions : « A payé sa licence fédérale », Oui/non, réservé
   (la case « Obligatoire » disparaît pour Oui/non). Ajouter aussi « Taille
   de t-shirt », Liste de choix (S, M, L), Rempli par le membre, obligatoire.
4. Ajouter un champ de membre « numero de licence federale ». **Attendu** :
   `422` sur le libellé (doublon sans casse ni accents).
5. Liste de choix avec un seul choix. **Attendu** : `422` sur les choix.
6. Déplacer un champ au clavier avec « Monter ». **Attendu** : ordre conservé
   après rechargement, annonce dans la région `status`.

### 2. Champs de membre (US2)

1. Fiche de Lucas (administration) : saisir `1234567`. **Attendu** :
   enregistré, affiché dans « Informations complémentaires ».
2. Ajouter un champ de membre réservé « Remarque du bureau », le remplir.
3. Se connecter en Claire → « Mes membres » → Lucas. **Attendu** : le numéro
   est modifiable ; « Remarque du bureau » n'apparaît pas (vérifier aussi la
   réponse réseau de `get-my-member`).
4. En Claire, envoyer à la main `PUT /api/me/members/{id}` avec le `fieldId`
   de « Remarque du bureau ». **Attendu** : `422` « Ce champ n'existe pas ou
   n'est pas modifiable. », valeur inchangée.
5. Rendre le numéro obligatoire. **Attendu** : la section signale les fiches
   sans valeur. En Claire, créer une fiche sans numéro. **Attendu** : `422`
   sur le numéro. En administrateur, rendre « Remarque du bureau »
   obligatoire et vide sur la fiche de Lucas, puis modifier son téléphone en
   Claire. **Attendu** : enregistré (champ réservé non exigé du membre).

### 3. Champs d'adhésion (US3)

1. En Claire, demander l'adhésion de Lucas. **Attendu** : « Taille de
   t-shirt » proposée et exigée ; « A payé sa licence fédérale » absent.
2. En administrateur, ouvrir la demande, cocher « A payé sa licence
   fédérale », valider. **Attendu** : valeur enregistrée avec la validation.
3. Constater le paiement de l'adhésion, puis décocher et recocher le champ.
   **Attendu** : enregistré (`update-membership-fields`), état et paiement
   inchangés.
4. Créer une adhésion de Lucas pour la saison suivante. **Attendu** : le
   champ repart à « Non ».
5. Annuler une adhésion. **Attendu** : valeurs affichées en lecture seule ;
   `PUT …/fields` renvoie `409`.

### 4. Filtres et recherche (US4)

1. « Adhésions », saison en cours, filtre « A payé sa licence fédérale :
   Non ». **Attendu** : seules les adhésions non cochées ; nombre annoncé ;
   filtre conservé dans l'URL.
2. « Membres », rechercher `3456`. **Attendu** : Lucas trouvé.
3. Archiver le champ licence, rechercher `3456`. **Attendu** : aucun
   résultat ; le filtre n'est plus proposé. Rétablir : valeur intacte et de
   nouveau trouvée.

### 5. Suppression (US5)

1. Supprimer « Taille de t-shirt ». **Attendu** : dialogue annonçant le
   nombre de valeurs perdues, proposition d'archiver à la place.
2. Confirmer. **Attendu** : champ absent partout ; valeurs supprimées.
3. Tenter de retirer d'une liste un choix utilisé. **Attendu** : `422` sur
   la ligne du choix ; le renommer : nouvelle étiquette affichée sur les
   adhésions concernées.
