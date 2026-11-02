package app

import (
	"context"
	"io"

	"github.com/drawiks/manta-cli/internal/aggregate"
	"github.com/drawiks/manta-cli/internal/replay"
	"github.com/drawiks/manta-cli/stats"
)

func ParseFile(ctx context.Context, path string) (*stats.Match, error) {
	a := aggregate.New()
	if err := replay.ParseFile(ctx, path, a); err != nil {
		return nil, err
	}
	return a.Result(), nil
}

func ParseStream(ctx context.Context, r io.Reader) (*stats.Match, error) {
	a := aggregate.New()
	if err := replay.ParseStream(ctx, r, a); err != nil {
		return nil, err
	}
	return a.Result(), nil
}
