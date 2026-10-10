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
    valid_from TIMESTAMPTZ,             -- Certificate valid from
    valid_to TIMESTAMPTZ,               -- Certificate valid to
    enabled BOOLEAN NOT NULL DEFAULT TRUE -- Whether Hub publishes this certificate
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_cert_name
    ON portal_cert(name);
