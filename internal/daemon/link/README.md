# Vine Link

**English** | [简体中文](README.zh-CN.md)

Link is the application-side access layer. It registers local applications with Hub, synchronizes configuration and service-discovery state, and provides a unified Rpc forwarding entry point between local and remote applications.

## Directory Structure

```text
internal/daemon/link/
├── skel/                 Link skeleton definitions
└── src/
    └── server/           Link server runtime
        ├── app/          Assembly of Link modules and servicers
        ├── comp/         Shared components such as Hub info and Redis/NATS clients
        ├── flag/         Link flags and default normalization
        ├── impl/         Implementations of Link registration and configuration services
        └── mod/          Runtime modules
            ├── minder/   Local app ownership, mutation, registration, heartbeat, and health checks
            ├── config/   Configuration reads and Redis change subscriptions
            ├── event/    Event consumer management and delivery
            ├── ingress/  External HTTP entry point dispatching requests to proxies
            ├── rpcproxy/ Local and remote Rpc forwarding
            ├── task/     Task consumer management and delivery
            └── webproxy/ Local and remote Web forwarding
```

Generated Link code lives in `internal/core/link/skeled`, outside the runtime packages, so the Core layer can own the shared protocol types. Generate it with:

```bash
bash script/gen-skel.sh link
```

Do not edit `internal/core/link/skeled` directly. Modify contracts under `skel/`, regenerate them with the script, and verify every protocol consumer.

## Runtime Model

Link has four primary responsibilities:

1. Local application registration
   After an application starts, it calls `RegistryService`. `minder` records the local instance as source state, publishes application registrations to Hub, and starts heartbeat and health checks.

2. Configuration and discovery
   `config.Reader` loads configuration from Hub Redis and continuously watches for changes. `rpcproxy` and `webproxy` maintain Rpc and Web discovery state respectively.

3. Asynchronous events and tasks
   `event` and `task` subscribe to the listener and runner capabilities declared by local instances in `minder`, then deliver messages received through NATS.

4. Rpc forwarding
   Local application Rpc calls first enter `rpcproxy out`. Link selects a local or remote target from service-discovery state. Requests arriving through the external HTTP entry point pass through `ingress` and then `rpcproxy in` before reaching a local application.

## Key Design

`minder` is the single owner of local application instances:

- Each local application instance has one set of source facts: `appInfo`, endpoint, Rpc services, Web names, event listeners, and task runners.
- `minder` owns the source state, mutator broadcasts, and lifecycle actions.
- `rpcproxy`, `webproxy`, `event`, `task`, and `config` observe instance changes through mutators and retain only their own derived runtime state.

Module responsibilities are:

- `minder`: stores local declarations; provides `Register`, `Unregister`, and `Instance`; broadcasts mutations; and owns register/unregister, Hub registration, heartbeat, and health checks.
- `config`: owns configuration references, snapshots, and subscriptions only.
- `rpcproxy`: owns Rpc routing, the local service index, and the remote discovery cache only.
- `webproxy`: owns Web routing and the local Web index only.
- `event`: owns event-listener runtime state and delivery only.
- `task`: owns task-runner runtime state and delivery only.
- `ingress`: dispatches external HTTP requests to `rpcproxy` or `webproxy` only.

Preserve these boundaries when modifying modules:

- Do not create a second source of local application instance state in `rpcproxy`, `webproxy`, `event`, `task`, or `config`.
- When adding an application capability or registration field, first extend the source facts and mutators owned by `minder`, then update derived state in downstream modules.
- Modules must not bypass `minder` to own register, unregister, heartbeat, or health-check lifecycles.
- When Rpc/Web endpoint, event, or task protocols change, update Hub Redis structures, application registration code, and forwarding/delivery tests together.

## Dependencies

Module dependencies are:

- `rpcproxy -> minder`
- `webproxy -> minder`
- `event -> minder`
- `task -> minder`
- `config -> minder`
- `ingress -> rpcproxy, webproxy`

External services map to modules as follows:

- `RegistryService -> minder`
- `ConfigService -> config`
- `EventService -> event`
- `TaskService -> task`

## Inproc Mode

Link can run as a component in a single-process runtime:

- `LinkApp` can start inproc, registering its Rpc services with the `inproc` transport instead of exposing them over HTTP.
- When `HubInprocMode` is enabled, Link treats Hub as an in-process component:
  - `HubEndpoint` is fixed to the Hub inproc endpoint.
  - `config.Reader`, `rpcproxy`, and `webproxy` access Hub through an in-process Redis connection instead of an external TCP port.
  - `minder` does not start heartbeat.
  - Local application health checks in `minder` become no-ops.

Link retains configuration and service subscriptions in this mode; only the underlying Redis transport changes from network to in-process access.

### Failure-Semantics Differences

Inproc and standalone modes are intended for single-process execution, integration tests, and local debugging. They retain registration, discovery, forwarding, and subscription semantics, but they do not fully model distributed process liveness:

- Hub registration state does not expire through TTL and instead depends on explicit unregister during application shutdown.
- Link does not start heartbeat, so renewal failures cannot reveal Hub or network failures.
- Link does not run local application console health checks, so it cannot model a live process with an unhealthy HTTP/Rpc handler.
- Rpc/Web forwarding still passes through `rpcproxy` and `webproxy`, but endpoints may use `rpc+inproc://`, `web+inproc://`, or `link+inproc://`.

Use separate Hub, Link, and Portal processes to validate heartbeat, TTL expiry, network disconnection, or independent process crashes.

When changing Link connection, registration, discovery, or forwarding behavior, cover both normal network and inproc modes. Passing inproc tests does not establish correct heartbeat, TTL, health-check, or cross-process endpoint behavior.
