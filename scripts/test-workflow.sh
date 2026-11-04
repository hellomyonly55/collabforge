#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
# A fresh independent stack keeps the full three-account journey within real registration limits.
exec ./scripts/test-e2e.sh --config=playwright.release.config.ts "$@"
