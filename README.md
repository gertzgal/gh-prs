# gh-prs

[![ci](https://github.com/gertzgal/gh-prs/actions/workflows/ci.yml/badge.svg)](https://github.com/gertzgal/gh-prs/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/gertzgal/gh-prs?sort=semver)](https://github.com/gertzgal/gh-prs/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/gertzgal/gh-prs.svg)](https://pkg.go.dev/github.com/gertzgal/gh-prs)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A `gh` CLI extension that shows a compact overview of the current user's open PRs in the current repo — stacks rendered as trees, standalone PRs flat. Single GraphQL round-trip, live CI/review status, OSC8 clickable PR numbers. Ships as a precompiled Go binary.

## Install

```
gh extension install gertzgal/gh-prs
```

Needs `gh` authenticated (`gh auth login`). No other runtime — the binary is self-contained.

## Usage

```
gh prs                  # human-readable output
gh prs --json           # JSON to stdout (no colors, no spinner)
gh prs --debug          # print equivalent gh api REST calls to stderr
gh prs --help
```

Exit codes: `0` success · `1` gh/network failure · `2` not in a GitHub repo · `3` no authored open PRs.

## Local development

```bash
# Install from this directory (symlinks the repo to ~/.local/share/gh/extensions/gh-prs/)
gh extension install .

# Rebuild the binary after editing Go source:
go build -o ./gh-prs ./cmd/gh-prs
```

Tests:

```bash
go test ./...
go test ./... -cover
```

Cross-compile matrix:

```bash
for os in darwin linux windows; do
  for arch in amd64 arm64; do
    ext=""; [[ "$os" == "windows" ]] && ext=".exe"
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
      -o "/tmp/gh-prs-cross/$os-$arch$ext" ./cmd/gh-prs
  done
done
```

## Release

Tag-push releases are built by [`cli/gh-extension-precompile@v2`](https://github.com/cli/gh-extension-precompile) (see `.github/workflows/release.yml`). Pushing a tag `vX.Y.Z` triggers a cross-platform build and uploads `dist/{os}-{arch}[.exe]` binaries to a GitHub Release; `gh extension install gertzgal/gh-prs` then picks the correct binary for the host platform.

## Layout

- `cmd/gh-prs/main.go` — entry point.
- `internal/github/` — GraphQL client (single round-trip; uses `cli/go-gh/v2`).
- `internal/stacks/` — base/head stack grouping.
- `internal/render/` — text + JSON formatters, ANSI + OSC8 helpers, status glyphs.
- `internal/cli/` — cobra root, env/TTY detection, spinner, debug output, exit codes.
- `internal/app/` — orchestration.
- `testdata/` — GraphQL fixture responses + golden expected outputs.

## History

Originally written in Bun/TypeScript. Rewritten in Go (April 2026) to ship as a single ~7 MB precompiled binary with zero runtime dependencies, matching the ecosystem-standard path for `gh` extensions (see `github/gh-skyline`, `dlvhdr/gh-dash`).
