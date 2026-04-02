include .env
export

DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

run:
	go run ./cmd/server/main.go

build:
	go build -o bin/server ./cmd/server/main.go

migrate:
	for f in internal/db/migrations/*.sql; do psql "$(DB_URL)" -f $$f; done

tidy:
	go mod tidy

test:
	go test ./...
