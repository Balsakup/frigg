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
    last_login_at TEXT
) STRICT;

-- Schéma imposé par scs/sqlite3store.
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    expiry REAL NOT NULL
) STRICT;

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

-- +goose Down
DROP TABLE sessions;
DROP TABLE accounts;
DROP TABLE association;
