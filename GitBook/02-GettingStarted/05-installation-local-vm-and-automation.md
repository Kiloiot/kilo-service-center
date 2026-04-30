# Installation: Local VM and Automation

## Goal

Create repeatable local environments for testing and team onboarding.

## Local VM Pattern

Provision a Linux VM, clone the repository, and follow the steps from [Installation: Docker Compose](03-installation-docker-compose.md). Docker Compose is the recommended approach for VM-based setups since it handles all services — including KC-Web — automatically.

```bash
cd KiloServiceCenter
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
