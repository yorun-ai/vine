# Vine Hub

**English** | [简体中文](README.zh-CN.md)

Hub is Vine's configuration and service registry. It broadly follows a DDD-style layered architecture, maintains configuration, application state, and Rpc service registrations, and exposes read and subscription capabilities through Redis.

## Directory Structure

```text
internal/daemon/hub/
├── api/                  Public APIs exposed by Hub to other runtime components
│   ├── app/              Shared constants such as the Hub inproc endpoint
│   ├── nats/             Hub NATS inproc access
│   ├── redis/            Hub Redis client, events, and inproc access
│   ├── redised/          Redis value structures and key formatting
│   └── skeled/           Generated control/admin Go packages
├── skel/                 Control and admin skeleton definitions
└── src/
    ├── dashboard/        Dashboard frontend source
    └── server/           Hub server runtime
        ├── app/          Assembly of Hub components, modules, and servicers
        ├── comp/         Shared runtime components such as Redis and NATS servers
        ├── core/         Domain state and Core/Repo interfaces
        ├── flag/         Hub flags and default normalization
        ├── impl/         Implementations split by exposed API boundary
        │   ├── control/  Link/Portal-facing Control API services
        │   └── admin/    Dashboard admin Rpc and Web services
        ├── mod/          Runtime modules such as controlapi, initializer, seeder, and syncer
        └── repo/         Infrastructure adapters implementing Core Repo interfaces
```

## Dashboard Packaging

- During development, set `VINE_HUB_DASHBOARD_DEV_PROXY` to proxy requests directly to a running `pnpm dev` server.
- After changing Dashboard source, run `pnpm typecheck` and `pnpm build` in `src/dashboard`.
- Do not update the embedded `dashboard.tar.zst` during ordinary development. Rebuild it only when the task explicitly includes updating release assets.
- Keep user-facing text synchronized between `src/i18n/dictionaries/cn.ts` and `en.ts`.

The Dashboard source lives in `src/dashboard`. At runtime, Hub serves the build embedded in `src/server/impl/admin/dashboard/assets/dashboard.tar.zst`.

When updating release assets, always run:

```bash
bash script/build-dashboard-assets.sh
```

The script runs `pnpm run build` in `src/dashboard` and packages `dist` as a new `dashboard.tar.zst`. The build generates `THIRD_PARTY_LICENSES.md` for dependencies included in the Dashboard bundle and includes it in the archive. Do not assemble the archive manually. Commit the updated archive whenever release assets are refreshed; otherwise, the embedded Hub Dashboard and its license inventory will remain stale.

## Layering and Change Constraints

Keep Hub's layer responsibilities distinct:

- `core` defines domain state and Repo interfaces without depending on concrete database or Redis implementations.
- `repo` implements persistence and Redis synchronization without owning external service orchestration.
- `impl/control` implements only the Link/Portal-facing Control API services,
  while `impl/admin` and its `debug` and `dashboard` subpackages implement
  the Dashboard admin surface through `core` and `repo`.
- `mod` contains runtime flows such as the Control API listener, initializer,
  seeder, syncer, scheduler, and sweeper.
- `comp` provides shared runtime components such as Redis and NATS.
- `app` only assembles components, modules, and servicers.

### Domain Writes

Configuration, site, rule, and certificate writes go through their corresponding
Core. `Validate` checks and normalizes a complete entity without accessing
storage; `Save` creates or replaces by name and owns identity handling, along
with versioning and built-in protection where applicable. API updates merge
provided fields into the existing entity before validation.

Seeder and Dashboard imports validate all supplied entities before writing,
then call Core `Save`. Validation does not make an entire import transactional:
a database failure can still leave some entities saved. The YAML conversion
layer maps configuration fields only; it does not assign database identity or
manage versions.

`PortalSiteCore.EnsureDashboardSite` and `PortalRuleCore.EnsureDashboardRule`
own built-in Dashboard provisioning. `RegistryCore` owns schema registration
and expired-lease removal. Initializer and Sweeper coordinate runtime publication
through Syncer. The seed-applied marker remains startup bookkeeping in Seeder.

### Change Constraints

Additional constraints for Hub changes:

- Database schema changes must update both `src/server/repo/db/model/sql/sqlite` and `src/server/repo/db/model/sql/pgsql`.
- Redis keys, Redis value JSON, and event formats are protocols shared by Hub, Link, and Portal. Update every producer, consumer, and test together.
- `redisserver` is a runtime distribution layer. Do not turn it into a second source of business state that bypasses Repo/Core.
- Registration semantics differ between normal and inproc modes for TTL, heartbeat, and sweeper behavior. Validate both modes separately.

## Runtime Model

Hub has four primary responsibilities:

1. Configuration center
   Hub reads configuration from the database and exposes it through `AppConfigRepo`. During startup, `initializer` loads configuration into Redis for Link to read and subscribe to.

2. Service registry
   Link writes application state and Rpc service registrations to Hub. Hub persists them through `RegistryRepo` and exposes queries and heartbeat lease renewal.

3. Redis distribution layer
   `redisserver` maintains an in-memory Redis dataset. Configuration, application state, Rpc/Web endpoints, and schemas are synchronized into it. Link and Portal read snapshots and subscribe to change events through Redis.

   The embedded Redis protocol requires authentication before any data command. It defines three users with resource-level ACLs:

   - `vine.hub` has full command and key access. Its password is generated randomly for the current process.
   - `vine.link` can read configuration, Rpc endpoint registrations, and the revision key; it can subscribe only to configuration channels and Rpc registration patterns.
   - `vine.portal` can read Portal rules, sites, certificates, actor/service/resource schemas, Rpc/Web endpoint registrations, and the revision key; it can subscribe only to the corresponding list patterns.

   Link and Portal use empty Redis passwords for in-process mode and separated-deployment debugging. With backend mTLS enabled, the client certificate authenticates the caller and binds its SPIFFE identity to the matching Redis username. Without mTLS, the usernames only select least-privilege roles and do not authenticate the caller, so the Redis endpoint must remain on loopback or a trusted private network protected by a firewall.

4. Separated API listeners
   The Control API listener exposes the `vine.hub.control` domain, containing
   only `InfoService` and `RegistryService`, to Link and Portal. The main Hub
   listener exposes the `vine.hub.admin` domain containing Dashboard
   admin Rpc services and `DashboardWeb`. This keeps component traffic
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

At startup, `--seed-yaml-file` imports initial configuration, Portal sites,
rules, and certificates from local YAML into the database. Hub reads this state
through its repos and publishes it to Redis for Link and Portal.

Portal rule YAML uses flat fields in this order: `matchScheme`, `matchHost`,
`matchPort`, `matchPathPrefix`, `routeType`, `routeSiteName`,
`routeRedirectionPattern`, and `routePathPrefix`.

The `mod/seeder` package owns the shared YAML compatibility decoder used by
startup seeding and Dashboard imports. Both accept legacy rule fields with a
warning per field; mixing old and new fields in one rule fails before applying
imported data. YAML cannot replace built-in Dashboard sites or rules.

Admin API and Redis use only the new fields; upgrade Hub and Portal together.
Existing database columns are migrated to matching `match_*` / `route_*` names.

## Skeleton Generation

Hub maintains independent Skel source directories at `skel/control` and
`skel/admin`. Go code is generated into the matching
`api/skeled/control` and `api/skeled/admin` packages; TypeScript code is
generated into matching directories under `src/dashboard/src/skeled`. Use the
top-level script:

```bash
bash script/gen-skel.sh hub
```

Do not edit generated files directly. Modify the corresponding contracts under
`skel/control` or `skel/admin`, regenerate both Go and TypeScript code
with the script, and verify that callers on both sides remain consistent.

## Inproc Mode

Hub can run as a component in a single-process runtime:

- The Hub Control API registers at `rpc+inproc://vine/hub`, while Dashboard
  admin Rpc and Web handlers register below
  `rpc+inproc://vine/hub/admin` and
  `web+inproc://vine/hub/admin` instead of being exposed over HTTP.
- `redisserver` does not open an external TCP port and retains only the in-process Redis server.
- `vined` keeps a pointer to that in-process Redis server so an inproc `RedisClient` can access it directly.

Hub retains its configuration-center and registry responsibilities; only the underlying exposure changes from network access to in-process access.

## TTL and Heartbeat

Hub handles registrations differently in normal and inproc modes:

- Normal mode
  - Application state and Rpc service registrations are written to Redis with a TTL.
  - Link continuously renews leases through heartbeat.
  - Hub's registry sweeper scans expired application leases, unregisters expired instances, and publishes delete events.
  - Redis key TTL is a fallback cleanup mechanism; the Hub sweeper publishes the actual registration-expiration events.

- Inproc mode
  - Application state and Rpc service registrations do not use a TTL.
  - `KeepAppStatus` and `KeepRpcServiceRegistration` become no-ops.
  - The registry sweeper does not start.
  - State remains valid until explicit unregister removes it.

This removes the need for heartbeat-based lease maintenance in single-process mode.
