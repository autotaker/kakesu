SHELL := /bin/sh

GO ?= go
GOFMT ?= gofmt
CARGO ?= cargo
UV ?= uv
NODE ?= node
PNPM ?= pnpm
PRODUCT_ROOT ?= $(abspath $(dir $(shell git rev-parse --path-format=absolute --git-common-dir)))
WORK_ROOT ?= $(abspath $(PRODUCT_ROOT)/../agent-harness-work)
WIKI_CONTEXT_TARGET ?= task
WIKI_PROFILE ?=
WIKI_MODEL ?= gpt-5.6-terra
GO_ENV := GOCACHE=$(CURDIR)/.build/go-cache
UV_ENV := UV_CACHE_DIR=$(CURDIR)/.build/uv-cache

.PHONY: build build-core build-memory build-governance node-deps
.PHONY: test test-core test-memory test-governance test-tabletop test-docs test-process
.PHONY: lint lint-core lint-memory lint-governance lint-docs
.PHONY: check clean work-init work-agent work-config-sync task-create task-check work-check backlog-view worktree-create worktree-remove wiki-index wiki-context wiki-ingest

build: build-core build-memory build-governance

build-core:
	@mkdir -p .build
	cd core && $(GO_ENV) $(GO) build -o ../.build/kakesu ./cmd/kakesu

build-memory:
	$(UV_ENV) $(UV) sync --project memory --group dev
	$(UV_ENV) $(UV) build --project memory

build-governance:
	cd governance && $(CARGO) build --locked

test: test-core test-memory test-governance test-tabletop test-docs test-process

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

test-process: node-deps
	$(PNPM) test:process

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

work-init: node-deps
	$(NODE) scripts/task/init-work.mjs --work-root "$(WORK_ROOT)"

work-agent: node-deps
	@test -n "$(TASK)" || (echo "TASK is required" >&2; exit 1)
	@test -n "$(ACTION)" || (echo "ACTION is required" >&2; exit 1)
	$(NODE) scripts/task/run-work-agent.mjs --work-root "$(WORK_ROOT)" --task "$(TASK)" --action "$(ACTION)" $(if $(PROFILE),--profile "$(PROFILE)",) $(if $(MODEL),--model "$(MODEL)",) $(if $(EFFORT),--effort "$(EFFORT)",)

work-config-sync: node-deps
	$(NODE) scripts/task/agent-routing.mjs --work-root "$(WORK_ROOT)" --mode "$(if $(CHECK),check,sync)"

task-create: node-deps
	@test -n "$(ID)" || (echo "ID is required" >&2; exit 1)
	@test -n "$(SLUG)" || (echo "SLUG is required" >&2; exit 1)
	@test -n "$(TITLE)" || (echo "TITLE is required" >&2; exit 1)
	@test -n "$(EPIC)" || (echo "EPIC is required" >&2; exit 1)
	$(NODE) scripts/task/create-task.mjs --work-root "$(WORK_ROOT)" --id "$(ID)" --slug "$(SLUG)" --title "$(TITLE)" --epic "$(EPIC)" --type "$(or $(TYPE),feature)" --priority "$(or $(PRIORITY),P2)"

task-check: node-deps
	@test -n "$(TASK)" || (echo "TASK is required" >&2; exit 1)
	$(NODE) scripts/task/check-task.mjs --work-root "$(WORK_ROOT)" --task "$(TASK)"

work-check: node-deps
	$(NODE) scripts/task/validate-work.mjs --work-root "$(WORK_ROOT)"

backlog-view: node-deps work-check
	$(NODE) scripts/task/build-backlog-viewer.mjs --work-root "$(WORK_ROOT)"

worktree-create: node-deps
	@test -n "$(TASK)" || (echo "TASK is required" >&2; exit 1)
	$(NODE) scripts/task/worktree.mjs --work-root "$(WORK_ROOT)" --task "$(TASK)" --action create

worktree-remove: node-deps
	@test -n "$(TASK)" || (echo "TASK is required" >&2; exit 1)
	$(NODE) scripts/task/worktree.mjs --work-root "$(WORK_ROOT)" --task "$(TASK)" --action remove

wiki-index: node-deps
	$(NODE) scripts/task/wiki-index.mjs --work-root "$(WORK_ROOT)"

wiki-context: node-deps
	@test -n "$(TASK)" || (echo "TASK is required" >&2; exit 1)
	@test "$(WIKI_CONTEXT_TARGET)" = "task" -o "$(WIKI_CONTEXT_TARGET)" = "plan" || (echo "WIKI_CONTEXT_TARGET must be task or plan" >&2; exit 1)
	$(NODE) scripts/task/run-wiki-agent.mjs --work-root "$(WORK_ROOT)" --task "$(TASK)" --action "context-$(WIKI_CONTEXT_TARGET)" $(if $(WIKI_PROFILE),--profile "$(WIKI_PROFILE)",) $(if $(WIKI_MODEL),--model "$(WIKI_MODEL)",) $(if $(WIKI_EFFORT),--effort "$(WIKI_EFFORT)",)

wiki-ingest: node-deps
	@test -n "$(TASK)" || (echo "TASK is required" >&2; exit 1)
	$(NODE) scripts/task/run-wiki-agent.mjs --work-root "$(WORK_ROOT)" --task "$(TASK)" --action ingest $(if $(WIKI_PROFILE),--profile "$(WIKI_PROFILE)",) $(if $(WIKI_MODEL),--model "$(WIKI_MODEL)",) $(if $(WIKI_EFFORT),--effort "$(WIKI_EFFORT)",)

clean:
	rm -rf .build memory/dist memory/.venv governance/target
