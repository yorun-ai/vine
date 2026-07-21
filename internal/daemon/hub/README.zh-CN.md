# Vine Hub

[English](README.md) | **简体中文**

配置与服务注册中心，大体遵循 DDD 分层设计，负责维护配置、应用状态与 Rpc 服务注册，并通过 Redis 对外提供读取与订阅能力。

## 目录结构

```text
internal/daemon/hub/
├── api/                  Hub 对其他 runtime 组件暴露的公共 API
│   ├── app/              Hub inproc endpoint 等公共常量
│   ├── nats/             Hub NATS inproc 访问入口
│   ├── redis/            Hub Redis client、事件与 inproc 访问入口
│   ├── redised/          写入 Redis 的结构与 key 格式
│   └── skeled/           Hub skel 生成的公共类型、client 与 schema
├── skel/                 Hub 自身的 skeleton 定义
└── src/
    ├── dashboard/        Dashboard 前端源码
    └── server/           Hub 服务端运行目录
        ├── app/          应用装配层，决定 Hub 启用哪些 component、module 和 servicer
        ├── comp/         运行时共享组件，如 `redisserver`、`natsserver`
        ├── core/         领域层，定义状态对象、Core 与 Repo 接口
        ├── flag/         Hub 启动参数与默认值规范化
        ├── impl/         接口实现层，承载 Hub 对外暴露的服务实现
        ├── mod/          运行时模块层，如 initializer、seeder、syncer
        └── repo/         基础设施适配层，实现 core 中定义的 Repo 接口
```

## Dashboard 打包

- 调试时使用 `VINE_HUB_DASHBOARD_DEV_PROXY` 环境变量，直接转发请求到启动的 `pnpm dev`。
- 修改 Dashboard 源码后，在 `src/dashboard` 运行 `pnpm typecheck` 和 `pnpm build`。
- 普通开发任务不要更新嵌入的 `dashboard.tar.zst`；只有任务明确要求更新发布资源时才重新打包。
- 面向用户的文案需要同步更新 `src/i18n/dictionaries/cn.ts` 和 `en.ts`。

Dashboard 前端源码位于 `src/dashboard`，Hub 运行时读取的是嵌入在 `src/server/impl/dashboard/assets/dashboard.tar.zst` 中的构建产物。

发布时更新打包产物必须运行：

```bash
bash script/build-dashboard-assets.sh
```

该脚本会进入 `src/dashboard` 执行 `pnpm run build`，为 Dashboard bundle 中实际包含的依赖生成 `THIRD_PARTY_LICENSES.md`，再把 `dist` 打包为新的 `dashboard.tar.zst`。不要手工组装归档。更新发布资源时必须同时提交新的 `dashboard.tar.zst`，否则内置 Hub Dashboard 及其许可证清单仍会使用旧版本。

## 分层与变更约束

Hub 的层次职责必须保持清晰：

- `core` 定义领域状态与 Repo 接口，不依赖具体数据库或 Redis 实现。
- `repo` 实现持久化和 Redis 同步，不承载对外服务编排。
- `impl` 实现 Hub 对外服务，通过 `core` 和 `repo` 完成业务操作。
- `mod` 承载 initializer、seeder、syncer、scheduler、sweeper 等运行时流程。
- `comp` 提供 Redis、NATS 等共享运行时组件。
- `app` 只负责装配 component、module 和 servicer。

修改 Hub 时还应遵守：

- 数据库表结构必须同时更新 `src/server/repo/db/model/sql/sqlite` 和 `src/server/repo/db/model/sql/pgsql`。
- Redis key、Redis value JSON 和事件格式属于 Hub、Link、Portal 之间的协议；修改时必须同步所有生产者、消费者和测试。
- `redisserver` 是运行时分发层，不应成为绕过 Repo/Core 直接实现业务规则的第二套状态源。
- 普通模式与 inproc 模式的 TTL、heartbeat、sweeper 语义不同；修改注册逻辑时必须分别验证。

## 运行机制

Hub 的职责可以拆成三条主线：

1. 配置中心
   Hub 从数据库读取配置，并通过 `AppConfigRepo` 对外提供配置读取能力。启动时 `initializer` 会把配置装入 Redis，供 Link 侧读取和订阅。

2. 服务注册中心
   Link 会把应用状态与 Rpc 服务注册写入 Hub。Hub 通过 `RegistryRepo` 持久化这些状态，并对外提供查询与心跳续租能力。

3. Redis 分发层
   `redisserver` 维护一份内存 Redis 数据。配置、应用状态、Rpc/Web endpoint 和 schema 都会同步写入其中，Link 与 Portal 通过 Redis 读取快照并监听变更事件。

## 配置与注册来源

Hub 当前支持两类数据库配置来源：

- SQLite
- PostgreSQL

启动时可以通过 `--seed-yaml-file` 让 `seeder` 从本地 YAML 文件一次性导入初始配置、站点规则和证书到数据库；导入后 Hub 仍然统一从数据库 repo 读取，再写入 Redis，对 Link 暴露一致的读取与订阅语义。

## Skeleton 生成

Hub 的 Go 公共代码生成到 `api/skeled`，Dashboard 使用的 TypeScript 代码生成到 `src/dashboard/src/skeled`。统一使用顶层脚本：

```bash
bash script/gen-skel.sh hub
```

不要直接修改 `api/skeled` 或 `src/dashboard/src/skeled`。应修改 `skel/` 中的契约，再通过上述脚本同时生成 Go 和 TypeScript 代码，并检查两端调用是否仍然一致。

## Inproc 模式

Hub 支持作为单进程内组件运行：

- Hub Rpc 服务不再暴露为 HTTP，而是注册到 `inproc` transport。
- `redisserver` 不再启动对外 TCP 端口，只保留进程内 Redis server。
- `vined` 中会保存这份进程内 Redis server 指针，供 inproc 模式下的 `RedisClient` 直接使用。

此时 Hub 仍然承担配置中心和注册中心职责，只是底层不再通过网络暴露。

## TTL 与 Heartbeat

Hub 在普通模式和 inproc 模式下，对注册信息的处理不同：

- 普通模式
  - 应用状态与 Rpc 服务注册写入 Redis 时会带 TTL。
  - Link 通过 heartbeat 持续续租。
  - Hub 通过 registry sweeper 扫描过期 app lease，主动 unregister 过期实例并发布 delete 事件。
  - Redis key TTL 是兜底清理机制，实际的注册失效事件由 Hub sweeper 负责发布。

- Inproc 模式
  - 应用状态与 Rpc 服务注册写入 Redis 时不再设置 TTL。
  - `KeepAppStatus` 和 `KeepRpcServiceRegistration` 变为 noop。
  - registry sweeper 不启动。
  - 状态改为长期有效，依赖显式 unregister 清理。

这使得单进程模式下不再需要 heartbeat 维持注册状态。
