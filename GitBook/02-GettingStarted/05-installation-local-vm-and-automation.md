# Installation: Local VM and Automation

## Before starting an installation

Use these instructions only in an isolated evaluation environment with test data. The current
examples contain published credentials and signing-key values, and some ports are reachable beyond
the host unless your network blocks them. Changing only the administrator password does not correct
all of these defaults. Do not expose the example installation to the Internet or use it for customer
data. Read the [installation safety notice](../05-Security/02-installation-safety.md) before running commands.


## Goal

Create repeatable local environments for testing and team onboarding.

## Local VM Pattern

Provision a Linux VM, clone the repository, and follow the steps from [Installation: Docker Compose](03-installation-docker-compose.md). Docker Compose is the recommended approach for VM-based setups since it handles all services — including KC-Web — automatically.

```bash
cd kilo-service-center
cp .env.example .env
docker compose up --build -d
```

Certificates are generated automatically on the first `docker compose up`.

Open `http://localhost/` to verify KC-Web is running.

## What to Keep Consistent

Keep these stable across VM environments:

- Docker Compose service set (`postgres`, `redis`, `mosquitto`, `certgen`, `kc-identity`, `kilocenter`, `kc-gateway`, `kc-web`)
- TLS certificate generation (automatic on first `docker compose up` -- see [Docker Compose Installation](03-installation-docker-compose.md))
- Port mappings and health check endpoints

## Why Use This Mode

- Clean-room validation from a fresh machine
- Repeatable onboarding for new team members
- Pre-production environment rehearsals
