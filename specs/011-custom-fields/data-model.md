# Data Model: Champs personnalisés

**Feature**: `011-custom-fields` | Migration `00011_custom_fields.sql` (append-only)

Tables `STRICT`. Horodatages en texte RFC 3339 UTC, comme le reste du schéma.
Commentaires SQL des requêtes sqlc en ASCII.

## custom_fields

| Colonne | Type | Règle |
| --- | --- | --- |
| `id` | INTEGER PK | |
| `scope` | TEXT NOT NULL | `member` \| `membership` |
| `label` | TEXT NOT NULL | 1–80 ; unique par `scope` après `textfold.Fold` (contrôlé en Go dans la transaction d'écriture, archivés compris) |
| `type` | TEXT NOT NULL | `short_text` \| `long_text` \| `number` \| `date` \| `bool` \| `choice` ; immuable dès qu'une valeur existe |
| `visibility` | TEXT NOT NULL DEFAULT `admin` | `admin` \| `visible` \| `editable` |
| `required` | INTEGER NOT NULL DEFAULT 0 | 0/1 ; `CHECK (required = 0 OR type <> 'bool')` |
| `help` | TEXT | NULL ou 1–200 |
| `position` | INTEGER NOT NULL | ordre d'affichage dans la portée |
| `archived_at` | TEXT | NULL = actif |
| `created_at`, `updated_at` | TEXT NOT NULL | |

Index : `custom_fields_scope_position (scope, archived_at, position)`.
Limite de 30 champs actifs par portée : contrôlée en Go (création,
rétablissement).

## custom_field_choices

| Colonne | Type | Règle |
| --- | --- | --- |
| `id` | INTEGER PK | |
| `field_id` | INTEGER NOT NULL | → `custom_fields(id)` ON DELETE CASCADE |
| `label` | TEXT NOT NULL | 1–80 ; unique par champ après `textfold.Fold` (Go) |
| `position` | INTEGER NOT NULL | |

2 à 50 choix pour un champ `choice`, aucun pour les autres types (Go).

## member_field_values

| Colonne | Type | Règle |
| --- | --- | --- |
| `id` | INTEGER PK | rowid de l'index FTS |
| `member_id` | INTEGER NOT NULL | → `members(id)` ON DELETE CASCADE |
| `field_id` | INTEGER NOT NULL | → `custom_fields(id)` ON DELETE CASCADE |
| `value_text` | TEXT | 1–200 (texte court, seul indexé) |
| `value_long` | TEXT | 1–2000 |
| `value_number` | REAL | |
| `value_date` | TEXT | `date(value_date) = value_date` |
| `value_bool` | INTEGER | 0/1 |
| `value_choice_id` | INTEGER | → `custom_field_choices(id)` DEFERRABLE INITIALLY DEFERRED (NO ACTION différé, voir ci-dessous) |
| `updated_at` | TEXT NOT NULL | |

- `UNIQUE (member_id, field_id)` ; index `(field_id)` pour les compteurs et
  les filtres.
- `CHECK` : exactement une des six colonnes `value_*` est non NULL.
- `value_choice_id` : contrainte **NO ACTION différée** (`DEFERRABLE INITIALLY
  DEFERRED`, sans clause `ON DELETE`), vérifiée à la validation de la
  transaction. Supprimer seul un choix utilisé échoue toujours ; supprimer
  un champ réussit, car ses valeurs et ses choix partent dans la même
  cascade. `ON DELETE RESTRICT` est écarté : SQLite l'applique
  immédiatement, même sur une contrainte différée, et la suppression d'un
  champ échoue alors selon l'ordre de création des tables (vérifié sur
  SQLite 3.51).
- La cohérence colonne ↔ type du champ (et choix ↔ champ) est garantie par le
  domaine `internal/customfield` ; le type étant immuable dès qu'une valeur
  existe, elle ne peut plus diverger.
- Absence de ligne = « Non renseigné » (« Non » pour `bool`).

**FTS** : `member_field_values_fts` (`value_text`, `content =
'member_field_values'`, `content_rowid = 'id'`, `tokenize = "trigram
remove_diacritics 1"`) et triggers `_ai`, `_ad`, `_au` (`AFTER UPDATE OF
value_text`), sur le modèle de `members_fts`. Les lignes à `value_text NULL`
sont insérées avec un contenu NULL, sans effet sur la recherche.

## membership_field_values

Identique à `member_field_values` avec `membership_id` → `memberships(id)`
ON DELETE CASCADE et `UNIQUE (membership_id, field_id)`. **Pas** d'index FTS
(recherche par champ d'adhésion hors périmètre).

## Relations

```text
custom_fields 1 ─── * custom_field_choices
custom_fields 1 ─── * member_field_values     * ─── 1 members
custom_fields 1 ─── * membership_field_values * ─── 1 memberships
custom_field_choices 1 ─── * *_field_values.value_choice_id (NO ACTION différé)
```

Le `scope` du champ correspond à la table de valeurs (contrôlé en Go à
l'écriture).

## États d'un champ

```text
            archive               delete (confirmé)
  actif ───────────────▶ archivé ─────────────────▶ (supprimé)
    ▲                       │
    └──── restore ──────────┘   (refusé si 30 actifs ; placé en fin de liste)
  actif ──── delete (confirmé) ────▶ (supprimé)
```

Un champ archivé : absent des formulaires, des sorties de fiche et
d'adhésion, des filtres et de la recherche ; valeurs conservées ; plus exigé
s'il était obligatoire.

## Écriture des valeurs (adhésions)

| État de l'adhésion | Admin | Owner (`editable` seulement) |
| --- | --- | --- |
| `pending` | `update-membership-fields`, `validate-membership` | `update-my-membership-fields` |
| `validated` (payée ou non) | `update-membership-fields` | `update-my-membership-fields` |
| `refused`, `cancelled` | lecture seule (`409`) | lecture seule (`409`) |

Une nouvelle adhésion ne reprend jamais les valeurs d'une autre (FR-013).

## Validation (domaine `internal/customfield`)

| Type | Entrée acceptée | Stockage |
| --- | --- | --- |
| `short_text` | `text`, espaces de bord retirés, 1–200, une ligne | `value_text` |
| `long_text` | `text`, 1–2000, retours à la ligne conservés | `value_long` |
| `number` | `text` = saisie brute (« 12,5 » ou « 12.5 »), convertie par le serveur (pas de règle dupliquée dans le navigateur) ; renvoyé dans `number` | `value_number`, fini, \|x\| < 10¹² |
| `date` | `date` `AAAA-MM-JJ` valide | `value_date` |
| `bool` | `checked` ; `false` supprime la ligne | `value_bool = 1` |
| `choice` | `choiceId` appartenant au champ | `value_choice_id` |

Toute entrée sans aucune valeur (ou texte vide) **efface** la valeur.
