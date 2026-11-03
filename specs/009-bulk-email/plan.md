# Implementation Plan: Envoi d'e-mails groupés aux adhérents

**Branch**: `009-bulk-email` | **Date**: 2026-11-03 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/009-bulk-email/spec.md`

## Summary

Remplacer la rubrique d'administration « Communication » (aujourd'hui « bientôt
disponible ») par l'envoi d'e-mails groupés aux adhérents d'une saison. La
cible combine une saison, des situations de paiement (`up_to_date`,
`announced`, `due`, calculées en SQL depuis `memberships`) et, au besoin, des
tarifs ; trois préréglages (« Tous les adhérents », « À jour », « En impayé »)
couvrent les cas courants. L'adresse de contact est celle du compte qui gère
la fiche, à défaut celle de la fiche ; les destinataires sont dédoublonnés
par adresse. Les pièces jointes sont déposées une à une dès leur ajout,
vérifiées sur leur contenu (bibliothèque standard), stockées dans le
répertoire de données et rattachées à une clé de préparation qui sert aussi
d'idempotence. L'envoi, en une transaction, enregistre la communication et
met un e-mail par destinataire dans la file existante ; le worker joint les
fichiers et des déclencheurs SQL recopient l'état de chaque envoi dans
l'historique. Détails : [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 5.9 + React 19 (frontend)

**Primary Dependencies**: aucune nouvelle. Backend : huma/v2
(`MultipartFormFiles`, `FormFile`), sqlc (`sqlc.slice`), goose, `go-mail`
(`AttachReader`, `FromFormat`, déjà présent), `archive/zip`,
`net/http.DetectContentType`, `crypto/sha256`, `crypto/rand`.
Frontend : base-ui (`Checkbox`, `CheckboxGroup`, `RadioGroup`, `Select`,
`Dialog`), TanStack Query, `openapi-fetch` (`bodySerializer` FormData),
react-hook-form, lucide-react (`Mail`, `Send`, `Paperclip`, `FileText`,
`FileSpreadsheet`, `Image`, `X`).

**Storage**: SQLite ; nouvelle migration `00009_communications.sql` : tables
`communications`, `communication_attachments`, `communication_recipients`,
colonnes `communication_id` et `from_name` sur `outbound_emails`, deux
déclencheurs ([data-model.md](./data-model.md)). Fichiers dans
`FRIGG_DATA_DIR/communications/` sous un nom aléatoire.

**Testing**: `go test` : tests en table de `internal/communication`
(préréglages, audience, détection de type, noms de fichiers) ;
`internal/mail` (rendu, worker avec pièces et `from_name`) ; tests d'API
httptest (multipart, audience, envoi, idempotence, `409`, déclencheurs,
balayage à horloge injectée, téléchargement, suppression de compte) ; matrice
d'autorisation étendue automatiquement ; test de migration sur base `00008`
peuplée. Playwright + axe (`communications.spec.ts`) avec Mailpit
([R13](./research.md#r13-tests)).

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` ;
navigateurs récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: décompte de la cible < 2 s pour 1 000 adhérents (une
requête indexée, SC-002) ; envoi accepté immédiatement (transaction unique,
quelques dizaines de ms pour 1 000 destinataires, SC-004) ; 500 e-mails
partis en ≈ 13 min avec la cadence par défaut du worker (20 toutes les 30 s)

**Constraints**: aucun e-mail envoyé dans la requête HTTP (file d'attente) ;
type de fichier jamais cru sur le nom ni l'en-tête ; aucun contenu saisi
interprété comme du HTML ; aucune adresse, objet, texte ni nom de fichier dans
les journaux ; chemins de fichiers construits uniquement depuis un nom
généré validé ; commentaires sqlc en ASCII ; `MaxBodyBytes` du dépôt borné à
5 Mio + 64 Kio

**Scale/Scope**: une association, quelques centaines à quelques milliers
d'adhérents par saison ; 8 opérations ajoutées, 1 modifiée
(`delete-account`) ; 3 écrans nouveaux (dont un remplace `coming-soon`), un
composant `ui` ajouté

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution v1.7.0.

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Aucune notion de locataire ; nom de l'association lu dans le singleton existant pour l'expéditeur et le gabarit | ✅ |
| II. Authentification partout | 8 opérations, toutes `admin` ; aucune route publique ; téléchargement des pièces derrière le middleware huma (lien de même origine avec cookie de session), fichiers jamais servis comme statiques | ✅ |
| III. Deux rôles | Niveaux déclarés par `register` ; aucun test de rôle dans les handlers ; matrice d'autorisation automatique | ✅ |
| IV. Un shell, un design system | Rubrique déjà présente dans la navigation ; composants `ui/*`, nouveau `ui/Checkbox` sur base-ui ; tokens `ok`/`info`/`warn`/`danger` ; icônes via `Icon` ; envoi derrière `ConfirmDialog` ; retrait d'une pièce de préparation sans confirmation (action réversible, pas destructive pour des données enregistrées) ; RGAA : région `status` pour le décompte, champ fichier natif, `iframe` d'aperçu titrée ; 320 px | ✅ |
| V. Binaire unique en Go pur | Bibliothèque standard pour la détection de type et les archives ; `go-mail` déjà présent ; migration embarquée ; seules dépendances d'exécution : SQLite, répertoire de données, SMTP | ✅ |
| VI. API typée et SQL généré | Énumérations et bornes déclarées dans huma (situations, UUID, longueurs, `MaxBodyBytes`) ; requêtes sqlc avec `sqlc.slice` (pas de SQL concaténé) ; le frontend ne recalcule ni la situation ni le décompte ; pas de nouvelle recherche textuelle (pas de FTS5 requise) ; comparaison d'adresses en minuscules ASCII justifiée en [R2](./research.md#r2-adresse-de-contact-et-dédoublonnage) | ✅ |
| VII. Sécurité et RGPD | Fichiers : nom généré, type vérifié sur le contenu, taille bornée, chemin résolu dans le répertoire de données ; formats à macros refusés ; téléchargement `attachment` + `nosniff` ; texte échappé, aperçu en `iframe` sandbox ; purge des préparations abandonnées (24 h) ; adresse effacée de l'historique à la suppression d'un compte ; journaux sans données personnelles ([R12](./research.md#r12-journalisation-et-données-personnelles)) ; CSRF : opérations d'écriture sous `CrossOriginProtection` existant (multipart compris) | ✅ |
| Contraintes techniques | E-mails insérés dans la transaction de l'envoi, dépilés par le worker existant (nouvelles tentatives, états) ; fichiers sur disque, métadonnées en base ; balayage interne au binaire ; écriture en `BEGIN IMMEDIATE` | ✅ |
| Conventions de langue | Identifiants en anglais (`communications`, `up_to_date`, `draftKey`, `send-communication`) ; textes, messages d'erreur et journaux en français | ✅ |
| Workflow | Nouvelle migration (append-only) ; matrice d'autorisation étendue ; contrôles de [quickstart.md](./quickstart.md) | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance :

1. **Le type enregistré est toujours celui détecté** par
   `communication.DetectType` ; aucun chemin n'écrit `content_type` depuis
   l'en-tête multipart (revue + test d'un exécutable renommé).
2. **Aucun chemin de fichier n'est construit depuis une saisie** :
   `stored_name` est généré et revalidé (`^[0-9a-f]{32}$`) avant toute
   ouverture, au dépôt, à l'envoi, au téléchargement et dans le worker.
3. **Idempotence** : l'index unique sur `communications.draft_key` et la
   relecture dans `BEGIN IMMEDIATE` garantissent un seul envoi même si deux
   requêtes arrivent ensemble.
4. **L'état d'envoi survit à la purge de la file** grâce aux déclencheurs et à
   `ON DELETE SET NULL` ; un test purge la file puis relit l'historique.
5. **Le contenu saisi n'est jamais interprété** : texte inséré via
   `text/template` et `html/template` ; la conversion des retours à la ligne et
   des liens opère sur des segments déjà échappés (test avec `<script>`,
   `javascript:` et attributs).

## Project Structure

### Documentation (this feature)

```text
specs/009-bulk-email/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── api.md
│   └── ui-routes.md
├── checklists/
│   └── requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
internal/
├── communication/                      # nouveau : domaine pur, sans base
│   ├── target.go                       # Situation, Preset, SituationsOf, PresetOf, TargetLabel
│   ├── audience.go                     # Candidate, BuildAudience (adresse de contact, dédoublonnage)
│   ├── attachment.go                   # Limits, DetectType, SanitizeFilename, StoredName
│   ├── store.go                        # FileStore : écriture atomique, ouverture, suppression, orphelins
│   └── *_test.go
├── database/
│   ├── migrations/00009_communications.sql     # nouvelle
│   ├── queries/communications.sql      # audience, communications, pièces, destinataires
│   ├── queries/emails.sql              # EnqueueCommunicationEmail (communication_id, from_name)
│   └── sqlc/                           # régénéré
├── mail/
│   ├── render.go                       # Data.Footer, fonction paragraphs (texte → HTML échappé)
│   ├── smtp.go                         # pièces jointes, FromFormat
│   ├── worker.go                       # lecture des pièces par communication, AttachmentStore
│   └── templates/communication.{txt,html}.tmpl, layout.* (Footer)
├── api/
│   ├── communications.go               # audience, preview, send, list, get
│   ├── communication_attachments.go    # upload, delete, download
│   ├── communication_sweeper.go        # CommunicationAttachmentSweeper
│   ├── accounts.go                     # delete-account : effacement de l'adresse
│   ├── api.go                          # registerCommunications, Deps.Files
│   └── communications*_test.go
└── cli/serve.go                        # FileStore, worker avec pièces, lancement du balayage

web/
├── app/
│   ├── routes.ts                       # « communication » retirée de adminComingSoon
│   ├── routes/app/admin/communications-layout.tsx  # nouveau
│   ├── routes/app/admin/communications.tsx         # nouveau (historique)
│   ├── routes/app/admin/communication-new.tsx      # nouveau
│   ├── routes/app/admin/communication-detail.tsx   # nouveau
│   ├── communications/                 # AudienceSummary, AttachmentsField, MessagePreview, DeliveryBadge
│   ├── ui/Checkbox.tsx                 # nouveau
│   ├── api/communications.ts           # requêtes et mutations
│   ├── api/openapi.json, api/schema.d.ts   # régénérés
│   └── lib/format.ts                   # formatBytes (« 1,2 Mo »)
└── e2e/communications.spec.ts          # nouveau

README.md / documentation d'utilisation # rubrique Communication, limites des pièces jointes
```

**Structure Decision**: structure existante (API Go + SPA dans `web/`). Un
paquet nouveau, `internal/communication`, porte les règles pures (cible,
audience, fichiers), testables sans base, comme `internal/membership` ;
l'orchestration (transactions, file d'e-mails, journaux) reste dans
`internal/api`, et le worker de `internal/mail` reste générique (il joint les
pièces d'une communication sans connaître ses règles).

## Complexity Tracking

Aucune violation de la constitution v1.7.0. Écarts de complexité retenus :

| Élément | Pourquoi | Alternative plus simple écartée |
| --- | --- | --- |
| Dépôt des pièces avant l'envoi, avec clé de préparation et balayage | Refus immédiat d'un fichier (FR-017d) et idempotence de l'envoi (FR-014) avec une seule clé ; purge des abandons (FR-017g) | Un seul envoi multipart : erreurs de fichier connues seulement à l'envoi, 10 Mo renvoyés à chaque `409` |
| Déclencheurs SQL recopiant l'état des e-mails | L'état par destinataire doit survivre à la purge de la file (30 jours) sans modifier chaque chemin du worker | Lecture directe de `outbound_emails` : état perdu après purge ; mises à jour dans le worker : trois chemins à maintenir |
| Retrait d'une pièce de préparation sans dialogue de confirmation (interprétation du principe IV) | Le fichier n'est pas encore une donnée de l'association : il n'est rattaché à aucune communication, se redépose en un clic et serait purgé sous 24 h ; une confirmation à chaque retrait alourdirait la préparation. Les pièces d'une communication envoyée ne sont jamais supprimables | Dialogue de confirmation à chaque retrait : friction sans protection réelle d'une donnée enregistrée |
| Détection de type maison (ZIP inspecté) | Vérification sur le contenu exigée, sans dépendance nouvelle (liste fermée) | Bibliothèque `mimetype` : dépendance pour six types |
