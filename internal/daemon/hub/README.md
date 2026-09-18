# Vine Hub

**English** | [简体中文](README.zh-CN.md)

Hub is Vine's configuration and service registry. It broadly follows a DDD-style layered architecture, maintains configuration, application state, and Rpc service registrations, and exposes read and subscription capabilities through Watch using the Redis protocol.

Without a database option, Hub defaults to `--no-db` and requires
`--seed-data-file`. Configuration is loaded into an isolated in-memory SQLite
database on each start. After initialization, configuration repos reject writes
to app configs, Portal entries, sites, rules, and certificates. Edit the seed file and
restart Hub to apply changes. Dashboard exposes this state and disables editing;
registration, schemas, and leases remain writable. Explicit `--db-sqlite-file`
or `--db-postgres-url` keeps writable persistence and is mutually exclusive
with `--no-db`. This also applies to standalone, which can alternatively receive
inline YAML through `Option.SeedHubData`, mutually exclusive with the seed file;
it uses the same import and read-only behavior.

## Directory Structure

```text
internal/daemon/hub/
├── api/                  Public APIs exposed by Hub to other runtime components
│   ├── app/              Shared constants such as the Hub inproc endpoint
│   ├── nats/             Hub NATS inproc access
│   ├── watch/            Hub Watch client, events, and inproc access
│   ├── watched/          Watch value structures and key formatting
│   └── skeled/           Generated control/admin Go packages
├── skel/                 Control and admin skeleton definitions
└── src/
    ├── dashboard/        Dashboard frontend source
    └── server/           Hub server runtime
        ├── app/          Assembly of Hub components, modules, and servicers
        ├── comp/         Shared runtime components such as Watch and NATS servers
        ├── core/         Domain state and Core/Repo interfaces
        ├── flag/         Hub flags and default normalization
        ├── impl/         Implementations split by exposed API boundary
        │   ├── control/  Link/Portal-facing Control API services
        │   └── admin/    Dashboard admin Rpc and Web services
        ├── mod/          Runtime modules such as controlapi, initializer, seeder, and syncer
        └── repo/         Infrastructure adapters implementing Core Repo interfaces
```

## Dashboard Packaging

- Builds containing only the empty `.gitkeep` placeholder automatically probe the Vite server that `script/dev-hub-dashboard.sh` starts on `localhost:7098`. If it is not running, Hub returns 404 with the command to start it.
- After changing Dashboard source, run `pnpm typecheck` and `pnpm build` in `src/dashboard`.
- Container and release builds generate `src/server/mod/admin/assets/dashboard/` immediately before compiling Vine. Generated files are ignored; the empty tracked `.gitkeep` is preserved.
- Keep user-facing text synchronized between `src/i18n/dictionaries/cn.ts` and `en.ts`.

The Dashboard source lives in `src/dashboard`. At runtime, release and container builds serve the Dashboard embedded in the Vine binary. A build without an embedded `index.html` or `index.html.br` uses the local Vite server. Local builds also embed previously packaged assets.

Generate embedded assets with the script; do not edit generated files by hand:

```bash
bash script/build-dashboard-assets.sh
GOWORK=off go build -o bin/vine ./cmd/vine
```

The script type-checks and builds into a temporary directory, then preserves the
output paths while choosing one representation per file: Brotli for text when
smaller, original bytes for images, fonts, and other files. It replaces the
complete assets directory only after packaging succeeds. A temporary Vite report
collects bundled JS licenses; imported CSS/font packages and the maintained uiw
notice supplement it. Packaging checks these against the Dashboard section of
root `THIRD_PARTY_LICENSES.txt`, then removes the report. Regenerate the unified
inventory with `bash script/gen-third-party-licenses.sh` after changing bundled
dependencies. No archive, Go byte array, or separate Dashboard license file is
shipped. Release archives include the unified inventory; all three images include
it and `LICENSE` in `/usr/share/licenses/vine/`. Every Go build embeds the directory
through `go:embed`, without a build tag. Packaging preserves the tracked empty
`.gitkeep` so it does not make the release checkout dirty; HTTP never serves it.
To switch a previously packaged local checkout back to Vite, remove generated
assets while keeping `.gitkeep`, then rebuild Go.

Dashboard uses the shared Asset Server for MIME types, HEAD, and compression
negotiation. Accepted Brotli files are sent directly; clients without Brotli
support receive a decoded or negotiated representation. Already compressed
images and fonts bypass dynamic compression. HTML navigation falls back to
`index.html`; missing static files return 404.

## Layering and Change Constraints

Keep Hub's layer responsibilities distinct:

- `core` defines domain state and Repo interfaces without depending on concrete database or Redis implementations. Repository interfaces name the storage primitives they provide - `List`, `GetById`, `GetByName`, `Save`, `Remove` - while the cores built on them expose the use cases, so the two layers share those verbs on purpose.
- `repo` implements persistence and Watch synchronization without owning external service orchestration. Repositories assemble the entities they return: derived values such as a site's Web mount path and Rpc services, a configuration's definition and status, and stored provenance such as field sources.
- `impl/control` implements only the Link/Portal-facing Control API services,
  while `impl/admin` and its `debug` and `dashboard` subpackages implement
  the Dashboard admin surface through `core`.
- `mod` contains runtime flows such as the Control API listener, initializer,
  seeder, syncer, scheduler, and sweeper.
- `comp` provides shared runtime components such as Watch and NATS.
- `app` only assembles components, modules, and servicers.

### Domain Writes

Configuration, entry, site, rule, and certificate writes go through their
corresponding Core. `Validate` checks and normalizes a complete entity without
writing. Rule validation does not resolve sites: Portal derives effective rule
paths from Web mount-path metadata published with sites. `Save`
creates or replaces by name and owns identity handling, along with versioning and
field sources where applicable. API updates merge provided fields into the
existing entity before validation.

An entry owns the scheme, host, and port Portal serves; rules reference the entry
and never store access of their own. `PortalRuleCore` resolves the entry of the
access a rule declares, so rules that share an access share one entry. Changing
an entry changes every rule it routes, and Hub republishes those rules so Portal
receives the access the entry now serves.

The Admin API reaches a rule's access through its entry: `PortalRuleCreation`
names the entry a new rule belongs to, and `PortalRuleUpdate` cannot change an
access at all. Seed YAML keeps declaring the access on the rule, because Hub
aggregates the declared access into entries while it applies the seed.

`PortalEntryCore` creates an entry for an access no user entry serves, addresses
it by `id` afterwards, so the Admin API and the Dashboard rename it, change the
access it serves, or flip its switch, and removes an entry that routes no rule:
rules belong to the operator, so deleting their entry fails instead of leaving
them without an access. The entry list returns an entry that routes nothing,
because an operator creates the entry before the rules that use it.

Two rules that match the same request are reported, not rejected: the path a rule
matches comes from the Web mount path its site declares, and Hub reads those
schemas only after an application registers them, which happens after Hub starts.
Hub audits the requests its published rules match once the schemas arrive: a
read-only Hub refuses to serve such a configuration, because it has no surface to
fix it from, and a stored configuration keeps both rules until the operator
resolves the request from the Dashboard. Only the access migration separates
rules on its own, because stored data is not edited by hand.

Seeder validates every entity the document declares before it writes anything,
and then calls Core `Save` per entity. Validation does not make an entire seed
transactional: a database failure can still leave some entities saved. The YAML
conversion layer maps configuration fields only; it does not assign database
identity or manage versions.

Hub serves the Admin API and the Dashboard on the admin module's own listener
(`--admin-listen`, default `127.0.0.1:7099`), the way the Control API owns
`--control-listen`: the listener
answers the API on `/api/invoke`, the path the Dashboard calls, and serves the
embedded Dashboard build for every other path, so the Dashboard is not part
of the Portal configuration. Hub provisions no
entry, site, or rule for it, and Portal never routes it. `RegistryCore` owns schema registration
and expired-lease removal. Initializer and Sweeper coordinate runtime publication
through Syncer. The seed-applied marker remains startup bookkeeping in Seeder.

### Change Constraints

Additional constraints for Hub changes:

- Database schema changes must update both `src/server/repo/db/model/sql/sqlite` and `src/server/repo/db/model/sql/pgsql`.
- Redis keys, Redis value JSON, and event formats are protocols shared by Hub, Link, and Portal. Update every producer, consumer, and test together.
- `watchserver` is a runtime distribution layer. Do not turn it into a second source of business state that bypasses Repo/Core.
- Registration semantics differ between normal and inproc modes for TTL, heartbeat, and sweeper behavior. Validate both modes separately.

## Runtime Model

Hub has four primary responsibilities:

1. Configuration center
   Hub reads configuration from the database and exposes it through `AppConfigRepo`. During startup, `initializer` loads configuration into Watch for Link to read and subscribe to.

2. Service registry
   Link writes application state and Rpc service registrations to Hub. Hub persists them through `RegistryRepo` and exposes queries and heartbeat lease renewal.

3. Watch distribution layer
   `watchserver` maintains an in-memory Watch dataset. Configuration, application state, Rpc/Web endpoints, and schemas are synchronized into it. Link and Portal read snapshots and subscribe to change events through Redis.

   The embedded Redis protocol requires authentication before any data command. It defines three users with resource-level ACLs:

   - `vine.hub` has full command and key access. Its password is generated randomly for the current process.
   - `vine.link` can read configuration, Rpc endpoint registrations, and the revision key; it can subscribe only to configuration channels and Rpc registration patterns.
   - `vine.portal` can read Portal rules, sites, certificates, actor/service/resource schemas, Rpc/Web endpoint registrations, and the revision key; it can subscribe only to the corresponding list patterns.

   Link and Portal use empty Redis passwords for in-process mode and separated-deployment debugging. With backend mTLS enabled, the client certificate authenticates the caller and binds its SPIFFE identity to the matching Redis username. Without mTLS, the usernames only select least-privilege roles and do not authenticate the caller, so the Redis endpoint must remain on loopback or a trusted private network protected by a firewall.

4. Separated API listeners
   The Control API listener exposes the `vine.hub.control` domain, containing
   only `InfoService` and `RegistryService`, to Link and Portal. The admin
   listener exposes the `vine.hub.admin` domain containing Dashboard
   admin Rpc services and `DashboardWeb`, in cleartext because a browser carries
   no mesh certificate. This keeps component traffic
   separate from the privileged admin surface without splitting Hub's
   process or state.

When embedded NATS is enabled, its server component provisions the
`VINE_EVENTS` and `VINE_TASKS` JetStream streams with memory storage. External
NATS deployments own stream provisioning and storage policy; Hub publishers
only use the existing streams.

## Configuration and Registration Sources

Hub currently supports two database backends:

- SQLite
- PostgreSQL

At startup, `--seed-data-file` imports initial configuration, Portal sites,
rules, and certificates from local YAML into the database. Hub reads this state
through its repos and publishes it to Watch for Link and Portal.

Database metadata records completion of the initial seed. Subsequent starts skip
all seed, variable, and source inputs; seed entries have no `override` switch.
No-db mode creates a fresh store and imports the seed on every start.

Field source metadata stores the original field template as JSON and each
resolved binding (relative path, variable name, placeholder, applied JSON value,
and default-use flag). AppConfig metadata groups nested substitutions under the
top-level value key. Admin edits clear obsolete templates and bindings. The
admin API and Dashboard expose this metadata alongside field origins.

Portal rule YAML uses flat fields in this order: `matchScheme`, `matchHost`,
`matchPort`, `matchPathPrefix`, `routeType`, `routeSiteName`,
`routeRedirectionPattern`, and `routePathPrefix`.

Hub stores an `enabled` switch on every Portal site, entry, rule, and
certificate, and the Dashboard edits it. A seed declares the same switch as
`disabled`, which defaults to false, so a seed names only the entities it turns
off; a seed that writes `enabled` fails instead of silently publishing the
entity. Hub keeps a disabled entity in its database and stops publishing it to
Watch, so Portal never sees it: Hub omits a disabled rule, the rules of a
disabled entry, the SITE rules of a disabled site, and a disabled certificate.
A database that predates the switch keeps every stored entity enabled.

Portal entry YAML declares `name`, `scheme`, `host`, `port`, and `disabled`. A seed applies
entries before rules, so a rule joins the entry that serves its access and keeps
the name the seed gave it; an entry may route no rule yet. Hub derives the name
`scheme[:host]:port` only for the entry it creates on its own, which is why the
entry a rule joins without a declared entry is named after its access. The
The Portal sections are typed: Hub rejects an entity that declares a field the
section does not name, so a misspelled or renamed field fails instead of
silently leaving the entity at its default. The built-in marker is not part of
the seed: Hub provisions no entity itself, and a field a Portal section does not
declare fails instead of nominating an entity.

Names stay unique per entity kind, which is why a site, an entry, and a rule may
share one.

A Portal rule joins an entry either by naming it with `entryName` or by declaring
the access the entry serves. The two are mutually exclusive: a rule never does
both, and one seed document uses one style for every rule it declares. A seed
that declares `portalEntries` names them, so its rules reference an entry with
`entryName` instead of declaring an access. A seed is self-contained: a rule only
references an entry the same document declares, so Hub never reads stored data to
complete the relationship. Hub rejects such a document before it writes anything,
because the entry owns the access.

Seeds keep declaring `matchScheme`, `matchHost`, and `matchPort` on rules, so a
seed written before entries had names keeps working. Hub aggregates the declared
access into entries while the seed is applied, so a seed never stores the same
access on every rule. Portal still receives rules carrying the access of their
entry, and the entry is Hub-side state rather than a Watch key.

The `mod/seeder` package owns the seed YAML contract. It decodes a document into
the domain entities Hub applies; the payload structs and the parsed document
stay private to the package. It accepts legacy rule fields with a warning per
field, and mixing old and new fields in one rule fails before Hub writes
anything. A Portal section declares only the fields Hub names for it.

Admin API and Watch use only the new fields; upgrade Hub and Portal together.
The database upgrade baseline is Vine v0.15.7, with `match_*` / `route_*`
columns already present. Start older databases with v0.15.7 to complete migration
before upgrading; current Hub no longer migrates legacy Portal rule columns.

Upgrading Hub from a release that stored rule access migrates `portal_rule` in
place: Hub creates `portal_entry`, groups the stored `match_scheme`,
`match_host`, and `match_port` values into entries, and indexes the rule path
within its entry. Hub stops reading those columns here and removes them in a
later release, because dropping a column of a database Hub does not own cannot
be undone; until then Hub keeps them filled with the entry access. An unset port
migrates to the port Portal serves. Two rules that only differed by an unset port
can share an entry and a path after the upgrade; Hub keeps the rule with the
explicit port and moves the rule that used the default port to a `/migrated`
path, and logs every move. An upgraded database is therefore never corrected by
hand, and Hub starts. A Hub rolled back to the release that predates entries
keeps reading and writing that database: the access columns it reads are still
there, the enable switch defaults to published, and a rule it inserts without an
entry joins its entry again on the next upgrade.

## Admin Display Strings

Skeleton, configuration-schema, and debug-schema display fields use strings.
Missing descriptions, deprecation reasons, examples, identifiers, access methods,
and permission codes are represented by an empty string, not null. Update and
debug-request fields retain optionality where omission has a separate meaning.
Regenerate custom Admin clients when adopting this contract; deploy the matching
Dashboard assets with Hub.

## Admin Payloads

Admin list methods return a `*ListItem` payload: the values the Dashboard shows
in a row, without the entity's stored provenance. Detail responses (`get`,
`create`, `update`) return the entity itself, including `fieldSources`, so a
Dashboard form can read back what it just wrote. App config detail is addressed
by `key`, because Hub also serves a configuration an application declares
without a value; `update` and `remove` keep addressing the stored row by `id`.

## Skeleton Generation

Hub maintains independent Skel source directories at `skel/control` and
`skel/admin`. Go code is generated into the matching
`api/skeled/control` and `api/skeled/admin` packages; TypeScript code is
generated only for admin into `src/dashboard/src/skeled/admin`. Use the
top-level script:

```bash
bash script/gen-skel.sh hub
```

Do not edit generated files directly. Modify the corresponding contracts under
`skel/control` or `skel/admin`, regenerate the corresponding code
with the script, and verify that all callers remain consistent.

## Inproc Mode

Hub can run as a component in a single-process runtime:

- The Hub Control API registers at `rpc+inproc://vine/hub`. The Admin API keeps
  its own listener instead, so an in-process Hub serves no Admin API unless its
  caller declares an admin address.
- `watchserver` does not open an external TCP port and retains only the in-process Watch server.
- `vined` keeps a pointer to that in-process Watch server so an inproc `WatchClient` can access it directly.

Hub retains its configuration-center and registry responsibilities; only the underlying exposure changes from network access to in-process access.

## TTL and Heartbeat

Hub handles registrations differently in normal and inproc modes:

- Normal mode
  - Application state and Rpc service registrations are written to Watch with a TTL.
  - Link continuously renews leases through heartbeat.
  - Hub's registry sweeper scans expired application leases, unregisters expired instances, and publishes delete events.
  - Watch key TTL is a fallback cleanup mechanism; the Hub sweeper publishes the actual registration-expiration events.

- Inproc mode
  - Application state and Rpc service registrations do not use a TTL.
  - `KeepAppStatus` and `KeepRpcServiceRegistration` become no-ops.
  - The registry sweeper does not start.
  - State remains valid until explicit unregister removes it.

This removes the need for heartbeat-based lease maintenance in single-process mode.
