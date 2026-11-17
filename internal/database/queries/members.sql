-- Member records (research.md 005). Member-space queries are always scoped by
-- account_id (R4); admin queries live below.

-- name: ListMembersOfAccount :many
SELECT * FROM members WHERE account_id = sqlc.arg(account_id);

-- name: GetMemberOfAccount :one
SELECT * FROM members WHERE id = sqlc.arg(id) AND account_id = sqlc.arg(account_id);

-- name: CreateMemberForAccount :one
INSERT INTO members (
    account_id, is_self, first_name, last_name, birth_date, email, phone,
    address_line1, address_line2, postal_code, city, created_at, updated_at
) VALUES (
    sqlc.arg(account_id), sqlc.arg(is_self), sqlc.arg(first_name), sqlc.arg(last_name),
    sqlc.arg(birth_date), sqlc.narg(email), sqlc.narg(phone), sqlc.narg(address_line1),
    sqlc.narg(address_line2), sqlc.narg(postal_code), sqlc.narg(city), sqlc.arg(now), sqlc.arg(now)
)
RETURNING *;

-- name: UpdateMemberOfAccount :execrows
UPDATE members
SET is_self = sqlc.arg(is_self),
    first_name = sqlc.arg(first_name),
    last_name = sqlc.arg(last_name),
    birth_date = sqlc.arg(birth_date),
    email = sqlc.narg(email),
    phone = sqlc.narg(phone),
    address_line1 = sqlc.narg(address_line1),
    address_line2 = sqlc.narg(address_line2),
    postal_code = sqlc.narg(postal_code),
    city = sqlc.narg(city),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND account_id = sqlc.arg(account_id);

-- name: DeleteMemberOfAccount :execrows
DELETE FROM members WHERE id = sqlc.arg(id) AND account_id = sqlc.arg(account_id);

-- Admin queries. ListMembers returns every matching row: sorting without
-- accents and paging are done in Go (R7). The text query also matches short
-- text custom field values of active fields; field_filters is a JSON list of
-- custom field filters read with json_each (research.md 011 R5, R6).

-- name: ListMembers :many
SELECT m.id, m.account_id, m.is_self, m.first_name, m.last_name, m.birth_date,
       m.email, m.phone, m.address_line1, m.address_line2, m.postal_code, m.city,
       m.created_at, m.updated_at, m.origin, m.email_confirmed_at, m.consented_at,
       a.email AS account_email, a.disabled_at AS account_disabled_at
FROM members m
LEFT JOIN accounts a ON a.id = m.account_id
WHERE (CAST(sqlc.narg(query) AS TEXT) IS NULL
       OR m.id IN (SELECT rowid FROM members_fts WHERE members_fts MATCH CAST(sqlc.narg(query) AS TEXT))
       OR m.id IN (SELECT v.member_id FROM member_field_values v
                   JOIN custom_fields cf ON cf.id = v.field_id AND cf.archived_at IS NULL
                   WHERE v.id IN (SELECT rowid FROM member_field_values_fts
                                  WHERE member_field_values_fts MATCH CAST(sqlc.narg(query) AS TEXT))))
  AND (CAST(sqlc.narg(no_account) AS INTEGER) IS NULL OR m.account_id IS NULL)
  AND (CAST(sqlc.narg(account_id) AS INTEGER) IS NULL OR m.account_id = CAST(sqlc.narg(account_id) AS INTEGER))
  AND (CAST(sqlc.narg(origin) AS TEXT) IS NULL OR m.origin = CAST(sqlc.narg(origin) AS TEXT))
  AND NOT EXISTS (
      SELECT 1 FROM json_each(coalesce(CAST(sqlc.narg(field_filters) AS TEXT), '[]')) ff
      WHERE CASE json_extract(ff.value, '$.op')
          WHEN 'true' THEN NOT EXISTS (SELECT 1 FROM member_field_values fv
              WHERE fv.member_id = m.id AND fv.field_id = json_extract(ff.value, '$.field') AND fv.value_bool = 1)
          WHEN 'false' THEN EXISTS (SELECT 1 FROM member_field_values fv
              WHERE fv.member_id = m.id AND fv.field_id = json_extract(ff.value, '$.field') AND fv.value_bool = 1)
          WHEN 'empty' THEN EXISTS (SELECT 1 FROM member_field_values fv
              WHERE fv.member_id = m.id AND fv.field_id = json_extract(ff.value, '$.field'))
          WHEN 'choice' THEN NOT EXISTS (SELECT 1 FROM member_field_values fv
              WHERE fv.member_id = m.id AND fv.field_id = json_extract(ff.value, '$.field')
                AND fv.value_choice_id = json_extract(ff.value, '$.choice'))
          ELSE 1 END);

-- name: GetMember :one
SELECT m.id, m.account_id, m.is_self, m.first_name, m.last_name, m.birth_date,
       m.email, m.phone, m.address_line1, m.address_line2, m.postal_code, m.city,
       m.created_at, m.updated_at, m.origin, m.email_confirmed_at, m.consented_at,
       a.email AS account_email, a.disabled_at AS account_disabled_at
FROM members m
LEFT JOIN accounts a ON a.id = m.account_id
WHERE m.id = sqlc.arg(id);

-- name: CreateMember :one
INSERT INTO members (
    account_id, is_self, first_name, last_name, birth_date, email, phone,
    address_line1, address_line2, postal_code, city, created_at, updated_at
) VALUES (
    sqlc.narg(account_id), 0, sqlc.arg(first_name), sqlc.arg(last_name),
    sqlc.arg(birth_date), sqlc.narg(email), sqlc.narg(phone), sqlc.narg(address_line1),
    sqlc.narg(address_line2), sqlc.narg(postal_code), sqlc.narg(city), sqlc.arg(now), sqlc.arg(now)
)
RETURNING id;

-- name: UpdateMember :execrows
UPDATE members
SET account_id = sqlc.narg(account_id),
    is_self = sqlc.arg(is_self),
    first_name = sqlc.arg(first_name),
    last_name = sqlc.arg(last_name),
    birth_date = sqlc.arg(birth_date),
    email = sqlc.narg(email),
    phone = sqlc.narg(phone),
    address_line1 = sqlc.narg(address_line1),
    address_line2 = sqlc.narg(address_line2),
    postal_code = sqlc.narg(postal_code),
    city = sqlc.narg(city),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: DeleteMember :execrows
DELETE FROM members WHERE id = sqlc.arg(id);

-- Shared queries.

-- name: CountMembersOfAccount :one
SELECT count(*) FROM members WHERE account_id = sqlc.arg(account_id);

-- name: ListMembersByBirthDate :many
SELECT id, first_name, last_name FROM members WHERE birth_date = sqlc.arg(birth_date);

-- name: ListMemberIdentities :many
-- Identites de toutes les fiches, pour la detection des doublons a l'import.
SELECT id, first_name, last_name, birth_date FROM members;

-- name: ListMemberIDsOfAccount :many
SELECT id FROM members WHERE account_id = sqlc.arg(account_id) ORDER BY id;

-- name: DeleteMembersOfAccount :execrows
DELETE FROM members WHERE account_id = sqlc.arg(account_id);

-- name: DetachMembersOfAccount :execrows
UPDATE members SET account_id = NULL, is_self = 0, updated_at = sqlc.arg(now)
WHERE account_id = sqlc.arg(account_id);

-- Members tracked by a personal link (research.md 016 R1): members without an
-- account whose email matches the link, compared without case (FR-014).

-- name: ListTrackedMembers :many
SELECT * FROM members
WHERE account_id IS NULL AND email = sqlc.arg(email) COLLATE NOCASE
ORDER BY last_name, first_name, id;

-- name: GetTrackedMember :one
SELECT * FROM members
WHERE id = sqlc.arg(id) AND account_id IS NULL AND email = sqlc.arg(email) COLLATE NOCASE;

-- name: CountTrackedMembers :one
SELECT count(*) FROM members
WHERE account_id IS NULL AND email = sqlc.arg(email) COLLATE NOCASE;

-- name: FindTrackedMemberByIdentity :many
-- Candidates for deduplication; names are compared in Go (member.IdentityKey).
SELECT id, first_name, last_name FROM members
WHERE account_id IS NULL AND email = sqlc.arg(email) COLLATE NOCASE
  AND birth_date = sqlc.arg(birth_date)
ORDER BY id;

-- name: FindAccountMemberByIdentity :many
-- Candidates for deduplication; names are compared in Go (member.IdentityKey).
SELECT id, first_name, last_name FROM members
WHERE account_id = sqlc.arg(account_id) AND birth_date = sqlc.arg(birth_date)
ORDER BY id;

-- name: CreatePublicMember :one
INSERT INTO members (
    account_id, is_self, first_name, last_name, birth_date, email, phone,
    address_line1, address_line2, postal_code, city, origin,
    email_confirmed_at, consented_at, created_at, updated_at
) VALUES (
    sqlc.narg(account_id), 0, sqlc.arg(first_name), sqlc.arg(last_name),
    sqlc.arg(birth_date), sqlc.arg(email), sqlc.narg(phone), sqlc.narg(address_line1),
    sqlc.narg(address_line2), sqlc.narg(postal_code), sqlc.narg(city), 'public',
    sqlc.arg(now), sqlc.arg(consented_at), sqlc.arg(now), sqlc.arg(now)
)
RETURNING id;

-- name: ResetMemberEmailConfirmation :exec
UPDATE members SET email_confirmed_at = NULL WHERE id = sqlc.arg(id);

-- name: CopyMemberEmailIfEmpty :exec
-- Attaching a public request (research.md 016 R13): the confirmed email of
-- the source member is copied to a target member without one.
UPDATE members
SET email = (SELECT s.email FROM members s WHERE s.id = sqlc.arg(source_id)),
    email_confirmed_at = (SELECT s.email_confirmed_at FROM members s WHERE s.id = sqlc.arg(source_id)),
    updated_at = sqlc.arg(now)
WHERE members.id = sqlc.arg(target_id) AND members.email IS NULL
  AND (SELECT s.email FROM members s WHERE s.id = sqlc.arg(source_id)) IS NOT NULL;

-- name: CopyMissingMemberFieldValues :execrows
-- Copies the custom field values of the source member that the target lacks.
INSERT INTO member_field_values (
    member_id, field_id, value_text, value_long, value_number, value_date,
    value_bool, value_choice_id, updated_at
)
SELECT sqlc.arg(target_id), v.field_id, v.value_text, v.value_long, v.value_number, v.value_date,
       v.value_bool, v.value_choice_id, sqlc.arg(now)
FROM member_field_values v
WHERE v.member_id = sqlc.arg(source_id)
  AND v.field_id NOT IN (SELECT t.field_id FROM member_field_values t WHERE t.member_id = sqlc.arg(target_id));

-- name: DeleteMemberIfEmpty :execrows
-- Deletes a member left without any membership or event registration.
DELETE FROM members
WHERE members.id = sqlc.arg(member_id)
  AND NOT EXISTS (SELECT 1 FROM memberships ms WHERE ms.member_id = sqlc.arg(member_id))
  AND NOT EXISTS (SELECT 1 FROM event_registrations er WHERE er.member_id = sqlc.arg(member_id));

-- Account created from a personal link (research.md 016 R12): the tracked
-- members of the email are attached to the new account.

-- name: AttachTrackedMembers :execrows
UPDATE members SET account_id = sqlc.arg(account_id), is_self = 0, updated_at = sqlc.arg(now)
WHERE account_id IS NULL AND email = sqlc.arg(email) COLLATE NOCASE;

-- name: SetSelfMemberOfAccount :execrows
UPDATE members SET is_self = 1, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND account_id = sqlc.arg(account_id);

-- Inactive public members (research.md 016 R14, FR-041): origin public,
-- no account, created before the cutoff, no pending membership, no validated
-- membership of a season ending on or after the cutoff day, no confirmed
-- registration to an event ending at or after the cutoff instant. cutoff_on
-- is the cutoff day (YYYY-MM-DD), cutoff_at the same day at midnight in
-- Paris, in UTC. Mirrored by public.Inactive.

-- name: ListInactivePublicMembers :many
SELECT m.id FROM members m
WHERE m.origin = 'public' AND m.account_id IS NULL
  AND m.created_at < sqlc.arg(cutoff_at)
  AND NOT EXISTS (SELECT 1 FROM memberships ms
                  WHERE ms.member_id = m.id AND ms.status = 'pending')
  AND NOT EXISTS (SELECT 1 FROM memberships ms JOIN seasons s ON s.id = ms.season_id
                  WHERE ms.member_id = m.id AND ms.status = 'validated'
                    AND s.ends_on >= sqlc.arg(cutoff_on))
  AND NOT EXISTS (SELECT 1 FROM event_registrations r JOIN events e ON e.id = r.event_id
                  WHERE r.member_id = m.id AND r.status = 'confirmed'
                    AND e.ends_at >= sqlc.arg(cutoff_at))
ORDER BY m.id;
