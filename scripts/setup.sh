#!/bin/bash
# One-time local setup for the HydraDNS monorepo.
# core, ui, scanner, and cli live under apps/ in this repo (no submodules), so
# a plain clone already has everything needed to run the stack. (The landing
# marketing site lives in its own hydradns/hydradns-landing repo, not here.)
# This script just prepares a local .env.
set -e

if [ ! -f .env ] && [ -f .env.example ]; then
  cp .env.example .env
  echo "Created .env from .env.example (edit it if you need non-default paths)."
fi

echo "Setup complete. Start the stack with: docker compose up -d"
