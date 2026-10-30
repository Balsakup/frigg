-- name: EnqueueEmail :exec
INSERT INTO outbound_emails (recipient, subject, body_text, body_html, status, next_attempt_at, created_at)
VALUES (sqlc.arg(recipient), sqlc.arg(subject), sqlc.arg(body_text), sqlc.arg(body_html), 'pending', sqlc.arg(now), sqlc.arg(now));

-- name: ListDueEmails :many
SELECT * FROM outbound_emails
WHERE status = 'pending' AND next_attempt_at <= sqlc.arg(now)
ORDER BY next_attempt_at, id
LIMIT sqlc.arg(max_count);

-- name: MarkEmailSent :exec
UPDATE outbound_emails SET status = 'sent', attempts = attempts + 1, sent_at = sqlc.arg(sent_at), last_error = NULL
WHERE id = sqlc.arg(id);

-- name: MarkEmailRetry :exec
UPDATE outbound_emails SET attempts = sqlc.arg(attempts), next_attempt_at = sqlc.arg(next_attempt_at), last_error = sqlc.arg(last_error)
WHERE id = sqlc.arg(id);

-- name: MarkEmailFailed :exec
UPDATE outbound_emails SET status = 'failed', attempts = sqlc.arg(attempts), last_error = sqlc.arg(last_error)
WHERE id = sqlc.arg(id);

-- name: PurgeSentEmails :exec
DELETE FROM outbound_emails WHERE status = 'sent' AND sent_at < sqlc.arg(before);
