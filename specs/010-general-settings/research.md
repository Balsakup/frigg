# Research: Paramètres généraux de l'association

**Feature**: `010-general-settings` | **Date**: 2026-11-03 | **Spec**: [spec.md](./spec.md)

Aucun point de la spécification ne restait à clarifier. Les choix ci-dessous
partent du code existant (001 à 009) et de la constitution **v1.8.0**, dont le
principe II autorise désormais l'**identité visuelle** (nom, couleurs, logo) sans
session.

## R1. Stockage des paramètres

**Decision** : la table singleton `association` (00001) reçoit, par une
migration `00010_association_settings.sql`, une colonne par champ : `tagline`,
`short_name`, `purpose`, `rna`, `siret`, `founded_on`, `address_line1`,
`address_line2`, `postal_code`, `city`, `contact_email`, `phone`, `website`,
`facebook_url`, `instagram_url`, `linkedin_url`, `youtube_url`,
`mastodon_url`, `bluesky_url`, `primary_color`, `accent_color`. Toutes sont
facultatives (`NULL` = non renseigné) et bornées par des `CHECK` qui reprennent
les longueurs et formats de la spec (défense en profondeur). Le logo a sa
propre table singleton `association_logo` (R4).

**Rationale** : principe I (singleton, pas de locataire) ; des colonnes typées
sont lisibles par sqlc sans désérialisation, et la liste des réseaux sociaux
est fermée (spec, Assumptions). `ALTER TABLE … ADD COLUMN` est accepté sur une
table `STRICT` tant que la colonne est nullable sans défaut non constant.

**Alternatives considered** : table clé/valeur (`settings(key, value)`) : perd
le typage sqlc et les `CHECK` ; colonne JSON : idem, et validation dupliquée ;
table `association_social_links` : inutile pour six réseaux fixes.

## R2. Validation des champs

**Decision** :

- **Déclaratif dans huma** (principe VI) : longueurs maximales (`maxLength`),
  `format:"email"` pour l'adresse de contact, `pattern` pour les couleurs
  (`^#[0-9a-fA-F]{6}$`) et le code postal (`^[0-9]{5}$`), `format:"date"`
  pour la date de création.
- **Domaine pur** `internal/association` (`Normalize`, testé en table), pour
  ce que huma ne peut exprimer, appelé par le handler avant toute écriture :
  - suppression des espaces de début et de fin ; chaîne vide → `NULL`
    (FR-014) ; nom vide après suppression → erreur `name` (FR-004) ;
  - **RNA** : majuscule forcée, `^W[0-9]{9}$` (FR-007) ;
  - **SIRET** : espaces retirés, 14 chiffres, clé de Luhn ; exception connue
    de La Poste (SIREN `356000000`) : somme des chiffres multiple de 5
    (FR-008) ;
  - **date de création** ≤ date du jour à Paris (`season.Today`, horloge
    injectable) (FR-009) ;
  - **téléphone** : espaces, points, tirets retirés ; formes acceptées
    `0[1-9]` + 8 chiffres, `+33`/`0033` + 9 chiffres, et les indicatifs des
    départements d'outre-mer à numérotation à 10 chiffres (`+262`, `+269`,
    `+590`, `+594`, `+596`) ramenés à la forme nationale ; stocké sous la
    forme nationale à 10 chiffres, affiché par groupes de deux (`01 23 45 67
    89`) côté interface via `lib/format.ts` ;
  - **adresses web** (site et réseaux) : `net/url`, schéma `http` ou `https`,
    hôte non vide, 300 caractères au plus.
- Toutes les erreurs sont rassemblées puis renvoyées en un seul `422` avec
  `errors[].location = "body.<champ>"`, par `FieldErrorDetails` comme pour
  l'IBAN (`settings.go`) : tout-ou-rien (FR-013).

**Rationale** : le serveur fait autorité ; le frontend ne recopie aucune règle
et affiche les `422` sur les champs (adaptateur existant).

**Alternatives considered** : bibliothèque `libphonenumber` (port Go) :
dépendance lourde pour un seul pays ; validation côté client : interdite par
le principe VI.

## R3. Palette dérivée et contraste

**Decision** : un domaine pur `internal/brand` calcule, depuis la couleur
principale et la couleur d'accent (ou les valeurs par défaut de l'artefact :
`#1f5278` et `#c44822`), une **palette** :

| Jeton | Rôle | Règle |
| --- | --- | --- |
| `brand` / `accent` | fond (boutons, élément actif, bandeau d'e-mail) | couleur choisie, inchangée |
| `onBrand` / `onAccent` | texte posé sur ce fond | blanc si contraste ≥ 4,5:1, sinon `ink` (#181f29) si ≥ 4,5:1, sinon noir |
| `brandInk` / `accentInk` | texte, liens, bordures, contours, cases cochées et indicateurs dans cette teinte, sur `surface`, `canvas` et le fond teinté | couleur choisie, assombrie pas à pas (luminosité OKLCH −0,02) jusqu'à un contraste ≥ 4,5:1 avec le plus foncé des fonds clairs : `canvas` (#f5f6f8) et le **fond teinté** (couleur à 10 % sur `canvas`, utilisé par `bg-brand/10`) |

Les contrastes suivent la formule de luminance relative du WCAG 2.x (celle
qu'utilise le RGAA). Le test en table balaie un échantillon de couleurs (dont
jaunes pâles, gris moyens, noir, blanc) et vérifie chaque couple (SC-004). La
palette sert à l'interface (variables CSS) **et** aux e-mails (R6), d'où un
calcul unique en Go.

**Rationale** : FR-020 impose d'ajuster plutôt que de refuser. Assombrir dans
OKLCH conserve la teinte perçue. Toute couleur atteint ≥ 4,58:1 avec le blanc
ou le noir, d'où la garantie pour `on*`.

**Usages dans l'interface** : la couleur choisie (`brand`, `accent`) n'est
plus employée que comme **fond plein** (`bg-brand`, `bg-brand/90`), toujours
avec `on*` par-dessus. Tout usage en texte ou en repère non textuel passe par
`*-ink` : les 36 `text-brand` deviennent `text-brand-ink`, et `border-brand`,
`border-brand/40`, `outline-brand`, `accent-brand` deviennent leurs variantes
`brand-ink`. Un contraste ≥ 4,5:1 couvre aussi l'exigence de 3:1 du RGAA pour
les composants d'interface. Les fonds teintés (`bg-brand/10`, `bg-brand/5`)
restent dérivés de la couleur choisie, et `brandInk` est vérifié sur le plus
foncé d'entre eux.

**Alternatives considered** : calcul en TypeScript : règle dupliquée pour les
e-mails ; refus des couleurs insuffisantes : contraire à FR-020 ; bibliothèque
de couleurs : quelques dizaines de lignes suffisent.

## R4. Logo : vérification, normalisation, variantes

**Decision** :

- Dépôt multipart (`huma.MultipartFormFiles`, comme les pièces jointes de 009),
  corps borné à 1 Mio + 16 Kio (`MaxBodyBytes`). « 1 Mo » dans la spec
  désigne 1 048 576 octets. Un fichier plus gros est refusé par
  `logo.Process` (`ErrTooBig`, lecture de 1 048 577 octets au plus) avec un
  `422` sur `form.file` ; un corps au-delà de `MaxBodyBytes` reçoit un `413`
  que l'écran affiche aussi sur le champ, avec le même message.
- Type détecté sur le contenu (`image.DecodeConfig` avec décodeurs PNG, JPEG,
  WebP enregistrés) ; SVG, GIF et tout le reste refusés (FR-015). Dimensions
  bornées **avant décodage complet** : 16 mégapixels et 8 000 px par côté au
  plus (protection contre les bombes de décompression).
- L'image est **décodée puis réencodée** : ce qui est stocké n'est jamais le
  fichier reçu. Deux variantes sont produites :
  - `web` : côté le plus long ≤ 512 px, pour l'interface et les écrans de
    connexion ;
  - `email` : hauteur ≤ 96 px et largeur ≤ 400 px (affichage à 48 px de haut,
    écrans haute densité), poids ≤ 100 Ko (FR-028b) ; si l'encodage dépasse,
    réduction de 25 % et nouvel essai.
  - Format : JPEG (qualité 88) si la source est un JPEG, PNG sinon (WebP peut
    porter de la transparence, mal gérée par les messageries).
- Redimensionnement : `golang.org/x/image/draw` (`CatmullRom`). Décodage WebP :
  `golang.org/x/image/webp`.
- Les variantes sont écrites par le `communication.FileStore` existant
  (écriture atomique, nom aléatoire de 32 caractères hexadécimaux, empreinte
  SHA-256), instancié sur `FRIGG_DATA_DIR/logo/`. Les métadonnées sont dans
  `association_logo` (data-model). Un nouveau dépôt remplace la ligne dans la
  transaction ; les anciens fichiers sont supprimés **après** validation de la
  transaction (FR-016) ; un échec de suppression est journalisé et rattrapé au
  démarrage par un balayage des fichiers orphelins du répertoire `logo/`.

**Rationale** : le réencodage retire les métadonnées (EXIF, coordonnées GPS
d'une photo prise au téléphone) et neutralise les fichiers polyglottes ; une
variante légère accélère l'écran de connexion. Spec : « format vérifié d'après
le contenu ».

**Alternatives considered** : conserver le fichier d'origine : métadonnées
personnelles possibles, poids jusqu'à 1 Mo à chaque connexion ; refuser WebP
pour éviter la dépendance : contraire à la spec ; redimensionnement maison
« plus proche voisin » : rendu crénelé sur un logo.

**Dépendance ajoutée** : `golang.org/x/image` (dépôt officiel Go, Go pur,
sans CGO, même famille que `x/text`, `x/term`, `x/time` déjà admis).
Justifiée dans [plan.md › Complexity Tracking](./plan.md#complexity-tracking).

## R5. Identité visuelle publique et application des couleurs

**Decision** :

- Opération **publique** `get-identity` (`GET /api/identity`) : nom (ou « Mon
  association »), URL du logo `web` ou `null`, palette complète (R3). Rien
  d'autre (principe II v1.8.0) ; un test vérifie la liste exacte des clés de
  la réponse.
- Opération **publique** `get-identity-logo` (`GET /api/identity/logo/{name}`)
  : sert la variante `web` si `name` est le nom généré courant (sinon `404`),
  `Content-Type` détecté au dépôt, `X-Content-Type-Options: nosniff`,
  `Content-Disposition: inline`, `Cache-Control: public, max-age=31536000,
  immutable` (le nom change à chaque dépôt : pas de cache périmé). Le nom
  d'origine n'est jamais exposé.
- Les deux opérations entrent dans `allowedPublic` (`access_test.go`) avec la
  référence à l'amendement 1.8.0.
- Frontend : `root.tsx` précharge l'identité dans son `clientLoader`
  (`ensureQueryData`) avant le premier rendu ; un composant `BrandStyle` écrit
  les variables `--brand`, `--brand-ink`, `--on-brand`, `--accent`,
  `--accent-ink`, `--on-accent` (triplets RVB, format actuel de `app.css`) sur
  `document.documentElement`. Pendant le chargement, le `HydrateFallback`
  neutre existant s'affiche : aucun clignotement de l'identité de Frigg
  (spec, Edge Cases). En cas d'erreur, la requête renvoie l'identité par
  défaut et l'écran reste utilisable (FR-028).
- Les écrans hors session (`routes/auth/layout.tsx`) affichent logo ou
  initiales et nom depuis cette requête ; le commentaire « le nom n'est pas
  affiché hors connexion » est retiré.
- Après un enregistrement, les mutations invalident `identity` et `session` :
  l'administrateur voit le résultat immédiatement ; les autres au prochain
  chargement (FR-027).

**Rationale** : un seul canal pour l'apparence, dans la session comme hors
session ; les fichiers statiques restent sans donnée de l'association.

**Alternatives considered** : injecter la palette dans `index.html` côté
serveur : le SPA est servi tel quel depuis `embed`, et le principe II interdit
des données de l'association dans les fichiers statiques ; palette dans la
session seulement : rien pour l'écran de connexion.

## R6. Habillage des e-mails au moment de l'envoi

**Constat** : aujourd'hui `mail.Render` exécute le gabarit complet (layout
compris) au moment de la mise en file, et `outbound_emails` stocke le HTML
final. La spec exige que nom, logo et couleurs soient ceux **en vigueur à
l'envoi** (Edge Cases) et qu'un logo retiré ne laisse pas d'image cassée.

**Decision** :

- `mail.Render` ne rend plus que le **contenu** (objet, corps texte, corps
  HTML) ; le gabarit `layout` est appliqué par une nouvelle fonction
  `mail.Wrap(msg, Identity)` **dans le worker**, juste avant l'envoi.
  `Identity` = nom, palette, variante `email` du logo (octets) ; elle est lue
  une fois par passage du worker.
- `outbound_emails` reçoit deux colonnes (migration 00010) : `wrap INTEGER NOT
  NULL DEFAULT 0` (1 = layout à appliquer à l'envoi) et `footer TEXT` (mention
  de pied propre à la communication, déjà un champ de `mail.Data`). Les lignes
  existantes gardent `wrap = 0` et partent telles quelles : pas de migration
  de contenu.
- Le layout HTML utilise la palette : bandeau d'en-tête `brand` avec texte
  `onBrand`, logo à gauche du nom (`alt=""`, le nom restant écrit à côté :
  FR-028b et lisibilité sans images), boutons `brand`/`onBrand`, liens
  `brandInk`. Les couleurs codées en dur `#1f5278` des gabarits de contenu
  sont remplacées par des appels de gabarit (`{{template "button" …}}`)
  définis dans le layout.
- Le logo est joint en **image embarquée** (`go-mail` `EmbedReader`,
  `Content-ID: logo`), référencée par `cid:logo` ; sans logo, ni balise
  `<img>` ni pièce embarquée.
- L'aperçu d'une communication (009, `preview-communication`) appelle aussi
  `Wrap` avec l'identité courante ; le logo y est fourni en `data:` URI, l'aperçu
  étant affiché dans l'`iframe` sandbox existante.
- **Nom d'expéditeur** : pour toute ligne `wrap = 1`, le worker pose le nom en
  vigueur comme nom d'affichage de l'expéditeur (`FromFormat`), quel que soit
  le gabarit (spec US1 › scénario 5, FR-026). La colonne `from_name` de 009
  n'est plus lue pour ces lignes ; elle reste utilisée pour les lignes
  `wrap = 0` antérieures et n'est plus renseignée par les nouvelles mises en
  file.

**Rationale** : un renommage ou un changement de logo pendant qu'une
communication de 500 messages part (≈ 13 min) reste cohérent, sans conserver
d'anciens fichiers de logo pour les e-mails en attente. Le contenu, lui, reste
figé à la mise en file (il cite parfois le nom, ce qui est accepté).

**Alternatives considered** : figer l'habillage à la mise en file et garder
chaque ancien logo tant qu'un e-mail en attente le référence : balayage et
compteur de références pour un cas rare ; joindre le logo dans chaque ligne de
la file : jusqu'à 50 Mo en base pour une communication.

## R7. Opérations d'administration et formulaires

**Decision** :

- `get-general-settings` / `update-general-settings`
  (`GET|PUT /api/admin/settings/general`, `admin`) : tous les champs de la
  section « Général ».
- `update-appearance` (`PUT /api/admin/settings/appearance`, `admin`) :
  couleurs (`null` = valeur par défaut) ; « Rétablir les couleurs par défaut »
  envoie deux `null` après le dialogue de confirmation.
- `upload-logo` (`PUT /api/admin/settings/appearance/logo`, multipart) et
  `delete-logo` (`DELETE …/logo`, après dialogue de confirmation).
- `preview-palette` (`GET /api/admin/settings/appearance/palette?primary=&accent=`,
  `admin`) : palette dérivée pour l'aperçu, sans rien enregistrer ; appelée
  avec un délai de 150 ms après la dernière modification.
- Enregistrement de « Apparence » : si un nouveau logo est choisi, il est
  déposé d'abord ; en cas de refus, rien d'autre n'est envoyé et l'erreur
  s'affiche sur le champ ; sinon les couleurs sont enregistrées. L'aperçu du
  logo avant enregistrement utilise une URL d'objet locale.
- Formulaires `react-hook-form` ; erreurs `422` affichées par l'adaptateur
  existant ; modifications non enregistrées : `useBlocker` + `ConfirmDialog`,
  comme `communication-new.tsx` (FR-030).

**Rationale** : opérations courtes et typées ; la palette d'aperçu vient du
même code Go que la palette appliquée.

**Alternatives considered** : un seul multipart couleurs + logo : champs non
fichiers moins bien décrits dans l'OpenAPI ; palette calculée dans le
navigateur : règle dupliquée.

## R8. Données exposées aux membres

**Decision** :

- `get-session` : `association` passe de `{name}` à `{name, tagline,
  logoUrl}` (barre latérale, en-tête de l'accueil).
- `get-association-contact` (`GET /api/association/contact`, `member`) :
  adresse du siège, adresse de contact, téléphone, site, réseaux sociaux ;
  rien de l'identité légale. Lue par l'accueil membre ; encart masqué si tout
  est vide (FR-025).
- Liens externes : `target="_blank" rel="noopener noreferrer"` ; téléphone en
  `tel:+33…` ; adresse en `mailto:` ; chaque réseau a un libellé lisible
  (« Instagram de {nom} ») et l'icône générique `Share2` décorative (lucide-react 1.x n'a plus d'icônes de marque).

## R9. Journalisation

**Decision** : `slog` (français, sans données personnelles) : « paramètres
généraux modifiés » avec `account_id` et la liste des **noms** de champs
modifiés (jamais leurs valeurs, FR-031) ; « logo déposé » (`account_id`, type,
dimensions, poids des variantes) ; « logo retiré » ; « couleurs modifiées »
(les codes couleur ne sont pas des données personnelles et sont journalisés).

## R10. Tests

**Decision** :

- **Domaine** (tests en table) : `internal/association` (RNA, SIRET dont La
  Poste, téléphone, URL, date, normalisation) ; `internal/brand` (contraste de
  chaque couple sur un balayage de couleurs) ; `internal/association/logo`
  (PNG, JPEG, WebP acceptés ; SVG, GIF, PDF, PNG tronqué, fichier renommé,
  image de 20 000 × 1 px refusés ; EXIF absent de la sortie ; variante `email`
  ≤ 100 Ko).
- **Mail** : `Wrap` avec et sans logo (aucune balise `<img>` sans logo),
  couleurs dans le bandeau ; worker : ligne `wrap = 1` habillée avec
  l'identité courante, ligne `wrap = 0` inchangée.
- **API** (httptest) : `422` groupés, tout-ou-rien, autorisations (matrice
  automatique : 10 opérations ajoutées, 1 modifiée) ; `get-identity` anonyme renvoie
  exactement `name`, `logoUrl`, `palette` ; logo anonyme servi sur le nom
  courant, `404` sur l'ancien ; aucune autre donnée de l'association lisible
  sans session (SC-006).
- **Migration** : `00010` sur base vierge et sur base `00009` peuplée (e-mails
  en attente conservés, `wrap = 0`).
- **Playwright + axe** : `general-settings.spec.ts` (saisie, erreurs, barre
  latérale, accueil membre, encart contact), `appearance.spec.ts` (logo,
  couleur claire, réinitialisation, écran de connexion habillé, e-mail
  Mailpit avec bandeau et logo embarqué).
