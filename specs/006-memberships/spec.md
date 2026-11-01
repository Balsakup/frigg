# Feature Specification: Adhésions

**Feature Branch**: `006-memberships`

**Created**: 2026-11-01

**Status**: Draft

**Input**: User description: "Adhésion côté admin et membre"

## Vocabulaire

- **Adhésion** : l'inscription d'un membre (fiche, 005) à l'association pour
  une saison (003), à un tarif de cette saison (004).
- **Demande d'adhésion** : adhésion déposée depuis l'espace membre, en attente
  de la décision d'un administrateur. Toute demande est validée ou refusée
  manuellement.
- **Type d'adhésion** : « Première adhésion » ou « Renouvellement ». L'adhérent
  le déclare ; l'administrateur retient le type définitif à la validation.
- **Montant appliqué** : le montant du tarif correspondant au type retenu
  (montant de renouvellement si le tarif en a un et que le type retenu est
  « Renouvellement », montant principal sinon), figé à la validation.
- **Moyen de paiement** : espèces, chèque ou virement. Chaque moyen est activé
  ou non et décrit (instructions, coordonnées bancaires…) dans les paramètres
  de Frigg.
- **État du paiement** d'une adhésion validée : « À payer » (aucun moyen
  choisi), « Paiement annoncé » (l'adhérent a choisi un moyen), « Payée »
  (un administrateur a constaté la réception). Une adhésion gratuite
  (montant appliqué de 0 €) n'a pas de paiement.
- **Référence de paiement** : code unique attribué à chaque adhésion validée,
  à reporter sur un virement ou au dos d'un chèque pour faciliter le
  rapprochement.

## Clarifications

### Session 2026-11-01

- Q: Frigg gère-t-il le paiement de l'adhésion ? → A: Une fois l'adhésion
  validée par un administrateur, un lien est envoyé pour payer. L'adhérent
  choisit son moyen de paiement (espèces, chèque ou virement) ; les
  informations du moyen choisi s'affichent. Ces informations sont paramétrables
  dans les paramètres de Frigg.
- Q: L'adhérent choisit-il librement son tarif ? → A: Choix libre parmi tous
  les tarifs de la saison ; l'administrateur corrige à la validation.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Demander l'adhésion d'un de mes membres (Priority: P1)

Claire ouvre la rubrique « Adhésions » de l'espace membre. La saison 2026-2027
est ouverte et porte trois tarifs. Elle choisit « Demander une adhésion »,
sélectionne son fils Léo parmi ses membres, choisit le tarif « Jeune », indique
que Léo était déjà adhérent la saison passée (« Renouvellement ») et voit le
montant correspondant, 40 € au lieu de 60 €. Elle envoie la demande : Léo
apparaît dans la rubrique avec l'état « En attente de validation ».

**Why this priority**: c'est la raison d'être de Frigg pour les familles ; sans
demande, aucune adhésion n'existe.

**Independent Test**: avec une saison ouverte portant des tarifs et un compte
gérant deux fiches, déposer une demande pour une fiche, la retrouver en attente,
constater que la même fiche ne peut pas faire une seconde demande pour cette
saison.

**Acceptance Scenarios**:

1. **Given** au moins une saison ouverte, non close, portant au moins un tarif,
   et un compte gérant au moins une fiche, **When** le titulaire ouvre
   « Adhésions », **Then** il peut choisir « Demander une adhésion ».
2. **Given** le formulaire de demande, **When** le titulaire choisit la
   saison (présélectionnée s'il n'y en a qu'une), le membre, le tarif et
   déclare « Première adhésion » ou « Renouvellement », **Then** le montant
   correspondant au tarif et au type déclaré s'affiche avant l'envoi.
3. **Given** un tarif sans montant de renouvellement, **When** le titulaire
   déclare « Renouvellement », **Then** le montant affiché est le montant
   principal du tarif.
4. **Given** le formulaire complété, **When** le titulaire envoie la demande,
   **Then** elle est enregistrée « En attente de validation » et un message
   confirme l'action.
5. **Given** un membre ayant déjà une demande en attente ou une adhésion validée
   pour une saison, **When** le titulaire ouvre le formulaire pour cette
   saison, **Then** ce membre n'est pas proposé, et le serveur refuse toute
   seconde demande.
6. **Given** un membre dont la demande a été refusée ou annulée pour une
   saison, **When** le titulaire dépose une nouvelle demande pour la même
   saison, **Then** elle est acceptée.
7. **Given** aucune saison ouverte avec tarif, **When** le titulaire ouvre
   « Adhésions », **Then** un message indique que les adhésions ne sont pas
   ouvertes pour le moment et aucune demande n'est proposée.
8. **Given** un compte sans fiche, **When** le titulaire ouvre « Adhésions »,
   **Then** un message l'invite à créer d'abord un membre, avec un lien vers
   « Mes membres ».
9. **Given** une saison en brouillon, close, ou ouverte sans tarif, **When**
   le titulaire ouvre le formulaire, **Then** cette saison n'est pas proposée,
   et le serveur refuse une demande qui la viserait.

---

### User Story 2 - Valider ou refuser les demandes (administration) (Priority: P1)

Dans l'espace administration, la rubrique « Adhésions » affiche les demandes en
attente en tête. L'administrateur ouvre celle de Léo : il voit le membre, le
compte qui l'a déposée, le tarif choisi, le type déclaré, et l'aide « Adhérent
la saison précédente dans Frigg : oui ». Il valide en retenant
« Renouvellement » : l'adhésion est validée à 40 €. Il ouvre ensuite une
demande au tarif « Jeune » pour une personne de 45 ans, corrige le tarif en
« Adulte » et valide. Il refuse enfin une demande déposée en double pour une
personne déjà inscrite sous une autre fiche, en indiquant le motif.

**Why this priority**: toutes les demandes sont validées manuellement ; sans
cette étape, aucune adhésion n'aboutit.

**Independent Test**: avec trois demandes en attente, en valider une sans
changement, en valider une en changeant le tarif et le type, en refuser une
avec motif ; vérifier l'état, le type retenu et le montant appliqué de chacune.

**Acceptance Scenarios**:

1. **Given** des demandes en attente, **When** l'administrateur ouvre la
   rubrique « Adhésions », **Then** les demandes en attente apparaissent en
   premier, des plus anciennes aux plus récentes, et leur nombre est indiqué.
2. **Given** une demande, **When** l'administrateur l'ouvre, **Then** il voit
   le membre (prénom, nom, âge, lien vers sa fiche), le compte de
   rattachement, la saison, le tarif choisi, le type déclaré, le montant
   correspondant, la date de la demande et l'aide « Adhérent la saison
   précédente dans Frigg » (oui, non ou inconnu).
3. **Given** une demande, **When** l'administrateur valide sans rien changer,
   **Then** l'adhésion est validée avec le type déclaré, le tarif choisi et le
   montant appliqué correspondant.
4. **Given** une demande, **When** l'administrateur change le type retenu ou le
   tarif (parmi les tarifs de la même saison) avant de valider, **Then**
   l'adhésion est validée avec ces valeurs et le montant appliqué recalculé ;
   le type déclaré par l'adhérent reste consultable.
5. **Given** une demande, **When** l'administrateur la refuse en saisissant un
   motif, **Then** elle passe à l'état « Refusée » et le motif est visible du
   compte qui l'a déposée ; refuser sans motif est impossible.
6. **Given** une demande validée, refusée ou annulée entre-temps, **When**
   l'administrateur valide ou refuse, **Then** l'action est refusée avec un
   message explicite et la demande est rafraîchie.
7. **Given** une validation ou un refus, **When** l'action est enregistrée,
   **Then** un e-mail informe le compte qui a déposé la demande de la décision
   (et du motif en cas de refus) ; l'e-mail de validation contient le lien
   vers la page de paiement de l'adhésion, sauf si elle est gratuite.
8. **Given** un membre connecté, **When** il tente d'accéder à la rubrique
   « Adhésions » de l'administration ou à une de ses actions, **Then** l'accès
   est refusé.

---

### User Story 3 - Suivre et annuler mes demandes (Priority: P1)

Claire retrouve dans « Adhésions » toutes les adhésions de ses membres,
regroupées par saison : Léo « Validée — Renouvellement, 40 € », sa fille
« En attente de validation ». S'étant trompée de tarif pour sa fille, elle
annule la demande et en dépose une nouvelle.

**Why this priority**: sans suivi, les familles sollicitent le bureau pour
connaître l'état de leurs demandes ; l'annulation corrige une erreur sans
intervention.

**Independent Test**: avec des adhésions dans chaque état, vérifier
l'affichage de chacune pour le compte, annuler une demande en attente,
constater qu'une adhésion validée ne peut pas être annulée par le compte.

**Acceptance Scenarios**:

1. **Given** des adhésions pour les membres du compte, **When** le titulaire
   ouvre « Adhésions », **Then** il voit, par saison de la plus récente à la
   plus ancienne, chaque adhésion avec le membre, le tarif, le type, le
   montant et l'état (en attente, validée, refusée avec son motif, annulée),
   ainsi que l'état du paiement d'une adhésion validée et, s'il reste à payer,
   l'action « Payer ».
2. **Given** une demande en attente, **When** le titulaire choisit « Annuler la
   demande » et confirme dans le dialogue, **Then** elle passe à l'état
   « Annulée ».
3. **Given** une adhésion validée ou refusée, **When** le titulaire consulte la
   rubrique, **Then** aucune action d'annulation n'est proposée et le serveur
   refuse toute annulation.
4. **Given** une adhésion d'un membre d'un autre compte, **When** le titulaire
   tente de la consulter ou de l'annuler, quel que soit le moyen, **Then** la
   réponse est identique à celle d'une adhésion inexistante.
5. **Given** la fiche d'un de ses membres, **When** le titulaire l'ouvre,
   **Then** elle affiche l'état de l'adhésion de ce membre pour chaque saison
   ouverte (« Non adhérent » le cas échéant).

---

### User Story 4 - Payer une adhésion validée (Priority: P1)

Claire reçoit l'e-mail « Adhésion de Léo validée » avec un lien « Payer
l'adhésion ». Connectée, elle arrive sur la page de paiement : montant 40 €,
référence « ADH-2627-0042 », et le choix entre espèces, chèque et virement.
Elle choisit « Virement » : la page affiche le titulaire du compte, l'IBAN, le
BIC, la référence à indiquer et les instructions saisies par le bureau. Elle
confirme son choix ; l'adhésion passe à « Paiement annoncé — virement ». Une
semaine plus tard, le trésorier constate le virement et marque l'adhésion
« Payée ».

**Why this priority**: l'adhésion n'est complète qu'une fois payée ; afficher
au bon moment les informations du moyen choisi évite aux familles de les
demander au bureau.

**Independent Test**: avec les trois moyens paramétrés, valider une demande,
suivre le lien de l'e-mail, choisir chaque moyen tour à tour et vérifier les
informations affichées, confirmer le virement, puis marquer l'adhésion payée
côté administration.

**Acceptance Scenarios**:

1. **Given** une adhésion validée à payer, **When** le titulaire suit le lien
   de l'e-mail ou choisit « Payer » dans « Adhésions », **Then** une page
   affiche le membre, la saison, le tarif, le montant, la référence de
   paiement et les moyens de paiement activés ; non connecté, il est d'abord
   invité à se connecter puis ramené sur cette page.
2. **Given** la page de paiement, **When** le titulaire sélectionne
   « Espèces », **Then** les instructions paramétrées pour les espèces
   s'affichent (lieu, permanences…).
3. **Given** la page de paiement, **When** le titulaire sélectionne
   « Chèque », **Then** l'ordre du chèque, l'adresse d'envoi éventuelle, les
   instructions paramétrées et la référence à inscrire au dos s'affichent.
4. **Given** la page de paiement, **When** le titulaire sélectionne
   « Virement », **Then** le titulaire du compte bancaire, l'IBAN, le BIC
   éventuel, les instructions paramétrées et la référence à indiquer dans le
   libellé s'affichent, chaque information pouvant être copiée en un geste.
5. **Given** un moyen sélectionné, **When** le titulaire confirme son choix,
   **Then** l'adhésion passe à « Paiement annoncé » avec ce moyen, et les
   informations restent consultables depuis « Adhésions ».
6. **Given** un paiement annoncé non encore constaté, **When** le titulaire
   choisit un autre moyen, **Then** le nouveau choix remplace le précédent.
7. **Given** un moyen désactivé dans les paramètres, **When** la page de
   paiement s'affiche, **Then** ce moyen n'est pas proposé ; s'il avait été
   choisi auparavant, la page invite à en choisir un autre.
8. **Given** aucun moyen activé, **When** la page de paiement s'affiche,
   **Then** un message invite à contacter le bureau pour régler l'adhésion.
9. **Given** une adhésion à payer ou au paiement annoncé, **When** un
   administrateur choisit « Marquer comme payée », indique la date de
   réception et le moyen effectivement utilisé (prérempli avec le moyen
   annoncé) et confirme, **Then** l'adhésion passe à « Payée » et le compte
   de rattachement en est informé par e-mail.
10. **Given** une adhésion « Payée » par erreur, **When** un administrateur
    annule la constatation et confirme, **Then** l'adhésion revient à l'état
    de paiement précédent.
11. **Given** une adhésion gratuite, **When** elle est validée, **Then**
    aucune page de paiement ni action « Payer » n'est proposée et l'adhésion
    est affichée « Gratuite ».
12. **Given** une adhésion payée, refusée, annulée ou en attente, ou une
    adhésion d'une fiche gérée par un autre compte, **When** le titulaire
    ouvre son lien de paiement, **Then** la page indique que l'adhésion n'est
    plus à payer, ou répond comme pour une adhésion inexistante.

---

### User Story 5 - Paramétrer les moyens de paiement (administration) (Priority: P1)

Avant d'ouvrir les adhésions, le trésorier ouvre « Paramètres » dans la
section Configuration de l'administration. Dans « Moyens de paiement », il
active le chèque (ordre « Association Frigg », adresse d'envoi, instructions)
et le virement (titulaire, IBAN, BIC, instructions), et laisse les espèces
désactivées.

**Why this priority**: sans paramétrage, la page de paiement ne peut rien
afficher ; c'est un prérequis du parcours de paiement.

**Independent Test**: activer et renseigner chaque moyen, constater les
informations sur une page de paiement, désactiver un moyen et constater sa
disparition.

**Acceptance Scenarios**:

1. **Given** un administrateur, **When** il ouvre « Paramètres », **Then** la
   section « Moyens de paiement » présente espèces, chèque et virement, chacun
   avec un interrupteur d'activation et ses champs.
2. **Given** le moyen « Virement » activé, **When** l'administrateur
   enregistre sans titulaire ou avec un IBAN invalide, **Then**
   l'enregistrement est refusé et le champ concerné affiche l'erreur.
3. **Given** le moyen « Chèque » activé, **When** l'administrateur enregistre
   sans ordre, **Then** l'enregistrement est refusé et le champ affiche
   l'erreur.
4. **Given** un moyen désactivé, **When** l'administrateur enregistre, **Then**
   ses informations sont conservées pour une réactivation ultérieure, mais ne
   sont plus affichées aux adhérents.
5. **Given** des informations modifiées, **When** un adhérent consulte une
   page de paiement, **Then** il voit les informations à jour.
6. **Given** aucun moyen activé et au moins une saison ouverte avec tarif,
   **When** l'administrateur consulte le tableau de bord, **Then** un encart
   l'invite à paramétrer les moyens de paiement.
7. **Given** un membre connecté, **When** il tente d'accéder à « Paramètres »
   ou de les modifier, **Then** l'accès est refusé.

---

### User Story 6 - Enregistrer une adhésion directement (administration) (Priority: P2)

Une famille remet un bulletin papier. L'administrateur crée l'adhésion depuis
la rubrique « Adhésions » ou depuis la fiche du membre : il choisit la saison,
le membre, le tarif et le type ; l'adhésion est enregistrée directement à
l'état « Validée ».

**Why this priority**: indispensable pour les familles sans accès numérique et
les fiches « Sans compte », mais la majorité des adhésions passe par les
demandes (US1).

**Independent Test**: créer une adhésion pour une fiche sans compte et pour une
fiche rattachée ; vérifier qu'elles sont validées et que la seconde apparaît
dans « Adhésions » du compte.

**Acceptance Scenarios**:

1. **Given** une saison ouverte avec tarif, **When** l'administrateur crée une
   adhésion en choisissant le membre par recherche, le tarif et le type,
   **Then** l'adhésion est validée avec le montant appliqué correspondant.
2. **Given** un membre ayant déjà une demande en attente ou une adhésion
   validée pour la saison, **When** l'administrateur tente de lui créer une
   adhésion, **Then** la création est refusée et le message propose d'ouvrir
   l'adhésion existante.
3. **Given** une fiche « Sans compte », **When** l'administrateur lui crée une
   adhésion, **Then** elle est validée et n'apparaît dans aucun espace membre.
4. **Given** le formulaire de création, **When** l'administrateur indique que
   le règlement a été reçu (moyen et date), **Then** l'adhésion est créée
   directement « Payée » ; sinon elle est « À payer » et, si la fiche a un
   compte, l'e-mail de validation avec le lien de paiement est envoyé.

---

### User Story 7 - Annuler ou corriger une adhésion validée (administration) (Priority: P2)

Un adhérent quitte l'association en début de saison, ou l'administrateur
s'aperçoit d'une erreur de tarif après validation. Il peut corriger le tarif et
le type d'une adhésion validée, ou l'annuler en indiquant un motif.

**Why this priority**: les erreurs et départs sont rares mais doivent pouvoir
être corrigés sans manipulation technique.

**Independent Test**: corriger le tarif d'une adhésion validée et vérifier le
montant appliqué recalculé ; annuler une adhésion validée et vérifier que le
membre peut de nouveau adhérer pour cette saison.

**Acceptance Scenarios**:

1. **Given** une adhésion validée non payée d'une saison non close, **When**
   l'administrateur change son tarif ou son type retenu, **Then** le montant
   appliqué est recalculé à partir du tarif actuel ; une adhésion « Payée »
   ne peut être corrigée qu'après annulation de la constatation du paiement.
2. **Given** une adhésion validée d'une saison non close, **When**
   l'administrateur choisit « Annuler l'adhésion », saisit un motif et
   confirme, **Then** elle passe à l'état « Annulée » et le compte de
   rattachement en est informé par e-mail ; si elle était payée, le dialogue
   rappelle qu'un éventuel remboursement se fait hors de Frigg.
3. **Given** une adhésion d'une saison close, **When** l'administrateur la
   consulte, **Then** elle est en lecture seule, à l'exception de la
   constatation du paiement, qui reste possible.

---

### User Story 8 - Suivre les chiffres de l'adhésion (administration) (Priority: P3)

Le bureau prépare l'assemblée générale : pour chaque saison, il veut connaître
le nombre d'adhérents, la part de premières adhésions et de renouvellements,
la répartition par tarif, le montant total attendu et ce qui a été encaissé.

**Why this priority**: l'utilisateur veut ces statistiques, mais elles
découlent des données enregistrées par les autres parcours.

**Independent Test**: avec des adhésions de chaque type et état sur une
saison, vérifier que les compteurs ne prennent en compte que les adhésions
validées et correspondent aux données.

**Acceptance Scenarios**:

1. **Given** une saison, **When** l'administrateur consulte la rubrique
   « Adhésions » filtrée sur cette saison, **Then** il voit le nombre
   d'adhésions validées, dont premières adhésions et renouvellements, le
   nombre de demandes en attente, la répartition par tarif, la somme des
   montants appliqués, le montant encaissé (adhésions payées) et le montant
   restant à percevoir, ce dernier détaillé par moyen annoncé.
2. **Given** le tableau de bord de l'administration, **When** des demandes
   sont en attente, **Then** un encart indique leur nombre et mène à la liste
   filtrée ; sans demande en attente, aucun encart n'apparaît.
3. **Given** la liste des adhésions, **When** l'administrateur filtre sur
   l'état du paiement (à payer, paiement annoncé, payée) ou recherche une
   référence de paiement, **Then** seules les adhésions correspondantes
   s'affichent.

---

### Edge Cases

- **Aide « adhérent la saison précédente »** : « oui » si le membre a une
  adhésion validée dans la saison qui précède celle de la demande ; « non » si
  cette saison existe dans Frigg et compte au moins une adhésion validée, mais
  aucune pour ce membre ; « inconnu » sinon (pas de saison précédente, ou
  saison antérieure à l'usage de Frigg pour les adhésions). L'aide ne décide
  jamais du type.
- **Tarif modifié après la demande** : le montant affiché sur une demande en
  attente suit le tarif actuel ; il est figé à la validation. Modifier un
  tarif ensuite ne change pas le montant appliqué des adhésions validées.
- **Tarif supprimé** : un tarif utilisé par au moins une adhésion (quel que
  soit son état) ne peut pas être supprimé ; le message l'explique. Cette
  règle complète 004.
- **Saison close pendant l'attente** : une demande en attente sur une saison
  devenue close peut encore être validée ou refusée par un administrateur ;
  aucune nouvelle demande ni création n'est possible sur une saison close.
- **Saison ouverte sans tarif** : non proposée à l'adhésion (004).
- **Fiche supprimée** : un compte ne peut pas supprimer une fiche portant une
  demande en attente ou une adhésion validée ; le message invite à annuler la
  demande ou à contacter le bureau. Un administrateur peut supprimer la fiche
  après un dialogue qui nomme les adhésions supprimées avec elle. Cette règle
  complète 005.
- **Rattachement changé** (005) : les adhésions suivent la fiche et
  apparaissent dans l'espace du nouveau compte ; une fiche devenue « Sans
  compte » conserve ses adhésions.
- **Compte supprimé en conservant les fiches** (005) : les adhésions sont
  conservées avec les fiches.
- **Deux demandes simultanées** pour le même membre et la même saison : une
  seule est enregistrée ; l'autre reçoit un message explicite.
- **Membre mineur ou âge incohérent avec le tarif** : aucun contrôle
  automatique ; l'administrateur corrige le tarif à la validation.
- **Compte désactivé** (002) : ses demandes en attente restent traitables par
  les administrateurs ; les e-mails de décision ne lui sont pas envoyés.
- **Paiement d'une fiche sans compte** : aucun lien n'est envoyé ;
  l'administrateur constate le paiement directement.
- **Rattachement changé après validation** : le lien de paiement est valable
  pour le compte qui gère la fiche au moment où il est ouvert ; l'ancien compte
  reçoit la réponse d'une adhésion inexistante.
- **Paramètres modifiés après un choix** : la page affiche toujours les
  informations en vigueur ; un paiement annoncé n'est pas modifié.
- **Montant différent reçu** : pas de paiement partiel ; l'administrateur
  constate le paiement ou non, et peut noter un commentaire libre (par exemple
  le numéro du chèque).
- **Référence de paiement** : unique dans l'instance, courte (au plus 16
  caractères), sans données personnelles, conservée même si le tarif est
  corrigé.

## Requirements *(mandatory)*

### Functional Requirements

**Adhésion**

- **FR-001**: Une adhésion MUST porter sur une fiche membre, une saison et un
  tarif de cette saison, et avoir l'un des états « En attente de
  validation », « Validée », « Refusée » ou « Annulée ».
- **FR-002**: Une adhésion MUST conserver le type déclaré par l'adhérent (pour
  une demande), le type retenu et le montant appliqué (à la validation), le
  compte à l'origine de la demande ou l'administrateur qui l'a créée, les dates
  de demande et de décision, l'administrateur qui a décidé, et le motif en cas
  de refus ou d'annulation. Une adhésion validée MUST en outre conserver sa
  référence de paiement, l'état de son paiement, le moyen annoncé, et, une fois
  payée, le moyen effectif, la date de réception, l'administrateur qui l'a
  constatée et un commentaire facultatif.
- **FR-003**: Un membre MUST avoir au plus une adhésion en attente ou validée
  par saison ; les adhésions refusées ou annulées n'empêchent pas une nouvelle
  demande.
- **FR-004**: Le montant appliqué MUST être figé à la validation et ne plus
  dépendre des modifications ultérieures du tarif, sauf correction explicite
  par un administrateur (FR-016).
- **FR-005**: Un tarif utilisé par au moins une adhésion MUST NOT pouvoir être
  supprimé.

**Espace membre**

- **FR-006**: Tout compte connecté MUST pouvoir déposer une demande
  d'adhésion pour une fiche qu'il gère, sur une saison ouverte, non close,
  portant au moins un tarif.
- **FR-007**: Le formulaire de demande MUST faire choisir la saison, le
  membre, le tarif (avec libellé, montant(s) et description) et le type
  déclaré, et afficher le montant correspondant avant l'envoi.
- **FR-008**: Un compte MUST pouvoir consulter les adhésions des fiches qu'il
  gère, par saison, avec leur état, leur tarif, leur type, leur montant et le
  motif d'un refus ou d'une annulation, et uniquement celles-ci ; toute
  tentative d'accès à une autre adhésion MUST recevoir la même réponse qu'une
  adhésion inexistante.
- **FR-009**: Un compte MUST pouvoir annuler une demande en attente des fiches
  qu'il gère, après confirmation ; il MUST NOT pouvoir modifier ni annuler une
  adhésion validée ou refusée.
- **FR-010**: La fiche d'un membre (espace membre et administration) MUST
  afficher ses adhésions par saison.
- **FR-010a**: Un compte MUST NOT pouvoir supprimer une fiche portant une
  demande en attente ou une adhésion validée.

**Administration**

- **FR-011**: Les administrateurs MUST pouvoir lister les adhésions,
  filtrées par saison (par défaut la saison courante, ou à défaut la plus
  récente ouverte) et par état, avec recherche sur le prénom ou le nom du
  membre selon les règles de 005 ; les demandes en attente apparaissent en
  premier, des plus anciennes aux plus récentes. Les filtres sont reflétés dans
  l'adresse de la page.
- **FR-012**: Les administrateurs MUST pouvoir valider une demande en
  retenant le type et le tarif (de la même saison) de leur choix, et la
  refuser avec un motif obligatoire.
- **FR-013**: La consultation d'une demande MUST afficher l'aide « Adhérent la
  saison précédente dans Frigg » (oui, non, inconnu) selon la règle décrite
  dans les cas limites.
- **FR-014**: Les administrateurs MUST pouvoir créer une adhésion directement
  validée pour toute fiche, rattachée ou sans compte, sur une saison ouverte,
  non close, portant au moins un tarif.
- **FR-015**: Les administrateurs MUST pouvoir annuler une adhésion validée
  d'une saison non close, avec un motif obligatoire et après confirmation.
- **FR-016**: Les administrateurs MUST pouvoir corriger le tarif et le type
  retenu d'une adhésion validée non payée d'une saison non close ; le montant
  appliqué est alors recalculé à partir du tarif actuel.
- **FR-017**: Les administrateurs MUST pouvoir valider ou refuser une demande
  en attente d'une saison close et constater le paiement d'une adhésion d'une
  saison close ; les adhésions d'une saison close sont sinon en lecture seule.
- **FR-018**: La rubrique MUST afficher, pour la saison filtrée, le nombre
  d'adhésions validées, dont premières adhésions et renouvellements, le nombre
  de demandes en attente, la répartition par tarif, la somme des montants
  appliqués, le montant encaissé et le montant restant à percevoir détaillé
  par moyen annoncé ; seules les adhésions validées entrent dans les comptes.
  La liste MUST pouvoir être filtrée par état du paiement et la recherche MUST
  accepter une référence de paiement.
- **FR-019**: Le tableau de bord de l'administration MUST afficher un encart
  indiquant le nombre de demandes en attente lorsqu'il y en a, menant à la
  liste filtrée.
- **FR-020**: La rubrique « Adhésions » de l'administration et ses actions
  MUST être refusées aux membres et aux personnes non connectées.
- **FR-021**: La suppression d'une fiche par un administrateur MUST, si elle
  porte des adhésions, les nommer dans le dialogue de confirmation et les
  supprimer avec elle.

**Paiement**

- **FR-022**: Frigg MUST NOT encaisser de paiement : il présente les
  informations du moyen choisi par l'adhérent et enregistre la constatation
  du paiement par un administrateur.
- **FR-022a**: Toute adhésion validée d'un montant appliqué non nul MUST
  recevoir une référence de paiement unique et l'état de paiement « À payer » ;
  une adhésion de 0 € est « Gratuite » et n'a pas de paiement.
- **FR-022b**: Le compte qui gère la fiche MUST pouvoir ouvrir la page de
  paiement d'une adhésion validée non payée, y choisir un moyen parmi les
  moyens activés, voir les informations paramétrées de ce moyen avec le
  montant et la référence, et confirmer son choix (« Paiement annoncé »). Il
  MUST pouvoir changer de moyen tant que le paiement n'est pas constaté.
- **FR-022c**: La page de paiement MUST exiger une session ; le lien de
  l'e-mail ramène sur cette page après connexion.
- **FR-022d**: Les administrateurs MUST pouvoir marquer une adhésion « Payée »
  (moyen effectif, date de réception ni future ni antérieure à la validation,
  commentaire facultatif) et annuler cette constatation, après confirmation.
  Pour un règlement saisi à la création directe d'une adhésion (bulletin
  papier), la date de réception ne peut être ni future ni antérieure au début
  de la saison.
- **FR-022e**: Un compte MUST NOT pouvoir marquer lui-même une adhésion
  comme payée.

**Paramètres**

- **FR-022f**: L'administration MUST offrir une rubrique « Paramètres »,
  placée dans la section « Configuration » de la barre latérale après
  « Comptes » et avant « Aide », réservée aux administrateurs. Elle est
  organisée en sous-pages, une par section, chacune avec sa propre adresse et
  une navigation entre sections ; la première section est « Paiements ».
- **FR-022g**: La section « Moyens de paiement » MUST permettre d'activer ou
  de désactiver chacun des trois moyens et de renseigner :
  - espèces : instructions (obligatoires si activé, 1 000 caractères au plus) ;
  - chèque : ordre (obligatoire si activé, 120 caractères au plus), adresse
    d'envoi et instructions (facultatives) ;
  - virement : titulaire du compte (obligatoire si activé, 120 caractères au
    plus), IBAN (obligatoire si activé, clé de contrôle vérifiée), BIC
    (facultatif, format vérifié) et instructions (facultatives).
- **FR-022h**: Les informations d'un moyen désactivé MUST être conservées et
  ne plus être montrées aux adhérents.
- **FR-022i**: Le tableau de bord de l'administration MUST afficher un encart
  invitant à paramétrer les moyens de paiement lorsqu'aucun n'est activé et
  qu'une saison ouverte porte au moins un tarif.

**Commun**

- **FR-023**: Chaque validation, refus, annulation et constatation de
  paiement MUST déclencher un e-mail au compte de rattachement actif de la
  fiche, envoyé de façon différée ; l'e-mail de validation d'une adhésion non
  gratuite contient le lien vers sa page de paiement. Aucun e-mail n'est envoyé
  pour une fiche sans compte.
- **FR-024**: Chaque erreur de saisie MUST être affichée sur le champ concerné
  avec un message en français ; une action sur une adhésion dont l'état a
  changé entre-temps MUST être refusée avec un message explicite.
- **FR-025**: Après chaque action, un message de confirmation annoncé aux
  lecteurs d'écran MUST s'afficher et les listes MUST refléter immédiatement le
  nouvel état.
- **FR-026**: Le système MUST enregistrer dans les journaux de l'application
  chaque demande, décision, correction, annulation, choix de moyen,
  constatation de paiement et modification des paramètres, avec le compte à
  l'origine, l'adhésion visée et la date, sans données personnelles du membre
  ni coordonnées bancaires.
- **FR-027**: Les écrans MUST être en français, utilisables au clavier seul,
  compatibles avec les lecteurs d'écran et utilisables dès 320 pixels de
  large ; la rubrique « Adhésions » remplace la page « bientôt disponible »
  dans les deux espaces (001, FR-025).

### Key Entities

- **Adhésion** : inscription d'un membre à une saison. Attributs : fiche
  membre, saison, tarif, état, type déclaré, type retenu, montant appliqué,
  origine (compte demandeur ou administrateur créateur), dates de demande et de
  décision, administrateur décideur, motif de refus ou d'annulation.
- **Paiement d'une adhésion** : rattaché à une adhésion validée non gratuite.
  Attributs : référence, état (à payer, paiement annoncé, payée), moyen
  annoncé, moyen effectif, date de réception, administrateur ayant constaté,
  commentaire.
- **Paramètres des moyens de paiement** : données singleton de l'instance.
  Pour chaque moyen (espèces, chèque, virement) : activation et informations
  affichées aux adhérents.
- **Membre** (existant, 005) : porte au plus une adhésion en attente ou validée
  par saison ; ne peut être supprimé par son compte s'il en porte une.
- **Saison** (existante, 003) : seule une saison ouverte, non close, avec tarif
  accepte de nouvelles adhésions.
- **Tarif** (existant, 004) : choisi par l'adhésion ; non supprimable dès
  qu'une adhésion l'utilise.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Un titulaire de compte dépose une demande d'adhésion pour un de
  ses membres en moins de 2 minutes depuis la rubrique « Adhésions ».
- **SC-002**: Un administrateur traite (valide ou refuse) une demande en moins
  de 30 secondes depuis la liste des demandes en attente.
- **SC-003**: Aucune combinaison d'actions ne permet d'obtenir deux adhésions
  en attente ou validées pour un même membre et une même saison.
- **SC-004**: Pour 100 % des opérations, un compte n'obtient ni ne modifie
  jamais une adhésion d'une fiche qu'il ne gère pas, et les opérations
  d'administration sont refusées aux membres et aux personnes non connectées
  (vérifié par des tests automatisés).
- **SC-005**: Les chiffres d'une saison (adhésions validées, premières
  adhésions, renouvellements, montant total) correspondent à 100 % aux
  adhésions enregistrées.
- **SC-006**: Depuis le lien de l'e-mail, un titulaire de compte connaît le
  montant, la référence et les informations du moyen choisi en moins de
  1 minute, sans contacter le bureau.
- **SC-007**: Le montant encaissé et le montant restant à percevoir d'une
  saison correspondent à 100 % aux paiements constatés et aux adhésions
  validées non payées.
- **SC-008**: Aucun écran livré ne présente de violation d'accessibilité grave
  ou critique lors des contrôles automatisés et d'un parcours complet au
  clavier.

## Assumptions

- **Renouvellement déclaré** : l'association existait avant Frigg ;
  l'adhérent déclare s'il renouvelle, l'administrateur retient le type à la
  validation et Frigg ne fait que l'aider (décision du 2026-10-31).
- **Choix du tarif** : libre parmi tous les tarifs de la saison ; aucune
  catégorie ni condition d'éligibilité ; l'administrateur corrige le tarif à
  la validation (décision du 2026-11-01).
- **Paiement hors ligne** : pas de paiement en ligne, pas de paiement en
  plusieurs fois, pas de paiement partiel, pas de remboursement ni de reçu
  fiscal ; aucune relance automatique (le bureau relance à partir de la liste
  filtrée « À payer »). Le lien entre les adhésions payées et la comptabilité
  relèvera de la future fonctionnalité « Finances ».
- **Paramètres** : la rubrique « Paramètres » ne contient que les moyens de
  paiement dans cette fonctionnalité ; d'autres paramètres de l'association y
  seront ajoutés plus tard.
- **Pas de pièce justificative** : certificat médical, autorisation parentale,
  justificatif de tarif réduit et attestation d'adhésion sont hors périmètre.
- **Pas de formule ni d'activité** : une adhésion porte sur l'association pour
  la saison (004).
- **Pas d'e-mail aux administrateurs** à chaque demande : l'encart du tableau
  de bord et la liste suffisent.
- **Pas de modification d'une demande** par l'adhérent : il l'annule et en
  dépose une nouvelle.
- **Une page de liste affiche 25 adhésions**, comme pour les membres.
- **Historique** : pas de journal consultable dans l'interface ; la
  traçabilité passe par les dates et auteurs enregistrés sur l'adhésion et les
  journaux de l'application.
