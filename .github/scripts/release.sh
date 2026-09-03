#!/usr/bin/env bash
set -euo pipefail

release_version() {
  [[ "$1" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$ ]] || {
    echo "Invalid release tag: $1" >&2; return 1;
  }
  printf '%s\n' "${1#v}"
}

select_artifacts() {
  case "$1" in
    all) printf '{"binaries":true,"images":true}\n' ;;
    binaries) printf '{"binaries":true,"images":false}\n' ;;
    images) printf '{"binaries":false,"images":true}\n' ;;
    *) echo "Invalid artifacts selection: $1" >&2; return 1 ;;
  esac
}

archive_names() {
  local version
  version=$(release_version "$1")
  printf 'vine_%s_%s.tar.gz\n' "$version" darwin_amd64 "$version" darwin_arm64 "$version" linux_amd64 "$version" linux_arm64
}

require_successful_ci() {
  jq -e --arg sha "$1" '
    map(select(.head_sha == $sha and .event == "push" and .head_branch == "main")) |
    sort_by(.id) | last |
    if .status == "completed" and .conclusion == "success" then true
    else error("Latest main CI for the exact release commit must succeed") end'
}

require_new_assets() {
  local names
  names=$(archive_names "$1" | jq -Rsc 'split("\n")[:-1] + ["checksums.txt"]')
  jq -e --argjson names "$names" '
    map(.name | select(. as $name | $names | index($name))) |
    if length == 0 then true else error("Existing binary assets; use images-only recovery: " + join(", ")) end'
}

verify_release_jobs() {
  local selected
  selected=$(select_artifacts "$1")
  jq -e --argjson selected "$selected" '
    . as $needs | .validate.result == "success" and
    all(["binaries", "images"][]; . as $job |
      $needs[$job].result == (if $selected[$job] then "success" else "skipped" end)) |
    if . then true else error("Release failed or contains an unexpected skipped job") end'
}

should_publish_latest() {
  jq --arg tag "$1" '.prerelease == false and .tag_name == $tag'
}

verify_archive_contents() (
  local names
  names=$(archive_names "$1" | jq -Rsc 'split("\n")[:-1] | sort')
  cd "$2"
  # Validate exact filenames before letting sha256sum open anything from the list.
  jq -Rse --argjson names "$names" '
    split("\n") | if last == "" then .[:-1] else . end |
    if length == 4 and all(.[]; test("^[0-9a-f]{64}  \\S+$")) and
      (map(.[66:]) | sort) == $names then true
    else error("Invalid, duplicate, or incomplete checksum entries") end' checksums.txt
  sha256sum --check --strict checksums.txt
)

image_platforms() {
  jq -e '
    [.manifests[] | select(.platform.os != "unknown")] |
    if (map(.platform.os + "/" + .platform.architecture) | sort) == ["linux/amd64", "linux/arm64"] then .
    else error("Image must contain exactly Linux AMD64 and ARM64") end'
}

verify_labels() {
  jq -e --arg tag "$1" --arg sha "$2" --arg repo "$3" '
    if .["org.opencontainers.image.version"] == $tag and
       .["org.opencontainers.image.revision"] == $sha and
       .["org.opencontainers.image.source"] == ("https://github.com/" + $repo)
    then true else error("Image labels do not match the release source") end'
}

verify_images() {
  local repo="$1" tag="$2" sha="$3" check_latest="$4"
  local owner target image token index platforms digest descriptor config latest
  owner=$(printf '%s' "${repo%%/*}" | tr '[:upper:]' '[:lower:]')
  local accept='application/vnd.oci.image.index.v1+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.docker.distribution.manifest.v2+json'
  for target in hub link portal; do
    image="$owner/vine-$target"
    # Do not use GH_TOKEN here: users must be able to pull anonymously.
    token=$(curl --fail --silent --show-error --location --max-time 30 "https://ghcr.io/token?service=ghcr.io&scope=repository:$image:pull" | jq -er .token)
    local headers=(-H "Authorization: Bearer $token" -H "Accept: $accept")
    index=$(curl --fail --silent --show-error --location --max-time 30 "${headers[@]}" "https://ghcr.io/v2/$image/manifests/$tag")
    platforms=$(image_platforms <<< "$index")
    while IFS= read -r digest; do
      descriptor=$(curl --fail --silent --show-error --location --max-time 30 "${headers[@]}" "https://ghcr.io/v2/$image/manifests/$digest")
      digest=$(jq -er .config.digest <<< "$descriptor")
      config=$(curl --fail --silent --show-error --location --max-time 30 "${headers[@]}" "https://ghcr.io/v2/$image/blobs/$digest")
      jq -e .config.Labels <<< "$config" | verify_labels "$tag" "$sha" "$repo"
    done <<< "$(jq -r '.[].digest' <<< "$platforms")"
    if [[ "$check_latest" == true ]]; then
      latest=$(curl --fail --silent --show-error --location --max-time 30 "${headers[@]}" "https://ghcr.io/v2/$image/manifests/latest")
      jq -en --argjson latest "$latest" --argjson index "$index" '$latest == $index'
    fi
    echo "Verified ghcr.io/$image:$tag (anonymous, AMD64/ARM64, release revision)"
  done
}

verify_binaries() (
  local directory name names
  directory=$(mktemp -d "${TMPDIR:-/tmp}/vine-release-verify.XXXXXXXX")
  # Only this newly created temporary directory is removed.
  trap 'rm -rf -- "$directory"' EXIT
  names=$(archive_names "$2")
  local patterns=(--pattern checksums.txt)
  while IFS= read -r name; do patterns+=(--pattern "$name"); done <<< "$names"
  gh release download "$2" --repo "$1" --dir "$directory" "${patterns[@]}"
  verify_archive_contents "$2" "$directory"
)

release_main() {
  local command="${1:-}" tag="$RELEASE_TAG" repo="$GITHUB_REPOSITORY"
  local selected version sha release pages latest_tag publish
  selected=$(select_artifacts "${ARTIFACTS:-all}")
  version=$(release_version "$tag")
  case "$command" in
    validate)
      sha=$(git rev-parse HEAD)
      [[ "$(git rev-parse "refs/tags/$tag^{commit}")" == "$sha" ]]
      git merge-base --is-ancestor "$sha" refs/remotes/origin/main
      release=$(gh api "repos/$repo/releases/tags/$tag")
      jq -e --arg tag "$tag" '.draft == false and .tag_name == $tag' <<< "$release"
      jq -Rse --arg prefix "## [$version] - " '
        split("\n") | map(select(startswith($prefix))) |
        length == 1 and (.[0] | ltrimstr($prefix) | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}$"))' CHANGELOG.md
      pages=$(gh api --paginate --slurp "repos/$repo/actions/workflows/ci.yml/runs?event=push&branch=main&head_sha=$sha&per_page=100")
      jq '[.[].workflow_runs[]]' <<< "$pages" | require_successful_ci "$sha"
      if [[ "$(jq -r .binaries <<< "$selected")" == true ]]; then
        jq .assets <<< "$release" | require_new_assets "$tag"
      fi
      printf 'commit=%s\n' "$sha" >> "$GITHUB_OUTPUT"
      jq -r 'to_entries[] | "\(.key)=\(.value)"' <<< "$selected" >> "$GITHUB_OUTPUT"
      ;;
    preflight-binaries)
      gh api "repos/$repo/releases/tags/$tag" --jq .assets | require_new_assets "$tag"
      ;;
    verify|verify-latest)
      sha="$RELEASE_COMMIT"
      [[ "$sha" =~ ^[0-9a-f]{40}$ ]]
      if [[ "$command" == verify-latest ]]; then
        verify_images "$repo" "$tag" "$sha" true
        return
      fi
      verify_release_jobs "${ARTIFACTS:-all}" <<< "$NEEDS"
      verify_binaries "$repo" "$tag"
      verify_images "$repo" "$tag" "$sha" false
      release=$(gh api "repos/$repo/releases/tags/$tag")
      publish=false
      if [[ "$(jq -r .prerelease <<< "$release")" == false ]]; then
        latest_tag=$(gh api "repos/$repo/releases/latest" --jq .tag_name)
        publish=$(should_publish_latest "$latest_tag" <<< "$release")
      fi
      printf 'publish-latest=%s\n' "$publish" >> "$GITHUB_OUTPUT"
      ;;
    *) echo "Usage: release.sh validate|preflight-binaries|verify|verify-latest" >&2; return 1 ;;
  esac
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then release_main "$@"; fi
