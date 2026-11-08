# manta-cli

Go CLI and library that parse a single Dota 2 replay (`.dem` / `.dem.bz2` /
`.dem.zst`) with [`manta`](https://github.com/dotabuff/manta) and prints
per-player statistics as one JSON object to stdout.

The output is a drop-in match for `odota-cli`'s `Match` JSON: same field
names, same types, same hero naming, same `player_id` (0..9) and the same
`duration_sec`/`buff_duration`/`duration` conventions, plus a `mana_restored`
extension. Save, shield, haste and purge classification includes documented
corrections; see `docs/accuracy.md`. No Docker, Java or `odota/parser` is
required.

## Install

```sh
go install github.com/drawiks/manta-cli/cmd/manta-cli@latest
```

Prebuilt binaries and checksums are published on the releases page. A Docker image is published as `ghcr.io/drawiks/manta-cli`.

## Usage

```
manta-cli <replay> [--stdin] [--output PATH] [--verbose] [--quiet] [--compact] [--dump-fields]
manta-cli --version
```

- `.dem` is read directly; `.dem.bz2` is decompressed on the fly with the
  stdlib `compress/bzip2`; `.dem.zst` is supported via
  `klauspost/compress/zstd`. The format is detected from magic bytes, so a
  zstd stream named `.dem.bz2` (Valve's current naming) works too.
- `--stdin` reads the replay from standard input and takes no file argument;
  `-` is the same input.
- `--output PATH` writes the JSON atomically instead of printing it.
- `--verbose` prints a human-readable table to stderr after the JSON.
- `--quiet` suppresses the verbose table and other non-error diagnostics.
- `--compact` emits compact JSON.
- `--dump-fields` dumps the `CDOTA_PlayerResource`, the team-data entities and
  a hero entity to stderr for re-spiking field names after a Dota patch.
- `--version` prints the binary, commit, build date, Go and manta versions.

Exit codes: `0` success, `1` parse or I/O error, `2` usage error, `130`
interrupted. stdout carries only the JSON; diagnostics go to stderr.

```sh
manta-cli 9064911131.dem > match.json
manta-cli 9053590215.dem --verbose
manta-cli 9053590215.dem --output match.json
cat 9065268582.dem | manta-cli --stdin --compact
manta-cli --version
```

## Library

The same parser is available as a Go package.

```go
import "github.com/drawiks/manta-cli"

m, err := manta.ParseFile(ctx, "match.dem")
if err != nil {
    return err
}
fmt.Println(m.MatchID, len(m.Players))
```

`manta.ParseReader(ctx, r)` reads from an `io.Reader` and does not close it.
Both functions detect `.dem`, `.dem.bz2` and `.dem.zst` from magic bytes.
`manta.Match` is an alias for `stats.Match`; the field documentation is in
`stats`. `ParseFile` uses the leading digits of the file name as a fallback
match ID.

Errors are classified with `errors.Is`:

```go
if errors.Is(err, manta.ErrNotReplay) {
    // the input is not a Source 2 replay
}
if errors.Is(err, manta.ErrNoPlayerResource) {
    // the replay has no CDOTA_PlayerResource
}
```

The public API is documented in `docs/api.md`. `v0.5.0` is the first public
API release; the shape is frozen for `v0.5.x`, and formal semantic-versioning
compatibility starts at `v1.0.0`.

## Build

```sh
make build
```

The binary is pure Go and statically linkable. GoReleaser builds the six release binaries; the Docker image uses `scratch` and a non-root user.

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o manta-cli ./cmd/manta-cli
```

## Output schema

```json
{
  "match_id": 9064911131,
  "duration_sec": 2420.5334,
  "radiant_win": false,
  "players": [
    {
      "steam_id": 76561198835005953,
      "player_id": 0,
      "hero_id": 46,
      "hero": "templar_assassin",
      "team": "radiant",
      "name": "iLLeGGal",
      "position": 1,
      "lane": "safe",
      "kills": 4,
      "deaths": 4,
      "assists": 5,
      "level": 23,
      "last_hits": 440,
      "networth": 19861,
      "gpm": 658,
      "xpm": 884,
      "healing": 0,
      "mana_restored": 0,
      "hero_damage": 14380,
      "damage_taken": 12767,
      "tower_damage": 627,
      "time_dead": 265,
      "stun_duration": 0,
      "buff_duration": 0,
      "fear_duration": 0,
      "roots_duration": 0,
      "leash_duration": 0,
      "trap_duration": 0,
      "taunt_duration": 0,
      "silence_duration": 0,
      "break_duration": 0,
      "disarm_duration": 0,
      "gold_spent_wards": 0,
      "gold_spent_smoke": 0,
      "gold_spent_dust": 0,
      "gold_lost": 1735,
      "camps_stacked": 0,
      "creeps_stacked": 0,
      "rune_pickups": 5,
      "wisdoms_captured": 0,
      "watchers_captured": 0,
      "lotuses_gathered": 0,
      "first_blood": true,
      "courier_kills": 0
    }
  ]
}
```

`position`/`lane` are omitted when the first 600 seconds do not contain enough
position samples. `stun_sources`, `buff_sources`, `fear_sources`, `root_sources`,
`leash_sources`, `trap_sources`, `taunt_sources`, `silence_sources`,
`break_sources` and `disarm_sources` are omitted when empty.

## Where the numbers come from

Everything is decoded from the replay by manta:

- **Epilogue**: `CDemoFileInfo` (`match_id`, `playback_time`, `game_winner`,
  player roster). This is also the `duration_sec` and `radiant_win` source.
- **Players and slots**: `CDOTA_PlayerResource` scanned exactly like
  `odota/parser` (indices 0..29, team 2/3, team slot, steam id, selected hero),
  so `player_id` is the 0..9 slot and hero names are `npc_dota_hero_*` slugs.
- **Intervals (1/s)**: player resource KDA/level, team-data scoreboard fields
  (`m_iTotalEarnedGold/XP`, `m_iLastHitCount`, `m_iNetWorth`, `m_fStuns`,
  camps/creeps/runes, wisdoms/watchers/lotuses, sentry wards), hero position
  (`CBodyComponent.m_cellX/Y` + `m_vecX/Y`) and `m_lifeState`.
- **Combat log**: damage, healing, modifier add/remove, purchases, gold
  reasons, deaths, ability casts. `CombatLogNames` resolves names and purchase
  item ids. GPM/XPM use the last interval and its game time; `time_dead` is
  the summed interval dead time, matching `odota-cli`'s interval model.
- **Chat events**: first blood and Tormentor shard grants.
- **Ability/item entities**: talent-gated purges and starting inventory (for
  starting smoke/dust).
- **Roles**: lane/position from the first 600 seconds' world positions plus
  networth/last-hit/XP farm at 600 s, with the same 2-1-2 heuristic as
  `odota-cli`.

Tracked effect semantics follow `odota-cli`: buffs/debuffs are paired
ADD→REMOVE using event times, self-cast modifiers are ignored, `healing` is the
total of ally HEAL events plus HoT ticks attributed to the caster, and
`stun_duration` is the scoreboard `m_fStuns` total.

## Validation

Parity was checked against `odota-cli` v0.2.1 on five modern replays. Every field matched exactly for v0.5.0. Later catalog corrections, `mana_restored`, and client-sourced `gold_lost`/`tower_damage` intentionally differ; `docs/accuracy.md` lists the known differences.

The parity harness is `tests/replay/parity_test.go`. It expects the external
corpus (`MANTA_REPLAY_DIR`, default `../../../replays`) with `<id>.dem` and
`<id>.out.json` pairs; it skips when the corpus is absent.

## Development

```sh
make tools
make hooks
make check
```

`make check` runs formatting, lint, tests, `go vet` and the comment rules. `make parity` needs the external corpus:

```sh
MANTA_REPLAY_DIR=/path/to/replays make parity
```

Hooks run `gofumpt`, the fast linter and the comment rules on commit, check the commit message, and run the full test suite on push. CI is authoritative and runs the same checks plus `govulncheck`.

- `stats` — public match, player and source types.
- `manta` — public `ParseFile` and `ParseReader` facade.
- `cmd/manta-cli` — command wiring.
- `internal/aggregate`, `internal/catalog`, `internal/roles` — pure statistics and domain tests.
- `internal/replay` — decompression, fuzz and the manta adapter.
- `internal/cli` — flags, stdin, output and exit codes.
- `tests/replay` — external parity harness.

See `docs/api.md`, `docs/architecture.md`, `docs/accuracy.md`, `docs/development.md` and `docs/schema.md`.

## License

MIT — see [LICENSE](LICENSE).
