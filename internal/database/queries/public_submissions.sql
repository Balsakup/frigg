-- Unconfirmed public submissions (research.md 016 R3). A submission has no
-- business existence: no member, membership or registration query reads it.

-- name: CreatePublicSubmission :one
INSERT INTO public_submissions (
    kind, season_id, fee_id, declared_kind, event_id,
    first_name, last_name, birth_date, email, phone,
    address_line1, address_line2, postal_code, city,
    custom_values, create_account, consented_at, token_hash, expires_at, created_at
) VALUES (
    sqlc.arg(kind), sqlc.narg(season_id), sqlc.narg(fee_id), sqlc.narg(declared_kind), sqlc.narg(event_id),
    sqlc.arg(first_name), sqlc.arg(last_name), sqlc.arg(birth_date), sqlc.arg(email), sqlc.narg(phone),
    sqlc.narg(address_line1), sqlc.narg(address_line2), sqlc.narg(postal_code), sqlc.narg(city),
    sqlc.arg(custom_values), sqlc.arg(create_account), sqlc.arg(consented_at), sqlc.arg(token_hash),
    sqlc.arg(expires_at), sqlc.arg(created_at)
)
RETURNING id;

-- name: GetPublicSubmissionByTokenHash :one
-- Expiry is checked by the caller: an expired submission gets the same
-- answer as an unknown one.
SELECT * FROM public_submissions WHERE token_hash = sqlc.arg(token_hash);

-- name: DeletePublicSubmission :exec
DELETE FROM public_submissions WHERE id = sqlc.arg(id);

-- name: PurgePublicSubmissions :execrows
DELETE FROM public_submissions WHERE created_at < sqlc.arg(before);

-- name: CountMembershipFeeForPublic :one
-- 1 when the fee belongs to the season, 0 otherwise.
SELECT count(*) FROM membership_fees
WHERE id = sqlc.arg(fee_id) AND season_id = sqlc.arg(season_id);
