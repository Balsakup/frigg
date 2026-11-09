-- Payment alerts for administrators (research.md 008 R10).

-- name: CreatePaymentAlert :exec
INSERT INTO payment_alerts (membership_id, registration_id, online_payment_id, kind, created_at)
VALUES (sqlc.narg(membership_id), sqlc.narg(registration_id), sqlc.arg(online_payment_id), sqlc.arg(kind), sqlc.arg(now))
ON CONFLICT (online_payment_id, kind) DO NOTHING;

-- name: ListPaymentAlertsOfMembership :many
SELECT pa.*, a.email AS resolved_by_email FROM payment_alerts pa
LEFT JOIN accounts a ON a.id = pa.resolved_by
WHERE pa.membership_id = CAST(sqlc.arg(membership_id) AS INTEGER)
ORDER BY pa.resolved_at IS NOT NULL, pa.created_at;

-- name: GetPaymentAlert :one
SELECT * FROM payment_alerts WHERE id = sqlc.arg(id);

-- name: ResolvePaymentAlert :execrows
UPDATE payment_alerts SET resolved_at = sqlc.arg(now), resolved_by = sqlc.arg(actor_id)
WHERE id = sqlc.arg(id) AND resolved_at IS NULL;

-- name: CountOpenMembershipPaymentAlerts :one
SELECT count(*) FROM payment_alerts WHERE resolved_at IS NULL AND membership_id IS NOT NULL;

-- name: CountOpenRegistrationPaymentAlerts :one
SELECT count(*) FROM payment_alerts WHERE resolved_at IS NULL AND registration_id IS NOT NULL;

-- name: ListOpenAlertCounts :many
SELECT CAST(membership_id AS INTEGER) AS membership_id, count(*) AS open_alerts FROM payment_alerts
WHERE resolved_at IS NULL AND membership_id IS NOT NULL
GROUP BY membership_id;

-- name: ResolveAlertsOfOnlinePayment :exec
UPDATE payment_alerts SET resolved_at = sqlc.arg(now)
WHERE online_payment_id = sqlc.arg(online_payment_id) AND resolved_at IS NULL;

-- name: ListPaymentAlertsOfRegistration :many
SELECT pa.*, a.email AS resolved_by_email FROM payment_alerts pa
LEFT JOIN accounts a ON a.id = pa.resolved_by
WHERE pa.registration_id = CAST(sqlc.arg(registration_id) AS INTEGER)
ORDER BY pa.resolved_at IS NOT NULL, pa.created_at;

-- name: ListOpenAlertCountsByRegistration :many
SELECT CAST(registration_id AS INTEGER) AS registration_id, count(*) AS open_alerts FROM payment_alerts
WHERE resolved_at IS NULL AND registration_id IS NOT NULL
GROUP BY registration_id;
