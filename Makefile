SHELL := /bin/bash

GO_ENV := GOSUMDB=sum.golang.org
GO      := $(GO_ENV) go
NPM     := npm
STEPS  ?= 1
CONFIG ?= config.yaml

.PHONY: build test test-with-threshold tidy run lint node-install web-install web-test web-test-coverage web-test-with-threshold web-lint web-sync check-web-sync go-coverage ci clean migrate migrate-down migrate-version storage-export storage-import storage-verify openapi-lint typegen typecheck lint-all lint-fix web-lint-fix fmt fmt-check fmt-fix web-fmt web-fmt-check web-fmt-fix quality-check

tidy:
	$(GO) mod tidy

build:
	$(GO) build ./cmd/server

test:
	$(GO) test ./...

# 带覆盖率阈值检查的测试（阶段目标：50%）
test-with-threshold:
	@echo "Running Go tests with coverage threshold check..."
	@$(GO) test ./... -coverprofile=coverage.out
	@COVERAGE=$$($(GO) tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	THRESHOLD=50; \
	echo "Coverage: $$COVERAGE% (threshold: $$THRESHOLD%)"; \
	if [ $$(echo "$$COVERAGE < $$THRESHOLD" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below threshold $$THRESHOLD%"; \
		rm -f coverage.out; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets threshold $$THRESHOLD%"; \
		rm -f coverage.out; \
	fi

run:
	$(GO) run ./cmd/server

lint:
	$(GO) vet ./...

# 运行所有 lint 检查
lint-all: lint web-lint
	@echo "✅ All lint checks passed"

# 自动修复 lint 问题
lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix; \
	else \
		echo "⚠️  golangci-lint not installed, skipping Go lint fix"; \
	fi

# 格式化 Go 代码
fmt:
	@echo "Formatting Go code..."
	@$(GO) fmt ./...
	@echo "✅ Go code formatted"

# 检查 Go 代码格式
fmt-check:
	@echo "Checking Go code format..."
	@UNFORMATTED=$$(gofmt -l .); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "❌ The following files are not formatted:"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	else \
		echo "✅ All Go files are properly formatted"; \
	fi

# 格式化所有代码（Go + 前端）
fmt-fix: fmt web-fmt-fix
	@echo "✅ All code formatted"

# 前端代码格式化
web-fmt:
	@echo "Formatting frontend code..."
	@if command -v prettier >/dev/null 2>&1; then \
		prettier --write "web/**/*.{js,ts,json,css,html}"; \
	else \
		echo "⚠️  prettier not installed, trying npx..."; \
		npx prettier --write "web/**/*.{js,ts,json,css,html}"; \
	fi
	@echo "✅ Frontend code formatted"

# 检查前端代码格式
web-fmt-check:
	@echo "Checking frontend code format..."
	@if command -v prettier >/dev/null 2>&1; then \
		prettier --check "web/**/*.{js,ts,json,css,html}"; \
	else \
		npx prettier --check "web/**/*.{js,ts,json,css,html}"; \
	fi
	@echo "✅ Frontend code format is correct"

# 自动修复前端代码格式
web-fmt-fix: web-fmt

# 完整的代码质量检查（格式 + lint + 类型检查）
quality-check: fmt-check lint-all typecheck
	@echo "✅ All quality checks passed"

migrate:
	@if [ -z "$(DSN)" ]; then echo "Usage: make migrate DSN=postgres://user:pass@host:5432/dbname"; exit 1; fi
	$(GO) run ./cmd/migrate -dsn "$(DSN)" -action up

migrate-down:
	@if [ -z "$(DSN)" ]; then echo "Usage: make migrate-down DSN=postgres://user:pass@host:5432/dbname [STEPS=N]"; exit 1; fi
	$(GO) run ./cmd/migrate -dsn "$(DSN)" -action down -steps $(STEPS)

migrate-version:
	@if [ -z "$(DSN)" ]; then echo "Usage: make migrate-version DSN=postgres://user:pass@host:5432/dbname"; exit 1; fi
	$(GO) run ./cmd/migrate -dsn "$(DSN)" -action version

storage-export:
	$(GO) run ./cmd/storageutil -config "$(CONFIG)" -mode export $(if $(FILE),-file "$(FILE)")

storage-import:
	@if [ -z "$(FILE)" ]; then echo "Usage: make storage-import FILE=path/to/data.json [CONFIG=...]"; exit 1; fi
	$(GO) run ./cmd/storageutil -config "$(CONFIG)" -mode import -file "$(FILE)"

storage-verify:
	@if [ -z "$(FILE)" ]; then echo "Usage: make storage-verify FILE=path/to/data.json [CONFIG=...]"; exit 1; fi
	$(GO) run ./cmd/storageutil -config "$(CONFIG)" -mode verify -file "$(FILE)"

node-install:
	$(NPM) install

web-install:
	cd web && $(NPM) install

web-test:
	cd web && $(NPM) test -- --runInBand

# 带覆盖率阈值检查的前端测试（阶段目标：40%）
web-test-with-threshold:
	@echo "Running frontend tests with coverage threshold check..."
	@cd web && $(NPM) run test:coverage
	@echo "✅ Frontend tests passed with coverage threshold"

web-lint:
	$(NPM) run lint

# 自动修复前端 lint 问题
web-lint-fix:
	@echo "Running ESLint with auto-fix..."
	@cd web && $(NPM) run lint:fix || true

web-sync:
	cd web && $(NPM) run build:ts

check-web-sync:
	./scripts/check_web_sync.sh

go-coverage:
	$(GO) test ./... -coverprofile=coverage.out
	GOSUMDB=sum.golang.org GO_BIN=go ./scripts/check_go_coverage.sh coverage.out
	rm -f coverage.out

web-test-coverage:
	cd web && $(NPM) run test:coverage

openapi-lint:
	npx --yes @redocly/cli@1.15.0 lint docs/openapi/openapi.yaml

typegen:
	$(NPM) run typegen

typecheck:
	$(NPM) run typecheck

clean:
	rm -rf dist build

# CI 流水线（包含所有检查和测试）
ci: tidy node-install typegen web-install web-sync check-web-sync typecheck web-type-coverage bundle-size test-with-threshold web-test-with-threshold build lint-all

# 快速 CI（跳过耗时的检查）
ci-fast: tidy web-install typecheck test build lint

.PHONY: web-type-coverage bundle-size

web-type-coverage:
	cd web && npm run type:coverage

bundle-size:
	node scripts/check_web_bundle_size.mjs
