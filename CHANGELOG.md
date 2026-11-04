# Changelog

All notable changes to this project are documented here. The format follows Keep a Changelog and the project uses Semantic Versioning.

## 0.5.3 - 2026-11-04

### Added

- `mana_restored` per player, counting mana restored to allied heroes. It is sourced from `DOTA_COMBATLOG_MANA_RESTORED`; current HLTV replays may not emit type 24.
- Save sources for Shadow Demon Disruption and Linken's active spellblock.
- Shield sources for Pipe, Pavise, Crimson Guard and Solar Crest.
- Haste sources for Marci Rebound, Grimstroke Ink Swell and the current Drum active.
- `buff_stats` sources for Sven Warcry, Troll Battle Trance with the level 20 talent, and Boots of Bearing.

### Fixed

- Corrected the Omniknight Repel, Pipe, Pavise, Marci Bodyguard, Chen Divine Favor and Drum modifier keys.
- Removed self-only Bounty Hunter Wind Walk from `invisibility`.
- Removed the empty `mana_restore` buff category.

## 0.5.2 - 2026-11-04

### Fixed

- Undying's Tombstone Bunker counts as a save through `modifier_undying_tombstone_bunker_friendly`.

## 0.5.1 - 2026-11-04

### Fixed

- Oracle False Promise is counted as a save through `modifier_oracle_false_promise_timer`.
- Wind Waker is counted as a save.
- Eul's Cyclone, Pugna Decrepify, Underlord Dark Rift and Abaddon Borrowed Time are no longer save buffs.
- Disperser's movespeed buff is counted as a purge application instead of a save.

### Notes

- These corrections intentionally differ from `odota-cli` for the affected modifiers. See `docs/accuracy.md`.

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
