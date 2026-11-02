# ADR 0002: CLI compatibility

## Context

The CLI is single-command and already documented with flags. Subcommands would break scripts and add a command tree that the product does not need.

## Decision

Keep the flag-based CLI. Use `pflag` for GNU parsing, interspersed flags, `--` and long/short names. Add `--version`, `--output`, `--quiet` and `--compact`. Do not add subcommands until a real user-facing command exists.

## Consequences

The CLI stays composable and small. If subcommands become necessary, Cobra can replace pflag because the options already live in one struct.
