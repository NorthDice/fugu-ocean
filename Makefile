.PHONY: test lint migrate-up migrate-down migrate-status migrate-create

GOLANGCI_LINT ?= $(shell go env GOPATH)/bin/golangci-lint

CONFIG ?= config.yaml
MIGRATIONS_DIR ?= migrations

test:
	go test ./...

lint:
	$(GOLANGCI_LINT) run ./...

migrate-up:
	go run ./cmd/migrate -config $(CONFIG) -dir $(MIGRATIONS_DIR) up

migrate-down:
	go run ./cmd/migrate -config $(CONFIG) -dir $(MIGRATIONS_DIR) down

migrate-status:
	go run ./cmd/migrate -config $(CONFIG) -dir $(MIGRATIONS_DIR) status

# make migrate-create NAME=add_users_table
migrate-create:
	@test -n "$(NAME)" || { echo "NAME is required, e.g. make migrate-create NAME=add_users_table" >&2; exit 1; }
	go run ./cmd/migrate -config $(CONFIG) -dir $(MIGRATIONS_DIR) create $(NAME) sql
