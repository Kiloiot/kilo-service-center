# Getting Started

## Scope

This section covers architecture, prerequisites, installation, and baseline configuration for KiloCenter deployments.

## Recommended Path

For most users evaluating KiloCenter locally:

1. Copy `.env.example` to `.env`.
2. Start the full stack: `docker compose up --build -d`
   (TLS certificates are generated automatically on first start.)
3. Open `http://localhost/` in your browser.

## Installation Paths

| Mode | Guide | TLS Certificates |
|------|-------|-----------------|
| Docker Compose | [Installation: Docker Compose](03-installation-docker-compose.md) | Auto-generated on first start |
| Linux Host (source) | [Installation: Linux Host](04-installation-linux-host.md) | Manual (`KC-Core/certgen`) |
| Local VM | [Installation: Local VM](05-installation-local-vm-and-automation.md) | Auto-generated (Docker) |
| Kubernetes (Helm) | [Installation: Kubernetes](07-installation-kubernetes.md) | cert-manager or manual |

## Pages

1. [Architecture and Components](01-architecture-and-components.md)
2. [Prerequisites](02-prerequisites.md)
3. [Installation: Docker Compose](03-installation-docker-compose.md)
4. [Installation: Linux Host](04-installation-linux-host.md)
5. [Installation: Local VM and Automation](05-installation-local-vm-and-automation.md)
6. [Configuration Basics](06-configuration-basics.md)
7. [Installation: Kubernetes (Helm)](07-installation-kubernetes.md)

## After Setup

Once your services are running, continue to:

- [Onboarding](../03-Onboarding/README.md) -- connect base stations and register endpoints
- [Integrations](../04-Integrations/README.md) -- set up gRPC and MQTT consumers
- [Security](../05-Security/README.md) -- TLS and access control
- [Operations](../06-Operations/README.md) -- monitoring and troubleshooting
