# Contrat d'interface : rubrique « Saisons »

Complète [002 › ui-routes.md](../../002-admin-account-management/contracts/ui-routes.md).
Aucune route publique ajoutée.

## Navigation

- Section du bas de la barre latérale : clé `settings` (ex-`resources`).
  Titre « Configuration » dans l'espace administration, « Ressources » dans
  l'espace membre.
- Espace administration, section « Configuration » : **Saisons**
  (`/admin/saisons`, icône `CalendarRange`) · Comptes · Aide.

## Routes

| Chemin | Fichier de route | Accès | Contenu |
| --- | --- | --- | --- |
| `/admin/saisons` | `routes/app/admin/seasons.tsx` | admin | Liste ou état vide |
| `/admin/saisons/nouvelle` | `routes/app/admin/season-new.tsx` | admin | Formulaire de création |
| `/admin/saisons/:id` | `routes/app/admin/season-detail.tsx` | admin | Fiche et actions ; saison inexistante → « Page introuvable » |

Les trois routes sont enfants de `routes/app/admin/seasons-layout.tsx` (fil
d'Ariane « Saisons »), sur le modèle de `accounts-layout.tsx`. `nouvelle` est
déclarée avant `:id`.

## Liste des saisons

- En-tête de page : titre « Saisons », introduction « Périodes d'activité de
  l'association, qui servent aussi d'exercice comptable. », action « Nouvelle
  saison » (lien vers `/admin/saisons/nouvelle`, masquée tant qu'aucune saison
  n'existe : l'état vide porte le bouton).
- **Tableau** (`ui/Table`, `<caption>` « Saisons de l'association, de la plus
  récente à la plus ancienne ») : Saison (lien vers la fiche, `th scope="row"`)
  · Début · Fin (« 1 septembre 2026 ») · État (badge : `warning` Brouillon,
  `ok` Ouverte, `muted` Close) · mention « Saison courante » (badge `brand`)
  dans la cellule Saison.
- **État vide** (aucune saison) : titre « Aucune saison pour le moment »,
  texte « Une saison délimite une période d'activité : adhésions, événements
  et comptes de l'année s'y rattachent. Créez la première pour mettre en place
  Frigg. », bouton principal « Créer la première saison ».
- À 320 px, le tableau défile dans son `ScrollRegion`, jamais la page.

## Création

- Fil d'Ariane : Tableau de bord › Saisons › Nouvelle saison. Titre
  « Nouvelle saison ».
- Champs (`ui/Field`, `type="date"`) :
  - « Date de début » : saisissable s'il n'y a aucune saison (aide « Elle peut
    être passée si la saison a déjà commencé. ») ; sinon champ en lecture
    seule avec aide « Le lendemain de la fin de la saison {libellé}. » ;
  - « Date de fin » : `min`/`max` depuis `nextEndsOnMin`/`nextEndsOnMax`
    (une saison suivante peut être passée), ou `today` pour la première
    saison ; aide « Dernier jour inclus. ».
- Pas d'aperçu du libellé : il est calculé par le serveur et s'affiche sur la
  fiche après création.
- Boutons « Créer la saison » (principal) et « Annuler » (lien vers la
  liste). Succès → fiche de la saison créée, message « Saison {libellé}
  créée. ». `409` → `Alert` `danger` au-dessus du formulaire, données de
  saisons rechargées.

## Fiche d'une saison

- Fil d'Ariane : Tableau de bord › Saisons › {libellé}. Titre `h1` =
  « Saison {libellé} » ; introduction = badges d'état et « Saison courante ».
- **Actions d'en-tête** (seules celles annoncées par `actions`) :
  « Ouvrir la saison » (principal) · « Modifier les dates » (secondaire) ·
  « Dupliquer » (secondaire) · « Supprimer » (`danger-secondary`).
- `openBlockedReason` non nul → `Alert` `info` en tête de fiche avec ce texte.
- Liste de définitions : Début · Fin · État · Ouverte le (si renseigné) ·
  Créée le.
- **Ouvrir** : `ConfirmDialog` (ton neutre) « Ouvrir la saison {libellé} ? »,
  « Les adhésions et inscriptions pourront s'y rattacher. Une saison ouverte
  ne peut plus être supprimée et seule sa date de fin restera modifiable. »
- **Modifier les dates** : `Dialog` base-ui, mêmes champs que la création ;
  « Date de début » présente seulement si `editStartsOn` ; `min`/`max` depuis
  `endsOnMin`/`endsOnMax`. Si `shiftsSeasonLabel` est non nul, aide
  « La saison {shiftsSeasonLabel} commencera le lendemain. »
- **Dupliquer** : pas de confirmation ; succès → fiche de la nouvelle saison,
  message « Saison {libellé} créée en brouillon. ».
- **Supprimer** : `ConfirmDialog` (`danger`) « Supprimer la saison
  {libellé} ? », « Cette action est définitive. » ; succès → liste, message
  « Saison {libellé} supprimée. ».
- Messages de succès et d'erreur dans la région `role="status"` existante ;
  focus rendu au déclencheur après fermeture d'un dialogue.

## Tableau de bord de l'administration

Encart (`ui/Alert`) au-dessus du message de bienvenue selon `setup` :

| `setup` | Ton | Texte | Lien |
| --- | --- | --- | --- |
| `missing` | `info` | « Créez la première saison pour mettre en place Frigg. » | « Créer la saison » → `/admin/saisons/nouvelle` |
| `no-current` | `info` | « Aucune saison n'est en cours. » | « Créer la saison suivante » → `/admin/saisons/nouvelle` |
| `current-draft` | `warning` | « La saison {libellé} a commencé mais n'est pas encore ouverte. » | « Voir la saison » → `/admin/saisons/{currentId}` |
| `ready` | — | aucun encart | — |

Le texte de bienvenue mentionne aussi la rubrique « Saisons ».
