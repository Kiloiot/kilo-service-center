# KiloCenter

KiloCenter is an open-source MIOTY Service Center which can be used to set up and manage MIOTY LPWAN networks. KiloCenter provides a web interface for the management of base stations, endpoints, and network traffic as well as data integrations for routing endpoint data to external systems, databases, and cloud platforms. KiloCenter provides a gRPC-based API that can be used to integrate with or extend KiloCenter, and supports MQTT for real-time event streaming.

## Editions

| Edition | Status | Description |
|---------|--------|-------------|
| **Community** | Available | Full service center engine, open source, self-hosted. Includes KC-Core, KC-Gateway, KC-Web, gRPC API, API key management, and MQTT integration. |
| **Enterprise** | Planned | Adds user authentication, organization management, multi-tenancy, tenant isolation, and extended MIOTY endpoint profile fields. |
| **Cloud** | Planned | Fully managed hosting with SLA-backed operations. |

Community Edition is licensed under [AGPL-3.0-or-later](https://www.gnu.org/licenses/agpl-3.0.html). Source at [github.com/Kiloiot/kilo-service-center](https://github.com/Kiloiot/kilo-service-center).

## New to MIOTY?

If you are unfamiliar with MIOTY, start with [What is MIOTY?](what-is-mioty.md) for an overview of the protocol, its terminology, and how KiloCenter fits in.

## Documentation

- [Project Overview](01-Overview/01-project-overview.md) -- editions, capabilities, and data flow
- [Getting Started](02-GettingStarted/README.md) -- architecture, prerequisites, and installation
- [Onboarding](03-Onboarding/README.md) -- connect a base station and register your first endpoint
- [Integrations](04-Integrations/README.md) -- gRPC API and MQTT setup
- [Security](05-Security/README.md) -- TLS, credentials, and hardening
- [Operations](06-Operations/README.md) -- monitoring and troubleshooting
