# manta-cli

Standalone Go CLI that parses a single Dota 2 replay (`.dem` / `.dem.bz2` /
`.dem.zst`) with [`manta`](https://github.com/dotabuff/manta) and prints
aggregated per-player statistics as one JSON object to stdout.

## Usage

```
manta_cli <match.dem|match.dem.bz2|match.dem.zst> [--stdin] [--verbose] [--dump-fields]
```

- `.dem` is read directly; `.dem.bz2` is decompressed on the fly with the
  stdlib `compress/bzip2` (streaming, nothing written to disk). Valve now
  stores fresh replays as zstd even when the file is named `.dem.bz2`, so
  `.dem.zst` is also transparently supported via `klauspost/compress/zstd`.
- `--stdin` reads the replay stream from standard input instead of a file,
  takes no file argument, and applies to both the JSON and `--dump-fields`
  outputs. The format is detected from the stream's magic bytes (raw, zstd or
  bzip2). Since there is no filename, `match_id` comes only from the replay's
  own `CDemoFileInfo` metadata.
- `--verbose` prints a human-readable stats table to **stderr** after the JSON.
- `--dump-fields` prints every field of the `CDOTA_PlayerResource` and the
  `CDOTA_DataRadiant`/`CDOTA_DataDire` team-data entities to **stderr** (spike
  for re-validating field names after Dota patches).

Streams and exit codes:

| code | meaning                                   |
|------|-------------------------------------------|
| 0    | success                                   |
| 1    | parse error (unreadable/corrupt replay, `CDOTA_PlayerResource` not found) |
| 2    | usage error (no file given, unknown flag, `--stdin` with a file) |

stdout carries only the JSON; all errors are prefixed with `error:` on stderr.

```sh
$ manta_cli 8552595443.dem.bz2 > match.json
$ manta_cli 8927186729.dem.zst --verbose
$ cat 8926354517.dem | manta_cli --stdin > match.json
$ echo $?   # 0
```

## Build

```sh
go build -o manta_cli .
```

The full stack (manta + protobuf + snappy + bzip2/zstd) is pure Go, so the
binary is statically linkable and cross-compiles to any platform:

```sh
# fully static linux binaries, no Go runtime needed on the target
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o manta_cli .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o manta_cli_arm64 .
```

## Output schema

```json
{
  "match_id": 7047839963,
  "duration_sec": 2612,
  "radiant_win": true,
  "players": [
    {
      "steam_id": 76561198123456789,
      "player_id": 0,
      "hero_id": 62,
      "hero": "Slark",
      "team": "radiant",
      "name": "player name",
      "kills": 8, "deaths": 3, "assists": 12,
      "level": 25,
      "last_hits": 230,
      "networth": 18900,
      "gpm": 512, "xpm": 640,
      "healing": 5200,
      "hero_damage": 18750,
      "damage_taken": 12400,
      "tower_damage": 1800,
      "time_dead": 240.56671,
      "stun_duration": 14.3,
      "gold_spent_wards": 900,
      "gold_spent_smoke": 200,
      "gold_spent_dust": 80,
      "camps_stacked": 0,
      "creeps_stacked": 0,
      "rune_pickups": 2,
      "first_blood": true
    }
  ]
}
```

Field order is fixed; `match_id` is omitted when unknown (0).

### Data sources and known limitations

- `CDOTA_PlayerResource` snapshot (captured during parse, mutated in place by
  manta): steam_id, team, team slot, name, KDA, level, selected hero.
- `CDOTA_DataRadiant`/`CDOTA_DataDire` team-data entities (scoreboard, the
  same source OpenDota/Valve's match API uses): GPM/XPM, tower damage, stun
  duration, last hits, net worth, healing, rune pickups, camps/creeps stacked.
  Per-player values are read via `m_vecDataTeam.<teamSlot>.m_*` (camps/creeps
  stacked are team-wide and copied to every player, mirroring OpenDota).
- GPM/XPM are computed exactly like the in-game scoreboard:
  `floor(totalEarned * 60 / gameTime)`, where the divisor is the un-truncated
  float game clock (e.g. 2463.5334) — not `duration_sec`'s truncation. This
  reproduces every OpenDota `gold_per_min`/`xp_per_min` value on the reference
  match.
- `duration_sec` is `int(postGame - gameInProgress)` (truncated); `radiant_win`
  comes from the destroyed ancient's team.
- `stun_duration`, `time_dead` and `healing` are serialized with float32
  round-trip formatting (`12.433472`, not `12.4334716796875`), matching the
  match API exactly.
- Combat-log fallback: when the team-data entities are absent (old replays),
  GPM/XPM/tower damage/healing/last hits/net worth/stuns fall back to
  `dota_combatlog` aggregation; first blood is credited from the `FIRST_BLOOD`
  (type 18) combat-log event when present and falls back to the first hero
  `DEATH` otherwise (the type-18 event carries no attacker index, so its credit
  is taken from the first hero death at/after that tick).
- Two combat-log id-spaces are bridged: pid (even `m_iPlayerID` of hero
  entities, also the `PLAYERSTATS` target) vs xpID (the `PLAYERSTATS` value,
  used in `attacker`/`target` of damage/heal/gold/xp events).
- Field names/layout inside `CDOTA_PlayerResource` drift between Dota patches;
  use `--dump-fields` to re-spike after big updates.
- `hero_damage` / `damage_taken`: summed from combat-log `DAMAGE` events
  (target hero, illusions excluded). Self-inflicted damage is excluded from
  `hero_damage` but **included** in `damage_taken`, mirroring the end-game
  summary and OpenDota. `damage_taken` matches OpenDota exactly for all 10
  players on the reference match; `hero_damage` agrees on 6/10 (reference) and
  8/10 (engine-1 replay). The residuals (e.g. Pudge +9128, Rubick −1123) come
  from damage sources the combat log records under a different owner
  (illusions, stolen/reflected damage) — a documented approximation.
- `time_dead` reproduces OpenDota's `life_state_dead`: for every hero `DEATH`
  it sums `min(m_flRespawnTime, buyback, postGame) − deathClock`.
  `m_flRespawnTime` never resets on buyback (it holds the scheduled respawn),
  so each buyback caps its own window; heroes still dead when the game ends are
  capped at `postGame`. When the post-game clock is missing (0), the window
  falls back to the respawn cap alone. All 10 players on the reference match
  are within ±2 s of OpenDota (the 30-tick quantization). Deaths and respawns
  are paired by index within a slot, which stays exact as long as every death
  produces a respawn update (documented approximation).
- `gold_spent_wards` = scoreboard `m_iSentryWardsPlaced` × 50, matching the
  in-game ward-gold display (sentry wards are 50 gold and placed by their
  buyer; observer wards are free on current patches and count as 0). Gem of
  True Sight is unobservable in a replay and counted as 0.
- `gold_spent_smoke` = scoreboard `m_iSmokesUsed` × 50. It counts smokes *used*,
  so smokes bought but never used are missed (OpenDota's purchase log counts
  those, e.g. Tusk 0 vs 100 gold on the reference match).
- `gold_spent_dust` = combat-log item-use events × 80; dust held at the end of
  the game (never used) is not counted.
- `hero_healing` cannot be reproduced from replay data for every player: on
  the reference match OpenDota reports healing totals (Phoenix 9440, Dazzle
  15328, Legion Commander 259, Primal Beast 244) that match neither the
  scoreboard's `m_fHealing` nor any sum of combat-log HEAL events (LC/PB emit
  no heal events to other heroes at all). manta reports the scoreboard value
  `m_fHealing`, which agrees with OpenDota on 6/10 players.
- The vendored `manta` package logs parser diagnostics straight to stdout when
  the `DEBUG` (level 1) or `TRACE` (level 10) environment variables are set;
  there is no `manta.DEBUG` variable. Run the CLI with an unset environment to
  keep stdout JSON-only (the CLI itself never writes to stdout except JSON).
- Replay files whose name starts with `-` (e.g. `-12345.dem`) are rejected by
  the CLI's flag parser; rename the file to parse it.

## Tests

Synthetic fixtures exercise the aggregation and the JSON contract against a
golden file; CLI exit codes (including the `--stdin` mode) are covered
separately. When the repo's replay fixtures are present, integration tests
also push a real `.dem` through stdin and check the `match_id` fallback:

```sh
go test ./...
```

## License

MIT — see [LICENSE](LICENSE).
