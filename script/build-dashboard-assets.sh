#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd -- "${script_dir}/.." && pwd -P)"
dashboard_dir="${repo_dir}/internal/daemon/hub/src/server/mod/admin/dashboard"
assets_dir="${repo_dir}/internal/daemon/hub/src/server/mod/admin/dashboard/dist"
build_dir="$(mktemp -d "${TMPDIR:-/tmp}/vine-dashboard.XXXXXXXX")"
trap 'rm -rf -- "$build_dir"' EXIT

cd "${dashboard_dir}"
pnpm typecheck
node scripts/build-dashboard.mjs "${build_dir}/dist" "${build_dir}/dashboard-licenses.txt"
node scripts/dashboard-licenses.mjs --check "${build_dir}/dashboard-licenses.txt" "${repo_dir}/THIRD_PARTY_LICENSES.txt"
node scripts/package-assets.mjs "${build_dir}/dist" "${assets_dir}"
