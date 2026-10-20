# Vine Portal

**English** | [简体中文](README.zh-CN.md)

Portal is Vine's external access layer. It synchronizes entries, sites, certificates, and access/descriptor state from Hub Watch, then routes incoming HTTP, Rpc, and Web requests to runtime endpoints.

## Directory Structure

```text
internal/daemon/portal/
└── src/
    └── server/             Portal server runtime
        ├── app/            Assembly of Portal components and modules
        ├── comp/           Shared components such as Hub info and Watch clients
        ├── flag/           Portal flags and default normalization
        ├── mod/            Runtime modules
        │   ├── access/     Rpc/Web authorization and auth/permission endpoint watches
        │   ├── desc/       Descriptor subscriptions, caches, and API type dependencies
        │   ├── entry/      HTTP/HTTPS listeners and portal-rule dispatch
        │   ├── epmgr/      Rpc/Web endpoint subscriptions and round-robin selection
        │   ├── site/       Portal site management and RpcGW/WebGW creation
        │   └── vault/      TLS certificate loading, watching, and matching
        └── util/           Portal utilities such as gateway forwarding
```

## Runtime Model

Portal has four primary responsibilities:

1. Entry listeners
   `entry` reads `portal:entry:*` and `portal:rule:*` configuration from Watch, maintains HTTP/HTTPS listeners by scheme, listen IP, and port, and dispatches requests to the corresponding site. An empty entry Host matches only HTTP IP access. Hub publishes `effectiveHTTP`; Portal uses its transport switches, redirect switch, and ports directly, without deriving them from Host or certificate presence. Certificate validity affects per-request redirect checks and TLS certificate selection, not listener enablement. Every TLS handshake, including session resumption, matches SNI against entries on that listener before asking `vault` for a certificate; absent, IP, and unmatched SNI are rejected. HTTPS requests must keep Host consistent with SNI (otherwise 421). Both HTTP and HTTPS select an entry before matching its rules, preferring exact hosts over wildcards and exact IPs over the empty-host HTTP IP entry; an unmatched path never falls back to another entry or inherits its automatic redirect. Entries independently create their effective transports, including rule-free HTTP 308 redirects when `autohttpsEnabled` is enabled and the request host has a matching, currently valid configured certificate. Otherwise, requests follow normal HTTP rules. Rules reference entries by `entryName`; transport and listener settings exist only on entries. Path rewriting preserves the escaped suffix and query string and is owned by `entry`, not the target gateway.

2. Site routing
   `site` reads `portal:site:*` configuration from Watch and maintains RpcGW and WebGW instances by site type. Each gateway is responsible only for matching and forwarding requests within its site.

3. Endpoint discovery
   `epmgr` subscribes to Rpc/Web endpoint keys, manages watcher reference counts, and provides `NextRpcEndpoint` and `NextWebEndpoint` to gateways.

4. Authentication, permission, and certificates
   `desc` reads and watches `descriptor:actor:*`, `descriptor:service:*`, `descriptor:web:*`, and `descriptor:resource:*` state. `access` queries this cache and observes actor/resource changes to maintain auth and permission endpoint watches. Before forwarding, RpcGW asks `access` to perform authentication and permission admission, which may invoke backend auth services, actor permission services, and resource check services. `vault` reads and watches certificates from Watch for HTTPS SNI matching.

   At startup, `desc` reads a complete `descriptor:*` snapshot and selects API services' transitive Data and Enum dependencies, including generic arguments and collection types. A single background worker coalesces service/type changes, loads dependencies outside the cache lock, and publishes paired service/type snapshots only if its generation is still current. Admission uses the latest service definitions immediately; a slow or failed type refresh does not block requests or delay service revocation. Failed refreshes retain the previous type snapshot and subscription and retry with bounded backoff. Shutdown cancels pending work and waits for the worker. Actor/resource observers also run outside the cache lock, with ordered delivery.

   Selected type keys share one PubSub connection. Attachment and reconnect wait for all subscription acknowledgements before one reconciliation, using bounded GET pipelines. A replacement subscription is ready before its predecessor is released. Missing definitions remain subscribed until published. `desc.Manager.Snapshot()` returns the last paired service/type snapshot, its missing keys, and a `Syncing` flag when newer inputs are pending. RpcGW inspect uses this snapshot directly and returns 503 during synchronization or when a requested schema cannot be resolved.

   The Hub Watch client uses the `vine.portal` user. Its ACL is limited to Portal entries, rules, sites, certificates, descriptors, Rpc/Web endpoint discovery, the shared revision key, and their required subscriptions. The Redis password is empty for in-process mode and separated-deployment debugging. With backend mTLS enabled, the Portal certificate authenticates the client and binds its SPIFFE identity to the `vine.portal` user. Without mTLS, the username only selects an ACL role; because that role can read TLS private keys, the Redis endpoint must remain restricted to a trusted network.

## Dependencies

The primary Portal module dependencies are:

- `entry -> site, vault`
- `site -> access, desc, epmgr`
- `access -> desc, epmgr`
- `desc -> hubwatch`
- `vault -> hubwatch`
- `epmgr -> hubwatch`

External callers should assemble only `src/server/app` and `src/server/flag`; they must not depend directly on internal modules.

Preserve these ownership and dependency boundaries when modifying Portal:

- `epmgr` is the single owner of Rpc/Web endpoint subscriptions, watcher reference counts, and round-robin selection. Sites and access logic obtain endpoints through it.
- `desc` owns all descriptor subscriptions and caches, including API type dependencies. `access` owns Rpc/Web admission and its auth/permission endpoint watches. Gateways and access must not maintain a second descriptor cache.
- `entry` owns listener entries and rule dispatch only; `site` owns RpcGW/WebGW instances only; `vault` owns certificate loading and matching only.
- RpcGW and WebGW must continue propagating the active trace, initiator, actor, deadline, and remaining timeout. Do not replace the request context with a new background context.
- When headers, forwarding paths, descriptors, or endpoint formats change, update Link, Hub Watch structures, callers, and gateway tests together.

## Inproc Mode

Portal can run in process as part of a standalone runtime. It still reads and watches portal rules, portal sites, certificates, descriptors, and endpoint registrations from Hub Watch, but its Hub Watch client uses an in-process connection instead of external TCP.

In inproc and standalone modes:

- The module boundaries and subscription semantics of `entry`, `site`, `desc`, `access`, `epmgr`, and `vault` remain unchanged.
- RpcGW and WebGW still forward through discovered endpoints, but targets may use `link+inproc://` and avoid an external Link ingress TCP port.
- Portal does not own heartbeat or TTL renewal. Registrations in an inproc Hub are normally long-lived and depend on explicit application unregister plus Hub Watch events for cleanup.
- This mode is suitable for testing routing, admission, descriptor watching, and gateway forwarding, but it does not model external network disconnection, independent Link/Portal process crashes, unreachable TLS listeners, or distributed lease expiry.

Use normal process mode to validate real network entries, TLS listeners, cross-process endpoint reachability, or registration lease expiry.

Rpc service/method authentication modes are `required`, `optional`, and
`anonymous`; method declarations also support `inherit`. Web additionally supports
`off`, while services and Web cannot use `inherit`. Local registration and Hub
reject explicit Rpc `off`.

Generated descriptors carry each method's `EffectiveAuthMode` and
`EffectiveRequire`. Local registration and Hub validate these values against the
declared policies. Portal applies them directly without inheriting authentication
modes or merging permission requirements. Web uses its validated `AuthMode`.

Legacy schema compatibility is confined to registration. The adapter converts
`auth` to `required`, and `noauth` to `optional` for Rpc or `off` for Web. Empty and
`unset` service/Web modes become `required`; empty and `unset` method modes become
`inherit`. The adapter computes effective policies before validating the converted
descriptor. New descriptors must already contain valid effective policies.

Invalid supplied credentials and auth-service errors never fall back to anonymous
access when Portal authentication is enabled. Web `off` uses an anonymous actor
and preserves Authorization for native authentication. Rpc and Web `required`,
`optional`, and `anonymous` remove Authorization after admission, forwarding only
the admitted actor.

Hub publishes selected Web descriptors under `descriptor:web:*`; Portal subscribes
to updates and deletes and rejects named Web requests when the descriptor is
unavailable. Deploy matching Hub, Link, and Portal versions together because the
registration payloads and Watch keys use descriptors. Previously generated Web
contracts now require authentication by default. To allow anonymous access or
native authentication, explicitly select `optional` or `off` and regenerate the
contracts. Old Rpc `noauth` authenticates supplied credentials and rejects invalid
credentials instead of silently admitting the request anonymously.

A WEBGW site requires `actorSkelName`, and the configured actor must exist. Vine ignores legacy `via` metadata. RPCGW admits calls by the service's actor audiences, then applies authentication and permission rules. Web authentication follows the declared mode; `required` rejects requests when the actor has no authentication configured. Only `off` preserves native Authorization credentials for the handler.

When changing entry, discovery, authentication, or forwarding behavior, validate standalone/inproc and normal network deployments separately. Inproc mode establishes routing and subscription semantics only; it does not establish correct real-listener, TLS, disconnection, or lease behavior.

## RpcGW Inspect

RPCGW sites expose read-only `GET /inspect/_list`,
`GET /inspect/{service}/_list`, and `GET /inspect/{service}/{method}` when the
site's `inspectEnabled` switch is true. `/inspect` and `/inspect/` alias the
service list. The switch defaults to false and updates live through Watch.
Inspection requires no credentials; invocation admission remains unchanged.

Responses use `application/json`: service lists return `{services: [...]}`,
method lists return `{service: {...}, methods: [...]}` with the same service
summary as the catalog, and method details are objects.
Failures use HTTP status codes with a `{code, message}` body. Lists contain
stable, wire-name-sorted service or method summaries. Method details carry effective auth/permission
metadata and independent JSON Schema 2020-12 `inputSchema` and `outputSchema`
documents for params and result values. The converter in
`internal/core/skel/jsonschema` resolves generic Data and Enum references with
local `$defs`, including recursive contracts; it retains no descriptor cache.
Only configured API services accepting the site's current actor are visible.
No live backend endpoint is required. Responses are not cacheable, have no
revision, and share site CORS with GET-specific preflight methods.

## RpcGW MCP

The independent `mcpEnabled` site switch (default false) exposes `POST /mcp`.
An entry mapping `/api` to the site root therefore exposes `/api/mcp`.
Only MCP `2026-07-28` is supported. The gateway requires that protocol version
header and rejects legacy initialization; the official MCP Go SDK owns Streamable
HTTP framing, request metadata validation, discovery, and JSON-RPC errors.
The gateway uses stateless JSON responses and advertises only tools, without
list-change notifications.

Inspect and MCP share the site's Actor/API catalog selection. MCP builds tool
schemas from the current `desc` snapshot without another descriptor cache.
`tools/list` is sorted and paginated; `tools/call` resolves only its selected
method. Both reject syncing snapshots or unresolved schemas. Tool names use
`{serviceSkelName}.{methodSkelName}`, with a deterministic SHA-256 name for wire
names outside MCP's character/length recommendations. Input and output are
validated against the same JSON Schema contracts as Inspect. `structuredContent`
contains the API result directly, including scalars, arrays, and `null`, with
the same JSON in text `content`.

Calls reuse Rpc admission and forwarding. Only Authorization is copied from the
MCP request; Portal supplies backend client, trace and initiator metadata, and
admission supplies the actor. Credentials are stripped before business-service
forwarding. MCP requests share the legacy `/invoke` disconnect policy: accepted business
execution ignores HTTP and SDK cancellation, while the server-controlled
timeout and Portal shutdown still cancel it. `params._meta["vmcp-options"]` uses
the same options string as `vrpc-options` (for example, `timeout=60s`): the default
is 30 seconds and the maximum is 120 seconds. Invalid options return JSON-RPC
Invalid Params before admission; unknown options are ignored. API failures are tool
results with `isError`; catalog/protocol failures are JSON-RPC errors.
Origin validation applies before handling MCP requests or preflight. Responses
are non-cacheable. This endpoint uses Vine credentials, not an OAuth discovery
or authorization-server implementation.
