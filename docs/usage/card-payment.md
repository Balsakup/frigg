# Paiement par carte

**Rôle requis** : Membre pour payer ; Administrateur pour activer la carte et
suivre les paiements

Lorsque l'association l'a activé, le moyen « Carte bancaire » permet de payer
une [adhésion](./memberships) ou une [inscription](./events) par carte, via
[HelloAsso](https://www.helloasso.com). L'activation côté serveur est décrite
dans [Paiement par carte (HelloAsso)](../installation/helloasso).

## Côté membre

1. Sur la page de paiement, choisissez « Carte bancaire ». Frigg vous
   prévient que le paiement se fait chez HelloAsso.
2. Vous êtes redirigé vers HelloAsso pour payer le montant exact, en une
   seule fois. HelloAsso propose une **contribution volontaire** à son
   fonctionnement : elle est facultative et ne revient pas à l'association.
3. De retour sur Frigg, l'adhésion ou l'inscription est **payée**.

Si le paiement échoue ou que vous l'abandonnez, rien ne change : vous pouvez
réessayer ou choisir un autre moyen.

Frigg ne voit jamais votre numéro de carte. Seuls votre adresse électronique
et le nom de votre fiche « Vous » sont transmis à HelloAsso pour préremplir le
formulaire.

## Côté administration

### Activer la carte

Dans *Configuration › Paramètres › Paiements*, la carte bancaire présente :

- l'état des identifiants HelloAsso (renseignés ou non dans la configuration
  du serveur) et le mode (test ou réel) ;
- l'**adresse de notification** à copier dans l'espace HelloAsso ;
- le bouton **Tester la connexion** ;
- l'interrupteur d'activation, disponible seulement après un test réussi.

En mode test, les pages de paiement et les adhésions payées portent la
mention « Mode test », et ces paiements ne comptent pas dans l'encaissé.

### Suivre les paiements

- Un paiement réussi est constaté **automatiquement**, même si le membre ne
  revient pas sur Frigg : HelloAsso prévient Frigg, qui vérifie le paiement
  auprès de lui. Il ne peut pas être annulé depuis Frigg.
- L'adhésion payée indique la date, le montant reçu, la référence HelloAsso
  et un lien vers l'espace HelloAsso. Une tentative en cours est visible, et
  **Vérifier auprès de HelloAsso** permet d'en connaître l'état.
- La liste des adhésions se filtre par moyen effectif, et la recherche accepte
  une référence HelloAsso.

### Anomalies et remboursements

- Un paiement reçu en double, d'un montant inattendu ou pour une adhésion
  annulée est **signalé** dans la liste, sur l'adhésion et sur le tableau de
  bord, jusqu'à ce qu'un administrateur indique l'avoir traité.
- Les **remboursements** se font dans l'espace HelloAsso. Frigg les constate
  ensuite : le paiement passe à « Remboursé » et l'adhésion est annulée.
- Le tableau de bord signale une connexion à HelloAsso en échec alors que la
  carte est activée.
