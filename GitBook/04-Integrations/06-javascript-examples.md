# JavaScript Examples

This page provides runnable Node.js examples for the KiloCenter gRPC API using dynamic proto loading. For the full API reference, see [API Reference](02-api.md).

For general Node.js gRPC documentation, see [grpc.io/docs/languages/node](https://grpc.io/docs/languages/node/).

## Prerequisites

> All commands in this guide assume your working directory is `kilocenter-modules/`.

Install the gRPC and proto-loader packages:

```bash
npm install @grpc/grpc-js @grpc/proto-loader
```

No stub generation is needed — `@grpc/proto-loader` loads `.proto` files at runtime. Ensure the proto files are accessible at the paths referenced below (relative to the KiloCenter repository root):

- `KC-Core/api/proto/kilocenter.proto`
- `KC-Core/api/proto/core.proto`
- `KC-Core/api/proto/identity.proto`

## Get System Status

Retrieves the current system status including version, uptime, and service health.

```javascript
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");

// KiloCenter KC-Gateway gRPC endpoint
const server = "localhost:9090";

const packageDef = protoLoader.loadSync("KC-Core/api/proto/kilocenter.proto", {
  keepCase: false,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
  includeDirs: ["KC-Core/api/proto"],
});

const proto = grpc.loadPackageDefinition(packageDef);
const client = new proto.kilocenter.api.v1.KiloCenterService(
  server,
  grpc.credentials.createInsecure()
);

client.GetSystemStatus({}, (err, resp) => {
  if (err) {
    console.error("GetSystemStatus failed:", err.message);
    process.exit(1);
  }

  console.log("Version:            ", resp.version);
  console.log("Status:             ", resp.status);
  console.log("Uptime:             ", resp.uptime);
  console.log("Active Endpoints:   ", resp.activeEndpoints);
  console.log("Active Base Stations:", resp.activeBasestations);
  console.log("Messages Processed: ", resp.messagesProcessed);

  for (const svc of resp.services) {
    console.log(`  Service: ${svc.name}  healthy=${svc.healthy}  latency=${svc.latencyMs}ms`);
  }
});
```

## List Endpoints

Lists registered endpoints with pagination support. High-volume profile: default page size 100, max 1000.

```javascript
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");

const server = "localhost:9090";

const packageDef = protoLoader.loadSync("KC-Core/api/proto/kilocenter.proto", {
  keepCase: false,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
  includeDirs: ["KC-Core/api/proto"],
});

const proto = grpc.loadPackageDefinition(packageDef);
const client = new proto.kilocenter.api.v1.KiloCenterService(
  server,
  grpc.credentials.createInsecure()
);

function listEndpoints(pageToken = "") {
  client.ListEndPoints({ pageSize: 100, pageToken }, (err, resp) => {
    if (err) {
      console.error("ListEndPoints failed:", err.message);
      process.exit(1);
    }

    console.log("Total endpoints:", resp.totalCount);

    for (const ep of resp.endpoints) {
      console.log(`  EUI: ${ep.epEui}  Name: ${ep.name}  Status: ${ep.status}  Class: ${ep.epClass}`);
    }

    if (resp.nextPageToken) {
      listEndpoints(resp.nextPageToken);
    }
  });
}

listEndpoints();
```

## Authentication

### Community Edition

Community Edition runs in single-tenant mode with authentication and organization
enforcement disabled (`auth.enabled: false`, `org_enforcement_enabled: false`).
The examples above work without any headers.

### Enterprise: JWT User Principal

Requires three headers: `authorization`, `x-organization-id`, and `x-user-id`. The `x-user-id` value must match the authenticated user in the JWT. Missing `x-user-id` returns `ErrTokenUserIDHeaderRequired`; a mismatch returns `ErrTokenIdentityMismatch`.

```javascript
const token = "your-jwt-token";
const orgId = "your-organization-uuid";
const userId = "your-user-uuid";

const metadata = new grpc.Metadata();
metadata.set("authorization", "Bearer " + token);
metadata.set("x-organization-id", orgId);
metadata.set("x-user-id", userId);

client.ListEndPoints({ pageSize: 100 }, metadata, (err, resp) => {
  if (err) {
    console.error("ListEndPoints failed:", err.message);
    return;
  }
  console.log("Total endpoints:", resp.totalCount);
});
```

### Enterprise: Service-Account API Key

Requires two headers: `authorization` and `x-organization-id`. Do **not** send `x-user-id` — including it returns `ErrTokenIdentityMismatch` to prevent user injection.

```javascript
const apiKey = "your-api-key";
const orgId = "your-organization-uuid";

const metadata = new grpc.Metadata();
metadata.set("authorization", "Bearer " + apiKey);
metadata.set("x-organization-id", orgId);

client.ListEndPoints({ pageSize: 100 }, metadata, (err, resp) => {
  if (err) {
    console.error("ListEndPoints failed:", err.message);
    return;
  }
  console.log("Total endpoints:", resp.totalCount);
});
```
