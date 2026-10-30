-- name: GetAssociation :one
SELECT name FROM association WHERE id = 1;

-- name: SetAssociationName :exec
UPDATE association SET name = sqlc.arg(name), updated_at = sqlc.arg(updated_at) WHERE id = 1;
