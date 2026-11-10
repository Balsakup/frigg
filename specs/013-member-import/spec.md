# Feature Specification: Importation des membres

**Feature Branch**: `013-member-import`

**Created**: 2026-11-10

**Status**: Draft

**Input**: User description: "Importation des membres"

## Vocabulaire

- **Fichier d'import** : un fichier CSV fourni par un administrateur (export
  d'un ancien registre, d'un logiciel précédent ou d'un tableau tenu à la
  main et enregistré au format CSV depuis le tableur), dont chaque ligne
  décrit une personne et la première ligne nomme les colonnes.
- **Correspondance des colonnes** : l'association, choisie par
  l'administrateur, entre chaque colonne du fichier et un champ de la fiche
  membre (005) ou un champ personnalisé de membre (011), ou l'indication
  « Ne pas importer ».
- **Aperçu** : l'état de chaque ligne du fichier avant toute écriture :
  **à importer**, **en erreur** (donnée manquante ou invalide) ou **doublon**
  (la personne existe déjà dans Frigg ou plus haut dans le fichier).
- **Rapport d'import** : le bilan affiché après l'importation (fiches créées,
  lignes ignorées et pourquoi).

## Clarifications

### Session 2026-11-10

- Q : D'où viennent les données à importer ? → R : d'un tableur générique ;
  l'administrateur associe chaque colonne à un champ de la fiche, champs
  personnalisés compris. Aucun format propre à un prestataire n'est reconnu.
- Q : Quels formats de fichier sont acceptés ? → R : uniquement le CSV ; un
  classeur (.xlsx, .ods…) est enregistré au format CSV depuis le tableur
  avant l'import.
- Q : Que crée l'importation, au-delà des fiches ? → R : seulement des fiches
  membres, « Sans compte ». Ni compte, ni invitation, ni adhésion.
- Q : Que faire d'une ligne qui correspond à une fiche existante (même
  prénom, nom et date de naissance) ? → R : l'ignorer ; la fiche existante
  n'est pas modifiée et la ligne est signalée dans le rapport.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Importer les membres d'un tableur (Priority: P1)

L'association existait bien avant Frigg : Sophie, trésorière et
administratrice, tient la liste des adhérents dans un tableur. Depuis la
rubrique « Membres » de l'administration, elle choisit « Importer », dépose
son tableur enregistré au format CSV, et Frigg lui propose une correspondance des colonnes en
reconnaissant « Prénom », « NOM », « Date de naissance », « Mail » et « Tél. ».
Elle confirme, voit dans l'aperçu que 212 lignes sont prêtes à être
importées, puis lance l'importation. Le rapport annonce 212 fiches créées ;
elle les retrouve dans la liste « Membres », signalées « Sans compte ».

**Why this priority**: sans import, le bureau doit ressaisir à la main des
centaines de fiches avant de pouvoir utiliser Frigg ; c'est le principal frein
à la mise en service pour une association existante.

**Independent Test**: avec un fichier de quelques lignes valides, réaliser
l'import de bout en bout et vérifier que chaque ligne a donné une fiche « Sans
compte » avec les bonnes valeurs.

**Acceptance Scenarios**:

1. **Given** un administrateur dans la rubrique « Membres », **When** il
   choisit « Importer », **Then** une page explique le déroulé (fichier,
   correspondance, aperçu, import), le format accepté (CSV, avec la marche à
   suivre pour enregistrer un tableur dans ce format), les champs
   obligatoires, et permet de déposer un fichier.
2. **Given** un fichier dont la première ligne porte des intitulés usuels
   (« Prénom », « nom », « Date de naissance », « E-mail », « Téléphone »,
   « Code postal », « Ville »…), **When** le fichier est lu, **Then** chaque
   colonne reconnue est déjà associée au champ correspondant, sans tenir
   compte de la casse, des accents ni de la ponctuation, et les colonnes non
   reconnues sont proposées en « Ne pas importer ».
3. **Given** l'étape de correspondance, **When** l'administrateur modifie
   l'association d'une colonne, **Then** chaque colonne affiche son intitulé
   et quelques valeurs d'exemple, et un même champ ne peut être associé qu'à
   une seule colonne.
4. **Given** une correspondance où prénom, nom ou date de naissance n'est
   associé à aucune colonne, **When** l'administrateur veut passer à
   l'aperçu, **Then** il en est empêché et le champ manquant est nommé.
5. **Given** une correspondance complète, **When** l'administrateur passe à
   l'aperçu, **Then** il voit le nombre de lignes à importer, en erreur et en
   doublon, et peut consulter chaque ligne avec son numéro dans le fichier.
   Aucune fiche n'est encore créée.
6. **Given** l'aperçu, **When** l'administrateur confirme l'importation,
   **Then** une fiche « Sans compte » est créée pour chaque ligne à importer,
   un rapport indique le nombre de fiches créées et de lignes ignorées, et la
   liste « Membres » reflète immédiatement les nouvelles fiches.
7. **Given** l'aperçu ou l'étape de correspondance, **When** l'administrateur
   abandonne ou revient en arrière, **Then** aucune fiche n'est créée et il
   peut corriger la correspondance ou déposer un autre fichier.
8. **Given** un membre connecté ou une personne non connectée, **When** il
   tente d'accéder à l'importation par quelque moyen que ce soit, **Then**
   l'accès est refusé.

---

### User Story 2 - Comprendre et corriger les lignes rejetées (Priority: P1)

Le tableur de Sophie est imparfait : certaines dates sont écrites « 3/4/12 »,
une adresse électronique a perdu son arobase, deux lignes n'ont pas de nom, et
les Martin figurent deux fois. L'aperçu les signale ligne par ligne avec la
raison. Sophie importe les lignes valides, télécharge le fichier des lignes
rejetées, le corrige dans son tableur et l'importe à son tour.

**Why this priority**: un fichier tenu à la main n'est jamais propre ; sans
explication précise, l'administrateur ne peut ni faire confiance à l'import ni
le compléter. Ce récit conditionne la valeur de US1 sur de vrais fichiers.

**Independent Test**: importer un fichier mêlant lignes valides, lignes
invalides de chaque sorte et doublons ; vérifier l'aperçu, le rapport, le
fichier des lignes rejetées, puis réimporter ce fichier corrigé.

**Acceptance Scenarios**:

1. **Given** une ligne sans prénom, sans nom ou sans date de naissance, ou dont
   une valeur ne respecte pas les règles de la fiche (005) ou du champ
   personnalisé (011), **When** l'aperçu s'affiche, **Then** la ligne est
   « en erreur », avec son numéro, la colonne en cause et un message en
   français expliquant la correction attendue ; toutes les erreurs de la
   ligne sont listées, pas seulement la première.
2. **Given** une ligne de même prénom, même nom et même date de naissance
   qu'une fiche existante (sans tenir compte de la casse, des accents ni des
   espaces superflus), **When** l'aperçu s'affiche, **Then** la ligne est
   « doublon » et sera ignorée ; la fiche existante est indiquée avec un lien
   vers elle.
3. **Given** deux lignes du fichier décrivant la même personne selon la même
   règle, **When** l'aperçu s'affiche, **Then** seule la première ligne
   valide est à importer et les suivantes sont « doublon », avec le numéro de
   celle-ci ; une ligne en erreur n'est pas retenue comme première.
4. **Given** l'aperçu, **When** l'administrateur filtre sur « En erreur » ou
   « Doublon », **Then** seules les lignes de cet état s'affichent.
5. **Given** un aperçu comportant des lignes en erreur, **When**
   l'administrateur confirme l'importation, **Then** seules les lignes à
   importer sont créées ; le rapport rappelle le nombre de lignes en erreur
   et en doublon.
6. **Given** le rapport ou l'aperçu, **When** l'administrateur télécharge les
   lignes rejetées, **Then** il obtient un fichier CSV reprenant les colonnes
   d'origine des lignes en erreur et en doublon, complétées d'une colonne
   indiquant la raison du rejet, réimportable après correction.
7. **Given** un aperçu sans aucune ligne à importer, **When** il s'affiche,
   **Then** l'importation n'est pas proposée et un message invite à corriger
   le fichier ou la correspondance.

---

### User Story 3 - Importer les informations complémentaires (Priority: P2)

L'association a défini un champ personnalisé « Numéro de licence » et un champ
« Catégorie » de type liste de choix (011). Le tableur de Sophie contient ces
deux colonnes : elle les associe aux champs personnalisés correspondants et
les valeurs sont enregistrées sur les fiches créées.

**Why this priority**: les informations propres à l'association font partie
de l'historique à reprendre, mais un import limité à l'identité et aux
coordonnées apporte déjà l'essentiel.

**Independent Test**: définir un champ de chaque type, importer un fichier
contenant une colonne par champ, et vérifier les valeurs enregistrées et les
erreurs signalées pour des valeurs hors limites.

**Acceptance Scenarios**:

1. **Given** des champs personnalisés de membre actifs, **When**
   l'administrateur définit la correspondance, **Then** ces champs sont
   proposés comme cibles, quelle que soit leur visibilité, et une colonne
   dont l'intitulé est le libellé d'un champ y est associée d'office.
2. **Given** un champ « liste de choix », **When** une valeur du fichier
   correspond à l'un de ses choix sans tenir compte de la casse ni des
   accents, **Then** ce choix est retenu ; sinon la ligne est en erreur et le
   message rappelle les choix possibles.
3. **Given** un champ « oui/non », **When** la valeur vaut « oui », « non »,
   « vrai », « faux », « 1 », « 0 » ou « x » (quelle que soit la casse),
   **Then** elle est interprétée ; une autre valeur met la ligne en erreur.
4. **Given** un champ personnalisé obligatoire non associé ou vide sur une
   ligne, **When** la ligne est importée, **Then** la fiche est créée sans
   cette valeur, et l'aperçu comme le rapport indiquent combien de fiches
   resteront à compléter.
5. **Given** un champ personnalisé archivé, **When** l'administrateur définit
   la correspondance, **Then** il n'est pas proposé.

---

### User Story 4 - Partir d'un modèle de fichier (Priority: P3)

Marc, nouvel administrateur d'une association qui tenait ses adhérents sur
papier, télécharge depuis la page d'import un modèle de fichier CSV dont les
colonnes portent déjà les intitulés reconnus, le remplit et l'importe sans
avoir à retoucher la correspondance.

**Why this priority**: confort qui réduit les erreurs de correspondance, mais
l'import fonctionne avec tout fichier CSV.

**Independent Test**: télécharger le modèle, le remplir de quelques lignes et
constater que toutes les colonnes sont reconnues à l'étape de correspondance.

**Acceptance Scenarios**:

1. **Given** la page d'import, **When** l'administrateur télécharge le modèle,
   **Then** il obtient un fichier CSV comportant une colonne par champ de la fiche
   et par champ personnalisé de membre actif, sans aucune donnée de membre,
   avec les champs obligatoires signalés et le format de date attendu
   rappelé.
2. **Given** le modèle rempli, **When** il est importé, **Then** toutes ses
   colonnes sont associées d'office au bon champ.

---

### Edge Cases

- **Fichier illisible** : un fichier d'un autre format (classeur .xlsx ou
  .ods compris), vide, sans ligne d'intitulés ou dont les lignes n'ont pas
  une structure cohérente (guillemets non refermés…) est refusé avec un
  message rappelant que seul le CSV est accepté et comment y enregistrer un
  tableur ; rien n'est créé.
- **Taille** : au-delà de 2 000 lignes de données ou de 5 Mo, le fichier est
  refusé avec un message invitant à le découper.
- **Encodage et séparateur** : un fichier texte encodé en UTF-8 (avec ou sans
  marque d'ordre des octets) ou en Windows-1252, séparé par des virgules, des
  points-virgules ou des tabulations, est lu correctement sans que
  l'administrateur ait à le préciser ; les accents sont conservés.
- **Intitulés en double ou vides** : deux colonnes de même intitulé restent
  distinctes et sont désignées par leur position ; une colonne sans intitulé
  est proposée en « Ne pas importer ».
- **Lignes vides** : les lignes entièrement vides sont ignorées sans être
  comptées comme erreurs.
- **Espaces superflus** : les espaces en début et fin de valeur sont retirés ;
  une valeur ne contenant que des espaces est considérée comme vide.
- **Dates** : sont acceptés les formats JJ/MM/AAAA, JJ-MM-AAAA, JJ.MM.AAAA
  et AAAA-MM-JJ ; le jour précède toujours le mois. Une année sur deux chiffres est refusée (ambiguïté de siècle).
- **Téléphones et codes postaux** : un code postal ou un numéro de téléphone
  dont le zéro initial a été perdu par le tableur (« 6123456 » au lieu de
  « 06… », « 1000 » au lieu de « 01000 ») est signalé en erreur, sans
  correction devinée.
- **Fiche créée entre l'aperçu et la confirmation** : la règle de doublon est
  réappliquée à la confirmation ; la ligne correspondante est ignorée et
  comptée comme doublon dans le rapport.
- **Champ personnalisé modifié ou archivé entre l'aperçu et la confirmation** :
  l'importation est refusée avec un message explicite et l'aperçu est
  recalculé.
- **Deux importations simultanées** par deux administrateurs : chacune
  applique la règle de doublon au moment de sa confirmation ; aucun doublon
  n'est créé par leur concurrence.
- **Abandon** : quitter la page ou fermer le navigateur avant la
  confirmation n'écrit rien et ne laisse aucune copie du fichier dans Frigg ;
  pour reprendre, l'administrateur dépose de nouveau le fichier.
- **Échec en cours d'import** : l'importation est tout ou rien pour les lignes
  à importer ; en cas d'incident, aucune fiche n'est créée et un message
  invite à recommencer.
- **Valeur trop longue** : une valeur dépassant la longueur maximale du champ
  met la ligne en erreur ; elle n'est jamais tronquée.
- **Formules et contenu actif** : les valeurs sont lues comme du texte ;
  dans le fichier des lignes rejetées, une valeur commençant par « = », « + »,
  « - » ou « @ » est neutralisée pour ne pas être interprétée comme une
  formule à l'ouverture.

## Requirements *(mandatory)*

### Functional Requirements

**Fichier**

- **FR-001**: Les administrateurs MUST pouvoir importer des fiches membres à
  partir d'un fichier CSV, depuis la rubrique « Membres » de
  l'administration. Aucun autre format n'est accepté.
- **FR-002**: Le système MUST refuser un fichier vide, illisible, d'un autre
  format, de plus de 5 Mo ou de plus de 2 000 lignes de données, avec un
  message en français indiquant la limite ou le format attendu.
- **FR-003**: Le système MUST détecter sans intervention l'encodage (UTF-8 ou
  Windows-1252) et le séparateur (virgule, point-virgule, tabulation) du
  fichier, et MUST respecter les valeurs entre guillemets (séparateurs et
  sauts de ligne compris).
- **FR-004**: La première ligne du fichier MUST être traitée comme la ligne
  des intitulés de colonnes ; les lignes entièrement vides sont ignorées.

**Correspondance des colonnes**

- **FR-005**: Le système MUST proposer, pour chaque colonne, une
  correspondance déduite de son intitulé (sans casse, accents, espaces ni
  ponctuation, synonymes usuels compris) parmi les champs de la fiche membre
  (prénom, nom, date de naissance, adresse électronique, téléphone, ligne
  d'adresse, complément, code postal, commune) et les champs personnalisés de
  membre actifs ; une colonne non reconnue est proposée en « Ne pas
  importer ».
- **FR-006**: L'administrateur MUST pouvoir modifier chaque correspondance en
  voyant l'intitulé et des valeurs d'exemple de la colonne ; un champ MUST
  être associé à une colonne au plus.
- **FR-007**: Le passage à l'aperçu MUST être refusé tant que le prénom, le
  nom et la date de naissance ne sont pas associés chacun à une colonne.

**Aperçu et validation**

- **FR-008**: Avant toute écriture, le système MUST présenter un aperçu
  donnant le nombre de lignes à importer, en erreur et en doublon, et la
  liste des lignes avec leur numéro dans le fichier, filtrable par état.
- **FR-009**: Chaque ligne MUST être validée selon les règles de la fiche
  membre (005, FR-001 et FR-002) et des champs personnalisés (011, FR-003 et
  FR-014) ; une ligne non conforme est « en erreur » et l'aperçu MUST lister
  toutes ses erreurs, avec la colonne concernée et un message en français
  expliquant la correction attendue.
- **FR-010**: Les dates MUST être lues au format jour-mois-année (séparateurs
  « / », « - » ou « . ») ou au format année-mois-jour avec tirets ; une
  année sur deux chiffres est refusée.
- **FR-011**: Une valeur de liste de choix MUST correspondre à un choix
  existant du champ sans tenir compte de la casse ni des accents ; une valeur
  oui/non MUST être reconnue parmi « oui », « non », « vrai », « faux »,
  « 1 », « 0 » et « x ». Aucune valeur n'est tronquée ni corrigée par
  supposition.
- **FR-012**: Une ligne MUST être « doublon » si une fiche existante, ou une
  ligne précédente du fichier à importer, a le même prénom, le même nom et la
  même date de naissance selon la règle de doublon de 005 (FR-005) ; l'aperçu
  indique la fiche existante (avec un lien) ou le numéro de la ligne
  précédente.
- **FR-013**: L'absence de valeur pour un champ personnalisé obligatoire MUST
  NOT empêcher l'import de la ligne ; l'aperçu et le rapport MUST indiquer le
  nombre de fiches importées auxquelles il manquera une valeur obligatoire.

**Importation**

- **FR-014**: À la confirmation, le système MUST créer une fiche membre
  « Sans compte » pour chaque ligne à importer, avec les valeurs associées
  (champs personnalisés compris), et ignorer les lignes en erreur et en
  doublon. Aucun compte, aucune invitation, aucune adhésion ni aucun message
  électronique ne résulte de l'importation.
- **FR-015**: La règle de doublon MUST être réappliquée au moment de la
  confirmation, de sorte qu'aucune importation, même simultanée à une autre
  ou à une saisie manuelle, ne crée de doublon.
- **FR-016**: La création des fiches d'une importation MUST être tout ou
  rien : en cas d'incident, aucune fiche de cette importation n'est créée.
- **FR-017**: Si un champ personnalisé de membre a été créé, modifié ou
  archivé entre l'aperçu et la confirmation, l'importation MUST être refusée avec un
  message explicite et l'aperçu recalculé.
- **FR-018**: Après l'importation, le système MUST afficher un rapport
  donnant le nombre de fiches créées, de lignes en erreur, de lignes en
  doublon et de fiches à compléter, avec un lien vers la liste « Membres ».
- **FR-019**: L'administrateur MUST pouvoir télécharger, depuis l'aperçu et le
  rapport, un fichier CSV des lignes rejetées (en erreur et en doublon) reprenant
  leurs colonnes d'origine et une colonne « Raison du rejet » ; les valeurs
  susceptibles d'être interprétées comme des formules MUST y être
  neutralisées.
- **FR-020**: L'administrateur MUST pouvoir télécharger un modèle de fichier
  CSV comportant une colonne par champ importable, dont les intitulés sont
  reconnus d'office à l'étape de correspondance.

**Données personnelles et sécurité**

- **FR-021**: Le fichier déposé MUST n'être jamais conservé par Frigg : il
  n'est lu que le temps de traiter chaque étape et n'est écrit ni sur disque
  ni en base. Seules les fiches créées à la confirmation sont enregistrées.
- **FR-022**: Toutes les étapes de l'importation MUST être refusées aux
  membres et aux personnes non connectées ; ce contrôle est appliqué côté
  serveur.
- **FR-023**: Le système MUST enregistrer dans les journaux de l'application
  chaque importation confirmée, avec l'administrateur à l'origine, la date et
  les nombres de fiches créées, de lignes en erreur et en doublon, sans y
  faire figurer le contenu du fichier ni aucune donnée personnelle.

**Commun**

- **FR-024**: Les écrans MUST être en français, utilisables au clavier seul,
  compatibles avec les lecteurs d'écran (étape courante annoncée, résultats
  de l'aperçu et du rapport restitués) et utilisables dès 320 pixels de
  large.

### Key Entities

- **Importation** : une opération menée par un administrateur, du dépôt du
  fichier à la confirmation. Attributs : fichier déposé (gardé par le
  navigateur de l'administrateur, jamais par Frigg) et correspondance des
  colonnes. Rien n'en est conservé hors des fiches créées ; seule sa trace
  subsiste dans les journaux.
- **Ligne d'aperçu** : numéro dans le fichier, état (à importer, en erreur,
  doublon), erreurs par colonne, fiche ou ligne en double le cas échéant.
- **Membre** (existant, 005) : fiche créée « Sans compte » par l'importation,
  identique à une fiche saisie à la main.
- **Champ personnalisé de membre** (existant, 011) : cible possible d'une
  colonne ; ses règles valident les valeurs importées.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Un administrateur importe un fichier de 500 membres aux
  intitulés usuels en moins de 5 minutes, du dépôt du fichier au rapport.
- **SC-002**: Pour un fichier aux intitulés usuels (ceux du modèle et leurs
  variantes courantes), au moins 90 % des colonnes utiles sont associées au
  bon champ sans intervention.
- **SC-003**: 100 % des lignes non importées figurent dans l'aperçu et dans le
  fichier des lignes rejetées, avec leur numéro et une raison compréhensible.
- **SC-004**: Aucune importation ne crée de doublon au sens de la règle de
  005, y compris en cas d'importations simultanées (vérifié par des tests
  automatisés).
- **SC-005**: Un fichier des lignes rejetées, corrigé puis réimporté, ne
  produit aucune nouvelle erreur sur les lignes corrigées et aucun doublon des
  lignes déjà importées.
- **SC-006**: 100 % des opérations de l'importation sont refusées à un compte
  membre et à une personne non connectée.
- **SC-007**: Aucun écran livré ne présente de violation d'accessibilité grave
  ou critique lors des contrôles automatisés et d'un parcours complet au
  clavier.

## Assumptions

- **Fiches seulement** (décision du 2026-11-10) : l'import ne crée que des
  fiches « Sans compte ». Le rattachement à un compte se fait ensuite par la
  fiche (005, US4) ; la création de comptes ou d'invitations en masse et la
  reprise de l'historique des adhésions sont hors périmètre et pourront faire
  l'objet de spécifications ultérieures.
- **Pas de mise à jour** (décision du 2026-11-10) : une ligne qui correspond à
  une fiche existante est ignorée ; l'import ne modifie jamais une fiche
  existante.
- **Format générique** (décision du 2026-11-10) : aucun format propre à un
  prestataire (HelloAsso…) n'est reconnu en tant que tel ; un export de ce
  type s'importe comme tout fichier CSV, via la correspondance des colonnes.
- **Date de naissance obligatoire** : comme pour toute fiche (005), une ligne
  sans date de naissance est en erreur ; l'association complète son fichier
  avant l'import.
- **Champs personnalisés obligatoires** : leur absence n'empêche pas l'import,
  afin de ne pas bloquer la reprise d'un historique incomplet ; les fiches
  concernées apparaissent dans le décompte des valeurs manquantes de 011
  (FR-026).
- **« Non » sur un champ oui/non** : comme dans la fiche (011), « non »
  n'enregistre aucune valeur. Un champ oui/non ne pouvant pas être
  obligatoire (011), cela n'ajoute aucune fiche à compléter.
- **Pas d'annulation d'une importation** : une fois confirmée, l'importation
  donne des fiches ordinaires, supprimables une à une ; l'aperçu obligatoire
  sert de garde-fou.
- **Pas de mémorisation de la correspondance** : elle est proposée à chaque
  import à partir des intitulés ; le modèle de fichier évite de la retoucher.
- **Limite de 20 fiches par compte** (005, FR-004) : sans objet, les fiches
  importées n'étant rattachées à aucun compte.
- **Volume** : 2 000 lignes par fichier couvrent l'effectif d'une
  association visée par Frigg ; un effectif plus grand s'importe en plusieurs
  fichiers, la règle de doublon empêchant les doubles imports.
- **CSV uniquement** (décision du 2026-11-10) : tout tableur courant sait
  enregistrer au format CSV ; les classeurs (.xlsx, .xls, .ods) ne sont pas
  lus directement. Les fichiers produits par Frigg (modèle, lignes rejetées)
  sont eux aussi en CSV, encodés de façon à s'ouvrir avec les accents
  corrects dans un tableur courant.
