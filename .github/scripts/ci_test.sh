#!/usr/bin/env bash
set -euo pipefail
script="$(cd "$(dirname "$0")" && pwd)/ci.sh"
# shellcheck source=.github/scripts/ci.sh
source "$script"

check_paths() {
  local event="$1" expected="$2"
  shift 2
  printf '%s\0' "$@" | classify_changes "$event" | jq -e --argjson expected "$expected" '. == $expected' >/dev/null
}

for event in push pull_request; do
  check_paths "$event" '{"dashboard":false,"container":false,"workflow":false}' README.md CHANGELOG.md
  for file in .github/workflows/release.yml .github/scripts/ci.sh; do
    check_paths "$event" '{"dashboard":false,"container":true,"workflow":true}' "$file"
  done
done
for file in Dockerfile .dockerignore go.mod go.sum; do
  check_paths pull_request '{"dashboard":false,"container":true,"workflow":false}' "$file"
done
for file in cmd/vine/main.go internal/repo/schema.sql internal/assets/dashboard.tar.zst $'internal/path with\nnewline.go'; do
  check_paths pull_request '{"dashboard":false,"container":false,"workflow":false}' "$file"
  check_paths push '{"dashboard":false,"container":true,"workflow":false}' "$file"
done
check_paths pull_request '{"dashboard":true,"container":false,"workflow":false}' internal/daemon/hub/src/dashboard/src/App.tsx
check_paths pull_request '{"dashboard":true,"container":true,"workflow":true}' .github/workflows/ci.yml
check_paths pull_request '{"dashboard":false,"container":true,"workflow":true}' .github/workflows/example.yml
for event in push pull_request; do
  for file in package.json pnpm-lock.yaml; do
    container=false
    [[ "$event" != push ]] || container=true
    check_paths "$event" "{\"dashboard\":true,\"container\":$container,\"workflow\":false}" "internal/daemon/hub/src/dashboard/$file"
  done
done

for selected in true false; do
  needs=$(jq -n --arg selected "$selected" '
    reduce ["changes", "go-test", "go-static", "go-race", "go-lifecycle", "security"][] as $job ({}; .[$job].result = "success") |
    reduce ["dashboard", "container", "workflow"][] as $job (.;
      .changes.outputs[$job] = $selected | .[$job].result = (if $selected == "true" then "success" else "skipped" end))')
  verify_ci_results <<< "$needs" >/dev/null
  for job in changes go-test go-static go-race go-lifecycle security dashboard container workflow; do
    for result in failure cancelled skipped success; do
      [[ "$(jq -r --arg job "$job" '.[$job].result' <<< "$needs")" == "$result" ]] && continue
      bad=$(jq --arg job "$job" --arg result "$result" '.[$job].result = $result' <<< "$needs")
      if NEEDS="$bad" bash "$script" verify >/dev/null 2>&1; then
        echo "Incorrectly accepted $job=$result" >&2; exit 1
      fi
    done
  done
  bad=$(jq 'del(.changes.outputs.container)' <<< "$needs")
  if NEEDS="$bad" bash "$script" verify >/dev/null 2>&1; then exit 1; fi
done

# Exercise actual git diff/merge-base and output handling without touching this repo.
directory=$(mktemp -d "${TMPDIR:-/tmp}/vine-ci-test.XXXXXXXX")
trap 'rm -rf -- "$directory"' EXIT
(
  cd "$directory"
  git init -q -b main
  git config user.name 'CI fixture'
  git config user.email 'ci@example.invalid'
  git commit -qm initial --allow-empty
  base=$(git rev-parse HEAD)
  git checkout -qb feature
  mkdir -p internal
  touch 'internal/example with spaces.go'
  git add .
  git commit -qm runtime
  head=$(git rev-parse HEAD)
  git checkout -q main
  touch Dockerfile
  git add .
  git commit -qm packaging
  advanced_base=$(git rev-parse HEAD)
  GITHUB_EVENT_NAME=pull_request CHANGE_BASE="$advanced_base" CHANGE_HEAD="$head" GITHUB_OUTPUT="$directory/output" bash "$script" changes >/dev/null
  [[ "$(< "$directory/output")" == $'dashboard=false\ncontainer=false\nworkflow=false' ]]
  GITHUB_EVENT_NAME=push CHANGE_BASE="$base" CHANGE_HEAD="$head" GITHUB_OUTPUT="$directory/push-output" bash "$script" changes >/dev/null
  [[ "$(< "$directory/push-output")" == $'dashboard=false\ncontainer=true\nworkflow=false' ]]
  GITHUB_EVENT_NAME=push CHANGE_BASE=0000000000000000000000000000000000000000 CHANGE_HEAD="$head" GITHUB_OUTPUT="$directory/initial-output" bash "$script" changes >/dev/null
  [[ "$(< "$directory/initial-output")" == $'dashboard=true\ncontainer=true\nworkflow=true' ]]
  if GITHUB_EVENT_NAME=push CHANGE_BASE=invalid CHANGE_HEAD="$head" GITHUB_OUTPUT="$directory/output" bash "$script" changes >/dev/null 2>&1; then exit 1; fi
  if GITHUB_EVENT_NAME=push CHANGE_BASE=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa CHANGE_HEAD="$head" GITHUB_OUTPUT="$directory/output" bash "$script" changes >/dev/null 2>&1; then exit 1; fi
)
echo 'CI policy tests passed'
