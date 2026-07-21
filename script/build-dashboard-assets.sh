#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd -- "${script_dir}/.." && pwd -P)"
dashboard_dir="${repo_dir}/internal/daemon/hub/src/dashboard"
assets_dir="${repo_dir}/internal/daemon/hub/src/server/impl/dashboard/assets"
dashboard_tar_zst_file="${assets_dir}/dashboard.tar.zst"
dashboard_license_file="${dashboard_dir}/dist/THIRD_PARTY_LICENSES.md"
font_license_file="${dashboard_dir}/node_modules/@fontsource-variable/geist/LICENSE"
uiw_license_file="${dashboard_dir}/licenses/uiw-react-codemirror-MIT.txt"

mkdir -p "${assets_dir}"

command -v zstd >/dev/null 2>&1 || {
  echo "zstd is required to build dashboard assets" >&2
  exit 1
}

cd "${dashboard_dir}"
pnpm run build

for required_license_file in "${dashboard_license_file}" "${font_license_file}" "${uiw_license_file}"; do
  if [[ ! -f "${required_license_file}" ]]; then
    printf 'Required license file not found: %s\n' "${required_license_file}" >&2
    exit 1
  fi
done

font_version="$(node -p "require('./node_modules/@fontsource-variable/geist/package.json').version")"
uiw_version="$(node -p "require('./node_modules/@uiw/react-codemirror/package.json').version")"

{
  printf '\n## @fontsource-variable/geist - %s (OFL-1.1)\n\n' "${font_version}"
  sed -e '$a\' "${font_license_file}"
  printf '\n## @uiw/codemirror-extensions-basic-setup and @uiw/react-codemirror - %s (MIT)\n\n' "${uiw_version}"
  sed -e '$a\' "${uiw_license_file}"
} >> "${dashboard_license_file}"

tar -C "${dashboard_dir}/dist" -cf - . | zstd -19 -q -f -o "${dashboard_tar_zst_file}"
printf 'Dashboard assets written: %s (%s)\n' "${dashboard_tar_zst_file}" "$(du -h "${dashboard_tar_zst_file}" | awk '{print $1}')"
