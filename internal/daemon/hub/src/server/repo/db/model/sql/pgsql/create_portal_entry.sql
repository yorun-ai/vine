CREATE TABLE IF NOT EXISTS portal_entry (
    id SERIAL PRIMARY KEY,                  -- Primary key ID
    created_at TIMESTAMPTZ,                 -- Creation time
    updated_at TIMESTAMPTZ,                 -- Update time
    deleted_at TIMESTAMPTZ,                 -- Soft deletion time
    name TEXT NOT NULL,                     -- Entry name, unique among user entries
    protocol TEXT NOT NULL DEFAULT '',      -- Protocol family; populated by migration
    http_config TEXT NOT NULL DEFAULT '',   -- Canonical HTTP configuration JSON
    scheme TEXT NOT NULL,                   -- Scheme, only http / https are supported
    host TEXT NOT NULL,                     -- Domain or IP, empty string means no restriction
    port INTEGER NOT NULL,                  -- Port Portal listens on
    listen_ips TEXT NOT NULL DEFAULT '[]',  -- Explicit listener IPs; empty keeps legacy wildcard TCP
    enabled BOOLEAN NOT NULL DEFAULT TRUE    -- Whether Hub publishes the rules of this entry
);

-- An entry name identifies the entry the Dashboard shows and links to.
CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_entry_name
    ON portal_entry(name) WHERE deleted_at IS NULL;

-- One entry serves an access, and Hub creates an entry again when rules return
-- to an access whose entry was removed.
CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_entry_access
    ON portal_entry(scheme, host, port) WHERE deleted_at IS NULL AND scheme <> '';
