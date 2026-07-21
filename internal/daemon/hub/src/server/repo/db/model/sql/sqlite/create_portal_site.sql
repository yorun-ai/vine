CREATE TABLE IF NOT EXISTS portal_site (
    id INTEGER PRIMARY KEY,                 -- Primary key ID
    created_at DATETIME,                    -- Creation time
    updated_at DATETIME,                    -- Update time
    deleted_at DATETIME,                    -- Soft deletion time
    name TEXT NOT NULL,                     -- Entry name
    type TEXT NOT NULL,                     -- Site type: rpcgw / webgw
    actor_skel_name TEXT NOT NULL,          -- Actor skel name
    actor_via TEXT NOT NULL,            -- Actor via
    cors_mode TEXT NOT NULL DEFAULT 'SAME_DOMAIN', -- CORS mode
    cors_origins TEXT NOT NULL DEFAULT '[]',-- CORS allowed origins JSON
    web_name TEXT NOT NULL,                 -- Web skel name
    built_in BOOLEAN NOT NULL DEFAULT FALSE     -- Whether this site is built in
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_portal_site_name
    ON portal_site(name);
