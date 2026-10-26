SHELL := /bin/sh

.PHONY: .env up down logs ps run build test vet fmt openapigen

.env:
	cp .env.example .env

up: .env
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

vet:
	go vet ./...

fmt:
	gofmt -w ./cmd ./internal

openapigen:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.1 \
		--config api/oapi-codegen.yaml \
		api/openapi.yaml

run: .env
	@set -a; . ./.env; set +a; \
	DB_DSN="$$MYSQL_USER:$$MYSQL_PASSWORD@tcp(127.0.0.1:$$MYSQL_HOST_PORT)/$$MYSQL_DATABASE?parseTime=true&loc=UTC&charset=utf8mb4" \
	REDIS_ADDR="127.0.0.1:$$REDIS_HOST_PORT" \
	go run ./cmd/api
