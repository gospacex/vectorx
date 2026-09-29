#!/bin/bash
# pgvector 初始化脚本：在容器启动时自动安装 vector 扩展
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE EXTENSION IF NOT EXISTS vector;
    COMMENT ON EXTENSION vector IS 'pgvector: open-source vector similarity search';
EOSQL

echo "✅ pgvector extension installed successfully"
