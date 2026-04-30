# Multi-stage Dockerfile for KC-Identity
# Build context: kilocenter-modules/ root
# Stage 1: Builder
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git make gcc musl-dev

WORKDIR /build

# Copy go.work first to enable workspace mode
COPY go.work go.work.sum ./

# Copy all modules referenced in go.work (required for workspace)
COPY KC-Identity/ ./KC-Identity/
COPY KC-Core/ ./KC-Core/
COPY KC-DB/ ./KC-DB/
COPY KC-Gateway/ ./KC-Gateway/
COPY KC-MQTT/ ./KC-MQTT/
COPY pkg/ ./pkg/

WORKDIR /build/KC-Identity

RUN go mod download

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s" \
    -o identity ./cmd/identity

# Stage 2: Runtime
FROM alpine:3.19
ARG SOURCE_URL=https://github.com/Kiloiot/KiloServiceCenter
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

RUN mkdir -p /app/config /app/migrations && \
    chown -R kilocenter:kilocenter /app

COPY --from=builder /build/KC-Identity/identity /usr/local/bin/identity
COPY --from=builder /build/KC-DB/migrations /app/migrations

USER kilocenter

WORKDIR /app

# Expose ports (gRPC: 50052, Health: 8088)
EXPOSE 50052 8088

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8088/health || exit 1

STOPSIGNAL SIGTERM

ENTRYPOINT ["identity"]
CMD ["-config", "/app/config/config.yaml"]
