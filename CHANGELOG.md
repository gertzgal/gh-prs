# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `--format <text|json|toon>` (short `-f`, env `GH_PRS_FORMAT`) selects the
  output format. `toon` emits Token-Oriented Object Notation — a compact,
  agent-friendly tabular format that produces ~50% fewer bytes than JSON on
  typical PR payloads while carrying an explicit field schema.
- `stackId` and `stackPos` (e.g. `"2/3"`) columns on every PR in machine
  formats. Inline columns make stack membership self-describing so agents
  don't have to re-derive topology from `baseRefName`/`headRefName`.
  Standalone PRs have `null` for both. Populated by `stacks.Annotate`.
- SWR (stale-while-revalidate) disk cache for GraphQL responses. Serves stale
  data instantly while refreshing in the background. Default TTL increased to
  300s (was 60s). Process lingers up to 3s to allow background refresh to
  complete. Cache is account-scoped to prevent cross-viewer contamination.
- `--no-cache` flag (also `GH_PRS_NO_CACHE=1`) to bypass the cache.
- `--cache-ttl <duration>` flag (also `GH_PRS_CACHE_TTL`) to override the TTL.
- `cacheAgeMs`, `fromCache`, and `isStale` fields in JSON and TOON output.
- Text footer now shows `● 0pt` on cache hit and displays cache age
  (`cached Xm ago`) or staleness (`stale Xm ago`).

### Changed
- **Breaking:** `--json` removed — use `--format json` instead.
- GraphQL query switched from `search` to `repository.pullRequests` with
  `orderBy: {field: UPDATED_AT, direction: DESC}` to restore chronological
  ordering and simplify the query.
- `@me` author filter is now resolved post-fetch using the viewer login from
  the GraphQL response, enabling accurate client-side filtering with a single
  shared cache key.
- `--debug` now logs the actual GraphQL request/response — URL, headers, query
  body, variables, response body, timing — via go-gh's httpretty logger. The
  previous static "REST equivalent" block is still printed above it for
  orientation.

## [0.1.0] - 2026-04-18

### Added
- Initial release as a `gh` CLI extension.
- Compact overview of the current user's open PRs in the current repo: stacks
  rendered as trees, standalone PRs flat.
- Single GraphQL round-trip for PR, CI, and review status.
- OSC8 clickable PR numbers in TTY output.
- `--json` flag for machine-readable output.
- `--debug` flag printing equivalent `gh api` calls to stderr.
- Precompiled cross-platform binaries (darwin/linux/windows × amd64/arm64) via
  `cli/gh-extension-precompile`.

[Unreleased]: https://github.com/gertzgal/gh-prs/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/gertzgal/gh-prs/releases/tag/v0.1.0
