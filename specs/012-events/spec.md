# Feature Specification: Événements de l'association

**Feature Branch**: `012-events`

**Created**: 2026-11-09

**Status**: Draft

**Input**: User description: "Ajout des événements"

## Vocabulaire

- **Événement** : une activité ponctuelle organisée par l'association
  (assemblée générale, sortie, stage, tournoi, repas…), avec une date et une
  heure de début, une date et une heure de fin, un lieu et une description.
  Il est rattaché à une saison (003).
- **État** d'un événement :
  - **Brouillon** : en préparation, visible seulement des administrateurs ;
  - **Publié** : visible dans l'espace membre ;
  - **Annulé** : l'événement n'aura pas lieu ; il reste visible, signalé
    comme annulé, et n'accepte plus d'inscription.
  Un événement publié dont la date de fin est passée est présenté comme
  **Terminé**, sans action de l'administrateur.
- **Ouverture** d'un événement, choisie par l'administrateur :
  - **Adhérents de la saison** (par défaut) : seuls les membres dont
    l'adhésion à la saison de l'événement est validée (006) peuvent être
    inscrits ;
  - **Toutes les fiches** : toute fiche membre (005) peut être inscrite,
    adhérente ou non (événement « découverte », portes ouvertes…).
- **Membre éligible** : membre qui satisfait l'ouverture de l'événement.
- **Inscription** : la participation annoncée d'un membre (fiche, 005) à un
  événement. Elle est faite par le compte qui gère la fiche, depuis l'espace
  membre, ou par un administrateur. Elle est **Confirmée** ou **Annulée** ;
  une inscription annulée reste dans l'historique et ne compte plus dans les
  places.
- **Période d'inscription** : l'intervalle pendant lequel les inscriptions
  sont ouvertes dans l'espace membre ; par défaut, de la publication jusqu'au
  début de l'événement.
- **Capacité** : nombre maximal d'inscriptions confirmées, facultatif. Un
  événement sans capacité accepte un nombre illimité d'inscrits. Un événement
  dont la capacité est atteinte est **Complet**.
- **Prix** : montant facultatif demandé pour chaque inscription. Un événement
  sans prix (ou à 0 €) est **gratuit**. Le prix en vigueur au moment de
  l'inscription est figé sur l'inscription (**montant dû**).
- **État du paiement** d'une inscription payante, sur le modèle de 006 :
  « À payer » (aucun moyen choisi), « Paiement annoncé » (le compte a choisi
  un moyen), « Payée » (réception constatée par un administrateur, ou
  automatiquement pour la carte bancaire, 008). Une inscription gratuite n'a
  pas de paiement.
- **Référence de paiement** : code unique attribué à chaque inscription
  payante, distinct de ceux des adhésions, à reporter sur un virement ou au
  dos d'un chèque.
- **À régler** : rubrique de l'espace membre qui rassemble tout ce que le
  compte doit encore payer pour ses fiches : adhésions et inscriptions.

Les autres termes (compte, fiche, membres d'un compte, saison, adhésion,
adhérent d'une saison, moyen de paiement, carte bancaire, HelloAsso, adresse
de contact) sont ceux des spécifications 001 à 011.

## Clarifications

### Session 2026-11-09

- Q: Les inscriptions font-elles partie de cette fonctionnalité, et
  peuvent-elles être payantes ? → A: Oui : agenda et inscriptions, avec un
  prix facultatif payé par les moyens de paiement existants (espèces,
  chèque, virement, carte bancaire via HelloAsso).
- Q: Qui peut être inscrit à un événement ? → A: L'administrateur choisit pour
  chaque événement : « Adhérents de la saison » ou « Toutes les fiches ».
- Q: Que se passe-t-il quand l'événement est complet ? → A: Refus simple
  (« Complet ») ; pas de liste d'attente.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Créer et publier un événement (administration) (Priority: P1)

Une administratrice ouvre la rubrique « Événements » de l'espace
administration. Elle crée « Assemblée générale 2026 » : samedi 12 décembre
de 14 h à 17 h, salle polyvalente, gratuite, réservée aux adhérents de la
saison, avec une description qui détaille l'ordre du jour. L'événement est
enregistré en brouillon ; elle le relit puis le publie. Elle crée ensuite un
« Stage de printemps » ouvert à toutes les fiches, de 20 places, à 35 €,
inscriptions ouvertes jusqu'au 1er avril.

**Why this priority**: sans événement, rien d'autre n'existe ; un agenda
publié est déjà utile aux membres même sans inscription.

**Independent Test**: créer un événement, vérifier qu'il n'apparaît pas dans
l'espace membre tant qu'il est en brouillon, le publier, le modifier, et
retrouver les informations attendues dans les listes « À venir » et
« Passés ».

**Acceptance Scenarios**:

1. **Given** la rubrique « Événements » sans événement, **When** un
   administrateur l'ouvre, **Then** un état vide explique à quoi servent les
   événements et propose d'en créer un.
2. **Given** le formulaire de création, **When** l'administrateur saisit un
   titre, une date et une heure de début et de fin, un lieu, une description
   facultative, l'ouverture, éventuellement une capacité, un prix et une
   période d'inscription, puis enregistre, **Then** l'événement est créé en
   brouillon, rattaché à la saison qui contient sa date de début, et un
   message confirme l'action.
3. **Given** une date de fin antérieure à la date de début, ou une date de
   début qui ne tombe dans aucune saison, **When** l'administrateur
   enregistre, **Then** l'enregistrement est refusé avec un message sur le
   champ concerné.
4. **Given** un événement en brouillon, **When** l'administrateur le publie,
   **Then** il apparaît dans l'espace membre et son état devient « Publié ».
5. **Given** un événement publié, **When** l'administrateur modifie sa date,
   son heure ou son lieu, **Then** la modification est visible immédiatement
   dans l'espace membre et les inscrits en sont avertis par e-mail.
6. **Given** un événement qui a des inscriptions, **When** l'administrateur
   modifie son prix, **Then** le nouveau prix s'applique aux inscriptions
   suivantes seulement ; les montants dus déjà figés ne changent pas.
7. **Given** la liste des événements, **When** l'administrateur la consulte,
   **Then** il voit séparément les événements à venir (par date croissante) et
   passés (par date décroissante), avec leur état, leur date, leur lieu, leur
   prix et le nombre d'inscrits sur la capacité, et peut filtrer par saison
   et par état.
8. **Given** un membre (non administrateur), **When** il tente de créer, de
   modifier, de publier, d'annuler ou de supprimer un événement, **Then**
   l'accès est refusé.

---

### User Story 2 - Consulter les événements à venir (espace membre) (Priority: P1)

Claire ouvre « Événements à venir » dans l'espace membre. Elle voit la liste
des événements publiés à venir, du plus proche au plus lointain, avec la date,
l'heure, le lieu, le prix et, pour ceux qui ont une capacité, le nombre de
places restantes. Elle ouvre l'assemblée générale pour lire l'ordre du jour.

**Why this priority**: c'est la raison d'être de la rubrique déjà présente
dans la navigation ; informer les membres est la valeur première d'un
événement.

**Independent Test**: publier deux événements à venir, un brouillon, un
annulé et un passé ; vérifier depuis un compte membre ce qui est listé et
dans quel ordre.

**Acceptance Scenarios**:

1. **Given** des événements publiés à venir, **When** un compte membre ouvre
   « Événements à venir », **Then** ils sont listés par date de début
   croissante, avec titre, date, heures, lieu, prix (ou « Gratuit ») et
   places restantes le cas échéant.
2. **Given** un événement en brouillon, **When** un compte membre consulte la
   liste ou tente d'ouvrir l'événement par son adresse, **Then** il ne le voit
   pas et reçoit « Page introuvable ».
3. **Given** un événement publié puis annulé, **When** un compte membre
   consulte la liste avant sa date, **Then** il le voit signalé « Annulé »,
   sans possibilité de s'inscrire.
4. **Given** aucun événement à venir, **When** un compte membre ouvre la
   rubrique, **Then** un état vide l'indique.
5. **Given** la page d'un événement, **When** un compte membre l'ouvre,
   **Then** il voit la description complète, le lieu, les dates, le prix,
   l'ouverture, la période d'inscription et, pour chaque fiche qu'il gère, si
   elle est inscrite, inscriptible ou non éligible avec la raison.

---

### User Story 3 - Inscrire ses membres à un événement (espace membre) (Priority: P1)

Claire inscrit son fils Lucas à l'assemblée générale depuis la page de
l'événement, puis s'inscrit elle-même. Elle reçoit une confirmation par e-mail
et retrouve les inscriptions dans « Mes inscriptions ». Une semaine plus tard,
Lucas ne peut plus venir : elle annule son inscription, ce qui libère sa
place.

**Why this priority**: la navigation prévoit « Mes inscriptions » et
« Inscriptions » ; savoir qui vient est le besoin principal des organisateurs.

**Independent Test**: sur un événement gratuit réservé aux adhérents, depuis
un compte qui gère une fiche adhérente et une fiche non adhérente, inscrire
la fiche adhérente, vérifier le refus pour l'autre, l'e-mail, la place
décomptée et « Mes inscriptions », puis annuler l'inscription et vérifier que
la place est libérée.

**Acceptance Scenarios**:

1. **Given** un événement publié dont la période d'inscription est ouverte,
   **When** le compte gestionnaire choisit une ou plusieurs fiches éligibles
   et confirme, **Then** chaque fiche est inscrite (« Confirmée »), la page
   l'indique et un e-mail de confirmation est envoyé à l'adresse de contact.
2. **Given** un événement « Adhérents de la saison » et une fiche sans
   adhésion validée à cette saison, **When** le compte gestionnaire consulte
   l'événement, **Then** la fiche est présentée comme non éligible avec la
   raison, et une tentative d'inscription est refusée par le serveur.
3. **Given** un événement « Toutes les fiches », **When** le compte
   gestionnaire inscrit une fiche non adhérente, **Then** l'inscription est
   acceptée.
4. **Given** une fiche déjà inscrite (inscription confirmée), **When** une
   nouvelle inscription est demandée pour elle au même événement, **Then**
   elle est refusée sans créer de doublon.
5. **Given** un événement complet, **When** le compte gestionnaire tente
   d'inscrire une fiche, **Then** l'inscription est refusée avec le message
   « Complet » ; aucune liste d'attente n'est proposée.
6. **Given** une inscription gratuite, ou payante mais ni payée ni en cours de
   paiement par carte, et une période d'inscription encore ouverte, **When**
   le compte gestionnaire l'annule après confirmation, **Then** elle passe
   « Annulée », la place est libérée et la fiche peut être réinscrite.
7. **Given** une période d'inscription close (dépassée ou pas encore ouverte),
   **When** le compte gestionnaire consulte l'événement, **Then** il ne peut
   ni inscrire ni annuler, et la page indique la période d'inscription.
8. **Given** « Mes inscriptions », **When** le compte l'ouvre, **Then** il voit
   les inscriptions à venir de toutes ses fiches, puis les passées, avec le nom
   du membre, l'événement, l'état de l'inscription et, si elle est payante,
   l'état du paiement.

---

### User Story 4 - Payer une inscription (espace membre) (Priority: P1)

Claire inscrit Lucas au stage de printemps à 35 €. L'inscription est
confirmée et sa place réservée ; la page l'emmène vers le paiement :
montant 35 €, référence « EVT-2627-0007 », et le choix entre les moyens de
paiement activés. Elle paie par carte bancaire chez HelloAsso ; au retour,
l'inscription est « Payée ». Une autre fois, elle choisit le chèque :
l'inscription passe « Paiement annoncé — chèque », puis « Payée » quand le
trésorier constate la réception. La rubrique « À régler » lui montre à tout
moment ce qui reste à payer, adhésions et inscriptions confondues.

**Why this priority**: sans paiement, un événement payant ne peut pas être
proposé ; c'est le choix retenu pour cette fonctionnalité.

**Independent Test**: sur un événement payant, inscrire une fiche, vérifier
la page de paiement et « À régler », annoncer un chèque, faire constater la
réception par un administrateur ; puis, avec la carte bancaire activée en mode
test, payer une seconde inscription par carte et vérifier la constatation
automatique.

**Acceptance Scenarios**:

1. **Given** un événement payant, **When** le compte gestionnaire inscrit une
   fiche, **Then** l'inscription est confirmée avec le prix figé comme montant
   dû, une référence de paiement et l'état « À payer », et la page de
   paiement de l'inscription s'affiche.
2. **Given** la page de paiement d'une inscription, **When** elle s'affiche,
   **Then** elle présente l'événement, le membre, le montant dû, la référence
   et les moyens de paiement activés dans les paramètres, avec les mêmes
   informations et le même fonctionnement que pour une adhésion (006, 008).
3. **Given** le choix d'espèces, de chèque ou de virement, **When** le compte
   confirme, **Then** l'inscription passe « Paiement annoncé » avec ce moyen.
4. **Given** la carte bancaire activée, **When** le compte paie par carte et
   que le paiement réussit, **Then** l'inscription passe « Payée » sans
   intervention d'un administrateur, même si le compte ne revient pas sur
   Frigg ; l'intitulé transmis à HelloAsso comporte le titre de l'événement
   et la référence, sans donnée personnelle du membre.
5. **Given** un paiement par carte échoué, abandonné ou expiré, **When** le
   compte revient sur Frigg, **Then** l'inscription reste dans son état
   précédent et une nouvelle tentative ou un autre moyen est possible.
6. **Given** des inscriptions et des adhésions à payer, **When** le compte
   ouvre « À régler », **Then** il voit chacune avec le membre, l'objet
   (adhésion à telle saison ou inscription à tel événement), le montant dû,
   l'état du paiement et un accès à sa page de paiement ; un état vide
   indique qu'il n'y a rien à régler.
7. **Given** une inscription payée ou dont un paiement par carte est en cours,
   **When** le compte gestionnaire tente de l'annuler, **Then** l'annulation
   lui est refusée et la page l'invite à contacter l'association.
8. **Given** une inscription gratuite, **When** elle est créée, **Then**
   aucune page de paiement n'est proposée et elle n'apparaît pas dans
   « À régler ».

---

### User Story 5 - Suivre les inscrits et les paiements d'un événement (administration) (Priority: P2)

Avant le stage, l'administratrice ouvre l'événement : elle voit la liste des
inscrits avec la date d'inscription, le compte qui a inscrit chacun et l'état
du paiement, ainsi que le total dû et le total encaissé. Elle inscrit à la
main un membre qui l'a appelée et note qu'il a payé en espèces, constate un
chèque reçu, retire une inscription, puis exporte la liste au format CSV pour
l'animateur.

**Why this priority**: les inscriptions et paiements faits depuis l'espace
membre suffisent à un premier usage ; la gestion par l'administration la
complète.

**Independent Test**: sur un événement payant avec des inscrits, ajouter une
inscription marquée payée, constater un paiement annoncé, retirer une
inscription, puis exporter la liste et vérifier son contenu et les totaux.

**Acceptance Scenarios**:

1. **Given** un événement avec des inscrits, **When** un administrateur ouvre
   l'événement, **Then** il voit la liste des inscriptions confirmées (nom,
   prénom, adhérent ou non, date d'inscription, compte auteur, état du
   paiement), leur nombre sur la capacité, le total dû et le total encaissé,
   et peut afficher aussi les inscriptions annulées.
2. **Given** un membre éligible, **When** l'administrateur l'inscrit, **Then**
   l'inscription est créée même hors période d'inscription, il peut la marquer
   directement « Payée » en indiquant le moyen, et un e-mail de confirmation
   est envoyé à son adresse de contact.
3. **Given** un événement complet, **When** l'administrateur inscrit un membre,
   **Then** l'inscription est refusée tant qu'il n'a pas relevé la capacité.
4. **Given** une inscription « À payer » ou « Paiement annoncé », **When**
   l'administrateur choisit « Marquer comme payée », indique la date et le
   moyen et confirme, **Then** elle passe « Payée » et l'adresse de contact
   est prévenue par e-mail ; une constatation manuelle erronée peut être
   annulée, contrairement à un paiement par carte.
5. **Given** une inscription, **When** l'administrateur la retire après
   confirmation, **Then** elle passe « Annulée », la place est libérée et
   l'adresse de contact est prévenue par e-mail ; si elle était payée ou
   qu'un paiement par carte est en cours, le dialogue rappelle que le
   remboursement se fait hors de Frigg ou dans l'espace HelloAsso.
6. **Given** la liste des inscrits, **When** l'administrateur l'exporte,
   **Then** il obtient un fichier CSV lisible par un tableur (nom, prénom,
   date de naissance, téléphone, adresse de contact, adhérent ou non, date
   d'inscription, montant dû, état du paiement, moyen, référence).
7. **Given** la rubrique « Inscriptions » de l'administration, **When** un
   administrateur l'ouvre, **Then** il voit les inscriptions de tous les
   événements, filtrables par événement, par état de l'inscription et par
   état du paiement, et recherchables par nom de membre ou par référence de
   paiement.

---

### User Story 6 - Annuler ou supprimer un événement (administration) (Priority: P2)

Le tournoi de mars est annulé faute de salle. L'administratrice annule
l'événement en saisissant un message ; les inscrits sont prévenus par e-mail
et le dialogue lui rappelle les inscriptions déjà payées à rembourser. Elle
supprime par ailleurs un brouillon créé par erreur.

**Why this priority**: événement imprévu mais courant, qui doit prévenir les
participants sans ressaisie.

**Independent Test**: annuler un événement publié avec des inscrits payés et
non payés, vérifier les e-mails en file d'attente, l'affichage côté membre et
le retrait des inscriptions de « À régler » ; supprimer un brouillon sans
inscription.

**Acceptance Scenarios**:

1. **Given** un événement publié à venir, **When** l'administrateur l'annule
   après un dialogue de confirmation, avec un message facultatif, **Then**
   l'événement passe « Annulé », les inscriptions sont conservées pour
   l'historique, les inscriptions non payées sortent de « À régler », et
   chaque adresse de contact d'un inscrit reçoit un seul e-mail d'annulation
   reprenant le message.
2. **Given** un événement qui a des inscriptions payées, **When**
   l'administrateur ouvre le dialogue d'annulation, **Then** le dialogue
   indique leur nombre et leur montant, et rappelle que les remboursements se
   font hors de Frigg ou dans l'espace HelloAsso.
3. **Given** un événement annulé à venir, **When** l'administrateur le
   rétablit, **Then** il redevient « Publié » avec ses inscriptions et leurs
   états de paiement, et les inscrits en sont prévenus.
4. **Given** un événement sans aucune inscription (confirmée ou annulée),
   **When** l'administrateur le supprime après confirmation, **Then** il
   disparaît définitivement.
5. **Given** un événement qui a au moins une inscription, **When**
   l'administrateur tente de le supprimer, **Then** la suppression est
   refusée et l'annulation lui est proposée.

---

### Edge Cases

- Un événement sur plusieurs jours (stage du lundi au vendredi) est accepté :
  seule la fin postérieure au début est exigée ; l'affichage indique les deux
  dates.
- Deux inscriptions simultanées pour la dernière place : une seule aboutit,
  l'autre reçoit « Complet ».
- Une inscription payante non payée réserve sa place ; elle n'est jamais
  libérée automatiquement. L'administrateur peut la retirer.
- L'administrateur abaisse la capacité sous le nombre d'inscriptions
  confirmées : la modification est refusée avec le nombre actuel.
- L'administrateur passe un événement de « Toutes les fiches » à « Adhérents
  de la saison » alors que des non-adhérents sont inscrits : la modification
  est acceptée, les inscriptions existantes sont conservées et signalées
  « non éligible » dans la liste des inscrits ; l'administrateur décide de les
  retirer. Il en va de même pour un inscrit dont l'adhésion est annulée.
- Un événement gratuit devient payant après des inscriptions : les
  inscriptions existantes restent gratuites ; seules les suivantes portent le
  nouveau prix. L'inverse laisse les montants dus déjà figés inchangés.
- Un remboursement fait dans l'espace HelloAsso d'un paiement d'inscription
  est constaté comme pour une adhésion (008) : le paiement passe
  « Remboursé » et l'inscription est annulée si elle ne l'est pas déjà.
- Un paiement par carte reçu en double, d'un montant différent ou sur une
  inscription annulée est enregistré et signalé aux administrateurs, comme
  pour une adhésion (008).
- Une fiche membre est supprimée (005) : ses inscriptions sont supprimées
  avec elle, conformément au droit à l'effacement ; la suppression d'une fiche
  ayant une inscription payée à un événement à venir est précédée d'un
  avertissement dans le dialogue de confirmation.
- Un compte cesse de gérer une fiche : les inscriptions de la fiche restent
  valables et passent sous la gestion du nouveau compte gestionnaire, y
  compris leur paiement.
- Une fiche sans adresse de contact est inscrite par un administrateur :
  l'inscription est créée sans e-mail et la liste le signale.
- Un événement publié passe sa date de fin : il devient « Terminé », n'est
  plus modifiable dans ses dates, n'accepte plus d'inscription ni
  d'annulation depuis l'espace membre, et reste consultable avec ses
  inscrits ; un paiement encore dû reste payable et visible dans
  « À régler ».
- Aucun moyen de paiement n'est activé : l'inscription payante reste possible,
  la page de paiement indique de contacter l'association, comme pour une
  adhésion.
- Les dates et heures sont saisies et affichées dans le fuseau de
  l'association (Europe/Paris), passages à l'heure d'été compris.
- Le titre ou la description contiennent du balisage : il est affiché comme
  du texte, jamais interprété.

## Requirements *(mandatory)*

### Functional Requirements

**Événements**

- **FR-001**: L'administration MUST offrir une rubrique « Événements » listant
  les événements à venir puis passés, filtrables par saison et par état.
- **FR-002**: Un événement MUST comporter : un titre (1 à 120 caractères), une
  date et heure de début, une date et heure de fin postérieure au début, un
  lieu (1 à 200 caractères), une description facultative (au plus 5 000
  caractères, texte avec retours à la ligne), une ouverture (« Adhérents de la
  saison » par défaut, ou « Toutes les fiches »), une capacité facultative
  (entier de 1 à 10 000), un prix facultatif (de 0 à 10 000 €, au centime) et
  une période d'inscription facultative (ouverture et clôture, la clôture au
  plus tard à la fin de l'événement).
- **FR-003**: Un événement MUST être rattaché à la saison qui contient sa date
  de début ; un événement dont la date de début ne tombe dans aucune saison
  MUST être refusé.
- **FR-004**: Un événement MUST être créé en état « Brouillon » ; les
  administrateurs MUST pouvoir le publier, l'annuler et rétablir un événement
  annulé tant qu'il n'est pas terminé.
- **FR-005**: Un événement publié dont la date de fin est passée MUST être
  présenté comme « Terminé » ; ses dates MUST NOT être modifiables.
- **FR-006**: Les administrateurs MUST pouvoir supprimer un événement sans
  aucune inscription, après un dialogue de confirmation ; un événement ayant
  au moins une inscription, même annulée, MUST NOT pouvoir être supprimé.
- **FR-007**: La capacité MUST NOT pouvoir être abaissée sous le nombre
  d'inscriptions confirmées.
- **FR-008**: Une modification du prix ou de l'ouverture MUST s'appliquer
  aux inscriptions suivantes seulement ; les inscriptions existantes et leurs
  montants dus restent inchangés.

**Espace membre**

- **FR-009**: La rubrique « Événements à venir » de l'espace membre MUST
  lister les événements publiés ou annulés dont la fin n'est pas passée, par
  date de début croissante ; les brouillons MUST NOT être transmis à un compte
  non administrateur.
- **FR-010**: La page d'un événement MUST présenter titre, dates, lieu,
  description, prix ou « Gratuit », ouverture, période d'inscription, places
  restantes (si capacité) et, pour chaque fiche gérée par le compte, son
  statut : inscrite (avec l'état du paiement), inscriptible, ou non éligible
  avec la raison.

**Inscriptions**

- **FR-011**: L'éligibilité MUST être vérifiée par le serveur à chaque
  inscription : pour « Adhérents de la saison », la fiche MUST avoir une
  adhésion validée à la saison de l'événement ; pour « Toutes les fiches »,
  toute fiche est éligible.
- **FR-012**: Un compte MUST pouvoir inscrire à un événement publié, pendant
  sa période d'inscription, toute fiche éligible qu'il gère ; il MUST NOT
  pouvoir inscrire ni désinscrire une fiche qu'il ne gère pas.
- **FR-013**: Un compte MUST pouvoir annuler, pendant la période
  d'inscription, une inscription d'une fiche qu'il gère si elle est gratuite,
  ou payante sans être payée ni en cours de paiement par carte.
- **FR-014**: Une fiche MUST NOT avoir plus d'une inscription confirmée au
  même événement ; une fiche dont l'inscription est annulée MUST pouvoir être
  réinscrite.
- **FR-015**: Le système MUST garantir que le nombre d'inscriptions confirmées
  ne dépasse jamais la capacité, y compris lors d'inscriptions simultanées ;
  un événement complet MUST refuser toute nouvelle inscription, sans liste
  d'attente.
- **FR-016**: Les administrateurs MUST pouvoir inscrire tout membre éligible,
  y compris hors période d'inscription, et retirer toute inscription, avant la
  fin de l'événement, dans la limite de la capacité.
- **FR-017**: Chaque inscription MUST conserver la date d'inscription, le
  compte qui l'a faite, son état et, si elle est annulée, la date et l'auteur
  de l'annulation.
- **FR-018**: La rubrique « Mes inscriptions » MUST lister les inscriptions des
  fiches gérées par le compte, à venir puis passées, avec leur état et l'état
  du paiement.
- **FR-019**: La rubrique « Inscriptions » de l'administration MUST lister les
  inscriptions de tous les événements, filtrables par événement, par état et
  par état du paiement, et recherchables par nom ou prénom du membre selon
  les règles de recherche sans accents du projet, ou par référence de
  paiement. Elle MUST permettre de n'afficher que les inscriptions portant
  un signalement de paiement non traité, et l'encart du tableau de bord MUST
  y renvoyer.
- **FR-020**: Les administrateurs MUST pouvoir exporter la liste des inscrits
  d'un événement au format CSV.

**Paiement des inscriptions**

- **FR-021**: Une inscription à un événement payant MUST figer le prix en
  vigueur comme montant dû, recevoir une référence de paiement unique et
  l'état « À payer » ; la place est réservée dès l'inscription.
- **FR-022**: Le compte qui gère la fiche MUST pouvoir payer une inscription
  par les moyens de paiement activés dans les paramètres (espèces, chèque,
  virement, carte bancaire), avec les mêmes informations affichées, les mêmes
  états et les mêmes règles que pour une adhésion (006, 008), pour le montant
  dû exact, en un seul versement.
- **FR-023**: Le paiement par carte d'une inscription MUST reprendre toutes les
  garanties de 008 : constatation seulement après vérification auprès de
  HelloAsso, constatation même sans retour du compte, constatation unique,
  signalement des anomalies, prise en compte des remboursements, mention
  « Mode test ». Il MUST utiliser l'adresse de notification existante, sans
  nouvelle adresse accessible sans session.
- **FR-024**: L'intitulé transmis à HelloAsso MUST se composer du titre de
  l'événement et de la référence de paiement, sans donnée personnelle du
  membre.
- **FR-025**: Les administrateurs MUST pouvoir constater la réception d'un
  paiement (date, moyen), annuler une constatation manuelle erronée, et
  marquer une inscription « Payée » dès sa création ; un paiement par carte
  MUST NOT pouvoir être annulé par un administrateur.
- **FR-026**: La page d'un événement en administration MUST afficher le total
  dû et le total encaissé (hors paiements en mode test) des inscriptions
  confirmées.
- **FR-027**: L'annulation d'un événement ou le retrait d'une inscription payée
  ou au paiement par carte en cours MUST rappeler, dans le dialogue de
  confirmation, que le remboursement se fait hors de Frigg ou dans l'espace
  HelloAsso ; Frigg ne rembourse pas.
- **FR-028**: Une inscription annulée, ou dont l'événement est annulé, MUST
  NOT être payable ; un paiement en cours ou reçu ensuite est traité comme une
  anomalie (FR-023).
- **FR-029**: La rubrique « À régler » de l'espace membre MUST rassembler les
  adhésions et les inscriptions des fiches gérées par le compte dont le
  paiement est « À payer » ou « Paiement annoncé », avec un accès à leur page
  de paiement.

**Notifications**

- **FR-030**: Le système MUST envoyer un e-mail à l'adresse de contact du
  membre lors de son inscription (avec, si elle est payante, le montant, la
  référence et le lien de paiement), de la constatation de son paiement, de
  son retrait par un administrateur, d'un changement de date, d'heure ou de
  lieu d'un événement publié, de l'annulation et du rétablissement de
  l'événement.
- **FR-031**: Une même adresse de contact MUST recevoir un seul e-mail par
  changement d'événement, même si elle représente plusieurs inscrits ; l'e-mail
  nomme alors chacun d'eux.
- **FR-032**: Les e-mails MUST passer par la file d'attente existante, insérés
  dans la même opération que l'action qui les déclenche.

**Autorisations et interface**

- **FR-033**: La création, la modification, la publication, l'annulation, la
  suppression des événements, l'inscription d'un membre que l'on ne gère pas,
  la liste complète des inscrits, la constatation des paiements et l'export
  MUST être réservés aux administrateurs, contrôlés côté serveur.
- **FR-034**: Les rubriques « Événements à venir », « Mes inscriptions »,
  « À régler », « Événements » et « Inscriptions » MUST remplacer les pages
  « Bientôt disponible » actuelles.
- **FR-035**: L'annulation d'un événement, la suppression d'un événement, le
  retrait ou l'annulation d'une inscription et l'annulation d'une
  constatation de paiement MUST passer par un dialogue de confirmation.
- **FR-036**: Le système MUST enregistrer dans les journaux de l'application
  chaque création, publication, annulation et suppression d'événement, chaque
  inscription et annulation d'inscription, et chaque événement de paiement,
  avec l'événement visé, le compte à l'origine et la référence de paiement,
  sans données personnelles du membre.
- **FR-037**: Tous les écrans MUST être en français, utilisables au clavier et
  au lecteur d'écran, et dès 320 pixels de large.

### Key Entities

- **Événement** : titre, description, lieu, début, fin, saison de
  rattachement, ouverture, capacité facultative, prix facultatif, période
  d'inscription facultative, état (brouillon, publié, annulé), message
  d'annulation, dates de création et de modification.
- **Inscription** : lie une fiche membre à un événement ; état (confirmée,
  annulée), date d'inscription, compte auteur, date et auteur de
  l'annulation ; pour une inscription payante, montant dû, référence de
  paiement, état du paiement, moyen, date de constatation. Au plus une
  inscription confirmée par couple fiche–événement.
- **Paiement en ligne** (008) : une tentative de paiement par carte, désormais
  rattachée soit à une adhésion, soit à une inscription.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Un administrateur crée et publie un événement complet en moins de
  3 minutes.
- **SC-002**: Un compte membre inscrit une fiche à un événement gratuit en
  moins de 1 minute et en 3 actions au plus depuis la liste des événements.
- **SC-003**: Un compte membre inscrit une fiche à un événement payant et le
  paie par carte en moins de 3 minutes, temps passé chez HelloAsso compris.
- **SC-004**: Aucun événement ne compte plus d'inscriptions confirmées que sa
  capacité, même avec 50 inscriptions simultanées sur les dernières places.
- **SC-005**: 100 % des paiements par carte réussis d'inscriptions sont
  constatés, sans intervention d'un administrateur, une seule fois chacun.
- **SC-006**: 100 % des inscrits d'un événement annulé ou modifié (date,
  heure, lieu) reçoivent un e-mail, à raison d'un seul par adresse de contact.
- **SC-007**: La liste des inscrits d'un événement de 500 participants, avec
  ses totaux, s'affiche et s'exporte en moins de 2 secondes.
- **SC-008**: Aucun brouillon ni aucune donnée d'un membre non géré n'est
  accessible depuis un compte non administrateur (vérifié par les tests
  d'autorisation).
- **SC-009**: Les parcours de la fonctionnalité ne présentent aucune violation
  d'accessibilité de niveau grave ou critique.

## Assumptions

- Les événements sont visibles uniquement par les comptes connectés : aucun
  agenda public ni inscription d'une personne sans compte (principe II de la
  constitution). « Toutes les fiches » vise les fiches existantes, y compris
  non adhérentes ; une personne extérieure doit d'abord avoir une fiche,
  créée par un compte ou par un administrateur.
- Un seul prix par événement : pas de tarif réduit, de tarif adhérent /
  non-adhérent ni d'options payantes (repas, transport) dans cette version.
- Pas de paiement partiel ni d'acompte ; pas de délai de paiement libérant
  automatiquement une place non payée.
- Frigg ne rembourse pas : les remboursements se font hors de Frigg, ou dans
  l'espace HelloAsso pour la carte bancaire (008).
- Les moyens de paiement et leurs informations (006, 008) sont communs aux
  adhésions et aux inscriptions ; aucun paramétrage propre aux événements.
- Pas de liste d'attente (événement complet : refus simple).
- Pas d'événement récurrent (série hebdomadaire d'entraînements) : chaque
  événement est créé individuellement ; la duplication d'un événement est
  hors périmètre de cette version.
- Pas de pointage des présences le jour de l'événement, ni d'export au format
  calendrier (iCal), ni de rappel automatique avant l'événement dans cette
  version.
- Pas de champ personnalisé d'événement ni de question posée à l'inscription
  (régime alimentaire, covoiturage…) dans cette version.
- Le fuseau horaire de l'association est Europe/Paris.
- Les communications groupées (009) ne sont pas étendues aux inscrits d'un
  événement dans cette version ; les notifications de FR-030 suffisent.
- La rubrique « Finances » de l'administration et le tableau de bord ne sont
  pas modifiés, hormis l'encart d'anomalies de paiement de 008, qui couvre
  désormais aussi les inscriptions, avec un lien distinct vers la liste
  filtrée de chaque cible (adhésions, inscriptions).
