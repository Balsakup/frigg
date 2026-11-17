# Feature Specification: Parcours publics (adhésion et inscription sans compte)

**Feature Branch**: `016-public-journeys`

**Created**: 2026-11-17

**Status**: Draft

**Input**: User description: "Parcours publics : demande d'adhésion et
inscription aux événements sans être connecté. Une personne extérieure peut
demander à adhérer ou s'inscrire à un événement sans compte, choisir d'en
créer un ou non, puis suivre et payer sa demande depuis un lien personnel."

## Vocabulaire

- **Personne extérieure** : une personne qui n'est pas connectée à Frigg,
  qu'elle ait ou non un compte.
- **Parcours public** : un écran ou une action de Frigg accessible sans
  session. La liste des parcours publics de cette fonctionnalité est donnée
  en FR-001 ; chacun exige un amendement du principe II de la constitution.
- **Page d'adhésion publique** : la page, à adresse stable, où une personne
  extérieure dépose une demande d'adhésion. L'association la diffuse
  elle-même (site, réseaux, affiche).
- **Lien public d'un événement** : l'adresse, non devinable, de la page
  publique d'un événement ouvert au public. Seul l'administrateur la connaît
  et la diffuse ; aucun listing ne la révèle.
- **Ouverture « Ouvert au public »** : nouvelle valeur d'ouverture d'un
  événement (012), en plus de « Adhérents de la saison » et « Toutes les
  fiches ». Toute fiche y est éligible, et une personne extérieure peut s'y
  inscrire depuis le lien public.
- **Dépôt** : une demande d'adhésion ou une inscription envoyée depuis un
  parcours public, tant que son adresse électronique n'est pas confirmée. Un
  dépôt n'est visible de personne et ne produit aucun effet (ni fiche, ni
  adhésion, ni place réservée).
- **Adresse confirmée** : l'adresse électronique saisie dans un dépôt, une
  fois que la personne a suivi le lien de confirmation reçu à cette adresse.
  La confirmation transforme le dépôt en demande d'adhésion en attente (006)
  ou en inscription confirmée (012).
- **Origine « Publique »** : marque portée par une fiche, une adhésion ou une
  inscription créée par un parcours public, visible des administrateurs.
- **Lien personnel** : lien secret, envoyé uniquement par e-mail à une
  adresse, qui donne accès sans compte à l'**espace de suivi** de cette
  adresse. Il a une durée de validité limitée et peut être redemandé. La
  possession de la messagerie tient lieu d'identification.
- **Fiche suivie par lien** : toute fiche « Sans compte » dont l'adresse
  électronique est celle du lien, quelle que soit son origine (parcours
  public, import 013, saisie par un administrateur).
- **Espace de suivi** : écran accessible par lien personnel, qui rassemble
  les fiches suivies par lien d'une adresse, avec leurs adhésions et leurs
  inscriptions, leur état, leur paiement et les actions permises.
- **Fiche publique inactive** : fiche d'origine « Publique » sans compte,
  créée il y a plus de 12 mois. Elle ne porte aucune demande en attente,
  aucune adhésion validée d'une saison terminée depuis moins de 12 mois, et
  aucune inscription confirmée à un événement terminé depuis moins de
  12 mois. Elle est signalée aux administrateurs pour suppression.

Les autres termes (compte, fiche, compte gestionnaire, « Sans compte »,
« Vous », saison, tarif, adhésion, moyen de paiement, état du paiement,
référence de paiement, carte bancaire, HelloAsso, événement, ouverture,
inscription, capacité, adresse de contact, champ personnalisé, montant à
payer) sont ceux des spécifications 001 à 014.

## Clarifications

### Session 2026-11-17

- Q: Quand l'adresse d'un dépôt est déjà associée à un compte, que se
  passe-t-il à la confirmation ? → A: La fiche est rattachée automatiquement
  à ce compte : la possession de l'adresse vaut preuve. La demande ou
  l'inscription apparaît dans l'espace membre du compte (FR-033).
- Q: Le lien personnel est-il réservé aux fiches d'origine publique ? → A:
  Non. Toute fiche « Sans compte » qui porte une adresse électronique est
  suivie par lien, y compris les fiches importées (013) ou saisies par un
  administrateur. Elle reçoit les e-mails et peut payer en ligne (FR-014,
  FR-035, FR-040).
- Q: Que deviennent les fiches de participants ponctuels ou de demandes
  refusées ? → A: Elles sont signalées aux administrateurs pour suppression
  12 mois après leur dernière activité. Aucune suppression n'est automatique
  (FR-041).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Demander une adhésion sans compte (Priority: P1)

Une personne découvre l'association et ouvre la page d'adhésion publique.
Elle y voit le nom et le logo de l'association, la saison ouverte et ses
tarifs. Elle saisit son identité (prénom, nom, date de naissance), son
adresse électronique, et en option son téléphone et son adresse postale.
Elle choisit un tarif et un type, remplit les champs personnalisés « Rempli
par le membre » et donne son consentement au traitement de ses données. Elle
reçoit un e-mail de confirmation. Après avoir suivi le lien, sa demande
apparaît aux administrateurs comme toute demande en attente (006). Un
administrateur la valide ou la refuse.

**Why this priority**: c'est le cœur du besoin. Aujourd'hui, une personne
extérieure ne peut pas adhérer sans qu'un administrateur l'invite ou saisisse
sa fiche et son bulletin à sa place.

**Independent Test**: l'accueil des demandes publiques est activé et une
saison ouverte porte un tarif. Déposer une demande depuis une navigation
privée, confirmer l'adresse, puis constater que la demande en attente, d'origine
« Publique », apparaît dans la rubrique « Adhésions » de l'administration et
peut être validée.

**Acceptance Scenarios**:

1. **Given** l'accueil des demandes publiques activé et une saison ouverte,
   non close, portant au moins un tarif, **When** une personne non connectée
   ouvre la page d'adhésion publique, **Then** elle voit le nom, le logo et
   les couleurs de l'association, la saison, et pour chaque tarif son
   libellé, son ou ses montants et sa description, puis un formulaire de
   demande.
2. **Given** le formulaire rempli avec des données valides et le
   consentement coché, **When** la personne l'envoie, **Then** un message lui
   indique de consulter sa messagerie. Ce message est identique quelle que
   soit l'adresse saisie. Un e-mail contenant un lien de confirmation est
   envoyé à l'adresse saisie, et aucune fiche ni adhésion n'apparaît à
   l'administration.
3. **Given** un dépôt non confirmé, **When** la personne suit le lien de
   confirmation dans son délai de validité, **Then** une fiche « Sans
   compte » d'origine « Publique » est créée avec l'adresse confirmée comme
   adresse électronique, ainsi qu'une demande d'adhésion « En attente de
   validation ». La personne arrive sur son espace de suivi, et un e-mail lui
   fournit son lien personnel.
4. **Given** un lien de confirmation déjà utilisé, **When** la personne le
   suit à nouveau, **Then** aucune nouvelle demande n'est créée, et la page
   l'invite à demander un lien personnel.
5. **Given** un dépôt jamais confirmé, **When** son délai de confirmation
   est dépassé, **Then** le lien est refusé avec un message invitant à
   recommencer la demande, et le dépôt est supprimé au plus tard 7 jours
   après son envoi.
6. **Given** une demande publique en attente, **When** un administrateur
   l'ouvre, **Then** il voit l'origine « Publique », l'adresse confirmée, les
   champs saisis, et le signalement d'un doublon probable si une fiche
   existante a les mêmes prénom, nom et date de naissance (005 FR-005).
7. **Given** une demande publique en attente avec un doublon probable,
   **When** l'administrateur choisit de la rattacher à la fiche existante
   puis la valide, **Then** l'adhésion est portée par la fiche existante, la
   fiche créée par le dépôt est supprimée, et les règles d'unicité de 006
   FR-003 sont vérifiées sur la fiche existante.
8. **Given** l'accueil des demandes publiques désactivé, ou aucune saison
   ouverte portant un tarif, **When** une personne ouvre la page d'adhésion
   publique, **Then** elle voit seulement l'identité visuelle et un message
   indiquant que les demandes d'adhésion ne sont pas ouvertes, et tout dépôt
   est refusé par le serveur.
9. **Given** une personne déjà connectée, **When** elle ouvre la page
   d'adhésion publique, **Then** un lien l'invite à faire sa demande depuis
   son espace membre (006).

---

### User Story 2 - Suivre et payer sa demande par lien personnel (Priority: P1)

Une personne sans compte suit son lien personnel. Elle voit l'état de sa
demande d'adhésion et peut l'annuler tant qu'elle est en attente. Une fois la
demande validée, elle reçoit un e-mail contenant un lien personnel vers sa
page de paiement. Elle y choisit un moyen de paiement activé : espèces,
chèque ou virement, avec les informations correspondantes, ou carte bancaire
via HelloAsso. Le paiement est constaté comme pour un adhérent connecté. Si
son lien a expiré, elle en redemande un en saisissant son adresse.

**Why this priority**: une demande publique n'a de valeur que si la personne
peut la suivre et la payer sans compte. Sans ce parcours, l'administrateur
devrait relancer chaque personne par un autre canal.

**Independent Test**: depuis une demande publique validée et non gratuite,
ouvrir le lien personnel reçu par e-mail, payer par carte en mode test, puis
constater que l'adhésion passe à « Payée » et qu'un e-mail de confirmation est
envoyé à l'adresse confirmée.

**Acceptance Scenarios**:

1. **Given** un lien personnel valide, **When** la personne le suit, **Then**
   elle arrive sur son espace de suivi. Celui-ci liste uniquement les fiches
   suivies par lien de son adresse, avec leurs adhésions et leurs
   inscriptions. Pour chacune, elle voit l'état, le tarif ou l'événement, le
   montant à payer, l'état du paiement et le motif d'un refus ou d'une
   annulation.
2. **Given** une demande publique en attente, **When** la personne l'annule
   depuis son espace de suivi après confirmation, **Then** la demande passe à
   « Annulée », comme une annulation par le compte gestionnaire (006 FR-009).
3. **Given** une demande publique, **When** un administrateur la valide, la
   refuse, l'annule ou constate son paiement, **Then** un e-mail est envoyé à
   l'adresse confirmée, avec un lien personnel. Pour une validation non
   gratuite, ce lien mène à la page de paiement.
4. **Given** une adhésion publique validée « À payer » ou « Paiement
   annoncé », **When** la personne ouvre sa page de paiement par son lien
   personnel, **Then** elle voit le montant à payer, la référence et les
   moyens activés. Elle peut annoncer un moyen et en changer tant que le
   paiement n'est pas constaté, avec les mêmes règles que 006 FR-022b.
5. **Given** la carte bancaire activée, **When** la personne choisit de payer
   par carte, **Then** elle est informée du départ vers HelloAsso et de la
   contribution volontaire facultative. Le paiement est prérempli avec
   l'adresse confirmée et le prénom et le nom de la fiche, sans autre donnée
   personnelle. Au retour, elle revient sur sa page de paiement, sans avoir à
   se connecter.
6. **Given** un paiement par carte réussi, **When** HelloAsso le notifie ou
   que la personne revient, **Then** l'adhésion passe à « Payée » avec toutes
   les garanties de 008 : constatation après vérification, constatation
   unique, constatation même sans retour, signalement des anomalies, mention
   « Mode test ».
7. **Given** un lien personnel expiré ou invalide, **When** la personne le
   suit, **Then** elle voit un message l'invitant à demander un nouveau lien,
   sans aucune donnée.
8. **Given** la page « Recevoir mon lien de suivi », **When** une personne
   saisit une adresse, **Then** le même message neutre s'affiche dans tous
   les cas. Un e-mail contenant un lien personnel est envoyé seulement si
   l'adresse est celle d'au moins une fiche suivie par lien.
9. **Given** une personne qui a deux fiches sans compte sous la même adresse,
   par exemple elle-même et son enfant, **When** elle ouvre son espace de
   suivi, **Then** elle voit les adhésions et inscriptions des deux fiches,
   chacune nommée.
10. **Given** une fiche importée (013) sans compte, avec une adresse et une
    adhésion validée « À payer », **When** un administrateur valide ou
    constate une adhésion de cette fiche, **Then** l'e-mail correspondant
    part à l'adresse de la fiche avec un lien personnel. La personne peut
    alors payer en ligne sans compte, comme pour une demande publique.

---

### User Story 3 - S'inscrire à un événement public par son lien (Priority: P2)

Un administrateur ouvre un événement au public et partage son lien public. Une
personne extérieure ouvre ce lien, voit les détails de l'événement et
s'inscrit en saisissant son identité et son adresse. Après confirmation de son
adresse, son inscription est confirmée immédiatement et sa place est
réservée, sans validation par un administrateur. Si l'événement est payant,
elle paie depuis son lien personnel. L'administrateur peut retirer
l'inscription au besoin.

**Why this priority**: les événements ouverts (portes ouvertes, stage
découverte, tournoi) sont le premier contact de beaucoup de futurs adhérents.
Ce parcours dépend toutefois des mêmes fondations que les stories 1 et 2.

**Independent Test**: publier un événement payant « Ouvert au public » avec
une capacité de 2, s'y inscrire depuis le lien public, confirmer l'adresse,
constater l'inscription confirmée et la place décomptée, puis payer par le
lien personnel.

**Acceptance Scenarios**:

1. **Given** un événement en brouillon ou publié, **When** l'administrateur
   choisit l'ouverture « Ouvert au public », **Then** la page de l'événement
   en administration affiche son lien public, copiable, avec la mention que
   ce lien est accessible à toute personne qui le reçoit.
2. **Given** un événement publié « Ouvert au public », **When** une personne
   non connectée ouvre son lien public, **Then** elle voit l'identité
   visuelle de l'association, le titre, les dates, le lieu, la description,
   le prix ou « Gratuit », la période d'inscription, les places restantes si
   une capacité est fixée, et un formulaire d'inscription si les inscriptions
   sont ouvertes. Elle ne voit aucun inscrit.
3. **Given** un événement en brouillon, ou dont l'ouverture n'est pas
   « Ouvert au public », ou un lien public inconnu, **When** une personne
   ouvre ce lien, **Then** elle reçoit la même page « Événement introuvable ».
4. **Given** un événement public annulé, terminé, complet ou hors période
   d'inscription, **When** la personne ouvre son lien, **Then** elle voit
   l'événement avec la raison pour laquelle l'inscription n'est pas possible,
   sans formulaire.
5. **Given** le formulaire d'inscription rempli (prénom, nom, date de
   naissance, adresse électronique, consentement), **When** la personne
   l'envoie, **Then** le même message neutre l'invite à consulter sa
   messagerie, et aucune place n'est encore réservée.
6. **Given** un dépôt d'inscription, **When** la personne confirme son
   adresse alors que l'événement accepte encore des inscriptions, **Then**
   une fiche « Sans compte » d'origine « Publique » est créée, ainsi qu'une
   inscription confirmée, avec le prix figé, une référence et l'état « À
   payer » si l'événement est payant. La place est réservée, et l'e-mail de
   confirmation d'inscription (012 FR-030) contient le lien personnel.
7. **Given** un dépôt d'inscription, **When** la personne confirme son
   adresse alors que l'événement est devenu complet, annulé ou hors période,
   **Then** aucune inscription n'est créée, et la page explique pourquoi.
8. **Given** une adresse confirmée dont une fiche d'origine publique de même
   identité (prénom, nom, date de naissance) est déjà inscrite à l'événement,
   **When** la personne confirme un nouveau dépôt pour cette identité,
   **Then** aucune seconde inscription n'est créée et la page renvoie vers
   l'espace de suivi.
9. **Given** une inscription publique, **When** la personne l'annule depuis
   son espace de suivi, **Then** les règles de 012 FR-013 s'appliquent :
   pendant la période d'inscription, si l'inscription est gratuite, ou
   payante sans être payée ni en cours de paiement par carte.
10. **Given** une inscription publique, **When** un administrateur la retire
    (012 FR-016), **Then** l'e-mail de retrait part à l'adresse confirmée et
    la place est libérée.
11. **Given** un compte connecté, **When** il consulte un événement « Ouvert
    au public » dans son espace membre, **Then** il peut y inscrire toute
    fiche qu'il gère, comme pour « Toutes les fiches ».

---

### User Story 4 - Créer un compte depuis un parcours public (Priority: P3)

Au moment de son dépôt, ou plus tard depuis son espace de suivi, la personne
peut choisir de créer un compte. Après confirmation de son adresse, elle
choisit son mot de passe. Son compte est créé avec le rôle membre, et les
fiches d'origine publique de son adresse lui sont rattachées. Elle retrouve
ses demandes et inscriptions dans son espace membre et n'utilise plus de lien
personnel.

**Why this priority**: c'est un confort pour les personnes qui reviendront :
renouvellement, autres événements, plusieurs membres de la famille. Les
stories 1 à 3 fonctionnent sans compte.

**Independent Test**: déposer une demande en cochant « Créer un compte »,
confirmer l'adresse, choisir un mot de passe, puis constater que la personne
est connectée, que sa fiche porte « Vous » et que sa demande est listée dans
son espace membre.

**Acceptance Scenarios**:

1. **Given** un formulaire public, **When** la personne coche « Créer un
   compte » puis confirme son adresse, **Then** après la création de la
   demande ou de l'inscription, elle est invitée à choisir un mot de passe
   selon la règle de 001 FR-008.
2. **Given** le mot de passe choisi, **When** la personne le valide, **Then**
   un compte membre à adresse confirmée est créé et elle est connectée.
   Toutes les fiches suivies par lien de cette adresse, dans la limite de 20
   (005 FR-004), sont rattachées à ce compte. Si une seule fiche est rattachée, elle est
   marquée « Vous ». S'il y en a plusieurs, la personne indique laquelle est
   la sienne, ou aucune.
3. **Given** un compte créé, **When** une personne suit un lien personnel
   émis avant la création, **Then** le lien est refusé et l'invite à se
   connecter.
4. **Given** une personne qui n'a pas coché « Créer un compte », **When**
   elle ouvre son espace de suivi, **Then** elle peut y lancer la création
   d'un compte, avec le même parcours.
5. **Given** une adresse déjà associée à un compte, **When** une personne
   l'utilise dans un parcours public, **Then** les réponses à l'écran sont
   identiques à celles d'une adresse sans compte. À la confirmation, la fiche
   est rattachée à ce compte (FR-033), et la page comme l'e-mail invitent à
   se connecter pour retrouver la demande dans l'espace membre. La case
   « Créer un compte » est sans effet.
6. **Given** une personne qui abandonne le choix du mot de passe, **When**
   elle revient plus tard, **Then** sa demande ou son inscription reste
   valide, accessible par lien personnel, et elle peut relancer la création
   de compte depuis son espace de suivi.

---

### User Story 5 - Ouvrir et superviser les parcours publics (administration) (Priority: P3)

Un administrateur active l'accueil des demandes d'adhésion publiques dans les
paramètres, copie l'adresse de la page d'adhésion publique pour la diffuser,
et repère dans les listes les demandes, inscriptions et fiches d'origine
publique.

**Why this priority**: c'est la maîtrise de l'ouverture par l'association.
Le paramètre est désactivé par défaut pour qu'aucune instance existante ne
s'expose sans décision explicite.

**Independent Test**: activer le paramètre, constater que la page publique
accepte les dépôts, le désactiver, puis constater le refus. Filtrer la liste
des adhésions sur l'origine « Publique ».

**Acceptance Scenarios**:

1. **Given** une instance qui vient d'être mise à jour, **When** un
   administrateur consulte les paramètres, **Then** l'accueil des demandes
   d'adhésion publiques est désactivé, et la page publique indique que les
   demandes ne sont pas ouvertes.
2. **Given** les paramètres généraux, **When** l'administrateur active
   l'accueil des demandes publiques, **Then** l'adresse de la page d'adhésion
   publique est affichée, copiable, avec un rappel que les tarifs de la
   saison ouverte deviennent visibles de tous.
3. **Given** des demandes publiques et des demandes faites depuis l'espace
   membre, **When** l'administrateur filtre la liste des adhésions, des
   inscriptions ou des membres sur l'origine « Publique », **Then** seules
   les entrées d'origine publique sont listées. Le filtre est reflété dans
   l'adresse de la page.
4. **Given** une fiche d'origine publique sans compte, **When**
   l'administrateur l'ouvre, **Then** il voit son origine, l'adresse
   confirmée et la date de confirmation. Il peut la modifier, la rattacher à
   un compte ou la supprimer avec les règles de 005 et 006.
5. **Given** l'accueil désactivé alors que des demandes publiques existent,
   **When** les personnes suivent leur lien personnel, **Then** le suivi,
   l'annulation et le paiement restent possibles. Seuls les nouveaux dépôts
   d'adhésion sont refusés.

---

### Edge Cases

- **Adresse saisie avec une faute** : le dépôt n'est jamais confirmé et
  disparaît après expiration. Rien n'est visible des administrateurs.
- **Plusieurs dépôts identiques** avant confirmation (double clic, renvoi du
  formulaire) : chaque dépôt a son propre lien. La confirmation du premier
  crée la demande, et la confirmation des suivants pour la même identité, la
  même saison ou le même événement ne crée rien de plus (FR-011).
- **Demande publique pour une identité qui a déjà une adhésion en attente ou
  validée sur la saison**, sous une fiche existante : le dépôt crée une fiche
  d'origine publique, et l'administrateur voit le doublon probable. Le
  rattachement à la fiche existante est refusé par la règle d'unicité de 006
  FR-003 ; l'administrateur refuse alors la demande.
- **Tarif supprimé, saison close ou accueil désactivé entre le dépôt et la
  confirmation** : la confirmation est refusée avec un message explicite, et
  aucune demande n'est créée.
- **Tarif modifié entre le dépôt et la confirmation** : la demande porte le
  tarif tel qu'il est à la confirmation. Le montant est figé à la validation,
  comme en 006.
- **Champ personnalisé archivé ou ajouté entre le dépôt et la confirmation** :
  les valeurs des champs archivés sont ignorées. Un champ obligatoire ajouté
  entre-temps n'est pas exigé de la personne ; il le sera de
  l'administrateur à la validation si la règle de 011 FR-015 l'impose.
- **Dernière place disputée** : deux personnes confirment leur adresse au
  même moment pour la dernière place. Une seule inscription est créée, et
  l'autre personne voit que l'événement est complet (012 FR-015).
- **Ouverture changée après des inscriptions publiques** : passer un
  événement de « Ouvert au public » à une autre ouverture désactive son lien
  public. Les inscriptions existantes restent valides, comme le prévoit 012
  FR-008.
- **Lien public régénéré** : l'administrateur peut régénérer le lien public
  d'un événement, par exemple s'il a circulé au-delà du public voulu. L'ancien
  lien mène alors à « Événement introuvable ».
- **Lien personnel transféré à un tiers** : le tiers voit l'espace de suivi
  jusqu'à l'expiration du lien. Il peut annuler une demande en attente ou
  annoncer un moyen de paiement, mais ne peut ni modifier l'identité, ni
  constater un paiement, ni changer d'adresse. Le risque est accepté et
  borné par la durée de validité.
- **Fiche suivie par lien supprimée par un administrateur** : les liens
  personnels de son adresse ne montrent plus cette fiche. Si l'adresse n'a
  plus aucune fiche suivie par lien, le lien affiche un espace vide.
- **Fiche suivie par lien rattachée à un compte par un administrateur**
  (005 FR-012) : elle sort de l'espace de suivi, et ses notifications partent
  désormais au compte.
- **Adresse d'une fiche sans compte modifiée par un administrateur** : les
  liens personnels déjà émis pour l'ancienne adresse ne montrent plus la
  fiche. La nouvelle adresse peut demander un lien et reçoit les
  notifications suivantes.
- **Adresse mal saisie par un administrateur** sur une fiche sans compte : le
  titulaire de l'adresse erronée peut recevoir les notifications et un lien
  personnel donnant accès à cette fiche. Ce risque est accepté. Il est réduit
  par l'avertissement de FR-040 à la saisie et par le contenu limité de
  l'espace de suivi (FR-016).
- **Dépôt avec l'adresse d'un compte qui gère déjà 20 fiches** : la fiche ne
  peut pas être rattachée. Elle reste « Sans compte », suivie par lien, et
  l'e-mail l'explique.
- **Dépôt avec l'adresse d'un compte désactivé** (002) : la fiche n'est pas
  rattachée. Elle reste « Sans compte », suivie par lien, et la réponse ne
  révèle pas l'état du compte.
- **Fiche publique inactive qui redevient active** (nouvelle demande, nouvelle
  inscription) avant sa suppression : son signalement disparaît.
- **Paiement par carte lancé depuis le lien personnel puis lien expiré** : la
  constatation ne dépend pas du lien (008 FR-008). Au retour, la personne voit
  un message neutre lui indiquant que son paiement sera constaté et qu'elle
  peut demander un nouveau lien pour le vérifier.
- **Robot remplissant le formulaire** : un dépôt qui renseigne le champ piège
  reçoit la même réponse qu'un dépôt valide, mais aucun e-mail n'est envoyé et
  rien n'est conservé.
- **Saturation de la messagerie d'un tiers** : une personne malveillante
  saisit l'adresse d'un tiers dans de nombreux dépôts. La limitation par
  adresse (FR-028) borne le nombre d'e-mails envoyés à cette adresse.

## Requirements *(mandatory)*

### Functional Requirements

**Parcours publics et constitution**

- **FR-001**: Les parcours suivants, et eux seuls, MUST être accessibles sans
  session. Leur ajout à la liste des exceptions du principe II de la
  constitution est un préalable à la planification :
  1. consultation de la page d'adhésion publique : identité visuelle, saison
     ouverte et ses tarifs, champs personnalisés « Rempli par le membre »
     actifs, ou message de fermeture ;
  2. dépôt d'une demande d'adhésion ;
  3. consultation de la page publique d'un événement par son lien public ;
  4. dépôt d'une inscription à un événement ;
  5. confirmation d'une adresse par le lien reçu après un dépôt ;
  6. demande d'un lien personnel (« Recevoir mon lien de suivi ») ;
  7. accès à l'espace de suivi par lien personnel, et les actions qu'il
     permet : consultation, annulation, page de paiement, choix d'un moyen,
     lancement d'un paiement par carte et retour de HelloAsso ;
  8. création d'un compte depuis un parcours public : choix du mot de passe
     et, le cas échéant, choix de la fiche « Vous ».
- **FR-002**: Aucun parcours public MUST NOT transmettre de donnée d'une autre
  personne, d'un autre événement ou de l'association au-delà de ce que FR-001
  énumère. En particulier, aucun inscrit, aucun nombre d'adhérents, aucune
  coordonnée de l'association, aucun brouillon et aucun champ « Réservé à
  l'administration » ou « Visible par le membre » n'est transmis.
- **FR-003**: Toute action modifiant l'état depuis un parcours public MUST
  bénéficier des mêmes protections contre les requêtes d'origine tierce que
  le reste de l'application.

**Accueil des demandes d'adhésion publiques**

- **FR-004**: Les paramètres généraux (010) MUST comporter l'interrupteur
  « Accepter les demandes d'adhésion publiques », désactivé par défaut,
  réservé aux administrateurs. Lorsqu'il est activé, l'adresse de la page
  d'adhésion publique est affichée et copiable.
- **FR-005**: La page d'adhésion publique MUST proposer le dépôt seulement si
  l'interrupteur est activé et qu'au moins une saison ouverte, non close,
  porte au moins un tarif. Sinon, elle affiche un message de fermeture, et le
  serveur refuse tout dépôt.
- **FR-006**: Le formulaire de demande publique MUST faire saisir le prénom,
  le nom et la date de naissance (règles de 005 FR-001), l'adresse
  électronique (obligatoire), le téléphone et l'adresse postale
  (facultatifs, règles de 005 FR-002), la saison si plusieurs sont ouvertes,
  le tarif et le type déclaré, et afficher le montant correspondant avant
  l'envoi (006 FR-007). Il présente aussi les champs personnalisés de membre
  et d'adhésion « Rempli par le membre » actifs, dont les obligatoires sont
  exigés (011 FR-015), et une case de consentement obligatoire.
- **FR-007**: Le texte de consentement MUST indiquer le nom de l'association
  comme destinataire des données et la finalité du traitement (gestion de
  l'adhésion ou de l'inscription). Il précise qu'une demande non confirmée
  est supprimée sous 7 jours, et qu'une fiche sans activité depuis 12 mois
  est proposée à la suppression. La date du consentement est conservée avec
  la fiche.

**Dépôt et confirmation de l'adresse**

- **FR-008**: Un dépôt MUST être conservé seulement le temps de sa
  confirmation. Il n'est visible d'aucun écran, ne crée ni fiche, ni
  adhésion, ni inscription, ne réserve aucune place, et il est supprimé
  définitivement au plus tard 7 jours après son envoi, confirmé ou non.
- **FR-009**: Tout dépôt accepté MUST provoquer l'envoi à l'adresse saisie
  d'un e-mail contenant un lien de confirmation à usage unique, valable
  48 heures. La réponse affichée MUST être identique quelle que soit
  l'adresse, qu'elle ait ou non un compte, une fiche ou une demande.
- **FR-010**: À la confirmation, le système MUST revérifier toutes les
  conditions du dépôt : accueil activé, saison ouverte et non close, tarif
  existant, événement publié « Ouvert au public » dans sa période
  d'inscription et non complet. Si l'une manque, il ne crée rien et affiche
  la raison.
- **FR-011**: À la confirmation d'un dépôt valide, le système MUST, dans une
  même opération :
  - créer une fiche d'origine « Publique », dont l'adresse électronique est
    l'adresse confirmée, « Sans compte » ou rattachée selon FR-033 ;
  - créer la demande d'adhésion en attente (006) ou l'inscription confirmée
    (012) ;
  - enregistrer les valeurs des champs personnalisés et la date de
    confirmation ;
  - mettre en file l'e-mail portant le lien personnel.
  Le système MUST toutefois ne pas créer de nouvelle fiche lorsqu'une fiche
  de même identité (005 FR-005) existe déjà parmi les fiches suivies par lien
  de l'adresse confirmée ou, dans le cas de FR-033, parmi les fiches du
  compte. Il réutilise alors cette fiche. Si elle porte déjà une demande en attente ou une
  adhésion validée sur la saison, ou une inscription confirmée à
  l'événement, la confirmation ne crée rien et renvoie vers l'espace de
  suivi.
- **FR-012**: Une adhésion ou une inscription créée par un parcours public
  MUST enregistrer comme auteur « Demande publique », avec l'adresse
  confirmée, à la place du compte à l'origine (006 FR-002, 012 FR-017).

**Lien personnel et espace de suivi**

- **FR-013**: Un lien personnel MUST être aléatoire, non devinable, valable
  30 jours, et stocké sous une forme qui ne permet pas de le reconstituer.
  Plusieurs liens valides peuvent coexister pour une même adresse. Tous sont
  révoqués quand un compte est créé pour cette adresse (FR-031).
- **FR-014**: Un lien personnel MUST donner accès uniquement aux fiches
  suivies par lien de l'adresse pour laquelle il a été émis, et à leurs
  adhésions et inscriptions. Ce sont les fiches « Sans compte » dont
  l'adresse électronique est cette adresse, comparée sans tenir compte de la
  casse, quelle que soit leur origine. Le périmètre est évalué à chaque
  accès : une fiche rattachée à un compte, supprimée ou dont l'adresse a
  changé en sort aussitôt. Toute tentative d'accès à une autre donnée MUST
  recevoir la même réponse qu'une donnée inexistante.
- **FR-015**: L'espace de suivi MUST lister, par fiche, les adhésions et les
  inscriptions accessibles avec leur état, leur tarif ou leur événement, leur
  montant à payer, l'état de leur paiement, le motif d'un refus ou d'une
  annulation, et un accès à leur page de paiement lorsqu'elles sont à
  régler.
- **FR-016**: Depuis l'espace de suivi, la personne MUST pouvoir, après
  confirmation, annuler une demande d'adhésion en attente (006 FR-009) et
  annuler une inscription selon 012 FR-013. Elle MUST NOT pouvoir modifier
  l'identité ni les coordonnées d'une fiche, ni une valeur de champ
  personnalisé, ni marquer un paiement comme reçu.
- **FR-017**: La page « Recevoir mon lien de suivi » MUST accepter une
  adresse et afficher toujours le même message neutre. Un lien personnel est
  envoyé seulement si l'adresse est celle d'au moins une fiche suivie par
  lien.
- **FR-018**: Un lien personnel expiré, révoqué ou inconnu MUST mener à la
  même page, qui invite à demander un nouveau lien, sans aucune donnée.

**Paiement par lien personnel**

- **FR-019**: La page de paiement d'une adhésion ou d'une inscription
  accessible par lien personnel MUST offrir les mêmes informations, moyens,
  états et règles que la page de paiement de l'espace membre (006 FR-022b,
  008, 012 FR-022, 014 FR-007), sans exiger de session. Cette exception est
  propre aux fiches suivies par lien (FR-014) ; elle lève 006 FR-022c et
  l'hypothèse de 008 « pas de paiement pour les fiches sans compte » pour
  celles-ci uniquement.
- **FR-020**: Un paiement par carte lancé par lien personnel MUST préremplir
  chez HelloAsso l'adresse du lien, ainsi que le prénom et le nom de la fiche
  concernée, sans autre donnée personnelle. L'intitulé reste celui de
  008 FR-004 ou 012 FR-024. Le retour de HelloAsso ramène à la page de
  paiement par lien personnel.
- **FR-021**: La constatation d'un paiement par carte lancé par lien
  personnel MUST offrir toutes les garanties de 008 (FR-006 à FR-011a) et
  utiliser l'adresse de notification existante, sans nouvelle adresse
  accessible sans session. La tentative de paiement enregistre « Demande
  publique » comme origine, à la place du compte.
- **FR-022**: La personne MUST NOT pouvoir ajuster le montant à payer. Seuls
  les administrateurs le peuvent (014 FR-008).

**Événements ouverts au public**

- **FR-023**: L'ouverture d'un événement (012 FR-002) MUST accepter une
  troisième valeur, « Ouvert au public ». Pour un compte connecté, elle se
  comporte comme « Toutes les fiches » (012 FR-011).
- **FR-024**: Un événement « Ouvert au public » MUST posséder un lien public
  non devinable, affiché et copiable sur sa page d'administration. Les
  administrateurs MUST pouvoir le régénérer après confirmation, ce qui
  invalide l'ancien lien.
- **FR-025**: La page publique d'un événement MUST être servie seulement si
  l'événement est publié, ou annulé ou terminé avec la mention
  correspondante, et que son ouverture est « Ouvert au public ». Elle ne
  présente que les informations de l'acceptance scenario 2 de la User Story
  3. Dans tous les autres cas, la réponse MUST être identique à celle d'un
  lien inconnu.
- **FR-026**: Le formulaire d'inscription publique MUST faire saisir le
  prénom, le nom, la date de naissance, l'adresse électronique et le
  consentement. Il propose les champs personnalisés de membre « Rempli par le
  membre » actifs, sans les exiger, et la case « Créer un compte ».
- **FR-027**: Une inscription publique MUST suivre toutes les règles de 012
  et de 014 : confirmation immédiate (à la confirmation de l'adresse),
  capacité garantie y compris en cas de confirmations simultanées, prix figé,
  référence de paiement, retrait par un administrateur à tout moment avant la
  fin de l'événement, notifications de 012 FR-030 et FR-031 envoyées à
  l'adresse confirmée avec un lien personnel.

**Protection contre les abus**

- **FR-028**: Les dépôts, les confirmations d'adresse, les demandes de lien
  personnel et les accès par lien personnel MUST être limités par origine
  réseau et par adresse électronique visée. Un dépassement reçoit un message
  neutre, qui ne révèle ni l'existence d'une adresse ni celle d'une demande.
  Une même adresse MUST NOT recevoir plus de 5 e-mails de confirmation ou de
  lien personnel par heure.
- **FR-029**: Les formulaires de dépôt MUST comporter un champ piège invisible
  pour les personnes, y compris avec un lecteur d'écran. Un dépôt qui le
  renseigne reçoit la réponse d'un dépôt accepté, sans envoi d'e-mail ni
  conservation.
- **FR-030**: Aucun parcours public MUST NOT dépendre d'un service tiers de
  vérification (captcha).

**Création de compte**

- **FR-031**: Une personne MUST pouvoir créer un compte depuis un parcours
  public, soit en cochant « Créer un compte » au dépôt, soit depuis son
  espace de suivi, à condition que son adresse confirmée ne soit associée à
  aucun compte. Le compte est créé avec le rôle membre, une adresse confirmée
  et le mot de passe choisi (001 FR-008), puis la personne est connectée. Les
  fiches suivies par lien de l'adresse, dans la limite de 20, lui sont
  rattachées, et tous les liens personnels de l'adresse sont révoqués.
  Au-delà de 20 fiches, la création est refusée avec un message invitant à
  contacter l'association.
- **FR-032**: Lorsque plusieurs fiches sont rattachées à la création du
  compte, la personne MUST pouvoir désigner la fiche « Vous » ou n'en
  désigner aucune. Avec une seule fiche, elle est marquée « Vous » d'office.
- **FR-033**: Lorsque l'adresse d'un dépôt est déjà associée à un compte, le
  système MUST répondre à l'écran au dépôt comme pour toute autre adresse.
  À la confirmation, la fiche MUST être rattachée à ce compte, ou une fiche
  de même identité qu'il gère déjà est réutilisée (FR-011). La demande ou
  l'inscription apparaît alors dans son espace membre, la page de
  confirmation et l'e-mail invitent à se connecter, aucun lien personnel
  n'est émis et la case « Créer un compte » est ignorée. Les notifications
  suivantes partent au compte (006 FR-023). Le rattachement n'a pas lieu, et
  la fiche reste « Sans compte » suivie par lien, lorsque le compte est
  désactivé ou gère déjà 20 fiches. Aucune réponse ne révèle laquelle de ces
  situations s'applique.
- **FR-034**: La création d'un compte MUST NOT être possible depuis un parcours
  public sans dépôt confirmé. Aucune inscription libre n'est offerte.

**Notifications**

- **FR-035**: Pour toute fiche suivie par lien, quelle que soit son origine,
  les e-mails de 006 FR-023 (validation, refus, annulation, constatation de
  paiement) et de 012 FR-030 MUST être envoyés à l'adresse de la fiche.
  Chacun contient un lien personnel neuf, qui mène à la page de paiement
  lorsque l'adhésion ou l'inscription est à régler. Cette règle remplace
  l'exclusion « Aucun e-mail n'est envoyé pour une fiche sans compte » de
  006 FR-023. Une fiche sans compte et sans adresse ne reçoit toujours rien.
- **FR-036**: Les e-mails des parcours publics MUST passer par la file
  d'attente existante, insérés dans la même opération que l'action qui les
  déclenche. Ils ne contiennent aucune donnée d'une autre personne.

**Administration**

- **FR-037**: Les listes et les fiches de l'administration (membres,
  adhésions, inscriptions) MUST afficher l'origine « Publique » et permettre
  de filtrer sur cette origine. Le filtre se combine avec les filtres
  existants et est reflété dans l'adresse de la page.
- **FR-038**: À la consultation d'une demande publique en attente,
  l'administrateur MUST voir le signalement d'un doublon probable (005
  FR-005). Avant de valider, il MUST pouvoir rattacher la demande à la fiche
  existante signalée : l'adhésion est alors transférée sur cette fiche, sous
  réserve de 006 FR-003, et la fiche d'origine publique, si elle ne porte
  plus rien, est supprimée dans la même opération. L'adresse confirmée est
  reportée sur la fiche existante si celle-ci n'a pas d'adresse.
- **FR-039**: Le rattachement d'une demande à une fiche existante MUST passer
  par un dialogue de confirmation qui nomme les deux fiches.
- **FR-040**: Les fiches « Sans compte » qui ne viennent pas d'un parcours
  public (import 013, saisie par un administrateur) et qui portent une
  adresse MUST être suivies par lien comme les autres (FR-014, FR-035). Dans
  l'administration, le champ d'adresse d'une fiche sans compte MUST
  signaler que cette adresse recevra les notifications et un accès au suivi
  et au paiement. La fiche d'un membre sans compte MUST permettre à un
  administrateur d'envoyer un lien personnel à la demande.
- **FR-041**: Les fiches publiques inactives (voir Vocabulaire) MUST être
  signalées aux administrateurs : par un filtre « Inactives à supprimer »
  dans la liste des membres, reflété dans l'adresse de la page, et par un
  encart du tableau de bord qui donne leur nombre et mène à ce filtre.
  Depuis ce filtre, les administrateurs MUST pouvoir supprimer une fiche ou
  toutes les fiches listées. La suppression se fait après un dialogue de
  confirmation qui en donne le nombre, et selon les règles de suppression
  existantes (005, 006 FR-021, 012). Aucune fiche MUST NOT être supprimée
  sans action d'un administrateur.

**Commun**

- **FR-042**: Le système MUST enregistrer dans les journaux de l'application
  chaque dépôt accepté ou rejeté (avec le motif : limite, piège, condition),
  chaque confirmation, chaque émission et chaque usage de lien personnel,
  chaque création de compte depuis un parcours public, chaque rattachement à
  une fiche existante et chaque régénération de lien public. Les journaux ne
  contiennent ni les liens, ni les adresses, ni aucune donnée personnelle.
- **FR-043**: Les règles d'autorisation de chaque parcours public MUST être
  couvertes par des tests automatisés : un lien personnel ne donne accès qu'à
  son périmètre (FR-014), et un lien expiré, révoqué ou inconnu est refusé.
  Pour une opération publique, une session de membre n'élargit pas l'accès.
  Les opérations non listées en FR-001 restent refusées à toute personne non
  connectée.
- **FR-044**: Les écrans publics MUST reprendre l'identité visuelle de
  l'association et le shell des écrans hors session (001). Ils sont en
  français, utilisables au clavier seul et au lecteur d'écran, et dès
  320 pixels de large. Les erreurs de saisie sont affichées sur le champ
  concerné.
- **FR-045**: Les pages publiques MUST rester exclues de l'indexation par les
  moteurs de recherche.

### Key Entities

- **Dépôt** : demande d'adhésion ou inscription en attente de confirmation
  d'adresse. Il porte la nature (adhésion ou inscription), la cible (saison
  et tarif, ou événement), l'identité et les coordonnées saisies, les valeurs
  des champs personnalisés, le choix « Créer un compte », la date du
  consentement, un lien de confirmation (stocké de façon non réversible) et
  ses dates d'envoi et d'expiration. Il est supprimé à la confirmation ou au
  plus tard 7 jours après l'envoi.
- **Lien personnel** : accès sans compte à l'espace de suivi d'une adresse
  confirmée. Il porte l'adresse, la date d'émission, la date d'expiration et
  la date de révocation éventuelle, et il est stocké de façon non
  réversible.
- **Fiche membre** (005), étendue : origine (« Publique » ou non), adresse
  confirmée et date de confirmation, date du consentement.
- **Adhésion** (006) et **Inscription** (012), étendues : auteur « Demande
  publique » possible à la place d'un compte, avec l'adresse confirmée.
- **Événement** (012), étendu : ouverture « Ouvert au public » et lien public,
  régénérable.
- **Paiement en ligne** (008), étendu : origine « Demande publique » possible
  à la place du compte à l'origine.
- **Paramètres généraux** (010), étendus : interrupteur « Accepter les
  demandes d'adhésion publiques ».

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Une personne sans compte dépose une demande d'adhésion et
  confirme son adresse en moins de 5 minutes, depuis l'ouverture de la page
  d'adhésion publique.
- **SC-002**: Une personne sans compte paie par carte une adhésion validée en
  moins de 3 minutes à partir de l'e-mail de validation, temps passé chez
  HelloAsso compris, sans se connecter.
- **SC-003**: Une personne sans compte s'inscrit à un événement gratuit
  depuis son lien public, confirmation d'adresse comprise, en moins de
  3 minutes.
- **SC-004**: Aucun dépôt non confirmé n'est visible d'un administrateur ni ne
  consomme de place, et 100 % des dépôts sont supprimés au plus tard 7 jours
  après leur envoi.
- **SC-005**: Aucune réponse d'un parcours public ne permet de déterminer si
  une adresse possède un compte, une fiche ou une demande : les réponses à
  l'écran sont identiques pour une adresse connue et une adresse inconnue
  (vérifié par les tests).
- **SC-006**: Aucun événement ne compte plus d'inscriptions confirmées que sa
  capacité, même avec 50 confirmations publiques simultanées sur les
  dernières places.
- **SC-007**: Un lien personnel ne donne accès à aucune donnée hors de son
  périmètre, et un lien expiré ou révoqué n'en donne aucune (vérifié par les
  tests d'autorisation, sur 100 % des opérations publiques).
- **SC-008**: Une même adresse ne reçoit jamais plus de 5 e-mails de
  confirmation ou de lien personnel par heure, quel que soit le nombre de
  dépôts envoyés.
- **SC-009**: 100 % des paiements par carte réussis lancés par lien personnel
  sont constatés une seule fois, sans intervention d'un administrateur.
- **SC-010**: Sur une instance mise à jour, aucune page d'adhésion publique
  n'accepte de dépôt tant qu'un administrateur n'a pas activé l'accueil.
- **SC-011**: Les parcours publics ne présentent aucune violation
  d'accessibilité de niveau grave ou critique.
- **SC-012**: Un membre importé sans compte, avec une adresse, paie en ligne
  une adhésion validée à partir de l'e-mail reçu, sans intervention d'un
  administrateur ni création de compte.
- **SC-013**: 100 % des fiches publiques inactives sont signalées aux
  administrateurs, et aucune fiche n'est supprimée sans leur action.

## Assumptions

- **Préalable constitutionnel** : la planification de cette fonctionnalité
  suppose un amendement de la constitution, probablement en version MINOR.
  Le principe II doit ajouter les parcours de FR-001 à ses exceptions et
  admettre un accès par lien personnel, borné à un périmètre, qui n'est pas
  un rôle. Le principe VII doit étendre la limitation de débit et la règle de
  non-révélation des adresses aux parcours publics. Le principe III est
  inchangé : il n'y a ni nouveau rôle ni compte sans rôle.
- La page d'adhésion publique a une adresse stable et unique par instance.
  Elle ne nécessite aucun secret, puisqu'elle est destinée à être diffusée.
  Le lien public d'un événement est au contraire non devinable, faute de
  listing public.
- Une demande publique porte sur une seule personne. Pour plusieurs
  personnes, par exemple une famille, on fait plusieurs dépôts avec la même
  adresse, qui sont ensuite rassemblés dans le même espace de suivi.
- La date de naissance reste obligatoire pour toute fiche (005 FR-001), y
  compris pour une inscription publique à un événement : elle sert à la
  détection des doublons.
- Les inscriptions publiques ne passent pas par une validation
  administrative : un doublon de fiche créé par une inscription publique est
  signalé (doublon probable) mais pas fusionné automatiquement. La fusion de
  deux fiches reste hors périmètre.
- Les champs personnalisés d'adhésion ne s'appliquent qu'aux demandes
  d'adhésion. Les inscriptions publiques ne proposent que les champs de
  membre, facultatifs, faute de champ personnalisé d'événement (012).
- Les durées retenues (48 heures pour la confirmation, 7 jours avant
  suppression d'un dépôt, 30 jours de validité d'un lien personnel, 5 e-mails
  par heure et par adresse) sont des valeurs par défaut fixes, non
  paramétrables dans cette version.
- Un lien personnel n'est pas à usage unique : il reste valable jusqu'à son
  expiration ou à sa révocation, afin que les liens des e-mails précédents
  continuent de fonctionner. Il ne donne aucun droit sur les données
  d'identité. Le risque d'un lien transféré est accepté et borné par sa
  durée.
- Une personne connectée qui ouvre un parcours public n'utilise pas sa
  session pour y agir : elle est invitée à passer par son espace membre.
- Les fiches importées (013) ou créées par un administrateur ne sont pas
  d'origine « Publique ». Elles sont toutefois suivies par lien dès qu'elles
  portent une adresse et n'ont pas de compte (FR-040). Elles ne sont jamais
  signalées comme inactives (FR-041).
- **Changement de comportement à la mise à jour** : les fiches sans compte
  avec adresse déjà présentes sur une instance commencent à recevoir les
  notifications d'adhésion (FR-035) pour les actions faites après la mise à
  jour. Aucun e-mail rétroactif n'est envoyé.
- Le rattachement automatique à un compte existant (FR-033) repose sur la
  possession de l'adresse, comme la réinitialisation du mot de passe (001).
  Il ne donne au compte aucun droit que l'association ne lui aurait pas
  donné en rattachant elle-même la fiche.
- Le tableau de bord de l'administration n'a pas de nouvel encart : les
  demandes publiques confirmées entrent dans l'encart existant des demandes
  en attente (006 FR-019).

## Hors périmètre

- Agenda ou listing public des événements.
- Liste d'attente pour un événement complet.
- Demande groupée pour plusieurs personnes dans un même formulaire.
- Don en ligne, et tout montant libre saisi par la personne.
- Service tiers de vérification (captcha).
- Inscription libre d'un compte sans demande ni inscription associée.
- Fusion de deux fiches membres.
- Modification de son identité ou de ses coordonnées par une personne sans
  compte.
- Page publique présentant l'association au-delà de son nom, de son logo et
  de ses couleurs (objet, coordonnées, réseaux sociaux).
