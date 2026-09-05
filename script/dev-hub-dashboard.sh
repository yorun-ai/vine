#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd -- "${script_dir}/.." && pwd -P)"
dashboard_dir="${repo_dir}/internal/daemon/hub/src/dashboard"

cd "${dashboard_dir}"
exec pnpm run dev --strictPort "$@"
