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

// compressBZ compresses data at level through the public Writer and returns the stream.
func compressBZ(t *testing.T, level int, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, level)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// stdlibDecode decompresses compressed with the reference implementation directly.
// This is the correctness oracle for the encoder: anything the Writer emits must
// be understood by the standard library decoder back to the identical input.
func stdlibDecode(t *testing.T, compressed []byte) []byte {
	t.Helper()
	out, err := io.ReadAll(stdbz2.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		t.Fatalf("stdlib decode failed: %v", err)
	}
	return out
}

// cyclingBytes returns n bytes where every byte differs from its predecessor,
// so the RLE stage emits exactly one block byte per input byte. This gives exact
// control over where the encoded block boundary lands: the first block absorbs
// exactly nblockMax input bytes.
func cyclingBytes(n int) []byte {
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(i)
	}
	return data
}

func randBytes(rng *rand.Rand, n int) []byte {
	data := make([]byte, n)
	for i := 0; i < len(data); i += 8 {
		v := rng.Uint64()
		for j := 0; j < 8 && i+j < len(data); j++ {
			data[i+j] = byte(v >> (8 * j))
		}
	}
	return data
}

// TestOracleBlockBoundariesAllLevels exercises every compression level at sizes
// straddling the exact block capacity: nblockMax-1 and nblockMax fit in a single
// block, nblockMax+1 forces a second block containing one byte.
func TestOracleBlockBoundariesAllLevels(t *testing.T) {
	levels := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	if raceDetectorEnabled {
		levels = []int{1, 5, 9}
	}
	for _, level := range levels {
		nblockMax := 100000*level - 19
		for _, delta := range []int{-1, 0, 1} {
			n := nblockMax + delta
			data := cyclingBytes(n)
			got := stdlibDecode(t, compressBZ(t, level, data))
			if !bytes.Equal(got, data) {
				t.Fatalf("level=%d n=%d: roundtrip mismatch", level, n)
			}
		}
	}
}

// TestOracleTwoFullBlocks writes exactly two full blocks plus overflow for a few
// levels, crossing the boundary where the second block begins.
func TestOracleTwoFullBlocks(t *testing.T) {
	levels := []int{1, 5, 9}
	if raceDetectorEnabled {
		levels = []int{1, 5}
	}
	for _, level := range levels {
		nblockMax := 100000*level - 19
		for _, n := range []int{2 * nblockMax, 2*nblockMax + 1, 2*nblockMax + 4096} {
			data := cyclingBytes(n)
			got := stdlibDecode(t, compressBZ(t, level, data))
			if !bytes.Equal(got, data) {
				t.Fatalf("level=%d n=%d: roundtrip mismatch", level, n)
			}
		}
	}
}

// TestOracleIncompressibleAtBoundary feeds random incompressible data at sizes
// around the block capacity for the extreme levels.
func TestOracleIncompressibleAtBoundary(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	levels := []int{1, 9}
	if raceDetectorEnabled {
		levels = []int{1}
	}
	for _, level := range levels {
		nblockMax := 100000*level - 19
		for _, n := range []int{nblockMax - 1, nblockMax, nblockMax + 1, nblockMax + 300000} {
			data := randBytes(rng, n)
			got := stdlibDecode(t, compressBZ(t, level, data))
			if !bytes.Equal(got, data) {
				t.Fatalf("level=%d n=%d: roundtrip mismatch", level, n)
			}
		}
	}
}

// TestOracleRunCrossesBlockBoundary makes a long same-byte run straddle the block
// boundary. The RLE run state must carry into the next block correctly.
func TestOracleRunCrossesBlockBoundary(t *testing.T) {
	levels := []int{1, 9}
	if raceDetectorEnabled {
		levels = []int{1}
	}
	for _, level := range levels {
		nblockMax := 100000*level - 19
		// Fill the block almost completely, then start a run longer than one
		// RLE pair (max 255 per pair) that must continue in the next block.
		data := append(cyclingBytes(nblockMax-2), bytes.Repeat([]byte{0xaa}, 2000)...)
		data = append(data, cyclingBytes(513)...)
		got := stdlibDecode(t, compressBZ(t, level, data))
		if !bytes.Equal(got, data) {
			t.Fatalf("level=%d: run-across-boundary mismatch", level)
		}
	}
}

// TestOraclePayloadShapes covers canonical edge-case inputs at several levels.
func TestOraclePayloadShapes(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	shapes := map[string][]byte{
		"empty":           {},
		"one_byte":        {0x42},
		"two_same":        {0x7f, 0x7f},
		"run_4":           bytes.Repeat([]byte{'z'}, 4),
		"run_255":         bytes.Repeat([]byte{'z'}, 255),
		"run_256":         bytes.Repeat([]byte{'z'}, 256),
		"run_2000":        bytes.Repeat([]byte{'z'}, 2000),
		"run_1M":          bytes.Repeat([]byte{0x00}, 1<<20),
		"alternating":     bytes.Repeat([]byte("ab"), 40000),
		"all_256_bytes":   cyclingBytes(256),
		"all_256_twice":   cyclingBytes(512),
		"random_64k":      randBytes(rng, 64*1024),
		"random_1byte":    randBytes(rng, 1),
		"high_bytes_only": bytes.Repeat([]byte{0xff, 0xfe}, 30000),
	}
	for _, level := range []int{1, 5, 9} {
		for name, data := range shapes {
			t.Run(name, func(t *testing.T) {
				got := stdlibDecode(t, compressBZ(t, level, data))
				if !bytes.Equal(got, data) {
					t.Fatalf("level=%d %s len=%d: mismatch", level, name, len(data))
				}
			})
		}
	}
}

// TestOracleMultiBlockMixedLarge streams over a megabyte alternating compressible
// and incompressible segments so block splitting happens on varied content.
func TestOracleMultiBlockMixedLarge(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	var data []byte
	for len(data) < 1500*1024 {
		if len(data)%(128*1024) < 64*1024 {
			data = append(data, bytes.Repeat([]byte("the quick brown fox\n"), 4096)...)
		} else {
			data = append(data, randBytes(rng, 8192)...)
		}
	}
	levels := []int{1, 9}
	if raceDetectorEnabled {
		levels = []int{1}
	}
	for _, level := range levels {
		got := stdlibDecode(t, compressBZ(t, level, data))
		if !bytes.Equal(got, data) {
			t.Fatalf("level=%d: mixed multi-block mismatch (in=%d)", level, len(data))
		}
	}
}

// TestOracleChunkedWriteEquivalence asserts the encoder is a pure function of the
// byte stream: the same input written in small chunks must produce byte-identical
// output to a single Write call.
func TestOracleChunkedWriteEquivalence(t *testing.T) {
	rng := rand.New(rand.NewSource(13))
	data := randBytes(rng, 200000)
	copy(data[1000:], bytes.Repeat([]byte("chunk"), 1000))
	want := compressBZ(t, 5, data)
	for _, step := range []int{1, 7, 4096, 65536} {
		var buf bytes.Buffer
		w, err := bzip2.NewWriter(&buf, 5)
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
		if !bytes.Equal(buf.Bytes(), want) {
			t.Fatalf("step=%d: chunked output differs from single-write output", step)
		}
	}
}
