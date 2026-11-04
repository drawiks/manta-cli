# Accuracy

The output contract is the `odota-cli` match JSON plus the `mana_restored` extension. Validation uses five modern replays with a reference `.out.json` per match.

The corpus covers identity, KDA, level, last hits, stuns, damage, healing, mana restoration, buffs, debuffs, positions, lanes, gold counters, first blood and courier kills. Fields are compared exactly except for the documented catalog corrections and the `mana_restored` extension.

The interval is sampled after packet entities are applied. Sampling on the net tick sees the previous packet and leaves networth and death transitions one tick behind the reference.

## Catalog corrections

`v0.5.0` matched `odota-cli` exactly. The following corrections intentionally differ from it:

- Oracle False Promise is tracked through `modifier_oracle_false_promise_timer` and counted as a save.
- Wind Waker (`modifier_wind_waker`) is counted as a save.
- Eul's Cyclone, Pugna Decrepify, Underlord Dark Rift and Abaddon Borrowed Time are not save buffs.
- Disperser's movespeed buff is counted as a purge application, not a save.
- Undying's Tombstone Bunker (`modifier_undying_tombstone_bunker_friendly`) is counted as a save.
- Shadow Demon Disruption and Linken's active spellblock are saves.
- Pipe, Pavise, Crimson Guard and Solar Crest barriers are shields; Solar Crest no longer counts as `buff_stats`.
- Marci Rebound and Grimstroke Ink Swell count as haste; Drum and Boots of Bearing use their current active modifiers.
- Sven Warcry, Troll Battle Trance with the level 20 talent, and Boots of Bearing are `buff_stats`.
- Stale modifier keys for Omniknight Repel, Pipe, Pavise, Marci Bodyguard, Chen Divine Favor and Drum were corrected.
- Bounty Hunter Wind Walk was removed from `invisibility`; it is self-only.
- `mana_restore` was removed from buff categories. `mana_restored` counts mana restored to allied heroes only.
- `mana_restored` uses `DOTA_COMBATLOG_MANA_RESTORED` (type 24). Current HLTV replays in the corpus do not emit type 24, so the field is zero there; a replay that carries the event will populate it.

Known differences in the five-replay corpus:

- `9048749977`: Hoodwink has `wind_waker` save duration 9 where the reference has 0.
- `9065268582`: Oracle has `oracle_false_promise` save duration 59 where the reference has 0.

Undying is not present in the five-replay corpus, so the Tombstone Bunker correction adds no harness difference there. The catalog corrections and `mana_restored` add further expected differences on matches with the affected items and abilities.

`make parity` reports these differences. Do not add a tolerance; update or rebase the references, or add an explicit divergence allowlist, before claiming exact parity again.

## Harness

The parity harness is `tests/replay/parity_test.go`. It reads `MANTA_REPLAY_DIR` and skips when the corpus is absent. The corpus is not committed because the files are hundreds of megabytes.

If a future patch moves an interval field, add a focused comparison that prints the values around the boundary, change the sampling strategy in `internal/replay`, and run the parity harness before and after. Do not add a tolerance without a recorded reason.
