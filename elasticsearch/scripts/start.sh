#!/usr/bin/env bash
# Start Elasticsearch environment
set -euo pipefail
cd "$(dirname "$0")/.."
echo "Starting Elasticsearch..."
docker compose up -d
echo "Waiting for Elasticsearch to be healthy..."
for i in $(seq 1 60); do
  if curl -sf http://localhost:9200/_cluster/health >/dev/null 2>&1; then
    echo "Elasticsearch is ready on :9200"
    exit 0
  fi
  sleep 2
done
echo "ERROR: Elasticsearch failed to become healthy within 120s"
exit 1
