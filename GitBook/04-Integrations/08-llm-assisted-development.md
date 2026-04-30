# LLM-Assisted Development

## Overview

KiloCenter ships with machine-readable project summaries designed for use with Large Language Models (LLMs) such as ChatGPT, Claude, Copilot, Cursor, and other AI coding assistants. These files give an LLM enough context about the project to generate working integration code, answer architecture questions, and help debug issues without reading the entire codebase.

## Files

Two files are provided at the repository root:

| File | Size | Purpose |
|------|------|---------|
| [`llms.txt`](../../llms.txt) | ~100 lines | Concise project summary — architecture, deployment, API overview, key directories |
| [`llms-full.txt`](../../llms-full.txt) | ~350 lines | Comprehensive reference — all RPC domains, MQTT topics, protocol details, config parameters, security, onboarding |

## How to Use

### With Chat-Based LLMs (ChatGPT, Claude)

Paste the contents of `llms.txt` (or `llms-full.txt` for deeper questions) at the start of your conversation:

```
Here is context about the KiloCenter project I'm working with:

<paste contents of llms.txt>

Now help me write a Python client that streams uplink messages.
```

### With IDE Assistants (Copilot, Cursor, Cline)

Most AI-powered IDEs can pick up context files automatically. Add `llms-full.txt` to your project root or reference it in your assistant's context configuration. For example, in Cursor you can add it to `.cursorrules` or reference it directly in a prompt.

### With Claude Code

Clone the repository and run Claude Code from the project root. It will automatically discover `llms.txt` and `llms-full.txt` for project context.

## What the LLM Can Help With

With `llms.txt` loaded, an LLM can:

- **Generate gRPC client code** in any language using the proto definitions and API reference
- **Write MQTT consumers** with correct topic patterns and payload formats
- **Create Helm values overrides** for your specific infrastructure
- **Debug connectivity issues** using the port map and health check endpoints
- **Explain MIOTY protocol concepts** like BSSCI three-way handshakes, EUIs, and session keys
- **Build Docker Compose variations** for different deployment scenarios

## Example Prompts

**Generate a Go gRPC client:**
```
Using the KiloCenter gRPC API on port 9090, write a Go program that
lists all base stations and prints their EUI and online status.
```

**Write an MQTT uplink consumer:**
```
Write a Python script that subscribes to all uplink messages from
KiloCenter's MQTT broker and logs the endpoint EUI, payload hex,
and RSSI for each message.
```

**Debug a deployment:**
```
My KC-Core pod is returning 503 on the readiness probe at /health/ready.
KC-Identity and PostgreSQL are running. What should I check?
```

**Create a Helm override:**
```
I have PostgreSQL at db.internal:5432 and Redis at cache.internal:6379.
Generate a minimal Helm values file for a production deployment with
ingress on kilocenter.mycompany.com using nginx ingress controller.
```

## Keeping Context Current

The `llms.txt` and `llms-full.txt` files are maintained alongside the documentation. When the API surface, deployment model, or architecture changes, these files are updated to reflect the current state. Always use the version from the branch you are working with.
