# Makefile

# Variables
APP_NAME=sema	
DOCKER_COMPOSE=docker compose
GO=go
MIGRATE=migrate
MIGRATE_PATH=./database/schema
DATABASE="postgres://sema:sema@localhost:5432/sema?sslmode=disable"

# Default target
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make up          - Start all Docker containers"
	@echo "  make down        - Stop all Docker containers"
	@echo "  make restart     - Restart containers"
	@echo "  make logs        - View Docker logs"
	@echo "  make air         - Run Go app with Air (hot reload)"
	@echo "  make build       - Build Go binary"
	@echo "  make run         - Run Go app normally"
	@echo "  make tidy        - Run go mod tidy"
	@echo "  make test        - Run tests"
	@echo "  make migrate-up  - Run database migrations"
	@echo "  make migrate-down - Run database migrations"
	@echo "  make migrate-redo - Run database migrations"
	@echo "  make migrate-create <name> - Create new migration"

# migration commands
.PHONY: migrate-up migrate-down migrate-redo migrate-create
migrate-up:
	$(MIGRATE) -path $(MIGRATE_PATH) -database $(DATABASE) up

migrate-down:
	$(MIGRATE) -path $(MIGRATE_PATH) -database $(DATABASE) down

migrate-redo:
	$(MIGRATE) -path $(MIGRATE_PATH) -database $(DATABASE) redo

migrate-create:
	$(MIGRATE) create -dir $(MIGRATE_PATH) -ext sql $(name)

# Docker commands
.PHONY: up down restart logs
up:
	$(DOCKER_COMPOSE) up -d

down:
	$(DOCKER_COMPOSE) down

restart:
	$(DOCKER_COMPOSE) down && $(DOCKER_COMPOSE) up -d

logs:
	$(DOCKER_COMPOSE) logs -f

# Go commands
.PHONY: air build run tidy test
air:
	air

build:
	$(GO) build -o $(APP_NAME) .

run:
	$(GO) run main.go

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./... -v

.PHONY: sqlc
sqlc:
	sqlc generate -f config/sqlc.yaml
.PHONY: docs
docs:
	swag init
	
	
