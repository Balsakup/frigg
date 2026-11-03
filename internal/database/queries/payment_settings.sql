-- Payment methods shown to members (research.md 006 R7): single row id = 1.

-- name: GetPaymentSettings :one
SELECT * FROM payment_settings WHERE id = 1;

-- name: UpdatePaymentSettings :exec
UPDATE payment_settings
SET cash_enabled = sqlc.arg(cash_enabled),
    cash_instructions = sqlc.narg(cash_instructions),
    cheque_enabled = sqlc.arg(cheque_enabled),
    cheque_payee = sqlc.narg(cheque_payee),
    cheque_address = sqlc.narg(cheque_address),
    cheque_instructions = sqlc.narg(cheque_instructions),
    transfer_enabled = sqlc.arg(transfer_enabled),
    transfer_holder = sqlc.narg(transfer_holder),
    transfer_iban = sqlc.narg(transfer_iban),
    transfer_bic = sqlc.narg(transfer_bic),
    transfer_instructions = sqlc.narg(transfer_instructions),
    card_enabled = sqlc.arg(card_enabled),
    updated_at = sqlc.arg(now)
WHERE id = 1;

-- HelloAsso connection state (research.md 008 R7, R9).

-- name: RecordHelloAssoTest :exec
UPDATE payment_settings
SET helloasso_test_at = sqlc.arg(now), helloasso_test_environment = sqlc.arg(environment),
    helloasso_test_ok = sqlc.arg(ok), helloasso_test_error = sqlc.narg(error_code),
    helloasso_organization_name = sqlc.narg(organization_name),
    helloasso_error_at = CASE WHEN sqlc.arg(ok) = 1 THEN NULL ELSE helloasso_error_at END,
    helloasso_error_code = CASE WHEN sqlc.arg(ok) = 1 THEN NULL ELSE helloasso_error_code END
WHERE id = 1;

-- name: RecordHelloAssoError :exec
UPDATE payment_settings SET helloasso_error_at = sqlc.arg(now), helloasso_error_code = sqlc.arg(error_code)
WHERE id = 1;

-- name: ClearHelloAssoError :exec
UPDATE payment_settings SET helloasso_error_at = NULL, helloasso_error_code = NULL
WHERE id = 1 AND helloasso_error_at IS NOT NULL;
