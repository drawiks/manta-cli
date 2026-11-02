# Accuracy

The output contract is the `odota-cli` match JSON. Validation uses five modern replays with a reference `.out.json` per match.

Every field is compared. The current residuals are:

| Field | Residual | Cause |
|---|---:|---|
| `time_dead` | ±2 s | death and respawn transitions are sampled once per second |
| `networth` | ±1 | the last interval can be one tick before the reference sample |
| `gpm` | ±1 | the last interval gold can be one tick before the reference sample |

Everything else matches exactly on the corpus: identity, KDA, level, last hits, stuns, damage, healing, buffs, debuffs, positions, lanes, gold counters, first blood and courier kills.

The parity harness is `tests/replay/parity_test.go`. It reads `MANTA_REPLAY_DIR` and skips when the corpus is absent. The corpus is not committed because the files are hundreds of megabytes.

To tighten a residual, add a focused comparison that prints the values around the boundary, change the sampling strategy in `internal/replay`, and run the parity harness before and after. Do not add a tolerance without a recorded reason.
