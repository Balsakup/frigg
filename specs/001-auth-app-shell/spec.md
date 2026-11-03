# Feature Specification: Authentification et shell applicatif membre/administrateur

**Feature Branch**: `001-auth-app-shell`

**Created**: 2026-10-30

**Status**: Draft

**Input**: User description: "Authentification et shell applicatif membre/administrateur"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Se connecter et accéder à son espace (Priority: P1)

Une personne disposant d'un compte (membre ou administrateur) ouvre Frigg. Tant
qu'elle n'est pas connectée, elle ne voit que l'écran de connexion. Elle saisit
son adresse électronique et son mot de passe, puis arrive sur la page d'accueil
de son espace, dans le shell commun : barre latérale avec le nom de
l'association et la navigation, en-tête, fil d'Ariane, en-tête de page et pied
de page. Elle peut se déconnecter à tout moment depuis le menu de son compte.

**Why this priority**: sans connexion, aucune autre fonctionnalité de Frigg n'est
accessible ; c'est le socle de toute l'application.

**Independent Test**: créer un compte, se connecter, vérifier l'arrivée sur la
page d'accueil dans le shell, se déconnecter, puis vérifier qu'aucune page n'est
accessible sans reconnexion.

**Acceptance Scenarios**:

1. **Given** une personne non connectée, **When** elle ouvre n'importe quelle
   adresse de l'application, **Then** elle est redirigée vers l'écran de
   connexion, puis ramenée à la page demandée après s'être connectée.
2. **Given** un compte existant, **When** la personne saisit l'adresse et le bon
   mot de passe, **Then** elle accède à la page d'accueil de son espace.
3. **Given** un compte existant, **When** la personne saisit un mauvais mot de
   passe ou une adresse inconnue, **Then** un message unique « Adresse
   électronique ou mot de passe incorrect » s'affiche, sans indiquer lequel des
   deux est faux.
4. **Given** une personne connectée, **When** elle choisit « Se déconnecter »,
   **Then** sa session est fermée et toute navigation ultérieure la ramène à
   l'écran de connexion, y compris avec le bouton « Précédent » du navigateur.
5. **Given** une personne connectée et inactive au-delà de la durée de session,
   **When** elle effectue une action, **Then** elle est renvoyée vers l'écran de
   connexion avec un message expliquant que sa session a expiré.

---

### User Story 2 - Navigation adaptée au rôle (Priority: P1)

Le shell est identique pour tous, mais le menu dépend du rôle. Un membre voit
uniquement les rubriques de l'espace membre. Un administrateur, qui peut faire
tout ce que fait un membre, dispose en haut de la barre latérale d'un sélecteur
d'espace « Administration / Membre » : dans l'espace administration il voit les
rubriques d'administration, dans l'espace membre il voit exactement ce que voit
un membre.

**Why this priority**: le contrôle d'accès par rôle est un principe fondateur ;
il doit être en place avant d'ajouter la moindre rubrique métier.

**Independent Test**: se connecter successivement avec un compte membre et un
compte administrateur ; comparer les menus ; tenter d'ouvrir directement
l'adresse d'une rubrique d'administration avec le compte membre.

**Acceptance Scenarios**:

1. **Given** un membre connecté, **When** il regarde la barre latérale, **Then**
   il ne voit ni sélecteur d'espace ni rubrique d'administration.
2. **Given** un membre connecté, **When** il saisit directement l'adresse d'une
   rubrique d'administration, **Then** une page « Accès refusé » s'affiche dans
   le shell et aucune donnée d'administration n'est transmise.
3. **Given** un administrateur connecté, **When** il utilise le sélecteur
   d'espace, **Then** la navigation bascule entre les rubriques d'administration
   et les rubriques membres, et il arrive sur l'accueil de l'espace choisi.
4. **Given** un administrateur, **When** il se connecte, **Then** il arrive dans
   l'espace administration ; son dernier espace choisi est conservé lors de sa
   prochaine visite sur le même appareil.
5. **Given** un administrateur dans l'espace membre, **When** il ouvre
   directement l'adresse d'une rubrique d'administration, **Then** la rubrique
   s'affiche et le sélecteur passe sur l'espace administration.
6. **Given** un utilisateur connecté, **When** il ouvre une rubrique, **Then**
   l'élément actif est mis en évidence dans le menu, le fil d'Ariane et le titre
   de la page sont à jour.
7. **Given** un écran large, **When** l'utilisateur replie la barre latérale,
   **Then** elle n'affiche plus que les icônes avec une infobulle pour chacune,
   et ce choix est conservé lors de sa prochaine visite sur le même appareil.
8. **Given** un écran de téléphone, **When** l'utilisateur ouvre le menu,
   **Then** la barre latérale s'affiche par-dessus le contenu et se ferme par le
   bouton de fermeture, la touche Échap ou un appui en dehors.

---

### User Story 3 - Initialiser l'instance avec un premier administrateur (Priority: P1)

La personne qui installe Frigg pour l'association crée le premier compte
administrateur depuis le serveur, en ligne de commande, en saisissant une
adresse électronique et un mot de passe masqué à la saisie.

**Why this priority**: aucune création de compte n'est possible depuis
l'interface sans un premier administrateur ; c'est la condition de démarrage.

**Independent Test**: sur une instance vierge, lancer la commande, puis se
connecter avec le compte créé et constater l'accès aux rubriques
d'administration.

**Acceptance Scenarios**:

1. **Given** une instance vierge, **When** l'installateur lance la commande de
   création d'administrateur et saisit une adresse et un mot de passe valides
   (deux fois), **Then** le compte administrateur est créé, considéré comme
   vérifié, et peut se connecter.
2. **Given** une adresse déjà utilisée par un compte, **When** la commande est
   lancée avec cette adresse, **Then** elle échoue avec un message explicite et
   ne modifie rien.
3. **Given** un mot de passe non conforme à la règle (FR-008) ou deux saisies
   différentes, **When** la commande est lancée, **Then** elle refuse et
   explique la règle.
4. **Given** une instance dont le nom d'association n'est pas encore défini,
   **When** la commande est lancée, **Then** elle demande aussi le nom de
   l'association, qui s'affiche ensuite dans le shell ; si le nom est déjà
   défini, elle ne le redemande pas.

---

### User Story 4 - Inviter une personne à créer son compte (Priority: P2)

Dans l'espace administration, un administrateur ouvre la rubrique « Comptes »,
saisit l'adresse électronique d'une personne et choisit son rôle (membre par
défaut, ou administrateur). La personne reçoit un e-mail d'invitation contenant
un lien ; en le suivant, elle choisit son mot de passe et arrive connectée dans
son espace. La rubrique liste les invitations en attente, que l'administrateur
peut renvoyer ou annuler.

**Why this priority**: c'est le seul moyen d'ouvrir des comptes depuis
l'interface ; nécessaire pour faire vivre l'application, mais le socle
(connexion, shell) est utilisable sans.

**Independent Test**: inviter une adresse, suivre le lien reçu, choisir un mot
de passe, constater l'arrivée connectée dans l'espace correspondant au rôle
choisi ; vérifier que le lien ne fonctionne qu'une fois.

**Acceptance Scenarios**:

1. **Given** un administrateur dans la rubrique « Comptes », **When** il invite
   une adresse inconnue avec le rôle membre, **Then** un e-mail d'invitation est
   envoyé et l'invitation apparaît dans la liste des invitations en attente.
2. **Given** une adresse déjà associée à un compte, **When** l'administrateur
   l'invite, **Then** l'invitation est refusée avec un message indiquant que ce
   compte existe déjà.
3. **Given** un lien d'invitation valide, **When** la personne choisit un mot de
   passe conforme, **Then** son compte est créé avec le rôle prévu et une adresse
   confirmée, et elle arrive connectée sur l'accueil de son espace.
4. **Given** un lien d'invitation expiré, annulé ou déjà utilisé, **When** la
   personne l'ouvre, **Then** un message l'informe que l'invitation n'est plus
   valable et l'invite à contacter l'association.
5. **Given** une invitation en attente, **When** l'administrateur la renvoie,
   **Then** un nouveau lien est envoyé et le précédent devient invalide.
6. **Given** un membre connecté, **When** il tente d'accéder à la rubrique
   « Comptes » ou d'envoyer une invitation, **Then** l'action est refusée.

---

### User Story 5 - Réinitialiser un mot de passe oublié (Priority: P2)

Depuis l'écran de connexion, une personne qui a oublié son mot de passe demande
un lien de réinitialisation. Elle reçoit un e-mail contenant un lien à usage
unique, choisit un nouveau mot de passe et peut se connecter avec.

**Why this priority**: sans ce parcours, chaque oubli exige l'intervention d'un
administrateur ; important mais l'application reste utilisable sans.

**Independent Test**: demander une réinitialisation, suivre le lien reçu,
définir un nouveau mot de passe, se connecter avec ; vérifier que l'ancien ne
fonctionne plus et que le lien ne fonctionne qu'une fois.

**Acceptance Scenarios**:

1. **Given** l'écran de connexion, **When** la personne demande une
   réinitialisation pour une adresse, **Then** le même message de confirmation
   s'affiche que l'adresse corresponde à un compte ou non.
2. **Given** un lien de réinitialisation valide, **When** la personne définit un
   nouveau mot de passe conforme, **Then** il remplace l'ancien, toutes ses
   autres sessions ouvertes sont fermées, et elle peut se connecter.
3. **Given** un lien déjà utilisé ou expiré, **When** la personne l'ouvre,
   **Then** un message l'informe que le lien n'est plus valide et lui propose
   d'en demander un nouveau.

---

### User Story 6 - Confirmer son adresse électronique (Priority: P2)

Une personne dont l'adresse n'est pas encore confirmée voit, en tête de chaque
page, une alerte l'invitant à la confirmer, avec la possibilité de renvoyer le
lien. En suivant le lien reçu par e-mail, son adresse est confirmée et l'alerte
disparaît.

**Why this priority**: une adresse confirmée conditionne la réception des
e-mails de l'association (réinitialisation, messages futurs) ; non bloquant pour
naviguer.

**Independent Test**: avec un compte non vérifié, constater l'alerte, renvoyer
le lien, le suivre, constater la disparition de l'alerte.

**Acceptance Scenarios**:

1. **Given** un compte dont l'adresse n'est pas confirmée, **When** son titulaire
   navigue, **Then** l'alerte de confirmation est visible en tête du contenu de
   chaque page.
2. **Given** l'alerte affichée, **When** la personne clique sur « Renvoyer le
   lien », **Then** un nouveau lien est envoyé, les liens précédents deviennent
   invalides, et un message de confirmation s'affiche.
3. **Given** un lien de confirmation valide, **When** la personne l'ouvre,
   **Then** son adresse est confirmée, y compris si elle n'est pas connectée à
   ce moment-là.

---

### User Story 7 - Gérer son mot de passe depuis « Mon compte » (Priority: P3)

Depuis le menu du compte, une personne connectée ouvre « Mon compte » et change
son mot de passe en saisissant l'actuel puis le nouveau.

**Why this priority**: confort et sécurité, mais la réinitialisation par e-mail
couvre déjà le besoin minimal.

**Independent Test**: changer son mot de passe, se déconnecter, se reconnecter
avec le nouveau.

**Acceptance Scenarios**:

1. **Given** une personne connectée, **When** elle saisit son mot de passe actuel
   correct et un nouveau mot de passe conforme, **Then** le changement est
   enregistré et ses autres sessions ouvertes sont fermées.
2. **Given** un mot de passe actuel erroné, **When** elle valide, **Then** le
   changement est refusé avec un message associé au champ concerné.

---

### Edge Cases

- Trop de tentatives de connexion échouées pour une même adresse ou depuis une
  même origine : les tentatives sont temporairement bloquées avec un message
  indiquant de réessayer plus tard, sans révéler si le compte existe.
- Demandes répétées de réinitialisation ou de renvoi du lien de confirmation :
  même limitation, même message neutre.
- Compte désactivé par un administrateur (fonctionnalité future) : la connexion
  est refusée avec le même message que pour des identifiants incorrects, et ses
  sessions ouvertes sont fermées.
- Changement de rôle d'un compte connecté (fonctionnalité future) : le menu et
  les droits reflètent le nouveau rôle au plus tard à la requête suivante.
- Adresse saisie avec des majuscules ou des espaces superflus : elle est
  reconnue comme la même adresse.
- Serveur d'e-mails indisponible : la demande de l'utilisateur aboutit (message
  de confirmation), l'envoi est retenté automatiquement plus tard.
- Adresse inconnue de l'application (page inexistante) : une page « Page
  introuvable » s'affiche dans le shell pour une personne connectée.
- Connexion déjà active : ouvrir l'écran de connexion renvoie directement vers
  l'accueil.

## Requirements *(mandatory)*

### Functional Requirements

**Accès et session**

- **FR-001**: Le système MUST exiger une session valide pour toute page et toute
  donnée, à l'exception de l'écran de connexion, de la demande et de la saisie
  d'un nouveau mot de passe, de la confirmation d'adresse électronique et de
  l'acceptation d'une invitation.
- **FR-002**: Les personnes MUST pouvoir se connecter avec leur adresse
  électronique et leur mot de passe.
- **FR-003**: Le système MUST afficher un message d'échec identique, que
  l'adresse soit inconnue ou le mot de passe erroné.
- **FR-004**: Le système MUST ramener la personne, après connexion, vers la page
  qu'elle avait demandée, à condition qu'elle appartienne à l'application.
- **FR-005**: Les personnes connectées MUST pouvoir se déconnecter ; la
  déconnexion invalide la session immédiatement.
- **FR-006**: Le système MUST faire expirer une session après 7 jours
  d'inactivité, et dans tous les cas 30 jours après la connexion.
- **FR-007**: Le système MUST limiter les tentatives de connexion, de demande de
  réinitialisation et de renvoi du lien de confirmation, par adresse et par
  origine, et répondre par un message neutre en cas de dépassement.

**Mots de passe et adresses**

- **FR-008**: Le système MUST exiger un mot de passe d'au moins 8 caractères,
  comprenant au moins une lettre majuscule et au moins un chiffre ; la règle est
  affichée à côté du champ et chaque critère non respecté est signalé.
- **FR-009**: Les personnes MUST pouvoir demander un lien de réinitialisation ;
  ce lien est à usage unique et expire après 1 heure.
- **FR-010**: Le système MUST fermer toutes les autres sessions d'un compte après
  un changement ou une réinitialisation de son mot de passe.
- **FR-011**: Le système MUST envoyer un lien de confirmation d'adresse, valable
  48 heures, et permettre d'en redemander un ; seul le plus récent est valable.
- **FR-012**: Le système MUST traiter les adresses électroniques sans tenir
  compte de la casse ni des espaces en début et fin.
- **FR-013**: Les personnes connectées MUST pouvoir changer leur mot de passe en
  fournissant leur mot de passe actuel.

**Rôles et comptes**

- **FR-014**: Le système MUST connaître exactement deux rôles, membre et
  administrateur ; un administrateur dispose de tous les droits d'un membre.
- **FR-015**: Le système MUST refuser, côté serveur, toute action ou donnée
  d'administration demandée par un membre, quel que soit le moyen d'accès.
- **FR-016**: Le système MUST permettre de créer un compte administrateur depuis
  la ligne de commande du serveur, avec un mot de passe masqué à la saisie ; ce
  compte est considéré comme ayant une adresse confirmée.
- **FR-017**: Les administrateurs MUST pouvoir inviter une personne par son
  adresse électronique, en choisissant son rôle (membre par défaut ou
  administrateur) ; l'invitation est envoyée par e-mail avec un lien à usage
  unique valable 7 jours.
- **FR-017a**: En suivant un lien d'invitation valide, la personne MUST pouvoir
  choisir son mot de passe ; son compte est alors créé avec le rôle prévu et une
  adresse confirmée, et elle est connectée.
- **FR-017b**: Les administrateurs MUST pouvoir consulter les invitations en
  attente, les renvoyer (le lien précédent devient invalide) et les annuler.
- **FR-017c**: Le système MUST refuser d'inviter une adresse déjà associée à un
  compte ou faisant déjà l'objet d'une invitation en attente.

**Shell applicatif**

- **FR-018**: Le système MUST présenter, pour toute page connectée, le même
  shell : barre latérale (identité de l'association, navigation principale,
  navigation secondaire, menu du compte), en-tête, fil d'Ariane, en-tête de page
  (titre, introduction, actions), zone de contenu et pied de page.
- **FR-019**: Le système MUST n'afficher dans la navigation que les rubriques
  autorisées pour le rôle de la personne connectée.
- **FR-019a**: Le système MUST proposer aux seuls administrateurs un sélecteur
  d'espace « Administration / Membre » qui bascule la navigation ; l'espace
  administration est l'espace par défaut et le dernier choix est mémorisé sur
  l'appareil.
- **FR-020**: Le système MUST permettre de replier la barre latérale en icônes
  sur grand écran et mémoriser ce choix sur l'appareil ; sur petit écran, la
  barre latérale s'ouvre en surimpression.
- **FR-021**: Le menu du compte MUST indiquer l'identité et le rôle de la
  personne connectée et proposer « Mon compte » et « Se déconnecter ».
- **FR-022**: Le système MUST afficher en tête du contenu une alerte permanente
  tant que l'adresse électronique n'est pas confirmée, avec une action de renvoi
  du lien.
- **FR-023**: Le système MUST afficher, dans le shell, une page « Accès refusé »
  pour une rubrique non autorisée et une page « Page introuvable » pour une
  adresse inexistante.
- **FR-024**: L'en-tête MUST comporter le bouton d'ouverture et de repli de la
  barre latérale. La recherche globale et le panneau de notifications de
  l'artefact n'y figurent pas dans cette fonctionnalité ; ils seront ajoutés par
  les fonctionnalités qui leur fourniront du contenu, sans changer la
  disposition du reste du shell.
- **FR-025**: Les rubriques métier de la navigation (membres, adhésions,
  événements…) MUST s'afficher comme des pages vides « bientôt disponible » tant
  que les fonctionnalités correspondantes n'existent pas ; seules les rubriques
  « Accueil » (espace membre), « Tableau de bord » (espace administration),
  « Comptes » (administration) et « Mon compte »
  sont livrées par cette fonctionnalité.
- **FR-026**: L'ensemble des écrans MUST être en français, utilisable au clavier
  seul, compatible avec les lecteurs d'écran et utilisable dès 320 pixels de
  large.

### Key Entities

- **Compte** : une personne pouvant se connecter. Adresse électronique (unique),
  mot de passe (jamais conservé en clair), rôle (membre ou administrateur),
  adresse confirmée ou non, date de création, date de dernière connexion.
- **Session** : la connexion ouverte d'un compte sur un appareil. Rattachée à un
  compte, avec une date d'expiration ; plusieurs sessions simultanées possibles.
- **Jeton à usage unique** : lien envoyé par e-mail pour réinitialiser un mot de
  passe, confirmer une adresse ou accepter une invitation. Rattaché à un compte, avec un objet, une date
  d'expiration et un état utilisé ou non.
- **E-mail en attente** : message à envoyer (réinitialisation, confirmation,
  invitation),
  avec son destinataire, son état d'envoi et le nombre de tentatives.
- **Association** : l'unique association de l'instance ; dans cette
  fonctionnalité, seul son nom est utilisé, pour l'affichage dans le shell. Il
  est enregistré par l'application et pourra être modifié depuis la future page
  de paramètres.
- **Invitation** : proposition de création de compte envoyée par un
  administrateur. Adresse électronique, rôle prévu, administrateur à l'origine,
  date d'expiration, état (en attente, acceptée, annulée, expirée).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Une personne disposant de ses identifiants accède à sa page
  d'accueil en moins de 30 secondes depuis l'ouverture de l'application.
- **SC-002**: 100 % des pages et données d'administration sont refusées à un
  compte membre et à une personne non connectée (vérifié pour chaque rubrique).
- **SC-003**: Une personne ayant oublié son mot de passe retrouve l'accès en
  moins de 5 minutes, e-mail compris, dans des conditions normales d'envoi.
- **SC-004**: Après une déconnexion ou une réinitialisation de mot de passe,
  0 page protégée n'est accessible avec l'ancienne session.
- **SC-005**: Aucun écran livré ne présente de défaut d'accessibilité bloquant
  lors des contrôles automatisés et d'un parcours complet au clavier.
- **SC-006**: L'installateur crée le premier administrateur et se connecte en
  moins de 2 minutes après l'installation.
- **SC-007**: Les pages du shell s'affichent et restent utilisables, sans
  défilement horizontal, sur un écran de 320 pixels de large.

## Assumptions

- Pas d'inscription libre : un compte naît uniquement de la commande de création
  d'administrateur ou d'une invitation (constitution, principe II). La page
  d'acceptation d'une invitation fait partie des parcours d'obtention d'une
  session autorisés sans connexion.
- Un compte correspond à une personne et à une seule adresse électronique ; la
  gestion de plusieurs membres par compte (familles) relève d'une fonctionnalité
  ultérieure.
- Le nom de l'association est enregistré par l'application. Tant que la page de
  paramètres n'existe pas, il est demandé par la commande de création
  d'administrateur s'il n'est pas encore défini, et « Mon association »
  s'affiche à défaut ; la page de paramètres permettra de le modifier. Logo et
  couleurs personnalisées relèvent de cette même fonctionnalité de paramétrage ;
  les couleurs par défaut de l'artefact s'appliquent.
  *Note (2026-11-03)* : livré par
  [010-general-settings](../010-general-settings/spec.md). Le nom, le logo et
  les couleurs de l'association remplacent désormais l'identité de Frigg sur
  les écrans hors session (amendement 1.8.0 du principe II).
- Au-delà de l'invitation, la gestion des comptes par un administrateur
  (liste des comptes, désactivation, changement de rôle, suppression) relève
  d'une fonctionnalité ultérieure ; les cas limites
  correspondants sont décrits pour que le socle les supporte.
- Le changement d'adresse électronique depuis « Mon compte » relève d'une
  fonctionnalité ultérieure.
- Le pied de page affiche le nom de l'association et la version de Frigg. Les
  liens légaux de l'artefact (mentions légales, données personnelles,
  accessibilité) seront ajoutés avec leurs pages, dans une fonctionnalité
  ultérieure.
- Dans cette fonctionnalité, tous les comptes naissent avec une adresse confirmée
  (commande de création d'administrateur, acceptation d'une invitation).
  L'alerte et le renvoi du lien (User Story 6) préparent le changement d'adresse
  électronique, prévu dans une fonctionnalité ultérieure ; ils sont vérifiés
  avec un compte non confirmé créé par le jeu de données de test.
- Pas de double authentification dans cette version.
- Un serveur d'e-mails est configuré sur l'instance ; sans lui, les parcours de
  réinitialisation et de confirmation ne peuvent aboutir.
- Le rendu visuel suit l'artefact de référence désigné par la constitution
  (principe IV), en thème clair uniquement.
