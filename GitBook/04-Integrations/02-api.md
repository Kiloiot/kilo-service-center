# API Reference

KiloCenter provides a gRPC API for integrating with or extending the platform. All API definitions use Protocol Buffers.

## Interactive API Reference

A browsable API reference with all RPC endpoints is available at:

[https://servicecenter-api.kiloiot.io/](https://servicecenter-api.kiloiot.io/)

## gRPC Overview

gRPC is a high-performance RPC framework that uses Protocol Buffers for serialization and HTTP/2 for transport. For background, see [grpc.io](https://grpc.io/).

KiloCenter's API is defined across three proto files:

| File | Description |
|------|-------------|
| `kilocenter.proto` | Unified backward-compatible service (all 117 RPCs) |
| `core.proto` | Core domain messages (endpoints, base stations, messages, downlinks, events, certificates, blueprints) |
| `identity.proto` | Identity domain messages (users, organizations, API keys, auth) |

## Client Generation

Proto files ship in the `api/proto/` directory. Use your language's gRPC toolchain to
generate client stubs. See [grpc.io/docs/languages](https://grpc.io/docs/languages/)
for per-language quickstarts, or use the [Language Examples](#language-examples) below.

## Authentication

### Community Edition

Community Edition runs in single-tenant mode with authentication and organization
enforcement disabled (`auth.enabled: false`, `org_enforcement_enabled: false`).
No headers are required — all RPCs are accessible directly.

```bash
grpcurl -plaintext -d '{}' \
  localhost:9090 kilocenter.api.v1.KiloCenterService/GetSystemStatus
```

### Enterprise Edition

When authentication is enabled (`KILOCENTER_AUTH_ENABLED=true`), three metadata headers apply:

| Header | Required For | Value |
|--------|-------------|-------|
| `authorization` | All non-public RPCs | `Bearer <JWT_TOKEN>` or `Bearer <API_KEY>` |
| `x-organization-id` | All non-exempt RPCs | UUID of the target organization |
| `x-user-id` | RPCs called by user principals (JWT) | UUID of the acting user (must match JWT subject). Required for user principals; must NOT be sent for service-account API keys. |

**Dual bearer auth:** The `authorization` header accepts two token shapes:

1. **JWT tokens** (format: `header.payload.signature`) — validated via JWKS/HMAC. Tenant, org, and user are extracted from JWT claims.
2. **Opaque API keys** (any non-JWT shape) — hashed and looked up server-side. Tenant and org come from the key record. User context is set only for user-type keys; service-account keys carry no user identity.

**Identity validation:** When auth establishes identity (JWT or API key), the `x-user-id` and `x-organization-id` headers must *confirm* the authenticated identity — they cannot replace it. A mismatch between header values and the authenticated context returns `PermissionDenied`.

**Public methods** (no headers required):
`Login`, `RefreshTokens`, `GetAuthSettings`, `ExchangeOIDC`, `ExchangeOAuth2`, `GetReleaseInfo`, `RegisterAccount`

**Org-exempt methods** (auth required, no org header):
`GetSystemStatus`, `GetProfile`, `Logout`, `ChangePassword`, User/Org/Membership CRUD (org context resolved from request fields)

API tokens are created through KC-Web or the `CreateApiKey` RPC.

## API Reference by Domain

### Endpoints (7)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `CreateEndPoint` | CreateEndPointRequest | EndPoint | Register new endpoint |
| `GetEndPoint` | GetEndPointRequest | EndPoint | Get endpoint details |
| `UpdateEndPoint` | UpdateEndPointRequest | EndPoint | Update endpoint configuration |
| `DeleteEndPoint` | DeleteEndPointRequest | Empty | Remove endpoint |
| `ListEndPoints` | ListEndPointsRequest | ListEndPointsResponse | List endpoints with pagination |
| `AttachEndPoint` | AttachEndPointRequest | AttachEndPointResponse | Attach endpoint to base station |
| `DetachEndPoint` | DetachEndPointRequest | DetachEndPointResponse | Detach endpoint from base station |

### Base Stations (7)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `CreateBaseStation` | CreateBaseStationRequest | BaseStation | Register new base station |
| `GetBaseStation` | GetBaseStationRequest | BaseStation | Get base station details |
| `UpdateBaseStation` | UpdateBaseStationRequest | BaseStation | Update base station configuration |
| `DeleteBaseStation` | DeleteBaseStationRequest | Empty | Remove base station |
| `ListBaseStations` | ListBaseStationsRequest | ListBaseStationsResponse | List base stations with pagination |
| `GetBaseStationStats` | GetBaseStationStatsRequest | GetBaseStationStatsResponse | Get base station statistics |
| `UpdateBaseStationEui` | UpdateBaseStationEuiRequest | BaseStation | Update base station EUI |

#### BaseStation Location Fields

The `BaseStation` message includes geolocation fields:

| Field | Type | Description |
|-------|------|-------------|
| `latitude` | `google.protobuf.DoubleValue` | WGS-84 latitude (-90 to 90). Nullable: nil means not set, 0.0 is a valid coordinate. |
| `longitude` | `google.protobuf.DoubleValue` | WGS-84 longitude (-180 to 180). Nullable: nil means not set, 0.0 is a valid coordinate. |
| `altitude` | `google.protobuf.DoubleValue` | Altitude in meters. Nullable: nil means not set, 0.0 is a valid value. |
| `location_source` | `string` | `"gps"` (reported by the base station via BSSCI) or `"manual"` (set via API). Read-only. |
| `location_updated_at` | `google.protobuf.Timestamp` | Timestamp of the last location change. Read-only. |

#### UpdateBaseStation Location Semantics

`UpdateBaseStationRequest` uses `google.protobuf.FieldMask` via `update_mask` to specify which fields to update. Supported paths: `name`, `description`, `latitude`, `longitude`, `altitude`.

- **Nullable wrappers:** When a field path is included in `update_mask` but the corresponding wrapper value is nil, the stored value is cleared (set to NULL). This allows explicit removal of location data.
- **Lat/lon pair rule:** `latitude` and `longitude` must both be present in the mask or both omitted. Setting one without the other returns `INVALID_ARGUMENT`. `altitude` is independent.
- **GPS-authoritative:** When a base station reports geolocation via the BSSCI connect or status handshake, the reported coordinates overwrite any manually set values and `location_source` is set to `"gps"`. Manual updates via `UpdateBaseStation` set `location_source` to `"manual"`.

### Messages (11)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `GetMessage` | GetMessageRequest | Message | Get single message by ID |
| `ListMessages` | ListMessagesRequest | ListMessagesResponse | List messages with filtering |
| `StreamMessages` | StreamMessagesRequest | stream Message | Real-time message stream |
| `ListBaseStationMessages` | ListBaseStationMessagesRequest | ListBaseStationMessagesResponse | Messages for a specific base station |
| `GetBaseStationMessage` | GetBaseStationMessageRequest | GetBaseStationMessageResponse | Single base station message |
| `GetBaseStationMessageStats` | GetBaseStationMessageStatsRequest | GetBaseStationMessageStatsResponse | Base station message statistics |
| `SearchBaseStationMessages` | SearchBaseStationMessagesRequest | SearchBaseStationMessagesResponse | Search messages |
| `ExportBaseStationMessages` | ExportBaseStationMessagesRequest | ExportBaseStationMessagesResponse | Export messages |
| `StreamBaseStationMessages` | StreamBaseStationMessagesRequest | stream BaseStationMessage | Real-time base station message stream |
| `ListEndpointMessages` | ListEndpointMessagesRequest | ListEndpointMessagesResponse | Messages for a specific endpoint |
| `ListBaseStationActivity` | ListBaseStationActivityRequest | ListBaseStationActivityResponse | Unified activity feed (events + messages) for a base station |

### Downlinks (4)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `SendDownlink` | SendDownlinkRequest | SendDownlinkResponse | Queue downlink message |
| `RevokeDownlink` | RevokeDownlinkRequest | RevokeDownlinkResponse | Cancel queued downlink |
| `ListDownlinkQueue` | ListDownlinkQueueRequest | ListDownlinkQueueResponse | List pending downlinks |
| `GetDownlinkResults` | GetDownlinkResultsRequest | GetDownlinkResultsResponse | Get downlink transmission results |

### UL Transmit (1)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `SendULTransmit` | SendULTransmitRequest | SendULTransmitResponse | Transmit uplink via base station |

### Base Station Control (2)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `RequestBaseStationStatus` | BaseStationStatusRequest | BaseStationStatusResponse | Request status from base station |
| `InitiatePing` | InitiatePingRequest | InitiatePingResponse | Ping base station |

### DL RX Status (3)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `GetDLRXStatus` | GetDLRXStatusRequest | GetDLRXStatusResponse | Get downlink RX status |
| `QueryDLRXStatus` | QueryDLRXStatusRequest | QueryDLRXStatusResponse | Query DL RX status |
| `GetDLRXStatusQueries` | GetDLRXStatusQueriesRequest | GetDLRXStatusQueriesResponse | List DL RX status queries |

### System (3)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `GetSystemStatus` | Empty | SystemStatus | System health status |
| `GetStatistics` | GetStatisticsRequest | Statistics | System statistics |
| `GetReleaseInfo` | Empty | ReleaseInfo | Version, build info, and disclosure metadata |

`GetReleaseInfo` also returns: `edition`, `license_id`, `license_url`, `source_url`, `documentation_url`, `homepage_url`, `trademark_notice`.

### API Keys (4)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `CreateApiKey` | CreateApiKeyRequest | CreateApiKeyResponse | Create API key |
| `GetApiKey` | GetApiKeyRequest | GetApiKeyResponse | Get API key details |
| `DeleteApiKey` | DeleteApiKeyRequest | DeleteApiKeyResponse | Revoke API key |
| `ListApiKeys` | ListApiKeysRequest | ListApiKeysResponse | List API keys |

### Integrations (5)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `CreateIntegration` | CreateIntegrationRequest | Integration | Create integration |
| `GetIntegration` | GetIntegrationRequest | Integration | Get integration details |
| `UpdateIntegration` | UpdateIntegrationRequest | Integration | Update integration |
| `DeleteIntegration` | DeleteIntegrationRequest | Empty | Delete integration |
| `ListIntegrations` | ListIntegrationsRequest | ListIntegrationsResponse | List integrations |

### Analytics (3)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `GetAnalyticsOverview` | GetAnalyticsOverviewRequest | GetAnalyticsOverviewResponse | Dashboard overview |
| `GetActivityAnalytics` | GetActivityAnalyticsRequest | GetActivityAnalyticsResponse | Activity metrics |
| `GetSignalQualityAnalytics` | GetSignalQualityAnalyticsRequest | GetSignalQualityAnalyticsResponse | Signal quality metrics |

### Events & Alerts (8)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `ListEvents` | ListEventsRequest | ListEventsResponse | List system events |
| `ListBaseStationEvents` | ListBaseStationEventsRequest | ListBaseStationEventsResponse | Base station events |
| `ListEndPointEvents` | ListEndPointEventsRequest | ListEndPointEventsResponse | Endpoint events |
| `StreamEvents` | StreamEventsRequest | stream Event | Real-time system event stream |
| `StreamBaseStationEvents` | StreamBaseStationEventsRequest | stream Event | Real-time base station event stream |
| `StreamEndPointEvents` | StreamEndPointEventsRequest | stream Event | Real-time endpoint event stream |
| `ListAlerts` | ListAlertsRequest | ListAlertsResponse | List alerts |
| `GetAlertSummary` | GetAlertSummaryRequest | GetAlertSummaryResponse | Alert summary |

### SCACI Monitoring (6)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `ListScaciSessions` | ListScaciSessionsRequest | ListScaciSessionsResponse | List SCACI sessions |
| `GetScaciSession` | GetScaciSessionRequest | GetScaciSessionResponse | Get session details |
| `GetScaciStatistics` | GetScaciStatisticsRequest | GetScaciStatisticsResponse | SCACI statistics |
| `ListScaciErrors` | ListScaciErrorsRequest | ListScaciErrorsResponse | SCACI error log |
| `ListScaciQueues` | ListScaciQueuesRequest | ListScaciQueuesResponse | SCACI queue status |
| `GetScaciStatus` | GetScaciStatusRequest | GetScaciStatusResponse | SCACI system status |

### Certificates (6)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `GenerateCertificate` | GenerateCertificateRequest | GenerateCertificateResponse | Generate client certificate |
| `DownloadCertificate` | DownloadCertificateRequest | DownloadCertificateResponse | Download certificate |
| `DownloadBaseStationCertificate` | DownloadBaseStationCertificateRequest | DownloadCertificateResponse | Download base station certificate |
| `GenerateServerCertificates` | GenerateServerCertificatesRequest | GenerateServerCertificatesResponse | Generate server certificates |
| `RenewServerCertificates` | RenewServerCertificatesRequest | RenewServerCertificatesResponse | Renew server certificates |
| `GetServerCertificateStatus` | GetServerCertificateStatusRequest | GetServerCertificateStatusResponse | Certificate status |

### Manufacturers (5)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `CreateManufacturer` | CreateManufacturerRequest | CreateManufacturerResponse | Create manufacturer |
| `GetManufacturer` | GetManufacturerRequest | GetManufacturerResponse | Get manufacturer |
| `UpdateManufacturer` | UpdateManufacturerRequest | UpdateManufacturerResponse | Update manufacturer |
| `DeleteManufacturer` | DeleteManufacturerRequest | DeleteManufacturerResponse | Delete manufacturer |
| `ListManufacturers` | ListManufacturersRequest | ListManufacturersResponse | List manufacturers |

### Device Models (5)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `CreateDeviceModel` | CreateDeviceModelRequest | CreateDeviceModelResponse | Create device model |
| `GetDeviceModel` | GetDeviceModelRequest | GetDeviceModelResponse | Get device model |
| `UpdateDeviceModel` | UpdateDeviceModelRequest | UpdateDeviceModelResponse | Update device model |
| `DeleteDeviceModel` | DeleteDeviceModelRequest | DeleteDeviceModelResponse | Delete device model |
| `ListDeviceModels` | ListDeviceModelsRequest | ListDeviceModelsResponse | List device models |

### Blueprints (8)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `CreateBlueprint` | CreateBlueprintRequest | CreateBlueprintResponse | Create blueprint |
| `GetBlueprint` | GetBlueprintRequest | GetBlueprintResponse | Get blueprint |
| `UpdateBlueprint` | UpdateBlueprintRequest | UpdateBlueprintResponse | Update blueprint |
| `DeleteBlueprint` | DeleteBlueprintRequest | DeleteBlueprintResponse | Delete blueprint |
| `ListBlueprints` | ListBlueprintsRequest | ListBlueprintsResponse | List blueprints |
| `SetDefaultBlueprint` | SetDefaultBlueprintRequest | SetDefaultBlueprintResponse | Set default blueprint |
| `SubmitBlueprintToRegistry` | SubmitBlueprintToRegistryRequest | SubmitBlueprintToRegistryResponse | Submit to registry |
| `CreateDeviceModelWithBlueprint` | CreateDeviceModelWithBlueprintRequest | CreateDeviceModelWithBlueprintResponse | Create device model with default blueprint atomically |

### Blueprint Utilities (1)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `DecodePreview` | DecodePreviewRequest | DecodePreviewResponse | Preview blueprint payload decoding |

### Endpoint Stats (2)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `GetEndPointStats` | GetEndPointStatsRequest | GetEndPointStatsResponse | Get endpoint statistics |
| `GetEndPointOperations` | GetEndPointOperationsRequest | GetEndPointOperationsResponse | Get endpoint operation history |

### Global Coverage (1) — Admin Only

Cross-tenant RPCs for server administrators. These require authentication but are org-exempt (no `x-organization-id` header needed). Access is restricted to users with admin privileges.

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `ListAllBaseStationLocations` | ListAllBaseStationLocationsRequest | ListAllBaseStationLocationsResponse | Returns all base station locations across all tenants |

### Auth & Session (8) — Enterprise

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `Login` | LoginRequest | LoginResponse | Authenticate user |
| `RefreshTokens` | RefreshTokensRequest | RefreshTokensResponse | Refresh access token |
| `GetProfile` | GetProfileRequest | GetProfileResponse | Get current user profile |
| `GetAuthSettings` | GetAuthSettingsRequest | GetAuthSettingsResponse | Get auth configuration |
| `Logout` | LogoutRequest | LogoutResponse | Invalidate session |
| `ChangePassword` | ChangePasswordRequest | ChangePasswordResponse | Update password |
| `ExchangeOIDC` | ExchangeOIDCRequest | LoginResponse | Exchange OIDC code for tokens |
| `ExchangeOAuth2` | ExchangeOAuth2Request | LoginResponse | Exchange OAuth2 code for tokens |

### Self-Service Registration (1) — Enterprise

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `RegisterAccount` | RegisterAccountRequest | LoginResponse | Register new account and receive tokens |

### Users (6) — Enterprise

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `CreateUser` | CreateUserRequest | CreateUserResponse | Create user account |
| `GetUser` | GetUserRequest | GetUserResponse | Get user details |
| `UpdateUser` | UpdateUserRequest | UpdateUserResponse | Update user account |
| `DeleteUser` | DeleteUserRequest | DeleteUserResponse | Delete user account |
| `ListUsers` | ListUsersRequest | ListUsersResponse | List users |
| `UpdateUserPassword` | UpdateUserPasswordRequest | UpdateUserPasswordResponse | Update user password (admin) |

### Organizations (5) — Enterprise

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `CreateOrganization` | CreateOrganizationRequest | CreateOrganizationResponse | Create organization |
| `GetOrganization` | GetOrganizationRequest | GetOrganizationResponse | Get organization details |
| `UpdateOrganization` | UpdateOrganizationRequest | UpdateOrganizationResponse | Update organization |
| `DeleteOrganization` | DeleteOrganizationRequest | DeleteOrganizationResponse | Delete organization |
| `ListOrganizations` | ListOrganizationsRequest | ListOrganizationsResponse | List organizations |

### Organization Memberships (6) — Enterprise

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `AddOrganizationUser` | AddOrganizationUserRequest | AddOrganizationUserResponse | Add user to organization |
| `GetOrganizationUser` | GetOrganizationUserRequest | GetOrganizationUserResponse | Get membership details |
| `UpdateOrganizationUser` | UpdateOrganizationUserRequest | UpdateOrganizationUserResponse | Update membership |
| `RemoveOrganizationUser` | RemoveOrganizationUserRequest | RemoveOrganizationUserResponse | Remove from organization |
| `ListOrganizationUsers` | ListOrganizationUsersRequest | ListOrganizationUsersResponse | List organization members |
| `ListUserOrganizations` | ListUserOrganizationsRequest | ListUserOrganizationsResponse | List organizations a user belongs to |

## Streaming RPCs

Five RPCs use server-side streaming to deliver real-time data:

| RPC | Use Case |
|-----|----------|
| `StreamMessages` | Real-time uplink messages |
| `StreamBaseStationMessages` | Base station message feed |
| `StreamEvents` | System event notifications |
| `StreamBaseStationEvents` | Base station event feed |
| `StreamEndPointEvents` | Endpoint event feed |

```typescript
const stream = client.streamMessages(request, metadata);
stream.on('data', (message) => { /* handle message */ });
stream.on('error', (err) => { /* handle error */ });
stream.on('end', () => { /* stream closed */ });
```

## Pagination

All `List*` RPCs support offset-based pagination via `page_size` and `page_token` fields.

**Standard list RPCs:** default 20, max 100
**High-volume lists** (endpoints, base stations, downlinks): default 100, max 1000

```protobuf
message ListEndPointsRequest {
  int32 page_size = 1;    // Default 20, max 100
  string page_token = 2;  // From previous ListEndPointsResponse.next_page_token
}
```

## Error Handling

All errors use gRPC status codes with machine-readable error tokens.

### Error Response Format

```json
{
  "code": 3,
  "message": "KC-GRPC-ERR-001: invalid endpoint EUI format",
  "details": []
}
```

### Common Error Codes

| gRPC Code | HTTP Equiv | Common Causes |
|-----------|------------|---------------|
| `INVALID_ARGUMENT` (3) | 400 | Malformed request, validation failures |
| `NOT_FOUND` (5) | 404 | Resource does not exist |
| `ALREADY_EXISTS` (6) | 409 | Duplicate resource |
| `PERMISSION_DENIED` (7) | 403 | Authorization failure |
| `UNAUTHENTICATED` (16) | 401 | Missing or invalid token |
| `FAILED_PRECONDITION` (9) | 400 | State-based failure |
| `INTERNAL` (13) | 500 | Server error |

### Error Token Ranges

Error tokens follow the pattern `KC-GRPC-ERR-NNN`. The numeric ranges group errors by domain:

| Range | Domain |
|-------|--------|
| 001-099 | Core validation |
| 100-199 | Auth and session |
| 200-299 | Messaging |
| 300-399 | Base station |
| 400-419 | Integration and endpoint stats |
| 420-42F | Registry provider |

## Health Check

KC-Core implements the standard gRPC Health Checking Protocol:

```bash
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

## API Console

gRPC reflection is enabled on KC-Gateway. Compatible tools:

```bash
# grpcui — interactive web UI
grpcui -plaintext localhost:9090

# grpcurl — command-line client
grpcurl -plaintext localhost:9090 list

# Describe a service
grpcurl -plaintext localhost:9090 describe kilocenter.api.v1.KiloCenterService

# Call an RPC with auth headers
grpcurl -plaintext \
  -H "authorization: Bearer <token>" \
  -H "x-organization-id: <org-uuid>" \
  -H "x-user-id: <user-uuid>" \
  -d '{}' \
  localhost:9090 kilocenter.api.v1.KiloCenterService/GetSystemStatus

# Postman — use the gRPC tab with server reflection
# BloomRPC — desktop gRPC client
```

## Language Examples

Runnable code examples for common API calls (system status, endpoint listing, authentication):

- [Go Examples](04-go-examples.md)
- [Python Examples](05-python-examples.md)
- [JavaScript Examples](06-javascript-examples.md)
- [C# Examples](07-csharp-examples.md)

## Further Reading

- [gRPC First Steps](01-grpc-first-steps.md) — verify connectivity and discover methods
