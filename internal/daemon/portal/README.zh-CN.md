# Vine Portal

[English](README.md) | **简体中文**

外部访问入口层，负责从 Hub Watch 同步入口、站点、证书和 access/descriptor 信息，并把进入 Portal 的 HTTP、Rpc、Web 请求路由到后端 runtime endpoint。

## 目录结构

```text
internal/daemon/portal/
└── src/
    └── server/             Portal 服务端运行目录
        ├── app/            应用装配层，决定 Portal 启用哪些 component 和 module
        ├── comp/           运行时共享组件，如 `hubinfo` 与 Hub Watch client
        ├── flag/           Portal 启动参数与默认值规范化
        ├── mod/            运行时模块层
        │   ├── access/     Rpc/Web 认证授权与 auth/permission endpoint 订阅
        │   ├── desc/       descriptor 订阅、缓存与 API 类型依赖
        │   ├── entry/      HTTP/HTTPS 监听入口与 portal rule 分发
        │   ├── epmgr/      Rpc/Web endpoint 订阅与轮询选择
        │   ├── site/       Portal site 管理与 RpcGW/WebGW 创建
        │   └── vault/      TLS 证书加载、监听与匹配
        └── util/           Portal 内部工具包，如 gateway 转发工具
```

## 运行机制

Portal 的职责可以拆成四条主线：

1. 入口监听
   `entry` 从 Watch 读取 `portal:entry:*` 和 `portal:rule:*` 配置，按 scheme/listen IP/port 维护 HTTP/HTTPS listener，并把请求交给对应 site。入口独立建立启用的传输协议监听，`autohttpsEnabled` 的 HTTP 308 跳转不依赖规则，但要求已配置的证书匹配请求 host 且处于有效期内，否则继续走普通 HTTP 规则。HTTP 和 HTTPS 都先选入口再匹配入口内的规则，精确 Host 优先于通配符，精确 IP 优先于空 Host 的 HTTP IP 入口；路径未匹配时不会回退到其他入口或继承其自动跳转。规则通过 `entryName` 引用入口，传输协议和监听配置只由入口持有。路径改写保留转义后的后缀和 query string，并由 `entry` 负责。

2. 站点路由
   `site` 从 Watch 读取 `portal:site:*` 配置，按类型维护 RpcGW 和 WebGW。RpcGW/WebGW 只负责各自 site 内的请求匹配与转发。

3. Endpoint 发现
   `epmgr` 统一订阅 Rpc/Web endpoint key，内部维护 watcher 引用计数，并为 gateway 提供 `NextRpcEndpoint` 和 `NextWebEndpoint`。

4. Auth、Permission 与证书
   `desc` 从 `descriptor:actor:*`、`descriptor:service:*`、`descriptor:web:*`、`descriptor:resource:*` 读取并监听准入相关 descriptor。`access` 查询这份缓存，并通过 Actor/Resource 变更通知维护认证、权限检查所需的 endpoint 订阅。RpcGW 在转发前调用 `access`，由它按需调用后端 auth service、actor permission service 和 resource check service。`vault` 从 Watch 读取并监听证书，用于 HTTPS SNI 匹配。

   启动时，`desc` 读取完整的 `descriptor:*` 快照，筛选 API Service 递归依赖的 Data、Enum，包括泛型参数和容器类型。一个后台 worker 合并 Service/类型变更，在缓存锁外加载依赖，仅在版本仍然匹配时发布配套的 Service/类型快照。访问控制立即使用最新 Service 定义，类型同步缓慢或失败不会阻塞请求，也不会推迟接口撤销。同步失败保留上一份类型快照和订阅，通过有上限的退避重试；停止时取消未完成的工作并等待 worker 退出。Actor/Resource 的通知也在缓存锁外按顺序执行。

   所选类型 key 共用一条 PubSub 连接，建立订阅和重连时收齐所有订阅确认后统一对账，使用分批 GET pipeline 读取。替代订阅就绪后才释放原订阅。缺失定义保持订阅，等待后续发布。`desc.Manager.Snapshot()` 返回上一份配套的 Service/类型快照、缺失 key，以及表示有新输入等待同步的 `Syncing` 状态；本次不实现 inspect。

   Hub Watch client 使用 `vine.portal` 用户，其 ACL 仅允许读取 Portal rule、site、证书、descriptor、Rpc/Web endpoint、共享 revision key 以及所需的订阅。Redis 密码为空，用于进程内模式和分离部署调试。启用后端 mTLS 时，Portal 证书会认证客户端，并把其 SPIFFE 身份绑定到 `vine.portal` 用户。未启用 mTLS 时，用户名只能选择 ACL 角色；由于该角色可以读取 TLS 私钥，Redis endpoint 必须限制在受信网络中。

## 依赖关系

Portal 内部主要依赖关系是：

- `entry -> site, vault`
- `site -> access, epmgr`
- `access -> desc, epmgr`
- `desc -> hubwatch`
- `vault -> hubwatch`
- `epmgr -> hubwatch`

外部调用方只应装配 `src/server/app` 和 `src/server/flag`，不要直接依赖内部 module。

修改 Portal 时必须保持以下 owner 和依赖边界：

- `epmgr` 是 Rpc/Web endpoint 订阅、watcher 引用计数和轮询选择的唯一 owner；site 和 access 通过它获取 endpoint。
- `desc` 是所有 descriptor 订阅、缓存及 API 类型依赖的 owner；`access` 负责 Rpc/Web 准入及认证、权限检查所需的 endpoint 订阅。Gateway 和 access 不应维护第二份 descriptor 缓存。
- `entry` 只管理监听入口和 rule 分发，`site` 只管理 RpcGW/WebGW，`vault` 只管理证书加载与匹配。
- RpcGW/WebGW 必须继续传播当前 trace、initiator、actor、deadline 和剩余 timeout，不能用新的后台 context 覆盖请求上下文。
- 修改 header、转发路径、descriptor 或 endpoint 格式时，必须同步 Link、Hub Watch 结构、调用方以及 gateway 测试。

## Inproc 模式

Portal 支持作为 standalone 运行时的一部分在进程内启动。此时 Portal 仍然从 Hub Watch 读取并监听 portal rule、portal site、证书、descriptor 和 endpoint 注册信息，只是底层 Hub Watch client 会走进程内连接，而不是外部 TCP Redis。

Inproc/standalone 模式下需要注意：

- `entry`、`site`、`desc`、`access`、`epmgr`、`vault` 的模块边界和数据订阅语义保持不变。
- RpcGW/WebGW 仍然通过 endpoint 发现结果转发请求，但目标 endpoint 可能是 `link+inproc://`，不会经过外部 Link ingress TCP 端口。
- Portal 本身不负责 heartbeat 或 TTL 续租；注册信息在 inproc Hub 中通常长期有效，依赖应用显式 unregister 和 Hub Watch 事件驱动清理。
- 这种模式适合验证路由、准入、descriptor 监听和 gateway 转发逻辑，但不模拟外部网络断连、独立 Link/Portal 进程崩溃、TLS listener 端口不可达等分布式故障。

如果要验证真实网络入口、TLS 监听、跨进程 endpoint 可达性或注册租约过期，应使用普通进程模式启动 Portal。

修改入口、发现、认证或转发逻辑时，应分别验证 standalone/inproc 和普通网络部署。inproc 模式只保证路由与订阅语义，不代表真实监听、TLS、断连和租约行为已经覆盖。

RPC service/method 的认证模式支持 `required`、`optional`、`anonymous`；method 声明还支持
`inherit`。Web 另支持 `off`，service 和 Web 不允许使用 `inherit`。本地注册和 Hub 接收入口
均拒绝 RPC 显式使用 `off`。

生成的 descriptor 包含每个 method 的 `EffectiveAuthMode` 和 `EffectiveRequire`。
本地注册和 Hub 根据声明的策略校验这些派生值，Portal 直接应用它们，不再继承认证模式或
合并权限要求。Web 使用已校验的 `AuthMode`。

旧 schema 的兼容仅在注册入口处理。适配器将 `auth` 转为 `required`，RPC 的 `noauth`
转为 `optional`，Web 的 `noauth` 转为 `off`。service/Web 的空值和 `unset` 转为
`required`，method 的空值和 `unset` 转为 `inherit`。适配器计算 effective 策略后，
再校验转换出的 descriptor。新的 descriptor 必须已经包含有效的 effective 策略。

启用 Portal 认证时，无效凭证和认证服务错误均拒绝，不退回匿名。Web `off` 使用匿名 actor
并保留 Authorization，供 handler 自行认证。RPC 和 Web 的 `required`、`optional`、
`anonymous` 在准入通过后删除 Authorization，仅向下传递已准入的 actor。

Hub 在 `descriptor:web:*` 发布选中的 Web descriptor，Portal 订阅更新及删除事件；
具名 Web 请求缺少 descriptor 时拒绝。注册载荷和 Watch key 已切换为 descriptor，
Hub、Link 与 Portal 必须配套升级。旧 Web 契约现在默认要求认证；需要匿名访问或原生认证时，
必须显式选择 `optional` 或 `off` 并重新生成。旧 RPC `noauth` 会认证提供的凭证，
无效凭证会报错，不再隐式匿名放行。

WEBGW site 必须配置 `actorSkelName`，且指定的 actor 必须存在。Vine 忽略旧的 `via`
元数据；RPCGW 按 service 声明的 Actor audience 准入，再执行鉴权和权限检查。
Web 认证遵循声明的模式；actor 未配置认证时，`required` 会拒绝请求。
只有 `off` 会为 handler 保留原生 Authorization 凭证。

## RpcGW MCP

RPCGW site 的独立开关 `mcpEnabled` 默认为 false，开启后提供 `POST /mcp`。
Entry 将 `/api` 映射到站点根路径时，对外地址为 `/api/mcp`。仅支持 MCP
`2026-07-28`，入口要求对应的版本头并拒绝旧版初始化。官方 MCP Go SDK
负责 Streamable HTTP、请求元数据校验、服务发现和 JSON-RPC 错误；Portal 使用无状态
JSON 响应，仅声明 tools 能力，不发送工具列表变更通知。

MCP 与 Inspect 共用按站点 Actor 筛选的 API 目录，直接读取 `desc` 快照，不另设
描述符缓存。`tools/list` 排序分页，`tools/call` 只生成目标方法的 schema；同步中或
缺失类型时拒绝请求。工具名为 `{serviceSkelName}.{methodSkelName}`，超出 MCP 名称
字符或长度建议时使用稳定的 SHA-256 名称。输入和输出均执行 JSON Schema 校验，
`structuredContent` 直接包含 API 返回值，包括标量、数组及 `null`，文本 `content`
提供相同的 JSON。

调用复用 Rpc 准入和转发，仅复制调用方的 Authorization；Portal 生成后端 client、
trace 和 initiator，准入逻辑生成 actor，并在业务转发前删除凭据。MCP 与原有 `/invoke` 使用相同的断连策略：请求受理后，业务执行屏蔽 HTTP 和
SDK 取消，由服务端总时限和 Portal 停止控制。`params._meta["vmcp-options"]`
使用与 `vrpc-options` 相同的字符串格式（例如 `timeout=60s`），默认 30 秒，
最多 120 秒。非法选项在准入前返回 JSON-RPC Invalid Params，未知选项忽略。API 错误转换为 `isError`
工具结果，目录及协议错误使用 JSON-RPC error。所有 MCP 请求和预检均校验 Origin，
响应禁止缓存。该入口使用 Vine 凭据，不提供 OAuth 发现或授权服务器。
