CREATE TABLE IF NOT EXISTS portal_rule (
    id INTEGER PRIMARY KEY,                       -- Primary key ID
    created_at DATETIME,                          -- Creation time
    updated_at DATETIME,                          -- Update time
    deleted_at DATETIME,                          -- Soft deletion time
    name TEXT NOT NULL,                           -- Rule name
    entry_id INTEGER,                             -- Portal entry that owns the access; null while an older Hub owns the rule
    match_path_prefix TEXT NOT NULL,              -- Path prefix, empty string matches all paths
    route_type TEXT NOT NULL,                     -- Target type: SITE / PERMANENT_REDIRECT / TEMPORARY_REDIRECT
    route_site_name TEXT NOT NULL,                -- Target site name, empty string when target is not SITE
    route_redirection_pattern TEXT NOT NULL,      -- Redirection pattern, empty string when target is not Redirect
    route_path_prefix TEXT NOT NULL DEFAULT '',
    built_in BOOLEAN NOT NULL DEFAULT FALSE,      -- TODO: Drop with the built-in cleanup; nothing sets it TRUE
    enabled BOOLEAN NOT NULL DEFAULT TRUE,        -- Whether Hub publishes this rule; an older Hub leaves the default
    match_scheme TEXT NOT NULL,                   -- TODO: Drop with entry_id; the entry stores the access, and only an earlier Hub reads these
    match_host TEXT NOT NULL,                     -- TODO: Drop with entry_id
    match_port INTEGER NOT NULL                   -- TODO: Drop with entry_id
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_rule_name
    ON portal_rule(name);
