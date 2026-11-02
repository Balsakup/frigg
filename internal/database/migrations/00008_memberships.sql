-- +goose Up
-- Adhésions d'une fiche membre à une saison, avec leur paiement
-- (research.md 006 › R1). Montants en centimes d'euro.
-- Les CHECK de cohérence sont des filets derrière le domaine internal/membership.
CREATE TABLE memberships (
    id INTEGER PRIMARY KEY,
    member_id INTEGER NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    season_id INTEGER NOT NULL REFERENCES seasons (id) ON DELETE RESTRICT,
    fee_id INTEGER NOT NULL REFERENCES membership_fees (id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('pending', 'validated', 'refused', 'cancelled')),
    -- Type déclaré par le compte ; NULL si l'adhésion a été créée par un administrateur.
    declared_kind TEXT CHECK (declared_kind IN ('first', 'renewal')),
    -- Type retenu et montant appliqué, figés à la validation.
    kind TEXT CHECK (kind IN ('first', 'renewal')),
    amount_cents INTEGER CHECK (amount_cents BETWEEN 0 AND 1000000),
    reason TEXT CHECK (reason IS NULL OR length(reason) BETWEEN 1 AND 500),
    requested_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    created_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    requested_at TEXT NOT NULL,
    decided_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    decided_at TEXT,
    cancelled_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    cancelled_at TEXT,
    -- Référence de paiement ADH-AAAA-NNNN (R4) ; jamais modifiée une fois attribuée.
    payment_reference TEXT UNIQUE CHECK (payment_reference IS NULL OR length(payment_reference) BETWEEN 8 AND 16),
    -- NULL = pas de paiement (adhésion non validée ou gratuite).
    payment_status TEXT CHECK (payment_status IN ('due', 'announced', 'paid')),
    -- La carte (HelloAsso, 008) ne s'annonce pas : elle se paie en ligne.
    announced_method TEXT CHECK (announced_method IN ('cash', 'cheque', 'transfer')),
    paid_method TEXT CHECK (paid_method IN ('cash', 'cheque', 'transfer', 'card')),
    paid_on TEXT CHECK (paid_on IS NULL OR date(paid_on) = paid_on),
    paid_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    paid_at TEXT,
    payment_note TEXT CHECK (payment_note IS NULL OR length(payment_note) BETWEEN 1 AND 200),
    updated_at TEXT NOT NULL,
    -- Tentative HelloAsso qui a constate un paiement par carte (008).
    paid_online_payment_id INTEGER REFERENCES online_payments (id) ON DELETE SET NULL,
    -- Montant recu par carte, hors contribution volontaire (008).
    paid_amount_cents INTEGER CHECK (paid_amount_cents BETWEEN 0 AND 1000000),
    CHECK (status <> 'pending' OR (kind IS NULL AND amount_cents IS NULL)),
    CHECK (status <> 'validated' OR (kind IS NOT NULL AND amount_cents IS NOT NULL)),
    -- Une adhésion annulée garde l'état de son paiement pour l'historique.
    CHECK (payment_status IS NULL OR (status IN ('validated', 'cancelled') AND amount_cents > 0 AND payment_reference IS NOT NULL)),
    CHECK (status <> 'validated' OR amount_cents = 0 OR payment_status IS NOT NULL),
    CHECK ((payment_status IS 'paid') = (paid_method IS NOT NULL AND paid_on IS NOT NULL)),
    CHECK (payment_status IS NOT 'announced' OR announced_method IS NOT NULL),
    CHECK (status <> 'refused' OR reason IS NOT NULL),
    CHECK (requested_by IS NULL OR declared_kind IS NOT NULL),
    -- Seule la verification HelloAsso constate un paiement par carte (008).
    CHECK (paid_method IS NOT 'card' OR (paid_online_payment_id IS NOT NULL AND paid_amount_cents IS NOT NULL AND paid_by IS NULL)),
    CHECK (paid_online_payment_id IS NULL OR paid_method = 'card')
) STRICT;

-- Une adhésion en cours (en attente ou validée) au plus par fiche et par saison (FR-003).
CREATE UNIQUE INDEX memberships_one_active ON memberships (member_id, season_id)
WHERE status IN ('pending', 'validated');
CREATE INDEX memberships_season_status ON memberships (season_id, status);
CREATE INDEX memberships_member_id ON memberships (member_id);
CREATE INDEX memberships_fee_id ON memberships (fee_id);
CREATE INDEX memberships_status_requested ON memberships (status, requested_at);

-- Recherche par fragment de référence de paiement, sans casse (R10).
CREATE VIRTUAL TABLE memberships_fts USING fts5(
    payment_reference,
    content = 'memberships',
    content_rowid = 'id',
    tokenize = "trigram remove_diacritics 1"
);

-- +goose StatementBegin
CREATE TRIGGER memberships_fts_ai AFTER INSERT ON memberships BEGIN
    INSERT INTO memberships_fts (rowid, payment_reference) VALUES (new.id, new.payment_reference);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER memberships_fts_ad AFTER DELETE ON memberships BEGIN
    INSERT INTO memberships_fts (memberships_fts, rowid, payment_reference)
    VALUES ('delete', old.id, old.payment_reference);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER memberships_fts_au AFTER UPDATE OF payment_reference ON memberships BEGIN
    INSERT INTO memberships_fts (memberships_fts, rowid, payment_reference)
    VALUES ('delete', old.id, old.payment_reference);
    INSERT INTO memberships_fts (rowid, payment_reference) VALUES (new.id, new.payment_reference);
END;
-- +goose StatementEnd

-- Moyens de paiement proposés aux adhérents : ligne unique de l'instance (R7).
CREATE TABLE payment_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    cash_enabled INTEGER NOT NULL DEFAULT 0 CHECK (cash_enabled IN (0, 1)),
    cash_instructions TEXT CHECK (cash_instructions IS NULL OR length(cash_instructions) BETWEEN 1 AND 1000),
    cheque_enabled INTEGER NOT NULL DEFAULT 0 CHECK (cheque_enabled IN (0, 1)),
    cheque_payee TEXT CHECK (cheque_payee IS NULL OR length(cheque_payee) BETWEEN 1 AND 120),
    cheque_address TEXT CHECK (cheque_address IS NULL OR length(cheque_address) BETWEEN 1 AND 300),
    cheque_instructions TEXT CHECK (cheque_instructions IS NULL OR length(cheque_instructions) BETWEEN 1 AND 1000),
    transfer_enabled INTEGER NOT NULL DEFAULT 0 CHECK (transfer_enabled IN (0, 1)),
    transfer_holder TEXT CHECK (transfer_holder IS NULL OR length(transfer_holder) BETWEEN 1 AND 120),
    transfer_iban TEXT CHECK (transfer_iban IS NULL OR (length(transfer_iban) BETWEEN 15 AND 34 AND transfer_iban GLOB '[A-Z][A-Z][0-9][0-9]*')),
    transfer_bic TEXT CHECK (transfer_bic IS NULL OR length(transfer_bic) IN (8, 11)),
    transfer_instructions TEXT CHECK (transfer_instructions IS NULL OR length(transfer_instructions) BETWEEN 1 AND 1000),
    updated_at TEXT NOT NULL,
    -- Carte bancaire via HelloAsso et etat de la connexion (research.md 008 R7, R9).
    -- Les identifiants sont dans la configuration de l'instance, jamais en base.
    card_enabled INTEGER NOT NULL DEFAULT 0 CHECK (card_enabled IN (0, 1)),
    helloasso_test_at TEXT,
    helloasso_test_environment TEXT CHECK (helloasso_test_environment IN ('sandbox', 'production')),
    helloasso_test_ok INTEGER CHECK (helloasso_test_ok IN (0, 1)),
    helloasso_test_error TEXT CHECK (length(helloasso_test_error) <= 60),
    helloasso_organization_name TEXT CHECK (length(helloasso_organization_name) <= 250),
    helloasso_error_at TEXT,
    helloasso_error_code TEXT CHECK (length(helloasso_error_code) <= 60),
    CHECK (cash_enabled = 0 OR cash_instructions IS NOT NULL),
    CHECK (cheque_enabled = 0 OR cheque_payee IS NOT NULL),
    CHECK (transfer_enabled = 0 OR (transfer_holder IS NOT NULL AND transfer_iban IS NOT NULL))
) STRICT;

INSERT INTO payment_settings (id, updated_at)
VALUES (1, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'));

-- Tentatives de paiement par carte via HelloAsso (research.md 008 R4).
CREATE TABLE online_payments (
    id INTEGER PRIMARY KEY,
    membership_id INTEGER NOT NULL REFERENCES memberships (id) ON DELETE CASCADE,
    account_id INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    environment TEXT NOT NULL CHECK (environment IN ('sandbox', 'production')),
    amount_cents INTEGER NOT NULL CHECK (amount_cents BETWEEN 1 AND 1000000),
    -- refunded : paiement rembourse dans HelloAsso, constate apres verification.
    status TEXT NOT NULL CHECK (status IN ('pending', 'succeeded', 'failed', 'abandoned', 'expired', 'refunded')),
    checkout_intent_id INTEGER,
    redirect_url TEXT CHECK (redirect_url IS NULL OR length(redirect_url) <= 2000),
    order_id INTEGER,
    helloasso_payment_id INTEGER,
    received_amount_cents INTEGER CHECK (received_amount_cents BETWEEN 0 AND 1000000),
    paid_at TEXT,
    error_code TEXT CHECK (error_code IS NULL OR length(error_code) <= 60),
    created_at TEXT NOT NULL,
    finished_at TEXT,
    last_verified_at TEXT,
    refunded_at TEXT,
    updated_at TEXT NOT NULL,
    CHECK (status NOT IN ('succeeded', 'refunded') OR (order_id IS NOT NULL AND helloasso_payment_id IS NOT NULL AND received_amount_cents IS NOT NULL AND paid_at IS NOT NULL)),
    CHECK (status = 'pending' OR finished_at IS NOT NULL),
    CHECK ((status = 'refunded') = (refunded_at IS NOT NULL))
) STRICT;

CREATE UNIQUE INDEX online_payments_intent ON online_payments (environment, checkout_intent_id)
WHERE checkout_intent_id IS NOT NULL;
CREATE UNIQUE INDEX online_payments_helloasso_payment ON online_payments (environment, helloasso_payment_id)
WHERE helloasso_payment_id IS NOT NULL;
CREATE INDEX online_payments_membership ON online_payments (membership_id, created_at);
CREATE INDEX online_payments_pending ON online_payments (status, created_at) WHERE status = 'pending';
CREATE INDEX online_payments_order ON online_payments (order_id);

-- Signalements de paiement a traiter par un administrateur (research.md 008 R10).
CREATE TABLE payment_alerts (
    id INTEGER PRIMARY KEY,
    membership_id INTEGER NOT NULL REFERENCES memberships (id) ON DELETE CASCADE,
    online_payment_id INTEGER NOT NULL REFERENCES online_payments (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('duplicate', 'amount_mismatch', 'refund_due', 'unverified')),
    created_at TEXT NOT NULL,
    resolved_at TEXT,
    resolved_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    CHECK (resolved_by IS NULL OR resolved_at IS NOT NULL)
) STRICT;

CREATE UNIQUE INDEX payment_alerts_unique ON payment_alerts (online_payment_id, kind);
CREATE INDEX payment_alerts_open ON payment_alerts (resolved_at, created_at);

-- +goose Down
DROP TABLE payment_alerts;
DROP TABLE online_payments;
DROP TABLE payment_settings;
DROP TRIGGER memberships_fts_au;
DROP TRIGGER memberships_fts_ad;
DROP TRIGGER memberships_fts_ai;
DROP TABLE memberships_fts;
DROP TABLE memberships;
