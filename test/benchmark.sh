#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd -- "${script_dir}/.." && pwd -P)"

benchmark_pattern="${VINE_BENCH_PATTERN:-.}"
benchmark_time="${VINE_BENCH_TIME:-1s}"
benchmark_count="${VINE_BENCH_COUNT:-5}"

packages=(
  ./internal/core/rpc/transport/http
  ./internal/daemon/portal/src/server/mod/access
  ./internal/daemon/portal/src/server/util/computil
  ./internal/infra/redis
  ./internal/core/event
  ./internal/core/task
)

cd "${repo_dir}"
GOWORK=off go test "${packages[@]}" \
  -run '^$' \
  -bench "${benchmark_pattern}" \
  -benchmem \
  -benchtime "${benchmark_time}" \
  -count "${benchmark_count}" \
  "$@"
