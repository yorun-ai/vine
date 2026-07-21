# Vine Portal

**English** | [简体中文](README.zh-CN.md)

Portal is Vine's external access layer. It synchronizes entries, sites, certificates, and access/schema state from Hub Redis, then routes incoming HTTP, Rpc, and Web requests to runtime endpoints.

## Directory Structure

```text
internal/daemon/portal/
└── src/
    └── server/             Portal server runtime
        ├── app/            Assembly of Portal components and modules
        ├── comp/           Shared components such as Hub info and Redis clients
        ├── flag/           Portal flags and default normalization
        ├── mod/            Runtime modules
        │   ├── access/     Actor/service/resource schemas and Rpc/Web authorization
        │   ├── entry/      HTTP/HTTPS listeners and portal-rule dispatch
        │   ├── epmgr/      Rpc/Web endpoint subscriptions and round-robin selection
        │   ├── site/       Portal site management and RpcGW/WebGW creation
        │   └── vault/      TLS certificate loading, watching, and matching
        └── util/           Portal utilities such as gateway forwarding
```

## Runtime Model

Portal has four primary responsibilities:

1. Entry listeners
   `entry` reads `portal:rule:*` configuration from Redis, maintains HTTP/HTTPS listeners by scheme and port, and dispatches requests to the corresponding site.

2. Site routing
   `site` reads `portal:site:*` configuration from Redis and maintains RpcGW and WebGW instances by site type. Each gateway is responsible only for matching and forwarding requests within its site.

3. Endpoint discovery
   `epmgr` subscribes to Rpc/Web endpoint keys, manages watcher reference counts, and provides `NextRpcEndpoint` and `NextWebEndpoint` to gateways.

4. Authentication, permission, and certificates
   `access` reads and watches `schema:actor:*`, `schema:service:*`, and `schema:resource:*` state. Before forwarding, RpcGW asks `access` to perform authentication and permission admission, which may invoke backend auth services, actor permission services, and resource check services. `vault` reads and watches certificates from Redis for HTTPS SNI matching.

## Dependencies

The primary Portal module dependencies are:

- `entry -> site, vault`
- `site -> access, epmgr`
- `access -> epmgr`
- `vault -> hubredis`
- `epmgr -> hubredis`

External callers should assemble only `src/server/app` and `src/server/flag`; they must not depend directly on internal modules.

Preserve these ownership and dependency boundaries when modifying Portal:

- `epmgr` is the single owner of Rpc/Web endpoint subscriptions, watcher reference counts, and round-robin selection. Sites and access logic obtain endpoints through it.
- `access` owns actor/service/resource schemas and Rpc/Web admission state. Gateways must not maintain a second authentication or permission schema cache.
- `entry` owns listener entries and rule dispatch only; `site` owns RpcGW/WebGW instances only; `vault` owns certificate loading and matching only.
- RpcGW and WebGW must continue propagating the active trace, initiator, actor, deadline, and remaining timeout. Do not replace the request context with a new background context.
- When headers, forwarding paths, schemas, or endpoint formats change, update Link, Hub Redis structures, callers, and gateway tests together.

## Inproc Mode

Portal can run in process as part of a standalone runtime. It still reads and watches portal rules, portal sites, certificates, schemas, and endpoint registrations from Hub Redis, but its Hub Redis client uses an in-process connection instead of external TCP.

In inproc and standalone modes:

- The module boundaries and subscription semantics of `entry`, `site`, `access`, `epmgr`, and `vault` remain unchanged.
- RpcGW and WebGW still forward through discovered endpoints, but targets may use `link+inproc://` and avoid an external Link ingress TCP port.
- Portal does not own heartbeat or TTL renewal. Registrations in an inproc Hub are normally long-lived and depend on explicit application unregister plus Hub Redis events for cleanup.
- This mode is suitable for testing routing, admission, schema watching, and gateway forwarding, but it does not model external network disconnection, independent Link/Portal process crashes, unreachable TLS listeners, or distributed lease expiry.

Use normal process mode to validate real network entries, TLS listeners, cross-process endpoint reachability, or registration lease expiry.

When changing entry, discovery, authentication, or forwarding behavior, validate standalone/inproc and normal network deployments separately. Inproc mode establishes routing and subscription semantics only; it does not establish correct real-listener, TLS, disconnection, or lease behavior.
