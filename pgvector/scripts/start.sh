#!/usr/bin/env bash
# Start PostgreSQL + pgvector environment
set -euo pipefail
cd "$(dirname "$0")/.."
echo "Starting PostgreSQL + pgvector..."
docker compose up -d
echo "Waiting for PostgreSQL to be healthy..."
for i in $(seq 1 30); do
  if docker compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1; then
    echo "PostgreSQL + pgvector is ready on :5432"
    exit 0
  fi
  sleep 2
done
echo "ERROR: PostgreSQL failed to become healthy within 60s"
exit 1
