# Changelog

All notable changes to Vine are documented in this file.

The project follows [Semantic Versioning](https://semver.org/). The public
version history starts at `v0.9.0`; versions from the former private repository
are not part of the public compatibility commitment.

## [Unreleased]

### Added

- `redis.Lock.TryUnlock()` for atomically checking local lock availability and
  attempting a token-checked Redis release, returning `false` for an unavailable
  or no-longer-owned lock while retaining fail-fast Redis command errors

### Changed

- Raised the minimum Go version to 1.27.0, configured CI checks to follow the
  latest Go 1.27 patch, and pinned release binaries to Go 1.27.1
- Replaced Vine's direct use of `github.com/google/uuid` with Go 1.27's
  standard-library `uuid` package; `skel.NewUUID` now accepts the
  standard-library UUID type
- Migrated HTTP integration tests to Go 1.27's test-owned
  `httptest.NewTestServer` lifecycle while retaining loopback networking where
  reverse proxies, h2c transports, or connection upgrades require real sockets
- Applied a shared limit of 128 header values to application, Hub control,
  Link ingress, and Portal entry HTTP servers using Go 1.27's
  `http.Server.MaxHeaderValueCount`
- Reworked timer-, cancellation-, scheduler-, lock-, and in-process transport
  tests around Go 1.27 `testing/synctest`, replacing wall-clock polling with
  deterministic synchronization, randomized test ordering, and tighter global
  state and in-process endpoint cleanup
- Made Rpc, Web, and Link ingress in-process endpoint registries concurrency
  safe and lifecycle-owned through idempotent registration cleanup functions
- Added independently instantiable registries for domain schemas, configuration,
  actors, Rpc, events, tasks, and Web contracts while retaining the existing
  process-wide registration functions as default-registry facades
- Encapsulated the permanent process-wide application type and name creation
  guards with concurrent creation protection and failed-construction rollback

### Fixed

- Redis locks now apply the existing infrastructure fail-fast policy when
  `Unlock()` cannot execute its Redis command or finds that its token no longer
  owns the lock; background refresh failures retain their causes on the lock
  context and mark the lock broken without panicking from the refresh goroutine

## [0.13.2] - 2026-08-20

### Changed

- Clarified that in-process Rpc guarantees request and result value isolation,
  not JSON/CBOR encoding, normalization, custom marshaling, or codec failure
  equivalence with network transports
- Raised the minimum Go toolchain to 1.26.6 to include the latest standard
  library security fixes

### Fixed

- Hub Service Debug now keeps the forwarding context alive until a remote Link
  response body is consumed, preventing independently linked H2C calls from
  failing with `context canceled`
- Link Rpc proxy forwarding now preserves the original response body so callers
  close the network body after buffering its contents
- Portal public entries now bound request-header processing, idle connections,
  and header size without imposing global read or write timeouts on streaming
  Web traffic

## [0.13.1] - 2026-08-14

### Changed

- Reduced successful in-process Rpc allocation and latency overhead by lazily
  building server log metadata and reusing the immutable OK error value
- Added optional generated request and result clone hooks for in-process Rpc,
  while retaining serialization-based cloning for older generated service specs

## [0.13.0] - 2026-08-13

### Changed

- The embedded Hub Dashboard now uses the current Vine branding and visual
  palette, with refreshed audited frontend dependencies
- Embedded NATS now provisions the Event and Task JetStream streams with memory
  storage, while Vine clients no longer select stream or consumer storage and
  require external NATS deployments to pre-provision both streams

### Upgrade notes

- External NATS deployments must create `VINE_EVENTS` for `event.>` with
  interest retention and `VINE_TASKS` for `task.>` with work-queue retention
  before starting Hub or Link; the deployment owns each stream's storage policy

## [0.12.0] - 2026-08-03

### Added

- Backend mTLS for Hub, Link, and Portal using exact SPIFFE X.509-SVID
  identities, including protected Hub Control/Admin APIs, embedded Redis and
  NATS, Link ingress, authenticated Redis role binding, and plaintext downgrade
  rejection for discovered backend endpoints
- Process-local temporary self-signed HTTPS certificates for Portal entries
  without a configured public certificate when backend mTLS is enabled, while
  preserving configured certificate precedence
- `app.NewBundled(...)` for running multiple applications in one lifecycle
  while they connect to an external Link
- `app/linked.Option` certificate fields and matching `--mtls-*-file` flags for
  authenticating an in-process Link to an mTLS-enabled external Hub

### Changed

- Hub now isolates the Link/Portal Control API from Dashboard admin Rpc
  and Web handlers on separate listeners; `--control-listen` defaults to the
  existing `127.0.0.1:7071`, while `--admin-listen` defaults to
  `127.0.0.1:7075`; Hub Redis remains on `127.0.0.1:7072`; the former Hub
  `--api-listen` flag and
  `VINE_API_LISTEN` environment variable have been removed
- Hub Skel contracts are split into the `vine.hub.control` domain for
  Link/Portal traffic and the `vine.hub.admin` domain for Dashboard
  administration; generated Go and TypeScript packages now use matching
  `skeled/control` and `skeled/admin` directories, and Hub Rpc service
  implementations are separated under `impl/control` and `impl/admin`
- Link continues to allow a non-loopback App API listener for unusual
  deployments, but now logs a warning because cross-host App-to-Link traffic is
  unauthenticated h2c and is not the expected sidecar topology
- The English and Simplified Chinese READMEs now provide a complete runtime
  architecture, deployment-mode comparison, CLI guide, public package map,
  ecosystem overview, and production boundary summary
- Release builds now require a dated `CHANGELOG.md` heading matching the
  release tag before producing or uploading binaries

### Fixed

- Redis locks now reject non-positive timeouts, bound refresh commands
  and retries to the remaining lease, and stop immediately after ownership is
  lost
- Redis snapshot subscriptions now hold events behind a publication barrier
  until Link and Portal install the corresponding snapshot, preventing stale
  local state from surviving the initialization window
- Link configuration and RpcProxy state, together with Hub Syncer caches, now
  remain protected from concurrent map access and mutable state escaping its
  lock
- Hub Scheduler jobs now contain background panics and transient NATS errors,
  preserve the last valid schedule after an invalid refresh, and wait for the
  refresh loop and in-flight jobs during shutdown
- HTTP servers now force-close active connections after graceful shutdown times
  out, and failed embedded NATS startup removes its temporary JetStream store
  and shuts down any partially started server

### Upgrade notes

- Hub Skel names have moved from `vine.hub.*` to either
  `vine.hub.control.*` or `vine.hub.admin.*`. Clients using generated Hub
  contracts must regenerate or update their imports and service paths.
- Deployments using the former Hub `--api-listen` flag or `VINE_API_LISTEN`
  variable must configure the Control and Admin listeners separately.
- Linked applications that connect to an mTLS-enabled Hub must configure the
  Link identity through `linked.Option.MTLSCAFile`, `MTLSCertFile`, and
  `MTLSKeyFile`, or through the matching CLI flags and environment variables.
- Applications creating fixed Redis locks must provide a positive timeout;
  zero and negative values are rejected so every lock key has a lease.

## [0.11.0] - 2026-08-02

### Added

- `vine dev` for running Hub and Portal in process while keeping a network Link
  API available to separately running business applications
- Automatic temporary SQLite storage for `vine dev` when no database is
  configured, with cleanup after graceful shutdown

### Changed

- The embedded Hub Redis server now rejects anonymous data commands and uses
  separate `vine.hub`, `vine.link`, and `vine.portal` users with least-privilege
  command, key, scan, and subscription ACLs
- Graceful shutdown now runs on the caller goroutine so lifecycle hook panics
  are visible to the lifecycle owner
- Link unregistration failures are logged while local shutdown cleanup
  continues on a best-effort basis
- Embedded JetStream temporary storage now uses the operating system's selected
  temporary directory instead of a hard-coded `/tmp` base
- The required CI gate now aggregates the full test, race, security, license,
  and Dashboard checks under a stable check name

### Upgrade notes

- Custom clients that connect directly to the embedded Hub Redis endpoint must
  authenticate with an allowed Vine Redis user before issuing data commands.
  Built-in Hub, Link, and Portal clients already follow the new protocol.
- `vine.link` and `vine.portal` currently use empty passwords only to select
  their ACL roles. This does not authenticate the caller, so the Redis endpoint
  must remain on a trusted network until component transport authentication is
  implemented.
- `StopGracefully()` keeps its existing public signature and timeout ownership,
  but a panic raised by a lifecycle hook now propagates on the calling
  goroutine.

## [0.10.1] - 2026-07-29

### Added

- Deprecated status and reason metadata in Vine schemas, Hub APIs, and the Hub
  Dashboard

### Changed

- Documentation links now target the public Vine site and its English and
  Simplified Chinese locale paths
- UUID map keys have explicit JSON and CBOR compatibility coverage for Skel
  code generation

## [0.10.0] - 2026-07-26

### Added

- Structured sensitive-data redaction for Rpc, Event, and Task lifecycle logs,
  including Skel metadata and bounded binary summaries
- Rpc, Event, and Task invocation logs with application and subsystem log-level
  controls

### Changed

- Error and panic stack capture now preserves the most relevant originating
  stack, and framework logger names use consistent `vine:core` and `vine:infra`
  prefixes

### Fixed

- Container methods are resolved on the actual target instance type
- Release binaries are built directly from the checked-out release tag

## [0.9.3] - 2026-07-22

### Fixed

- Invalid credentials now map to an unauthorized response
- Raised system errors preserve their original stacks
- Web errors map to the corresponding HTTP status
- Dependencies with known vulnerabilities were updated

## [0.9.2] - 2026-07-21

### Fixed

- Internal runtime suffixes are no longer exposed in Rpc server identity
  headers

## [0.9.1] - 2026-07-21

### Added

- Automated publication of Vine CLI binaries for macOS and Linux on amd64 and
  arm64

### Changed

- The Go toolchain baseline is Go 1.26.5 or later

### Fixed

- Long-lived Web upgrade streams use traffic-aware idle timeouts

## [0.9.0] - 2026-07-21

Initial public release.

### Included

- Application, component, and module lifecycle management
- Type-based dependency injection and execution scopes
- Rpc, Web, Event, Task, configuration, Redis, and relational database support
- Standalone, linked, and separated Hub, Link, Portal deployment modes
- Skel-powered Go and TypeScript contracts
