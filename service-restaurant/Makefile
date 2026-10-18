.PHONY: fmt vet test test-race test-integration build run db-up db-down db-logs migrate-up migrate-down migrate-status

DATABASE_URL ?= postgres://restaurant:change-me@localhost:5432/restaurant?sslmode=disable
export DATABASE_URL

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race ./...

# Las pruebas de integracion viven en tests/integration y requieren Docker (testcontainers).
test-integration:
	go test -count=1 ./tests/integration/...

build:
	go build -o bin/api ./cmd/api

run:
	go run ./cmd/api

db-up:
	docker compose up -d --wait db

db-down:
	docker compose down

db-logs:
	docker compose logs -f db

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status
