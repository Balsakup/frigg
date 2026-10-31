# Quickstart : valider les tarifs de l'adhésion

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[ui-routes.md](./contracts/ui-routes.md). Prérequis et mise en place : ceux de
[001 › quickstart.md](../001-auth-app-shell/quickstart.md) (Mailpit,
`create-admin`, `go tool task dev`).

## Contrôles automatisés (constitution)

```sh
go tool task check
```

Attendu en plus de 003 :

- la migration `00005_membership_fees.sql` s'applique sur une base vierge et
  sur une base de développement existante ;
- `CGO_ENABLED=0 go list -deps ./cmd/frigg` inclut
  `golang.org/x/text/unicode/norm` et aucun paquet CGO ;
- la matrice d'autorisation couvre les 4 nouvelles opérations (anonyme → 401,
  membre → 403, administrateur accepté) ;
- tests unitaires de `internal/textfold` (casse, accents, espaces, ligatures
  courantes) ;
- tests en table de `internal/season/fee.go` : `ParseAmount` (« 30 »,
  « 15,5 », « 15.50 », « 1 000 », « 0 », « -1 », « 1,234 », « abc »,
  « 10000,01 »), nettoyage et longueurs, unicité repliée, plafond, actions
  selon l'état ;
- tests d'API : ajout, modification, suppression ; `422` du contrat ; `409`
  sur saison close (horloge avancée) et au 21e tarif ; `404` pour un tarif
  d'une autre saison ; duplication recopiant les tarifs (copies
  indépendantes) ; suppression de saison en cascade ; deux ajouts simultanés
  du même libellé → un seul créé ;
- Vitest : `formatAmount` (« 15,50 € », « Gratuit ») et `amountInput`
  (« 15,5 », « 30 ») ;
- Playwright + axe sur la fiche d'une saison avec et sans tarif, dialogues
  ouverts, à 320 px : aucune violation `serious` ou `critical`.

## Jeu de données

Les tests Playwright des tarifs forment un second test du fichier
`admin-seasons.spec.ts`, exécuté en série après celui des saisons : ce fichier
est le seul à créer des saisons (le premier test exige une base sans saison),
et le test des tarifs réutilise les saisons qu'il laisse. Les dates sont
calculées à partir de la date du jour.

## Scénarios manuels

| # | Parcours (spec) | Étapes | Résultat attendu |
| --- | --- | --- | --- |
| 1 | US1 — état vide | Ouvrir la fiche d'une saison en brouillon sans tarif | Section « Tarifs de l'adhésion », message informatif, bouton « Ajouter un tarif » |
| 2 | US1 — ajout | Ajouter « Plein tarif » 30 ; « Tarif réduit » 15,50 avec description | Tableau : Plein tarif 30,00 € puis Tarif réduit 15,50 € et sa description ; message informatif disparu |
| 2b | US1 — renouvellement | Ajouter « Adulte compétition » 85, cocher « Tarif de renouvellement différent », 55 ; puis cocher sans saisir de montant | « 85,00 € » et « Renouvellement 55,00 € » ; case cochée sans montant → erreur « Saisissez le montant du renouvellement. » |
| 3 | US1 — gratuit | Ajouter « Membre d'honneur » 0 | « Gratuit », en dernier |
| 4 | US1 — erreurs | Libellé « plein TARIF » ; montant « -5 », « 12,345 », « 20000 » ; libellé vide | Erreur sur le champ concerné, message du contrat |
| 5 | US3 — modification | Modifier le montant et la description d'un tarif d'une saison ouverte | Valeurs enregistrées ; ordre recalculé |
| 6 | US3 — suppression | Supprimer tous les tarifs d'une saison ouverte | Confirmation demandée ; dernier supprimé ; message informatif réapparu ; saison toujours ouverte |
| 7 | US4 — ouverture | Saison en brouillon sans tarif dont la précédente est ouverte | « Ouvrir la saison » proposé ; ouverture possible |
| 8 | US2 — duplication | Dupliquer la dernière saison portant 3 tarifs | Fiche de la nouvelle saison avec les 3 tarifs ; en modifier un, l'original est inchangé |
| 9 | US2 — création manuelle | « Nouvelle saison » | Fiche sans tarif |
| 10 | US3 — saison close | Fiche d'une saison close avec tarifs (créer en base une saison finie hier) | Tarifs affichés sans bouton ni colonne d'actions ; pas de message informatif ; `curl POST` → 409 |
| 11 | Plafond | Atteindre 20 tarifs | Bouton remplacé par « 20 tarifs au plus par saison. » ; `curl POST` → 409 |
| 12 | Suppression de saison | Supprimer une saison en brouillon portant des tarifs | Saison et tarifs supprimés (vérifier en base) |
| 13 | Journaux | Effectuer chaque action | Une ligne `INFO` par écriture, sans libellé ; `fees_copied` sur la duplication |
| 14 | Accessibilité | Parcours au clavier, à 320 px, lecteur d'écran | Boutons d'action nommés par tarif ; erreurs annoncées ; focus rendu après fermeture |
