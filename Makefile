.PHONY: test lint

GOLANGCI_LINT ?= $(shell go env GOPATH)/bin/golangci-lint

test:
	go test ./...

lint:
	$(GOLANGCI_LINT) run ./...
