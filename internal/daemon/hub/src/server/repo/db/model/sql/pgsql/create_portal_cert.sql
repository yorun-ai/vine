CREATE TABLE IF NOT EXISTS portal_cert (
    id SERIAL PRIMARY KEY,              -- Primary key ID
    created_at TIMESTAMPTZ,             -- Creation time
    updated_at TIMESTAMPTZ,             -- Update time
    deleted_at TIMESTAMPTZ,             -- Soft deletion time
    name TEXT NOT NULL,                 -- Certificate name
    issuer TEXT NOT NULL,               -- Certificate issuer
    domains TEXT NOT NULL,              -- Certificate domains
    public_key_base64 TEXT NOT NULL,    -- Public key in base64 format
    private_key_base64 TEXT NOT NULL,   -- Private key in base64 format
    valid_from TIMESTAMPTZ,             -- Certificate valid from
    valid_to TIMESTAMPTZ                -- Certificate valid to
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_cert_name
    ON portal_cert(name);
