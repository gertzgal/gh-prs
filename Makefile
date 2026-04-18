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

lint: ## Run golangci-lint (requires golangci-lint on PATH)
	golangci-lint run ./...

check: fmt-check vet lint test ## Run the full CI gate locally

install: build ## Install this repo as the `gh prs` extension (symlink)
	gh extension install .

uninstall: ## Remove the locally installed `gh prs` extension
	gh extension remove $(EXT_NAME)

dist: ## Cross-compile release binaries into ./dist (delegates to script/build.sh)
	./script/build.sh

clean: ## Remove build artifacts
	rm -f ./$(BINARY)
	rm -rf ./dist
