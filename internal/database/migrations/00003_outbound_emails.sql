-- +goose Up
-- File d'attente des e-mails, dépilée par le worker (constitution, E-mails).
CREATE TABLE outbound_emails (
    id INTEGER PRIMARY KEY,
    recipient TEXT NOT NULL,
    subject TEXT NOT NULL,
    body_text TEXT NOT NULL,
    body_html TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'sent', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TEXT NOT NULL,
    last_error TEXT,
    created_at TEXT NOT NULL,
    sent_at TEXT
) STRICT;

CREATE INDEX outbound_emails_status_next_idx ON outbound_emails (status, next_attempt_at);

-- +goose Down
DROP TABLE outbound_emails;
