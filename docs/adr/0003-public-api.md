# ADR 0003: public Go API

## Context

The project started as a CLI with every implementation package under `internal/`. Users who wanted the parsed match in Go had no supported import path. Exposing `internal/app` or the manta adapter directly would make internal refactors and decoder changes part of a public contract.

## Decision

The repository is both a library and a command:

- the module root is `package manta` and exposes `ParseFile`, `ParseReader`, the public error sentinels and type aliases;
- canonical data types live in the public leaf package `stats`;
- the command lives in `cmd/manta-cli`;
- all decoder, aggregation and CLI implementation remains under `internal/`.

The public API is deliberately small. It returns concrete `*stats.Match` values, accepts `io.Reader`, takes `context.Context` first and exposes no raw events, options or decoder types. Internal replay errors are translated to public sentinels at the facade boundary.

`v0.5.0` is the first public API release and is a preview. Formal Go module compatibility starts at `v1.0.0`.

## Consequences

The binary install path changes to `github.com/drawiks/manta-cli/cmd/manta-cli`. The old module-root path is now a library and no longer installs a command.

The public `stats` fields and JSON tags become compatibility commitments at `v1.0.0`. Adding fields is a minor release; removing or renaming them requires a major release. Internal packages remain free to change.
