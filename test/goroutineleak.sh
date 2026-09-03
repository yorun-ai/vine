#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd -- "${script_dir}/.." && pwd -P)"

packages=(
  ./internal/testutil/goroutineleak
  ./internal/app
  ./internal/core/rpc/transport/inproc
  ./internal/daemon/hub/src/server/mod/scheduler
  ./internal/infra/redis
)

cd "${repo_dir}"
GOWORK=off go test -count=1 -tags=goroutineleak -run '^TestGoroutineLeak' "${packages[@]}"
