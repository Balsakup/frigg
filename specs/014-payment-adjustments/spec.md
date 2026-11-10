# Feature Specification: Remises et dons sur les paiements

**Feature Branch**: `014-payment-adjustments`

**Created**: 2026-11-10

**Status**: Draft

**Input**: User description: "Ajustement des montants payés : remises et dons,
pour les adhésions et les inscriptions aux événements. Un administrateur peut
saisir le montant réellement à payer (prérempli par le montant théorique, motif
obligatoire s'il diffère) et le montant réellement reçu (prérempli par le
montant à payer, l'excédent étant un don). Pas de codes de réduction."

## Contexte

Dans une association réelle, le montant payé ne correspond pas toujours au
tarif :

- une **remise** est accordée à un membre avant qu'il paie (par exemple 70 €
  au lieu de 75 €, « réduction de 5 € ») ;
- un membre **donne plus** que ce qui lui est demandé (50 € au lieu de 45 €) ;
- un paiement unique **dépasse** le montant dû (un virement de 80 € pour une
  adhésion à 45 €) : l'excédent est compté comme un don.

Aujourd'hui, ces écarts ne sont notés que dans le commentaire libre du
paiement (006). Les chiffres de la saison et des événements comptent le
montant du tarif, et ne reflètent donc pas ce que l'association a réellement
demandé ni reçu. Les codes de réduction ont été écartés : trop lourds à gérer
pour une petite association, ils ne couvrent pas les dons.

## Vocabulaire

- **Montant théorique** : ce que Frigg calcule sans intervention.
  - Adhésion : le montant du tarif retenu, de première adhésion ou de
    renouvellement selon le type retenu (004, 006).
  - Inscription : le prix de l'événement au moment de l'inscription (012).
- **Montant à payer** : ce qui est réellement demandé au membre. Égal au
  montant théorique par défaut, il peut être **ajusté** par un administrateur.
  Il remplace, dans tout Frigg, ce que 006 appelle « montant appliqué » et 012
  « montant dû ».
- **Ajustement** : écart entre le montant à payer et le montant théorique.
  Négatif, c'est une **remise** ; positif, une **majoration**.
- **Motif de l'ajustement** : texte court, obligatoire dès qu'il y a un
  ajustement, qui explique la décision (« réduction famille », « bénévole
  encadrant »…).
- **Montant reçu** : ce que l'association a effectivement reçu pour une
  adhésion ou une inscription payée. Il est au moins égal au montant à payer.
- **Don** : excédent du montant reçu sur le montant à payer.

Les autres termes (adhésion, inscription, tarif, type retenu, état du paiement,
référence de paiement, moyen de paiement, carte bancaire, HelloAsso, À régler,
chiffres de la saison) sont ceux des spécifications 004 à 012.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Accorder une remise à la validation d'une adhésion (administration) (Priority: P1)

Une administratrice valide la demande de première adhésion de Paul au tarif
« Adulte loisir » (75 €). Le champ « Montant à payer » est déjà rempli avec
75 €. Le bureau a accordé à Paul une réduction de 5 € : elle saisit 70 €, et
un champ « Motif » apparaît. Elle écrit « réduction de 5 € accordée par le
bureau » et valide. Paul voit 70 € à régler dans son espace et paie 70 €,
par virement ou par carte.

**Why this priority**: c'est le cas le plus courant, et il conditionne tout le
reste : sans montant à payer ajustable, les paiements par carte et les
relances demandent un montant faux.

**Independent Test**: valider une demande en ajustant le montant, puis
vérifier le montant affiché au membre, dans l'e-mail de validation et sur la
page de paiement par carte, ainsi que le motif visible en administration.

**Acceptance Scenarios**:

1. **Given** une demande d'adhésion en attente, **When** un administrateur
   ouvre le formulaire de validation, **Then** le champ « Montant à payer »
   est prérempli avec le montant théorique du tarif et du type proposés.
2. **Given** le formulaire de validation dont le montant n'a pas été modifié
   à la main, **When** l'administrateur change de tarif ou de type, **Then**
   le montant à payer est recalculé avec le nouveau montant théorique.
3. **Given** le formulaire de validation dont le montant a été modifié à la
   main, **When** l'administrateur change de tarif ou de type, **Then** le
   montant saisi est conservé, le nouveau montant théorique est affiché à
   côté, et une action permet de revenir au montant théorique.
4. **Given** un montant à payer différent du montant théorique, **When**
   l'administrateur valide sans motif, **Then** la validation est refusée
   avec un message sur le champ « Motif ».
5. **Given** un montant à payer égal au montant théorique, **When**
   l'administrateur valide, **Then** aucun motif n'est demandé et aucun motif
   n'est conservé.
6. **Given** une adhésion validée avec une remise, **When** le membre consulte
   « À régler », l'e-mail de validation ou le paiement par carte, **Then**
   seul le montant à payer est demandé.
7. **Given** une adhésion validée avec un ajustement, **When** un
   administrateur l'ouvre, **Then** il voit le montant théorique, le montant à
   payer, l'ajustement (remise ou majoration) et son motif.
8. **Given** un montant à payer saisi à 0 €, **When** l'administrateur valide
   avec un motif, **Then** l'adhésion est gratuite : pas de référence de
   paiement ni de paiement attendu, comme une adhésion à un tarif gratuit.
9. **Given** un membre non administrateur, **When** il tente de fixer ou
   d'ajuster un montant à payer, **Then** l'accès est refusé.

---

### User Story 2 - Constater un paiement supérieur au montant à payer (administration) (Priority: P1)

Un administrateur constate le virement de Léa pour son renouvellement à 45 €.
Le champ « Montant reçu » est prérempli avec 45 €. Léa a viré 50 € : il
saisit 50 €. L'adhésion est payée, et Frigg affiche « dont 5 € de don ». Le
mois suivant, il reçoit un virement unique de 80 € pour une autre adhésion à
45 € : il saisit 80 €, et les 35 € d'excédent sont comptés comme un don.

**Why this priority**: sans ce champ, les sommes reçues en plus restent
invisibles dans les chiffres et le rapprochement avec le relevé bancaire est
impossible.

**Independent Test**: constater un paiement en espèces, par chèque ou par
virement avec un montant reçu supérieur au montant à payer, puis vérifier
l'affichage du don sur l'adhésion et dans les chiffres.

**Acceptance Scenarios**:

1. **Given** une adhésion validée à payer, **When** un administrateur ouvre la
   constatation du paiement, **Then** le champ « Montant reçu » est prérempli
   avec le montant à payer.
2. **Given** un montant reçu supérieur au montant à payer, **When**
   l'administrateur enregistre, **Then** l'adhésion est payée, et le montant
   reçu ainsi que le don sont conservés et affichés.
3. **Given** un montant reçu inférieur au montant à payer, **When**
   l'administrateur enregistre, **Then** l'enregistrement est refusé, et le
   message indique qu'il faut d'abord ajuster le montant à payer, avec un
   motif.
4. **Given** la création par un administrateur d'une adhésion déjà payée,
   **When** il remplit le paiement, **Then** le champ « Montant reçu » est
   proposé avec les mêmes règles.
5. **Given** une constatation de paiement annulée (006), **When**
   l'administrateur la refait, **Then** le montant reçu est saisi de nouveau ;
   l'ancien n'est plus compté.
6. **Given** un paiement constaté avec un don, **When** l'e-mail de
   confirmation du paiement est envoyé, **Then** il indique le montant reçu et
   remercie pour le don en précisant son montant.

---

### User Story 3 - Ajuster le montant d'une adhésion déjà validée (administration) (Priority: P2)

Après la validation, le bureau décide d'accorder une réduction à Marc, qui n'a
pas encore payé. L'administrateur ouvre la correction de l'adhésion (006),
saisit le nouveau montant à payer et le motif. Marc voit le nouveau montant
dans « À régler ».

**Why this priority**: les remises sont parfois décidées après la validation ;
c'est aussi le seul moyen d'enregistrer un paiement inférieur au montant
demandé.

**Independent Test**: ajuster une adhésion validée non payée, puis vérifier
le montant demandé au membre et l'impossibilité d'ajuster une adhésion payée.

**Acceptance Scenarios**:

1. **Given** une adhésion validée non payée, **When** un administrateur ouvre
   sa correction, **Then** le montant à payer et le motif actuels sont
   préremplis, avec le montant théorique affiché à côté.
2. **Given** une correction qui ajuste le montant, **When** elle est
   enregistrée, **Then** le nouveau montant à payer s'applique et la
   référence de paiement est conservée.
3. **Given** une adhésion payée, **When** un administrateur tente d'ajuster
   son montant, **Then** l'ajustement est refusé tant que la constatation du
   paiement n'est pas annulée (règle existante de 006). Pour un paiement par
   carte, l'ajustement reste impossible.
4. **Given** un paiement par carte en cours pour cette adhésion, **When**
   l'administrateur ajuste le montant, **Then** il est averti avant de
   confirmer (règle existante de 008). Si le paiement aboutit avec l'ancien
   montant, l'écart est signalé comme aujourd'hui.
5. **Given** une adhésion gratuite par ajustement (0 €), **When**
   l'administrateur la corrige à un montant positif, **Then** une référence de
   paiement lui est attribuée et le paiement redevient attendu.
6. **Given** une adhésion dont le montant a été ajusté, **When**
   l'administrateur choisit de revenir au montant théorique, **Then**
   l'ajustement et son motif sont effacés.

---

### User Story 4 - Remises et dons sur les inscriptions aux événements (administration) (Priority: P2)

Une administratrice gère le stage de printemps à 35 €. Elle réduit à 20 €
l'inscription d'un jeune bénévole qui encadre une partie du stage, avec le
motif « bénévole encadrant ». Un autre inscrit paie 40 € en espèces : elle
saisit 40 €, dont 5 € de don.

**Why this priority**: les inscriptions suivent les mêmes règles de paiement
que les adhésions (012) ; sans cela, les chiffres des événements restent
faux.

**Independent Test**: ajuster le montant d'une inscription non payée,
constater un paiement supérieur sur une autre, puis vérifier la page de
l'événement, l'espace du membre et l'export des inscrits.

**Acceptance Scenarios**:

1. **Given** une inscription confirmée non payée, **When** un administrateur
   ajuste son montant à payer, **Then** les mêmes règles que pour une
   adhésion s'appliquent : motif obligatoire s'il y a un écart avec le prix
   figé à l'inscription, montant affiché au membre, paiement par carte au
   nouveau montant, 0 € rend l'inscription gratuite.
2. **Given** l'inscription d'un membre par un administrateur, payant ou
   gratuit, **When** il la crée, **Then** il peut fixer d'emblée le montant à
   payer, prérempli avec le prix de l'événement (0 € s'il est gratuit), et le
   cas échéant le montant reçu.
3. **Given** une inscription payée en espèces, par chèque ou par virement,
   **When** l'administrateur constate le paiement, **Then** les règles du
   montant reçu et du don s'appliquent comme pour une adhésion.
4. **Given** un événement dont le prix change, **When** des inscriptions
   existent déjà, **Then** leur montant théorique, leur ajustement et leur
   montant à payer restent inchangés (règle existante de 012).
5. **Given** l'export des inscrits d'un événement, **When** un administrateur
   le télécharge, **Then** il contient pour chaque inscription le montant
   théorique, le montant à payer, le motif d'ajustement, le montant reçu et
   le don.
6. **Given** une inscription faite par un membre depuis son espace, **When**
   il s'inscrit, **Then** il ne peut pas modifier le montant : le prix de
   l'événement s'applique.

---

### User Story 5 - Chiffres fidèles à la réalité (administration) (Priority: P3)

En fin de saison, le trésorier consulte les chiffres de la saison. Il voit le
montant théorique total, le total des remises et des majorations, le montant
à payer, le montant encaissé, dont les dons, et le restant à percevoir. Les
totaux correspondent à son relevé bancaire.

**Why this priority**: c'est la finalité du suivi. Les chiffres découlent
toutefois des données saisies dans les récits précédents.

**Independent Test**: créer des adhésions avec et sans remise, payées avec et
sans don, puis vérifier chaque total des chiffres de la saison et de la page
d'un événement.

**Acceptance Scenarios**:

1. **Given** une saison, **When** un administrateur consulte ses chiffres,
   **Then** il voit, pour les adhésions validées : le montant théorique total,
   le total des remises, le total des majorations, le montant total à payer,
   le montant encaissé (montants reçus), dont le total des dons, et le
   restant à percevoir (montants à payer non encore payés).
2. **Given** la répartition par tarif, **When** elle est affichée, **Then**
   elle indique pour chaque tarif le nombre d'adhésions et le montant à payer
   cumulé.
3. **Given** le détail de l'encaissé par moyen (008), **When** il est
   affiché, **Then** chaque moyen compte les montants reçus, dons compris.
4. **Given** la page d'un événement en administration, **When** un
   administrateur la consulte, **Then** il voit, pour les inscriptions
   confirmées : le montant théorique total, les remises, les majorations, le
   montant total à payer, l'encaissé, dont les dons, et le restant à
   percevoir.
5. **Given** des paiements par carte en mode test, **When** les chiffres sont
   calculés, **Then** ils restent exclus de l'encaissé réel, comme
   aujourd'hui.

---

### Edge Cases

- **Montant à payer supérieur au théorique** : accepté avec un motif. Il
  s'agit d'une majoration décidée par l'association, distincte d'un don
  constaté au paiement.
- **Paiement par carte** : le montant demandé à HelloAsso est le montant à
  payer. Le montant reçu par carte reste celui que la vérification HelloAsso
  constate, hors contribution volontaire (008) : aucun don n'est saisi pour
  la carte. Un montant reçu par carte différent du montant à payer reste
  signalé comme une anomalie (008).
- **Tentative de paiement par carte en cours, puis ajustement** : la tentative
  en cours n'est pas reprise, puisque le montant a changé (règle existante de
  008). Une nouvelle tentative demande le nouveau montant.
- **Paiement annoncé, puis ajustement** : l'annonce du moyen est conservée ;
  le membre voit le nouveau montant dans « À régler ».
- **Ajustement à 0 € d'une adhésion dont le paiement est annoncé** : le
  paiement n'est plus attendu, l'annonce n'a plus d'objet, et la référence
  déjà attribuée est conservée pour l'historique.
- **Changement de tarif ou de type à la correction d'une adhésion ajustée** :
  le montant à payer saisi est conservé tant que l'administrateur ne revient
  pas au montant théorique. Le motif reste obligatoire si un écart subsiste.
- **Saison close** : l'ajustement d'une adhésion déjà validée suit la règle
  de correction de 006 ; il n'est plus possible une fois la saison close. La
  constatation d'un paiement, montant reçu compris, reste possible (006).
- **Le montant saisi redevient égal au théorique** : l'ajustement disparaît,
  et le motif est effacé.
- **Adhésion annulée ou inscription annulée** : montant à payer, ajustement,
  montant reçu et don restent dans l'historique, mais ne comptent plus dans
  les chiffres (règle existante).
- **Paiements constatés avant cette fonctionnalité** : leur montant reçu est
  considéré égal à leur montant à payer, sans don. Pour corriger l'un d'eux,
  l'administrateur annule la constatation, ajuste si besoin, puis la refait.
- **Montants hors bornes** : montant à payer et montant reçu sont compris
  entre 0 € et 10 000 €, au centime près, comme les tarifs.
- **Motif trop long ou vide** : refusé avec un message sur le champ (1 à 200
  caractères après suppression des espaces superflus).

## Requirements *(mandatory)*

### Functional Requirements

**Montant à payer**

- **FR-001**: Une adhésion validée et une inscription MUST conserver leur
  montant théorique, leur montant à payer et, s'ils diffèrent, le motif de
  l'ajustement.
- **FR-002**: Les administrateurs MUST pouvoir fixer le montant à payer d'une
  adhésion à sa validation, à sa création et à sa correction tant qu'elle
  n'est pas payée et que sa saison n'est pas close (règle de correction de
  006). Le champ est prérempli avec le montant théorique.
- **FR-003**: Dans le formulaire, le montant à payer MUST suivre le montant
  théorique quand le tarif ou le type change, tant que l'administrateur ne
  l'a pas modifié à la main. Une action MUST permettre de revenir au montant
  théorique.
- **FR-004**: Les administrateurs MUST pouvoir fixer le montant à payer d'une
  inscription à sa création par un administrateur, puis l'ajuster tant
  qu'elle est confirmée et non payée. Le montant théorique est le prix figé à
  l'inscription.
- **FR-005**: Un montant à payer différent du montant théorique MUST être
  accompagné d'un motif de 1 à 200 caractères. Un montant égal au théorique
  MUST NOT conserver de motif.
- **FR-006**: Un montant à payer de 0 € MUST rendre l'adhésion ou
  l'inscription gratuite, avec les mêmes effets qu'un tarif ou un prix
  gratuit. Repasser à un montant positif MUST rendre le paiement de nouveau
  attendu, avec une référence de paiement (la référence existante si elle a
  déjà été attribuée).
- **FR-007**: Le montant à payer MUST être le seul montant demandé au membre :
  « À régler », pages de l'adhésion et de l'inscription, e-mails, paiement
  par carte via HelloAsso. Le membre MUST NOT voir le motif de l'ajustement.
- **FR-008**: Seuls les administrateurs MUST pouvoir fixer ou ajuster un
  montant à payer. Un membre qui s'inscrit ou demande une adhésion depuis son
  espace MUST NOT pouvoir le modifier.
- **FR-009**: Chaque ajustement MUST être journalisé avec l'administrateur,
  le montant théorique, l'ancien et le nouveau montant à payer. Comme les
  autres motifs (006), le texte du motif n'est pas écrit dans le journal :
  il reste consultable sur l'adhésion ou l'inscription.
- **FR-010**: Un ajustement MUST NOT être possible sur une adhésion ou une
  inscription payée ; pour un paiement constaté manuellement, il faut d'abord
  annuler la constatation (006, 012).

**Montant reçu et dons**

- **FR-011**: À la constatation d'un paiement en espèces, par chèque ou par
  virement, par la constatation directe, à la création d'une adhésion ou
  d'une inscription déjà payée, les administrateurs MUST saisir un montant
  reçu, prérempli avec le montant à payer.
- **FR-012**: Un montant reçu inférieur au montant à payer MUST être refusé,
  avec un message qui invite à ajuster d'abord le montant à payer.
- **FR-013**: L'excédent du montant reçu sur le montant à payer MUST être
  conservé comme un don et affiché sur l'adhésion ou l'inscription, en
  administration et dans l'espace du membre.
- **FR-014**: Le montant reçu par carte MUST rester celui constaté par la
  vérification HelloAsso, hors contribution volontaire (008). Aucun don n'est
  saisi pour un paiement par carte.
- **FR-015**: L'annulation d'une constatation manuelle MUST effacer le montant
  reçu et le don.
- **FR-016**: L'e-mail de confirmation de paiement MUST indiquer le montant
  reçu et, s'il y a un don, son montant avec un remerciement.
- **FR-017**: Les paiements constatés avant cette fonctionnalité MUST être
  considérés comme reçus pour leur montant à payer, sans don.

**Chiffres et exports**

- **FR-018**: Les chiffres d'une saison MUST distinguer, pour les adhésions
  validées : montant théorique total, total des remises, total des
  majorations, montant total à payer, encaissé (somme des montants reçus,
  détaillée par moyen), dont dons, et restant à percevoir (somme des montants
  à payer non payés).
- **FR-019**: La répartition par tarif MUST cumuler les montants à payer.
- **FR-020**: La page d'un événement en administration MUST afficher les
  mêmes totaux pour ses inscriptions confirmées.
- **FR-021**: L'export des inscrits d'un événement MUST inclure le montant
  théorique, le montant à payer, le motif d'ajustement, le montant reçu et le
  don.
- **FR-022**: Les paiements par carte en mode test MUST rester exclus de
  l'encaissé réel et des dons (008).

### Key Entities

- **Adhésion** (006) : s'enrichit du montant théorique, du motif d'ajustement
  et, une fois payée, du montant reçu, quel que soit le moyen. Le montant
  appliqué de 006 devient le montant à payer.
- **Inscription** (012) : s'enrichit du montant théorique, du motif
  d'ajustement et, une fois payée, du montant reçu. Le montant dû de 012
  devient le montant à payer.
- Le **don** et l'**ajustement** ne sont pas stockés à part : ils se déduisent
  du montant reçu, du montant à payer et du montant théorique.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Un administrateur accorde une remise en validant une adhésion en
  moins de 30 secondes de plus qu'une validation sans remise.
- **SC-002**: Pour une saison où toutes les réceptions ont été saisies,
  l'encaissé affiché correspond à 100 % aux sommes réellement reçues (relevé
  bancaire, espèces et chèques), dons compris.
- **SC-003**: Plus aucun écart de montant n'a besoin d'être consigné dans le
  commentaire libre du paiement pour être pris en compte dans les chiffres.
- **SC-004**: 100 % des montants demandés aux membres (espace membre, e-mails,
  paiement par carte) sont égaux au montant à payer.
- **SC-005**: Chaque remise ou majoration est justifiée par un motif visible
  sur l'adhésion ou l'inscription, et chaque ajustement est retrouvable dans
  le journal.

## Assumptions

- **Pas de codes de réduction** : les remises sont décidées au cas par cas par
  les administrateurs. Les codes, les remises automatiques (fratrie,
  bénévoles) et les remises en pourcentage sont hors périmètre.
- **Un paiement, une adhésion ou une inscription** : un paiement qui couvre
  plusieurs personnes n'est pas réparti. L'excédent est un don, et le
  commentaire libre peut préciser à qui il se rapporte. Aucune fiche n'est
  créée pour les tiers mentionnés dans un paiement.
- **Motif réservé aux administrateurs** : le motif peut contenir des
  informations sensibles (situation financière) ; le membre voit seulement
  le montant à payer.
- **Pas d'e-mail dédié à l'ajustement** : un membre dont le montant change
  après la validation le voit dans « À régler ». Les e-mails existants
  (validation, confirmation d'inscription, paiement) portent le montant à
  payer en vigueur.
- **Dons sans reçu fiscal** : Frigg compte les dons mais n'émet pas de reçu
  fiscal ; c'est hors périmètre.
- **Pas de paiement partiel** : la règle de 006 reste en vigueur. Un montant
  inférieur passe par un ajustement motivé, jamais par un reste dû.
- Les bornes des montants (0 € à 10 000 €) et la précision au centime sont
  celles des tarifs existants.
