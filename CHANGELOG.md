# Changelog

All notable changes to Vine are documented in this file.

The project follows [Semantic Versioning](https://semver.org/). The public
version history starts at `v0.9.0`; versions from the former private repository
are not part of the public compatibility commitment.

## [Unreleased]

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
