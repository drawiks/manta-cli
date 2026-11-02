package replay

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/drawiks/manta-cli/internal/events"
)

type nopHandler struct{}

func (nopHandler) Epilogue(events.Epilogue)         {}
func (nopHandler) Slot(events.Slot)                 {}
func (nopHandler) Interval(events.Interval)         {}
func (nopHandler) Combat(events.Combat)             {}
func (nopHandler) Chat(events.Chat)                 {}
func (nopHandler) AbilityLevel(events.AbilityLevel) {}

func FuzzDecompress(f *testing.F) {
	f.Add([]byte("plain replay stream"))
	f.Add([]byte{0x28, 0xb5, 0x2f, 0xfd})
	f.Add([]byte("BZh9"))

	f.Fuzz(func(t *testing.T, data []byte) {
		r, closer, err := decompressReader(bytes.NewReader(data))
		if err != nil {
			return
		}
		if closer != nil {
			defer func() { _ = closer.Close() }()
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(r, 1<<20))
	})
}

func FuzzParseStream(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("not a replay"))

	f.Fuzz(func(t *testing.T, data []byte) {
		_ = ParseStream(context.Background(), bytes.NewReader(data), nopHandler{})
	})
}
