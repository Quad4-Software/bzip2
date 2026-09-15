// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

func TestCompressStreamRoundtrip(t *testing.T) {
	want := strings.Repeat("example pipeline payload\n", 4000)
	var buf bytes.Buffer
	if err := compressStream(&buf, strings.NewReader(want), 9); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte(want)) {
		t.Fatal("compressStream roundtrip mismatch")
	}
}

func TestCompressStreamLevels(t *testing.T) {
	for level := 1; level <= 9; level++ {
		var buf bytes.Buffer
		if err := compressStream(&buf, strings.NewReader("x"), level); err != nil {
			t.Fatalf("level %d: %v", level, err)
		}
		if buf.Len() == 0 {
			t.Fatalf("level %d: no output", level)
		}
	}
	for _, level := range []int{-1, 0, 10} {
		var buf bytes.Buffer
		if err := compressStream(&buf, strings.NewReader("x"), level); err == nil {
			t.Fatalf("level %d: expected error", level)
		}
	}
}

// errReader fails on the first Read so the io.Copy branch can be checked.
type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failure")
}

func TestCompressStreamReadError(t *testing.T) {
	var buf bytes.Buffer
	if err := compressStream(&buf, errReader{}, 9); err == nil {
		t.Fatal("expected copy error")
	}
}

func TestCompressStreamEmptyInput(t *testing.T) {
	var buf bytes.Buffer
	if err := compressStream(&buf, strings.NewReader(""), 5); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty output, got %d bytes", len(got))
	}
}
