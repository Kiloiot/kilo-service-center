# Project Overview

## What KiloCenter Is

KiloCenter is an open-source MIOTY Service Center platform for running MIOTY LPWAN networks. It handles base station communication, endpoint management, uplink and downlink data processing, and exposes data through gRPC and MQTT interfaces.

## What You Can Do With KiloCenter

- Connect MIOTY base stations over the BSSCI protocol (TCP/TLS)
- Register and manage MIOTY endpoints
- Process uplink data from endpoints in real time
- Send downlink commands to bidirectional endpoints
- Integrate with external systems through gRPC and MQTT
- Monitor network health and endpoint status through the web interface

## Editions

### Community Edition

The Community Edition is fully open-source and self-hosted. It includes:

Licensed under AGPL-3.0-or-later. See [Licensing and Trademarks](03-licensing-and-trademarks.md).

- KC-Core service center engine (BSSCI and SCACI protocol handling)
- KC-Gateway for external gRPC-web API access
- KC-Web operator interface
- gRPC API for programmatic access
- API key management
- MQTT integration for real-time data streaming
- PostgreSQL-backed persistence with Redis caching

### Enterprise Edition

The Enterprise Edition extends the Community Edition with:

- Organization management and multi-tenancy
- User authentication and role-based access
- Tenant isolation for base stations and endpoints
- Extended MIOTY endpoint profile fields (dual channel mode, DL repetition, wide carrier offset, long interblock distance)

### KiloCenter Cloud

A planned managed hosting option. Not yet available.

## How MIOTY Data Flows Through KiloCenter

1. MIOTY endpoints transmit data over the air using the MIOTY radio protocol.
2. Base stations receive the radio frames and forward them to KiloCenter over BSSCI (TCP with TLS).
3. KC-Core processes the protocol frames, validates sessions, and persists messages.
4. KC-Gateway exposes the gRPC-web API, proxying requests to KC-Core.
5. Operators view data through KC-Web (which connects via KC-Gateway). External systems consume data through gRPC (via KC-Gateway) or MQTT.
