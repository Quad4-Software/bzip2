// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

func TestRoundtripEmpty(t *testing.T) {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 9)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("want empty, got %d bytes", len(out))
	}
}

func TestStdlibHelloVector(t *testing.T) {
	want := []byte("hello world\n")
	hexStr := "425a68393141592653594eece83600000251800010400006449080200031064c4101a7a9a580bb9431f8bb9229c28482776741b0"
	wantHex, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(bzip2.NewReader(bytes.NewReader(wantHex)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, want) {
		t.Fatal("stdlib vector mismatch")
	}
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(want); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), wantHex) {
		t.Logf("compressed len ours=%d ref=%d", buf.Len(), len(wantHex))
	}
	got, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("roundtrip mismatch")
	}
}

func TestRoundtripRandomSizes(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for n := 0; n < 5000; n += 137 {
		data := make([]byte, n)
		for i := range data {
			data[i] = byte(rng.Intn(256))
		}
		var buf bytes.Buffer
		w, err := bzip2.NewWriter(&buf, 5)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(bzip2.NewReader(&buf))
		if err != nil {
			t.Fatalf("n=%d: %v", n, err)
		}
		if !bytes.Equal(got, data) {
			t.Fatalf("n=%d mismatch", n)
		}
	}
}

type errWriter struct{}

func (errWriter) Write(p []byte) (int, error) {
	return 0, errors.New("mock write error")
}

func TestWriterDstError(t *testing.T) {
	w, err := bzip2.NewWriter(errWriter{}, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "x"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err == nil {
		t.Fatal("expected error")
	}
	if _, err := w.Write([]byte("y")); err == nil {
		t.Fatal("expected error after failed close")
	}
}

type chunkWriter struct {
	dst io.Writer
	max int
}

func (c *chunkWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := min(len(p), c.max)
	return c.dst.Write(p[:n])
}

func TestRoundtripChunkedDestination(t *testing.T) {
	data := bytes.Repeat([]byte("abcdefghij"), 5000)
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&chunkWriter{dst: &buf, max: 7}, 6)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("len got=%d want=%d", len(got), len(data))
	}
}

type zeroNilWriter struct{}

func (zeroNilWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return 0, nil
}

func TestWriterShortWriteStall(t *testing.T) {
	w, err := bzip2.NewWriter(zeroNilWriter{}, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != io.ErrShortWrite {
		t.Fatalf("got %v want ErrShortWrite", err)
	}
	if _, err := w.Write([]byte("x")); err != io.ErrShortWrite {
		t.Fatalf("got %v want ErrShortWrite", err)
	}
	if err := w.Close(); err != io.ErrShortWrite {
		t.Fatalf("got %v want ErrShortWrite", err)
	}
}

type errWriterWithErr struct{ err error }

func (e errWriterWithErr) Write(p []byte) (int, error) {
	return 0, e.err
}

func TestCloseAfterWriteFailure(t *testing.T) {
	mockErr := errors.New("mock")
	w, err := bzip2.NewWriter(errWriterWithErr{err: mockErr}, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "z"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); !errors.Is(err, mockErr) {
		t.Fatalf("got %v want %v", err, mockErr)
	}
	if err := w.Close(); !errors.Is(err, mockErr) {
		t.Fatalf("got %v want %v", err, mockErr)
	}
}

func TestWriterClosed(t *testing.T) {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 9)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte{1}); err != bzip2.ErrClosed {
		t.Fatalf("got %v", err)
	}
}

func TestNewWriterLevel(t *testing.T) {
	var buf bytes.Buffer
	if _, err := bzip2.NewWriter(&buf, 0); err != bzip2.ErrLevelRange {
		t.Fatal("expected ErrLevelRange")
	}
	if _, err := bzip2.NewWriter(&buf, 10); err != bzip2.ErrLevelRange {
		t.Fatal("expected ErrLevelRange")
	}
}

func TestNewWriterNil(t *testing.T) {
	if _, err := bzip2.NewWriter(nil, 5); err != bzip2.ErrNilWriter {
		t.Fatalf("got %v", err)
	}
}

func TestCloseTwice(t *testing.T) {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 9)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != bzip2.ErrClosed {
		t.Fatalf("got %v", err)
	}
}
