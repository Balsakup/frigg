# Adhésions

**Rôle requis** : Membre pour demander et payer une adhésion ; Administrateur
pour les valider et constater les paiements

Une **adhésion** porte sur une fiche de membre, une [saison](./seasons) et un
[tarif](./fees) de cette saison. Un membre a au plus une adhésion en attente
ou validée par saison.

## États

| État | Signification |
| --- | --- |
| **En attente de validation** | Demande déposée par un membre, à traiter par un administrateur |
| **Validée** | L'adhésion est acquise ; son paiement est suivi à part |
| **Refusée** | Refusée par un administrateur, avec un motif |
| **Annulée** | Demande retirée par le membre, ou adhésion annulée par un administrateur avec un motif |

Une adhésion refusée ou annulée n'empêche pas une nouvelle demande.

## Première adhésion ou renouvellement

Chaque adhésion a un **type** : « Première adhésion » ou « Renouvellement ».
Si le tarif a un montant de renouvellement, c'est lui qui s'applique à un
renouvellement.

Beaucoup d'associations existaient avant d'installer Frigg : son historique
ne suffit donc pas à savoir qui renouvelle. C'est pourquoi le membre
**déclare** le type dans sa demande, et l'administrateur **retient** le type
à la validation. Pour l'aider, Frigg affiche « Adhérent la saison précédente
dans Frigg : oui, non ou inconnu ». Le type retenu alimente les chiffres de
la saison (premières adhésions et renouvellements).

## Côté membre

Dans *Adhésions* :

1. **Demander une adhésion** : choisissez la saison (ouverte et dotée d'au
   moins un tarif), la fiche, le tarif et le type. Le montant s'affiche avant
   l'envoi.
2. Tant qu'elle est en attente, vous pouvez **annuler** votre demande.
3. Une fois l'adhésion validée, vous recevez un e-mail avec un lien vers la
   **page de paiement** : choisissez un moyen parmi ceux proposés par
   l'association (espèces, chèque, virement, carte bancaire). Ses
   instructions s'affichent avec le montant et la **référence de paiement**
   à rappeler. Confirmer votre choix fait passer le paiement à « Paiement
   annoncé ». Vous pouvez changer de moyen tant que le paiement n'est pas
   constaté.

*À régler* rassemble les adhésions et les inscriptions qui restent à payer.
Une fiche portant une demande en attente ou une adhésion validée ne peut pas
être supprimée.

## Côté administration

La rubrique *Adhésions* liste les adhésions d'une saison (par défaut, la
saison courante), les demandes en attente en premier.

- **Recherche** par prénom, nom ou référence de paiement. **Filtres** par
  état, état du paiement, moyen effectif et [champs personnalisés](./custom-fields).
- **Valider** une demande en retenant le type et le tarif ; le montant à
  payer peut être ajusté (voir [Remises et dons](./adjustments)).
- **Refuser** une demande, avec un motif obligatoire.
- **Créer** directement une adhésion validée pour n'importe quelle fiche, y
  compris sans compte (bulletin papier), éventuellement déjà payée.
- **Corriger** le tarif, le type ou le montant à payer d'une adhésion validée
  non payée.
- **Annuler** une adhésion validée, avec un motif.

Le tableau de bord signale les demandes en attente.

Les adhésions d'une saison close sont en lecture seule, sauf pour traiter les
demandes en attente et constater les paiements.

## Paiement

Frigg n'encaisse pas d'argent, sauf par carte via HelloAsso : il présente au
membre les informations du moyen choisi et enregistre la constatation du
paiement par un administrateur.

| État du paiement | Signification |
| --- | --- |
| **Gratuite** | Montant à payer de 0 €, aucun paiement attendu |
| **À payer** | Adhésion validée, aucun moyen choisi |
| **Paiement annoncé** | Le membre a choisi un moyen |
| **Payée** | Paiement constaté |

- À la validation, chaque adhésion non gratuite reçoit une **référence de
  paiement** unique.
- Un administrateur **marque l'adhésion payée** en indiquant le moyen
  effectif, la date de réception, le montant reçu (un excédent est compté
  comme un [don](./adjustments)) et un commentaire facultatif. Il peut annuler
  cette constatation. Un membre ne peut jamais marquer lui-même une adhésion
  payée.
- Un paiement par carte est constaté automatiquement (voir
  [Paiement par carte](./card-payment)).

Les moyens proposés se règlent dans *Configuration › Paramètres ›
Paiements* :

| Moyen | Informations |
| --- | --- |
| Espèces | Instructions (obligatoires) |
| Chèque | Ordre (obligatoire), adresse d'envoi et instructions |
| Virement | Titulaire et IBAN (obligatoires, IBAN vérifié), BIC et instructions |
| Carte bancaire | Voir [Paiement par carte](./card-payment) |

Les informations d'un moyen désactivé sont conservées mais ne sont plus
montrées aux membres.

## Chiffres de la saison

Pour la saison affichée, la rubrique présente :

- le nombre d'adhésions validées, dont premières adhésions et
  renouvellements, et le nombre de demandes en attente ;
- la répartition par tarif ;
- les montants : théorique, remises, majorations, à payer, encaissé (par
  moyen, dont dons) et restant à percevoir.

## E-mails

Le compte qui gère la fiche reçoit un e-mail à chaque validation (avec le
lien vers la page de paiement), refus, annulation et constatation de
paiement. Aucun e-mail n'est envoyé pour une fiche sans compte.
