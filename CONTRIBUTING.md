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

The Go module targets Go 1.27. Depending on the area being changed, development
may also require:

- Node.js 24.11.0 and pnpm 11.15.0 for the documentation site and Hub Dashboard
- `skelc` for regenerating Skel contracts

Download Go dependencies and run the baseline test suite with:

```bash
go mod download
bash test/test.sh
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
- Use `testing/synctest` for tests whose correctness depends on goroutine
  scheduling, timers, tickers, cancellation, or deadlines. Prefer
  `synctest.Wait` and `synctest.Sleep` over wall-clock polling and arbitrary
  sleeps. Keep bounded real-time guards only around external processes and
  actual network or storage I/O, which cannot run deterministically in a
  synctest bubble.
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
bash test/test.sh
```

The ordinary test script reuses cached results. Run `bash test/shuffle.sh` for
targeted order checks or `VINE_SHUFFLE_SCOPE=all bash test/shuffle.sh` for the full
suite. Shuffle prints a seed that can be reproduced with `go test -shuffle=<seed>`.
Main CI uses the full shuffled suite instead of a second ordinary test pass.

Also run `go vet ./...` after changes involving public APIs, concurrency,
reflection, or runtime wiring.

Run the focused performance benchmarks with:

```bash
bash test/benchmark.sh
```

The script reports allocation data and defaults to five one-second samples.
Set `GOTOOLCHAIN` to compare Go releases, or use `VINE_BENCH_PATTERN`,
`VINE_BENCH_TIME`, and `VINE_BENCH_COUNT` to narrow or tune a run. For example:

```bash
GOTOOLCHAIN=go1.27.1 VINE_BENCH_PATTERN=RPC bash test/benchmark.sh
```

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

Public documentation is maintained in the
[`yorun-ai/vine-site`](https://github.com/yorun-ai/vine-site) repository. Keep
its English and Simplified Chinese documentation trees synchronized:

- English: `docs`
- Simplified Chinese:
  `i18n/zh-CN/docusaurus-plugin-content-docs/current`

Do not manually modify versioned documentation snapshots. After changing files
in `vine-site`, build both locales from that repository:

```bash
pnpm install
pnpm build
```

For bilingual Markdown files elsewhere in the repository, update both language
versions and keep their language-switch links intact.

## Hub Dashboard

To start the Dashboard development server from the repository root:

```bash
pnpm --dir internal/daemon/hub/src/dashboard install
bash script/dev-hub-dashboard.sh
```

The script starts Vite on port 7098 and forwards additional arguments to Vite.
It fails if the port is occupied so Hub's development proxy keeps targeting the
correct server. Start Hub separately and open its Dashboard URL; when the local
binary has no embedded assets, it automatically probes Vite at `localhost:7098`.
If Vite is not running, the Dashboard returns 404 and tells you to run
`script/dev-hub-dashboard.sh`.

For Dashboard source changes, run:

```bash
cd internal/daemon/hub/src/dashboard
pnpm install
pnpm typecheck
pnpm build
```

Keep user-facing strings synchronized between `src/i18n/dictionaries/cn.ts` and
`en.ts`.

A fresh checkout has only an empty `.gitkeep` in the assets directory, so Hub
probes Vite. If `index.html` or `index.html.br` is present when Go compiles, Hub
serves that embedded Dashboard instead. Local builds also use previously packaged
assets; remove generated files while keeping `.gitkeep` to return to Vite.
Docker and Release build the frontend before compiling Vine.
To build an embedded Dashboard locally, run from the repository root:

```bash
bash script/build-dashboard-assets.sh
GOWORK=off go build -o bin/vine ./cmd/vine
```

The script type-checks and builds the frontend in a temporary directory. It
writes text as Brotli only when smaller and copies other files unchanged into
`internal/daemon/hub/src/server/mod/admin/assets/dashboard/`, preserving paths.
Each file has one representation. The complete directory replaces the previous
build, so old hashed files cannot survive. Generated files are ignored by Git;
the empty tracked `.gitkeep` is preserved and is never served over HTTP.
No archive or Go byte array is generated. A temporary Vite dependency report,
plus imported CSS and font notices, is checked against the Dashboard section of
`THIRD_PARTY_LICENSES.txt` before assets are replaced. The report is removed before
packaging; no separate license file is shipped inside the Dashboard.

After bundled dependency changes, run `bash script/gen-third-party-licenses.sh`
with Dashboard dependencies installed and include the updated unified inventory
in the change. Release archives include it beside the binary; all three images
include it and `LICENSE` in `/usr/share/licenses/vine/`.

All builds use `go:embed` with no build tag. A fresh checkout needs no frontend
output or Node.js toolchain. Asset Server serves precompressed Brotli directly
when accepted by the client and decodes it otherwise. Missing static files
return 404; HTML navigation falls back to `index.html`.

## Kubernetes Manifests

Keep shared resources in `deploy/k8s/base`, release image tags in
`deploy/k8s/overlays/stable/kustomization.yaml`, and reusable mTLS patches in
`deploy/k8s/components/mtls`. The root deployment and the `stable-mtls` overlay
both use the stable images.

During release preparation, update all three stable image tags together and run:

```bash
bash test/k8s.sh vX.Y.Z
```

The script requires `jq` and mikefarah/yq v4. It uses `kustomize` or `kubectl`
when available, otherwise a pinned Kustomize Go binary. It renders the manifests
locally without contacting a cluster and checks all five container and
init-container image references, pull policies, and mTLS configuration. Omit the
version argument to validate changes outside release preparation.

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
