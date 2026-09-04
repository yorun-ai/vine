# CI and Release Maintenance

This directory owns repository automation, not public deployment documentation.

| Event | Workflow | Responsibility |
| --- | --- | --- |
| PR targeting main | `ci.yml` | Ordinary tests, static/security/license checks, targeted race/shuffle and lifecycle checks; optional jobs below |
| Push to main | `ci.yml` | Test the actual integrated commit, using full race/shuffle; optional jobs below |
| Tag push | None | Mark a version only; never publish artifacts |
| Published Release | `release.yml` | Validate the tag and its main CI, then publish binaries and images in parallel |
| Manual Release workflow | `release.yml` | Recover artifacts for an existing published release |
| Manual dispatch | `audit.yml` | Audit existing Dashboard dependencies |

## Required CI Gate

Keep the name `CI / Required Checks`: the organization ruleset requires it.
Do not add workflow-level path filters. The `changes` job selects optional jobs,
and the final gate requires all unconditional jobs to succeed. Optional jobs
must succeed when selected and must be skipped only when explicitly unselected.
Failed classification, cancellation, or an unexpected skipped job fails the gate.

- Dashboard source, its packaging script, or `ci.yml` changes select Dashboard
  build checks. Release-only workflow changes do not select the Dashboard build.
- Only Dashboard `package.json` or `pnpm-lock.yaml` changes select the separate
  dependency audit on PR/main runs. Selected audits must pass the required gate.
  Source-only changes and workflow edits do not call the npm audit service.
- Dockerfile, Docker ignore rules, or Go module files select the Hub build on PRs.
- Main also selects the Hub build for runtime source and embedded input changes,
  including SQL and the embedded Dashboard archive.
- Workflow/helper changes select workflow tests and the Hub build check.
- Pure README/changelog changes do not select these expensive optional jobs.
- Go checks still cover the full repository rather than only changed packages.

Tests use the latest Go `1.27.x`; container and release builds use Go `1.27.1`.
PR updates cancel older runs for the same PR. Main CI uses per-commit concurrency
groups, so a subsequent merge cannot cancel validation of a release candidate.
The Hub check builds Linux AMD64 without publishing. All three image targets and
both Linux architectures are published only by the Release workflow.

The dependency audit reuses `audit.yml`, which also supports manual dispatch.
It reads the committed lockfile without installing packages or
building the Dashboard. Vulnerabilities and registry failures fail the audit;
manual audit results are separate from PR/main CI.

## Release Sequence and Recovery

1. Prepare the dated changelog in a PR, pass CI, and merge it.
2. Wait for main CI on that exact merged commit to succeed.
3. Create the version tag at that commit, then publish its GitHub Release.
4. Shared validation checks tag, changelog, main ancestry, and the latest main
   push CI run for that exact SHA. Missing, pending, cancelled, or failed CI
   blocks publication; a green PR or a green different SHA is insufficient.
5. Binaries and all three images publish independently after validation.
6. Completion verifies four archive checksums, anonymous access to all three
   images, Linux AMD64/ARM64, and release version/source/revision labels.
   Only then may the current non-prerelease update image `latest` tags.
7. A separate promotion job shares one concurrency group across all versions.
   It rechecks latest-release eligibility after acquiring the lock, and holds
   the lock through promotion and verification. Builds remain parallel. Pending
   promotion jobs may be superseded under GitHub's default concurrency policy;
   rerun a cancelled job if its release is still the intended latest.

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
shellcheck .github/scripts/*.sh
bash .github/scripts/ci_test.sh
bash .github/scripts/release_test.sh
GOWORK=off go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
git diff --check
```

Workflow CI also exercises the actual pinned Docker metadata action against a
detached checkout with tag refs, without publishing. Keep this regression check
when changing checkout behavior or metadata configuration. Test all three image
targets when changing shared image stages or release publication.
