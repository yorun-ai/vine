#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_dir="$(cd "${script_dir}/.." && pwd)"

cd "${repo_dir}"

usage() {
  cat <<'EOF'
Usage: bash script/gen-skel.sh [all|app|hub|link]...

Targets:
  all   generate all skeleton code
  app   generate internal/core/app/skeled
  hub   generate Hub control/admin Go and admin TypeScript skeled packages
  link  generate internal/core/link/skeled
EOF
}

generate_app_skel() {
  local skel_dir="${repo_dir}/internal/core/app/skel"
  local target_dir="${repo_dir}/internal/core/app/skeled"

  skelc --strict gen go --skel-in "${skel_dir}" --go-out "${target_dir}"
}

generate_hub_skel_domain() {
  local skel_dir="$1"
  local api_dir="$2"
  local frontend_dir="${3:-}"

  skelc --strict gen go --skel-in "${skel_dir}" --go-out "${api_dir}"
  if [[ -n "${frontend_dir}" ]]; then
    skelc --strict gen ts --api --skel-in "${skel_dir}" --ts-out "${frontend_dir}"
  fi
}

generate_hub_skel() {
  generate_hub_skel_domain \
    "${repo_dir}/internal/daemon/hub/skel/control" \
    "${repo_dir}/internal/daemon/hub/api/skeled/control"
  generate_hub_skel_domain \
    "${repo_dir}/internal/daemon/hub/skel/admin" \
    "${repo_dir}/internal/daemon/hub/api/skeled/admin" \
    "${repo_dir}/internal/daemon/hub/src/server/mod/admin/dashboard/src/skeled/admin"
}

generate_link_skel() {
  local skel_dir="${repo_dir}/internal/daemon/link/skel"
  local target_dir="${repo_dir}/internal/core/link/skeled"

  skelc --strict gen go --skel-in "${skel_dir}" --go-out "${target_dir}"
}

run_target() {
  local target="$1"

  case "${target}" in
  all)
    generate_app_skel
    generate_hub_skel
    generate_link_skel
    ;;
  app)
    generate_app_skel
    ;;
  hub)
    generate_hub_skel
    ;;
  link)
    generate_link_skel
    ;;
  -h | --help | help)
    usage
    ;;
  *)
    echo "unknown gen-skel target: ${target}" >&2
    usage >&2
    exit 1
    ;;
  esac
}

if [[ $# -eq 0 ]]; then
  run_target all
else
  for target in "$@"; do
    run_target "${target}"
  done
fi
