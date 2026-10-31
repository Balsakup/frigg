-- +goose Up
-- Association unique de l'instance (principe I).
CREATE TABLE association (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    name TEXT,
    updated_at TEXT NOT NULL
) STRICT;

INSERT INTO association (id, name, updated_at)
VALUES (1, NULL, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'));

CREATE TABLE accounts (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('member', 'admin')),
    email_verified_at TEXT,
    auth_epoch INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    last_login_at TEXT,
    -- NULL = compte actif ; date de désactivation sinon.
    disabled_at TEXT
) STRICT;

-- Recherche par fragment d'adresse, sans casse ni accents (principe VI :
-- FTS5 à contenu externe, trigram pour les champs d'identité courts).
CREATE VIRTUAL TABLE accounts_fts USING fts5(
    email,
    content = 'accounts',
    content_rowid = 'id',
    tokenize = "trigram remove_diacritics 1"
);

-- +goose StatementBegin
CREATE TRIGGER accounts_fts_ai AFTER INSERT ON accounts BEGIN
    INSERT INTO accounts_fts (rowid, email) VALUES (new.id, new.email);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER accounts_fts_ad AFTER DELETE ON accounts BEGIN
    INSERT INTO accounts_fts (accounts_fts, rowid, email) VALUES ('delete', old.id, old.email);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER accounts_fts_au AFTER UPDATE OF email ON accounts BEGIN
    INSERT INTO accounts_fts (accounts_fts, rowid, email) VALUES ('delete', old.id, old.email);
    INSERT INTO accounts_fts (rowid, email) VALUES (new.id, new.email);
END;
-- +goose StatementEnd

-- Schéma imposé par scs/sqlite3store.
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    expiry REAL NOT NULL
) STRICT;

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

-- +goose Down
DROP TABLE sessions;
DROP TRIGGER accounts_fts_au;
DROP TRIGGER accounts_fts_ad;
DROP TRIGGER accounts_fts_ai;
DROP TABLE accounts_fts;
DROP TABLE accounts;
DROP TABLE association;
