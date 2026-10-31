# Feature Specification: Gestion des comptes par les administrateurs

**Feature Branch**: `002-admin-account-management`

**Created**: 2026-10-31

**Status**: Draft

**Input**: User description: "Gestion des comptes par les admins"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Consulter et retrouver les comptes (Priority: P1)

Dans l'espace administration, un administrateur ouvre la rubrique « Comptes ».
En plus des invitations (livrées par la fonctionnalité 001), il voit la liste
de tous les comptes de l'association : adresse électronique, rôle, état (actif
ou désactivé), adresse confirmée ou non, date de création et date de dernière
connexion. Il peut rechercher un compte par son adresse, filtrer par rôle et par
état, et ouvrir la fiche d'un compte pour en voir le détail.

**Why this priority**: toutes les autres actions de gestion partent de cette
liste ; sans elle, un administrateur ne sait même pas quels comptes existent.

**Independent Test**: avec un jeu de comptes de rôles et d'états variés, ouvrir
la rubrique, vérifier que tous les comptes apparaissent avec leurs informations,
rechercher une adresse partielle, appliquer chaque filtre, ouvrir une fiche.

**Acceptance Scenarios**:

1. **Given** un administrateur dans la rubrique « Comptes », **When** la page
   s'affiche, **Then** la liste présente tous les comptes, triés par adresse
   électronique, avec rôle, état, confirmation de l'adresse, date de création et
   date de dernière connexion (« Jamais » si aucune).
2. **Given** la liste des comptes, **When** l'administrateur saisit un fragment
   d'adresse d'au moins 3 caractères (par exemple `pont` pour
   `jean.dupont@exemple.fr`), **Then** seuls les comptes correspondants
   s'affichent, sans tenir compte de la casse ni des accents ; avec moins de
   3 caractères, une indication rappelle le minimum et la liste reste entière.
3. **Given** la liste des comptes, **When** l'administrateur filtre sur
   « Administrateurs » ou sur « Désactivés », **Then** seuls les comptes
   correspondants s'affichent, et les filtres se combinent avec la recherche.
4. **Given** plus de comptes qu'une page n'en affiche, **When** l'administrateur
   parcourt la liste, **Then** il peut passer d'une page à l'autre et connaît le
   nombre total de comptes correspondant à ses critères.
5. **Given** une recherche sans résultat, **When** la liste se met à jour,
   **Then** un message indique qu'aucun compte ne correspond et propose
   d'effacer les critères.
6. **Given** l'administrateur, **When** il ouvre la fiche d'un compte, **Then**
   il voit l'ensemble des informations du compte et les actions disponibles ;
   son propre compte y est signalé comme tel.
7. **Given** un membre connecté, **When** il tente d'obtenir la liste ou la fiche
   d'un compte, **Then** l'accès est refusé.

---

### User Story 2 - Désactiver et réactiver un compte (Priority: P1)

Lorsqu'une personne quitte l'association ou ne doit plus accéder à Frigg, un
administrateur désactive son compte depuis sa fiche. La personne est déconnectée
immédiatement de tous ses appareils et ne peut plus se connecter. Le compte et
son historique sont conservés : l'administrateur peut le réactiver plus tard.

**Why this priority**: c'est la mesure de sécurité la plus urgente (départ d'un
bénévole, appareil perdu, compte compromis) et elle est réversible, donc sans
risque de perte de données.

**Independent Test**: se connecter avec un compte membre dans un navigateur ;
dans un autre, le désactiver en tant qu'administrateur ; constater que la
première session est fermée et que la reconnexion échoue ; réactiver le compte
et constater que la connexion fonctionne à nouveau.

**Acceptance Scenarios**:

1. **Given** un compte actif, **When** l'administrateur choisit « Désactiver »
   et confirme dans le dialogue, **Then** le compte passe à l'état désactivé et
   un message confirme l'action.
2. **Given** un compte qui vient d'être désactivé et dispose de sessions
   ouvertes, **When** son titulaire effectue une action, **Then** il est renvoyé
   vers l'écran de connexion.
3. **Given** un compte désactivé, **When** son titulaire tente de se connecter
   avec le bon mot de passe, **Then** la connexion est refusée avec le même
   message que pour des identifiants incorrects.
4. **Given** un compte désactivé, **When** son titulaire demande une
   réinitialisation de mot de passe ou suit un lien reçu avant la
   désactivation, **Then** aucun accès n'est rendu possible et les réponses
   restent identiques à celles d'une adresse inconnue.
5. **Given** un compte désactivé, **When** l'administrateur choisit
   « Réactiver », **Then** le compte redevient actif et son titulaire peut se
   connecter avec son mot de passe habituel.
6. **Given** l'administrateur sur sa propre fiche, **When** il cherche à se
   désactiver, **Then** l'action n'est pas proposée et le serveur la refuse.

---

### User Story 3 - Changer le rôle d'un compte (Priority: P2)

Un administrateur promeut un membre au rôle d'administrateur (nouveau membre du
bureau) ou rétrograde un administrateur au rôle de membre (fin de mandat),
depuis la fiche du compte.

**Why this priority**: les mandats du bureau changent régulièrement ; sans ce
parcours, il faudrait supprimer et réinviter la personne.

**Independent Test**: promouvoir un membre, se connecter avec ce compte et
constater l'accès à l'espace administration ; le rétrograder et constater que
l'accès est retiré dès la requête suivante.

**Acceptance Scenarios**:

1. **Given** un compte membre, **When** l'administrateur le passe au rôle
   administrateur et confirme, **Then** le rôle est enregistré et la personne
   dispose des rubriques d'administration dès sa requête suivante.
2. **Given** un compte administrateur connecté, **When** un autre administrateur
   le rétrograde au rôle membre, **Then** dès sa requête suivante la personne
   perd l'accès aux rubriques et données d'administration, le sélecteur
   d'espace disparaît et elle est ramenée à l'accueil de l'espace membre.
3. **Given** l'administrateur sur sa propre fiche, **When** il cherche à changer
   son rôle, **Then** l'action n'est pas proposée et le serveur la refuse.
4. **Given** un rôle identique au rôle actuel, **When** l'administrateur valide,
   **Then** rien ne change et aucun message d'erreur n'est affiché.

---

### User Story 4 - Supprimer définitivement un compte (Priority: P2)

À la demande d'une personne (droit à l'effacement) ou lors d'un nettoyage, un
administrateur supprime définitivement un compte. Les données du compte sont
effacées ; les traces qui mentionnaient la personne (par exemple l'auteur d'une
invitation) ne la désignent plus.

**Why this priority**: obligation RGPD (constitution, principe VII) ; moins
fréquente que la désactivation, qui couvre la plupart des départs.

**Independent Test**: supprimer un compte ayant envoyé des invitations ;
constater qu'il n'apparaît plus dans la liste, que la connexion avec son
adresse échoue, que ses invitations restent consultables sans auteur, et que son
adresse peut de nouveau être invitée.

**Acceptance Scenarios**:

1. **Given** la fiche d'un compte, **When** l'administrateur choisit
   « Supprimer », **Then** un dialogue de confirmation explique que l'action est
   irréversible, rappelle l'adresse concernée et propose la désactivation comme
   alternative.
2. **Given** le dialogue de suppression, **When** l'administrateur confirme,
   **Then** le compte, ses sessions et ses liens en cours (réinitialisation,
   confirmation) sont supprimés, et il est ramené à la liste avec un message de
   confirmation.
3. **Given** un compte supprimé, **When** quelqu'un tente de se connecter avec
   son adresse, **Then** la réponse est identique à celle d'une adresse inconnue.
4. **Given** un compte supprimé qui avait invité d'autres personnes, **When**
   l'administrateur consulte les invitations, **Then** elles restent visibles et
   leur auteur est indiqué comme « Compte supprimé ».
5. **Given** l'adresse d'un compte supprimé, **When** un administrateur l'invite,
   **Then** l'invitation est acceptée comme pour une nouvelle personne.
6. **Given** l'administrateur sur sa propre fiche, **When** il cherche à se
   supprimer, **Then** l'action n'est pas proposée et le serveur la refuse.

---

### User Story 5 - Envoyer un lien de réinitialisation à une personne (Priority: P3)

Une personne n'arrive plus à se connecter et contacte le bureau. Depuis sa
fiche, un administrateur lui envoie un lien de réinitialisation du mot de passe,
sans jamais voir ni choisir ce mot de passe.

**Why this priority**: la personne peut déjà le faire seule depuis l'écran de
connexion ; ce parcours aide les bénévoles peu à l'aise, sans être
indispensable.

**Independent Test**: depuis la fiche d'un compte actif, envoyer le lien ;
le suivre ; définir un nouveau mot de passe ; se connecter.

**Acceptance Scenarios**:

1. **Given** la fiche d'un compte actif, **When** l'administrateur choisit
   « Envoyer un lien de réinitialisation », **Then** la personne reçoit le même
   e-mail que lors d'une demande depuis l'écran de connexion et un message
   confirme l'envoi à l'administrateur.
2. **Given** un lien envoyé par l'administrateur, **When** la personne l'utilise,
   **Then** le parcours est identique à une réinitialisation classique
   (lien à usage unique, valable 1 heure, autres sessions fermées).
3. **Given** un compte désactivé, **When** l'administrateur consulte sa fiche,
   **Then** l'action n'est pas proposée et le serveur la refuse.

---

### Edge Cases

- Deux administrateurs agissent en même temps sur le même compte (l'un le
  désactive, l'autre change son rôle) : chaque action s'applique à l'état
  courant ; la fiche affichée est rafraîchie après chaque action, et une action
  sur un compte entre-temps supprimé affiche « Ce compte n'existe plus ».
- Deux administrateurs se rétrogradent ou se désactivent mutuellement en même
  temps : l'instance conserve toujours au moins un administrateur actif (chaque
  administrateur ne peut agir que sur les autres, et l'action est refusée si elle
  retirait le dernier administrateur actif).
- Un administrateur rétrogradé ou désactivé alors qu'il a la fiche d'un autre
  compte ouverte : sa prochaine action est refusée (désactivé : renvoi vers la
  connexion ; rétrogradé : retour à l'accueil de l'espace membre avec le
  message « Vos droits d'administration ont été retirés. », comme en US3).
- Réactivation d'un compte dont l'adresse n'était pas confirmée : il reste non
  confirmé et l'alerte de confirmation s'affiche à sa prochaine connexion.
- Un e-mail de réinitialisation est en file d'attente au moment où le compte est
  désactivé ou supprimé : le lien qu'il contient est inutilisable.
- Un compte désactivé reste listé et compte comme existant : son adresse ne peut
  pas être invitée.
- Fiche d'un compte inexistant (adresse saisie à la main ou compte supprimé) :
  page « Page introuvable » dans le shell.
- Recherche contenant des caractères spéciaux ou des accents : traitée comme du
  texte, sans erreur.

## Requirements *(mandatory)*

### Functional Requirements

**Consultation**

- **FR-001**: Les administrateurs MUST pouvoir consulter, dans la rubrique
  « Comptes », la liste de tous les comptes avec adresse électronique, rôle,
  état (actif ou désactivé), confirmation de l'adresse, date de création et date
  de dernière connexion.
- **FR-002**: La liste MUST être triée par adresse électronique et paginée, avec
  le nombre total de comptes correspondant aux critères.
- **FR-003**: Les administrateurs MUST pouvoir rechercher un compte par
  n'importe quel fragment d'au moins 3 caractères de son adresse électronique
  (`pont`, `n.dup` ou `exemple` trouvent `jean.dupont@exemple.fr`), sans tenir
  compte de la casse ni des accents ; plusieurs fragments séparés par des
  espaces doivent tous correspondre. Tant que la saisie ne contient aucun
  fragment de 3 caractères, l'interface l'indique et ne filtre pas la liste.
- **FR-004**: Les administrateurs MUST pouvoir filtrer la liste par rôle (tous,
  membres, administrateurs) et par état (tous, actifs, désactivés), filtres
  combinables avec la recherche. Les critères sont reflétés dans l'adresse de
  la page, afin qu'un rechargement ou un retour arrière les conserve.
- **FR-005**: Les administrateurs MUST pouvoir ouvrir la fiche d'un compte
  présentant toutes ses informations, les actions disponibles pour ce compte et,
  le cas échéant, l'indication qu'il s'agit de leur propre compte.

**Actions de gestion**

- **FR-006**: Les administrateurs MUST pouvoir désactiver un compte actif ; la
  désactivation ferme immédiatement toutes ses sessions et invalide ses liens de
  réinitialisation et de confirmation en cours.
- **FR-007**: Le système MUST refuser la connexion d'un compte désactivé avec le
  même message que pour des identifiants incorrects, et MUST NOT envoyer de lien
  de réinitialisation à un compte désactivé, tout en conservant une réponse
  identique à celle d'une adresse inconnue.
- **FR-008**: Les administrateurs MUST pouvoir réactiver un compte désactivé ; le
  compte retrouve son rôle, son mot de passe et son état de confirmation
  antérieurs.
- **FR-009**: Les administrateurs MUST pouvoir changer le rôle d'un compte entre
  membre et administrateur ; le nouveau rôle s'applique au plus tard à la
  requête suivante du compte concerné, sans le déconnecter.
- **FR-010**: Les administrateurs MUST pouvoir supprimer définitivement un compte ;
  la suppression efface le compte, ses sessions et ses liens en cours, et les
  invitations qu'il avait envoyées sont conservées sans auteur.
- **FR-011**: Les administrateurs MUST pouvoir déclencher l'envoi d'un lien de
  réinitialisation du mot de passe à un compte actif ; ce lien suit les mêmes
  règles que celui demandé depuis l'écran de connexion. Aucun administrateur ne
  peut voir ni définir le mot de passe d'un autre compte.
- **FR-012**: La désactivation, le changement de rôle et la suppression MUST
  passer par un dialogue de confirmation qui nomme le compte concerné ; le
  dialogue de suppression précise que l'action est irréversible et propose la
  désactivation comme alternative.

**Garde-fous**

- **FR-013**: Le système MUST refuser, côté serveur, qu'un administrateur
  désactive, supprime, change le rôle ou envoie un lien de réinitialisation à
  son propre compte ; l'interface ne propose pas ces actions sur sa propre
  fiche.
- **FR-014**: Le système MUST refuser toute action qui laisserait l'instance sans
  administrateur actif, y compris lorsque deux actions concurrentes y
  conduiraient.
- **FR-015**: Le système MUST refuser, côté serveur, toute consultation ou action
  de cette fonctionnalité demandée par un membre ou par une personne non
  connectée.
- **FR-016**: Le système MUST enregistrer, pour chaque désactivation,
  réactivation, changement de rôle, suppression et envoi de lien de
  réinitialisation, l'administrateur à l'origine, le compte visé et la date,
  dans les journaux de l'application, sans y faire figurer de mot de passe ni de
  jeton.

**Interface**

- **FR-017**: Après chaque action, l'administrateur MUST voir un message de
  confirmation annoncé aux lecteurs d'écran, et la liste comme la fiche MUST
  refléter immédiatement le nouvel état.
- **FR-018**: La liste, les filtres, la fiche et les dialogues MUST être en
  français, utilisables au clavier seul, compatibles avec les lecteurs d'écran
  et utilisables dès 320 pixels de large.

### Key Entities

- **Compte** (existant, fonctionnalité 001) : complété d'un **état** (actif ou
  désactivé) et de la **date de désactivation**. Seuls les comptes actifs
  peuvent se connecter, recevoir un lien de réinitialisation et compter comme
  administrateur pour la règle FR-014.
- **Invitation** (existante) : son administrateur à l'origine peut désormais être
  absent lorsque le compte correspondant a été supprimé ; elle est alors
  présentée avec l'auteur « Compte supprimé ».
- **Session** et **Jeton à usage unique** (existants) : supprimés ou invalidés
  lors de la désactivation ou de la suppression du compte auquel ils sont
  rattachés.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Un administrateur retrouve la fiche d'un compte à partir d'une
  partie de son adresse en moins de 15 secondes, dans une association de
  2 000 comptes.
- **SC-002**: Après une désactivation, 0 page protégée et 0 donnée n'est
  accessible avec une session ouverte du compte désactivé, dès sa requête
  suivante.
- **SC-003**: Après un changement de rôle, les droits du compte concerné
  correspondent à son nouveau rôle dès sa requête suivante, dans 100 % des cas.
- **SC-004**: Aucune séquence d'actions, y compris concurrentes, ne laisse
  l'instance sans administrateur actif (vérifié par des tests automatisés).
- **SC-005**: 100 % des opérations de cette fonctionnalité sont refusées à un
  compte membre et à une personne non connectée (vérifié pour chaque opération).
- **SC-006**: Un administrateur désactive un compte, de l'ouverture de la
  rubrique à la confirmation, en moins de 30 secondes.
- **SC-007**: Aucun écran livré ne présente de défaut d'accessibilité bloquant
  lors des contrôles automatisés et d'un parcours complet au clavier.

## Assumptions

- Cette fonctionnalité prolonge la rubrique « Comptes » de la fonctionnalité 001 :
  la section des invitations y reste, la liste des comptes s'y ajoute, et les
  fiches de compte sont des pages filles de la rubrique dans le fil d'Ariane.
- Un compte correspond toujours à une personne et à une adresse ; les données
  d'adhérent (fiche membre, cotisations, familles) relèvent de fonctionnalités
  ultérieures. Quand elles existeront, la suppression d'un compte devra préciser
  leur sort ; dans cette fonctionnalité, seul le compte et ses données
  d'authentification sont concernés.
- La suppression est un effacement réel, pas une anonymisation : aucune donnée
  métier ne justifie encore de conserver une trace du compte.
- La modification par un administrateur de l'adresse électronique d'un compte,
  la création directe d'un compte sans invitation, la fermeture ciblée de
  certaines sessions et l'export des données personnelles d'un compte relèvent
  de fonctionnalités ultérieures.
- La traçabilité des actions d'administration passe par les journaux de
  l'application ; un historique consultable dans l'interface relève d'une
  fonctionnalité ultérieure.
- Une page de liste affiche 25 comptes ; ce nombre est un choix d'interface
  ajustable sans modifier la spécification.
- La règle « au moins un administrateur actif » est déjà assurée par
  l'interdiction d'agir sur son propre compte ; FR-014 la garantit en plus face
  aux actions concurrentes.
- L'interdiction d'agir sur son propre compte n'empêche pas un administrateur de
  quitter ses fonctions : un autre administrateur le rétrograde ou le désactive.
  Si l'instance n'en compte qu'un, la commande de création d'administrateur
  permet d'en ajouter un.
- Suivi : l'export des données personnelles d'un compte exigé par la
  constitution (principe VII) fera l'objet d'une spécification future « Export
  des données personnelles (RGPD) ».
