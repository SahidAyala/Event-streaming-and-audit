.PHONY: proto build run docker-up docker-down test lint tidy

PROTO_DIR := proto
GEN_DIR   := gen/proto

proto:
	mkdir -p $(GEN_DIR)
	protoc \
		--go_out=$(GEN_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) \
		--go-grpc_opt=paths=source_relative \
		-I $(PROTO_DIR) \
		$(PROTO_DIR)/events.proto

build:
	go build -o bin/ingest-api ./cmd/ingest-api
	go build -o bin/consumer-service ./cmd/consumer-service
	go build -o bin/replay-service ./cmd/replay-service

run:
	go run ./cmd/ingest-api

run-replay:
	go run ./cmd/replay-service

tidy:
	go mod tidy

test:
	go test ./...

lint:
	golangci-lint run ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
