// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"io"
	"math/rand"
	"testing"

	"github.com/Quad4-Software/bzip2/pkg/bzip2"
	"github.com/Quad4-Software/pbt/pkg/pbt"
)

// payloadGen produces byte slices mixing several compressibility profiles so the
// RLE and Huffman stages see varied input: incompressible noise, single-byte
// runs, medium runs, text-like data, and tiny alphabets.
var payloadGen = pbt.NewGenerator("payload", func(r *rand.Rand, size int) []byte {
	n := 0
	if size > 0 {
		n = r.Intn(size + 1)
	}
	data := make([]byte, n)
	switch r.Intn(6) {
	case 0:
		for i := range data {
			data[i] = byte(r.Intn(256))
		}
	case 1:
		b := byte(r.Intn(256))
		for i := range data {
			data[i] = b
		}
	case 2:
		for i := 0; i < n; {
			b := byte(r.Intn(256))
			run := 1 + r.Intn(600)
			for j := 0; j < run && i < n; j++ {
				data[i] = b
				i++
			}
		}
	case 3:
		for i := range data {
			data[i] = byte(32 + r.Intn(95))
		}
	case 4:
		alpha := 1 + r.Intn(4)
		for i := range data {
			data[i] = byte(i % alpha)
		}
	case 5:
		// Cycle through all 256 byte values: no adjacent repeats, exercises
		// the worst case for the RLE stage.
		for i := range data {
			data[i] = byte(i)
		}
	}
	return data
})

func roundtripOK(data []byte, level int) bool {
	var buf bytes.Buffer
	w, err := bzip2.NewWriter(&buf, level)
	if err != nil {
		return false
	}
	if _, err := w.Write(data); err != nil {
		return false
	}
	if err := w.Close(); err != nil {
		return false
	}
	got, err := io.ReadAll(bzip2.NewReader(&buf))
	if err != nil {
		return false
	}
	return bytes.Equal(got, data)
}

// TestPropertyRoundtripIdentity: for arbitrary generated payloads, compressing
// then decompressing through the public API yields the identical input.
func TestPropertyRoundtripIdentity(t *testing.T) {
	prop := pbt.ForAll(
		"bzip2 roundtrip is identity",
		payloadGen,
		func(data []byte) bool { return roundtripOK(data, 6) },
		pbt.WithShrinker[[]byte](pbt.SliceShrinker[byte]()),
		pbt.WithLabeler(func(data []byte) []string {
			switch {
			case len(data) == 0:
				return []string{"empty"}
			case len(data) < 64:
				return []string{"small"}
			case len(data) < 4096:
				return []string{"medium"}
			default:
				return []string{"large"}
			}
		}),
	)
	pbt.Check(t, prop, pbt.WithRuns(400), pbt.WithMaxSize(20000), pbt.WithSeed(2026))
}

// TestPropertyChunkedWriteRoundtrip: splitting the input across arbitrarily
// chunked Write calls must not change the decoded result.
func TestPropertyChunkedWriteRoundtrip(t *testing.T) {
	gen := pbt.Tuple2("payload-chunk", payloadGen, pbt.IntRange(1, 4096))
	prop := pbt.ForAll(
		"chunked writes roundtrip",
		gen,
		func(v pbt.Tuple2Value[[]byte, int]) bool {
			data, step := v.First, v.Second
			if step < 1 {
				step = 1
			}
			var buf bytes.Buffer
			w, err := bzip2.NewWriter(&buf, 3)
			if err != nil {
				return false
			}
			for i := 0; i < len(data); i += step {
				j := i + step
				if j > len(data) {
					j = len(data)
				}
				if _, err := w.Write(data[i:j]); err != nil {
					return false
				}
			}
			if err := w.Close(); err != nil {
				return false
			}
			got, err := io.ReadAll(bzip2.NewReader(&buf))
			if err != nil {
				return false
			}
			return bytes.Equal(got, data)
		},
	)
	pbt.Check(t, prop, pbt.WithRuns(250), pbt.WithMaxSize(12000), pbt.WithSeed(77))
}

// TestPropertyDecoderNeverPanics: feeding arbitrary bytes to NewReader must
// never panic; an error or empty output are both acceptable outcomes.
func TestPropertyDecoderNeverPanics(t *testing.T) {
	prop := pbt.ForAll(
		"decoder never panics",
		payloadGen,
		func(data []byte) (ok bool) {
			defer func() {
				if r := recover(); r != nil {
					ok = false
				}
			}()
			_, _ = io.ReadAll(bzip2.NewReader(bytes.NewReader(data)))
			return true
		},
	)
	pbt.Check(t, prop, pbt.WithRuns(400), pbt.WithMaxSize(2048), pbt.WithSeed(555))
}
