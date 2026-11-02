-- Health check: proves the read pool can run a query.
-- name: HealthCheck :one
SELECT 1;
