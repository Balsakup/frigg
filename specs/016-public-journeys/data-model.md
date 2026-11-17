# Data Model: Parcours publics

Migration unique : `internal/database/migrations/00014_public_journeys.sql`.
Elle est transactionnelle et n'effectue aucune reconstruction de table
([research.md › R4, R5](./research.md)). Les commentaires SQL sont en ASCII
(sqlc). Toutes les dates sont en texte UTC ISO 8601, comme dans l'existant.

## Tables nouvelles

### `public_submissions` : dépôts non confirmés (R3)

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `id` | INTEGER | PK |
| `kind` | TEXT | NOT NULL, `IN ('membership', 'registration')` |
| `season_id` | INTEGER | FK `seasons` ON DELETE CASCADE ; requis si `kind = 'membership'` |
| `fee_id` | INTEGER | FK `membership_fees` ON DELETE CASCADE ; requis si `membership` |
| `declared_kind` | TEXT | `IN ('first', 'renewal')` ; requis si `membership` |
| `event_id` | INTEGER | FK `events` ON DELETE CASCADE ; requis si `kind = 'registration'` |
| `first_name`, `last_name` | TEXT | NOT NULL, 1..80 (005 FR-001) |
| `birth_date` | TEXT | NOT NULL, `AAAA-MM-JJ` |
| `email` | TEXT | NOT NULL, 3..254, normalisé (`auth.NormalizeEmail`) |
| `phone` | TEXT | nullable, normalisé (005) |
| `address_line1`, `address_line2`, `postal_code`, `city` | TEXT | nullable, tout ou rien (mêmes CHECK que `members`) |
| `custom_values` | TEXT | NOT NULL, défaut `'[]'`, JSON canonique produit par `customfield` |
| `create_account` | INTEGER | NOT NULL, 0/1 |
| `consented_at` | TEXT | NOT NULL |
| `token_hash` | BLOB | NOT NULL UNIQUE (SHA-256) |
| `expires_at` | TEXT | NOT NULL (création + 48 h) |
| `created_at` | TEXT | NOT NULL |

- `CHECK` de cohérence `kind` ↔ colonnes de cible.
- Index `public_submissions_created_at (created_at)` pour la purge à 7 jours.
- Pas d'adresse IP ni d'agent utilisateur conservés (collecte minimale).
- Le `ON DELETE CASCADE` vers la cible supprime un dépôt dont la saison, le
  tarif ou l'événement disparaît. La confirmation le traite comme expiré.

### `tracking_links` : liens personnels (R1, R2)

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `id` | INTEGER | PK |
| `email` | TEXT | NOT NULL, normalisé |
| `token_hash` | BLOB | NOT NULL UNIQUE |
| `expires_at` | TEXT | NOT NULL (émission + 30 j) |
| `revoked_at` | TEXT | nullable |
| `last_used_at` | TEXT | nullable, mis à jour au plus une fois par minute |
| `created_at` | TEXT | NOT NULL |

- Index `tracking_links_email (email)`, pour la révocation à la création de
  compte.
- Index `tracking_links_expires_at (expires_at)`, pour la purge (expiré
  depuis plus de 24 h).
- Le lien ne référence aucune fiche : son périmètre est recalculé à chaque
  accès à partir de `email` (FR-014).

## Colonnes ajoutées

### `members` (005)

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `origin` | TEXT | NOT NULL DEFAULT `'internal'`, `IN ('internal', 'public')` |
| `email_confirmed_at` | TEXT | nullable ; renseigné à la confirmation d'un dépôt |
| `consented_at` | TEXT | nullable ; date du consentement du formulaire public |

- Index `members_tracked_email ON members (email COLLATE NOCASE) WHERE
  account_id IS NULL AND email IS NOT NULL` : il sert au périmètre d'un lien,
  qu'il ne concerne que les fiches suivies par lien.
- Index `members_origin_created ON members (origin, created_at) WHERE
  origin = 'public'` pour les fiches inactives (R14).
- La modification de l'adresse d'une fiche par un administrateur remet
  `email_confirmed_at` à `NULL` (Edge Cases).

### `memberships` (006)

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `via_public` | INTEGER | NOT NULL DEFAULT 0, `IN (0, 1)`, `CHECK (via_public = 0 OR (requested_by IS NULL AND created_by IS NULL AND declared_kind IS NOT NULL))` |

### `event_registrations` (012)

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `via_public` | INTEGER | NOT NULL DEFAULT 0, `IN (0, 1)`, `CHECK (via_public = 0 OR registered_by IS NULL)` |

### `online_payments` (008, 012)

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `via_link` | INTEGER | NOT NULL DEFAULT 0, `IN (0, 1)`, `CHECK (via_link = 0 OR account_id IS NULL)` |

### `events` (012)

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `public_slug` | TEXT | nullable, 22 caractères base64url, `CHECK (public_slug IS NULL OR openness = 'all_members')` |

- Index unique partiel `events_public_slug ON events (public_slug) WHERE
  public_slug IS NOT NULL` (`ADD COLUMN` n'accepte pas `UNIQUE`).

### `association` (010)

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `public_membership_enabled` | INTEGER | NOT NULL DEFAULT 0, `IN (0, 1)` |

## Domaine

### `internal/public` (nouveau, pur, sans base)

- `Submission` : entrée normalisée d'un dépôt. `ValidateMembershipSubmission`
  et `ValidateRegistrationSubmission` composent `member.Validate` (identité et
  coordonnées), l'exigence d'une adresse, le consentement et le piège
  (`Honeypot(string) bool`).
- `Durations` : `ConfirmTTL = 48h`, `SubmissionRetention = 7×24h`,
  `LinkTTL = 30×24h`, `InactiveAfter = 12 mois`.
- `ConfirmDecision(state) Outcome` : à partir de l'état relu en transaction
  (compte trouvé, actif ou désactivé, nombre de fiches, fiche identique
  existante, conditions de cible), il renvoie `tracking | account | duplicate
  | closed(reason)`. Il est testé en table.
- `Inactive(member, lastActivity, cutoff) bool`, miroir testable de la
  requête de R14.

### `internal/event` (012)

- `Openness` gagne `Public`. `OpennessOf(openness string, slug sql.NullString)`
  et `StorageOf(Openness) (openness string, keepSlug bool)`.
- `Eligibility` traite `Public` comme `AllMembers` pour un compte (FR-023).
- `PublicVisible(status, phase, openness) bool` gouverne FR-025.

### `internal/auth` (001)

- `Access` gagne `Link`. `Allows(role, Link)` est toujours faux, et
  `String()` vaut `"link"`.
- `Limits` gagne les neuf limiteurs de R9.

## États et transitions

### Dépôt

```text
(envoi) ──► en attente de confirmation ──(lien suivi, conditions OK)──► supprimé + effets (FR-011)
                    │                    ──(lien suivi, conditions KO)──► supprimé, aucun effet
                    │                    ──(fiche identique existante)──► supprimé, renvoi au suivi
                    └──(48 h sans confirmation)──► lien refusé ──(7 j après l'envoi)──► purgé
```

### Lien personnel

```text
émis ──(ouvert)──► en session ──(30 j)──► expiré ──(+24 h)──► purgé
  └────────────(compte créé pour l'adresse)──► révoqué ──(+24 h après expiration)──► purgé
```

### Adhésion et inscription publiques

Ce sont les cycles de 006 et de 012, inchangés. Seules la création (par
confirmation de dépôt) et l'auteur (`via_public`) diffèrent. L'annulation
par lien est enregistrée avec `cancelled_by` à `NULL` et journalisée
`space=tracking`.

## Règles de validation (rappel des exigences)

| Règle | Source | Où |
| --- | --- | --- |
| Identité et coordonnées | 005 FR-001, FR-002 | `member.Validate` |
| Adresse obligatoire, consentement obligatoire | FR-006, FR-026 | `public.Validate*` |
| Champs « Rempli par le membre » obligatoires (adhésion) | 011 FR-015 | `customfield.Apply`, audience `Owner` |
| Aucun champ personnalisé (inscription) | FR-026, principe II | `custom_values = '[]'` pour `kind = 'registration'` |
| Une adhésion active par fiche et saison | 006 FR-003 | index `memberships_one_active` |
| Capacité | 012 FR-015 | comptage en transaction |
| Au plus 20 fiches par compte | 005 FR-004 | `ConfirmDecision`, création de compte |
| Périmètre d'un lien | FR-014 | requêtes `…OfTrackedEmail` |
