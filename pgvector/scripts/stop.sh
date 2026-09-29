#!/usr/bin/env bash
# Stop PostgreSQL + pgvector environment
set -euo pipefail
cd "$(dirname "$0")/.."
echo "Stopping PostgreSQL + pgvector..."
docker compose down
echo "Done."
