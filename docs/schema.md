# Output schema

`schema/match.schema.json` is the machine form of the match object. The Go type is `stats.Match`. The field names, types and omission rules match the `odota-cli` output so the two tools are interchangeable.

The top-level object has `match_id`, `duration_sec`, `radiant_win` and `players`. A player has identity, position and lane, combat totals, `mana_restored`, the tracked effect totals and sources, gold counters, objectives and first blood. `mana_restored` is a manta-cli extension; odota-cli has no equivalent field. It counts mana restored to allies, including the actual Bottle healing-over-time cast on an ally; self-use is excluded. `gold_lost` and `tower_damage` use the game's client counters when available. `position` and `lane` are omitted when there are no position samples. Source arrays are omitted when empty. `healing` is the single healing total; `heal_duration`, `heal_value` and `heal_sources` were removed.

Do not add fields in a patch release. A field addition is a minor release and needs a matching schema update and a parity run.
