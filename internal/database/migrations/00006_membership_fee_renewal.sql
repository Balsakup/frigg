-- +goose Up
-- Montant de renouvellement en centimes ; NULL = même montant pour toute
-- adhésion (research.md 004 › R12).
ALTER TABLE membership_fees ADD COLUMN renewal_amount_cents INTEGER
    CHECK (renewal_amount_cents IS NULL OR renewal_amount_cents BETWEEN 0 AND 1000000);

-- +goose Down
ALTER TABLE membership_fees DROP COLUMN renewal_amount_cents;
