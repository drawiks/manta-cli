package replay

import (
	"bytes"
	"io"
	"testing"

	"github.com/klauspost/compress/zstd"
)

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
