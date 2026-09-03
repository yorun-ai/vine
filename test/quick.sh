#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd -- "${script_dir}/.." && pwd -P)"

usage() {
  printf '%s\n' \
    'Usage: bash test/quick.sh <scope> [scope...]' \
    'Scopes: app cli core rpc web event task infra hub link portal all' >&2
}

if [[ "$#" -eq 0 ]]; then
  usage
  exit 2
fi

patterns=()
for scope in "$@"; do
  case "${scope}" in
    app)
      patterns+=(
        ./app/...
        ./internal/app
        ./internal/core/app/...
        ./internal/core/conf
        ./internal/core/ctr
        ./internal/core/di
        ./internal/core/runtime
      )
      ;;
    cli)
      patterns+=(
        ./cmd/vine
        ./internal/appcli
        ./internal/cli
      )
      ;;
    core)
      patterns+=(
        ./core/...
        ./internal/app
        ./internal/core/...
      )
      ;;
    rpc)
      patterns+=(
        ./core/rpc
        ./internal/app
        ./internal/core/rpc/...
        ./internal/daemon/hub/src/server/impl/admin/debug
        ./internal/daemon/link/src/server/mod/rpcproxy
        ./internal/daemon/portal/src/server/mod/access
        ./internal/daemon/portal/src/server/mod/site/rpcgw
        ./internal/daemon/portal/src/server/util/gwutil
      )
      ;;
    web)
      patterns+=(
        ./core/web
        ./internal/app
        ./internal/core/web/...
        ./internal/daemon/link/src/server/mod/ingress
        ./internal/daemon/link/src/server/mod/webproxy
        ./internal/daemon/portal/src/server/mod/site/webgw
        ./internal/daemon/portal/src/server/util/gwutil
      )
      ;;
    event)
      patterns+=(
        ./core/event
        ./internal/app
        ./internal/core/event/...
        ./internal/daemon/hub/api/nats
        ./internal/daemon/link/src/server/comp/nats
        ./internal/daemon/link/src/server/mod/event
      )
      ;;
    task)
      patterns+=(
        ./core/task
        ./internal/app
        ./internal/core/task/...
        ./internal/daemon/hub/api/nats
        ./internal/daemon/link/src/server/comp/nats
        ./internal/daemon/link/src/server/mod/task
      )
      ;;
    infra)
      patterns+=(
        ./infra/...
        ./internal/infra/...
        ./internal/daemon/hub/src/server/repo/...
      )
      ;;
    hub)
      patterns+=(
        ./internal/daemon/hub/...
        ./internal/daemon/link/src/server/comp/hubinfo
        ./internal/daemon/link/src/server/comp/hubredis
        ./internal/daemon/link/src/server/mod/config
        ./internal/daemon/link/src/server/mod/minder
        ./internal/daemon/link/src/server/mod/rpcproxy
        ./internal/daemon/portal/src/server/comp/hubinfo
        ./internal/daemon/portal/src/server/comp/hubredis
        ./internal/daemon/portal/src/server/mod/access
        ./internal/daemon/portal/src/server/mod/epmgr
        ./internal/daemon/portal/src/server/mod/site/...
        ./internal/daemon/portal/src/server/mod/vault
      )
      ;;
    link)
      patterns+=(
        ./app/linked
        ./app/standalone
        ./internal/app
        ./internal/core/link/...
        ./internal/daemon/link/...
        ./internal/daemon/portal/src/server/mod/site/...
      )
      ;;
    portal)
      patterns+=(
        ./app/standalone
        ./internal/daemon/portal/...
      )
      ;;
    all)
      patterns=(./...)
      break
      ;;
    *)
      printf 'Unsupported quick-test scope: %s\n' "${scope}" >&2
      usage
      exit 2
      ;;
  esac
done

cd "${repo_dir}"
resolved="$(GOWORK=off go list "${patterns[@]}")"
resolved="$(printf '%s\n' "${resolved}" | sort -u)"
packages=()
while IFS= read -r package; do
  if [[ -n "${package}" ]]; then
    packages+=("${package}")
  fi
done <<< "${resolved}"

printf 'Running quick tests for scopes: %s\n' "$*"
GOWORK=off go test -vet=off "${packages[@]}"
