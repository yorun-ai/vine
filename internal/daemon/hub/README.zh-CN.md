# Vine Hub

[English](README.md) | **简体中文**

配置与服务注册中心，大体遵循 DDD 分层设计，负责维护配置、应用状态与 Rpc 服务注册，并通过采用 Redis 协议的 Watch 服务对外提供读取与订阅能力。

未指定数据库参数时，Hub 默认启用 `--no-db`，必须提供 `--seed-data-file`。
每次启动将配置加载到独立的内存 SQLite，初始化完成后，repo 层禁止修改
应用配置、Portal entry、站点、规则和证书。请编辑 seed 文件后重启 Hub。
Dashboard 展示只读提示并禁用编辑入口；注册、schema 和租约仍可写。
显式指定 `--db-sqlite-file` 或 `--db-postgres-url` 则保留可写持久化行为，
它们与 `--no-db` 互斥。standalone 也遵循这些规则，它还可通过
`Option.SeedHubData` 传入内联 YAML，与 seed 文件互斥，使用相同的导入和只读机制。

## 目录结构

```text
internal/daemon/hub/
├── api/                  Hub 对其他 runtime 组件暴露的公共 API
│   ├── app/              Hub inproc endpoint 等公共常量
│   ├── nats/             Hub NATS inproc 访问入口
│   ├── watch/            Hub Watch client、事件与 inproc 访问入口
│   ├── watched/          Watch 数据结构与 key 格式
│   └── skeled/           生成的 control/admin Go package
├── skel/                 Control 与 Admin skeleton 定义
└── src/
    ├── dashboard/        Dashboard 前端源码
    └── server/           Hub 服务端运行目录
        ├── app/          应用装配层，决定 Hub 启用哪些 component、module 和 servicer
        ├── comp/         运行时共享组件，如 `watchserver`、`natsserver`
        ├── core/         领域层，定义状态对象、Core 与 Repo 接口
        ├── flag/         Hub 启动参数与默认值规范化
        ├── impl/         按对外 API 边界拆分的接口实现层
        │   ├── control/  面向 Link/Portal 的 Control API 服务
        │   └── admin/    Dashboard Admin Rpc 与 Web 服务
        ├── mod/          运行时模块层，如 controlapi、initializer、seeder、syncer
        └── repo/         基础设施适配层，实现 core 中定义的 Repo 接口
```

## Dashboard 打包

- 修改 Dashboard 源码后，在 `src/server/mod/admin/dashboard` 运行 `pnpm typecheck` 和 `pnpm build`。
- 容器和 Release 二进制构建直接内嵌已提交的 `src/server/mod/admin/dashboard/dist/`，不重新构建前端；随源码变更更新并提交这些资源，PR CI 要求重建结果与提交内容完全一致。
- 面向用户的文案需要同步更新 `src/i18n/dictionaries/cn.ts` 和 `en.ts`。

Dashboard 前端源码位于 `src/server/mod/admin/dashboard`。Release 和容器构建把 Dashboard 编进 Vine；本地构建同样内嵌已提交的 `dashboard/dist` 目录。启动 Hub 时设置 `VINE_HUB_DASHBOARD_DEV_PROXY=1`，会探测 `localhost:7098` 的 Vite 服务，可用时优先代理，否则使用内嵌资源。环境变量未设置或为空时不探测；Admin API 请求始终由 Hub 处理。

使用脚本生成内嵌资源，不要手工编辑生成文件：

```bash
bash script/build-dashboard-assets.sh
GOWORK=off go build -o bin/vine ./cmd/vine
```

脚本先做类型检查，将前端构建到临时目录，再按原路径保留全部构建文件，不额外压缩。
全部成功后替换整个 dist 目录，避免残留旧文件。临时 Vite 报告收集实际打包的 JS 依赖声明，
再补充导入的 CSS、字体及已维护的 uiw 许可证。打包先与根目录
`THIRD_PARTY_LICENSES.txt` 的 Dashboard 部分比较，再移除临时报告。
打包依赖变化后，用 `bash script/gen-third-party-licenses.sh` 更新统一清单。
不分发归档、Go 字节数组或独立的 Dashboard 许可证文件。Release 归档携带统一清单，
三个镜像均在 `/usr/share/licenses/vine/` 下携带清单和 `LICENSE`。
所有 Go 构建都通过 `go:embed` 内嵌该目录，无需构建标签。

Dashboard 通过共享的 Asset Server 处理 MIME、HEAD 和压缩协商。文本响应根据客户端
支持的格式动态压缩。已压缩的图片和字体不再动态压缩。HTML 页面导航回退到 `index.html`，缺失的静态文件返回 404。

## 分层与变更约束

Hub 的层次职责必须保持清晰：

- `core` 定义领域状态与 Repo 接口，不依赖具体数据库或 Redis 实现。Repo 接口命名的是它提供的存储原语（`List`、`GetById`、`GetByName`、`Save`、`Remove`），建立在其上的 core 暴露的是用例，因此两层有意共用这些动词。
- `repo` 实现持久化和 Watch 同步，不承载对外服务编排。repo 负责装配它返回的完整实体：站点 Web 挂载路径与 Rpc 服务等派生值、配置的定义与状态，以及 field source 这类已存储的溯源信息。
- `impl/control` 只实现面向 Link/Portal 的 Control API；`impl/admin`
  及其 `debug`、`dashboard` 子包通过 `core` 实现 Dashboard
  Admin API 能力。
- `mod` 承载 Control API listener、initializer、seeder、syncer、scheduler、
  sweeper 等运行时流程。
- `comp` 提供 Watch、NATS 等共享运行时组件。
- `app` 只负责装配 component、module 和 servicer。

修改 Hub 时还应遵守：

- Core 的 `Validate` 只做校验与归一化，不写存储。规则校验不再查询站点；Hub 随站点发布 Web 挂载路径元数据，由 Portal 生成实际生效的规则路径。
- 应用配置、entry、站点、规则和证书的写入都经过各自的 Core。entry 拥有 Portal
  监听的 scheme、host 和 port，规则只引用 entry，不再自行保存访问配置。
  `PortalRuleCore` 按规则声明的访问配置解析 entry，因此访问配置相同的规则共用
  同一个 entry。修改 entry 会改变它路由的全部规则，Hub 随即重新发布这些规则，
  让 Portal 读到该 entry 当前服务的访问配置。
- `PortalEntryCore` 为尚无用户 entry 的访问配置创建 entry，之后按 `id` 寻址，
  因此 Admin API 与 Dashboard 可以改名、修改它服务的访问配置或切换开关；并允许
  删除没有路由任何规则的 entry：规则属于用户，删除仍有规则的 entry 会报错，而不是
  让规则失去访问配置。entry 列表会返回尚未路由规则的 entry，因为用户先建 entry、
  再添加使用它的规则。
- Hub 在 admin 模块自己的监听上提供 Admin API 与 Dashboard（`--admin-listen`，
  默认 `127.0.0.1:7099`，与 Control API 的 `--control-listen` 对称）：该监听在 RPC
  路径 `/api/invoke` 上响应 API（Dashboard 调用的就是该路径），其它路径
  都返回内嵌的 Dashboard 构建产物，Dashboard 因此不再属于 Portal 配置——Hub 不为
  它创建任何 entry、站点或规则，Portal 也不会路由 Dashboard。
- Portal 站点、entry、规则与证书在库里都有 `enabled` 开关（默认启用），Dashboard
  可编辑；seed 用 `disabled`（默认 false）声明同一个开关，只标出要停用的实体，
  写 `enabled` 的 seed 会直接报错，避免被忽略后继续发布。Hub 会把停用的实体保留在
  数据库里但停止发布到 Watch，Portal 因此完全看不到它：停用的规则、停用 entry 下
  的规则、停用站点上的 SITE 规则以及停用的证书都会从发布内容中移除。早于该开关的
  数据库中的实体保持启用。
- entry 有自己的名称：seed 的 `portalEntries` 段声明 `name`、`scheme`、`host`、
  `port`，并在规则之前应用，因此规则会加入服务其访问配置的 entry 并沿用该名称；
  entry 也可以暂时不承载任何规则。Hub 只为它自行创建的 entry 推导
  `scheme[:host]:port` 名称，所以没有显式声明 entry 时规则加入的 entry 以访问配置
- Portal 各段是强类型的：实体声明了该段没有的字段时 Hub 直接报错，避免拼错或改名
  后的字段被静默忽略、实体停留在默认值；Hub 自己不再创建任何实体。
- name 只在同一类实体内唯一，所以站点、entry 与规则可以同名。
- 规则加入 entry 有两种写法：用 `entryName` 指定名称，或直接声明该 entry 服务的
  访问配置（`matchScheme` / `matchHost` / `matchPort`）。两者互斥：同一条规则不
  能同时使用两种写法，同一份 seed 文档也只能全部使用其中一种；声明了
  `portalEntries` 的 seed 必须用 `entryName` 引用这些 entry，而不是在规则上声明
  访问配置。seed 必须自洽：规则只能引用同一份文档声明的 entry，Hub 不会用库里的
  数据补全关系。访问配置属于 entry，Hub 会在写入任何内容之前拒绝这类文档。
- Admin API 通过 entry 触达规则的访问配置：`PortalRuleCreation` 指定新规则属于哪个
  entry，`PortalRuleUpdate` 完全不能修改访问配置。seed YAML 仍在规则上声明访问
  配置，由 Hub 在应用 seed 时聚合为 entry。
- 两条规则匹配同一个请求时行为是「报告」而不是「拒绝」：规则匹配的路径由 Web
  声明的 mount 决定，而这些 schema 是应用在 Hub 启动之后才注册的，所以写入时
  无法判断。Hub 在 schema 到位后审计并报出重复的请求（`no-db` 的只读配置直接
  拒绝启动，因为没有可修复的界面；有数据库的配置继续运行，由操作者在 Dashboard
  上解决）。只有访问配置迁移会自行消解这类冲突，因为已存储的数据不靠人工修改。
- 数据库表结构必须同时更新 `src/server/repo/db/model/sql/sqlite` 和 `src/server/repo/db/model/sql/pgsql`。
- Redis key、Redis value JSON 和事件格式属于 Hub、Link、Portal 之间的协议；修改时必须同步所有生产者、消费者和测试。
- `watchserver` 是运行时分发层，不应成为绕过 Repo/Core 直接实现业务规则的第二套状态源。
- 普通模式与 inproc 模式的 TTL、heartbeat、sweeper 语义不同；修改注册逻辑时必须分别验证。

## 运行机制

Hub 的职责可以拆成四条主线：

1. 配置中心
   Hub 从数据库读取配置，并通过 `AppConfigRepo` 对外提供配置读取能力。启动时 `initializer` 会把配置装入 Redis，供 Link 侧读取和订阅。

2. 服务注册中心
   Link 会把应用状态与 Rpc 服务注册写入 Hub。Hub 通过 `RegistryRepo` 持久化这些状态，并对外提供查询与心跳续租能力。

3. Watch 分发层
   `watchserver` 维护一份内存 Redis 数据。配置、应用状态、Rpc/Web endpoint 和 schema 都会同步写入其中，Link 与 Portal 通过 Redis 读取快照并监听变更事件。

   内嵌 Redis 协议要求客户端在执行数据命令前完成认证，并为三个用户分别配置资源级 ACL：

   - `vine.hub` 拥有完整的命令与 key 权限，密码在当前进程中随机生成。
   - `vine.link` 可以读取配置、Rpc endpoint 注册与 revision key，并且只能订阅配置 channel 和 Rpc 注册 pattern。
   - `vine.portal` 可以读取 Portal rule、site、证书、actor/service/resource schema、Rpc/Web endpoint 注册与 revision key，并且只能订阅对应的列表 pattern。

   Link 与 Portal 的 Redis 密码为空，用于进程内模式和分离部署调试。启用后端 mTLS 时，客户端证书会认证调用方，并把其 SPIFFE 身份绑定到对应的 Redis 用户名。未启用 mTLS 时，用户名只能选择最小权限角色，不能认证调用方，因此 Redis endpoint 必须位于回环地址或受信私有网络，并通过防火墙限制访问。

4. 分离的 API listener
   Control API listener 向 Link 和 Portal 暴露 `vine.hub.control` 域，其中
   只包含 `InfoService` 与 `RegistryService`。admin listener 以明文暴露
   `vine.hub.admin` 域，其中包含 Dashboard Admin Rpc 服务和
   `DashboardWeb`（浏览器不持有 mesh 证书）。二者共享同一个 Hub 进程和状态，但组件流量无法直接进入管理面。

启用内嵌 NATS 时，server component 会使用内存存储预创建 `VINE_EVENTS` 和
`VINE_TASKS` JetStream stream。外部 NATS 部署负责创建 stream 并决定存储策略；
Hub publisher 只使用已经存在的 stream。

## 配置与注册来源

Hub 当前支持两类数据库配置来源：

- SQLite
- PostgreSQL

启动时可以通过 `--seed-data-file` 让 `seeder` 从本地 YAML 文件一次性导入初始配置、站点规则和证书到数据库；导入后 Hub 仍然统一从数据库 repo 读取，再写入 Redis，对 Link 暴露一致的读取与订阅语义。

数据库升级基线为 Vine `v0.15.7`，规则表应已具备 `match_*` / `route_*` 列。
更早的数据库应先用 `v0.15.7` 启动完成迁移。当前 Hub 不再执行该基线之前的
Portal rule 列重命名和路由路径列迁移；仍会按下文所述，把 `match_scheme`、
`match_host`、`match_port` 中的规则访问配置迁移到 `portal_entry`。

从把访问配置存在规则上的版本升级时，Hub 会原地迁移 `portal_rule`：建立
`portal_entry`，把已存储的 `match_scheme`、`match_host`、`match_port` 归入
entry。这些列保留到后续版本再删除：Hub 从
升级后就不再读取它们，并会一直写入所属 entry 的访问配置，因为删除用户数据库
上的列无法撤销。未设置的端口会迁移成 Portal 实际监听的端口。若两条规则此前只
靠未设置的端口区分，迁移后落在同一 entry 的同一路径上，Hub 保留显式写了端口的
那条，把使用默认端口的规则挪到 `/migrated` 路径，并逐条记录日志：升级不会要求
用户手工修库，Hub 也会正常启动。退回旧版本后 Hub 仍能读写该数据库：它读取的访问
列仍在，enabled 有"默认启用"的默认值，它插入的不带 entry 的规则会在下次升级时
重新归入对应 entry。

数据库 metadata 记录首次 seed 完成状态。后续启动跳过全部 seed、变量和来源输入，seed 条目不再提供 `override` 开关。无数据库模式每次建立新存储并导入 seed。

字段来源以 JSON 保存原始字段模板，并记录每次替换的相对路径、变量名、占位符、实际应用的 JSON 值和默认值使用标记。AppConfig 的嵌套替换归属 value 的一级 key；管理接口修改字段后清除旧模板和替换记录。管理 API 与 Dashboard 一同展示这些信息及字段来源。

`mod/seeder` 负责 seed YAML 契约：它把文档解码成 Hub 要应用的领域实体，payload 结构体与解析结果保持包内私有。它接受旧规则字段并逐字段告警；同一条规则混用新旧字段会在 Hub 写入任何内容之前失败。

seed 仍在规则上声明 `matchScheme`、`matchHost` 和 `matchPort`。应用 seed 时 Hub
会把声明的访问配置聚合为 entry，因此 seed 不会把同一份访问配置写到每条规则上。
Portal 收到的规则仍携带其 entry 的访问配置；entry 是 Hub 侧状态，不是 Watch key。

## Admin 载荷约定

Admin 的 list 方法返回 `*ListItem`：只包含 Dashboard 列表需要展示的值，不含实体的溯源信息。详情响应（`get`、`create`、`update`）返回实体本身并携带 `fieldSources`，因此 Dashboard 表单可以直接回读刚写入的内容。AppConfig 的详情按 `key` 而不是 `id` 定位，因为 Hub 也会返回应用声明但尚未配置值的配置项；`update` 与 `remove` 仍按 `id` 定位已存储的行。

## Skeleton 生成

Hub 在 `skel/control` 与 `skel/admin` 中分别维护两套契约。Go 代码生成到
对应的 `api/skeled/control` 与 `api/skeled/admin` package，TypeScript
仅为 admin 生成代码，输出到 `src/server/mod/admin/dashboard/src/skeled/admin`。统一使用顶层脚本：

```bash
bash script/gen-skel.sh hub
```

不要直接修改生成文件。应修改 `skel/control` 或 `skel/admin` 中对应的
契约，再通过上述脚本重新生成对应代码，并检查所有调用方是否仍然一致。

## Inproc 模式

Hub 支持作为单进程内组件运行：

- Hub Control API 注册在 `rpc+inproc://vine/hub`；Admin API 只走自己的监听，
  因此除非调用方声明了 admin 监听地址，进程内运行的 Hub 不提供 Admin API。
- `watchserver` 不再启动对外 TCP 端口，只保留进程内 Redis server。
- `vined` 中会保存这份进程内 Redis server 指针，供 inproc 模式下的 `WatchClient` 直接使用。

此时 Hub 仍然承担配置中心和注册中心职责，只是底层不再通过网络暴露。

## TTL 与 Heartbeat

Hub 在普通模式和 inproc 模式下，对注册信息的处理不同：

- 普通模式
  - 应用状态与 Rpc 服务注册写入 Watch 时会带 TTL。
  - Link 通过 heartbeat 持续续租。
  - Hub 通过 registry sweeper 扫描过期 app lease，主动 unregister 过期实例并发布 delete 事件。
  - Watch key TTL 是兜底清理机制，实际的注册失效事件由 Hub sweeper 负责发布。

- Inproc 模式
  - 应用状态与 Rpc 服务注册写入 Watch 时不再设置 TTL。
  - `KeepAppStatus` 和 `KeepRpcServiceRegistration` 变为 noop。
  - registry sweeper 不启动。
  - 状态改为长期有效，依赖显式 unregister 清理。

这使得单进程模式下不再需要 heartbeat 维持注册状态。

`dashboard/dist` 的真实构建产物随源码提交；随源码变更运行 `bash script/build-dashboard-assets.sh` 更新，不放占位文件。
