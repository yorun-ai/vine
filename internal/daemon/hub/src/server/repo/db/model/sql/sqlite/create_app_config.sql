CREATE TABLE IF NOT EXISTS app_config (
    id INTEGER PRIMARY KEY,      -- Primary key ID
    created_at DATETIME,         -- Creation time
    updated_at DATETIME,         -- Update time
    deleted_at DATETIME,         -- Soft deletion time
    name TEXT NOT NULL,          -- Config name
    value TEXT NOT NULL,         -- Config JSON value
    version INTEGER NOT NULL     -- Config version
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_app_config_name ON app_config(name);
