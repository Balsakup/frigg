-- name: DeleteUnusedTokens :exec
DELETE FROM one_time_tokens
WHERE account_id = sqlc.arg(account_id) AND purpose = sqlc.arg(purpose) AND used_at IS NULL;

-- name: CreateToken :exec
INSERT INTO one_time_tokens (account_id, purpose, token_hash, expires_at, created_at)
VALUES (sqlc.arg(account_id), sqlc.arg(purpose), sqlc.arg(token_hash), sqlc.arg(expires_at), sqlc.arg(created_at));

-- name: DeleteAllUnusedTokens :exec
DELETE FROM one_time_tokens WHERE account_id = sqlc.arg(account_id) AND used_at IS NULL;

-- name: GetValidToken :one
SELECT t.* FROM one_time_tokens t
JOIN accounts a ON a.id = t.account_id
WHERE t.token_hash = sqlc.arg(token_hash)
  AND t.purpose = sqlc.arg(purpose)
  AND t.used_at IS NULL
  AND t.expires_at > sqlc.arg(now)
  AND a.disabled_at IS NULL;

-- name: MarkTokenUsed :exec
UPDATE one_time_tokens SET used_at = sqlc.arg(used_at) WHERE id = sqlc.arg(id);
