-- +goose Up
-- Tarifs de l'adhésion d'une saison (research.md 004 › R1).
-- Montant en centimes d'euro.
-- Unicité du libellé sans casse ni accents contrôlée par le domaine (R2) :
-- aucun index unique, aucune colonne repliée.
CREATE TABLE membership_fees (
    id INTEGER PRIMARY KEY,
    season_id INTEGER NOT NULL REFERENCES seasons (id) ON DELETE CASCADE,
    label TEXT NOT NULL CHECK (length(label) BETWEEN 1 AND 80),
    amount_cents INTEGER NOT NULL CHECK (amount_cents BETWEEN 0 AND 1000000),
    description TEXT CHECK (description IS NULL OR length(description) BETWEEN 1 AND 500),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX membership_fees_season_id ON membership_fees (season_id);

-- +goose Down
DROP TABLE membership_fees;
