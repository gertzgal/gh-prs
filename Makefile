# gh-prs — Makefile
#
# Run `make` (no args) or `make help` to see all targets.
# Every command the CI gate runs is reachable through `make check`.

SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

BINARY := gh-prs
EXT_NAME := prs
ARGS ?=

GOLANGCI_LINT_VERSION := v2.11.4

.PHONY: help build run test cover fmt fmt-check vet lint check install uninstall dist clean

help: ## Print this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target>\n\nTargets:\n"} \
	/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build ./gh-prs for the host platform
	go build -o ./$(BINARY) .

run: ## Run via `go run` — pass flags with ARGS, e.g. `make run ARGS="--debug"`
	go run . $(ARGS)

test: ## Run tests (matches CI: -race -count=1)
	go test ./... -race -count=1

cover: ## Run tests with coverage summary
	go test ./... -cover

fmt: ## Format all Go files in place
	gofmt -w .

fmt-check: ## Fail if any Go files are unformatted (matches CI)
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
	  echo "Unformatted files:"; echo "$$unformatted"; exit 1; \
	fi

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (auto-installs pinned version to GOPATH/bin if missing)
	@installed=$$(golangci-lint version --format short 2>/dev/null || true); \
	if [ "$$installed" != "$(GOLANGCI_LINT_VERSION)" ]; then \
	  echo "installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
	  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | \
	    sh -s -- -b "$$(go env GOPATH)/bin" $(GOLANGCI_LINT_VERSION); \
	fi
	@PATH="$$(go env GOPATH)/bin:$$PATH" golangci-lint run ./...

check: fmt-check vet lint test ## Run the full CI gate locally

install: build ## Build and install the `gh prs` extension (idempotent)
	-gh extension remove $(EXT_NAME)
	gh extension install .

install-release: ## Remove local build and install the published release from GitHub
	-gh extension remove $(EXT_NAME)
	gh extension install gertzgal/gh-prs

uninstall: ## Remove the locally installed `gh prs` extension
	gh extension remove $(EXT_NAME)

dist: ## Cross-compile release binaries into ./dist (delegates to script/build.sh)
	./script/build.sh

clean: ## Remove build artifacts
	rm -f ./$(BINARY)
	rm -rf ./dist

demo: ## Run against the demo repo at ../gh-prs-demo-repo
	cd ../gh-prs-demo-repo && ../gh-prs/gh-prs
