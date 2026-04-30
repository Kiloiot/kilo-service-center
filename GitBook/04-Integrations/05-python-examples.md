# Python Examples

This page provides runnable Python examples for the KiloCenter gRPC API. For the full API reference, see [API Reference](02-api.md).

For general Python gRPC documentation, see [grpc.io/docs/languages/python](https://grpc.io/docs/languages/python/).

## Prerequisites

> All commands in this guide assume your working directory is `kilocenter-modules/`.

Install the gRPC Python packages:

```bash
pip install grpcio grpcio-tools
```

Generate Python stubs from the proto definitions:

```bash
python -m grpc_tools.protoc \
  -I KC-Core/api/proto \
  --python_out=./gen \
  --grpc_python_out=./gen \
  KC-Core/api/proto/kilocenter.proto \
  KC-Core/api/proto/core.proto \
  KC-Core/api/proto/identity.proto
```

This produces `kilocenter_pb2.py`, `core_pb2.py`, `identity_pb2.py`, and their `_grpc` counterparts in the `gen/` directory.

## Get System Status

Retrieves the current system status including version, uptime, and service health.

```python
import sys
sys.path.insert(0, "gen")

import grpc
from google.protobuf.empty_pb2 import Empty

import kilocenter_pb2_grpc

# KiloCenter KC-Gateway gRPC endpoint
server = "localhost:9090"

channel = grpc.insecure_channel(server)
client = kilocenter_pb2_grpc.KiloCenterServiceStub(channel)

resp = client.GetSystemStatus(Empty())

print(f"Version:             {resp.version}")
print(f"Status:              {resp.status}")
print(f"Uptime:              {resp.uptime}")
print(f"Active Endpoints:    {resp.active_endpoints}")
print(f"Active Base Stations: {resp.active_basestations}")
print(f"Messages Processed:  {resp.messages_processed}")

for svc in resp.services:
    print(f"  Service: {svc.name}  healthy={svc.healthy}  latency={svc.latency_ms}ms")
```

## List Endpoints

Lists registered endpoints with pagination support. High-volume profile: default page size 100, max 1000.

```python
import sys
sys.path.insert(0, "gen")

import grpc

import kilocenter_pb2
import kilocenter_pb2_grpc

server = "localhost:9090"

channel = grpc.insecure_channel(server)
client = kilocenter_pb2_grpc.KiloCenterServiceStub(channel)

page_token = ""
page_size = 100

while True:
    resp = client.ListEndPoints(kilocenter_pb2.ListEndPointsRequest(
        page_size=page_size,
        page_token=page_token,
    ))

    print(f"Total endpoints: {resp.total_count}")

    for ep in resp.endpoints:
        print(f"  EUI: {ep.epEui}  Name: {ep.name}  Status: {ep.status}  Class: {ep.ep_class}")

    if not resp.next_page_token:
        break
    page_token = resp.next_page_token
```

## Authentication

### Community Edition

Community Edition runs in single-tenant mode with authentication and organization
enforcement disabled (`auth.enabled: false`, `org_enforcement_enabled: false`).
The examples above work without any headers.

### Enterprise: JWT User Principal

Requires three headers: `authorization`, `x-organization-id`, and `x-user-id`. The `x-user-id` value must match the authenticated user in the JWT. Missing `x-user-id` returns `ErrTokenUserIDHeaderRequired`; a mismatch returns `ErrTokenIdentityMismatch`.

```python
token = "your-jwt-token"
org_id = "your-organization-uuid"
user_id = "your-user-uuid"

metadata = [
    ("authorization", "Bearer " + token),
    ("x-organization-id", org_id),
    ("x-user-id", user_id),
]

resp = client.ListEndPoints(
    kilocenter_pb2.ListEndPointsRequest(page_size=100),
    metadata=metadata,
)
```

### Enterprise: Service-Account API Key

Requires two headers: `authorization` and `x-organization-id`. Do **not** send `x-user-id` — including it returns `ErrTokenIdentityMismatch` to prevent user injection.

```python
api_key = "your-api-key"
org_id = "your-organization-uuid"

metadata = [
    ("authorization", "Bearer " + api_key),
    ("x-organization-id", org_id),
]

resp = client.ListEndPoints(
    kilocenter_pb2.ListEndPointsRequest(page_size=100),
    metadata=metadata,
)
```
