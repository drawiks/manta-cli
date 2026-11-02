package manta_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	manta "github.com/drawiks/manta-cli"
	"github.com/drawiks/manta-cli/stats"
)

func TestParseReaderRejectsNonReplay(t *testing.T) {
	for _, data := range []string{
		"",
		"not a replay",
		"PUFDEMS\x00",
	} {
		_, err := manta.ParseReader(context.Background(), strings.NewReader(data))
		if !errors.Is(err, manta.ErrNotReplay) {
			t.Errorf("ParseReader(%q) error = %v, want ErrNotReplay", data, err)
		}
	}
}

func TestParseReaderClassifiesMissingPlayerResource(t *testing.T) {
	header := make([]byte, 16)
	copy(header, "PBDEMS2\x00")

	_, err := manta.ParseReader(context.Background(), strings.NewReader(string(header)))
	if !errors.Is(err, manta.ErrNoPlayerResource) {
		t.Fatalf("ParseReader(header only) error = %v, want ErrNoPlayerResource", err)
	}
}

func TestParseReaderCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := manta.ParseReader(ctx, strings.NewReader("not a replay"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ParseReader(canceled) error = %v, want context.Canceled", err)
	}
}

func TestParseFileMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.dem")

	_, err := manta.ParseFile(context.Background(), path)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ParseFile(%q) error = %v, want fs.ErrNotExist", path, err)
	}
}

func TestParseFileRejectsNonReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.dem")
	if err := os.WriteFile(path, []byte("not a replay"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := manta.ParseFile(context.Background(), path)
	if !errors.Is(err, manta.ErrNotReplay) {
		t.Fatalf("ParseFile(%q) error = %v, want ErrNotReplay", path, err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("ParseFile(%q) error = %q, want path in message", path, err)
	}
}

func TestTypesAreAliases(t *testing.T) {
	var match stats.Match = manta.Match{}
	var player stats.Player = manta.Player{}
	var source stats.SourceEntry = manta.SourceEntry{}
	_, _, _ = match, player, source
}

func ExampleParseReader() {
	_, err := manta.ParseReader(context.Background(), strings.NewReader("not a replay"))
	if errors.Is(err, manta.ErrNotReplay) {
		fmt.Println("not a replay")
	}

	// Output: not a replay
}
