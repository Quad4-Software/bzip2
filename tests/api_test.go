// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

// TestWriteReturnsFullCount verifies Write reports len(p) consumed on success.
func TestWriteReturnsFullCount(t *testing.T) {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 4)
	if err != nil {
		t.Fatal(err)
	}
	p := []byte("count me precisely")
	n, err := w.Write(p)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(p) {
		t.Fatalf("Write returned n=%d want %d", n, len(p))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestWriteEmptyOnOpenWriter: writing an empty slice must succeed with n=0 and
// must not disturb the stream.
func TestWriteEmptyOnOpenWriter(t *testing.T) {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 2)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := w.Write(nil); n != 0 || err != nil {
		t.Fatalf("Write(nil) = %d, %v", n, err)
	}
	if n, err := w.Write([]byte{}); n != 0 || err != nil {
		t.Fatalf("Write(empty) = %d, %v", n, err)
	}
	if _, err := w.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("x")) {
		t.Fatalf("got %q", got)
	}
}

// TestNewWriterValidLevels checks every level 1-9 constructs and round-trips.
func TestNewWriterValidLevels(t *testing.T) {
	for level := 1; level <= 9; level++ {
		var buf bytes.Buffer
		w, err := bzip2.NewWriter(&buf, level)
		if err != nil {
			t.Fatalf("level %d: %v", level, err)
		}
		payload := []byte("level check")
		if _, err := w.Write(payload); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(bzip2.NewReader(&buf))
		if err != nil {
			t.Fatalf("level %d decode: %v", level, err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("level %d mismatch", level)
		}
	}
}

// TestNewWriterInvalidLevels covers out-of-range levels on both sides.
func TestNewWriterInvalidLevels(t *testing.T) {
	var buf bytes.Buffer
	for _, level := range []int{-100, -1, 0, 10, 11, 1000} {
		if _, err := bzip2.NewWriter(&buf, level); !errors.Is(err, bzip2.ErrLevelRange) {
			t.Fatalf("level %d: got %v want ErrLevelRange", level, err)
		}
	}
}

// TestResetNilDestination: Reset must reject a nil writer.
func TestResetNilDestination(t *testing.T) {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Reset(nil); !errors.Is(err, bzip2.ErrNilWriter) {
		t.Fatalf("got %v want ErrNilWriter", err)
	}
}

// TestResetAfterErrorRecovers documents that Reset clears a latched stream error
// and makes the Writer usable again.
func TestResetAfterErrorRecovers(t *testing.T) {
	w, err := bzip2.NewWriter(errWriter{}, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "boom"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err == nil {
		t.Fatal("expected write error")
	}
	var buf bytes.Buffer
	if err := w.Reset(&buf); err != nil {
		t.Fatalf("Reset after error: %v", err)
	}
	want := []byte("recovered stream")
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

// TestResetDiscardsUnclosedStream: Reset before Close must drop the partial
// stream; only post-Reset data may decode.
func TestResetDiscardsUnclosedStream(t *testing.T) {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 5)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "discarded"); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := w.Reset(&buf); err != nil {
		t.Fatal(err)
	}
	want := []byte("kept")
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

// TestWriteAndCloseOnClosedWriter verifies both methods report ErrClosed after a
// successful Close and that Close is idempotent in its error.
func TestWriteAndCloseOnClosedWriter(t *testing.T) {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "done"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("late")); !errors.Is(err, bzip2.ErrClosed) {
		t.Fatalf("Write: got %v want ErrClosed", err)
	}
	if err := w.Close(); !errors.Is(err, bzip2.ErrClosed) {
		t.Fatalf("second Close: got %v want ErrClosed", err)
	}
	// The already-written stream must still be intact and decodable.
	got, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("done")) {
		t.Fatalf("got %q", got)
	}
}

// TestWriteErrorLatches: once the destination fails, later Writes return the
// same error and Close reports it.
func TestWriteErrorLatches(t *testing.T) {
	mock := errors.New("destination exploded")
	w, err := bzip2.NewWriter(errWriterWithErr{err: mock}, 9)
	if err != nil {
		t.Fatal(err)
	}
	// Push enough data to force a block flush inside Write so the error
	// surfaces from Write itself, not only from Close.
	big := bytes.Repeat([]byte("push it"), 200000)
	if _, err := w.Write(big); !errors.Is(err, mock) {
		t.Fatalf("Write: got %v want %v", err, mock)
	}
	if _, err := w.Write([]byte("x")); !errors.Is(err, mock) {
		t.Fatalf("latched Write: got %v want %v", err, mock)
	}
	if err := w.Close(); !errors.Is(err, mock) {
		t.Fatalf("Close: got %v want %v", err, mock)
	}
}
