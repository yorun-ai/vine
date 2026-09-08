#!/usr/bin/env bash
set -euo pipefail
script="$(cd "$(dirname "$0")" && pwd)/ci.sh"
# shellcheck source=.github/scripts/ci.sh
source "$script"

# Expected job sets are independent of the classifier's implementation.
check_paths() {
  local event="$1" selected="$2"
  shift 2
  local actual
  actual=$(printf '%s\0' "$@" | classify_changes "$event")
  jq -e --arg selected "$selected" '
    keys == ["container","dashboard","go-race","go-static","go-test","k8s","licenses","workflow"] and
    ([to_entries[] | select(.value == true) | .key] | sort) == ($selected | split(" ") | map(select(length > 0)) | sort) and
    all(.[]; type == "boolean")
  ' <<< "$actual" >/dev/null || { echo "Unexpected classification ($event): $* => $actual" >&2; return 1; }
}

go_jobs='go-test go-static go-race'
all_jobs="$go_jobs licenses dashboard container workflow k8s"
check_paths pull_request '' README.md CHANGELOG.md .github/CI.md .github/scripts/README.md deploy/k8s/README.md
check_paths push "$go_jobs licenses" README.md
for file in go.mod go.sum app/example.go $'internal/path with\nnewline.go'; do
  check_paths pull_request "$go_jobs licenses" "$file"
  check_paths push "$go_jobs licenses container" "$file"
done
for file in internal/app/example_test.go internal/daemon/hub/src/server/repo/db/model/sql/sqlite/create_portal_rule.sql internal/daemon/hub/src/server/impl/admin/dashboard/assets/dashboard.tar.zst internal/testdata/input.json; do
  check_paths pull_request "$go_jobs" "$file"
  check_paths push "$go_jobs licenses container" "$file"
done
for file in Dockerfile .dockerignore; do
  check_paths pull_request container "$file"
  check_paths push "$go_jobs licenses container" "$file"
done
for file in .github/workflows/ci.yml .github/scripts/ci.sh .github/scripts/ci_test.sh; do
  check_paths pull_request "$all_jobs" "$file"
  check_paths push "$all_jobs" "$file"
done
for file in .github/workflows/release.yml .github/scripts/release.sh .github/scripts/release_test.sh; do
  check_paths pull_request 'workflow container' "$file"
done
check_paths pull_request workflow .github/workflows/example.yml
for file in test/test.sh test/race.sh test/shuffle.sh test/goroutineleak.sh; do
  check_paths pull_request "$go_jobs workflow" "$file"
done
check_paths pull_request 'k8s workflow' test/k8s.sh
check_paths pull_request k8s deploy/k8s/overlays/stable/kustomization.yaml
check_paths pull_request licenses THIRD_PARTY_LICENSES.txt
check_paths pull_request 'licenses workflow' script/gen-third-party-licenses.sh
check_paths pull_request 'dashboard workflow' script/build-dashboard-assets.sh
for file in src/App.tsx package.json pnpm-lock.yaml; do
  check_paths pull_request dashboard "internal/daemon/hub/src/dashboard/$file"
  check_paths push "$go_jobs licenses dashboard" "internal/daemon/hub/src/dashboard/$file"
done
check_paths pull_request '' internal/daemon/hub/src/dashboard/README.md
check_paths pull_request "$go_jobs licenses dashboard" README.md go.sum internal/daemon/hub/src/dashboard/src/App.tsx

for selected in '' "$all_jobs" "$go_jobs licenses" 'dashboard k8s'; do
  needs=$(jq -n --arg selected "$selected" '
    ($selected | split(" ")) as $selected |
    {changes: {result:"success",outputs:{}}, security: {result:"success"}} |
    reduce ["go-test","go-static","go-race","licenses","dashboard","container","workflow","k8s"][] as $job (.;
      ($selected | index($job) != null) as $enabled |
      .changes.outputs[$job] = ($enabled | tostring) |
      .[$job].result = (if $enabled then "success" else "skipped" end))')
  verify_ci_results <<< "$needs" >/dev/null
  for job in changes security go-test go-static go-race licenses dashboard container workflow k8s; do
    for result in failure cancelled skipped success; do
      [[ "$(jq -r --arg job "$job" '.[$job].result' <<< "$needs")" == "$result" ]] && continue
      bad=$(jq --arg job "$job" --arg result "$result" '.[$job].result = $result' <<< "$needs")
      if NEEDS="$bad" bash "$script" verify >/dev/null 2>&1; then
        echo "Incorrectly accepted $job=$result" >&2; exit 1
      fi
    done
  done
  for job in go-test go-static go-race licenses dashboard container workflow k8s; do
    for value in null '"invalid"' true; do
      bad=$(jq --arg job "$job" --argjson value "$value" '.changes.outputs[$job] = $value' <<< "$needs")
      if NEEDS="$bad" bash "$script" verify >/dev/null 2>&1; then exit 1; fi
    done
    bad=$(jq --arg job "$job" 'del(.[$job]) | del(.changes.outputs[$job])' <<< "$needs")
    if NEEDS="$bad" bash "$script" verify >/dev/null 2>&1; then exit 1; fi
  done
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
  diff -u <(printf '%s\0' 'internal/example with spaces.go' | classify_changes pull_request | jq -r 'to_entries[] | "\(.key)=\(.value)"') "$directory/output"
  GITHUB_EVENT_NAME=push CHANGE_BASE="$base" CHANGE_HEAD="$head" GITHUB_OUTPUT="$directory/push-output" bash "$script" changes >/dev/null
  grep -qx 'container=true' "$directory/push-output"
  grep -qx 'go-test=true' "$directory/push-output"
  GITHUB_EVENT_NAME=push CHANGE_BASE=0000000000000000000000000000000000000000 CHANGE_HEAD="$head" GITHUB_OUTPUT="$directory/initial-output" bash "$script" changes >/dev/null
  [[ "$(grep -c '=true$' "$directory/initial-output")" == 8 ]]
  git checkout -q feature
  git mv 'internal/example with spaces.go' example.md
  git commit -qm rename
  renamed=$(git rev-parse HEAD)
  GITHUB_EVENT_NAME=pull_request CHANGE_BASE="$head" CHANGE_HEAD="$renamed" GITHUB_OUTPUT="$directory/rename-output" bash "$script" changes >/dev/null
  grep -qx 'go-test=true' "$directory/rename-output"
  git checkout -q --detach "$head"
  git rm -q 'internal/example with spaces.go'
  git commit -qm delete
  deleted=$(git rev-parse HEAD)
  GITHUB_EVENT_NAME=pull_request CHANGE_BASE="$head" CHANGE_HEAD="$deleted" GITHUB_OUTPUT="$directory/delete-output" bash "$script" changes >/dev/null
  grep -qx 'go-test=true' "$directory/delete-output"
  if GITHUB_EVENT_NAME=push CHANGE_BASE=invalid CHANGE_HEAD="$head" GITHUB_OUTPUT="$directory/output" bash "$script" changes >/dev/null 2>&1; then exit 1; fi
  if GITHUB_EVENT_NAME=push CHANGE_BASE=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa CHANGE_HEAD="$head" GITHUB_OUTPUT="$directory/output" bash "$script" changes >/dev/null 2>&1; then exit 1; fi
)
echo 'CI policy tests passed'
