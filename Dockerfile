# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.27.0

FROM --platform=$BUILDPLATFORM node:24.11.0-slim AS dashboard

WORKDIR /src
RUN corepack enable \
    && corepack prepare pnpm@11.15.0 --activate
COPY internal/daemon/hub/src/dashboard/package.json internal/daemon/hub/src/dashboard/pnpm-lock.yaml internal/daemon/hub/src/dashboard/pnpm-workspace.yaml ./internal/daemon/hub/src/dashboard/
RUN pnpm --dir internal/daemon/hub/src/dashboard install --frozen-lockfile
COPY internal/daemon/hub/src/dashboard/ ./internal/daemon/hub/src/dashboard/
COPY script/build-dashboard-assets.sh ./script/build-dashboard-assets.sh
COPY THIRD_PARTY_LICENSES.txt ./THIRD_PARTY_LICENSES.txt
RUN bash script/build-dashboard-assets.sh

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=v0.0.0-dev

WORKDIR /src

# Keep dependency downloads in a separate layer so source-only changes reuse
# the Go module cache during image builds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=dashboard /src/internal/daemon/hub/src/server/mod/admin/assets/dashboard/ internal/daemon/hub/src/server/mod/admin/assets/dashboard/
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w -X go.yorun.ai/vine/buildinfo.ldModuleVersion=${VERSION}" \
    -o /out/vine ./cmd/vine

FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates libcap-setcap \
    && addgroup -S vine \
    && adduser -S -D -H -G vine vine \
    && mkdir -p /data \
    && chown vine:vine /data

COPY --from=build /out/vine /usr/local/bin/vine
COPY LICENSE THIRD_PARTY_LICENSES.txt /usr/share/licenses/vine/

# Portal may need to bind the default HTTP/HTTPS ports (80/443). Grant only
# the low-port capability so all images can still run as the unprivileged user.
RUN setcap cap_net_bind_service=+ep /usr/local/bin/vine

WORKDIR /data
USER vine:vine
ENTRYPOINT ["/usr/local/bin/vine"]

# Backend mTLS is enabled when all three file variables are supplied at
# runtime. Certificate material stays outside the image and should be mounted
# read-only by Docker or Kubernetes.
ENV VINE_MTLS_CA_FILE="" \
    VINE_MTLS_CERT_FILE="" \
    VINE_MTLS_KEY_FILE=""

# Build with --target hub to produce the Hub image.
FROM runtime AS hub

ENV VINE_CONTROL_LISTEN=0.0.0.0:7071 \
    VINE_ADMIN_LISTEN=0.0.0.0:7099 \
    VINE_WATCH_LISTEN=0.0.0.0:7072 \
    VINE_DB_SQLITE_FILE="" \
    VINE_DB_POSTGRES_URL="" \
    VINE_SEED_DATA_FILE=""

EXPOSE 7071 7072 7099
CMD ["hub", "serve"]

# Build with --target portal to produce the Portal image.
FROM runtime AS portal

ENV VINE_HUB_ENDPOINT=http://hub:7071

# Portal creates HTTP/HTTPS listeners from the rules stored in Hub. These are
# the default entry ports; additional configured entry ports can also be used.
EXPOSE 80 443
CMD ["portal", "serve"]

# Build with --target link to produce the Link image.
FROM runtime AS link

ENV VINE_HUB_ENDPOINT=http://hub:7071 \
    VINE_API_LISTEN=0.0.0.0:7079 \
    VINE_INGRESS_LISTEN=0.0.0.0:7082

EXPOSE 7079 7082
CMD ["link", "serve"]
