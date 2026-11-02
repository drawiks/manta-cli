# Output schema

`schema/match.schema.json` is the machine form of the match object. The Go type is `stats.Match`. The field names, types and omission rules match the `odota-cli` output so the two tools are interchangeable.

The top-level object has `match_id`, `duration_sec`, `radiant_win` and `players`. A player has identity, position and lane, combat totals, the tracked effect totals and sources, gold counters, objectives and first blood. `position` and `lane` are omitted when there are no position samples. Source arrays are omitted when empty.

Do not add fields in a patch release. A field addition is a minor release and needs a matching schema update and a parity run.
