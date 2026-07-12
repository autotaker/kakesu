SHELL := /bin/sh

GO ?= go
GOFMT ?= gofmt
CARGO ?= cargo
UV ?= uv
NODE ?= node
PNPM ?= pnpm
GO_ENV := GOCACHE=$(CURDIR)/.build/go-cache
UV_ENV := UV_CACHE_DIR=$(CURDIR)/.build/uv-cache

.PHONY: build build-core build-memory build-governance node-deps
.PHONY: test test-core test-memory test-governance test-tabletop test-docs
.PHONY: lint lint-core lint-memory lint-governance lint-docs
.PHONY: check clean

build: build-core build-memory build-governance

build-core:
	@mkdir -p .build
	cd core && $(GO_ENV) $(GO) build -o ../.build/kakesu ./cmd/kakesu

build-memory:
	$(UV_ENV) $(UV) sync --project memory --group dev
	$(UV_ENV) $(UV) build --project memory

build-governance:
	cd governance && $(CARGO) build --locked

test: test-core test-memory test-governance test-tabletop test-docs

test-core:
	cd core && $(GO_ENV) $(GO) test ./...

test-memory:
	$(UV_ENV) $(UV) run --project memory pytest

test-governance:
	cd governance && $(CARGO) test --locked

test-tabletop:
	$(NODE) scripts/validate-tabletop-scenarios.mjs
	$(NODE) scripts/test-tabletop-validator.mjs

node-deps: node_modules/.modules.yaml

node_modules/.modules.yaml: package.json pnpm-lock.yaml
	$(PNPM) install --frozen-lockfile

test-docs: node-deps
	$(PNPM) test:terminology

lint: lint-core lint-memory lint-governance lint-docs

lint-core:
	@test -z "$$($(GOFMT) -l core)" || (echo "Go files required formatting" >&2; exit 1)
	cd core && $(GO_ENV) $(GO) vet ./...

lint-memory:
	$(UV_ENV) $(UV) run --project memory ruff check memory
	$(UV_ENV) $(UV) run --project memory ruff format --check memory

lint-governance:
	cd governance && $(CARGO) fmt --check
	cd governance && $(CARGO) clippy --locked --all-targets -- -D warnings

lint-docs: node-deps
	$(UV_ENV) $(UV) run --project memory python scripts/validate-terminology.py
	$(PNPM) lint:docs
	git diff --check

check: build test lint
	$(NODE) scripts/build-tabletop-viewer-data.mjs
	git diff --check

clean:
	rm -rf .build memory/dist memory/.venv governance/target
