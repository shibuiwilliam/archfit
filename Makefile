# archfit — developer entrypoints.
# Keep this file short. Every target must be idempotent and safe to run on a clean checkout.

GO          ?= go
PKG          := ./...
BIN_DIR     := bin
BIN         := $(BIN_DIR)/archfit
VERSION     := $(shell cat VERSION 2>/dev/null || echo dev)
GIT_DIRTY   := $(shell git diff --quiet 2>/dev/null || echo "-dirty")
BUILD_VER   := $(VERSION)$(GIT_DIRTY)
LDFLAGS     := -s -w -X github.com/shibuiwilliam/archfit/internal/version.Version=$(BUILD_VER)

.PHONY: all
all: lint test build

.PHONY: build
build: ## Build the archfit CLI
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/archfit

.PHONY: test
test: ## Run unit + pack tests with race and determinism checks
	$(GO) test -race -count=1 -timeout=60s $(PKG)

.PHONY: test-short
test-short:
	$(GO) test -short -count=1 -timeout=30s $(PKG)

.PHONY: lint
lint: ## Format, vet, and (when installed) golangci-lint / go-arch-lint.
	$(GO) vet $(PKG)
	@if command -v gofmt >/dev/null; then \
	  out=$$(gofmt -l -s $$($(GO) list -f '{{.Dir}}' $(PKG))); \
	  if [ -n "$$out" ]; then echo "gofmt issues:"; echo "$$out"; exit 1; fi; \
	fi
	@if command -v golangci-lint >/dev/null; then \
	  golangci-lint run ./... ; \
	else \
	  echo "golangci-lint not installed — skipping (install via https://golangci-lint.run/usage/install/)"; \
	fi
	@if command -v go-arch-lint >/dev/null; then \
	  go-arch-lint check ; \
	else \
	  echo "go-arch-lint not installed — skipping (install via https://github.com/fe3dback/go-arch-lint)"; \
	fi

.PHONY: e2e
e2e: ## End-to-end golden tests against testdata/e2e
	$(GO) test -race -count=1 -timeout=60s ./testdata/e2e

.PHONY: update-golden
update-golden: ## Regenerate testdata/e2e/*/expected.json. Review the diff carefully.
	$(GO) test -count=1 ./testdata/e2e -update

.PHONY: self-scan
self-scan: build ## Run archfit on itself. Must exit 0 under --fail-on=error.
	$(BIN) scan --fail-on=error .

.PHONY: self-scan-json
self-scan-json: build
	$(BIN) scan --json .

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)

.PHONY: help
help:
	@awk 'BEGIN{FS=":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
