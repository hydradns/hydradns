#!/bin/bash
# One-time local setup for the HydraDNS monorepo.
# The five services live under apps/ in this repo (no submodules), so a plain
# clone already has everything. This script just prepares a local .env.
set -e

if [ ! -f .env ] && [ -f .env.example ]; then
  cp .env.example .env
  echo "Created .env from .env.example (edit it if you need non-default paths)."
fi

echo "Setup complete. Start the stack with: docker compose up -d"
