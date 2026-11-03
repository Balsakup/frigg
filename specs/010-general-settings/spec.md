# Feature Specification: Paramètres généraux de l'association

**Feature Branch**: `010-general-settings`

**Created**: 2026-11-03

**Status**: Draft

**Input**: User description: "Paramètres généraux (Nom de l'asso, tagline, etc.)"

## Vocabulaire

- **Paramètres généraux** : informations décrivant l'association, uniques pour
  l'instance (principe I) : identité, identité légale, coordonnées, réseaux
  sociaux, logo et couleurs.
- **Nom** : nom complet de l'association, affiché dans l'application et en
  expéditeur des e-mails. Il est déjà saisi à la création du premier
  administrateur (`frigg create-admin`).
- **Tagline** : courte phrase d'accroche qui accompagne le nom (par exemple
  « Randonnées et sorties nature depuis 1987 »).
- **Nom court** : sigle ou abréviation (par exemple « SVR »), utilisé lorsque la
  place manque.
- **Objet** : texte libre qui décrit le but de l'association, tel qu'énoncé
  dans ses statuts.
- **Identité légale** : numéro RNA (répertoire national des associations,
  forme `W` suivi de 9 chiffres), numéro SIRET (14 chiffres) et date de
  création de l'association.
- **Coordonnées** : adresse du siège, adresse électronique de contact, numéro
  de téléphone et site web de l'association.
- **Couleur principale** et **couleur d'accent** : les deux couleurs de
  l'association qui remplacent celles de Frigg dans l'interface (boutons, liens,
  éléments actifs, mises en avant). Les couleurs sémantiques (succès,
  avertissement, danger, information) ne changent pas.
- **Section « Général »** et **section « Apparence »** : deux nouvelles sections
  de la rubrique « Paramètres » de l'espace administration, à côté de la
  section « Paiements » existante.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Renseigner le nom et la tagline de l'association (Priority: P1)

Un administrateur ouvre « Paramètres », section « Général ». Il voit le nom
actuel de l'association (celui saisi à l'installation) et une tagline vide. Il
corrige le nom, saisit une tagline, puis enregistre. Le nouveau nom apparaît
immédiatement dans la barre latérale, le titre de l'onglet du navigateur et le
pied de page ; la tagline apparaît sous le nom dans la barre latérale et dans
l'en-tête de l'accueil de l'espace membre. Les e-mails envoyés ensuite portent
le nouveau nom.

**Why this priority**: le nom n'est aujourd'hui modifiable qu'à l'installation ;
une faute de frappe ou un changement de nom ne peut être corrigé sans
intervention technique. C'est le minimum utile de la fonctionnalité.

**Independent Test**: modifier le nom et la tagline, enregistrer, recharger
l'application avec un compte membre et constater l'affichage partout où le nom
et la tagline sont attendus ; déclencher un e-mail et vérifier l'expéditeur.

**Acceptance Scenarios**:

1. **Given** un administrateur sur la section « Général », **When** il modifie
   le nom, saisit une tagline et enregistre, **Then** une notification confirme
   l'enregistrement et la barre latérale affiche le nouveau nom et la tagline
   sans rechargement de la page.
2. **Given** une tagline enregistrée, **When** un membre ouvre l'accueil de son
   espace, **Then** la tagline s'affiche dans l'en-tête de la page.
3. **Given** un administrateur qui vide le champ du nom, **When** il enregistre,
   **Then** l'enregistrement est refusé et un message sur le champ indique que
   le nom est obligatoire.
4. **Given** un administrateur qui vide la tagline, **When** il enregistre,
   **Then** la tagline disparaît de la barre latérale et de l'accueil membre,
   sans espace vide laissé à sa place.
5. **Given** un nom modifié, **When** un e-mail est envoyé ensuite (invitation,
   notification d'adhésion, communication), **Then** il porte le nouveau nom en
   expéditeur et dans son en-tête.
6. **Given** un membre (non administrateur), **When** il tente d'accéder à la
   section « Général » ou d'enregistrer des paramètres, **Then** l'accès est
   refusé.

---

### User Story 2 - Renseigner l'identité légale et les coordonnées (Priority: P2)

Un administrateur complète, dans la même section « Général », le nom court,
l'objet, le numéro RNA, le numéro SIRET, la date de création, l'adresse du
siège, l'adresse électronique de contact, le téléphone et le site web. Chaque
champ est facultatif ; ceux qui sont remplis sont vérifiés avant
l'enregistrement. Ces informations sont conservées pour les fonctionnalités
futures (reçus, attestations, mentions légales).

**Why this priority**: ces données ne changent presque jamais mais
conditionnent des documents officiels à venir ; les saisir une fois, au bon
endroit et dans un format vérifié, évite des erreurs plus tard.

**Independent Test**: remplir tous les champs avec des valeurs valides,
enregistrer, recharger la section et retrouver les valeurs à l'identique ;
saisir une valeur invalide dans chaque champ vérifié et constater le refus.

**Acceptance Scenarios**:

1. **Given** la section « Général », **When** l'administrateur saisit un numéro
   RNA `W751234567`, un SIRET valide et une date de création passée, puis
   enregistre, **Then** les valeurs sont conservées et réaffichées à
   l'identique.
2. **Given** un numéro RNA qui ne respecte pas la forme `W` + 9 chiffres,
   **When** l'administrateur enregistre, **Then** l'enregistrement est refusé
   et le champ RNA affiche un message expliquant le format attendu.
3. **Given** un SIRET de 14 chiffres dont la clé de contrôle est fausse,
   **When** l'administrateur enregistre, **Then** l'enregistrement est refusé
   avec un message sur le champ SIRET.
4. **Given** un SIRET saisi avec des espaces (`123 456 789 00012`), **When**
   l'administrateur enregistre, **Then** les espaces sont ignorés et le numéro
   est conservé sous sa forme compacte.
5. **Given** une date de création dans le futur, **When** l'administrateur
   enregistre, **Then** l'enregistrement est refusé avec un message sur le
   champ.
6. **Given** une adresse électronique de contact mal formée ou un site web qui
   ne commence pas par `http://` ou `https://`, **When** l'administrateur
   enregistre, **Then** l'enregistrement est refusé avec un message sur le
   champ concerné.
7. **Given** plusieurs champs invalides en même temps, **When**
   l'administrateur enregistre, **Then** tous les champs en erreur sont signalés
   en une seule fois et aucune valeur n'est enregistrée.

---

### User Story 3 - Personnaliser le logo et les couleurs (Priority: P2)

Un administrateur ouvre « Paramètres », section « Apparence ». Il dépose le logo
de l'association et choisit une couleur principale et une couleur d'accent. Un
aperçu montre le rendu (barre latérale, bouton principal, lien) avant
l'enregistrement. Une fois enregistrés, le logo remplace les initiales dans la
barre latérale et les couleurs s'appliquent à toute l'interface, y compris aux
écrans de connexion, qui affichent alors le nom et le logo de l'association à la
place de ceux de Frigg, ainsi qu'aux e-mails envoyés par l'application. Il peut à
tout moment retirer le logo ou revenir aux couleurs par défaut de Frigg.

**Why this priority**: l'appropriation visuelle compte pour les bénévoles et
les adhérents, mais l'application reste pleinement utilisable avec l'apparence
par défaut.

**Independent Test**: déposer un logo, choisir deux couleurs, enregistrer, se
déconnecter et constater l'écran de connexion à l'identité de l'association, se
connecter avec un compte membre et constater le logo et les couleurs, déclencher
un e-mail et constater son bandeau, ses boutons et son logo ; puis
retirer le logo et réinitialiser les couleurs et constater le retour à
l'apparence par défaut.

**Acceptance Scenarios**:

1. **Given** la section « Apparence » sans logo, **When** l'administrateur
   dépose une image PNG de 200 Ko, **Then** le logo s'affiche dans l'aperçu puis,
   après enregistrement, à la place des initiales dans la barre latérale.
2. **Given** un fichier qui n'est pas une image acceptée (PDF, SVG, fichier
   renommé en `.png`) ou une image de plus de 1 Mo (1 048 576 octets), **When** l'administrateur le
   dépose, **Then** le dépôt est refusé avec un message indiquant les formats et
   la taille acceptés, et le logo actuel est conservé.
3. **Given** un logo enregistré, **When** l'administrateur choisit « Retirer le
   logo » et confirme dans le dialogue de confirmation, **Then** les initiales
   de l'association réapparaissent dans la barre latérale.
4. **Given** l'administrateur qui choisit une couleur principale, **When** il
   modifie la valeur, **Then** l'aperçu se met à jour immédiatement, sans que le
   reste de l'interface change avant l'enregistrement.
5. **Given** une couleur principale très claire (par exemple jaune pâle),
   **When** elle est enregistrée, **Then** le texte posé sur cette couleur et
   les textes dérivés de cette couleur restent lisibles avec un contraste d'au
   moins 4,5:1, quitte à être ajustés automatiquement, et l'aperçu montre ce
   rendu ajusté.
6. **Given** des couleurs personnalisées, **When** l'administrateur choisit
   « Rétablir les couleurs par défaut » et confirme, **Then** les couleurs de
   Frigg s'appliquent de nouveau.
7. **Given** des couleurs personnalisées enregistrées par un administrateur,
   **When** un membre déjà connecté recharge l'application, **Then** il voit
   les nouvelles couleurs.
8. **Given** un logo et des couleurs enregistrés, **When** une personne non
   connectée affiche l'écran de connexion (ou de réinitialisation, de
   vérification, d'acceptation d'invitation), **Then** elle voit le logo, le nom
   et les couleurs de l'association, et rien d'autre de l'association
   (ni tagline, ni coordonnées, ni aucune autre donnée).
9. **Given** aucun logo ni couleur personnalisés, **When** une personne non
   connectée affiche l'écran de connexion, **Then** elle voit les initiales et
   le nom de l'association sur les couleurs par défaut de Frigg.
10. **Given** un logo et des couleurs enregistrés, **When** un e-mail est envoyé
   (invitation, réinitialisation, notification d'adhésion, communication),
   **Then** son bandeau d'en-tête et ses boutons sont aux couleurs de
   l'association et le logo y figure, intégré au message.
11. **Given** un e-mail reçu dans un logiciel qui bloque les images, **When** le
   destinataire l'ouvre, **Then** le nom de l'association reste lisible en
   texte et le message reste compréhensible sans le logo.

---

### User Story 4 - Renseigner les réseaux sociaux et présenter le contact aux membres (Priority: P3)

Un administrateur saisit, dans la section « Général », les liens des réseaux
sociaux de l'association. Sur l'accueil de son espace, un membre voit un encart
« Contacter l'association » regroupant l'adresse électronique de contact, le
téléphone, l'adresse du siège, le site web et les réseaux sociaux renseignés.

**Why this priority**: utile aux adhérents mais secondaire ; ces informations
existent souvent ailleurs (site web, affichage au local).

**Independent Test**: renseigner une adresse de contact, un téléphone et deux
réseaux sociaux, puis ouvrir l'accueil membre et vérifier l'encart et ses liens.

**Acceptance Scenarios**:

1. **Given** un lien Instagram et un lien Facebook renseignés, **When** un
   membre ouvre l'accueil, **Then** l'encart « Contacter l'association » affiche
   ces deux liens, chacun avec le nom du réseau lisible par un lecteur d'écran.
2. **Given** une adresse électronique et un téléphone de contact renseignés,
   **When** un membre active le lien correspondant, **Then** son logiciel de
   messagerie ou d'appel s'ouvre avec la bonne adresse ou le bon numéro.
3. **Given** aucune coordonnée ni aucun réseau social renseigné, **When** un
   membre ouvre l'accueil, **Then** l'encart « Contacter l'association »
   n'apparaît pas.
4. **Given** un lien de réseau social qui n'est pas une adresse web
   (`http://` ou `https://`), **When** l'administrateur enregistre, **Then**
   l'enregistrement est refusé avec un message sur le champ concerné.
5. **Given** un lien externe affiché dans l'encart, **When** un membre l'active,
   **Then** la page s'ouvre sans donner à ce site l'accès à la page de Frigg qui
   l'a ouverte.

---

### Edge Cases

- **Nom jamais défini** (instance antérieure à l'obligation de le saisir) : la
  section « Général » affiche un champ vide, l'application continue d'afficher
  « Mon association » jusqu'au premier enregistrement.
- **Espaces superflus** : les espaces en début et en fin de chaque champ texte
  sont retirés ; un champ qui ne contient que des espaces est considéré vide
  (et donc refusé pour le nom).
- **Nom ou tagline très long** : les longueurs maximales sont appliquées à la
  saisie ; dans la barre latérale, un nom ou une tagline qui dépasse la largeur
  est tronqué visuellement, le texte complet restant accessible (lecteur d'écran,
  info-bulle).
- **Barre latérale repliée** : seul le logo (ou les initiales) reste visible ;
  le nom et la tagline restent accessibles aux lecteurs d'écran.
- **Écran étroit (320 px)** : les deux sections et l'aperçu restent utilisables
  sans défilement horizontal.
- **Deux administrateurs enregistrent en même temps** : le dernier
  enregistrement l'emporte ; aucune donnée n'est corrompue.
- **Logo très large ou très haut** : il est affiché entier, proportions
  conservées, dans l'emplacement prévu, sans déformation ni rognage.
- **Logo remplacé** : l'ancien fichier n'est plus servi ni conservé ; les
  navigateurs affichent le nouveau logo dès le rechargement, sans cache périmé.
- **Dépôt interrompu ou annulé** : le logo précédent reste en place.
- **Couleur saisie au clavier** : un code hexadécimal à 6 chiffres (`#1f6f5c`)
  est accepté, avec ou sans majuscules ; toute autre valeur est refusée avec un
  message sur le champ.
- **Couleur principale et couleur d'accent identiques** : accepté ; aucune règle
  n'impose qu'elles diffèrent.
- **Téléphone** : saisi avec ou sans espaces, points ou indicatif `+33` ; il est
  vérifié comme un numéro français et réaffiché dans un format lisible
  (`01 23 45 67 89`).
- **Code postal** : 5 chiffres ; une adresse du siège partielle (ville sans rue,
  par exemple) est acceptée, chaque élément de l'adresse étant facultatif.
- **Identité indisponible au chargement de l'écran de connexion** (erreur
  passagère) : l'écran s'affiche avec l'identité de Frigg et reste pleinement
  utilisable ; la connexion n'est jamais bloquée par l'apparence.
- **Clignotement** : à l'ouverture d'un écran de connexion, l'identité de Frigg
  ne s'affiche pas brièvement avant celle de l'association.
- **E-mails en attente au moment d'un changement** : un e-mail prend le nom, le
  logo et les couleurs en vigueur au moment de son envoi effectif.
- **Logo retiré** : les e-mails envoyés ensuite affichent le nom seul, sans
  image manquante.
- **Modification non enregistrée** : quitter la section avec des modifications
  non enregistrées demande confirmation avant de les perdre.

## Requirements *(mandatory)*

### Functional Requirements

**Accès et organisation**

- **FR-001**: La rubrique « Paramètres » de l'espace administration MUST
  comporter une section « Général » et une section « Apparence », en plus de la
  section « Paiements » ; l'ouverture de « Paramètres » MUST mener à la section
  « Général ».
- **FR-002**: La consultation des sections « Général » et « Apparence » et toute
  modification des paramètres généraux MUST être réservées aux administrateurs.
- **FR-003**: Le nom, la tagline, le logo et les couleurs MUST être disponibles
  pour toute personne connectée (membre ou administrateur), afin d'être affichés
  dans l'interface.
- **FR-003a**: Seuls le nom, le logo et les couleurs (l'« identité visuelle »)
  MUST être accessibles sans session, pour l'affichage des écrans hors session ;
  toute autre donnée de l'association (tagline, nom court, objet, identité
  légale, coordonnées, réseaux sociaux) MUST NOT l'être. Cette exposition
  publique est autorisée par l'amendement 1.8.0 du principe II de la
  constitution, dans ce seul périmètre.

**Section « Général »**

- **FR-004**: Les administrateurs MUST pouvoir modifier le nom de l'association :
  obligatoire, de 1 à 120 caractères.
- **FR-005**: Les administrateurs MUST pouvoir saisir une tagline facultative de
  150 caractères au plus et un nom court facultatif de 20 caractères au plus.
- **FR-006**: Les administrateurs MUST pouvoir saisir un objet facultatif de
  2 000 caractères au plus, sur plusieurs lignes.
- **FR-007**: Les administrateurs MUST pouvoir saisir un numéro RNA facultatif ;
  s'il est renseigné, il MUST être de la forme `W` suivi de 9 chiffres (le `w`
  minuscule est accepté et converti).
- **FR-008**: Les administrateurs MUST pouvoir saisir un numéro SIRET
  facultatif ; s'il est renseigné, il MUST comporter 14 chiffres (espaces
  ignorés) et une clé de contrôle valide.
- **FR-009**: Les administrateurs MUST pouvoir saisir une date de création
  facultative, qui MUST NOT être postérieure à la date du jour.
- **FR-010**: Les administrateurs MUST pouvoir saisir l'adresse du siège en
  éléments facultatifs : ligne d'adresse, complément, code postal (5 chiffres)
  et ville (100 caractères au plus chacun).
- **FR-011**: Les administrateurs MUST pouvoir saisir une adresse électronique
  de contact, un numéro de téléphone français et un site web, tous facultatifs
  et vérifiés lorsqu'ils sont renseignés (adresse bien formée ; numéro français
  valide ; adresse web en `http://` ou `https://`).
- **FR-012**: Les administrateurs MUST pouvoir saisir un lien facultatif pour
  chacun des réseaux sociaux suivants : Facebook, Instagram, LinkedIn, YouTube,
  Mastodon, Bluesky ; chaque lien renseigné MUST être une adresse web en
  `http://` ou `https://`.
- **FR-013**: L'enregistrement de la section « Général » MUST être
  tout-ou-rien : si un champ est invalide, aucune valeur n'est enregistrée et
  chaque champ en erreur est signalé avec un message explicite en français.
- **FR-014**: Les espaces de début et de fin de chaque champ texte MUST être
  retirés avant vérification ; un champ facultatif vide MUST être enregistré
  comme non renseigné.

**Section « Apparence »**

- **FR-015**: Les administrateurs MUST pouvoir déposer un logo au format PNG,
  JPEG ou WebP, de 1 Mo au plus ; le format MUST être vérifié d'après le contenu
  du fichier, pas d'après son nom. Les images vectorielles (SVG) ne sont pas
  acceptées.
- **FR-016**: Le dépôt d'un nouveau logo MUST remplacer le précédent, qui cesse
  d'être servi et n'est pas conservé.
- **FR-017**: Les administrateurs MUST pouvoir retirer le logo, après
  confirmation dans un dialogue ; les initiales de l'association le remplacent
  alors.
- **FR-018**: Les administrateurs MUST pouvoir choisir une couleur principale
  et une couleur d'accent, par un sélecteur de couleur ou en saisissant un code
  hexadécimal à 6 chiffres.
- **FR-019**: La section « Apparence » MUST présenter un aperçu (élément de
  barre latérale, bouton principal, lien, mise en avant) qui reflète les
  couleurs et le logo choisis avant l'enregistrement, sans modifier le reste de
  l'interface.
- **FR-020**: Quelles que soient les couleurs choisies, le texte posé sur la
  couleur principale ou d'accent et les teintes de texte dérivées de ces
  couleurs MUST présenter un contraste d'au moins 4,5:1 avec leur fond ; le
  système ajuste ces teintes automatiquement plutôt que de refuser la couleur.
- **FR-021**: Les administrateurs MUST pouvoir rétablir les couleurs par défaut
  de Frigg, après confirmation dans un dialogue.
- **FR-022**: Les couleurs sémantiques (succès, avertissement, danger,
  information) MUST NOT être modifiables ni affectées par les couleurs de
  l'association.

**Affichage**

- **FR-023**: La barre latérale MUST afficher le logo (à défaut, les initiales
  du nom), le nom et, si elle est renseignée, la tagline sous le nom, en plus de
  l'indication de l'espace (membre ou administration).
- **FR-024**: L'en-tête de l'accueil de l'espace membre MUST afficher la
  tagline lorsqu'elle est renseignée.
- **FR-025**: L'accueil de l'espace membre MUST afficher un encart « Contacter
  l'association » regroupant les coordonnées et réseaux sociaux renseignés
  (adresse électronique et téléphone sous forme de liens actionnables, liens
  externes ouverts sans exposer la page d'origine) ; l'encart MUST être absent
  si aucune de ces informations n'est renseignée.
- **FR-026**: Le titre de l'onglet du navigateur, le pied de page et les e-mails
  envoyés (expéditeur et en-tête) MUST utiliser le nom en vigueur au moment de
  l'affichage ou de l'envoi.
- **FR-027**: Après un enregistrement, l'administrateur qui l'a fait MUST voir
  le nom, la tagline, le logo et les couleurs mis à jour sans recharger la
  page ; les autres personnes connectées les voient au plus tard à leur
  prochain chargement de l'application.
- **FR-028**: Les écrans accessibles sans session (connexion, réinitialisation,
  vérification, acceptation d'invitation) MUST afficher le logo (à défaut, les
  initiales), le nom et les couleurs de l'association, avec la même garantie de
  contraste que le reste de l'interface ; si ces informations ne peuvent pas
  être obtenues, l'identité de Frigg s'affiche et l'écran reste utilisable.
- **FR-028a**: Les e-mails envoyés par l'application MUST reprendre les couleurs
  de l'association (bandeau d'en-tête, boutons, liens) avec un contraste d'au
  moins 4,5:1, et le logo lorsqu'il existe, intégré au message lui-même (sans
  lien vers un fichier à télécharger depuis l'instance).
- **FR-028b**: Le logo intégré aux e-mails MUST être une version réduite,
  adaptée à l'affichage dans un message, qui n'alourdit pas chaque e-mail de
  plus de 100 Ko ; le nom de l'association MUST rester présent en texte.

**Qualité et traçabilité**

- **FR-029**: Les sections « Général » et « Apparence » MUST être utilisables
  au clavier, avec des libellés associés à chaque champ, des messages d'erreur
  reliés aux champs, et rester utilisables dès 320 px de large.
- **FR-030**: Quitter une section avec des modifications non enregistrées MUST
  demander confirmation.
- **FR-031**: Le système MUST enregistrer dans les journaux de l'application
  chaque modification des paramètres généraux, du logo et des couleurs (compte
  auteur, nature de la modification), sans y faire figurer les valeurs des
  coordonnées.

### Key Entities

- **Association** (unique pour l'instance) : nom, tagline, nom court, objet,
  numéro RNA, numéro SIRET, date de création, adresse du siège (ligne,
  complément, code postal, ville), adresse électronique de contact, téléphone,
  site web, liens des réseaux sociaux, couleur principale, couleur d'accent,
  référence au logo, date de dernière modification.
- **Logo** : fichier image déposé, associé à l'association ; conservé sous un
  nom généré avec ses métadonnées (nom d'origine, type, taille, date de dépôt,
  compte auteur), ainsi qu'une version réduite destinée aux e-mails. Au plus un
  logo à la fois.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Un administrateur corrige le nom de l'association et ajoute une
  tagline en moins de 1 minute, sans aide extérieure ni intervention technique.
- **SC-002**: Après un enregistrement, 100 % des emplacements où figure le nom
  (barre latérale, titre d'onglet, pied de page, e-mails envoyés ensuite)
  affichent la nouvelle valeur.
- **SC-003**: 100 % des saisies invalides (RNA, SIRET, date, adresse
  électronique, téléphone, adresses web, couleurs, logo) sont refusées avec un
  message sur le champ concerné, et aucune valeur invalide n'est conservée.
- **SC-004**: Pour toute couleur choisie, 100 % des textes affichés sur les
  couleurs de l'association ou dérivés d'elles atteignent un contraste d'au
  moins 4,5:1.
- **SC-005**: Un administrateur personnalise logo et couleurs, aperçu compris,
  en moins de 3 minutes.
- **SC-006**: Sans session valide, seuls le nom, le logo et les couleurs de
  l'association peuvent être obtenus ; aucune autre donnée de l'association ne
  l'est.
- **SC-008**: 100 % des écrans hors session et des modèles d'e-mails affichent
  l'identité visuelle de l'association lorsqu'elle est définie.
- **SC-007**: Les parcours des sections « Général » et « Apparence » et
  l'accueil membre ne présentent aucune violation d'accessibilité de niveau
  sérieux ou critique.

## Assumptions

- **Saisie initiale du nom** : le nom reste demandé par `frigg create-admin` à
  l'installation ; cette fonctionnalité ajoute seulement sa modification depuis
  l'interface. Le libellé de repli « Mon association » est conservé pour les
  instances où il n'a jamais été défini.
- **Usage des données légales** : RNA, SIRET, date de création, objet et nom
  court sont conservés pour des fonctionnalités futures (reçus fiscaux,
  attestations, mentions légales) ; cette fonctionnalité ne les affiche qu'à
  l'administrateur dans les paramètres. Le nom court n'est pas encore utilisé
  ailleurs.
- **Écrans hors session et constitution** : le principe II autorise déjà le
  logo sur l'écran de connexion, mais interdit d'y exposer d'autres données de
  l'association. Afficher le nom et les couleurs avant connexion exige un
  amendement du principe II (via `/speckit-constitution`), limité à l'identité
  visuelle (nom, logo, couleurs). Ces informations sont considérées comme
  publiques : l'association les affiche de toute façon sur ses supports.
- **E-mails** : nom, logo et couleurs y figurent ; la tagline, les coordonnées
  et les réseaux sociaux n'y sont pas ajoutés dans cette fonctionnalité. Les
  logiciels de messagerie qui bloquent les images affichent le nom en texte.
- **Réseaux sociaux** : liste fermée de six réseaux couramment utilisés par les
  associations ; ajouter un réseau relève d'une évolution ultérieure. Aucun
  contrôle n'impose que le lien pointe vers le domaine du réseau.
- **Téléphone** : seuls les numéros français (métropole et outre-mer) sont
  acceptés, l'application visant uniquement des associations françaises.
- **Logo** : formats matriciels uniquement ; les SVG sont exclus car ils peuvent
  embarquer du contenu actif. Le logo n'est pas utilisé comme icône d'onglet
  (favicon) dans cette fonctionnalité.
- **Couleurs** : deux couleurs personnalisables (principale et accent), qui
  correspondent aux deux couleurs d'identité du design system ; le thème reste
  clair uniquement (principe IV).
- **Concurrence** : le dernier enregistrement l'emporte ; pas de verrouillage ni
  de détection de conflit entre administrateurs.
- **Historique** : pas de journal des modifications consultable dans
  l'interface ; la traçabilité passe par les journaux de l'application.
- **Coordonnées et RGPD** : les coordonnées saisies sont celles de
  l'association (et non d'une personne) ; si un administrateur y saisit une
  adresse ou un numéro personnel, il en porte la responsabilité.
