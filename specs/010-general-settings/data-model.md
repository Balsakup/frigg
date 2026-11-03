# Data Model: Paramètres généraux de l'association

**Feature**: `010-general-settings` | **Migration**: `00010_association_settings.sql`

Les commentaires SQL de la migration et des requêtes sqlc restent en ASCII
(contrainte sqlc connue).

## `association` (existante, singleton `id = 1`) : colonnes ajoutées

Toutes nullable ; `NULL` = non renseigné. Le handler écrit des valeurs déjà
normalisées ([research.md › R2](./research.md#r2-validation-des-champs)) ; les
`CHECK` sont une défense en profondeur.

| Colonne | Type | Contrainte | Spec |
| --- | --- | --- | --- |
| `name` (existante) | TEXT | inchangée ; l'API refuse vide, `length ≤ 120` | FR-004 |
| `tagline` | TEXT | `length(tagline) BETWEEN 1 AND 150` | FR-005 |
| `short_name` | TEXT | `length BETWEEN 1 AND 20` | FR-005 |
| `purpose` | TEXT | `length BETWEEN 1 AND 2000` | FR-006 |
| `rna` | TEXT | `rna GLOB 'W[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]'` | FR-007 |
| `siret` | TEXT | `length = 14 AND siret NOT GLOB '*[^0-9]*'` (clé vérifiée en Go) | FR-008 |
| `founded_on` | TEXT | date ISO `AAAA-MM-JJ` (`date(founded_on) = founded_on`) | FR-009 |
| `address_line1`, `address_line2`, `city` | TEXT | `length BETWEEN 1 AND 100` | FR-010 |
| `postal_code` | TEXT | 5 chiffres (`GLOB`) | FR-010 |
| `contact_email` | TEXT | `length ≤ 254` | FR-011 |
| `phone` | TEXT | 10 chiffres commençant par `0` (forme nationale) | FR-011 |
| `website` | TEXT | `length ≤ 300`, commence par `http://` ou `https://` | FR-011 |
| `facebook_url`, `instagram_url`, `linkedin_url`, `youtube_url`, `mastodon_url`, `bluesky_url` | TEXT | idem `website` | FR-012 |
| `primary_color`, `accent_color` | TEXT | `#` + 6 hexadécimaux en minuscules ; `NULL` = couleur par défaut | FR-018, FR-021 |
| `updated_at` (existante) | TEXT | mise à jour à chaque enregistrement | — |

Le nom reste nullable pour les instances où il n'a jamais été défini ;
l'affichage retombe sur « Mon association » (Edge Cases).

## `association_logo` (nouvelle, singleton)

Au plus une ligne ; absence de ligne = pas de logo.

| Colonne | Type | Contrainte |
| --- | --- | --- |
| `id` | INTEGER PK | `CHECK (id = 1)` |
| `original_name` | TEXT NOT NULL | nom d'origine assaini (`communication.SanitizeFilename`), jamais exposé publiquement |
| `source_type` | TEXT NOT NULL | `image/png`, `image/jpeg` ou `image/webp` (détecté) |
| `source_size_bytes` | INTEGER NOT NULL | `≤ 1048576` |
| `web_name` | TEXT NOT NULL UNIQUE | nom généré (`^[0-9a-f]{32}$`) de la variante interface |
| `web_type` | TEXT NOT NULL | `image/png` ou `image/jpeg` |
| `web_size_bytes`, `web_width`, `web_height` | INTEGER NOT NULL | côté le plus long ≤ 512 |
| `web_sha256` | TEXT NOT NULL | empreinte vérifiée à la lecture |
| `email_name` | TEXT NOT NULL UNIQUE | nom généré de la variante e-mail |
| `email_type` | TEXT NOT NULL | `image/png` ou `image/jpeg` |
| `email_size_bytes` | INTEGER NOT NULL | `≤ 102400` (FR-028b) |
| `email_width`, `email_height` | INTEGER NOT NULL | hauteur ≤ 96, largeur ≤ 400 |
| `email_sha256` | TEXT NOT NULL | |
| `uploaded_at` | TEXT NOT NULL | |
| `uploaded_by` | INTEGER | `REFERENCES accounts (id) ON DELETE SET NULL` |

Fichiers : `FRIGG_DATA_DIR/logo/<web_name>` et `…/<email_name>`.

**Cycle de vie** :

```text
(aucun logo) ──upload-logo──▶ (logo L1) ──upload-logo──▶ (logo L2)
      ▲                           │                          │
      └────────delete-logo────────┴──────────delete-logo─────┘
```

À chaque transition, la ligne est remplacée ou supprimée dans une transaction
`BEGIN IMMEDIATE` ; les fichiers de l'état précédent sont supprimés après la
validation. Au démarrage, tout fichier de `logo/` qui n'est pas référencé par
la ligne courante est supprimé (reprise après incident).

## `outbound_emails` (existante) : colonnes ajoutées

| Colonne | Type | Rôle |
| --- | --- | --- |
| `wrap` | INTEGER NOT NULL DEFAULT 0 | `CHECK (wrap IN (0, 1))` ; 1 = le worker applique le layout avec l'identité courante ([research.md › R6](./research.md#r6-habillage-des-e-mails-au-moment-de-lenvoi)) |
| `footer` | TEXT | mention de pied propre au message (`mail.Data.Footer`), `NULL` = mention par défaut |

Les lignes existantes prennent `wrap = 0` et sont envoyées telles quelles.

**Down** : suppression de `association_logo` ; les colonnes ajoutées à
`association` et à `outbound_emails` sont retirées par `ALTER TABLE … DROP
COLUMN` (aucune ne porte de clé étrangère ni d'index, et leurs `CHECK` ne
portent que sur elles-mêmes). Les déclencheurs de 00009 sont conservés.

## Requêtes sqlc (`internal/database/queries`)

| Fichier | Requête | Usage |
| --- | --- | --- |
| `association.sql` | `GetAssociation` (étendue : toutes les colonnes) | settings, contact, identité, mail |
| | `UpdateGeneralSettings` | tous les champs « Général » en une requête |
| | `UpdateAssociationColors` | couleurs (`NULL` admis) |
| | `GetAssociationLogo`, `UpsertAssociationLogo`, `DeleteAssociationLogo` | logo |
| `emails.sql` | `EnqueueEmail`, `EnqueueCommunicationEmail` (étendues : `wrap`, `footer`) | file d'e-mails |

`GetAssociation` renvoyait jusqu'ici `name` seul : ses appelants (`mail.go`,
`me.go`, `createadmin.go`, `e2eseed`) passent au champ `Name` de la ligne.

## Types du domaine (Go, sans base)

- `association.Settings` : champs « Général » normalisés ; `Normalize(in,
  today) (Settings, []FieldError)`.
- `brand.Palette` : `Brand`, `BrandInk`, `OnBrand`, `Accent`, `AccentInk`,
  `OnAccent` (`brand.RGB`) ; `brand.Derive(primary, accent *RGB) Palette` ;
  `brand.Default`.
- `logo.Variants` : produit par `logo.Process(r io.Reader) (Variants, error)` :
  type source, deux images encodées avec dimensions.
- `mail.Identity` : `Name`, `Palette`, `Logo *Attachment` (variante e-mail).
