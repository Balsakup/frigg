# Research: Envoi d'e-mails groupés aux adhérents

**Feature**: `009-bulk-email` | **Date**: 2026-11-03 | **Spec**: [spec.md](./spec.md)

Aucun point de la spécification ne restait à clarifier. Les recherches ci-dessous
fixent les choix techniques à partir du code existant (001 à 008) et de la
constitution v1.7.0.

## R1. Situation de paiement et sélection de la cible

**Decision** : la situation de paiement est calculée en SQL, dans une seule
requête sqlc `ListAudienceCandidates`, à partir des adhésions validées de la
saison :

| Situation (`situation`) | Condition sur `memberships` |
| --- | --- |
| `up_to_date` | `amount_cents = 0` ou `payment_status = 'paid'` |
| `announced` | `payment_status = 'announced'` |
| `due` | `payment_status = 'due'` |

Le paiement par carte non abouti laisse `payment_status = 'due'` (008) : il
tombe naturellement dans `due` (FR-005). *Implémentation* : la requête renvoie
toutes les adhésions validées de la saison et le filtre par situation et par
tarif est fait par `communication.Filter` (domaine pur, testé en table) ; un
tableau de tarifs vide signifie « tous les tarifs ». Quelques milliers de
lignes au plus : plus simple que deux variantes de requête avec
`sqlc.slice()`. La correspondance des trois cibles prédéfinies (`all`, `up_to_date`,
`unpaid`) vers des situations est faite par le domaine pur
`internal/communication` ; l'API reçoit toujours la liste explicite des
situations, le préréglage n'étant qu'une aide de l'interface.

**Rationale** : une seule source de vérité de la situation, testable en table ;
pas de recalcul côté client (principe VI).

**Alternatives considered** : calcul en Go après lecture de toutes les
adhésions de la saison (plus de données lues, logique dupliquée avec les
statistiques) ; colonne dérivée stockée (dénormalisation inutile).

## R2. Adresse de contact et dédoublonnage

**Decision** : la requête renvoie, pour chaque adhérent ciblé, `account_email`
(compte qui gère la fiche, `members.account_id`) et `member_email`. Le domaine
`communication.BuildAudience` retient `account_email` s'il existe, sinon
`member_email`, sinon classe l'adhérent « non joignable ». Le dédoublonnage se
fait sur `strings.ToLower(strings.TrimSpace(adresse))` (les adresses sont
validées ASCII à la saisie en 001 et 005 ; `textfold.Fold`, qui retire aussi
les accents, n'apporte rien ici et pourrait fusionner des adresses
internationalisées distinctes). La première graphie rencontrée, dans l'ordre
nom/prénom, est conservée. Un compte désactivé reste utilisé (cas limite de la
spec).

**Rationale** : un parent gérant trois enfants reçoit un seul message ; la règle
est pure et testée sans base.

**Alternatives considered** : `GROUP BY lower(email)` en SQL (perd la liste des
adhérents par destinataire nécessaire à l'aperçu) ; envoi aux deux adresses
(compte et fiche) — écarté par la spec.

## R3. Pièces jointes : dépôt immédiat lié à une préparation

**Decision** : chaque fichier est déposé dès son ajout par une opération
dédiée (`upload-communication-attachment`, multipart, un fichier par requête),
rattaché à une **clé de préparation** (`draftKey`, UUID v4 généré par le
navigateur à l'ouverture du formulaire) et à l'administrateur qui le dépose.
Les limites (FR-017a à FR-017c) sont vérifiées au dépôt sur l'ensemble des
pièces de la même préparation : nombre ≤ 5, fichier ≤ 5 Mo, total ≤ 10 Mo.
L'envoi référence la clé de préparation et emporte toutes ses pièces. Une
pièce peut être retirée (`delete-communication-attachment`) tant qu'elle n'est
pas rattachée à une communication envoyée.

huma v2.40 fournit `huma.MultipartFormFiles[T]` et `huma.FormFile` (nom, type
annoncé, taille, lecteur) ; `MaxBodyBytes` de l'opération est fixé à
5 Mo + 64 Kio (en-têtes multipart), au-delà huma renvoie `413` traduit en
message français. Le type annoncé par le navigateur n'est pas cru (R4).
`humago.MultipartMaxMemory` est porté à cette même taille : par défaut
(8 Kio), `net/http` écrit le fichier dans le répertoire temporaire, en lecture
seule en conteneur (spec 007).

**Rationale** : retour immédiat sur un fichier refusé (FR-017d), envoi léger
(JSON), et la même clé sert d'idempotence (R7).

**Alternatives considered** : un seul envoi multipart avec texte et fichiers
(erreurs de pièces connues seulement à l'envoi, 10 Mo renvoyés à chaque
tentative) ; stockage en base (BLOB) — contraire à la règle « Fichiers » de la
constitution.

## R4. Vérification du type sur le contenu

**Decision** : `communication.DetectType(r io.ReaderAt, size int64)` :

| Type accepté | Détection |
| --- | --- |
| PDF | `http.DetectContentType` → `application/pdf` (signature `%PDF-`) |
| JPEG, PNG | `http.DetectContentType` → `image/jpeg`, `image/png` |
| DOCX, XLSX | archive ZIP (`archive/zip`, lecture du répertoire seulement) contenant `[Content_Types].xml` dont le type principal est `…wordprocessingml.document.main+xml` ou `…spreadsheetml.sheet.main+xml` ; les variantes à macros (`macroEnabled`) sont refusées |
| ODT, ODS | archive ZIP dont la première entrée est `mimetype`, stockée sans compression, valant `application/vnd.oasis.opendocument.text` ou `…spreadsheet` |

Le type MIME enregistré et transmis est celui détecté, jamais celui annoncé.
Les anciens formats binaires Office (`.doc`, `.xls`, conteneur OLE) ne sont pas
acceptés : leur distinction exige d'analyser le conteneur et ils portent
souvent des macros ; l'interface invite à enregistrer en PDF, DOCX ou ODT.
La lecture de `[Content_Types].xml` est bornée à 64 Kio décompressés (pas de
décompression du reste de l'archive : pas d'exposition aux bombes ZIP).
L'extension du nom d'origine n'intervient pas dans la décision ; le nom est
seulement nettoyé (R5).

**Rationale** : FR-017b exige une vérification sur le contenu ; bibliothèque
standard uniquement (principe V, aucune dépendance ajoutée).

**Alternatives considered** : bibliothèque de détection MIME tierce
(`gabriel-vasile/mimetype`) — dépendance supplémentaire pour six types ;
confiance dans l'extension — contraire à la spec.

## R5. Stockage des fichiers et nom d'origine

**Decision** : fichiers sous `FRIGG_DATA_DIR/communications/<32 hex aléatoires>`,
sans extension, écrits dans un fichier temporaire du même répertoire puis
renommés (écriture atomique), permissions `0640`. La base conserve le nom
généré, le nom d'origine nettoyé, le type détecté, la taille et la somme
SHA-256 (contrôle d'intégrité à l'envoi et SC-006). Le nom d'origine est
réduit à son dernier élément (`filepath.Base` après remplacement des `\`),
débarrassé des caractères de contrôle, tronqué à 150 caractères en conservant
l'extension ; un nom vide devient « piece-jointe ». Le chemin d'un fichier est
toujours construit par `filepath.Join(dir, stored)` avec `stored` validé par
expression régulière `^[0-9a-f]{32}$` (pas de traversée, principe VII).

**Rationale** : règle « Fichiers » et principe VII de la constitution ; la
sauvegarde complète reste « base + répertoire de données ».

**Alternatives considered** : nom d'origine sur disque (collisions, traversée) ;
sous-répertoires par date (inutile à ce volume).

## R6. Nettoyage des préparations abandonnées

**Decision** : un balayage `CommunicationAttachmentSweeper`, lancé par
`frigg serve` toutes les 15 minutes, supprime les pièces non rattachées à une
communication déposées depuis plus de 24 heures (FR-017g) : suppression de la
ligne en transaction, puis du fichier ; un échec de suppression de fichier est
journalisé et retenté au passage suivant via une seconde passe qui supprime
les fichiers du répertoire `communications/` absents de la base et plus vieux
d'une heure (fichiers orphelins après un arrêt brutal).

**Rationale** : même motif que `OnlinePaymentSweeper` (008) ; tolérant aux
pannes sans transaction distribuée base/disque.

**Alternatives considered** : suppression au départ de la page (non fiable :
onglet fermé, coupure réseau) ; aucune purge (fichiers personnels conservés
indéfiniment, contraire au RGPD).

## R7. Envoi : transaction unique, idempotence et contrôle du décompte

**Decision** : `send-communication` reçoit `draftKey`, la cible, l'objet, le
texte et `expectedRecipients`. Dans une transaction `BEGIN IMMEDIATE` :

1. si une communication porte déjà ce `draftKey` (index unique), elle est
   renvoyée telle quelle avec `200` (double soumission, FR-014) ;
2. la cible est recalculée (R1, R2) ; si le nombre de destinataires diffère de
   `expectedRecipients`, `409` avec le nouveau décompte (FR-013) ; s'il est nul,
   `422` ;
3. les pièces de la préparation (même auteur, non rattachées) sont relues,
   leur nombre et leur total revérifiés, chaque fichier relu pour contrôler
   taille et SHA-256 ;
4. insertion de `communications`, rattachement des pièces, insertion de
   `communication_recipients` et d'un `outbound_emails` par destinataire.

L'API répond `201` dès la validation (FR-015) ; l'envoi réel est fait par le
worker existant. Pour 1 000 destinataires, la transaction insère 2 000 lignes,
soit quelques dizaines de millisecondes en WAL.

**Rationale** : aucune communication partielle ; idempotence sans état en
mémoire ; contrainte « e-mails mis en file dans la transaction de l'action ».

**Alternatives considered** : jeton d'idempotence séparé de la clé de
préparation (redondant) ; insertion des e-mails par lot hors transaction
(communication incomplète en cas d'arrêt).

## R8. File d'e-mails : pièces jointes, expéditeur et état par destinataire

**Decision** : nouvelle migration `00009_communications.sql` :

- `outbound_emails` gagne `communication_id` (référence facultative) et
  `from_name` (nom d'affichage de l'expéditeur, facultatif) ;
- le worker, pour un e-mail portant `communication_id`, lit la liste des
  pièces (`ListCommunicationAttachments`, mise en cache le temps d'un lot) et
  les joint via `go-mail` (`AttachReader` avec nom d'origine et type détecté) ;
  le fichier est ouvert par un `mail.AttachmentStore` injecté (racine
  `FRIGG_DATA_DIR/communications`). Fichier introuvable ou somme différente :
  échec définitif immédiat de cet e-mail, journalisé sans adresse ;
- `SMTPSender` utilise `FromFormat(from_name, MAIL_FROM)` quand `from_name`
  est renseigné (FR-016 : nom de l'association comme expéditeur) ; les autres
  e-mails sont inchangés ;
- l'état par destinataire est recopié par **déclencheurs SQL** :
  `AFTER UPDATE OF status ON outbound_emails` met à jour
  `communication_recipients.status` (et `sent_at`) de la ligne liée ;
  `AFTER DELETE ON outbound_emails WHEN old.status = 'pending'` la passe en
  `failed`. La purge des e-mails envoyés après 30 jours (`PurgeSentEmails`)
  ne fait donc pas perdre l'état, la référence passant à `NULL`.

Le corps rendu est stocké dans chaque ligne `outbound_emails` (≈ 2 × 10 Ko au
plus par destinataire, purgé après envoi + 30 jours) : le worker reste
générique et n'a pas à connaître le rendu des communications.

**Rationale** : le worker existant garde nouvelles tentatives, délais et états
(FR-015) ; les déclencheurs évitent de modifier chaque chemin d'écriture du
worker et restent cohérents même si `outbound_emails` est modifié ailleurs.

**Alternatives considered** : stockage du corps une seule fois et rendu par le
worker (couple le worker aux communications) ; mises à jour explicites dans le
worker (trois chemins à maintenir, oubli possible lors d'une suppression).

## R9. Rendu du message

**Decision** : gabarits `communication.txt.tmpl` / `communication.html.tmpl`.
Le texte saisi est inséré tel quel dans la version texte ; dans la version
HTML, il est échappé par `html/template`, les paragraphes (lignes vides) et
retours à la ligne sont convertis par une fonction `paragraphs` qui produit
`template.HTML` à partir de segments déjà échappés, et les URL `http(s)://`
sont rendues cliquables (`rel="noopener"`). Aucun autre balisage n'est
interprété (FR-017). Le layout gagne un champ `Footer` : pour une
communication, « Vous recevez ce message en tant qu'adhérent de {association}
pour la saison {saison}. » remplace « Merci de ne pas y répondre ». La même
fonction de rendu sert à l'aperçu (`preview-communication`), qui renvoie
l'objet, le texte et le HTML rendus ; l'interface affiche le HTML dans une
`iframe` `sandbox` sans aucun droit (`sandbox=""`, `srcdoc`).

**Rationale** : « l'aperçu tel qu'il sera reçu » (FR-011) sans dupliquer le
rendu côté client ; HTML produit par le serveur et isolé dans l'iframe.

**Alternatives considered** : aperçu reconstruit en React (divergence avec
l'e-mail réel) ; Markdown (mise en forme riche hors périmètre).

## R10. Téléchargement des pièces jointes

**Decision** : `download-communication-attachment` (`GET`, admin) renvoie le
fichier avec `Content-Type` détecté, `Content-Disposition: attachment;
filename*=UTF-8''…` (RFC 6266), `X-Content-Type-Options: nosniff` et
`Cache-Control: private, no-store`. Seules les pièces rattachées à une
communication sont téléchargeables (une pièce de préparation renvoie `404`).
L'interface utilise un lien `<a href download>` : requête de navigation de même
origine, cookie de session envoyé, contrôle d'accès du middleware huma
appliqué ; aucun `fetch` hors client `openapi-fetch`.

**Rationale** : FR-017f ; le navigateur gère l'enregistrement sans charger le
fichier en mémoire JavaScript.

**Alternatives considered** : `openapi-fetch` avec `parseAs: "blob"` puis URL
d'objet (plus de code, fichier entier en mémoire).

## R11. Interface

**Decision** :

- la rubrique existante « Communication » (`/admin/communication`, déjà dans
  `navigation.ts` et servie par `coming-soon`) devient la liste ; ajout de
  `/admin/communication/nouvelle` et `/admin/communication/:id` ;
- formulaire : `Select` de saison (courante par défaut, brouillons exclus),
  `RadioGroup` des cibles prédéfinies avec option « Personnalisée » qui révèle
  un groupe de cases (nouveau composant `ui/Checkbox.tsx` sur `Checkbox` et
  `CheckboxGroup` de base-ui) pour les situations et les tarifs ;
- le décompte (`get-communication-audience`) est relu à chaque changement de
  cible (TanStack Query, clé dépendant de la cible) et annoncé dans une région
  `status` polie ; les listes des adhérents ciblés et non joignables
  s'ouvrent dans un `Dialog` avec un tableau ;
- pièces jointes : `<input type="file" multiple>` natif habillé (accessible au
  clavier et aux lecteurs d'écran), un dépôt par fichier via
  `openapi-fetch` (`bodySerializer` `FormData`), liste avec nom, taille, état
  et bouton « Retirer » ; erreur affichée sur le fichier concerné ;
- envoi : `ConfirmDialog` récapitulant objet, cible, nombre de destinataires et
  pièces ; sur `409`, le dialogue affiche le nouveau décompte et redemande la
  confirmation.

**Rationale** : principe IV (shell unique, base-ui, tokens) ; aucune nouvelle
dépendance frontend.

**Alternatives considered** : zone de glisser-déposer seule (moins accessible ;
elle peut s'ajouter plus tard au-dessus du champ natif).

## R12. Journalisation et données personnelles

**Decision** : journaux `info` : « communication envoyée » (`communication_id`,
`author_id`, `recipients`, `attachments`), « pièce jointe déposée »
(`attachment_id`, `size`, `type`), « pièce jointe supprimée ». Jamais
d'adresse, d'objet, de texte ni de nom de fichier (FR-022, principe VII).
À la suppression d'un compte (002), `communication_recipients.email` des
lignes portant son adresse passe à `NULL` (affiché « Adresse supprimée »), à
côté du `DeleteEmailsByRecipient` existant, pour honorer l'effacement RGPD.

**Rationale** : FR-022 ; cohérence avec l'effacement déjà appliqué à la file
d'e-mails.

**Alternatives considered** : conserver l'adresse (contredit l'effacement
demandé lors d'une suppression de compte).

## R13. Tests

**Decision** :

- `internal/communication` : tests en table de `SituationsOf` (préréglages),
  `BuildAudience` (priorité compte/fiche, dédoublonnage, non joignables),
  `DetectType` (fichiers d'exemple minimaux générés en test : PDF, PNG, JPEG,
  DOCX, DOCM refusé, ODT, ZIP quelconque refusé, exécutable renommé refusé),
  `SanitizeFilename`, `Limits` ;
- `internal/mail` : rendu `communication` (échappement, liens, paragraphes),
  worker avec pièces jointes (faux `Sender` qui capture les pièces), fichier
  manquant → échec définitif, `from_name` ;
- `internal/api` : httptest pour dépôt multipart (types, tailles, 413,
  6ᵉ fichier, total dépassé, isolation entre administrateurs), audience
  (chaque situation, tarifs, saison brouillon refusée), envoi (409 sur
  décompte changé, 422 cible vide, double soumission → même communication,
  e-mails en file, pièces rattachées), téléchargement (en-têtes, pièce de
  préparation → 404), déclencheurs d'état, balayage à horloge injectée,
  suppression de compte → adresse effacée ; matrice d'autorisation étendue
  automatiquement (toutes les opérations sont `admin`) ;
- e2e Playwright + axe (`communications.spec.ts`) : relance des impayés avec
  une pièce jointe, réception vérifiée dans Mailpit (un exemplaire par
  adresse, pièce identique), historique et états.

**Rationale** : workflow de la constitution ; Mailpit est déjà utilisé par
les e2e.
