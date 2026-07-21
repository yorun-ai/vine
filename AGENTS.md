# Vine Agent Guidelines

## Working in the Repository

- Some directories contain a `README.md` with additional instructions. Read the
  applicable README before modifying that directory or any of its descendants.
- Preserve the ownership, dependency, and lifecycle boundaries documented by
  those READMEs. Update the applicable README when a change alters them.

## Go Version and Syntax

- Target Go 1.26 syntax. Prefer `new` with a composite literal when creating a
  pointer, for example: `record := new(SomeStruct{Field: "value"})`.

## Naming

- Use `kind` when `type` would otherwise be the natural local variable name.
- Prefix unexported package-local production type declarations with `_`, such as
  `_App` and `_Config`. This convention applies only to types; do not prefix
  unexported constants, variables, functions, or methods with `_`. Test fixture
  types may use descriptive lowercase names such as `testApp` or
  `configRepoSpy`.
- Use `Rpc`, not `RPC`, in identifiers.

## Generated Code

- Do not manually edit files under `internal/core/*/skeled`,
  `internal/daemon/hub/api/skeled`, or
  `internal/daemon/hub/src/dashboard/src/skeled`.
- Modify the corresponding `.skel` source and regenerate code with
  `bash script/gen-skel.sh [app|hub|link]`.
- Keep the import rewriting and formatting performed by `script/gen-skel.sh`.
  Generated runtime code intentionally imports internal packages.

## Public API Boundaries

- User-facing framework APIs belong in `app`, `core/*`, and `infra/*`; reusable
  public helpers belong in `util/*`. Keep the framework API packages as
  documented facades over implementations in `internal`.
- Do not expose packages under `internal/daemon` or other implementation details
  through public API signatures.
- When changing a public facade, update its GoDoc, facade tests, user
  documentation, and compatibility notes as applicable.
- Add a useful GoDoc comment for every newly exported public symbol.

## API and Implementation Design

- Vine is an application framework, and most arguments are passed by code within
  the project. Do not add unnecessary nil or empty-value checks.
- Understand a method's responsibility and intended usage before changing it.
  Avoid indiscriminate defensive checks that do not belong to its contract.
- Do not add defensive behavior to production code solely to accommodate tests.
- Preserve the active `meta.Context`, trace, actor, initiator, cancellation, and
  deadline when forwarding or deriving work. Do not replace an active request
  context with `context.Background()`.
- Preserve component and module lifecycle ordering unless the task explicitly
  changes the lifecycle contract.

## Protocol and Persistence Boundaries

- Treat Rpc/Web headers, Redis key formats, serialized JSON/CBOR fields, Skel
  schemas, and generated contracts as cross-component protocol boundaries.
- When changing one of these formats, update all producers, consumers, tests,
  documentation, and compatibility notes together.

## Documentation

- Public Vine documentation is maintained in `yorun-ai/vine-doc`
  repository under `content/vine`, with English translations under
  `i18n/en/docusaurus-plugin-content-docs-vine/current`.
- When changing public behavior, update the corresponding current documentation
  in `vine-doc` in the same delivery and keep both locales synchronized.
- Vine and skelc have independent documentation versions. Do not manually edit
  versioned documentation snapshots in `vine-doc`.

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
- Run targeted package tests while iterating, then run `go test ./...` for
  repository-wide Go changes.
- Run `go vet ./...` after changing public APIs, concurrency, reflection, or
  runtime wiring.
- Run `pnpm build` in `vine-doc` after changing Vine public documentation there.
- After regenerating Skel code, inspect the generated diff and run all affected
  Go and frontend checks.
