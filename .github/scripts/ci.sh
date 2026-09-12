#!/usr/bin/env bash
set -euo pipefail

# Read NUL-delimited paths so whitespace in filenames is preserved.
classify_changes() {
  jq -Rse --arg event "$1" '
    if $event != "pull_request" then error("Unsupported event") else
      split("\u0000") | map(select(length > 0)) |
      any(.[]; . == ".github/scripts/ci.sh" or . == ".github/scripts/ci_test.sh") as $policy |
      ($policy or any(.[]; . == ".github/workflows/ci.yml")) as $ci |
      any(.[]; (startswith(".github/workflows/") or startswith(".github/scripts/") or
        ((startswith("test/") or startswith("script/")) and endswith(".sh"))) and
        (test("\\.(md|mdx)$") | not)) as $workflow |
      any(.[]; startswith("internal/daemon/hub/src/dashboard/") and (test("\\.(md|mdx)$") | not)) as $frontend |
      ($ci or $frontend or any(.[]; . == "script/build-dashboard-assets.sh" or . == ".github/workflows/ci-dashboard.yml")) as $dashboard |
      any(.[]; . == "go.mod" or . == "go.sum") as $dependencies |
      any(.[]; endswith(".go")) as $go_files |
      any(.[]; endswith(".go") and (endswith("_test.go") | not)) as $go_source |
      any(.[]; test("^(app|buildinfo|cmd|core|infra|internal|util)/") and
        (startswith("internal/daemon/hub/src/dashboard/") | not) and (test("\\.(md|mdx)$") | not) and (endswith("_test.go") | not)) as $runtime |
      ($ci or $dependencies or $go_files or $runtime or
        any(.[]; startswith("test/") and endswith(".sh") and . != "test/k8s.sh")) as $go |
      ($ci or any(.[]; . == "Dockerfile" or . == ".dockerignore" or
        . == ".github/workflows/release.yml" or startswith(".github/scripts/release"))) as $packaging |
      {
        "go-test": $go, "go-static": ($ci or $dependencies or $go_source or $runtime or
          any(.[]; startswith("test/") and endswith(".sh") and . != "test/k8s.sh")), "go-race": $go,
        licenses: ($ci or $dependencies or
          any(.[]; endswith(".go") and (endswith("_test.go") | not)) or
          any(.[]; . == "THIRD_PARTY_LICENSES.txt" or . == "script/gen-third-party-licenses.sh")),
        dashboard: $dashboard,
        container: ($packaging or $dependencies or $runtime),
        workflow: $workflow,
        k8s: ($ci or any(.[]; (startswith("deploy/k8s/") and (test("\\.(md|mdx)$") | not)) or . == "test/k8s.sh" or . == ".github/workflows/ci-k8s.yml")),
        "release-policy": ($ci or any(.[]; . == ".github/workflows/release.yml" or startswith(".github/scripts/release")))
      }
    end'
}

verify_ci_results() {
  jq -e '
    . as $needs |
    all(["changes", "security"][];
      . as $job | $needs[$job].result == "success") and
    all(["go-test", "go-static", "go-race", "licenses", "dashboard", "container", "workflow", "k8s", "release-policy"][];
      . as $flag | $needs.changes.outputs[$flag] | . == "true" or . == "false") and
    all(["go-test", "go-race", "dashboard", "container", "workflow", "k8s", "go-checks"][];
      . as $job |
      (if $job == "go-checks" then
        ($needs.changes.outputs["go-static"] == "true" or $needs.changes.outputs.licenses == "true")
      else $needs.changes.outputs[$job] == "true" end) as $selected |
      ($selected and $needs[$job].result == "success") or
      (($selected | not) and $needs[$job].result == "skipped")) and
    ($needs.changes.outputs["release-policy"] != "true" or $needs.changes.outputs.workflow == "true")
    | if . then true else error("CI failed or contains an unexpected skipped job") end'
}

ci_main() {
  case "${1:-}" in
    changes)
      [[ "$CHANGE_HEAD" =~ ^[0-9a-f]{40}$ && "$CHANGE_BASE" =~ ^[0-9a-f]{40}$ ]] || {
        echo "Invalid change range: base=$CHANGE_BASE head=$CHANGE_HEAD" >&2; return 1;
      }
      [[ "$GITHUB_EVENT_NAME" == pull_request ]] || { echo "Unsupported event" >&2; return 1; }
      local base flags
      base=$(git merge-base "$CHANGE_BASE" "$CHANGE_HEAD")
      flags=$(git diff --name-only --no-renames -z "$base" "$CHANGE_HEAD" | classify_changes "$GITHUB_EVENT_NAME")
      jq -r 'to_entries[] | "\(.key)=\(.value)"' <<< "$flags" | tee -a "$GITHUB_OUTPUT"
      ;;
    verify) verify_ci_results <<< "$NEEDS" ;;
    *) echo "Usage: ci.sh changes|verify" >&2; return 1 ;;
  esac
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then ci_main "$@"; fi
