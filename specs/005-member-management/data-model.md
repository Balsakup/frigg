# Modèle de données : Gestion des membres

Complète [002 › data-model.md](../002-admin-account-management/data-model.md).
Décisions : [research.md](./research.md).

## Table `members` (migration `00007_members.sql`)

| Colonne | Type | Contraintes | Rôle |
| --- | --- | --- | --- |
| `id` | `INTEGER` | `PRIMARY KEY` | Identifiant |
| `account_id` | `INTEGER` | `NULL`, `REFERENCES accounts(id) ON DELETE SET NULL` | Compte qui gère la fiche ; `NULL` = « Sans compte » |
| `is_self` | `INTEGER` | `NOT NULL DEFAULT 0`, `CHECK (is_self IN (0, 1))`, `CHECK (is_self = 0 OR account_id IS NOT NULL)` | Fiche du titulaire du compte (« Vous ») |
| `first_name` | `TEXT` | `NOT NULL`, `CHECK (length(first_name) BETWEEN 1 AND 80)` | Prénom nettoyé |
| `last_name` | `TEXT` | `NOT NULL`, `CHECK (length(last_name) BETWEEN 1 AND 80)` | Nom nettoyé |
| `birth_date` | `TEXT` | `NOT NULL`, `CHECK (birth_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]')` | Date de naissance `YYYY-MM-DD` |
| `email` | `TEXT` | `NULL`, `CHECK (email IS NULL OR length(email) BETWEEN 3 AND 254)` | Adresse électronique, minuscules |
| `phone` | `TEXT` | `NULL`, `CHECK (phone IS NULL OR length(phone) BETWEEN 9 AND 16)` | Téléphone normalisé (`0612345678`, `+3247…`) |
| `address_line1` | `TEXT` | `NULL`, ≤ 100 | Ligne d'adresse |
| `address_line2` | `TEXT` | `NULL`, ≤ 100 | Complément |
| `postal_code` | `TEXT` | `NULL`, `CHECK (postal_code IS NULL OR postal_code GLOB '[0-9][0-9][0-9][0-9][0-9]')` | Code postal |
| `city` | `TEXT` | `NULL`, ≤ 100 | Commune |
| `created_at` | `TEXT` | `NOT NULL` | Création (UTC) |
| `updated_at` | `TEXT` | `NOT NULL` | Dernière modification (UTC) |

Table `STRICT`, plus `CHECK` « adresse tout ou rien » :
`(address_line1 IS NULL) = (postal_code IS NULL) AND (postal_code IS NULL) = (city IS NULL)
AND (address_line2 IS NULL OR address_line1 IS NOT NULL)`.

Index :

- `members_account_id` sur `account_id` (« Mes membres », filtre, `SET NULL`) ;
- `members_birth_date` sur `birth_date` (doublons, [R8](./research.md#r8-doublon-probable)) ;
- `members_one_self` : `UNIQUE (account_id) WHERE is_self = 1`
  ([R10](./research.md#r10-plafond-de-20-fiches-par-compte-et-fiche-vous)).

Les `CHECK` sont des filets de sécurité derrière la validation du domaine.

## Table FTS `members_fts`

```text
members_fts USING fts5(first_name, last_name, email, phone,
                       content='members', content_rowid='id',
                       tokenize="trigram remove_diacritics 1")
```

Triggers `members_fts_ai` (insertion), `members_fts_ad` (suppression),
`members_fts_au` (`AFTER UPDATE OF first_name, last_name, email, phone`),
sur le modèle de `accounts_fts` ([R6](./research.md#r6-recherche-administration)).
Les valeurs `NULL` sont indexées comme absentes.

## Règles (contrôlées par `internal/member`)

| Règle | Détail | Exigence |
| --- | --- | --- |
| Prénom, nom | espaces nettoyés ; non vides ; 80 caractères (schéma) | FR-001 |
| Date de naissance | `format: date` (schéma) ; ≤ aujourd'hui ; ≥ aujourd'hui − 120 ans | FR-001 |
| Adresse électronique | facultative ; `net/mail` sans nom ; 254 caractères ; minuscules | FR-002 |
| Téléphone | facultatif ; normalisé ([R3](./research.md#r3-téléphone)) | FR-002 |
| Adresse postale | tout ou rien (ligne 1, code postal, commune) ; code à 5 chiffres ; complément facultatif | FR-002 |
| Rattachement | au plus un compte ; ce compte gère moins de 20 fiches avant ajout | FR-003, FR-004 |
| « Vous » | une fiche au plus par compte ; jamais sur une fiche sans compte ; retiré au changement de rattachement | FR-003 |
| Doublon probable | même `Fold(first_name)`, `Fold(last_name)`, `birth_date` ailleurs dans l'instance → confirmation requise | FR-005 |

Champs facultatifs vides après nettoyage → `NULL`.

## Attributs calculés (jamais stockés)

| Attribut | Calcul | Exigences |
| --- | --- | --- |
| `age` | années révolues entre `birth_date` et `s.today()` | FR-009, US3 |
| `account` | `{ id, email, disabled }` du compte de rattachement, par jointure ; `null` si sans compte (administration seulement) | US3 §1, §5 |
| `actions` (« Mes membres ») | `add` = moins de 20 fiches ; `markSelf` = aucune fiche « Vous » | FR-004, US1 §4 |

## Transitions du rattachement

```text
                 create-member(accountId)        update-member(accountId = B)
 (aucune) ─────────────────────────────► compte A ───────────────────────► compte B
    │                                       │  ▲                          (is_self ← 0)
    │ create-member(sans compte)            │  │ update-member(accountId = A)
    ▼                                       ▼  │
 Sans compte ◄──────────────────────────────┘──┘
           update-member(accountId = null)        (is_self ← 0)
           delete-account(members=keep)           (is_self ← 0)

 compte A ── delete-account(members=delete) ──► fiche effacée
 toute fiche ── delete-member / delete-my-member ──► fiche effacée
```

## Entités existantes touchées

- **Compte** : sa suppression traite d'abord ses fiches selon `members`
  ([R9](./research.md#r9-suppression-dun-compte-gérant-des-fiches-us5)) ;
  sa désactivation ne change pas ses fiches ([R14](./research.md#r14-fiches-dun-compte-désactivé)).
