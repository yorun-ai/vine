# Vine Kubernetes 部署示例

[English](README.md)

此目录包含 Hub、Link 和 Portal 的最小分离式部署示例。示例使用 Kustomize
base，可以直接执行 `kubectl apply -k examples/k8s`。

## 直接拉取镜像

清单已经引用已发布到 Docker Hub 的镜像，并设置了
`imagePullPolicy: Always`。镜像仓库公开后，无需本地构建即可直接部署：

```bash
docker pull docker.io/yorunai/vine-hub:latest
docker pull docker.io/yorunai/vine-link:latest
docker pull docker.io/yorunai/vine-portal:latest
kubectl apply -k examples/k8s
```

稳定部署时，请将所有清单中的 `:latest` 替换为工作流发布的不可变
`:vX.Y.Z` 标签。

## 后端 mTLS 覆盖层

基础清单使用 HTTP，因此不需要证书即可启动示例。如果需要启用后端 mTLS，
请为每个组件创建一个 Kubernetes Secret，然后应用 mTLS overlay。证书必须使用
仓库安全章节中说明的 SPIFFE 身份：

```text
spiffe://<trust-domain>/vine/daemon/vine.hub
spiffe://<trust-domain>/vine/daemon/vine.link
spiffe://<trust-domain>/vine/daemon/vine.portal
```

请使用仓库之外保存的证书文件创建 namespace 和各组件 Secret：

```bash
kubectl apply -f examples/k8s/base/namespace.yaml

kubectl -n vine create secret generic vine-hub-mtls \
  --from-file=ca.pem=mtls/ca.pem \
  --from-file=cert.pem=mtls/hub.pem \
  --from-file=key.pem=mtls/hub-key.pem \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n vine create secret generic vine-link-mtls \
  --from-file=ca.pem=mtls/ca.pem \
  --from-file=cert.pem=mtls/link.pem \
  --from-file=key.pem=mtls/link-key.pem \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n vine create secret generic vine-portal-mtls \
  --from-file=ca.pem=mtls/ca.pem \
  --from-file=cert.pem=mtls/portal.pem \
  --from-file=key.pem=mtls/portal-key.pem \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -k examples/k8s/overlays/mtls
```

该 overlay 会将每个 Secret 挂载到 `/run/vine/mtls`，设置三个
`VINE_MTLS_*` 环境变量，并将 Link/Portal 的 Hub 地址改为
`https://hub:7071`。Secret 文件以只读方式挂载，不会存储在 Git 中。
如需恢复为非 mTLS 示例，请执行 `kubectl apply -k examples/k8s`，并在不再
需要时删除这三个 Secret。

## 本地构建和发布镜像

在仓库根目录开发或向其他镜像仓库发布时，可以构建三个 Linux 镜像：

```bash
docker build --target hub -t vine-hub:local .
docker build --target link -t vine-link:local .
docker build --target portal -t vine-portal:local .
```

如果远程集群使用其他镜像仓库，请为镜像打标签并推送，然后替换
`base/hub.yaml`、`base/link.yaml` 和 `base/portal.yaml` 中的 `image:` 值，
包括匹配的 `wait-for-hub` init 容器镜像：

```bash
docker tag vine-hub:local registry.example.com/vine/hub:v0.1.0
docker tag vine-link:local registry.example.com/vine/link:v0.1.0
docker tag vine-portal:local registry.example.com/vine/portal:v0.1.0
docker push registry.example.com/vine/hub:v0.1.0
docker push registry.example.com/vine/link:v0.1.0
docker push registry.example.com/vine/portal:v0.1.0
```

仓库的容器镜像工作流会将发布镜像推送到
`docker.io/yorunai/vine-hub`、`docker.io/yorunai/vine-link` 和
`docker.io/yorunai/vine-portal`；基础清单已经使用这些名称。
远程集群清单不要使用 `:local` 标签。

对于 kind 或 minikube，请将本地构建的镜像加载到集群，而不是推送到镜像仓库：

```bash
kind load docker-image docker.io/yorunai/vine-hub:latest
kind load docker-image docker.io/yorunai/vine-link:latest
kind load docker-image docker.io/yorunai/vine-portal:latest
```

首次成功发布 CI 镜像后，请将 Docker Hub 中的每个仓库设置为 **Public**。
公开仓库在 Kubernetes 中不需要 `imagePullSecret`；私有仓库则需要创建 registry
Secret，并在工作负载 Pod 规范中配置 `imagePullSecrets`。

## 应用

```bash
kubectl apply -k examples/k8s
kubectl -n vine get pods,svc,pvc
kubectl -n vine logs statefulset/hub
```

Link 和 Portal Pod 都包含一个 init 容器，会等待 Hub 在 `hub:7071` 上提供
Control API。因此，虽然 Kubernetes 会独立启动三个工作负载，执行
`kubectl apply -k` 仍然是安全的。Hub 仍必须先就绪，Link 和 Portal 才能完成
初始化；如果 Pod 反复重启，请检查对应日志。

Hub Service 有意设置为 headless。内嵌 NATS 会选择动态端口，并通过 Hub 的
`InfoService` 上报；headless Service 让 Link 和 Portal 直接将 `hub` 解析到
Hub Pod IP，从而仍能访问动态端口。使用默认 SQLite 存储时，请将 Hub 保持为
单副本。如果切换到外部 NATS 和 PostgreSQL，请使用普通 Service，并相应更新
Hub 环境变量。

## 访问端点

- Hub Control API：`hub.vine.svc.cluster.local:7071`（内部）
- Hub Redis：`hub.vine.svc.cluster.local:7072`（内部）
- Hub Admin/Dashboard：`hub.vine.svc.cluster.local:7075`（内部）
- Link API：`link.vine.svc.cluster.local:7079`（内部）
- Link Ingress：`link.vine.svc.cluster.local:7082`（内部）
- Portal HTTP：Service 端口 `80`
- Portal HTTPS：Service 端口 `443`
- Portal 默认 Dashboard 入口：Service 端口 `7099`

Portal 默认使用 `LoadBalancer` Service。在没有云负载均衡器的集群中，请将其
改为 `ClusterIP`，通过 Ingress controller 暴露，或者在开发时使用
`kubectl port-forward`。

## 生产配置

Hub 镜像本身默认不选择数据库或 NATS 模式。基础清单显式选择内嵌 NATS 和 SQLite，
适用于小规模的单副本部署，并在组件之间使用 HTTP。`overlays/mtls` 示例展示了如何为
每个组件提供独立的后端 mTLS 文件 Secret，以只读方式挂载，设置 `VINE_MTLS_CA_FILE`、
`VINE_MTLS_CERT_FILE` 和 `VINE_MTLS_KEY_FILE`，并将 `VINE_HUB_ENDPOINT`
改为 HTTPS 地址。需要独立扩缩容和持久化基础设施时，请使用托管 PostgreSQL
数据库和外部 NATS。
