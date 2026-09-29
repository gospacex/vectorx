#!/usr/bin/env bash
# Stop Elasticsearch environment
set -euo pipefail
cd "$(dirname "$0")/.."
echo "Stopping Elasticsearch..."
docker compose down
echo "Done."
