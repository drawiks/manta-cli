package replay

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestParseStreamClassifiesBadMagic(t *testing.T) {
	for _, data := range [][]byte{
		nil,
		[]byte("not a replay"),
		[]byte("PUFDEMS\x00"),
	} {
		err := ParseStream(context.Background(), bytes.NewReader(data), nopHandler{})
		if !errors.Is(err, ErrNotReplay) {
			t.Errorf("ParseStream(%q) = %v, want ErrNotReplay", data, err)
		}
	}
}

func TestParseStreamClassifiesMissingPlayerResource(t *testing.T) {
	header := make([]byte, 16)
	copy(header, replayMagic)

	err := ParseStream(context.Background(), bytes.NewReader(header), nopHandler{})
	if !errors.Is(err, ErrNoPlayerResource) {
		t.Fatalf("ParseStream(header only) = %v, want ErrNoPlayerResource", err)
	}
}
