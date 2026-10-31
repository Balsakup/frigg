# Quickstart : valider la gestion des saisons

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[ui-routes.md](./contracts/ui-routes.md). Prérequis et mise en place : ceux de
[001 › quickstart.md](../001-auth-app-shell/quickstart.md) (Mailpit,
`create-admin`, `go tool task dev`).

## Contrôles automatisés (constitution)

```sh
go tool task check
```

Attendu en plus de 002 :

- la migration `00004_seasons.sql` s'applique sur une base vierge et sur une
  base de développement existante ;
- la matrice d'autorisation couvre les 7 nouvelles opérations (anonyme → 401,
  membre → 403, administrateur accepté) ;
- tests unitaires de `internal/season` (en table) : libellés (année civile,
  deux années, trois années, homonymes désambiguïsés), états et saison
  courante autour des bornes (veille, jour de fin, lendemain), duplication
  (annuelle, semestrielle, mois entiers, nombre de jours, 29 février),
  actions autorisées pour chaque état et position, bornes `endsOnMin`/
  `endsOnMax`, validations (fin passée, 24 mois, début imposé, suivante
  invalide) ;
- tests d'API avec horloge injectée : création, modification avec décalage de
  la suivante, refus `409` et `422` du contrat, clôture constatée en avançant
  l'horloge, deux duplications concurrentes → une seule saison créée ;
- tests Vitest : `formatDay` (pas de décalage de fuseau), navigation
  (section « Configuration », ordre Saisons · Comptes · Aide) ;
- Playwright + axe sur `/admin/saisons` (vide et rempli),
  `/admin/saisons/nouvelle`, `/admin/saisons/:id` et le tableau de bord, y
  compris à 320 px : aucune violation `serious` ou `critical`.

## Jeu de données

Les tests Playwright partent d'une base sans saison et créent les leurs, dans
un seul fichier exécuté en série (`admin-seasons.spec.ts`) : l'état vide et
l'encart du tableau de bord doivent être observés avant toute création. Les
dates sont calculées à partir de la date du jour.

## Scénarios manuels

| # | Parcours (spec) | Étapes | Résultat attendu |
| --- | --- | --- | --- |
| 1 | Navigation | En administrateur, regarder la barre latérale, puis passer dans l'espace membre | Section « Configuration » : Saisons, Comptes, Aide ; espace membre : « Ressources » inchangé |
| 2 | US5 — tableau de bord vide | Base sans saison, ouvrir `/admin` | Encart « Créez la première saison… », lien vers la création |
| 3 | US1 — état vide | Ouvrir `/admin/saisons` | État vide et bouton « Créer la première saison » ; pas de bouton « Nouvelle saison » |
| 4 | US1 — création | Créer une saison commencée le 1er septembre dernier et finissant le 31 août prochain | Fiche « Saison AAAA-AAAA », Brouillon, « Saison courante » ; tableau de bord : encart « …n'est pas encore ouverte » |
| 5 | US1 — erreurs | Fin avant début ; fin hier ; 25 mois | Erreur sur le champ « Date de fin », message du contrat |
| 6 | US2 — ouverture | « Ouvrir la saison », confirmer | État Ouverte ; tableau de bord sans encart ; « Supprimer » disparaît |
| 7 | US2 — duplication | « Dupliquer » | Fiche de la saison suivante, Brouillon, dates décalées d'un an ; la liste montre les deux, une seule « Saison courante » |
| 8 | US2 — pré-inscriptions | Ouvrir la saison suivante | Deux saisons Ouvertes ; la courante ne change pas |
| 9 | US2 — ordre d'ouverture | Dupliquer deux fois, ouvrir la fiche de la dernière | Message « Ouvrez d'abord la saison … », pas de bouton d'ouverture |
| 10 | US2 — création manuelle | « Nouvelle saison » | Date de début en lecture seule (lendemain de la dernière fin) |
| 11 | US3 — modification | Modifier la fin d'une saison ouverte suivie d'un brouillon | Le début du brouillon suit ; impossible de choisir une fin passée ; pas de champ « Date de début » |
| 12 | US3 — suivante ouverte | Saison suivie d'une saison ouverte | « Modifier les dates » absent ; `curl PATCH` → 409 |
| 13 | US3 — suppression | Supprimer le dernier brouillon, confirmer | Retour à la liste, message de confirmation ; action absente sur les autres saisons |
| 14 | US4 — clôture | Avec une saison finissant aujourd'hui, revenir le lendemain (ou créer en base une saison finie hier) | La saison apparaît Close, sans action sauf « Dupliquer » si elle est la dernière ; tableau de bord : « Aucune saison n'est en cours » s'il n'y a pas de suivante |
| 15 | Homonymes | Créer deux saisons semestrielles d'une même année | Libellés « AAAA (janv.–juin) » et « AAAA (juil.–déc.) » |
| 16 | Concurrence | Deux onglets sur la dernière saison ; « Dupliquer » dans les deux | Le second affiche « La liste des saisons a changé… » ; une seule saison créée |
| 17 | Journaux | Effectuer chaque action, lire la sortie du serveur | Une ligne `INFO` par écriture avec `action`, `actor_id`, `season_id` |
| 18 | Accessibilité | Parcours complet au clavier, à 320 px, avec un lecteur d'écran | Champs de date utilisables au clavier ; confirmations annoncées ; focus rendu après fermeture d'un dialogue |
