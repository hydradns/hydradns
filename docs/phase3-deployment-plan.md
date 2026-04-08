# Phase 3: Plug-and-Play Deployment Plan

**Date:** 2026-04-07
**Status:** In Progress

## Key Findings

The current docker-compose is broken — no dataplane service, no config mounting, port mismatches, missing Dockerfiles for UI/scanner/landing. The architecture splits controlplane + dataplane into separate processes sharing the same SQLite file, which is problematic across containers.

## Architecture Decision: Combined Core Container

Instead of separate controlplane + dataplane containers (which would fight over SQLite), combine them into a **single `core` container** that runs both processes. This:
- Avoids SQLite sharing across containers (same filesystem)
- Simplifies deployment (one container = one service)
- Matches Pi target (fewer moving parts)
- gRPC communicates over localhost:50051 inside the container

## Fixes Needed

### 1. Create combined core Dockerfile
- Multi-stage Go build: compile both binaries
- Copy configs into image
- Create data directory with correct permissions
- Entrypoint script that starts dataplane (background) then controlplane (foreground)

### 2. Fix config.yaml
- Control plane listen_addr: 8086 → 8080 (matches compose port mapping)
- gRPC listen_addr: "dataplane:50051" → "localhost:50051" (same container)

### 3. Update docker-compose.yml
- Single `core` service with combined Dockerfile
- Persistent volume for `/app/data`
- Config mounted or baked in
- DNS port exposed (1053 mapped to 53 for real DNS, or keep 1053 for dev)

### 4. Create UI Dockerfile
- Multi-stage: install deps → build → serve with standalone Next.js

### 5. Create scanner Dockerfile
- Multi-stage Go build

### 6. Make CORS configurable
- Read allowed origins from env var, fall back to localhost:3000

## Execution Order

1. Fix config.yaml port mismatch
2. Make CORS configurable via env var
3. Create combined core Dockerfile with entrypoint script
4. Create UI Dockerfile
5. Create scanner Dockerfile (minimal, service is simple)
6. Update docker-compose.yml with all fixes
7. Build and test with docker compose
