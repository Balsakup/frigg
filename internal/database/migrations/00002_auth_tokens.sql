-- +goose Up
CREATE TABLE one_time_tokens (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    purpose TEXT NOT NULL CHECK (purpose IN ('password_reset', 'email_verification')),
    token_hash BLOB NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    used_at TEXT,
    created_at TEXT NOT NULL
) STRICT;

CREATE INDEX one_time_tokens_account_purpose_idx ON one_time_tokens (account_id, purpose);

CREATE TABLE invitations (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('member', 'admin')),
    token_hash BLOB NOT NULL UNIQUE,
    invited_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'accepted', 'cancelled')),
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

-- Une seule invitation en attente par adresse (FR-017c).
CREATE UNIQUE INDEX invitations_pending_email_idx ON invitations (email) WHERE status = 'pending';

-- +goose Down
DROP TABLE invitations;
DROP TABLE one_time_tokens;
