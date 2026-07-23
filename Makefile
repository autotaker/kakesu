SHELL := /bin/sh

GO ?= go
GOFMT ?= gofmt
CARGO ?= cargo
UV ?= uv
NODE ?= node
PNPM ?= pnpm
PRODUCT_ROOT ?= $(abspath $(dir $(shell git rev-parse --path-format=absolute --git-common-dir)))
MAIN_ROOT ?= $(PRODUCT_ROOT)
SOURCE_REF ?= d030db5dc2974056387616d047197823b94602ce
SOURCE_HEAD ?= a49338d5013f8e54f72a9c7cc4f92c4a76c52d91
WIKI_CONTEXT_TARGET ?= task
WIKI_PROFILE ?=
WIKI_MODEL ?= gpt-5.6-terra
GO_ENV := GOCACHE=$(CURDIR)/.build/go-cache
UV_ENV := UV_CACHE_DIR=$(CURDIR)/.build/uv-cache

.PHONY: build build-core build-memory build-governance node-deps
.PHONY: test test-core test-memory test-governance test-tabletop test-docs test-process
.PHONY: lint lint-core lint-memory lint-governance lint-docs
.PHONY: check clean explorer-agent task-start task-check task-preflight work-check backlog-view wiki-index wiki-context wiki-ingest
.PHONY: evidence-commit task-pr task-scope-check sync bootstrap-plan bootstrap-apply bootstrap-verify bootstrap-freeze bootstrap-unfreeze

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

explorer-agent: node-deps
	@test -n "$$QUESTION" || (echo "QUESTION is required" >&2; exit 1)
	$(NODE) scripts/task/run-explorer-agent.mjs --root "$(if $(EXPLORER_ROOT),$$EXPLORER_ROOT,$(CURDIR))" --question "$$QUESTION" $(if $(DRY_RUN),--dry-run "$$DRY_RUN",)

task-start: node-deps
	@test -n "$(ID)" || (echo "ID is required" >&2; exit 1)
	@test -n "$(SLUG)" || (echo "SLUG is required" >&2; exit 1)
	@test -n "$(TITLE)" || (echo "TITLE is required" >&2; exit 1)
	$(NODE) scripts/task/unified-lifecycle.mjs --action task-start --main-root "$(MAIN_ROOT)" --id "$(ID)" --slug "$(SLUG)" --title "$(TITLE)" --epic "$(or $(EPIC),EPIC-001)" --type "$(or $(TYPE),feature)" --priority "$(or $(PRIORITY),P2)" --push "$(if $(NO_PUSH),false,true)"

evidence-commit: node-deps
	@test -n "$(TASK)" || (echo "TASK is required" >&2; exit 1)
	@test -n "$(ACTION)" || (echo "ACTION is required" >&2; exit 1)
	$(NODE) scripts/task/unified-lifecycle.mjs --action evidence-commit --main-root "$(MAIN_ROOT)" --task "$(TASK)" --evidence-action "$(ACTION)" --message "$(or $(MESSAGE),task: $(ACTION) $(TASK))" --push "$(if $(NO_PUSH),false,true)"

task-pr: node-deps
	@test -n "$(TASK)" || (echo "TASK is required" >&2; exit 1)
	$(NODE) scripts/task/unified-lifecycle.mjs --action task-pr --main-root "$(MAIN_ROOT)" --task "$(TASK)" --repo "$(or $(REPO),autotaker/kakesu)" --dry-run "$(if $(DRY_RUN),true,false)"

task-check: node-deps
	@test -n "$(TASK)" || (echo "TASK is required" >&2; exit 1)
	$(NODE) scripts/task/check-task.mjs --work-root "$(MAIN_ROOT)" --task "$(TASK)"

task-preflight: node-deps
	@test -n "$(TASK)" || (echo "TASK is required" >&2; exit 1)
	$(NODE) scripts/task/check-task.mjs --work-root "$(MAIN_ROOT)" --task "$(TASK)" --phase preflight

work-check: node-deps
	$(NODE) scripts/task/validate-work.mjs --work-root "$(MAIN_ROOT)" --schema-root "$(CURDIR)"

backlog-view: node-deps work-check
	$(NODE) scripts/task/build-backlog-viewer.mjs --work-root "$(MAIN_ROOT)"

wiki-index: node-deps
	$(NODE) scripts/task/wiki-index.mjs --work-root "$(MAIN_ROOT)"

wiki-context: node-deps
	@test -n "$(TASK)" || (echo "TASK is required" >&2; exit 1)
	@test "$(WIKI_CONTEXT_TARGET)" = "task" -o "$(WIKI_CONTEXT_TARGET)" = "plan" || (echo "WIKI_CONTEXT_TARGET must be task or plan" >&2; exit 1)
	$(NODE) scripts/task/run-wiki-agent.mjs --work-root "$(MAIN_ROOT)" --task "$(TASK)" --action "context-$(WIKI_CONTEXT_TARGET)" $(if $(WIKI_PROFILE),--profile "$(WIKI_PROFILE)",) $(if $(WIKI_MODEL),--model "$(WIKI_MODEL)",) $(if $(WIKI_EFFORT),--effort "$(WIKI_EFFORT)",)

wiki-ingest: node-deps
	@test -n "$(TASK)" || (echo "TASK is required" >&2; exit 1)
	$(NODE) scripts/task/run-wiki-agent.mjs --work-root "$(MAIN_ROOT)" --task "$(TASK)" --action ingest $(if $(WIKI_PROFILE),--profile "$(WIKI_PROFILE)",) $(if $(WIKI_MODEL),--model "$(WIKI_MODEL)",) $(if $(WIKI_EFFORT),--effort "$(WIKI_EFFORT)",)

task-scope-check: node-deps
	$(NODE) scripts/task/unified-lifecycle.mjs --action scope-check --main-root "$(MAIN_ROOT)" --event "$(EVENT)" --base "$(BASE)" --head "$(HEAD)"

sync: node-deps
	$(NODE) scripts/task/unified-lifecycle.mjs --action sync --main-root "$(MAIN_ROOT)" --fast "$(or $(FAST),0)" --repo "$(or $(REPO),autotaker/kakesu)" --push "$(if $(NO_PUSH),false,true)"

bootstrap-plan bootstrap-apply bootstrap-freeze bootstrap-unfreeze: node-deps
	@test -n "$(SOURCE_ROOT)" || (echo "SOURCE_ROOT is required" >&2; exit 1)
	$(NODE) scripts/task/migrate-operations.mjs --mode "$(patsubst bootstrap-%,%,$@)" --source "$(SOURCE_ROOT)" --source-ref "$(SOURCE_REF)" --expected-head "$(SOURCE_HEAD)" --target "$(MAIN_ROOT)"

bootstrap-verify: node-deps
	$(NODE) scripts/task/migrate-operations.mjs --mode verify --target "$(MAIN_ROOT)"

clean:
	rm -rf .build memory/dist memory/.venv governance/target
