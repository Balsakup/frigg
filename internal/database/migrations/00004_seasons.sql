-- +goose Up
-- Saisons de l'association, qui servent aussi d'exercice comptable.
-- Dates civiles AAAA-MM-JJ, bornes incluses.
-- opened_at : NULL = brouillon ; l'état close est calculé à partir de la date
-- du jour (research.md 003 › R2).
CREATE TABLE seasons (
    id INTEGER PRIMARY KEY,
    starts_on TEXT NOT NULL UNIQUE CHECK (date(starts_on) = starts_on),
    ends_on TEXT NOT NULL CHECK (date(ends_on) = ends_on),
    opened_at TEXT,
    created_at TEXT NOT NULL,
    CHECK (ends_on > starts_on)
) STRICT;

-- +goose Down
DROP TABLE seasons;
