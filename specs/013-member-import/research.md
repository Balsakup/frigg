# Research : importation des membres

Décisions de conception de la fonctionnalité 013. Chaque section suit le
format Décision / Justification / Alternatives écartées.

## R1. Importation sans état côté serveur

**Décision** : le serveur ne conserve **jamais** le fichier. Le navigateur
garde le `File` choisi en mémoire et le renvoie à chaque étape (analyse,
aperçu, import, lignes rejetées) dans une requête multipart. Chaque requête
relit, revalide et reclasse le fichier entier, puis l'oublie.

**Justification** :

- FR-021 (fichier conservé seulement le temps de l'importation, au plus une
  heure, accessible au seul déposant) est satisfaite par construction, sans
  table, sans répertoire temporaire, sans balayage ni contrôle de
  propriétaire : rien n'est conservé.
- `humago.MultipartMaxMemory` vaut déjà `UploadMaxBody` (5 Mio + 64 Kio,
  `internal/api/api.go`) : le fichier est lu en mémoire et n'atteint jamais
  le disque, pas même dans le répertoire temporaire du système.
- Un fichier de 5 Mo au plus, envoyé trois ou quatre fois par import, est
  négligeable ; la validation de 2 000 lignes prend quelques millisecondes.
- L'import final revalide tout dans sa transaction : aucun aperçu périmé ne
  peut écrire une ligne devenue invalide.
- Un rechargement de la page perd le fichier : il faut le déposer de
  nouveau, ce que prévoit le cas limite « Abandon » de la spécification.

**Alternatives écartées** :

- *Fichier stocké dans le répertoire de données avec une table
  `member_imports` et un balayage horaire* : plus de code (migration,
  requêtes, sweeper, contrôle du déposant), une donnée personnelle de plus
  sur disque, pour seulement éviter quelques renvois de 5 Mo.
- *Analyse du CSV dans le navigateur, envoi de lignes JSON* : duplique les
  règles de validation côté client, contraire au principe VI (le serveur
  fait autorité, pas de schéma écrit à la main).

## R2. Lecture du CSV : encodage, séparateur, guillemets

**Décision** : nouveau paquet pur `internal/csvfile` :

- **Encodage** : marque d'ordre des octets UTF-8 retirée ; contenu
  `utf8.Valid` → UTF-8, sinon décodé en Windows-1252 par
  `golang.org/x/text/encoding/charmap` (module déjà dépendance directe).
  Un octet NUL refuse le fichier (fichier binaire, classeur .xlsx ou .ods
  renommé).
- **Séparateur** : sur la première ligne logique (hors guillemets), compte
  de `;`, `,` et tabulations ; le plus fréquent l'emporte, `;` en cas
  d'égalité (export par défaut des tableurs en français).
- **Lecture** : `encoding/csv` (bibliothèque standard), `LazyQuotes = false`
  (un guillemet non refermé refuse le fichier, cas limite « Fichier
  illisible »), `FieldsPerRecord = -1`, `ReuseRecord = false`. Une ligne plus
  courte que l'en-tête est complétée de valeurs vides ; une ligne plus longue
  dont les cellules excédentaires sont non vides est « en erreur ».
- **Numéro de ligne** : numéro d'enregistrement, la ligne d'intitulés valant
  1, soit le numéro de ligne affiché par le tableur qui ouvre le fichier
  (différent du numéro de ligne physique quand une valeur contient un saut de
  ligne).
- **Limites** : 5 Mio (vérifié avant lecture, `form.file`), 2 000 lignes de
  données non vides, 100 colonnes ; dépassement → `422` sur `form.file`.
- **Neutralisation inverse** : une cellule commençant par `'` suivi de `=`,
  `+`, `-`, `@`, tabulation ou retour chariot perd l'apostrophe. C'est
  l'inverse exact de `csvCell` (export des inscrits, 012) : un fichier de
  lignes rejetées produit par Frigg se réimporte sans altérer un téléphone
  `+32…` (SC-005).

**Justification** : `encoding/csv` gère les guillemets, les séparateurs et
sauts de ligne inclus ; `x/text` est déjà présent (`internal/textfold`).
Aucune dépendance ajoutée.

**Alternatives écartées** : détection d'encodage statistique (aucune
bibliothèque nécessaire : en pratique les tableurs français produisent de
l'UTF-8 ou du Windows-1252, et UTF-8 se reconnaît sans ambiguïté) ;
demander l'encodage et le séparateur à l'administrateur (contraire à FR-003).

## R3. Écriture des CSV produits par Frigg

**Décision** : `csvfile.NewWriter` écrit la marque d'ordre des octets UTF-8
puis des enregistrements séparés par `;`, chaque cellule passée par
`csvfile.SafeCell` (déplacement de `csvCell` de
`internal/api/registrations_admin.go`, qui l'appelle désormais). Utilisé pour
le modèle (FR-020) et les lignes rejetées (FR-019).

**Justification** : un seul endroit pour la protection contre l'injection de
formule et pour le format qui s'ouvre avec les accents corrects dans Excel
et LibreOffice (même choix que l'export des inscrits de 012).

## R4. Correspondance des colonnes

**Décision** : paquet pur `internal/memberimport`, fichier `mapping.go`.

- **Cibles** : `firstName`, `lastName`, `birthDate`, `email`, `phone`,
  `addressLine1`, `addressLine2`, `postalCode`, `city`, et `field:<id>` pour
  chaque champ personnalisé de membre **actif**, quelle que soit sa
  visibilité (audience administrateur).
- **Normalisation d'un intitulé** : `textfold.Fold`, suppression de `*`,
  puis suppression de tout caractère non alphanumérique (« Date de
  naissance * » → `datedenaissance`, « Tél. » → `tel`). Une seconde forme
  retire en plus une parenthèse finale (« Date de naissance (JJ/MM/AAAA) »).
- **Proposition** : pour chaque colonne, dans l'ordre : synonyme d'un champ
  de la fiche, puis libellé normalisé d'un champ personnalisé (forme
  complète, puis sans parenthèse finale) ; une cible déjà prise par une
  colonne précédente n'est pas reproposée (FR-006 : une colonne au plus par
  cible). Synonymes (formes normalisées) :

  | Cible | Synonymes |
  | --- | --- |
  | firstName | prenom, prenoms, firstname, prenomdumembre, prenomadherent |
  | lastName | nom, noms, nomdefamille, lastname, name, nomdumembre, nomadherent, nomdusage |
  | birthDate | datedenaissance, naissance, datenaissance, ddn, nele, neele, birthdate, dateofbirth |
  | email | email, mail, courriel, adresseemail, adressemail, adresseelectronique, emailadherent |
  | phone | telephone, tel, portable, mobile, telportable, telephoneportable, phone, numerodetelephone |
  | addressLine1 | adresse, adressepostale, rue, adresseligne1, ligne1, address |
  | addressLine2 | complement, complementdadresse, adresseligne2, ligne2, adresse2 |
  | postalCode | codepostal, cp, zip, postalcode |
  | city | commune, ville, localite, city |

- **Validation de la correspondance** (`422`, `form.mapping`) : tableau de
  même longueur que les colonnes ; valeurs connues ; cible au plus une fois ;
  `firstName`, `lastName` et `birthDate` présents (FR-007), message nommant
  le champ manquant.

**Justification** : règles déterministes et testables en table ; le modèle
de fichier (R7) emploie des intitulés reconnus d'office (SC-002).

**Alternatives écartées** : distance d'édition entre intitulés (propositions
surprenantes, « Nom » ↔ « Nom du parent ») ; mémoriser la dernière
correspondance (hors périmètre, hypothèse de la spécification).

## R5. Validation d'une ligne

**Décision** : `memberimport.ParseRow(cells, mapping, defs, today)` produit
un `member.Input`, les valeurs de champs personnalisés et **toutes** les
erreurs, chacune rattachée à une colonne (FR-009).

- **Fiche** : la date de naissance est d'abord lue par `ParseDate`, puis la
  saisie passe par `member.Validate` (005) sans aucune règle nouvelle ;
  chaque `member.FieldError` est rattachée à la colonne de son champ, une
  erreur d'adresse incomplète à la colonne manquante, ou à défaut à la
  colonne « adresse ». Les messages sont ceux du formulaire de fiche.
- **Dates** (`ParseDate`, fiche et champs de type date) : `J/M/AAAA`,
  `J-M-AAAA`, `J.M.AAAA` (jour et mois sur 1 ou 2 chiffres) et `AAAA-MM-JJ` ;
  date calendaire vérifiée ; année sur 2 chiffres refusée avec un message
  dédié (« Indiquez l'année sur 4 chiffres »).
- **Champs personnalisés** : chaque valeur passe par
  `customfield.NormalizeValue` (011) après conversion :
  - texte court, texte long : tel quel (valeur multiligne refusée pour un
    texte court, limites de longueur sans troncature) ;
  - nombre : texte brut (`ParseNumber` accepte « 12,5 ») ;
  - date : `ParseDate` puis ISO ;
  - oui/non : `oui|vrai|1|x` → coché, `non|faux|0` → non coché (repliés) ;
    autre valeur → erreur ; comme dans le formulaire, « non » n'enregistre
    pas de valeur ;
  - liste de choix : choix dont le libellé replié égale la valeur repliée ;
    sinon erreur listant les choix (tronquée à 10 choix et « … »).
- **Champ personnalisé obligatoire** : `customfield.Apply` n'est **pas**
  utilisé, car il impose les champs obligatoires ; une valeur obligatoire
  absente n'est pas une erreur (FR-013) mais compte dans
  `missingRequired`.

**Justification** : les mêmes règles que la saisie manuelle (SC-005 : un
fichier corrigé passe), sans dupliquer de validation.

## R6. Doublons

**Décision** : `memberimport.Classify` reçoit les lignes valides et la liste
des identités existantes (`id`, prénom, nom, date de naissance) ; la clé est
`member.IdentityKey(prénom, nom)` + date ISO, exactement la règle de
`checkDuplicate` (005). Une ligne dont la clé existe en base est « doublon »
avec la fiche ; sinon, une ligne dont la clé figure plus haut dans le fichier
est « doublon » avec le numéro de cette ligne. Une ligne en erreur n'occupe
pas de clé (la ligne suivante identique et valide est importée).

Nouvelle requête `ListMemberIdentities` (id, first_name, last_name,
birth_date) lue :

- sur le pool de lecture pour l'aperçu ;
- **dans la transaction d'écriture** pour l'import (FR-015) : l'écrivain
  unique en `BEGIN IMMEDIATE` sérialise les imports et les saisies ; deux
  imports concurrents ou une fiche créée entre l'aperçu et la confirmation
  ne produisent aucun doublon (SC-004). Les lignes devenues doublons sont
  comptées dans `duplicates` du rapport, avec `newDuplicates` pour celles
  apparues depuis l'aperçu. Sans état côté serveur, le navigateur renvoie à
  la confirmation le nombre de lignes `ready` de l'aperçu (`expectedReady`) :
  `newDuplicates = max(expectedReady - created, 0)` (une erreur nouvelle
  supposerait des champs changés, donc un `409`).

**Alternatives écartées** : index unique sur l'identité repliée (exigerait
une colonne repliée, interdite par le principe VI ; et le doublon reste
confirmable dans la saisie manuelle) ; requête par date de naissance pour
chaque ligne (2 000 requêtes au lieu d'une).

## R7. Modèle de fichier

**Décision** : `GET /admin/members/import/template` renvoie un CSV (R3) à
une seule ligne d'intitulés : « Prénom * », « Nom * », « Date de naissance *
(JJ/MM/AAAA) », « Adresse électronique », « Téléphone », « Adresse »,
« Complément d'adresse », « Code postal », « Commune », puis un intitulé par
champ personnalisé de membre actif, dans l'ordre défini : son libellé,
suivi d'une indication entre parenthèses selon le type — « (oui/non) »,
« (JJ/MM/AAAA) », « (nombre) », ou les choix (« (Poussin, Benjamin, …) »,
10 au plus). Fichier `modele-import-membres.csv`.

**Justification** : la normalisation de R4 retire `*` et la parenthèse
finale, donc chaque intitulé du modèle est reconnu d'office (US4, SC-002).
Seuls les trois champs exigés par l'import portent `*` : un champ
personnalisé obligatoire n'empêche pas l'import (FR-013).

## R8. Version des champs personnalisés

**Décision** : l'analyse et l'aperçu renvoient `fieldsVersion`, empreinte
SHA-256 (hexadécimal, 16 premiers octets) de la définition canonique des
champs personnalisés de membre actifs : identifiant, type, libellé,
identifiants et libellés des choix, dans l'ordre des identifiants. L'import
reçoit `fieldsVersion` ; s'il diffère de l'empreinte calculée dans la
transaction, réponse `409` « Les champs personnalisés ont changé depuis
l'aperçu » et le frontend relance l'aperçu (FR-017).

**Justification** : sans état serveur (R1), une empreinte suffit ; aucune
colonne de version à ajouter à `custom_fields`.

**Alternatives écartées** : colonne `updated_at` sur les définitions et les
choix (migration, et ne couvre pas l'archivage sans soin particulier).

## R9. Écriture de l'import

**Décision** : `WriteTx` unique : empreinte (R8), identités (R6), puis pour
chaque ligne à importer `CreateMember` (`account_id` NULL) et
`UpsertMemberFieldValue` pour chaque valeur non vide. Toute erreur annule
l'ensemble (FR-016). Aucune file d'e-mail n'est alimentée (FR-014).
Aucune ligne à importer après reclassement → `422` « Aucune ligne à
importer » sans écriture.

**Performances** : 2 000 insertions (et leurs triggers FTS5) dans une
transaction prennent bien moins d'une seconde en WAL ; la transaction bloque
les autres écritures pendant ce temps, ce qui reste acceptable pour une
opération rare.

## R10. Journaux

**Décision** : une ligne `slog` par import confirmé (FR-023) :
`action=member.import`, `actor_id`, `created`, `errors`, `duplicates`,
`new_duplicates`, `missing_required`, `first_member_id`, `last_member_id`
(identifiants contigus : insertions consécutives sous l'écrivain unique,
dans une seule transaction). Aucun nom de fichier, intitulé, valeur ni
donnée personnelle. Elle tient lieu de trace `member.create` de 005
(FR-017) pour les fiches importées, qui naissent « Sans compte » (pas de
`member.attach`). Un test vérifie l'absence de prénom, nom ou adresse dans
les journaux.

## R11. Opérations et contrat

**Décision** : cinq opérations `auth.Admin`, tag `members` (détails :
[contracts/api.md](./contracts/api.md)) :

| operationId | Méthode et chemin | Rôle |
| --- | --- | --- |
| `get-member-import-template` | `GET /admin/members/import/template` | modèle CSV |
| `analyze-member-import` | `POST /admin/members/import/analyze` | colonnes, exemples, proposition |
| `preview-member-import` | `POST /admin/members/import/preview` | aperçu complet |
| `run-member-import` | `POST /admin/members/import` | création des fiches |
| `export-member-import-rejected` | `POST /admin/members/import/rejected` | CSV des lignes rejetées |

Les quatre `POST` sont multipart (`file`, plus `mapping` en chaîne JSON et
`fieldsVersion` selon l'opération), `MaxBodyBytes = UploadMaxBody`. Les
routes `/admin/members/import…` sont déclarées avant `/admin/members/{id}`
et l'identifiant de fiche est numérique : aucune ambiguïté de routage.
`analyze` ne fait que lire le fichier et proposer ; il est séparé de
`preview` pour que l'étape de correspondance s'affiche avant toute
validation des lignes.

## R12. Frontend

**Décision** : route `/admin/membres/importer`
(`routes/app/admin/member-import.tsx`), déclarée avant `:id`, avec un
assistant à quatre étapes tenues dans l'état du composant (le fichier ne
peut pas vivre dans l'adresse) : Fichier → Correspondance → Aperçu →
Rapport. Composants dans `web/app/members/import/`. Appels dans
`web/app/api/memberImport.ts` (multipart par `bodySerializer` comme
`uploadAttachment`, téléchargements par `parseAs: "blob"` comme l'export des
inscrits). Bouton « Importer » dans l'en-tête de la page « Membres ».
Détails : [contracts/ui-routes.md](./contracts/ui-routes.md).

**Accessibilité** : liste ordonnée des étapes avec `aria-current="step"` ;
au changement d'étape, focus sur le titre `h2` de l'étape ; compteurs de
l'aperçu et du rapport dans une région `aria-live="polite"` ; filtre de
l'aperçu par `RadioGroup` base-ui ; tableau de l'aperçu avec en-têtes de
colonne, erreurs en liste dans la cellule ; dépôt par `input type=file`
étiqueté (glisser-déposer en complément, jamais seul moyen).

**Données** : les réponses d'analyse et d'aperçu sont des résultats de
mutation (`useMutation`), pas des requêtes en cache : elles dépendent d'un
fichier local. Après import réussi, invalidation des requêtes de la liste
des membres et du décompte des valeurs obligatoires manquantes (011).

## R13. Tests

- `internal/csvfile` : encodages (UTF-8 avec et sans BOM, Windows-1252 avec
  accents), séparateurs, égalité → `;`, guillemets avec séparateur et saut
  de ligne, guillemet non refermé, octet NUL, apostrophe de neutralisation
  aller-retour avec `SafeCell`.
- `internal/memberimport` (tests en table) : normalisation des intitulés et
  propositions (synonymes, champs personnalisés, parenthèse finale, cible
  déjà prise, intitulés vides ou en double), validation de correspondance,
  `ParseDate`, oui/non, choix, nombres, erreurs multiples par ligne,
  classification des doublons (base, fichier, ligne en erreur).
- API (`members_import_test.go`, httptest) : parcours complet ; modèle
  reconnu d'office ; limites de taille, de lignes, de colonnes ; fichier
  .xlsx refusé ; aucune ligne à importer ; empreinte changée → `409` ;
  doublon créé entre aperçu et import ; deux imports concurrents du même
  fichier → aucune fiche en double ; atomicité (échec injecté → aucune
  fiche) ; lignes rejetées réimportées après correction (SC-005) ; aucune
  donnée personnelle dans les journaux ; aucun e-mail en file.
- Matrice d'autorisation existante : cinq opérations `admin` ajoutées.
- Playwright + axe : `web/e2e/member-import.spec.ts` (fichier d'exemple avec
  lignes valides, erreurs et doublons ; parcours au clavier ; aucune
  violation `serious` ni `critical`).
