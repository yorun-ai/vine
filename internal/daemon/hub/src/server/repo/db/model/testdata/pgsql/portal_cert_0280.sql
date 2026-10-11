CREATE TABLE IF NOT EXISTS portal_cert (
    id SERIAL PRIMARY KEY,              -- Primary key ID
    created_at TIMESTAMPTZ,             -- Creation time
    updated_at TIMESTAMPTZ,             -- Update time
    deleted_at TIMESTAMPTZ,             -- Soft deletion time
    name TEXT NOT NULL,                 -- Certificate name
    issuer TEXT NOT NULL,               -- Certificate issuer
    domains TEXT NOT NULL,              -- Certificate domains
    certificate TEXT NOT NULL DEFAULT '', -- PEM certificate chain
    private_key TEXT NOT NULL DEFAULT '', -- PEM private key
    -- TODO: Remove both Base64 columns when the legacy database upgrade window closes,
    -- together with the model fields and PEM migration.
    public_key_base64 TEXT NOT NULL DEFAULT '',    -- Deprecated: migrate to certificate
    private_key_base64 TEXT NOT NULL DEFAULT '',   -- Deprecated: migrate to private_key
    valid_from TIMESTAMPTZ,             -- Certificate valid from
    valid_to TIMESTAMPTZ,               -- Certificate valid to
    enabled BOOLEAN NOT NULL DEFAULT TRUE -- Whether Hub publishes this certificate
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_cert_name
    ON portal_cert(name);
