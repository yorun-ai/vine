CREATE TABLE IF NOT EXISTS metadata (
    id INTEGER PRIMARY KEY,      -- Primary key ID
    created_at DATETIME,         -- Creation time
    updated_at DATETIME,         -- Update time
    deleted_at DATETIME,         -- Soft deletion time
    name TEXT NOT NULL,          -- Metadata name
    value TEXT NOT NULL          -- Metadata value
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_metadata_name ON metadata(name);
