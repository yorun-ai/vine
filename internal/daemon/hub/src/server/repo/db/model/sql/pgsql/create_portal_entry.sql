CREATE TABLE IF NOT EXISTS portal_entry (
    id SERIAL PRIMARY KEY,                  -- Primary key ID
    created_at TIMESTAMPTZ,                 -- Creation time
    updated_at TIMESTAMPTZ,                 -- Update time
    deleted_at TIMESTAMPTZ,                 -- Soft deletion time
    name TEXT NOT NULL,                     -- Entry name, unique among user entries
    scheme TEXT NOT NULL,                   -- Scheme, only http / https are supported
    host TEXT NOT NULL,                     -- Domain or IP, empty string means no restriction
    port INTEGER NOT NULL,                  -- Port Portal listens on
    built_in BOOLEAN NOT NULL DEFAULT FALSE -- Whether this entry carries built-in Hub rules
);

-- An entry name identifies the entry the Dashboard shows and links to.
CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_entry_name
    ON portal_entry(name) WHERE built_in = FALSE AND deleted_at IS NULL;

-- One user entry serves an access: the built-in Dashboard entry is not part of
-- user entries, a user rule never joins it, and Hub creates an entry again when
-- rules return to an access whose entry was removed.
CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_entry_access
    ON portal_entry(scheme, host, port) WHERE built_in = FALSE AND deleted_at IS NULL;
