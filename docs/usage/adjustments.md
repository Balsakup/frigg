# Remises et dons

**Rôle requis** : Administrateur

Le montant payé ne correspond pas toujours au tarif : le bureau accorde une
**remise**, ou un membre donne plus que ce qui lui est demandé. Frigg suit ces
écarts sur les [adhésions](./memberships) et les
[inscriptions](./events). Il n'y a pas de code de réduction.

## Les montants

| Montant | Définition |
| --- | --- |
| **Montant théorique** | Calculé par Frigg : le tarif retenu (première adhésion ou renouvellement), ou le prix de l'événement à l'inscription |
| **Montant à payer** | Ce qui est réellement demandé au membre. Égal au montant théorique par défaut, il peut être ajusté |
| **Montant reçu** | Ce que l'association a effectivement reçu ; au moins égal au montant à payer |
| **Don** | L'excédent du montant reçu sur le montant à payer |

## Accorder une remise

Le champ **Montant à payer** est prérempli avec le montant théorique :

- à la validation, à la création et à la correction d'une adhésion non payée ;
- à la création d'une inscription par un administrateur, et ensuite tant
  qu'elle est confirmée et non payée.

Saisissez un autre montant, et un champ **Motif** apparaît (obligatoire, 200
caractères au plus), par exemple « réduction famille ». Un montant inférieur
est une remise, un montant supérieur une majoration. Tant que vous n'avez pas
modifié le montant à la main, il suit le tarif ou le type choisi ; une action
permet de revenir au montant théorique.

- Un montant à payer de **0 €** rend l'adhésion ou l'inscription gratuite.
- Le membre voit le montant à payer partout (*À régler*, e-mails, paiement par
  carte), mais **jamais le motif**.
- Une adhésion ou une inscription payée ne peut plus être ajustée : annulez
  d'abord la constatation du paiement.

## Enregistrer un don

Quand vous constatez un paiement en espèces, par chèque ou par virement, le
champ **Montant reçu** est prérempli avec le montant à payer.

- Saisissez le montant réellement reçu : l'excédent est enregistré comme un
  **don**, affiché sur l'adhésion ou l'inscription, et le membre est remercié
  dans l'e-mail de confirmation.
- Un montant reçu inférieur au montant à payer est refusé : ajustez d'abord
  le montant à payer.
- Pour un paiement par carte, le montant reçu est celui constaté par
  HelloAsso, hors contribution volontaire : aucun don n'est saisi.

## Dans les chiffres

Les chiffres de la saison et ceux de chaque événement distinguent le montant
théorique, les remises, les majorations, le montant à payer, l'encaissé (dont
les dons) et le restant à percevoir. L'export des inscrits d'un événement
inclut ces montants et le motif.
