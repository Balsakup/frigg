-- name: ListMembershipFees :many
-- Fees of one season; the domain sorts them (folded labels).
SELECT * FROM membership_fees WHERE season_id = sqlc.arg(season_id) ORDER BY id;

-- name: CreateMembershipFee :one
INSERT INTO membership_fees (season_id, label, amount_cents, renewal_amount_cents, description, created_at, updated_at)
VALUES (sqlc.arg(season_id), sqlc.arg(label), sqlc.arg(amount_cents), sqlc.narg(renewal_amount_cents), sqlc.narg(description), sqlc.arg(now), sqlc.arg(now))
RETURNING *;

-- name: UpdateMembershipFee :execrows
UPDATE membership_fees
SET label = sqlc.arg(label), amount_cents = sqlc.arg(amount_cents),
    renewal_amount_cents = sqlc.narg(renewal_amount_cents),
    description = sqlc.narg(description), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND season_id = sqlc.arg(season_id);

-- name: DeleteMembershipFee :execrows
DELETE FROM membership_fees WHERE id = sqlc.arg(id) AND season_id = sqlc.arg(season_id);

-- name: CopyMembershipFees :execrows
-- Season duplication: copies every fee of the source season.
INSERT INTO membership_fees (season_id, label, amount_cents, renewal_amount_cents, description, created_at, updated_at)
SELECT sqlc.arg(target_id), src.label, src.amount_cents, src.renewal_amount_cents, src.description, sqlc.arg(now), sqlc.arg(now)
FROM membership_fees AS src WHERE src.season_id = sqlc.arg(source_id);
