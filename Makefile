# sheets-mcp — Run 'make' or 'make help' to see available commands

.DEFAULT_GOAL := help

BIN         := ./bin
BINARY      := $(BIN)/sheets-mcp
MODULE      := github.com/mab-go/sheets-mcp
VERSION_PKG := $(MODULE)/internal/version
GOLANGCI    := $(BIN)/golangci-lint
GOIMPORTS   := $(BIN)/goimports
GOCYCLO     := $(BIN)/gocyclo
# Pinned golangci-lint release for reproducible `make lint`; bump if unsupported on current Go (see go.mod).
GOLANGCI_LINT_VERSION ?= v2.11.3
# Pinned goimports (golang.org/x/tools); bump if `make fmt` fails or is incompatible with go.mod Go version.
GOIMPORTS_VERSION ?= v0.38.0
# Pinned gocyclo (github.com/fzipp/gocyclo); bump for `make cyclo` reproducibility.
GOCYCLO_VERSION ?= v0.6.0

# golangci-lint must be built with Go >= go.mod; auto follows deps' older go version, so pin to module Go.
GO_MOD_VERSION := $(shell grep -E '^go ' go.mod | head -1 | awk '{print $$2}')
TOOLCHAIN_FOR_TOOLS ?= go$(GO_MOD_VERSION)

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +%Y-%m-%d)

LDFLAGS := -X $(VERSION_PKG).Version=$(VERSION) \
           -X $(VERSION_PKG).Commit=$(COMMIT) \
           -X $(VERSION_PKG).Date=$(DATE)

RACE ?= 1
OPEN ?= $(shell command -v xdg-open 2>/dev/null || echo "open")

.PHONY: help \
        setup \
        build install run \
        test test\:cover \
        lint lint\:fix fmt vet cyclo \
        mod\:tidy mod\:verify \
        clean clean\:cache clean\:all \
        versions \
        tokens

#------------------------------------------------------------------------------
# Help
#------------------------------------------------------------------------------

help: ## Show available commands
	@awk '\
		/^#-+$$/ { next } \
		/^# [A-Za-z]/ { section = substr($$0, 3); next } \
		/^[a-zA-Z_:\\-]+:.*## / { \
			gsub(/\\:/, ":", $$0); \
			match($$0, /## /); \
			desc = substr($$0, RSTART + 3); \
			prefix = substr($$0, 1, RSTART - 1); \
			gsub(/: [^:]*$$/, "", prefix); \
			target = prefix; \
			targets[section] = targets[section] sprintf("  \033[36m%-22s\033[0m %s\n", target, desc); \
			order[section] = order[section] ? order[section] : ++count; \
		} \
		END { \
			for (i = 1; i <= count; i++) { \
				for (s in order) { \
					if (order[s] == i) { \
						if (i > 1) printf "\n"; \
						printf "\033[1m%s\033[0m\n", s; \
						printf "%s", targets[s]; \
					} \
				} \
			} \
		}' $(MAKEFILE_LIST)

#------------------------------------------------------------------------------
# Setup
#------------------------------------------------------------------------------

setup: ## Install required Go tools into ./bin (project-local)
	@mkdir -p $(BIN)
	GOTOOLCHAIN=$(TOOLCHAIN_FOR_TOOLS) GOBIN=$(abspath $(BIN)) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	GOTOOLCHAIN=$(TOOLCHAIN_FOR_TOOLS) GOBIN=$(abspath $(BIN)) go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
	GOTOOLCHAIN=$(TOOLCHAIN_FOR_TOOLS) GOBIN=$(abspath $(BIN)) go install github.com/fzipp/gocyclo/cmd/gocyclo@$(GOCYCLO_VERSION)
	@echo ""
	@echo "Setup complete: $(GOLANGCI), $(GOIMPORTS), $(GOCYCLO)"
	@echo ""

#------------------------------------------------------------------------------
# Build
#------------------------------------------------------------------------------

build: ## Build binary to ./bin/sheets-mcp with version ldflags
	@mkdir -p $(BIN)
	go build -o $(BINARY) -ldflags "$(LDFLAGS)" ./cmd/sheets-mcp

install: ## Run go install with same ldflags (installs to GOPATH/bin or GOBIN)
	go install -ldflags "$(LDFLAGS)" ./cmd/sheets-mcp

#------------------------------------------------------------------------------
# Run
#------------------------------------------------------------------------------

run: ## Run via go run (optional: ARGS="--flags")
	go run -ldflags "$(LDFLAGS)" ./cmd/sheets-mcp $(ARGS)

#------------------------------------------------------------------------------
# Test
#------------------------------------------------------------------------------

test: ## Run all tests (RACE=1 default; RACE=0 to disable -race)
	go test $(if $(filter 1,$(RACE)),-race,) ./...

test\:cover: ## Coverage report; opens HTML unless CI is set (override OPEN=...)
	go test $(if $(filter 1,$(RACE)),-race,) -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@if [ -z "$$CI" ]; then $(OPEN) coverage.html; else echo "Wrote coverage.html (CI set, skipping browser)"; fi

#------------------------------------------------------------------------------
# Lint and Format
#------------------------------------------------------------------------------

lint: ## Run golangci-lint
	@test -x $(GOLANGCI) || (echo "Run 'make setup' to install golangci-lint" && exit 1)
	$(GOLANGCI) run ./...

lint\:fix: ## Run golangci-lint with --fix
	@test -x $(GOLANGCI) || (echo "Run 'make setup' to install golangci-lint" && exit 1)
	$(GOLANGCI) run --fix ./...

fmt: ## Format with goimports (gofmt + import fixes; -l lists changed files, -w writes)
	@test -x $(GOIMPORTS) || (echo "Run 'make setup' to install goimports into ./bin" && exit 1)
	$(GOIMPORTS) -l -w .

vet: ## Run go vet
	go vet ./...

cyclo: ## Run gocyclo; run 'make setup' first
	@test -x $(GOCYCLO) || (echo "Run 'make setup' to install gocyclo" && exit 1)
	$(GOCYCLO) -over 10 .

#------------------------------------------------------------------------------
# Module
#------------------------------------------------------------------------------

mod\:tidy: ## Run go mod tidy
	go mod tidy

mod\:verify: ## Run go mod verify
	go mod verify

#------------------------------------------------------------------------------
# Clean
#------------------------------------------------------------------------------

clean: ## Remove built binary and coverage artifacts
	rm -f $(BINARY) coverage.out coverage.html

clean\:cache: ## Clear Go test cache
	go clean -testcache

clean\:all: clean ## Run clean plus remove ./bin (Go tools)
	rm -rf $(BIN)

#------------------------------------------------------------------------------
# Utilities
#------------------------------------------------------------------------------

tokens: ## Count tool definition tokens via Anthropic count_tokens API (requires ANTHROPIC_API_KEY and jq; sheets-mcp must be configured)
	@command -v jq >/dev/null 2>&1 || (echo "Error: jq is required" && exit 1)
	@test -n "$$ANTHROPIC_API_KEY" || (echo "Error: ANTHROPIC_API_KEY is not set" && exit 1)
	@TOOLS=$$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"count-tokens","version":"0.0.1"}}}\n{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}\n' \
		| timeout 5 go run ./cmd/sheets-mcp serve 2>/dev/null \
		| tail -1 \
		| jq '.result.tools | map({name: .name, description: .description, input_schema: .inputSchema})'); \
	BODY=$$(jq -n --argjson tools "$$TOOLS" \
		'{model: "claude-sonnet-4-6", tools: $$tools, messages: [{role: "user", content: "hi"}]}'); \
	RESULT=$$(curl -s https://api.anthropic.com/v1/messages/count_tokens \
		-H "x-api-key: $$ANTHROPIC_API_KEY" \
		-H "anthropic-version: 2023-06-01" \
		-H "content-type: application/json" \
		-d "$$BODY"); \
	echo "$$RESULT" | jq -r 'if .input_tokens then "Tool definition tokens: \(.input_tokens)" else "API error: \(.error.message)" | error end'

versions: ## Show Go and required tool versions
	@echo "Go: $$(go version)"
	@if test -x $(GOLANGCI); then $(GOLANGCI) version; else echo "golangci-lint: not installed (run make setup)"; fi
	@if test -x $(GOIMPORTS); then echo "goimports (module metadata):"; go version -m $(GOIMPORTS) 2>&1 | head -4; else echo "goimports: not installed (run make setup)"; fi
	@if test -x $(GOCYCLO); then echo "gocyclo (module metadata):"; go version -m $(GOCYCLO) 2>&1 | head -4; else echo "gocyclo: not installed (run make setup)"; fi
