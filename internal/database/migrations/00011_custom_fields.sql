-- +goose Up
-- Champs personnalises des fiches membres et des adhesions (research.md 011 R1).
-- Definitions + valeurs a colonnes typees : une colonne value_* par type,
-- exactement une renseignee. Absence de ligne = non renseigne.

CREATE TABLE custom_fields (
    id INTEGER PRIMARY KEY,
    scope TEXT NOT NULL CHECK (scope IN ('member', 'membership')),
    label TEXT NOT NULL CHECK (length(label) BETWEEN 1 AND 80),
    type TEXT NOT NULL CHECK (type IN ('short_text', 'long_text', 'number', 'date', 'bool', 'choice')),
    visibility TEXT NOT NULL DEFAULT 'admin' CHECK (visibility IN ('admin', 'visible', 'editable')),
    required INTEGER NOT NULL DEFAULT 0 CHECK (required IN (0, 1)),
    help TEXT CHECK (help IS NULL OR length(help) BETWEEN 1 AND 200),
    position INTEGER NOT NULL,
    archived_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    -- Un oui/non vaut Non tant qu'il n'est pas coche : jamais obligatoire.
    CHECK (required = 0 OR type <> 'bool')
) STRICT;

CREATE INDEX custom_fields_scope_position ON custom_fields (scope, archived_at, position);

CREATE TABLE custom_field_choices (
    id INTEGER PRIMARY KEY,
    field_id INTEGER NOT NULL REFERENCES custom_fields (id) ON DELETE CASCADE,
    label TEXT NOT NULL CHECK (length(label) BETWEEN 1 AND 80),
    position INTEGER NOT NULL
) STRICT;

CREATE INDEX custom_field_choices_field ON custom_field_choices (field_id, position);

-- value_choice_id : NO ACTION differe (verifie a la fin de la transaction).
-- Supprimer seul un choix utilise echoue ; supprimer un champ reussit car
-- valeurs et choix partent dans la meme cascade. ON DELETE RESTRICT est
-- applique immediatement par SQLite, meme differe, et ferait dependre la
-- suppression d'un champ de l'ordre de creation des tables (data-model.md).
CREATE TABLE member_field_values (
    id INTEGER PRIMARY KEY,
    member_id INTEGER NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    field_id INTEGER NOT NULL REFERENCES custom_fields (id) ON DELETE CASCADE,
    value_text TEXT CHECK (value_text IS NULL OR length(value_text) BETWEEN 1 AND 200),
    value_long TEXT CHECK (value_long IS NULL OR length(value_long) BETWEEN 1 AND 2000),
    value_number REAL,
    value_date TEXT CHECK (value_date IS NULL OR date(value_date) = value_date),
    value_bool INTEGER CHECK (value_bool IN (0, 1)),
    value_choice_id INTEGER REFERENCES custom_field_choices (id) DEFERRABLE INITIALLY DEFERRED,
    updated_at TEXT NOT NULL,
    UNIQUE (member_id, field_id),
    CHECK (((value_text IS NOT NULL) + (value_long IS NOT NULL) + (value_number IS NOT NULL)
        + (value_date IS NOT NULL) + (value_bool IS NOT NULL) + (value_choice_id IS NOT NULL)) = 1)
) STRICT;

CREATE INDEX member_field_values_field ON member_field_values (field_id);
CREATE INDEX member_field_values_choice ON member_field_values (value_choice_id);

CREATE TABLE membership_field_values (
    id INTEGER PRIMARY KEY,
    membership_id INTEGER NOT NULL REFERENCES memberships (id) ON DELETE CASCADE,
    field_id INTEGER NOT NULL REFERENCES custom_fields (id) ON DELETE CASCADE,
    value_text TEXT CHECK (value_text IS NULL OR length(value_text) BETWEEN 1 AND 200),
    value_long TEXT CHECK (value_long IS NULL OR length(value_long) BETWEEN 1 AND 2000),
    value_number REAL,
    value_date TEXT CHECK (value_date IS NULL OR date(value_date) = value_date),
    value_bool INTEGER CHECK (value_bool IN (0, 1)),
    value_choice_id INTEGER REFERENCES custom_field_choices (id) DEFERRABLE INITIALLY DEFERRED,
    updated_at TEXT NOT NULL,
    UNIQUE (membership_id, field_id),
    CHECK (((value_text IS NOT NULL) + (value_long IS NOT NULL) + (value_number IS NOT NULL)
        + (value_date IS NOT NULL) + (value_bool IS NOT NULL) + (value_choice_id IS NOT NULL)) = 1)
) STRICT;

CREATE INDEX membership_field_values_field ON membership_field_values (field_id);
CREATE INDEX membership_field_values_choice ON membership_field_values (value_choice_id);

-- Recherche par fragment dans les textes courts des fiches (R5 ; principe VI :
-- FTS5 a contenu externe, trigram pour les identifiants courts). Les textes
-- longs (value_long) ne sont pas indexes.
CREATE VIRTUAL TABLE member_field_values_fts USING fts5(
    value_text,
    content = 'member_field_values',
    content_rowid = 'id',
    tokenize = "trigram remove_diacritics 1"
);

-- +goose StatementBegin
CREATE TRIGGER member_field_values_fts_ai AFTER INSERT ON member_field_values BEGIN
    INSERT INTO member_field_values_fts (rowid, value_text) VALUES (new.id, new.value_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER member_field_values_fts_ad AFTER DELETE ON member_field_values BEGIN
    INSERT INTO member_field_values_fts (member_field_values_fts, rowid, value_text)
    VALUES ('delete', old.id, old.value_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER member_field_values_fts_au AFTER UPDATE OF value_text ON member_field_values BEGIN
    INSERT INTO member_field_values_fts (member_field_values_fts, rowid, value_text)
    VALUES ('delete', old.id, old.value_text);
    INSERT INTO member_field_values_fts (rowid, value_text) VALUES (new.id, new.value_text);
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER member_field_values_fts_au;
DROP TRIGGER member_field_values_fts_ad;
DROP TRIGGER member_field_values_fts_ai;
DROP TABLE member_field_values_fts;
DROP TABLE membership_field_values;
DROP TABLE member_field_values;
DROP TABLE custom_field_choices;
DROP TABLE custom_fields;
