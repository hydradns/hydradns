#!/bin/bash
set -e

echo "Creating apps directory..."
mkdir -p apps

echo "Initializing submodules..."
git submodule init
git submodule update

echo "Adding hydra-core submodule..."
git submodule add https://github.com/hydradns/hydra-core apps/core

echo "Adding hydra-ui submodule..."
git submodule add https://github.com/hydradns/hydra-ui apps/ui

echo "Adding scanner submodule..."
git submodule add https://github.com/hydradns/scanner apps/scanner

echo "Adding landing page submodule..."
git submodule add https://github.com/hydradns/hydradns-landing apps/landing

# CLI submodule — uncomment when the repo is available
# echo "Adding CLI submodule..."
# git submodule add https://github.com/hydradns/hydra-cli apps/cli

echo "Setup complete!"
