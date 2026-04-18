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
gh prs                      # human-readable (default)
gh prs --format json        # structured JSON to stdout
gh prs --format toon        # Token-Oriented Object Notation (agent-friendly)
gh prs -f toon              # same, short form
gh prs --debug              # log the actual GraphQL request/response to stderr
gh prs --no-cache           # bypass the disk cache
gh prs --cache-ttl 2m       # override the default 60s cache TTL
gh prs --help
```

## Formats

Pick the right output for the consumer:

| Format | When to use |
|--------|-------------|
| `text` | Default. Colorized TTY output with stacks as trees and OSC8-clickable PR numbers. |
| `json` | Piping into `jq`, dashboards, or any conventional tool. Same shape as the GraphQL source plus derived `stackId`/`stackPos` fields. |
| `toon` | Passing context to an LLM or coding agent. ~50% fewer bytes than JSON, with an explicit tabular schema the model reads in one glance. |

### Why TOON for agents

[TOON](https://toonformat.dev/) collapses uniform arrays of objects into a single header line plus one CSV-like row per item. Our PR list is exactly that shape, so the token win is real: for a typical 4-PR response, JSON is ~2.1 KB and TOON is ~1.0 KB. The header declares fields once:

```
prs[4]{number,title,url,isDraft,headRefName,baseRefName,additions,deletions,changedFiles,reviewDecision,ciState,mergeStateStatus,stackId,stackPos}:
  1001,"[ACME-100] Add feature foundation (1/4)","https://github.com/acme-org/widget/pull/1001",false,ACME-100/feature-1-foundation,main,515,59,18,REVIEW_REQUIRED,SUCCESS,BLOCKED,1,1/4
  1002,"[ACME-100] Add feature UI (2/4)","...",false,ACME-100/feature-2-ui,ACME-100/feature-1-foundation,342,27,12,null,SUCCESS,CLEAN,1,2/4
  ...
```

**Stack membership is inline.** Every PR row carries `stackId` (1-based stack index, or `null` for standalone) and `stackPos` (e.g. `"2/4"`). Agents don't need to walk `baseRefName`/`headRefName` chains to understand topology — it's a column lookup. `stackId`/`stackPos` are also added to `--format json` output for consistency.

Also honoured via `GH_PRS_FORMAT=<name>`.

**Caching.** Responses are cached to disk (platform cache dir, under `gh-prs/`)
with a 60s TTL by default. Repeat invocations within that window skip the
network entirely. Override with `--cache-ttl` or `GH_PRS_CACHE_TTL=2m`, or
disable with `--no-cache` / `GH_PRS_NO_CACHE=1`.

**Exit codes:** `0` ok · `1` gh/network · `2` not a GitHub repo · `3` no open PRs.

## Development

```bash
make help                      # list all targets
make install                   # build + install as the `gh prs` extension
make build                     # build ./gh-prs
make test                      # run tests (matches CI: -race -count=1)
make check                     # full CI gate: fmt-check + vet + lint + test
make run ARGS="--debug"        # go run . --debug
```

`make lint` requires [`golangci-lint`](https://golangci-lint.run/welcome/install/) on your PATH. CI installs it automatically.

## Release

Pushing a tag `vX.Y.Z` triggers [`cli/gh-extension-precompile@v2`](https://github.com/cli/gh-extension-precompile) to build cross-platform binaries and publish a GitHub Release.

## License

MIT
