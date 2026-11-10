-- Event registrations (research.md 012 R1, R6, R11, R12). Member-space
-- queries always join members and are scoped by members.account_id.

-- name: CreateRegistration :one
INSERT INTO event_registrations (
    event_id, member_id, status, registered_by, registered_at, amount_cents, payment_reference,
    payment_status, base_amount_cents, adjustment_reason, updated_at
) VALUES (
    sqlc.arg(event_id), sqlc.arg(member_id), 'confirmed', sqlc.narg(actor_id), sqlc.arg(now),
    sqlc.arg(amount_cents), sqlc.narg(payment_reference), sqlc.narg(payment_status),
    sqlc.narg(base_amount_cents), sqlc.narg(adjustment_reason), sqlc.arg(now)
)
RETURNING *;

-- name: CountConfirmedRegistrations :one
SELECT count(*) FROM event_registrations WHERE event_id = sqlc.arg(event_id) AND status = 'confirmed';

-- name: CountMemberConfirmedRegistration :one
SELECT count(*) FROM event_registrations
WHERE event_id = sqlc.arg(event_id) AND member_id = sqlc.arg(member_id) AND status = 'confirmed';

-- name: ListRegistrationReferences :many
SELECT CAST(payment_reference AS TEXT) AS payment_reference FROM event_registrations
WHERE payment_reference IS NOT NULL
  AND substr(payment_reference, 1, length(CAST(sqlc.arg(prefix) AS TEXT))) = CAST(sqlc.arg(prefix) AS TEXT);

-- Members with a validated membership in a season (eligibility, R7).

-- name: ListSeasonMemberIDs :many
SELECT DISTINCT member_id FROM memberships
WHERE season_id = sqlc.arg(season_id) AND status = 'validated';

-- name: GetRegistration :one
SELECT sqlc.embed(r), sqlc.embed(e), m.first_name, m.last_name, m.birth_date, m.phone,
       m.email AS member_email, m.account_id AS member_account_id,
       a.email AS account_email, a.disabled_at AS account_disabled_at,
       rb.email AS registered_by_email, xb.email AS cancelled_by_email, pb.email AS paid_by_email,
       EXISTS (SELECT 1 FROM memberships ms WHERE ms.member_id = r.member_id AND ms.season_id = e.season_id
               AND ms.status = 'validated') AS is_season_member
FROM event_registrations r
JOIN events e ON e.id = r.event_id
JOIN members m ON m.id = r.member_id
LEFT JOIN accounts a ON a.id = m.account_id
LEFT JOIN accounts rb ON rb.id = r.registered_by
LEFT JOIN accounts xb ON xb.id = r.cancelled_by
LEFT JOIN accounts pb ON pb.id = r.paid_by
WHERE r.id = sqlc.arg(id);

-- name: GetRegistrationOfAccount :one
SELECT sqlc.embed(r), sqlc.embed(e), m.first_name, m.last_name
FROM event_registrations r
JOIN events e ON e.id = r.event_id
JOIN members m ON m.id = r.member_id
WHERE r.id = sqlc.arg(id) AND m.account_id = sqlc.arg(account_id);

-- name: ListRegistrationsOfAccount :many
SELECT sqlc.embed(r), sqlc.embed(e), m.first_name, m.last_name
FROM event_registrations r
JOIN events e ON e.id = r.event_id
JOIN members m ON m.id = r.member_id
WHERE m.account_id = sqlc.arg(account_id)
ORDER BY e.starts_at, r.registered_at, r.id;

-- name: ListAccountRegistrationsOfEvent :many
SELECT r.* FROM event_registrations r
JOIN members m ON m.id = r.member_id
WHERE r.event_id = sqlc.arg(event_id) AND r.status = 'confirmed' AND m.account_id = sqlc.arg(account_id);

-- name: ListRegistrationsOfEvent :many
SELECT sqlc.embed(r), m.first_name, m.last_name, m.birth_date, m.phone,
       m.email AS member_email, a.email AS account_email, a.disabled_at AS account_disabled_at,
       rb.email AS registered_by_email,
       EXISTS (SELECT 1 FROM memberships ms WHERE ms.member_id = r.member_id AND ms.season_id = sqlc.arg(season_id)
               AND ms.status = 'validated') AS is_season_member,
       op.environment AS paid_environment
FROM event_registrations r
JOIN members m ON m.id = r.member_id
LEFT JOIN accounts a ON a.id = m.account_id
LEFT JOIN accounts rb ON rb.id = r.registered_by
LEFT JOIN online_payments op ON op.id = r.paid_online_payment_id
WHERE r.event_id = sqlc.arg(event_id)
  AND (CAST(sqlc.narg(status) AS TEXT) IS NULL OR r.status = CAST(sqlc.narg(status) AS TEXT))
ORDER BY r.registered_at, r.id;

-- name: ListRegistrations :many
SELECT sqlc.embed(r), e.title AS event_title, e.starts_at AS event_starts_at, e.status AS event_status,
       m.first_name, m.last_name
FROM event_registrations r
JOIN events e ON e.id = r.event_id
JOIN members m ON m.id = r.member_id
WHERE (CAST(sqlc.narg(event_id) AS INTEGER) IS NULL OR r.event_id = CAST(sqlc.narg(event_id) AS INTEGER))
  AND (CAST(sqlc.narg(status) AS TEXT) IS NULL OR r.status = CAST(sqlc.narg(status) AS TEXT))
  AND (CAST(sqlc.narg(payment) AS TEXT) IS NULL
       OR (CAST(sqlc.narg(payment) AS TEXT) = 'free' AND r.payment_status IS NULL)
       OR r.payment_status = CAST(sqlc.narg(payment) AS TEXT))
  AND (CAST(sqlc.narg(query) AS TEXT) IS NULL
       OR r.id IN (SELECT rowid FROM event_registrations_fts WHERE event_registrations_fts MATCH CAST(sqlc.narg(query) AS TEXT))
       OR r.member_id IN (SELECT rowid FROM members_fts WHERE members_fts MATCH CAST(sqlc.narg(query) AS TEXT)))
  AND (CAST(sqlc.narg(open_alerts) AS INTEGER) IS NULL
       OR r.id IN (SELECT pa.registration_id FROM payment_alerts pa WHERE pa.resolved_at IS NULL))
ORDER BY r.registered_at DESC, r.id DESC;

-- name: CancelRegistration :execrows
UPDATE event_registrations
SET status = 'cancelled', cancelled_by = sqlc.narg(actor_id), cancelled_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'confirmed';

-- name: AnnounceRegistrationPaymentOfAccount :execrows
UPDATE event_registrations
SET payment_status = 'announced', announced_method = sqlc.arg(method), updated_at = sqlc.arg(now)
WHERE event_registrations.id = sqlc.arg(id) AND status = 'confirmed' AND payment_status IN ('due', 'announced')
  AND member_id IN (SELECT members.id FROM members WHERE members.account_id = sqlc.arg(account_id));

-- name: RecordRegistrationPayment :execrows
UPDATE event_registrations
SET payment_status = 'paid', paid_method = sqlc.arg(method), paid_on = sqlc.arg(paid_on),
    paid_by = sqlc.narg(actor_id), paid_at = sqlc.arg(now), payment_note = sqlc.narg(note),
    paid_amount_cents = sqlc.arg(paid_amount_cents), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'confirmed' AND payment_status IN ('due', 'announced');

-- AdjustRegistrationAmount sets the amount to pay of an unpaid confirmed
-- registration (research.md 014 R6). base_amount_cents and adjustment_reason
-- are NULL when the amount equals the theoretical one.

-- name: AdjustRegistrationAmount :execrows
UPDATE event_registrations
SET amount_cents = sqlc.arg(amount_cents), base_amount_cents = sqlc.narg(base_amount_cents),
    adjustment_reason = sqlc.narg(adjustment_reason), payment_reference = sqlc.narg(payment_reference),
    payment_status = sqlc.narg(payment_status), announced_method = sqlc.narg(announced_method),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'confirmed' AND payment_status IS NOT 'paid';

-- name: CancelRegistrationPayment :execrows
UPDATE event_registrations
SET payment_status = CASE WHEN announced_method IS NULL THEN 'due' ELSE 'announced' END,
    paid_method = NULL, paid_on = NULL, paid_by = NULL, paid_at = NULL, payment_note = NULL,
    paid_amount_cents = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND payment_status = 'paid' AND paid_method IS NOT 'card';

-- RecordRegistrationCardPayment is only called by the HelloAsso verification
-- (research.md 008 R5, 012 R4).

-- name: RecordRegistrationCardPayment :execrows
UPDATE event_registrations
SET payment_status = 'paid', paid_method = 'card', paid_on = sqlc.arg(paid_on), paid_by = NULL,
    paid_at = sqlc.arg(now), payment_note = NULL, paid_amount_cents = sqlc.arg(paid_amount_cents),
    paid_online_payment_id = sqlc.arg(online_payment_id), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'confirmed' AND payment_status IN ('due', 'announced');

-- name: CancelRefundedRegistration :execrows
UPDATE event_registrations
SET status = 'cancelled', cancelled_by = NULL, cancelled_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'confirmed' AND paid_method = 'card'
  AND paid_online_payment_id = sqlc.arg(online_payment_id);

-- Dues of an account (R11): confirmed, payable, event not cancelled.

-- name: ListDueRegistrationsOfAccount :many
SELECT sqlc.embed(r), e.title AS event_title, m.first_name, m.last_name
FROM event_registrations r
JOIN events e ON e.id = r.event_id
JOIN members m ON m.id = r.member_id
WHERE m.account_id = sqlc.arg(account_id) AND r.status = 'confirmed' AND e.status <> 'cancelled'
  AND r.payment_status IN ('due', 'announced')
ORDER BY r.registered_at, r.id;

-- name: CountPaidUpcomingRegistrationsOfMember :one
SELECT count(*) FROM event_registrations r
JOIN events e ON e.id = r.event_id
WHERE r.member_id = sqlc.arg(member_id) AND r.status = 'confirmed' AND r.payment_status = 'paid'
  AND e.status <> 'cancelled' AND e.ends_at > sqlc.arg(now);

-- Notice recipients (R10): confirmed registrations with both addresses; the
-- account address only when the account is active.

-- name: ListNoticeRecipientsOfEvent :many
SELECT r.id, r.member_id, r.payment_status, r.payment_reference, r.amount_cents, m.first_name, m.last_name,
       CAST(CASE WHEN a.disabled_at IS NULL THEN coalesce(a.email, '') ELSE '' END AS TEXT) AS account_email,
       CAST(coalesce(m.email, '') AS TEXT) AS member_email
FROM event_registrations r
JOIN members m ON m.id = r.member_id
LEFT JOIN accounts a ON a.id = m.account_id
WHERE r.event_id = sqlc.arg(event_id) AND r.status = 'confirmed';

-- name: GetRegistrationNotice :one
SELECT r.id, r.member_id, r.event_id, r.payment_status, r.payment_reference, r.amount_cents,
       r.paid_method, r.paid_on, r.paid_amount_cents, m.first_name, m.last_name,
       CAST(CASE WHEN a.disabled_at IS NULL THEN coalesce(a.email, '') ELSE '' END AS TEXT) AS account_email,
       CAST(coalesce(m.email, '') AS TEXT) AS member_email
FROM event_registrations r
JOIN members m ON m.id = r.member_id
LEFT JOIN accounts a ON a.id = m.account_id
WHERE r.id = sqlc.arg(id);
