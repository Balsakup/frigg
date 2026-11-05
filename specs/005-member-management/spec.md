# Feature Specification: Gestion des membres

**Feature Branch**: `005-member-management`

**Created**: 2026-11-01

**Status**: Draft

**Input**: User description: "Gestion des membres (Admin et Membre), l'admin peut gérer tous les membres, un membre ne peut gérer que les siens"

## Vocabulaire

- **Compte** : l'accès à Frigg d'une personne (adresse électronique, mot de
  passe, rôle membre ou administrateur), livré par 001 et 002.
- **Membre** : une personne de l'association, décrite par une fiche (identité,
  coordonnées). Un membre n'a pas forcément de compte : un parent gère par
  exemple la fiche de ses enfants depuis son propre compte.
- **Membres d'un compte** (« Mes membres ») : les fiches gérées par ce compte.
  Le titulaire du compte peut y figurer lui-même s'il est adhérent.

Le « rôle membre » d'un compte et la « fiche membre » sont deux notions
distinctes : cette fonctionnalité porte sur les fiches.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Gérer mes membres depuis mon compte (Priority: P1)

Claire, titulaire d'un compte membre, ouvre la rubrique « Mes membres ». Elle
n'a encore aucune fiche : la page explique qu'un membre est une personne
qu'elle inscrit à l'association (elle-même, ses enfants…) et l'invite à en
ajouter un. Elle crée sa propre fiche, puis celles de ses deux enfants. Elle
retrouve ensuite ses trois membres dans la rubrique, ouvre la fiche de son fils
pour corriger son numéro de téléphone, et supprime une fiche créée par erreur.

**Why this priority**: c'est le point d'entrée de toute adhésion future ; sans
fiche, aucune personne ne pourra adhérer. Le parcours membre est le plus
fréquent (chaque famille) et apporte de la valeur seul.

**Independent Test**: avec un compte membre sans fiche, constater l'état vide,
créer trois fiches, en modifier une, en supprimer une, et vérifier que la
rubrique reflète chaque action.

**Acceptance Scenarios**:

1. **Given** un compte sans membre, **When** son titulaire ouvre « Mes
   membres », **Then** un état vide explique la notion de membre et propose
   « Ajouter un membre ».
2. **Given** le formulaire d'ajout, **When** le titulaire saisit prénom, nom et
   date de naissance puis valide, **Then** la fiche est créée, rattachée à son
   compte, et un message confirme l'action.
3. **Given** le formulaire d'ajout, **When** le titulaire coche « Il s'agit de
   moi », **Then** l'adresse électronique du compte est proposée pour la fiche,
   et la fiche est signalée « Vous » dans la liste.
4. **Given** un compte dont une fiche est déjà signalée « Vous », **When** le
   titulaire crée une autre fiche, **Then** la case « Il s'agit de moi »
   n'est pas proposée et le serveur refuse une seconde fiche « Vous ».
5. **Given** plusieurs membres, **When** la rubrique s'affiche, **Then** chaque
   membre apparaît avec ses prénom, nom et âge, la fiche « Vous » en premier,
   puis les autres par prénom.
6. **Given** une de ses fiches, **When** le titulaire modifie un champ et
   valide, **Then** la nouvelle valeur est enregistrée et affichée.
7. **Given** une de ses fiches, **When** le titulaire choisit « Supprimer » et
   confirme dans le dialogue, **Then** la fiche disparaît de la rubrique.
8. **Given** un prénom ou un nom vide, une date de naissance future ou
   antérieure à 120 ans, une adresse électronique ou un téléphone mal formé,
   **When** le titulaire valide, **Then** la fiche n'est pas enregistrée et le
   champ concerné affiche l'erreur.

---

### User Story 2 - Ne jamais voir ni modifier les membres d'un autre compte (Priority: P1)

Claire et Marc ont chacun un compte membre. Claire ne voit que ses fiches ;
même en saisissant l'adresse de la fiche d'un enfant de Marc, elle n'y accède
pas et ne peut ni la modifier ni la supprimer.

**Why this priority**: les fiches contiennent des données personnelles, dont
celles de mineurs ; un cloisonnement défaillant serait une fuite de données
(constitution, principes II et VII). Ce cloisonnement conditionne la mise en
service de US1.

**Independent Test**: créer une fiche avec le compte A, puis, connecté avec le
compte B, tenter de la lister, de l'ouvrir, de la modifier et de la supprimer
par l'interface et par des requêtes directes.

**Acceptance Scenarios**:

1. **Given** deux comptes membres A et B ayant chacun des fiches, **When** A
   ouvre « Mes membres », **Then** seules ses fiches apparaissent.
2. **Given** la fiche d'un membre de B, **When** A tente de l'ouvrir, de la
   modifier ou de la supprimer, quel que soit le moyen, **Then** la réponse
   est identique à celle d'une fiche inexistante (« Page introuvable »).
3. **Given** un membre connecté, **When** il tente d'accéder à la rubrique
   « Membres » de l'administration, **Then** l'accès est refusé.
4. **Given** un administrateur dans l'espace membre, **When** il ouvre « Mes
   membres », **Then** il ne voit que les fiches de son propre compte, comme
   tout membre.

---

### User Story 3 - Retrouver et consulter tous les membres (administration) (Priority: P1)

Dans l'espace administration, un administrateur ouvre la rubrique « Membres ».
Il voit la liste de tous les membres de l'association avec prénom, nom, âge et
compte de rattachement. Il recherche « dupo » pour retrouver les Dupont, ouvre
une fiche et voit l'ensemble de ses informations ainsi que le compte qui la
gère.

**Why this priority**: le bureau doit pouvoir répondre à toute question sur un
membre (contact, identité) ; c'est la base des futures validations
d'adhésion.

**Independent Test**: avec des fiches réparties sur plusieurs comptes, ouvrir
la rubrique, rechercher un fragment de nom, paginer, ouvrir une fiche.

**Acceptance Scenarios**:

1. **Given** un administrateur dans la rubrique « Membres », **When** la page
   s'affiche, **Then** tous les membres sont listés par nom puis prénom, avec
   âge et adresse électronique du compte de rattachement (« Sans compte » le
   cas échéant).
2. **Given** la liste, **When** l'administrateur saisit un fragment d'au moins
   3 caractères du prénom, du nom, de l'adresse électronique ou du téléphone
   d'un membre, **Then** seuls les membres correspondants s'affichent, sans
   tenir compte de la casse ni des accents ; en dessous de 3 caractères, une
   indication rappelle le minimum et la liste reste entière.
3. **Given** plus de membres qu'une page n'en affiche, **When** l'administrateur
   parcourt la liste, **Then** il passe d'une page à l'autre et connaît le
   nombre total de membres correspondant à ses critères.
4. **Given** une recherche sans résultat, **When** la liste se met à jour,
   **Then** un message indique qu'aucun membre ne correspond et propose
   d'effacer la recherche.
5. **Given** une fiche, **When** l'administrateur l'ouvre, **Then** il voit
   toutes ses informations, le compte qui la gère (avec un lien vers la fiche
   du compte) et les dates de création et de dernière modification.

---

### User Story 4 - Créer, modifier et supprimer n'importe quel membre (administration) (Priority: P2)

Une famille sans aisance numérique remet un bulletin papier. L'administrateur
crée les fiches depuis la rubrique « Membres », en les rattachant au compte de
la famille. Plus tard, il corrige une date de naissance mal saisie par un
parent et supprime un doublon.

**Why this priority**: indispensable pour les cas d'assistance et les
corrections, mais la majorité des fiches est saisie par les familles (US1).

**Independent Test**: créer une fiche rattachée à un compte existant, se
connecter avec ce compte et la retrouver dans « Mes membres » ; modifier puis
supprimer une fiche créée par un membre.

**Acceptance Scenarios**:

1. **Given** le formulaire d'ajout de l'administration, **When**
   l'administrateur saisit l'identité et choisit le compte de rattachement par
   recherche sur son adresse, **Then** la fiche est créée et apparaît dans
   « Mes membres » de ce compte.
2. **Given** n'importe quelle fiche, **When** l'administrateur modifie un champ,
   **Then** la modification est enregistrée et visible du compte qui la gère.
3. **Given** une fiche, **When** l'administrateur change son compte de
   rattachement, **Then** la fiche disparaît de « Mes membres » de l'ancien
   compte et apparaît dans celui du nouveau ; un éventuel signalement « Vous »
   est retiré.
4. **Given** n'importe quelle fiche, **When** l'administrateur la supprime et
   confirme dans le dialogue, **Then** elle disparaît de la liste et de « Mes
   membres » du compte concerné.
5. **Given** le formulaire d'ajout, **When** l'administrateur ne choisit aucun
   compte de rattachement, **Then** la fiche est créée « Sans compte » : elle
   n'est gérée que par le bureau et n'apparaît dans aucun « Mes membres ».
6. **Given** une fiche sans compte, **When** l'administrateur lui rattache un
   compte, **Then** elle apparaît dans « Mes membres » de ce compte.
7. **Given** une fiche rattachée à un compte, **When** l'administrateur retire
   le rattachement, **Then** la fiche devient « Sans compte » et disparaît de
   « Mes membres » de l'ancien compte ; un éventuel signalement « Vous » est
   retiré.
8. **Given** la liste « Membres », **When** l'administrateur filtre sur « Sans
   compte », **Then** seules les fiches sans compte de rattachement
   s'affichent.

---

### User Story 5 - Choisir le sort des fiches à la suppression d'un compte (Priority: P2)

Un parent demande l'effacement de ses données. Depuis la fiche de son compte
(002), l'administrateur choisit « Supprimer » : le dialogue indique que le
compte gère trois fiches et lui demande s'il faut les supprimer avec le compte
ou les conserver sans compte (par exemple parce que les enfants restent
adhérents et seront rattachés à l'autre parent).

**Why this priority**: le droit à l'effacement (RGPD, principe VII) impose de
pouvoir supprimer toutes les données d'une personne, sans pour autant perdre
les fiches de membres qui restent dans l'association.

**Independent Test**: supprimer un compte gérant des fiches avec chacune des
deux options ; vérifier que les fiches sont effacées, ou conservées « Sans
compte » et rattachables à un autre compte.

**Acceptance Scenarios**:

1. **Given** un compte gérant au moins une fiche, **When** l'administrateur
   choisit « Supprimer » sur ce compte, **Then** le dialogue indique le nombre
   de fiches gérées, les nomme, et impose de choisir entre « Supprimer aussi
   ces fiches » et « Conserver ces fiches sans compte », sans choix par
   défaut.
2. **Given** l'option « Supprimer aussi ces fiches », **When** l'administrateur
   confirme, **Then** le compte et toutes ses fiches sont effacés.
3. **Given** l'option « Conserver ces fiches sans compte », **When**
   l'administrateur confirme, **Then** le compte est effacé, ses fiches
   deviennent « Sans compte » et le signalement « Vous » est retiré.
4. **Given** un compte ne gérant aucune fiche, **When** l'administrateur le
   supprime, **Then** le dialogue de suppression de 002 est inchangé.
5. **Given** une fiche ajoutée au compte pendant que le dialogue est ouvert,
   **When** l'administrateur confirme, **Then** la suppression est refusée
   avec un message explicite et le dialogue est rafraîchi avec la liste à
   jour.

---

### Edge Cases

- **Doublon probable** : à la création ou à la modification, si une fiche de
  même prénom, même nom (sans tenir compte de la casse, des accents ni des
  espaces superflus) et même date de naissance existe déjà dans l'instance, un
  avertissement le signale et demande de confirmer ; pour un membre, le
  message ne révèle pas à quel compte appartient la fiche existante.
- **Fiche supprimée ou rattachement modifié entre-temps** : une action sur
  une fiche supprimée, ou dont le rattachement a changé depuis son affichage
  (par un autre administrateur), est refusée avec un message explicite, et la
  fiche ou la liste est rafraîchie.
- **Compte désactivé** (002) : ses fiches restent rattachées, consultables et
  modifiables par les administrateurs ; elles réapparaissent dans « Mes
  membres » à la réactivation.
- **Fiche « Vous » rattachée à un autre compte** : le signalement est retiré ;
  le nouveau compte peut ensuite signaler sa propre fiche.
- **Membre mineur** : l'âge est calculé à la date du jour à partir de la date
  de naissance ; aucune règle particulière n'est appliquée aux mineurs dans
  cette fonctionnalité.
- **Nombre de fiches par compte** : un compte gère au plus 20 fiches ; au-delà,
  l'ajout est refusé avec un message invitant à contacter le bureau.
- **Recherche** avec caractères spéciaux ou accents : traitée comme du texte,
  sans erreur.

## Requirements *(mandatory)*

### Functional Requirements

**Fiche membre**

- **FR-001**: Une fiche membre MUST comporter un prénom et un nom
  (obligatoires, 1 à 80 caractères chacun) et une date de naissance
  (obligatoire, ni future ni antérieure de plus de 120 ans).
- **FR-002**: Une fiche membre MAY comporter une adresse électronique, un
  numéro de téléphone et une adresse postale (ligne d'adresse, complément,
  code postal, commune), chacun facultatif et validé lorsqu'il est saisi.
- **FR-003**: Une fiche MUST être rattachée à au plus un compte, qui la gère ;
  une fiche sans compte (« Sans compte ») n'est gérée que par les
  administrateurs. Deux comptes ne peuvent pas gérer la même fiche.
  Une fiche au plus par compte MAY être signalée comme celle du titulaire
  (« Vous »).
- **FR-004**: Un compte MUST gérer au plus 20 fiches.
- **FR-005**: Le système MUST signaler un doublon probable (même prénom, même
  nom, même date de naissance, comparaison sans casse, accents ni espaces
  superflus) et demander une confirmation explicite avant d'enregistrer.

**Espace membre (« Mes membres »)**

- **FR-006**: Tout compte connecté (membre ou administrateur) MUST pouvoir
  lister, consulter, créer, modifier et supprimer les fiches qu'il gère, et
  uniquement celles-ci.
- **FR-007**: Une fiche créée depuis « Mes membres » MUST être rattachée au
  compte qui la crée ; un membre MUST NOT pouvoir changer le rattachement
  d'une fiche.
- **FR-008**: Toute tentative d'un compte d'accéder à une fiche qu'il ne gère
  pas MUST recevoir la même réponse qu'une fiche inexistante, quel que soit le
  moyen d'accès ; ce contrôle est appliqué côté serveur.
- **FR-009**: La liste « Mes membres » MUST présenter prénom, nom et âge, la
  fiche « Vous » en premier puis les autres par prénom, et un état vide
  explicatif lorsqu'il n'y a aucune fiche.

**Administration (« Membres »)**

- **FR-010**: Les administrateurs MUST pouvoir lister tous les membres, triés
  par nom puis prénom, paginés, avec le nombre total correspondant aux
  critères.
- **FR-011**: Les administrateurs MUST pouvoir rechercher un membre par tout
  fragment d'au moins 3 caractères de son prénom, de son nom, de son adresse
  électronique ou de son téléphone, sans tenir compte de la casse ni des
  accents ; plusieurs fragments séparés par des espaces doivent tous
  correspondre. La recherche est reflétée dans l'adresse de la page.
- **FR-012**: Les administrateurs MUST pouvoir consulter, créer, modifier et
  supprimer n'importe quelle fiche, et choisir, changer ou retirer son compte
  de rattachement, y compris rattacher un compte à une fiche sans compte.
- **FR-012a**: Les administrateurs MUST pouvoir filtrer la liste sur les
  fiches sans compte ; le filtre se combine avec la recherche et est reflété
  dans l'adresse de la page.
- **FR-012b**: La suppression d'un compte gérant au moins une fiche MUST
  imposer à l'administrateur un choix explicite, sans valeur par défaut :
  supprimer ces fiches avec le compte, ou les conserver sans compte (le
  signalement « Vous » est alors retiré). Le dialogue nomme les fiches
  concernées ; si la liste a changé entre l'affichage et la confirmation, la
  suppression est refusée. Le reste du comportement de suppression d'un
  compte (002) est inchangé.
- **FR-013**: La consultation et les actions de la rubrique « Membres » MUST
  être refusées aux membres et aux personnes non connectées.

**Commun**

- **FR-014**: La suppression d'une fiche MUST passer par un dialogue de
  confirmation qui nomme le membre concerné. La suppression est un effacement
  réel.
- **FR-015**: Chaque erreur de saisie MUST être affichée sur le champ concerné
  avec un message en français expliquant comment la corriger.
- **FR-016**: Après chaque action, un message de confirmation annoncé aux
  lecteurs d'écran MUST s'afficher et les listes MUST refléter immédiatement
  le nouvel état.
- **FR-017**: Le système MUST enregistrer dans les journaux de l'application
  chaque création, modification de rattachement et suppression de fiche, avec
  le compte à l'origine, la fiche visée et la date, sans y faire figurer les
  données personnelles de la fiche.
- **FR-018**: Les écrans MUST être en français, utilisables au clavier seul,
  compatibles avec les lecteurs d'écran et utilisables dès 320 pixels de
  large.

### Key Entities

- **Membre** : personne de l'association. Attributs : prénom, nom, date de
  naissance, adresse électronique, téléphone et adresse postale facultatifs,
  compte de rattachement facultatif, indicateur « fiche du titulaire », dates de création
  et de modification. Les adhésions livrées plus tard porteront sur un membre
  et une saison.
- **Compte** (existant, 001/002) : gère de zéro à vingt membres, dont au plus
  un le désigne lui-même. Sa suppression supprime ses fiches ou les laisse
  sans compte, au choix de l'administrateur.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Un titulaire de compte crée la fiche d'un membre en moins de
  1 minute depuis la rubrique « Mes membres ».
- **SC-002**: Pour 100 % des opérations sur les fiches, un compte n'obtient
  jamais ni ne modifie une fiche qu'il ne gère pas (vérifié par des tests
  automatisés pour chaque opération).
- **SC-003**: 100 % des opérations de la rubrique « Membres » sont refusées à
  un compte membre et à une personne non connectée.
- **SC-004**: Un administrateur retrouve la fiche d'un membre à partir d'un
  fragment de son nom en moins de 15 secondes, dans une association de
  2 000 membres.
- **SC-005**: Aucun écran livré ne présente de violation d'accessibilité grave
  ou critique lors des contrôles automatisés et d'un parcours complet au
  clavier.

## Assumptions

- **Un seul compte gestionnaire** : une fiche est gérée par un seul compte au
  plus ; le partage d'une fiche entre deux parents est hors périmètre
  (décision du 2026-11-01). L'autre parent passe par le parent gestionnaire
  ou par le bureau.
- **Fiches sans compte** : autorisées pour les personnes gérées par le seul
  bureau ; à la suppression d'un compte, l'administrateur décide du sort des
  fiches (décision du 2026-11-01, motivée par le droit à l'effacement).
- **Pas de lien automatique compte ↔ fiche** : la création d'un compte ne crée
  pas de fiche ; le titulaire crée la sienne s'il est adhérent.
- **Données collectées** : limitées à l'identité et aux coordonnées (RGPD,
  principe VII). Informations médicales, contacts d'urgence, autorisations
  parentales, droit à l'image, photo et documents sont hors périmètre.
- **Pas d'adhésion existante** : aucune adhésion n'existe encore, la
  suppression d'une fiche n'a donc pas de rattachement à vérifier. Quand les
  adhésions existeront, elles préciseront les règles de suppression.
- **Un membre ne voit que ses fiches** : pas d'annuaire des membres dans
  l'espace membre.
- **Une page de liste affiche 25 membres** ; ce nombre est un choix
  d'interface.
- **Export** : l'export des données personnelles (RGPD) reste couvert par la
  future spécification « Export des données personnelles ».
- **Historique** : pas de journal des modifications consultable dans
  l'interface ; la traçabilité passe par les journaux de l'application.
- **Informations complémentaires** : les champs propres à l'association
  (numéro de licence fédérale…) sont ajoutés aux fiches par la spécification
  011 « Champs personnalisés des membres et des adhésions ».
