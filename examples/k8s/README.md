# Vine Kubernetes deployment examples

This directory contains a minimal separated deployment example for Hub, Link,
and Portal. It is a Kustomize base and can be applied directly with
`kubectl apply -k examples/k8s`.

## Direct image pulls

The manifests already reference the published Docker Hub images and use
`imagePullPolicy: Always`. After the Container images workflow has published
the repositories and they are public, deploy directly without building
anything locally:

```bash
docker pull docker.io/yorunai/vine-hub:latest
docker pull docker.io/yorunai/vine-link:latest
docker pull docker.io/yorunai/vine-portal:latest
kubectl apply -k examples/k8s
```

For a stable deployment, replace `:latest` with an immutable `:vX.Y.Z` tag
published by the workflow in all manifests.

## Build and publish images locally

Build the three Linux images from the repository root when developing locally
or publishing to another registry:

```bash
docker build --target hub -t vine-hub:local .
docker build --target link -t vine-link:local .
docker build --target portal -t vine-portal:local .
```

For a remote cluster using another registry, tag and push the images, then
replace the `image:` values in `hub.yaml`, `link.yaml`, and `portal.yaml`
(including the matching `wait-for-hub` init container image):

```bash
docker tag vine-hub:local registry.example.com/vine/hub:v0.1.0
docker tag vine-link:local registry.example.com/vine/link:v0.1.0
docker tag vine-portal:local registry.example.com/vine/portal:v0.1.0
docker push registry.example.com/vine/hub:v0.1.0
docker push registry.example.com/vine/link:v0.1.0
docker push registry.example.com/vine/portal:v0.1.0
```

The repository's Container images workflow publishes release images to
`docker.io/yorunai/vine-hub`, `docker.io/yorunai/vine-link`, and
`docker.io/yorunai/vine-portal`; the base manifests already use these names.
Do not use `:local` in a remote cluster manifest.

For kind or minikube, load the locally built images into the cluster instead of
pushing them:

```bash
kind load docker-image docker.io/yorunai/vine-hub:latest
kind load docker-image docker.io/yorunai/vine-link:latest
kind load docker-image docker.io/yorunai/vine-portal:latest
```

After the first successful CI publish, change each package's visibility to
**Public** in GitHub: open the repository's **Packages** page, select the
`vine-hub`, `vine-link`, and `vine-portal` packages, open **Package settings**,
choose **Change visibility**, and select **Public**. Public packages do not need
an `imagePullSecret` in Kubernetes.

## Apply

```bash
kubectl apply -k examples/k8s
kubectl -n vine get pods,svc,pvc
kubectl -n vine logs statefulset/hub
```

The Link and Portal Pods have an init container that waits for Hub's Control API
on `hub:7071`. This makes `kubectl apply -k` safe to use even though Kubernetes
starts the three workloads independently. Hub must still become ready before
Link and Portal can initialize; inspect their logs if a Pod is restarting.

The Hub Service is intentionally headless. Embedded NATS chooses a dynamic
port and reports it through Hub's `InfoService`; a headless Service lets Link
and Portal resolve `hub` directly to the Hub Pod IP so that dynamic port remains
reachable. Keep Hub at one replica when using the default SQLite storage. If
you switch to external NATS and PostgreSQL, use a normal Service and update the
Hub environment accordingly.

## Endpoints

- Hub Control API: `hub.vine.svc.cluster.local:7071` (internal)
- Hub Redis: `hub.vine.svc.cluster.local:7072` (internal)
- Hub Admin/Dashboard: `hub.vine.svc.cluster.local:7075` (internal)
- Link API: `link.vine.svc.cluster.local:7079` (internal)
- Link Ingress: `link.vine.svc.cluster.local:7082` (internal)
- Portal HTTP: Service port `80`
- Portal HTTPS: Service port `443`
- Portal default Dashboard entry: Service port `7099`

Portal uses a `LoadBalancer` Service by default. On clusters without a cloud
load balancer, change it to `ClusterIP` and expose it through your Ingress
controller or use `kubectl port-forward` for development.

## Production configuration

The base manifests use HTTP between components and Hub's embedded NATS with
SQLite for a small single-replica deployment. For production, provide separate
Secrets for each component's backend mTLS files, mount them under a read-only
path, set `VINE_MTLS_CA_FILE`, `VINE_MTLS_CERT_FILE`, and
`VINE_MTLS_KEY_FILE`, and change `VINE_HUB_ENDPOINT` to an HTTPS endpoint.
Use a managed PostgreSQL database and external NATS when you need independent
scaling and durable infrastructure ownership.
