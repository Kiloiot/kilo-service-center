# Multi-stage Dockerfile for KC-Web
# Build context: kilocenter-modules/ root

# Stage 1: Builder
FROM oven/bun:1.2-alpine AS builder

WORKDIR /build

# Release manifest required by vite.config.ts for version injection
COPY release/manifest.json ./release/manifest.json

COPY KC-Web/package.json KC-Web/bun.lock ./KC-Web/

# Proto sources needed by grpc:generate during postinstall
COPY KC-Core/api/proto/ /build/KC-Core/api/proto/

WORKDIR /build/KC-Web

# Install dependencies without postinstall (grpc:sync needs protoc which
# may not be available on all architectures). Pre-generated stubs are
# copied in the next COPY step.
RUN bun install --frozen-lockfile --ignore-scripts

COPY KC-Web/ .

# Wire pre-generated gRPC stubs into node_modules (no protoc needed).
# Stubs are committed in src/services/grpc/.
RUN bun run grpc:setup

# VITE_GRPC_URL empty — nginx proxies gRPC-web to kc-gateway on the internal network
ENV NODE_OPTIONS="--max-old-space-size=4096"
RUN VITE_GRPC_URL="" bun run --silent tsc -b && VITE_GRPC_URL="" bunx vite build

# Stage 2: Runtime
FROM nginx:1.27-alpine
ARG SOURCE_URL=https://github.com/Kiloiot/kilo-service-center
ARG DOCS_URL=https://docs.kiloiot.io/
LABEL org.opencontainers.image.title="KiloCenter Web"
LABEL org.opencontainers.image.description="Open-source MIOTY Service Center — web interface"
LABEL org.opencontainers.image.url=$SOURCE_URL
LABEL org.opencontainers.image.source=$SOURCE_URL
LABEL org.opencontainers.image.documentation=$DOCS_URL
LABEL org.opencontainers.image.licenses="AGPL-3.0-or-later"
LABEL org.opencontainers.image.vendor="Tim Kravchunovsky"
LABEL org.opencontainers.image.authors="Tim Kravchunovsky"

COPY --from=builder /build/KC-Web/dist /usr/share/nginx/html
COPY deployments/docker/nginx/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost/ || exit 1

STOPSIGNAL SIGTERM
