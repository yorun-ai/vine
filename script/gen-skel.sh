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
  hub   generate internal/daemon/hub/api/skeled and dashboard TS skeled
  link  generate internal/core/link/skeled
EOF
}

rewrite_data_imports() {
  local target_dir="$1"

  if [[ ! -f "${target_dir}/data.go" ]]; then
    return
  fi

  perl -0pi -e '
    s#"go\.yorun\.ai/vine/core/skel"#"go.yorun.ai/vine/internal/core/skel"#g;
    s#"go\.yorun\.ai/vine/core/rpc"#rpc "go.yorun.ai/vine/internal/core/rpc/spec"#g;
  ' "${target_dir}/data.go"
}

rewrite_schema_imports() {
  local target_dir="$1"

  if [[ ! -f "${target_dir}/schema.go" ]]; then
    return
  fi

  perl -0pi -e '
    s#"go\.yorun\.ai/vine/core/skel"#"go.yorun.ai/vine/internal/core/skel"#g;
  ' "${target_dir}/schema.go"
}

rewrite_actor_imports() {
  local target_dir="$1"

  if [[ ! -f "${target_dir}/actor.go" ]]; then
    return
  fi

  perl -0pi -e '
    s#"go\.yorun\.ai/vine/core/skel"#"go.yorun.ai/vine/internal/core/skel"#g;
  ' "${target_dir}/actor.go"
}

rewrite_web_imports() {
  local target_dir="$1"

  if [[ ! -f "${target_dir}/web.go" ]]; then
    return
  fi

  perl -0pi -e '
    s#"go\.yorun\.ai/vine/core/web"#web "go.yorun.ai/vine/internal/core/web/spec"#g;
  ' "${target_dir}/web.go"
}

rewrite_service_imports() {
  local target_dir="$1"

  if [[ ! -f "${target_dir}/service.go" ]]; then
    return
  fi

  perl -0pi -e '
    s#"go\.yorun\.ai/vine/core/ex"#"go.yorun.ai/vine/internal/core/ex"#g;
    s#"go\.yorun\.ai/vine/core/skel"#"go.yorun.ai/vine/internal/core/skel"#g;
    s#"go\.yorun\.ai/vine/core/rpc"#rpcclient "go.yorun.ai/vine/internal/core/rpc/client"\n\trpcspec "go.yorun.ai/vine/internal/core/rpc/spec"#g;
    s/\brpc\.Register\(/rpcspec.Register(/g;
    s/\brpc\.ServiceSpec\b/rpcspec.ServiceSpec/g;
    s/\brpc\.ServiceSpecType/rpcspec.ServiceSpecType/g;
    s/\brpc\.MethodSpec\b/rpcspec.MethodSpec/g;
    s/\brpc\.InvokeOption\b/rpcclient.InvokeOption/g;
    s/\brpc\.(CheckValueNotNil|JoinPath|JoinIndex|JoinMapKey)\b/rpcspec.$1/g;
    s/\*rpc\.Client\b/*rpcclient.Client/g;
  ' "${target_dir}/service.go"
}

rewrite_resource_imports() {
  local target_dir="$1"

  if [[ ! -f "${target_dir}/resource.go" ]]; then
    return
  fi

  perl -0pi -e '
    s#"go\.yorun\.ai/vine/core/ex"#"go.yorun.ai/vine/internal/core/ex"#g;
    s#"go\.yorun\.ai/vine/core/rpc"#rpc "go.yorun.ai/vine/internal/core/rpc/spec"#g;
    s/\brpc\.Register\(/rpc.Register(/g;
    s/\brpc\.ServiceSpec\b/rpc.ServiceSpec/g;
    s/\brpc\.ServiceSpecType/rpc.ServiceSpecType/g;
    s/\brpc\.MethodSpec\b/rpc.MethodSpec/g;
    s/\brpc\.(CheckValueNotNil|JoinPath|JoinIndex|JoinMapKey)\b/rpc.$1/g;
  ' "${target_dir}/resource.go"
}

rewrite_event_imports() {
  local target_dir="$1"

  if [[ ! -f "${target_dir}/event.go" ]]; then
    return
  fi

  perl -0pi -e '
    s#"go\.yorun\.ai/vine/core/ex"#"go.yorun.ai/vine/internal/core/ex"#g;
    s#"go\.yorun\.ai/vine/core/skel"#"go.yorun.ai/vine/internal/core/skel"#g;
    s#"go\.yorun\.ai/vine/core/event"#event "go.yorun.ai/vine/internal/core/event"\n\teventspec "go.yorun.ai/vine/internal/core/event/spec"#g;
    s/\bevent\.Register\(/eventspec.Register(/g;
    s/\bevent\.EventSpec\b/eventspec.EventSpec/g;
    s/\bevent\.EventSpecType\b/eventspec.EventSpecType/g;
  ' "${target_dir}/event.go"
}

rewrite_ts_service_comments() {
  local target_dir="$1"

  if [[ ! -f "${target_dir}/service.ts" ]]; then
    return
  fi

  perl -0pi -e '
    s/(\* \@param params - )[^\r\n]*/$1Request parameters, or null for methods without input/g;
    s/(\* \@param options - )[^\r\n]*/$1Optional invocation options/g;
  ' "${target_dir}/service.ts"
}

rewrite_common_go_imports() {
  local target_dir="$1"

  rewrite_data_imports "${target_dir}"
  rewrite_schema_imports "${target_dir}"
  rewrite_service_imports "${target_dir}"
  rewrite_resource_imports "${target_dir}"
  gofmt -w "${target_dir}"/*.go
}

generate_app_skel() {
  local skel_dir="${repo_dir}/internal/core/app/skel"
  local target_dir="${repo_dir}/internal/core/app/skeled"

  skelc gen go --skel-in "${skel_dir}" --go-out "${target_dir}"
  rewrite_common_go_imports "${target_dir}"
}

generate_hub_skel() {
  local skel_dir="${repo_dir}/internal/daemon/hub/skel"
  local api_dir="${repo_dir}/internal/daemon/hub/api/skeled"
  local frontend_dir="${repo_dir}/internal/daemon/hub/src/dashboard/src/skeled"

  skelc gen go --skel-in "${skel_dir}" --go-out "${api_dir}"
  skelc gen ts --skel-in "${skel_dir}" --ts-out "${frontend_dir}"
  rewrite_ts_service_comments "${frontend_dir}"

  rewrite_common_go_imports "${api_dir}"
  rewrite_actor_imports "${api_dir}"
  rewrite_web_imports "${api_dir}"
  rewrite_event_imports "${api_dir}"
  gofmt -w "${api_dir}"/*.go
}

generate_link_skel() {
  local skel_dir="${repo_dir}/internal/daemon/link/skel"
  local target_dir="${repo_dir}/internal/core/link/skeled"

  skelc gen go --skel-in "${skel_dir}" --go-out "${target_dir}"
  rewrite_common_go_imports "${target_dir}"
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
