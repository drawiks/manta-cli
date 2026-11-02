# Architecture

`main.go` parses the process environment, builds a context, wires the CLI and exits. All logic lives under `internal/`.

```
main.go
  internal/cli          flags, I/O, output, exit codes
  internal/app          builds the aggregator and the replay reader
  internal/replay       manta adapter, decompression, intervals, entity dump
  internal/events       typed events shared by replay and aggregate
  internal/aggregate    event fold producing the match result
  internal/catalog      hero ids, modifier classes, item costs
  internal/roles        lane and position assignment
  internal/model        output schema
  internal/version      build metadata
```

The dependency direction is:

```
cli -> app -> replay -> events
          \-> aggregate -> events, catalog, roles, model
```

`events` and `model` are leaves. `catalog` and `roles` are pure. `replay` imports manta and never the aggregator. `aggregate` never imports manta. `cli` never contains parsing or statistics logic.

The replay adapter emits `Epilogue`, `Slot`, `Interval`, `Combat`, `Chat` and `AbilityLevel` values. The aggregator implements `events.Handler`; its state is per run, so two parses share nothing.

`context.Context` travels from `main.go` to the replay callbacks. Cancellation calls `Parser.Stop` and returns the context error. Output is written atomically when `--output` names a path.

The decoder reads a stream and never holds the replay in memory. The aggregator keeps O(players + live modifiers), not O(events).
