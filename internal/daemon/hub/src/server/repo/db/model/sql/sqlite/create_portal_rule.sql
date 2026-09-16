CREATE TABLE IF NOT EXISTS portal_rule (
    id INTEGER PRIMARY KEY,                 -- Primary key ID
    created_at DATETIME,                    -- Creation time
    updated_at DATETIME,                    -- Update time
    deleted_at DATETIME,                    -- Soft deletion time
    name TEXT NOT NULL,                     -- Rule name
    entry_id INTEGER NOT NULL DEFAULT 0,          -- Portal entry that owns the access configuration
    match_path_prefix TEXT NOT NULL,              -- Path prefix, empty string matches all paths
    route_type TEXT NOT NULL,              -- Target type: SITE / PERMANENT_REDIRECT / TEMPORARY_REDIRECT
    route_site_name TEXT NOT NULL,                -- Target site name, empty string when target is not SITE
    route_redirection_pattern TEXT NOT NULL,      -- Redirection pattern, empty string when target is not Redirect
    route_path_prefix TEXT NOT NULL DEFAULT '',
    built_in BOOLEAN NOT NULL DEFAULT FALSE     -- Whether this rule is built in
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_rule_entry_path
    ON portal_rule(entry_id, match_path_prefix);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_rule_name
    ON portal_rule(name);
