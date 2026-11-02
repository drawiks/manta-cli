# Accuracy

The output contract is the `odota-cli` match JSON. Validation uses five modern replays with a reference `.out.json` per match.

Every field is compared and must match exactly. The corpus covers identity, KDA, level, last hits, stuns, damage, healing, buffs, debuffs, positions, lanes, gold counters, first blood and courier kills.

The interval is sampled after packet entities are applied. Sampling on the net tick sees the previous packet and leaves networth and death transitions one tick behind the reference.

The parity harness is `tests/replay/parity_test.go`. It reads `MANTA_REPLAY_DIR` and skips when the corpus is absent. The corpus is not committed because the files are hundreds of megabytes.

If a future patch moves an interval field, add a focused comparison that prints the values around the boundary, change the sampling strategy in `internal/replay`, and run the parity harness before and after. Do not add a tolerance without a recorded reason.
