#!/usr/bin/env bash
# smoke-test.sh — validates the full Postgres → Kafka → Consumer → Elasticsearch pipeline.
# Usage: bash scripts/smoke-test.sh
#   API_URL=http://localhost:8080  (default)
#   AUTH_API_KEY=dev-api-key       (default)
set -euo pipefail

API="${API_URL:-http://localhost:8080}"
API_KEY="${AUTH_API_KEY:-dev-api-key}"
STREAM_ID="smoke-test:$(date +%s)"

# --- helpers -----------------------------------------------------------------
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
fail() { echo -e "${RED}✗ $*${NC}" >&2; exit 1; }
pass() { echo -e "${GREEN}✓ $*${NC}"; }
info() { echo -e "${YELLOW}→ $*${NC}"; }

# --- 1. wait for API ---------------------------------------------------------
info "Waiting for API at $API ..."
for i in $(seq 1 30); do
  if curl -sf "$API/health" > /dev/null 2>&1; then
    pass "API healthy"
    break
  fi
  [ "$i" -eq 30 ] && fail "API not reachable after 30s — is the stack running? (make up)"
  sleep 1
done

# --- 2. ingest event ---------------------------------------------------------
info "Ingesting event to stream '$STREAM_ID' ..."
INGEST_RESP=$(curl -sf -X POST "$API/events" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"stream_id\": \"$STREAM_ID\",
    \"type\":      \"smoke.test\",
    \"source\":    \"smoke-test\",
    \"payload\":   {\"ok\": true}
  }")

echo "$INGEST_RESP" | grep -q '"id"'      || fail "Ingest response missing 'id' — raw: $INGEST_RESP"
echo "$INGEST_RESP" | grep -q '"version"' || fail "Ingest response missing 'version'"

EVENT_ID=$(echo "$INGEST_RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
VERSION=$(echo  "$INGEST_RESP" | grep -o '"version":[0-9]*' | head -1 | cut -d':' -f2)
pass "Event ingested (id=$EVENT_ID, version=$VERSION)"

# --- 3. wait for Kafka → Elasticsearch pipeline ------------------------------
info "Waiting 3s for Kafka → consumer → Elasticsearch ..."
sleep 3

# --- 4. query read model -----------------------------------------------------
info "Querying stream '$STREAM_ID' from Elasticsearch ..."
QUERY_RESP=$(curl -sf "$API/events/$STREAM_ID" -H "X-API-Key: $API_KEY")

echo "$QUERY_RESP" | grep -q '"smoke.test"' \
  || fail "Event not found in Elasticsearch read model — raw: $QUERY_RESP"

TOTAL=$(echo "$QUERY_RESP" | grep -o '"total":[0-9]*' | head -1 | cut -d':' -f2)
pass "Found $TOTAL event(s) in Elasticsearch read model"

# --- done --------------------------------------------------------------------
echo ""
pass "Smoke test passed — full pipeline is working"
echo ""
echo "  Stream : $STREAM_ID"
echo "  EventID: $EVENT_ID"
echo "  API    : $API"
