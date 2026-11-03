-- +goose Up
-- Parametres generaux de l'association (specs/010-general-settings).
-- Toutes les colonnes sont facultatives : NULL = non renseigne. Les valeurs
-- sont normalisees par internal/association avant ecriture ; les CHECK sont
-- une defense en profondeur.
ALTER TABLE association ADD COLUMN tagline TEXT CHECK (tagline IS NULL OR length(tagline) BETWEEN 1 AND 150);
ALTER TABLE association ADD COLUMN short_name TEXT CHECK (short_name IS NULL OR length(short_name) BETWEEN 1 AND 20);
ALTER TABLE association ADD COLUMN purpose TEXT CHECK (purpose IS NULL OR length(purpose) BETWEEN 1 AND 2000);
ALTER TABLE association ADD COLUMN rna TEXT CHECK (rna IS NULL OR rna GLOB 'W[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]');
ALTER TABLE association ADD COLUMN siret TEXT CHECK (siret IS NULL OR (length(siret) = 14 AND siret NOT GLOB '*[^0-9]*'));
ALTER TABLE association ADD COLUMN founded_on TEXT CHECK (founded_on IS NULL OR date(founded_on) IS founded_on);
ALTER TABLE association ADD COLUMN address_line1 TEXT CHECK (address_line1 IS NULL OR length(address_line1) BETWEEN 1 AND 100);
ALTER TABLE association ADD COLUMN address_line2 TEXT CHECK (address_line2 IS NULL OR length(address_line2) BETWEEN 1 AND 100);
ALTER TABLE association ADD COLUMN postal_code TEXT CHECK (postal_code IS NULL OR postal_code GLOB '[0-9][0-9][0-9][0-9][0-9]');
ALTER TABLE association ADD COLUMN city TEXT CHECK (city IS NULL OR length(city) BETWEEN 1 AND 100);
ALTER TABLE association ADD COLUMN contact_email TEXT CHECK (contact_email IS NULL OR length(contact_email) BETWEEN 3 AND 254);
ALTER TABLE association ADD COLUMN phone TEXT CHECK (phone IS NULL OR phone GLOB '0[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]');
ALTER TABLE association ADD COLUMN website TEXT CHECK (website IS NULL OR (length(website) <= 300 AND (website GLOB 'http://?*' OR website GLOB 'https://?*')));
ALTER TABLE association ADD COLUMN facebook_url TEXT CHECK (facebook_url IS NULL OR (length(facebook_url) <= 300 AND (facebook_url GLOB 'http://?*' OR facebook_url GLOB 'https://?*')));
ALTER TABLE association ADD COLUMN instagram_url TEXT CHECK (instagram_url IS NULL OR (length(instagram_url) <= 300 AND (instagram_url GLOB 'http://?*' OR instagram_url GLOB 'https://?*')));
ALTER TABLE association ADD COLUMN linkedin_url TEXT CHECK (linkedin_url IS NULL OR (length(linkedin_url) <= 300 AND (linkedin_url GLOB 'http://?*' OR linkedin_url GLOB 'https://?*')));
ALTER TABLE association ADD COLUMN youtube_url TEXT CHECK (youtube_url IS NULL OR (length(youtube_url) <= 300 AND (youtube_url GLOB 'http://?*' OR youtube_url GLOB 'https://?*')));
ALTER TABLE association ADD COLUMN mastodon_url TEXT CHECK (mastodon_url IS NULL OR (length(mastodon_url) <= 300 AND (mastodon_url GLOB 'http://?*' OR mastodon_url GLOB 'https://?*')));
ALTER TABLE association ADD COLUMN bluesky_url TEXT CHECK (bluesky_url IS NULL OR (length(bluesky_url) <= 300 AND (bluesky_url GLOB 'http://?*' OR bluesky_url GLOB 'https://?*')));
ALTER TABLE association ADD COLUMN primary_color TEXT CHECK (primary_color IS NULL OR primary_color GLOB '#[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]');
ALTER TABLE association ADD COLUMN accent_color TEXT CHECK (accent_color IS NULL OR accent_color GLOB '#[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]');

-- Logo de l'association : au plus une ligne. Les deux variantes reencodees
-- sont rangees dans FRIGG_DATA_DIR/logo sous un nom genere.
CREATE TABLE association_logo (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    original_name TEXT NOT NULL CHECK (length(original_name) BETWEEN 1 AND 150),
    source_type TEXT NOT NULL CHECK (source_type IN ('image/png', 'image/jpeg', 'image/webp')),
    source_size_bytes INTEGER NOT NULL CHECK (source_size_bytes BETWEEN 1 AND 1048576),
    web_name TEXT NOT NULL UNIQUE CHECK (length(web_name) = 32 AND web_name NOT GLOB '*[^0-9a-f]*'),
    web_type TEXT NOT NULL CHECK (web_type IN ('image/png', 'image/jpeg')),
    web_size_bytes INTEGER NOT NULL CHECK (web_size_bytes >= 1),
    web_width INTEGER NOT NULL CHECK (web_width BETWEEN 1 AND 512),
    web_height INTEGER NOT NULL CHECK (web_height BETWEEN 1 AND 512),
    web_sha256 TEXT NOT NULL CHECK (length(web_sha256) = 64),
    email_name TEXT NOT NULL UNIQUE CHECK (length(email_name) = 32 AND email_name NOT GLOB '*[^0-9a-f]*'),
    email_type TEXT NOT NULL CHECK (email_type IN ('image/png', 'image/jpeg')),
    email_size_bytes INTEGER NOT NULL CHECK (email_size_bytes BETWEEN 1 AND 102400),
    email_width INTEGER NOT NULL CHECK (email_width BETWEEN 1 AND 400),
    email_height INTEGER NOT NULL CHECK (email_height BETWEEN 1 AND 96),
    email_sha256 TEXT NOT NULL CHECK (length(email_sha256) = 64),
    uploaded_at TEXT NOT NULL,
    uploaded_by INTEGER REFERENCES accounts (id) ON DELETE SET NULL
) STRICT;

-- File d'e-mails : wrap = 1 si le worker applique le gabarit commun (nom,
-- couleurs, logo) au moment de l'envoi ; footer remplace la mention de pied.
ALTER TABLE outbound_emails ADD COLUMN wrap INTEGER NOT NULL DEFAULT 0 CHECK (wrap IN (0, 1));
ALTER TABLE outbound_emails ADD COLUMN footer TEXT CHECK (footer IS NULL OR length(footer) <= 500);

-- +goose Down
-- Aucune de ces colonnes ne porte de cle etrangere ni d'index : DROP COLUMN
-- suffit (les CHECK ne portent que sur la colonne supprimee).
ALTER TABLE outbound_emails DROP COLUMN footer;
ALTER TABLE outbound_emails DROP COLUMN wrap;
DROP TABLE association_logo;
ALTER TABLE association DROP COLUMN accent_color;
ALTER TABLE association DROP COLUMN primary_color;
ALTER TABLE association DROP COLUMN bluesky_url;
ALTER TABLE association DROP COLUMN mastodon_url;
ALTER TABLE association DROP COLUMN youtube_url;
ALTER TABLE association DROP COLUMN linkedin_url;
ALTER TABLE association DROP COLUMN instagram_url;
ALTER TABLE association DROP COLUMN facebook_url;
ALTER TABLE association DROP COLUMN website;
ALTER TABLE association DROP COLUMN phone;
ALTER TABLE association DROP COLUMN contact_email;
ALTER TABLE association DROP COLUMN city;
ALTER TABLE association DROP COLUMN postal_code;
ALTER TABLE association DROP COLUMN address_line2;
ALTER TABLE association DROP COLUMN address_line1;
ALTER TABLE association DROP COLUMN founded_on;
ALTER TABLE association DROP COLUMN siret;
ALTER TABLE association DROP COLUMN rna;
ALTER TABLE association DROP COLUMN purpose;
ALTER TABLE association DROP COLUMN short_name;
ALTER TABLE association DROP COLUMN tagline;
