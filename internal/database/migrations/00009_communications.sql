-- +goose Up
-- Communications : e-mails groupes envoyes aux adherents d'une saison
-- (specs/009-bulk-email, data-model.md). Une communication n'est jamais
-- modifiee apres insertion.
CREATE TABLE communications (
    id INTEGER PRIMARY KEY,
    -- Cle de preparation (UUID genere par le navigateur) : idempotence de l'envoi.
    draft_key TEXT NOT NULL UNIQUE CHECK (length(draft_key) = 36),
    author_id INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    season_id INTEGER NOT NULL REFERENCES seasons (id) ON DELETE RESTRICT,
    -- Situations de paiement ciblees, triees et separees par des virgules.
    situations TEXT NOT NULL CHECK (situations IN (
        'announced', 'announced,due', 'announced,due,up_to_date', 'announced,up_to_date',
        'due', 'due,up_to_date', 'up_to_date'
    )),
    -- Tarifs cibles, tries et separes par des virgules ; vide = tous.
    fee_ids TEXT NOT NULL DEFAULT '',
    -- Description lisible de la cible, figee a l'envoi.
    target_label TEXT NOT NULL CHECK (length(target_label) BETWEEN 1 AND 300),
    subject TEXT NOT NULL CHECK (length(subject) BETWEEN 1 AND 150),
    body TEXT NOT NULL CHECK (length(body) BETWEEN 1 AND 10000),
    -- Adherents cibles, non joignables compris.
    members_count INTEGER NOT NULL CHECK (members_count >= 1),
    unreachable_count INTEGER NOT NULL CHECK (unreachable_count >= 0),
    recipients_count INTEGER NOT NULL CHECK (recipients_count >= 1),
    created_at TEXT NOT NULL
) STRICT;

CREATE INDEX communications_created_at ON communications (created_at DESC, id DESC);

-- Pieces jointes : deposees pour une preparation, puis rattachees a la
-- communication envoyee. Le fichier est sur disque sous stored_name.
CREATE TABLE communication_attachments (
    id INTEGER PRIMARY KEY,
    draft_key TEXT NOT NULL CHECK (length(draft_key) = 36),
    -- SET NULL : les pieces d'une communication envoyee survivent a la
    -- suppression du compte de leur auteur ; les pieces de preparation
    -- orphelines sont purgees par le balayage de 24 h.
    uploaded_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    communication_id INTEGER REFERENCES communications (id) ON DELETE CASCADE,
    stored_name TEXT NOT NULL UNIQUE CHECK (length(stored_name) = 32 AND stored_name NOT GLOB '*[^0-9a-f]*'),
    original_name TEXT NOT NULL CHECK (length(original_name) BETWEEN 1 AND 150),
    content_type TEXT NOT NULL CHECK (content_type IN (
        'application/pdf',
        'image/jpeg',
        'image/png',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'application/vnd.oasis.opendocument.text',
        'application/vnd.oasis.opendocument.spreadsheet'
    )),
    size_bytes INTEGER NOT NULL CHECK (size_bytes BETWEEN 1 AND 5242880),
    sha256 TEXT NOT NULL CHECK (length(sha256) = 64 AND sha256 NOT GLOB '*[^0-9a-f]*'),
    created_at TEXT NOT NULL
) STRICT;

CREATE INDEX communication_attachments_draft ON communication_attachments (draft_key, uploaded_by);
CREATE INDEX communication_attachments_communication ON communication_attachments (communication_id);
CREATE INDEX communication_attachments_drafts ON communication_attachments (created_at)
WHERE communication_id IS NULL;

-- File d'e-mails : piece jointes d'une communication et nom d'expediteur.
ALTER TABLE outbound_emails ADD COLUMN communication_id INTEGER REFERENCES communications (id) ON DELETE SET NULL;
ALTER TABLE outbound_emails ADD COLUMN from_name TEXT CHECK (from_name IS NULL OR length(from_name) <= 120);
CREATE INDEX outbound_emails_communication_id ON outbound_emails (communication_id);

-- Destinataires : une adresse par communication, avec l'etat de son envoi
-- recopie depuis outbound_emails par les declencheurs ci-dessous.
CREATE TABLE communication_recipients (
    id INTEGER PRIMARY KEY,
    communication_id INTEGER NOT NULL REFERENCES communications (id) ON DELETE CASCADE,
    -- NULL apres suppression du compte (droit a l'effacement).
    email TEXT CHECK (email IS NULL OR length(email) BETWEEN 3 AND 254),
    members_count INTEGER NOT NULL CHECK (members_count >= 1),
    outbound_email_id INTEGER UNIQUE REFERENCES outbound_emails (id) ON DELETE SET NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'sent', 'failed')),
    sent_at TEXT
) STRICT;

CREATE INDEX communication_recipients_status ON communication_recipients (communication_id, status);
CREATE INDEX communication_recipients_email ON communication_recipients (email);

-- +goose StatementBegin
CREATE TRIGGER outbound_emails_status_to_recipient
AFTER UPDATE OF status ON outbound_emails
WHEN new.communication_id IS NOT NULL
BEGIN
    UPDATE communication_recipients
    SET status = new.status, sent_at = new.sent_at
    WHERE outbound_email_id = new.id;
END;
-- +goose StatementEnd

-- Un e-mail en attente supprime (suppression de compte) ne partira jamais.
-- +goose StatementBegin
CREATE TRIGGER outbound_emails_pending_deleted
BEFORE DELETE ON outbound_emails
WHEN old.status = 'pending' AND old.communication_id IS NOT NULL
BEGIN
    UPDATE communication_recipients SET status = 'failed' WHERE outbound_email_id = old.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER outbound_emails_pending_deleted;
DROP TRIGGER outbound_emails_status_to_recipient;
DROP TABLE communication_recipients;
DROP INDEX outbound_emails_communication_id;

-- SQLite refuse DROP COLUMN sur une colonne porteuse d'une cle etrangere :
-- la table est reconstruite a l'identique de 00003.
CREATE TABLE outbound_emails_old (
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
INSERT INTO outbound_emails_old (id, recipient, subject, body_text, body_html, status, attempts, next_attempt_at, last_error, created_at, sent_at)
SELECT id, recipient, subject, body_text, body_html, status, attempts, next_attempt_at, last_error, created_at, sent_at
FROM outbound_emails;
DROP TABLE outbound_emails;
ALTER TABLE outbound_emails_old RENAME TO outbound_emails;
CREATE INDEX outbound_emails_status_next_idx ON outbound_emails (status, next_attempt_at);

DROP TABLE communication_attachments;
DROP TABLE communications;
