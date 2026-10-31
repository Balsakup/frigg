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

-- name: ListAccounts :many
SELECT a.id, a.email, a.role, a.email_verified_at, a.disabled_at, a.created_at, a.last_login_at
FROM accounts a
WHERE (CAST(sqlc.narg(query) AS TEXT) IS NULL
       OR a.id IN (SELECT rowid FROM accounts_fts WHERE accounts_fts MATCH CAST(sqlc.narg(query) AS TEXT)))
  AND (CAST(sqlc.narg(role) AS TEXT) IS NULL OR a.role = CAST(sqlc.narg(role) AS TEXT))
  AND (CAST(sqlc.narg(disabled) AS INTEGER) IS NULL
       OR (a.disabled_at IS NOT NULL) = CAST(sqlc.narg(disabled) AS INTEGER))
ORDER BY a.email, a.id
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: CountAccounts :one
SELECT count(*)
FROM accounts a
WHERE (CAST(sqlc.narg(query) AS TEXT) IS NULL
       OR a.id IN (SELECT rowid FROM accounts_fts WHERE accounts_fts MATCH CAST(sqlc.narg(query) AS TEXT)))
  AND (CAST(sqlc.narg(role) AS TEXT) IS NULL OR a.role = CAST(sqlc.narg(role) AS TEXT))
  AND (CAST(sqlc.narg(disabled) AS INTEGER) IS NULL
       OR (a.disabled_at IS NOT NULL) = CAST(sqlc.narg(disabled) AS INTEGER));

-- name: DisableAccount :execrows
UPDATE accounts
SET disabled_at = sqlc.arg(now), auth_epoch = auth_epoch + 1
WHERE accounts.id = sqlc.arg(id)
  AND accounts.disabled_at IS NULL
  AND (accounts.role <> 'admin'
       OR EXISTS (SELECT 1 FROM accounts o WHERE o.role = 'admin' AND o.disabled_at IS NULL AND o.id <> sqlc.arg(id)));

-- name: EnableAccount :execrows
UPDATE accounts SET disabled_at = NULL
WHERE id = sqlc.arg(id) AND disabled_at IS NOT NULL;

-- name: SetAccountRole :execrows
UPDATE accounts SET role = sqlc.arg(role)
WHERE accounts.id = sqlc.arg(id)
  AND accounts.role <> sqlc.arg(role)
  AND (sqlc.arg(role) = 'admin'
       OR accounts.disabled_at IS NOT NULL
       OR EXISTS (SELECT 1 FROM accounts o WHERE o.role = 'admin' AND o.disabled_at IS NULL AND o.id <> sqlc.arg(id)));

-- name: DeleteAccount :execrows
DELETE FROM accounts
WHERE accounts.id = sqlc.arg(id)
  AND (accounts.role <> 'admin'
       OR accounts.disabled_at IS NOT NULL
       OR EXISTS (SELECT 1 FROM accounts o WHERE o.role = 'admin' AND o.disabled_at IS NULL AND o.id <> sqlc.arg(id)));
