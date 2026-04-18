# gh-prs

[![ci](https://github.com/gertzgal/gh-prs/actions/workflows/ci.yml/badge.svg)](https://github.com/gertzgal/gh-prs/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/gertzgal/gh-prs?sort=semver)](https://github.com/gertzgal/gh-prs/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/gertzgal/gh-prs.svg)](https://pkg.go.dev/github.com/gertzgal/gh-prs)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> Your open PRs in the current repo — stacks as trees, standalone flat.
> One GraphQL round-trip, live CI/review status, OSC8-clickable PR numbers.

## Demo

<!-- DEMO_OUTPUT_START -->

```
acme/widgets · main · @octocat

  stack · 2 PRs

  ┬ #1042  ✓ ci  ● review  +320-45  feat: billing webhook receiver (1/2)  1/2
  │          feat/billing-webhooks-1-receiver
  └ #1043  ✓ ci  ○ review  +180-12  feat: wire billing webhooks to notifier (2/2)  2/2
             feat/billing-webhooks-2-notifier

  stack · 3 PRs

  ┬ #1058  ✓ ci  ● review  +512-88  refactor: session store (1/3)  1/3
  │          refactor/session-1-store
  ├ #1059  ✓ ci  ○ review  +240-30  refactor: session middleware (2/3)  2/3
  │          refactor/session-2-middleware
  └ #1060  ✗ ci  ○ review  +95-14  refactor: session cleanup (3/3)  3/3
             refactor/session-3-cleanup

  standalone · 2 PRs

  #1063  ✓ ci  ● review  +60-8  fix: off-by-one in pagination cursor
           fix/pagination-cursor

  #1071  ✓ ci  ○ review  +140-22  chore: bump go to 1.23
           chore/bump-go-1.23

  980ms · ● 1pt · 4999 remaining
```

<!-- DEMO_OUTPUT_END -->

## Install

```
gh extension install gertzgal/gh-prs
```

Requires an authenticated `gh` (`gh auth login`). The binary is self-contained.

## Usage

```
gh prs                    # human-readable
gh prs --json             # JSON to stdout
gh prs --debug            # log the actual GraphQL request/response to stderr
gh prs --no-cache         # bypass the disk cache
gh prs --cache-ttl 2m     # override the default 60s cache TTL
gh prs --help
```

**Caching.** Responses are cached to disk (platform cache dir, under `gh-prs/`)
with a 60s TTL by default. Repeat invocations within that window skip the
network entirely. Override with `--cache-ttl` or `GH_PRS_CACHE_TTL=2m`, or
disable with `--no-cache` / `GH_PRS_NO_CACHE=1`.

**Exit codes:** `0` ok · `1` gh/network · `2` not a GitHub repo · `3` no open PRs.

## Development

```bash
gh extension install .                # symlink this repo as the extension
go build -o ./gh-prs .                # rebuild after edits
go test ./... -cover                  # tests
```

## Release

Pushing a tag `vX.Y.Z` triggers [`cli/gh-extension-precompile@v2`](https://github.com/cli/gh-extension-precompile) to build cross-platform binaries and publish a GitHub Release.

## License

MIT
