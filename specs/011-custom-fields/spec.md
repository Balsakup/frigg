# Feature Specification: Champs personnalisés des membres et des adhésions

**Feature Branch**: `011-custom-fields`

**Created**: 2026-11-05

**Status**: Draft

**Input**: User description: "Champs personnalisés pour les membres et les adhésions. Exemple pour les membres : Numéro de licence fédérale. Exemple pour les adhésions : A payé sa licence fédérale"

## Vocabulaire

- **Champ personnalisé** : information supplémentaire, définie par un
  administrateur, que Frigg ne prévoit pas d'origine. Il s'ajoute soit aux
  **fiches membres** (005), soit aux **adhésions** (006).
- **Champ de membre** : valeur attachée à la personne, valable d'une saison à
  l'autre (par exemple « Numéro de licence fédérale »).
- **Champ d'adhésion** : valeur attachée à une adhésion, donc propre à une
  saison (par exemple « A payé sa licence fédérale ») ; elle repart vide à
  chaque nouvelle adhésion.
- **Portée** d'un champ : ce à quoi il s'applique, les fiches membres ou les
  adhésions. Elle est choisie à la création et ne change plus.
- **Type** d'un champ : la forme de la valeur attendue — texte court, texte
  long, nombre, date, oui/non, liste de choix (une seule valeur parmi une liste
  définie par l'administrateur).
- **Visibilité** d'un champ, réglée par l'administrateur :
  - **Réservé à l'administration** : ni visible ni modifiable depuis l'espace
    membre (par exemple « A payé sa licence fédérale ») ;
  - **Visible par le membre** : affiché en lecture seule au compte qui gère la
    fiche, rempli par un administrateur ;
  - **Rempli par le membre** : affiché et modifiable par le compte qui gère la
    fiche (par exemple « Numéro de licence fédérale »).
- **Champ archivé** : champ retiré des formulaires et des fiches sans perte
  des valeurs déjà saisies ; il peut être rétabli.
- **Section « Champs personnalisés »** : nouvelle section de la rubrique
  « Paramètres » de l'espace administration, à côté des sections existantes.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Définir des champs personnalisés (administration) (Priority: P1)

Un administrateur ouvre « Paramètres », section « Champs personnalisés ». La
section présente deux listes, « Membres » et « Adhésions », vides au départ,
avec une explication et des exemples. Il ajoute au membre un champ « Numéro de
licence fédérale » de type texte court, rempli par le membre, avec le texte
d'aide « Figure sur votre licence, 7 chiffres ». Il ajoute à l'adhésion un champ
« A payé sa licence fédérale » de type oui/non, réservé à l'administration. Il
réordonne ensuite les champs, corrige un libellé et archive un champ devenu
inutile.

**Why this priority**: sans définition de champ, rien d'autre n'existe ; c'est
le socle de la fonctionnalité.

**Independent Test**: créer un champ de chaque type pour les membres et pour
les adhésions, en modifier un, en réordonner deux, en archiver un, recharger la
section et retrouver l'état attendu.

**Acceptance Scenarios**:

1. **Given** la section « Champs personnalisés » sans champ, **When** un
   administrateur l'ouvre, **Then** un état vide explique la différence entre
   champ de membre et champ d'adhésion, avec les exemples de la licence
   fédérale.
2. **Given** le formulaire d'ajout d'un champ de membre, **When**
   l'administrateur saisit un libellé, choisit un type, une visibilité, coche
   éventuellement « Obligatoire », saisit un texte d'aide facultatif et valide,
   **Then** le champ apparaît en fin de liste et un message confirme l'action.
3. **Given** le type « Liste de choix », **When** l'administrateur valide sans
   au moins deux choix distincts, **Then** l'enregistrement est refusé avec un
   message sur la liste des choix.
4. **Given** un libellé déjà utilisé par un autre champ de la même portée
   (membres ou adhésions), à la casse et aux accents près, **When**
   l'administrateur enregistre, **Then** l'enregistrement est refusé avec un
   message sur le libellé.
5. **Given** plusieurs champs, **When** l'administrateur déplace un champ vers
   le haut ou vers le bas (à la souris ou au clavier), **Then** le nouvel ordre
   est conservé et repris dans tous les formulaires et fiches.
6. **Given** un champ qui porte déjà des valeurs, **When** l'administrateur le
   modifie, **Then** il peut changer le libellé, le texte d'aide, la
   visibilité, le caractère obligatoire et ajouter ou renommer des choix, mais
   pas changer le type.
7. **Given** un champ, **When** l'administrateur l'archive, **Then** il
   disparaît des formulaires et des fiches, reste listé dans une partie
   « Archivés » de la section, et ses valeurs réapparaissent intactes s'il est
   rétabli.
8. **Given** un membre (non administrateur), **When** il tente d'ouvrir la
   section ou de créer, modifier, archiver ou supprimer un champ, **Then**
   l'accès est refusé.

---

### User Story 2 - Renseigner les champs de membre (Priority: P1)

Un administrateur ouvre la fiche de Lucas et saisit son numéro de licence
fédérale dans une partie « Informations complémentaires ». Claire, qui gère la
fiche de son fils Lucas, ouvre cette fiche depuis « Mes membres » : elle voit
le numéro et peut le corriger, car le champ est rempli par le membre. Elle ne
voit aucun champ réservé à l'administration.

**Why this priority**: c'est l'exemple principal de la demande (numéro de
licence) et l'usage le plus fréquent : compléter la fiche d'une personne avec
une information propre à l'association.

**Independent Test**: avec un champ de chaque visibilité, renseigner les
valeurs depuis l'administration, puis ouvrir la fiche depuis le compte qui la
gère et vérifier ce qui est affiché, modifiable ou absent.

**Acceptance Scenarios**:

1. **Given** des champs de membre actifs, **When** un administrateur crée ou
   modifie une fiche, **Then** les champs apparaissent dans l'ordre défini,
   sous les informations prévues par Frigg, avec leur texte d'aide.
2. **Given** un champ « Rempli par le membre », **When** le compte gestionnaire
   crée ou modifie la fiche depuis « Mes membres », **Then** il peut saisir et
   enregistrer la valeur.
3. **Given** un champ « Visible par le membre », **When** le compte
   gestionnaire ouvre la fiche, **Then** il voit la valeur en lecture seule ;
   toute tentative de la modifier est refusée par le serveur.
4. **Given** un champ « Réservé à l'administration », **When** le compte
   gestionnaire consulte ou modifie la fiche, **Then** ni le libellé ni la
   valeur ne lui sont transmis, et une tentative d'écriture est refusée.
5. **Given** une valeur qui ne respecte pas le type (texte dans un nombre, date
   invalide, choix inexistant) ou une longueur maximale dépassée, **When**
   l'enregistrement est demandé, **Then** il est refusé avec un message sur le
   champ concerné.
6. **Given** un champ obligatoire que la personne qui enregistre a le droit de
   remplir, **When** elle enregistre la fiche sans valeur, **Then**
   l'enregistrement est refusé avec un message sur le champ.
7. **Given** un champ obligatoire réservé à l'administration, **When** le
   compte gestionnaire enregistre la fiche, **Then** l'absence de valeur ne
   bloque pas son enregistrement.

---

### User Story 3 - Renseigner les champs d'adhésion (Priority: P1)

À la validation de la demande d'adhésion de Lucas pour 2026-2027,
l'administrateur coche « A payé sa licence fédérale » s'il a reçu le paiement,
ou y revient plus tard depuis l'adhésion. Lorsque l'association définit un champ
d'adhésion rempli par le membre (par exemple « Taille de t-shirt », liste de
choix), Claire le renseigne en déposant la demande.

**Why this priority**: c'est le second exemple de la demande ; une information
qui vaut pour une saison seulement ne peut pas être portée par la fiche.

**Independent Test**: définir un champ d'adhésion réservé à l'administration et
un champ rempli par le membre ; déposer une demande depuis l'espace membre,
la valider en renseignant le champ réservé, puis le modifier depuis
l'adhésion validée.

**Acceptance Scenarios**:

1. **Given** un champ d'adhésion « Rempli par le membre », **When** un compte
   dépose une demande d'adhésion, **Then** le formulaire de demande propose le
   champ et enregistre la valeur avec la demande.
2. **Given** une demande en attente, **When** un administrateur l'ouvre pour la
   valider, **Then** il voit et peut modifier tous les champs d'adhésion avant
   de valider.
3. **Given** une adhésion validée, **When** un administrateur modifie un champ
   d'adhésion, **Then** la valeur est enregistrée sans changer l'état de
   l'adhésion ni son paiement.
4. **Given** un membre qui a une adhésion pour 2025-2026 avec « A payé sa
   licence fédérale » à « Oui », **When** une adhésion pour 2026-2027 est
   créée, **Then** le champ de la nouvelle adhésion part vide (« Non »).
5. **Given** une adhésion annulée ou refusée, **When** un administrateur la
   consulte, **Then** les valeurs des champs d'adhésion restent affichées en
   lecture seule.
6. **Given** un champ obligatoire réservé à l'administration, **When** un
   administrateur valide une demande ou crée une adhésion directement sans
   valeur, **Then** l'action est refusée avec un message sur le champ.
7. **Given** le compte gestionnaire, **When** il consulte une adhésion de ses
   fiches, **Then** les champs d'adhésion lui sont présentés selon leur
   visibilité, comme pour les champs de membre.

---

### User Story 4 - Retrouver les membres et adhésions selon un champ (administration) (Priority: P2)

En début de saison, le trésorier veut relancer les adhérents qui n'ont pas
payé leur licence. Dans la rubrique « Adhésions », il filtre la saison
2026-2027 sur « A payé sa licence fédérale : Non » et obtient la liste. Dans la
rubrique « Membres », il retrouve Lucas en tapant une partie de son numéro de
licence dans la recherche.

**Why this priority**: une information saisie mais impossible à exploiter perd
la moitié de sa valeur ; le suivi de la licence fédérale est précisément un
travail de liste.

**Independent Test**: renseigner des valeurs variées sur plusieurs membres et
adhésions, appliquer chaque filtre et la recherche, et comparer les résultats
au jeu de données.

**Acceptance Scenarios**:

1. **Given** un champ oui/non ou liste de choix actif, **When** un
   administrateur ouvre la liste correspondante (membres ou adhésions), **Then**
   un filtre par valeur de ce champ est proposé, avec l'option « Non
   renseigné » pour les listes de choix.
2. **Given** un filtre sur un champ d'adhésion combiné au filtre de saison,
   **When** il est appliqué, **Then** seules les adhésions de la saison qui
   ont cette valeur sont listées, et le nombre de résultats est annoncé.
3. **Given** un champ de membre de type texte court, **When** un administrateur
   saisit dans la recherche des membres au moins 3 caractères contenus dans sa
   valeur, à la casse et aux accents près, **Then** le membre est trouvé.

---

### User Story 5 - Supprimer définitivement un champ (administration) (Priority: P3)

Un administrateur a créé par erreur un champ « Licence » en double. Il le
supprime : un dialogue de confirmation indique combien de fiches ou
d'adhésions portent une valeur pour ce champ et prévient que ces valeurs seront
perdues.

**Why this priority**: l'archivage couvre l'essentiel des besoins ; la
suppression sert au nettoyage des erreurs.

**Independent Test**: créer un champ, renseigner quelques valeurs, le
supprimer en confirmant, puis vérifier qu'il ne figure plus nulle part et que
ses valeurs ont disparu.

**Acceptance Scenarios**:

1. **Given** un champ portant 12 valeurs, **When** l'administrateur demande sa
   suppression, **Then** un dialogue de confirmation annonce les 12 valeurs qui
   seront perdues et propose l'archivage comme alternative.
2. **Given** le dialogue de confirmation, **When** l'administrateur confirme,
   **Then** le champ et toutes ses valeurs sont supprimés et un message confirme
   l'action.
3. **Given** un champ sans valeur, **When** l'administrateur le supprime,
   **Then** la confirmation est demandée sans mention de valeurs perdues.

---

### Edge Cases

- **Champ obligatoire ajouté après coup** : les fiches et adhésions existantes
  restent valides ; la valeur n'est exigée qu'au prochain enregistrement du
  formulaire par une personne qui a le droit de la remplir. La section
  signale combien de fiches, ou d'adhésions en attente ou validées de la saison
  en cours, n'ont pas encore de valeur.
- **Oui/non obligatoire** : un champ oui/non vaut « Non » tant qu'il n'est pas
  coché ; l'option « Obligatoire » n'est pas proposée pour ce type.
- **Choix renommé** : toutes les valeurs qui le portaient affichent le nouveau
  libellé.
- **Choix retiré** : un choix utilisé par au moins une valeur ne peut pas être
  retiré ; un choix inutilisé peut l'être.
- **Visibilité restreinte après coup** : passer un champ de « Rempli par le
  membre » à « Réservé à l'administration » conserve les valeurs déjà saisies
  par les membres, qui cessent immédiatement de les voir.
- **Champ archivé** : ses valeurs ne sont ni affichées, ni modifiables, ni
  utilisables en filtre ou en recherche, mais restent conservées ; un champ
  obligatoire archivé n'est plus exigé.
- **Fiche ou adhésion supprimée** : ses valeurs de champs personnalisés sont
  supprimées avec elle.
- **Nombre** : accepte les nombres décimaux saisis avec une virgule ou un
  point ; il est affiché au format français.
- **Valeur vidée** : effacer la valeur d'un champ facultatif la retire ; la
  fiche est alors considérée « Non renseigné » pour ce champ.
- **Modifications concurrentes** : deux administrateurs qui modifient en même
  temps la définition d'un champ ou une même fiche : la dernière modification
  enregistrée prévaut, comme pour les autres paramètres et fiches. Seule
  exception : un nouvel ordre des champs est refusé si la liste a changé
  entre-temps (champ ajouté, archivé ou déplacé par un autre administrateur),
  avec un message invitant à recharger la page, pour ne pas écraser un ordre
  que l'administrateur n'a pas vu.
- **Limite atteinte** : au-delà de 30 champs actifs pour les membres (ou pour
  les adhésions), l'ajout est refusé avec un message invitant à archiver un
  champ inutile.

## Requirements *(mandatory)*

### Functional Requirements

**Définition des champs**

- **FR-001**: L'administration MUST offrir, dans la rubrique « Paramètres »,
  une section « Champs personnalisés » présentant séparément les champs de
  membre et les champs d'adhésion, actifs puis archivés.
- **FR-002**: Un champ MUST comporter : un libellé (obligatoire, 1 à 80
  caractères, unique dans sa portée à la casse et aux accents près), un
  type, une visibilité, un caractère obligatoire ou facultatif, un texte d'aide
  facultatif (au plus 200 caractères) et une position dans l'ordre d'affichage.
- **FR-003**: Les types disponibles MUST être : texte court (au plus 200
  caractères), texte long (au plus 2 000 caractères), nombre, date, oui/non et
  liste de choix (2 à 50 choix distincts de 1 à 80 caractères, une seule
  valeur retenue).
- **FR-004**: Les visibilités disponibles MUST être « Réservé à
  l'administration », « Visible par le membre » et « Rempli par le membre » ;
  un nouveau champ est « Réservé à l'administration » par défaut.
- **FR-005**: Le type d'un champ MUST NOT pouvoir être changé dès qu'une valeur
  existe pour ce champ ; les autres attributs restent modifiables.
- **FR-006**: Un choix MUST pouvoir être ajouté ou renommé à tout moment ; un
  choix utilisé par au moins une valeur MUST NOT pouvoir être retiré.
- **FR-007**: Les administrateurs MUST pouvoir réordonner les champs, à la
  souris et au clavier.
- **FR-008**: Les administrateurs MUST pouvoir archiver un champ et le
  rétablir ; l'archivage conserve toutes ses valeurs.
- **FR-009**: Les administrateurs MUST pouvoir supprimer un champ, après un
  dialogue de confirmation qui annonce le nombre de valeurs perdues ; la
  suppression efface ces valeurs.
- **FR-010**: Le système MUST refuser l'ajout d'un 31e champ actif dans une même
  portée (membres ou adhésions).

**Saisie et affichage des valeurs**

- **FR-011**: Les fiches membres (consultation, création, modification) MUST
  présenter les champs de membre actifs autorisés à la personne connectée, dans
  l'ordre défini, dans une partie « Informations complémentaires ».
- **FR-012**: Les adhésions (demande, validation, création directe,
  consultation, correction) MUST présenter de même les champs d'adhésion
  actifs autorisés à la personne connectée.
- **FR-013**: Une valeur de champ de membre MUST être attachée à la fiche et
  conservée d'une saison à l'autre ; une valeur de champ d'adhésion MUST être
  attachée à une seule adhésion et ne jamais être recopiée sur une autre.
- **FR-014**: Chaque valeur MUST être validée selon le type et les limites de
  son champ avant d'être enregistrée.
- **FR-015**: Un champ obligatoire MUST être exigé seulement de la personne qui
  a le droit de le remplir, au moment où elle enregistre le formulaire : le
  compte gestionnaire pour un champ « Rempli par le membre », un administrateur
  pour tout champ. Pour un champ d'adhésion réservé à l'administration ou
  visible par le membre, l'exigence s'applique à la validation, à la création
  directe et à la correction d'une adhésion.
- **FR-016**: Les valeurs des adhésions annulées ou refusées MUST rester
  consultables par les administrateurs, sans pouvoir être modifiées.

**Visibilité et autorisations**

- **FR-017**: La définition des champs (création, modification, ordre,
  archivage, suppression) MUST être réservée aux administrateurs.
- **FR-018**: Les administrateurs MUST pouvoir lire et modifier tous les
  champs actifs de toute fiche et de toute adhésion modifiable.
- **FR-019**: Le compte qui gère une fiche MUST voir les champs « Visible par
  le membre » et « Rempli par le membre » de cette fiche et de ses adhésions,
  et ne pouvoir modifier que les champs « Rempli par le membre ».
- **FR-020**: Un champ « Réservé à l'administration » (libellé, définition et
  valeurs) MUST NOT être transmis à un compte qui n'est pas administrateur.
- **FR-021**: Toute tentative d'écriture d'une valeur non autorisée MUST être
  refusée par le serveur, quel que soit ce que présente l'interface.
- **FR-022**: Un compte MUST NOT pouvoir lire ni modifier les valeurs des
  fiches et adhésions qu'il ne gère pas (règle de 005 et 006 étendue aux
  champs personnalisés).

**Exploitation**

- **FR-023**: Les listes « Membres » et « Adhésions » de l'administration MUST
  proposer un filtre par valeur pour chaque champ actif de type oui/non ou
  liste de choix de la portée correspondante (avec « Non renseigné » pour les
  listes de choix), combinable avec les filtres existants.
- **FR-024**: La recherche des membres de l'administration MUST trouver un
  membre à partir d'au moins 3 caractères contenus dans la valeur d'un champ de
  membre actif de type texte court, à la casse et aux accents près.
- **FR-025**: Les champs archivés MUST NOT être proposés en filtre ni pris en
  compte par la recherche.
- **FR-026**: La section MUST indiquer, pour chaque champ obligatoire actif,
  le nombre de fiches membres (toutes les fiches) ou, pour un champ
  d'adhésion, le nombre d'adhésions en attente ou validées de la saison en
  cours, qui n'ont pas encore de valeur. Sans saison en cours, aucun nombre
  n'est affiché pour les champs d'adhésion.

**Transverse**

- **FR-027**: La création, la modification, l'archivage, le rétablissement et
  la suppression d'un champ MUST être enregistrés dans les journaux de
  l'application (auteur, champ, action), sans les valeurs saisies.
- **FR-028**: Chaque erreur de saisie MUST être affichée sur le champ concerné ;
  après chaque action, un message de confirmation est annoncé aux technologies
  d'assistance.
- **FR-029**: Les écrans MUST être en français, utilisables au clavier seul,
  dès 320 px de large, et conformes aux exigences d'accessibilité du projet ;
  chaque champ personnalisé porte son libellé et son texte d'aide associés de
  façon accessible.

### Key Entities

- **Définition de champ** : décrit un champ personnalisé — portée (membres
  ou adhésions), libellé, type, visibilité, obligatoire, texte d'aide, position,
  état (actif ou archivé). Unique pour l'instance (principe I).
- **Choix** : une valeur proposée par un champ de type liste de choix —
  libellé, position. Appartient à une définition.
- **Valeur de champ de membre** : la valeur d'une définition « membres » pour
  une fiche membre ; au plus une par couple fiche–champ.
- **Valeur de champ d'adhésion** : la valeur d'une définition « adhésions »
  pour une adhésion ; au plus une par couple adhésion–champ.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Un administrateur crée un nouveau champ personnalisé et le voit
  apparaître dans les fiches en moins d'une minute, sans intervention
  technique.
- **SC-002**: Le trésorier obtient la liste des adhérents d'une saison qui
  n'ont pas payé leur licence fédérale en moins de 30 secondes et en au plus
  3 actions depuis la rubrique « Adhésions ».
- **SC-003**: 100 % des tentatives d'un compte non administrateur de lire ou
  d'écrire un champ réservé à l'administration échouent, vérifiées par des
  tests automatisés.
- **SC-004**: Un membre est retrouvé à partir d'un fragment de son numéro de
  licence en moins de 2 secondes, sur une association de 2 000 membres.
- **SC-005**: Archiver puis rétablir un champ restitue 100 % des valeurs
  saisies.
- **SC-006**: L'association n'a plus besoin de tenir un tableur à côté de Frigg
  pour suivre les numéros et le paiement des licences fédérales.

## Assumptions

- **Portée** : les champs personnalisés s'appliquent aux fiches membres et aux
  adhésions uniquement ; ni les comptes, ni les saisons, ni les tarifs n'en
  reçoivent.
- **Champs d'adhésion pour toutes les saisons** : un champ d'adhésion vaut pour
  toutes les saisons et tous les tarifs ; le restreindre à une saison ou à un
  tarif est hors périmètre.
- **Pas de pré-remplissage** : une nouvelle adhésion (y compris un
  renouvellement) ne reprend pas les valeurs de l'adhésion précédente ; une
  information durable relève d'un champ de membre.
- **Pas d'unicité des valeurs** : deux fiches peuvent porter le même numéro de
  licence ; contrôler l'unicité ou un format particulier (expression régulière)
  est hors périmètre.
- **Pas de champ fichier** : les pièces jointes (certificat médical…) relèvent
  d'une future fonctionnalité de documents.
- **Hors périmètre de cette version** : colonnes personnalisées dans les
  tableaux, filtres sur les nombres et les dates, recherche dans les textes
  longs, utilisation des champs comme critère de destinataires des envois
  groupés (009) ou comme variables dans les e-mails.
- **Export RGPD** : la future spécification « Export des données
  personnelles » devra inclure les valeurs des champs personnalisés ; les
  valeurs « Réservé à l'administration » d'une personne font partie de ses
  données.
- **Dépendances** : 005 (fiches membres), 006 (adhésions) et 010 (rubrique
  « Paramètres ») sont livrées et réutilisées.
