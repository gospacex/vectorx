#!/usr/bin/env bash
# Stop Qdrant environment
set -euo pipefail
cd "$(dirname "$0")/.."
echo "Stopping Qdrant..."
docker compose down
echo "Done."
