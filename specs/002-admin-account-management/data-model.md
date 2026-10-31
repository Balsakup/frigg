# Modèle de données : Gestion des comptes par les administrateurs

Complète le [modèle de 001](../001-auth-app-shell/data-model.md) ; mêmes
conventions (identifiants anglais, dates ISO 8601 UTC en texte, `STRICT`).

## accounts (modifiée)

| Colonne ajoutée | Type | Contraintes |
| --- | --- | --- |
| `disabled_at` | TEXT | NULL = compte actif ; date de désactivation sinon |

Règles :

- **Actif** ⇔ `disabled_at IS NULL`. Seul un compte actif peut se connecter,
  recevoir un lien de réinitialisation et compter comme administrateur pour la
  règle « au moins un administrateur actif ».
- **Désactiver** : `disabled_at = now`, `auth_epoch = auth_epoch + 1`,
  suppression des `one_time_tokens` non utilisés du compte.
- **Réactiver** : `disabled_at = NULL` ; `auth_epoch`, rôle, mot de passe et
  `email_verified_at` inchangés.
- **Invariant** : il existe toujours au moins une ligne
  `role = 'admin' AND disabled_at IS NULL`. Garanti par les requêtes
  conditionnelles ci-dessous, pas par une contrainte SQL.

```text
            désactiver (époque +1, jetons supprimés)
  actif ───────────────────────────────────────────▶ désactivé
    ▲                                                   │
    └──────────────────── réactiver ────────────────────┘
    │                                                   │
    └──────────── supprimer ──▶ (effacé) ◀── supprimer ─┘
```

Rôle : `member ⇄ admin`, possible dans les deux états.

### Requêtes conditionnelles (garde-fou « dernier administrateur »)

Chaque écriture qui peut retirer un administrateur actif porte la condition :

```text
cible n'est pas un administrateur actif
OU EXISTS (autre compte o : o.role = 'admin' AND o.disabled_at IS NULL AND o.id <> cible)
```

| Requête | Effet | 0 ligne modifiée signifie |
| --- | --- | --- |
| `DisableAccount` | `disabled_at`, `auth_epoch + 1` si actif | cible absente, déjà désactivée, ou dernier administrateur actif |
| `EnableAccount` | `disabled_at = NULL` si désactivé | cible absente ou déjà active |
| `SetAccountRole` | `role` si différent | cible absente, rôle identique, ou dernier administrateur actif |
| `DeleteAccount` | supprime la ligne | cible absente ou dernier administrateur actif |

Le handler distingue les cas en relisant la cible dans la même transaction.

## accounts_fts (nouvelle, virtuelle)

Index plein texte FTS5 à contenu externe (principe VI).

| Élément | Valeur |
| --- | --- |
| Colonnes indexées | `email` |
| Options | `content='accounts'`, `content_rowid='id'`, `tokenize="trigram remove_diacritics 1"` (champ d'identité court, constitution 1.4.0) |
| Synchronisation | triggers `accounts_fts_ai` (après insertion), `accounts_fts_ad` (après suppression), `accounts_fts_au` (après modification de `email`) |
| Remplissage initial | `INSERT INTO accounts_fts (accounts_fts) VALUES ('rebuild')` dans la migration |

Requête : `a.id IN (SELECT rowid FROM accounts_fts WHERE accounts_fts MATCH ?)`,
avec l'expression produite par `database.FTSTrigramQuery` (voir
[research.md › R7](./research.md#r7-recherche-par-adresse-fts5)). Une
sous-chaîne d'au moins 3 caractères suffit, où qu'elle soit dans l'adresse.

## Tables existantes touchées

| Table | Changement |
| --- | --- |
| `one_time_tokens` | supprimés à la désactivation (non utilisés) ; en cascade à la suppression ; `GetValidToken` ignore les comptes désactivés |
| `invitations` | `invited_by` → `NULL` à la suppression de l'auteur (contrainte existante) ; les invitations **adressées** à l'adresse supprimée sont effacées |
| `outbound_emails` | les messages dont `recipient` est l'adresse supprimée sont effacés |
| `sessions` | inchangée ; sessions invalidées par l'époque, l'état désactivé ou l'absence du compte |

## Représentation dans l'API

```text
AdminAccount {
  id: int, email: string, role: "member"|"admin",
  status: "active"|"disabled", emailVerified: bool,
  createdAt: datetime, lastLoginAt: datetime|null, disabledAt: datetime|null,
  isSelf: bool            // compte de l'administrateur connecté
}
```

## Migration

Pas de nouvelle migration : le projet n'a jamais été déployé, `00001_init.sql`
est modifiée (constitution 1.4.0, [research.md › R13](./research.md#r13-modifier-les-migrations-existantes)) :

1. `disabled_at TEXT` ajoutée dans le `CREATE TABLE accounts` ;
2. création de `accounts_fts` et des trois triggers
   (`-- +goose StatementBegin` / `StatementEnd`) après `accounts` ; pas de
   `'rebuild'` nécessaire (table vide à la création) ;
3. `Down` : suppression des triggers et de `accounts_fts` avant `accounts`.

Les bases de développement existantes sont à recréer.
