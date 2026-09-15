// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package enc

import (
	"bytes"
	"testing"
)

func BenchmarkWriteBlockPrepared(b *testing.B) {
	const n = 600000
	raw := bytes.Repeat([]byte("x"), n)
	block, inUse, crc := EncodeRLEBlock(raw, n+100)

	var bw BitWriter
	bw.Grow(EncodedBitBufferCap(len(block)))
	var scratch Scratch
	scratch.PrepareEncoderAux()
	WriteBlock(&bw, block, inUse, crc, &scratch)

	b.ReportAllocs()
	b.SetBytes(int64(len(raw)))
	b.ResetTimer()
	for b.Loop() {
		bw.ResetForNewStream()
		WriteBlock(&bw, block, inUse, crc, &scratch)
	}
}
