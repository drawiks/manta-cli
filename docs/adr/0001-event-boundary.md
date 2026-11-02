# ADR 0001: the event boundary

## Context

The manta adapter and the statistics fold started in one package. That made the fold depend on manta, made tests carry replay fixtures and prevented replacing the decoder.

## Decision

The replay adapter emits typed values from `internal/events`. The aggregator consumes them and produces `stats.Match`. Manta types never cross into the aggregator.

## Consequences

The fold is tested with synthetic events. The decoder is tested with decompression, fuzz and the external corpus. A new event needs a field on the event, an adapter mapping and a fold branch.
