SHELL := /bin/sh

.PHONY: up down logs ps run build test vet fmt openapigen migrate-up migrate-down test-integration web web-build

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f app

ps:
	docker compose ps

build:
	go build ./...

test:
	go test ./...

test-integration:
	go test -tags=integration -count=1 -v ./internal/adapter/mysql/ -run TestTeamStatsRepositoryIntegration

vet:
	go vet ./...

fmt:
	gofmt -w ./cmd ./internal

openapigen:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.1 \
		--config api/oapi-codegen.yaml \
		api/openapi.yaml

run:
	@if [ ! -f .env ]; then echo "Error: .env not found. Copy .env.example to .env and edit it." >&2; exit 1; fi
	@set -a; . ./.env; set +a; \
	DB_DSN="$$MYSQL_USER:$$MYSQL_PASSWORD@tcp(127.0.0.1:$$MYSQL_HOST_PORT)/$$MYSQL_DATABASE?parseTime=true&loc=UTC&charset=utf8mb4" \
	REDIS_ADDR="127.0.0.1:$$REDIS_HOST_PORT" \
	go run ./cmd/api

migrate-up:
	@if [ ! -f .env ]; then echo "Error: .env not found. Copy .env.example to .env and edit it." >&2; exit 1; fi
	@(set -a; . ./.env; set +a; \
	DSN=$${DB_DSN:-$${MYSQL_USER}:$${MYSQL_PASSWORD}@tcp(127.0.0.1:$${MYSQL_HOST_PORT:-3306})/$${MYSQL_DATABASE}?parseTime=true&loc=UTC&charset=utf8mb4}; \
	go run github.com/pressly/goose/v3/cmd/goose -dir migrations mysql "$$DSN" up)

migrate-down:
	@if [ ! -f .env ]; then echo "Error: .env not found. Copy .env.example to .env and edit it." >&2; exit 1; fi
	@(set -a; . ./.env; set +a; \
	DSN=$${DB_DSN:-$${MYSQL_USER}:$${MYSQL_PASSWORD}@tcp(127.0.0.1:$${MYSQL_HOST_PORT:-3306})/$${MYSQL_DATABASE}?parseTime=true&loc=UTC&charset=utf8mb4}; \
	go run github.com/pressly/goose/v3/cmd/goose -dir migrations mysql "$$DSN" down)

web:
	cd web && npm run dev

web-build:
	cd web && npm run build
