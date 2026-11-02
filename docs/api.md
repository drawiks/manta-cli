# Public API

The module exports two public packages:

- `github.com/drawiks/manta-cli` — the parser facade.
- `github.com/drawiks/manta-cli/stats` — the canonical match, player and source types.

The command is `github.com/drawiks/manta-cli/cmd/manta-cli`.

## Parse functions

```go
func ParseFile(ctx context.Context, path string) (*Match, error)
func ParseReader(ctx context.Context, r io.Reader) (*Match, error)
```

`ParseFile` opens the path, detects `.dem`, `.dem.bz2` or `.dem.zst` from magic bytes, and closes the file. If the replay has no `CDemoFileInfo` match ID, the leading decimal digits of the file's base name are used.

`ParseReader` reads the replay from `r`, detects compression from magic bytes, and does not close `r`. The match ID comes from `CDemoFileInfo`; it is zero when the replay has none.

Both functions:

- accept a non-nil `context.Context` as their first argument;
- return `ctx.Err()` when the context is already canceled;
- check cancellation while parsing;
- return a `*Match` that is independent of other calls;
- are safe to call concurrently;
- do not retain the replay after returning.

Cancellation cannot interrupt an `io.Reader` whose `Read` is already blocked. Use a reader with its own timeout or a file when this matters.

## Types

`manta.Match`, `manta.Player` and `manta.SourceEntry` are aliases for `stats.Match`, `stats.Player` and `stats.SourceEntry`.

The `Match` JSON encoding is the odota-cli match object. The schema is [`schema/match.schema.json`](../schema/match.schema.json). `Players` is ordered by `player_id`. Source arrays are omitted when empty and are sorted deterministically. `position` and `lane` are omitted when there are not enough position samples.

Field documentation lives in the `stats` package.

## Errors

```go
var (
    ErrNotReplay        = errors.New("not a Dota 2 replay")
    ErrNoPlayerResource = errors.New("CDOTA_PlayerResource not found")
)
```

| Error | Meaning |
| --- | --- |
| `ErrNotReplay` | The input does not start with the Source 2 `PBDEMS2` magic. Source 1 is not supported. |
| `ErrNoPlayerResource` | The replay was decoded but contains no `CDOTA_PlayerResource`. |

`ParseFile` wraps both errors with the path. `ParseReader` returns them directly. Other parser and decompressor errors are diagnostics, not stable categories. I/O errors from the caller's reader and context cancellation are preserved for `errors.Is`.

```go
m, err := manta.ParseFile(ctx, "match.dem")
if err != nil {
    if errors.Is(err, manta.ErrNotReplay) {
        // unsupported or non-replay input
    }
    return err
}
```

Do not compare error text.

## Stability

`v0.5.0` is the first public API release. The API shape is frozen for feedback during the `v0.5.x` line; breaking changes are possible before `v1.0.0`, which is the formal Go module compatibility commitment.

From the first stable release onward:

- adding functions, fields, packages and error categories is a minor release;
- removing or renaming exported names is a major release;
- JSON field names, types and omission rules follow the schema version;
- internal packages remain private and may change at any time.

## Non-goals

The public API does not expose raw replay events, the manta decoder, HTTP fetching, NDJSON streaming, catalog generation, options or byte-slice parsing. Use `bytes.NewReader` for an in-memory replay.
