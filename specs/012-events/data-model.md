# Data Model: Événements de l'association

**Feature**: `012-events` | **Migration**: `00012_events.sql` (`-- +goose NO TRANSACTION`, voir [research.md › R2](./research.md#r2-rattacher-les-paiements-par-carte-aux-inscriptions--reconstruction-de-online_payments-et-payment_alerts))

Conventions de 006 : tables `STRICT`, montants en centimes, instants en texte
UTC `AAAA-MM-JJTHH:MM:SSZ`, dates en `AAAA-MM-JJ`, commentaires SQL en ASCII
(sqlc). Les `CHECK` sont des filets derrière le domaine `internal/event`.

## events

| Colonne | Type | Règles |
| --- | --- | --- |
| `id` | INTEGER PK | |
| `season_id` | INTEGER NOT NULL | → `seasons(id)` `ON DELETE RESTRICT` ; saison qui contient la date locale (Paris) du début (FR-003, R8) |
| `title` | TEXT NOT NULL | 1–120 caractères |
| `location` | TEXT NOT NULL | 1–200 caractères |
| `description` | TEXT | NULL ou 1–5 000 caractères |
| `starts_at` | TEXT NOT NULL | instant UTC |
| `ends_at` | TEXT NOT NULL | instant UTC, `> starts_at` |
| `openness` | TEXT NOT NULL | `season_members` (défaut) \| `all_members` |
| `capacity` | INTEGER | NULL (illimitée) ou 1–10 000 |
| `price_cents` | INTEGER NOT NULL | 0–1 000 000, 0 = gratuit |
| `registration_opens_at` | TEXT | NULL = dès la publication |
| `registration_closes_at` | TEXT | NULL = début de l'événement ; `≤ ends_at` ; `> registration_opens_at` si les deux |
| `status` | TEXT NOT NULL | `draft` \| `published` \| `cancelled` |
| `cancel_message` | TEXT | NULL ou 1–1 000 caractères |
| `published_at` | TEXT | renseigné à la première publication |
| `cancelled_at`, `cancelled_by` | TEXT, INTEGER | → `accounts` `ON DELETE SET NULL` ; renseignés si `cancelled` |
| `created_by`, `updated_by` | INTEGER | → `accounts` `ON DELETE SET NULL` |
| `created_at`, `updated_at` | TEXT NOT NULL | |

Contraintes : `CHECK (status <> 'cancelled' OR cancelled_at IS NOT NULL)`,
`CHECK (status = 'draft' OR published_at IS NOT NULL)`.

Index : `events_starts_at (starts_at)`, `events_season (season_id,
starts_at)`, `events_status (status, starts_at)`.

## event_registrations

| Colonne | Type | Règles |
| --- | --- | --- |
| `id` | INTEGER PK | |
| `event_id` | INTEGER NOT NULL | → `events(id)` `ON DELETE RESTRICT` (suppression d'un événement seulement sans inscription, R9) |
| `member_id` | INTEGER NOT NULL | → `members(id)` `ON DELETE CASCADE` (R14) |
| `status` | TEXT NOT NULL | `confirmed` \| `cancelled` |
| `registered_by` | INTEGER | → `accounts` `ON DELETE SET NULL` ; compte auteur (FR-017) |
| `registered_at` | TEXT NOT NULL | |
| `cancelled_by`, `cancelled_at` | INTEGER, TEXT | renseignés si `cancelled` |
| `amount_cents` | INTEGER NOT NULL | prix figé à l'inscription (FR-021), 0–1 000 000 |
| `payment_reference` | TEXT UNIQUE | `EVT-AAAA-NNNN` (R5) ; NULL si gratuite ; jamais modifiée |
| `payment_status` | TEXT | NULL (gratuite) \| `due` \| `announced` \| `paid` |
| `announced_method` | TEXT | `cash` \| `cheque` \| `transfer` |
| `paid_method` | TEXT | `cash` \| `cheque` \| `transfer` \| `card` |
| `paid_on` | TEXT | date |
| `paid_by`, `paid_at` | INTEGER, TEXT | administrateur qui a constaté ; NULL pour la carte |
| `payment_note` | TEXT | NULL ou 1–200 caractères |
| `paid_online_payment_id` | INTEGER | → `online_payments(id)` `ON DELETE SET NULL` |
| `paid_amount_cents` | INTEGER | montant reçu par carte, hors contribution |
| `updated_at` | TEXT NOT NULL | |

Contraintes (reprises de `memberships`) :

- `CHECK ((payment_status IS NULL) = (amount_cents = 0))` ;
- `CHECK (payment_status IS NULL OR payment_reference IS NOT NULL)` ;
- `CHECK ((payment_status IS 'paid') = (paid_method IS NOT NULL AND paid_on IS NOT NULL))` ;
- `CHECK (payment_status IS NOT 'announced' OR announced_method IS NOT NULL)` ;
- `CHECK (paid_method IS NOT 'card' OR (paid_online_payment_id IS NOT NULL AND paid_amount_cents IS NOT NULL AND paid_by IS NULL))` ;
- `CHECK (paid_online_payment_id IS NULL OR paid_method = 'card')` ;
- `CHECK ((status = 'cancelled') = (cancelled_at IS NOT NULL))`.

Index :

- `event_registrations_one_confirmed` UNIQUE `(event_id, member_id) WHERE
  status = 'confirmed'` (FR-014) ;
- `event_registrations_event (event_id, status)` ;
- `event_registrations_member (member_id)` ;
- `event_registrations_payment (payment_status) WHERE payment_status IN
  ('due', 'announced')` (« À régler »).

`event_registrations_fts` : FTS5 `payment_reference`, `content =
'event_registrations'`, `content_rowid = 'id'`, `tokenize = "trigram
remove_diacritics 1"`, triggers `_ai`, `_ad`, `_au` (R12), comme
`memberships_fts`.

## online_payments (reconstruite)

Colonnes de 008 inchangées, sauf :

| Colonne | Changement |
| --- | --- |
| `membership_id` | devient facultative (`REFERENCES memberships ON DELETE CASCADE`) |
| `registration_id` | **nouvelle**, `REFERENCES event_registrations(id) ON DELETE CASCADE` |

`CHECK ((membership_id IS NULL) <> (registration_id IS NULL))` : une
tentative a exactement une cible. Index existants recréés, plus
`online_payments_registration (registration_id, created_at)`. Les lignes
existantes sont copiées avec `registration_id = NULL`.

## payment_alerts (reconstruite)

Même changement : `membership_id` facultative, `registration_id` nouvelle
(`ON DELETE CASCADE`), exactement une des deux renseignée. Index existants
recréés, plus `payment_alerts_registration (registration_id)`.

## Relations

```text
seasons 1─* events 1─* event_registrations *─1 members
                              │ 1
                              ├─* online_payments  ─┐
                              └─* payment_alerts  ──┴─ (ou memberships, 008)
event_registrations.paid_online_payment_id ─→ online_payments
accounts ─ registered_by / cancelled_by / paid_by / created_by (SET NULL)
```

Suppression d'une fiche → ses inscriptions → leurs tentatives et
signalements (cascade). Suppression d'un événement : refusée s'il a une
inscription. Suppression d'une saison : refusée si elle a un événement.

## États d'un événement

```text
           publier              annuler
 draft ───────────▶ published ──────────▶ cancelled
   │                    ▲                     │
   │ supprimer          └──── rétablir ───────┘  (avant la fin)
   ▼
 (supprimé)      published + ends_at passé ⇒ phase « Terminé » (calculée)
```

- `draft` : modifiable entièrement, invisible hors administration, aucune
  inscription.
- `published` à venir : modifiable ; dates non modifiables une fois terminé
  (FR-005) ; capacité ≥ nombre d'inscrits (FR-007).
- `cancelled` : inscriptions conservées, non payables (R9).
- Suppression : tout état, à condition de n'avoir aucune inscription.

## États d'une inscription

```text
            annuler (compte, FR-013) / retirer (admin)
 confirmed ───────────────────────────────────────────▶ cancelled
     ▲                                                     │
     └──────── nouvelle inscription (nouvelle ligne) ──────┘
```

Paiement (inscription payante), identique à 006 :

```text
 due ⇄ announced ──▶ paid        (constatation admin ou carte)
  ▲                    │
  └─ annulation de constatation (admin, sauf carte)
```

Le compte peut annuler seulement si `payment_status` est NULL, `due` ou
`announced`, sans tentative par carte en cours (< 60 min), pendant la
période d'inscription. Un remboursement HelloAsso vérifié annule
l'inscription (`RefundCancels`).

## Validation (domaine `internal/event`)

| Règle | Source |
| --- | --- |
| titre 1–120, lieu 1–200, description ≤ 5 000 (espaces de bord retirés) | FR-002 |
| fin > début ; heure locale inexistante refusée | FR-002, R8 |
| début dans une saison | FR-003 |
| capacité 1–10 000, prix 0–10 000 € au centime | FR-002 |
| ouverture < clôture ; clôture ≤ fin | FR-002 |
| capacité ≥ inscrits confirmés | FR-007 |
| dates non modifiables si terminé | FR-005 |
| éligibilité selon l'ouverture | FR-011, R7 |
| période d'inscription ouverte (membre) ; avant la fin (admin) | FR-012, FR-016 |
| places suffisantes, tout ou rien | FR-015, R6 |

Erreurs champ par champ en `season.FieldErrors`, converties en `422` par
l'adaptateur existant.
