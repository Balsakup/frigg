# Contrat des écrans : paiement par carte

Aucune nouvelle route : les écrans de 006 évoluent
([006 › ui-routes.md](../../006-memberships/contracts/ui-routes.md)).

| Route | Fichier | Évolution |
| --- | --- | --- |
| `/adhesions/:id` | `routes/app/member/membership-detail.tsx` | Carte bancaire en premier ; `CardPaymentPanel` ; traitement des paramètres de retour `carte` et `tentative` |
| `/admin/parametres/paiements` | `routes/app/admin/settings-payments.tsx` | Section « Carte bancaire (HelloAsso) » |
| `/admin/adhesions/:id` | `routes/app/admin/membership-detail.tsx` | Bloc « Paiements en ligne », signalements, vérification |
| `/admin/adhesions` | `routes/app/admin/memberships.tsx` | Filtres « Moyen » et « Signalements », badge |
| `/admin` | `routes/app/admin/home.tsx` | Encarts signalements et connexion HelloAsso |

## `/adhesions/:id` (page de paiement)

- Paramètres d'adresse lus au chargement : `carte` ∈ `retour | erreur |
  annule`, `tentative` (entier), `checkoutIntentId` (ajouté par HelloAsso).
  Ils déclenchent `confirm-my-card-payment`, puis sont retirés de l'adresse
  (`replace`), pour qu'un rechargement ne rejoue rien.
- États annoncés dans une région `role="status"` :
  - `succeeded` : alerte `ok` « Paiement reçu, merci. L'adhésion de Léo est
    réglée. » ;
  - `failed` : alerte `danger` « Le paiement n'a pas abouti. Aucune somme n'a
    été débitée. » + « Réessayer » et choix d'un autre moyen ;
  - `abandoned` : retour silencieux sur la page de paiement ;
  - `pending` : « Confirmation du paiement en cours… », nouvel appel toutes
    les 3 s pendant 30 s, puis « Votre paiement sera confirmé dans quelques
    minutes ; vous recevrez un e-mail. ».
- `CardPaymentPanel` (`web/app/memberships/CardPaymentPanel.tsx`) : montant,
  texte « Vous allez être redirigé vers HelloAsso, la plateforme de paiement
  des associations. HelloAsso vous proposera une contribution volontaire
  facultative à son fonctionnement. », bouton principal « Payer 40,00 € par
  carte » (`aria-describedby` vers ce texte), état de chargement, erreur
  `503` affichée sans quitter la page.
- Mode test : bandeau `warn` « Mode test — aucun débit réel » au-dessus des
  moyens.

## `/admin/parametres/paiements`

Carte « Carte bancaire (HelloAsso) », placée avant les espèces :

- non configuré : texte expliquant que l'hébergeur renseigne les variables
  `FRIGG_HELLOASSO_*`, lien vers la documentation d'installation ;
  interrupteur désactivé (`aria-disabled` + explication liée) ;
- configuré : badge `Mode test` (`warn`) ou `Production` (`ok`), slug,
  adresse de notification en lecture seule avec bouton « Copier » (annonce
  « Adresse copiée »), bouton « Tester la connexion » (état de chargement,
  résultat en alerte `ok` ou `danger` avec la cause traduite), interrupteur
  d'activation (désactivé tant qu'aucun test n'a réussi pour cet
  environnement) ;
- erreurs `422` de huma affichées sur l'interrupteur.

## `/admin/adhesions/:id`

- Paiement par carte constaté : « Payée — carte bancaire », date et heure,
  montant reçu, « Commande HelloAsso n° 98765 » avec lien externe
  (`target="_blank"`, `rel="noopener noreferrer"`, mention « nouvel onglet »
  pour les lecteurs d'écran).
- Bloc « Paiements en ligne » : tableau des tentatives (date, état en badge,
  montant, référence, cause d'échec traduite).
- « Vérifier auprès de HelloAsso » si `actions.verifyCardPayment`.
- Signalements : alerte `warn` par signalement ouvert, bouton « Marquer comme
  traité » → dialogue de confirmation.
- Dialogue d'annulation ou de correction : avertissement si
  `cardPaymentPending` ou paiement par carte reçu (« Le remboursement se fait
  dans l'espace HelloAsso de l'association. »).

## Accessibilité

Mêmes exigences que 006 : clavier seul, focus visible, 320 px, `axe` sans
violation `serious` ou `critical` sur ces écrans, y compris l'état de retour
de HelloAsso.
