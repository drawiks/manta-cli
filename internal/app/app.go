package app

import (
	"context"
	"io"

	"github.com/drawiks/manta-cli/internal/aggregate"
	"github.com/drawiks/manta-cli/internal/model"
	"github.com/drawiks/manta-cli/internal/replay"
)

func ParseFile(ctx context.Context, path string) (*model.Result, error) {
	a := aggregate.New()
	if err := replay.ParseFile(ctx, path, a); err != nil {
		return nil, err
	}
	return a.Result(), nil
}

func ParseStream(ctx context.Context, r io.Reader) (*model.Result, error) {
	a := aggregate.New()
	if err := replay.ParseStream(ctx, r, a); err != nil {
		return nil, err
	}
	return a.Result(), nil
}
