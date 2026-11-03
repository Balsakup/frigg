-- Communications (specs/009-bulk-email). Every query is admin-only.

-- Audience: every validated membership of a season with its payment situation
-- and contact addresses. Filtering by situation and fee, sorting without
-- accents and deduplication are done in Go (internal/communication).

-- name: ListAudienceCandidates :many
SELECT ms.member_id, ms.fee_id, f.label AS fee_label,
       m.first_name, m.last_name,
       CAST(CASE
           WHEN ms.amount_cents = 0 OR ms.payment_status = 'paid' THEN 'up_to_date'
           WHEN ms.payment_status = 'announced' THEN 'announced'
           ELSE 'due'
       END AS TEXT) AS situation,
       a.email AS account_email, m.email AS member_email
FROM memberships ms
JOIN members m ON m.id = ms.member_id
JOIN membership_fees f ON f.id = ms.fee_id
LEFT JOIN accounts a ON a.id = m.account_id
WHERE ms.season_id = sqlc.arg(season_id) AND ms.status = 'validated'
ORDER BY ms.member_id;

-- name: GetCommunicationIDByDraftKey :one
SELECT id FROM communications WHERE draft_key = sqlc.arg(draft_key);

-- name: CreateCommunication :one
INSERT INTO communications (
    draft_key, author_id, season_id, situations, fee_ids, target_label,
    subject, body, members_count, unreachable_count, recipients_count, created_at
) VALUES (
    sqlc.arg(draft_key), sqlc.arg(author_id), sqlc.arg(season_id), sqlc.arg(situations),
    sqlc.arg(fee_ids), sqlc.arg(target_label), sqlc.arg(subject), sqlc.arg(body),
    sqlc.arg(members_count), sqlc.arg(unreachable_count), sqlc.arg(recipients_count), sqlc.arg(now)
)
RETURNING id;

-- name: GetCommunication :one
SELECT sqlc.embed(c), a.email AS author_email
FROM communications c
LEFT JOIN accounts a ON a.id = c.author_id
WHERE c.id = sqlc.arg(id);

-- name: ListCommunications :many
SELECT c.id, c.subject, c.target_label, c.situations, c.author_id, a.email AS author_email,
       c.recipients_count, c.created_at,
       (SELECT count(*) FROM communication_attachments ca WHERE ca.communication_id = c.id) AS attachments_count,
       (SELECT count(*) FROM communication_recipients r WHERE r.communication_id = c.id AND r.status = 'pending') AS pending_count,
       (SELECT count(*) FROM communication_recipients r WHERE r.communication_id = c.id AND r.status = 'sent') AS sent_count,
       (SELECT count(*) FROM communication_recipients r WHERE r.communication_id = c.id AND r.status = 'failed') AS failed_count
FROM communications c
LEFT JOIN accounts a ON a.id = c.author_id
ORDER BY c.created_at DESC, c.id DESC
LIMIT sqlc.arg(max_count) OFFSET sqlc.arg(skip);

-- name: CountCommunications :one
SELECT count(*) FROM communications;

-- name: CreateCommunicationRecipient :exec
INSERT INTO communication_recipients (communication_id, email, members_count, outbound_email_id, status)
VALUES (sqlc.arg(communication_id), sqlc.arg(email), sqlc.arg(members_count), sqlc.arg(outbound_email_id), 'pending');

-- name: ListCommunicationRecipients :many
SELECT email, members_count, status, sent_at
FROM communication_recipients
WHERE communication_id = sqlc.arg(communication_id)
ORDER BY CASE status WHEN 'failed' THEN 0 WHEN 'pending' THEN 1 ELSE 2 END, email, id;

-- Account deletion (right to erasure): the address leaves the history.

-- name: EraseRecipientEmail :exec
UPDATE communication_recipients SET email = NULL WHERE lower(email) = lower(sqlc.arg(email));

-- Attachments.

-- name: CreateCommunicationAttachment :one
INSERT INTO communication_attachments (
    draft_key, uploaded_by, stored_name, original_name, content_type, size_bytes, sha256, created_at
) VALUES (
    sqlc.arg(draft_key), sqlc.arg(uploaded_by), sqlc.arg(stored_name), sqlc.arg(original_name),
    sqlc.arg(content_type), sqlc.arg(size_bytes), sqlc.arg(sha256), sqlc.arg(now)
)
RETURNING *;

-- name: ListDraftAttachments :many
SELECT * FROM communication_attachments
WHERE draft_key = sqlc.arg(draft_key) AND uploaded_by = sqlc.arg(uploaded_by) AND communication_id IS NULL
ORDER BY id;

-- name: GetDraftAttachment :one
SELECT * FROM communication_attachments
WHERE id = sqlc.arg(id) AND uploaded_by = sqlc.arg(uploaded_by) AND communication_id IS NULL;

-- name: DeleteDraftAttachment :execrows
DELETE FROM communication_attachments WHERE id = sqlc.arg(id) AND communication_id IS NULL;

-- name: AttachDraftToCommunication :execrows
UPDATE communication_attachments SET communication_id = sqlc.arg(communication_id)
WHERE draft_key = sqlc.arg(draft_key) AND uploaded_by = sqlc.arg(uploaded_by) AND communication_id IS NULL;

-- name: ListCommunicationAttachments :many
SELECT * FROM communication_attachments WHERE communication_id = sqlc.arg(communication_id) ORDER BY id;

-- name: GetCommunicationAttachment :one
SELECT * FROM communication_attachments WHERE id = sqlc.arg(id) AND communication_id = sqlc.arg(communication_id);

-- name: ListExpiredDraftAttachments :many
SELECT * FROM communication_attachments
WHERE communication_id IS NULL AND created_at < sqlc.arg(before)
ORDER BY created_at
LIMIT 100;

-- name: ListAllStoredNames :many
SELECT stored_name FROM communication_attachments;
