-- +goose Up
-- Remises et dons (research.md 014 R1, R2). Montants en centimes.
-- amount_cents devient le montant a payer. base_amount_cents garde le
-- montant theorique seulement quand il differe (ajustement) ; NULL = pas
-- d'ajustement. adjustment_reason est present exactement avec lui.
ALTER TABLE memberships ADD COLUMN base_amount_cents INTEGER
    CHECK (base_amount_cents IS NULL OR (base_amount_cents BETWEEN 0 AND 1000000
        AND amount_cents IS NOT NULL AND base_amount_cents <> amount_cents));
ALTER TABLE memberships ADD COLUMN adjustment_reason TEXT
    CHECK ((adjustment_reason IS NULL) = (base_amount_cents IS NULL))
    CHECK (adjustment_reason IS NULL OR length(adjustment_reason) BETWEEN 1 AND 200);

ALTER TABLE event_registrations ADD COLUMN base_amount_cents INTEGER
    CHECK (base_amount_cents IS NULL OR (base_amount_cents BETWEEN 0 AND 1000000
        AND base_amount_cents <> amount_cents));
ALTER TABLE event_registrations ADD COLUMN adjustment_reason TEXT
    CHECK ((adjustment_reason IS NULL) = (base_amount_cents IS NULL))
    CHECK (adjustment_reason IS NULL OR length(adjustment_reason) BETWEEN 1 AND 200);

-- paid_amount_cents devient le montant recu de tout paiement constate : les
-- paiements manuels deja constates sont repris pour leur montant a payer.
UPDATE memberships SET paid_amount_cents = amount_cents
WHERE payment_status = 'paid' AND paid_amount_cents IS NULL;
UPDATE event_registrations SET paid_amount_cents = amount_cents
WHERE payment_status = 'paid' AND paid_amount_cents IS NULL;

-- +goose Down
UPDATE event_registrations SET paid_amount_cents = NULL
WHERE payment_status = 'paid' AND paid_method <> 'card';
UPDATE memberships SET paid_amount_cents = NULL
WHERE payment_status = 'paid' AND paid_method <> 'card';
-- Le CHECK du motif lit base_amount_cents : le motif part en premier.
ALTER TABLE event_registrations DROP COLUMN adjustment_reason;
ALTER TABLE event_registrations DROP COLUMN base_amount_cents;
ALTER TABLE memberships DROP COLUMN adjustment_reason;
ALTER TABLE memberships DROP COLUMN base_amount_cents;
