#!/usr/bin/env bash
# Verify Elasticsearch is reachable
set -euo pipefail
echo "Checking Elasticsearch health..."
if curl -sf http://localhost:9200/_cluster/health >/dev/null 2>&1; then
  echo "OK: Elasticsearch is healthy on :9200"
else
  echo "FAIL: Elasticsearch not reachable on :9200"
  exit 1
fi
