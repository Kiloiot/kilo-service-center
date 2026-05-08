# Multi-stage Dockerfile for KC-Gateway
# Build context: kilocenter-modules/ root
# Stage 1: Builder
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git make gcc musl-dev

WORKDIR /build

# Copy go.work first to enable workspace mode
COPY go.work go.work.sum ./

# Copy all modules referenced in go.work (required for workspace)
COPY KC-Gateway/ ./KC-Gateway/
COPY KC-Core/ ./KC-Core/
COPY KC-DB/ ./KC-DB/
COPY KC-Identity/ ./KC-Identity/
COPY KC-MQTT/ ./KC-MQTT/
COPY pkg/ ./pkg/

WORKDIR /build/KC-Gateway

RUN go mod download

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s" \
    -o gateway ./cmd/gateway

# Stage 2: Runtime
FROM alpine:3.23
ARG SOURCE_URL=https://github.com/Kiloiot/kilo-service-center
ARG DOCS_URL=https://docs.kiloiot.io/
LABEL org.opencontainers.image.title="KiloCenter"
LABEL org.opencontainers.image.description="Open-source MIOTY Service Center"
LABEL org.opencontainers.image.url=$SOURCE_URL
LABEL org.opencontainers.image.source=$SOURCE_URL
LABEL org.opencontainers.image.documentation=$DOCS_URL
LABEL org.opencontainers.image.licenses="AGPL-3.0-or-later"
LABEL org.opencontainers.image.vendor="Tim Kravchunovsky"
LABEL org.opencontainers.image.authors="Tim Kravchunovsky"

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1000 kilocenter && \
    adduser -D -u 1000 -G kilocenter kilocenter

RUN mkdir -p /app/config && \
    chown -R kilocenter:kilocenter /app

COPY --from=builder /build/KC-Gateway/gateway /usr/local/bin/gateway

USER kilocenter

WORKDIR /app

# Expose ports (gRPC-web: 9090, Health: 8087)
EXPOSE 9090 8087

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8087/health || exit 1

STOPSIGNAL SIGTERM

ENTRYPOINT ["gateway"]
CMD ["-config", "/app/config/config.yaml"]
