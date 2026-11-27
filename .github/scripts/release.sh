#!/usr/bin/env bash
set -euo pipefail

fail() { echo "Release validation: $*" >&2; return 1; }

# GET only. curl retries transient HTTP errors/timeouts, not permission failures.
# Buffer the body so a partial failed response never reaches the JSON parser.
read_url() (
  local response cleanup
  response=$(mktemp "${TMPDIR:-/tmp}/vine-http.XXXXXXXX") || return
  printf -v cleanup 'rm -f -- %q' "$response"
  # Capture the quoted path now; locals may be gone when EXIT runs after failure.
  # shellcheck disable=SC2064
  trap "$cleanup" EXIT
  curl --fail --silent --show-error --location --connect-timeout 10 --max-time 30 \
    --retry 3 --retry-max-time 90 --output "$response" "$@" || return $?
  cat "$response"
)

require_release_checkout() {
  local tag="$1" sha="$2" tag_sha
  tag_sha=$(git rev-parse --verify "refs/tags/$tag^{commit}") || { fail "Cannot resolve release tag $tag"; return 1; }
  [[ "$tag_sha" == "$sha" ]] || { fail "Checkout $sha does not match $tag ($tag_sha)"; return 1; }
  git merge-base --is-ancestor "$sha" refs/remotes/origin/main || {
    fail "Release commit $sha is not an ancestor of origin/main (or main is unavailable)"; return 1;
  }
}

require_published_release() {
  jq -e --arg tag "$1" '
    if .draft == false and .tag_name == $tag then true
    else error("Expected published release " + $tag + "; actual tag=" + (.tag_name // "missing") + ", draft=" + (.draft | tostring)) end'
}

require_changelog() {
  jq -Rse --arg prefix "## [$1] - " '
    split("\n") | map(select(startswith($prefix))) |
    if length != 1 then error("Expected exactly one changelog heading starting with " + $prefix)
    elif (.[0] | ltrimstr($prefix) | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}$") | not)
    then error("Changelog release date must use YYYY-MM-DD: " + .[0])
    else true end'
}

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

# Include drafts, and propagate API/auth failures instead of treating them as missing.
find_release() {
  gh api --paginate --slurp "repos/$1/releases?per_page=100" |
    jq -ce --arg tag "$2" '
      [.[][] | select(.tag_name == $tag)] |
      if length == 0 then {} elif length == 1 then .[0]
      else error("Duplicate release tag") end'
}

release_notes() {
  require_changelog "$1" < CHANGELOG.md >/dev/null
  awk -v prefix="## [$1] - " '
    index($0, prefix) == 1 { found = 1; next }
    found && /^## / { exit }
    found { print; if ($0 ~ /[^[:space:]]/) body = 1 }
    END { if (!body) exit 1 }
  ' CHANGELOG.md
}

require_draft() {
  jq -e --arg tag "$1" '.tag_name == $tag and .draft == true' >/dev/null || {
    fail "Expected unpublished draft $1; refusing to overwrite published assets"; return 1;
  }
}

require_binary_metadata() {
  awk -v version="$1" -v revision="$2" '
    $1 == "mod" && $2 == "go.yorun.ai/vine" && $3 == version { module_ok = 1 }
    $1 == "build" && $2 == "vcs.revision=" revision { revision_ok = 1 }
    $1 == "build" && $2 == "vcs.modified=false" { clean = 1 }
    END { exit !(module_ok && revision_ok && clean) }
  ' || { fail "Binary metadata must match clean release $1 at $2"; return 1; }
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

image_build_needed() {
  local image="$1" tag="$2" token status
  token=$(read_url --user "$GITHUB_ACTOR:$GH_TOKEN" \
    "https://ghcr.io/token?service=ghcr.io&scope=repository:$image:pull" | jq -er .token) || return $?
  status=$(curl --silent --show-error --location --connect-timeout 10 --max-time 30 \
    --output /dev/null --write-out '%{http_code}' \
    -H "Authorization: Bearer $token" \
    -H 'Accept: application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json' \
    "https://ghcr.io/v2/$image/manifests/$tag") || return $?
  case "$status" in
    200) echo false ;;
    404) echo true ;;
    *) fail "Cannot check ghcr.io/$image:$tag (HTTP $status)"; return 1 ;;
  esac
}

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
    token=$(read_url "https://ghcr.io/token?service=ghcr.io&scope=repository:$image:pull" | jq -er .token)
    local headers=(-H "Authorization: Bearer $token" -H "Accept: $accept")
    index=$(read_url "${headers[@]}" "https://ghcr.io/v2/$image/manifests/$tag")
    platforms=$(image_platforms <<< "$index")
    while IFS= read -r digest; do
      descriptor=$(read_url "${headers[@]}" "https://ghcr.io/v2/$image/manifests/$digest")
      digest=$(jq -er .config.digest <<< "$descriptor")
      config=$(read_url "${headers[@]}" "https://ghcr.io/v2/$image/blobs/$digest")
      jq -e .config.Labels <<< "$config" | verify_labels "$tag" "$sha" "$repo"
    done <<< "$(jq -r '.[].digest' <<< "$platforms")"
    if [[ "$check_latest" == true ]]; then
      latest=$(read_url "${headers[@]}" "https://ghcr.io/v2/$image/manifests/latest")
      jq -en --arg image "$image" --arg tag "$tag" --argjson latest "$latest" --argjson index "$index" \
        'if $latest == $index then true else error($image + ": latest does not match " + $tag) end'
    fi
    echo "Verified ghcr.io/$image:$tag (anonymous, AMD64/ARM64, release revision)"
  done
}

verify_binaries() (
  local directory name names cleanup
  directory=$(mktemp -d "${TMPDIR:-/tmp}/vine-release-verify.XXXXXXXX")
  # Only this newly created temporary directory is removed.
  printf -v cleanup 'rm -rf -- %q' "$directory"
  # shellcheck disable=SC2064
  trap "$cleanup" EXIT
  names=$(archive_names "$2")
  local patterns=(--pattern checksums.txt)
  while IFS= read -r name; do patterns+=(--pattern "$name"); done <<< "$names"
  gh release download "$2" --repo "$1" --dir "$directory" "${patterns[@]}"
  verify_archive_contents "$2" "$directory"
)

release_main() {
  local command="${1:-}" tag="$RELEASE_TAG" repo="$GITHUB_REPOSITORY"
  local selected version sha release latest_tag publish
  selected=$(select_artifacts "${ARTIFACTS:-all}")
  version=$(release_version "$tag")
  case "$command" in
    image-build-needed)
      case "${IMAGE_TARGET:-}" in hub|link|portal) ;; *) fail "Invalid image target"; return 1 ;; esac
      local image build
      image="$(printf '%s' "${repo%%/*}" | tr '[:upper:]' '[:lower:]')/vine-$IMAGE_TARGET"
      build=$(image_build_needed "$image" "$tag")
      printf 'build=%s\n' "$build" >> "$GITHUB_OUTPUT"
      if [[ "$build" == false ]]; then echo "Skipping existing ghcr.io/$image:$tag; final release verification still applies"; fi
      ;;
    validate)
      sha=$(git rev-parse HEAD)
      require_release_checkout "$tag" "$sha"
      release_notes "$version" >/dev/null
      release=$(find_release "$repo" "$tag")
      if [[ "$(jq -r .binaries <<< "$selected")" == true ]]; then
        if [[ "$(jq -r .draft <<< "$release")" == false ]]; then
          fail "Published binary assets cannot be replaced; use images-only recovery"; return 1
        fi
      else
        # Images-only recovery requires existing complete binary attachments.
        verify_binaries "$repo" "$tag"
      fi
      printf 'commit=%s\n' "$sha" >> "$GITHUB_OUTPUT"
      jq -r 'to_entries[] | "\(.key)=\(.value)"' <<< "$selected" >> "$GITHUB_OUTPUT"
      ;;
    upload-binaries)
      verify_archive_contents "$tag" dist
      local notes prerelease=false
      notes="$RUNNER_TEMP/vine-release-notes.md"
      release_notes "$version" > "$notes"
      release=$(find_release "$repo" "$tag")
      [[ "$tag" != *-* ]] || prerelease=true
      if [[ "$(jq -r '.tag_name // empty' <<< "$release")" == "" ]]; then
        gh release create "$tag" --repo "$repo" --verify-tag --draft \
          --prerelease="$prerelease" --title "$tag" --notes-file "$notes"
      else
        require_draft "$tag" <<< "$release"
        gh release edit "$tag" --repo "$repo" --prerelease="$prerelease" \
          --title "$tag" --notes-file "$notes"
      fi
      gh release upload "$tag" dist/vine_*.tar.gz dist/checksums.txt --repo "$repo" --clobber
      verify_binaries "$repo" "$tag"
      ;;
    latest-eligible)
      release=$(gh api "repos/$repo/releases/tags/$tag")
      require_published_release "$tag" <<< "$release"
      publish=false
      if [[ "$(jq -r .prerelease <<< "$release")" == false ]]; then
        latest_tag=$(gh api "repos/$repo/releases/latest" --jq .tag_name)
        publish=$(should_publish_latest "$latest_tag" <<< "$release")
      fi
      printf 'publish-latest=%s\n' "$publish" >> "$GITHUB_OUTPUT"
      ;;
    verify|verify-latest)
      sha="$RELEASE_COMMIT"
      [[ "$sha" =~ ^[0-9a-f]{40}$ ]] || { fail "Invalid release commit: $sha"; return 1; }
      if [[ "$command" == verify-latest ]]; then
        verify_images "$repo" "$tag" "$sha" true
        return
      fi
      verify_release_jobs "${ARTIFACTS:-all}" <<< "$NEEDS"
      verify_binaries "$repo" "$tag"
      verify_images "$repo" "$tag" "$sha" false
      release=$(find_release "$repo" "$tag")
      if [[ "$(jq -r .draft <<< "$release")" == true ]]; then
        require_draft "$tag" <<< "$release"
        gh release edit "$tag" --repo "$repo" --verify-tag --draft=false
      else
        require_published_release "$tag" <<< "$release"
      fi
      # Publication must complete before querying latest eligibility.
      release_main latest-eligible
      ;;
    *) echo "Usage: release.sh validate|upload-binaries|image-build-needed|verify|latest-eligible|verify-latest" >&2; return 1 ;;
  esac
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then release_main "$@"; fi
