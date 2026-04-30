# What is MIOTY?

MIOTY is a low-power wide-area network (LPWAN) protocol developed by the Fraunhofer Institute for Integrated Circuits (IIS) and standardized through the MIOTY Alliance. It is designed for large-scale IoT deployments where reliability and range matter more than bandwidth.

## How MIOTY Differs from Other LPWAN Technologies

MIOTY uses **telegram splitting** -- each message is divided into small sub-packets and transmitted across multiple time slots and frequencies. The base station reassembles the full message from these sub-packets. This makes MIOTY significantly more resistant to interference and collisions compared to protocols that transmit a single continuous packet.

The result is reliable communication even in dense deployments with thousands of endpoints sharing the same spectrum.

## MIOTY Terminology

If you have worked with other IoT protocols, these mappings will help:

| MIOTY Term | Common Equivalent | Description |
|------------|-------------------|-------------|
| **Endpoint** | Device / sensor | The field device that transmits data over the air |
| **Base Station** | Gateway | Receives radio frames from endpoints and forwards them to the service center |
| **Service Center** | Network server | Manages base stations, processes messages, and routes data to applications |
| **Uplink** | Device-to-cloud | Data sent from an endpoint through a base station to the service center |
| **Downlink** | Cloud-to-device | Commands sent from the service center through a base station to an endpoint |

## Protocol Interfaces

KiloCenter implements two MIOTY protocol interfaces:

- **BSSCI** (Base Station Service Center Interface) -- the TCP/TLS protocol between base stations and the service center. Base stations connect to KiloCenter over BSSCI to forward uplink data and receive downlink commands.

- **SCACI** (Service Center Application Center Interface) -- the TCP/TLS protocol between the service center and application centers. External applications connect to KiloCenter over SCACI for direct protocol-level integration.

## Data Flow

```text
Endpoint  ──(radio)──>  Base Station  ──(BSSCI/TLS)──>  KiloCenter  ──(gRPC/MQTT)──>  Application
                                                              │
                                                         KC-Web UI
```

1. Endpoints transmit data over the MIOTY radio protocol.
2. Base stations receive the radio frames and forward them to KiloCenter over BSSCI (TCP with TLS).
3. KiloCenter processes the protocol frames, validates sessions, and persists messages.
4. Applications consume data through the gRPC API or MQTT, and operators monitor the network through KC-Web.

## Further Reading

- [MIOTY Alliance](https://mioty-alliance.com/) -- protocol specifications and ecosystem
- [Project Overview](01-Overview/01-project-overview.md) -- what KiloCenter does with these protocols
- [Getting Started](02-GettingStarted/README.md) -- install and run KiloCenter
