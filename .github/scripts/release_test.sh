#!/usr/bin/env bash
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
sha=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
[[ "$(release_version "$tag")" == 0.14.1 ]]
[[ "$(release_version v0.15.0-rc.1)" == 0.15.0-rc.1 ]]
for invalid in main ../v0.14.1 v0.14 v01.2.3 $'v1.2.3\nmain'; do expect_failure release_version "$invalid"; done
expect_failure select_artifacts unknown
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

green=$(jq -n --arg sha "$sha" '[{id:1,head_sha:$sha,head_branch:"main",event:"push",status:"completed",conclusion:"success"}]')
require_successful_ci "$sha" <<< "$green" >/dev/null
expect_failure require_successful_ci "$sha" <<< '[]'
for field in head_sha head_branch event status conclusion; do
  bad=$(jq --arg field "$field" '.[0][$field] = "wrong"' <<< "$green")
  expect_failure require_successful_ci "$sha" <<< "$bad"
done
bad=$(jq '. + [ (.[0] | .id = 2 | .conclusion = "failure") ]' <<< "$green")
expect_failure require_successful_ci "$sha" <<< "$bad"
require_new_assets "$tag" <<< '[{"name":"unrelated.txt"}]' >/dev/null
names=$(archive_names "$tag")
while IFS= read -r name; do
  expect_failure require_new_assets "$tag" <<< "$(jq -n --arg name "$name" '[{name:$name}]')"
done <<< "$names
checksums.txt"
[[ "$(should_publish_latest "$tag" <<< '{"tag_name":"v0.14.1","prerelease":false}')" == true ]]
[[ "$(should_publish_latest "$tag" <<< '{"tag_name":"v0.14.0","prerelease":false}')" == false ]]
[[ "$(should_publish_latest "$tag" <<< '{"tag_name":"v0.14.1","prerelease":true}')" == false ]]

directory=$(mktemp -d "${TMPDIR:-/tmp}/vine-release-test.XXXXXXXX")
trap 'rm -rf -- "$directory"' EXIT
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
echo 'Release policy tests passed'
