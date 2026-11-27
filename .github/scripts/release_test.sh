#!/usr/bin/env bash
# Each subshell initializes its own release environment; no state crosses fixtures.
# shellcheck disable=SC2030,SC2031
set -euo pipefail
script="$(cd "$(dirname "$0")" && pwd)/release.sh"
# shellcheck source=.github/scripts/release.sh
source "$script"

expect_failure() {
  # Run a separate shell: an if-condition must not disable errexit inside helpers.
  if bash -e -o pipefail -c 'source "$1"; shift; "$@"' _ "$script" "$@" >/dev/null 2>&1; then
    echo "Expected failure: $*" >&2; exit 1
  fi
}

tag=v0.14.1
(
  read_url() { printf '{"token":"fixture-token"}'; return "${TOKEN_EXIT:-0}"; }
  curl() { printf '%s' "$REGISTRY_STATUS"; return "${CURL_EXIT:-0}"; }
  export -f read_url curl image_build_needed fail
  expect_registry_failure() {
    if bash -e -o pipefail -c 'image_build_needed yorun-ai/vine-hub v0.15.8' >/dev/null 2>&1; then
      echo 'Expected registry check failure' >&2; exit 1
    fi
  }
  export GITHUB_ACTOR=fixture GH_TOKEN=fixture REGISTRY_STATUS=200
  [[ "$(image_build_needed yorun-ai/vine-hub v0.15.8)" == false ]]
  export REGISTRY_STATUS=404
  [[ "$(image_build_needed yorun-ai/vine-hub v0.15.8)" == true ]]
  for status in 401 403 429 500 503; do
    export REGISTRY_STATUS="$status"
    expect_registry_failure
  done
  export REGISTRY_STATUS=404 CURL_EXIT=28
  expect_registry_failure
  export CURL_EXIT=0
  export TOKEN_EXIT=22
  expect_registry_failure
)
sha=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
[[ "$(release_version "$tag")" == 0.14.1 ]]
[[ "$(release_version v0.15.0-rc.1)" == 0.15.0-rc.1 ]]
for invalid in main ../v0.14.1 v0.14 v01.2.3 $'v1.2.3\nmain'; do expect_failure release_version "$invalid"; done
expect_failure select_artifacts unknown
require_published_release "$tag" <<< '{"tag_name":"v0.14.1","draft":false}' >/dev/null
for release in '{"tag_name":"v0.14.1","draft":true}' '{"tag_name":"other","draft":false}'; do
  expect_failure require_published_release "$tag" <<< "$release"
done
require_changelog 0.14.1 <<< '## [0.14.1] - 2026-09-04' >/dev/null
for changelog in '' '## [0.14.1] - TBD' $'## [0.14.1] - 2026-09-04\n## [0.14.1] - 2026-09-04'; do
  expect_failure require_changelog 0.14.1 <<< "$changelog"
done
diagnostic=$(bash -c 'source "$1"; require_changelog 0.14.1' _ "$script" <<< '## [0.14.1] - TBD' 2>&1 || true)
[[ "$diagnostic" == *'YYYY-MM-DD'* ]]
for choice in all binaries images; do
  selected=$(select_artifacts "$choice")
  jq -e --arg choice "$choice" '.binaries == ($choice != "images") and .images == ($choice != "binaries")' <<< "$selected" >/dev/null
  needs=$(jq -n --argjson selected "$selected" '{validate: {result:"success"}} |
    reduce ["binaries","images"][] as $job (.; .[$job].result = (if $selected[$job] then "success" else "skipped" end))')
  verify_release_jobs "$choice" <<< "$needs" >/dev/null
  for job in validate binaries images; do
    for result in failure cancelled skipped success; do
      [[ "$(jq -r --arg job "$job" '.[$job].result' <<< "$needs")" == "$result" ]] && continue
      bad=$(jq --arg job "$job" --arg result "$result" '.[$job].result = $result' <<< "$needs")
      expect_failure verify_release_jobs "$choice" <<< "$bad"
    done
  done
done

require_draft "$tag" <<< '{"tag_name":"v0.14.1","draft":true}'
expect_failure require_draft "$tag" <<< '{"tag_name":"v0.14.1","draft":false}'
expect_failure require_draft "$tag" <<< '{"tag_name":"other","draft":true}'
metadata=$(printf 'mod go.yorun.ai/vine %s\nbuild vcs.revision=%s\nbuild vcs.modified=false\n' "$tag" "$sha")
require_binary_metadata "$tag" "$sha" <<< "$metadata"
for bad in "${metadata//$tag/v0.0.0-20260909114016-63ce09d208d1}" "${metadata//$tag/$tag+dirty}" "${metadata//false/true}" "${metadata//$sha/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb}" ''; do
  expect_failure require_binary_metadata "$tag" "$sha" <<< "$bad"
done
names=$(archive_names "$tag")
[[ "$(should_publish_latest "$tag" <<< '{"tag_name":"v0.14.1","prerelease":false}')" == true ]]
[[ "$(should_publish_latest "$tag" <<< '{"tag_name":"v0.14.0","prerelease":false}')" == false ]]
[[ "$(should_publish_latest "$tag" <<< '{"tag_name":"v0.14.1","prerelease":true}')" == false ]]

directory=$(mktemp -d "${TMPDIR:-/tmp}/vine-release-test.XXXXXXXX")
trap 'rm -rf -- "$directory"' EXIT
(
  mkdir "$directory/repo"
  cd "$directory/repo"
  git init -q -b main
  git config user.name 'Release fixture'
  git config user.email 'ci@example.invalid'
  git commit -qm initial --allow-empty
  commit=$(git rev-parse HEAD)
  git tag "$tag"
  git update-ref refs/remotes/origin/main "$commit"
  require_release_checkout "$tag" "$commit"
  expect_failure require_release_checkout missing "$commit"
  expect_failure require_release_checkout "$tag" "$sha"
  git commit -qm second --allow-empty
  git tag v0.14.2
  expect_failure require_release_checkout v0.14.2 "$(git rev-parse HEAD)"
)

# Test our curl invocation and buffering; retry scheduling belongs to curl.
(
  curl() {
    local output='' retry='' window='' connect='' limit=''
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --output) output="$2"; shift ;;
        --retry) retry="$2"; shift ;;
        --retry-max-time) window="$2"; shift ;;
        --connect-timeout) connect="$2"; shift ;;
        --max-time) limit="$2"; shift ;;
        --retry-all-errors|-X|--request|--data) return 99 ;;
      esac
      shift
    done
    [[ "$retry" == 3 && "$window" == 90 && "$connect" == 10 && "$limit" == 30 && -n "$output" ]] || return 99
    printf '%s' "$output" > "$CURL_RECORD"
    if [[ "$CURL_RESULT" == success ]]; then
      printf '{"ok":true}' > "$output"
    else
      printf 'partial response' > "$output"
      return 22
    fi
  }
  export -f curl
  export CURL_RECORD="$directory/curl-path" CURL_RESULT=success
  [[ "$(read_url https://example.invalid)" == '{"ok":true}' ]]
  [[ ! -e "$(< "$CURL_RECORD")" ]]
  export CURL_RESULT=failure
  if bash -c 'source "$1"; read_url https://example.invalid' _ "$script" > "$directory/response"; then exit 1; fi
  [[ ! -s "$directory/response" && ! -e "$(< "$CURL_RECORD")" ]]
  if response=$(read_url https://example.invalid); then exit 1; fi
  [[ -z "$response" && ! -e "$(< "$CURL_RECORD")" ]]
)

# Eligibility is queried again in the serialized job, not trusted from verify.
(
  # Exported to the child running release.sh.
  # shellcheck disable=SC2317,SC2329
  gh() {
    # shellcheck disable=SC2317
    case "$2" in
      */releases/latest) printf '%s\n' "$LATEST_TAG" ;;
      */releases/tags/*) printf '%s\n' "$RELEASE_JSON" ;;
      *) return 1 ;;
    esac
  }
  export -f gh
  export RELEASE_TAG="$tag" GITHUB_REPOSITORY=yorun-ai/vine
  export RELEASE_JSON='{"tag_name":"v0.14.1","draft":false,"prerelease":false}'
  export GITHUB_OUTPUT="$directory/eligibility"
  for latest in v0.14.1 v0.14.2; do
    export LATEST_TAG="$latest"
    : > "$GITHUB_OUTPUT"
    bash "$script" latest-eligible >/dev/null
    expected=false
    [[ "$latest" != "$tag" ]] || expected=true
    [[ "$(< "$GITHUB_OUTPUT")" == "publish-latest=$expected" ]]
  done
  export RELEASE_JSON='{"tag_name":"v0.14.1","draft":false,"prerelease":true}' LATEST_TAG="$tag"
  : > "$GITHUB_OUTPUT"
  bash "$script" latest-eligible >/dev/null
  [[ "$(< "$GITHUB_OUTPUT")" == 'publish-latest=false' ]]
)
while IFS= read -r name; do printf '%s' "$name" > "$directory/$name"; done <<< "$names"
(cd "$directory" && sha256sum vine_*.tar.gz > checksums.txt)
checksums=$(< "$directory/checksums.txt")
verify_archive_contents "$tag" "$directory" >/dev/null
printf '%s\n' "$checksums" | tail -n 3 > "$directory/checksums.txt"
expect_failure verify_archive_contents "$tag" "$directory"
printf '%s\n' "$checksums" | sed '1s@vine_[^ ]*@../escape@' > "$directory/checksums.txt"
expect_failure verify_archive_contents "$tag" "$directory"
printf '%s\n' "$checksums" | sed '2d' > "$directory/checksums.txt"
printf '%s\n' "$checksums" | head -n 1 >> "$directory/checksums.txt"
expect_failure verify_archive_contents "$tag" "$directory"
printf '%s\n' "$checksums" > "$directory/checksums.txt"
printf corrupt > "$directory/vine_0.14.1_linux_amd64.tar.gz"
expect_failure verify_archive_contents "$tag" "$directory"

index='{"manifests":[{"platform":{"os":"linux","architecture":"amd64"}},{"platform":{"os":"linux","architecture":"arm64"}},{"platform":{"os":"unknown","architecture":"unknown"}}]}'
image_platforms <<< "$index" >/dev/null
for filter in 'del(.manifests[0])' '.manifests[1].platform.architecture = "amd64"' '.manifests += [{platform:{os:"linux",architecture:"386"}}]'; do
  expect_failure image_platforms <<< "$(jq "$filter" <<< "$index")"
done
labels=$(jq -n --arg tag "$tag" --arg sha "$sha" '{"org.opencontainers.image.version":$tag,"org.opencontainers.image.revision":$sha,"org.opencontainers.image.source":"https://github.com/yorun-ai/vine"}')
verify_labels "$tag" "$sha" yorun-ai/vine <<< "$labels" >/dev/null
for field in version revision source; do
  expect_failure verify_labels "$tag" "$sha" yorun-ai/vine <<< "$(jq --arg field "org.opencontainers.image.$field" '.[$field] = "wrong"' <<< "$labels")"
done
# Exercise lifecycle ordering without GitHub writes or registry requests.
(
  cd "$directory"
  export RELEASE_TAG="$tag" GITHUB_REPOSITORY=yorun-ai/vine RUNNER_TEMP="$directory"
  export GITHUB_OUTPUT="$directory/lifecycle-output" RELEASE_COMMIT="$sha" ARTIFACTS=all
  export NEEDS='{"validate":{"result":"success"},"binaries":{"result":"success"},"images":{"result":"success"}}'
  printf '## [0.14.1] - 2026-10-05\n\n### Changed\n- Fixture.\n' > CHANGELOG.md
  state="$directory/release-state"
  events="$directory/release-events"
  printf '{}' > "$state"
  : > "$events"
  mkdir dist
  while IFS= read -r name; do echo fixture > "dist/$name"; done <<< "$names"
  (cd dist && sha256sum vine_*.tar.gz > checksums.txt)
  fixture_sha="$sha"
  git() { echo "$fixture_sha"; }
  require_release_checkout() { :; }
  verify_binaries() { echo binaries >> "$events"; test "${BAD_BINARY:-false}" = false; }
  verify_images() { echo images >> "$events"; test "${BAD_IMAGE:-false}" = false; }
  gh() {
    case "$1 $2" in
      'api --paginate')
        test "${API_FAIL:-false}" = false || return 1
        jq -c 'if .tag_name then [[.]] else [[]] end' "$state" ;;
      'api repos/yorun-ai/vine/releases/latest') echo "$tag" ;;
      'api repos/yorun-ai/vine/releases/tags/'*) cat "$state" ;;
      'release create')
        echo create >> "$events"
        printf '{"tag_name":"%s","draft":true,"prerelease":false}' "$tag" > "$state" ;;
      'release edit')
        if [[ " $* " == *' --draft=false '* ]]; then
          echo publish >> "$events"
          printf '{"tag_name":"%s","draft":false,"prerelease":false}' "$tag" > "$state"
        fi ;;
      'release upload') echo upload >> "$events"; test "${UPLOAD_FAIL:-false}" = false ;;
      *) echo "Unexpected gh call: $*" >&2; return 1 ;;
    esac
  }
  lifecycle_failure() {
    local result
    set +e
    (set -e; release_main "$1") > "$directory/lifecycle-error" 2>&1
    result=$?
    set -e
    [[ "$result" != 0 ]] || { echo "Expected lifecycle failure: $1" >&2; exit 1; }
  }
  API_FAIL=true lifecycle_failure validate
  release_main validate
  UPLOAD_FAIL=true lifecycle_failure upload-binaries
  jq -e '.draft == true' "$state" >/dev/null
  release_main upload-binaries
  : > "$events"
  BAD_BINARY=true lifecycle_failure verify
  jq -e '.draft == true' "$state" >/dev/null
  BAD_IMAGE=true lifecycle_failure verify
  jq -e '.draft == true' "$state" >/dev/null
  NEEDS='{"validate":{"result":"success"},"binaries":{"result":"failure"},"images":{"result":"success"}}' lifecycle_failure verify
  : > "$events"
  release_main verify
  [[ "$(cat "$events")" == $'binaries\nimages\npublish' ]]
  jq -e '.draft == false' "$state" >/dev/null
  : > "$events"
  lifecycle_failure upload-binaries
  lifecycle_failure validate
  [[ ! -s "$events" ]]
  # Published releases still support images-only recovery, without edits.
  ARTIFACTS=images
  NEEDS='{"validate":{"result":"success"},"binaries":{"result":"skipped"},"images":{"result":"success"}}'
  release_main validate
  release_main verify
  if grep -qE 'publish|upload|create' "$events"; then exit 1; fi
)
echo 'Release policy tests passed'
