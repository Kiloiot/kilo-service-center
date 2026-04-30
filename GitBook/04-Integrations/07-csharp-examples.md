# C# Examples

This page provides runnable C# examples for the KiloCenter gRPC API. For the full API reference, see [API Reference](02-api.md).

For general C# gRPC documentation, see [grpc.io/docs/languages/csharp](https://grpc.io/docs/languages/csharp/).

## Prerequisites

> All commands in this guide assume your working directory is `kilocenter-modules/`.

Create a new console project and add the required NuGet packages:

```bash
dotnet new console -n KiloCenterClient
cd KiloCenterClient
dotnet add package Grpc.Net.Client
dotnet add package Google.Protobuf
dotnet add package Grpc.Tools
```

Add the proto files to your `.csproj` for code generation:

```xml
<ItemGroup>
  <Protobuf Include="path/to/KC-Core/api/proto/kilocenter.proto"
            GrpcServices="Client"
            AdditionalImportDirs="path/to/KC-Core/api/proto" />
  <Protobuf Include="path/to/KC-Core/api/proto/core.proto"
            GrpcServices="None"
            AdditionalImportDirs="path/to/KC-Core/api/proto" />
  <Protobuf Include="path/to/KC-Core/api/proto/identity.proto"
            GrpcServices="None"
            AdditionalImportDirs="path/to/KC-Core/api/proto" />
</ItemGroup>
```

Replace `path/to/` with the actual path to the KiloCenter repository. Building the project auto-generates C# stubs.

## Get System Status

Retrieves the current system status including version, uptime, and service health.

```csharp
using Grpc.Net.Client;
using Google.Protobuf.WellKnownTypes;
using Kilocenter.Api.V1;

// KiloCenter KC-Gateway gRPC endpoint
var server = "http://localhost:9090";

using var channel = GrpcChannel.ForAddress(server);
var client = new KiloCenterService.KiloCenterServiceClient(channel);

var resp = await client.GetSystemStatusAsync(new Empty());

Console.WriteLine($"Version:             {resp.Version}");
Console.WriteLine($"Status:              {resp.Status}");
Console.WriteLine($"Uptime:              {resp.Uptime}");
Console.WriteLine($"Active Endpoints:    {resp.ActiveEndpoints}");
Console.WriteLine($"Active Base Stations: {resp.ActiveBasestations}");
Console.WriteLine($"Messages Processed:  {resp.MessagesProcessed}");

foreach (var svc in resp.Services)
{
    Console.WriteLine($"  Service: {svc.Name}  healthy={svc.Healthy}  latency={svc.LatencyMs}ms");
}
```

## List Endpoints

Lists registered endpoints with pagination support. High-volume profile: default page size 100, max 1000.

```csharp
using Grpc.Net.Client;
using Kilocenter.Api.V1;

var server = "http://localhost:9090";

using var channel = GrpcChannel.ForAddress(server);
var client = new KiloCenterService.KiloCenterServiceClient(channel);

var pageToken = "";
var pageSize = 100;

do
{
    var resp = await client.ListEndPointsAsync(new ListEndPointsRequest
    {
        PageSize = pageSize,
        PageToken = pageToken,
    });

    Console.WriteLine($"Total endpoints: {resp.TotalCount}");

    foreach (var ep in resp.Endpoints)
    {
        Console.WriteLine($"  EUI: {ep.EpEui}  Name: {ep.Name}  Status: {ep.Status}  Class: {ep.EpClass}");
    }

    pageToken = resp.NextPageToken;
}
while (!string.IsNullOrEmpty(pageToken));
```

## Authentication

### Community Edition

Community Edition runs in single-tenant mode with authentication and organization
enforcement disabled (`auth.enabled: false`, `org_enforcement_enabled: false`).
The examples above work without any headers.

### Enterprise: JWT User Principal

Requires three headers: `authorization`, `x-organization-id`, and `x-user-id`. The `x-user-id` value must match the authenticated user in the JWT. Missing `x-user-id` returns `ErrTokenUserIDHeaderRequired`; a mismatch returns `ErrTokenIdentityMismatch`.

```csharp
using Grpc.Core;

var token = "your-jwt-token";
var orgId = "your-organization-uuid";
var userId = "your-user-uuid";

var headers = new Metadata
{
    { "authorization", "Bearer " + token },
    { "x-organization-id", orgId },
    { "x-user-id", userId },
};

var resp = await client.ListEndPointsAsync(
    new ListEndPointsRequest { PageSize = 100 },
    headers
);
```

### Enterprise: Service-Account API Key

Requires two headers: `authorization` and `x-organization-id`. Do **not** send `x-user-id` — including it returns `ErrTokenIdentityMismatch` to prevent user injection.

```csharp
var apiKey = "your-api-key";
var orgId = "your-organization-uuid";

var headers = new Metadata
{
    { "authorization", "Bearer " + apiKey },
    { "x-organization-id", orgId },
};

var resp = await client.ListEndPointsAsync(
    new ListEndPointsRequest { PageSize = 100 },
    headers
);
```
