// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package bzip2_test

import (
	"bytes"
	"io"
	"testing"

	"git.quad4.io/Go-Libs/bzip2/pkg/bzip2"
	"git.quad4.io/Go-Libs/pbt/pkg/pbt"
)

func intSliceToBytes(xs []int) []byte {
	b := make([]byte, len(xs))
	for i, v := range xs {
		b[i] = byte(v)
	}
	return b
}

func TestPBTRoundtrip(t *testing.T) {
	gen := pbt.Map("[]byte",
		pbt.SliceOf(pbt.IntRange(0, 255), 0, 16384),
		intSliceToBytes,
	)
	prop := pbt.ForAll(
		"compress then decompress roundtrip",
		gen,
		func(data []byte) bool {
			var buf bytes.Buffer
			w, err := bzip2.NewWriter(&buf, 7)
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
		},
		pbt.WithShrinker[[]byte](pbt.SliceShrinker[byte]()),
	)
	pbt.Check(t, prop, pbt.WithRuns(200), pbt.WithSeed(42))
}
