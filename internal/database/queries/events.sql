-- Events (research.md 012 R1, R9). Instants are UTC text, compared as text.
-- Lists are small (a few hundred events): filtering by phase is done in Go.

-- name: CreateEvent :one
INSERT INTO events (
    season_id, title, location, description, starts_at, ends_at, openness, capacity, price_cents,
    registration_opens_at, registration_closes_at, status, created_by, updated_by, created_at, updated_at
) VALUES (
    sqlc.arg(season_id), sqlc.arg(title), sqlc.arg(location), sqlc.narg(description), sqlc.arg(starts_at),
    sqlc.arg(ends_at), sqlc.arg(openness), sqlc.narg(capacity), sqlc.arg(price_cents),
    sqlc.narg(registration_opens_at), sqlc.narg(registration_closes_at), 'draft', sqlc.narg(actor_id),
    sqlc.narg(actor_id), sqlc.arg(now), sqlc.arg(now)
)
RETURNING *;

-- name: UpdateEvent :execrows
UPDATE events
SET season_id = sqlc.arg(season_id), title = sqlc.arg(title), location = sqlc.arg(location),
    description = sqlc.narg(description), starts_at = sqlc.arg(starts_at), ends_at = sqlc.arg(ends_at),
    openness = sqlc.arg(openness), capacity = sqlc.narg(capacity), price_cents = sqlc.arg(price_cents),
    registration_opens_at = sqlc.narg(registration_opens_at),
    registration_closes_at = sqlc.narg(registration_closes_at),
    updated_by = sqlc.narg(actor_id), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status <> 'cancelled';

-- name: GetEvent :one
SELECT sqlc.embed(e),
       (SELECT count(*) FROM event_registrations r WHERE r.event_id = e.id AND r.status = 'confirmed') AS confirmed_count,
       (SELECT count(*) FROM event_registrations r WHERE r.event_id = e.id) AS registration_count,
       cb.email AS created_by_email, ub.email AS updated_by_email, xb.email AS cancelled_by_email
FROM events e
LEFT JOIN accounts cb ON cb.id = e.created_by
LEFT JOIN accounts ub ON ub.id = e.updated_by
LEFT JOIN accounts xb ON xb.id = e.cancelled_by
WHERE e.id = sqlc.arg(id);

-- name: ListEvents :many
SELECT sqlc.embed(e),
       (SELECT count(*) FROM event_registrations r WHERE r.event_id = e.id AND r.status = 'confirmed') AS confirmed_count
FROM events e
WHERE (CAST(sqlc.narg(season_id) AS INTEGER) IS NULL OR e.season_id = CAST(sqlc.narg(season_id) AS INTEGER))
ORDER BY e.starts_at, e.id;

-- Member space: drafts are never returned (FR-009).

-- name: ListMemberEvents :many
SELECT sqlc.embed(e),
       (SELECT count(*) FROM event_registrations r WHERE r.event_id = e.id AND r.status = 'confirmed') AS confirmed_count
FROM events e
WHERE e.status IN ('published', 'cancelled') AND e.ends_at > sqlc.arg(now)
ORDER BY e.starts_at, e.id;

-- name: GetMemberEvent :one
SELECT sqlc.embed(e),
       (SELECT count(*) FROM event_registrations r WHERE r.event_id = e.id AND r.status = 'confirmed') AS confirmed_count
FROM events e
WHERE e.id = sqlc.arg(id) AND e.status <> 'draft';

-- name: PublishEvent :execrows
UPDATE events SET status = 'published', published_at = coalesce(published_at, sqlc.arg(now)),
    updated_by = sqlc.narg(actor_id), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'draft';

-- name: CancelEvent :execrows
UPDATE events SET status = 'cancelled', cancel_message = sqlc.narg(message), cancelled_at = sqlc.arg(now),
    cancelled_by = sqlc.narg(actor_id), updated_by = sqlc.narg(actor_id), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'published';

-- name: RestoreEvent :execrows
UPDATE events SET status = 'published', cancel_message = NULL, cancelled_at = NULL, cancelled_by = NULL,
    updated_by = sqlc.narg(actor_id), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'cancelled';

-- name: DeleteEventWithoutRegistrations :execrows
DELETE FROM events
WHERE events.id = sqlc.arg(id)
  AND NOT EXISTS (SELECT 1 FROM event_registrations r WHERE r.event_id = events.id);

-- Season checks (R8): bounds are the UTC instants of local midnight of the
-- first day and of the day after the last day, computed in Go.

-- name: CountEventsOutsideSeason :one
SELECT count(*) FROM events
WHERE season_id = sqlc.arg(season_id)
  AND (starts_at < sqlc.arg(from_instant) OR starts_at >= sqlc.arg(to_instant));

-- name: CountEventsOfSeason :one
SELECT count(*) FROM events WHERE season_id = sqlc.arg(season_id);
