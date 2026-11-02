# Changelog

All notable changes to this project are documented here. The format follows Keep a Changelog and the project uses Semantic Versioning.

## 0.5.0 - 2026-11-02

### Added

- Public `manta` package with `ParseFile` and `ParseReader`.
- Public `stats` package with `Match`, `Player` and `SourceEntry`.
- `ErrNotReplay` and `ErrNoPlayerResource` error sentinels.

### Changed

- The command moved from the module root to `cmd/manta-cli`; install with `go install github.com/drawiks/manta-cli/cmd/manta-cli@latest`.
- The internal output model was renamed to `stats.Match` and made public.

### Notes

- `v0.5.0` is the first public API release and a preview. The API shape is frozen for `v0.5.x`; formal Go module compatibility starts at `v1.0.0`.

## 0.4.0 - 2026-11-02

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
