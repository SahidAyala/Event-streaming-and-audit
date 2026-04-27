#!/usr/bin/env bash
set -e

echo "🚀 Starting infrastructure..."
docker compose up -d kafka postgres minio elasticsearch kibana kafka-ui kafka-init

echo ""
echo "🔍 Waiting for services..."

# Elasticsearch
./scripts/wait-for.sh "Elasticsearch" "http://localhost:9200"

# MinIO
./scripts/wait-for.sh "MinIO" "http://localhost:9000/minio/health/live"

# Kafka (simple check con nc)
echo "⏳ Waiting for Kafka..."
for i in {1..20}; do
  nc -z localhost 9094 && echo "✅ Kafka ready" && break
  echo "Retrying Kafka..."
  sleep 2
done

# Postgres (real check)
echo "⏳ Waiting for Postgres..."
for i in {1..20}; do
  pg_isready -h localhost -p 5433 -U events && echo "✅ Postgres ready" && break
  echo "Retrying Postgres..."
  sleep 2
done

echo ""
echo "🎉 Infrastructure ready!"