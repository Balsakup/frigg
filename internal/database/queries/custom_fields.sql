-- Custom fields of member records and memberships (research.md 011).
-- Owner-space queries never load fields with visibility 'admin' (R3).

-- name: ListCustomFields :many
SELECT * FROM custom_fields
ORDER BY scope, archived_at IS NOT NULL, position, id;

-- name: ListCustomFieldsOfScope :many
SELECT * FROM custom_fields WHERE scope = sqlc.arg(scope) ORDER BY position, id;

-- name: ListActiveCustomFields :many
SELECT * FROM custom_fields
WHERE scope = sqlc.arg(scope) AND archived_at IS NULL
ORDER BY position, id;

-- name: ListOwnerCustomFields :many
SELECT * FROM custom_fields
WHERE scope = sqlc.arg(scope) AND archived_at IS NULL AND visibility <> 'admin'
ORDER BY position, id;

-- name: GetCustomField :one
SELECT * FROM custom_fields WHERE id = sqlc.arg(id);

-- name: CreateCustomField :one
INSERT INTO custom_fields (scope, label, type, visibility, required, help, position, created_at, updated_at)
VALUES (sqlc.arg(scope), sqlc.arg(label), sqlc.arg(type), sqlc.arg(visibility), sqlc.arg(required),
        sqlc.narg(help), sqlc.arg(position), sqlc.arg(now), sqlc.arg(now))
RETURNING *;

-- name: UpdateCustomField :execrows
UPDATE custom_fields
SET label = sqlc.arg(label), type = sqlc.arg(type), visibility = sqlc.arg(visibility),
    required = sqlc.arg(required), help = sqlc.narg(help), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: SetCustomFieldPosition :exec
UPDATE custom_fields SET position = sqlc.arg(position), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: ArchiveCustomField :execrows
UPDATE custom_fields SET archived_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND archived_at IS NULL;

-- name: RestoreCustomField :execrows
UPDATE custom_fields SET archived_at = NULL, position = sqlc.arg(position), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND archived_at IS NOT NULL;

-- name: DeleteCustomField :execrows
DELETE FROM custom_fields WHERE id = sqlc.arg(id);

-- name: MaxCustomFieldPosition :one
SELECT CAST(coalesce(max(position), 0) AS INTEGER) FROM custom_fields WHERE scope = sqlc.arg(scope);

-- name: CountActiveCustomFields :one
SELECT count(*) FROM custom_fields WHERE scope = sqlc.arg(scope) AND archived_at IS NULL;

-- name: ListCustomFieldChoices :many
SELECT * FROM custom_field_choices WHERE field_id = sqlc.arg(field_id) ORDER BY position, id;

-- name: ListAllCustomFieldChoices :many
SELECT * FROM custom_field_choices ORDER BY field_id, position, id;

-- name: CreateCustomFieldChoice :exec
INSERT INTO custom_field_choices (field_id, label, position)
VALUES (sqlc.arg(field_id), sqlc.arg(label), sqlc.arg(position));

-- name: UpdateCustomFieldChoice :exec
UPDATE custom_field_choices SET label = sqlc.arg(label), position = sqlc.arg(position)
WHERE id = sqlc.arg(id) AND field_id = sqlc.arg(field_id);

-- name: DeleteCustomFieldChoice :exec
DELETE FROM custom_field_choices WHERE id = sqlc.arg(id) AND field_id = sqlc.arg(field_id);

-- name: CountChoiceUses :one
SELECT count(*) FROM (
    SELECT mv.id FROM member_field_values mv WHERE mv.value_choice_id = sqlc.arg(choice_id)
    UNION ALL
    SELECT sv.id FROM membership_field_values sv WHERE sv.value_choice_id = sqlc.arg(choice_id)
);

-- name: CountCustomFieldValues :many
SELECT field_id, count(*) AS value_count
FROM (SELECT field_id FROM member_field_values
      UNION ALL
      SELECT field_id FROM membership_field_values)
GROUP BY field_id;

-- Required active fields and the number of records without a value (R10).

-- name: CountMembersMissingFields :many
SELECT cf.id AS field_id,
       (SELECT count(*) FROM members m
        WHERE NOT EXISTS (SELECT 1 FROM member_field_values v
                          WHERE v.member_id = m.id AND v.field_id = cf.id)) AS missing_count
FROM custom_fields cf
WHERE cf.scope = 'member' AND cf.required = 1 AND cf.archived_at IS NULL;

-- name: CountMembershipsMissingFields :many
SELECT cf.id AS field_id,
       (SELECT count(*) FROM memberships ms
        WHERE ms.season_id = sqlc.arg(season_id) AND ms.status IN ('pending', 'validated')
          AND NOT EXISTS (SELECT 1 FROM membership_field_values v
                          WHERE v.membership_id = ms.id AND v.field_id = cf.id)) AS missing_count
FROM custom_fields cf
WHERE cf.scope = 'membership' AND cf.required = 1 AND cf.archived_at IS NULL;

-- Values.

-- name: ListMemberFieldValues :many
SELECT * FROM member_field_values WHERE member_id = sqlc.arg(member_id);

-- name: UpsertMemberFieldValue :exec
INSERT INTO member_field_values (
    member_id, field_id, value_text, value_long, value_number, value_date, value_bool,
    value_choice_id, updated_at
) VALUES (
    sqlc.arg(owner_id), sqlc.arg(field_id), sqlc.narg(value_text), sqlc.narg(value_long),
    sqlc.narg(value_number), sqlc.narg(value_date), sqlc.narg(value_bool),
    sqlc.narg(value_choice_id), sqlc.arg(now)
)
ON CONFLICT (member_id, field_id) DO UPDATE SET
    value_text = excluded.value_text, value_long = excluded.value_long,
    value_number = excluded.value_number, value_date = excluded.value_date,
    value_bool = excluded.value_bool, value_choice_id = excluded.value_choice_id,
    updated_at = excluded.updated_at;

-- name: DeleteMemberFieldValue :exec
DELETE FROM member_field_values WHERE member_id = sqlc.arg(owner_id) AND field_id = sqlc.arg(field_id);

-- name: ListMembershipFieldValues :many
SELECT * FROM membership_field_values WHERE membership_id = sqlc.arg(membership_id);

-- name: UpsertMembershipFieldValue :exec
INSERT INTO membership_field_values (
    membership_id, field_id, value_text, value_long, value_number, value_date, value_bool,
    value_choice_id, updated_at
) VALUES (
    sqlc.arg(owner_id), sqlc.arg(field_id), sqlc.narg(value_text), sqlc.narg(value_long),
    sqlc.narg(value_number), sqlc.narg(value_date), sqlc.narg(value_bool),
    sqlc.narg(value_choice_id), sqlc.arg(now)
)
ON CONFLICT (membership_id, field_id) DO UPDATE SET
    value_text = excluded.value_text, value_long = excluded.value_long,
    value_number = excluded.value_number, value_date = excluded.value_date,
    value_bool = excluded.value_bool, value_choice_id = excluded.value_choice_id,
    updated_at = excluded.updated_at;

-- name: DeleteMembershipFieldValue :exec
DELETE FROM membership_field_values WHERE membership_id = sqlc.arg(owner_id) AND field_id = sqlc.arg(field_id);
