-- name: CreateAccount :one
INSERT INTO accounts (email, password_hash, role, email_verified_at, created_at)
VALUES (sqlc.arg(email), sqlc.arg(password_hash), sqlc.arg(role), sqlc.narg(email_verified_at), sqlc.arg(created_at))
RETURNING *;

-- name: GetAccountByEmail :one
SELECT * FROM accounts WHERE email = sqlc.arg(email);

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = sqlc.arg(id);

-- name: UpdateLastLogin :exec
UPDATE accounts SET last_login_at = sqlc.arg(last_login_at) WHERE id = sqlc.arg(id);

-- name: UpdatePasswordAndBumpEpoch :one
UPDATE accounts
SET password_hash = sqlc.arg(password_hash), auth_epoch = auth_epoch + 1
WHERE id = sqlc.arg(id)
RETURNING auth_epoch;

-- name: MarkEmailVerified :exec
UPDATE accounts SET email_verified_at = sqlc.arg(verified_at)
WHERE id = sqlc.arg(id) AND email_verified_at IS NULL;
