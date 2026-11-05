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
       m.created_at, m.updated_at,
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
       m.created_at, m.updated_at,
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

-- name: ListMemberIDsOfAccount :many
SELECT id FROM members WHERE account_id = sqlc.arg(account_id) ORDER BY id;

-- name: DeleteMembersOfAccount :execrows
DELETE FROM members WHERE account_id = sqlc.arg(account_id);

-- name: DetachMembersOfAccount :execrows
UPDATE members SET account_id = NULL, is_self = 0, updated_at = sqlc.arg(now)
WHERE account_id = sqlc.arg(account_id);
