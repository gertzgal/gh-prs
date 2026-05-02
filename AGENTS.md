# AGENTS.md

Agent-only operating guidance for this repository.

## Repository Purpose

- This repo ships `gh prs`, a GitHub CLI extension that shows open PRs for the current repo.
- The product contract is: fetch the matching PR set, derive stack topology once, then render the same data in `text`, `json`, or `toon`.
- Preserve output correctness over implementation cleverness.

## Canonical Commands

- Build: `make build`
- Test: `make test`
- Full gate: `make check`
- Install locally as the extension: `make install`
- Run locally: `make run ARGS="--help"`
- Demo repo smoke test: from `../gh-prs-demo-repo`, run `gh prs`

## Working Rules

- Prefer the smallest change that fully solves the requested problem.
- Touch only the packages required for the task.
- Do not silently change fetch semantics, cache semantics, or output shape.
- If behavior changes for users, update tests and any affected docs or changelog entries in the same change.
- If you change command behavior, check `README.md`, `CHANGELOG.md`, and help text in `internal/cli/root.go`.

## Verification Rules

- For code changes, run focused tests first if useful, then `make test`.
- Run `make check` before finishing any non-trivial code change.
- For output changes, verify the installed extension from `../gh-prs-demo-repo`.
- If you could not run a required check, say so explicitly.

## Package Boundaries

- `internal/cli`: flag parsing, env handling, help text, extension-facing UX.
- `internal/app`: orchestration only; fetch, apply list filters, annotate stacks, format, map exit codes.
- `internal/github`: GraphQL query construction, response translation, cache and SWR behavior.
- `internal/filter`: query filters and post-fetch list filters.
- `internal/stacks`: derive stack topology from `baseRefName` and `headRefName`.
- `internal/render`: presentation only. No fetching, no CLI parsing, no stack derivation policy.
- `internal/model`: shared data types and errors.

## Invariants

- `github.Client.FetchRepo(ctx, filters)` must honor fetch-time query filters.
- Cache entries must not collapse distinct effective query views.
- Stack annotations are derived centrally in `app.Run`, not inside renderers.
- Shared fields should stay consistent across `text`, `json`, and `toon`.
- `text` is the human contract; `json` and `toon` are machine contracts and should remain predictable.

## Common Change Patterns

- New CLI flag: update `internal/cli`, help text, tests, and docs if user-visible.
- New fetched field: update `internal/github/query.go`, `internal/model`, renderers, fixtures, and tests together.
- New filter: extend `internal/filter`, wire it in `internal/cli`, and keep fetch-time vs list-time behavior explicit.
- Render change: update renderer tests and any golden fixtures affected by output shape.

## Architecture Reference

- Read `docs/architecture.md` before making non-trivial changes.
