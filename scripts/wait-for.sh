#!/usr/bin/env bash
set -e

SERVICE_NAME=$1
URL=$2
MAX_RETRIES=${3:-30}

echo "⏳ Waiting for $SERVICE_NAME at $URL..."

for i in $(seq 1 $MAX_RETRIES); do
  if curl -sf "$URL" > /dev/null; then
    echo "✅ $SERVICE_NAME is ready"
    exit 0
  fi

  echo "Attempt $i/$MAX_RETRIES failed. Retrying in 2s..."
  sleep 2
done

echo "❌ $SERVICE_NAME not ready after $MAX_RETRIES attempts"
exit 1