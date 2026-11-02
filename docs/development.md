# Development

## Tools

```bash
make tools
make hooks
```

`make tools` installs pinned `gofumpt`, `golangci-lint` and `lefthook`. `make hooks` installs the git hooks; they are local convenience only, CI is authoritative.

## Checks

```bash
make fmt-check
make lint
make test
make test-race-short
make comments
make check
```

`make bench` runs the aggregation benchmarks. The pre-push hook uses `-short`, so it never parses the large corpus. `make parity` is the explicit full run and needs `MANTA_REPLAY_DIR` pointing at a directory with `<id>.dem` and `<id>.out.json` pairs.

## Fuzzing

```bash
go test -fuzz=FuzzDecompress -fuzztime=30s ./internal/replay
go test -fuzz=FuzzParseStream -fuzztime=30s ./internal/replay
```

The targets seed themselves; a found input is written under `testdata/fuzz` and becomes a regression test.

## Release

The release is tag-driven. Bump the version in the tag, push `vX.Y.Z`, and the release workflow runs tests, GoReleaser, SBOM and the Docker image. Local snapshot:

```bash
make build
goreleaser release --snapshot --clean
```

The binary reads its version from `internal/version`, filled by `-ldflags` in the Makefile and GoReleaser.
