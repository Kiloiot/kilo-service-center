# gRPC First Steps

## Goal
Verify local gRPC connectivity to KC-Core and discover available API methods.

## Check Service Reachability

```bash
grpcurl -plaintext localhost:9090 list
```

## Inspect Main Service

```bash
grpcurl -plaintext localhost:9090 describe kilocenter.api.v1.KiloCenterService
```

## Check gRPC Health Service

```bash
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

## Call a Basic API Method

```bash
grpcurl -plaintext -d '{}' \
  localhost:9090 kilocenter.api.v1.KiloCenterService/GetSystemStatus
```

## Authentication and Organization Context

The Community Edition runs with authentication disabled by default in development mode. The gRPC commands above work without auth headers in this configuration.

If you enable authentication (Enterprise Edition or by setting `KILOCENTER_AUTH_ENABLED=true`), include metadata headers in gRPC calls:

```bash
grpcurl -plaintext \
  -H "authorization: Bearer <JWT_TOKEN>" \
  -H "x-organization-id: <ORG_ID>" \
  -d '{}' \
  localhost:9090 kilocenter.api.v1.KiloCenterService/GetSystemStatus
```

## Next Steps

- [API Reference](./02-api.md) — full RPC catalog, authentication, pagination, and error handling
