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

func TestZeroValueWriterGuarded(t *testing.T) {
	var w bzip2.Writer
	if _, err := w.Write([]byte("x")); !errors.Is(err, bzip2.ErrWriterUninitialized) {
		t.Fatalf("Write error=%v, want ErrWriterUninitialized", err)
	}
	if err := w.Close(); !errors.Is(err, bzip2.ErrWriterUninitialized) {
		t.Fatalf("Close error=%v, want ErrWriterUninitialized", err)
	}
	var dst bytes.Buffer
	if err := w.Reset(&dst); !errors.Is(err, bzip2.ErrWriterUninitialized) {
		t.Fatalf("Reset error=%v, want ErrWriterUninitialized", err)
	}
}

func TestDecoderMutatedStreamNoPanic(t *testing.T) {
	payload := bytes.Repeat([]byte("attack-surface-"), 2048)
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
	enc := append([]byte(nil), buf.Bytes()...)
	for i := 0; i < len(enc); i += 257 {
		mut := append([]byte(nil), enc...)
		mut[i] ^= 0x5a
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic at mutation index %d: %v", i, r)
				}
			}()
			_, _ = io.ReadAll(bzip2.NewReader(bytes.NewReader(mut)))
		}()
	}
}
