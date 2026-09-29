#!/usr/bin/env bash
# Start Milvus standalone environment
set -euo pipefail
cd "$(dirname "$0")/.."
echo "Starting Milvus + etcd + minio..."
docker compose up -d
echo "Waiting for Milvus to be healthy..."
for i in $(seq 1 30); do
  if curl -sf http://localhost:9091/healthz >/dev/null 2>&1; then
    echo "Milvus is ready on :19530"
    exit 0
  fi
  sleep 2
done
echo "ERROR: Milvus failed to become healthy within 60s"
exit 1
