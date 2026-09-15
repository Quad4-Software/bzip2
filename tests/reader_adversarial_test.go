// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	stdbz2 "compress/bzip2"
	"io"
	"math/rand"
	"testing"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
)

func decode(t *testing.T, data []byte) ([]byte, error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("decoder panic: %v", r)
		}
	}()
	return io.ReadAll(bzip2.NewReader(bytes.NewReader(data)))
}

func TestReaderEmptyInput(t *testing.T) {
	out, err := decode(t, nil)
	if err == nil {
		t.Fatalf("empty input decoded to %d bytes with no error", len(out))
	}
	out, err = decode(t, []byte{})
	if err == nil {
		t.Fatalf("zero-length input decoded to %d bytes with no error", len(out))
	}
}

func TestReaderCorruptMagic(t *testing.T) {
	good := compressBZ(t, 9, []byte("magic test payload"))
	for _, tc := range []struct {
		name string
		mut  func([]byte)
	}{
		{"first_byte", func(b []byte) { b[0] = 'X' }},
		{"second_byte", func(b []byte) { b[1] = 'X' }},
		{"third_byte", func(b []byte) { b[2] = 'X' }},
		{"level_zero", func(b []byte) { b[3] = '0' }},
		{"level_colon", func(b []byte) { b[3] = ':' }},
	} {
		bad := append([]byte(nil), good...)
		tc.mut(bad)
		out, err := decode(t, bad)
		if err == nil {
			t.Fatalf("%s: decoded %d bytes with no error", tc.name, len(out))
		}
	}
}

// TestReaderEveryTruncation cuts a valid stream at every possible length and
// requires an error for each strict prefix; the full stream must decode.
func TestReaderEveryTruncation(t *testing.T) {
	want := []byte("truncate me at every byte boundary\n")
	full := compressBZ(t, 9, want)
	for cut := 0; cut < len(full); cut++ {
		out, err := decode(t, full[:cut])
		if err == nil {
			t.Fatalf("cut=%d/%d: decoded %d bytes without error", cut, len(full), len(out))
		}
	}
	got, err := decode(t, full)
	if err != nil {
		t.Fatalf("full stream: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("full stream mismatch")
	}
}

// TestReaderBitFlipInvariant flips each byte of a valid stream and asserts the
// decoder never panics and never silently returns wrong data: a corruption may
// produce an error or (for padding bits) the original output, never different
// output with a nil error.
func TestReaderBitFlipInvariant(t *testing.T) {
	want := bytes.Repeat([]byte("flip me "), 600)
	full := compressBZ(t, 9, want)
	for i := range full {
		for _, mask := range []byte{0x01, 0x80} {
			mut := append([]byte(nil), full...)
			mut[i] ^= mask
			out, err := decode(t, mut)
			if err == nil && !bytes.Equal(out, want) {
				t.Fatalf("byte %d mask %#02x: silent corruption, out len %d", i, mask, len(out))
			}
		}
	}
}

// TestReaderTrailingGarbage: the stdlib decoder parses trailing bytes as a
// continuation stream, so appended garbage must surface an error while still
// returning the first stream's payload.
func TestReaderTrailingGarbage(t *testing.T) {
	want := []byte("first stream contents")
	full := compressBZ(t, 9, want)
	dirty := append(append([]byte(nil), full...), 0xde, 0xad, 0xbe, 0xef)
	out, err := decode(t, dirty)
	if err == nil {
		t.Fatal("trailing garbage produced no error")
	}
	if !bytes.Equal(out, want) {
		t.Fatalf("payload before garbage: got %q want %q", out, want)
	}
}

// TestReaderConcatenatedStreams: stdlib supports multi-stream bzip2 files, so
// two concatenated Writer outputs must decode to the concatenated plaintext.
func TestReaderConcatenatedStreams(t *testing.T) {
	a := []byte("first part|")
	b := []byte("second part")
	var buf bytes.Buffer
	buf.Write(compressBZ(t, 9, a))
	buf.Write(compressBZ(t, 5, b))
	out, err := io.ReadAll(stdbz2.NewReader(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, append(append([]byte(nil), a...), b...)) {
		t.Fatalf("got %q", out)
	}
}

// TestReaderRandomGarbage: arbitrary bytes must never panic the decoder and
// almost always produce an error.
func TestReaderRandomGarbage(t *testing.T) {
	rng := rand.New(rand.NewSource(19))
	for i := 0; i < 200; i++ {
		data := randBytes(rng, rng.Intn(512))
		_, _ = decode(t, data)
	}
}
