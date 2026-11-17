-- +goose Up
-- Parcours publics : adhesion et inscription sans compte
-- (specs/016-public-journeys, research.md R1 a R5). Uniquement des
-- CREATE TABLE, CREATE INDEX et ADD COLUMN : aucune reconstruction.

-- Depots non confirmes (R3) : aucune existence metier tant que l'adresse
-- n'est pas confirmee. Purges 7 jours apres l'envoi par le balayeur.
-- Ni adresse IP ni agent utilisateur (collecte minimale).
CREATE TABLE public_submissions (
    id INTEGER PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('membership', 'registration')),
    season_id INTEGER REFERENCES seasons (id) ON DELETE CASCADE,
    fee_id INTEGER REFERENCES membership_fees (id) ON DELETE CASCADE,
    declared_kind TEXT CHECK (declared_kind IN ('first', 'renewal')),
    event_id INTEGER REFERENCES events (id) ON DELETE CASCADE,
    first_name TEXT NOT NULL CHECK (length(first_name) BETWEEN 1 AND 80),
    last_name TEXT NOT NULL CHECK (length(last_name) BETWEEN 1 AND 80),
    birth_date TEXT NOT NULL CHECK (birth_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    -- Adresse normalisee (auth.NormalizeEmail), obligatoire.
    email TEXT NOT NULL CHECK (length(email) BETWEEN 3 AND 254),
    phone TEXT CHECK (phone IS NULL OR length(phone) BETWEEN 9 AND 16),
    address_line1 TEXT CHECK (address_line1 IS NULL OR length(address_line1) BETWEEN 1 AND 100),
    address_line2 TEXT CHECK (address_line2 IS NULL OR length(address_line2) BETWEEN 1 AND 100),
    postal_code TEXT CHECK (postal_code IS NULL OR postal_code GLOB '[0-9][0-9][0-9][0-9][0-9]'),
    city TEXT CHECK (city IS NULL OR length(city) BETWEEN 1 AND 100),
    -- Valeurs des champs personnalises, JSON canonique produit par customfield.
    custom_values TEXT NOT NULL DEFAULT '[]',
    create_account INTEGER NOT NULL DEFAULT 0 CHECK (create_account IN (0, 1)),
    consented_at TEXT NOT NULL,
    -- Empreinte SHA-256 du jeton de confirmation (R2).
    token_hash BLOB NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    -- Coherence entre la nature du depot et sa cible.
    CHECK (
        (kind = 'membership' AND season_id IS NOT NULL AND fee_id IS NOT NULL
            AND declared_kind IS NOT NULL AND event_id IS NULL)
        OR (kind = 'registration' AND event_id IS NOT NULL AND season_id IS NULL
            AND fee_id IS NULL AND declared_kind IS NULL)
    ),
    -- Adresse postale tout ou rien, comme members.
    CHECK (
        (address_line1 IS NULL) = (postal_code IS NULL)
        AND (postal_code IS NULL) = (city IS NULL)
        AND (address_line2 IS NULL OR address_line1 IS NOT NULL)
    )
) STRICT;

CREATE INDEX public_submissions_created_at ON public_submissions (created_at);

-- Liens personnels de suivi (R1, R2) : reutilisables, 30 jours. Le lien ne
-- reference aucune fiche : son perimetre est recalcule a chaque acces a
-- partir de email (FR-014).
CREATE TABLE tracking_links (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL CHECK (length(email) BETWEEN 3 AND 254),
    token_hash BLOB NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    revoked_at TEXT,
    last_used_at TEXT,
    created_at TEXT NOT NULL
) STRICT;

CREATE INDEX tracking_links_email ON tracking_links (email);
CREATE INDEX tracking_links_expires_at ON tracking_links (expires_at);

-- Origine d'une fiche (R4) et preuve de l'adresse et du consentement.
ALTER TABLE members ADD COLUMN origin TEXT NOT NULL DEFAULT 'internal'
    CHECK (origin IN ('internal', 'public'));
ALTER TABLE members ADD COLUMN email_confirmed_at TEXT;
ALTER TABLE members ADD COLUMN consented_at TEXT;

-- Perimetre d'un lien : fiches sans compte d'une adresse.
CREATE INDEX members_tracked_email ON members (email COLLATE NOCASE)
WHERE account_id IS NULL AND email IS NOT NULL;
-- Fiches publiques inactives (R14).
CREATE INDEX members_origin_created ON members (origin, created_at)
WHERE origin = 'public';

-- Demande publique : ni demandeur ni createur, type declare obligatoire.
ALTER TABLE memberships ADD COLUMN via_public INTEGER NOT NULL DEFAULT 0
    CHECK (via_public IN (0, 1))
    CHECK (via_public = 0 OR (requested_by IS NULL AND created_by IS NULL AND declared_kind IS NOT NULL));

-- Inscription publique : sans auteur de compte.
ALTER TABLE event_registrations ADD COLUMN via_public INTEGER NOT NULL DEFAULT 0
    CHECK (via_public IN (0, 1))
    CHECK (via_public = 0 OR registered_by IS NULL);

-- Tentative de paiement par carte lancee par lien personnel (R11).
ALTER TABLE online_payments ADD COLUMN via_link INTEGER NOT NULL DEFAULT 0
    CHECK (via_link IN (0, 1))
    CHECK (via_link = 0 OR account_id IS NULL);

-- Evenement ouvert au public (R5) : slug base64url de 128 bits, stocke en
-- clair. ADD COLUMN n'accepte pas UNIQUE : index unique partiel.
ALTER TABLE events ADD COLUMN public_slug TEXT
    CHECK (public_slug IS NULL OR (length(public_slug) = 22 AND openness = 'all_members'));
CREATE UNIQUE INDEX events_public_slug ON events (public_slug) WHERE public_slug IS NOT NULL;

-- Accueil des demandes d'adhesion publiques (R6), desactive par defaut.
ALTER TABLE association ADD COLUMN public_membership_enabled INTEGER NOT NULL DEFAULT 0
    CHECK (public_membership_enabled IN (0, 1));

-- +goose Down
-- Les index qui lisent une colonne partent avant elle.
ALTER TABLE association DROP COLUMN public_membership_enabled;
DROP INDEX events_public_slug;
ALTER TABLE events DROP COLUMN public_slug;
ALTER TABLE online_payments DROP COLUMN via_link;
ALTER TABLE event_registrations DROP COLUMN via_public;
ALTER TABLE memberships DROP COLUMN via_public;
DROP INDEX members_origin_created;
DROP INDEX members_tracked_email;
ALTER TABLE members DROP COLUMN consented_at;
ALTER TABLE members DROP COLUMN email_confirmed_at;
ALTER TABLE members DROP COLUMN origin;
DROP INDEX tracking_links_expires_at;
DROP INDEX tracking_links_email;
DROP TABLE tracking_links;
DROP INDEX public_submissions_created_at;
DROP TABLE public_submissions;
