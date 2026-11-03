-- name: GetAssociation :one
SELECT * FROM association WHERE id = 1;

-- name: SetAssociationName :exec
UPDATE association SET name = sqlc.arg(name), updated_at = sqlc.arg(updated_at) WHERE id = 1;

-- name: UpdateGeneralSettings :exec
UPDATE association SET
    name = sqlc.arg(name),
    tagline = sqlc.narg(tagline),
    short_name = sqlc.narg(short_name),
    purpose = sqlc.narg(purpose),
    rna = sqlc.narg(rna),
    siret = sqlc.narg(siret),
    founded_on = sqlc.narg(founded_on),
    address_line1 = sqlc.narg(address_line1),
    address_line2 = sqlc.narg(address_line2),
    postal_code = sqlc.narg(postal_code),
    city = sqlc.narg(city),
    contact_email = sqlc.narg(contact_email),
    phone = sqlc.narg(phone),
    website = sqlc.narg(website),
    facebook_url = sqlc.narg(facebook_url),
    instagram_url = sqlc.narg(instagram_url),
    linkedin_url = sqlc.narg(linkedin_url),
    youtube_url = sqlc.narg(youtube_url),
    mastodon_url = sqlc.narg(mastodon_url),
    bluesky_url = sqlc.narg(bluesky_url),
    updated_at = sqlc.arg(updated_at)
WHERE id = 1;

-- name: UpdateAssociationColors :exec
UPDATE association SET
    primary_color = sqlc.narg(primary_color),
    accent_color = sqlc.narg(accent_color),
    updated_at = sqlc.arg(updated_at)
WHERE id = 1;

-- name: GetAssociationLogo :one
SELECT * FROM association_logo WHERE id = 1;

-- name: UpsertAssociationLogo :exec
INSERT INTO association_logo (id, original_name, source_type, source_size_bytes,
    web_name, web_type, web_size_bytes, web_width, web_height, web_sha256,
    email_name, email_type, email_size_bytes, email_width, email_height, email_sha256,
    uploaded_at, uploaded_by)
VALUES (1, sqlc.arg(original_name), sqlc.arg(source_type), sqlc.arg(source_size_bytes),
    sqlc.arg(web_name), sqlc.arg(web_type), sqlc.arg(web_size_bytes), sqlc.arg(web_width), sqlc.arg(web_height), sqlc.arg(web_sha256),
    sqlc.arg(email_name), sqlc.arg(email_type), sqlc.arg(email_size_bytes), sqlc.arg(email_width), sqlc.arg(email_height), sqlc.arg(email_sha256),
    sqlc.arg(uploaded_at), sqlc.narg(uploaded_by))
ON CONFLICT (id) DO UPDATE SET
    original_name = excluded.original_name, source_type = excluded.source_type, source_size_bytes = excluded.source_size_bytes,
    web_name = excluded.web_name, web_type = excluded.web_type, web_size_bytes = excluded.web_size_bytes,
    web_width = excluded.web_width, web_height = excluded.web_height, web_sha256 = excluded.web_sha256,
    email_name = excluded.email_name, email_type = excluded.email_type, email_size_bytes = excluded.email_size_bytes,
    email_width = excluded.email_width, email_height = excluded.email_height, email_sha256 = excluded.email_sha256,
    uploaded_at = excluded.uploaded_at, uploaded_by = excluded.uploaded_by;

-- name: DeleteAssociationLogo :exec
DELETE FROM association_logo WHERE id = 1;
