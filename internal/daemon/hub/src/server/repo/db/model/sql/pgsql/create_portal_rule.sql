CREATE TABLE IF NOT EXISTS portal_rule (
    id SERIAL PRIMARY KEY,                  -- Primary key ID
    created_at TIMESTAMPTZ,                 -- Creation time
    updated_at TIMESTAMPTZ,                 -- Update time
    deleted_at TIMESTAMPTZ,                 -- Soft deletion time
    name TEXT NOT NULL,                     -- Rule name
    match_scheme TEXT NOT NULL,                   -- Scheme, only http / https are supported
    match_host TEXT NOT NULL,                     -- Domain or IP, empty string means no restriction
    match_port INTEGER NOT NULL,                  -- Port, 0 means no restriction
    match_path_prefix TEXT NOT NULL,              -- Path prefix, empty string matches all paths
    route_type TEXT NOT NULL,              -- Target type: SITE / PERMANENT_REDIRECT / TEMPORARY_REDIRECT
    route_site_name TEXT NOT NULL,                -- Target site name, empty string when target is not SITE
    route_redirection_pattern TEXT NOT NULL,      -- Redirection pattern, empty string when target is not Redirect
    route_path_prefix TEXT NOT NULL DEFAULT '',
    built_in BOOLEAN NOT NULL DEFAULT FALSE     -- Whether this rule is built in
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_rule_match
    ON portal_rule(match_scheme, match_host, match_port, match_path_prefix);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_rule_name
    ON portal_rule(name);
