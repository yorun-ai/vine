# Vine Agent Guidelines

Read the applicable directory README for ownership, dependency, and lifecycle constraints.

## Go Version and Syntax

- Target Go 1.27 syntax. Prefer `new` with a composite literal when creating a
  pointer, for example: `record := new(SomeStruct{Field: "value"})`.
- Declare the type of every parameter and result independently. Write
  `func f(key string, token string)`, never `func f(key, token string)`; the same
  applies to results, named function types, function literals, and variables.

## Naming

- Use `kind` when `type` would otherwise be the natural local variable name.
- Prefix unexported package-local production type declarations with `_`, such as
  `_App` and `_Config`. This convention applies only to types; do not prefix
  unexported constants, variables, functions, or methods with `_`. Test fixture
  types may use descriptive lowercase names such as `testApp` or
  `configRepoSpy`.
- Use `Rpc`, not `RPC`, in identifiers.
- Name repository implementations after the repository they provide, not after
  the storage they happen to use: `repo.AppConfigRepo`, not
  `repo.DBAppConfigRepo`.

## Generated Code

- Do not manually edit files under `internal/core/*/skeled`,
  `internal/daemon/hub/api/skeled`, or
  `internal/daemon/hub/src/dashboard/src/skeled`.
- Modify the corresponding `.skel` source and regenerate code with
  `bash script/gen-skel.sh [app|hub|link]`.
- Before regenerating, verify that `skelc version` satisfies the current minimum
  in `internal/core/skel/version.go`. Do not regenerate with an older compiler.
- Keep the import rewriting and formatting performed by `script/gen-skel.sh`.
  Generated runtime code intentionally imports internal packages.
- Treat the embedded Hub Dashboard bundle
  (`internal/daemon/hub/src/server/mod/admin/assets/dashboard.tar.zst`)
  as generated: rebuild it with `bash script/build-dashboard-assets.sh` whenever
  Dashboard source or the admin API it calls changes, and commit it with that
  change. Never assemble the archive by hand, and never resolve a conflict on it
  by picking one side; rebuild it from the merged source.

## Public API Boundaries

- User-facing framework APIs belong in `app`, `core/*`, and `infra/*`; reusable
  public helpers belong in `util/*`. Keep the framework API packages as
  documented facades over implementations in `internal`.
- Do not expose packages under `internal/daemon` or other implementation details
  through public API signatures.
- When changing a public facade, update its GoDoc, facade tests, user
  documentation, and compatibility notes as applicable.
- Add a useful GoDoc comment for every newly exported public symbol.
- Expose callable APIs to Vine users as function declarations, never exported
  function-valued package variables. Internal adapters may use function variables
  for forwarding; type aliases are allowed at either boundary.

## API and Implementation Design

- Vine is an application framework, and most arguments are passed by code within
  the project. Do not add unnecessary nil or empty-value checks.
- Injected dependencies are never nil: an unbound type panics when resolved, and
  a factory that returns nil panics unless the binding declares
  `AsNullable()`. Only check an injected field for nil when its binding is
  nullable, and declare the binding nullable instead of guessing at the
  consumption site.
- Do not add defensive behavior to production code solely to accommodate tests.
- Repository implementations assemble the complete entities they return:
  derived values such as a site's Web mount path and Rpc services, the
  definition and status of a configuration, and stored provenance such as field
  sources. Domain code applies rules and invariants to the entities it receives;
  API services map entities to payloads instead of joining data from several
  sources.
- Add a domain type only when it carries rules. Do not introduce pass-through
  facades over a repository: a service that only queries storage may depend on
  the repository interface directly.
- Preserve the active `meta.Context`, trace, actor, initiator, cancellation, and
  deadline when forwarding or deriving work. Do not replace an active request
  context with `context.Background()`.
- Preserve component and module lifecycle ordering unless the task explicitly
  changes the lifecycle contract.

## Protocol and Persistence Boundaries

- Treat Rpc/Web headers, Redis key formats, serialized JSON/CBOR fields, Skel
  schemas, and generated contracts as cross-component protocol boundaries.
- When changing these formats, update affected producers, consumers and tests;
  correct existing documentation and add migration guidance when needed.
- Use Go's `encoding/json/v2` and `encoding/json/jsontext` APIs for Vine JSON;
  do not reintroduce the v1 `encoding/json` implementation.
- Encode Rpc, Event, and Task Skel payloads with the shared `vcode` encoder.
  Supported schemas use empty arrays/maps for nil collections.
- The runtime isolates in-process Rpc arguments and results by cloning them from
  the declared Go types with `util/vbean`. Skel contracts carry only generated
  scalars, lists, maps, nullable values and beans, so reflection covers them.
  `MethodSpec.CloneArguments` and `CloneResult` are deprecated and ignored, and
  the runtime must not restore serialization-based clone fallbacks.

## Documentation

- Public Vine documentation is maintained in the `yorun-ai/vine-site`
  repository. English source documents live under `docs`, and Simplified
  Chinese translations live under
  `i18n/zh-CN/docusaurus-plugin-content-docs/current`.
- Correct existing documentation made inaccurate by public behavior changes and
  document new user-facing features, keeping both locales synchronized. Internal
  changes and fixes restoring documented behavior do not need new site content.
- Do not manually edit versioned documentation snapshots in `vine-site`.

## Container Images

- Hub, Link, and Portal image targets must continue to use the same Vine binary
  from the shared multi-stage `Dockerfile`; target-specific stages should only
  define runtime configuration and entry commands.
- Keep the default `GO_VERSION` build argument in `Dockerfile` aligned with the
  Go version in `go.mod`. CI and release workflows may override it with the
  approved patch version.
- Hub defaults to no-db mode with a required seed YAML file and read-only
  configuration. Explicit SQLite or PostgreSQL selects writable persistence.
  Hub defaults to embedded MQ. External NATS requires mq-mode=nats and an endpoint;
  embedded mode rejects an external endpoint.
- For CI or publication changes, read `.github/CI.md` for required gates,
  release validation, concurrency, recovery, and dependency security policy.

## Release Preparation

- Follow `CONTRIBUTING.md` for Kubernetes image tags and validation, Dashboard
  asset rebuilding, and generated contracts; follow `.github/CI.md` for publication.
- Write `CHANGELOG.md` only in the release-preparation change: compare the commits
  since the previous tag, add the dated release heading with its entries, and
  retain an empty `[Unreleased]` section.
- Do not touch `CHANGELOG.md` in feature, fix, refactor, or CI commits. A change
  made early in a release cycle can be reverted before it ships, which leaves
  entries that never matched the released code; describe the user-visible effect
  in the change itself and in its pull request, and let release preparation
  collect it.
- After Go dependency changes, run `bash script/gen-third-party-licenses.sh`
  and commit inventory changes. Regenerate contracts with `bash script/gen-skel.sh all`
  and inspect drift. Build Dashboard archives only with the documented script.
- Versions come from release tags and build-time `ldflags`, not source constants.

## Tests

- Keep implementation tests paired with their source files. For example, tests
  for `reader.go` belong in `reader_test.go`. Shared setup may live in
  `test_helper_test.go`, but do not group unrelated implementation tests there.
- Restore modified globals, registries, environment variables, inproc endpoints,
  and background resources with `t.Cleanup`.
- Do not use `t.Parallel()` in packages that share global registries, global
  logger settings, application singletons, or inproc endpoint registries unless
  isolation is explicitly proven.
- A test package using `app/testkit` should start only one standalone runtime and
  share it through subtests.
- Do not add new production-compiled `*_fortest.go` hooks without explicit
  justification. Prefer test-local fakes and dependency injection.

## Validation

- Run `gofmt` on changed Go files and run `git diff --check`.
- Run `bash test/quick.sh <scope> [scope...]` for predefined affected-package
  groups while iterating; available scopes are `app`, `cli`, `core`, `rpc`,
  `web`, `event`, `task`, `infra`, `hub`, `link`, `portal`, and `all`. Then run
  `bash test/test.sh` for cache-friendly repository-wide Go changes. Run `bash
  test/shuffle.sh` for the targeted order-randomization suite and
  `VINE_SHUFFLE_SCOPE=all bash test/shuffle.sh` for the full suite. Run `bash
  test/race.sh` for the concurrency-focused race suite and
  `VINE_RACE_SCOPE=all bash test/race.sh` for the full release race suite.
  These scripts set `GOWORK=off` so the enclosing workspace cannot replace
  published module dependencies.
- Run `GOWORK=off go vet ./...` after changing public APIs, concurrency,
  reflection, or runtime wiring.
- For a release, run `bash test/test.sh`, full-scope shuffle and race checks,
  and cross-build `cmd/vine` for Darwin and Linux on AMD64 and ARM64. Validate
  that the dated changelog heading matches the intended release tag.
- After changing container build inputs, validate the Hub target. Changes to
  shared image stages or release publication must also validate all three image
  targets.
- Run `pnpm build` in `vine-site` after changing Vine public documentation there.
- After regenerating Skel code, inspect the generated diff and run all affected
  Go and frontend checks.
