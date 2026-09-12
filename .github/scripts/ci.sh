#!/usr/bin/env bash
set -euo pipefail

# Read NUL-delimited paths so whitespace in filenames is preserved.
classify_changes() {
  jq -Rse --arg event "$1" '
    if $event != "push" and $event != "pull_request" then error("Unsupported event") else
      split("\u0000") | map(select(length > 0)) |
      any(.[]; . == ".github/workflows/ci.yml" or . == ".github/scripts/ci.sh" or . == ".github/scripts/ci_test.sh") as $ci |
      any(.[]; (startswith(".github/workflows/") or startswith(".github/scripts/") or
        ((startswith("test/") or startswith("script/")) and endswith(".sh"))) and
        (test("\\.(md|mdx)$") | not)) as $workflow |
      any(.[]; startswith("internal/daemon/hub/src/dashboard/") and (test("\\.(md|mdx)$") | not)) as $frontend |
      ($ci or $frontend or any(.[]; . == "script/build-dashboard-assets.sh")) as $dashboard |
      any(.[]; . == "go.mod" or . == "go.sum") as $dependencies |
      any(.[]; endswith(".go")) as $go_files |
      any(.[]; endswith(".go") and (endswith("_test.go") | not)) as $go_source |
      any(.[]; test("^(app|buildinfo|cmd|core|infra|internal|util)/") and
        (startswith("internal/daemon/hub/src/dashboard/") | not) and (test("\\.(md|mdx)$") | not)) as $runtime |
      ($event == "push" or $ci or $dependencies or $go_files or $runtime or
        any(.[]; startswith("test/") and endswith(".sh") and . != "test/k8s.sh")) as $go |
      ($ci or any(.[]; . == "Dockerfile" or . == ".dockerignore" or
        . == ".github/workflows/release.yml" or startswith(".github/scripts/release"))) as $packaging |
      {
        "go-test": $go, "go-static": ($event == "push" or $ci or $dependencies or $go_source or $runtime or
          any(.[]; startswith("test/") and endswith(".sh") and . != "test/k8s.sh")), "go-race": $go,
        licenses: ($event == "push" or $ci or $dependencies or
          any(.[]; endswith(".go") and (endswith("_test.go") | not)) or
          any(.[]; . == "THIRD_PARTY_LICENSES.txt" or . == "script/gen-third-party-licenses.sh")),
        dashboard: $dashboard,
        container: ($packaging or ($event == "push" and ($dependencies or $runtime))),
        workflow: $workflow,
        k8s: ($ci or any(.[]; (startswith("deploy/k8s/") and (test("\\.(md|mdx)$") | not)) or . == "test/k8s.sh"))
      }
    end'
}

verify_ci_results() {
  jq -e '
    . as $needs |
    all(["changes", "security"][];
      . as $job | $needs[$job].result == "success") and
    all(["go-test", "go-static", "go-race", "licenses", "dashboard", "container", "workflow", "k8s"][];
      . as $job | $needs.changes.outputs[$job] as $selected |
      ($selected == "true" and $needs[$job].result == "success") or
      ($selected == "false" and $needs[$job].result == "skipped"))
    | if . then true else error("CI failed or contains an unexpected skipped job") end'
}

ci_main() {
  case "${1:-}" in
    changes)
      [[ "$CHANGE_HEAD" =~ ^[0-9a-f]{40}$ && "$CHANGE_BASE" =~ ^[0-9a-f]{40}$ ]] || {
        echo "Invalid change range: base=$CHANGE_BASE head=$CHANGE_HEAD" >&2; return 1;
      }
      local base="$CHANGE_BASE" flags
      if [[ "$base" =~ ^0+$ ]]; then
        flags=$(printf '%s\0' .github/workflows/ci.yml | classify_changes "$GITHUB_EVENT_NAME")
      else
        if [[ "$GITHUB_EVENT_NAME" == pull_request ]]; then
          base=$(git merge-base "$base" "$CHANGE_HEAD")
        fi
        flags=$(git diff --name-only --no-renames -z "$base" "$CHANGE_HEAD" | classify_changes "$GITHUB_EVENT_NAME")
      fi
      jq -r 'to_entries[] | "\(.key)=\(.value)"' <<< "$flags" | tee -a "$GITHUB_OUTPUT"
      ;;
    verify) verify_ci_results <<< "$NEEDS" ;;
    *) echo "Usage: ci.sh changes|verify" >&2; return 1 ;;
  esac
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then ci_main "$@"; fi
