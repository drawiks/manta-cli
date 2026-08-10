package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

// TestRunStdinNoData checks that --stdin with an empty stream fails cleanly
// with an error and nothing on stdout.
func TestRunStdinNoData(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"--stdin"}, bytes.NewReader(nil), &out, &errBuf); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(errBuf.String(), "error:") {
		t.Errorf("stderr should start with error:, got: %q", errBuf.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty on failure, got: %q", out.String())
	}
}

// TestRunStdinGarbage checks that a stream that is not a replay errors out
// without hanging.
func TestRunStdinGarbage(t *testing.T) {
	var out, errBuf bytes.Buffer
	garbage := bytes.NewReader(bytes.Repeat([]byte("garbage"), 4096))
	if code := run([]string{"--stdin"}, garbage, &out, &errBuf); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(errBuf.String(), "error:") {
		t.Errorf("stderr should start with error:, got: %q", errBuf.String())
	}
}

// TestRunStdinSingleDashFlag checks that the short -stdin form works too.
func TestRunStdinSingleDashFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"-stdin"}, bytes.NewReader(nil), &out, &errBuf); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

// TestRunStdinWithPath checks that --stdin combined with a file path is a
// usage error.
func TestRunStdinWithPath(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"--stdin", "x.dem"}, bytes.NewReader(nil), &out, &errBuf); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errBuf.String(), "error:") {
		t.Errorf("stderr should contain an error prefix, got: %q", errBuf.String())
	}
}

// TestDecompressReaderZstd round-trips a zstd-compressed stream through
// decompressReader to make sure the stdin pipeline decodes it.
func TestDecompressReaderZstd(t *testing.T) {
	plain := []byte("hello stdin replay stream")

	var compressed bytes.Buffer
	enc, err := zstd.NewWriter(&compressed)
	if err != nil {
		t.Fatalf("zstd.NewWriter: %v", err)
	}
	if _, err := enc.Write(plain); err != nil {
		t.Fatalf("zstd write: %v", err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("zstd close: %v", err)
	}

	r, closer, err := decompressReader(bytes.NewReader(compressed.Bytes()))
	if err != nil {
		t.Fatalf("decompressReader: %v", err)
	}
	if closer == nil {
		t.Fatal("closer should not be nil for a zstd stream")
	}
	defer closer.Close()

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("decoded = %q, want %q", got, plain)
	}
}

// TestDecompressReaderPlain checks that an uncompressed stream passes through
// unchanged and yields a nil closer.
func TestDecompressReaderPlain(t *testing.T) {
	plain := []byte("plain replay stream")
	r, closer, err := decompressReader(bytes.NewReader(plain))
	if err != nil {
		t.Fatalf("decompressReader: %v", err)
	}
	if closer != nil {
		t.Fatal("closer should be nil for a plain stream")
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("decoded = %q, want %q", got, plain)
	}
}

// TestRunStdinParsesReplay feeds a real replay through stdin and checks the
// JSON result. This also verifies the CDemoFileInfo fallback, since stdin mode
// cannot derive the match id from a filename.
func TestRunStdinParsesReplay(t *testing.T) {
	f, err := os.Open("8926354517.dem")
	if err != nil {
		t.Skip("replay fixture 8926354517.dem not present")
	}
	defer f.Close()

	var out, errBuf bytes.Buffer
	if code := run([]string{"--stdin"}, f, &out, &errBuf); code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, errBuf.String())
	}
	s := out.String()
	if !strings.HasPrefix(s, "{") {
		t.Errorf("stdout should be JSON, got: %q", s)
	}
	if !strings.Contains(s, `"match_id":`) {
		t.Errorf("stdout should contain match_id, got: %q", s)
	}
	if !strings.Contains(s, `"players"`) {
		t.Errorf("stdout should contain players, got: %q", s)
	}
}

// TestRunStdinDumpFields feeds a real replay through stdin with --dump-fields.
func TestRunStdinDumpFields(t *testing.T) {
	f, err := os.Open("8926354517.dem")
	if err != nil {
		t.Skip("replay fixture 8926354517.dem not present")
	}
	defer f.Close()

	var out, errBuf bytes.Buffer
	if code := run([]string{"--dump-fields", "--stdin"}, f, &out, &errBuf); code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), dataTeamDireClass) {
		t.Errorf("stderr should contain %s, got: %q", dataTeamDireClass, errBuf.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty for --dump-fields, got: %q", out.String())
	}
}
