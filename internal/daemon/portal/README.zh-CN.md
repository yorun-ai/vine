# Vine Portal

[English](README.md) | **简体中文**

外部访问入口层，负责从 Hub Watch 同步入口、站点、证书和 access/schema 信息，并把进入 Portal 的 HTTP、Rpc、Web 请求路由到后端 runtime endpoint。

## 目录结构

```text
internal/daemon/portal/
└── src/
    └── server/             Portal 服务端运行目录
        ├── app/            应用装配层，决定 Portal 启用哪些 component 和 module
        ├── comp/           运行时共享组件，如 `hubinfo` 与 Hub Watch client
        ├── flag/           Portal 启动参数与默认值规范化
        ├── mod/            运行时模块层
        │   ├── access/     actor/service/Web/resource schema 与 Rpc/Web 认证授权
        │   ├── entry/      HTTP/HTTPS 监听入口与 portal rule 分发
        │   ├── epmgr/      Rpc/Web endpoint 订阅与轮询选择
        │   ├── site/       Portal site 管理与 RpcGW/WebGW 创建
        │   └── vault/      TLS 证书加载、监听与匹配
        └── util/           Portal 内部工具包，如 gateway 转发工具
```

## 运行机制

Portal 的职责可以拆成四条主线：

1. 入口监听
   `entry` 从 Watch 读取 `portal:rule:*` 配置，按 scheme/port 维护 HTTP/HTTPS listener，并把请求交给对应 site。路径改写保留转义后的后缀和 query string，并由 `entry` 负责。

2. 站点路由
   `site` 从 Watch 读取 `portal:site:*` 配置，按类型维护 RpcGW 和 WebGW。RpcGW/WebGW 只负责各自 site 内的请求匹配与转发。

3. Endpoint 发现
   `epmgr` 统一订阅 Rpc/Web endpoint key，内部维护 watcher 引用计数，并为 gateway 提供 `NextRpcEndpoint` 和 `NextWebEndpoint`。

4. Auth、Permission 与证书
   `access` 从 `schema:actor:*`、`schema:service:*`、`schema:web:*`、`schema:resource:*` 读取并监听准入相关 schema。RpcGW 在转发前调用 `access`，由它按需调用后端 auth service、actor permission service 和 resource check service。`vault` 从 Watch 读取并监听证书，用于 HTTPS SNI 匹配。

   Hub Watch client 使用 `vine.portal` 用户，其 ACL 仅允许读取 Portal rule、site、证书、schema、Rpc/Web endpoint、共享 revision key 以及所需的订阅。Redis 密码为空，用于进程内模式和分离部署调试。启用后端 mTLS 时，Portal 证书会认证客户端，并把其 SPIFFE 身份绑定到 `vine.portal` 用户。未启用 mTLS 时，用户名只能选择 ACL 角色；由于该角色可以读取 TLS 私钥，Redis endpoint 必须限制在受信网络中。

## 依赖关系

Portal 内部主要依赖关系是：

- `entry -> site, vault`
- `site -> access, epmgr`
- `access -> epmgr`
- `vault -> hubwatch`
- `epmgr -> hubwatch`

外部调用方只应装配 `src/server/app` 和 `src/server/flag`，不要直接依赖内部 module。

修改 Portal 时必须保持以下 owner 和依赖边界：

- `epmgr` 是 Rpc/Web endpoint 订阅、watcher 引用计数和轮询选择的唯一 owner；site 和 access 通过它获取 endpoint。
- `access` 是 actor/service/Web/resource schema 与 Rpc/Web 准入状态的 owner；gateway 不应维护第二份认证或权限 schema 缓存。
- `entry` 只管理监听入口和 rule 分发，`site` 只管理 RpcGW/WebGW，`vault` 只管理证书加载与匹配。
- RpcGW/WebGW 必须继续传播当前 trace、initiator、actor、deadline 和剩余 timeout，不能用新的后台 context 覆盖请求上下文。
- 修改 header、转发路径、schema 或 endpoint 格式时，必须同步 Link、Hub Watch 结构、调用方以及 gateway 测试。

## Inproc 模式

Portal 支持作为 standalone 运行时的一部分在进程内启动。此时 Portal 仍然从 Hub Watch 读取并监听 portal rule、portal site、证书、schema 和 endpoint 注册信息，只是底层 Hub Watch client 会走进程内连接，而不是外部 TCP Redis。

Inproc/standalone 模式下需要注意：

- `entry`、`site`、`access`、`epmgr`、`vault` 的模块边界和数据订阅语义保持不变。
- RpcGW/WebGW 仍然通过 endpoint 发现结果转发请求，但目标 endpoint 可能是 `link+inproc://`，不会经过外部 Link ingress TCP 端口。
- Portal 本身不负责 heartbeat 或 TTL 续租；注册信息在 inproc Hub 中通常长期有效，依赖应用显式 unregister 和 Hub Watch 事件驱动清理。
- 这种模式适合验证路由、准入、schema 监听和 gateway 转发逻辑，但不模拟外部网络断连、独立 Link/Portal 进程崩溃、TLS listener 端口不可达等分布式故障。

如果要验证真实网络入口、TLS 监听、跨进程 endpoint 可达性或注册租约过期，应使用普通进程模式启动 Portal。

修改入口、发现、认证或转发逻辑时，应分别验证 standalone/inproc 和普通网络部署。inproc 模式只保证路由与订阅语义，不代表真实监听、TLS、断连和租约行为已经覆盖。

RPC service/method 支持 `required`、`optional`、`guest`，Web 另支持 `off`。
本地注册和 Hub 接收入口均拒绝 RPC 显式使用 `off`。旧值 `auth` 转为 `required`，
RPC 的 `noauth` 转为 `optional`，Web 的 `noauth` 转为 `off`，Portal 只消费新值。RPC method 先继承 service，
两者都省略或为 `unset` 时默认为 `required`。Web 未声明模式时保留原有依 actor 配置
决定认证行为的默认值，显式模式覆盖默认值。启用 Portal 认证时，无效凭证和认证服务
错误均拒绝，不退回匿名。Web `off` 使用匿名 actor 并保留 Authorization，供 handler 自行认证。
RPC 和 Web 的 `required`、`optional`、`guest` 在准入通过后删除 Authorization，
仅向下传递已准入的 actor。旧 Web 请求通过 Vine 认证后也删除 Authorization。

Hub 在 `schema:web:*` 发布选中的 Web schema，Portal 订阅更新及删除事件；
具名 Web 请求缺少 schema 时拒绝。Hub 与 Portal 必须配套升级：旧 Portal 不识别
升级后 Hub 发布的 `optional`、`off` 值。旧应用生成的契约无需重新生成；旧 RPC
`noauth` 现在会认证提供的凭证，无效凭证会报错，不再隐式匿名放行。
