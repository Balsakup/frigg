-- +goose Up
-- Fiches membre : personnes de l'association (research.md 005 › R1).
-- account_id NULL = fiche sans compte, gérée par le seul bureau. ON DELETE
-- SET NULL n'est qu'un filet : la suppression d'un compte efface ou détache
-- toujours ses fiches explicitement (R9).
CREATE TABLE members (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    is_self INTEGER NOT NULL DEFAULT 0 CHECK (is_self IN (0, 1)),
    first_name TEXT NOT NULL CHECK (length(first_name) BETWEEN 1 AND 80),
    last_name TEXT NOT NULL CHECK (length(last_name) BETWEEN 1 AND 80),
    birth_date TEXT NOT NULL CHECK (birth_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    email TEXT CHECK (email IS NULL OR length(email) BETWEEN 3 AND 254),
    -- Téléphone normalisé : 0612345678 ou +32470123456 (R3).
    phone TEXT CHECK (phone IS NULL OR length(phone) BETWEEN 9 AND 16),
    address_line1 TEXT CHECK (address_line1 IS NULL OR length(address_line1) BETWEEN 1 AND 100),
    address_line2 TEXT CHECK (address_line2 IS NULL OR length(address_line2) BETWEEN 1 AND 100),
    postal_code TEXT CHECK (postal_code IS NULL OR postal_code GLOB '[0-9][0-9][0-9][0-9][0-9]'),
    city TEXT CHECK (city IS NULL OR length(city) BETWEEN 1 AND 100),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    -- Une fiche « Vous » appartient forcément à un compte.
    CHECK (is_self = 0 OR account_id IS NOT NULL),
    -- Adresse postale tout ou rien ; complément seulement avec une adresse.
    CHECK (
        (address_line1 IS NULL) = (postal_code IS NULL)
        AND (postal_code IS NULL) = (city IS NULL)
        AND (address_line2 IS NULL OR address_line1 IS NOT NULL)
    )
) STRICT;

CREATE INDEX members_account_id ON members (account_id);

-- Recherche des doublons probables par date de naissance (R8).
CREATE INDEX members_birth_date ON members (birth_date);

-- Une fiche « Vous » au plus par compte (R10).
CREATE UNIQUE INDEX members_one_self ON members (account_id) WHERE is_self = 1;

-- Recherche par fragment, sans casse ni accents (R6 ; principe VI : FTS5 à
-- contenu externe, trigram pour les champs d'identité courts).
CREATE VIRTUAL TABLE members_fts USING fts5(
    first_name,
    last_name,
    email,
    phone,
    content = 'members',
    content_rowid = 'id',
    tokenize = "trigram remove_diacritics 1"
);

-- +goose StatementBegin
CREATE TRIGGER members_fts_ai AFTER INSERT ON members BEGIN
    INSERT INTO members_fts (rowid, first_name, last_name, email, phone)
    VALUES (new.id, new.first_name, new.last_name, new.email, new.phone);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER members_fts_ad AFTER DELETE ON members BEGIN
    INSERT INTO members_fts (members_fts, rowid, first_name, last_name, email, phone)
    VALUES ('delete', old.id, old.first_name, old.last_name, old.email, old.phone);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER members_fts_au AFTER UPDATE OF first_name, last_name, email, phone ON members BEGIN
    INSERT INTO members_fts (members_fts, rowid, first_name, last_name, email, phone)
    VALUES ('delete', old.id, old.first_name, old.last_name, old.email, old.phone);
    INSERT INTO members_fts (rowid, first_name, last_name, email, phone)
    VALUES (new.id, new.first_name, new.last_name, new.email, new.phone);
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER members_fts_au;
DROP TRIGGER members_fts_ad;
DROP TRIGGER members_fts_ai;
DROP TABLE members_fts;
DROP TABLE members;
