# Quickstart : valider les événements

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[ui-routes.md](./contracts/ui-routes.md). Modèle : [data-model.md](./data-model.md).
Prérequis : ceux de [006 › quickstart.md](../006-memberships/quickstart.md)
(`create-admin`, `go tool task dev`) ; une saison en cours avec un tarif ; un
compte membre « Claire » qui gère sa fiche « Vous » et celle de son fils
« Lucas » ; Claire adhérente validée de la saison, Lucas non ; chèque et
virement activés ; pour la carte, HelloAsso en mode test comme dans
[008 › quickstart.md](../008-helloasso-card-payment/quickstart.md).

## Contrôles automatisés (constitution)

```sh
go tool task check
```

Attendu en plus de 011 :

- `00012_events.sql` s'applique sur une base vierge et sur une base `00011`
  peuplée (adhésion payée par carte, signalement ouvert) : données
  identiques, `PRAGMA foreign_key_check` vide, `PRAGMA foreign_keys` = 1 ;
- `sqlc generate`, `frigg openapi` et `bun run gen:api` sans diff ;
- matrice d'autorisation : 27 opérations ajoutées (10 `member`, 17
  `admin`), aucune publique ;
- tests en table de `internal/event` ;
- test de concurrence : 50 inscriptions simultanées sur 3 places → 3
  confirmées, 47 `409` ;
- tests de paiement par carte d'une inscription avec `hatest` : constatation
  par retour et par notification, double paiement, remboursement ;
- Playwright + axe : `events.spec.ts` sans violation `serious` ni
  `critical`.

## Scénarios manuels

### 1. Créer et publier (US1)

1. Admin › Événements : état vide, bouton « Créer un événement ».
2. Créer « Assemblée générale », samedi 12 décembre 14 h – 17 h, salle
   polyvalente, adhérents de la saison, gratuit. Attendu : brouillon,
   saison courante affichée.
3. Saisir une fin avant le début → erreur sur « Fin ». Saisir un début hors
   de toute saison → erreur sur « Début ».
4. En Claire : l'événement n'apparaît pas ; `/evenements/<id>` → « Page
   introuvable ».
5. Publier. En Claire : l'événement apparaît, « Gratuit ».

### 2. S'inscrire (US2, US3)

1. Créer et publier « Stage de printemps », toutes les fiches, 2 places,
   35 €, clôture au 1er avril.
2. Sur l'AG, en Claire : « Vous » inscriptible, Lucas « non adhérent de la
   saison ». Inscrire « Vous » → e-mail de confirmation en file
   (`outbound_emails`), badge « Inscrit ».
3. Annuler l'inscription depuis « Mes inscriptions » → place libérée,
   réinscription possible.
4. Admin : passer la capacité de l'AG à 1 avec une inscription, puis à 0 →
   refus (« au moins 1 »).

### 3. Payer (US4)

1. Sur le stage, en Claire : cocher « Vous » et Lucas → « Inscrire et payer
   70 € » ; redirection vers la première inscription, référence
   `EVT-2627-0001`.
2. « À régler » liste les deux inscriptions (et une adhésion à payer s'il y
   en a).
3. Choisir le chèque → « Paiement annoncé — chèque ».
4. Admin › inscription : « Constater le paiement » → « Payée », e-mail
   `registration-paid`.
5. En Claire : « Annuler mon inscription » absent ; message « contactez
   l'association ».
6. Carte (mode test) sur la seconde inscription : paiement HelloAsso, retour
   → « Payée », mention « Mode test » ; sans revenir sur Frigg, la
   notification suffit.
7. Une troisième fiche tente de s'inscrire → « Complet », aucune liste
   d'attente.

### 4. Administration des inscrits (US5)

1. Admin › stage : inscrits, adhérent oui/non, totaux dû / encaissé.
2. « Inscrire un membre » hors période avec « Déjà payé — espèces » →
   inscription « Payée ».
3. Passer le stage en « Adhérents de la saison » → Lucas signalé « Non
   éligible », non retiré.
4. Exporter le CSV : ouverture correcte dans un tableur (accents, `;`).
5. Admin › Inscriptions : rechercher « EVT-26 », puis « luc ».

### 5. Annuler, rétablir, supprimer (US6)

1. Modifier le lieu du stage → un e-mail `event-changed` par adresse (Claire
   en reçoit un seul nommant ses deux fiches).
2. Annuler le stage avec un message → dialogue indiquant les inscriptions
   payées ; un e-mail par adresse ; côté membre, badge « Annulé », plus rien
   dans « À régler ».
3. Rétablir → « Publié », inscriptions et paiements intacts.
4. Supprimer le stage → refus ; supprimer un brouillon → disparu.

### 6. Effets de bord

1. Supprimer la fiche de Lucas (dialogue : « 1 inscription payée à un
   événement à venir ») → inscriptions supprimées.
2. Modifier les dates de la saison pour exclure l'AG → `409` explicite.
3. Rembourser un paiement d'inscription dans l'espace HelloAsso (test) →
   paiement « Remboursé », inscription annulée, e-mail.
