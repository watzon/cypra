SHELL := /bin/bash

GO ?= go
BUN ?= bun
DOCKER_COMPOSE ?= docker compose
PORTLESS ?= portless
AIR ?= air
COMPOSE_FILE ?= deploy/docker-compose.yml
DEV_COMPOSE_SERVICES ?= postgres
DEV_DASHBOARD_PORT ?= 5173
IMAGE_SIZE_TAG ?= cypra:image-size
IMAGE_SIZE_MAX_BYTES ?= 83886080

.PHONY: dev dev-up dev-down dev-reset setup-token build build-frontend build-go test test-go test-frontend test-go-coverage lint lint-go lint-frontend lint-audit-writes lint-instance-admin-escape format format-check typecheck typecheck-go typecheck-frontend ci ci-pipeline performance image-size cold-start lighthouse-baseline install

.env:
	cp .env.example .env
	@printf 'Created .env from .env.example. Review it before production use.\n'

dev-up: .env
	$(DOCKER_COMPOSE) --env-file .env -f $(COMPOSE_FILE) up -d --wait $(DEV_COMPOSE_SERVICES)

dev-down: .env
	$(DOCKER_COMPOSE) --env-file .env -f $(COMPOSE_FILE) stop $(DEV_COMPOSE_SERVICES)

# Mint a fresh setup token for the first instance admin. Requires the local
# postgres to be reachable (run `make dev-up` first if it isn't).
setup-token: .env
	@set -a; source .env; set +a; \
	$(GO) run ./cmd/cypra admin reset-bootstrap

# DESTRUCTIVE: stop the local dev stack and wipe the postgres volume so the
# next `make dev` migrates a fresh database. Use this to re-run onboarding
# from scratch. Local dev only — never run against a real install.
dev-reset: .env
	@printf 'This will permanently delete the local Cypra postgres volume.\n'
	@$(DOCKER_COMPOSE) --env-file .env -f $(COMPOSE_FILE) down -v $(DEV_COMPOSE_SERVICES)
	@printf 'Done. Run `make dev` to start fresh.\n'

dev: .env
	@set -euo pipefail; \
	command -v $(PORTLESS) >/dev/null || { printf 'portless is required for local HTTPS and wildcard tenant routing. Install it with: npm install -g portless\n' >&2; exit 127; }; \
	command -v $(AIR) >/dev/null || { printf 'air is required for live-reloading the Go server. Install it with: go install github.com/air-verse/air@latest\n' >&2; exit 127; }; \
	set -a; source .env; set +a; \
	listen_addr="$${LISTEN_ADDR:-:8080}"; \
	cypra_port="$${listen_addr##*:}"; \
	if ! [[ "$$cypra_port" =~ ^[0-9]+$$ ]]; then printf 'LISTEN_ADDR must end with a numeric port for make dev, got %s\n' "$$listen_addr" >&2; exit 1; fi; \
	$(DOCKER_COMPOSE) --env-file .env -f $(COMPOSE_FILE) up -d --wait $(DEV_COMPOSE_SERVICES); \
	$(PORTLESS) proxy start --wildcard; \
	$(PORTLESS) alias cypra "$$cypra_port" --force; \
	cleanup() { \
		status=$$?; \
		if [ -n "$${vite_pid:-}" ]; then kill $$vite_pid 2>/dev/null || true; wait $$vite_pid 2>/dev/null || true; fi; \
		$(PORTLESS) alias --remove cypra >/dev/null 2>&1 || true; \
		$(DOCKER_COMPOSE) --env-file .env -f $(COMPOSE_FILE) stop $(DEV_COMPOSE_SERVICES) >/dev/null; \
		exit $$status; \
	}; \
	trap cleanup EXIT; \
	trap 'exit 130' INT; \
	trap 'exit 143' TERM; \
	export VITE_DEV_SERVER="$${VITE_DEV_SERVER:-http://127.0.0.1:$(DEV_DASHBOARD_PORT)}"; \
	$(BUN) --cwd dashboard vite --host 127.0.0.1 --port $(DEV_DASHBOARD_PORT) --strictPort & \
	vite_pid=$$!; \
	vite_ready=0; \
	for _ in {1..80}; do \
		if (true >/dev/tcp/127.0.0.1/$(DEV_DASHBOARD_PORT)) >/dev/null 2>&1; then vite_ready=1; break; fi; \
		if ! kill -0 $$vite_pid 2>/dev/null; then wait $$vite_pid; exit $$?; fi; \
		sleep 0.25; \
	done; \
	if [ "$$vite_ready" != 1 ]; then printf 'Timed out waiting for dashboard dev server on 127.0.0.1:$(DEV_DASHBOARD_PORT)\n' >&2; exit 1; fi; \
	$(GO) run ./cmd/cypra migrate; \
	printf 'Dev stack is running at https://cypra.localhost and https://*.cypra.localhost. The dashboard dev server is %s. Go server reloads on changes via air. Press Ctrl-C to stop everything.\n' "$$VITE_DEV_SERVER"; \
	$(AIR) -c .air.toml

install:
	$(BUN) install --frozen-lockfile
	$(GO) mod download

build: build-frontend build-go

build-frontend:
	cd dashboard && $(BUN) run build

build-go:
	mkdir -p bin
	$(GO) build -trimpath -ldflags "-s -w" -o bin/cypra ./cmd/cypra

test: build-frontend test-go test-go-coverage test-frontend

test-go:
	$(GO) test -p 1 ./...

test-go-coverage:
	$(BUN) run go:coverage

test-frontend:
	$(BUN) run test

lint: lint-go lint-frontend lint-no-ops lint-audit-writes lint-instance-admin-escape

lint-go:
	golangci-lint run

lint-frontend:
	$(BUN) run lint

# lint-no-ops: catches dashboard buttons that have no working handler.
#   1) onClick={() => undefined} or onClick={() => {}}
#   2) <Button>Label</Button> with no onClick / disabled / type / data-primary-create
# Gallery files are deliberately exempt; they showcase static states.
lint-no-ops:
	@out=$$(grep -nE "onClick=\{?\(\)\s*=>\s*(undefined|\{\s*\})" \
	  dashboard/src/App.tsx dashboard/src/screens.tsx dashboard/src/components.tsx dashboard/src/modals.tsx \
	  | grep -v "PrimitiveGallery\|StateGallery\|States\b" || true); \
	if [ -n "$$out" ]; then \
	  echo "lint-no-ops: silent onClick handlers detected"; \
	  echo "$$out"; \
	  exit 1; \
	fi
	@out=$$(grep -nE "<Button[^/]*>[^<]*</Button>" \
	  dashboard/src/screens.tsx dashboard/src/modals.tsx \
	  | grep -v "onClick\|disabled\|data-primary-create\|type=\"submit\"\|Gallery\|States" || true); \
	if [ -n "$$out" ]; then \
	  echo "lint-no-ops: <Button> without onClick / disabled / type / data-primary-create"; \
	  echo "$$out"; \
	  exit 1; \
	fi

lint-audit-writes:
	@out=$$(grep -R -n "\.audit\.Write\|audit\.Write(" internal/httpserver --include='*.go' | grep -v "audit_writer.go" || true); \
	if [ -n "$$out" ]; then \
	  echo "lint-audit-writes: direct audit writes outside audit_writer.go"; \
	  echo "$$out"; \
	  exit 1; \
	fi

lint-instance-admin-escape:
	@out=$$(grep -R -n "\b\(AsInstanceAdmin\|ContextAsInstanceAdmin\)(" cmd internal --include='*.go' \
	  | grep -v "internal/db/instance_admin.go" \
	  | grep -v "internal/db/.*_test.go" || true); \
	if [ -n "$$out" ]; then \
	  echo "lint-instance-admin-escape: forbidden instance-admin escape hatch caller"; \
	  echo "$$out"; \
	  exit 1; \
	fi

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

ci-pipeline: install build lint format-check typecheck test performance cold-start

performance:
	CYPRA_PERF_GATE=1 $(GO) test -count=1 -run TestPhase18P99PerformanceAssertions ./internal/httpserver

image-size:
	docker build -t $(IMAGE_SIZE_TAG) .
	@bytes=$$(docker image save $(IMAGE_SIZE_TAG) | gzip -c | wc -c | tr -d ' '); \
	printf '$(IMAGE_SIZE_TAG) compressed: %s bytes\n' "$$bytes"; \
	if [ "$$bytes" -gt "$(IMAGE_SIZE_MAX_BYTES)" ]; then \
		printf 'image-size: compressed image exceeds %s bytes\n' "$(IMAGE_SIZE_MAX_BYTES)" >&2; \
		exit 1; \
	fi

cold-start: image-size
	$(BUN) run cold-start

lighthouse-baseline:
	$(BUN) run lighthouse:baseline
