#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd -- "${script_dir}/.." && pwd -P)"

packages=(
  ./internal/utilfortest/goroutineleak
  ./internal/app
  ./internal/core/rpc/transport/inproc
  ./internal/core/web/inproc
  ./internal/daemon/hub/src/server/mod/scheduler
)

cd "${repo_dir}"
GOWORK=off go test -vet=off -count=1 -tags=goroutineleak -run '^TestGoroutineLeak' "${packages[@]}"
