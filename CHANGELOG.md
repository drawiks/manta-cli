# Changelog

All notable changes to this project are documented here. The format follows Keep a Changelog and the project uses Semantic Versioning.

## Unreleased

### Added

- Product CLI with `--version`, `--output`, `--quiet`, `--compact`, `--stdin` and `-` input.
- Replay adapter, typed events, aggregator, catalog, roles, output model and app packages.
- Lefthook pre-commit, commit-msg and pre-push checks.
- CI, GoReleaser, SBOM, Docker image and JSON Schema.
- Resource benchmark and parity harness.

### Changed

- The module path is `github.com/drawiks/manta-cli`.
- Manta is pinned to v1.5.2.

## 0.1.0 - 2026-11-02

### Added

- First product release: parses a Dota 2 replay into the odota-cli match JSON.
