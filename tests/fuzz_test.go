// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"io"
	"testing"

	"git.quad4.io/Go-Libs/bzip2/pkg/bzip2"
)

func FuzzRoundtrip(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte("a"))
	f.Add(bytes.Repeat([]byte{0x5a}, 1024))
	f.Fuzz(func(t *testing.T, data []byte) {
		var buf bytes.Buffer
		w, err := bzip2.NewWriter(&buf, 9)
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
			t.Fatal("mismatch")
		}
	})
}

func FuzzRoundtripChunkedReset(f *testing.F) {
	f.Add([]byte("hello"), uint8(3), uint8(9))
	f.Add(bytes.Repeat([]byte("ab"), 4096), uint8(17), uint8(1))
	f.Add([]byte{}, uint8(1), uint8(5))
	f.Fuzz(func(t *testing.T, data []byte, chunkSz uint8, lvl uint8) {
		level := int(lvl%9) + 1
		step := int(chunkSz)
		if step <= 0 {
			step = 1
		}
		var buf bytes.Buffer
		w, err := bzip2.NewWriter(&buf, level)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < len(data); i += step {
			j := min(i+step, len(data))
			if _, err := w.Write(data[i:j]); err != nil {
				t.Fatal(err)
			}
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(bzip2.NewReader(&buf))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data) {
			t.Fatal("first stream mismatch")
		}

		buf.Reset()
		if err := w.Reset(&buf); err != nil {
			t.Fatal(err)
		}
		for i := len(data); i > 0; {
			j := max(i-step, 0)
			if _, err := w.Write(data[j:i]); err != nil {
				t.Fatal(err)
			}
			i = j
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		got2, err := io.ReadAll(bzip2.NewReader(&buf))
		if err != nil {
			t.Fatal(err)
		}
		want2 := make([]byte, len(data))
		k := 0
		for i := len(data); i > 0; {
			j := max(i-step, 0)
			n := copy(want2[k:], data[j:i])
			k += n
			i = j
		}
		if !bytes.Equal(got2, want2) {
			t.Fatal("reset stream mismatch")
		}
	})
}

func FuzzDecodeNoPanic(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte("BZh9garbage"))
	f.Add([]byte{0x42, 0x5a, 0x68, 0x39, 0x00})
	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("decoder panic: %v", r)
			}
		}()
		_, _ = io.ReadAll(bzip2.NewReader(bytes.NewReader(data)))
	})
}
