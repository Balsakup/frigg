# Feature Specification: Envoi d'e-mails groupés aux adhérents

**Feature Branch**: `009-bulk-email`

**Created**: 2026-11-03

**Status**: Draft

**Input**: User description: "Communication / Envoi d'e-mail en masse avec possibilité de choisir la cible, par exemple tous les adhérents à jour ou non dans leur paiement, tous les adhérents en impayé"

## Vocabulaire

- **Communication** : un message rédigé par un administrateur (objet et
  texte) et envoyé en une fois à un ensemble de destinataires. Une
  communication envoyée est conservée dans l'historique.
- **Cible** : les critères qui désignent les adhérents concernés par une
  communication : une saison et un ou plusieurs états du paiement de
  l'adhésion.
- **Adhérent d'une saison** : un membre (fiche, 005) dont l'adhésion à cette
  saison est validée (006). Les demandes en attente, refusées ou annulées ne
  font pas d'un membre un adhérent.
- **Situation de paiement** d'un adhérent, déduite de l'état du paiement de son
  adhésion (006, 008) :
  - **À jour** : adhésion « Payée », ou gratuite (montant appliqué de 0 €) ;
  - **Paiement annoncé** : l'adhérent a choisi un moyen de paiement, la
    réception n'est pas encore constatée ;
  - **À payer** : aucun moyen de paiement choisi, ou paiement par carte non
    abouti.
  - **En impayé** désigne ensemble « Paiement annoncé » et « À payer », c'est-à-
    dire tout adhérent qui n'est pas à jour.
- **Destinataire** : une adresse électronique qui reçoit la communication. Un
  même destinataire peut représenter plusieurs adhérents (un parent dont les
  trois enfants adhèrent) ; il ne reçoit alors qu'un seul exemplaire.
- **Adresse de contact d'un adhérent** : l'adresse électronique du compte qui
  gère sa fiche ; pour une fiche sans compte, l'adresse électronique de la
  fiche ; à défaut, l'adhérent n'est pas joignable.

Les autres termes (compte, fiche, saison, tarif, adhésion, état du paiement)
sont ceux des spécifications 001 à 008.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Relancer les adhérents en impayé (Priority: P1)

Marc, trésorier et administrateur, ouvre la nouvelle rubrique
« Communication ». Il choisit « Nouvelle communication », sélectionne la
saison 2026-2027 et la cible prédéfinie « En impayé ». Frigg lui indique que
la cible compte 23 adhérents, joignables à 19 adresses (certains parents
gèrent plusieurs enfants), et que 2 adhérents n'ont aucune adresse : il peut
consulter la liste des adhérents ciblés et celle des non joignables. Il
rédige l'objet « Rappel : cotisation 2026-2027 » et un court texte, vérifie
l'aperçu, puis confirme l'envoi dans un dialogue qui rappelle le nombre de
destinataires. La communication part ; chaque destinataire reçoit son propre
e-mail, sans voir les autres adresses.

**Why this priority**: relancer les cotisations impayées est le besoin le plus
fréquent et le plus coûteux en temps pour le bureau ; c'est la valeur
centrale de la demande.

**Independent Test**: avec une saison portant des adhésions payées, gratuites,
annoncées et à payer, cibler « En impayé », vérifier le décompte et la liste,
envoyer, et constater que seules les adresses de contact des adhérents
annoncés ou à payer reçoivent le message, une seule fois chacune.

**Acceptance Scenarios**:

1. **Given** une saison avec des adhésions validées dans chaque situation de
   paiement, **When** l'administrateur choisit la cible « En impayé »,
   **Then** le décompte n'inclut que les adhérents « Paiement annoncé » et
   « À payer », et exclut les adhésions gratuites, payées, en attente,
   refusées et annulées.
2. **Given** un compte gérant deux enfants tous deux en impayé, **When** la
   communication est envoyée, **Then** l'adresse du compte reçoit un seul
   e-mail et le décompte la compte une seule fois.
3. **Given** un adhérent ciblé dont la fiche n'a pas de compte ni d'adresse,
   **When** l'administrateur prépare la communication, **Then** il apparaît
   dans la liste des non joignables et n'est pas compté parmi les
   destinataires.
4. **Given** une communication prête, **When** l'administrateur choisit
   « Envoyer », **Then** un dialogue de confirmation rappelle l'objet, la
   cible et le nombre de destinataires ; l'envoi n'a lieu qu'après
   confirmation.
5. **Given** un objet ou un texte vide, ou une cible sans aucun destinataire,
   **When** l'administrateur tente d'envoyer, **Then** l'envoi est refusé
   avec un message expliquant comment corriger.
6. **Given** une communication envoyée, **When** un destinataire l'ouvre,
   **Then** il voit le nom de l'association comme expéditeur, l'objet et le
   texte rédigés, et aucune autre adresse de destinataire.

---

### User Story 2 - Joindre des documents à une communication (Priority: P2)

Pour convoquer l'assemblée générale, Sophie, secrétaire, prépare une
communication et y joint la convocation et le formulaire de pouvoir en PDF.
Les deux fichiers apparaissent sous le texte avec leur nom et leur taille ;
elle retire le formulaire, joint par erreur dans une ancienne version, puis
joint la bonne. L'aperçu et le dialogue de confirmation listent les pièces
jointes. Chaque destinataire reçoit l'e-mail avec les deux documents.

**Why this priority**: les communications importantes (convocation, compte
rendu, attestation, calendrier) s'accompagnent presque toujours d'un document ;
sans pièce jointe, le bureau devrait passer par un autre outil.

**Independent Test**: préparer une communication avec deux fichiers valides,
tenter d'en ajouter un trop volumineux puis un d'un type refusé, envoyer, et
vérifier que chaque destinataire reçoit les deux documents intacts et que
l'historique permet de les télécharger.

**Acceptance Scenarios**:

1. **Given** une communication en préparation, **When** l'administrateur
   ajoute un ou plusieurs fichiers d'un type accepté, **Then** chacun apparaît
   avec son nom et sa taille, et peut être retiré avant l'envoi.
2. **Given** un fichier d'un type refusé, trop volumineux, ou qui ferait
   dépasser le nombre ou la taille totale autorisés, **When** l'administrateur
   l'ajoute, **Then** il est refusé avec un message indiquant la limite
   concernée, et les pièces déjà jointes sont conservées.
3. **Given** une communication avec pièces jointes, **When** l'administrateur
   affiche l'aperçu ou le dialogue de confirmation, **Then** la liste des
   pièces jointes (nom et taille) y figure.
4. **Given** une communication envoyée avec pièces jointes, **When** un
   destinataire ouvre l'e-mail, **Then** il y trouve chaque document sous son
   nom d'origine, identique au fichier déposé.
5. **Given** une communication envoyée avec pièces jointes, **When** un
   administrateur en consulte le détail, **Then** il peut télécharger chaque
   pièce jointe.

---

### User Story 3 - Choisir librement la cible (Priority: P2)

Avant l'assemblée générale, Sophie, secrétaire, veut écrire à tous les
adhérents de la saison, quelle que soit leur situation de paiement. Elle
choisit la cible « Tous les adhérents ». Une autre fois, elle remercie
uniquement les adhérents à jour (« À jour »). Pour une relance plus fine,
elle compose sa propre cible en cochant seulement « À payer » et en
restreignant à deux tarifs de la saison.

**Why this priority**: le besoin exprimé couvre plusieurs cibles ; une fois la
relance des impayés livrée, élargir le choix de la cible réutilise le même
parcours.

**Independent Test**: avec une saison peuplée, comparer le décompte de chaque
cible prédéfinie et d'une cible composée aux adhésions attendues.

**Acceptance Scenarios**:

1. **Given** une saison, **When** l'administrateur prépare une communication,
   **Then** il peut choisir parmi les cibles prédéfinies « Tous les
   adhérents », « À jour » et « En impayé », ou cocher lui-même une ou
   plusieurs situations de paiement (« À jour », « Paiement annoncé »,
   « À payer »).
2. **Given** une saison portant plusieurs tarifs, **When** l'administrateur
   restreint la cible à certains tarifs, **Then** seuls les adhérents à l'un
   de ces tarifs sont retenus ; sans restriction, tous les tarifs le sont.
3. **Given** plusieurs saisons ouvertes ou closes, **When** l'administrateur
   choisit la saison, **Then** la saison courante est proposée par défaut et
   toute saison ouverte ou close peut être choisie ; les saisons en
   brouillon ne sont pas proposées.
4. **Given** une modification de la cible, **When** elle est appliquée,
   **Then** le décompte des adhérents, des destinataires et des non
   joignables se met à jour sans quitter la page.

---

### User Story 4 - Consulter l'historique des communications (Priority: P3)

Un mois plus tard, Marc veut savoir s'il a déjà relancé les impayés et si les
messages sont bien partis. Il ouvre la rubrique « Communication » : la liste
présente chaque communication envoyée, la plus récente en premier, avec sa
date, son objet, sa cible, son auteur et le nombre de destinataires. Il ouvre
la dernière et voit son texte, la liste des destinataires et, pour chacun,
si l'e-mail est parti, en attente ou en échec.

**Why this priority**: utile pour éviter les doublons de relance et repérer les
adresses en échec, mais l'envoi apporte déjà de la valeur sans historique.

**Independent Test**: envoyer deux communications, vérifier leur présence et
leur ordre dans la liste, puis le détail et l'état d'envoi de chaque
destinataire, y compris une adresse en échec.

**Acceptance Scenarios**:

1. **Given** des communications envoyées, **When** un administrateur ouvre la
   rubrique, **Then** il voit la liste paginée, la plus récente en premier,
   avec date, objet, cible, auteur et nombre de destinataires.
2. **Given** une communication, **When** l'administrateur l'ouvre, **Then** il
   voit l'objet, le texte, la cible telle qu'elle était à l'envoi, et chaque
   destinataire avec l'état de son e-mail (en attente, envoyé, en échec).
3. **Given** aucune communication, **When** un administrateur ouvre la
   rubrique, **Then** un état vide explique l'usage de la rubrique et invite à
   créer une première communication.

---

### Edge Cases

- **Adhérent sans adresse de contact** : listé comme non joignable lors de la
  préparation, exclu des destinataires, sans bloquer l'envoi aux autres.
- **Fiche avec compte et adresse propre différente** : seule l'adresse du
  compte qui gère la fiche est utilisée, pour qu'une même famille ne reçoive
  pas plusieurs exemplaires.
- **Même adresse sur plusieurs fiches sans compte** : regroupées en un seul
  destinataire (comparaison sans tenir compte de la casse).
- **Compte désactivé** gérant une fiche ciblée : son adresse reste utilisée ;
  la communication concerne l'adhésion, pas l'accès à Frigg.
- **Situation qui change pendant la préparation** (un paiement constaté entre
  l'aperçu et l'envoi) : les destinataires sont déterminés au moment de la
  confirmation ; si leur nombre diffère de celui affiché, l'envoi est refusé
  et le décompte actualisé est présenté pour une nouvelle confirmation.
- **Double clic ou double soumission** : une même communication n'est envoyée
  qu'une seule fois.
- **Cible très large** (plusieurs centaines de destinataires) : l'envoi est
  accepté immédiatement ; les e-mails partent progressivement et leur état
  est suivi dans l'historique.
- **Serveur d'envoi indisponible** : la communication est enregistrée, les
  e-mails restent en attente et sont réessayés ; ceux qui échouent
  définitivement apparaissent « en échec » dans le détail.
- **Saison sans adhérent ou cible vide** : le décompte affiche 0 et l'envoi
  est impossible, avec un message explicite.
- **Membre (non administrateur)** tentant d'accéder à la rubrique ou à ses
  actions : refusé côté serveur, la rubrique n'apparaît pas dans son menu.
- **Texte contenant du balisage** : traité comme du texte ; aucun contenu
  actif ou mise en forme injectée n'est transmis aux destinataires.
- **Fichier dont l'extension ne correspond pas au contenu** (un exécutable
  renommé en `.pdf`) : refusé, le type étant vérifié sur le contenu.
- **Nom de fichier avec accents, espaces ou caractères spéciaux** : accepté et
  restitué lisiblement aux destinataires ; un nom contenant un chemin est
  réduit à son seul nom de fichier.
- **Deux pièces jointes portant le même nom** : acceptées, chacune restant
  distincte dans l'e-mail.
- **Préparation abandonnée** (page quittée sans envoyer) : les fichiers déposés
  pour cette préparation ne sont pas conservés durablement.
- **Pièce jointe refusée par la messagerie d'un destinataire** : l'e-mail de ce
  destinataire apparaît « en échec » dans le détail, sans affecter les autres.

## Requirements *(mandatory)*

### Functional Requirements

**Cible**

- **FR-001**: Les administrateurs MUST pouvoir cibler les adhérents d'une
  saison ouverte ou close, la saison courante étant proposée par défaut.
- **FR-002**: La cible MUST se définir par une ou plusieurs situations de
  paiement parmi « À jour », « Paiement annoncé » et « À payer », avec trois
  cibles prédéfinies : « Tous les adhérents » (les trois), « À jour » et
  « En impayé » (« Paiement annoncé » et « À payer »).
- **FR-003**: La cible MAY être restreinte à un ou plusieurs tarifs de la
  saison choisie ; sans restriction, tous les tarifs sont retenus.
- **FR-004**: Seuls les adhérents de la saison (adhésion validée) MUST être
  retenus ; les demandes en attente, refusées ou annulées sont exclues.
- **FR-005**: Une adhésion gratuite MUST être considérée « À jour » ; une
  adhésion dont le paiement par carte n'a pas abouti MUST être considérée
  « À payer ».

**Destinataires**

- **FR-006**: L'adresse de contact d'un adhérent MUST être celle du compte qui
  gère sa fiche, à défaut celle de la fiche ; un adhérent sans adresse de
  contact est non joignable.
- **FR-007**: Les destinataires MUST être dédoublonnés par adresse, sans tenir
  compte de la casse : une adresse reçoit au plus un exemplaire d'une
  communication.
- **FR-008**: Avant l'envoi, l'administrateur MUST voir le nombre d'adhérents
  ciblés, le nombre de destinataires et le nombre d'adhérents non joignables,
  et pouvoir consulter la liste des adhérents ciblés (prénom, nom, situation
  de paiement, adresse de contact) et celle des non joignables.
- **FR-009**: Chaque destinataire MUST recevoir un e-mail individuel ; aucune
  adresse de destinataire n'est visible des autres.

**Rédaction et envoi**

- **FR-010**: Une communication MUST comporter un objet (1 à 150 caractères) et
  un texte (1 à 10 000 caractères), en texte simple avec retours à la ligne
  préservés.
- **FR-011**: L'administrateur MUST pouvoir afficher un aperçu du message tel
  qu'il sera reçu avant de l'envoyer.
- **FR-012**: L'envoi MUST passer par un dialogue de confirmation rappelant
  l'objet, la cible, le nombre de destinataires et la liste des pièces
  jointes.
- **FR-013**: Les destinataires MUST être déterminés au moment de la
  confirmation ; si leur nombre diffère de celui présenté à l'administrateur,
  l'envoi MUST être refusé et le nouveau décompte présenté.
- **FR-014**: Une communication confirmée MUST être envoyée une seule fois,
  même en cas de double soumission.
- **FR-015**: L'envoi MUST être pris en compte immédiatement pour
  l'administrateur, les e-mails partant ensuite en arrière-plan avec nouvelles
  tentatives en cas d'échec passager.
- **FR-016**: Le message reçu MUST porter le nom de l'association comme
  expéditeur et une mention indiquant qu'il est envoyé aux adhérents de la
  saison concernée.
- **FR-017**: Le contenu saisi MUST être transmis comme du texte : aucun
  balisage ni contenu actif fourni par l'administrateur n'est interprété dans
  l'e-mail reçu.

**Pièces jointes**

- **FR-017a**: L'administrateur MUST pouvoir joindre jusqu'à 5 fichiers à une
  communication, et en retirer avant l'envoi.
- **FR-017b**: Les types acceptés MUST être : PDF, images JPEG et PNG,
  documents texte et tableurs Word (.docx), Excel (.xlsx) et OpenDocument
  (.odt, .ods). Les anciens formats (.doc, .xls) et les documents à macros
  sont refusés. Le type
  MUST être vérifié sur le contenu du fichier, pas seulement sur son nom.
- **FR-017c**: Chaque fichier MUST faire au plus 5 Mo, et l'ensemble des pièces
  jointes d'une communication au plus 10 Mo (1 Mo = 1 048 576 octets).
- **FR-017d**: Un fichier refusé MUST l'être dès son ajout, avec un message
  indiquant la règle non respectée (type, taille du fichier, taille totale,
  nombre de fichiers).
- **FR-017e**: Chaque destinataire MUST recevoir toutes les pièces jointes,
  sous leur nom d'origine et avec un contenu identique au fichier déposé.
- **FR-017f**: Les pièces jointes d'une communication envoyée MUST être
  conservées avec elle et téléchargeables par les administrateurs depuis son
  détail ; elles MUST NOT être accessibles aux membres ni aux personnes non
  connectées.
- **FR-017g**: Les fichiers déposés pour une préparation non envoyée MUST être
  supprimés au plus tard 24 heures après leur dépôt.

**Historique**

- **FR-018**: Chaque communication envoyée MUST être conservée avec sa date,
  son auteur, son objet, son texte, la description de sa cible au moment de
  l'envoi et la liste de ses destinataires.
- **FR-019**: Les administrateurs MUST pouvoir lister les communications (la
  plus récente en premier, paginées) et consulter le détail de chacune, dont
  l'état d'envoi de chaque destinataire (en attente, envoyé, en échec).
- **FR-020**: Une communication envoyée MUST NOT pouvoir être modifiée.

**Commun**

- **FR-021**: La rubrique « Communication » et toutes ses actions MUST être
  réservées aux administrateurs et refusées aux membres et aux personnes non
  connectées, côté serveur.
- **FR-022**: Le système MUST enregistrer dans les journaux de l'application
  chaque envoi de communication (auteur, date, nombre de destinataires), sans
  y faire figurer les adresses ni le contenu des messages.
- **FR-023**: Chaque erreur de saisie MUST être affichée sur le champ concerné
  avec un message en français ; après l'envoi, un message de confirmation
  annoncé aux lecteurs d'écran MUST s'afficher.
- **FR-024**: Les écrans MUST être en français, utilisables au clavier seul,
  compatibles avec les lecteurs d'écran et utilisables dès 320 pixels de
  large.

### Key Entities

- **Communication** : message envoyé par un administrateur. Attributs : objet,
  texte, auteur (compte administrateur), date d'envoi, cible (saison,
  situations de paiement, tarifs éventuels) figée à l'envoi, nombre
  d'adhérents ciblés et de destinataires, pièces jointes.
- **Pièce jointe** : fichier joint à une communication. Attributs : nom
  d'origine, type, taille, date de dépôt, auteur ; rattachée à une
  communication une fois celle-ci envoyée.
- **Destinataire d'une communication** : une adresse électronique retenue pour
  une communication, rattachée au message individuel mis en file d'envoi et à
  son état (en attente, envoyé, en échec). Elle conserve l'adresse utilisée
  même si le compte ou la fiche changent ensuite, sauf suppression du compte :
  l'adresse est alors effacée de l'historique (droit à l'effacement).
- **Adhésion** (existante, 006) et **Membre** (existant, 005) : sources de la
  cible ; elles ne sont pas modifiées par cette fonctionnalité.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Un administrateur prépare et envoie une relance des impayés
  d'une saison en moins de 3 minutes, rédaction comprise.
- **SC-002**: Le décompte des destinataires se met à jour en moins de
  2 secondes après chaque changement de cible, pour une saison de
  1 000 adhérents.
- **SC-003**: Dans 100 % des envois testés, chaque adresse ciblée reçoit
  exactement un exemplaire et aucune adresse hors cible n'en reçoit.
- **SC-004**: Une communication à 500 destinataires est acceptée
  instantanément pour l'administrateur et tous les e-mails sont partis en
  moins de 30 minutes lorsque le serveur d'envoi est disponible et ne limite
  pas le débit (avec pièces jointes, le temps de transfert vers le serveur
  d'envoi s'ajoute).
- **SC-005**: Pour toute communication, l'administrateur peut identifier en
  moins de 30 secondes les destinataires dont l'e-mail est en échec.
- **SC-006**: Dans 100 % des envois testés avec pièces jointes, chaque
  destinataire reçoit tous les documents, ouvrables et identiques aux
  fichiers déposés.
- **SC-007**: Le bureau n'a plus besoin d'exporter les adresses vers un autre
  outil pour relancer les cotisations.

## Assumptions

- Seuls les administrateurs envoient des communications ; il n'y a pas de
  rôle « communication » distinct (deux rôles seulement).
- Les communications portent sur la vie de l'association et l'adhésion
  (relances, convocations, informations) : elles relèvent de la relation
  avec l'adhérent et ne proposent pas de désinscription. Les campagnes
  promotionnelles et la gestion de désinscriptions sont hors périmètre.
- L'envoi réutilise la file d'attente d'e-mails et le serveur SMTP déjà
  configurés ; aucun nouveau prestataire d'envoi n'est introduit.
- Les limites de pièces jointes (5 fichiers, 5 Mo chacun, 10 Mo au total)
  restent sous les plafonds usuels des messageries ; un document plus lourd
  se partage par un lien inséré dans le texte.
- Pas de brouillon enregistré, de programmation différée, de mise en forme
  riche, de champs de personnalisation (prénom…) ni de suivi
  d'ouverture ou de clic dans cette version.
- Le dédoublonnage par adresse prime sur la personnalisation : un parent
  reçoit un seul message pour tous ses enfants ciblés.
- Les cibles ne portent que sur les adhérents d'une saison ; écrire à tous les
  comptes, aux demandes en attente ou à des membres sans adhésion est hors
  périmètre.
- Les réponses des destinataires arrivent à l'adresse d'expédition déjà
  configurée pour les e-mails de Frigg ; leur traitement est hors Frigg.
- Les communications sont conservées tant que l'instance existe ; leur
  purge éventuelle relève d'une fonctionnalité ultérieure liée au RGPD.
