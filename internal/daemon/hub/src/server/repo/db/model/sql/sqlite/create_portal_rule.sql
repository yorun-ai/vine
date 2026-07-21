CREATE TABLE IF NOT EXISTS portal_rule (
    id INTEGER PRIMARY KEY,                 -- Primary key ID
    created_at DATETIME,                    -- Creation time
    updated_at DATETIME,                    -- Update time
    deleted_at DATETIME,                    -- Soft deletion time
    name TEXT NOT NULL,                     -- Rule name
    scheme TEXT NOT NULL,                   -- Scheme, only http / https are supported
    host TEXT NOT NULL,                     -- Domain or IP, empty string means no restriction
    port INTEGER NOT NULL,                  -- Port, 0 means no restriction
    path_prefix TEXT NOT NULL,              -- Path prefix, empty string matches all paths
    target_type TEXT NOT NULL,              -- Target type: SITE / PERMANENT_REDIRECT / TEMPORARY_REDIRECT
    site_name TEXT NOT NULL,                -- Target site name, empty string when target is not SITE
    redirection_pattern TEXT NOT NULL,      -- Redirection pattern, empty string when target is not Redirect
    built_in BOOLEAN NOT NULL DEFAULT FALSE     -- Whether this rule is built in
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_rule_match
    ON portal_rule(scheme, host, port, path_prefix);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_rule_name
    ON portal_rule(name);
