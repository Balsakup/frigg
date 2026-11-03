-- Memberships (research.md 006). Member-space queries always join members and
-- are scoped by members.account_id (R5); admin queries live below.

-- name: ListMembershipsOfAccount :many
SELECT sqlc.embed(ms), sqlc.embed(f), m.first_name, m.last_name, m.birth_date
FROM memberships ms
JOIN members m ON m.id = ms.member_id
JOIN membership_fees f ON f.id = ms.fee_id
WHERE m.account_id = sqlc.arg(account_id)
  AND (CAST(sqlc.narg(member_id) AS INTEGER) IS NULL OR ms.member_id = CAST(sqlc.narg(member_id) AS INTEGER));

-- name: GetMembershipOfAccount :one
SELECT sqlc.embed(ms), sqlc.embed(f), m.first_name, m.last_name, m.birth_date
FROM memberships ms
JOIN members m ON m.id = ms.member_id
JOIN membership_fees f ON f.id = ms.fee_id
WHERE ms.id = sqlc.arg(id) AND m.account_id = sqlc.arg(account_id);

-- name: CancelMembershipOfAccount :execrows
UPDATE memberships
SET status = 'cancelled', cancelled_by = sqlc.arg(actor_id), cancelled_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE memberships.id = sqlc.arg(id) AND status = 'pending'
  AND member_id IN (SELECT members.id FROM members WHERE members.account_id = sqlc.arg(account_id));

-- name: AnnouncePaymentOfAccount :execrows
UPDATE memberships
SET payment_status = 'announced', announced_method = sqlc.arg(method), updated_at = sqlc.arg(now)
WHERE memberships.id = sqlc.arg(id) AND status = 'validated' AND payment_status IN ('due', 'announced')
  AND member_id IN (SELECT members.id FROM members WHERE members.account_id = sqlc.arg(account_id));

-- name: ListActiveSeasonsOfAccount :many
SELECT ms.member_id, ms.season_id
FROM memberships ms
JOIN members m ON m.id = ms.member_id
WHERE m.account_id = sqlc.arg(account_id) AND ms.status IN ('pending', 'validated');

-- Shared and admin queries.

-- name: CreateMembership :one
INSERT INTO memberships (
    member_id, season_id, fee_id, status, declared_kind, kind, amount_cents,
    requested_by, created_by, requested_at, decided_by, decided_at,
    payment_reference, payment_status, updated_at
) VALUES (
    sqlc.arg(member_id), sqlc.arg(season_id), sqlc.arg(fee_id), sqlc.arg(status),
    sqlc.narg(declared_kind), sqlc.narg(kind), sqlc.narg(amount_cents),
    sqlc.narg(requested_by), sqlc.narg(created_by), sqlc.arg(now),
    sqlc.narg(decided_by), sqlc.narg(decided_at),
    sqlc.narg(payment_reference), sqlc.narg(payment_status), sqlc.arg(now)
)
RETURNING id;

-- name: GetMembership :one
SELECT sqlc.embed(ms), sqlc.embed(f), m.first_name, m.last_name, m.birth_date,
       m.account_id AS member_account_id,
       a.email AS account_email, a.disabled_at AS account_disabled_at,
       rb.email AS requested_by_email, cb.email AS created_by_email,
       db.email AS decided_by_email, xb.email AS cancelled_by_email,
       pb.email AS paid_by_email
FROM memberships ms
JOIN members m ON m.id = ms.member_id
JOIN membership_fees f ON f.id = ms.fee_id
LEFT JOIN accounts a ON a.id = m.account_id
LEFT JOIN accounts rb ON rb.id = ms.requested_by
LEFT JOIN accounts cb ON cb.id = ms.created_by
LEFT JOIN accounts db ON db.id = ms.decided_by
LEFT JOIN accounts xb ON xb.id = ms.cancelled_by
LEFT JOIN accounts pb ON pb.id = ms.paid_by
WHERE ms.id = sqlc.arg(id);

-- ListMemberships returns every matching row: sorting without accents and
-- paging are done in Go (R10). The text query matches the payment reference
-- or the member record.

-- name: ListMemberships :many
SELECT sqlc.embed(ms), f.label AS fee_label, m.first_name, m.last_name, m.birth_date
FROM memberships ms
JOIN members m ON m.id = ms.member_id
JOIN membership_fees f ON f.id = ms.fee_id
WHERE (CAST(sqlc.narg(season_id) AS INTEGER) IS NULL OR ms.season_id = CAST(sqlc.narg(season_id) AS INTEGER))
  AND (CAST(sqlc.narg(status) AS TEXT) IS NULL OR ms.status = CAST(sqlc.narg(status) AS TEXT))
  AND (CAST(sqlc.narg(payment) AS TEXT) IS NULL
       OR (CAST(sqlc.narg(payment) AS TEXT) = 'free' AND ms.status = 'validated' AND ms.amount_cents = 0)
       OR (ms.status = 'validated' AND ms.payment_status = CAST(sqlc.narg(payment) AS TEXT)))
  AND (CAST(sqlc.narg(member_id) AS INTEGER) IS NULL OR ms.member_id = CAST(sqlc.narg(member_id) AS INTEGER))
  AND (CAST(sqlc.narg(query) AS TEXT) IS NULL
       OR ms.id IN (SELECT rowid FROM memberships_fts WHERE memberships_fts MATCH CAST(sqlc.narg(query) AS TEXT))
       OR ms.member_id IN (SELECT rowid FROM members_fts WHERE members_fts MATCH CAST(sqlc.narg(query) AS TEXT))
       OR ms.id IN (SELECT op.membership_id FROM online_payments op
                    WHERE op.order_id = CAST(sqlc.narg(order_id) AS INTEGER)))
  AND (CAST(sqlc.narg(paid_method) AS TEXT) IS NULL
       OR (ms.payment_status = 'paid' AND ms.paid_method = CAST(sqlc.narg(paid_method) AS TEXT)))
  AND (CAST(sqlc.narg(open_alerts) AS INTEGER) IS NULL
       OR ms.id IN (SELECT pa.membership_id FROM payment_alerts pa WHERE pa.resolved_at IS NULL));

-- name: ValidateMembership :execrows
UPDATE memberships
SET status = 'validated', fee_id = sqlc.arg(fee_id), kind = sqlc.arg(kind),
    amount_cents = sqlc.arg(amount_cents), payment_reference = sqlc.narg(payment_reference),
    payment_status = sqlc.narg(payment_status), decided_by = sqlc.arg(actor_id),
    decided_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: RefuseMembership :execrows
UPDATE memberships
SET status = 'refused', reason = sqlc.arg(reason), decided_by = sqlc.arg(actor_id),
    decided_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: UpdateValidatedMembership :execrows
UPDATE memberships
SET fee_id = sqlc.arg(fee_id), kind = sqlc.arg(kind), amount_cents = sqlc.arg(amount_cents),
    payment_reference = sqlc.narg(payment_reference), payment_status = sqlc.narg(payment_status),
    announced_method = sqlc.narg(announced_method), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'validated' AND payment_status IS NOT 'paid';

-- name: CancelMembership :execrows
UPDATE memberships
SET status = 'cancelled', reason = sqlc.arg(reason), cancelled_by = sqlc.arg(actor_id),
    cancelled_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'validated';

-- name: RecordMembershipPayment :execrows
UPDATE memberships
SET payment_status = 'paid', paid_method = sqlc.arg(method), paid_on = sqlc.arg(paid_on),
    paid_by = sqlc.arg(actor_id), paid_at = sqlc.arg(now), payment_note = sqlc.narg(note),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'validated' AND payment_status IN ('due', 'announced');

-- name: CancelMembershipPayment :execrows
UPDATE memberships
SET payment_status = CASE WHEN announced_method IS NULL THEN 'due' ELSE 'announced' END,
    paid_method = NULL, paid_on = NULL, paid_by = NULL, paid_at = NULL, payment_note = NULL,
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'validated' AND payment_status = 'paid'
  AND paid_method IS NOT 'card';

-- RecordCardPayment is only called by the HelloAsso verification
-- (research.md 008 R5), after the payment has been checked with HelloAsso.

-- name: RecordCardPayment :execrows
UPDATE memberships
SET payment_status = 'paid', paid_method = 'card', paid_on = sqlc.arg(paid_on), paid_by = NULL,
    paid_at = sqlc.arg(now), payment_note = NULL, paid_amount_cents = sqlc.arg(paid_amount_cents),
    paid_online_payment_id = sqlc.arg(online_payment_id), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'validated' AND payment_status IN ('due', 'announced');

-- CancelRefundedMembership cancels a membership whose card payment was
-- refunded in HelloAsso (verified), whatever the season state.

-- name: CancelRefundedMembership :execrows
UPDATE memberships
SET status = 'cancelled', reason = sqlc.arg(reason), cancelled_by = NULL, cancelled_at = sqlc.arg(now),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'validated' AND paid_method = 'card'
  AND paid_online_payment_id = sqlc.arg(online_payment_id);

-- Helpers.

-- name: ListPaymentReferences :many
SELECT payment_reference FROM memberships
WHERE payment_reference IS NOT NULL AND substr(payment_reference, 1, length(CAST(sqlc.arg(prefix) AS TEXT))) = CAST(sqlc.arg(prefix) AS TEXT);

-- name: CountValidatedInSeason :one
SELECT count(*) FROM memberships WHERE season_id = sqlc.arg(season_id) AND status = 'validated';

-- name: HasValidatedMembership :one
SELECT count(*) FROM memberships
WHERE member_id = sqlc.arg(member_id) AND season_id = sqlc.arg(season_id) AND status = 'validated';

-- name: ListSeasonMembershipsForStats :many
SELECT ms.status, ms.kind, ms.amount_cents, ms.fee_id, f.label AS fee_label,
       ms.payment_status, ms.announced_method, ms.paid_method, ms.paid_amount_cents,
       op.environment AS paid_environment
FROM memberships ms
JOIN membership_fees f ON f.id = ms.fee_id
LEFT JOIN online_payments op ON op.id = ms.paid_online_payment_id
WHERE ms.season_id = sqlc.arg(season_id);

-- name: CountPendingMemberships :one
SELECT count(*) FROM memberships WHERE status = 'pending';

-- name: CountMembershipsOfFee :one
SELECT count(*) FROM memberships WHERE fee_id = sqlc.arg(fee_id);

-- name: ListUsedFeeIDs :many
SELECT DISTINCT fee_id FROM memberships WHERE season_id = sqlc.arg(season_id);

-- name: CountActiveMembershipsOfMember :one
SELECT count(*) FROM memberships
WHERE member_id = sqlc.arg(member_id) AND status IN ('pending', 'validated');

-- name: ListMembershipsOfMember :many
SELECT * FROM memberships WHERE member_id = sqlc.arg(member_id);
