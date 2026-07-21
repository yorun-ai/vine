#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd -- "${script_dir}/.." && pwd -P)"
scope="${VINE_RACE_SCOPE:-targeted}"

case "${scope}" in
  targeted)
    packages=(
      ./app/testkit
      ./internal/app
      ./internal/core/di
      ./internal/core/event
      ./internal/core/link/ingressinproc
      ./internal/core/logger
      ./internal/core/rpc/client
      ./internal/core/rpc/log
      ./internal/core/rpc/server
      ./internal/core/rpc/transport/http
      ./internal/core/rpc/transport/inproc
      ./internal/core/task
      ./internal/core/web/assets
      ./internal/core/web/inproc
      ./internal/core/web/proxy
      ./internal/daemon/hub/api/nats
      ./internal/daemon/hub/api/redis
      ./internal/daemon/hub/src/server/app
      ./internal/daemon/hub/src/server/comp/natsserver
      ./internal/daemon/hub/src/server/comp/redisserver
      ./internal/daemon/hub/src/server/comp/redisserver/embedded
      ./internal/daemon/hub/src/server/core
      ./internal/daemon/hub/src/server/mod/scheduler
      ./internal/daemon/hub/src/server/repo/...
      ./internal/daemon/link/src/server/comp/hubredis
      ./internal/daemon/link/src/server/comp/nats
      ./internal/daemon/link/src/server/mod/config
      ./internal/daemon/link/src/server/mod/event
      ./internal/daemon/link/src/server/mod/ingress
      ./internal/daemon/link/src/server/mod/minder
      ./internal/daemon/link/src/server/mod/rpcproxy
      ./internal/daemon/link/src/server/mod/task
      ./internal/daemon/link/src/server/mod/webproxy
      ./internal/daemon/portal/src/server/cacheutil
      ./internal/daemon/portal/src/server/comp/hubredis
      ./internal/daemon/portal/src/server/mod/access
      ./internal/daemon/portal/src/server/mod/entry
      ./internal/daemon/portal/src/server/mod/epmgr
      ./internal/daemon/portal/src/server/mod/site/...
      ./internal/daemon/portal/src/server/mod/vault
      ./internal/infra/rdb
      ./internal/infra/redis
      ./internal/util/goutil
      ./internal/util/httputil
      ./util/vmap
      ./util/vnet
      ./util/vslice
    )
    ;;
  all)
    packages=(./...)
    ;;
  *)
    printf 'Unsupported VINE_RACE_SCOPE: %s (expected targeted or all)\n' "${scope}" >&2
    exit 2
    ;;
esac

cd "${repo_dir}"
GORACE="${GORACE:-atexit_sleep_ms=0}" GOWORK=off go test -race "${packages[@]}"
