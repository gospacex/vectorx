#!/usr/bin/env bash
# Start Qdrant standalone environment
set -euo pipefail
cd "$(dirname "$0")/.."
echo "Starting Qdrant..."
docker compose up -d
echo "Waiting for Qdrant to be healthy..."
for i in $(seq 1 30); do
  if curl -sf http://localhost:6333/healthz >/dev/null 2>&1; then
    echo "Qdrant is ready on :6333 (HTTP) / :6334 (gRPC)"
    exit 0
  fi
  sleep 2
done
echo "ERROR: Qdrant failed to become healthy within 60s"
exit 1
