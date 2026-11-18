# Vine Link

**English** | [简体中文](README.zh-CN.md)

Link is the application-side access layer. It registers local applications with Hub, synchronizes configuration and service-discovery state, and provides a unified Rpc forwarding entry point between local and remote applications. Link is deployed as the application's sidecar: it may share the application process or run as another process or container on the same host, and normally remains in one deployment trust boundary with the application. A non-loopback Link API is allowed for unusual deployments, but Link warns because this cross-host h2c path is unauthenticated and is not the expected topology.

## Directory Structure

```text
internal/daemon/link/
├── skel/                 Link skeleton definitions
└── src/
    └── server/           Link server runtime
        ├── app/          Assembly of Link modules and servicers
        ├── comp/         Shared components such as Hub info and Watch/NATS clients
        ├── flag/         Link flags and default normalization
        ├── impl/         Implementations of Link registration and configuration services
        └── mod/          Runtime modules
            ├── minder/   Local app ownership, mutation, registration, heartbeat, and health checks
            ├── config/   Configuration reads and Watch change subscriptions
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

Modify contracts under `skel/`, regenerate them with the script, and verify every protocol consumer.

## Runtime Model

Link has four primary responsibilities:

1. Local application registration
   After an application starts, it calls `RegistryService`. `minder` records the local instance, publishes its registration to Hub, and manages lease renewal and health checks according to the deployment mode. Unregister stops renewal and recovery, serializes the Hub operation with in-flight registration work, then drains and removes the local instance. Local cleanup completes after either a successful or failed Hub unregister.

2. Configuration and discovery
   `config.Reader` loads configuration from Hub Watch and watches for changes. `rpcproxy` maintains remote Rpc discovery state; `webproxy` derives local Web routes from `minder` registrations.

   The Hub Watch client uses the `vine.link` user. Its ACL is limited to configuration, Rpc endpoint discovery, the shared revision key, and their required subscriptions. The Redis password is empty for in-process mode and separated-deployment debugging. With backend mTLS enabled, the Link certificate authenticates the client and binds its SPIFFE identity to the `vine.link` user. Without mTLS, the username only selects an ACL role, so the Redis endpoint still requires network isolation.

3. Asynchronous events and tasks
   `event` and `task` subscribe to the listener and runner capabilities declared by local instances in `minder`, then deliver messages received through NATS.

   Embedded Hub NATS provisions the required JetStream streams; external NATS deployments
   must provision them before Link starts and own their storage policy.

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

- When adding an application capability or registration field, first extend the source facts and mutators owned by `minder`, then update derived state in downstream modules.
- When Rpc/Web endpoint, event, or task protocols change, update Hub Watch structures, application registration code, and forwarding/delivery tests together.

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

Link supports in-process operation at two boundaries:

- An inproc `LinkApp` registers its Rpc services with the `inproc` transport.
  Local application liveness follows registration and shutdown. A separately
  running Link checks applications through their console endpoints.
- `HubInprocMode` selects Hub's inproc endpoint and an in-memory Redis connection
  for `config.Reader` and `rpcproxy`. These modules initialize snapshots and
  subscribe to changes. Registrations with the inproc Hub remain valid until
  explicit unregister; connections to an external Hub renew leases through heartbeat.

`rpcproxy` and `webproxy` route requests through the selected transport, including
`rpc+inproc://`, `web+inproc://`, and `link+inproc://` endpoints.

Cover registration, subscriptions, and forwarding in inproc tests. Use separate
processes to validate heartbeat, lease expiration, console health checks,
network disconnection, and independent process shutdown.
