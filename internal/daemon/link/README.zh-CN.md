# Vine Link

[English](README.md) | **简体中文**

应用侧接入层，负责把本地应用注册到 Hub、同步配置与服务发现状态，并为本地与远端应用之间提供统一的 Rpc 转发入口。

## 目录结构

```text
internal/daemon/link/
├── skel/                 Link 自身的 skeleton 定义
└── src/
    └── server/           Link 服务端运行目录
        ├── app/          应用装配层，决定 Link 启用哪些 module 和 servicer
        ├── comp/         运行时共享组件，如 `hubinfo`、Hub Redis client 与 NATS client
        ├── flag/         Link 启动参数与默认值规范化
        ├── impl/         对外服务实现层，承载 Link 暴露的注册与配置服务
        └── mod/          运行时模块层
            ├── minder/   本地应用管理、变更广播、注册、heartbeat 与健康检查
            ├── config/   配置读取与 Redis 变化订阅
            ├── event/    事件 consumer 管理与投递
            ├── ingress/  对外 HTTP 入口，把请求分发给 proxy
            ├── rpcproxy/ 本地与远端应用之间的 Rpc 转发
            ├── task/     任务 consumer 管理与投递
            └── webproxy/ 本地与远端应用之间的 Web 转发
```

Link 的 skel 生成代码统一放在 `internal/core/link/skeled`，由 runtime 外的 Core 层承载公共协议类型。生成入口为：

```bash
bash script/gen-skel.sh link
```

不要直接修改 `internal/core/link/skeled`。应修改 `skel/` 中的契约，再通过上述脚本生成代码并检查所有协议使用方。

## 运行机制

Link 的职责可以拆成四条主线：

1. 本地应用注册
   app 启动后调用 `RegistryService`，`minder` 会写入本地实例真源，向 Hub 发布应用注册信息，并启动 heartbeat 与健康检查。

2. 配置与发现
   `config.Reader` 会从 Hub 对应的 Redis 中拉取配置并持续监听变更；`rpcproxy` 和 `webproxy` 分别维护 Rpc 与 Web 的发现状态。

3. 异步事件与任务
   `event` 和 `task` 订阅 `minder` 中本地应用声明的监听能力与运行能力，负责把 NATS 中的消息投递到对应本地应用。

4. Rpc 转发
   本地应用发起 Rpc 时，先进入 `rpcproxy out`。Link 基于服务发现结果决定请求应该发往本地实例还是远端实例；若请求从外部 HTTP 入口进入，则先经过 `ingress`，再由 `rpcproxy in` 转发到本地应用。

## 关键设计

当前 Link 以 `minder` 为本地 app instance 的单一 owner：

- 每个本地应用实例只保留一份基础事实：`appInfo`、endpoint、Rpc 服务、Web 名称、事件监听、任务执行声明。
- `minder` 同时负责实例真源、mutator 广播与生命周期动作。
- `rpcproxy`、`webproxy`、`event`、`task`、`config` 都通过 mutator 机制感知 app instance 变化，并只维护各自的派生运行态。

各 module 的职责是：

- `minder`：保存本地应用声明，提供 `Register`、`Unregister`、`Instance` 与 mutator 广播，并处理 register/unregister、Hub register、heartbeat、healthcheck。
- `config`：只负责配置值引用、快照与监听。
- `rpcproxy`：只负责 Rpc 路由、本地服务索引与远端服务发现缓存。
- `webproxy`：只负责 Web 路由与本地 Web 索引。
- `event`：只负责事件监听运行态与派发。
- `task`：只负责任务执行运行态与派发。
- `ingress`：只负责把外部 HTTP 请求分发到 `rpcproxy` 或 `webproxy`。

修改这些模块时必须保持以下边界：

- 不要在 `rpcproxy`、`webproxy`、`event`、`task` 或 `config` 中建立第二份本地 app instance 真源。
- 新增 app 能力或注册字段时，先扩展 `minder` 保存的基础事实和 mutator，再让下游 module 维护派生状态。
- module 不应绕过 `minder` 直接承担 register、unregister、heartbeat 或 healthcheck 生命周期。
- Rpc/Web endpoint、事件和任务协议发生变化时，必须同步 Hub Redis 结构、App 侧注册代码和相关转发/投递测试。

## 依赖关系

module 之间的依赖关系是：

- `rpcproxy -> minder`
- `webproxy -> minder`
- `event -> minder`
- `task -> minder`
- `config -> minder`
- `ingress -> rpcproxy, webproxy`

对外 service 与 module 的关系是：

- `RegistryService -> minder`
- `ConfigService -> config`
- `EventService -> event`
- `TaskService -> task`

## Inproc 模式

Link 支持作为单进程内组件运行：

- `LinkApp` 本身可以 inproc 启动，此时 Link 的 Rpc 服务不再暴露为 HTTP，而是注册到 `inproc` transport。
- `HubInprocMode` 为真时，Link 会把 Hub 视为进程内组件：
  - `HubEndpoint` 固定为 Hub 的 inproc endpoint。
  - `config.Reader`、`rpcproxy`、`webproxy` 都通过进程内 Redis 连接访问 Hub，而不是外部 TCP Redis 端口。
  - `minder` 不再启动 heartbeat。
  - `minder` 中的本地应用健康检查直接 noop。

在这种模式下，Link 仍然保留配置与服务监听机制，只是底层从网络 Redis 连接切换为进程内 Redis 连接。

### 故障语义差异

Inproc/standalone 模式主要用于单进程运行、集成测试和本地调试，它保留的是注册、发现、转发、订阅等运行时语义，不完整模拟分布式部署中的进程存活判断：

- Hub 侧注册信息不依赖 TTL 过期清理，而是依赖应用停止时显式 unregister。
- Link 不启动 heartbeat，因此不会通过续租失败来发现 Hub 或网络异常。
- Link 不启动本地 app console healthcheck，因此不会模拟“业务 app 进程仍在但 HTTP/Rpc handler 不健康”这类故障。
- Rpc/Web 转发仍会经过 `rpcproxy` 和 `webproxy` 的路由和发现逻辑，但底层 endpoint 可能是 `rpc+inproc://`、`web+inproc://` 或 `link+inproc://`。

如果要验证 heartbeat、TTL 过期、网络断连、独立进程崩溃等分布式故障语义，应使用普通 Hub、Link、Portal 独立进程模式。

修改 Link 的连接、注册、发现或转发逻辑时，应同时覆盖普通网络模式和 inproc 模式。不要因为 inproc 测试通过就假定 heartbeat、TTL、healthcheck 或跨进程 endpoint 行为正确。
