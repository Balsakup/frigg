-- name: CreateInvitation :one
INSERT INTO invitations (email, role, token_hash, invited_by, status, expires_at, created_at, updated_at)
VALUES (sqlc.arg(email), sqlc.arg(role), sqlc.arg(token_hash), sqlc.narg(invited_by), 'pending', sqlc.arg(expires_at), sqlc.arg(now), sqlc.arg(now))
RETURNING *;

-- name: GetPendingInvitationByEmail :one
SELECT * FROM invitations WHERE email = sqlc.arg(email) AND status = 'pending';

-- name: GetInvitationByTokenHash :one
SELECT * FROM invitations WHERE token_hash = sqlc.arg(token_hash);

-- name: GetInvitationByID :one
SELECT * FROM invitations WHERE id = sqlc.arg(id);

-- name: ListPendingInvitations :many
SELECT i.id, i.email, i.role, i.expires_at, i.created_at, a.email AS invited_by_email
FROM invitations i
LEFT JOIN accounts a ON a.id = i.invited_by
WHERE i.status = 'pending'
ORDER BY i.created_at DESC, i.id DESC;

-- name: ReplaceInvitationToken :one
UPDATE invitations
SET token_hash = sqlc.arg(token_hash), expires_at = sqlc.arg(expires_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending'
RETURNING *;

-- name: CancelInvitation :execrows
UPDATE invitations SET status = 'cancelled', updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: MarkInvitationAccepted :exec
UPDATE invitations SET status = 'accepted', updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: DeleteInvitationsByEmail :exec
DELETE FROM invitations WHERE email = sqlc.arg(email);
