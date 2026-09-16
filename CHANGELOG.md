# Changelog

All notable changes to Vine are documented in this file.

The project follows [Semantic Versioning](https://semver.org/). The public
version history starts at `v0.9.0`; versions from the former private repository
are not part of the public compatibility commitment.

## [Unreleased]

### Added

- Hub's Portal entry service can create an entry for an access and delete an
  entry that routes no rule, and the entry list returns an entry that routes no
  rule so it stays selectable while the operator adds the rules that use it. The
  Dashboard Portal entry page gains New and Delete actions and shows an entry
  that routes nothing.
- The Portal rule interface no longer carries an access: `PortalRuleCreation`
  names the Portal entry a rule belongs to, and `PortalRuleUpdate` has no
  protocol, host, or port at all, so only the entry page changes an access. The
  Dashboard rule editor selects an entry when it creates a rule, displays the
  entry while editing, and links to the entry page from the rule detail, which
  names the entry instead of repeating its protocol, host, and port. Portal site,
  rule, and certificate details render their values as text instead of
  input-like boxes, so a read-only field no longer looks editable.

### Changed

- Hub stores Portal access entries instead of deriving them from rules. An entry
  owns the scheme, host, and port Portal serves, and rules reference it, so
  changing an entry access updates one row rather than every rule that used it.
  Portal continues to receive rules carrying the access of their entry, and seed
  YAML keeps declaring `matchScheme`, `matchHost`, and `matchPort` on rules; Hub
  aggregates the declared access into entries while it applies the seed.
  Upgrading a database groups the stored rule access into entries and drops the
  rule access columns. Rules that only differed by an unset port can now share an
  entry and a path, so Hub keeps the rule with the explicit port on its path and
  moves the rule that used the default port to a `/migrated` path, logging each
  move instead of refusing to start on stored data. A seed, Dashboard import, or
  Admin write that declares a request another rule already serves fails with the
  rule that serves it, so Hub never rewrites a path the operator did not ask for;
  a seed that declared both an unset and an explicit default port for one path
  must drop one of them, because Hub no longer stores one request twice. A rule
  that leaves `matchPort` unset reports the port Portal serves (`80` or `443`) in
  Admin API responses instead of `0`. Regenerate custom Admin clients and deploy
  the matching Dashboard assets with this release: `PortalRuleCreation` and
  `PortalRuleUpdate` changed.

## [0.19.0] - 2026-09-16

### Added

- Portal uses a Web's declared mount path as both the match and forwarding
  prefix of rules targeting that site, overriding configured rule prefixes.
  Site and schema changes refresh the effective paths automatically. Dashboard
  shows both fields with the Web path, disables editing, and identifies Web as
  their source. Seed rules can omit both prefixes; existing stored values need
  no migration and apply again if the Web has no mount path. Built-in Dashboard
  access and redirect rules are unchanged. During a rolling upgrade, upgrade
  Hub first and verify that it publishes the resolved rule prefixes before
  upgrading Portal; the new Portal depends on those fields.

### Changed

- The Hub admin API returns field sources with the detail of an entity instead
  of through a separate query, and separates the list payload from the detail
  payload for app configs, Portal sites, rules and certificates. An app config
  detail is selected by key rather than by database id, so the Dashboard can
  show a configuration an application declared but Hub has no value for. The app
  config list carries only the identity of the configuration and its schema
  instead of the configuration JSON and the complete schema.
- The Dashboard seed preview compares every field the seed declares. A Portal
  site whose seed changes its CORS mode or origins now reports those fields as
  changed instead of applying them silently.

## [0.18.0] - 2026-09-15

### Fixed

- Keep the Hub image's legacy watch-listener environment default so existing
  `VINE_REDIS_LISTEN` and `--redis-listen` overrides continue to work. Explicit
  `VINE_WATCH_LISTEN` and `--watch-listen` inputs still take precedence.

- Hub, Link, and Portal identify themselves with the Vine runtime version in
  Rpc metadata, including when embedded in a business application. Identity
  headers always carry a plain semantic version such as `0.17.0`, because the
  vRPC identity format rejects the Go module form `v0.17.0`. Peers that still
  send the module form remain accepted, so mixed-version clusters keep working.

- Application versions are validated as full semantic versions with an optional
  leading `v`, so an incomplete version such as `1.2` or `01.2.3` is rejected
  when the application is created instead of producing an identity header that
  other components cannot read. Before upgrading, replace such historical
  versions with full semantic versions, for example `1.2.0` or `1.2.3`.
  The stricter validation also applies to received application identities.

- Link and Portal recover from a Hub restart that advertises different endpoints
  instead of requiring a restart of their own. Link re-reads Hub information
  when a heartbeat reports that Hub no longer knows the application instance,
  and reconnects the watch, MQ and lock clients only when an advertised endpoint
  actually moved. Re-established watch subscriptions are reconciled, so keys
  that changed while the endpoint was stale are reported. Portal re-reads Hub
  information on a timer, so it notices a moved watch endpoint without a
  restart.

### Added

- Hub tracks running Portal daemons. Portal registers its instance ID and Vine
  runtime version on startup, heartbeats every 10 seconds and unregisters on
  shutdown; Hub drops a Portal instance 30 seconds after its last heartbeat, so a
  terminated Portal stops being reported. The Hub Dashboard lists the registered
  Portal instances next to the application instances. Standalone Portal shares
  Hub's process, so it registers without a heartbeat. Hub forgets Portal
  instances when it restarts and each Portal registers again on its next
  heartbeat. During a rolling upgrade, upgrade Hub before Portal; a new Portal
  continues running against an older Hub, but retries the unavailable Portal
  registry service and logs a warning on each 10-second heartbeat.

- Hub connection information includes its Vine runtime `version`.

- Hub provides an embedded in-memory lease lock service by default. Configure
  `--lock-mode=embedded|redis|disable`; Redis mode requires `--lock-redis-endpoint`,
  which Hub advertises through its connection information. Inproc Hub always uses
  embedded locks. Active locks reject repeated acquisition, including the same token.

- Link exposes lease lock operations through its LockService, routing requests to
  Hub or directly to the advertised external Redis endpoint.
- Applications can inject `core/lock.Locker` for application-scoped locks or
  `core/lock.UniversalLocker` to share locks across applications using the same
  lock backend.

### Changed

- Configure Hub MQ using `--mq-mode=embedded|nats` (default: `embedded`) and
  `--mq-nats-endpoint` (environment: `VINE_MQ_MODE`, `VINE_MQ_NATS_ENDPOINT`).
  Embedded mode rejects an endpoint; nats mode requires one. The published
  `mq-embedded-nats` and `mq-external-nats-url` inputs remain accepted at the CLI
  boundary with a deprecation warning; explicit new inputs take precedence.

- Hub MQ information explicitly reports `mqEmbedded`, `mqNatsPort`, and
  `mqNatsEndpoint`. Link prefers these fields while retaining compatibility
  with `natsPort` and `mqEndpoint` from older Hubs.

- Configure the Hub watch listener with `--watch-listen` or `VINE_WATCH_LISTEN`.
  The deprecated `--redis-listen` and `VINE_REDIS_LISTEN` inputs remain accepted
  with a warning; explicit watch inputs take precedence.

- Hub connection information now advertises `watchPort` for configuration and
  service discovery. Link and Portal prefer it and fall back to `redisPort`
  when connecting to older Hubs. Hub retains the deprecated `redisPort` field
  for existing clients.

## [0.17.0] - 2026-09-14

### Fixed

- Prefer Portal TLS certificates within their validity period when selecting
  among matching certificates.

- Accept nil values for nullable Task arguments, including strings, lists, and
  maps, during launch and execution. Preserve null versus empty collections
  while continuing to reject a nil argument object.

### Changed

- Remove Rpc `MethodSpec.ValidateArguments` / `ValidateResult` callbacks and
  their `MethodInfo` methods. Supported skelc versions no longer generate these
  hooks; request decoding and argument/result cloning remain unchanged.

- Require Vine v0.15.7 or later as the database upgrade baseline. Remove old
  Portal rule column renaming, route-path column migration, and Portal site
  CORS column migration. Start older
  databases with v0.15.7 first to complete migration before upgrading.

- Require contracts generated by skelc v0.17.1 or later. Regenerate older
  contracts before upgrading. Remove legacy null-collection encoding, Rpc
  `arg:"n"` tags, `skel.PermissionCode` and `TypeKindSkelPermissionCode`, and
  implicit permission-check parameter names. Manual contracts must use
  `skel:"index(n)"`, string permission codes, and explicit `CodeArgumentName`.

- Hub seed inputs use `SeedHubData`, `SeedHubSource`, and their `File` forms,
  plus `SeedHubVarsFile`. CLI flags and environment variables use
  `seed-hub-{data,source,vars}-file` and `VINE_SEED_HUB_{DATA,SOURCE,VARS}_FILE`.
  All old seed input names are removed, including `--seed-yaml-file`,
  `VINE_SEED_YAML_FILE`, and the `SeedYAMLFile` Go option; update callers to
  the corresponding Hub data names.

- Hub seed is applied only once, as recorded in database metadata. Later starts
  skip seed, variable, and source files entirely. Removed the per-item `override`
  flag; use Hub configuration updates for persistent databases. No-db mode
  continues to seed its fresh in-memory store on every start.

### Added

- Field sources retain the original field template and resolved variable
  bindings, including substitution paths and whether defaults were used. Hub
  exposes these records to Dashboard comments and tooltips; editing a field
  clears its obsolete template and bindings.

- Seed deployment variables and optional field source maps through `--seed-hub-vars-file`
  and `--seed-hub-source-file`. Standalone can embed seed and source maps together;
  variables are always supplied by file. Hub stores
  field origins with configuration records and exposes them in the Dashboard.
  SQLite/PostgreSQL store field sources in a separate `field_source` table keyed by entity kind and ID.
- Nested camelCase variable paths (`${database.port}`) and missing-key defaults
  (`${database.port:5432}`). Registered `app.Vars` schemas validate referenced
  variables, and registered config schemas validate whole-object substitutions.
  Explicit null and zero values do not select defaults; unused variables are
  not required and extra object fields are ignored.

## [0.16.0] - 2026-09-13

### Added

- `--no-db` for Hub, standalone, and `vine dev`: initialize configuration from
  seed without a persistent database. App configs, Portal sites, rules, and
  certificates become read-only, with corresponding Dashboard guidance.
- Unified JSON5/YAML configuration editor with type-aware controls and validation.
- Enum keys and values in configuration maps.
- Structured YAML app config seed values; existing JSON string values remain supported.
- Standalone `Option.SeedYAML` for embedded seed content, mutually exclusive with
  `SeedYAMLFile`.

### Changed

- Hub, standalone, and `vine dev` default to `--no-db` when no database is
  specified and require a seed source. Update that source and restart to apply
  changes; select SQLite/PostgreSQL explicitly to retain writable persistence.
- YAML seeds and editing reject anchors, aliases, merge keys, numeric separators,
  non-decimal bases, scientific notation, and ambiguous leading zeros. Expand
  references and use ordinary decimal numbers when migrating existing seeds.

### Fixed

- Prevent unsafe integer saves and improve configuration validation errors.

## [0.15.8] - 2026-09-13

### Fixed

- Release archives embed the exact tagged Go module version and clean source
  revision, instead of an untagged pseudo-version.

- Embed the IANA timezone database in the Vine binary so cron expressions with
  `CRON_TZ`, such as `CRON_TZ=Asia/Shanghai 0 1 * * *`, and database timestamp
  scanning resolve named locations inside the Hub, Link, and Portal images,
  which do not ship a system tzdata package.

## [0.15.7] - 2026-09-09

### Added

- Add `ApplicationSpec.InitHooks` with injected callbacks for `BeforeAppStart`,
  `AfterAppStart`, `BeforeAppStop`, and `AfterAppStop`, without a dedicated
  module or component. App callbacks surround the component and module lifecycle;
  `BeforeAppStart` may return an error to abort startup.

## [0.15.6] - 2026-09-09

### Added

- Add `rpc.WithDestination(appName)` to route backend Rpc calls only to
  instances of the named application, with independent round-robin selection.
  The name must be non-empty; an unavailable destination returns
  `ServiceUnavailable` without falling back to another application.
- Carry the destination as a private App-to-Link extension in `vrpc-options`.
  Link consumes it and forwards other options; Portal strips unknown options.
  Upgrade Link before using the option, since older versions do not support it.

## [0.15.5] - 2026-09-09

### Changed

- Migrate internal Hub Admin services to explicit API contracts and update the
  bundled Dashboard together. Existing application contracts remain supported.
- Generate Vine contracts with skelc v0.18.1 in strict mode. Application control
  services remain backend public services.
- Show API badges in Hub service lists, details, and debug selectors. Hub Debug
  retains direct invocation with supplied Actor identity and explains that it
  skips Portal authentication and admission checks.
- Upgrade `go.yorun.ai/vrpc` to v0.12.0.

## [0.15.4] - 2026-09-09

### Upgrade notes

- Upgrade Hub, Link, Portal, and application Vine dependencies before deploying
  contracts that mark services as API. Existing generated contracts remain
  supported, including legacy services with client rules.

### Added

- Add `rpc.Client.InvokeAs[T]` for typed results with the existing invocation options and error behavior.

### Changed

- Reuse scalar types from `go.yorun.ai/vrpc/skel` v0.11.0 through the existing Skel API.
- Restrict explicit API services to Portal calls: publish client-facing schemas to Portal and reject backend API calls at Link. Legacy services with client rules retain both paths.
- Reuse the HTTP transport from `go.yorun.ai/vrpc` for request/response envelopes, protocol helpers and round-trip handling while retaining Vine schema encoding and error mapping.
- Refresh generated contracts with skelc v0.17.1 and update the third-party license inventory.

## [0.15.3] - 2026-09-08

### Changed

- Update Go runtime dependencies, including SQLite, PostgreSQL, NATS, Redis,
  CBOR, compression, CLI parsing, and networking libraries, and refresh the
  bundled third-party license inventory.

## [0.15.2] - 2026-09-08

### Upgrade notes

- Upgrade Portal before sending requests that omit optional credential fields.
  Generate nullable credentials with skelc v0.17.1 or later. Required fields
  and explicitly supplied optional fields must be non-empty; requests containing
  an empty credential value are now rejected.
- Hub bundles the matching Dashboard assets. Custom Admin clients should
  regenerate their contracts for display fields changing from nullable strings
  to strings; absent display values are now empty strings.

### Changed

- Hub Admin schema display strings now return an empty string instead of null
  when absent. This includes descriptions, deprecation reasons, examples,
  Actor identifier fields, access methods, and permission codes. Optional update
  and debug-request parameters are unchanged.

### Fixed

- Hub SQLite startup no longer mistakes current Portal rule column names for
  legacy columns during schema migration.

### Added

- Portal authentication accepts omitted nullable credential fields. Required
  fields and any supplied optional fields must contain non-empty values; at
  least one credential value is required. Regenerate actor schemas with
  skelc v0.17.1 or later to use `string?` credential declarations.

- Hub Dashboard Actor details show the authentication realm and Info identifier
  field. Embedded Dashboard assets include the updated Admin contracts.
- `vstring.Optional` converts an empty string to nil and preserves non-empty
  strings, including whitespace, as pointers.

## [0.15.1] - 2026-09-07

### Upgrade notes

- The minimum supported skelc version remains v0.14.0. Using `@identifier`
  requires a matching skelc release and regenerated actor Info types and schemas.
- Upgrade forwarding runtimes together when relying on actor identity
  propagation. Receivers use the transmitted realm and identifier; headers
  without those fields retain empty values.

- Internal App, Link, and Hub contracts are now generated with skelc v0.16.0.
  Nil collections use empty arrays/maps on JSON and CBOR boundaries, and
  generated collection non-null validation is removed. Clients consuming these
  infrastructure APIs must accept the current collection representation.

### Changed

- Regenerate built-in contracts with skelc v0.16.0 while retaining the minimum
  supported business-contract compiler version at v0.14.0.

### Added

- Add `Actor.Realm()` and `Actor.Identifier()`, schema-driven identity extraction in Portal and Hub debug calls, tag-based identity extraction for `NewAuthenticatedActor(info)` without floating-point conversion, and identity propagation.

### Fixed

- Preserve backend error reasons in Portal Rpc responses when authentication,
  actor permission service calls, or resource permission checks fail.

## [0.15.0] - 2026-09-07

### Upgrade notes

- Upgrade Hub and Portal together: Portal rule fields changed in both the Admin
  API and Redis. Update custom Admin clients to the new `match*` / `route*`
  fields. Existing application binaries do not need to be rebuilt for this
  infrastructure upgrade.
- Back up the Hub database before upgrading. Hub automatically renames legacy
  rule columns and adds the route path prefix column; reverting only the Hub
  binary does not reverse this migration. Do not run old and new Hub versions
  against the same database.
- Legacy rule YAML remains accepted with warnings, but a single rule must not
  mix legacy and current field names. Stricter validation may reject previously
  accepted inputs when they are saved or imported again.
- Applications that upgrade their Vine dependency must review configuration
  values that intentionally contain leading or trailing whitespace. Use
  `skel:"noTrim"` on those config fields; `sensitive` alone does not preserve
  whitespace. The `.skel` annotation and generator support are not included.
- Upgrade Portal before publishing schemas with a custom permission code
  argument name. Existing schemas continue to use `code`, and existing
  generated argument tags remain supported.
- Update deployment scripts that reference `examples/k8s` to use `deploy/k8s`.

### Added

- Read comma-separated `skel` struct tag attributes: config fields can use
  `noTrim` to preserve string whitespace, including nullable and collection
  values; Rpc argument fields can use `index(n)` in place of `arg:"n"`.
  Legacy argument tags remain supported, and conflicting or invalid indexes
  fail registration. `sensitive` continues to redact when combined with either
  attribute, including values with custom JSON marshalers. Upgrade Vine before
  generating combined tags; Skel syntax and generator support are separate work.

- Added `skel.PermCheckInvocation.CodeArgumentName` so Portal can inject a
  resource permission code into a schema-selected argument. An omitted or empty
  name retains the legacy `code` argument; custom names allow business arguments
  named `code` without overwriting the injected permission code. Upgrade Portal
  before publishing schemas that use a custom name.

- Add `vstring.TrimSpacePtr` for nil-safe string trimming and
  `vstring.FirstNonBlank` for selecting the first trimmed non-blank value.

- Infer PostgreSQL UUID and SQLite TEXT columns for untagged `uuid.UUID` and
  `*uuid.UUID` fields on RDB connections, preserving explicit type/serializer tags.

- Add `rdb.UModel` and `rdb.UDeletableModel` with automatically generated UUIDv7
  primary keys using Go’s `uuid.UUID` (PostgreSQL `uuid`, SQLite `TEXT`), sharing the existing database, DAO, and query implementation.
  Model constraints use a marker method instead of unused integer ID accessors;
  existing integer-key models and storage remain unchanged.
- Generate UUID primary keys in a GORM create callback before user hooks, without
  requiring models to chain an embedded BeforeCreate method.
- DAO queries normalize UUID primary-key shorthand, map conditions, and UUID
  lists before GORM parses conditions, without mutating caller arguments.
- UUID serialization supports nullable `*uuid.UUID` fields, preserving the
  distinction between SQL NULL and the zero UUID.
- RDB connections automatically bind Go `uuid.UUID` parameters for direct and
  prepared SQL, including transactions and UUID lists expanded by GORM.

- Export `app.ManagedComponent`, `BaseManagedComponent`, `ComponentManager`,
  `BaseComponentManager`, and `ComponentLifecycle` for external component
  implementations. Internal `FrameworkComponent` names are now `ManagedComponent`;
  existing Redis and RDB embedding APIs retain their behavior.

- Public DI bindings support `WithDependencies` to declare additional dependencies
  and run a callback before the constructed instance is returned to consumers.

- Portal SITE entry rules support `routePathPrefix` to replace the matched path
  prefix before forwarding. Empty values retain existing behavior; Hub migrates
  existing rule tables automatically. Upgrade Portal before enabling non-empty
  target paths.

### Changed

- Breaking: configuration reads now trim leading and trailing Unicode whitespace
  from string fields, nullable strings, list elements, and map values for both
  lifecycles. Map keys, JSON contents, and named scalars are preserved. Review
  existing whitespace-sensitive values before upgrading; config fields can opt
  out with `skel:"noTrim"`.

- Upgrade GORM from v1.31.1 to v1.31.2.

- Configuration versions now advance only when values change. Certificate issuer,
  domains, and validity dates are derived from certificate content across the
  Admin API, startup seeds, and Dashboard imports.
- Configuration, site, rule, and certificate inputs are validated consistently
  across the Admin API and YAML imports. Imports validate all supplied items
  before writing and reject replacements of built-in Dashboard sites and rules.

- Portal rules use flat `match*` and `route*` fields across Go, Admin API, Redis, Dashboard, and YAML. Existing database columns are migrated. Legacy YAML fields remain supported with warnings; mixing legacy and new fields in one rule is rejected. Upgrade Hub and Portal together and regenerate Admin clients.

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

- Complete managed component manager initialization before injecting the component
  into consumers, including dependencies of other managers. Initialization no longer
  requires dependency-first component registration; lifecycle hook order is unchanged.

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
