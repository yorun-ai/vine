#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd -- "${script_dir}/.." && pwd -P)"
scope="${VINE_SHUFFLE_SCOPE:-targeted}"

case "${scope}" in
  targeted)
    packages=(
      ./internal/app
      ./internal/core/app
      ./internal/core/conf
      ./internal/core/event/spec
      ./internal/core/link/ingressinproc
      ./internal/core/logger
      ./internal/core/meta
      ./internal/core/rpc/spec
      ./internal/core/rpc/transport/inproc
      ./internal/core/skel
      ./internal/core/task/spec
      ./internal/core/web/inproc
      ./internal/core/web/spec
    )
    ;;
  all)
    packages=(./...)
    ;;
  *)
    printf 'Unsupported VINE_SHUFFLE_SCOPE: %s (expected targeted or all)\n' "${scope}" >&2
    exit 2
    ;;
esac

cd "${repo_dir}"
GOWORK=off go test -vet=off -count=1 -shuffle=on "${packages[@]}"
