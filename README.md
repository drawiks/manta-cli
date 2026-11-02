# manta-cli

Standalone Go CLI that parses a single Dota 2 replay (`.dem` / `.dem.bz2` /
`.dem.zst`) with [`manta`](https://github.com/dotabuff/manta) and prints
per-player statistics as one JSON object to stdout.

The output is a drop-in match for `odota-cli`'s `Match` JSON: same field
names, same types, same hero naming, same `player_id` (0..9) and the same
`duration_sec`/`buff_duration`/`duration` conventions. No Docker, Java or
`odota/parser` is required.

## Usage

```
manta_cli <match.dem|match.dem.bz2|match.dem.zst> [--stdin] [--verbose] [--dump-fields]
```

- `.dem` is read directly; `.dem.bz2` is decompressed on the fly with the
  stdlib `compress/bzip2`; `.dem.zst` is supported via
  `klauspost/compress/zstd`. The format is detected from magic bytes, so a
  zstd stream named `.dem.bz2` (Valve's current naming) works too.
- `--stdin` reads the replay from standard input and takes no file argument.
- `--verbose` prints a human-readable table to stderr after the JSON.
- `--dump-fields` dumps the `CDOTA_PlayerResource`, the team-data entities and
  a hero entity to stderr for re-spiking field names after a Dota patch.

Exit codes: `0` success, `1` parse error, `2` usage error. stdout carries only
the JSON; diagnostics go to stderr.

```sh
manta_cli 9064911131.dem > match.json
manta_cli 9053590215.dem --verbose
cat 9065268582.dem | manta_cli --stdin > match.json
```

## Build

```sh
go build -o manta_cli .
```

Pure Go, statically linkable:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o manta_cli .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o manta_cli_arm64 .
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
      "heal_duration": 0,
      "heal_value": 0,
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
`break_sources`, `disarm_sources` and `heal_sources` are omitted when empty.

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
ADD→REMOVE using event times, self-cast modifiers are ignored, heals are ally
HEAL events plus HoT ticks attributed to the caster, and `stun_duration` is the
scoreboard `m_fStuns` total.

## Validation

Parity was checked against `odota-cli` v0.2.1 on five modern replays. All
fields match exactly except for these sampling/rounding residuals:

| field | residual | cause |
|---|---|---|
| `time_dead` | ±2 s | death/respawn transitions are sampled once per second |
| `networth` | ±1 | last interval can be one tick before clarity's sample |
| `gpm` | ±1 | last-interval gold can be one tick before clarity's sample |

The parity harness is `tests/replay/parity_test.go`. It expects the external
corpus (`MANTA_REPLAY_DIR`, default `../../../replays`) with `<id>.dem` and
`<id>.out.json` pairs; it skips when the corpus is absent.

## Tests

```sh
go test ./...
MANTA_REPLAY_DIR=/path/to/replays go test ./tests/replay -run TestParityWithOdotaCLI -v
```

- `internal/stats` — extractor/aggregator/role unit tests.
- `tests/cli` — CLI exit codes, stdin and dump modes.
- `tests/replay` — external parity harness.

## License

MIT — see [LICENSE](LICENSE).
