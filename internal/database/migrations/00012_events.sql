-- +goose NO TRANSACTION
-- +goose Up
-- Evenements de l'association et inscriptions des fiches membres
-- (research.md 012 R1). Montants en centimes d'euro, instants en UTC.
--
-- online_payments et payment_alerts sont reconstruites pour viser une
-- adhesion OU une inscription (R2). Les cles etrangeres sont desactivees le
-- temps de la reconstruction : sinon DROP TABLE online_payments declencherait
-- le ON DELETE SET NULL de memberships.paid_online_payment_id. Le pool
-- d'ecriture n'a qu'une connexion : le PRAGMA porte sur celle qui migre et
-- le dernier statement le retablit. En cas d'echec, la transaction n'est
-- jamais validee et le demarrage echoue.
PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE events (
    id INTEGER PRIMARY KEY,
    season_id INTEGER NOT NULL REFERENCES seasons (id) ON DELETE RESTRICT,
    title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 120),
    location TEXT NOT NULL CHECK (length(location) BETWEEN 1 AND 200),
    description TEXT CHECK (description IS NULL OR length(description) BETWEEN 1 AND 5000),
    starts_at TEXT NOT NULL,
    ends_at TEXT NOT NULL,
    openness TEXT NOT NULL DEFAULT 'season_members' CHECK (openness IN ('season_members', 'all_members')),
    capacity INTEGER CHECK (capacity IS NULL OR capacity BETWEEN 1 AND 10000),
    price_cents INTEGER NOT NULL DEFAULT 0 CHECK (price_cents BETWEEN 0 AND 1000000),
    registration_opens_at TEXT,
    registration_closes_at TEXT CHECK (registration_closes_at IS NULL OR registration_closes_at <= ends_at),
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'cancelled')),
    cancel_message TEXT CHECK (cancel_message IS NULL OR length(cancel_message) BETWEEN 1 AND 1000),
    published_at TEXT,
    cancelled_at TEXT,
    cancelled_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    created_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    updated_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK (ends_at > starts_at),
    CHECK (registration_opens_at IS NULL OR registration_closes_at IS NULL OR registration_closes_at > registration_opens_at),
    CHECK (status <> 'cancelled' OR cancelled_at IS NOT NULL),
    CHECK (status = 'draft' OR published_at IS NOT NULL)
) STRICT;

CREATE INDEX events_starts_at ON events (starts_at);
CREATE INDEX events_season ON events (season_id, starts_at);
CREATE INDEX events_status ON events (status, starts_at);

-- Inscriptions : colonnes de paiement calquees sur memberships (R1).
CREATE TABLE event_registrations (
    id INTEGER PRIMARY KEY,
    event_id INTEGER NOT NULL REFERENCES events (id) ON DELETE RESTRICT,
    member_id INTEGER NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('confirmed', 'cancelled')),
    registered_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    registered_at TEXT NOT NULL,
    cancelled_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    cancelled_at TEXT,
    -- Prix fige a l'inscription.
    amount_cents INTEGER NOT NULL CHECK (amount_cents BETWEEN 0 AND 1000000),
    -- Reference EVT-AAAA-NNNN (R5) ; jamais modifiee une fois attribuee.
    payment_reference TEXT UNIQUE CHECK (payment_reference IS NULL OR length(payment_reference) BETWEEN 8 AND 16),
    -- NULL = inscription gratuite.
    payment_status TEXT CHECK (payment_status IN ('due', 'announced', 'paid')),
    announced_method TEXT CHECK (announced_method IN ('cash', 'cheque', 'transfer')),
    paid_method TEXT CHECK (paid_method IN ('cash', 'cheque', 'transfer', 'card')),
    paid_on TEXT CHECK (paid_on IS NULL OR date(paid_on) = paid_on),
    paid_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    paid_at TEXT,
    payment_note TEXT CHECK (payment_note IS NULL OR length(payment_note) BETWEEN 1 AND 200),
    paid_online_payment_id INTEGER REFERENCES online_payments (id) ON DELETE SET NULL,
    paid_amount_cents INTEGER CHECK (paid_amount_cents BETWEEN 0 AND 1000000),
    updated_at TEXT NOT NULL,
    CHECK ((payment_status IS NULL) = (amount_cents = 0)),
    CHECK (payment_status IS NULL OR payment_reference IS NOT NULL),
    CHECK ((payment_status IS 'paid') = (paid_method IS NOT NULL AND paid_on IS NOT NULL)),
    CHECK (payment_status IS NOT 'announced' OR announced_method IS NOT NULL),
    CHECK (paid_method IS NOT 'card' OR (paid_online_payment_id IS NOT NULL AND paid_amount_cents IS NOT NULL AND paid_by IS NULL)),
    CHECK (paid_online_payment_id IS NULL OR paid_method = 'card'),
    CHECK ((status = 'cancelled') = (cancelled_at IS NOT NULL))
) STRICT;

-- Une inscription confirmee au plus par fiche et par evenement (FR-014).
CREATE UNIQUE INDEX event_registrations_one_confirmed ON event_registrations (event_id, member_id)
WHERE status = 'confirmed';
CREATE INDEX event_registrations_event ON event_registrations (event_id, status);
CREATE INDEX event_registrations_member ON event_registrations (member_id);
CREATE INDEX event_registrations_payment ON event_registrations (payment_status)
WHERE payment_status IN ('due', 'announced');

-- Recherche par fragment de reference de paiement (R12).
CREATE VIRTUAL TABLE event_registrations_fts USING fts5(
    payment_reference,
    content = 'event_registrations',
    content_rowid = 'id',
    tokenize = "trigram remove_diacritics 1"
);

-- +goose StatementBegin
CREATE TRIGGER event_registrations_fts_ai AFTER INSERT ON event_registrations BEGIN
    INSERT INTO event_registrations_fts (rowid, payment_reference) VALUES (new.id, new.payment_reference);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER event_registrations_fts_ad AFTER DELETE ON event_registrations BEGIN
    INSERT INTO event_registrations_fts (event_registrations_fts, rowid, payment_reference)
    VALUES ('delete', old.id, old.payment_reference);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER event_registrations_fts_au AFTER UPDATE OF payment_reference ON event_registrations BEGIN
    INSERT INTO event_registrations_fts (event_registrations_fts, rowid, payment_reference)
    VALUES ('delete', old.id, old.payment_reference);
    INSERT INTO event_registrations_fts (rowid, payment_reference) VALUES (new.id, new.payment_reference);
END;
-- +goose StatementEnd

-- Tentatives de paiement par carte : une adhesion ou une inscription (R2).
CREATE TABLE online_payments_new (
    id INTEGER PRIMARY KEY,
    membership_id INTEGER REFERENCES memberships (id) ON DELETE CASCADE,
    registration_id INTEGER REFERENCES event_registrations (id) ON DELETE CASCADE,
    account_id INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    environment TEXT NOT NULL CHECK (environment IN ('sandbox', 'production')),
    amount_cents INTEGER NOT NULL CHECK (amount_cents BETWEEN 1 AND 1000000),
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
    CHECK ((membership_id IS NULL) <> (registration_id IS NULL)),
    CHECK (status NOT IN ('succeeded', 'refunded') OR (order_id IS NOT NULL AND helloasso_payment_id IS NOT NULL AND received_amount_cents IS NOT NULL AND paid_at IS NOT NULL)),
    CHECK (status = 'pending' OR finished_at IS NOT NULL),
    CHECK ((status = 'refunded') = (refunded_at IS NOT NULL))
) STRICT;

INSERT INTO online_payments_new (
    id, membership_id, account_id, environment, amount_cents, status, checkout_intent_id,
    redirect_url, order_id, helloasso_payment_id, received_amount_cents, paid_at, error_code,
    created_at, finished_at, last_verified_at, refunded_at, updated_at
)
SELECT
    id, membership_id, account_id, environment, amount_cents, status, checkout_intent_id,
    redirect_url, order_id, helloasso_payment_id, received_amount_cents, paid_at, error_code,
    created_at, finished_at, last_verified_at, refunded_at, updated_at
FROM online_payments;

DROP TABLE online_payments;
ALTER TABLE online_payments_new RENAME TO online_payments;

CREATE UNIQUE INDEX online_payments_intent ON online_payments (environment, checkout_intent_id)
WHERE checkout_intent_id IS NOT NULL;
CREATE UNIQUE INDEX online_payments_helloasso_payment ON online_payments (environment, helloasso_payment_id)
WHERE helloasso_payment_id IS NOT NULL;
CREATE INDEX online_payments_membership ON online_payments (membership_id, created_at);
CREATE INDEX online_payments_registration ON online_payments (registration_id, created_at);
CREATE INDEX online_payments_pending ON online_payments (status, created_at) WHERE status = 'pending';
CREATE INDEX online_payments_order ON online_payments (order_id);

-- Signalements de paiement : une adhesion ou une inscription (R2).
CREATE TABLE payment_alerts_new (
    id INTEGER PRIMARY KEY,
    membership_id INTEGER REFERENCES memberships (id) ON DELETE CASCADE,
    registration_id INTEGER REFERENCES event_registrations (id) ON DELETE CASCADE,
    online_payment_id INTEGER NOT NULL REFERENCES online_payments (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('duplicate', 'amount_mismatch', 'refund_due', 'unverified')),
    created_at TEXT NOT NULL,
    resolved_at TEXT,
    resolved_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    CHECK ((membership_id IS NULL) <> (registration_id IS NULL)),
    CHECK (resolved_by IS NULL OR resolved_at IS NOT NULL)
) STRICT;

INSERT INTO payment_alerts_new (id, membership_id, online_payment_id, kind, created_at, resolved_at, resolved_by)
SELECT id, membership_id, online_payment_id, kind, created_at, resolved_at, resolved_by FROM payment_alerts;

DROP TABLE payment_alerts;
ALTER TABLE payment_alerts_new RENAME TO payment_alerts;

CREATE UNIQUE INDEX payment_alerts_unique ON payment_alerts (online_payment_id, kind);
CREATE INDEX payment_alerts_open ON payment_alerts (resolved_at, created_at);
CREATE INDEX payment_alerts_registration ON payment_alerts (registration_id);

COMMIT;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

BEGIN;

DELETE FROM payment_alerts WHERE registration_id IS NOT NULL;
DELETE FROM online_payments WHERE registration_id IS NOT NULL;

CREATE TABLE payment_alerts_old (
    id INTEGER PRIMARY KEY,
    membership_id INTEGER NOT NULL REFERENCES memberships (id) ON DELETE CASCADE,
    online_payment_id INTEGER NOT NULL REFERENCES online_payments (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('duplicate', 'amount_mismatch', 'refund_due', 'unverified')),
    created_at TEXT NOT NULL,
    resolved_at TEXT,
    resolved_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    CHECK (resolved_by IS NULL OR resolved_at IS NOT NULL)
) STRICT;

INSERT INTO payment_alerts_old (id, membership_id, online_payment_id, kind, created_at, resolved_at, resolved_by)
SELECT id, membership_id, online_payment_id, kind, created_at, resolved_at, resolved_by FROM payment_alerts;

DROP TABLE payment_alerts;
ALTER TABLE payment_alerts_old RENAME TO payment_alerts;

CREATE UNIQUE INDEX payment_alerts_unique ON payment_alerts (online_payment_id, kind);
CREATE INDEX payment_alerts_open ON payment_alerts (resolved_at, created_at);

CREATE TABLE online_payments_old (
    id INTEGER PRIMARY KEY,
    membership_id INTEGER NOT NULL REFERENCES memberships (id) ON DELETE CASCADE,
    account_id INTEGER REFERENCES accounts (id) ON DELETE SET NULL,
    environment TEXT NOT NULL CHECK (environment IN ('sandbox', 'production')),
    amount_cents INTEGER NOT NULL CHECK (amount_cents BETWEEN 1 AND 1000000),
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

INSERT INTO online_payments_old (
    id, membership_id, account_id, environment, amount_cents, status, checkout_intent_id,
    redirect_url, order_id, helloasso_payment_id, received_amount_cents, paid_at, error_code,
    created_at, finished_at, last_verified_at, refunded_at, updated_at
)
SELECT
    id, membership_id, account_id, environment, amount_cents, status, checkout_intent_id,
    redirect_url, order_id, helloasso_payment_id, received_amount_cents, paid_at, error_code,
    created_at, finished_at, last_verified_at, refunded_at, updated_at
FROM online_payments;

DROP TABLE online_payments;
ALTER TABLE online_payments_old RENAME TO online_payments;

CREATE UNIQUE INDEX online_payments_intent ON online_payments (environment, checkout_intent_id)
WHERE checkout_intent_id IS NOT NULL;
CREATE UNIQUE INDEX online_payments_helloasso_payment ON online_payments (environment, helloasso_payment_id)
WHERE helloasso_payment_id IS NOT NULL;
CREATE INDEX online_payments_membership ON online_payments (membership_id, created_at);
CREATE INDEX online_payments_pending ON online_payments (status, created_at) WHERE status = 'pending';
CREATE INDEX online_payments_order ON online_payments (order_id);

DROP TRIGGER event_registrations_fts_au;
DROP TRIGGER event_registrations_fts_ad;
DROP TRIGGER event_registrations_fts_ai;
DROP TABLE event_registrations_fts;
DROP TABLE event_registrations;
DROP TABLE events;

COMMIT;

PRAGMA foreign_keys = ON;
