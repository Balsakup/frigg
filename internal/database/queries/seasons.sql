-- name: ListSeasons :many
-- Full ordered sequence: the domain rules need every season.
SELECT * FROM seasons ORDER BY starts_on;

-- name: CreateSeason :one
INSERT INTO seasons (starts_on, ends_on, created_at)
VALUES (sqlc.arg(starts_on), sqlc.arg(ends_on), sqlc.arg(created_at))
RETURNING *;

-- name: UpdateSeasonDates :exec
UPDATE seasons SET starts_on = sqlc.arg(starts_on), ends_on = sqlc.arg(ends_on)
WHERE id = sqlc.arg(id);

-- name: OpenSeason :execrows
UPDATE seasons SET opened_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND opened_at IS NULL;

-- name: DeleteSeason :execrows
DELETE FROM seasons WHERE id = sqlc.arg(id);
