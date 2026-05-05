SHELL := /bin/bash

GO ?= go
BUN ?= bun
DOCKER_COMPOSE ?= docker compose

.PHONY: dev build build-frontend build-go test test-go test-frontend lint lint-go lint-frontend format format-check typecheck typecheck-go typecheck-frontend ci ci-pipeline image-size install

dev:
	$(DOCKER_COMPOSE) -f deploy/docker-compose.yml up -d postgres
	@printf 'Postgres is running. Start portless for https://cypra.localhost and https://*.cypra.localhost.\n'
	$(GO) run ./cmd/cypra serve

install:
	$(BUN) install --frozen-lockfile
	$(GO) mod download

build: build-frontend build-go

build-frontend:
	cd dashboard && $(BUN) run build

build-go:
	mkdir -p bin
	$(GO) build -trimpath -ldflags "-s -w" -o bin/cypra ./cmd/cypra

test: build-frontend test-go test-frontend

test-go:
	$(GO) test ./...

test-frontend:
	$(BUN) run test

lint: lint-go lint-frontend

lint-go:
	golangci-lint run

lint-frontend:
	$(BUN) run lint

format: 
	gofumpt -w cmd internal dashboard sdk db deploy docs examples
	goimports -w cmd internal dashboard sdk db deploy docs examples
	$(BUN) run format:write

format-check:
	@test -z "$$($(GO) fmt ./...)"
	@test -z "$$(gofmt -l cmd internal dashboard sdk db deploy docs examples)"
	$(BUN) run format

typecheck: typecheck-go typecheck-frontend

typecheck-go:
	$(GO) vet ./...

typecheck-frontend:
	$(BUN) run typecheck

ci: ci-pipeline

ci-pipeline: install build lint format-check typecheck test

image-size: build-go
	@wc -c < bin/cypra | awk '{ printf "bin/cypra: %d bytes\n", $$1 }'
