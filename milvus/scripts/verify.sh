#!/usr/bin/env bash
# Verify Milvus is reachable
set -euo pipefail
echo "Checking Milvus health..."
if curl -sf http://localhost:9091/healthz >/dev/null 2>&1; then
  echo "OK: Milvus is healthy on :9091"
else
  echo "FAIL: Milvus not reachable on :9091"
  exit 1
fi
