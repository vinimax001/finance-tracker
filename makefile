APP=finance-tracker
BIN=bin/$(APP)

.PHONY: all fmt vet tidy test build run run-pg run-dev docker-build docker-run compose-up compose-down compose-dev compose-logs
all: fmt vet tidy test build

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

test:
	go test ./... -v

build:
	mkdir -p bin
	go build -o $(BIN) ./cmd/api

run:
	STORAGE=memory HTTP_ADDR=:8080 go run ./cmd/api

# Ajuste DATABASE_URL para seu Postgres local
run-pg:
	STORAGE=postgres DATABASE_URL="postgres://financeuser:financepass@localhost:5432/financedb?sslmode=disable" HTTP_ADDR=:8080 go run ./cmd/api

# Sobe o banco dev e depois executa a aplicação
run-dev: compose-dev
	@echo "⏳ Aguardando PostgreSQL iniciar..."
	@sleep 3
	@echo "🚀 Iniciando aplicação..."
	STORAGE=postgres DATABASE_URL="postgres://financeuser:financepass@localhost:5432/financedb?sslmode=disable" HTTP_ADDR=:8080 go run ./cmd/api

docker-build:
	docker build -f infra/Dockerfile -t $(APP):latest .

docker-run:
	docker run --rm -p 8080:8080 -e STORAGE=memory $(APP):latest

# Docker Compose - Stack completa (API + PostgreSQL)
compose-up:
	docker-compose -f infra/docker-compose.yml up -d

compose-down:
	docker-compose -f infra/docker-compose.yml down

compose-down-v:
	docker-compose -f infra/docker-compose.yml down -v

compose-logs:
	docker-compose -f infra/docker-compose.yml logs -f

compose-restart:
	docker-compose -f infra/docker-compose.yml restart

# Docker Compose - Modo desenvolvimento (apenas PostgreSQL)
compose-dev:
	docker-compose -f infra/docker-compose.dev.yml up -d

compose-dev-down:
	docker-compose -f infra/docker-compose.dev.yml down