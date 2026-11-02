package manta

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/drawiks/manta-cli/internal/app"
	"github.com/drawiks/manta-cli/internal/replay"
	"github.com/drawiks/manta-cli/stats"
)

// Match is the parsed statistics for one Dota 2 match.
//
// It is an alias for [stats.Match], which documents the fields.
type Match = stats.Match

// Player is the statistics for one player in a match.
//
// It is an alias for [stats.Player], which documents the fields.
type Player = stats.Player

// SourceEntry is one inflictor's contribution to a tracked effect.
//
// It is an alias for [stats.SourceEntry], which documents the fields.
type SourceEntry = stats.SourceEntry

// ParseFile parses the Dota 2 replay at path and returns its match statistics.
//
// It detects .dem, .dem.bz2 and .dem.zst input from magic bytes. If the replay
// has no CDemoFileInfo match ID, the leading decimal digits of the base name
// are used.
//
// If ctx is canceled, ParseFile returns ctx.Err(). Errors that match
// [ErrNotReplay] or [ErrNoPlayerResource] are returned wrapped with the path.
func ParseFile(ctx context.Context, path string) (*Match, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m, err := app.ParseFile(ctx, path)
	if err != nil {
		return nil, wrapFileError(path, err)
	}
	return m, nil
}

// ParseReader parses a Dota 2 replay from r and returns its match statistics.
//
// It detects .dem, .dem.bz2 and .dem.zst input from magic bytes and does not
// close r. The match ID comes from CDemoFileInfo.
//
// If ctx is canceled, ParseReader returns ctx.Err(). Cancellation is checked
// while parsing; it cannot interrupt an [io.Reader] that is already blocked in
// Read.
func ParseReader(ctx context.Context, r io.Reader) (*Match, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m, err := app.ParseStream(ctx, r)
	if err != nil {
		return nil, wrapReaderError(err)
	}
	return m, nil
}

func wrapFileError(path string, err error) error {
	switch {
	case errors.Is(err, replay.ErrNotReplay):
		return fmt.Errorf("%w: %s", ErrNotReplay, path)
	case errors.Is(err, replay.ErrNoPlayerResource):
		return fmt.Errorf("%w: %s", ErrNoPlayerResource, path)
	default:
		return fmt.Errorf("parse %s: %w", path, err)
	}
}

func wrapReaderError(err error) error {
	switch {
	case errors.Is(err, replay.ErrNotReplay):
		return ErrNotReplay
	case errors.Is(err, replay.ErrNoPlayerResource):
		return ErrNoPlayerResource
	default:
		return fmt.Errorf("parse replay: %w", err)
	}
}
