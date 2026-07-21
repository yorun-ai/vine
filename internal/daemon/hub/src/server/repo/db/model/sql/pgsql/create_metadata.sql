CREATE TABLE IF NOT EXISTS metadata (
    id SERIAL PRIMARY KEY,       -- Primary key ID
    created_at TIMESTAMPTZ,      -- Creation time
    updated_at TIMESTAMPTZ,      -- Update time
    deleted_at TIMESTAMPTZ,      -- Soft deletion time
    name TEXT NOT NULL,          -- Metadata name
    value TEXT NOT NULL          -- Metadata value
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_metadata_name ON metadata(name);
