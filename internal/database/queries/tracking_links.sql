-- Personal tracking links (research.md 016 R1, R2). Only the SHA-256 hash of
-- the token is stored. A link references no member: its scope is computed
-- from email on every access (FR-014).

-- name: CreateTrackingLink :one
INSERT INTO tracking_links (email, token_hash, expires_at, created_at)
VALUES (sqlc.arg(email), sqlc.arg(token_hash), sqlc.arg(expires_at), sqlc.arg(created_at))
RETURNING id;

-- name: GetTrackingLinkByTokenHash :one
-- Revocation and expiry are checked by the caller.
SELECT * FROM tracking_links WHERE token_hash = sqlc.arg(token_hash);

-- name: GetActiveTrackingLink :one
SELECT * FROM tracking_links
WHERE id = sqlc.arg(id) AND revoked_at IS NULL AND expires_at > sqlc.arg(now);

-- name: TouchTrackingLink :exec
-- Called at most once a minute per link (accessMiddleware).
UPDATE tracking_links SET last_used_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: RevokeTrackingLinksOfEmail :execrows
UPDATE tracking_links SET revoked_at = sqlc.arg(now)
WHERE email = sqlc.arg(email) COLLATE NOCASE AND revoked_at IS NULL;

-- name: PurgeTrackingLinks :execrows
DELETE FROM tracking_links WHERE expires_at < sqlc.arg(before);
