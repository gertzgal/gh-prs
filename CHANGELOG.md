# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
