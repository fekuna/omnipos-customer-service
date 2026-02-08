.PHONY: run build test migrate_up migrate_down migrate_create migrate_force migrate_version proto help

# Database Configuration
DB_NAME=omnipos_customer_db
DB_USER=omnipos
DB_PASSWORD=omnipos
DB_HOST=localhost
DB_PORT=5433
DB_SSL=disable
DB_URL="postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL)"

# Default target
help:
	@echo "OmniPOS Customer Service Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  run             - Run the service locally"
	@echo "  build           - Build the binary"
	@echo "  test            - Run tests"
	@echo "  migrate_up      - Run all up migrations"
	@echo "  migrate_down    - Rollback one migration"
	@echo "  migrate_create  - Create a new migration file (usage: make migrate_create name=description)"
	@echo "  migrate_force   - Force set migration version (usage: make migrate_force version=1)"
	@echo "  migrate_version - Print current migration version"
	@echo "  proto           - Generate protobuf files using buf"

run:
	go run ./cmd/main.go

build:
	go build -v -o bin/server ./cmd

test:
	go test -v -cover ./internal/...

migrate_up:
	migrate -database $(DB_URL) -path migrations up

migrate_down:
	migrate -database $(DB_URL) -path migrations down 1

migrate_create:
	@if [ -z "$(name)" ]; then echo "Error: name is required. Usage: make migrate_create name=description"; exit 1; fi
	migrate create -ext sql -dir migrations -seq $(name)

migrate_force:
	@if [ -z "$(version)" ]; then echo "Error: version is required"; exit 1; fi
	migrate -database $(DB_URL) -path migrations force $(version)

migrate_version:
	migrate -database $(DB_URL) -path migrations version

proto:
	buf generate
