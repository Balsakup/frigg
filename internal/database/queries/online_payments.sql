-- Card payment attempts through HelloAsso (research.md 008 R4, R5, R8).

-- A target is either a membership or an event registration (012 R2).

-- name: CreateOnlinePayment :one
INSERT INTO online_payments (membership_id, registration_id, account_id, environment, amount_cents, status, created_at, updated_at)
VALUES (sqlc.narg(membership_id), sqlc.narg(registration_id), sqlc.narg(account_id), sqlc.arg(environment),
        sqlc.arg(amount_cents), 'pending', sqlc.arg(now), sqlc.arg(now))
RETURNING *;

-- name: GetOnlinePayment :one
SELECT * FROM online_payments WHERE id = sqlc.arg(id);

-- Member space: only attempts of a member record managed by the account.

-- name: GetOnlinePaymentOfAccount :one
SELECT op.* FROM online_payments op
JOIN memberships ms ON ms.id = op.membership_id
JOIN members m ON m.id = ms.member_id
WHERE op.id = sqlc.arg(id) AND op.membership_id = CAST(sqlc.arg(membership_id) AS INTEGER)
  AND m.account_id = sqlc.arg(account_id);

-- name: FindResumableOnlinePayment :one
SELECT * FROM online_payments
WHERE membership_id = CAST(sqlc.arg(membership_id) AS INTEGER) AND environment = sqlc.arg(environment)
  AND amount_cents = sqlc.arg(amount_cents) AND status = 'pending'
  AND redirect_url IS NOT NULL AND created_at > sqlc.arg(since)
ORDER BY created_at DESC
LIMIT 1;

-- name: SetOnlinePaymentCheckout :exec
UPDATE online_payments
SET checkout_intent_id = sqlc.arg(checkout_intent_id), redirect_url = sqlc.arg(redirect_url),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: FailOnlinePayment :exec
UPDATE online_payments
SET status = 'failed', error_code = sqlc.arg(error_code), finished_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- CloseOnlinePayment moves a non-paid attempt to failed, abandoned or expired.

-- name: CloseOnlinePayment :execrows
UPDATE online_payments
SET status = sqlc.arg(status), error_code = coalesce(sqlc.narg(error_code), error_code),
    finished_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: SucceedOnlinePayment :execrows
UPDATE online_payments
SET status = 'succeeded', order_id = sqlc.arg(order_id), helloasso_payment_id = sqlc.arg(helloasso_payment_id),
    received_amount_cents = sqlc.arg(received_amount_cents), paid_at = sqlc.arg(paid_at),
    error_code = NULL, finished_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status <> 'succeeded';

-- name: TouchOnlinePaymentVerified :exec
UPDATE online_payments SET last_verified_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: ListOnlinePaymentsOfMembership :many
SELECT * FROM online_payments WHERE membership_id = CAST(sqlc.arg(membership_id) AS INTEGER)
ORDER BY created_at DESC, id DESC;

-- name: LatestOnlinePaymentOfMembership :one
SELECT * FROM online_payments WHERE membership_id = CAST(sqlc.arg(membership_id) AS INTEGER)
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: ListPendingOnlinePaymentsBefore :many
SELECT * FROM online_payments
WHERE status = 'pending' AND created_at < sqlc.arg(before)
ORDER BY coalesce(last_verified_at, created_at)
LIMIT 50;

-- Attempts an administrator may verify: pending ones, attempts closed without
-- payment less than a day ago (a payment may have been found later), and
-- succeeded ones (a refund may have been made in HelloAsso).

-- name: ListVerifiableOnlinePaymentsOfMembership :many
SELECT * FROM online_payments
WHERE membership_id = CAST(sqlc.arg(membership_id) AS INTEGER) AND checkout_intent_id IS NOT NULL
  AND (status IN ('pending', 'succeeded')
       OR (status IN ('failed', 'abandoned', 'expired') AND finished_at > sqlc.arg(since)))
ORDER BY created_at;

-- Payer prefill: the account's own member record, if any (FR-004).

-- name: GetSelfMemberOfAccount :one
SELECT first_name, last_name FROM members WHERE account_id = sqlc.arg(account_id) AND is_self = 1
LIMIT 1;

-- Refund notifications carry the HelloAsso payment, not the Frigg metadata.

-- name: FindOnlinePaymentByHelloAsso :one
SELECT * FROM online_payments
WHERE environment = sqlc.arg(environment)
  AND (helloasso_payment_id = sqlc.arg(helloasso_payment_id) OR order_id = sqlc.arg(order_id))
ORDER BY id DESC
LIMIT 1;

-- name: RefundOnlinePayment :execrows
UPDATE online_payments
SET status = 'refunded', refunded_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'succeeded';

-- Event registrations (012 R4): same queries, scoped to a registration.

-- name: GetOnlinePaymentOfRegistrationAccount :one
SELECT op.* FROM online_payments op
JOIN event_registrations r ON r.id = op.registration_id
JOIN members m ON m.id = r.member_id
WHERE op.id = sqlc.arg(id) AND op.registration_id = CAST(sqlc.arg(registration_id) AS INTEGER)
  AND m.account_id = sqlc.arg(account_id);

-- name: FindResumableRegistrationOnlinePayment :one
SELECT * FROM online_payments
WHERE registration_id = CAST(sqlc.arg(registration_id) AS INTEGER) AND environment = sqlc.arg(environment)
  AND amount_cents = sqlc.arg(amount_cents) AND status = 'pending'
  AND redirect_url IS NOT NULL AND created_at > sqlc.arg(since)
ORDER BY created_at DESC
LIMIT 1;

-- name: ListOnlinePaymentsOfRegistration :many
SELECT * FROM online_payments WHERE registration_id = CAST(sqlc.arg(registration_id) AS INTEGER)
ORDER BY created_at DESC, id DESC;

-- name: LatestOnlinePaymentOfRegistration :one
SELECT * FROM online_payments WHERE registration_id = CAST(sqlc.arg(registration_id) AS INTEGER)
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: ListVerifiableOnlinePaymentsOfRegistration :many
SELECT * FROM online_payments
WHERE registration_id = CAST(sqlc.arg(registration_id) AS INTEGER) AND checkout_intent_id IS NOT NULL
  AND (status IN ('pending', 'succeeded')
       OR (status IN ('failed', 'abandoned', 'expired') AND finished_at > sqlc.arg(since)))
ORDER BY created_at;
