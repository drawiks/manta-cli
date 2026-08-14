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
      "stun_sources": [
        { "inflictor": "axe_berserkers_call", "category": "taunt", "seconds": 5.5 }
      ],
      "buffs_duration": 33.0,
      "buff_sources": [
        { "inflictor": "dazzle_shadow_wave", "category": "heal", "value": 1234.5, "count": 12 },
        { "inflictor": "dazzle_shallow_grave", "category": "save", "value": 33.0, "count": 7 }
      ],
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
  stacked are per-player: each player's own row, verified against OpenDota on
  matches where it reports `camps_stacked`).
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
- `stun_duration` is the scoreboard's `m_fStuns` when the team-data section is
  present, and it is authoritative: it matches the in-game scoreboard and
  OpenDota exactly (20/20 on the reference matches OpenDota exposes `stuns`
  for). Supplemental ADD→REMOVE disable uptime (fear/sleep/root/taunt/silence/
  break/mute/banish/leash/trap) is only added when the scoreboard section is
  missing entirely (legacy engine-0 replays). `stun_sources` is a single
  `scoreboard` entry on modern matches and a per-inflictor breakdown on the
  fallback path.
- Two combat-log id-spaces are bridged: pid (even `m_iPlayerID` of hero
  entities, also the `PLAYERSTATS` target) vs xpID (the `PLAYERSTATS` value,
  used in `attacker`/`target` of damage/heal/gold/xp events).
- Field names/layout inside `CDOTA_PlayerResource` drift between Dota patches;
  use `--dump-fields` to re-spike after big updates.
- `hero_damage` / `damage_taken`: summed from combat-log `DAMAGE` events
  (target hero, illusions excluded). Self-inflicted damage is excluded from
  `hero_damage` but **included** in `damage_taken`, mirroring the end-game
  summary and OpenDota. `damage_taken` matches OpenDota exactly for all 10
  players on the reference match; `hero_damage` agrees on 6/10 (reference),
  8/10 (engine-1 replay) and 65/70 within 10% across the modern reference
  matches. The residuals (e.g. Pudge +9128, Rubick −1123) come from damage
  sources the combat log records under a different owner (illusions,
  stolen/reflected damage) — a documented approximation.
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
- `healing` is the scoreboard's `m_fHealing` and matches OpenDota on 61/70
  player values across the modern reference matches (tolerance 50 or 2%) and
  matches Dotabuff exactly (10/10) on the reference match whose scoreboard data
  is sound. Combat-log HEAL sums were evaluated and rejected as a fallback: raw
  sums are dominated by lifesteal (Juggernaut 42.7k gross vs 11.5k actual) and
  per-ability sums overcount channeled heals (Enchantress attendants 15.0k vs
  4.0k, Winter Wyvern cold embrace 8.8k vs 5.0k) while missing heals attributed
  to summoned units (Juggernaut's Healing Ward) and items whose heal events do
  not fire (Spirit Vessel). On two reference replays the replay's own scoreboard
  data is corrupted (dire team of 8941955514: Juggernaut/Ember 0, Shadow Demon
  13.2k; 8943313464: Terrorblade 2.3k though he has no heal) — these are
  client-side scoreboard values the replay cannot fix, and no better source
  exists.
- `buff_sources` (per player, `omitempty`) credits positive effects a hero cast
  on same-team heroes, including self-casts (an ally-capable ability cast on
  its owner is a real buff). `category` is one of `heal`, `save`, `purge`,
  `shield`, `spell_immunity`, `invisibility`, `buff_stats`, `buff_haste`,
  `mana_restore`; `value` is uptime seconds for duration buffs and total HP for
  `heal`; `count` is applications / heal events. `buffs_duration` is the summed
  uptime across all duration buffs (always present, `0` when none). Heal credit
  comes from combat-log `HEAL` events (ability name is present on every event);
  lifesteal/regen and enemy-target heals are excluded, self-heals count.
  Duration buffs come from paired `MODIFIER_ADD`→`MODIFIER_REMOVE` events whose
  `elapsed_duration` is the uptime; passive auras, hidden modifiers, pure
  self-only buffs (e.g. Fiery Soul, Warpath) and illusion targets are excluded.
  Caveats: only action buffs in a curated catalog are tracked (the catalog
  lives in `internal/stats/buffs.go`); `MANA_RESTORED`/`HERO_SAVED`
  combat-log events never fire on observed patches, so `mana_restore` is only
  KotL chakra-magic uptime and Arcane Boots' mana grant is untrackable (its
  modifier is self-targeted and hidden); Naga Siren's Song is a disable and
  stays in `stun_sources`.
- The vendored `manta` package logs parser diagnostics straight to stdout when
  the `DEBUG` (level 1) or `TRACE` (level 10) environment variables are set;
  there is no `manta.DEBUG` variable. Run the CLI with an unset environment to
  keep stdout JSON-only (the CLI itself never writes to stdout except JSON).
- Replay files whose name starts with `-` (e.g. `-12345.dem`) are rejected by
  the CLI's flag parser; rename the file to parse it.

### Reference-match validation (modern patches)

Seven modern reference replays cross-checked against the OpenDota `/matches`
API and Dotabuff at authoring time:

| field | vs OpenDota |
|---|---|
| kills / deaths / assists / last_hits / level | 70/70 exact |
| stun_duration (`m_fStuns`) | 20/20 where OpenDota exposes `stuns` |
| healing (`m_fHealing`) | 61/70 (residuals are corrupted replay scoreboards, see above) |
| hero_damage | 65/70 within 10% |
| first blood | matches OpenDota where it exposes `firstblood_claimed`; the combat-log model attributes it in all matches |

## Tests

Layout:

- `internal/stats/*_test.go` — unit tests (aggregation, disable/buff classification, golden JSON)
- `tests/cli/` — CLI exit codes and stdin mode
- `tests/replay/` — replay integration and stun/buff regressions (`tests/replays/` fixtures)

```sh
go test ./...
```

When replay fixtures are present under `tests/replays/`, integration tests parse real `.dem` files and check stun regressions.

## License

MIT — see [LICENSE](LICENSE).
