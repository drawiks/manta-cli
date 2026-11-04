# Accuracy

The output contract is the `odota-cli` match JSON. Validation uses five modern replays with a reference `.out.json` per match.

Every field is compared and must match exactly, except for the documented save/purge corrections below. The corpus covers identity, KDA, level, last hits, stuns, damage, healing, buffs, debuffs, positions, lanes, gold counters, first blood and courier kills.

The interval is sampled after packet entities are applied. Sampling on the net tick sees the previous packet and leaves networth and death transitions one tick behind the reference.

## Save and purge corrections

`v0.5.0` matched `odota-cli` exactly. The following corrections intentionally differ from it:

- Oracle False Promise is tracked through `modifier_oracle_false_promise_timer` and counted as a save.
- Wind Waker (`modifier_wind_waker`) is counted as a save.
- Eul's Cyclone, Pugna Decrepify, Underlord Dark Rift and Abaddon Borrowed Time are not save buffs.
- Disperser's movespeed buff is counted as a purge application, not a save.

Known differences in the five-replay corpus:

- `9048749977`: Hoodwink has `wind_waker` save duration 9 where the reference has 0.
- `9065268582`: Oracle has `oracle_false_promise` save duration 59 where the reference has 0.

`make parity` reports these differences. Do not add a tolerance; update or rebase the references, or add an explicit divergence allowlist, before claiming exact parity again.

## Harness

The parity harness is `tests/replay/parity_test.go`. It reads `MANTA_REPLAY_DIR` and skips when the corpus is absent. The corpus is not committed because the files are hundreds of megabytes.

If a future patch moves an interval field, add a focused comparison that prints the values around the boundary, change the sampling strategy in `internal/replay`, and run the parity harness before and after. Do not add a tolerance without a recorded reason.
