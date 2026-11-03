# Implementation Plan: Paramètres généraux de l'association

**Branch**: `010-general-settings` | **Date**: 2026-11-03 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/010-general-settings/spec.md`

## Summary

Ajouter à « Paramètres » deux sections : **Général** (nom, tagline, nom court,
objet, identité légale, coordonnées, réseaux sociaux) et **Apparence** (logo,
couleur principale, couleur d'accent). Les champs sont des colonnes du
singleton `association`, normalisés et vérifiés par un domaine pur
`internal/association` (RNA, SIRET, téléphone, URL) et renvoyés en un seul
`422` groupé. Une palette (`brand`, `brandInk`, `onBrand`, et leurs
équivalents d'accent) est dérivée en Go par `internal/brand` avec une garantie
de contraste ≥ 4,5:1. Elle sert à l'interface et aux e-mails. Le logo est
vérifié sur son contenu, décodé puis réencodé en deux variantes (interface ≤
512 px, e-mail ≤ 100 Ko), sans métadonnées. Comme le permet l'amendement 1.8.0
du principe II, une opération publique `get-identity` expose seulement le nom,
le logo et la palette. Le `root` du SPA la précharge pour habiller toute
l'application, écrans de connexion compris, sans clignotement. Les e-mails
sont désormais habillés **au moment de l'envoi** par le worker (layout
appliqué avec l'identité courante, logo embarqué `cid:logo`), et non plus à
la mise en file. Détails : [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 6.0 + React 19 (frontend)

**Primary Dependencies**: **une nouvelle** : `golang.org/x/image` (`webp`,
`draw`), justifiée ci-dessous. Existantes : huma/v2
(`MultipartFormFiles`, `FormFile`, `pattern`, `format`), sqlc, goose,
`go-mail` (`EmbedReader`), `image/png`, `image/jpeg`, `net/url`,
`communication.FileStore`. Frontend : base-ui (`Dialog` via `ConfirmDialog`),
TanStack Query, `openapi-fetch` (`bodySerializer` FormData), react-hook-form,
`useBlocker` de React Router, lucide-react (`Palette`, `Image`, `Building2`,
`Mail`, `Phone`, `Globe`, `MapPin`, `Share2` ; lucide-react 1.x ne fournit
plus d'icônes de marque, les réseaux sociaux sont donc désignés par leur nom
en toutes lettres avec une icône générique, principe IV).

**Storage**: SQLite ; migration `00010_association_settings.sql` : colonnes
ajoutées à `association` et à `outbound_emails` (`wrap`, `footer`), nouvelle
table `association_logo` ([data-model.md](./data-model.md)). Variantes du logo
dans `FRIGG_DATA_DIR/logo/` sous un nom aléatoire.

**Testing**: `go test` : tests en table de `internal/association`,
`internal/brand`, `internal/association/logo` ; `internal/mail` (`Wrap`,
worker) ; tests d'API httptest (validation groupée, multipart, opérations
publiques, clés exactes de `Identity`, `404` sur un ancien logo) ; matrice
d'autorisation étendue automatiquement ; migration sur base `00009` peuplée.
Playwright + axe (`general-settings.spec.ts`, `appearance.spec.ts`) avec Mailpit
([R10](./research.md#r10-tests)).

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0` ;
navigateurs récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: pas d'objectif chiffré propre à cette fonctionnalité ;
`get-identity` lit une seule ligne, la variante de logo de l'écran de connexion
(≤ 512 px) est mise en cache de façon permanente (nom immuable) et l'aperçu de
palette est demandé 150 ms après la dernière modification

**Constraints**: aucune donnée de l'association hors identité visuelle sans
session ; aucun e-mail envoyé dans la requête HTTP ; type d'image jamais cru
sur le nom ni l'en-tête ; fichier reçu jamais stocké tel quel (réencodage) ;
bornes de pixels vérifiées avant décodage complet ; aucune valeur de
coordonnées dans les journaux ; commentaires sqlc en ASCII ; `MaxBodyBytes`
du dépôt de logo à 1 Mio + 16 Kio

**Scale/Scope**: une association ; 10 opérations ajoutées et 1 modifiée
(`get-session`) ; 2 écrans nouveaux ; 4 écrans ou composants modifiés (barre
latérale, accueil membre, gabarit hors session, `root`) ; 9 gabarits d'e-mail
adaptés au layout appliqué à l'envoi

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution v1.8.0.

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Colonnes ajoutées au singleton `association` ; `association_logo` contrainte à `id = 1` ; aucune notion de locataire | ✅ |
| II. Authentification partout | Deux opérations publiques, `get-identity` et `get-identity-logo`, exactement dans le périmètre de l'amendement 1.8.0 (nom, couleurs, fichier du logo ; nom d'origine et métadonnées jamais exposés), ajoutées à `allowedPublic` ; un test fige les clés de `Identity`. Coordonnées : `member` ; paramètres : `admin`. Fichiers statiques du SPA toujours sans donnée de l'association (identité chargée à l'exécution) | ✅ |
| III. Deux rôles | Niveaux déclarés par `register` ; aucun test de rôle dans les handlers ; matrice d'autorisation automatique | ✅ |
| IV. Un shell, un design system | Couleurs appliquées par variables CSS (`--brand`, `--brand-ink`, `--on-brand`, `--accent`, `--accent-ink`, `--on-accent`), sans bibliothèque de thème ; contraste ≥ 4,5:1 garanti par `internal/brand` ; couleurs sémantiques fixes ; composants `ui/*` ; `ConfirmDialog` pour retirer le logo et rétablir les couleurs ; RGAA : `fieldset`/`legend`, erreurs reliées, logo `alt=""` à côté du nom, région `status` dans l'aperçu ; thème clair ; 320 px | ✅ |
| V. Binaire unique en Go pur | `golang.org/x/image` est en Go pur (vérifié par `CGO_ENABLED=0 go list -deps`) ; migration embarquée ; dépendances d'exécution inchangées (SQLite, répertoire de données, SMTP) | ✅ (dépendance justifiée) |
| VI. API typée et SQL généré | Bornes, `format` et `pattern` déclarés dans huma ; règles non exprimables (Luhn, téléphone, date ≤ aujourd'hui) dans un domaine Go unique, erreurs `422` par champ ; palette d'aperçu calculée par le serveur (pas de règle dupliquée dans le frontend) ; requêtes sqlc ; pas de nouvelle recherche textuelle | ✅ |
| VII. Sécurité et RGPD | Logo : type vérifié sur le contenu, bornes de taille et de pixels, réencodage (EXIF et contenu polyglotte retirés), nom généré revalidé, chemin résolu dans le répertoire de données, `nosniff` ; SVG refusé ; journaux sans valeurs de coordonnées ; CSRF : écritures sous `CrossOriginProtection` existant (multipart compris) | ✅ |
| Contraintes techniques | E-mails toujours insérés en file dans la transaction, habillés par le worker existant ; fichiers sur disque, métadonnées en base ; écritures en `BEGIN IMMEDIATE` | ✅ |
| Conventions de langue | Identifiants en anglais (`tagline`, `short_name`, `get-identity`, `brandInk`) ; textes, erreurs et journaux en français | ✅ |
| Workflow | Nouvelle migration (append-only) ; matrice d'autorisation étendue ; contrôles de [quickstart.md](./quickstart.md) | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance :

1. **Périmètre public figé** : `Identity` n'a que `name`, `logoUrl`,
   `palette` ; le test compare la liste des clés JSON, pour qu'un champ ajouté
   par mégarde (tagline…) fasse échouer la CI.
2. **Le logo servi publiquement est toujours une variante réencodée** :
   aucun chemin n'écrit sur disque les octets reçus ; `get-identity-logo`
   refuse tout nom qui n'est pas le `web_name` courant (pas d'accès à la
   variante e-mail ni aux anciens fichiers).
3. **Bombes de décompression** : `image.DecodeConfig` avant `image.Decode`,
   avec refus au-delà de 16 Mpx ou 8 000 px par côté ; test avec un PNG de
   20 000 × 1 px.
4. **Couleur choisie jamais utilisée en texte ni en repère** : après la
   migration des classes, `grep -rE "(text|border|outline|accent)-brand([^-]|$)"`
   sur `web/app` ne renvoie que des variantes `-brand-ink`.
5. **Habillage à l'envoi** : les lignes `wrap = 0` (antérieures) partent
   telles quelles ; les lignes `wrap = 1` sont habillées avec l'identité lue
   au début du passage du worker. Test : changer le nom entre la mise en file
   et le passage du worker.
6. **Aucun clignotement** : l'identité est chargée par le `clientLoader` de
   `root` avant le premier rendu ; en cas d'échec, l'identité par défaut est
   utilisée sans bloquer la connexion.

## Project Structure

### Documentation (this feature)

```text
specs/010-general-settings/
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
├── association/                        # nouveau : domaine pur, sans base
│   ├── settings.go                     # Settings, Normalize, FieldError
│   ├── identifiers.go                  # RNA, SIRET (Luhn, La Poste), téléphone, URL
│   ├── logo/process.go                 # Process : détection, bornes, réencodage, variantes
│   └── *_test.go (+ testdata/ : images de test)
├── brand/                              # nouveau : domaine pur
│   ├── color.go                        # RGB, ParseHex, luminance, contraste, OKLCH
│   ├── palette.go                      # Palette, Derive, Default
│   └── palette_test.go
├── database/
│   ├── migrations/00010_association_settings.sql   # nouvelle
│   ├── queries/association.sql         # étendue : settings, couleurs, logo
│   ├── queries/emails.sql              # wrap, footer
│   └── sqlc/                           # régénéré
├── mail/
│   ├── render.go                       # Render : contenu seul ; Wrap(msg, Identity) ; Identity
│   ├── smtp.go                         # logo embarqué (EmbedReader, cid:logo)
│   ├── worker.go                       # lecture de l'identité par passage, Wrap des lignes wrap = 1
│   ├── outbox.go                       # Enqueue : wrap = 1, footer
│   └── templates/layout.*.tmpl, gabarits de contenu (boutons via le layout)
├── api/
│   ├── general_settings.go             # get/update-general-settings
│   ├── appearance.go                   # get/update-appearance, preview-palette, upload/delete-logo
│   ├── identity.go                     # get-identity, get-identity-logo (publiques), get-association-contact
│   ├── me.go, types.go                 # Session.Association étendue
│   ├── mail.go                         # enqueueMail/enqueueNotice : contenu seul
│   ├── communications.go               # aperçu habillé avec l'identité courante
│   ├── api.go                          # registerIdentity, registerSettings étendu, Deps.LogoFiles
│   ├── access_test.go                  # allowedPublic : get-identity, get-identity-logo
│   └── *_test.go
├── cli/serve.go                        # FileStore logo/, balayage des orphelins au démarrage, worker avec identité
├── cli/createadmin.go, tools/e2eseed   # GetAssociation étendue
└── (go.mod)                            # golang.org/x/image

web/
├── app/
│   ├── root.tsx                        # préchargement de identity, BrandStyle
│   ├── shell/BrandStyle.tsx            # nouveau
│   ├── shell/Sidebar.tsx               # logo, tagline
│   ├── routes.ts                       # parametres/general, parametres/apparence
│   ├── routes/app/admin/SettingsPage.tsx        # sections Général, Apparence, Paiements
│   ├── routes/app/admin/settings-index.tsx      # redirection vers general
│   ├── routes/app/admin/settings-general.tsx    # nouveau
│   ├── routes/app/admin/settings-appearance.tsx # nouveau
│   ├── settings/                       # AppearancePreview, ColorField, LogoField, ContactCard
│   ├── routes/app/member/home.tsx      # tagline, encart ContactCard
│   ├── routes/auth/layout.tsx          # identité de l'association
│   ├── api/identity.ts, api/settings.ts         # requêtes et mutations
│   ├── api/openapi.json, api/schema.d.ts        # régénérés
│   └── lib/format.ts                   # formatPhone (« 01 23 45 67 89 »)
└── e2e/general-settings.spec.ts, e2e/appearance.spec.ts  # nouveaux

README.md / documentation d'utilisation # rubrique Paramètres › Général et Apparence
```

**Structure Decision**: structure existante (API Go + SPA dans `web/`). Deux
paquets purs nouveaux, comme `internal/membership` et `internal/communication` :
`internal/association` (règles de saisie et traitement du logo) et
`internal/brand` (couleurs et contraste, partagé par l'API et le mail). Le
paquet `internal/mail` reste générique : il reçoit une `Identity` déjà
construite et ne lit pas la table `association` lui-même (le worker reçoit une
fonction de lecture injectée par `cli/serve.go`).

## Complexity Tracking

Aucune violation de la constitution v1.8.0. Écarts de complexité retenus :

| Élément | Pourquoi | Alternative plus simple écartée |
| --- | --- | --- |
| Dépendance `golang.org/x/image` (`webp`, `draw`) | La spec accepte le WebP (pas de décodeur dans la bibliothèque standard) ; un redimensionnement de qualité est nécessaire pour les deux variantes. Dépôt officiel Go, Go pur, même famille que `x/text`, `x/term` et `x/time` | Refuser le WebP et redimensionner « au plus proche voisin » : contraire à la spec, logo crénelé |
| Habillage des e-mails au moment de l'envoi (colonnes `wrap`, `footer`) | Nom, couleurs et logo en vigueur à l'envoi, sans image cassée après un retrait de logo ni ancien fichier à conserver (spec, Edge Cases) | Habillage figé à la mise en file : il faudrait garder chaque ancien logo tant qu'un e-mail en attente le référence |
| Réencodage du logo en deux variantes | Retire les métadonnées personnelles (EXIF/GPS) et les contenus polyglottes, allège l'écran de connexion, respecte la limite de 100 Ko par e-mail | Servir le fichier reçu : jusqu'à 1 Mo à chaque affichage, métadonnées exposées publiquement |
| Remplacement du logo sans dialogue de confirmation (interprétation du principe IV) | Choisir un fichier puis « Enregistrer » est une intention explicite, et l'aperçu montre le nouveau logo avant l'enregistrement ; un dialogue de plus alourdirait un geste rare sans protéger de donnée de l'association (le logo se redépose en un clic). Le **retrait** du logo, lui, passe par un dialogue | Dialogue « Remplacer le logo actuel ? » à chaque enregistrement d'un nouveau logo |
| Opération `preview-palette` | L'aperçu doit montrer la palette ajustée réelle, sans recopier dans le navigateur la règle de contraste | Calcul en TypeScript : règle dupliquée, risque d'écart avec les e-mails |
