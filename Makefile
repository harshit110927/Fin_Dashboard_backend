ifneq (,$(wildcard .env))
include .env
export
endif

DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

run:
	go run ./cmd/server/main.go

build:
	go build -o bin/server ./cmd/server

migrate:
	for f in internal/db/migrations/*.sql; do psql "$(DB_URL)" -f $$f; done

seed:
	psql "$(DB_URL)" -f internal/db/seed_dummy_data.sql

tidy:
	go mod tidy

test:
	go test ./...

e2e:
	node scripts/e2e-test.js
