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
    updated_at = sqlc.arg(now)
WHERE id = 1;
