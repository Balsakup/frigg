-- Payment alerts for administrators (research.md 008 R10).

-- name: CreatePaymentAlert :exec
INSERT INTO payment_alerts (membership_id, online_payment_id, kind, created_at)
VALUES (sqlc.arg(membership_id), sqlc.arg(online_payment_id), sqlc.arg(kind), sqlc.arg(now))
ON CONFLICT (online_payment_id, kind) DO NOTHING;

-- name: ListPaymentAlertsOfMembership :many
SELECT pa.*, a.email AS resolved_by_email FROM payment_alerts pa
LEFT JOIN accounts a ON a.id = pa.resolved_by
WHERE pa.membership_id = sqlc.arg(membership_id)
ORDER BY pa.resolved_at IS NOT NULL, pa.created_at;

-- name: GetPaymentAlert :one
SELECT * FROM payment_alerts WHERE id = sqlc.arg(id);

-- name: ResolvePaymentAlert :execrows
UPDATE payment_alerts SET resolved_at = sqlc.arg(now), resolved_by = sqlc.arg(actor_id)
WHERE id = sqlc.arg(id) AND resolved_at IS NULL;

-- name: CountOpenPaymentAlerts :one
SELECT count(*) FROM payment_alerts WHERE resolved_at IS NULL;

-- name: ListOpenAlertCounts :many
SELECT membership_id, count(*) AS open_alerts FROM payment_alerts
WHERE resolved_at IS NULL
GROUP BY membership_id;

-- name: ResolveAlertsOfOnlinePayment :exec
UPDATE payment_alerts SET resolved_at = sqlc.arg(now)
WHERE online_payment_id = sqlc.arg(online_payment_id) AND resolved_at IS NULL;
