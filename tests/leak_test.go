// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"io"
	"runtime"
	"testing"
	"time"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

// numGoroutines settles briefly so transient runtime goroutines from prior
// tests do not skew the baseline.
func numGoroutines() int {
	time.Sleep(10 * time.Millisecond)
	return runtime.NumGoroutine()
}

// TestNoGoroutineLeak verifies Write/Close/Read cycles spawn no goroutines and
// leave none behind.
func TestNoGoroutineLeak(t *testing.T) {
	before := numGoroutines()
	payload := bytes.Repeat([]byte("leak check "), 5000)
	for i := 0; i < 40; i++ {
		var buf bytes.Buffer
		w, err := bzip2.NewWriter(&buf, 9)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(payload); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := io.ReadAll(bzip2.NewReader(&buf)); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		after := runtime.NumGoroutine()
		if after <= before {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("goroutine leak: before=%d after=%d", before, after)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestAllocsPerRunSmall(t *testing.T) {
	n := testing.AllocsPerRun(100, func() {
		var buf bytes.Buffer
		w, err := bzip2.NewWriter(&buf, 9)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	})
	if n > 40 {
		t.Fatalf("allocs per run: %f (new Writer + small payload each iteration)", n)
	}
}

func TestAllocsWriterReuseDiscard(t *testing.T) {
	const payloadMul = 2000
	payload := bytes.Repeat([]byte("y"), payloadMul)

	w, err := bzip2.NewWriter(io.Discard, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	n := testing.AllocsPerRun(200, func() {
		if err := w.Reset(io.Discard); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(payload); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	})
	if n > 0 {
		t.Fatalf("expected zero heap allocs after warm buffers with io.Discard, got %f", n)
	}
}

func TestWriterResetRoundtripSequences(t *testing.T) {
	streams := [][]byte{
		[]byte("short"),
		bytes.Repeat([]byte("ABCDEFGH"), 80_000),
		[]byte("tail"),
	}
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, 9)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range streams {
		buf.Reset()
		if i > 0 {
			if err := w.Reset(&buf); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := w.Write(want); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(bzip2.NewReader(&buf))
		if err != nil {
			t.Fatalf("stream %d decode: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("stream %d: got len %d want len %d", i, len(got), len(want))
		}
	}
}
