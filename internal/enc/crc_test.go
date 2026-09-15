// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package enc

import "testing"

func TestBlockCRCMatchesKnown(t *testing.T) {
	const want uint32 = 0x4eece836
	got := blockCRCFromBytes([]byte("hello world\n"))
	if got != want {
		t.Fatalf("crc got %#x want %#x", got, want)
	}
}
