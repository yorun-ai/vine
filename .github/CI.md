# CI and Release Maintenance

Keep this guide named `CI.md`: GitHub prioritizes `.github/README.md` over the
root README when displaying the repository homepage.

This directory owns repository automation, not public deployment documentation.

| Event | Workflow | Responsibility |
| --- | --- | --- |
| PR targeting main | `ci.yml` | Run checks selected by changed inputs; always scan secrets and verify the required gate |
| Tag push | None | Mark a version only; never publish artifacts |
| Published Release | `release.yml` | Validate the tag and publish binaries and images in parallel |
| Manual Release workflow | `release.yml` | Recover artifacts for an existing published release |

## Required CI Gate

Keep the name `CI / Required Checks`: the organization ruleset requires it.
Do not add workflow-level path filters. Change classification and secret scanning
always run. Every selected job must succeed, and only explicitly unselected jobs
may be skipped. Missing selection outputs, cancellation, or classification failure
fail the gate.

| Changed PR inputs | Selected checks, in addition to secrets and the gate |
| --- | --- |
| Documentation only | None |
| Go production source, module files, backend resources or test fixtures | Full ordinary Go tests and leak checks, static checks, targeted race, Hub image build |
| Go test files only | Full ordinary Go tests and leak checks, targeted race |
| Non-test Go source or module files | Also regenerate and check the CLI license inventory |
| Go test shell scripts | Go checks and workflow checks |
| Dashboard source/dependencies or packaging script | Dashboard build, including type checking; shell scripts also select workflow checks |
| Dockerfile or Docker ignore rules | Hub image build |
| Kubernetes manifests or their validation script | Render and validate Kubernetes overlays; shell scripts also select workflow checks |
| License inventory or its generator | License checks; the generator also selects workflow checks |
| CI orchestration workflow | All optional checks, to validate job wiring |
| Dashboard/Kubernetes reusable workflow | Its corresponding check and workflow lint |
| Classification helpers | All optional checks, to validate the gate and its wiring |
| Release workflow/helpers | Workflow checks, release policy/metadata checks and Hub image build |
| Other workflow or shell scripts | Workflow checks |

Go tests cover all packages when selected; PRs do not maintain a dependency-based
package filter. Test-only Go changes select Go test and targeted race checks but
skip static checks. Backend resource changes include embedded SQL, Dashboard
archives, and test fixtures. Frontend source and Markdown do not select Go checks
by themselves.

The ordinary Go test job also runs the separate build-tagged goroutine leak tests.
PRs run ordinary tests and targeted race checks. Full shuffled and race suites are
available through local release validation when needed.
Race and leak scripts disable implicit vet,
because the Go checks job runs full vet once. Static checks and license validation
share one job and Go setup, but retain independent step conditions. License-only
changes do not run vet or module tidiness checks.

Go module and backend production input changes validate the Hub image in the PR.
There is no main or tag CI. PR image builds read the shared cache without exporting
it. Pure frontend changes select the Dashboard build, not a rebuild of the unchanged
embedded archive. Dashboard and Kubernetes steps live in reusable workflows; changes to either
select that check. The orchestration workflow and classification helpers still
select all jobs to exercise their wiring and the complete gate.

Tests use the latest Go `1.27.x`; container and release builds use Go `1.27.1`.
PR updates cancel older runs for the same PR.
The Hub check builds Linux AMD64 without publishing. All three image targets and
both Linux architectures are published only by the Release workflow.

## Dependency Security

Dependabot owns dependency vulnerability alerts and automated security update
PRs for Go and Dashboard dependencies. Keep the dependency graph, Dependabot
alerts, and Dependabot security updates enabled in GitHub repository settings.
Verify that the graph recognizes `go.mod` and
`internal/daemon/hub/src/dashboard/pnpm-lock.yaml`.

CI does not run `pnpm audit` or `govulncheck`, and there is no scheduled or
manual audit workflow. Dependabot alerts monitor the default branch rather than
acting as a PR merge gate. No scheduled version-update configuration is needed
for security updates. Secret scanning always runs; third-party license checks are required whenever
selected by the change policy.

## Release Sequence and Recovery

1. Prepare the dated changelog in a PR, pass CI, and merge it.
2. Create the version tag at that commit, then publish its GitHub Release.
3. Shared validation checks tag, changelog, and main ancestry.
4. Binaries and all three images publish independently after validation.
6. Completion verifies four archive checksums, anonymous access to all three
   images, Linux AMD64/ARM64, and release version/source/revision labels.
   Only then may the current non-prerelease update image `latest` tags.
7. A separate promotion job shares one concurrency group across all versions.
   It rechecks latest-release eligibility after acquiring the lock, and holds
   the lock through promotion and verification. Builds remain parallel. Pending
   promotion jobs may be superseded under GitHub's default concurrency policy;
   rerun a cancelled job if its release is still the intended latest.

Image jobs check their version tag with authenticated GHCR access before building.
Existing version images are skipped; only HTTP 404 permits a build. Authentication,
network, and other registry errors fail the job. Final verification still checks
all three images, so existing invalid images require manual inspection. This
workflow check does not prevent another registry client from overwriting a tag.
The `latest` promotion remains unchanged.

Manual runs select `artifacts: all`, `binaries`, or `images`. Use `images` when
binary assets already exist. The workflow rejects existing expected binary
assets (including partial uploads), and upload never uses `--clobber`. A partial
binary upload requires explicit maintainer inspection and cleanup before retry;
automation does not delete published assets. Normal failed-job reruns can also
recover failures without rerunning successful jobs.

Artifact selection controls publication, not the definition of a complete
release: the completion check still validates both artifact families. If a
package is not publicly accessible, completion fails and `latest` is not
promoted. Set its visibility explicitly and retry after checking permissions.
Binary-only recovery may promote already-published images once the complete
release passes verification; it does not rebuild those images.

For workflow fixes, dispatch from updated main against the original tag. Helper
scripts are checked out from the workflow revision separately from release
source, so older tags do not need to contain those scripts. Do not move tags or
rebuild existing binary attachments merely to recover image publication.

## Validating Changes

Automation uses YAML, Bash, `gh`, and `jq`; anonymous GHCR verification uses
`curl`, and archive verification uses `sha256sum`. These tools are available on
the Ubuntu runner. No custom JavaScript helpers are required. Dashboard builds
still use their own Node.js toolchain.

Anonymous GHCR GETs retry up to three times for curl's transient HTTP errors and
timeouts, with a 10-second connection timeout, a 30-second attempt limit, and a
90-second retry window. Response bodies are buffered before JSON validation.
Authentication errors, invalid JSON, checksum/label mismatches, and publication
operations are not retried automatically. GitHub CLI failures still fail the job.

```bash
shellcheck .github/scripts/*.sh test/*.sh
bash .github/scripts/ci_test.sh
bash .github/scripts/release_test.sh
GOWORK=off go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
git diff --check
```

Release policy tests and the metadata fixture run only when the release workflow,
release helpers, CI orchestration or classification helpers change. Other workflow
and test-script edits run lint and CI policy tests without release checks.

For those release-related changes, workflow CI exercises the actual pinned Docker metadata action against a
detached checkout with tag refs, without publishing. Keep this regression check
when changing checkout behavior or metadata configuration. Test all three image
targets when changing shared image stages or release publication.

Binary builds fetch tag refs, require a clean checkout, and verify that each
binary embeds the exact release version, source revision, and `vcs.modified=false`.
The CLI version override alone is not sufficient to validate release metadata.
