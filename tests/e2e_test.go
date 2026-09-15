// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

// TestEndToEndPipeline exercises the whole public API as a streaming pipeline:
// io.Copy into the Writer, then io.Copy out of the Reader.
func TestEndToEndPipeline(t *testing.T) {
	var src bytes.Buffer
	for i := 0; i < 20000; i++ {
		src.WriteString("line of pipeline data\n")
	}
	want := src.Bytes()

	var compressed bytes.Buffer
	w, err := bzip2.NewWriter(&compressed, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(w, &src); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if compressed.Len() >= len(want) {
		t.Fatalf("compressible input did not shrink: in=%d out=%d", len(want), compressed.Len())
	}

	var out bytes.Buffer
	if _, err := io.Copy(&out, bzip2.NewReader(&compressed)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), want) {
		t.Fatalf("pipeline mismatch: got %d want %d bytes", out.Len(), len(want))
	}
}

// oneByteReader drip-feeds the decoder a single byte per Read to stress partial
// reads of the compressed stream.
type oneByteReader struct {
	r io.Reader
}

func (o *oneByteReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return o.r.Read(p)
}

func TestEndToEndSlowReader(t *testing.T) {
	want := bytes.Repeat([]byte("drip feed "), 5000)
	compressed := compressBZ(t, 9, want)
	got, err := io.ReadAll(bzip2.NewReader(&oneByteReader{r: bytes.NewReader(compressed)}))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("slow-reader decode mismatch")
	}
}

// TestEndToEndDoubleRoundtrip decompresses a stream and recompresses it through
// the public API, verifying the second generation round-trips identically.
func TestEndToEndDoubleRoundtrip(t *testing.T) {
	want := bytes.Repeat([]byte("recompress me "), 8000)
	first := compressBZ(t, 9, want)

	mid, err := io.ReadAll(bzip2.NewReader(bytes.NewReader(first)))
	if err != nil {
		t.Fatal(err)
	}
	second := compressBZ(t, 3, mid)
	got, err := io.ReadAll(bzip2.NewReader(bytes.NewReader(second)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("double roundtrip mismatch")
	}
}

// TestEndToEndTwoStreamsOneBuffer writes two independent streams to one buffer
// through separate Writers, the file-layout equivalent of concatenation.
func TestEndToEndTwoStreamsOneBuffer(t *testing.T) {
	var buf bytes.Buffer
	for i, part := range [][]byte{[]byte("alpha"), []byte("omega")} {
		w, err := bzip2.NewWriter(&buf, 1+i)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(part); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	}
	got, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("alphaomega")) {
		t.Fatalf("got %q", got)
	}
}
