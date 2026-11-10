# Contrat d'interface : Remises et dons sur les paiements

Aucune route nouvelle. Les écrans existants reçoivent deux composants
partagés. Le frontend n'analyse ni ne compare les montants (constitution
VI) : il affiche les valeurs de l'API et transmet le texte saisi ; le serveur
décide et renvoie les erreurs 422 sur les champs.

## Composants partagés

### `AmountToPayField` (`web/app/payments/AmountToPayField.tsx`, nouveau)

Champ « Montant à payer » en euros, avec le montant théorique à côté et le
motif.

- Props : `theoreticalCents` (montant de l'API : `fee.amounts[kind]` pour
  une adhésion, prix figé pour une inscription), valeur initiale
  (`amountCents` et `adjustmentReason` actuels pour une correction),
  erreurs `amount` et `adjustmentReason`.
- **Suivi** : à l'ouverture, le champ est suivi s'il n'y a pas d'ajustement.
  Tant qu'il est suivi, il affiche le montant théorique formaté (« 75,00 »)
  et se met à jour quand `theoreticalCents` change. Il cesse d'être suivi à
  la première saisie, et l'est de nouveau avec le bouton « Revenir au
  montant théorique » (lien-bouton, visible seulement quand le champ n'est
  pas suivi).
- **Motif** : la zone de texte « Motif de l'ajustement » (200 caractères,
  compteur) apparaît quand le champ n'est pas suivi. Elle est reliée au champ
  par `aria-describedby` et son apparition est annoncée (`aria-live`). Le
  serveur décide si le motif est requis.
- Texte d'aide : « Montant théorique : 75,00 € ». Quand le champ n'est pas
  suivi, il le reste ; l'écart n'est pas calculé côté client.
- Envoi : quand le champ est suivi, `amount` n'est pas transmis (le serveur
  applique le théorique) ; sinon, `amount` et `adjustmentReason` le sont.
- Logique de suivi isolée dans un hook `useTrackedAmount`, testé avec Vitest.

### `RecordPaymentDialog` (`web/app/payments/AdminPayments.tsx`, modifié)

- Nouvelle prop `amountCents` (montant à payer) ; nouveau champ « Montant
  reçu » prérempli avec ce montant formaté, avant la date de réception.
- Aide : « Un montant supérieur au montant à payer est enregistré comme un
  don. »
- Le champ est transmis s'il a été modifié (`receivedAmount`) ; l'erreur 422
  `receivedAmount` s'affiche sur le champ.
- Le même champ est ajouté aux formulaires « déjà payé » de
  `membership-new.tsx` et de `RegisterMemberDialog.tsx`.

## Écrans modifiés

| Écran | Changement |
| --- | --- |
| `admin/membership-detail.tsx` › `DecisionDialog` (valider, corriger) | `AmountSummary` remplacé par `AmountToPayField` (théorique = `fee.amounts[kind]`). En correction, valeur initiale = `amountCents` et `adjustmentReason`, non suivie s'il y a un ajustement. |
| `admin/membership-detail.tsx` › vue | Bloc « Montant » : montant à payer ; s'il y a un ajustement, la ligne « Théorique 75,00 € · Remise 5,00 € » (ou « Majoration »), à partir de `baseAmountCents` et `adjustmentCents` fournis par l'API (aucun calcul côté client), et le motif. Paiement : « Reçu 50,00 € dont 5,00 € de don ». |
| `admin/membership-new.tsx` | `AmountToPayField` sous le choix du tarif et du type ; « Montant reçu » dans le bloc « déjà payé ». |
| `admin/memberships.tsx` › chiffres | Tuiles : « Montant théorique », « Remises » (et « Majorations » si non nul), « Total à payer », « Encaissé » (détail par moyen inchangé, puis « dont dons X € »), « Reste à percevoir ». |
| `admin/registration-detail.tsx` | Action « Ajuster le montant » (si `actions.adjustAmount`), dialogue avec `AmountToPayField` (théorique = `baseAmountCents`) ; si `cardPaymentPending`, le dialogue affiche l'avertissement de paiement par carte en cours, comme la correction d'une adhésion. Montants (`adjustmentCents`) et don affichés comme pour une adhésion. |
| `admin/event-detail.tsx` | Totaux : théorique, remises, majorations, total à payer, encaissé dont dons, reste à percevoir. `RecordPaymentDialog` reçoit `amountCents`. `RegisterMemberDialog` : `AmountToPayField` (théorique = prix de l'événement, 0 € s'il est gratuit) pour tout événement ; le bloc « déjà payé », avec « Montant reçu », est proposé pour tout événement, et le serveur l'ignore si l'inscription est gratuite après ajustement (comportement actuel de `register-member`). Le frontend ne compare pas les montants. |
| `member/membership-detail.tsx`, `member/registration-detail.tsx` | Paiement payé : « Reçu X € » et, s'il y a un don, « dont Y € de don, merci ! ». Aucun montant théorique ni motif. |

## Accessibilité (RGAA)

- Champs étiquetés, aide et erreurs reliées par `aria-describedby`.
- Le bouton « Revenir au montant théorique » est un vrai `<button>`, et le
  focus revient sur le champ après son utilisation.
- L'apparition du motif est annoncée par une région `aria-live="polite"`.
- Playwright et axe : aucune violation `serious` ou `critical` dans le
  dialogue de validation avec remise, ni dans la constatation avec don.
