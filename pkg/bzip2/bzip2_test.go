// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"testing"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

func TestNewWriterContract(t *testing.T) {
	var buf bytes.Buffer
	if _, err := bzip2.NewWriter(nil, 5); !errors.Is(err, bzip2.ErrNilWriter) {
		t.Fatalf("nil writer: got %v want ErrNilWriter", err)
	}
	for _, level := range []int{-2, 0, 10, 99} {
		if _, err := bzip2.NewWriter(&buf, level); !errors.Is(err, bzip2.ErrLevelRange) {
			t.Fatalf("level %d: got %v want ErrLevelRange", level, err)
		}
	}
	for level := 1; level <= 9; level++ {
		if _, err := bzip2.NewWriter(&buf, level); err != nil {
			t.Fatalf("level %d: %v", level, err)
		}
	}
}

func TestZeroValueWriterContract(t *testing.T) {
	var w bzip2.Writer
	if _, err := w.Write([]byte("x")); !errors.Is(err, bzip2.ErrWriterUninitialized) {
		t.Fatalf("Write: got %v want ErrWriterUninitialized", err)
	}
	if err := w.Close(); !errors.Is(err, bzip2.ErrWriterUninitialized) {
		t.Fatalf("Close: got %v want ErrWriterUninitialized", err)
	}
	if err := w.Reset(io.Discard); !errors.Is(err, bzip2.ErrWriterUninitialized) {
		t.Fatalf("Reset: got %v want ErrWriterUninitialized", err)
	}
}

func TestCloseContract(t *testing.T) {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); !errors.Is(err, bzip2.ErrClosed) {
		t.Fatalf("second Close: got %v want ErrClosed", err)
	}
	if _, err := w.Write([]byte("late")); !errors.Is(err, bzip2.ErrClosed) {
		t.Fatalf("Write after Close: got %v want ErrClosed", err)
	}
}

// badNWriter returns n larger than the slice, exercising writeFull's
// invalid-Write-result guard.
type badNWriter struct{}

func (badNWriter) Write(p []byte) (int, error) { return len(p) + 1, nil }

// negNWriter returns a negative count, the other invalid Write result.
type negNWriter struct{}

func (negNWriter) Write(p []byte) (int, error) { return -1, nil }

// noProgressWriter accepts nothing, driving writeFull's short-write stall.
type noProgressWriter struct{}

func (noProgressWriter) Write(p []byte) (int, error) { return 0, nil }

// partialWriter accepts at most max bytes per call so writeFull must loop.
type partialWriter struct {
	buf bytes.Buffer
	max int
}

func (p *partialWriter) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	n := min(len(b), p.max)
	return p.buf.Write(b[:n])
}

func TestWriteFullEdgeWriters(t *testing.T) {
	// Incompressible input (no two adjacent bytes equal) forces a block flush
	// inside Write at level 1, so destination errors surface there and not
	// only at Close.
	payload := make([]byte, 150000)
	for i := range payload {
		payload[i] = byte(i)
	}

	w, err := bzip2.NewWriter(badNWriter{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(payload); err == nil {
		t.Fatal("badNWriter: expected invalid Write result error")
	}
	// The failure latches: later Write and Close calls report the same error.
	if _, err := w.Write([]byte("x")); err == nil {
		t.Fatal("latched Write: expected error")
	}
	if err := w.Close(); err == nil {
		t.Fatal("latched Close: expected error")
	}

	w, err = bzip2.NewWriter(negNWriter{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(payload); err == nil {
		t.Fatal("negNWriter: expected invalid Write result error")
	}

	w, err = bzip2.NewWriter(noProgressWriter{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(payload); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("noProgressWriter: got %v want ErrShortWrite", err)
	}

	// A small write reaches the destination only at Close; the short write
	// must surface there.
	w, err = bzip2.NewWriter(noProgressWriter{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("small")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("close flush: got %v want ErrShortWrite", err)
	}

	pw := &partialWriter{max: 3}
	w, err = bzip2.NewWriter(pw, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(bzip2.NewReader(&pw.buf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("partial-writer roundtrip mismatch")
	}
}

func TestResetContract(t *testing.T) {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(io.Discard, 7)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Reset(nil); !errors.Is(err, bzip2.ErrNilWriter) {
		t.Fatalf("Reset(nil): got %v want ErrNilWriter", err)
	}
	if err := w.Reset(&buf); err != nil {
		t.Fatal(err)
	}
	want := []byte("reused writer stream")
	if _, err := w.Write(want); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

// TestNewReaderDecodesStdlibStream checks the public reader against a known
// libbz2-produced stream for "hello world\n".
func TestNewReaderDecodesStdlibStream(t *testing.T) {
	const hexStream = "425a68393141592653594eece83600000251800010400006449080200031064c4101a7a9a580bb9431f8bb9229c28482776741b0"
	raw, err := hex.DecodeString(hexStream)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(bzip2.NewReader(bytes.NewReader(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("hello world\n")) {
		t.Fatalf("got %q", got)
	}
}
