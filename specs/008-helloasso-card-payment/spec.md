# Feature Specification: Paiement des adhésions par carte bancaire (HelloAsso)

**Feature Branch**: `008-helloasso-card-payment`

**Created**: 2026-11-02

**Status**: Draft

**Input**: User description: "Ajout du paiement des adhésions via carte bancaire (HelloAsso)"

## Vocabulaire

- **HelloAsso** : plateforme de paiement en ligne réservée aux associations
  françaises. L'association y possède un compte ; les sommes payées par carte
  sont versées sur ce compte, puis reversées à l'association par HelloAsso.
- **Carte bancaire** : nouveau moyen de paiement, qui s'ajoute aux espèces, au
  chèque et au virement (006). Contrairement à ceux-ci, il est **en ligne** :
  l'adhérent paie sur la page sécurisée de HelloAsso et Frigg constate le
  paiement sans intervention d'un administrateur.
- **Paiement en ligne** : une tentative de paiement par carte pour une
  adhésion. Elle est « En cours » (l'adhérent a été envoyé chez HelloAsso),
  puis « Réussie », « Échouée », « Abandonnée » ou « Expirée ».
- **Contribution volontaire** : pourboire que HelloAsso propose à l'adhérent
  pendant le paiement pour financer la plateforme ; elle est facultative,
  s'ajoute au montant de l'adhésion et ne revient pas à l'association.
- **Mode test** : utilisation de l'environnement de démonstration de HelloAsso,
  où aucune somme réelle n'est débitée.

- **Notification de HelloAsso** : message que HelloAsso envoie à Frigg dès
  qu'un paiement change d'état, à une adresse de Frigg que l'administrateur
  déclare dans l'espace HelloAsso de l'association.

Les autres termes (adhésion, montant appliqué, état du paiement, référence de
paiement, paramètres) sont ceux de 006.

## Clarifications

### Session 2026-11-02

- Q: Comment Frigg apprend-il un paiement dont l'adhérent n'est pas revenu sur
  Frigg ? → A: Par la notification de HelloAsso, reçue sur une adresse de
  Frigg accessible sans session (amendement du principe II), complétée par une
  vérification auprès de HelloAsso au retour de l'adhérent.
- Q: Où les identifiants d'accès à HelloAsso sont-ils renseignés ? → A: Dans
  la configuration de l'instance (fichier `.env`), par la personne qui
  l'installe. « Paramètres » affiche l'état de la connexion et permet
  d'activer la carte bancaire ; l'activation est bloquée tant que les
  identifiants ne sont pas renseignés dans la configuration.
- Q: Quel intitulé est transmis à HelloAsso ? → A: Saison, tarif et
  référence de paiement, sans nom du membre (souvent un enfant) : aucune
  donnée personnelle du membre ne sort de Frigg, la référence suffit au
  rapprochement.
- Q: Que fait Frigg d'un remboursement fait dans l'espace HelloAsso ? → A:
  À la réception de la notification de HelloAsso, après vérification du
  remboursement auprès de HelloAsso, l'adhésion réglée par ce paiement est
  annulée si elle ne l'est pas déjà.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Payer mon adhésion par carte (Priority: P1)

Claire reçoit l'e-mail « Adhésion de Léo validée » et suit le lien « Payer
l'adhésion ». Sur la page de paiement, à côté des espèces, du chèque et du
virement, elle choisit « Carte bancaire ». La page lui rappelle le montant
(40 €), lui indique qu'elle va être redirigée vers HelloAsso, plateforme de
paiement des associations, et que HelloAsso lui proposera une contribution
volontaire facultative. Elle choisit « Payer 40 € par carte », saisit sa carte
chez HelloAsso, valide, puis revient sur Frigg : la page annonce « Paiement
reçu, merci » et l'adhésion de Léo est « Payée — carte bancaire ».

**Why this priority**: c'est la raison d'être de la fonctionnalité ; payer en
quelques secondes, sans déplacement ni virement à saisir, lève le principal
frein au règlement des adhésions et supprime le rapprochement manuel pour le
trésorier.

**Independent Test**: en mode test, avec la carte bancaire activée, valider
une demande, suivre le lien de l'e-mail, payer avec une carte de test et
vérifier que l'adhésion passe à « Payée » sans action d'un administrateur.

**Acceptance Scenarios**:

1. **Given** la carte bancaire activée et une adhésion validée à payer ou au
   paiement annoncé, **When** le titulaire ouvre la page de paiement, **Then**
   « Carte bancaire » est proposé parmi les moyens activés, en premier.
2. **Given** « Carte bancaire » sélectionné, **When** la page s'affiche,
   **Then** elle indique le montant, la redirection vers HelloAsso, la
   contribution volontaire facultative proposée par HelloAsso, et un bouton
   « Payer [montant] par carte ».
3. **Given** le bouton activé, **When** le titulaire le choisit, **Then** il
   est redirigé vers la page de paiement de HelloAsso pour le montant appliqué
   exact, avec l'intitulé de l'adhésion (saison, tarif et référence de
   paiement, sans nom du membre, par exemple « Adhésion 2026-2027 — Jeune —
   ADH-2627-0042 »), son adresse électronique préremplie, ainsi que son
   prénom et son nom s'il a une fiche « Vous ».
4. **Given** un paiement accepté par HelloAsso, **When** le titulaire revient
   sur Frigg, **Then** la page confirme le paiement, l'adhésion est « Payée »
   avec le moyen « Carte bancaire », la date du paiement et la référence de la
   transaction HelloAsso, et l'e-mail de constatation (006, FR-023) est envoyé.
5. **Given** un paiement refusé par la banque ou une erreur chez HelloAsso,
   **When** le titulaire revient sur Frigg, **Then** la page indique que le
   paiement n'a pas abouti, qu'aucune somme n'a été débitée, et propose de
   réessayer ou de choisir un autre moyen ; l'adhésion reste à payer.
6. **Given** le titulaire qui quitte HelloAsso par « Retour » sans payer,
   **When** il revient sur Frigg, **Then** la page de paiement s'affiche de
   nouveau et l'adhésion reste dans son état précédent.
7. **Given** un paiement accepté, **When** le titulaire ferme son navigateur
   avant d'être revenu sur Frigg, **Then** l'adhésion passe tout de même à
   « Payée » dès réception de la notification de HelloAsso, après
   vérification du paiement auprès de HelloAsso.
8. **Given** un paiement déjà constaté par la notification, **When** le
   titulaire revient ensuite sur Frigg, **Then** la page confirme le paiement
   sans le constater une seconde fois (un seul e-mail de constatation).
9. **Given** une adhésion « Payée », « Gratuite », refusée, annulée, en
   attente, ou une adhésion d'une fiche gérée par un autre compte, **When**
   quelqu'un tente d'initier un paiement par carte, **Then** le serveur le
   refuse (réponse identique à une adhésion inexistante pour un autre compte).
10. **Given** une adhésion au « Paiement annoncé » par chèque, **When** le
    titulaire paie finalement par carte, **Then** le paiement par carte est
    constaté et remplace le moyen annoncé.

---

### User Story 2 - Paramétrer HelloAsso (administration) (Priority: P1)

La personne qui héberge l'instance a renseigné dans la configuration les
identifiants HelloAsso de l'association, en mode test. Le trésorier ouvre
« Paramètres », section « Paiements ». Sous les espèces, le chèque et le
virement, une carte « Carte bancaire (HelloAsso) » indique « Identifiants
renseignés — mode test » et affiche l'adresse de notification à déclarer dans
l'espace HelloAsso. Il la copie dans l'espace HelloAsso, revient et choisit
« Tester la connexion » : Frigg confirme « Connexion à HelloAsso réussie —
Association Frigg ». Il active la carte bancaire et fait un essai de bout en
bout avec une carte de test. L'hébergeur bascule ensuite la configuration en
mode réel.

**Why this priority**: sans paramétrage, aucun paiement par carte n'est
possible ; l'essai en mode test évite qu'une erreur de configuration ne soit
découverte par un adhérent.

**Independent Test**: avec des identifiants de démonstration valides dans la
configuration, tester la connexion, activer le moyen, constater son apparition
sur une page de paiement ; sans identifiants dans la configuration, constater
que l'activation est impossible.

**Acceptance Scenarios**:

1. **Given** un administrateur, **When** il ouvre la section « Paiements »,
   **Then** la carte « Carte bancaire (HelloAsso) » présente un interrupteur
   d'activation, l'état des identifiants (renseignés ou non dans la
   configuration, sans jamais afficher le secret), le mode (test ou réel),
   l'adresse de notification à déclarer dans l'espace HelloAsso, avec un
   bouton pour la copier, et un lien vers la documentation de HelloAsso.
2. **Given** des identifiants renseignés dans la configuration, **When**
   l'administrateur choisit « Tester la connexion », **Then** Frigg indique en
   moins de 10 secondes si la connexion a réussi (avec le nom de
   l'association tel que connu de HelloAsso) ou la cause de l'échec
   (identifiants refusés, association introuvable, droits insuffisants,
   HelloAsso injoignable).
3. **Given** des identifiants absents de la configuration, **When**
   l'administrateur ouvre la section, **Then** l'interrupteur d'activation est
   désactivé, un message explique que les identifiants doivent être
   renseignés dans la configuration de l'instance par la personne qui
   l'héberge, et le serveur refuse toute activation.
4. **Given** des identifiants renseignés mais dont le dernier test a échoué,
   **When** l'administrateur tente d'activer la carte bancaire, **Then**
   l'activation est refusée avec la cause du dernier échec.
5. **Given** la configuration en mode test, **When** un adhérent ou un administrateur
   voit la page de paiement ou une adhésion payée en mode test, **Then** un
   bandeau « Mode test — aucun débit réel » est affiché, et les paiements de
   test sont distingués des paiements réels dans la liste et les chiffres.
6. **Given** la carte bancaire désactivée, **When** un adhérent ouvre une page
   de paiement, **Then** elle n'est plus proposée ; les paiements par carte
   déjà constatés restent inchangés.
7. **Given** un membre connecté, **When** il tente de consulter ou de modifier
   ces paramètres ou de tester la connexion, **Then** l'accès est refusé.
8. **Given** la carte bancaire activée, **When** les identifiants sont
   retirés de la configuration et l'instance redémarrée, **Then** la carte
   bancaire n'est plus proposée aux adhérents, la section l'indique, et
   l'activation est conservée pour le retour des identifiants.

---

### User Story 3 - Suivre les paiements par carte (administration) (Priority: P2)

Dans « Adhésions », le trésorier filtre sur le moyen « Carte bancaire » : il
voit les adhésions payées par carte, la date et la référence HelloAsso de
chacune, et peut ouvrir la transaction correspondante dans l'espace HelloAsso
de l'association. Les chiffres de la saison indiquent le montant encaissé par
carte. Une adhésion annulée après un paiement par carte porte un avertissement
« Paiement par carte reçu — remboursement à effectuer dans HelloAsso ».

**Why this priority**: le paiement se constate seul (US1) ; ce suivi sert au
rapprochement avec les reversements de HelloAsso et au traitement des cas
particuliers, moins fréquents.

**Independent Test**: avec des adhésions payées par carte, par chèque et non
payées, filtrer par moyen, vérifier les chiffres par moyen et la présence de la
référence HelloAsso ; annuler une adhésion payée par carte et constater
l'avertissement.

**Acceptance Scenarios**:

1. **Given** des adhésions payées par carte, **When** l'administrateur ouvre
   l'une d'elles, **Then** il voit le moyen « Carte bancaire », la date et
   l'heure du paiement, le montant reçu, la référence de la transaction
   HelloAsso et un lien vers celle-ci dans l'espace HelloAsso (à défaut, vers
   l'espace HelloAsso de l'association).
2. **Given** la liste des adhésions, **When** l'administrateur filtre sur le
   moyen de paiement effectif ou recherche une référence HelloAsso, **Then**
   seules les adhésions correspondantes s'affichent.
3. **Given** une saison, **When** l'administrateur consulte ses chiffres,
   **Then** le montant encaissé est détaillé par moyen, carte bancaire
   comprise ; la contribution volontaire n'y figure pas.
4. **Given** une adhésion payée par carte, **When** l'administrateur tente
   d'annuler la constatation du paiement, **Then** l'action n'est pas
   proposée, le paiement ayant été constaté par HelloAsso.
5. **Given** une adhésion payée par carte, **When** l'administrateur annule
   l'adhésion, **Then** le dialogue rappelle que le remboursement se fait dans
   l'espace HelloAsso, et l'adhésion annulée garde l'avertissement de
   remboursement jusqu'à ce qu'un administrateur indique « Remboursement
   effectué ».
6. **Given** une adhésion avec un paiement par carte en cours, **When**
   l'administrateur la consulte, **Then** il voit « Paiement par carte en
   cours » avec l'heure de début, et peut choisir « Vérifier auprès de
   HelloAsso » : un paiement réussi est alors constaté, un paiement échoué ou
   expiré clôt la tentative.

---

### Edge Cases

- **Double paiement** (deux onglets, double clic, paiement par carte d'une
  adhésion déjà réglée par chèque entre-temps) : le premier paiement reçu
  constate l'adhésion ; tout paiement supplémentaire est enregistré sur
  l'adhésion et signalé aux administrateurs (« Paiement en double —
  remboursement à effectuer dans HelloAsso ») dans la liste et par un encart
  du tableau de bord.
- **Paiement en cours** : tant qu'une tentative est en cours et non expirée,
  choisir de nouveau « Payer par carte » reprend cette tentative plutôt que
  d'en créer une seconde, si le montant n'a pas changé.
- **Montant reçu différent** du montant appliqué (tarif corrigé par un
  administrateur pendant le paiement) : l'adhésion est marquée « Payée » avec
  le montant effectivement reçu et signalée aux administrateurs (« Montant
  reçu différent du montant attendu »).
- **Adhésion annulée pendant le paiement** : le paiement reçu est enregistré
  sur l'adhésion annulée, qui porte l'avertissement de remboursement.
- **Correction ou annulation pendant le paiement** : l'administrateur est
  averti qu'un paiement par carte est en cours avant de confirmer.
- **Tentative expirée** : une tentative restée sans issue au-delà de la durée
  de validité fixée par HelloAsso est considérée comme expirée ; l'adhésion
  reste à payer.
- **HelloAsso injoignable** au moment de payer : la page l'indique et propose
  les autres moyens activés ; aucune tentative n'est créée.
- **Identifiants révoqués ou expirés** après activation : les nouvelles
  tentatives échouent avec un message invitant à choisir un autre moyen ; les
  administrateurs voient un encart « Connexion à HelloAsso en échec » sur le
  tableau de bord.
- **Retour ou notification falsifiés ou rejoués** : ni l'adresse de retour ni
  le contenu d'une notification ne suffisent à marquer une adhésion payée ;
  Frigg vérifie le paiement auprès de HelloAsso et l'associe à l'adhésion qui
  l'a initié. Une notification sans rapport avec une tentative connue est
  ignorée (et journalisée), et la réponse ne révèle rien de l'association.
- **Notification et retour simultanés** : le paiement n'est constaté qu'une
  fois ; un seul e-mail de constatation est envoyé.
- **Notification perdue** (instance indisponible, adresse mal déclarée dans
  HelloAsso) : le paiement est constaté au retour de l'adhérent ou par
  « Vérifier auprès de HelloAsso » ; une tentative restée en cours au-delà de
  sa durée de validité est vérifiée automatiquement, et signalée aux
  administrateurs seulement si HelloAsso reste injoignable plus de 24 heures.
- **Notifications d'autres ventes** de l'association sur HelloAsso (dons,
  billetterie hors Frigg) : ignorées.
- **Contribution volontaire** : jamais comptée dans le montant reçu de
  l'adhésion ni dans les chiffres de la saison.
- **Remboursement fait dans HelloAsso** : après vérification auprès de
  HelloAsso, le paiement est marqué « Remboursé » ; l'adhésion qu'il a réglée
  est annulée (motif « Paiement remboursé dans HelloAsso. », e-mail
  d'annulation) si elle ne l'est pas déjà, et ses signalements sont clos. Le
  remboursement d'un paiement en double ne touche pas l'adhésion. Une
  notification perdue se rattrape par « Vérifier auprès de HelloAsso ».
- **Bascule du mode test au mode réel** (changement de configuration) : un
  nouveau test de connexion est exigé avant que la carte bancaire ne soit de
  nouveau proposée ; les tentatives de test en cours
  deviennent caduques ; les paiements de test déjà constatés restent marqués
  « test » et ne comptent pas dans le montant encaissé réel.
- **Fiche sans compte** : aucun paiement par carte possible, faute de page de
  paiement (006).
- **Montant appliqué inférieur au minimum accepté par HelloAsso** : la carte
  bancaire n'est pas proposée pour cette adhésion.

## Requirements *(mandatory)*

### Functional Requirements

**Moyen « Carte bancaire »**

- **FR-001**: Le système MUST ajouter « Carte bancaire » aux moyens de
  paiement de 006 ; il MUST être proposé sur la page de paiement uniquement
  s'il est activé et que la connexion à HelloAsso est paramétrée.
- **FR-002**: Le compte qui gère la fiche MUST pouvoir payer par carte une
  adhésion validée non gratuite, à payer ou au paiement annoncé, pour son
  montant appliqué exact, en un seul versement.
- **FR-003**: Avant la redirection, la page MUST informer que le paiement se
  fait chez HelloAsso et que la contribution volontaire proposée par HelloAsso
  est facultative.
- **FR-004**: Le paiement MUST être transmis à HelloAsso avec un intitulé
  composé de la saison, du tarif et de la référence de paiement, sans aucune
  donnée personnelle du membre, et préremplir l'adresse électronique du
  titulaire du compte, ainsi que le prénom et le nom de sa fiche « Vous » s'il
  en a une (un compte ne porte pas de nom) ; aucune autre donnée personnelle
  n'est transmise.
- **FR-005**: Frigg MUST NOT recevoir, afficher ni conserver les numéros de
  carte ou toute autre donnée bancaire du payeur.

**Constatation**

- **FR-006**: Un paiement par carte réussi MUST faire passer l'adhésion à
  « Payée » avec le moyen « Carte bancaire », la date et l'heure du paiement,
  le montant reçu (hors contribution volontaire) et la référence de la
  transaction HelloAsso, sans intervention d'un administrateur.
- **FR-007**: Un paiement MUST être constaté seulement après vérification de
  son état auprès de HelloAsso et de son rattachement à l'adhésion qui l'a
  initié ; aucune information fournie par le navigateur ne suffit.
- **FR-008**: Un paiement réussi MUST être constaté même si l'adhérent ne
  revient pas sur Frigg : Frigg MUST recevoir les notifications de HelloAsso
  sur une adresse accessible sans session, et vérifier aussi l'état du
  paiement au retour de l'adhérent. Cette adresse MUST seulement déclencher
  une vérification auprès de HelloAsso, MUST NOT exposer de donnée de
  l'association et MUST être ajoutée à la liste des exceptions publiques de
  la constitution (principe II).
- **FR-008a**: Un même paiement MUST être constaté une seule fois, quels que
  soient le nombre et l'ordre des notifications et retours reçus.
- **FR-008b**: Les administrateurs MUST pouvoir vérifier auprès de HelloAsso
  l'état d'une tentative en cours ; une tentative restée en cours au-delà de
  sa durée de validité MUST être vérifiée automatiquement auprès de HelloAsso
  (constatée si payée, close sinon) et signalée aux administrateurs seulement
  si cette vérification reste impossible plus de 24 heures.
- **FR-009**: Une tentative échouée, abandonnée ou expirée MUST laisser
  l'adhésion dans son état de paiement précédent et permettre une nouvelle
  tentative ou le choix d'un autre moyen.
- **FR-010**: La constatation d'un paiement par carte MUST déclencher l'e-mail
  de constatation de 006 (FR-023) et MUST NOT pouvoir être annulée par un
  administrateur.
- **FR-011**: Tout paiement reçu en double, d'un montant différent du montant
  attendu ou sur une adhésion annulée MUST être enregistré et signalé aux
  administrateurs dans la liste, sur l'adhésion et par un encart du tableau de
  bord, jusqu'à ce qu'un administrateur indique l'avoir traité.
- **FR-011a**: Un remboursement fait dans l'espace HelloAsso MUST être
  constaté, après vérification auprès de HelloAsso, à la réception de sa
  notification ou par « Vérifier auprès de HelloAsso » : le paiement passe à
  « Remboursé », l'adhésion qu'il a réglée est annulée si elle ne l'est pas
  déjà, et les signalements de ce paiement sont clos.

**Paramètres**

- **FR-012**: L'identifiant de l'association sur HelloAsso, les identifiants
  d'accès et le mode (test ou réel) MUST être fournis par la configuration de
  l'instance ; ils MUST NOT être saisis ni modifiés depuis l'interface. Ces
  variables sont facultatives : leur absence ne bloque pas le démarrage, mais
  rend la carte bancaire indisponible. Elles sont documentées avec les autres
  variables de configuration.
- **FR-012a**: La section « Paiements » des paramètres MUST présenter la carte
  bancaire avec : interrupteur d'activation, état des identifiants (renseignés
  ou non), mode (test ou réel), adresse de notification à déclarer dans
  l'espace HelloAsso (copiable), et un test de connexion indiquant le
  résultat et la cause d'un échec.
- **FR-013**: L'activation de la carte bancaire MUST être refusée par le
  serveur lorsque les identifiants ne sont pas renseignés dans la
  configuration, ou tant que la connexion n'a pas été testée avec succès avec
  les identifiants et le mode en vigueur.
- **FR-014**: Le secret d'accès à HelloAsso MUST NOT être affiché, renvoyé au
  navigateur, stocké en base ni écrit dans les journaux.
- **FR-015**: En mode test, toute page de paiement et toute adhésion payée en
  mode test MUST porter la mention « Mode test » ; les paiements de test MUST
  être exclus du montant encaissé réel.

**Administration**

- **FR-016**: La consultation d'une adhésion payée par carte MUST afficher la
  date et l'heure, le montant reçu, la référence HelloAsso et un lien vers la
  transaction dans l'espace HelloAsso, ou, si HelloAsso ne permet pas d'y
  accéder directement, vers l'espace HelloAsso de l'association ; une
  tentative en cours MUST être visible.
- **FR-017**: La liste des adhésions MUST pouvoir être filtrée par moyen
  effectif, et la recherche MUST accepter une référence HelloAsso ; les
  chiffres de la saison MUST détailler le montant encaissé par moyen.
- **FR-018**: L'annulation ou la correction d'une adhésion avec un paiement
  par carte reçu ou en cours MUST rappeler, dans le dialogue de confirmation,
  que tout remboursement se fait dans l'espace HelloAsso.
- **FR-019**: Le tableau de bord MUST afficher un encart lorsque la connexion à
  HelloAsso échoue alors que la carte bancaire est activée.
- **FR-020**: Les paramètres HelloAsso, le test de connexion et le traitement
  des signalements MUST être refusés aux membres et aux personnes non
  connectées ; l'initiation d'un paiement MUST être refusée pour toute adhésion
  d'une fiche que le compte ne gère pas.

**Commun**

- **FR-021**: Le système MUST enregistrer dans les journaux de l'application
  chaque tentative (création, issue), constatation, signalement, test de
  connexion et modification des paramètres HelloAsso, avec l'adhésion visée,
  le compte à l'origine et la référence HelloAsso, sans données personnelles
  du membre, sans secret ni donnée bancaire.
- **FR-022**: Les écrans MUST être en français, utilisables au clavier seul,
  compatibles avec les lecteurs d'écran et utilisables dès 320 pixels de
  large ; le départ vers HelloAsso et le retour MUST être annoncés clairement.

### Key Entities

- **Paiement en ligne** : tentative de paiement par carte d'une adhésion.
  Attributs : adhésion, compte à l'origine, montant demandé, état (en cours,
  réussie, échouée, abandonnée, expirée), identifiant de la tentative chez
  HelloAsso, référence de la transaction, montant reçu, date de début et
  d'issue, mode test ou réel. Une adhésion peut en porter plusieurs.
- **Signalement de paiement** : anomalie à traiter par un administrateur
  (paiement en double, montant différent, paiement sur adhésion annulée).
  Attributs : adhésion, paiement en ligne concerné, nature, date, traité ou
  non, administrateur et date de traitement.
- **Configuration HelloAsso** : fournie par la configuration de l'instance,
  hors base : identifiant de l'association, identifiants d'accès, mode (test
  ou réel).
- **Paramètres HelloAsso** : données singleton de l'instance, intégrées aux
  paramètres des moyens de paiement (006) : activation, date, mode et
  résultat du dernier test de connexion.
- **Adhésion** (existante, 006) : son paiement peut désormais être constaté
  par HelloAsso ; le moyen effectif peut valoir « Carte bancaire ».

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Depuis le lien de l'e-mail, un titulaire de compte paie son
  adhésion par carte en moins de 2 minutes, hors temps de saisie chez
  HelloAsso.
- **SC-002**: 100 % des paiements réussis chez HelloAsso sont constatés dans
  Frigg, y compris lorsque l'adhérent ne revient pas sur Frigg, sans
  intervention d'un administrateur.
- **SC-003**: Aucune adhésion n'est marquée payée par carte sans paiement
  réussi vérifié auprès de HelloAsso (vérifié par des tests automatisés de
  retours falsifiés et rejoués).
- **SC-004**: 100 % des paiements en double ou d'un montant inattendu sont
  signalés aux administrateurs.
- **SC-005**: Une fois les identifiants renseignés dans la configuration, un
  administrateur déclare l'adresse de notification, teste la connexion et
  active la carte bancaire en moins de 5 minutes.
- **SC-005a**: Un paiement réussi est constaté dans Frigg en moins de
  1 minute après sa validation chez HelloAsso, que l'adhérent revienne ou non
  sur Frigg (hors indisponibilité de l'instance ou de HelloAsso).
- **SC-006**: Le montant encaissé par carte d'une saison correspond à 100 %
  aux paiements réussis hors contribution volontaire, et rapproché du
  relevé HelloAsso sans écart.
- **SC-007**: Aucun écran livré ne présente de violation d'accessibilité grave
  ou critique lors des contrôles automatisés et d'un parcours complet au
  clavier.

## Assumptions

- **Prestataire unique** : HelloAsso est le seul prestataire de paiement en
  ligne ; il est gratuit pour l'association, se finance par la contribution
  volontaire et exige une association française disposant d'un compte
  HelloAsso validé.
- **Un seul versement** : pas de paiement en plusieurs fois, pas de
  prélèvement SEPA, pas de paiement partiel, conformément à 006.
- **Pas de remboursement depuis Frigg** : les remboursements se font dans
  l'espace HelloAsso ; Frigg rappelle ceux qui sont à faire et constate ceux
  qui sont faits (FR-011a). Seul un remboursement total est pris en compte.
- **Pas de paiement pour les fiches sans compte**, ni de lien de paiement
  envoyé à une personne non connectée (006, FR-022c).
- **Pas de reçu fiscal** : le reçu de paiement est envoyé par HelloAsso ;
  Frigg envoie son e-mail de constatation habituel.
- **Paiement des adhésions uniquement** : dons, billetterie et boutique sont
  hors périmètre.
- **Reversements** : le rapprochement des reversements de HelloAsso avec la
  comptabilité relève de la future fonctionnalité « Finances ».
- **Amendement de la constitution** nécessaire avant le plan : principe II
  (nouvelle adresse publique pour les notifications de HelloAsso) et
  principe V (HelloAsso, facultatif, devient une dépendance d'exécution
  externe).
- **Instance joignable depuis Internet** : la réception des notifications
  suppose que l'adresse publique de l'instance soit accessible à HelloAsso ;
  à défaut, les paiements sont constatés au retour de l'adhérent ou par
  vérification manuelle.
- **Configuration par l'hébergeur** : les identifiants HelloAsso et le mode
  sont gérés par la personne qui héberge l'instance ; un changement exige une
  modification de la configuration et un redémarrage.
- **Disponibilité** : une indisponibilité de HelloAsso n'empêche aucune autre
  fonction de Frigg ; les autres moyens restent proposés.
- **Ordre des moyens** : la carte bancaire est présentée en premier lorsqu'elle
  est activée, le paiement en ligne étant le plus simple pour l'adhérent et le
  bureau.
- **Inscriptions aux événements** : le paiement par carte, sa constatation,
  les notifications, les signalements et les remboursements s'appliquent aussi
  aux inscriptions payantes de la spécification 012 « Événements de
  l'association », par la même adresse de notification.
