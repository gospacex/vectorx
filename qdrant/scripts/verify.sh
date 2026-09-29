#!/usr/bin/env bash
# Verify Qdrant is reachable
set -euo pipefail
echo "Checking Qdrant health..."
if curl -sf http://localhost:6333/healthz >/dev/null 2>&1; then
  echo "OK: Qdrant is healthy on :6333"
else
  echo "FAIL: Qdrant not reachable on :6333"
  exit 1
fi
