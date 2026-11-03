# Data Model: Envoi d'e-mails groupés aux adhérents

**Feature**: `009-bulk-email` | **Migration**: `internal/database/migrations/00009_communications.sql`

Dates au format ISO 8601 UTC (`2006-01-02T15:04:05Z`), comme le reste du
schéma. Tables `STRICT`. Commentaires SQL en ASCII (sqlc).

## communications

Une communication envoyée. Jamais modifiée après insertion (FR-020).

| Colonne | Type | Contraintes | Notes |
| --- | --- | --- | --- |
| `id` | INTEGER | PK | |
| `draft_key` | TEXT | NOT NULL, UNIQUE, longueur 36 | Clé de préparation (UUID), idempotence (research R7) |
| `author_id` | INTEGER | REFERENCES accounts ON DELETE SET NULL | « Compte supprimé » si NULL |
| `season_id` | INTEGER | NOT NULL, REFERENCES seasons ON DELETE RESTRICT | |
| `situations` | TEXT | NOT NULL | Liste triée séparée par des virgules, sous-ensemble non vide de `up_to_date,announced,due` (CHECK par GLOB) |
| `fee_ids` | TEXT | NOT NULL DEFAULT '' | Identifiants de tarifs triés, séparés par des virgules ; vide = tous |
| `target_label` | TEXT | NOT NULL, 1–300 | Description lisible figée à l'envoi : « 2026-2027 · En impayé · Tarifs : Jeune, Adulte » (FR-018) |
| `subject` | TEXT | NOT NULL, 1–150 | |
| `body` | TEXT | NOT NULL, 1–10 000 | Texte simple |
| `members_count` | INTEGER | NOT NULL, ≥ 1 | Adhérents ciblés, non joignables compris |
| `unreachable_count` | INTEGER | NOT NULL, ≥ 0 | |
| `recipients_count` | INTEGER | NOT NULL, ≥ 1 | |
| `created_at` | TEXT | NOT NULL | Date d'envoi |

Index : `communications_created_at (created_at DESC, id DESC)` pour la liste.

> `season_id` est `RESTRICT` : une saison ayant reçu une communication ne peut
> plus être supprimée. Seules les saisons en brouillon sont supprimables (003)
> et elles ne peuvent pas être ciblées : aucune règle existante n'est touchée.

## communication_attachments

Fichier déposé pour une préparation, puis rattaché à la communication envoyée.

| Colonne | Type | Contraintes | Notes |
| --- | --- | --- | --- |
| `id` | INTEGER | PK | |
| `draft_key` | TEXT | NOT NULL, longueur 36 | Préparation d'origine |
| `uploaded_by` | INTEGER | REFERENCES accounts ON DELETE SET NULL | Une préparation n'appartient qu'à son auteur ; NULL après suppression du compte (les pièces envoyées restent avec leur communication, les pièces de préparation orphelines sont purgées par le balayage de 24 h) |
| `communication_id` | INTEGER | REFERENCES communications ON DELETE CASCADE | NULL tant que non envoyée |
| `stored_name` | TEXT | NOT NULL, UNIQUE, GLOB 32 hex | Nom sur disque (research R5) |
| `original_name` | TEXT | NOT NULL, 1–150 | Nom nettoyé |
| `content_type` | TEXT | NOT NULL, CHECK IN (6 types) | Type détecté (research R4) |
| `size_bytes` | INTEGER | NOT NULL, 1–5 242 880 | |
| `sha256` | TEXT | NOT NULL, 64 hex | Intégrité |
| `created_at` | TEXT | NOT NULL | Base du délai de 24 h |

Index : `(draft_key, uploaded_by)` ; `(communication_id)` ;
`communication_attachments_drafts (created_at) WHERE communication_id IS NULL`
pour le balayage.

Types acceptés (`content_type`) :
`application/pdf`, `image/jpeg`, `image/png`,
`application/vnd.openxmlformats-officedocument.wordprocessingml.document`,
`application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`,
`application/vnd.oasis.opendocument.text`,
`application/vnd.oasis.opendocument.spreadsheet`.

Règles (domaine `internal/communication`, revérifiées à l'envoi) :

- au plus 5 pièces par `(draft_key, uploaded_by)` non rattachées ;
- somme des `size_bytes` ≤ 10 485 760 ;
- une pièce rattachée (`communication_id` non NULL) n'est ni supprimable ni
  purgée.

## communication_recipients

Une adresse retenue pour une communication (FR-007, FR-019).

| Colonne | Type | Contraintes | Notes |
| --- | --- | --- | --- |
| `id` | INTEGER | PK | |
| `communication_id` | INTEGER | NOT NULL, REFERENCES communications ON DELETE CASCADE | |
| `email` | TEXT | 3–254, NULL possible | Adresse utilisée ; NULL après suppression du compte (research R12) |
| `members_count` | INTEGER | NOT NULL, ≥ 1 | Nombre d'adhérents représentés |
| `outbound_email_id` | INTEGER | REFERENCES outbound_emails ON DELETE SET NULL | NULL après purge de la file |
| `status` | TEXT | NOT NULL, IN (`pending`, `sent`, `failed`) | Recopié par déclencheurs |
| `sent_at` | TEXT | | |

Index : `(communication_id, status)` ; `UNIQUE (outbound_email_id)` ;
`(email)` pour l'effacement à la suppression d'un compte.

### Transitions de `status`

```text
pending ──(worker : MarkEmailSent)────────────▶ sent
pending ──(worker : MarkEmailFailed)──────────▶ failed
pending ──(DELETE de l'e-mail en attente)─────▶ failed   (suppression de compte)
sent / failed : états finaux
```

Déclencheurs (dans `00009`) :

- `outbound_emails_status_to_recipient` : `AFTER UPDATE OF status ON
  outbound_emails WHEN new.communication_id IS NOT NULL` → `UPDATE
  communication_recipients SET status = new.status, sent_at = new.sent_at
  WHERE outbound_email_id = new.id` ;
- `outbound_emails_pending_deleted` : `BEFORE DELETE ON outbound_emails WHEN
  old.status = 'pending' AND old.communication_id IS NOT NULL` → `status =
  'failed'` sur la ligne liée (BEFORE, pour agir avant le `SET NULL`).

## outbound_emails (modifiée)

Colonnes ajoutées par `ALTER TABLE … ADD COLUMN` :

| Colonne | Type | Notes |
| --- | --- | --- |
| `communication_id` | INTEGER REFERENCES communications ON DELETE SET NULL | Pièces jointes à joindre, déclencheurs |
| `from_name` | TEXT, longueur ≤ 120 | Nom d'affichage de l'expéditeur (FR-016) |

Index : `outbound_emails_communication_id (communication_id)`.

## Entités du domaine (`internal/communication`, sans base)

- `Situation` : `up_to_date` | `announced` | `due`.
- `Preset` : `all` → toutes ; `up_to_date` → {`up_to_date`} ; `unpaid` →
  {`announced`, `due`}. `PresetOf(situations)` renvoie le préréglage
  correspondant ou `custom` (affichage de la cible dans l'historique).
- `Candidate` (ligne lue) : `MemberID`, `FirstName`, `LastName`, `Situation`,
  `FeeLabel`, `AccountEmail`, `MemberEmail`.
- `Audience` : `Members []Target` (avec adresse de contact), `Unreachable
  []Target`, `Recipients []Recipient{Email, MemberIDs}` ;
  `BuildAudience([]Candidate) Audience` (research R2).
- `Limits` : `MaxFiles = 5`, `MaxFileBytes = 5 << 20`, `MaxTotalBytes = 10 << 20`,
  `SubjectMax = 150`, `BodyMax = 10000`.
- `DetectType(io.ReaderAt, int64) (string, error)` et
  `SanitizeFilename(string) string` (research R4, R5).

## Données lues (inchangées)

- `memberships` (006/008) : `season_id`, `fee_id`, `status = 'validated'`,
  `amount_cents`, `payment_status`.
- `members` (005) : prénom, nom, `email`, `account_id`.
- `accounts` (001) : `email`.
- `membership_fees` (004) : libellé du tarif.
- `seasons` (003) : libellé, état (brouillon exclu).
