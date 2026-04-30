# Go Examples

This page provides runnable Go examples for the KiloCenter gRPC API. For the full API reference, see [API Reference](02-api.md).

For general Go gRPC documentation, see [grpc.io/docs/languages/go](https://grpc.io/docs/languages/go/).

## Prerequisites

> All commands in this guide assume your working directory is `kilocenter-modules/`.

Generate Go stubs from the proto definitions using buf:

```bash
cd KC-Core/api/proto && buf generate
```

This produces generated code in `KC-Core/api/gen/kilocenter/v1/`.

Install the required Go modules:

```bash
go get google.golang.org/grpc
go get google.golang.org/protobuf
```

## Get System Status

Retrieves the current system status including version, uptime, and service health.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	kilocenterv1 "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
)

// server is the KiloCenter KC-Gateway gRPC endpoint.
var server = "localhost:9090"

func main() {
	conn, err := grpc.NewClient(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := kilocenterv1.NewKiloCenterServiceClient(conn)

	resp, err := client.GetSystemStatus(context.Background(), &emptypb.Empty{})
	if err != nil {
		log.Fatalf("GetSystemStatus failed: %v", err)
	}

	fmt.Printf("Version:            %s\n", resp.Version)
	fmt.Printf("Status:             %s\n", resp.Status)
	fmt.Printf("Uptime:             %s\n", resp.Uptime.AsTime())
	fmt.Printf("Active Endpoints:   %d\n", resp.ActiveEndpoints)
	fmt.Printf("Active Base Stations: %d\n", resp.ActiveBasestations)
	fmt.Printf("Messages Processed: %d\n", resp.MessagesProcessed)

	for _, svc := range resp.Services {
		fmt.Printf("  Service: %s  healthy=%v  latency=%dms\n", svc.Name, svc.Healthy, svc.LatencyMs)
	}
}
```

## List Endpoints

Lists registered endpoints with pagination support. High-volume profile: default page size 100, max 1000.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	kilocenterv1 "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
)

var server = "localhost:9090"

func main() {
	conn, err := grpc.NewClient(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := kilocenterv1.NewKiloCenterServiceClient(conn)

	pageToken := ""
	pageSize := int32(100)

	for {
		resp, err := client.ListEndPoints(context.Background(), &kilocenterv1.ListEndPointsRequest{
			PageSize:  pageSize,
			PageToken: pageToken,
		})
		if err != nil {
			log.Fatalf("ListEndPoints failed: %v", err)
		}

		fmt.Printf("Total endpoints: %d\n", resp.TotalCount)

		for _, ep := range resp.Endpoints {
			fmt.Printf("  EUI: %s  Name: %s  Status: %s  Class: %s\n",
				ep.EpEui, ep.Name, ep.Status, ep.EpClass)
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
}
```

## Authentication

### Community Edition

Community Edition runs in single-tenant mode with authentication and organization
enforcement disabled (`auth.enabled: false`, `org_enforcement_enabled: false`).
The examples above work without any headers.

### Enterprise: JWT User Principal

Requires three headers: `authorization`, `x-organization-id`, and `x-user-id`. The `x-user-id` value must match the authenticated user in the JWT. Missing `x-user-id` returns `ErrTokenUserIDHeaderRequired`; a mismatch returns `ErrTokenIdentityMismatch`.

```go
import "google.golang.org/grpc/metadata"

token := "your-jwt-token"
orgID := "your-organization-uuid"
userID := "your-user-uuid"

ctx := metadata.AppendToOutgoingContext(context.Background(),
	"authorization", "Bearer "+token,
	"x-organization-id", orgID,
	"x-user-id", userID,
)

resp, err := client.ListEndPoints(ctx, &kilocenterv1.ListEndPointsRequest{
	PageSize: 100,
})
```

### Enterprise: Service-Account API Key

Requires two headers: `authorization` and `x-organization-id`. Do **not** send `x-user-id` — including it returns `ErrTokenIdentityMismatch` to prevent user injection.

```go
apiKey := "your-api-key"
orgID := "your-organization-uuid"

ctx := metadata.AppendToOutgoingContext(context.Background(),
	"authorization", "Bearer "+apiKey,
	"x-organization-id", orgID,
)

resp, err := client.ListEndPoints(ctx, &kilocenterv1.ListEndPointsRequest{
	PageSize: 100,
})
```
