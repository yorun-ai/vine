# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.27.0

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=v0.0.0-dev

WORKDIR /src

# Keep dependency downloads in a separate layer so source-only changes reuse
# the Go module cache during image builds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
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
    VINE_ADMIN_LISTEN=0.0.0.0:7075 \
    VINE_REDIS_LISTEN=0.0.0.0:7072 \
    VINE_DB_SQLITE_FILE=/data/hub.sqlite \
    VINE_DB_POSTGRES_URL="" \
    VINE_MQ_EXTERNAL_NATS_URL="" \
    VINE_MQ_EMBEDDED_NATS=true \
    VINE_SEED_YAML_FILE="" \
    VINE_DASHBOARD_URL=""

EXPOSE 7071 7072 7075
CMD ["hub", "serve"]

# Build with --target portal to produce the Portal image.
FROM runtime AS portal

ENV VINE_HUB_ENDPOINT=http://hub:7071

# Portal creates HTTP/HTTPS listeners from the rules stored in Hub. These are
# the default ports; the Hub-seeded Dashboard entry uses 7099, and additional
# configured entry ports can also be used.
EXPOSE 80 443 7099
CMD ["portal", "serve"]

# Build with --target link to produce the Link image.
FROM runtime AS link

ENV VINE_HUB_ENDPOINT=http://hub:7071 \
    VINE_API_LISTEN=0.0.0.0:7079 \
    VINE_INGRESS_LISTEN=0.0.0.0:7082

EXPOSE 7079 7082
CMD ["link", "serve"]
