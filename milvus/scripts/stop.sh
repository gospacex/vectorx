#!/usr/bin/env bash
# Stop Milvus environment
set -euo pipefail
cd "$(dirname "$0")/.."
echo "Stopping Milvus..."
docker compose down
echo "Done."
