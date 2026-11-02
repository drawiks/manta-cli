# Architecture

`cmd/manta-cli/main.go` parses the process environment, builds a context, wires the CLI and exits. The public Go API lives in the module root and in `stats`; all implementation packages are under `internal/`.

```
cmd/manta-cli              command wiring and exit code
manta.go                   public ParseFile, ParseReader and aliases
stats/                     public Match, Player and SourceEntry types
internal/cli               flags, I/O, output and exit codes
internal/app               builds the aggregator and the replay reader
internal/replay            manta adapter, decompression, intervals, entity dump
internal/events            typed events shared by replay and aggregate
internal/aggregate         event fold producing the match result
internal/catalog           hero ids, modifier classes, item costs
internal/roles             lane and position assignment
internal/version           build metadata
```

The dependency direction is:

```
cmd/manta-cli -> internal/cli -> manta, internal/replay
manta -> internal/app, internal/replay, stats
internal/app -> internal/replay, internal/aggregate, stats
internal/aggregate -> internal/events, internal/catalog, internal/roles, stats
internal/roles -> stats
internal/replay -> internal/events, internal/catalog
```

`events` and `stats` are leaves. `catalog` and `roles` are pure. `replay` imports manta and never the aggregator. `aggregate` never imports manta. `cli` never contains parsing or statistics logic.

The public root package is a facade: it owns `ParseFile`, `ParseReader` and the public error sentinels, translates internal replay errors at the boundary, and aliases the canonical types in `stats`. Public types never expose manta types.

The replay adapter emits `Epilogue`, `Slot`, `Interval`, `Combat`, `Chat` and `AbilityLevel` values. The aggregator implements `events.Handler`; its state is per run, so two parses share nothing.

`context.Context` travels from the caller through the replay callbacks. Cancellation calls `Parser.Stop` and returns the context error. Output is written atomically when `--output` names a path.

The decoder reads a stream and never holds the replay in memory. The aggregator keeps O(players + live modifiers), not O(events).
