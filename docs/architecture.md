# Architecture For Agents

This document is for coding agents working in this repo. It is intentionally short and change-oriented.

## Runtime Flow

1. `internal/cli` parses flags and environment, selects the formatter, constructs `filter.Set`, and wires dependencies.
2. `internal/app.Run` calls `Client.FetchRepo`.
3. `internal/github` builds the GraphQL request, translates the response into `model.Repo`, and may serve from SWR cache.
4. `internal/app.Run` resolves list filters, derives stack annotations through `internal/stacks`, and finalizes render context.
5. `internal/render` formats the already-prepared repo into `text`, `json`, or `toon`.
6. `internal/app` maps empty-state and error conditions to exit codes.

## Package Responsibilities

### `internal/cli`

- Owns flags, env vars, help text, spinner behavior, and local installation workflow assumptions.
- This is where default policy lives, such as defaulting author filters to `@me`.

### `internal/app`

- Owns orchestration, not product policy discovery.
- This is the only place that should combine fetch results, list filtering, stack derivation, and formatting.

### `internal/github`

- Owns request semantics, response translation, and cache behavior.
- Changes here are high-risk because query semantics and cache keying must remain aligned.
- If you change what data is fetched, audit tests, fixtures, and any user-facing claims about caching or filtering.

### `internal/filter`

- Separates fetch-time query filters from post-fetch list filters.
- If a filter has both roles, keep both explicit.

### `internal/stacks`

- Owns stack detection and derived `stackId` and `stackPos`.
- Renderers should consume these annotations, not recompute topology.

### `internal/render`

- Owns presentation only.
- Keep output formats consistent where they describe the same underlying field.
- Output changes often require fixture or golden updates.

### `internal/model`

- Shared schema for repo and PR data.
- Avoid putting workflow logic here.

## Stable Boundaries

- CLI policy should not leak into renderers.
- Render logic should not reach back into GitHub querying.
- Cache wrappers should depend on explicit inputs, not concrete implementation details.
- Tests should pin the behavior that users observe, not just internal helper behavior.

## High-Risk Change Types

- Fetch query changes
- Cache key changes
- Author filtering changes
- Help text and exit code changes
- Output shape changes in `json` or `toon`

## Safe Extension Points

- Add flags in `internal/cli` when the feature is clearly user-facing.
- Add filters in `internal/filter` when semantics can be stated clearly at fetch-time or list-time.
- Add derived presentation fields after data has been fetched and normalized in `internal/app`.

## Minimal Review Checklist

- Did fetch semantics change?
- Did cache behavior still match the fetch semantics?
- Did user-visible output or help text change?
- Were tests updated at the same layer as the behavior change?
