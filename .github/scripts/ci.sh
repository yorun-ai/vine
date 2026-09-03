#!/usr/bin/env bash
set -euo pipefail

# Read NUL-delimited paths so whitespace in filenames is preserved.
classify_changes() {
  jq -Rse --arg event "$1" '
    if $event != "push" and $event != "pull_request" then error("Unsupported event") else
      split("\u0000") | map(select(length > 0)) |
      any(.[]; startswith(".github/workflows/") or startswith(".github/scripts/")) as $workflow |
      ($workflow or any(.[]; startswith("internal/daemon/hub/src/dashboard/") or . == "script/build-dashboard-assets.sh")) as $dashboard |
      ($workflow or any(.[]; . == "Dockerfile" or . == ".dockerignore" or . == "go.mod" or . == "go.sum")) as $packaging |
      any(.[]; test("^(app|buildinfo|cmd|core|infra|internal|util)/") and (test("\\.(md|mdx)$") | not)) as $runtime |
      {dashboard: $dashboard, container: ($packaging or ($event == "push" and $runtime)), workflow: $workflow}
    end'
}

verify_ci_results() {
  jq -e '
    . as $needs |
    all(["changes", "go-test", "go-static", "go-race", "go-lifecycle", "security"][];
      . as $job | $needs[$job].result == "success") and
    all(["dashboard", "container", "workflow"][];
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
        flags=$(printf '.github/workflows/ci.yml\0' | classify_changes "$GITHUB_EVENT_NAME")
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
