# Quickstart : valider les remises et les dons

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[ui-routes.md](./contracts/ui-routes.md). Modèle : [data-model.md](./data-model.md).

Prérequis : ceux de [006 › quickstart.md](../006-memberships/quickstart.md)
(`create-admin`, `go tool task dev`) ; une saison en cours avec le tarif
« Adulte loisir » à 75 € (renouvellement 45 €) ; un compte membre « Paul »
avec une demande de première adhésion en attente ; virement et espèces
activés ; pour la carte, HelloAsso en mode test comme dans
[008 › quickstart.md](../008-helloasso-card-payment/quickstart.md) ; un
événement payant « Stage » à 35 €.

## Contrôles automatisés (constitution)

```sh
go tool task check
```

Attendu en plus de 013 :

- `00013_payment_adjustments.sql` s'applique sur une base vierge et sur une
  base `00012` peuplée. Les paiements manuels ont `paid_amount_cents =
  amount_cents`, la carte est inchangée, et le `Down` restaure le schéma
  `00012` ;
- `sqlc generate`, `frigg openapi` et `bun run gen:api` sans diff ;
- matrice d'autorisation : 1 opération ajoutée (`adjust-registration-amount`,
  `admin`) ;
- tests en table de `internal/membership/amount.go` ;
- Playwright + axe sur la validation avec remise et la constatation avec don :
  aucune violation `serious` ni `critical`.

## Scénarios manuels

### 1. Remise à la validation (US1)

1. Admin › Adhésions › demande de Paul › Valider. Attendu : « Montant à
   payer » prérempli à 75,00, « Montant théorique : 75,00 € ».
2. Passer le type à « Renouvellement » : le champ passe à 45,00. Revenir à
   « Première adhésion » : 75,00.
3. Saisir 70 : la zone « Motif de l'ajustement » apparaît. Changer de type :
   le champ garde 70, l'aide affiche le nouveau théorique. Revenir à
   « Première adhésion ».
4. Valider sans motif : erreur sur « Motif ». Saisir « réduction de 5 € »,
   valider.
5. Attendu : la vue affiche 70,00 €, « Théorique 75,00 € · Remise 5,00 € »
   et le motif. L'e-mail de validation en file (`outbound_emails`) indique
   70,00 €.
6. En Paul : « À régler » demande 70,00 € ; aucun motif ni montant théorique
   visible, ni dans l'écran ni dans la réponse brute de
   `GET /api/me/memberships/<id>`.
7. Payer par carte (sandbox) : HelloAsso demande 70,00 € ; après le retour,
   l'adhésion est payée sans signalement.

### 2. Paiement avec don (US2)

1. Créer et valider une adhésion de renouvellement à 45 € sans ajustement.
2. Constater le paiement : « Montant reçu » prérempli à 45,00. Saisir 40 :
   erreur « … Ajustez d'abord le montant à payer. ».
3. Saisir 80, virement, date du jour. Attendu : « Reçu 80,00 € dont
   35,00 € de don » ; l'e-mail de paiement indique 80,00 € et le don de
   35,00 €.
4. Annuler la constatation, puis la refaire avec 45 : plus de don.

### 3. Correction après validation (US3)

1. Sur une adhésion validée non payée, ouvrir « Corriger ». Le montant et le
   motif actuels sont préremplis. Changer de tarif : le montant ajusté est
   conservé.
2. « Revenir au montant théorique », enregistrer : l'ajustement disparaît.
3. Saisir 0 avec un motif : l'adhésion devient gratuite, la référence est
   conservée, rien n'est à régler. Revenir à 75 : paiement attendu, même
   référence.
4. Sur une adhésion payée : « Corriger » n'est pas proposé ; l'appel direct
   renvoie `409`.

### 4. Inscriptions (US4)

1. Inscrire un membre au « Stage » depuis l'administration avec un montant de
   20 et le motif « bénévole encadrant ». Attendu : à payer 20,00 €.
2. Sur une autre inscription, « Ajuster le montant » à 0 : inscription
   gratuite. Revenir à 35 : à payer, même référence `EVT-`.
3. Constater 40 € en espèces sur une inscription à 35 € : don de 5 €.
4. Exporter les inscrits : colonnes « Montant théorique », « Montant à
   payer », « Motif d'ajustement », « Montant reçu », « Don » remplies.
5. Modifier le prix de l'événement : les montants des inscriptions existantes
   ne changent pas.

### 5. Chiffres (US5)

1. Admin › Adhésions › Chiffres de la saison. Avec les scénarios ci-dessus,
   vérifier : théorique − remises + majorations = total à payer ; encaissé =
   somme des montants reçus hors mode test, dont dons = 35 € ; reste à
   percevoir = montants à payer non payés.
2. Page du « Stage » : mêmes totaux pour les inscriptions confirmées.

## Correction des paiements existants

Après le déploiement, pour chaque paiement dont le commentaire signale un
écart :

- **remise** : annuler la constatation, corriger le montant à payer avec le
  motif, puis constater de nouveau ;
- **don** : annuler la constatation, puis la refaire avec le montant reçu
  réel.
