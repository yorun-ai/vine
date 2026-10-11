CREATE TABLE IF NOT EXISTS portal_entry (
    id SERIAL PRIMARY KEY,                  -- Primary key ID
    created_at TIMESTAMPTZ,                 -- Creation time
    updated_at TIMESTAMPTZ,                 -- Update time
    deleted_at TIMESTAMPTZ,                 -- Soft deletion time
    name TEXT NOT NULL,                     -- Entry name, unique among user entries
    protocol TEXT NOT NULL DEFAULT '',      -- Protocol family; populated by migration
    http_config TEXT NOT NULL DEFAULT '',   -- Canonical HTTP configuration JSON
    listen_ips TEXT NOT NULL DEFAULT '[]',  -- Explicit listener IPs; empty keeps legacy wildcard TCP
    enabled BOOLEAN NOT NULL DEFAULT TRUE    -- Whether Hub publishes the rules of this entry
);

-- An entry name identifies the entry the Dashboard shows and links to.
CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_entry_name
    ON portal_entry(name) WHERE deleted_at IS NULL;
