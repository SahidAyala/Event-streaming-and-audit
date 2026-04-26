.PHONY: up down logs build run run-consumer run-replay test test-e2e lint tidy swag proto \
        docker-up docker-down docker-logs

# =============================================================================
# Primary targets — full stack via Docker
# =============================================================================

## up: Build images and start the full stack (infra + ingest-api + consumer-service).
up:
	docker compose up --build -d
	@echo ""
	@echo "Stack is starting. Run 'make logs' to follow output."
	@echo "API will be available at http://localhost:8080"
	@echo "Swagger UI: http://localhost:8080/swagger/index.html"

## down: Stop and remove all containers (volumes are preserved).
down:
	docker compose down

## logs: Tail logs from all services.
logs:
	docker compose logs -f

## test-e2e: Run end-to-end smoke test against the running stack.
##           Validates Postgres → Kafka → Consumer → Elasticsearch pipeline.
test-e2e:
	@bash scripts/smoke-test.sh

# =============================================================================
# Local development — run services directly (requires infra via make up)
# =============================================================================

## run: Run ingest-api locally against the Docker infra.
run:
	go run ./cmd/ingest-api

## run-consumer: Run consumer-service locally against the Docker infra.
run-consumer:
	go run ./cmd/consumer-service

## run-replay: Run replay-service locally against the Docker infra.
run-replay:
	go run ./cmd/replay-service

# =============================================================================
# Build, test, lint
# =============================================================================

## build: Compile all binaries into bin/.
build:
	go build -o bin/ingest-api      ./cmd/ingest-api
	go build -o bin/consumer-service ./cmd/consumer-service
	go build -o bin/replay-service  ./cmd/replay-service

## test: Run unit tests.
test:
	go test ./...

## lint: Run golangci-lint.
lint:
	golangci-lint run ./...

## tidy: Tidy go.mod and go.sum.
tidy:
	go mod tidy

# =============================================================================
# Code generation
# =============================================================================

## swag: Regenerate Swagger docs from handler annotations.
swag:
	swag init --generalInfo cmd/ingest-api/main.go --output docs --parseDependency --parseInternal

## proto: Regenerate gRPC stubs from proto definitions.
proto:
	mkdir -p gen/proto
	protoc \
		--go_out=gen/proto \
		--go_opt=paths=source_relative \
		--go-grpc_out=gen/proto \
		--go-grpc_opt=paths=source_relative \
		-I proto \
		proto/events.proto

# =============================================================================
# Aliases (backward compat)
# =============================================================================
docker-up:   up
docker-down: down
docker-logs: logs
