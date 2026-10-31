# Feature Specification: Gestion des tarifs de l'adhésion

**Feature Branch**: `004-membership-fees`

**Created**: 2026-10-31

**Status**: Draft

**Input**: User description: "Gestion des tarifs de l'adhésion. Dans le détail d'une saison, on peut gérer les tarifs de l'adhésion (ces tarifs font partie de la duplication)"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Définir les tarifs d'une saison (Priority: P1)

Un administrateur vient de créer la saison 2026-2027. Depuis la fiche de la
saison, il consulte une section « Tarifs de l'adhésion » : aucun tarif n'existe
encore, la section explique qu'un tarif est le prix que paie un adhérent pour
adhérer pendant la saison et l'invite à en ajouter un. Il ajoute « Plein tarif »
à 30 €, puis « Tarif réduit » à 15 € avec la description « Étudiants et
demandeurs d'emploi ». Les tarifs apparaissent dans la fiche avec leur libellé,
leur montant et leur description.

**Why this priority**: sans tarif, aucune adhésion ne pourra être proposée ni
payée ; c'est la brique qui suit directement la création de la saison.

**Independent Test**: sur une saison en brouillon sans tarif, constater l'état
vide de la section, ajouter deux tarifs, les retrouver dans la fiche avec leurs
informations.

**Acceptance Scenarios**:

1. **Given** une saison sans tarif, **When** l'administrateur ouvre sa fiche,
   **Then** la section « Tarifs de l'adhésion » explique le rôle des tarifs et
   propose un bouton « Ajouter un tarif ».
2. **Given** le formulaire d'ajout, **When** l'administrateur saisit le libellé
   « Plein tarif » et le montant 30 € puis valide, **Then** le tarif apparaît
   dans la section et un message confirme l'action.
3. **Given** le formulaire d'ajout, **When** l'administrateur saisit une
   description facultative, **Then** elle s'affiche sous le libellé du tarif.
4. **Given** un tarif « Plein tarif » dans la saison, **When** l'administrateur
   ajoute un autre tarif nommé « plein TARIF », **Then** l'ajout est refusé et
   le champ libellé indique qu'un tarif porte déjà ce nom dans la saison.
5. **Given** un libellé vide, un montant négatif, un montant avec plus de deux
   décimales ou un montant supérieur au maximum, **When** l'administrateur
   valide, **Then** le tarif n'est pas créé et le champ concerné affiche
   l'erreur.
6. **Given** un montant de 0 €, **When** l'administrateur valide, **Then** le
   tarif est créé et affiché comme « Gratuit ».
7. **Given** plusieurs tarifs, **When** la fiche s'affiche, **Then** ils
   apparaissent du montant le plus élevé au plus faible, puis par libellé.
8. **Given** un membre connecté, **When** il tente de consulter, d'ajouter, de
   modifier ou de supprimer un tarif, **Then** l'accès est refusé.
9. **Given** le formulaire d'ajout, **When** l'administrateur saisit « Adulte
   compétition » à 85 €, coche « Tarif de renouvellement différent » et saisit
   55 €, **Then** le tarif affiche les deux montants (« 85,00 € » et
   « Renouvellement 55,00 € »).
10. **Given** la case « Tarif de renouvellement différent » cochée, **When**
    l'administrateur valide sans montant de renouvellement ou avec un montant
    invalide, **Then** le tarif n'est pas créé et le champ du montant de
    renouvellement affiche l'erreur.
11. **Given** la case décochée, **When** l'administrateur valide, **Then** le
    tarif n'a qu'un montant, valable pour toute adhésion.

---

### User Story 2 - Reprendre les tarifs en dupliquant la saison (Priority: P1)

En fin de saison, l'administrateur duplique la dernière saison pour préparer la
suivante. Les tarifs de la saison dupliquée sont recopiés dans la nouvelle
saison, à l'identique ; il n'a plus qu'à ajuster ceux qui changent.

**Why this priority**: c'est le geste annuel de l'administrateur ; ressaisir
chaque année des tarifs le plus souvent inchangés serait fastidieux et source
d'erreurs.

**Independent Test**: avec une saison 2026-2027 portant deux tarifs,
dupliquer ; vérifier que 2027-2028 porte les deux mêmes tarifs, puis modifier
un tarif de 2027-2028 et constater que celui de 2026-2027 est inchangé.

**Acceptance Scenarios**:

1. **Given** la dernière saison portant « Plein tarif » (30 €) et « Tarif
   réduit » (15 €, avec description), **When** l'administrateur la duplique,
   **Then** la nouvelle saison porte deux tarifs de mêmes libellés, montants et
   descriptions.
2. **Given** une saison dupliquée, **When** l'administrateur modifie ou
   supprime un tarif de la nouvelle saison, **Then** les tarifs de la saison
   d'origine sont inchangés, et inversement.
3. **Given** la dernière saison sans tarif, **When** l'administrateur la
   duplique, **Then** la nouvelle saison est créée sans tarif.
4. **Given** des saisons existantes, **When** l'administrateur crée une saison
   sans dupliquer, **Then** la nouvelle saison est créée sans tarif.

---

### User Story 3 - Modifier ou supprimer un tarif (Priority: P2)

L'administrateur s'est trompé de montant, veut reformuler une description ou
retirer un tarif devenu inutile. Il modifie ou supprime le tarif depuis la fiche
de la saison, tant que celle-ci n'est pas close.

**Why this priority**: les corrections sont courantes mais secondaires par
rapport à la saisie initiale et à la reprise d'une saison à l'autre.

**Independent Test**: sur une saison en brouillon, modifier le montant d'un
tarif puis en supprimer un autre ; sur une saison close, constater qu'aucune
action n'est proposée.

**Acceptance Scenarios**:

1. **Given** une saison en brouillon, **When** l'administrateur modifie le
   libellé, le montant, le montant de renouvellement ou la description d'un
   tarif, **Then** les nouvelles valeurs sont enregistrées et affichées ;
   décocher « Tarif de renouvellement différent » supprime le montant de
   renouvellement.
2. **Given** une saison ouverte, **When** l'administrateur modifie le libellé,
   le montant ou la description d'un tarif, **Then** les nouvelles valeurs
   sont enregistrées et affichées.
3. **Given** une saison en brouillon ou ouverte, **When** l'administrateur
   choisit « Supprimer » sur un tarif et confirme dans le dialogue, **Then** le
   tarif disparaît de la fiche, y compris s'il s'agissait du dernier.
4. **Given** une saison close, **When** l'administrateur consulte sa fiche,
   **Then** les tarifs sont affichés en lecture seule, sans action d'ajout, de
   modification ni de suppression.
5. **Given** un tarif supprimé ou une saison close entre-temps par un autre
   administrateur ou par le passage de la date, **When** l'administrateur
   valide une action sur ce tarif, **Then** l'action est refusée avec un
   message explicite et la fiche est rafraîchie.

---

### User Story 4 - Être prévenu qu'une saison sans tarif ne sera pas proposée (Priority: P3)

Une saison sans tarif ne sera pas proposée à l'adhésion. Rien n'empêche
l'administrateur de l'ouvrir, mais la fiche de la saison l'en informe pour
qu'il n'oublie pas d'ajouter les tarifs.

**Why this priority**: simple message d'information ; l'ouverture et la gestion
des tarifs fonctionnent sans lui.

**Independent Test**: sur une saison non close sans tarif, constater le
message ; ajouter un tarif, constater qu'il disparaît.

**Acceptance Scenarios**:

1. **Given** une saison en brouillon ou ouverte sans tarif, **When**
   l'administrateur consulte sa fiche, **Then** un message informatif indique
   que la saison ne sera pas proposée à l'adhésion tant qu'elle n'a aucun
   tarif.
2. **Given** la même saison après l'ajout d'un tarif, **When** l'administrateur
   consulte sa fiche, **Then** le message n'apparaît plus.
3. **Given** une saison en brouillon sans tarif dont la précédente n'est pas en
   brouillon, **When** l'administrateur consulte sa fiche, **Then** l'action
   « Ouvrir la saison » reste proposée comme auparavant.
4. **Given** une saison close sans tarif, **When** l'administrateur consulte
   sa fiche, **Then** seul l'état vide de la section s'affiche, sans le
   message informatif.

---

### Edge Cases

- **Saison close créée pour combler un trou** : elle est close dès sa
  création ; si elle provient d'une duplication, elle porte les tarifs
  recopiés, en lecture seule ; sinon elle reste sans tarif.
- **Saison supprimée** : la suppression d'une saison (dernière, jamais ouverte)
  supprime ses tarifs avec elle.
- **Clôture pendant la saisie** : si la saison devient close (minuit passé)
  pendant que le formulaire est ouvert, l'enregistrement est refusé avec un
  message explicite.
- **Deux administrateurs simultanés** : si deux administrateurs ajoutent un
  tarif de même libellé en même temps, un seul est créé ; l'autre reçoit
  l'erreur d'unicité sur le champ libellé.
- **Libellés proches** : l'unicité ignore la casse, les accents, les espaces
  en début et fin et les espaces répétés (« Tarif réduit », « tarif reduit »,
  « Tarif réduit  » et « Tarif  réduit » sont le même libellé).
- **Montants** : saisie en euros avec au plus deux décimales, virgule ou point
  accepté comme séparateur ; affichage au format français (« 15,50 € »).
- **Nombre de tarifs** : une saison porte au plus 20 tarifs ; au-delà, l'ajout
  est refusé avec un message.

## Requirements *(mandatory)*

### Functional Requirements

**Tarif**

- **FR-001**: Un tarif MUST appartenir à une seule saison et être défini par un
  libellé (obligatoire, 1 à 80 caractères), un montant en euros (obligatoire)
  et une description (facultative, 500 caractères au plus).
- **FR-002**: Le montant MUST être compris entre 0 € et 10 000 € inclus, avec
  au plus deux décimales ; un montant de 0 € MUST être affiché « Gratuit ».
- **FR-002a**: Un tarif MAY porter un montant de renouvellement, distinct du
  montant de première adhésion, activé par la case « Tarif de renouvellement
  différent ». Case cochée, ce montant est obligatoire et suit les règles de
  FR-002 ; aucune relation n'est imposée avec le montant principal. Case
  décochée, le montant principal vaut pour toute adhésion. Le tarif MUST
  afficher les deux montants quand il en a deux.
- **FR-003**: Le libellé MUST être unique au sein d'une saison, sans tenir
  compte de la casse, des accents, des espaces en début et fin ni des espaces
  répétés ; deux saisons
  différentes peuvent porter des tarifs de même libellé.
- **FR-004**: Une saison MUST porter au plus 20 tarifs.

**Gestion**

- **FR-005**: La fiche d'une saison MUST présenter une section « Tarifs de
  l'adhésion » listant ses tarifs du montant le plus élevé au plus faible, puis
  par libellé, avec libellé, montant et description.
- **FR-006**: Sans tarif, la section MUST afficher un état vide expliquant le
  rôle des tarifs et, si la saison n'est pas close, proposer d'en ajouter un.
- **FR-007**: Les administrateurs MUST pouvoir ajouter, modifier et supprimer
  les tarifs d'une saison en brouillon.
- **FR-008**: Les administrateurs MUST pouvoir ajouter, modifier (libellé,
  montants et description) et supprimer les tarifs d'une saison ouverte, y
  compris son dernier tarif.
- **FR-009**: Les tarifs d'une saison close MUST être affichés en lecture
  seule ; aucun ajout, aucune modification ni suppression n'est admis.
- **FR-010**: La suppression d'un tarif MUST passer par un dialogue de
  confirmation.
- **FR-011**: Toutes les opérations sur les tarifs MUST être réservées aux
  administrateurs.

**Duplication et cycle de vie de la saison**

- **FR-012**: Dupliquer une saison MUST recopier tous ses tarifs (libellé,
  montant, montant de renouvellement, description) dans la nouvelle saison,
  qui en devient propriétaire :
  les tarifs des deux saisons évoluent ensuite indépendamment.
- **FR-013**: Une saison créée sans duplication MUST être créée sans tarif.
- **FR-014**: Supprimer une saison MUST supprimer ses tarifs.
- **FR-015**: L'absence de tarif MUST NOT empêcher l'ouverture d'une saison ;
  les règles d'ouverture de la gestion des saisons (003) sont inchangées.
- **FR-016**: La fiche d'une saison en brouillon ou ouverte sans tarif MUST
  afficher un message informatif indiquant qu'elle ne sera pas proposée à
  l'adhésion tant qu'elle n'a aucun tarif.

**Saisie**

- **FR-017**: Chaque erreur de saisie MUST être affichée sur le champ concerné
  avec un message en français expliquant comment la corriger.
- **FR-018**: Une action sur un tarif ou une saison qui n'existe plus ou dont
  l'état ne permet plus l'action MUST être refusée avec un message explicite.

### Key Entities

- **Tarif de l'adhésion** : prix d'une adhésion pour une saison donnée.
  Attributs : saison, libellé, montant en euros, montant de renouvellement
  facultatif, description facultative. Les adhésions livrées plus tard
  choisiront un tarif de leur saison.
- **Saison** (existante, voir 003) : porte de zéro à vingt tarifs ; sa
  duplication recopie ses tarifs ; son état détermine si ses tarifs sont
  modifiables.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Un administrateur ajoute un tarif à une saison en moins de
  30 secondes depuis la fiche de la saison.
- **SC-002**: Après duplication, 100 % des tarifs de la saison d'origine sont
  présents à l'identique dans la nouvelle saison, sans ressaisie.
- **SC-003**: Aucune combinaison d'actions ne permet de modifier un tarif d'une
  saison close, ni d'obtenir deux tarifs de même libellé dans une saison.
- **SC-004**: Pour toute opération sur les tarifs, un accès anonyme ou par un
  membre est refusé dans 100 % des cas.
- **SC-005**: La section des tarifs et ses formulaires sont utilisables au
  clavier et ne présentent aucune violation d'accessibilité grave ou critique.

## Assumptions

- **Périmètre** : un tarif est un prix simple. Conditions d'éligibilité (âge,
  statut, famille), calcul automatique du tarif applicable, paiement en
  plusieurs fois, codes promo et adhésions elles-mêmes sont hors de cette
  fonctionnalité.
- **Renouvellement** : l'association peut exister avant l'installation de
  Frigg, dont l'historique ne permet donc pas de savoir si une adhésion est un
  renouvellement. Lors de son adhésion (fonctionnalité ultérieure), l'adhérent
  déclarera s'il renouvelle et un administrateur validera la demande ;
  l'adhésion conservera le type retenu (renouvellement ou première adhésion),
  pour les statistiques, et le montant appliqué. Décision du 2026-10-31.
- **Un seul type d'adhésion** : les tarifs s'appliquent à l'adhésion à
  l'association pour la saison ; pas de notion de formule ou d'activité.
- **Membres** : les membres ne voient pas les tarifs dans cette
  fonctionnalité ; ils les découvriront avec la fonctionnalité d'adhésion.
- **Pas d'adhésion existante** : aucune adhésion n'existe encore, la
  suppression d'un tarif n'a donc pas à vérifier de rattachement. Quand les
  adhésions existeront, elles conserveront le montant payé, indépendamment des
  modifications ultérieures du tarif.
- **Saison sans tarif** : elle ne sera pas proposée à l'adhésion (règle
  appliquée par la future fonctionnalité d'adhésion) ; ici, seul un message
  informatif sur la fiche le rappelle. Pas d'encart sur le tableau de bord.
- **Ordre d'affichage** : par montant décroissant puis libellé ; pas de
  réordonnancement manuel.
- **Devise** : euro uniquement (association française).
- **Historique** : pas de journal des modifications des tarifs.
