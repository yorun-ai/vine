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
│   └── skeled/           Generated Hub types, clients, and schemas
├── skel/                 Hub skeleton definitions
└── src/
    ├── dashboard/        Dashboard frontend source
    └── server/           Hub server runtime
        ├── app/          Assembly of Hub components, modules, and servicers
        ├── comp/         Shared runtime components such as Redis and NATS servers
        ├── core/         Domain state and Core/Repo interfaces
        ├── flag/         Hub flags and default normalization
        ├── impl/         Implementations of externally exposed Hub services
        ├── mod/          Runtime modules such as initializer, seeder, and syncer
        └── repo/         Infrastructure adapters implementing Core Repo interfaces
```

## Dashboard Packaging

- During development, set `VINE_HUB_DASHBOARD_DEV_PROXY` to proxy requests directly to a running `pnpm dev` server.
- After changing Dashboard source, run `pnpm typecheck` and `pnpm build` in `src/dashboard`.
- Do not update the embedded `dashboard.tar.zst` during ordinary development. Rebuild it only when the task explicitly includes updating release assets.
- Keep user-facing text synchronized between `src/i18n/dictionaries/cn.ts` and `en.ts`.

The Dashboard source lives in `src/dashboard`. At runtime, Hub serves the build embedded in `src/server/impl/dashboard/assets/dashboard.tar.zst`.

When updating release assets, always run:

```bash
bash script/build-dashboard-assets.sh
```

The script runs `pnpm run build` in `src/dashboard` and packages `dist` as a new `dashboard.tar.zst`. The build generates `THIRD_PARTY_LICENSES.md` for dependencies included in the Dashboard bundle and includes it in the archive. Do not assemble the archive manually. Commit the updated archive whenever release assets are refreshed; otherwise, the embedded Hub Dashboard and its license inventory will remain stale.

## Layering and Change Constraints

Keep Hub's layer responsibilities distinct:

- `core` defines domain state and Repo interfaces without depending on concrete database or Redis implementations.
- `repo` implements persistence and Redis synchronization without owning external service orchestration.
- `impl` implements Hub services through `core` and `repo`.
- `mod` contains runtime flows such as initializer, seeder, syncer, scheduler, and sweeper.
- `comp` provides shared runtime components such as Redis and NATS.
- `app` only assembles components, modules, and servicers.

Additional constraints for Hub changes:

- Database schema changes must update both `src/server/repo/db/model/sql/sqlite` and `src/server/repo/db/model/sql/pgsql`.
- Redis keys, Redis value JSON, and event formats are protocols shared by Hub, Link, and Portal. Update every producer, consumer, and test together.
- `redisserver` is a runtime distribution layer. Do not turn it into a second source of business state that bypasses Repo/Core.
- Registration semantics differ between normal and inproc modes for TTL, heartbeat, and sweeper behavior. Validate both modes separately.

## Runtime Model

Hub has three primary responsibilities:

1. Configuration center
   Hub reads configuration from the database and exposes it through `AppConfigRepo`. During startup, `initializer` loads configuration into Redis for Link to read and subscribe to.

2. Service registry
   Link writes application state and Rpc service registrations to Hub. Hub persists them through `RegistryRepo` and exposes queries and heartbeat lease renewal.

3. Redis distribution layer
   `redisserver` maintains an in-memory Redis dataset. Configuration, application state, Rpc/Web endpoints, and schemas are synchronized into it. Link and Portal read snapshots and subscribe to change events through Redis.

## Configuration and Registration Sources

Hub currently supports two database backends:

- SQLite
- PostgreSQL

At startup, `--seed-yaml-file` can import initial configuration, site rules, and certificates from a local YAML file into the database. Hub still reads the imported state through its database repos and publishes it to Redis, preserving consistent read and subscription semantics for Link.

## Skeleton Generation

Hub Go code is generated into `api/skeled`, and Dashboard TypeScript code is generated into `src/dashboard/src/skeled`. Use the top-level script:

```bash
bash script/gen-skel.sh hub
```

Do not edit `api/skeled` or `src/dashboard/src/skeled` directly. Modify contracts under `skel/`, regenerate both Go and TypeScript code with the script, and verify that callers on both sides remain consistent.

## Inproc Mode

Hub can run as a component in a single-process runtime:

- Hub Rpc services register with the `inproc` transport instead of being exposed over HTTP.
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
