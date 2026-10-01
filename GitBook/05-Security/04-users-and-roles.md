# KiloCenter User Roles and Permissions

Every signed-in user acts with a set of roles. The roles decide which pages
KC-Web shows and which API calls the service center accepts; KC-Web hides what
your roles do not allow, and the service refuses it with `PERMISSION_DENIED`
even when a call is made directly through the API.

## What roles does KiloCenter have?

| Role | Setting in Users & Roles | Intended for |
|------|--------------------------|--------------|
| Admin | **Is admin** | Whoever runs the installation. Holds every other role as well. |
| Tenant Manager | **Is tenant manager** | Managing the members of your organizations (Enterprise Edition). |
| Base Station Manager | **Is base station manager** | Registering and operating base stations. |
| Endpoint Manager | **Is endpoint manager** | Registering and operating endpoints and their data. |

A user can hold any combination. Holding a role gives access only to your own
organization's data; roles never open another tenant's base stations,
endpoints or events.

## What can each role see and do?

| Area | Admin | Tenant Manager | Base Station Manager | Endpoint Manager |
|------|:-----:|:--------------:|:--------------------:|:----------------:|
| Dashboard counts, message traffic and service health | Yes | Yes | Yes | Yes |
| Dashboard alerts, with the link to the Events Log | Yes | -- | -- | -- |
| Base station map on the Base Stations page (every located base station on the server) | Yes | -- | -- | -- |
| Base stations: list, view, register, edit, delete, status and ping | Yes | -- | Yes | -- |
| Base station traffic and base station events | Yes | -- | Yes | -- |
| Base station certificates and private keys | Yes | -- | Yes | -- |
| Server certificate status | Yes | -- | Yes | -- |
| Generate or renew the server certificate | Yes | -- | -- | -- |
| Endpoints: list, view, register, edit, delete, attach and detach | Yes | -- | -- | Yes |
| Endpoint network and application keys: see whether they are set | Yes | -- | -- | Yes |
| Endpoint network and application keys: reveal them (recorded in the Audit Log) | Yes | -- | -- | Yes |
| Endpoint application key: remove it (recorded in the Audit Log) | Yes | -- | -- | Yes |
| Endpoint messages, events and downlinks | Yes | -- | -- | Yes |
| Traffic: uplinks, downlink queue and results | Yes | -- | -- | Yes |
| A base station's Traffic tab: the uplinks that station heard | Yes | -- | Yes | Yes |
| A base station's Traffic tab: the downlinks it holds and their results | Yes | -- | -- | Yes |
| Logs: Events Log and Errors Center (only the event categories your roles cover) | Yes | -- | Yes | Yes |
| Logs: Audit Log | Yes | -- | -- | -- |
| Application Center (SCACI) sessions | Yes | -- | -- | Yes |
| Integrations (data forwarding to external systems) | Yes | -- | -- | -- |
| Blueprints, manufacturers and device models of your organization | Yes | -- | -- | Yes |
| System catalog entries (shared by every organization) | Yes | -- | -- | -- |
| Organization members (Enterprise Edition) | Yes | Yes | -- | -- |
| Users, organizations and API keys of the whole installation | Yes | -- | -- | -- |
| Security, system and audit events, alerts, diagnostics | Yes | -- | -- | -- |

Event lists follow the same rule: each user sees only the event categories
their roles cover. Base station managers see base station and BSSCI events,
endpoint managers see endpoint, message, SCACI and roaming events, both see
protocol and session events, and only administrators see security, system,
audit and error events such as failed sign-ins, refused requests or service
starts and stops. The Activity table on a base station or endpoint page
applies the same filter. A view your roles do not cover shows "No access to
this view" instead of its data.

A few event types are worth naming. `scaci.session_opened` and
`scaci.session_closed` (an Application Center connecting and leaving) and
`dl_data_acknowledged` (an endpoint confirming a downlink) are endpoint
manager events, so a Base Station Manager does not see them, not even in the
Activity table of a base station that carried the downlink.
`scaci.connect_refused` (an Application Center turned away) depends on
whether the refused Application Center belongs to a tenant: if it does, the
event is a SCACI event of that tenant, so its endpoint managers see their own
Application Center being refused; if it cannot be tied to any tenant, it is a
security event of the platform tenant, which only administrators see.
`endpoint.keys_revealed` and `endpoint.keys_removed` (someone revealing or
removing an endpoint key) and `certificate.private_key_downloaded` (someone
downloading a base station's private key) are audit events, so only
administrators see them.

## Which roles can read the control plane, statistics and integrations through the API?

The Application Center control plane, statistics and analytics, integrations,
capabilities and diagnostics are API calls for the software that connects to
the service center; the web interface has no screen for them. The dashboard
shows the Application Center status and the statistics. The service checks
each call on its own:

- **Any role** may read statistics, analytics and capabilities.
- **Endpoint Managers** may read the Application Center (SCACI) status,
  sessions, queues, errors and statistics, which the dashboard shows them, and
  the downlink reception status and endpoint statistics.
- **Base Station Managers** may read base station availability, request a
  station's status, ping it and download its certificate.
- **Only administrators** may manage integrations, download the diagnostics
  bundle, read alerts and read the locations of every base station of the
  installation.

[Which roles does each API call require?](#which-roles-does-each-api-call-require)
lists every call.

Installation-wide events, such as users being created or deleted,
organizations or API keys being deleted, and server certificates being
generated or renewed, are recorded under the platform tenant (see
[What Is the Platform Tenant?](../02-GettingStarted/06-configuration-basics.md#what-is-the-platform-tenant)).
They are audit events, so only administrators can read them, even for members
whose organization belongs to that same tenant.

## Who can read endpoint keys and base station certificates?

Key material is returned only to the role that manages it:

- The **network session key** and **application key** of an endpoint are
  **masked** wherever the endpoint is shown or returned: in lists, on the
  detail and configuration pages, and in the edit dialog. You see only whether
  each key is set. An Endpoint Manager or Admin can **reveal** a key on
  request; every reveal is recorded in the Audit Log with who revealed which
  key of which endpoint, never the key itself. The application key can also be
  **removed** from the edit dialog, after a confirmation, when you save; the
  removal is recorded in the Audit Log the same way. The network session key
  is mandatory, so it can only be replaced.
- A base station's **client certificate and private key** can be downloaded
  only by Base Station Managers and Admins, and a base station registered by
  someone without that role is refused, so no private key is ever handed out
  to them.
- A base station's **private key leaves the service center once**: after the
  first download, from the download links shown when the certificate is issued
  or from the base station itself, it is no longer stored. Issue a new
  certificate if the key is lost. Every download is recorded in the Audit Log
  with who downloaded the key of which base station, never the key itself.
- **No secret leaves unrecorded.** Revealing an endpoint key, downloading a
  base station's private key and creating an API key fail with an internal
  error, without the secret, when the Audit Log cannot record them. Try again
  once the database is available: the private key is still stored, and the
  refused API key is unusable because nobody received it.
- **Integration settings are write-only.** They may hold passwords or access
  tokens, so reading an integration back names the settings it has, each shown
  as `configured`, never their values. Send the full settings again to change
  them.

## What does a new account see after self-registration?

Self-registration creates an account with no roles. The new user can sign in,
change their password and sign out, and every other page shows:

> **You don't have access yet** -- Your account is ready, but no roles have
> been granted to it. Ask an administrator to give you access in Users & Roles.

The page updates on its own once an administrator grants a role; the user does
not need to sign in again.

Self-registration is controlled by `registration_enabled` in the identity
configuration (`auth.registration_enabled`, or the environment variable
`KILOCENTER_AUTH_REGISTRATION_ENABLED`). It is on in the shipped Docker Compose
configuration and off by default everywhere else, including the Helm chart.
Accounts created by an OpenID Connect or OAuth2 sign-in start without roles in
the same way.

## What happens to existing accounts when I upgrade?

Nobody loses access. Before this release every signed-in user could read and
change every base station and endpoint of their organization, so the upgrade
switches on **Base Station Manager** and **Endpoint Manager** for every account
and every active organization membership that exists when it runs. Members who
could already manage their organization's members keep the Tenant Manager role
they had. The upgrade never makes anyone an administrator, and accounts created
after it start without roles. Review the roles in **Users & Roles** after the
upgrade and switch off what people do not need.

## How do I give a user access?

1. Sign in as an administrator and open **Users & Roles**.
2. Open the user and switch on the roles they need.
3. Save. The change applies within about 30 seconds, without the user signing
   in again.

Removing a role takes effect in the same time. Switching off **Is active**
removes every role at once.

## How long does a role change take to apply?

The service center reads a user's roles from the identity service and keeps
them for `grpc.rbac_role_cache_ttl_seconds` (30 seconds by default). A change
applies to new requests within that time. KC-Web refreshes the roles it shows
at the same interval and immediately after a request is refused.

Live streams that are already open follow the same rule. Before a stream
delivers each update, the service center checks the caller's roles again. When
the roles have changed since the stream opened, or the identity service cannot
answer, the service center ends the stream instead of delivering the update.
KC-Web then reopens the streams the new roles allow. For example, an administrator
who is downgraded to Endpoint Manager stops receiving security and system events
within the same time as any other request.

## How do organization roles combine with user roles in the Enterprise Edition?

In the Enterprise Edition each organization membership carries its own role
switches, named after the role each grants in that organization: **Tenant
Manager**, **Base Station Manager** and **Endpoint Manager**. The users table
shows the same three columns. While you act in an organization, you hold a
role if your user account has it **or** your active membership in that
organization grants it.

Membership switches never make a user an administrator. The owner of an
organization created through self-registration in the Enterprise Edition holds
all three switches for that organization. The same rule applies in the
Community Edition, where every user is a member of the single default
organization; a self-registered member starts with none of the switches.

A Tenant Manager can add, change and remove members of the organizations of
their own tenant, including their membership switches. User accounts,
organizations and API keys of the whole installation stay with administrators.

## Do API keys have roles?

A **user API key** acts with the roles of the user it belongs to.

A **service-account API key** belongs to an organization, not to a user. It
acts as a **Base Station Manager** and **Endpoint Manager** in that
organization, so an integration can read and change the organization's base
stations and endpoints, including their keys and certificates. It is never an
Admin or Tenant Manager, and it holds no role in any other organization: calls
for another organization are refused. Revoking or letting the key expire ends
its access at once.

## Which roles does each API call require?

Sign-in, registration, `GetAuthSettings`, `GetReleaseInfo`, `GetCEStatus` and
`CompleteCEOnboarding` need no role. `GetProfile`, `Logout` and
`ChangePassword` need only a signed-in user. Every other call needs one of the
roles below; an administrator may make every call.

| Required role | Calls |
|---------------|-------|
| Endpoint Manager | `CreateEndPoint`, `GetEndPoint`, `UpdateEndPoint`, `DeleteEndPoint`, `ListEndPoints`, `AttachEndPoint`, `DetachEndPoint`, `GetEndPointStats`, `GetEndPointOperations`, `ListEndpointMessages`, `ListEndpointActivity`, `GetMessage`, `ListMessages`, `StreamMessages`, `SendDownlink`, `RevokeDownlink`, `ListDownlinkQueue`, `GetDownlinkResults`, `UpdatePendingDownlink`, `SendULTransmit`, `GetDLRXStatus`, `QueryDLRXStatus`, `GetDLRXStatusQueries`, `ListScaciSessions`, `GetScaciSession`, `GetScaciStatistics`, `ListScaciErrors`, `ListScaciQueues`, `GetScaciStatus`, `CreateManufacturer`, `GetManufacturer`, `UpdateManufacturer`, `DeleteManufacturer`, `ListManufacturers`, `CreateDeviceModel`, `GetDeviceModel`, `UpdateDeviceModel`, `DeleteDeviceModel`, `ListDeviceModels`, `CreateBlueprint`, `GetBlueprint`, `UpdateBlueprint`, `DeleteBlueprint`, `ListBlueprints`, `SetDefaultBlueprint`, `SubmitBlueprintToRegistry`, `BulkAssignBlueprint`, `CreateDeviceModelWithBlueprint`, `DecodePreview` |
| Base Station Manager | `CreateBaseStation`, `GetBaseStation`, `UpdateBaseStation`, `DeleteBaseStation`, `ListBaseStations`, `GetBaseStationStats`, `UpdateBaseStationEui`, `GetBaseStationAvailability`, `GetBaseStationMessagesReceived`, `RequestBaseStationStatus`, `InitiatePing`, `ListBaseStationActivity`, `ListBaseStationMessages`, `GetBaseStationMessage`, `GetBaseStationMessageStats`, `SearchBaseStationMessages`, `ExportBaseStationMessages`, `StreamBaseStationMessages`, `GenerateCertificate`, `DownloadCertificate`, `DownloadBaseStationCertificate`, `GetServerCertificateStatus` |
| Any role | `GetSystemStatus`, `GetStatistics`, `GetAnalyticsOverview`, `GetActivityAnalytics`, `GetSignalQualityAnalytics`, `ListCapabilities`, `ListEvents`, `StreamEvents`, `ListErrorGroups` (each list shows only the categories your roles cover) |
| Tenant Manager | `AddOrganizationUser`, `GetOrganizationUser`, `UpdateOrganizationUser`, `RemoveOrganizationUser`, `ListOrganizationUsers` (organizations of your own tenant) |
| Admin | `GetDiagnosticsBundle`, `ListAlerts`, `GetAlertSummary`, `GenerateServerCertificates`, `RenewServerCertificates`, `ListAllBaseStationLocations`, `ListCEInstances`, `RevokeCEInstance`, `CreateIntegration`, `GetIntegration`, `UpdateIntegration`, `DeleteIntegration`, `ListIntegrations`, `CreateUser`, `GetUser`, `UpdateUser`, `DeleteUser`, `ListUsers`, `UpdateUserPassword`, `CreateOrganization`, `GetOrganization`, `UpdateOrganization`, `DeleteOrganization`, `ListOrganizations`, `ListUserOrganizations`, `CreateApiKey`, `GetApiKey`, `DeleteApiKey`, `ListApiKeys` |

Creating, changing or deleting a **system** catalog entry (a manufacturer,
device model or blueprint shared by every organization) also needs the Admin
role, even though the call itself is open to Endpoint Managers.

## Can KiloCenter run without sign-in?

No. Apart from the calls listed above as needing no role, every call needs a
signed-in user who holds a role, and a service center configured without an
identity service refuses to start. Keep authentication enabled on KC-Gateway
and KC-Identity, as the shipped configuration does.
