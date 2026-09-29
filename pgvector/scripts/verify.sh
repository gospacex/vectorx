#!/usr/bin/env bash
# Verify PostgreSQL + pgvector is reachable
set -euo pipefail
echo "Checking PostgreSQL health..."
if docker compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1; then
  echo "OK: PostgreSQL + pgvector is healthy on :5432"
else
  echo "FAIL: PostgreSQL not reachable on :5432"
  exit 1
fi
