# Changelog

All notable changes to Vine are documented in this file.

The project follows [Semantic Versioning](https://semver.org/). The public
version history starts at `v0.9.0`; versions from the former private repository
are not part of the public compatibility commitment.

## [Unreleased]

### Changed

- Moved Kubernetes manifests from `examples/k8s` to `deploy/k8s`, with a
  version-pinned stable default and composable backend mTLS configuration
- Build and publish release binaries and container images in parallel after
  shared release validation
- Consolidated PR and main checks into one required CI gate, with path-selected
  Dashboard, Hub image, and workflow checks; main retains full race/shuffle checks
- Require successful main CI for the exact release commit, verify release
  archives and public multi-platform images before promoting `latest`, and
  support independent binary/image recovery without overwriting binary assets
- Isolate main CI runs by commit, serialize cross-version `latest` promotion,
  and add bounded transient retries and clearer release validation diagnostics
- Delegate dependency vulnerability monitoring and security updates to
  Dependabot instead of running `pnpm audit` and `govulncheck` in CI;
  retain secret scanning and third-party license checks

### Fixed

- Fixed release image metadata extraction from a detached checkout and added
  image-only publication for existing releases without overwriting binary assets

## [0.14.1] - 2026-09-04

### Changed

- Moved Hub, Link, and Portal image publication to GHCR under `ghcr.io/yorun-ai`.
  Images are published by the Release workflow after release binaries succeed,
  rather than on tag pushes; prereleases and older release rebuilds do not
  replace the `latest` image tag

## [0.14.0] - 2026-09-04

### Added

- `redis.Lock.TryUnlock()` for atomically checking local lock availability and
  attempting a token-checked Redis release, returning `false` for an unavailable
  or no-longer-owned lock while retaining fail-fast Redis command errors
- Added multi-stage Hub, Link, and Portal container image builds together with
  Kubernetes base manifests and an optional mTLS overlay; release tags publish
  all three images for Linux AMD64 and ARM64; Hub deployments must explicitly
  select exactly one of SQLite or PostgreSQL and one of embedded or external
  NATS

### Changed

- Raised the minimum Go version to 1.27.0, configured CI checks to follow the
  latest Go 1.27 patch, and pinned release binaries to Go 1.27.1
- Darwin release binaries now require macOS 13 or later, following Go 1.27's
  raised minimum deployment target for macOS
- Migrated Vine's JSON encoding, decoding, validation, formatting, and redaction
  to Go 1.27's stable `encoding/json/v2` and `encoding/json/jsontext` APIs,
  including their stricter handling of malformed and ambiguous JSON
- Changed the default `vcode` JSON and CBOR profiles to encode nil slices and
  maps as empty arrays and maps; supported schemas generated with skelc v0.14.x
  automatically retain the legacy `null` representation, while schemas
  generated with skelc v0.15.0 or later use the current behavior consistently
  across Rpc, Event, and Task
- Raised the minimum supported skelc version from v0.9.0 to v0.14.0 and
  regenerated Vine's built-in contracts with skelc v0.14.1
- Replaced Vine's direct use of `github.com/google/uuid` with Go 1.27's
  standard-library `uuid` package; `skel.NewUUID` now accepts the
  standard-library UUID type
- Migrated HTTP integration tests to Go 1.27's test-owned
  `httptest.NewTestServer` lifecycle while retaining loopback networking where
  reverse proxies, h2c transports, or connection upgrades require real sockets
- Applied a shared limit of 128 header values to application, Hub control,
  Link ingress, and Portal entry HTTP servers using Go 1.27's
  `http.Server.MaxHeaderValueCount`
- Adopted standard-library helpers available under the Go 1.27 baseline,
  including `strings.CutLast`, `strings.SplitSeq`, typed atomics,
  `sync.WaitGroup.Go`, `maps.Copy`, `min`, `slices.Contains`,
  `slices.Backward`, and `errors.AsType`, where they simplify parsing,
  concurrency lifecycles, collection operations, bounds, error inspection,
  test counters, and reverse-order cleanup;
  reflection code now uses type iterators, `reflect.Pointer`, and
  `reflect.TypeFor` where the type is static, and remaining `interface{}`
  spellings now use `any`; removed the redundant internal `PointerTo` helper
- Adopted Go 1.27 promoted-field composite literals, replaced the remaining
  `golang.org/x/exp/constraints` usage with standard-library `cmp.Ordered`, and
  removed redundant URL copies after `http.Request.Clone`
- Added isolated Go 1.27 `goroutineleak` profile checks for application HTTP,
  in-process Rpc and Web, scheduler, and Redis lock lifecycle tests
- Added low-cardinality application and Skel labels to Rpc, Event, and Task
  execution so Go 1.27 tracebacks and pprof profiles identify active handlers
- Replaced the package-level `testkit.NewClient`, `testkit.NewClientER`, and
  `redis.NewCache` functions with the generic methods `Execution.NewClient`,
  `Execution.NewClientER`, and `Redis.NewCache`, and simplified application
  construction through the generic process guard
- Reworked timer-, cancellation-, scheduler-, lock-, HTTP shutdown-, Link
  dispatch concurrency-, and in-process transport tests around Go 1.27
  `testing/synctest`, replacing wall-clock polling with deterministic
  synchronization, randomized test ordering, and tighter global state and
  in-process endpoint cleanup
- Made the default repository-wide test suite cache-friendly, moved targeted
  order randomization and goroutine lifecycle checks into a parallel CI job,
  and split full vet and module checks into their own parallel static-analysis
  job; added composable affected-area quick-test groups for local iteration,
  while main-branch CI retains full-suite shuffled execution
- Made Rpc, Web, and Link ingress in-process endpoint registries concurrency
  safe and lifecycle-owned through idempotent registration cleanup functions
- Added independently instantiable registries for domain schemas, configuration,
  actors, Rpc, events, tasks, and Web contracts while retaining the existing
  process-wide registration functions as default-registry facades
- Encapsulated the permanent process-wide application type and name creation
  guards with concurrent creation protection and failed-construction rollback

### Removed

- Removed serialization-based in-process Rpc cloning for service specs without
  generated clone hooks; methods with arguments or results must now provide the
  corresponding clone hook

### Fixed

- In-process Web round trips now reject already-canceled requests before
  invoking handlers and prefer cancellation when a response becomes ready at
  the same time
- Hub Dashboard copy actions now fall back to a temporary text area when the
  Clipboard API is unavailable or denied, including when serving the Dashboard
  over plain HTTP
- Removed the duplicate source field from standard-library JSON log records
  exposed by strict JSON v2 decoding
- Bounded encoded Rpc request bodies to 32 MiB and Rpc response bodies to
  128 MiB across application decoding, Link and Portal forwarding, Portal
  access-service calls, and Hub Service Debug, preventing unbounded buffering
  without affecting generic Web, SSE, or WebSocket streaming
- Redis locks now apply the existing infrastructure fail-fast policy when
  `Unlock()` cannot execute its Redis command or finds that its token no longer
  owns the lock; background refresh failures retain their causes on the lock
  context and mark the lock broken without panicking from the refresh goroutine

### Upgrade Notes

- Upgrade the build toolchain to Go 1.27.0 or later. Prebuilt Darwin binaries
  require macOS 13 or later.
- Replace `github.com/google/uuid.UUID` values passed to `skel.NewUUID` with Go's
  standard-library `uuid.UUID` type.
- Replace the removed package-level `testkit.NewClient`, `testkit.NewClientER`,
  and `redis.NewCache` calls with `Execution.NewClient`, `Execution.NewClientER`,
  and `Redis.NewCache` respectively.
- Regenerate Rpc contracts with skelc v0.14.0 or later so methods with arguments
  or results provide the required in-process clone hooks. Manually constructed
  `MethodSpec` values must set `CloneArguments` and `CloneResult` when their
  corresponding types are present.
- Use skelc v0.14.0 or later when regenerating contracts. Existing contracts
  generated with skelc v0.14.x remain wire-compatible through Vine's temporary
  schema-aware JSON and CBOR compatibility profile; regenerate contracts made
  by older skelc versions before upgrading Vine.
- JSON inputs containing duplicate object member names or invalid UTF-8 are now
  rejected by the stricter JSON v2 decoder; update producers that emit either
  form before upgrading.
- Direct users of `vcode` that depend on nil slices or maps encoding as `null`
  must select that behavior explicitly; the default representation is now
  `[]` or `{}`.

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
