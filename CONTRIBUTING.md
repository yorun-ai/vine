# Contributing to Vine

Thank you for contributing to Vine. This guide describes the repository's
development workflow and the checks expected before a change is submitted.

## Before You Start

- For substantial API, protocol, storage, or architecture changes, open an issue
  or discussion before investing in an implementation.
- Read [AGENTS.md](AGENTS.md) for repository-wide coding and testing rules.
- Some directories contain their own README with additional ownership,
  dependency, lifecycle, and validation constraints. Read the applicable README
  before changing that directory or any of its descendants.
- Keep each pull request focused on one coherent change. Separate unrelated
  refactoring, formatting, and behavior changes.

The runtime-specific guides are:

- [Hub](internal/daemon/hub/README.md)
- [Link](internal/daemon/link/README.md)
- [Portal](internal/daemon/portal/README.md)

## Prerequisites

The Go module targets Go 1.26. Depending on the area being changed, development
may also require:

- Node.js 20 or later and pnpm for the documentation site and Hub Dashboard
- `skelc` for regenerating Skel contracts
- `zstd` when rebuilding the embedded Hub Dashboard release asset

Download Go dependencies and run the baseline test suite with:

```bash
go mod download
go test ./...
```

## Repository Boundaries

- User-facing framework APIs belong in `app`, `core/*`, and `infra/*`. Reusable
  public helpers belong in `util/*`.
- Public framework packages should remain documented facades over implementation
  packages under `internal`.
- Do not expose `internal/daemon` packages or other implementation details in a
  public API signature.
- Preserve the component and module lifecycle, request context, trace, identity,
  cancellation, and deadline propagation contracts.
- Treat Rpc/Web headers, Redis keys and values, JSON/CBOR fields, Skel schemas,
  and generated contracts as cross-component protocols. Update every producer,
  consumer, test, and relevant document together.

Every new exported public symbol must have useful GoDoc. Public API changes
should also include facade tests and user documentation where applicable.

## Go Changes

- Format changed Go files with `gofmt`.
- Follow the naming and implementation rules in [AGENTS.md](AGENTS.md), including
  the project's `Rpc` spelling and private production type naming convention.
- Keep implementation tests paired with their source files. Shared setup may
  live in `test_helper_test.go`.
- Use `t.Cleanup` to restore globals, registries, environment variables, inproc
  endpoints, and background resources.
- Do not enable `t.Parallel()` where tests share global registries, application
  singletons, logger settings, or inproc endpoint registries.
- A package using `app/testkit` should start one standalone runtime and share it
  through subtests.

While iterating, run the narrowest relevant package tests:

```bash
go test ./path/to/package
```

Before submitting a repository-wide Go change, run:

```bash
go test ./...
```

Also run `go vet ./...` after changes involving public APIs, concurrency,
reflection, or runtime wiring.

## Generated Skel Code

Do not manually edit generated files under:

- `internal/core/*/skeled`
- `internal/daemon/hub/api/skeled`
- `internal/daemon/hub/src/dashboard/src/skeled`

Modify the corresponding `.skel` contracts and run the repository script from
the repository root:

```bash
bash script/gen-skel.sh app
bash script/gen-skel.sh hub
bash script/gen-skel.sh link
```

Use `bash script/gen-skel.sh all` when every target must be regenerated. The
script intentionally rewrites generated imports and formats generated Go files;
do not replace it with a direct `skelc` invocation. Inspect the generated diff
and run all affected Go and frontend checks.

## Documentation

Public documentation is maintained in
[`yorun-ai/vine-doc`](https://github.com/yorun-ai/vine-doc) repository. Keep its
Chinese and English Vine documentation trees synchronized:

- Chinese: `content/vine`
- English: `i18n/en/docusaurus-plugin-content-docs-vine/current`

Do not manually modify versioned documentation snapshots. After changing files
in `vine-doc`, build both locales from that repository:

```bash
pnpm install
pnpm build
```

For bilingual Markdown files elsewhere in the repository, update both language
versions and keep their language-switch links intact.

## Hub Dashboard

For Dashboard source changes, run:

```bash
cd internal/daemon/hub/src/dashboard
pnpm install
pnpm typecheck
pnpm build
```

Keep user-facing strings synchronized between `src/i18n/dictionaries/cn.ts` and
`en.ts`.

Do not rebuild the embedded `dashboard.tar.zst` during ordinary development.
When a task explicitly includes updating release assets, run the following from
the repository root and commit the resulting archive:

```bash
bash script/build-dashboard-assets.sh
```

Do not assemble the archive manually.

## Compatibility

Vine is stabilizing its public API before `v1.0.0`. Patch releases remain
backward-compatible within the same minor release line, while minor releases may
include documented breaking changes.

Call out changes to public APIs, CLI behavior, configuration, Skel contracts,
wire protocols, Redis formats, or persistent schemas in the pull request. Add
migration notes whenever existing users must update code, configuration, or
generated artifacts.

## Pull Request Checklist

Before submitting a pull request, confirm that:

- The change is focused and follows the applicable repository and module rules.
- Changed Go files are formatted and `git diff --check` passes.
- Relevant targeted tests pass, and `go test ./...` passes for repository-wide
  Go changes.
- `go vet ./...` has been run when applicable.
- Generated files were produced by repository scripts and their diff was
  reviewed.
- Chinese and English documentation or Dashboard strings are synchronized.
- Compatibility impact and migration requirements are documented.
- No credentials, local paths, editor files, build output, or runtime data are
  included.

## License

Unless explicitly stated otherwise, any contribution intentionally submitted
for inclusion in Vine is licensed under the terms and conditions of the
[Apache License 2.0](LICENSE), in accordance with Section 5 of the license.

By submitting a contribution, you represent that you have the right to submit
it under these terms.
