# Research: Remises et dons sur les paiements

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

Aucune inconnue technique : pas de dépendance nouvelle, pas d'intégration
nouvelle. Les décisions ci-dessous fixent la manière d'étendre le modèle de
paiement existant (006, 008, 012).

## R1. Stockage du montant théorique

**Decision**: deux colonnes ajoutées à `memberships` et à
`event_registrations` :

- `base_amount_cents` : le montant théorique, **renseigné seulement quand un
  ajustement existe** ; `NULL` = pas d'ajustement, le montant théorique est
  alors `amount_cents` ;
- `adjustment_reason` : le motif, renseigné exactement quand
  `base_amount_cents` l'est.

`amount_cents` garde son rôle actuel et devient le **montant à payer** :
c'est la seule colonne lue par le paiement par carte, « À régler », les
e-mails, les références de paiement et la reprise des tentatives HelloAsso.

**Rationale**:

- Aucune reprise de données : les lignes existantes n'ont pas d'ajustement,
  leurs colonnes restent `NULL`.
- Tout le code qui lit `amount_cents` (carte, `FindResumableOnlinePayment`,
  `SettleOnline`, `PaymentAfter`, « À régler », e-mails) applique le montant
  ajusté sans modification. Une tentative de carte commencée avant un
  ajustement n'est plus reprise, puisque la recherche filtre sur le montant :
  c'est la règle 008 attendue par la spec.
- L'ajustement et la remise se déduisent (`amount_cents − base_amount_cents`)
  et ne sont pas stockés.

**Probe**: `ALTER TABLE … ADD COLUMN` avec un `CHECK` qui lit une autre
colonne a été vérifié sur SQLite 3.51 : la contrainte est contrôlée à
l'insertion et à la mise à jour, et les lignes existantes sont acceptées.
`00006` utilise déjà `ADD COLUMN … CHECK`, ce que sqlc analyse.

**Alternatives considered**:

- *Toujours stocker le montant théorique (NOT NULL), avec reprise* : il
  faudrait reconstruire la table pour garantir `NOT NULL` (SQLite n'ajoute
  pas de colonne `NOT NULL` sans valeur par défaut constante), soit une
  migration lourde comme `00012`, pour un bénéfice nul.
- *Recalculer le théorique depuis le tarif* : faux dès qu'un tarif est
  modifié après validation (006 FR-004 fige le montant).
- *Table d'historique des ajustements* : la spec ne demande que l'ajustement
  en vigueur ; l'historique est dans le journal (R7).

## R2. Montant reçu des paiements manuels

**Decision**: `paid_amount_cents`, déjà présent sur les deux tables et
réservé jusqu'ici à la carte, est rempli **pour tout paiement constaté**.
La migration le renseigne avec `amount_cents` pour les paiements manuels
déjà constatés (FR-017). L'annulation d'une constatation le remet à `NULL`.

Le **don** d'un paiement manuel vaut `paid_amount_cents − amount_cents` ;
il est nul pour la carte (FR-014). Il n'est pas stocké.

**Rationale**: une seule colonne « montant reçu », quel que soit le moyen.
Les chiffres (`ComputeStats`, `addTotals`) et les e-mails lisent déjà
`paid_amount_cents` quand elle est renseignée. La reprise rend la règle
simple : une ligne payée a toujours un montant reçu.

**Alternatives considered**:

- *Colonne `donation_cents`* : redondante avec le montant reçu, et il
  faudrait la garder cohérente.
- *Pas de reprise, avec un repli `COALESCE(paid_amount_cents,
  amount_cents)` partout* : le repli se disperse dans les requêtes, le
  domaine et les e-mails, et on risque d'en oublier un.

## R3. Règles dans le domaine pur

**Decision**: nouveau fichier `internal/membership/amount.go`, sans accès à
la base, partagé par les adhésions et les inscriptions (l'adaptateur
`event.PaymentSnapshot` de 012 › R3 le permet déjà) :

- `Adjust(base int64, requested *int64, reason string) (Adjustment, error)` :
  renvoie le montant à payer, le montant théorique à conserver (ou rien) et
  le motif nettoyé. Requête absente = montant théorique. Montant différent
  sans motif : `ErrAdjustmentReasonRequired`. Motif de plus de 200
  caractères : `ErrAdjustmentReasonTooLong`. Montant égal au théorique : le
  motif est ignoré.
- `CheckReceived(due, received int64) error` : `ErrReceivedBelowDue` si le
  montant reçu est inférieur au montant à payer.
- `Donation(method Method, due, received int64) int64` : `received − due`
  pour un moyen manuel, 0 pour la carte.
- `AmountTotals` et `(*AmountTotals).Add(AmountRow)` : cumul du théorique,
  des remises, des majorations, du montant à payer, de l'encaissé (par moyen),
  des dons, du restant et du mode test. `ComputeStats` (adhésions) et
  `addTotals` (événements) l'utilisent tous les deux.

Le motif suit le nettoyage de `CleanReason` (espaces réduits), avec une
borne de 200 caractères comme le commentaire de paiement.

**Rationale**: constitution VI (règles hors des handlers, testées en
table) et 012 › R3 (aucune règle de paiement dupliquée entre adhésions et
inscriptions).

## R4. Format des montants dans l'API

**Decision**: les montants saisis arrivent en **texte en euros** (« 70 »,
« 15,50 ») et sont analysés par `season.ParseAmount`, comme les tarifs (004
› R3) et le prix d'un événement (012). Champs :

- `amount` (montant à payer) et `adjustmentReason` dans les corps de
  validation, de correction et de création d'une adhésion, dans
  l'inscription par un administrateur, et dans le nouveau corps
  d'ajustement d'une inscription ;
- `receivedAmount` dans `RecordPaymentBody`, utilisé partout où un paiement
  manuel est constaté.

Champ absent ou vide : montant théorique (pour `amount`) ou montant à payer
(pour `receivedAmount`). Un ancien client qui ne les envoie pas garde donc
le comportement actuel.

Les réponses donnent des centimes (`amountCents`, `baseAmountCents`,
`paidAmountCents`, `donationCents`), comme partout ailleurs.

**Alternatives considered**: des centimes entiers en entrée. Écarté : le
frontend devrait convertir lui-même une saisie en euros, et la constitution
VI veut que le serveur fasse autorité, comme pour les tarifs.

## R5. Correction d'une adhésion déjà ajustée

**Decision**: `PUT /admin/memberships/{id}` reçoit toujours le montant à
payer et le motif voulus. Le montant théorique est recalculé depuis le tarif
et le type envoyés. Si le montant à payer reçu égale ce théorique,
l'ajustement disparaît (FR-005).

Côté formulaire, le champ est « suivi » (recalculé quand le tarif ou le type
change) tant qu'il est égal au montant théorique affiché. Il ne l'est plus
dès que l'administrateur le modifie, ou à l'ouverture d'une correction sur
une adhésion déjà ajustée. Le bouton « Revenir au montant théorique » rétablit
le suivi.

**Rationale**: le serveur reste sans état de formulaire. La règle « suivi
tant que non modifié » ne concerne que l'interface.

## R6. Ajustement d'une inscription

**Decision**: nouvelle opération admin `PUT
/admin/registrations/{id}/amount` (`adjust-registration-amount`), corps
`{ amount, adjustmentReason }`. Elle est autorisée sur une inscription
confirmée, non payée, d'un événement non annulé : `CanUpdate` sur
l'instantané de `event.PaymentSnapshot`, avec `SeasonClosed` à faux. Le
montant théorique est le prix figé à l'inscription :
`COALESCE(base_amount_cents, amount_cents)`.

Passer à 0 € met `payment_status` à `NULL` et efface l'annonce ; la
référence est conservée. Passer d'un montant nul à un montant positif
attribue une référence `EVT-` s'il n'y en a pas, puis remet l'état à « à
payer ». La logique de `membership.PaymentAfter` est réutilisée, avec un
pendant `registrationPaymentColumns` qui suit le modèle de `paymentColumns`.

**Rationale**: une inscription n'a pas d'action de correction aujourd'hui ;
une opération dédiée et étroite est plus simple que d'ouvrir une
modification générale.

## R7. Journalisation

**Decision**: les actions existantes (`membership.validate`,
`membership.update`, `membership.create`, `registration.create`) ajoutent
`base_amount_cents` et `adjusted` (booléen). La nouvelle action
`registration.adjust` journalise `base_amount_cents`, `old_amount_cents` et
`amount_cents`. Les constatations ajoutent `received_cents` et
`donation_cents`. Le **texte du motif n'est jamais journalisé**, comme les
motifs de refus ou d'annulation et les commentaires (006 › R12). La spec
FR-009 a été alignée sur cette règle.

## R8. E-mails

**Decision**: `mail.Data` reçoit `Donation` (montant formaté, vide sans
don). Les modèles `membership-paid` et `registration-paid` (txt et html)
affichent le montant reçu (déjà le cas via `paid_amount_cents`), puis, s'il y
a un don, la ligne « Dont {{.Donation}} de don : merci pour votre soutien ! ».
Les autres e-mails continuent de lire `amount_cents`, qui est le montant à
payer.

## R9. Visibilité pour le membre

**Decision**: le type `Membership` partagé (espace membre et
administration) ne reçoit **ni** le montant théorique, **ni** l'ajustement,
**ni** le motif. Ces champs (`baseAmountCents`, `adjustmentCents`,
`adjustmentReason`) sont ajoutés seulement à `AdminMembership`,
`AdminMembershipRow`, `AdminRegistration` et `AdminRegistrationRow`.
L'ajustement est calculé par le serveur (`membership.Adjustment`) :
l'interface affiche la remise ou la majoration sans calculer de montant. Le type `Payment` partagé
reçoit `donationCents`, que le membre voit (FR-013).

## R10. Tests

- **Domaine** : tests en table de `Adjust`, `CheckReceived`, `Donation`,
  `AmountTotals` (remise, majoration, gratuit par ajustement, carte avec
  écart, mode test, adhésion annulée).
- **Migration** : base `00012` peuplée (adhésions payées en espèces et par
  carte, inscription payée, inscription due). Après `00013`, chaque paiement
  manuel a un `paid_amount_cents` égal à son `amount_cents`, la carte est
  inchangée, et les `CHECK` refusent un motif sans montant théorique ou un
  montant théorique égal au montant à payer. Le `Down` est testé.
- **API (httptest)** : validation avec remise, sans motif (422 sur
  `body.adjustmentReason`), à 0 € (pas de référence) ; correction qui
  retire l'ajustement ; correction d'une adhésion payée (409) ; constatation
  avec don, avec un montant inférieur (422 sur `body.receivedAmount`) ;
  annulation de la constatation (montant reçu effacé) ; ajustement d'une
  inscription (passage à 0 € puis retour à un montant positif avec la même
  référence) ; paiement par carte au montant ajusté (`hatest`) ; chiffres de
  la saison et de l'événement ; colonnes du CSV ; contenu des e-mails.
- **Isolation** : la réponse brute d'une adhésion ou d'une inscription dans
  l'espace membre ne contient ni `baseAmountCents`, ni `adjustmentCents`, ni `adjustmentReason`.
- **Autorisation** : la matrice automatique couvre
  `adjust-registration-amount` (membre refusé).
- **Frontend** : Vitest sur le suivi du montant (recalcul, puis arrêt après
  saisie, puis retour au théorique) ; Playwright et axe sur la validation
  avec remise et sur la constatation avec don.
