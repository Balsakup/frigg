# Modèle de données : Authentification et shell applicatif

Base SQLite unique ; identifiants en anglais (constitution, Conventions de
langue). Dates stockées en texte ISO 8601 UTC (`2026-10-30T20:15:00Z`), sauf
`sessions.expiry` imposé par sqlite3store. Toutes les tables ont
`STRICT`.

## association

Singleton de l'instance (principe I).

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `id` | INTEGER | PK, `CHECK (id = 1)` |
| `name` | TEXT | NULL tant que non défini ; 1 à 120 caractères après suppression des espaces |
| `updated_at` | TEXT | NOT NULL |

- La migration initiale insère la ligne `(1, NULL, now)`.
- Affichage : `name` ou, s'il est NULL, « Mon association ».

## accounts

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `id` | INTEGER | PK |
| `email` | TEXT | NOT NULL, UNIQUE ; normalisé (espaces retirés, minuscules) |
| `password_hash` | TEXT | NOT NULL (format PHC argon2id) |
| `role` | TEXT | NOT NULL, `CHECK (role IN ('member','admin'))` |
| `email_verified_at` | TEXT | NULL = adresse non confirmée |
| `auth_epoch` | INTEGER | NOT NULL DEFAULT 1 ; incrémenté à chaque changement ou réinitialisation de mot de passe |
| `created_at` | TEXT | NOT NULL |
| `last_login_at` | TEXT | NULL |

Règles :
- Adresse : 3 à 254 caractères, un seul `@`, partie domaine non vide (validation
  de format simple ; la vraie preuve est la confirmation par e-mail).
- Mot de passe : ≥ 8 caractères, ≥ 1 majuscule, ≥ 1 chiffre (FR-008).
- `admin ⊇ member` : la fonction d'autorisation unique compare
  `rank(role) >= rank(required)` avec `member = 1`, `admin = 2`.
- Un compte créé par `create-admin` ou par acceptation d'invitation a
  `email_verified_at` renseigné.

## sessions

Imposée par `scs/sqlite3store`, créée par migration.

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `token` | TEXT | PK |
| `data` | BLOB | NOT NULL (contient `account_id`, `auth_epoch`) |
| `expiry` | REAL | NOT NULL, index `sessions_expiry_idx` |

## one_time_tokens

Jetons de réinitialisation et de confirmation d'adresse.

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `id` | INTEGER | PK |
| `account_id` | INTEGER | NOT NULL, FK → `accounts(id)` ON DELETE CASCADE |
| `purpose` | TEXT | NOT NULL, `CHECK (purpose IN ('password_reset','email_verification'))` |
| `token_hash` | BLOB | NOT NULL, UNIQUE (SHA-256, 32 octets) |
| `expires_at` | TEXT | NOT NULL |
| `used_at` | TEXT | NULL |
| `created_at` | TEXT | NOT NULL |

- Index `(account_id, purpose)`.
- Durées : `password_reset` 1 h (FR-009), `email_verification` 48 h (FR-011).
- Émettre un jeton supprime les jetons non utilisés de même `(account_id, purpose)`
  (« seul le plus récent est valable »).
- Valide ⇔ `used_at IS NULL AND expires_at > now`.

```text
émis ──(utilisé)──▶ utilisé
  │
  ├──(expiration)──▶ expiré (implicite)
  └──(nouvel envoi)──▶ supprimé
```

## invitations

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `id` | INTEGER | PK |
| `email` | TEXT | NOT NULL, normalisé |
| `role` | TEXT | NOT NULL, `CHECK (role IN ('member','admin'))` |
| `token_hash` | BLOB | NOT NULL, UNIQUE |
| `invited_by` | INTEGER | NULL, FK → `accounts(id)` ON DELETE SET NULL |
| `status` | TEXT | NOT NULL, `CHECK (status IN ('pending','accepted','cancelled'))` |
| `expires_at` | TEXT | NOT NULL (création ou renvoi + 7 jours, FR-017) |
| `created_at` | TEXT | NOT NULL |
| `updated_at` | TEXT | NOT NULL |

- Index unique partiel `ON invitations(email) WHERE status = 'pending'`
  (FR-017c : une seule invitation en attente par adresse).
- « Expirée » n'est pas stocké : une invitation `pending` dont `expires_at` est
  dépassé est présentée comme expirée et ne peut plus être acceptée ; elle peut
  être renvoyée (nouveau jeton, nouvelle échéance) ou annulée.
- Inviter une adresse qui a déjà un compte est refusé (FR-017c).

```text
pending ──(acceptée)──▶ accepted   (crée le compte, ouvre une session)
   │  ▲
   │  └──(renvoyée : nouveau jeton, nouvelle échéance)
   └──(annulée)──▶ cancelled
```

## outbound_emails

File d'attente des e-mails (constitution, E-mails).

| Colonne | Type | Contraintes |
| --- | --- | --- |
| `id` | INTEGER | PK |
| `recipient` | TEXT | NOT NULL |
| `subject` | TEXT | NOT NULL |
| `body_text` | TEXT | NOT NULL |
| `body_html` | TEXT | NOT NULL |
| `status` | TEXT | NOT NULL, `CHECK (status IN ('pending','sent','failed'))` |
| `attempts` | INTEGER | NOT NULL DEFAULT 0 |
| `next_attempt_at` | TEXT | NOT NULL |
| `last_error` | TEXT | NULL (message SMTP, jamais le corps) |
| `created_at` | TEXT | NOT NULL |
| `sent_at` | TEXT | NULL |

- Index `(status, next_attempt_at)`.
- Le corps contient des liens à jeton : la table n'est jamais exposée par l'API
  ni journalisée ; les messages `sent` de plus de 30 jours sont purgés par le
  worker.

```text
pending ──(envoi réussi)──▶ sent
   │ ▲
   │ └──(échec, attempts < 10 : next_attempt_at += backoff)
   └──(échec, attempts = 10)──▶ failed
```

## Migrations

Ordre prévu dans `internal/database/migrations` :

1. `00001_init.sql` : `association` (+ ligne initiale), `accounts`, `sessions`.
2. `00002_auth_tokens.sql` : `one_time_tokens`, `invitations`.
3. `00003_outbound_emails.sql` : `outbound_emails`.
