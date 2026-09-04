#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$repo_dir"

if (( $# > 1 )); then
  echo "Usage: bash test/k8s.sh [vX.Y.Z]" >&2
  exit 1
fi

command -v yq >/dev/null || { echo "Install mikefarah/yq v4 to validate Kubernetes YAML" >&2; exit 1; }
command -v jq >/dev/null || { echo "Install jq to validate rendered resources" >&2; exit 1; }
if command -v kustomize >/dev/null; then
  render=(kustomize build)
elif command -v kubectl >/dev/null; then
  render=(kubectl kustomize)
else
  render=(env GOWORK=off go run sigs.k8s.io/kustomize/kustomize/v5@v5.8.1 build)
fi

stable_version=$(yq -r '.images[0].newTag' deploy/k8s/overlays/stable/kustomization.yaml)
[[ "$stable_version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
  echo "Stable must pin a release version, got: $stable_version" >&2; exit 1;
}
if [[ $# == 1 && "$stable_version" != "$1" ]]; then
  echo "Stable version $stable_version does not match release $1" >&2
  exit 1
fi

check_render() {
  local path="$1" version="$2" policy="$3" mtls="$4"
  "${render[@]}" "$path" | yq -o=json -I=0 '.' | jq -se \
    --arg version "$version" --arg policy "$policy" --argjson mtls "$mtls" '
    def pods: .[] | select(.kind == "Deployment" or .kind == "StatefulSet");
    def containers: (.spec.template.spec.containers + (.spec.template.spec.initContainers // []))[];
    def envmap: (.env // []) | map({key: .name, value: .value}) | from_entries;
    (length == 7) and
    ([.[] | select(.kind == "Namespace") | .metadata.name] == ["vine"]) and
    (all(.[] | select(.kind != "Namespace"); .metadata.namespace == "vine")) and
    ([pods | .metadata.name] | sort == ["hub", "link", "portal"]) and
    ([pods | containers] | length == 5) and
    (all(pods; .metadata.name as $role | all(containers;
      .image == ("ghcr.io/yorun-ai/vine-" + $role + ":" + $version) and
      .imagePullPolicy == $policy))) and
    (all(pods; .metadata.name as $role |
      .spec.template.spec as $pod |
      all($pod.containers[];
        envmap as $env |
        (if $role != "hub" then
          $env.VINE_HUB_ENDPOINT == (if $mtls then "https://hub:7071" else "http://hub:7071" end)
        else true end) and
        (if $mtls then
          $env.VINE_MTLS_CA_FILE == "/run/vine/mtls/ca.pem" and
          $env.VINE_MTLS_CERT_FILE == "/run/vine/mtls/cert.pem" and
          $env.VINE_MTLS_KEY_FILE == "/run/vine/mtls/key.pem" and
          any(.volumeMounts[]; .name == "mtls" and .readOnly == true) and
          any($pod.volumes[]; .name == "mtls" and .secret.secretName == ("vine-" + $role + "-mtls"))
        else ($env | has("VINE_MTLS_CA_FILE") | not) end))))
    | if . then true else error("Invalid deployment resources, image versions, pull policy, or mTLS configuration") end
  ' >/dev/null
  echo "Validated $path ($version, mTLS=$mtls)"
}

check_render deploy/k8s "$stable_version" IfNotPresent false
check_render deploy/k8s/overlays/stable "$stable_version" IfNotPresent false
check_render deploy/k8s/overlays/stable-mtls "$stable_version" IfNotPresent true
